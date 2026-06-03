package invariants

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	pools "github.com/glifio/go-pools/abigen"
	_ "github.com/lib/pq"
)

// FetchGLFTotalSupply reads the GLF (governance ERC-20) totalSupply at h.
func FetchGLFTotalSupply(ctx context.Context, ethClient *ethclient.Client, glfAddr common.Address, height uint64) (*big.Int, error) {
	tok, err := pools.NewPoolTokenCaller(glfAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("token caller: %w", err)
	}
	return tok.TotalSupply(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(height)})
}

// FetchGLFTotalSupplyFromDB reconstructs total supply from the glf
// transfer table: sum(mints from 0x0) - sum(burns to 0x0).
func FetchGLFTotalSupplyFromDB(ctx context.Context, postgresURL string, h uint64) (*big.Int, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var s string
	err = db.QueryRowContext(ctx, `
		SELECT (
		    COALESCE((SELECT sum(amount) FROM glf
		              WHERE from_ = '0x0000000000000000000000000000000000000000' AND height <= $1), 0)
		  - COALESCE((SELECT sum(amount) FROM glf
		              WHERE to_   = '0x0000000000000000000000000000000000000000' AND height <= $1), 0)
		)::TEXT`, h).Scan(&s)
	if err != nil {
		return nil, err
	}
	v, _ := new(big.Int).SetString(s, 10)
	return v, nil
}

// FetchGLFHolderBalancesFromDB returns per-holder balance derived from
// the glf table at h. Skips zero balances unless includeZero is true.
func FetchGLFHolderBalancesFromDB(ctx context.Context, postgresURL string, h uint64, includeZero bool) (map[common.Address]*big.Int, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	q := `
		WITH txs AS (
		    SELECT to_   AS addr, amount AS amt FROM glf WHERE height <= $1
		    UNION ALL
		    SELECT from_ AS addr, -amount AS amt FROM glf WHERE height <= $1
		)
		SELECT addr, sum(amt)::TEXT
		FROM txs
		WHERE addr <> '0x0000000000000000000000000000000000000000'
		GROUP BY addr
	`
	if !includeZero {
		q += "HAVING sum(amt) <> 0"
	}
	rows, err := db.QueryContext(ctx, q, h)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[common.Address]*big.Int)
	for rows.Next() {
		var addr, bal string
		if err := rows.Scan(&addr, &bal); err != nil {
			return nil, err
		}
		v, _ := new(big.Int).SetString(bal, 10)
		out[common.HexToAddress(addr)] = v
	}
	return out, rows.Err()
}

// FetchGLFHolderBalancesFromContract reads token.balanceOf for each address
// in parallel goroutines (concurrency-bounded). No batch contract method is
// available for GLF (Query.sol exposes only iFIL holders), so we parallelize
// individual balanceOf calls.
func FetchGLFHolderBalancesFromContract(
	ctx context.Context,
	ethClient *ethclient.Client,
	glfAddr common.Address,
	height uint64,
	addresses []common.Address,
	concurrency int,
) (map[common.Address]*big.Int, error) {
	if concurrency < 1 {
		concurrency = 16
	}
	tok, err := pools.NewPoolTokenCaller(glfAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("token caller: %w", err)
	}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(height)}

	results := make([]*big.Int, len(addresses))
	errs := make([]error, len(addresses))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, addr := range addresses {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, a common.Address) {
			defer wg.Done()
			defer func() { <-sem }()
			b, err := tok.BalanceOf(opts, a)
			results[i] = b
			errs[i] = err
		}(i, addr)
	}
	wg.Wait()

	out := make(map[common.Address]*big.Int, len(addresses))
	for i, a := range addresses {
		if errs[i] == nil {
			out[a] = results[i]
		}
	}
	return out, nil
}
