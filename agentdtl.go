package invariants

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/filecoin-project/go-address"
	stbuiltin "github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/network"
	lotusapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	minertypes "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	miner8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/miner"
	poolsabigen "github.com/glifio/go-pools/abigen"
	"github.com/glifio/go-pools/econ"
	poolstypes "github.com/glifio/go-pools/types"
	"github.com/glifio/go-pools/util"
	invabigen "github.com/glifio/invariants/abigen"
	"github.com/glifio/invariants/singleton"
)

// AgentDTL is one agent's debt-to-liquidation-value snapshot, joined
// with the SP+ tier-specific maximum DTL allowed before liquidation.
//
// Computed live against chain on every call — no DB cache. The
// underlying termination-fee math no longer walks sectors per miner,
// so a fleet-wide refresh runs in ~30s with reasonable concurrency.
type AgentDTL struct {
	AgentID   uint64
	Address   common.Address
	Debt      *big.Int // principal + interest, from chain
	LV        *big.Int // liquidation value (balance - terminationFee), from chain
	DTL       *big.Int // debt × 1e18 / lv (zero if lv == 0)
	Tier      uint8    // SP+ tier number (0 = base tier)
	MaxDTL    *big.Int // tier's DebtToLiquidationValue, also × 1e18
	BufferBps uint64   // safety buffer in bps, treat agents within (max-buffer..max] as warning
	Severity  DTLSeverity
	Err       error // any per-agent fetch error; the fleet sweep tolerates partial failures
}

type DTLSeverity int

const (
	DTLOK      DTLSeverity = iota // dtl <= max - buffer
	DTLWarn                       // max - buffer < dtl <= max  (close to limit)
	DTLOverMax                    // dtl > max                  (eligible for liquidation)
	DTLNoLV                       // lv == 0 (no collateral; only flags if debt > 0)
	DTLError                      // fetch error
)

// tipsetGlobals are the three values that are byte-identical for every
// miner at a given tipset: smoothed network reward, smoothed network
// QA-power, and network version. Fetched once per fleet sweep instead
// of once per miner — that alone cuts ~50% of the Lotus traffic for
// `inv agent dtl --all`.
type tipsetGlobals struct {
	EpochReward    builtin.FilterEstimate
	TotalQAPower   builtin.FilterEstimate
	NetworkVersion network.Version
}

// FetchAgentDTLs computes per-agent DTL against chain truth at ts.
//
// Two batched chain calls happen up front:
//
//  1. tipset-globals (reward, power, network-version) for the per-miner
//     termination math
//  2. InvariantsQuery.GetAgentsDTLInputs for the four EVM reads per
//     agent (liquidAssets, miner list, principal, interest), collapsed
//     into one eth_call covering every agent in the sweep.
//
// Workers then only do per-miner Lotus state reads, which are the
// genuinely per-miner part of the computation.
//
// `invQueryAddr` is the deployed InvariantsQuery contract — see
// INVARIANTS_QUERY_ADDR in mainnet.env.
//
// `progress`, if non-nil, is invoked from worker goroutines as each
// agent completes. Use it for stderr ticking on long fleet sweeps.
func FetchAgentDTLs(
	ctx context.Context,
	sdk poolstypes.PoolsSDK,
	invQueryAddr common.Address,
	agentIDs []uint64,
	addresses []common.Address,
	ts *types.TipSet,
	bufferBps uint64,
	workers int,
) ([]AgentDTL, error) {
	if len(agentIDs) != len(addresses) {
		return nil, fmt.Errorf("agentIDs and addresses must align (got %d, %d)", len(agentIDs), len(addresses))
	}
	if workers < 1 {
		workers = 1
	}

	height := big.NewInt(int64(ts.Height()))
	tiers, err := sdk.Query().SPPlusTierInfo(ctx, height)
	if err != nil {
		return nil, fmt.Errorf("SPPlusTierInfo: %w", err)
	}

	// One Lotus connection shared across workers (FullNodeStruct's RPC
	// client is goroutine-safe), and one fetch of the three tipset-
	// globals reused for every miner compute. Use the singleton's
	// connection — it is routed through the process-wide RPC rate
	// limiter; sdk.Extern().ConnectLotusClient() is not.
	node := singleton.Lotus()
	if node == nil {
		return nil, fmt.Errorf("lotus singleton not initialized")
	}
	lapi := &node.Api

	globals, err := fetchTipsetGlobals(ctx, lapi, ts)
	if err != nil {
		return nil, fmt.Errorf("fetchTipsetGlobals: %w", err)
	}

	// Batched eth_call: pull liquidAssets/miner-list/principal/interest
	// for every agent in one round trip. Aligns one-to-one with the
	// agentIDs slice.
	dtlInputs, err := fetchAgentDTLInputs(ctx, sdk, invQueryAddr, agentIDs, addresses, ts)
	if err != nil {
		return nil, fmt.Errorf("fetchAgentDTLInputs: %w", err)
	}
	if len(dtlInputs) != len(agentIDs) {
		return nil, fmt.Errorf("InvariantsQuery returned %d DTL inputs for %d agents", len(dtlInputs), len(agentIDs))
	}

	out := make([]AgentDTL, len(agentIDs))
	type job struct{ i int }
	jobs := make(chan job, len(agentIDs))
	for i := range agentIDs {
		jobs <- job{i}
	}
	close(jobs)

	var (
		wg sync.WaitGroup
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				r := computeOne(ctx, sdk, lapi, agentIDs[j.i], addresses[j.i], dtlInputs[j.i], ts, height, tiers, globals, bufferBps)
				out[j.i] = r
			}
		}()
	}
	wg.Wait()
	return out, nil
}

// fetchAgentDTLInputs issues one batched eth_call against
// InvariantsQuery for every agent. Replaces the four sequential pool
// eth_calls per agent (AgentLiquidAssets + AgentMiners (1+N calls) +
// AgentPrincipal + AgentInterestOwed). At n=246 agents that's ~2400
// EVM calls collapsed to one.
func fetchAgentDTLInputs(
	ctx context.Context,
	sdk poolstypes.PoolsSDK,
	invQueryAddr common.Address,
	agentIDs []uint64,
	addresses []common.Address,
	ts *types.TipSet,
) ([]invabigen.InvariantsQueryAgentDTLInputs, error) {
	ethClient, err := sdk.Extern().ConnectEthClient()
	if err != nil {
		return nil, fmt.Errorf("connect eth: %w", err)
	}
	defer ethClient.Close()

	q, err := invabigen.NewInvariantsQueryCaller(invQueryAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("InvariantsQuery caller: %w", err)
	}

	idsBig := make([]*big.Int, len(agentIDs))
	for i, id := range agentIDs {
		idsBig[i] = new(big.Int).SetUint64(id)
	}

	// Read at the same height EstimateTerminationFeeAgent used: ts.Height()-1.
	height := big.NewInt(int64(ts.Height()) - 1)
	opts := &bind.CallOpts{Context: ctx, BlockNumber: height}
	return q.GetAgentsDTLInputs(opts, idsBig, addresses)
}

// fetchTipsetGlobals issues the three tipset-global reads once, in
// parallel. Returned struct is read-only and safe to share across
// miner-compute goroutines.
func fetchTipsetGlobals(ctx context.Context, api *lotusapi.FullNodeStruct, ts *types.TipSet) (*tipsetGlobals, error) {
	var (
		wg                    sync.WaitGroup
		rew, pow              builtin.FilterEstimate
		nv                    network.Version
		rewErr, powErr, nvErr error
	)
	wg.Add(3)
	go func() { defer wg.Done(); rew, rewErr = util.ThisEpochRewardsSmoothed(ctx, api, ts) }()
	go func() { defer wg.Done(); pow, powErr = util.TotalPowerSmoothed(ctx, api, ts) }()
	go func() { defer wg.Done(); nv, nvErr = api.StateNetworkVersion(ctx, ts.Key()) }()
	wg.Wait()
	if rewErr != nil {
		return nil, fmt.Errorf("ThisEpochRewardsSmoothed: %w", rewErr)
	}
	if powErr != nil {
		return nil, fmt.Errorf("TotalPowerSmoothed: %w", powErr)
	}
	if nvErr != nil {
		return nil, fmt.Errorf("StateNetworkVersion: %w", nvErr)
	}
	return &tipsetGlobals{EpochReward: rew, TotalQAPower: pow, NetworkVersion: nv}, nil
}

func computeOne(
	ctx context.Context,
	sdk poolstypes.PoolsSDK,
	lapi *lotusapi.FullNodeStruct,
	id uint64,
	addr common.Address,
	dtlInput invabigen.InvariantsQueryAgentDTLInputs,
	ts *types.TipSet,
	height *big.Int,
	tiers []poolsabigen.TierInfo,
	globals *tipsetGlobals,
	bufferBps uint64,
) AgentDTL {
	r := AgentDTL{AgentID: id, Address: addr, BufferBps: bufferBps}

	// Chain truth: liquid+miners+principal+interest came from the
	// batched InvariantsQuery call; we only need per-miner Lotus state
	// reads here.
	afi, err := computeAgentFiFromInputs(ctx, lapi, dtlInput, ts, globals)
	if err != nil {
		r.Err = fmt.Errorf("computeAgentFiFromInputs: %w", err)
		r.Severity = DTLError
		return r
	}
	r.Debt = afi.Debt()
	r.LV = afi.LiquidationValue()

	// SP+ tier — falls back to base tier 0 on error.
	tier, err := sdk.Query().SPPlusTierFromAgentAddress(ctx, addr, height)
	if err != nil {
		tier = 0
	}
	r.Tier = tier
	if int(tier) < len(tiers) {
		r.MaxDTL = new(big.Int).Set(tiers[tier].DebtToLiquidationValue)
	} else {
		r.MaxDTL = big.NewInt(0)
	}

	// DTL = debt × 1e18 / lv. Special-case zero collateral.
	wad := big.NewInt(1e18)
	if r.LV.Sign() == 0 {
		if r.Debt.Sign() == 0 {
			r.DTL = big.NewInt(0)
			r.Severity = DTLOK
		} else {
			r.DTL = new(big.Int).SetUint64(^uint64(0)) // sentinel "infinite"
			r.Severity = DTLNoLV
		}
		return r
	}
	r.DTL = new(big.Int).Quo(new(big.Int).Mul(r.Debt, wad), r.LV)

	// Severity buckets. bufferBps is a basis-points buffer below max
	// (e.g. 250 = 2.5%); the warning band is (max - buffer .. max].
	buffer := new(big.Int).Quo(
		new(big.Int).Mul(r.MaxDTL, new(big.Int).SetUint64(bufferBps)),
		big.NewInt(10000),
	)
	warnFloor := new(big.Int).Sub(r.MaxDTL, buffer)
	switch {
	case r.DTL.Cmp(r.MaxDTL) > 0:
		r.Severity = DTLOverMax
	case r.DTL.Cmp(warnFloor) > 0:
		r.Severity = DTLWarn
	default:
		r.Severity = DTLOK
	}
	return r
}

// computeAgentFiFromInputs assembles the agent's BaseFi from pre-
// batched InvariantsQuery inputs plus per-miner Lotus state reads.
// All EVM-side data (liquidAssets, miner IDs, principal, interest) is
// already in dtlInput; this function only issues Lotus RPCs.
func computeAgentFiFromInputs(
	ctx context.Context,
	lapi *lotusapi.FullNodeStruct,
	dtlInput invabigen.InvariantsQueryAgentDTLInputs,
	ts *types.TipSet,
	g *tipsetGlobals,
) (*econ.AgentFi, error) {
	minerAddrs := make([]address.Address, len(dtlInput.MinerIDs))
	for i, mid := range dtlInput.MinerIDs {
		a, err := address.NewIDAddress(mid)
		if err != nil {
			return nil, fmt.Errorf("address.NewIDAddress(%d): %w", mid, err)
		}
		minerAddrs[i] = a
	}

	tasks := make([]util.TaskFunc, len(minerAddrs))
	for i := range minerAddrs {
		m := minerAddrs[i]
		tasks[i] = func() (interface{}, error) {
			return computeMinerTerminationFeeCached(ctx, lapi, m, ts, g)
		}
	}
	results, err := util.Multiread(tasks)
	if err != nil {
		return nil, err
	}

	baseFis := make([]*econ.BaseFi, len(minerAddrs))
	for i, res := range results {
		baseFis[i] = res.(*econ.TerminateSectorResult).ToBaseFi()
	}

	return econ.NewAgentFi(
		dtlInput.LiquidAssets,
		econ.Liability{Principal: dtlInput.Principal, Interest: dtlInput.Interest},
		baseFis,
	), nil
}

// computeMinerTerminationFeeCached is a transcription of
// econ.ComputeMaxTerminationFee that takes the three tipset-globals as
// parameters instead of re-fetching them. Math identity is preserved —
// the only difference is which RPCs are made.
//
// Per-miner Lotus RPCs after caching:
//   - StateGetActor(miner)            — via util.LoadMinerActor
//   - ChainReadObj × 1–2 (state walk) — lazy via TieredBlockstore
//   - StateMinerPower(miner)
//   - StateMinerSectorCount(miner)
//
// Skipped (cached): StateGetActor(reward/power) + state walks + StateNetworkVersion.
func computeMinerTerminationFeeCached(
	ctx context.Context,
	api *lotusapi.FullNodeStruct,
	minerAddr address.Address,
	ts *types.TipSet,
	g *tipsetGlobals,
) (*econ.TerminateSectorResult, error) {
	result := &econ.TerminateSectorResult{
		TotalBalance:            big.NewInt(0),
		AvailableBalance:        big.NewInt(0),
		VestingFunds:            big.NewInt(0),
		InitialPledge:           big.NewInt(0),
		FeeDebt:                 big.NewInt(0),
		EstimatedTerminationFee: big.NewInt(0),
	}

	actor, mstate, err := util.LoadMinerActor(ctx, api, minerAddr, ts)
	if err != nil {
		return nil, fmt.Errorf("LoadMinerActor: %w", err)
	}

	lf, err := mstate.LockedFunds()
	if err != nil {
		return nil, fmt.Errorf("LockedFunds: %w", err)
	}
	result.TotalBalance = actor.Balance.Int
	result.VestingFunds = lf.VestingFunds.Int
	result.InitialPledge = lf.InitialPledgeRequirement.Int

	avail, err := mstate.AvailableBalance(actor.Balance)
	if err != nil {
		return nil, fmt.Errorf("AvailableBalance: %w", err)
	}

	feeDebt, err := mstate.FeeDebt()
	if err != nil {
		return nil, fmt.Errorf("FeeDebt: %w", err)
	}
	result.FeeDebt = feeDebt.Int

	switch {
	case feeDebt.Int.Sign() > 0 && avail.Int.Sign() > 0:
		return nil, fmt.Errorf("miner %s: fee debt and available balance both positive", minerAddr)
	case feeDebt.Int.Sign() > 0:
		result.AvailableBalance = big.NewInt(0)
	default:
		result.AvailableBalance = avail.Int
	}

	p, err := api.StateMinerPower(ctx, minerAddr, ts.Key())
	if err != nil {
		return nil, fmt.Errorf("StateMinerPower: %w", err)
	}
	sectorCount, err := api.StateMinerSectorCount(ctx, minerAddr, ts.Key())
	if err != nil {
		return nil, fmt.Errorf("StateMinerSectorCount: %w", err)
	}
	result.LiveSectors = sectorCount.Live
	result.FaultySectors = sectorCount.Faulty

	faultFee, err := minertypes.PledgePenaltyForContinuedFault(
		g.NetworkVersion,
		g.EpochReward,
		g.TotalQAPower,
		p.MinerPower.QualityAdjPower,
	)
	if err != nil {
		return nil, fmt.Errorf("PledgePenaltyForContinuedFault: %w", err)
	}

	penalty, err := minertypes.PledgePenaltyForTermination(
		g.NetworkVersion,
		lf.InitialPledgeRequirement,
		miner8.TerminationLifetimeCap*stbuiltin.EpochsInDay,
		faultFee,
	)
	if err != nil {
		return nil, fmt.Errorf("PledgePenaltyForTermination: %w", err)
	}
	result.EstimatedTerminationFee = penalty.Int

	return result, nil
}

// tierInfoLite is a thin local alias so the public type doesn't have
// to import the SPPlus abigen package.
type tierInfoLite = struct {
	CashBackPremium        *big.Int
	TokenLockAmount        *big.Int
	DebtToLiquidationValue *big.Int
}
