package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newPoolMetricsCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare the metrics from the API and the node at height",
		Args:  cobra.NoArgs,
		Run:   runPoolMetrics,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch")
	cmd.Flags().Bool("miner-count", false, "Check miner count (slow)")
	// Default tolerance: 10,000 wei = 1e-14 FIL.
	// The per-agent calculate_interest does mulwad/divwad with bigint
	// truncation; summed across ~100 active agents, the rounding floor
	// is empirically ~6,335 wei. 10k wei gives a comfortable margin
	// without hiding a real divergence (the smallest meaningful unit
	// of interest accrual at h+1 is ~1.5e18 wei = 1.5 FIL).
	cmd.Flags().Uint64("tolerance", 10000,
		"treat |API - node| <= tolerance wei as match (per-agent rounding floor; default 10000 = 1e-14 FIL)")
	return cmd
}

func runPoolMetrics(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	chainID := viper.GetUint64("chain_id")
	eventsURL := viper.GetString("events_api")

	fmt.Printf("ChainID: %v\n", chainID)
	fmt.Printf("Events URL: %v\n", eventsURL)

	err := initSingleton(ctx)
	if err != nil {
		log.Fatal(err)
	}

	epoch, err := cmd.Flags().GetUint64("epoch")
	if err != nil {
		log.Fatal(err)
	}

	if epoch == 0 {
		epoch, err = getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		epoch = epoch - 3
	}

	checkMinerCount, err := cmd.Flags().GetBool("miner-count")
	if err != nil {
		log.Fatal(err)
	}

	toleranceUint, err := cmd.Flags().GetUint64("tolerance")
	if err != nil {
		log.Fatal(err)
	}
	tolerance := new(big.Int).SetUint64(toleranceUint)

	metricsFromAPI, err := invariants.GetMetricsFromAPIAtHeight(ctx, eventsURL, epoch)
	if err != nil {
		log.Fatal(err)
	}
	metricsFromNode, resultEpoch, err := invariants.GetMetricsFromNode(ctx, epoch)
	if err != nil {
		log.Fatal(err)
	}
	var minerCountFromNode uint64
	if checkMinerCount {
		minerCountFromNode, resultEpoch, err = invariants.GetMinerCountFromNode(ctx, epoch)
		if err != nil {
			log.Fatal(err)
		}
	}

	fail := false

	taDiff := new(big.Int).Sub(metricsFromAPI.PoolTotalAssets, metricsFromNode.PoolTotalAssets)
	if new(big.Int).Abs(taDiff).Cmp(tolerance) <= 0 {
		if taDiff.Sign() == 0 {
			fmt.Printf("@%d: Success, pool total assets matches: %v\n", epoch, metricsFromAPI.PoolTotalAssets)
		} else {
			fmt.Printf("@%d: Success (within tolerance), pool total assets diff = %v wei (tolerance=%v)\n",
				epoch, taDiff, tolerance)
		}
	} else {
		fmt.Printf("@%d: Error, pool total assets from REST API doesn't match node (diff %v wei > tolerance %v).\n",
			epoch, taDiff, tolerance)
		fmt.Printf("  Node @%d: %v\n", resultEpoch, metricsFromNode.PoolTotalAssets)
		fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.PoolTotalAssets)
		printTotalAssetsDecomposition(metricsFromAPI, metricsFromNode, tolerance)
		fail = true
	}

	if metricsFromAPI.PoolTotalBorrowed.Cmp(metricsFromNode.PoolTotalBorrowed) == 0 {
		fmt.Printf("@%d: Success, pool total borrowed matches: %v\n", epoch, metricsFromAPI.PoolTotalBorrowed)
	} else {
		fmt.Printf("@%d: Error, pool total borrowed from REST API doesn't match node.\n", epoch)
		fmt.Printf("  Node @%d: %v\n", resultEpoch, metricsFromNode.PoolTotalBorrowed)
		fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.PoolTotalBorrowed)
		fail = true
	}

	if metricsFromAPI.TotalAgentCount == metricsFromNode.TotalAgentCount {
		fmt.Printf("@%d: Success, agent count matches: %v\n", epoch, metricsFromAPI.TotalAgentCount)
	} else {
		fmt.Printf("@%d: Error, agent count from REST API doesn't match node.\n", epoch)
		fmt.Printf("  Node @%d: %v\n", resultEpoch, metricsFromNode.TotalAgentCount)
		fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.TotalAgentCount)
		fail = true
	}

	if checkMinerCount {
		if metricsFromAPI.TotalMinersCount == minerCountFromNode {
			fmt.Printf("@%d: Success, miner count matches: %v\n", epoch, minerCountFromNode)
		} else {
			fmt.Printf("@%d: Error, miner count from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node @%d: %v\n", resultEpoch, minerCountFromNode)
			fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.TotalMinersCount)
			fail = true
		}
	}

	if fail {
		log.Fatal("FAIL: Metrics tests had errors.")
	}
}

func init() {
	flat := newPoolMetricsCmd("metrics [--epoch <epoch>]")
	flat.Deprecated = "use `inv pool metrics` instead"
	rootCmd.AddCommand(flat)
	poolCmd.AddCommand(newPoolMetricsCmd("metrics [--epoch <epoch>]"))
	poolCmd.AddCommand(newPoolMetricsBisectCmd())
}

// newPoolMetricsBisectCmd: binary-search across heights for the FIRST height
// where a chosen totalAssets component diverges between API and node. Useful
// when `inv pool metrics` shows a current-day drift and we want to find the
// event that introduced it.
func newPoolMetricsBisectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bisect [--lo <epoch>] [--hi <epoch>] [--component <name>]",
		Short: "Binary-search for the first epoch where a metrics component diverges",
		Args:  cobra.NoArgs,
		Run:   runPoolMetricsBisect,
	}
	cmd.Flags().Uint64("lo", 4215971, "lowest epoch to consider (default: v2 deploy)")
	cmd.Flags().Uint64("hi", 0, "highest epoch (default: API last_processed_height - 3)")
	cmd.Flags().String("component", "totalAssets",
		"which component to bisect: totalAssets|totalBorrowed|liquidAssets|accruedInterest|grossUnpaid|treasuryFeesOwed")
	cmd.Flags().Uint64("tolerance", 0, "treat |api-node| <= tolerance wei as a match")
	return cmd
}

func runPoolMetricsBisect(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	chainID := viper.GetUint64("chain_id")
	eventsURL := viper.GetString("events_api")
	fmt.Printf("ChainID: %v\n", chainID)
	fmt.Printf("Events URL: %v\n", eventsURL)
	if err := initSingleton(ctx); err != nil {
		log.Fatal(err)
	}

	lo, _ := cmd.Flags().GetUint64("lo")
	hi, _ := cmd.Flags().GetUint64("hi")
	component, _ := cmd.Flags().GetString("component")
	tolerance, _ := cmd.Flags().GetUint64("tolerance")

	if hi == 0 {
		head, err := getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		hi = head - 3
	}
	if lo >= hi {
		log.Fatalf("lo (%d) must be < hi (%d)", lo, hi)
	}
	tol := new(big.Int).SetUint64(tolerance)

	// Verify endpoints behave as expected: lo should match, hi should diverge.
	loDiverges, err := componentDiverges(ctx, eventsURL, lo, component, tol)
	if err != nil {
		log.Fatalf("read at lo=%d: %v", lo, err)
	}
	hiDiverges, err := componentDiverges(ctx, eventsURL, hi, component, tol)
	if err != nil {
		log.Fatalf("read at hi=%d: %v", hi, err)
	}
	fmt.Printf("@%d (%s): %s\n", lo, component, ternary(loDiverges, "DIVERGES", "matches"))
	fmt.Printf("@%d (%s): %s\n", hi, component, ternary(hiDiverges, "DIVERGES", "matches"))

	if loDiverges {
		fmt.Println("lo already diverges — extend --lo lower or fix earlier divergence first")
		return
	}
	if !hiDiverges {
		fmt.Println("hi matches — nothing to bisect (either issue self-healed or component is clean)")
		return
	}

	// Standard lower_bound bisect: find smallest h in (lo,hi] where it diverges.
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		d, err := componentDiverges(ctx, eventsURL, mid, component, tol)
		if err != nil {
			log.Fatalf("read at mid=%d: %v", mid, err)
		}
		if d {
			fmt.Printf("@%d: DIVERGES — narrowing right half\n", mid)
			hi = mid
		} else {
			fmt.Printf("@%d: matches — narrowing left half\n", mid)
			lo = mid
		}
	}
	fmt.Printf("\nFirst divergence at height %d (component=%s).\n", hi, component)
	fmt.Println("Inspect events around this height for the offending state change:")
	fmt.Printf("  SELECT type, agent_id, owner, idx, (amount/1e18)::NUMERIC(20,3) AS amt_fil, (principal/1e18)::NUMERIC(20,3) AS p_fil\n")
	fmt.Printf("  FROM tx WHERE height BETWEEN %d AND %d ORDER BY height, idx;\n", hi-5, hi+5)
}

// componentDiverges fetches API + node metrics at a height and returns true
// when the requested component differs by more than `tol` wei.
func componentDiverges(ctx context.Context, eventsURL string, epoch uint64, component string, tol *big.Int) (bool, error) {
	api, err := invariants.GetMetricsFromAPIAtHeight(ctx, eventsURL, epoch)
	if err != nil {
		return false, fmt.Errorf("API: %w", err)
	}
	node, _, err := invariants.GetMetricsFromNode(ctx, epoch)
	if err != nil {
		return false, fmt.Errorf("node: %w", err)
	}
	var apiVal, nodeVal *big.Int
	switch component {
	case "totalAssets":
		apiVal, nodeVal = api.PoolTotalAssets, node.PoolTotalAssets
	case "totalBorrowed":
		apiVal, nodeVal = api.PoolTotalBorrowed, node.PoolTotalBorrowed
	case "liquidAssets":
		apiVal, nodeVal = api.PoolLiquidAssets, node.PoolLiquidAssets
	case "accruedInterest":
		apiVal, nodeVal = api.PoolAccruedInterest, node.PoolAccruedInterest
	case "grossUnpaid":
		// API doesn't expose this directly; fall back to totalAssets.
		return false, fmt.Errorf("grossUnpaid not exposed by API; bisect on totalAssets or accruedInterest instead")
	case "treasuryFeesOwed":
		return false, fmt.Errorf("treasuryFeesOwed not exposed by API; bisect on accruedInterest instead")
	default:
		return false, fmt.Errorf("unknown component %q", component)
	}
	if apiVal == nil || nodeVal == nil {
		return false, fmt.Errorf("nil component value at h=%d", epoch)
	}
	diff := new(big.Int).Sub(apiVal, nodeVal)
	abs := new(big.Int).Abs(diff)
	return abs.Cmp(tol) > 0, nil
}

func ternary(b bool, a, c string) string {
	if b {
		return a
	}
	return c
}

// printTotalAssetsDecomposition is called when API.PoolTotalAssets diverges
// from node.PoolTotalAssets. Decomposes the contract identity:
//
//	totalAssets = liquid + totalBorrowed
//	            + (lp.accrued - lp.paid)         (= grossUnpaidInterest)
//	            - treasuryFeesOwed
//	            = locked() + interest()          (in the API's DB)
//
// API.PoolAccruedInterest is the SQL `interest(h)` result (= gross - treasury).
// API.PoolLiquidAssets is derived as totalAssets - accruedInterest - totalBorrowed,
// which equals our `locked() - totalBorrowed`.
//
// Comparing each component points at the offending one:
//   - liquidAssets mismatch  →  locked() / debit_view / harvest() drift
//   - totalBorrowed mismatch →  agent_tx principal sum drift
//   - accruedInterest mismatch → interest(h) drift
//                                (sub-decompose into gross vs treasury_fees_owed)
func printTotalAssetsDecomposition(api, node *invariants.MetricsResult, tolerance *big.Int) {
	fmt.Println("  Component decomposition (API vs node, both at same height):")
	printCmp("    totalBorrowed     ", api.PoolTotalBorrowed, node.PoolTotalBorrowed, tolerance)
	printCmp("    liquidAssets      ", api.PoolLiquidAssets, node.PoolLiquidAssets, tolerance)
	printCmp("    accruedInterest   ", api.PoolAccruedInterest, node.PoolAccruedInterest, tolerance)
	// API doesn't expose grossUnpaid or treasuryFeesOwed, so we can only
	// print the node-side values for context — useful for narrowing where in
	// interest(h) the drift is.
	intDiff := new(big.Int).Sub(api.PoolAccruedInterest, node.PoolAccruedInterest)
	if new(big.Int).Abs(intDiff).Cmp(tolerance) > 0 {
		fmt.Println("    interest(h) sub-decomposition (node-side only — API doesn't expose):")
		fmt.Printf("      node.grossUnpaid       = %v\n", node.PoolGrossUnpaidInterest)
		fmt.Printf("      node.treasuryFeesOwed  = %v\n", node.PoolTreasuryFeesOwed)
		fmt.Println("      Compare against the API DB:")
		fmt.Println("        SELECT (WITH p AS (SELECT agent_id, sum(principal) AS o FROM agent_tx WHERE height<=H GROUP BY agent_id),")
		fmt.Println("                     c AS (SELECT DISTINCT ON (agent_id) agent_id, epochs_paid FROM agent_cursor WHERE height<=H ORDER BY agent_id, height DESC)")
		fmt.Println("                SELECT COALESCE(sum(CASE WHEN p.o IS NULL OR p.o=0 THEN 0 WHEN c.epochs_paid IS NULL OR c.epochs_paid=0 OR c.epochs_paid>=H THEN 0")
		fmt.Println("                                         ELSE calculate_interest(H, rate(), p.o, c.epochs_paid) END),0)")
		fmt.Println("                FROM p LEFT JOIN c ON c.agent_id=p.agent_id) AS db_grossUnpaid;")
		fmt.Println("        SELECT treasury_fees_owed FROM accounting WHERE height<=H ORDER BY height DESC, idx DESC LIMIT 1;")
	}
}

func printCmp(label string, api, node, tolerance *big.Int) {
	if api == nil || node == nil {
		fmt.Printf("%s skipped (one side is nil)\n", label)
		return
	}
	if api.Cmp(node) == 0 {
		fmt.Printf("%s match: %v\n", label, api)
		return
	}
	diff := new(big.Int).Sub(api, node)
	if new(big.Int).Abs(diff).Cmp(tolerance) <= 0 {
		fmt.Printf("%s match (within tolerance): diff %v wei\n", label, diff)
		return
	}
	fmt.Printf("%s MISMATCH (api - node = %v wei, tolerance %v)\n", label, diff, tolerance)
	fmt.Printf("%s   API : %v\n", label, api)
	fmt.Printf("%s   node: %v\n", label, node)
}
