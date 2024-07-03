package invariants

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/filecoin-project/go-address"
	"github.com/glifio/invariants/singleton"
)

type AgentJSON struct {
	Address          string `json:"address"`
	AddressNative    string `json:"addressNative"`
	AvailableBalance string `json:"availableBalance"`
	Balance          string `json:"balance"`
	Height           uint64 `json:"height"`
	ID               uint64 `json:"id"`
	Miners           uint64 `json:"miners"`
	PrincipalBalance string `json:"principalBalance"`
	TxHash           string `json:"txHash"`
}

type Agent struct {
	Address          string
	AddressNative    common.Address
	AvailableBalance *big.Int
	Balance          *big.Int
	Height           uint64
	ID               uint64
	Miners           uint64
	PrincipalBalance *big.Int
	TxHash           string
}

// GetAgentFromAPI calls the REST API to get the record for an agent
func GetAgentFromAPI(ctx context.Context, eventsURL string, agentID uint64) (*Agent, error) {
	agents, err := GetAgentsFromAPI(ctx, eventsURL)
	if err != nil {
		return nil, err
	}
	for _, agent := range agents {
		if agent.ID == agentID {
			return &agent, nil
		}
	}
	return nil, nil
}

// GetAgentsFromAPI calls the REST API to get the list of agents
func GetAgentsFromAPI(ctx context.Context, eventsURL string) ([]Agent, error) {
	url := fmt.Sprintf("%s/agent", eventsURL)

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

	var response []AgentJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	/*
		j, _ := json.MarshalIndent(response, "", "  ")
		fmt.Printf("Response %+v\n", string(j))
	*/

	agents := make([]Agent, 0)
	for _, agentJSON := range response {
		addressNative := common.HexToAddress(agentJSON.AddressNative)
		availableBalance := big.NewInt(0)
		availableBalance.SetString(agentJSON.AvailableBalance, 10)
		balance := big.NewInt(0)
		balance.SetString(agentJSON.Balance, 10)
		principalBalance := big.NewInt(0)
		principalBalance.SetString(agentJSON.PrincipalBalance, 10)
		agent := Agent{
			Address:          agentJSON.Address,
			AddressNative:    addressNative,
			AvailableBalance: availableBalance,
			Balance:          balance,
			Height:           agentJSON.Height,
			ID:               agentJSON.ID,
			Miners:           agentJSON.Miners,
			PrincipalBalance: principalBalance,
			TxHash:           agentJSON.TxHash,
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

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

type AgentEconJSON struct {
	Id              uint64 `json:"id"`
	Assets          string `json:"assets"`
	Liability       string `json:"liability"`
	Equity          string `json:"equity"`
	CollateralValue string `json:"collateralValue"`
	BorrowNow       string `json:"borrowNow"`
	BorrowMax       string `json:"borrowMax"`
	Dte             string `json:"dte"`
}

type AgentEconResult struct {
	Id              uint64
	Assets          *big.Int
	Liability       *big.Int
	Equity          *big.Int
	CollateralValue *big.Int
	BorrowNow       *big.Int
	BorrowMax       *big.Int
	Dte             float64
}

// GetAgentEconFromAPI calls the REST API to get the latest econ values for an agent
func GetAgentEconFromAPI(ctx context.Context, eventsURL string, agentID uint64) (*AgentEconResult, error) {
	url := fmt.Sprintf("%s/agent/%d/econ", eventsURL, agentID)

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

	var response AgentEconJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	assets := big.NewInt(0)
	assets.SetString(response.Assets, 10)
	liability := big.NewInt(0)
	liability.SetString(response.Liability, 10)
	equity := big.NewInt(0)
	equity.SetString(response.Equity, 10)
	collateralValue := big.NewInt(0)
	collateralValue.SetString(response.CollateralValue, 10)
	borrowNow := big.NewInt(0)
	borrowNow.SetString(response.BorrowNow, 10)
	borrowMax := big.NewInt(0)
	borrowMax.SetString(response.BorrowMax, 10)
	dte, err := strconv.ParseFloat(response.Dte, 64)
	if err != nil {
		return nil, err
	}

	result := AgentEconResult{
		Id:              response.Id,
		Assets:          assets,
		Liability:       liability,
		Equity:          equity,
		CollateralValue: collateralValue,
		BorrowNow:       borrowNow,
		BorrowMax:       borrowMax,
		Dte:             dte,
	}

	return &result, nil
}

// GetAgentEconFromNode calls the node to get the econ values from the node
func GetAgentEconFromNode(ctx context.Context, address common.Address, height uint64) (*AgentEconResult, uint64, error) {
	height, err := getNextEpoch(ctx, height)
	if err != nil {
		return nil, height, err
	}

	blockNumber := big.NewInt(int64(height))
	q := singleton.PoolsSDK.Query()

	principal, err := q.AgentPrincipal(ctx, address, blockNumber)
	if err != nil {
		return nil, height, err
	}

	result := AgentEconResult{
		Liability: principal,
	}

	return &result, height, nil
}

type MinerDetailsJSON struct {
	Miner                  uint64          `json:"miner"`
	AgentId                uint64          `json:"agentId"`
	Actions                uint16          `json:"actions"`
	MinerAddr              address.Address `json:"minerAddr"`
	AvailableBalance       string          `json:"availableBalance"`
	Equity                 string          `json:"equity"`
	EstimatedWeeklyRewards string          `json:"estimatedWeeklyRewards"`
	QAP                    string          `json:"qap"`
	RBP                    string          `json:"rbp"`
	SlashingRisk           string          `json:"slashingRisk"`
	LiveSectors            string          `json:"liveSectors"`
	FaultySectors          string          `json:"faultySectors"`
	RecoveringSectors      string          `json:"recoveringSectors"`
	Ratio                  string          `json:"ratio"`
	TerminationPenalty     string          `json:"terminationPenalty"`
	LiquidationValue       string          `json:"liquidationValue"`
}

type MinerDetailsResult struct {
	Miner                  uint64
	AgentId                uint64
	Actions                uint16
	MinerAddr              address.Address
	AvailableBalance       *big.Int
	Equity                 *big.Int
	EstimatedWeeklyRewards *big.Int
	QAP                    *big.Int
	RBP                    *big.Int
	SlashingRisk           float64
	LiveSectors            uint64
	FaultySectors          uint64
	RecoveringSectors      uint64
	Ratio                  float64
	TerminationPenalty     *big.Int
	LiquidationValue       *big.Int
}

// GetAgentMinersFromAPI calls the REST API to get the miners for an agent
func GetAgentMinersAPI(ctx context.Context, eventsURL string, agentID uint64) ([]MinerDetailsResult, error) {
	url := fmt.Sprintf("%s/agent/%d/miners", eventsURL, agentID)

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

	var response []MinerDetailsJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	results := make([]MinerDetailsResult, 0)
	fmt.Printf("Jim response: %+v\n", response)
	for _, minerDetail := range response {
		availableBalance, _ := new(big.Int).SetString(minerDetail.AvailableBalance, 10)
		equity, _ := new(big.Int).SetString(minerDetail.Equity, 10)
		estimatedWeeklyRewards, _ := new(big.Int).SetString(minerDetail.EstimatedWeeklyRewards, 10)
		qap, _ := new(big.Int).SetString(minerDetail.QAP, 10)
		rbp, _ := new(big.Int).SetString(minerDetail.RBP, 10)
		slashingRisk, _ := strconv.ParseFloat(minerDetail.SlashingRisk, 64)
		liveSectors, _ := strconv.ParseUint(minerDetail.LiveSectors, 10, 64)
		faultySectors, _ := strconv.ParseUint(minerDetail.FaultySectors, 10, 64)
		recoveringSectors, _ := strconv.ParseUint(minerDetail.RecoveringSectors, 10, 64)
		ratio, _ := strconv.ParseFloat(minerDetail.Ratio, 64)
		terminationPenalty, _ := new(big.Int).SetString(minerDetail.TerminationPenalty, 10)
		liquidationValue, _ := new(big.Int).SetString(minerDetail.LiquidationValue, 10)
		results = append(results, MinerDetailsResult{
			Miner:                  minerDetail.Miner,
			AgentId:                minerDetail.AgentId,
			Actions:                minerDetail.Actions,
			MinerAddr:              minerDetail.MinerAddr,
			AvailableBalance:       availableBalance,
			Equity:                 equity,
			EstimatedWeeklyRewards: estimatedWeeklyRewards,
			QAP:                    qap,
			RBP:                    rbp,
			SlashingRisk:           slashingRisk,
			LiveSectors:            liveSectors,
			FaultySectors:          faultySectors,
			RecoveringSectors:      recoveringSectors,
			Ratio:                  ratio,
			TerminationPenalty:     terminationPenalty,
			LiquidationValue:       liquidationValue,
		})
	}

	return results, nil
}
