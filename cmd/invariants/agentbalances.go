package main

import (
	"context"
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
			examineTransactionHistory(ctx, eventsURL, agentID)
		} else {
			availableBalance, err := invariants.GetAgentAvailableBalanceAtHeightFromAPI(ctx, eventsURL, agentID, epoch)
			if err != nil {
				log.Fatal(err)
			}

			agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
			if err != nil {
				log.Fatal(err)
			}

			liquidAssets, err := getLiquidAssetsAtHeight(ctx, agent, epoch)
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

func examineTransactionHistory(ctx context.Context, eventsURL string, agentID uint64) {
	fmt.Println("Examining transaction history...")
	agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		log.Fatal(err)
	}
	txs, err := invariants.GetAgentTransactionsFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d transactions retrieved from REST API\n", len(txs))
	if len(txs) > 0 {
		// First
		tx := txs[0]
		firstIdx := 0
		fmt.Printf("First tx (idx:0) @%d: ", tx.Height)
		liquidAssets, err := getLiquidAssetsAtHeight(ctx, agent, tx.Height)
		if err != nil {
			log.Fatal(err)
		}
		if tx.AvailableBalance.Cmp(liquidAssets) == 0 {
			fmt.Printf("Matches: %v\n", liquidAssets)
		} else {
			fmt.Printf("Mismatch! Node: %v API: %v\n", liquidAssets, tx.AvailableBalance)
			return
		}

		// Last
		if len(txs) == 1 {
			fmt.Println("Only one transaction.")
			return
		}
		idx := len(txs) - 1
		tx = txs[idx]
		lastIdx := idx
		fmt.Printf("Last tx (idx:%d) @%d: ", idx, tx.Height)
		liquidAssets, err = getLiquidAssetsAtHeight(ctx, agent, tx.Height)
		if err != nil {
			log.Fatal(err)
		}
		if tx.AvailableBalance.Cmp(liquidAssets) == 0 {
			fmt.Printf("Matches: %v\n", liquidAssets) // Shouldn't happen
			return
		} else {
			fmt.Printf("Mismatch! Node: %v API: %v\n", liquidAssets, tx.AvailableBalance)
			binarySearch(ctx, agent, txs, firstIdx, lastIdx)
		}
	}
}

func binarySearch(
	ctx context.Context,
	agent *invariants.Agent,
	txs []invariants.Transaction,
	goodIdx int,
	badIdx int,
) {
	fmt.Printf("Binary searching between %d and %d\n", goodIdx, badIdx)
	searchIdx := (goodIdx + badIdx) / 2
	if searchIdx == goodIdx || searchIdx == badIdx {
		fmt.Printf("Last good tx via API (idx: %d) @%d: %v\n", goodIdx, txs[goodIdx].Height, txs[goodIdx].AvailableBalance)
		fmt.Printf("First bad tx via API (idx: %d) @%d\n", badIdx, txs[badIdx].Height)
		return
	}
	tx := txs[searchIdx]
	fmt.Printf("Tx (idx:%d) @%d: ", searchIdx, tx.Height)
	liquidAssets, err := getLiquidAssetsAtHeight(ctx, agent, tx.Height)
	if err != nil {
		log.Fatal(err)
	}
	if tx.AvailableBalance.Cmp(liquidAssets) == 0 {
		fmt.Printf("Matches: %v\n", liquidAssets)
		binarySearch(ctx, agent, txs, searchIdx, badIdx)
	} else {
		fmt.Printf("Mismatch! Node: %v API: %v\n", liquidAssets, tx.AvailableBalance)
		binarySearch(ctx, agent, txs, goodIdx, searchIdx)
	}
}

func getLiquidAssetsAtHeight(ctx context.Context, agent *invariants.Agent, height uint64) (*big.Int, error) {
	nextEpoch, err := getNextEpoch(ctx, height)
	if err != nil {
		return nil, err
	}

	q := singleton.PoolsArchiveSDK.Query()
	liquidAssets, err := q.AgentLiquidAssets(ctx, agent.AddressNative, big.NewInt(int64(nextEpoch)))
	if err != nil {
		return nil, err
	}

	return liquidAssets, nil
}
