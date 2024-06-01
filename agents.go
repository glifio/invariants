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

// GetAgentAvailableBalanceAtHeightFromAPI calls the REST API to get the available balance for an agent at a particular epoch
func GetAgentAvailableBalanceAtHeightFromAPI(ctx context.Context, eventsURL string, agentID uint64, height uint64) (*big.Int, error) {
	balance := big.NewInt(0)

	txs, err := GetAgentTransactionsFromAPI(ctx, eventsURL, agentID)
	if err != nil {
		return nil, err
	}
	for _, tx := range txs {
		if tx.Height > height {
			break
		}
		balance = tx.AvailableBalance
	}

	return balance, nil
}

type TransactionJSON struct {
	Amount           string `json:"amount"`
	AvailableBalance string `json:"availableBalance"`
	Balance          string `json:"balance"`
	Height           uint64 `json:"height"`
	ID               uint64 `json:"id"`
	Interest         string `json:"interest"`
	Principal        string `json:"principal"`
	Timestamp        uint64 `json:"timestamp"`
	TxHash           string `json:"txHash"`
	Type             string `json:"type"`
}

type Transaction struct {
	Amount           *big.Int
	AvailableBalance *big.Int
	Balance          *big.Int
	Height           uint64
	ID               uint64
	Interest         *big.Int
	Principal        *big.Int
	Timestamp        uint64
	TxHash           string
	Type             string
}

// GetAgentTransactionsFromAPI calls the REST API to get the transactions for an Agent
func GetAgentTransactionsFromAPI(ctx context.Context, eventsURL string, agentID uint64) ([]Transaction, error) {
	url := fmt.Sprintf("%s/agent/%d/tx", eventsURL, agentID)

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

	var response []TransactionJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	txs := make([]Transaction, 0)
	for _, txJSON := range response {
		amount := big.NewInt(0)
		amount.SetString(txJSON.Amount, 10)
		availableBalance := big.NewInt(0)
		availableBalance.SetString(txJSON.AvailableBalance, 10)
		interest := big.NewInt(0)
		interest.SetString(txJSON.Interest, 10)
		principal := big.NewInt(0)
		principal.SetString(txJSON.Principal, 10)
		tx := Transaction{
			Amount:           amount,
			AvailableBalance: availableBalance,
			Height:           txJSON.Height,
			ID:               txJSON.ID,
			Interest:         interest,
			Principal:        principal,
			Timestamp:        txJSON.Timestamp,
			TxHash:           txJSON.TxHash,
			Type:             txJSON.Type,
		}
		txs = append(txs, tx)
	}

	return txs, nil
}
