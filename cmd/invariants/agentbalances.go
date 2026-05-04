package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/glifio/invariants"
	"github.com/glifio/invariants/singleton"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newAgentBalancesCmd builds a fresh cobra.Command for the agent
// balance check. We construct it via a factory so the same logic can
// be registered both at the deprecated flat name (`inv agent-balances`)
// and under the `agent` subcommand group (`inv agent balances`).
func newAgentBalancesCmd(use string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Compare the balances from the API and the node for an agent",
		Args:  cobra.MaximumNArgs(1),
		Run:   runAgentBalances,
	}
	cmd.Flags().Uint64("epoch", 0, "Check at epoch")
	cmd.Flags().Uint64("random", 0, "Randomly select agents")
	cmd.Flags().Bool("all", false, "Check all agents")
	cmd.Flags().Uint64("max-lookback", 30000, "Skip transactions older than head minus this many epochs (0 disables). dRPC caps state-tree access at ~14d (~40k epochs); 30k gives a safety margin.")
	cmd.Flags().Uint64("parallel", 8, "Number of agents to check concurrently (--all / --random)")
	return cmd
}

func runAgentBalances(cmd *cobra.Command, args []string) {
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

	maxLookback, err := cmd.Flags().GetUint64("max-lookback")
	if err != nil {
		log.Fatal(err)
	}

	parallel, err := cmd.Flags().GetUint64("parallel")
	if err != nil {
		log.Fatal(err)
	}
	if parallel == 0 {
		parallel = 1
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

		failed, err := checkAgentBalance(ctx, eventsURL, epoch, maxLookback, agent)
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

		if randomAgents > 0 {
			if allAgents {
				cmd.Usage()
				return
			}
			if int(randomAgents) > len(agents) {
				randomAgents = uint64(len(agents))
			}
			rand.Shuffle(len(agents), func(i, j int) {
				agents[i], agents[j] = agents[j], agents[i]
			})
			agents = agents[:randomAgents]
		} else if !allAgents {
			cmd.Usage()
			return
		}

		// Snapshot a single epoch upfront and pin every check to
		// it: prevents per-agent drift between the node and DB
		// sides as the chain advances during the run. head-3 is
		// safely past the FEVM event-index lag at head.
		pinned := epoch
		if pinned == 0 {
			head, herr := getHeadEpoch(ctx)
			if herr != nil {
				log.Fatalf("get head: %v", herr)
			}
			if head < 3 {
				log.Fatalf("head epoch %d too low to pin", head)
			}
			pinned = head - 3
		}

		fmt.Printf("agent-balances: %d agents pinned at epoch=%d, parallel=%d\n",
			len(agents), pinned, parallel)
		failCount = checkAgentBalancesParallel(ctx, eventsURL, pinned, parallel, agents)
	}
	if failCount > 0 {
		log.Fatal("FAIL: Agent balances test had errors.")
	}
}

func init() {
	// Deprecated flat name: `inv agent-balances`
	flat := newAgentBalancesCmd("agent-balances [agent-id] [--all] [--random <num>] [--epoch <epoch>]")
	flat.Deprecated = "use `inv agent balances` instead"
	rootCmd.AddCommand(flat)
	// Grouped: `inv agent balances`
	agentCmd.AddCommand(newAgentBalancesCmd("balances [agent-id] [--all] [--random <num>] [--epoch <epoch>]"))
}

// checkAgentBalancesParallel runs the at-pinned-epoch comparison for
// every agent in `agents` across `parallel` goroutines. Returns the
// number of agents that failed (chain-vs-DB mismatch or RPC error).
//
// Pinning to a single epoch means both the node side
// (q.AgentLiquidAssets at epoch+1) and the DB side
// (/agent/{id}/available-balance?height=epoch) see the same chain
// state, eliminating per-agent drift the sequential loop produced
// when the chain advanced mid-run.
func checkAgentBalancesParallel(
	ctx context.Context,
	eventsURL string,
	epoch uint64,
	parallel uint64,
	agents []invariants.Agent,
) (failCount int) {
	type result struct {
		agentID  uint64
		dbBal    *big.Int
		ndBal    *big.Int
		diff     *big.Int
		err      error
		duration time.Duration
	}

	jobs := make(chan invariants.Agent, len(agents))
	results := make(chan result, len(agents))

	startedAt := time.Now()
	var wg sync.WaitGroup
	for w := uint64(0); w < parallel; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range jobs {
				t0 := time.Now()
				r := result{agentID: a.ID}
				dbBal, err := invariants.GetAgentAvailableBalanceAtHeightFromAPI(ctx, eventsURL, a.ID, epoch)
				if err != nil {
					r.err = fmt.Errorf("API at h=%d: %w", epoch, err)
					r.duration = time.Since(t0)
					results <- r
					continue
				}
				ndBal, _, err := getLiquidAssetsAtHeight(ctx, &a, epoch)
				if err != nil {
					r.err = fmt.Errorf("node at h=%d: %w", epoch, err)
					r.duration = time.Since(t0)
					results <- r
					continue
				}
				r.dbBal = dbBal
				r.ndBal = ndBal
				r.diff = new(big.Int).Sub(dbBal, ndBal)
				r.duration = time.Since(t0)
				results <- r
			}
		}()
	}

	for _, a := range agents {
		jobs <- a
	}
	close(jobs)
	wg.Wait()
	close(results)

	collected := make([]result, 0, len(agents))
	for r := range results {
		collected = append(collected, r)
	}
	sort.Slice(collected, func(i, j int) bool { return collected[i].agentID < collected[j].agentID })

	var diffsAbs []*big.Int
	for _, r := range collected {
		if r.err != nil {
			fmt.Printf("Agent %d: ERROR %v\n", r.agentID, r.err)
			failCount++
			continue
		}
		if r.diff.Sign() == 0 {
			continue
		}
		failCount++
		fmt.Printf("Agent %d: MISMATCH node=%s api=%s diff=%s\n",
			r.agentID, r.ndBal.String(), r.dbBal.String(), r.diff.String())
		diffsAbs = append(diffsAbs, new(big.Int).Abs(r.diff))
	}

	elapsed := time.Since(startedAt).Round(time.Millisecond)
	fmt.Printf("\nagent-balances summary: %d/%d failed @ epoch=%d in %s",
		failCount, len(agents), epoch, elapsed)
	if len(diffsAbs) > 0 {
		sort.Slice(diffsAbs, func(i, j int) bool { return diffsAbs[i].Cmp(diffsAbs[j]) < 0 })
		max := diffsAbs[len(diffsAbs)-1]
		p95 := diffsAbs[(len(diffsAbs)-1)*95/100]
		fmt.Printf(" maxDiff=%s p95Diff=%s", max.String(), p95.String())
	}
	fmt.Println()
	return failCount
}

func checkAgentBalance(ctx context.Context, eventsURL string, epoch uint64, maxLookback uint64, agent *invariants.Agent) (failed bool, err error) {
	agentID := agent.ID
	if epoch == 0 {
		availableBalanceResult, err := invariants.GetAgentAvailableBalanceFromAPI(ctx, eventsURL, agentID)
		if err != nil {
			return true, err
		}
		// Mutate for testing
		// availableBalanceResult.AvailableBalanceDB = big.NewInt(1234)
		if availableBalanceResult.AvailableBalanceDB.Cmp(availableBalanceResult.AvailableBalanceNd) == 0 {
			fmt.Printf("Agent %d: Success, latest available balances match: %v\n", agentID, availableBalanceResult.AvailableBalanceDB)
			return false, nil
		}
		fmt.Printf("Agent %d: Error, latest available balance from REST API doesn't match node.\n", agentID)
		fmt.Printf("  Node: %v\n", availableBalanceResult.AvailableBalanceNd)
		fmt.Printf("   API: %v\n", availableBalanceResult.AvailableBalanceDB)
		examineTransactionHistory(ctx, eventsURL, maxLookback, agent)
	} else {
		availableBalance, err := invariants.GetAgentAvailableBalanceAtHeightFromAPI(ctx, eventsURL, agentID, epoch)
		if err != nil {
			return true, err
		}

		agent, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
		if err != nil {
			return true, err
		}

		liquidAssets, interest, err := getLiquidAssetsAtHeight(ctx, agent, epoch)
		if err != nil {
			return true, err
		}

		fmt.Printf("Agent %d interest %d\n: ", agentID, interest)

		if availableBalance.Cmp(liquidAssets) == 0 {
			fmt.Printf("Agent %d @%d: Success, latest available balances match: %v\n", agentID, epoch, availableBalance)
			return false, nil
		}

		fmt.Printf("Agent %d @%d: Error, available balance from REST API doesn't match node.\n", agentID, epoch)
		fmt.Printf("  Node: %v\n", liquidAssets)
		fmt.Printf("   API: %v\n", availableBalance)
	}
	return true, nil
}

func checkAgentAcruedInterest(ctx context.Context, eventsURL string, epoch uint64, agent *invariants.Agent) (failed bool, err error) {
	agentID := agent.ID

	q := singleton.PoolsSDK.Query()

	interest, err := q.AgentInterestOwed(ctx, agent.AddressNative, big.NewInt(int64(epoch)))
	if err != nil {
		return true, err
	}

	availableBalance, err := invariants.GetAgentAvailableBalanceAtHeightFromAPI(ctx, eventsURL, agentID, epoch)
	if err != nil {
		return true, err
	}

	agent1, err := invariants.GetAgentFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		return true, err
	}

	liquidAssets, interest, err := getLiquidAssetsAtHeight(ctx, agent1, epoch)
	if err != nil {
		return true, err
	}

	fmt.Printf("Agent %d interest %d\n: ", agentID, interest)

	if availableBalance.Cmp(liquidAssets) == 0 {
		fmt.Printf("Agent %d @%d: Success, latest available balances match: %v\n", agentID, epoch, availableBalance)
		return false, nil
	}

	fmt.Printf("Agent %d @%d: Error, available balance from REST API doesn't match node.\n", agentID, epoch)
	fmt.Printf("  Node: %v\n", liquidAssets)
	fmt.Printf("   API: %v\n", availableBalance)

	return true, nil
}

func examineTransactionHistory(ctx context.Context, eventsURL string, maxLookback uint64, agent *invariants.Agent) {
	agentID := agent.ID
	fmt.Println("Examining transaction history...")
	txs, err := invariants.GetAgentTransactionsFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		log.Fatal(err)
	}
	totalTxs := len(txs)
	fmt.Printf("%d transactions retrieved from REST API\n", totalTxs)

	if maxLookback > 0 && totalTxs > 0 {
		head, err := getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		var floor uint64
		if head > maxLookback {
			floor = head - maxLookback
		}
		firstUsable := 0
		for firstUsable < len(txs) && txs[firstUsable].Height < floor {
			firstUsable++
		}
		if firstUsable > 0 {
			fmt.Printf("Skipping %d of %d transactions older than epoch %d (max-lookback %d)\n",
				firstUsable, totalTxs, floor, maxLookback)
			txs = txs[firstUsable:]
		}
	}
	if len(txs) == 0 {
		fmt.Println("No transactions in db.")
		txs = append(txs, invariants.Transaction{Height: agent.Height, AvailableBalance: big.NewInt(0)})
		height, err := getHeadEpoch(ctx)
		if err != nil {
			log.Fatal(err)
		}
		height = height - 2
		liquidAssets, _, err := getLiquidAssetsAtHeight(ctx, agent, height)
		if err != nil {
			log.Fatal(err)
		}
		txs = append(txs, invariants.Transaction{Height: height, AvailableBalance: liquidAssets})
		binarySearch(ctx, agent, txs, 0, 1)
	} else {
		// First
		tx := txs[0]
		firstIdx := 0
		fmt.Printf("First tx (idx:0) @%d: ", tx.Height)
		liquidAssets, _, err := getLiquidAssetsAtHeight(ctx, agent, tx.Height)
		if err != nil {
			log.Fatal(err)
		}
		if tx.AvailableBalance.Cmp(liquidAssets) == 0 {
			fmt.Printf("Matches: %v\n", liquidAssets)
		} else {
			fmt.Printf("Mismatch! Node: %v API: %v, Diff %v\n", liquidAssets, tx.AvailableBalance, tx.AvailableBalance.Sub(liquidAssets, tx.AvailableBalance))
			firstTx := invariants.Transaction{Height: agent.Height, AvailableBalance: big.NewInt(0)}
			txs = append([]invariants.Transaction{firstTx}, txs...)
			binarySearch(ctx, agent, txs, 0, 1)
			return
		}

		// Last
		if len(txs) == 1 {
			fmt.Println("Only one transaction in db.")
			height, err := getHeadEpoch(ctx)
			if err != nil {
				log.Fatal(err)
			}
			height = height - 3
			liquidAssets, _, err := getLiquidAssetsAtHeight(ctx, agent, height)
			if err != nil {
				log.Fatal(err)
			}
			txs = append(txs, invariants.Transaction{Height: height, AvailableBalance: liquidAssets})
			binarySearch(ctx, agent, txs, 0, 1)
			return
		}
		idx := len(txs) - 1
		tx = txs[idx]
		lastIdx := idx
		fmt.Printf("Last tx (idx:%d) @%d: ", idx, tx.Height)
		liquidAssets, _, err = getLiquidAssetsAtHeight(ctx, agent, tx.Height)
		if err != nil {
			log.Fatal(err)
		}
		if tx.AvailableBalance.Cmp(liquidAssets) == 0 {
			fmt.Printf("Matches: %v\n", liquidAssets)
			// Probably missing a transaction beyond last epoch in database
			latestHeight, err := getHeadEpoch(ctx)
			if err != nil {
				log.Fatal(err)
			}
			txs = append(txs, invariants.Transaction{Height: latestHeight - 1})
			binarySearch(ctx, agent, txs, idx, len(txs)-1)
			return
		} else {
			fmt.Printf("Mismatch! Node: %v API: %v, Diff: %v\n", liquidAssets, tx.AvailableBalance, tx.AvailableBalance.Sub(liquidAssets, tx.AvailableBalance))
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
		findBalanceTransitions(ctx, agent, txs[goodIdx], txs[badIdx])
		return
	}
	tx := txs[searchIdx]
	fmt.Printf("Tx (idx:%d) @%d: ", searchIdx, tx.Height)
	liquidAssets, _, err := getLiquidAssetsAtHeight(ctx, agent, tx.Height)
	if err != nil {
		log.Fatal(err)
	}
	if tx.AvailableBalance.Cmp(liquidAssets) == 0 {
		fmt.Printf("Matches: %v\n", liquidAssets)
		binarySearch(ctx, agent, txs, searchIdx, badIdx)
	} else {
		fmt.Printf("Mismatch! Node: %v API: %v, Diff: %v\n", liquidAssets, tx.AvailableBalance, tx.AvailableBalance.Sub(liquidAssets, tx.AvailableBalance))
		binarySearch(ctx, agent, txs, goodIdx, searchIdx)
	}
}

func findBalanceTransitions(
	ctx context.Context,
	agent *invariants.Agent,
	goodTx invariants.Transaction,
	badTx invariants.Transaction,
) {
	fmt.Printf("Looking for interim balance transitions on node for agent %d...\n", agent.ID)
	fmt.Printf("From %d to %d\n", goodTx.Height, badTx.Height)
	height := goodTx.Height
	balance := goodTx.AvailableBalance
	var err error
	for {
		fmt.Printf("%d: %v\n", height, balance)
		height, balance, err = findNextBalanceTransition(ctx, agent, height+1, balance, badTx.Height-1)
		if err != nil {
			log.Fatal(err)
		}
		if height == 0 {
			break
		}
	}
}

func findNextBalanceTransition(
	ctx context.Context,
	agent *invariants.Agent,
	minHeight uint64,
	prevBalance *big.Int,
	maxHeight uint64,
) (uint64, *big.Int, error) {
	if maxHeight < minHeight {
		return 0, nil, nil
	}
	fmt.Printf("  Searching %d to %d\n", minHeight, maxHeight)
	sampleHeight := (maxHeight-minHeight)/2 + minHeight
	liquidAssets, _, err := getLiquidAssetsAtHeight(ctx, agent, sampleHeight)
	if err != nil {
		return 0, nil, err
	}
	fmt.Printf("  Liquid assets @%d: %v\n", sampleHeight, liquidAssets)
	if prevBalance.Cmp(liquidAssets) == 0 {
		return findNextBalanceTransition(ctx, agent, sampleHeight+1, prevBalance, maxHeight)
	} else {
		height, balance, err := findNextBalanceTransition(ctx, agent, minHeight, prevBalance, sampleHeight-1)
		if err != nil {
			return 0, nil, err
		}
		if height > 0 {
			return height, balance, nil
		} else {
			return sampleHeight, liquidAssets, nil
		}
	}
}

func getLiquidAssetsAtHeight(ctx context.Context, agent *invariants.Agent, height uint64) (*big.Int, *big.Int, error) {
	nextEpoch, err := getNextEpoch(ctx, height)
	if err != nil {
		return nil, nil, err
	}

	q := singleton.PoolsSDK.Query()
	liquidAssets, err := q.AgentLiquidAssets(ctx, agent.AddressNative, big.NewInt(int64(nextEpoch)))
	if err != nil {
		return nil, nil, err
	}

	interest, err := q.AgentInterestOwed(ctx, agent.AddressNative, big.NewInt(int64(nextEpoch)))
	if err != nil {
		return nil, nil, err
	}

	return liquidAssets, interest, nil
}
