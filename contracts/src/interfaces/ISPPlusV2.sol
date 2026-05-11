// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface ISPPlusV2 {
    function ownerOf(uint256 tokenId) external view returns (address);
    function agentIdToTokenId(uint256 agentId) external view returns (uint256);
    function agentIdToCardOwner(uint256 agentId) external view returns (address);
}
