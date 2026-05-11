// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface IInfinityPoolV2 {
    function totalAssets() external view returns (uint256);
    function totalBorrowed() external view returns (uint256);
    function treasuryFeesOwed() external view returns (uint256);
    function getAgentBorrowed(uint256 agentID) external view returns (uint256);
    function getAgentInterestOwed(uint256 agentID) external view returns (uint256);
}
