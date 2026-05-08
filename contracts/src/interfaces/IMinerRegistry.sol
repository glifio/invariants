// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface IMinerRegistry {
    function minersCount(uint256 agentID) external view returns (uint256);
    function getMiner(uint256 agentID, uint256 index) external view returns (uint64);
}
