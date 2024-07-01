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
			epoch = epoch - 3
		}

		checkMinerCount, err := cmd.Flags().GetBool("miner-count")
		if err != nil {
			log.Fatal(err)
		}

		metricsFromAPI, err := invariants.GetMetricsFromAPIAtHeight(ctx, eventsURL, epoch)
		if err != nil {
			log.Fatal(err)
		}
		metricsFromNode, resultEpoch, err := invariants.GetMetricsFromNode(ctx, epoch)
		if err != nil {
			log.Fatal(err)
		}
		var minerCountFromNode uint64
		if checkMinerCount {
			minerCountFromNode, resultEpoch, err = invariants.GetMinerCountFromNode(ctx, epoch)
			if err != nil {
				log.Fatal(err)
			}
		}

		fail := false

		if metricsFromAPI.PoolTotalAssets.Cmp(metricsFromNode.PoolTotalAssets) == 0 {
			fmt.Printf("@%d: Success, pool total assets matches: %v\n", epoch, metricsFromAPI.PoolTotalAssets)
		} else {
			fmt.Printf("@%d: Error, pool total assets from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node @%d: %v\n", resultEpoch, metricsFromNode.PoolTotalAssets)
			fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.PoolTotalAssets)
			fail = true
		}

		if metricsFromAPI.PoolTotalBorrowed.Cmp(metricsFromNode.PoolTotalBorrowed) == 0 {
			fmt.Printf("@%d: Success, pool total borrowed matches: %v\n", epoch, metricsFromAPI.PoolTotalBorrowed)
		} else {
			fmt.Printf("@%d: Error, pool total borrowed from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node @%d: %v\n", resultEpoch, metricsFromNode.PoolTotalBorrowed)
			fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.PoolTotalBorrowed)
			fail = true
		}

		if metricsFromAPI.TotalAgentCount == metricsFromNode.TotalAgentCount {
			fmt.Printf("@%d: Success, agent count matches: %v\n", epoch, metricsFromAPI.TotalAgentCount)
		} else {
			fmt.Printf("@%d: Error, agent count from REST API doesn't match node.\n", epoch)
			fmt.Printf("  Node @%d: %v\n", resultEpoch, metricsFromNode.TotalAgentCount)
			fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.TotalAgentCount)
			fail = true
		}

		if checkMinerCount {
			if metricsFromAPI.TotalMinersCount == minerCountFromNode {
				fmt.Printf("@%d: Success, miner count matches: %v\n", epoch, minerCountFromNode)
			} else {
				fmt.Printf("@%d: Error, miner count from REST API doesn't match node.\n", epoch)
				fmt.Printf("  Node @%d: %v\n", resultEpoch, minerCountFromNode)
				fmt.Printf("   API @%d: %v\n", epoch, metricsFromAPI.TotalMinersCount)
				fail = true
			}
		}

		if fail {
			log.Fatal("FAIL: Metrics tests had errors.")
		}
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().Uint64("epoch", 0, "Check at epoch")
	metricsCmd.Flags().Bool("miner-count", false, "Check miner count (slow)")
}
