package invariants

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/glifio/invariants/singleton"
)

type IFILTotalSupplyJSON struct {
	Height          uint64 `json:"height"`
	IFILTotalSupply string `json:"iFILTotalSupply"`
}

type IFILTotalSupply struct {
	Height          uint64
	IFILTotalSupply *big.Int
}

// GetIFILTotalSupplyFromAPI calls the REST API to get the iFIL total supply
func GetIFILTotalSupplyFromAPI(ctx context.Context, eventsURL string, height uint64) (*IFILTotalSupply, error) {
	url := fmt.Sprintf("%s/ifil/%d/total-supply", eventsURL, height)

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

	var response IFILTotalSupplyJSON

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	totalSupply := big.NewInt(0)
	totalSupply.SetString(response.IFILTotalSupply, 10)
	iFILTotalSupply := IFILTotalSupply{
		Height:          response.Height,
		IFILTotalSupply: totalSupply,
	}

	return &iFILTotalSupply, nil
}

// GetIFILTotalSupplyFromNode calls the node to get the iFIL total supply
func GetIFILTotalSupplyFromNode(ctx context.Context, height uint64) (*IFILTotalSupply, error) {
	height, err := getNextEpoch(ctx, height)
	if err != nil {
		return nil, err
	}

	blockNumber := big.NewInt(int64(height))
	q := singleton.PoolsArchiveSDK.Query()

	totalSupply, err := q.IFILSupply(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	iFILTotalSupply := IFILTotalSupply{
		Height:          height,
		IFILTotalSupply: totalSupply,
	}

	return &iFILTotalSupply, nil
}

func getNextEpoch(ctx context.Context, epoch uint64) (uint64, error) {
	lotus := singleton.Lotus()

	ts, err := lotus.Api.ChainGetTipSetAfterHeight(ctx, abi.ChainEpoch(epoch+1), types.EmptyTSK)
	if err != nil {
		return 0, err
	}

	return uint64(ts.Height()), nil
}
