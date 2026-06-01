package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// poolCmd is the parent of every pool-level invariant check.
//
// Without a subcommand it runs every child check in sequence.
// With a subcommand (`inv pool metrics`, `inv pool ifil`, ...) it
// behaves like cobra normally would.
//
// Flat aliases (`inv metrics`, `inv ifil-total-supply`) stay as
// deprecated entry points for one or two releases.
var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Pool-level invariant checks",
	Long: `Run every pool-level invariant check in sequence, or invoke a
specific check via a subcommand:

  inv pool         # all pool checks
  inv pool metrics # totalAssets / totalBorrowed / agentCount
  inv pool ifil    # iFIL total supply`,
	Run: runPoolAll,
}

// runPoolAll execs each child as a subprocess so a log.Fatal in one
// check doesn't stop the others.
func runPoolAll(cmd *cobra.Command, _ []string) {
	var failed []string
	for _, child := range cmd.Commands() {
		if child.Name() == "help" {
			continue
		}
		fmt.Printf("\n=== inv pool %s ===\n", child.Name())
		c := exec.CommandContext(cmd.Context(), os.Args[0], "pool", child.Name())
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		if err := c.Run(); err != nil {
			failed = append(failed, child.Name())
		}
	}
	if len(failed) > 0 {
		fmt.Printf("\n=== FAILED: %v ===\n", failed)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(poolCmd)
}
