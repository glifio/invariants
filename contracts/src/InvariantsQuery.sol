// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./interfaces/IRouter.sol";
import "./interfaces/IInfinityPoolV2.sol";
import "./interfaces/IERC20.sol";
import "./interfaces/ILPPlus.sol";
import "./interfaces/ISPPlusV2.sol";
import "./interfaces/IAgent.sol";
import "./interfaces/IMinerRegistry.sol";

/// @title InvariantsQuery
/// @notice Read-only batch helper for the glifio/invariants tool. Bundles
/// per-agent state, pool metrics, and LP+/SP+ per-token state into single
/// view calls so the off-chain tool stays under the chain.love RPC budget
/// and the per-agent loop drops from 246 sequential calls to 1 batched call.
///
/// View only — no state changes, no admin. Cheap to redeploy if any
/// upstream contract address changes. The constructor pins the upstream
/// addresses; all reads route through them.
contract InvariantsQuery {
    IRouter public immutable router;
    IInfinityPoolV2 public immutable pool;
    IERC20 public immutable ifil;
    ILPPlus public immutable lpPlus;
    ISPPlusV2 public immutable spPlus;
    IMinerRegistry public immutable minerRegistry;
    uint256 public constant POOL_ID = 0;

    struct AgentState {
        uint256 startEpoch;
        uint256 principal;
        uint256 epochsPaid;
        uint256 interestOwed;
        bool defaulted;
    }

    struct PoolMetrics {
        uint256 totalAssets;
        uint256 totalBorrowed;
        uint256 treasuryFeesOwed;
        uint256 ifilSupply;
    }

    struct LPPlusTokenState {
        address owner;
        uint256 rwtBalance;
        uint256 ybtBalance;
        bool isActive;
    }

    struct SPPlusTokenState {
        address owner;
        uint256 agentId; // 0 if not bound (sentinel — agentId 0 doesn't exist)
    }

    /// @notice Per-agent inputs for off-chain DTL computation. The minerIDs
    /// are raw Filecoin actor IDs; the caller wraps them in f0 addresses to
    /// query Lotus. Pulls four reads (liquidAssets + miner list + principal
    /// + interest) into one batched eth_call so the off-chain tool drops
    /// from ~10 EVM calls per agent to one for the whole fleet.
    struct AgentDTLInputs {
        uint256 liquidAssets;
        uint64[] minerIDs;
        uint256 principal;
        uint256 interest;
    }

    constructor(
        address _router,
        address _pool,
        address _ifil,
        address _lpPlus,
        address _spPlus,
        address _minerRegistry
    ) {
        router = IRouter(_router);
        pool = IInfinityPoolV2(_pool);
        ifil = IERC20(_ifil);
        lpPlus = ILPPlus(_lpPlus);
        spPlus = ISPPlusV2(_spPlus);
        minerRegistry = IMinerRegistry(_minerRegistry);
    }

    // ------------------------------------------------------------------
    // Agent state
    // ------------------------------------------------------------------

    /// @notice Per-agent state: principal + epochsPaid + interestOwed +
    /// defaulted. One Router.getAccount + one pool.getAgentInterestOwed
    /// per agent, batched into a single returndata blob.
    function getAgentsState(uint256[] calldata agentIDs)
        external
        view
        returns (AgentState[] memory out)
    {
        out = new AgentState[](agentIDs.length);
        for (uint256 i = 0; i < agentIDs.length; i++) {
            Account memory acct = router.getAccount(agentIDs[i], POOL_ID);
            out[i] = AgentState({
                startEpoch: acct.startEpoch,
                principal: acct.principal,
                epochsPaid: acct.epochsPaid,
                interestOwed: pool.getAgentInterestOwed(agentIDs[i]),
                defaulted: acct.defaulted
            });
        }
    }

    // ------------------------------------------------------------------
    // Agent DTL inputs (chain-fresh)
    // ------------------------------------------------------------------

    /// @notice Per-agent inputs needed by the off-chain DTL alert.
    /// Bundles agent.liquidAssets() + minerRegistry miner list +
    /// router.getAccount.principal + pool.getAgentInterestOwed into one
    /// returndata blob. agentAddrs must align with agentIDs (the Go
    /// caller already has both from the indexer DB).
    function getAgentsDTLInputs(
        uint256[] calldata agentIDs,
        address[] calldata agentAddrs
    ) external view returns (AgentDTLInputs[] memory out) {
        require(agentIDs.length == agentAddrs.length, "len mismatch");
        out = new AgentDTLInputs[](agentIDs.length);
        for (uint256 i = 0; i < agentIDs.length; i++) {
            uint256 id = agentIDs[i];
            Account memory acct = router.getAccount(id, POOL_ID);
            uint256 count = minerRegistry.minersCount(id);
            uint64[] memory ids = new uint64[](count);
            for (uint256 j = 0; j < count; j++) {
                ids[j] = minerRegistry.getMiner(id, j);
            }
            out[i] = AgentDTLInputs({
                liquidAssets: IAgent(agentAddrs[i]).liquidAssets(),
                minerIDs: ids,
                principal: acct.principal,
                interest: pool.getAgentInterestOwed(id)
            });
        }
    }

    // ------------------------------------------------------------------
    // Pool metrics
    // ------------------------------------------------------------------

    function getPoolMetrics() external view returns (PoolMetrics memory) {
        return PoolMetrics({
            totalAssets: pool.totalAssets(),
            totalBorrowed: pool.totalBorrowed(),
            treasuryFeesOwed: pool.treasuryFeesOwed(),
            ifilSupply: ifil.totalSupply()
        });
    }

    // ------------------------------------------------------------------
    // iFIL per-holder balances
    // ------------------------------------------------------------------

    /// @notice Batch iFIL balanceOf. Replaces the defunct legacy query
    /// helper (QUERY_ADDR) the per-depositor check used to depend on —
    /// that contract pinned pre-upgrade addresses and now returns zeros.
    function getIFILBalances(address[] calldata holders)
        external
        view
        returns (uint256[] memory out)
    {
        out = new uint256[](holders.length);
        for (uint256 i = 0; i < holders.length; i++) {
            out[i] = ifil.balanceOf(holders[i]);
        }
    }

    // ------------------------------------------------------------------
    // LP Plus per-token state
    // ------------------------------------------------------------------

    function getLPPlusStates(uint256[] calldata tokenIds)
        external
        view
        returns (LPPlusTokenState[] memory out)
    {
        out = new LPPlusTokenState[](tokenIds.length);
        for (uint256 i = 0; i < tokenIds.length; i++) {
            uint256 id = tokenIds[i];
            // ownerOf reverts on burned/non-existent tokens; swallow with try/catch
            // so the caller can still see balances on the rest of the batch.
            address owner;
            try lpPlus.ownerOf(id) returns (address o) {
                owner = o;
            } catch {
                owner = address(0);
            }
            out[i] = LPPlusTokenState({
                owner: owner,
                rwtBalance: lpPlus.tokenIdToRWTBalance(id),
                ybtBalance: lpPlus.tokenIdToYBTBalance(id),
                isActive: lpPlus.isTokenActive(id)
            });
        }
    }

    function getLPPlusTotalSupply() external view returns (uint256) {
        return lpPlus.totalSupply();
    }

    // ------------------------------------------------------------------
    // SP Plus per-token state
    // ------------------------------------------------------------------

    function getSPPlusStates(uint256[] calldata tokenIds)
        external
        view
        returns (SPPlusTokenState[] memory out)
    {
        out = new SPPlusTokenState[](tokenIds.length);
        for (uint256 i = 0; i < tokenIds.length; i++) {
            uint256 id = tokenIds[i];
            address owner;
            try spPlus.ownerOf(id) returns (address o) {
                owner = o;
            } catch {
                owner = address(0);
            }
            out[i] = SPPlusTokenState({owner: owner, agentId: 0});
        }
    }

    /// @notice Resolve the token bound to each agentID (0 means unbound).
    function getSPPlusTokenIdsForAgents(uint256[] calldata agentIds)
        external
        view
        returns (uint256[] memory out)
    {
        out = new uint256[](agentIds.length);
        for (uint256 i = 0; i < agentIds.length; i++) {
            out[i] = spPlus.agentIdToTokenId(agentIds[i]);
        }
    }
}
