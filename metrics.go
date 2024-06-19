package invariants

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/glifio/go-pools/abigen"
	"github.com/glifio/invariants/singleton"
)

type MetricsJSON struct {
	Height                    uint64 `json:"height"`
	Timestamp                 uint64 `json:"timestamp"`
	PoolTotalAssets           string `json:"poolTotalAssets"`
	PoolTotalBorrowed         string `json:"poolTotalBorrowed"`
	PoolTotalBorrowableAssets string `json:"poolTotalBorrowableAssets"`
	PoolExitReserve           string `json:"poolExitReserve"`
	TotalAgentCount           uint64 `json:"totalAgentCount"`
	TotalMinerCollaterals     string `json:"totalMinerCollaterals"`
	TotalMinersCount          uint64 `json:"totalMinersCount"`
	TotalValueLocked          string `json:"totalValueLocked"`
	TotalMinersSectors        string `json:"totalMinersSectors"`
	TotalMinerQAP             string `json:"totalMinerQAP"`
	TotalMinerRBP             string `json:"totalMinerRBP"`
	TotalMinerEDR             string `json:"totalMinerEDR"`
}

type MetricsResult struct {
	Height                    uint64
	Timestamp                 uint64
	PoolTotalAssets           *big.Int
	PoolTotalBorrowed         *big.Int
	PoolTotalBorrowableAssets *big.Int
	PoolExitReserve           *big.Int
	TotalAgentCount           uint64
	TotalMinerCollaterals     *big.Int
	TotalMinersCount          uint64
	TotalValueLocked          *big.Int
	TotalMinersSectors        *big.Int
	TotalMinerQAP             *big.Int
	TotalMinerRBP             *big.Int
	TotalMinerEDR             *big.Int
}

// GetMetricsFromAPI calls the REST API to get the metrics
func GetMetricsFromAPI(ctx context.Context, eventsURL string) (*MetricsResult, error) {
	url := fmt.Sprintf("%s/metrics", eventsURL)
	return getMetrics(ctx, url)
}

// GetMetricsFromAPIAtHeight calls the REST API to get the metrics
func GetMetricsFromAPIAtHeight(ctx context.Context, eventsURL string, height uint64) (*MetricsResult, error) {
	url := fmt.Sprintf("%s/metrics/%d", eventsURL, height)
	return getMetrics(ctx, url)
}

func getMetrics(ctx context.Context, url string) (*MetricsResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Println("error creating request:", err)
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("error getting response:", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad http status: %v", res.StatusCode)
	}

	var response MetricsJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	poolTotalAssets := big.NewInt(0)
	poolTotalAssets.SetString(response.PoolTotalAssets, 10)
	poolTotalBorrowed := big.NewInt(0)
	poolTotalBorrowed.SetString(response.PoolTotalBorrowed, 10)
	poolTotalBorrowableAssets := big.NewInt(0)
	poolTotalBorrowableAssets.SetString(response.PoolTotalBorrowableAssets, 10)
	poolExitReserve := big.NewInt(0)
	poolExitReserve.SetString(response.PoolExitReserve, 10)
	totalMinerCollaterals := big.NewInt(0)
	totalMinerCollaterals.SetString(response.TotalMinerCollaterals, 10)
	totalValueLocked := big.NewInt(0)
	totalValueLocked.SetString(response.TotalValueLocked, 10)
	totalMinersSectors := big.NewInt(0)
	totalMinersSectors.SetString(response.TotalMinersSectors, 10)
	totalMinerQAP := big.NewInt(0)
	totalMinerQAP.SetString(response.TotalMinerQAP, 10)
	totalMinerRBP := big.NewInt(0)
	totalMinerRBP.SetString(response.TotalMinerRBP, 10)
	totalMinerEDR := big.NewInt(0)
	totalMinerEDR.SetString(response.TotalMinerEDR, 10)

	result := MetricsResult{
		Height:                    response.Height,
		Timestamp:                 response.Timestamp,
		PoolTotalAssets:           poolTotalAssets,
		PoolTotalBorrowed:         poolTotalBorrowed,
		PoolTotalBorrowableAssets: poolTotalBorrowableAssets,
		PoolExitReserve:           poolExitReserve,
		TotalAgentCount:           response.TotalAgentCount,
		TotalMinerCollaterals:     totalMinerCollaterals,
		TotalMinersCount:          response.TotalMinersCount,
		TotalValueLocked:          totalValueLocked,
		TotalMinersSectors:        totalMinersSectors,
		TotalMinerQAP:             totalMinerQAP,
		TotalMinerRBP:             totalMinerRBP,
		TotalMinerEDR:             totalMinerEDR,
	}
	return &result, nil
}

// GetMetricsFromNode calls the Lotus node to get the metrics
func GetMetricsFromNode(ctx context.Context, height uint64) (*MetricsResult, uint64, error) {
	sdk := singleton.PoolsSDK

	height, err := getNextEpoch(ctx, height)
	if err != nil {
		return nil, height, err
	}

	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		return nil, height, err
	}
	defer ethClient.Close()

	blockNumber := big.NewInt(int64(height))

	infinityPool := sdk.Query().InfinityPool()

	poolCaller, err := abigen.NewInfinityPoolCaller(infinityPool, ethClient)
	if err != nil {
		return nil, height, err
	}

	totalAssets, err := poolCaller.TotalAssets(&bind.CallOpts{Context: ctx, BlockNumber: blockNumber})
	if err != nil {
		return nil, height, err
	}

	totalBorrowed, err := poolCaller.TotalBorrowed(&bind.CallOpts{Context: ctx, BlockNumber: blockNumber})
	if err != nil {
		return nil, height, err
	}

	agentFactory := sdk.Query().AgentFactory()

	agentFactoryCaller, err := abigen.NewAgentFactoryCaller(agentFactory, ethClient)
	if err != nil {
		return nil, height, err
	}

	agentCount, err := agentFactoryCaller.AgentCount(&bind.CallOpts{Context: ctx, BlockNumber: blockNumber})
	if err != nil {
		return nil, height, err
	}

	/*
		minerRegistry := sdk.Query().MinerRegistry()

		minerRegistryCaller, err := abigen.NewMinerRegistryCaller(minerRegistry, ethClient)
		if err != nil {
			return nil, err
		}

		minerRegistryCaller.MinersCount()
	*/

	result := MetricsResult{
		Height:                    height,
		Timestamp:                 0, // unused
		PoolTotalAssets:           totalAssets,
		PoolTotalBorrowed:         totalBorrowed,
		PoolTotalBorrowableAssets: nil, // unused
		PoolExitReserve:           nil, // unused
		TotalAgentCount:           agentCount.Uint64(),
		TotalMinerCollaterals:     nil, // unused
		TotalMinersCount:          0,
		TotalValueLocked:          nil, // unused
		TotalMinersSectors:        nil, // unused
		TotalMinerQAP:             nil, // unused
		TotalMinerRBP:             nil, // unused
	}
	return &result, height, nil
}

// GetMinerCountFromNode calls the Lotus node to get the total miners for all the agents
func GetMinerCountFromNode(ctx context.Context, height uint64) (uint64, uint64, error) {
	sdk := singleton.PoolsSDK

	height, err := getNextEpoch(ctx, height)
	if err != nil {
		return 0, height, err
	}

	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		return 0, height, err
	}
	defer ethClient.Close()

	blockNumber := big.NewInt(int64(height))

	agentFactory := sdk.Query().AgentFactory()

	agentFactoryCaller, err := abigen.NewAgentFactoryCaller(agentFactory, ethClient)
	if err != nil {
		return 0, height, err
	}

	agentCount, err := agentFactoryCaller.AgentCount(&bind.CallOpts{Context: ctx, BlockNumber: blockNumber})
	if err != nil {
		return 0, height, err
	}

	minerRegistry := sdk.Query().MinerRegistry()

	minerRegistryCaller, err := abigen.NewMinerRegistryCaller(minerRegistry, ethClient)
	if err != nil {
		return 0, height, err
	}

	var totalMiners uint64
	for i := 0; i <= int(agentCount.Uint64()); i++ {
		count, err := minerRegistryCaller.MinersCount(&bind.CallOpts{Context: ctx, BlockNumber: blockNumber}, big.NewInt(int64(i)))
		if err != nil {
			return 0, height, err
		}
		fmt.Printf("Agent %d: %d\n", i, count)
		totalMiners += count.Uint64()
	}

	return totalMiners, height, nil
}
