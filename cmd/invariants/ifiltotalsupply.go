package main

import (
	"context"
	"fmt"
	"log"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// iFILTotalSupplyCmd represents the check-ifil-total-supply command
var iFILTotalSupplyCmd = &cobra.Command{
	Use:   "ifil-total-supply [--epoch <epoch>] [--find-missing]",
	Short: "Compare the iFIL Total Supply from the API and the node",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
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

		if apiTotalSupply.IFILTotalSupply.Cmp(nodeTotalSupply.IFILTotalSupply) == 0 {
			fmt.Printf("@%d: Success, iFIL total supply matches: %v\n", epoch, apiTotalSupply.IFILTotalSupply)
			return
		}
		fmt.Printf("@%d: Error, iFIL total supply from REST API doesn't match node.\n", epoch)
		fmt.Printf("  Node @%d: %v\n", resultEpoch, nodeTotalSupply.IFILTotalSupply)
		fmt.Printf("   API @%d: %v\n", epoch, apiTotalSupply.IFILTotalSupply)
		if findMissing {
			findMissingIFILEvents(ctx, eventsURL, epoch)
		}
	},
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
	rootCmd.AddCommand(iFILTotalSupplyCmd)
	iFILTotalSupplyCmd.Flags().Uint64("epoch", 0, "Check at epoch")
	iFILTotalSupplyCmd.Flags().Bool("find-missing", false, "Find missing transactions")
}
