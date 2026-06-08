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
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// monitorCmd: long-running daemon. Runs the same check set as `inv all`
// on a ticker and posts every tick's result to a Discord webhook.
//
// No hysteresis, no aliveness gating — every tick produces a Discord
// message. Earlier versions suppressed "still failing" ticks once the
// initial page had fired; that hid output changes (e.g. agent dtl
// going from 3 over → 2 over → 4 over) and turned multi-day failures
// into 1-message events. Posting every tick makes the channel itself
// the heartbeat: silence means the daemon is down, two messages a day
// at --interval 12h means it's alive.
//
// Knobs:
//
//	--interval N      tick cadence (default 12h)
//	--once            run a single tick and exit (for testing / cron mode)
//	--epoch           pin checks to this epoch (default: head-3 each tick)
//	--tolerance       wei tolerance forwarded to children
//
// Env / config:
//
//	DISCORD_WEBHOOK_URL              channel for tick rollups
//	DISCORD_WEBHOOK_URL_ERRORS       (optional) separate channel for
//	                                 unexpected daemon-level errors
//	                                 (defaults to DISCORD_WEBHOOK_URL)
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Run all invariant checks on a ticker, post every result to Discord",
	Long: `Long-running daemon that periodically runs every invariant check
(same set as ` + "`inv all`" + `) and reports each result to a Discord webhook.

Every tick posts: an all-green rollup when every check passes, or a
per-check failure rollup with the check's output when one or more
fail. The channel itself is the heartbeat — visible cadence in the
log means the daemon is alive; silence means it isn't.

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
		interval:   interval,
		webhook:    webhook,
		errWebhook: errWebhook,
		epoch:      epoch,
		tolerance:  tolerance,
	}

	if once {
		d.runOneTick(ctx)
		return
	}

	fmt.Printf("InvariantsMonitor starting: interval=%s webhook=%s\n",
		interval, redactURL(webhook))
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
	interval   time.Duration
	webhook    string
	errWebhook string
	epoch      uint64
	tolerance  uint64
}

type tickResult struct {
	name     string
	pass     bool
	output   string
	duration time.Duration
}

// runOneTick executes every child check once and posts a single
// rollup of the result to Discord. No hysteresis, no aliveness
// gating — the channel sees something every tick, which is the
// signal that the daemon is up.
func (d *monitorDaemon) runOneTick(ctx context.Context) {
	children, err := Checks(ctx, d.epoch, d.tolerance)
	if err != nil {
		// Treat a checks-build failure as a daemon-level error rather
		// than a check failure — log + post to the error channel and
		// skip this tick. The next tick will try again.
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

	pass := 0
	for _, r := range results {
		if r.pass {
			pass++
		}
	}

	var b strings.Builder
	if pass == len(results) {
		fmt.Fprintf(&b, "✅ InvariantsMonitor: %d/%d pass (tick %s)\n",
			pass, len(results), tickDur.Round(time.Second))
		for _, r := range results {
			fmt.Fprintf(&b, "  • `%s` (%s)\n", r.name, r.duration.Round(time.Millisecond))
		}
	} else {
		fmt.Fprintf(&b, "⚠️ InvariantsMonitor: %d/%d pass (tick %s)\n",
			pass, len(results), tickDur.Round(time.Second))
		// Failures first with their output, then a summary of the
		// passing checks for context.
		for _, r := range results {
			if !r.pass {
				fmt.Fprintf(&b, "❌ FAIL `%s` (%s). Output:\n```%s```\n",
					r.name, r.duration.Round(time.Millisecond),
					truncateForDiscord(r.output))
			}
		}
		for _, r := range results {
			if r.pass {
				fmt.Fprintf(&b, "  ✅ `%s` (%s)\n", r.name, r.duration.Round(time.Millisecond))
			}
		}
	}
	d.postOK(b.String())

	// Local one-line summary for log scrapers.
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
	monitorCmd.Flags().Duration("interval", 12*time.Hour, "tick cadence (e.g. 1h, 30m, 12h)")
	monitorCmd.Flags().Bool("once", false, "run a single tick and exit")
	monitorCmd.Flags().Uint64("epoch", 0, "pin checks to this epoch (default: head-3 each tick)")
	monitorCmd.Flags().Uint64("tolerance", 10000, "wei tolerance forwarded to each child check")
	rootCmd.AddCommand(monitorCmd)
}
