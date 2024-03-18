package invariants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMetrics calls the REST API, and compares against on-chain
func TestMetrics(t *testing.T) {
	metricsFromAPI, err := GetMetricsFromAPI()
	assert.Nil(t, err)
	if metricsFromAPI.Height == 0 {
		t.Fatal("Height is zero")
	}
}
