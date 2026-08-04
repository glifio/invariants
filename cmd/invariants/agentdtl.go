package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/filecoin-project/go-state-types/abi"
	ltypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newAgentDTLCmd: per-agent debt-to-liquidation-value alert.
//
// Computes DTL live from chain on every call (no DB cache). Two batched
// chain calls happen up front per fleet sweep:
//
//   - tipset-globals (smoothed reward, smoothed power, network version)
//     fetched once and reused across every miner compute
//   - InvariantsQuery.getAgentsDTLInputs returns liquidAssets + miner
//     list + principal + interest for every agent in one eth_call
//
// Workers then only do per-miner Lotus state reads (LoadMinerActor,
// StateMinerPower, StateMinerSectorCount), which is the genuinely
// per-miner part of the math.
//
// Severity buckets:
//
//	OK       dtl ≤ max − buffer
//	WARN     max − buffer < dtl ≤ max     (close, monitor)
//	OVER     dtl > max                    (eligible for liquidation)
//	NO-LV    lv == 0 with debt > 0        (zero collateral)
//
// Exit code is non-zero if any agent is OVER or NO-LV. WARN is
// informational and does not change the exit code (so the monitor
// only pages when an agent crosses the actual threshold).
func newAgentDTLCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Per-agent debt-to-liquidation-value alert (chain-fresh)",
		Args:  cobra.MaximumNArgs(1),
		Run:   runAgentDTL,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch (default: head-3)")
	cmd.Flags().Bool("all", false, "Check all agents")
	cmd.Flags().Uint64("buffer-bps", 250, "Warn band: bps below MaxDTL where the agent is flagged but not failed (default 250 = 2.5%)")
	cmd.Flags().Int("concurrency", 0, "parallel agent workers (0/1 = sequential; each agent also fans out per-miner via util.Multiread, so effective Lotus load = workers × miners_per_agent × ~4 RPCs — sequential is the safe default against chain.love's rate limit)")
	cmd.Flags().Bool("show-warn", false, "print per-agent WARN lines (agents inside the buffer band but not yet over max); off by default so the monitor's output stays quiet, since WARN doesn't fail the check")
	return cmd
}

func runAgentDTL(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set for agent dtl checks")
	}

	if err := initSingleton(ctx); err != nil {
		log.Fatal(err)
	}
	sdk := singleton.PoolsSDK

	epoch, _ := cmd.Flags().GetUint64("epoch")
	if epoch == 0 {
		head, err := getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		epoch = head - 3
	}

	allAgents, _ := cmd.Flags().GetBool("all")
	bufferBps, _ := cmd.Flags().GetUint64("buffer-bps")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	showWarn, _ := cmd.Flags().GetBool("show-warn")

	var agentIDs []uint64
	if !allAgents {
		if len(args) != 1 {
			log.Fatal("pass an agent id or use --all")
		}
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			log.Fatalf("agent id: %v", err)
		}
		agentIDs = []uint64{id}
	}

	// Resolve agent ids → addresses from DB. Same source the rest of
	// the suite uses (agents table). When --all is set, agentIDs is
	// empty and FetchAgentStateDB returns every agent.
	dbStates, err := invariants.FetchAgentStateDB(ctx, postgresURL, epoch, agentIDs)
	if err != nil {
		log.Fatalf("DB fetch: %v", err)
	}
	if len(dbStates) == 0 {
		fmt.Println("no agents to check")
		return
	}
	ids := make([]uint64, len(dbStates))
	addrs := make([]common.Address, len(dbStates))
	for i, s := range dbStates {
		ids[i] = s.ID
		addrs[i] = s.Address
	}

	// Singleton Lotus connection — routed through the RPC rate limiter.
	node := singleton.Lotus()
	if node == nil {
		log.Fatal("lotus singleton not initialized")
	}
	ts, err := node.Api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), ltypes.EmptyTSK)
	if err != nil {
		log.Fatalf("ChainGetTipSetByHeight: %v", err)
	}

	fmt.Printf("Agent DTL @%d (n=%d, buffer=%d bps, concurrency=%d)\n",
		epoch, len(ids), bufferBps, concurrency)

	invQueryAddrStr := viper.GetString("invariants_query_addr")
	if invQueryAddrStr == "" {
		log.Fatal("INVARIANTS_QUERY_ADDR must be set in mainnet.env")
	}
	invQueryAddr := common.HexToAddress(invQueryAddrStr)

	dtls, err := invariants.FetchAgentDTLs(ctx, sdk, invQueryAddr, ids, addrs, ts, bufferBps, concurrency)
	if err != nil {
		log.Fatalf("FetchAgentDTLs: %v", err)
	}

	overMax, warn, errs, noLV := 0, 0, 0, 0
	for _, d := range dtls {
		switch d.Severity {
		case invariants.DTLOverMax:
			fmt.Printf("  ❌ agent %d OVER  debt=%s lv=%s dtl=%s%% maxDTL=%s%% (tier %d)\n",
				d.AgentID, fil(d.Debt), fil(d.LV), pct(d.DTL), pct(d.MaxDTL), d.Tier)
			overMax++
		case invariants.DTLWarn:
			if showWarn {
				fmt.Printf("  ⚠ agent %d WARN  debt=%s lv=%s dtl=%s%% maxDTL=%s%% (tier %d)\n",
					d.AgentID, fil(d.Debt), fil(d.LV), pct(d.DTL), pct(d.MaxDTL), d.Tier)
			}
			warn++
		case invariants.DTLNoLV:
			fmt.Printf("  ❌ agent %d NO-LV  debt=%s lv=0 (zero collateral, positive debt)\n",
				d.AgentID, fil(d.Debt))
			noLV++
		case invariants.DTLError:
			fmt.Printf("  ! agent %d error: %v\n", d.AgentID, d.Err)
			errs++
		}
	}
	ok := len(dtls) - overMax - warn - noLV - errs
	fmt.Printf("\nDTL summary: %d ok, %d warn, %d over, %d no-lv, %d errors\n",
		ok, warn, overMax, noLV, errs)

	if overMax+noLV > 0 {
		log.Fatalf("FAIL: %d agents at/over liquidation threshold", overMax+noLV)
	}
}

// pct formats wad-scaled DTL (× 1e18) as a percentage string with 2dp.
func pct(wad interface{}) string {
	switch v := wad.(type) {
	case nil:
		return "?"
	default:
		_ = v
	}
	if w, ok := wad.(interface{ String() string }); ok {
		// 1.0 = 1e18, so percent = wad / 1e16; print integer + 2dp.
		s := w.String()
		// trivial impl: strip last 14 digits to get 4-digit-precision number,
		// then format. Falls back to raw on short strings.
		if len(s) > 16 {
			whole := s[:len(s)-16]
			frac := s[len(s)-16 : len(s)-14]
			return whole + "." + frac
		}
		return s
	}
	return "?"
}

// fil formats wei as FIL with 2dp.
func fil(wei interface{}) string {
	if w, ok := wei.(interface{ String() string }); ok {
		s := w.String()
		neg := false
		if len(s) > 0 && s[0] == '-' {
			neg = true
			s = s[1:]
		}
		var whole, frac string
		if len(s) > 18 {
			whole = s[:len(s)-18]
			frac = s[len(s)-18 : len(s)-16]
		} else {
			whole = "0"
			pad := 18 - len(s)
			padded := s
			for i := 0; i < pad; i++ {
				padded = "0" + padded
			}
			frac = padded[:2]
		}
		out := whole + "." + frac + " FIL"
		if neg {
			out = "-" + out
		}
		return out
	}
	return "?"
}

// silence unused-import linter on database/sql; we don't open a DB
// directly here but the package init pulls in the postgres driver
// transitively via FetchAgentStateDB.
var _ = sql.ErrNoRows

func init() {
	agentCmd.AddCommand(newAgentDTLCmd("dtl [agent-id|--all] [--buffer-bps N]"))
}
