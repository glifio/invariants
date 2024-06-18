package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics [epoch]",
	Short: "Compare the metrics from the API and the node at height",
	Args:  cobra.ExactArgs(1),
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

		height, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		metricsFromAPI, err := invariants.GetMetricsFromAPIAtHeight(ctx, eventsURL, height)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Jim rest %+v\n", metricsFromAPI)
		metricsFromNode, err := invariants.GetMetricsFromNode(ctx, height)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Jim chain %+v\n", metricsFromNode)

		fmt.Printf("REST: PoolTotalAssets %v\n", metricsFromAPI.PoolTotalAssets)
		fmt.Printf("Node: PoolTotalAssets %v\n", metricsFromNode.PoolTotalAssets)

		fmt.Printf("REST: PoolTotalBorrowed %v\n", metricsFromAPI.PoolTotalBorrowed)
		fmt.Printf("Node: PoolTotalBorrowed %v\n", metricsFromNode.PoolTotalBorrowed)

		fmt.Printf("REST: TotalAgentCount %v\n", metricsFromAPI.TotalAgentCount)
		fmt.Printf("Node: TotalAgentCount %v\n", metricsFromNode.TotalAgentCount)

	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}
