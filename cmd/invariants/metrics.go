package main

import (
	"fmt"
	"log"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics [--epoch <epoch>]",
	Short: "Compare the metrics from the API and the node at height",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		chainID := viper.GetUint64("chain_id")
		eventsURL := viper.GetString("events_api")

		fmt.Printf("ChainID: %v\n", chainID)
		fmt.Printf("Events URL: %v\n", eventsURL)

		err := initSingleton(ctx)
		if err != nil {
			log.Fatal(err)
		}

		epoch, err := cmd.Flags().GetUint64("epoch")
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

		metricsFromAPI, err := invariants.GetMetricsFromAPIAtHeight(ctx, eventsURL, epoch)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("Jim rest %+v\n", metricsFromAPI)
		metricsFromNode, err := invariants.GetMetricsFromNode(ctx, epoch)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("Jim chain %+v\n", metricsFromNode)

		fail := false

		if metricsFromAPI.PoolTotalAssets.Cmp(metricsFromNode.PoolTotalAssets) == 0 {
			fmt.Printf("@%d: Success, pool total assets matches: %v\n", epoch, metricsFromAPI.PoolTotalAssets)
		} else {
			fmt.Printf("@%d: Error, pool total assets from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node: %v\n", metricsFromNode.PoolTotalAssets)
			fmt.Printf("   API: %v\n", metricsFromAPI.PoolTotalAssets)
			fail = true
		}

		if metricsFromAPI.PoolTotalBorrowed.Cmp(metricsFromNode.PoolTotalBorrowed) == 0 {
			fmt.Printf("@%d: Success, pool total borrowed matches: %v\n", epoch, metricsFromAPI.PoolTotalBorrowed)
		} else {
			fmt.Printf("@%d: Error, pool total borrowed from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node: %v\n", metricsFromNode.PoolTotalBorrowed)
			fmt.Printf("   API: %v\n", metricsFromAPI.PoolTotalBorrowed)
			fail = true
		}

		if metricsFromAPI.TotalAgentCount == metricsFromNode.TotalAgentCount {
			fmt.Printf("@%d: Success, agent count matches: %v\n", epoch, metricsFromAPI.TotalAgentCount)
		} else {
			fmt.Printf("@%d: Error, agent count from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node: %v\n", metricsFromNode.TotalAgentCount)
			fmt.Printf("   API: %v\n", metricsFromAPI.TotalAgentCount)
			fail = true
		}

		if fail {
			log.Fatal("FAIL: Metrics tests had errors.")
		}
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().Uint64("epoch", 0, "Check at epoch")
}
