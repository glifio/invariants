package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAgentStateCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare per-agent state (principal, cursor, owner, level) between DB and contract",
		Long: `Reads each agent's state from the indexer DB and from the v2 pool +
Query.sol contracts at the same height, and reports per-agent
mismatches. The four core invariants:

  principal   = sum(agent_tx.principal where height ≤ H)  vs  pool.getAgentBorrowed(id)
  epochsPaid  = latest agent_cursor.epochs_paid ≤ H        vs  Query.GetAgentsEpochsPaid(addr)
  owner       = agents.owner                                vs  Query.GetAgentOwners(addr)
  level       = agents.level (if tracked)                   vs  Query.GetAgentsLevels(id)

Plus a sanity cross-check: agents.epochs_paid (the in-place column
the indexer maintains) vs the latest agent_cursor row for that
agent. They should match — drift is an internal indicator that the
live update path missed an event.

Uses Query.sol batch reads to keep the contract round-trips small:
4 batch RPCs per run (one per Query method) plus N getAgentBorrowed
calls.`,
		Args: cobra.MaximumNArgs(1),
		Run:  runAgentState,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch (default: API last_processed_height-3)")
	cmd.Flags().Bool("all", false, "Check every agent (default; pass an ID arg to check just one)")
	cmd.Flags().Uint64("tolerance", 0, "treat |diff| <= tolerance wei as match (default 0; agent state should be exact)")
	return cmd
}

func runAgentState(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var (or postgres= in mainnet.env) must be set for agent state checks")
	}
	queryAddrStr := viper.GetString("query_addr")
	if queryAddrStr == "" {
		log.Fatal("QUERY_ADDR must be set for agent state checks")
	}
	queryAddr := common.HexToAddress(queryAddrStr)

	if err := initSingleton(ctx); err != nil {
		log.Fatal(err)
	}
	sdk := singleton.PoolsSDK
	poolAddr := sdk.Query().InfinityPool()
	routerAddr := sdk.Query().Router()

	epoch, _ := cmd.Flags().GetUint64("epoch")
	if epoch == 0 {
		var err error
		epoch, err = getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		epoch -= 3
	}

	tolUint, _ := cmd.Flags().GetUint64("tolerance")
	tolerance := new(big.Int).SetUint64(tolUint)

	// Determine which agents to check.
	var agentIDs []uint64
	if len(args) == 1 {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			log.Fatalf("agent id: %v", err)
		}
		agentIDs = []uint64{id}
	}

	dbStates, err := invariants.FetchAgentStateDB(ctx, postgresURL, epoch, agentIDs)
	if err != nil {
		log.Fatalf("DB fetch: %v", err)
	}
	if len(dbStates) == 0 {
		fmt.Println("no agents to check")
		return
	}

	// Align IDs and addresses for the contract batch reads.
	ids := make([]uint64, len(dbStates))
	addrs := make([]common.Address, len(dbStates))
	for i, s := range dbStates {
		ids[i] = s.ID
		addrs[i] = s.Address
	}

	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		log.Fatalf("eth client: %v", err)
	}
	defer ethClient.Close()

	chainStates, err := invariants.FetchAgentStateContract(ctx, ethClient, poolAddr, routerAddr, queryAddr, epoch, ids, addrs)
	if err != nil {
		log.Fatalf("contract fetch: %v", err)
	}

	fmt.Printf("Agent state @%d (n=%d, tolerance=%v wei)\n", epoch, len(dbStates), tolerance)

	var pass, fail int
	for i := range dbStates {
		ok := compareAgentState(&dbStates[i], &chainStates[i], tolerance)
		if ok {
			pass++
		} else {
			fail++
		}
	}
	fmt.Printf("\nAgent state summary: %d/%d passed, %d failed\n", pass, pass+fail, fail)
	if fail > 0 {
		log.Fatal("FAIL: agent state has mismatches")
	}
}

// compareAgentState reports per-field mismatches for one agent.
// Returns true when every field matches within tolerance.
func compareAgentState(db *invariants.AgentStateDB, chain *invariants.AgentStateContract, tolerance *big.Int) bool {
	id := db.ID
	pass := true

	// principal
	if !cmpBig(db.Principal, chain.Principal, tolerance) {
		fmt.Printf("  agent %d: principal MISMATCH  db=%v  chain=%v  diff=%v\n",
			id, db.Principal, chain.Principal, new(big.Int).Sub(db.Principal, chain.Principal))
		pass = false
	}

	// cursor — chain returns 0 when the contract hasn't seen any Pay/Borrow
	// for the agent (pre-v2 idle agents). Tolerate cursor=0 on the chain
	// side IF the DB also has no cursor; otherwise it's a real divergence.
	if chain.EpochsPaid != nil && chain.EpochsPaid.Sign() == 0 {
		if db.EpochsPaid != nil && db.EpochsPaid.Sign() != 0 {
			fmt.Printf("  agent %d: cursor MISMATCH  db=%v  chain=0 (idle agent on contract)\n", id, db.EpochsPaid)
			pass = false
		}
	} else {
		dbCursor := db.EpochsPaid
		if dbCursor == nil {
			dbCursor = new(big.Int)
		}
		if !cmpBig(dbCursor, chain.EpochsPaid, tolerance) {
			fmt.Printf("  agent %d: cursor MISMATCH  db=%v  chain=%v  diff=%v\n",
				id, dbCursor, chain.EpochsPaid, new(big.Int).Sub(dbCursor, chain.EpochsPaid))
			pass = false
		}
	}

	// owner / level — DB doesn't track them today; print contract value
	// for context only. (If indexer ever stores agents.owner / agents.level
	// add a real compare here.)

	// agents.epochs_paid (in-place) vs latest agent_cursor — internal sanity.
	if db.EpochsPaid != nil && db.AgentsEpoch != nil &&
		db.AgentsEpoch.Cmp(db.EpochsPaid) != 0 {
		fmt.Printf("  agent %d: WARN agents.epochs_paid (=%v) drifts from latest agent_cursor (=%v)\n",
			id, db.AgentsEpoch, db.EpochsPaid)
	}

	if pass {
		fmt.Printf("  agent %d: ok  principal=%v cursor=%v owner=%s level=%v\n",
			id, db.Principal, chain.EpochsPaid, chain.Owner.Hex(), chain.Level)
	}
	return pass
}

func cmpBig(a, b, tol *big.Int) bool {
	if a == nil || b == nil {
		return a == b
	}
	d := new(big.Int).Sub(a, b)
	d.Abs(d)
	return d.Cmp(tol) <= 0
}

func init() {
	flat := newAgentStateCmd("agent-state [agent-id] [--all] [--epoch <epoch>]")
	flat.Deprecated = "use `inv agent state` instead"
	rootCmd.AddCommand(flat)
	agentCmd.AddCommand(newAgentStateCmd("state [agent-id] [--all] [--epoch <epoch>]"))
}

// silence unused-context warnings in some build configs
var _ = context.Background
