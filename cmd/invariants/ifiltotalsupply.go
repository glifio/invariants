package main

import (
	"fmt"
	"log"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// iFILTotalSupplyCmd represents the check-ifil-total-supply command
var iFILTotalSupplyCmd = &cobra.Command{
	Use:   "ifil-total-supply [--epoch <epoch>]",
	Short: "Compare the iFIL Total Supply from the API and the node",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		eventsURL := viper.GetString("events_api")

		err := initSingleton(ctx, true)
		if err != nil {
			log.Fatal(err)
		}

		epoch, err := cmd.Flags().GetUint64("epoch")
		if err != nil {
			log.Fatal(err)
		}

		if epoch == 0 {
			log.Fatal("Not implemented")
		} else {
			apiTotalSupply, err := invariants.GetIFILTotalSupplyFromAPI(ctx, eventsURL, epoch)
			if err != nil {
				log.Fatal(err)
			}

			nodeTotalSupply, err := invariants.GetIFILTotalSupplyFromNode(ctx, epoch)
			if err != nil {
				log.Fatal(err)
			}

			if apiTotalSupply.IFILTotalSupply.Cmp(nodeTotalSupply.IFILTotalSupply) == 0 {
				fmt.Printf("@%d: Success, iFIL total supply matches: %v\n", epoch, apiTotalSupply.IFILTotalSupply)
				return
			}
			fmt.Printf("@%d: Error, iFIL total supply from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node: %v\n", nodeTotalSupply.IFILTotalSupply)
			fmt.Printf("   API: %v\n", apiTotalSupply.IFILTotalSupply)
		}
	},
}

func init() {
	rootCmd.AddCommand(iFILTotalSupplyCmd)
	iFILTotalSupplyCmd.Flags().Uint64("epoch", 0, "Check at epoch")
}
