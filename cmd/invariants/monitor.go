package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// monitorCmd: long-running daemon. Runs the same check set as `inv all`
// on a ticker, posts results to a Discord webhook, applies hysteresis
// so a single transient RPC blip doesn't page, and emits a daily
// aliveness ping when everything's green.
//
// Replaces the admin-cli `monitor` daemon. Designed to drop into the
// existing monitor k8s deployment (long-lived, restarts on crash, env
// vars for webhooks).
//
// Knobs:
//
//	--interval N      tick cadence (default 24h; can shorten via env)
//	--once            run one tick and exit (for testing / cron mode)
//	--fail-threshold  page only after this many consecutive same-check
//	                  failures (default 2 — kills RPC flapping)
//	--epoch / --tolerance  forwarded to children, same semantics as `inv all`
//
// Env / config:
//
//	DISCORD_WEBHOOK_URL              channel for results + aliveness
//	DISCORD_WEBHOOK_URL_ERRORS       (optional) separate channel for
//	                                 unexpected daemon-level errors
//	                                 (defaults to DISCORD_WEBHOOK_URL)
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Run all invariant checks on a ticker, post results to Discord",
	Long: `Long-running daemon that periodically runs every invariant check
(same set as ` + "`inv all`" + `) and reports to a Discord webhook.

Hysteresis: a check has to fail two consecutive ticks before it pages,
which kills false alerts from transient RPC blips. State resets as
soon as the check returns to green.

Aliveness: once per --aliveness-interval (default 24h) the daemon
emits a single "all green" message even when nothing changed, so an
operator can tell the loop is running.

Designed to drop into the same k8s deployment that previously ran
` + "`admin-cli monitor`" + `.`,
	Args: cobra.NoArgs,
	Run:  runMonitor,
}

func runMonitor(cmd *cobra.Command, _ []string) {
	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	interval, _ := cmd.Flags().GetDuration("interval")
	once, _ := cmd.Flags().GetBool("once")
	failThreshold, _ := cmd.Flags().GetInt("fail-threshold")
	alivenessInterval, _ := cmd.Flags().GetDuration("aliveness-interval")
	epoch, _ := cmd.Flags().GetUint64("epoch")
	tolerance, _ := cmd.Flags().GetUint64("tolerance")

	webhook := viper.GetString("discord_webhook_url")
	errWebhook := viper.GetString("discord_webhook_url_errors")
	if errWebhook == "" {
		errWebhook = webhook
	}
	if webhook == "" && !once {
		fmt.Fprintln(os.Stderr, "WARN: DISCORD_WEBHOOK_URL not set; daemon will run but won't post to Discord")
	}

	// Checks(ctx, ...) calls getHeadEpoch when --epoch is unset, which
	// needs the lotus singleton. Initialize once at daemon startup
	// rather than per-tick so a transient init failure surfaces
	// immediately instead of being hidden behind the first tick.
	if err := initSingleton(ctx); err != nil {
		log.Fatalf("monitor: init singleton: %v", err)
	}

	d := &monitorDaemon{
		interval:          interval,
		alivenessInterval: alivenessInterval,
		webhook:           webhook,
		errWebhook:        errWebhook,
		failThreshold:     failThreshold,
		state:             make(map[string]int),
		epoch:             epoch,
		tolerance:         tolerance,
	}

	if once {
		d.runOneTick(ctx)
		return
	}

	fmt.Printf("InvariantsMonitor starting: interval=%s aliveness=%s fail-threshold=%d webhook=%s\n",
		interval, alivenessInterval, failThreshold, redactURL(webhook))
	d.postOK("InvariantsMonitor starting up. interval=" + interval.String())

	// First tick immediately, then on the ticker.
	d.runOneTick(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("InvariantsMonitor: shutting down")
			d.postOK("InvariantsMonitor shutting down.")
			return
		case <-ticker.C:
			d.runOneTick(ctx)
		}
	}
}

type monitorDaemon struct {
	interval          time.Duration
	alivenessInterval time.Duration
	webhook           string
	errWebhook        string
	failThreshold     int
	epoch             uint64
	tolerance         uint64

	mu                sync.Mutex
	state             map[string]int // check name → consecutive-fail count
	lastAlivenessPing time.Time
}

type tickResult struct {
	name     string
	pass     bool
	output   string
	duration time.Duration
}

// runOneTick executes every child check once and posts a Discord
// rollup capturing what just happened.
func (d *monitorDaemon) runOneTick(ctx context.Context) {
	children, err := Checks(ctx, d.epoch, d.tolerance)
	if err != nil {
		// Treat a checks-build failure as a daemon-level error rather
		// than a check failure — log + post to the error channel and
		// skip this tick. Don't update hysteresis state; the next
		// tick will try again.
		fmt.Fprintf(os.Stderr, "monitor: build check list: %v\n", err)
		if d.errWebhook != "" {
			_ = postDiscord(d.errWebhook,
				fmt.Sprintf(":warning: InvariantsMonitor: failed to build check list this tick: `%v`", err))
		}
		return
	}
	results := make([]tickResult, 0, len(children))

	tickStart := time.Now()
	for _, ch := range children {
		results = append(results, d.runChild(ctx, ch))
	}
	tickDur := time.Since(tickStart)

	// Update hysteresis state and decide what to post.
	d.mu.Lock()
	defer d.mu.Unlock()

	var nowFailing []tickResult     // crossed the threshold this tick
	var stillFailing []tickResult   // still failing but below threshold
	var recovered []string          // were failing, now green
	allPass := true

	for _, r := range results {
		prev := d.state[r.name]
		if r.pass {
			if prev >= d.failThreshold {
				recovered = append(recovered, r.name)
			}
			d.state[r.name] = 0
			continue
		}
		allPass = false
		newCount := prev + 1
		d.state[r.name] = newCount
		switch {
		case newCount == d.failThreshold:
			nowFailing = append(nowFailing, r)
		case newCount > d.failThreshold:
			// already paged; suppress to avoid spam
		default:
			stillFailing = append(stillFailing, r)
		}
	}

	// Compose the message.
	switch {
	case len(nowFailing) > 0 || len(recovered) > 0:
		var b strings.Builder
		fmt.Fprintf(&b, "**InvariantsMonitor — state change** (tick %s)\n", tickDur.Round(time.Second))
		for _, r := range nowFailing {
			fmt.Fprintf(&b, "❌ **FAIL** `%s` — failing %d consecutive ticks. Output:\n```%s```\n",
				r.name, d.state[r.name], truncateForDiscord(r.output))
		}
		for _, name := range recovered {
			fmt.Fprintf(&b, "✅ recovered: `%s`\n", name)
		}
		d.postOK(b.String())
	case allPass && time.Since(d.lastAlivenessPing) >= d.alivenessInterval:
		var b strings.Builder
		fmt.Fprintf(&b, "✅ InvariantsMonitor: %d/%d checks pass (tick %s)\n",
			len(results), len(results), tickDur.Round(time.Second))
		for _, r := range results {
			fmt.Fprintf(&b, "  • `%s` (%s)\n", r.name, r.duration.Round(time.Millisecond))
		}
		d.postOK(b.String())
		d.lastAlivenessPing = time.Now()
	case len(stillFailing) > 0:
		// Below threshold — log locally, don't page.
		for _, r := range stillFailing {
			fmt.Printf("transient fail [%d/%d]: %s\n", d.state[r.name], d.failThreshold, r.name)
		}
	}

	// Always log a one-line summary locally.
	pass := 0
	for _, r := range results {
		if r.pass {
			pass++
		}
	}
	fmt.Printf("tick %s: %d/%d passed\n", tickDur.Round(time.Second), pass, len(results))
}

// runChild spawns one of the existing subcommands, captures stdout +
// stderr, and reports pass/fail by exit code.
func (d *monitorDaemon) runChild(ctx context.Context, ch checkSpec) tickResult {
	start := time.Now()
	c := exec.CommandContext(ctx, os.Args[0], ch.args...)
	c.Env = os.Environ()
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	err := c.Run()
	return tickResult{
		name:     ch.name,
		pass:     err == nil,
		output:   buf.String(),
		duration: time.Since(start),
	}
}

// postOK posts to the main webhook. Errors are logged but never
// fatal — Discord is monitoring noise, not the source of truth.
func (d *monitorDaemon) postOK(content string) {
	if d.webhook == "" {
		return
	}
	if err := postDiscord(d.webhook, content); err != nil {
		fmt.Fprintf(os.Stderr, "discord post failed: %v\n", err)
	}
}

func postDiscord(webhook, content string) error {
	// Discord webhook content limit is 2000 chars.
	if len(content) > 1900 {
		content = content[:1900] + "\n…(truncated)"
	}
	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func truncateForDiscord(s string) string {
	const max = 800
	if len(s) <= max {
		return s
	}
	// Keep the tail — most check failures put the diagnostic at the end.
	return "…\n" + s[len(s)-max:]
}

// redactURL returns the host of a webhook URL for safe logging
// (Discord webhooks contain a secret in the path).
func redactURL(u string) string {
	if u == "" {
		return "<none>"
	}
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			return u[:i+3] + rest[:j] + "/<redacted>"
		}
	}
	return "<set>"
}

func init() {
	monitorCmd.Flags().Duration("interval", 24*time.Hour, "tick cadence (e.g. 1h, 30m, 24h)")
	monitorCmd.Flags().Bool("once", false, "run a single tick and exit")
	monitorCmd.Flags().Int("fail-threshold", 2, "page only after this many consecutive same-check failures")
	monitorCmd.Flags().Duration("aliveness-interval", 24*time.Hour, "send a green-state ping at most this often")
	monitorCmd.Flags().Uint64("epoch", 0, "pin checks to this epoch (default: head-3 each tick)")
	monitorCmd.Flags().Uint64("tolerance", 10000, "wei tolerance forwarded to each child check")
	rootCmd.AddCommand(monitorCmd)
}
