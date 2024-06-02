package main

import (
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// agentBalancesCmd represents the checkAgentBalance command
var agentBalancesCmd = &cobra.Command{
	Use:   "agent-balances [agent-id] [--epoch <epoch>]",
	Short: "Compare the balances from the API and the node for an agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		eventsURL := viper.GetString("events_api")

		err := initSingleton(ctx, true)
		if err != nil {
			log.Fatal(err)
		}

		agentID, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			log.Fatal(err)
		}

		epoch, err := cmd.Flags().GetUint64("epoch")
		if err != nil {
			log.Fatal(err)
		}

		if epoch == 0 {
			availableBalanceResult, err := invariants.GetAgentAvailableBalanceFromAPI(ctx, eventsURL, agentID)
			if err != nil {
				log.Fatal(err)
			}
			if availableBalanceResult.AvailableBalanceDB.Cmp(availableBalanceResult.AvailableBalanceNd) == 0 {
				fmt.Printf("Agent %d: Success, latest available balances match: %v\n", agentID, availableBalanceResult.AvailableBalanceDB)
				return
			}
			fmt.Printf("Agent %d: Error, latest available balance from REST API doesn't match node.\n", agentID)
			fmt.Printf("  Node: %v\n", availableBalanceResult.AvailableBalanceNd)
			fmt.Printf("   API: %v\n", availableBalanceResult.AvailableBalanceDB)
		} else {
			availableBalance, err := invariants.GetAgentAvailableBalanceAtHeightFromAPI(ctx, eventsURL, agentID, epoch)
			if err != nil {
				log.Fatal(err)
			}

			agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
			if err != nil {
				log.Fatal(err)
			}

			nextEpoch, err := getNextEpoch(ctx, epoch)
			if err != nil {
				log.Fatal(err)
			}

			q := singleton.PoolsArchiveSDK.Query()
			liquidAssets, err := q.AgentLiquidAssets(ctx, agent.AddressNative, big.NewInt(int64(nextEpoch)))
			if err != nil {
				log.Fatal(err)
			}

			if availableBalance.Cmp(liquidAssets) == 0 {
				fmt.Printf("Agent %d @%d: Success, latest available balances match: %v\n", agentID, epoch, availableBalance)
				return
			}
			fmt.Printf("Agent %d @%d: Error, available balance from REST API doesn't match node.\n", agentID, epoch)
			fmt.Printf("  Node: %v\n", liquidAssets)
			fmt.Printf("   API: %v\n", availableBalance)
		}
	},
}

func init() {
	rootCmd.AddCommand(agentBalancesCmd)
	agentBalancesCmd.Flags().Uint64("epoch", 0, "Check at epoch")
}
