package main

import (
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPoolResyncRangeCmd: chain-canonical re-derivation of iFIL rows
// in a height range. Writes a SQL transaction to stdout that deletes
// every existing row in [from..to] and inserts only the real Transfer
// logs returned by eth_getLogs filtered on IFIL_ADDR.
//
// Used to validate the "truncate+resync from chain" approach. The
// scan is single-threaded with a small inter-chunk sleep so it stays
// well under the chain.love rate limit.
func newPoolResyncRangeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resync-range --token <ifil|glf> --from H1 --to H2",
		Short: "Emit DELETE+INSERT SQL to resync ERC-20 rows for a height range from chain logs",
		Args:  cobra.NoArgs,
		Run:   runPoolResyncRange,
	}
	cmd.Flags().String("token", "ifil", "ifil | glf")
	cmd.Flags().Uint64("from", 0, "first height (inclusive)")
	cmd.Flags().Uint64("to", 0, "last height (inclusive)")
	cmd.Flags().Uint64("chunk-size", 5000, "eth_getLogs block-range chunk size")
	cmd.Flags().Duration("sleep", 500*time.Millisecond, "sleep between chunks (rate-limit pacing)")
	return cmd
}

func runPoolResyncRange(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	token, _ := cmd.Flags().GetString("token")
	from, _ := cmd.Flags().GetUint64("from")
	to, _ := cmd.Flags().GetUint64("to")
	chunkSize, _ := cmd.Flags().GetUint64("chunk-size")
	sleepDur, _ := cmd.Flags().GetDuration("sleep")
	if from == 0 || to == 0 || to < from {
		log.Fatal("--from and --to are required and to >= from")
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

	fmt.Fprintf(cmd.ErrOrStderr(), "Resync %s [%d..%d] (contract=%s, chunk=%d, sleep=%v)\n",
		token, from, to, contractAddr.Hex(), chunkSize, sleepDur)

	type chainRow struct {
		txhash string
		idx    uint
		height uint64
		from   common.Address
		to     common.Address
		amount *big.Int
	}
	var rows []chainRow

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
			Addresses: []common.Address{contractAddr},
			Topics:    [][]common.Hash{{transferTopic}},
		}
		// Paced retry loop.
		var logs []ethereumLogResult
		backoff := 2 * time.Second
		for attempt := 0; attempt < 8; attempt++ {
			ll, err := ethClient.FilterLogs(ctx, q)
			if err == nil {
				for _, l := range ll {
					logs = append(logs, ethereumLogResult{
						txhash: l.TxHash.Hex(),
						idx:    l.Index,
						height: l.BlockNumber,
						from:   common.BytesToAddress(l.Topics[1].Bytes()),
						to:     common.BytesToAddress(l.Topics[2].Bytes()),
						amount: new(big.Int).SetBytes(l.Data),
					})
				}
				break
			}
			es := err.Error()
			if strings.Contains(es, "429") || strings.Contains(es, "credits") || strings.Contains(es, "rate") {
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
		for _, l := range logs {
			rows = append(rows, chainRow{l.txhash, l.idx, l.height, l.from, l.to, l.amount})
		}
		chunkIdx++
		fmt.Fprintf(cmd.ErrOrStderr(), "  chunk %d/%d [%d..%d] %d logs (cumulative=%d)\n",
			chunkIdx, totalChunks, f, t, len(logs), len(rows))
		if sleepDur > 0 && f+chunkSize <= to {
			time.Sleep(sleepDur)
		}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\nFetched %d real %s Transfer logs in [%d..%d]\n", len(rows), token, from, to)

	// Emit SQL.
	fmt.Println("BEGIN;")
	fmt.Printf("DELETE FROM %s WHERE height BETWEEN %d AND %d;\n", dbTable, from, to)
	const chunkInsert = 200
	for i := 0; i < len(rows); i += chunkInsert {
		end := i + chunkInsert
		if end > len(rows) {
			end = len(rows)
		}
		fmt.Printf("INSERT INTO %s (txhash, idx, height, from_, to_, amount) VALUES\n", dbTable)
		for j := i; j < end; j++ {
			r := rows[j]
			sep := ","
			if j == end-1 {
				sep = ";"
			}
			fmt.Printf("  ('%s',%d,%d,'%s','%s',%s)%s\n",
				r.txhash, r.idx, r.height, r.from.Hex(), r.to.Hex(), r.amount.String(), sep)
		}
	}
	fmt.Println("COMMIT;")
}

type ethereumLogResult struct {
	txhash string
	idx    uint
	height uint64
	from   common.Address
	to     common.Address
	amount *big.Int
}

func init() {
	poolCmd.AddCommand(newPoolResyncRangeCmd())
}
