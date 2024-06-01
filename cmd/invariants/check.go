package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [epoch]",
	Short: "Compare the metrics from the API and the node at height",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		chainID := viper.GetUint64("chain_id")
		eventsURL := viper.GetString("events_api")

		fmt.Printf("ChainID: %v\n", chainID)
		fmt.Printf("Events URL: %v\n", eventsURL)

		useArchiveNode, err := cmd.Flags().GetBool("archive")
		if err != nil {
			log.Fatal(err)
		}

		err = initSingleton(ctx, useArchiveNode)
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
		//fmt.Printf("Jim rest %+v\n", metricsFromAPI)
		metricsFromNode, err := invariants.GetMetricsFromNode(ctx, height)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("Jim chain %+v\n", metricsFromNode)

		fmt.Printf("REST: PoolTotalAssets %v\n", metricsFromAPI.PoolTotalAssets)
		fmt.Printf("Node: PoolTotalAssets %v\n", metricsFromNode.PoolTotalAssets)

	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("archive", false, "use archive Lotus node")
}
