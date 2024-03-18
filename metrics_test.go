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
	metricsFromAPI, err := GetMetricsFromAPI()
	assert.Nil(t, err)
	if metricsFromAPI.Height == 0 {
		t.Fatal("Height is zero")
	}
}
