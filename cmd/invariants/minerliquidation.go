package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/glifio/go-pools/terminate"
	"github.com/glifio/go-pools/util"
	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// minerLiquidationCmd represents the minerLiquidation command
var minerLiquidationCmd = &cobra.Command{
	Use:   "miner-liquidation [miner-id] [--agent <id>] [--random <num>] [--epoch <epoch>] [--progress] [--timeout duration]",
	Short: "Compare liquidation values computed using various methods",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		timeout, err := cmd.Flags().GetDuration("timeout")
		if err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		eventsURL := viper.GetString("events_api")

		err = initSingleton(ctx)
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

		agentID, err := cmd.Flags().GetUint64("agent")
		if err != nil {
			log.Fatal(err)
		}

		randomMiners, err := cmd.Flags().GetUint64("random")
		if err != nil {
			log.Fatal(err)
		}

		showProgress, err := cmd.Flags().GetBool("progress")
		if err != nil {
			log.Fatal(err)
		}

		allAgents, err := cmd.Flags().GetBool("all-agents")
		if err != nil {
			log.Fatal(err)
		}

		maxPctVariance, err := cmd.Flags().GetFloat64("max-pct-variance")
		if err != nil {
			log.Fatal(err)
		}

		var failCount int

		if allAgents {
			agents, err := invariants.GetAgentsFromAPI(ctx, eventsURL)
			if err != nil {
				log.Fatal(err)
			}

			for _, agent := range agents {
				failed, err := checkTerminationsForAgent(ctx, eventsURL, agent.ID,
					epoch, showProgress, maxPctVariance)
				if err != nil {
					log.Fatal(err)
				}
				if failed {
					failCount++
				}
			}
		} else if agentID != 0 {
			failed, err := checkTerminationsForAgent(ctx, eventsURL, agentID, epoch,
				showProgress, maxPctVariance)
			if err != nil {
				log.Fatal(err)
			}
			if failed {
				failCount++
			}
		} else {
			if randomMiners == 0 {
				if len(args) != 1 {
					cmd.Usage()
					return
				}

				minerID := args[0]

				miner, err := address.NewFromString(minerID)
				if err != nil {
					log.Fatal(err)
				}

				failed, err := checkTerminations(ctx, epoch, miner, nil, nil, "",
					showProgress, maxPctVariance)
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

				if randomMiners > 0 {
					fmt.Println("Loading agents...")
					agents, err := invariants.GetAgentsFromAPI(ctx, eventsURL)
					if err != nil {
						log.Fatal(err)
					}

					type AgentMiner struct {
						agent *invariants.Agent
						miner int
					}
					allMiners := make([]AgentMiner, 0)
					for _, agent := range agents {
						for i := 1; i <= int(agent.Miners); i++ {
							allMiners = append(allMiners, AgentMiner{&agent, i})
						}
					}
					fmt.Printf("%d miners loaded.\n", len(allMiners))

					if int(randomMiners) > len(allMiners) {
						randomMiners = uint64(len(allMiners))
					}
					rand.Shuffle(len(agents), func(i, j int) {
						allMiners[i], allMiners[j] = allMiners[j], allMiners[i]
					})
					for i := 0; i < int(randomMiners); i++ {
						agentMiner := allMiners[i]
						agent := agentMiner.agent
						fmt.Printf("Agent %v @%d: %d miners, %0.3f FIL borrowed (via API)\n",
							agent.ID, agent.Height, agent.Miners, util.ToFIL(agent.PrincipalBalance))

						miners, err := invariants.GetAgentMinersFromAPI(ctx, eventsURL, agent.ID)
						if err != nil {
							log.Fatal(err)
						}
						for i, miner := range miners {
							if i == agentMiner.miner-1 {
								countStr := fmt.Sprintf("%d/%d", i+1, len(miners))
								failed, err := checkTerminations(ctx, epoch, miner.MinerAddr,
									agent, &miner, countStr, showProgress, maxPctVariance)
								if err != nil {
									log.Fatal(err)
								}
								if failed {
									failCount++
								}
							}
						}
					}
				} else {
					cmd.Usage()
				}
			}
		}
		if failCount > 0 {
			log.Fatal("FAIL: Miner liquidation test had errors.")
		}
	},
}

func init() {
	rootCmd.AddCommand(minerLiquidationCmd)
	minerLiquidationCmd.Flags().Uint64("epoch", 0, "Check at epoch")
	minerLiquidationCmd.Flags().Uint64("random", 0, "Randomly select miners")
	minerLiquidationCmd.Flags().Uint64("agent", 0, "Select only miners for a specific agent")
	minerLiquidationCmd.Flags().Bool("all-agents", false, "Loop over all agents")
	minerLiquidationCmd.Flags().Bool("progress", true, "Show progress bar")
	minerLiquidationCmd.Flags().Duration("timeout", time.Duration(15*time.Minute), "Stop query after timeout")
	minerLiquidationCmd.Flags().Float64("max-pct-variance", 5.0, "Acceptable percentage difference between quick and full methods")
}

func checkTerminationsForAgent(
	ctx context.Context,
	eventsURL string,
	agentID uint64,
	epoch uint64,
	showProgress bool,
	maxPctVariance float64,
) (failed bool, err error) {
	agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		return true, err
	}
	fmt.Printf("Agent %v @%d: %d miners, %0.3f FIL borrowed (via API)\n",
		agent.ID, agent.Height, agent.Miners, util.ToFIL(agent.PrincipalBalance))

	miners, err := invariants.GetAgentMinersFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		return true, err
	}
	var failCount int
	for i, miner := range miners {
		countStr := fmt.Sprintf("%d/%d", i+1, len(miners))
		failed, err = checkTerminations(ctx, epoch, miner.MinerAddr, agent, &miner,
			countStr, showProgress, maxPctVariance)
		if err != nil {
			return true, err
		}
		if failed {
			failCount++
		}
	}
	if failCount > 0 {
		return true, nil
	}
	return false, nil
}

func checkTerminations(
	ctx context.Context,
	epoch uint64,
	miner address.Address,
	agent *invariants.Agent,
	minerDetails *invariants.MinerDetailsResult,
	countStr string,
	showProgress bool,
	maxPctVariance float64,
) (failed bool, err error) {
	if countStr != "" {
		countStr += " "
	}
	prefix := ""
	if agent == nil {
		fmt.Printf("Checking termination burn for miner %v @%d:\n", miner, epoch)
	} else {
		prefix = fmt.Sprintf("  Agent %d: ", agent.ID)
	}

	lotus := singleton.Lotus()

	ts, err := lotus.Api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), types.EmptyTSK)
	if err != nil {
		return true, err
	}

	var failCount int

	// Quick
	start := time.Now()
	quickResult, err := terminate.PreviewTerminateSectorsQuick(ctx, &lotus.Api, miner, ts)
	if err != nil {
		return true, err
	}
	elapsed := time.Since(start).Seconds()
	fmt.Printf("%sMiner %s%v @%d: Quick method: %0.3f FIL (%d of %d sectors, offchain, %0.1fs)\n",
		prefix, countStr, miner, epoch, util.ToFIL(quickResult.SectorStats.TerminationPenalty),
		quickResult.SectorsTerminated, quickResult.SectorsCount, elapsed)

	// Sampled, onchain
	var sampledResult *terminate.PreviewTerminateSectorsReturn
	errorCh := make(chan error)
	// progressCh := make(chan *terminate.PreviewTerminateSectorsProgress)
	resultCh := make(chan *terminate.PreviewTerminateSectorsReturn)
	epochStr := fmt.Sprintf("@%d", epoch)
	start = time.Now()
	go terminate.PreviewTerminateSectors(
		ctx,
		&lotus.Api,
		miner,
		epochStr,
		0,            // vmHeight
		40,           // batchSize
		270000000000, // gasLimit
		true,         // useSampling
		true,         // optimize
		false,        // offchain
		21,           // maxPartitions
		errorCh, nil /* progressCh */, resultCh)

loopSampled:
	for {
		select {
		case result := <-resultCh:
			sampledResult = result
			elapsed = time.Since(start).Seconds()
			fmt.Printf("%sMiner %s%v @%d: Sampled method: %0.3f FIL (%d of %d sectors, onchain, %0.1fs)\n",
				prefix, countStr, miner, epoch, util.ToFIL(sampledResult.SectorStats.TerminationPenalty),
				sampledResult.SectorsTerminated, sampledResult.SectorsCount, elapsed)
			break loopSampled

			/*
				case progress := <-progressCh:
					fmt.Printf("Progress: %+v\n", progress)
			*/

		case err := <-errorCh:
			log.Fatal(err)
		}
	}

	if sampledResult.SectorStats.TerminationPenalty.Cmp(quickResult.SectorStats.TerminationPenalty) != 0 {
		fmt.Printf("%sMiner %v%v: Assertion failed: Quick vs Sampled don't match\n",
			prefix, countStr, miner)
		failCount++
	}

	// Full
	var fullResult *terminate.PreviewTerminateSectorsReturn
	var bar *progressbar.ProgressBar
	defer func() {
		if bar != nil {
			bar.Close()
		}
	}()
	errorCh = make(chan error)
	progressCh := make(chan *terminate.PreviewTerminateSectorsProgress)
	resultCh = make(chan *terminate.PreviewTerminateSectorsReturn)
	start = time.Now()
	go terminate.PreviewTerminateSectors(ctx, &lotus.Api, miner, epochStr, 0, 0, 0,
		false, false, false, 0, errorCh, progressCh, resultCh)
	var lastDeadlinePartIdx int = -1

loopFull:
	for {
		select {
		case result := <-resultCh:
			fullResult = result
			elapsedDuration := time.Since(start).Round(time.Second)
			if bar != nil {
				bar.Close()
				bar = nil
			}
			fmt.Printf("%sMiner %s%v @%d: Full method: %0.3f FIL (%d of %d sectors, onchain, %s)\n",
				prefix, countStr, miner, epoch, util.ToFIL(fullResult.SectorStats.TerminationPenalty),
				fullResult.SectorsTerminated, fullResult.SectorsCount, elapsedDuration)
			break loopFull

		case progress := <-progressCh:
			if showProgress && bar == nil && progress.DeadlinePartitionCount > 0 {
				bar = progressbar.NewOptions(progress.DeadlinePartitionCount,
					progressbar.OptionSetDescription("Partitions"),
					progressbar.OptionSetWriter(os.Stderr),
					progressbar.OptionSetWidth(10),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionShowCount(),
					progressbar.OptionShowIts(),
					progressbar.OptionSpinnerType(14),
					progressbar.OptionFullWidth(),
					progressbar.OptionSetRenderBlankState(true),
					progressbar.OptionClearOnFinish())
			}
			if bar != nil && progress.DeadlinePartitionIndex != lastDeadlinePartIdx {
				lastDeadlinePartIdx = progress.DeadlinePartitionIndex
				bar.Add(1)
			}

		case err := <-errorCh:
			log.Fatal(err)
		}
	}

	if bar != nil {
		bar.Close()
		// fmt.Println()
		bar = nil
	}

	if minerDetails != nil {
		apiDiff := new(big.Int).Sub(minerDetails.TerminationPenalty, quickResult.SectorStats.TerminationPenalty)
		// For testing assertion
		// apiDiff, _ = new(big.Int).SetString("650000000000000000", 10)
		fmt.Printf("%sMiner %s%v: Termination penalty via API: %0.3f FIL\n",
			prefix, countStr, miner, util.ToFIL(minerDetails.TerminationPenalty))

		// Assert that db value from API is withing range
		pctApi, _ := getPct(apiDiff, fullResult.SectorStats.TerminationPenalty, agent)
		if pctApi > maxPctVariance {
			fmt.Printf("%sMiner %v%v: Assertion failed: API vs Full %0.3f%% > %0.3f%%\n",
				prefix, countStr, miner, pctApi, maxPctVariance)
			failCount++
		}
	}

	// Variances
	fullVsQuick := new(big.Int).Sub(
		fullResult.SectorStats.TerminationPenalty,
		quickResult.SectorStats.TerminationPenalty,
	)

	if fullVsQuick.Sign() == 0 {
		fmt.Printf("%sMiner %s%v: Quick method and Full method agree (%d/%d sectors).\n",
			prefix, countStr, miner, quickResult.SectorsTerminated, quickResult.SectorsCount)
	} else {
		var pctNum float64
		var pctStr string
		if fullVsQuick.Sign() == -1 {
			fullVsQuick = new(big.Int).Abs(fullVsQuick)
			pctNum, pctStr = getPct(fullVsQuick, fullResult.SectorStats.TerminationPenalty, agent)
			fmt.Printf("%sMiner %s%v: Quick method overestimated: %0.3f FIL (%s, %d/%d sectors)\n",
				prefix, countStr, miner, util.ToFIL(fullVsQuick), pctStr,
				quickResult.SectorsTerminated, quickResult.SectorsCount)
		} else {
			pctNum, pctStr = getPct(fullVsQuick, fullResult.SectorStats.TerminationPenalty, agent)
			fmt.Printf("%sMiner %s%v: Quick method UNDERESTIMATED: %0.3f FIL (%s, %d/%d sectors)\n",
				prefix, countStr, miner, util.ToFIL(fullVsQuick), pctStr,
				quickResult.SectorsTerminated, quickResult.SectorsCount)
		}
		if pctNum > maxPctVariance {
			fmt.Printf("%sMiner %v%v: Assertion failed: Quick vs Full diff %0.3f%% > %0.3f%%\n",
				prefix, countStr, miner, pctNum, maxPctVariance)
			failCount++
		}
	}

	if failCount > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func getPct(diffBig *big.Int, referenceBig *big.Int, agent *invariants.Agent) (pctNum float64, pctStr string) {
	diffBig = new(big.Int).Abs(diffBig)
	diff, _ := diffBig.Float64()
	reference, _ := referenceBig.Float64()
	pctStr = "n/a"
	if reference > 0 {
		pctNum = diff / reference * 100
		pctStr = fmt.Sprintf("%0.3f%%", pctNum)
	}
	if agent != nil && agent.PrincipalBalance.Sign() == 1 {
		loaned, _ := agent.PrincipalBalance.Float64()
		pctStr += fmt.Sprintf(", %0.3f%% of agent principal", diff/loaned*100)
	}
	return pctNum, pctStr
}
