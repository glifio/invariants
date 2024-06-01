package invariants

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
)

type AvailableBalanceJSON struct {
	AvailableBalanceDB string `json:"availableBalanceDB"`
	AvailableBalanceNd string `json:"availableBalanceNd"`
}

type AvailableBalanceResult struct {
	AvailableBalanceDB *big.Int
	AvailableBalanceNd *big.Int
}

// GetAgentAvailableBalanceFromAPI calls the REST API to get the latest available balance for an agent
func GetAgentAvailableBalanceFromAPI(ctx context.Context, eventsURL string, agentID uint64) (*AvailableBalanceResult, error) {
	url := fmt.Sprintf("%s/agent/%d/available-balance", eventsURL, agentID)

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

	var response AvailableBalanceJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	availableBalanceDB := big.NewInt(0)
	availableBalanceDB.SetString(response.AvailableBalanceDB, 10)
	availableBalanceNd := big.NewInt(0)
	availableBalanceNd.SetString(response.AvailableBalanceNd, 10)

	result := AvailableBalanceResult{
		AvailableBalanceDB: availableBalanceDB,
		AvailableBalanceNd: availableBalanceNd,
	}

	return &result, nil
}
