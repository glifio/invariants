package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"github.com/glifio/invariants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// agentEconCmd represents the agentEcon command
var agentEconCmd = &cobra.Command{
	Use:   "agent-econ [agent-id] [--all] [--random <num>] [--epoch <epoch>]",
	Short: "Compare the econ values from the API and the node for an agent",
	Args:  cobra.MaximumNArgs(1),
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

		allAgents, err := cmd.Flags().GetBool("all")
		if err != nil {
			log.Fatal(err)
		}

		randomAgents, err := cmd.Flags().GetUint64("random")
		if err != nil {
			log.Fatal(err)
		}

		var failCount int

		if !allAgents && randomAgents == 0 {
			if len(args) != 1 {
				cmd.Usage()
				return
			}

			agentID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				log.Fatal(err)
			}

			agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
			if err != nil {
				log.Fatal(err)
			}

			failed, err := checkAgentEcon(ctx, eventsURL, epoch, agent)
			if err != nil {
				log.Fatal(err)
			}
			if failed {
				failCount++
			}
		} else {
			if len(args) != 0 {
				cmd.Usage()
				return
			}

			agents, err := invariants.GetAgentsFromAPI(ctx, eventsURL)
			if err != nil {
				log.Fatal(err)
			}

			if allAgents {
				if randomAgents > 0 {
					cmd.Usage()
					return
				}
				for _, agent := range agents {
					failed, err := checkAgentEcon(ctx, eventsURL, epoch, &agent)
					if err != nil {
						log.Fatal(err)
					}
					if failed {
						failCount++
					}
				}
			} else if randomAgents > 0 {
				if int(randomAgents) > len(agents) {
					randomAgents = uint64(len(agents))
				}
				rand.Shuffle(len(agents), func(i, j int) {
					agents[i], agents[j] = agents[j], agents[i]
				})
				for i := 0; i < int(randomAgents); i++ {
					agent := agents[i]
					failed, err := checkAgentEcon(ctx, eventsURL, epoch, &agent)
					if err != nil {
						log.Fatal(err)
					}
					if failed {
						failCount++
					}
				}
			} else {
				cmd.Usage()
			}
		}

		if failCount > 0 {
			log.Fatal("FAIL: Econ tests had errors.")
		}
	},
}

func init() {
	rootCmd.AddCommand(agentEconCmd)
	agentEconCmd.Flags().Uint64("epoch", 0, "Check at epoch")
	agentEconCmd.Flags().Uint64("random", 0, "Randomly select agents")
	agentEconCmd.Flags().Bool("all", false, "Check all agents")
}

func checkAgentEcon(ctx context.Context, eventsURL string, epoch uint64, agent *invariants.Agent) (failed bool, err error) {
	agentID := agent.ID

	if epoch == 0 {
		epoch, err = getHeadEpoch(ctx)
		if err != nil {
			return true, err
		}
		epoch = epoch - 3
	}

	econAPI, err := invariants.GetAgentEconFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		return true, err
	}
	// fmt.Printf("Econ api: %+v\n", econAPI)
	econNode, height, err := invariants.GetAgentEconFromNode(ctx, agent.AddressNative, epoch)
	if err != nil {
		return true, err
	}
	// fmt.Printf("Econ node @%d: %+v\n", height, econNode)

	// Mutate for testing
	// econAPI.Liability = big.NewInt(1234)

	if econAPI.Liability.Cmp(econNode.Liability) == 0 {
		fmt.Printf("Agent %d: Success, latest liabilities match: %v\n", agentID, econNode.Liability)
		return false, nil
	} else {
		fmt.Printf("Agent %d: Error, latest liability from REST API doesn't match node.\n", agentID)
		fmt.Printf("  Node @%d: %v\n", height, econNode.Liability)
		fmt.Printf("   API: %v\n", econAPI.Liability)
		return true, nil
	}
}
