package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// agentCmd is the parent of every agent-level invariant check.
//
// Without a subcommand it runs every child check in sequence with
// `--all` set so an operator can see the whole agent picture in one
// invocation. With a subcommand (`inv agent balances`, `inv agent
// econ`, ...) it behaves like cobra normally would and just runs that
// child.
//
// The flat names (`inv agent-balances`, `inv agent-econ`) are kept
// alongside as deprecated aliases for one or two releases so existing
// scripts keep working.
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent-level invariant checks",
	Long: `Run every agent-level invariant check in sequence, or invoke a
specific check via a subcommand:

  inv agent              # all agent checks, --all mode
  inv agent balances 97  # available-balance check for agent 97
  inv agent econ --all   # debt + LV for every agent`,
	Run: runAgentAll,
}

// runAgentAll execs each child as a subprocess so a log.Fatal in one
// check doesn't stop the others. Children inherit stdout/stderr so
// output streams live.
func runAgentAll(cmd *cobra.Command, _ []string) {
	var failed []string
	for _, child := range cmd.Commands() {
		if child.Name() == "help" {
			continue
		}
		fmt.Printf("\n=== inv agent %s ===\n", child.Name())
		args := []string{"agent", child.Name()}
		if f := child.Flags().Lookup("all"); f != nil && !f.Changed {
			args = append(args, "--all")
		}
		c := exec.CommandContext(cmd.Context(), os.Args[0], args...)
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
	rootCmd.AddCommand(agentCmd)
}
