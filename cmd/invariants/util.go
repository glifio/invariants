package main

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/glifio/invariants/singleton"
)

func getNextEpoch(ctx context.Context, epoch uint64) (uint64, error) {
	lotus := singleton.Lotus()

	ts, err := lotus.Api.ChainGetTipSetAfterHeight(ctx, abi.ChainEpoch(epoch+1), types.EmptyTSK)
	if err != nil {
		return 0, err
	}

	return uint64(ts.Height()), nil
}
