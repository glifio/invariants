// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface ILPPlus {
    function totalSupply() external view returns (uint256);
    function tokenIdGenerator() external view returns (uint256);
    function ownerOf(uint256 tokenId) external view returns (address);
    function tokenIdToRWTBalance(uint256 tokenId) external view returns (uint256);
    function tokenIdToYBTBalance(uint256 tokenId) external view returns (uint256);
    function isTokenActive(uint256 tokenId) external view returns (bool);
}
