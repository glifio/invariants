package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// allCmd runs every "binary-result" invariant check against the same
// epoch and prints one summary report. The intent is to give a single
// thumbs-up or thumbs-down on whether the indexer DB is consistent
// with chain state at H — once it passes, the result is permanent
// because both the chain state at H and the DB events through H are
// immutable.
//
// Each check is exec'd as a subprocess so a fatal in one doesn't take
// down the others. Children inherit stdout/stderr so output streams
// live; this command also captures the exit code per child to build
// the report.
var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run every invariant check at one epoch and produce a binary DB-vs-chain report",
	Long: `Runs each invariant check sequentially at the same epoch and
prints a single pass/fail report. Currently includes:

  pool metrics    pool.totalAssets identity (totalBorrowed, liquid,
                  accruedInterest, treasuryFeesOwed)
  agent state     per-agent principal + cursor (Router.GetAccount)

When ifil/glf/lpplus/spplus/hedgey checks are added they slot in
here.

The exit code is non-zero if ANY child check fails; 0 only if every
check passes within tolerance. That makes this command suitable as
the gate for "DB confirmed consistent through H" automation.`,
	Args: cobra.NoArgs,
	Run:  runAll,
}

type checkResult struct {
	name     string
	pass     bool
	duration time.Duration
}

// runAll: list of (subcommand args) to run. Add new checks here as
// invariants land. Each entry should produce a binary result and
// print enough detail on failure that an operator can act on it.
func runAll(cmd *cobra.Command, _ []string) {
	epoch, _ := cmd.Flags().GetUint64("epoch")
	tolerance, _ := cmd.Flags().GetUint64("tolerance")

	type child struct {
		name string
		args []string
	}
	var children []child
	common := []string{}
	if epoch != 0 {
		common = append(common, "--epoch", fmt.Sprintf("%d", epoch))
	}
	tolStr := fmt.Sprintf("%d", tolerance)
	children = []child{
		{"pool metrics", append([]string{"pool", "metrics", "--tolerance", tolStr}, common...)},
		{"agent state", append([]string{"agent", "state", "--tolerance", tolStr}, common...)},
		{"pool lpplus", append([]string{"pool", "lpplus"}, common...)},
		{"pool spplus", append([]string{"pool", "spplus"}, common...)},
		{"agent dtl", append([]string{"agent", "dtl", "--all"}, common...)},
		// TODO: add `pool ifil --per-depositor`, `pool glf --per-holder`,
		// `hedgey`, `miners` once each is wired with a binary pass/fail
		// suitable for the aggregate gate.
	}

	results := make([]checkResult, 0, len(children))
	for _, ch := range children {
		fmt.Printf("\n=== %s (args: %s) ===\n", ch.name, strings.Join(ch.args, " "))
		start := time.Now()
		c := exec.CommandContext(cmd.Context(), os.Args[0], ch.args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		err := c.Run()
		results = append(results, checkResult{
			name:     ch.name,
			pass:     err == nil,
			duration: time.Since(start),
		})
	}

	// Final report.
	fmt.Println("\n=== inv all: report ===")
	pass := 0
	for _, r := range results {
		status := "FAIL"
		if r.pass {
			status = "PASS"
			pass++
		}
		fmt.Printf("  %-15s %s  (%s)\n", r.name, status, r.duration.Round(time.Millisecond))
	}
	fmt.Printf("\n%d/%d checks passed", pass, len(results))
	if epoch != 0 {
		fmt.Printf(" at epoch %d", epoch)
	}
	fmt.Println()

	if pass != len(results) {
		fmt.Println("\nOne or more checks failed — DB is NOT confirmed consistent with chain.")
		os.Exit(1)
	}
	fmt.Println("\nAll checks passed — DB confirmed consistent with chain.")
}

func init() {
	allCmd.Flags().Uint64("epoch", 0, "Check at epoch (default: API last_processed_height-3)")
	allCmd.Flags().Uint64("tolerance", 10000, "wei tolerance forwarded to each child check")
	rootCmd.AddCommand(allCmd)
}
