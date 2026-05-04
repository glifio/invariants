package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// minerCmd is the parent of every miner-level invariant check.
//
// Without a subcommand it runs every child check in sequence.
// With a subcommand (`inv miner liquidation`, ...) it behaves
// like cobra normally would.
//
// `inv miner-liquidation` stays as a deprecated alias.
var minerCmd = &cobra.Command{
	Use:   "miner",
	Short: "Miner-level invariant checks",
	Long: `Run every miner-level invariant check in sequence, or invoke a
specific check via a subcommand:

  inv miner             # all miner checks
  inv miner liquidation # liquidation calc per-method`,
	Run: runMinerAll,
}

// runMinerAll execs each child as a subprocess so a log.Fatal in one
// check doesn't stop the others.
func runMinerAll(cmd *cobra.Command, _ []string) {
	var failed []string
	for _, child := range cmd.Commands() {
		if child.Name() == "help" {
			continue
		}
		fmt.Printf("\n=== inv miner %s ===\n", child.Name())
		c := exec.CommandContext(cmd.Context(), os.Args[0], "miner", child.Name())
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
	rootCmd.AddCommand(minerCmd)
}
