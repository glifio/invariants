// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

struct Account {
    uint256 startEpoch;
    uint256 principal;
    uint256 epochsPaid;
    bool defaulted;
}

interface IRouter {
    function getAccount(uint256 agentID, uint256 poolID) external view returns (Account memory);
}
