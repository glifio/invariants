package invariants

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/glifio/invariants/abigen"
	_ "github.com/lib/pq"
)

// AgentStateDB is the per-agent state we hold in the indexer DB.
type AgentStateDB struct {
	ID          uint64
	Address     common.Address
	Principal   *big.Int // sum(agent_tx.principal where height ≤ H)
	EpochsPaid  *big.Int // latest agent_cursor.epochs_paid where height ≤ H
	AgentsEpoch *big.Int // agents.epochs_paid (the in-place column, for self-consistency check)
	Interest    *big.Int // calculate_interest(h, rate(), principal, epochs_paid) — DB-derived
}

// AgentStateContract is the contract's view at the same height.
type AgentStateContract struct {
	ID                  uint64
	Address             common.Address
	Principal           *big.Int // pool.getAgentBorrowed(id)
	EpochsPaid          *big.Int // Query.GetAgentsEpochsPaid([addr])[0]
	GetAgentInterestOwed *big.Int // Query.GetAgentInterestOwed([id])[0]
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
		// DB-derived interest at h: calculate_interest(h, rate(), principal,
		// epochs_paid). Mirrors the per-agent term inside interest(h).
		var interestStr sql.NullString
		err = db.QueryRowContext(ctx, `
			SELECT COALESCE(calculate_interest($1::int, rate(),
				(SELECT COALESCE(sum(principal),0) FROM agent_tx WHERE agent_id = $2 AND height <= $1::int),
				(SELECT epochs_paid FROM agent_cursor WHERE agent_id = $2 AND height <= $1::int ORDER BY height DESC, idx DESC LIMIT 1)
			), 0)::TEXT`,
			h, id,
		).Scan(&interestStr)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("agent %d interest: %w", id, err)
		}

		s := AgentStateDB{
			ID:        id,
			Address:   common.HexToAddress(addrStr),
			Principal: bigOrZero(principalStr),
		}
		s.EpochsPaid = bigOrNil(cursorStr)
		s.AgentsEpoch = bigOrNil(agentsEpoch)
		s.Interest = bigOrZero(interestStr)
		out = append(out, s)
	}
	return out, nil
}

// FetchAgentStateContract reads contract-truth state for each agent at h
// via the InvariantsQuery batch helper — one chain RPC for all agents.
//
// `invQueryAddr` must be the deployed InvariantsQuery contract; see the
// `contracts/` directory and INVARIANTS_QUERY_ADDR in mainnet.env.
func FetchAgentStateContract(
	ctx context.Context,
	ethClient *ethclient.Client,
	invQueryAddr common.Address,
	height uint64,
	agentIDs []uint64,
	addresses []common.Address,
) ([]AgentStateContract, error) {
	if len(agentIDs) != len(addresses) {
		return nil, fmt.Errorf("agentIDs and addresses must align (got %d, %d)", len(agentIDs), len(addresses))
	}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(height)}

	q, err := abigen.NewInvariantsQueryCaller(invQueryAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("invariants query caller: %w", err)
	}
	idsBig := make([]*big.Int, len(agentIDs))
	for i, id := range agentIDs {
		idsBig[i] = new(big.Int).SetUint64(id)
	}
	states, err := q.GetAgentsState(opts, idsBig)
	if err != nil {
		return nil, fmt.Errorf("InvariantsQuery.GetAgentsState: %w", err)
	}
	if len(states) != len(agentIDs) {
		return nil, fmt.Errorf("InvariantsQuery returned %d states for %d agents", len(states), len(agentIDs))
	}
	out := make([]AgentStateContract, len(agentIDs))
	for i, id := range agentIDs {
		out[i] = AgentStateContract{
			ID:                   id,
			Address:              addresses[i],
			Principal:            states[i].Principal,
			EpochsPaid:           states[i].EpochsPaid,
			GetAgentInterestOwed: states[i].InterestOwed,
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
