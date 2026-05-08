// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "forge-std/Script.sol";
import "../src/InvariantsQuery.sol";

/// Forge deploy script. Pulls upstream addresses from env vars so the same
/// script works against mainnet / calibration without code edits:
///
///   ROUTER_ADDR             — pools Router (proxy)
///   INFINITY_POOL_ADDR      — InfinityPoolV2 (proxy)
///   IFIL_ADDR               — iFIL ERC-20 (LiquidStaking pool token)
///   LP_PLUS_PROXY           — LPPlus proxy
///   SP_PLUS_PROXY           — SPPlusV2 proxy
///   MINER_REGISTRY_ADDR     — pools MinerRegistry (used by DTL batch reads)
///
/// Run:
///   forge script script/Deploy.s.sol \
///     --rpc-url $RPC --private-key $PK --broadcast
contract Deploy is Script {
    function run() external returns (InvariantsQuery deployed) {
        address routerAddr = vm.envAddress("ROUTER_ADDR");
        address poolAddr = vm.envAddress("INFINITY_POOL_ADDR");
        address ifilAddr = vm.envAddress("IFIL_ADDR");
        address lpPlusAddr = vm.envAddress("LP_PLUS_PROXY");
        address spPlusAddr = vm.envAddress("SP_PLUS_PROXY");
        address minerRegistryAddr = vm.envAddress("MINER_REGISTRY_ADDR");

        vm.startBroadcast();
        deployed = new InvariantsQuery(
            routerAddr,
            poolAddr,
            ifilAddr,
            lpPlusAddr,
            spPlusAddr,
            minerRegistryAddr
        );
        vm.stopBroadcast();

        console.log("InvariantsQuery deployed:", address(deployed));
    }
}
