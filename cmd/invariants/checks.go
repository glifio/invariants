package main

import "fmt"

// checkSpec is one subcommand invocation that the aggregate gate (`inv all`)
// and the long-running monitor daemon (`inv monitor`) both shell out to.
// `name` is the human-readable label that appears in the rollup; `args`
// is the argv passed to the same binary (`os.Args[0]`). The child must
// exit non-zero on failure — that's what makes the aggregate gate work.
type checkSpec struct {
	name string
	args []string
}

// Checks returns the canonical list of invariant subcommands that count
// toward the binary "DB consistent with chain at H" verdict. Both
// `inv all` and `inv monitor` consume this single list — adding a new
// check here is the only change needed to surface it in both surfaces.
//
// epoch and tolerance are threaded through as common flags so the
// aggregate runs at a single pinned epoch with the operator's chosen
// tolerance; both can be left at zero to use child defaults.
//
// Requirement for new entries: the subcommand MUST exit non-zero on a
// real failure (log.Fatal / os.Exit / cobra's RunE returning error).
// The aggregate maps process exit code → check result; a check that
// prints "FAIL" but exits 0 will silently pass.
func Checks(epoch, tolerance uint64) []checkSpec {
	common := []string{}
	if epoch != 0 {
		common = append(common, "--epoch", fmt.Sprintf("%d", epoch))
	}
	tolStr := fmt.Sprintf("%d", tolerance)

	return []checkSpec{
		{"pool metrics", append([]string{"pool", "metrics", "--tolerance", tolStr}, common...)},
		{"agent state", append([]string{"agent", "state", "--tolerance", tolStr}, common...)},
		{"agent balances", append([]string{"agent", "balances", "--all"}, common...)},
		{"pool lpplus", append([]string{"pool", "lpplus"}, common...)},
		{"pool spplus", append([]string{"pool", "spplus"}, common...)},
		{"agent dtl", append([]string{"agent", "dtl", "--all"}, common...)},
		// iFIL + GLF: validate the per-holder reconstruction against
		// chain truth via the Query contract. Both exit non-zero via
		// log.Fatalf("FAIL: ...") when the total supply or any
		// per-holder balance disagrees. Slow-ish on prod (~7-8k iFIL
		// holders, fewer GLF) but well under the monitor's default
		// interval, so a 1h tick has plenty of headroom.
		{"pool ifil", append([]string{"pool", "ifil", "--per-depositor"}, common...)},
		{"pool glf", append([]string{"pool", "glf", "--per-holder"}, common...)},
	}
}
