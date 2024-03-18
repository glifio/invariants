package invariants

import (
	"context"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/glifio/invariants/singleton"
	"github.com/stretchr/testify/assert"
)

func init() {
	chainID, err := strconv.Atoi(os.Getenv("CHAIN_ID"))
	if err != nil {
		log.Fatal(err)
	}

	singleton.InitPoolsSDK(
		context.Background(),
		int64(chainID),
		os.Getenv("LOTUS_PRIVATE_ADDR"),
		os.Getenv("LOTUS_PRIVATE_TOKEN"),
	)
}

// TestMetrics calls the REST API, and compares against on-chain
func TestMetrics(t *testing.T) {
	ctx := context.Background()

	metricsFromAPI, err := GetMetricsFromAPI(ctx)
	assert.Nil(t, err)

	// fmt.Printf("Jim rest %+v\n", metricsFromAPI)

	if metricsFromAPI.Height == 0 {
		t.Fatal("Height is zero")
	}

	height := metricsFromAPI.Height
	metricsFromNode, err := GetMetricsFromNode(ctx, height)
	assert.Nil(t, err)

	// fmt.Printf("Jim chain %+v\n", metricsFromNode)

	// assert.Equal(t, metricsFromAPI.PoolTotalAssets, metricsFromNode.PoolTotalAssets, "Total assets should be equal")
	assert.Equal(t, metricsFromAPI.PoolTotalBorrowed, metricsFromNode.PoolTotalBorrowed, "Total borrowed should be equal")
	assert.Equal(t, metricsFromAPI.TotalAgentCount, metricsFromNode.TotalAgentCount, "Agent count should be equal")
}
