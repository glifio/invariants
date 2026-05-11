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

// newPoolSPPlusCmd: per-token SP Plus invariant.
//
// Mirrors the LP Plus check: the indexer's sp_plus_mint set + per-token
// owner vs the SPPlusV2 NFT contract via InvariantsQuery.getSPPlusStates.
//
//  1. Token count: count(distinct token_id) FROM sp_plus_mint
//     vs (chain SPPlusV2 has no totalSupply we exposed; we infer from
//     ownerOf returning non-zero on every DB-known token).
//  2. Per-token existence: chain.ownerOf(tokenId) returns non-zero.
//  3. Per-token owner drift: warn if chain.owner != mint.receiver
//     (SP+ NFTs may transfer; informational).
func newPoolSPPlusCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare SP Plus per-token state between DB and contract",
		Args:  cobra.NoArgs,
		Run:   runPoolSPPlus,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch (default: API last_processed_height-3)")
	return cmd
}

func runPoolSPPlus(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set for SP Plus checks")
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

	dbTokens, dbReceivers, err := fetchSPPlusMints(ctx, postgresURL, epoch)
	if err != nil {
		log.Fatalf("DB sp_plus_mint fetch: %v", err)
	}
	fmt.Printf("@%d: per-token — checking %d SP Plus tokens\n", epoch, len(dbTokens))
	if len(dbTokens) == 0 {
		return
	}

	states, err := q.GetSPPlusStates(opts, dbTokens)
	if err != nil {
		log.Fatalf("getSPPlusStates: %v", err)
	}
	if len(states) != len(dbTokens) {
		log.Fatalf("getSPPlusStates returned %d states for %d tokens", len(states), len(dbTokens))
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
	fmt.Printf("\nSP Plus summary: %d tokens total, %d missing on chain, %d transferred since mint\n",
		len(dbTokens), missing, transferred)

	if missing > 0 {
		log.Fatalf("FAIL: SP Plus invariants — missing=%d", missing)
	}
}

func fetchSPPlusMints(ctx context.Context, postgresURL string, h uint64) ([]*big.Int, map[string]common.Address, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres open: %w", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
		SELECT token_id::TEXT, receiver
		FROM sp_plus_mint
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
	poolCmd.AddCommand(newPoolSPPlusCmd("spplus [--epoch <epoch>]"))
}
