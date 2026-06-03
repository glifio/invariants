package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/glifio/invariants/singleton"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPoolHarvestTraceCmd: chain-canonical scan of every WFIL Transfer
// where from = pool and to ∈ (treasury1, treasury2) — i.e. the rows
// the SQL `harvest(h)` function sums. Emits the chain-truth list,
// the DB list, and the diff.
//
// Reason it exists: harvest() in 000002_functions.up.sql adds a
// hardcoded literal `137216670177581774010533` (~137,216.67 FIL) to
// the DB sum with no provenance. The likely cause is that the indexer
// started at h≈2,895,059 and missed earlier pool→treasury transfers;
// this command lets us prove or disprove that.
//
// Filter is pushed into eth_getLogs (no per-chunk post-filter), so
// each chunk returns ~zero rows except where harvests actually
// occurred. Result is small and fast.
func newPoolHarvestTraceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "harvest-trace",
		Short: "Diff chain WFIL pool→treasury transfers vs the indexer DB (recreate harvest constant)",
		Args:  cobra.NoArgs,
		Run:   runPoolHarvestTrace,
	}
	cmd.Flags().Uint64("from", 0, "first height (inclusive, default 0)")
	cmd.Flags().Uint64("to", 0, "last height (inclusive, default head-3)")
	cmd.Flags().Uint64("chunk-size", 100000, "eth_getLogs block-range chunk size")
	cmd.Flags().Duration("sleep", 100*time.Millisecond, "sleep between chunks")
	cmd.Flags().StringSlice("pool", []string{
		"0x43dae5624445e7679d16a63211c5ff368681500c", // POOL_V0 (the cutover-era v1 pool)
		"0xe764Acf02D8B7c21d2B6A8f0a96C78541e0DC3fd", // POOL_V2 (current; harvest() filters on this)
	}, "pool address(es) to treat as treasurer (sender of WFIL); accepts multiple")
	cmd.Flags().StringSlice("treasury", []string{
		"0xFf00000000000000000000000000000000219b24",
		"0xff0000000000000000000000000000000031b072",
	}, "treasury addresses (receivers; can repeat)")
	cmd.Flags().Bool("emit-sql", false, "emit SQL INSERT statements for chain rows missing from DB")
	return cmd
}

func runPoolHarvestTrace(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	from, _ := cmd.Flags().GetUint64("from")
	to, _ := cmd.Flags().GetUint64("to")
	chunkSize, _ := cmd.Flags().GetUint64("chunk-size")
	sleepDur, _ := cmd.Flags().GetDuration("sleep")
	poolStrs, _ := cmd.Flags().GetStringSlice("pool")
	treasuries, _ := cmd.Flags().GetStringSlice("treasury")
	emitSQL, _ := cmd.Flags().GetBool("emit-sql")

	if err := initSingleton(ctx); err != nil {
		log.Fatal(err)
	}
	sdk := singleton.PoolsSDK
	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		log.Fatalf("eth client: %v", err)
	}
	defer ethClient.Close()

	if to == 0 {
		head, err := getHeadEpoch(ctx)
		if err != nil {
			log.Fatalf("head: %v", err)
		}
		to = head - 3
	}

	wfilStr := viper.GetString("wfil_addr")
	if wfilStr == "" {
		log.Fatal("WFIL_ADDR must be set in mainnet.env")
	}
	wfil := common.HexToAddress(wfilStr)

	// Standard ERC-20 Transfer event signature.
	transferTopic := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	// Topic indexing: topics[1] = from (any pool), topics[2] = to (any treasury).
	poolAddrs := make([]common.Address, 0, len(poolStrs))
	fromTopics := make([]common.Hash, 0, len(poolStrs))
	for _, p := range poolStrs {
		a := common.HexToAddress(p)
		poolAddrs = append(poolAddrs, a)
		fromTopics = append(fromTopics, common.BytesToHash(common.LeftPadBytes(a.Bytes(), 32)))
	}
	toTopics := make([]common.Hash, 0, len(treasuries))
	treasuryAddrs := make([]common.Address, 0, len(treasuries))
	for _, t := range treasuries {
		a := common.HexToAddress(t)
		treasuryAddrs = append(treasuryAddrs, a)
		toTopics = append(toTopics, common.BytesToHash(common.LeftPadBytes(a.Bytes(), 32)))
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "harvest-trace [%d..%d] (pools=%v, treasuries=%v, chunk=%d)\n",
		from, to, poolStrs, treasuries, chunkSize)

	type chainRow struct {
		txhash string
		idx    uint
		height uint64
		from   common.Address
		to     common.Address
		amount *big.Int
	}
	// Build a fast set for the to-side post-filter.
	toSet := make(map[common.Address]struct{}, len(treasuryAddrs))
	for _, a := range treasuryAddrs {
		toSet[a] = struct{}{}
	}

	var rows []chainRow

	// Run one scan per (from address). Single-value topic filters keep
	// chain.love happy — multi-OR on Topics[2] was timing out.
	for fi, fromTopic := range fromTopics {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nScanning from=%s ...\n", poolAddrs[fi].Hex())
		totalChunks := (to-from)/chunkSize + 1
		chunkIdx := 0
		for f := from; f <= to; f += chunkSize {
			t := f + chunkSize - 1
			if t > to {
				t = to
			}
			q := ethereum.FilterQuery{
				FromBlock: new(big.Int).SetUint64(f),
				ToBlock:   new(big.Int).SetUint64(t),
				Addresses: []common.Address{wfil},
				Topics: [][]common.Hash{
					{transferTopic},
					{fromTopic},
				},
			}
			backoff := 2 * time.Second
			var matched int
			for attempt := 0; attempt < 8; attempt++ {
				ll, err := ethClient.FilterLogs(ctx, q)
				if err == nil {
					for _, l := range ll {
						toAddr := common.BytesToAddress(l.Topics[2].Bytes())
						if _, ok := toSet[toAddr]; !ok {
							continue
						}
						rows = append(rows, chainRow{
							txhash: l.TxHash.Hex(),
							idx:    l.Index,
							height: l.BlockNumber,
							from:   common.BytesToAddress(l.Topics[1].Bytes()),
							to:     toAddr,
							amount: new(big.Int).SetBytes(l.Data),
						})
						matched++
					}
					break
				}
				es := err.Error()
				if strings.Contains(es, "429") || strings.Contains(es, "credits") || strings.Contains(es, "rate") ||
					strings.Contains(es, "504") || strings.Contains(es, "timeout") || strings.Contains(es, "Time-out") {
					fmt.Fprintf(cmd.ErrOrStderr(), "  [%d..%d] rate-limited, retrying in %v...\n", f, t, backoff)
					time.Sleep(backoff)
					backoff *= 2
					if backoff > 60*time.Second {
						backoff = 60 * time.Second
					}
					continue
				}
				log.Fatalf("eth_getLogs [%d..%d]: %v", f, t, err)
			}
			chunkIdx++
			if matched > 0 || chunkIdx%20 == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  chunk %d/%d [%d..%d] +%d (cumulative=%d)\n",
					chunkIdx, totalChunks, f, t, matched, len(rows))
			}
			if sleepDur > 0 && f+chunkSize <= to {
				time.Sleep(sleepDur)
			}
		}
	}

	chainTotal := new(big.Int)
	for _, r := range rows {
		chainTotal.Add(chainTotal, r.amount)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "\nChain rows: %d, sum: %s wei (%s FIL)\n",
		len(rows), chainTotal.String(), formatFIL(chainTotal))

	// DB side.
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set")
	}
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	// Build a key set of chain rows for the diff.
	type rowKey struct {
		txhash string
		idx    uint
	}
	chainByKey := make(map[rowKey]chainRow, len(rows))
	for _, r := range rows {
		chainByKey[rowKey{strings.ToLower(r.txhash), r.idx}] = r
	}

	// Pull DB rows for the same filter+range.
	args := []interface{}{int64(from), int64(to)}
	fromPlaceholders := make([]string, 0, len(poolAddrs))
	for _, a := range poolAddrs {
		args = append(args, strings.ToLower(a.Hex()))
		fromPlaceholders = append(fromPlaceholders, fmt.Sprintf("$%d", len(args)))
	}
	toPlaceholders := make([]string, 0, len(treasuryAddrs))
	for _, a := range treasuryAddrs {
		args = append(args, strings.ToLower(a.Hex()))
		toPlaceholders = append(toPlaceholders, fmt.Sprintf("$%d", len(args)))
	}
	q := fmt.Sprintf(`
		SELECT txhash, idx, height, from_, to_, amount
		FROM wfil
		WHERE height BETWEEN $1 AND $2
		AND LOWER(from_) IN (%s)
		AND LOWER(to_) IN (%s)
		ORDER BY height, idx
	`, strings.Join(fromPlaceholders, ","), strings.Join(toPlaceholders, ","))
	dbRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		log.Fatalf("db query: %v", err)
	}
	defer dbRows.Close()
	dbByKey := make(map[rowKey]chainRow)
	dbTotal := new(big.Int)
	dbCount := 0
	for dbRows.Next() {
		var r chainRow
		var amountStr, fromStr, toStr string
		if err := dbRows.Scan(&r.txhash, &r.idx, &r.height, &fromStr, &toStr, &amountStr); err != nil {
			log.Fatalf("db scan: %v", err)
		}
		r.from = common.HexToAddress(fromStr)
		r.to = common.HexToAddress(toStr)
		r.amount, _ = new(big.Int).SetString(amountStr, 10)
		dbByKey[rowKey{strings.ToLower(r.txhash), r.idx}] = r
		dbTotal.Add(dbTotal, r.amount)
		dbCount++
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "DB rows:    %d, sum: %s wei (%s FIL)\n",
		dbCount, dbTotal.String(), formatFIL(dbTotal))

	// Diff.
	var missing, extra []chainRow
	for k, r := range chainByKey {
		if _, ok := dbByKey[k]; !ok {
			missing = append(missing, r)
		}
	}
	for k, r := range dbByKey {
		if _, ok := chainByKey[k]; !ok {
			extra = append(extra, r)
		}
	}

	delta := new(big.Int).Sub(chainTotal, dbTotal)
	fmt.Fprintf(cmd.ErrOrStderr(), "\nDelta (chain - db): %s wei (%s FIL)\n",
		delta.String(), formatFIL(delta))
	fmt.Fprintf(cmd.ErrOrStderr(), "Hardcoded harvest constant: 137216670177581774010533 wei (137216.67 FIL)\n")
	matchExact := new(big.Int)
	matchExact.SetString("137216670177581774010533", 10)
	if delta.Cmp(matchExact) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "✅ Delta exactly matches the hardcoded constant.")
	} else {
		off := new(big.Int).Sub(delta, matchExact)
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Delta is off by %s wei (%s FIL)\n",
			off.String(), formatFIL(off))
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\nMissing from DB: %d rows\n", len(missing))
	for _, r := range missing {
		fmt.Fprintf(cmd.ErrOrStderr(), "  h=%d %s idx=%d to=%s amount=%s wei (%s FIL)\n",
			r.height, r.txhash, r.idx, r.to.Hex(), r.amount.String(), formatFIL(r.amount))
	}
	if len(extra) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nIn DB but not on chain (suspicious): %d rows\n", len(extra))
		for _, r := range extra {
			fmt.Fprintf(cmd.ErrOrStderr(), "  h=%d %s idx=%d to=%s amount=%s wei\n",
				r.height, r.txhash, r.idx, r.to.Hex(), r.amount.String())
		}
	}

	if emitSQL && len(missing) > 0 {
		fmt.Println("BEGIN;")
		for _, r := range missing {
			fmt.Printf("INSERT INTO wfil (txhash, idx, height, from_, to_, amount) VALUES ('%s',%d,%d,'%s','%s',%s);\n",
				r.txhash, r.idx, r.height, r.from.Hex(), r.to.Hex(), r.amount.String())
		}
		fmt.Println("COMMIT;")
	}
}

func formatFIL(wei *big.Int) string {
	if wei.Sign() == 0 {
		return "0"
	}
	neg := wei.Sign() < 0
	a := new(big.Int).Abs(wei)
	whole := new(big.Int).Quo(a, big.NewInt(1e18))
	rem := new(big.Int).Mod(a, big.NewInt(1e18))
	// 6 digits of precision is fine for human reading.
	frac := new(big.Int).Quo(rem, big.NewInt(1e12))
	s := fmt.Sprintf("%s.%06d", whole.String(), frac.Int64())
	if neg {
		s = "-" + s
	}
	return s
}

func init() {
	poolCmd.AddCommand(newPoolHarvestTraceCmd())
}

// Force the compiler to keep context import even if unused below.
var _ = context.Background
