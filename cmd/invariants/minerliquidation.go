package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/glifio/go-pools/terminate"
	"github.com/glifio/go-pools/util"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
)

// minerLiquidationCmd represents the minerLiquidation command
var minerLiquidationCmd = &cobra.Command{
	Use:   "miner-liquidation [miner-id] [--agent <id>] [--all] [--random <num>] [--epoch <epoch>]",
	Short: "Compare liquidation values computed using various methods",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// eventsURL := viper.GetString("events_api")

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

		randomMiners, err := cmd.Flags().GetUint64("random")
		if err != nil {
			log.Fatal(err)
		}

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

			err = checkTerminations(ctx, epoch, miner)
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
	},
}

func init() {
	rootCmd.AddCommand(minerLiquidationCmd)
	minerLiquidationCmd.Flags().Uint64("epoch", 0, "Check at epoch")
	minerLiquidationCmd.Flags().Uint64("random", 0, "Randomly select miners")
	minerLiquidationCmd.Flags().Uint64("agent", 0, "Select only miners for a specific agent")
	minerLiquidationCmd.Flags().Bool("all", false, "Select all miners for a specific agent")
}

func checkTerminations(ctx context.Context, epoch uint64, miner address.Address) error {
	fmt.Printf("Checking termination burn for miner %v @%d:\n", miner, epoch)

	lotus := singleton.Lotus()

	ts, err := lotus.Api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), types.EmptyTSK)
	if err != nil {
		return err
	}
	start := time.Now()
	quick, err := terminate.PreviewTerminateSectorsQuick(ctx, &lotus.Api, miner, ts)
	if err != nil {
		return err
	}
	elapsed := time.Since(start).Seconds()
	fmt.Printf("Miner %v @%d: Quick method: %0.3f FIL (%d of %d sectors, offchain, %0.1fs)\n", miner, epoch,
		util.ToFIL(quick.SectorStats.TerminationPenalty), quick.SectorsTerminated, quick.SectorsCount, elapsed)

	errorCh := make(chan error)
	// progressCh := make(chan *terminate.PreviewTerminateSectorsProgress)
	resultCh := make(chan *terminate.PreviewTerminateSectorsReturn)

	start = time.Now()
	// epochStr := fmt.Sprintf("@%d", epoch)
	epochStr := "@head"
	// fmt.Sprintf("@%d", epoch)
	go terminate.PreviewTerminateSectors(ctx, &lotus.Api, miner, epochStr, 0, 0, 0,
		false, false, false, 0, errorCh, nil /* progressCh */, resultCh)

loop:
	for {
		select {
		case result := <-resultCh:
			full := result
			elapsed = time.Since(start).Seconds()
			fmt.Printf("Miner %v @%d: Full method: %0.3f FIL (%d of %d sectors, onchain, %0.1fs)\n", miner, epoch,
				util.ToFIL(full.SectorStats.TerminationPenalty), full.SectorsTerminated, full.SectorsCount, elapsed)
			break loop

			/*
				case progress := <-progressCh:
					fmt.Printf("Progress: %+v\n", progress)
			*/

		case err := <-errorCh:
			log.Fatal(err)
		}
	}

	return nil
}
