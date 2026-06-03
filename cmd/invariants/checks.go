package main

import (
	"context"
	"fmt"
)

// checkSpec is one subcommand invocation that the aggregate gate (`inv all`)
// and the long-running monitor daemon (`inv monitor`) both shell out to.
// `name` is the human-readable label that appears in the rollup; `args`
// is the argv passed to the same binary (`os.Args[0]`). The child must
// exit non-zero on failure — that's what makes the aggregate gate work.
type checkSpec struct {
	name string
	args []string
}

// DefaultEpochOffset is how far back from chain head we sample by default.
// Filecoin reorgs deeper than ~1 are vanishingly rare; head-3 leaves at
// least one epoch of margin past finalization while keeping the check
// close enough to head that the watcher's DB is realistically caught up.
// Matches what every child check already used as its own per-process
// default before we pinned at the aggregate.
const DefaultEpochOffset uint64 = 3

// Checks returns the canonical list of invariant subcommands that count
// toward the binary "DB consistent with chain at H" verdict. Both
// `inv all` and `inv monitor` consume this single list — adding a new
// check here is the only change needed to surface it in both surfaces.
//
// epoch and tolerance are forwarded to every child. epoch=0 means "pick
// head - DefaultEpochOffset NOW and pin every child to it", which:
//
//   - Issues ONE getHeadEpoch RPC per tick instead of one per child.
//     Children that previously called getHeadEpoch themselves see the
//     forced --epoch flag and skip their own resolution.
//   - Forces every check to read the SAME H. The aggregate's verdict
//     becomes a coherent "DB consistent with chain at this exact H"
//     instead of "8 checks sampled at near-but-not-identical heights"
//     — failures become cross-correlatable (metrics drift at H and
//     ifil drift at H almost certainly share a root cause).
//   - Sidesteps the per-child default divergence (pool ifil used to
//     pick head-2 while everyone else used head-3; agent balances
//     used the API watermark instead of head-N at all).
//
// Requirement for new entries: the subcommand MUST exit non-zero on a
// real failure (log.Fatal / os.Exit / cobra's RunE returning error).
// The aggregate maps process exit code → check result; a check that
// prints "FAIL" but exits 0 will silently pass.
func Checks(ctx context.Context, epoch, tolerance uint64) ([]checkSpec, error) {
	if epoch == 0 {
		head, err := getHeadEpoch(ctx)
		if err != nil {
			return nil, fmt.Errorf("Checks: resolve head: %w", err)
		}
		if head <= DefaultEpochOffset {
			return nil, fmt.Errorf("Checks: chain head %d < DefaultEpochOffset %d", head, DefaultEpochOffset)
		}
		epoch = head - DefaultEpochOffset
	}
	common := []string{"--epoch", fmt.Sprintf("%d", epoch)}
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
	}, nil
}
