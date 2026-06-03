package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	pools "github.com/glifio/go-pools/abigen"
	"github.com/glifio/invariants/singleton"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPoolBisectHolderCmd: binary-search the height where a single
// holder's DB-derived ERC-20 balance first stops matching the
// contract's balanceOf. The output is the first-bad height (or a
// small range) — feed that to `idx refill --from H1 --to H2` to
// re-process events in the gap.
func newPoolBisectHolderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bisect-holder --token <ifil|glf> --addr <0x...>",
		Short: "Binary-search the first height where a holder's DB balance diverges from contract.balanceOf",
		Args:  cobra.NoArgs,
		Run:   runPoolBisectHolder,
	}
	cmd.Flags().String("token", "", "ifil | glf (which ERC-20 to check)")
	cmd.Flags().String("addr", "", "holder address (hex)")
	cmd.Flags().Uint64("lo", 0, "lowest epoch to search (default: 1)")
	cmd.Flags().Uint64("hi", 0, "highest epoch (default: indexer last_processed_height - 3)")
	cmd.Flags().Uint64("tolerance", 0, "treat |db - chain| <= tolerance wei as match")
	return cmd
}

func runPoolBisectHolder(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	postgresURL := viper.GetString("postgres")
	if postgresURL == "" {
		log.Fatal("POSTGRES env var must be set")
	}
	token, _ := cmd.Flags().GetString("token")
	addrStr, _ := cmd.Flags().GetString("addr")
	if token == "" || addrStr == "" {
		log.Fatal("--token and --addr are required")
	}
	addr := common.HexToAddress(addrStr)

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
		log.Fatalf("unknown token %q (use ifil or glf)", token)
	}

	tok, err := pools.NewPoolTokenCaller(contractAddr, ethClient)
	if err != nil {
		log.Fatalf("token caller: %v", err)
	}
	dbConn, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("postgres open: %v", err)
	}
	defer dbConn.Close()

	lo, _ := cmd.Flags().GetUint64("lo")
	hi, _ := cmd.Flags().GetUint64("hi")
	tolUint, _ := cmd.Flags().GetUint64("tolerance")
	tolerance := new(big.Int).SetUint64(tolUint)

	if lo == 0 {
		// Default to the earliest event height for this token so we don't
		// hit the chain at heights before the contract was deployed.
		var minHeight sql.NullInt64
		if err := dbConn.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT MIN(height) FROM %s`, dbTable),
		).Scan(&minHeight); err != nil {
			log.Fatalf("min height: %v", err)
		}
		if !minHeight.Valid {
			log.Fatalf("%s table is empty — nothing to bisect", dbTable)
		}
		lo = uint64(minHeight.Int64)
	}
	if hi == 0 {
		head, err := getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		hi = head - 3
	}

	// Helper: returns true when DB balance differs from chain balance
	// by more than tolerance at h. (false = match.)
	diff := func(h uint64) (bool, error) {
		var dbBalStr string
		err := dbConn.QueryRowContext(ctx,
			fmt.Sprintf(`
				WITH txs AS (
				    SELECT to_   AS addr, amount AS amt FROM %s WHERE height <= $1
				    UNION ALL
				    SELECT from_ AS addr, -amount AS amt FROM %s WHERE height <= $1
				)
				SELECT COALESCE(sum(amt), 0)::TEXT
				FROM txs WHERE addr = $2`, dbTable, dbTable),
			h, addr.Hex(),
		).Scan(&dbBalStr)
		if err != nil {
			return false, fmt.Errorf("db balance: %w", err)
		}
		dbBal, _ := new(big.Int).SetString(dbBalStr, 10)
		chainBal, err := tok.BalanceOf(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(h)}, addr)
		if err != nil {
			return false, fmt.Errorf("chain balance @%d: %w", h, err)
		}
		d := new(big.Int).Sub(dbBal, chainBal)
		d.Abs(d)
		return d.Cmp(tolerance) > 0, nil
	}

	loDiff, err := diff(lo)
	if err != nil {
		log.Fatal(err)
	}
	hiDiff, err := diff(hi)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("addr=%s token=%s\n", addr.Hex(), token)
	fmt.Printf("@%d  %s\n", lo, ternaryStr(loDiff, "DIVERGES", "matches"))
	fmt.Printf("@%d  %s\n", hi, ternaryStr(hiDiff, "DIVERGES", "matches"))

	if loDiff {
		fmt.Println("lo already diverges — extend --lo lower (or fix earlier divergence first)")
		return
	}
	if !hiDiff {
		fmt.Println("hi matches — nothing to bisect (holder is currently consistent)")
		return
	}

	// Bisect: find smallest h in (lo, hi] where it diverges.
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		d, err := diff(mid)
		if err != nil {
			log.Fatal(err)
		}
		if d {
			fmt.Printf("@%d DIVERGES — narrow right\n", mid)
			hi = mid
		} else {
			fmt.Printf("@%d matches — narrow left\n", mid)
			lo = mid
		}
	}
	fmt.Printf("\nFirst divergence at height %d.\n", hi)
	fmt.Printf("Suggested backfill window:\n")
	fmt.Printf("  idx refill --from %d --to %d\n", hi-100, hi+100)
	fmt.Printf("(Adjust the ±100 epoch buffer if events around the divergence span more.)\n")
}

func ternaryStr(b bool, a, c string) string {
	if b {
		return a
	}
	return c
}

func init() {
	poolCmd.AddCommand(newPoolBisectHolderCmd())
}
