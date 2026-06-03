package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/glifio/invariants/abigen"
	"github.com/glifio/invariants/singleton"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPoolLPPlusCmd: per-token LP Plus invariant.
//
// Compares the indexer's lp_plus_mint set + per-token owner with the
// LPPlus NFT contract's view via InvariantsQuery.getLPPlusStates.
// Three layers:
//
//  1. Token count: count(distinct token_id) FROM lp_plus_mint
//     vs chain LPPlus.totalSupply()
//  2. Per-token existence: chain.ownerOf(tokenId) returns non-zero for
//     every token the indexer recorded a mint for.
//  3. Per-token owner drift (warning, not fail): chain.owner ==
//     mint.receiver. Drift indicates a token was transferred and the
//     indexer doesn't track LPPlus token transfers — informational.
//
// RWT/YBT balance comparisons are intentionally not in this version
// — would require parsing lp_plus_activity rows by activity type
// and is a separate phase.
func newPoolLPPlusCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare LP Plus per-token state between DB and contract",
		Args:  cobra.NoArgs,
		Run:   runPoolLPPlus,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch (default: API last_processed_height-3)")
	return cmd
}

func runPoolLPPlus(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set for LP Plus checks")
	}
	invQueryAddrStr := viper.GetString("invariants_query_addr")
	if invQueryAddrStr == "" {
		log.Fatal("INVARIANTS_QUERY_ADDR must be set in mainnet.env")
	}
	invQueryAddr := common.HexToAddress(invQueryAddrStr)

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

	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		log.Fatalf("eth client: %v", err)
	}
	defer ethClient.Close()

	q, err := abigen.NewInvariantsQueryCaller(invQueryAddr, ethClient)
	if err != nil {
		log.Fatalf("invariants query caller: %v", err)
	}
	opts := &bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(epoch)}

	// 1. Token count
	chainTotal, err := q.GetLPPlusTotalSupply(opts)
	if err != nil {
		log.Fatalf("getLPPlusTotalSupply: %v", err)
	}
	dbTokens, dbReceivers, err := fetchLPPlusMints(ctx, postgresURL, epoch)
	if err != nil {
		log.Fatalf("DB lp_plus_mint fetch: %v", err)
	}
	dbCount := big.NewInt(int64(len(dbTokens)))
	totalMatch := dbCount.Cmp(chainTotal) == 0
	if totalMatch {
		fmt.Printf("@%d: Success, LP Plus token count matches: %v\n", epoch, dbCount)
	} else {
		fmt.Printf("@%d: Token count MISMATCH  db=%v  chain=%v  diff=%v\n",
			epoch, dbCount, chainTotal, new(big.Int).Sub(dbCount, chainTotal))
	}

	if len(dbTokens) == 0 {
		return
	}

	// 2 + 3. Per-token state
	fmt.Printf("\n@%d: per-token — checking %d LP Plus tokens\n", epoch, len(dbTokens))
	states, err := q.GetLPPlusStates(opts, dbTokens)
	if err != nil {
		log.Fatalf("getLPPlusStates: %v", err)
	}
	if len(states) != len(dbTokens) {
		log.Fatalf("getLPPlusStates returned %d states for %d tokens", len(states), len(dbTokens))
	}

	missing, transferred := 0, 0
	zero := common.Address{}
	for i, id := range dbTokens {
		owner := states[i].Owner
		if owner == zero {
			fmt.Printf("  token %v MISSING on chain (DB recorded mint to %s)\n",
				id, dbReceivers[id.String()].Hex())
			missing++
			continue
		}
		if owner != dbReceivers[id.String()] {
			fmt.Printf("  token %v transferred  mint=%s  chain.owner=%s\n",
				id, dbReceivers[id.String()].Hex(), owner.Hex())
			transferred++
		}
	}

	fmt.Printf("\nLP Plus summary: %d tokens total, %d missing on chain, %d transferred since mint\n",
		len(dbTokens), missing, transferred)

	if !totalMatch || missing > 0 {
		log.Fatalf("FAIL: LP Plus invariants — count_match=%v missing=%d", totalMatch, missing)
	}
}

// fetchLPPlusMints returns the (token_id, receiver) set from lp_plus_mint
// for tokens minted at or before h. Returns parallel slices/maps so the
// caller can pass token_ids straight into InvariantsQuery.getLPPlusStates.
func fetchLPPlusMints(ctx context.Context, postgresURL string, h uint64) ([]*big.Int, map[string]common.Address, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres open: %w", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
		SELECT token_id::TEXT, receiver
		FROM lp_plus_mint
		WHERE height <= $1
		ORDER BY token_id`, h)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids []*big.Int
	receivers := make(map[string]common.Address)
	for rows.Next() {
		var idStr, receiverStr string
		if err := rows.Scan(&idStr, &receiverStr); err != nil {
			return nil, nil, err
		}
		id, ok := new(big.Int).SetString(idStr, 10)
		if !ok {
			return nil, nil, fmt.Errorf("bad token_id %q", idStr)
		}
		ids = append(ids, id)
		receivers[id.String()] = common.HexToAddress(receiverStr)
	}
	return ids, receivers, rows.Err()
}

func init() {
	poolCmd.AddCommand(newPoolLPPlusCmd("lpplus [--epoch <epoch>]"))
}
