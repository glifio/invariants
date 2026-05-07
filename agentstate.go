package invariants

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	pools "github.com/glifio/go-pools/abigen"
	"github.com/glifio/invariants/abigen"
	_ "github.com/lib/pq"
)

// AgentStateDB is the per-agent state we hold in the indexer DB.
//
// The agents table does NOT track owner or level today, so those are
// only fetched from the contract and printed (no DB-vs-chain compare).
// If the indexer schema gets an `owner` and `level` column, fill them
// here and add the comparison.
type AgentStateDB struct {
	ID          uint64
	Address     common.Address
	Principal   *big.Int // sum(agent_tx.principal where height ≤ H)
	EpochsPaid  *big.Int // latest agent_cursor.epochs_paid where height ≤ H
	AgentsEpoch *big.Int // agents.epochs_paid (the in-place column, for self-consistency check)
}

// AgentStateContract is the contract's view at the same height.
type AgentStateContract struct {
	ID                  uint64
	Address             common.Address
	Owner               common.Address
	Principal           *big.Int // pool.getAgentBorrowed(id)
	EpochsPaid          *big.Int // Query.GetAgentsEpochsPaid([addr])[0]
	GetAgentInterestOwed *big.Int // contract gross interest owed (for cross-check)
	Level               *big.Int // Query.GetAgentsLevels([id])[0]
}

// FetchAgentStateDB reads the indexer DB for one or more agents at h.
// Returns one row per agent ID requested.
func FetchAgentStateDB(ctx context.Context, postgresURL string, h uint64, agentIDs []uint64) ([]AgentStateDB, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	defer db.Close()

	if len(agentIDs) == 0 {
		// Default to every agent created at-or-before h.
		rows, err := db.QueryContext(ctx,
			`SELECT id FROM agents WHERE height <= $1 ORDER BY id`, h)
		if err != nil {
			return nil, fmt.Errorf("agents list: %w", err)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			agentIDs = append(agentIDs, uint64(id))
		}
		rows.Close()
	}

	out := make([]AgentStateDB, 0, len(agentIDs))
	for _, id := range agentIDs {
		var addrStr string
		var agentsEpoch sql.NullString
		err := db.QueryRowContext(ctx,
			`SELECT addr, epochs_paid::TEXT FROM agents WHERE id = $1`, id,
		).Scan(&addrStr, &agentsEpoch)
		if err != nil {
			return nil, fmt.Errorf("agent %d row: %w", id, err)
		}
		var principalStr sql.NullString
		err = db.QueryRowContext(ctx,
			`SELECT COALESCE(sum(principal), 0)::TEXT FROM agent_tx WHERE agent_id = $1 AND height <= $2`,
			id, h,
		).Scan(&principalStr)
		if err != nil {
			return nil, fmt.Errorf("agent %d principal: %w", id, err)
		}
		var cursorStr sql.NullString
		err = db.QueryRowContext(ctx,
			`SELECT epochs_paid::TEXT FROM agent_cursor WHERE agent_id = $1 AND height <= $2 ORDER BY height DESC, idx DESC LIMIT 1`,
			id, h,
		).Scan(&cursorStr)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("agent %d cursor: %w", id, err)
		}

		s := AgentStateDB{
			ID:        id,
			Address:   common.HexToAddress(addrStr),
			Principal: bigOrZero(principalStr),
		}
		s.EpochsPaid = bigOrNil(cursorStr)
		s.AgentsEpoch = bigOrNil(agentsEpoch)
		out = append(out, s)
	}
	return out, nil
}

// FetchAgentStateContract reads contract-truth state for each agent at h.
//
// Uses Router.GetAccount(agentID, poolID=0) for principal + cursor + defaulted —
// it's per-agent (not batched) but it's the canonical pool account state and
// avoids the address↔id ambiguity in Query.GetAgentsEpochsPaid.
//
// Uses Query.sol batches for level + owner + interest_owed where they work
// reliably (these are read-only views and don't revert on defaulted agents
// the way GetAgentsEpochsPaid does).
func FetchAgentStateContract(
	ctx context.Context,
	ethClient *ethclient.Client,
	poolAddr, routerAddr, queryAddr common.Address,
	height uint64,
	agentIDs []uint64,
	addresses []common.Address,
) ([]AgentStateContract, error) {
	if len(agentIDs) != len(addresses) {
		return nil, fmt.Errorf("agentIDs and addresses must align (got %d, %d)", len(agentIDs), len(addresses))
	}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(height)}

	// Batches via Query.sol with per-agent fallback on revert.
	q, err := abigen.NewQueryCaller(queryAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("query caller: %w", err)
	}
	ids32 := make([]uint32, len(agentIDs))
	for i, id := range agentIDs {
		ids32[i] = uint32(id)
	}
	levels, err := q.GetAgentsLevels(opts, ids32)
	if err != nil {
		levels = make([]*big.Int, len(ids32))
		for i, id := range ids32 {
			if r, e := q.GetAgentsLevels(opts, []uint32{id}); e == nil {
				levels[i] = r[0]
			}
		}
	}
	interestOwed, err := q.GetAgentInterestOwed(opts, ids32)
	if err != nil {
		interestOwed = make([]*big.Int, len(ids32))
		for i, id := range ids32 {
			if r, e := q.GetAgentInterestOwed(opts, []uint32{id}); e == nil {
				interestOwed[i] = r[0]
			}
		}
	}
	owners := make([]common.Address, len(addresses))
	for i, addr := range addresses {
		if r, e := q.GetAgentOwners(opts, []common.Address{addr}); e == nil {
			owners[i] = r[0]
		}
	}

	// Per-agent canonical account state via Router. principal + cursor +
	// defaulted come from one struct read.
	router, err := pools.NewRouterCaller(routerAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("router caller: %w", err)
	}
	out := make([]AgentStateContract, len(agentIDs))
	for i, id := range agentIDs {
		acct, err := router.GetAccount(opts,
			new(big.Int).SetUint64(id),
			big.NewInt(0)) // poolID 0 = the v2 InfinityPool
		if err != nil {
			return nil, fmt.Errorf("Router.GetAccount(%d): %w", id, err)
		}
		out[i] = AgentStateContract{
			ID:                   id,
			Address:              addresses[i],
			Owner:                owners[i],
			Principal:            acct.Principal,
			EpochsPaid:           acct.EpochsPaid,
			GetAgentInterestOwed: interestOwed[i],
			Level:                levels[i],
		}
	}
	return out, nil
}

func bigOrZero(s sql.NullString) *big.Int {
	if !s.Valid {
		return new(big.Int)
	}
	v, _ := new(big.Int).SetString(s.String, 10)
	if v == nil {
		return new(big.Int)
	}
	return v
}

func bigOrNil(s sql.NullString) *big.Int {
	if !s.Valid {
		return nil
	}
	v, _ := new(big.Int).SetString(s.String, 10)
	return v
}
