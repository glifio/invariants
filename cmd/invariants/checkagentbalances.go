package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// checkAgentBalancesCmd represents the checkAgentBalance command
var checkAgentBalancesCmd = &cobra.Command{
	Use:   "check-agent-balances [agent-id] [--epoch <epoch>]",
	Short: "Compare the balances from the API and the node for an agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		eventsURL := viper.GetString("events_api")

		err := initSingleton(ctx, true)
		if err != nil {
			log.Fatal(err)
		}

		agent, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		epoch, err := cmd.Flags().GetUint64("epoch")
		if err != nil {
			log.Fatal(err)
		}

		if epoch == 0 {
			availableBalanceResult, err := invariants.GetAgentAvailableBalanceFromAPI(ctx, eventsURL, agent)
			if err != nil {
				log.Fatal(err)
			}
			if availableBalanceResult.AvailableBalanceDB.Cmp(availableBalanceResult.AvailableBalanceNd) == 0 {
				fmt.Printf("Agent %d: Success, latest available balances match: %v\n", agent, availableBalanceResult.AvailableBalanceDB)
				return
			}
			fmt.Printf("Agent %d: Error, latest available balance from REST API doesn't match node.\n", agent)
			fmt.Printf("  Node: %v\n", availableBalanceResult.AvailableBalanceNd)
			fmt.Printf("   API: %v\n", availableBalanceResult.AvailableBalanceDB)
		} else {
			availableBalance, err := invariants.GetAgentAvailableBalanceAtHeightFromAPI(ctx, eventsURL, agent, epoch)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Jim balance: %v\n", availableBalance)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkAgentBalancesCmd)
	checkAgentBalancesCmd.Flags().Uint64("epoch", 0, "Check at epoch")
}
