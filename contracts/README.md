# InvariantsQuery contract

View-only batch helper for the `glifio/invariants` tool. Bundles
per-agent state, pool metrics, and LP+/SP+ per-token state into single
view calls so the off-chain tool stays under the chain.love RPC budget.

The contract is **tooling**, not protocol — it's pinned upstream
contract addresses at construction and routes reads through them.
Cheap to redeploy if any of those addresses change.

## Layout

```
contracts/
├── foundry.toml
├── src/
│   ├── InvariantsQuery.sol
│   └── interfaces/   # vendored stubs of Router/InfinityPoolV2/LPPlus/SPPlusV2/IERC20
└── script/
    └── Deploy.s.sol
```

The interface files are intentionally minimal — only the methods this
contract calls. They're copies, not imports — `glifio/invariants` is
self-contained at the source level.

## Build

```
make forge-build       # cd contracts && forge build
```

Requires Foundry: `curl -L https://foundry.paradigm.xyz | bash && foundryup`.

## Regenerate Go bindings

```
make abigen
```

Calls `forge build` then runs `abigen` against the artifact, writing to
`../abigen/InvariantsQuery.go`. Requires `abigen` from go-ethereum and
`jq` for ABI extraction.

## Deploy

```
ROUTER_ADDR=0x...           # pools Router (proxy)
INFINITY_POOL_ADDR=0x...    # InfinityPoolV2 (proxy)
IFIL_ADDR=0x...             # iFIL ERC-20
LP_PLUS_PROXY=0x...         # LPPlus proxy
SP_PLUS_PROXY=0x...         # SPPlusV2 proxy

forge script script/Deploy.s.sol \
  --rpc-url $RPC --private-key $PK --broadcast
```

Then update `mainnet.env` (or equivalent) with the new
`INVARIANTS_QUERY_ADDR`.

## Methods

- `getAgentsState(uint256[] agentIDs) → AgentState[]` — per-agent
  principal/epochsPaid/interestOwed/defaulted, one batched call instead
  of N×Router.getAccount + N×pool.getAgentInterestOwed.
- `getPoolMetrics() → PoolMetrics` — totalAssets, totalBorrowed,
  treasuryFeesOwed, ifilSupply.
- `getLPPlusStates(uint256[] tokenIds) → LPPlusTokenState[]` — owner +
  RWT/YBT balance + isActive per token.
- `getSPPlusStates(uint256[] tokenIds) → SPPlusTokenState[]` — owner per
  token (+ optional agent binding).
- `getSPPlusTokenIdsForAgents(uint256[] agentIds) → uint256[]` — resolve
  cards bound to agents.

All view-only. No mutating methods.
