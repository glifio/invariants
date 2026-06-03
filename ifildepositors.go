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

// IFILDepositorBalance is one address's iFIL balance.
type IFILDepositorBalance struct {
	Address common.Address
	DB      *big.Int // sum(transfers in) − sum(transfers out) at h, from ifil table
	Chain   *big.Int // Query.GetDepositorsIFILBalances([addr])[0]
}

// FetchIFILDepositorBalancesFromDB returns the iFIL balance for every
// address that has ever appeared as a transfer recipient (excluding
// 0x0 — that's the mint/burn sink). Returns one row per address with
// the at-h balance.
//
// Computes balance as
//   sum(amount where to_=addr AND height ≤ h) − sum(amount where from_=addr AND height ≤ h).
// Filters to balance != 0 to avoid wasting RPC time on dust holders.
// (Pass `includeZero=true` to keep them — useful when chasing a phantom
// holder that should be at zero but isn't.)
func FetchIFILDepositorBalancesFromDB(ctx context.Context, postgresURL string, h uint64, includeZero bool) (map[common.Address]*big.Int, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	defer db.Close()

	// One pass: per-address net balance from the ifil table.
	q := `
		WITH txs AS (
		    SELECT to_   AS addr, amount AS amt FROM ifil WHERE height <= $1
		    UNION ALL
		    SELECT from_ AS addr, -amount AS amt FROM ifil WHERE height <= $1
		)
		SELECT addr, sum(amt)::TEXT AS balance
		FROM txs
		WHERE addr <> '0x0000000000000000000000000000000000000000'
		GROUP BY addr
	`
	if !includeZero {
		q += "HAVING sum(amt) <> 0"
	}
	rows, err := db.QueryContext(ctx, q, h)
	if err != nil {
		return nil, fmt.Errorf("ifil balances: %w", err)
	}
	defer rows.Close()

	out := make(map[common.Address]*big.Int)
	for rows.Next() {
		var addr, bal string
		if err := rows.Scan(&addr, &bal); err != nil {
			return nil, err
		}
		v, ok := new(big.Int).SetString(bal, 10)
		if !ok {
			return nil, fmt.Errorf("bad balance %q for %s", bal, addr)
		}
		out[common.HexToAddress(addr)] = v
	}
	return out, rows.Err()
}

// FetchIFILDepositorBalancesFromContract reads Query.GetDepositorsIFILBalances
// in chunks (one batch RPC per chunk) for the supplied addresses at h.
// On batch revert (e.g. one bad address poisons the batch) it falls back
// to per-address calls within that chunk so the whole run survives one
// problem holder.
func FetchIFILDepositorBalancesFromContract(
	ctx context.Context,
	ethClient *ethclient.Client,
	queryAddr common.Address,
	height uint64,
	addresses []common.Address,
	chunkSize int,
) (map[common.Address]*big.Int, error) {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	q, err := abigen.NewQueryCaller(queryAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("query caller: %w", err)
	}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(height)}

	out := make(map[common.Address]*big.Int, len(addresses))
	for i := 0; i < len(addresses); i += chunkSize {
		end := i + chunkSize
		if end > len(addresses) {
			end = len(addresses)
		}
		chunk := addresses[i:end]
		bals, err := q.GetDepositorsIFILBalances(opts, chunk)
		if err != nil {
			// Per-address fallback for this chunk only.
			for _, addr := range chunk {
				if r, e := q.GetDepositorsIFILBalances(opts, []common.Address{addr}); e == nil {
					out[addr] = r[0]
				}
			}
			continue
		}
		for j, addr := range chunk {
			out[addr] = bals[j]
		}
	}
	return out, nil
}
