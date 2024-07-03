package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
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
	Use:   "miner-liquidation [miner-id] [--agent <id>] [--random <num>] [--epoch <epoch>] [--progress]",
	Short: "Compare liquidation values computed using various methods",
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

		if agentID != 0 {
			agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Agent %v @%d: %d miners, %0.3f FIL borrowed (via API)\n",
				agent.ID, agent.Height, agent.Miners, util.ToFIL(agent.PrincipalBalance))

			miners, err := invariants.GetAgentMinersFromAPI(ctx, eventsURL, agentID)
			if err != nil {
				log.Fatal(err)
			}
			for i, miner := range miners {
				countStr := fmt.Sprintf("%d/%d", i+1, len(miners))
				err = checkTerminations(ctx, epoch, miner.MinerAddr, agent, &miner, countStr, showProgress)
				if err != nil {
					log.Fatal(err)
				}
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

				err = checkTerminations(ctx, epoch, miner, nil, nil, "", showProgress)
				if err != nil {
					log.Fatal(err)
				}
				/*
					agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
					if err != nil {
						log.Fatal(err)
					}

					err = checkAgentEcon(ctx, eventsURL, epoch, agent)
					if err != nil {
						log.Fatal(err)
					}
				*/
			} else {
				if len(args) != 0 {
					cmd.Usage()
					return
				}

				if randomMiners > 0 {
					/*
						if int(randomAgents) > len(agents) {
							randomAgents = uint64(len(agents))
						}
						rand.Shuffle(len(agents), func(i, j int) {
							agents[i], agents[j] = agents[j], agents[i]
						})
						for i := 0; i < int(randomAgents); i++ {
							agent := agents[i]
							err := checkAgentEcon(ctx, eventsURL, epoch, &agent)
							if err != nil {
								log.Fatal(err)
							}
						}
					*/
				} else {
					cmd.Usage()
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(minerLiquidationCmd)
	minerLiquidationCmd.Flags().Uint64("epoch", 0, "Check at epoch")
	minerLiquidationCmd.Flags().Uint64("random", 0, "Randomly select miners")
	minerLiquidationCmd.Flags().Uint64("agent", 0, "Select only miners for a specific agent")
	minerLiquidationCmd.Flags().Bool("progress", true, "Show progress bar")
}

func checkTerminations(
	ctx context.Context,
	epoch uint64,
	miner address.Address,
	agent *invariants.Agent,
	minerDetails *invariants.MinerDetailsResult,
	countStr string,
	showProgress bool,
) error {
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
		return err
	}

	// Quick
	start := time.Now()
	quickResult, err := terminate.PreviewTerminateSectorsQuick(ctx, &lotus.Api, miner, ts)
	if err != nil {
		return err
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
				// fmt.Println()
				bar = nil
			}
			fmt.Printf("%sMiner %s%v @%d: Full method: %0.3f FIL (%d of %d sectors, onchain, %s)\n",
				prefix, countStr, miner, epoch, util.ToFIL(fullResult.SectorStats.TerminationPenalty),
				fullResult.SectorsTerminated, fullResult.SectorsCount, elapsedDuration)
			break loopFull

		case progress := <-progressCh:
			// fmt.Printf("Progress: %+v\n", progress)
			if showProgress && bar == nil && progress.DeadlinePartitionCount > 0 {
				bar = progressbar.NewOptions(progress.DeadlinePartitionCount,
					progressbar.OptionSetDescription("Partitions"),
					progressbar.OptionSetWriter(os.Stderr),
					progressbar.OptionSetWidth(10),
					progressbar.OptionThrottle(65*time.Millisecond),
					progressbar.OptionShowCount(),
					progressbar.OptionShowIts(),
					/*
						OptionOnCompletion(func() {
							fmt.Fprint(os.Stderr, "\n")
						}),
					*/
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
		fmt.Printf("%sMiner %s%v: Termination penalty via API: %0.3f FIL\n",
			prefix, countStr, miner, util.ToFIL(minerDetails.TerminationPenalty))
	}

	// Variances
	fullVsQuick := new(big.Int).Sub(
		fullResult.SectorStats.TerminationPenalty,
		quickResult.SectorStats.TerminationPenalty,
	)
	if fullVsQuick.Sign() == 0 {
		fmt.Printf("%sMiner %s%v: Quick method and Full method agree.",
			prefix, countStr, miner)
	} else if fullVsQuick.Sign() == -1 {
		pct := getPct(fullVsQuick, fullResult.SectorStats.TerminationPenalty, agent)
		fmt.Printf("%sMiner %s%v: Quick method overestimated: %0.3f FIL (%s)\n",
			prefix, countStr, miner, util.ToFIL(fullVsQuick), pct)
	} else {
		pct := getPct(fullVsQuick, fullResult.SectorStats.TerminationPenalty, agent)
		fmt.Printf("%sMiner %s%v: Quick method UNDERESTIMATED: %0.3f FIL (%s)\n",
			prefix, countStr, miner, util.ToFIL(fullVsQuick), pct)
	}

	return nil
}

func getPct(fullVsQuick *big.Int, fullBig *big.Int, agent *invariants.Agent) string {
	fullVsQuick = new(big.Int).Abs(fullVsQuick)
	diff, _ := fullVsQuick.Float64()
	full, _ := fullBig.Float64()
	pct := "n/a"
	if full > 0 {
		pct = fmt.Sprintf("%0.3f%%", diff/full*100)
	}
	if agent != nil && agent.PrincipalBalance.Sign() == 1 {
		loaned, _ := agent.PrincipalBalance.Float64()
		pct += fmt.Sprintf(", %0.3f%% of agent principal", diff/loaned*100)
	}
	return pct
}
