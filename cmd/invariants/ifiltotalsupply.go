package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newPoolIfilCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare the iFIL Total Supply (and optionally per-depositor balances) between API and node",
		Args:  cobra.NoArgs,
		Run:   runPoolIfil,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch")
	cmd.Flags().Bool("find-missing", false, "Find missing transactions")
	cmd.Flags().Bool("per-depositor", false,
		"Also reconcile every iFIL holder's balance vs InvariantsQuery.getIFILBalances (slow on prod; ~7-8k holders)")
	cmd.Flags().Int("chunk", 500, "Holders per Query batch RPC (--per-depositor only)")
	cmd.Flags().Bool("include-zero", false,
		"Include holders whose DB balance is zero in the per-depositor check (default: skip dust)")
	return cmd
}

func runPoolIfil(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	eventsURL := viper.GetString("events_api")

	err := initSingleton(ctx)
	if err != nil {
		log.Fatal(err)
	}

	epoch, err := cmd.Flags().GetUint64("epoch")
	if err != nil {
		log.Fatal(err)
	}

	findMissing, err := cmd.Flags().GetBool("find-missing")
	if err != nil {
		log.Fatal(err)
	}

	if epoch == 0 {
		epoch, err = getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		epoch = epoch - 2
	}

	apiTotalSupply, err := invariants.GetIFILTotalSupplyFromAPI(ctx, eventsURL, epoch)
	if err != nil {
		log.Fatal(err)
	}

	nodeTotalSupply, resultEpoch, err := invariants.GetIFILTotalSupplyFromNode(ctx, epoch)
	if err != nil {
		log.Fatal(err)
	}

	// Mutate for testing
	// nodeTotalSupply.IFILTotalSupply = big.NewInt(1234)

	totalSupplyMatch := apiTotalSupply.IFILTotalSupply.Cmp(nodeTotalSupply.IFILTotalSupply) == 0
	if totalSupplyMatch {
		fmt.Printf("@%d: Success, iFIL total supply matches: %v\n", epoch, apiTotalSupply.IFILTotalSupply)
	} else {
		fmt.Printf("@%d: Error, iFIL total supply from REST API doesn't match node.\n", epoch)
		fmt.Printf("  Node @%d: %v\n", resultEpoch, nodeTotalSupply.IFILTotalSupply)
		fmt.Printf("   API @%d: %v\n", epoch, apiTotalSupply.IFILTotalSupply)
		if findMissing {
			findMissingIFILEvents(ctx, eventsURL, epoch)
		}
	}

	perDepositor, _ := cmd.Flags().GetBool("per-depositor")
	chunk, _ := cmd.Flags().GetInt("chunk")
	includeZero, _ := cmd.Flags().GetBool("include-zero")

	depMismatch := 0
	if perDepositor {
		depMismatch = runIFILPerDepositor(ctx, epoch, chunk, includeZero)
	}

	if !totalSupplyMatch || depMismatch > 0 {
		log.Fatalf("FAIL: iFIL invariants — total supply match=%v, per-depositor mismatches=%d", totalSupplyMatch, depMismatch)
	}
}

// runIFILPerDepositor compares every (non-zero) DB-derived holder
// balance to InvariantsQuery.getIFILBalances at the same height.
// Returns the count of mismatches.
func runIFILPerDepositor(ctx context.Context, epoch uint64, chunkSize int, includeZero bool) int {
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set for --per-depositor")
	}
	queryAddrStr := viper.GetString("invariants_query_addr")
	if queryAddrStr == "" {
		log.Fatal("INVARIANTS_QUERY_ADDR must be set for --per-depositor")
	}
	queryAddr := common.HexToAddress(queryAddrStr)

	dbBals, err := invariants.FetchIFILDepositorBalancesFromDB(ctx, postgresURL, epoch, includeZero)
	if err != nil {
		log.Fatalf("DB depositor fetch: %v", err)
	}
	addresses := make([]common.Address, 0, len(dbBals))
	for addr := range dbBals {
		addresses = append(addresses, addr)
	}
	// Stable order so output is deterministic for diffing across runs.
	sort.Slice(addresses, func(i, j int) bool {
		return addresses[i].Hex() < addresses[j].Hex()
	})
	fmt.Printf("\n@%d: per-depositor — checking %d holders (chunk=%d include-zero=%v)\n",
		epoch, len(addresses), chunkSize, includeZero)

	ethClient, err := singleton.PoolsSDK.Extern().ConnectEthClient()
	if err != nil {
		log.Fatalf("eth client: %v", err)
	}
	defer ethClient.Close()

	chainBals, err := invariants.FetchIFILDepositorBalancesFromContract(ctx, ethClient, queryAddr, epoch, addresses, chunkSize)
	if err != nil {
		log.Fatalf("contract depositor fetch: %v", err)
	}

	mismatch := 0
	zero := big.NewInt(0)
	for _, addr := range addresses {
		dbBal := dbBals[addr]
		chainBal := chainBals[addr]
		if chainBal == nil {
			chainBal = zero
		}
		if dbBal.Cmp(chainBal) != 0 {
			diff := new(big.Int).Sub(dbBal, chainBal)
			fmt.Printf("  %s MISMATCH  db=%v  chain=%v  diff=%v\n",
				addr.Hex(), dbBal, chainBal, diff)
			mismatch++
		}
	}
	fmt.Printf("\nPer-depositor summary: %d/%d holders match, %d mismatches\n",
		len(addresses)-mismatch, len(addresses), mismatch)
	return mismatch
}


const step = 10000

func findMissingIFILEvents(ctx context.Context, eventsURL string, maxEpoch uint64) {
	fmt.Println("Searching for missing iFIL events")

	var goodEpoch uint64
	var err error
	epoch := int64(maxEpoch)
	for {
		minEpoch := max(epoch-step+1, 0)
		goodEpoch, err = searchPassingIFILTotalSupply(ctx, eventsURL, uint64(epoch), uint64(minEpoch), "")
		if err != nil {
			log.Fatal(err)
		}
		if goodEpoch != 0 {
			break
		}
		epoch = epoch - step
		if epoch < 0 {
			log.Fatal("No passing epochs found")
		}
	}
	fmt.Printf("Highest passing epoch: %v\n", goodEpoch)
}

func searchPassingIFILTotalSupply(ctx context.Context, eventsURL string, maxEpoch uint64, minEpoch uint64, indent string) (uint64, error) {
	if minEpoch > maxEpoch {
		return 0, nil
	}
	fmt.Printf("%sSearching for passing epoch between %d and %d\n", indent, minEpoch, maxEpoch)

	apiTotalSupply, err := invariants.GetIFILTotalSupplyFromAPI(ctx, eventsURL, minEpoch)
	if err != nil {
		return 0, err
	}

	nodeTotalSupply, _, err := invariants.GetIFILTotalSupplyFromNode(ctx, minEpoch)
	if err != nil {
		return 0, err
	}

	if apiTotalSupply.IFILTotalSupply.Cmp(nodeTotalSupply.IFILTotalSupply) == 0 {
		fmt.Printf("%s@%d pass\n", indent, minEpoch)
		splitEpoch := (maxEpoch-minEpoch)/2 + minEpoch + 1

		// Check top half
		topEpoch, err := searchPassingIFILTotalSupply(ctx, eventsURL, maxEpoch, splitEpoch, indent+"  ")
		if err != nil {
			return 0, nil
		}
		if topEpoch != 0 {
			return topEpoch, nil
		}

		// Check bottom half
		bottomEpoch, err := searchPassingIFILTotalSupply(ctx, eventsURL, splitEpoch-1, minEpoch+1, indent+"  ")
		if err != nil {
			return 0, nil
		}
		if bottomEpoch != 0 {
			return bottomEpoch, nil
		}
		return minEpoch, nil
	} else {
		fmt.Printf("%s@%d fail\n", indent, minEpoch)
	}

	return 0, nil
}

func init() {
	flat := newPoolIfilCmd("ifil-total-supply [--epoch <epoch>] [--find-missing]")
	flat.Deprecated = "use `inv pool ifil` instead"
	rootCmd.AddCommand(flat)
	poolCmd.AddCommand(newPoolIfilCmd("ifil [--epoch <epoch>] [--find-missing]"))
}
