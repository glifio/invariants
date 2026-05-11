package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/glifio/invariants/singleton"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPoolCleanHolderCmd: scans every iFIL row that involves a single
// holder, cross-checks each (txhash, idx) against the chain receipt,
// and reports/emits a DELETE statement for rows whose chain log isn't
// an iFIL Transfer (daemon-era contamination — wrong contract).
//
// The user reviews the output, runs the DELETE manually, then refills
// the affected height range to backfill any genuinely-missing real
// rows.
func newPoolCleanHolderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-holder --token <ifil|glf> --addr <0x...>",
		Short: "Find phantom rows for one holder by cross-checking chain receipts",
		Args:  cobra.NoArgs,
		Run:   runPoolCleanHolder,
	}
	cmd.Flags().String("token", "ifil", "ifil | glf")
	cmd.Flags().String("addr", "", "holder address (hex). Omit with --all to scan the whole table.")
	cmd.Flags().Bool("all", false, "scan every row in the table (ignores --addr)")
	cmd.Flags().Int("workers", 8, "concurrent receipt fetchers")
	cmd.Flags().Uint64("max-height", 0, "only scan rows with height <= this (0 = no limit; useful: 5986549 = end of daemon era)")
	cmd.Flags().Uint64("chunk-size", 10000, "eth_getLogs block-range chunk size when --all is set")
	return cmd
}

var transferTopic = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

func runPoolCleanHolder(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set")
	}
	token, _ := cmd.Flags().GetString("token")
	addrStr, _ := cmd.Flags().GetString("addr")
	scanAll, _ := cmd.Flags().GetBool("all")
	workers, _ := cmd.Flags().GetInt("workers")
	maxHeight, _ := cmd.Flags().GetUint64("max-height")
	chunkSize, _ := cmd.Flags().GetUint64("chunk-size")
	if !scanAll && addrStr == "" {
		log.Fatal("--addr is required (or use --all to scan the whole table)")
	}
	var addr common.Address
	if addrStr != "" {
		addr = common.HexToAddress(addrStr)
	}

	if err := initSingleton(ctx); err != nil {
		log.Fatal(err)
	}
	sdk := singleton.PoolsSDK
	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		log.Fatalf("eth client: %v", err)
	}
	defer ethClient.Close()

	var contractAddr common.Address
	var dbTable string
	switch strings.ToLower(token) {
	case "ifil":
		contractAddr = sdk.Query().IFIL()
		dbTable = "ifil"
	case "glf":
		glfAddrStr := viper.GetString("glf_addr")
		if glfAddrStr == "" {
			log.Fatal("GLF_ADDR must be set")
		}
		contractAddr = common.HexToAddress(glfAddrStr)
		dbTable = "glf"
	default:
		log.Fatalf("unknown token %q", token)
	}

	dbConn, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("postgres open: %v", err)
	}
	defer dbConn.Close()

	type row struct {
		txhash string
		idx    int
		height uint64
	}
	var (
		rows *sql.Rows
		err2 error
	)
	if scanAll {
		q := fmt.Sprintf(`SELECT txhash, idx, height FROM %s`, dbTable)
		args := []any{}
		if maxHeight > 0 {
			q += ` WHERE height <= $1`
			args = append(args, maxHeight)
		}
		q += ` ORDER BY height, idx`
		rows, err2 = dbConn.QueryContext(ctx, q, args...)
	} else {
		q := fmt.Sprintf(`SELECT txhash, idx, height FROM %s WHERE (from_ = $1 OR to_ = $1)`, dbTable)
		args := []any{addr.Hex()}
		if maxHeight > 0 {
			q += ` AND height <= $2`
			args = append(args, maxHeight)
		}
		q += ` ORDER BY height, idx`
		rows, err2 = dbConn.QueryContext(ctx, q, args...)
	}
	if err2 != nil {
		log.Fatalf("db query: %v", err2)
	}
	var allRows []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.txhash, &r.idx, &r.height); err != nil {
			log.Fatal(err)
		}
		allRows = append(allRows, r)
	}
	rows.Close()
	if scanAll {
		fmt.Printf("token=%s contract=%s scope=ALL ROWS (max-height=%d)\n", token, contractAddr.Hex(), maxHeight)
	} else {
		fmt.Printf("addr=%s token=%s contract=%s\n", addr.Hex(), token, contractAddr.Hex())
	}
	fmt.Printf("DB rows in scope: %d\n", len(allRows))
	if len(allRows) == 0 {
		return
	}

	type chainKey struct {
		txhash string
		logIdx uint
	}
	realIfilLogs := map[chainKey]struct{}{}

	if scanAll {
		// Use eth_getLogs filtered by contract — much faster than per-tx receipts.
		var minH, maxH uint64 = ^uint64(0), 0
		for _, r := range allRows {
			if r.height < minH {
				minH = r.height
			}
			if r.height > maxH {
				maxH = r.height
			}
		}
		fmt.Printf("Scanning chain via eth_getLogs over [%d..%d] in chunks of %d (workers=%d)\n", minH, maxH, chunkSize, workers)
		type chunk struct{ from, to uint64 }
		var chunks []chunk
		for from := minH; from <= maxH; from += chunkSize {
			to := from + chunkSize - 1
			if to > maxH {
				to = maxH
			}
			chunks = append(chunks, chunk{from, to})
		}
		work := make(chan chunk, len(chunks))
		for _, c := range chunks {
			work <- c
		}
		close(work)
		var mu sync.Mutex
		var wg sync.WaitGroup
		var done int
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for c := range work {
					q := ethereum.FilterQuery{
						FromBlock: new(big.Int).SetUint64(c.from),
						ToBlock:   new(big.Int).SetUint64(c.to),
						Addresses: []common.Address{contractAddr},
						Topics:    [][]common.Hash{{transferTopic}},
					}
					var logs []types.Log
					var err error
					backoff := 2 * time.Second
					for attempt := 0; attempt < 8; attempt++ {
						logs, err = ethClient.FilterLogs(ctx, q)
						if err == nil {
							break
						}
						es := err.Error()
						if strings.Contains(es, "429") || strings.Contains(es, "credits") || strings.Contains(es, "rate") {
							fmt.Fprintf(cmd.ErrOrStderr(), "  [%d..%d] rate-limited, retrying in %v...\n", c.from, c.to, backoff)
							time.Sleep(backoff)
							backoff *= 2
							if backoff > 60*time.Second {
								backoff = 60 * time.Second
							}
							continue
						}
						break
					}
					if err != nil {
						log.Fatalf("eth_getLogs [%d..%d] (after retries): %v", c.from, c.to, err)
					}
					mu.Lock()
					for _, l := range logs {
						realIfilLogs[chainKey{l.TxHash.Hex(), l.Index}] = struct{}{}
					}
					done++
					if done%20 == 0 {
						fmt.Fprintf(cmd.ErrOrStderr(), "  %d/%d chunks (cumulative real logs: %d)\n", done, len(chunks), len(realIfilLogs))
					}
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
	} else {
		uniqTx := map[string]struct{}{}
		for _, r := range allRows {
			uniqTx[r.txhash] = struct{}{}
		}
		fmt.Printf("Unique txhashes to fetch: %d (workers=%d)\n", len(uniqTx), workers)

		var mu sync.Mutex
		work := make(chan string, len(uniqTx))
		for h := range uniqTx {
			work <- h
		}
		close(work)
		var wg sync.WaitGroup
		var done int64
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for tx := range work {
					rec, err := receiptOf(ctx, ethClient, common.HexToHash(tx))
					if err != nil {
						log.Printf("receipt %s: %v", tx, err)
						continue
					}
					mu.Lock()
					for _, l := range rec.Logs {
						if l.Address != contractAddr {
							continue
						}
						if len(l.Topics) == 0 || l.Topics[0] != transferTopic {
							continue
						}
						realIfilLogs[chainKey{tx, l.Index}] = struct{}{}
					}
					done++
					if done%500 == 0 {
						fmt.Fprintf(cmd.ErrOrStderr(), "  fetched %d/%d receipts...\n", done, len(uniqTx))
					}
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
	}

	// Bucket DB rows.
	var phantom, real []row
	heightSet := map[uint64]struct{}{}
	for _, r := range allRows {
		k := chainKey{r.txhash, uint(r.idx)}
		if _, ok := realIfilLogs[k]; ok {
			real = append(real, r)
		} else {
			phantom = append(phantom, r)
			heightSet[r.height] = struct{}{}
		}
	}
	fmt.Printf("\nReal iFIL rows: %d   Phantom rows: %d\n", len(real), len(phantom))
	if len(phantom) == 0 {
		fmt.Println("No phantom rows for this holder.")
		return
	}

	// Build composite-key DELETE.
	fmt.Printf("\n-- DELETE %d phantom rows for %s\n", len(phantom), addr.Hex())
	fmt.Println("BEGIN;")
	const chunk = 500
	for i := 0; i < len(phantom); i += chunk {
		end := i + chunk
		if end > len(phantom) {
			end = len(phantom)
		}
		var pairs []string
		for _, r := range phantom[i:end] {
			pairs = append(pairs, fmt.Sprintf("('%s',%d)", r.txhash, r.idx))
		}
		fmt.Printf("DELETE FROM %s WHERE (txhash, idx) IN (%s);\n", dbTable, strings.Join(pairs, ","))
	}
	fmt.Println("COMMIT;")

	// Print refill range so caller can backfill any genuinely-missing real rows.
	var minH, maxH uint64
	for h := range heightSet {
		if minH == 0 || h < minH {
			minH = h
		}
		if h > maxH {
			maxH = h
		}
	}
	fmt.Printf("\n-- Affected heights: %d distinct, range [%d..%d]\n", len(heightSet), minH, maxH)
	fmt.Printf("-- After running DELETE, optionally refill any heights to recover real rows that\n")
	fmt.Printf("-- might have been deleted alongside phantoms (refill is no-op for unaffected rows):\n")
	fmt.Printf("--   idx refill --from %d --to %d   (caution: large range)\n", minH, maxH)
}

func receiptOf(ctx context.Context, c *ethclient.Client, h common.Hash) (*types.Receipt, error) {
	return c.TransactionReceipt(ctx, h)
}

func init() {
	poolCmd.AddCommand(newPoolCleanHolderCmd())
}
