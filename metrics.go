package invariants

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
)

const eventsURL = "http://127.0.0.1:8091"

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
	Height          uint64
	PoolTotalAssets *big.Int
}

// GetMetricsFromAPI calls the REST API to get the metrics
func GetMetricsFromAPI() (*MetricsResult, error) {
	ctx := context.Background()

	url := fmt.Sprintf("%s/metrics", eventsURL)

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

	fmt.Printf("Jim response %+v\n", response)

	result := MetricsResult{
		Height: response.Height,
	}
	return &result, nil
}
