package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newPoolGLFCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare GLF totalSupply and per-holder balances between DB and contract",
		Args:  cobra.NoArgs,
		Run:   runPoolGLF,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch (default: API last_processed_height-3)")
	cmd.Flags().Bool("per-holder", false,
		"Also reconcile every GLF holder's balance vs token.balanceOf (~2-3k holders, ~30s with concurrency=16)")
	cmd.Flags().Int("concurrency", 16, "parallel balanceOf workers (--per-holder only)")
	cmd.Flags().Bool("include-zero", false,
		"Include holders whose DB balance is zero (default: skip)")
	return cmd
}

func runPoolGLF(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set for GLF checks")
	}

	glfAddrStr := viper.GetString("glf_addr")
	if glfAddrStr == "" {
		log.Fatal("GLF_ADDR must be set in mainnet.env for GLF checks")
	}
	glfAddr := common.HexToAddress(glfAddrStr)

	if err := initSingleton(ctx); err != nil {
		log.Fatal(err)
	}
	sdk := singleton.PoolsSDK

	epoch, _ := cmd.Flags().GetUint64("epoch")
	if epoch == 0 {
		var err error
		epoch, err = getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		epoch -= 3
	}

	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		log.Fatalf("eth client: %v", err)
	}
	defer ethClient.Close()

	// Total supply.
	contractTotal, err := invariants.FetchGLFTotalSupply(ctx, ethClient, glfAddr, epoch)
	if err != nil {
		log.Fatalf("contract totalSupply: %v", err)
	}
	dbTotal, err := invariants.FetchGLFTotalSupplyFromDB(ctx, postgresURL, epoch)
	if err != nil {
		log.Fatalf("DB totalSupply: %v", err)
	}
	totalMatch := dbTotal.Cmp(contractTotal) == 0
	if totalMatch {
		fmt.Printf("@%d: Success, GLF total supply matches: %v\n", epoch, dbTotal)
	} else {
		fmt.Printf("@%d: Error, GLF total supply doesn't match.\n", epoch)
		fmt.Printf("  contract: %v\n", contractTotal)
		fmt.Printf("  DB:       %v\n", dbTotal)
		fmt.Printf("  diff:     %v wei (DB - contract)\n", new(big.Int).Sub(dbTotal, contractTotal))
	}

	perHolder, _ := cmd.Flags().GetBool("per-holder")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	includeZero, _ := cmd.Flags().GetBool("include-zero")

	mismatch := 0
	if perHolder {
		mismatch = runGLFPerHolder(ctx, ethClient, glfAddr, postgresURL, epoch, concurrency, includeZero)
	}

	if !totalMatch || mismatch > 0 {
		log.Fatalf("FAIL: GLF invariants — total supply match=%v, per-holder mismatches=%d", totalMatch, mismatch)
	}
}

func runGLFPerHolder(
	ctx context.Context, ethClient *ethclient.Client, glfAddr common.Address, postgresURL string,
	epoch uint64, concurrency int, includeZero bool,
) int {
	dbBals, err := invariants.FetchGLFHolderBalancesFromDB(ctx, postgresURL, epoch, includeZero)
	if err != nil {
		log.Fatalf("DB holder fetch: %v", err)
	}
	addresses := make([]common.Address, 0, len(dbBals))
	for a := range dbBals {
		addresses = append(addresses, a)
	}
	sort.Slice(addresses, func(i, j int) bool { return addresses[i].Hex() < addresses[j].Hex() })

	fmt.Printf("\n@%d: per-holder — checking %d GLF holders (concurrency=%d include-zero=%v)\n",
		epoch, len(addresses), concurrency, includeZero)

	chainBals, err := invariants.FetchGLFHolderBalancesFromContract(ctx, ethClient, glfAddr, epoch, addresses, concurrency)
	if err != nil {
		log.Fatalf("contract holder fetch: %v", err)
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
	fmt.Printf("\nPer-holder summary: %d/%d holders match, %d mismatches\n",
		len(addresses)-mismatch, len(addresses), mismatch)
	return mismatch
}

func init() {
	poolCmd.AddCommand(newPoolGLFCmd("glf [--epoch <epoch>] [--per-holder]"))
}
