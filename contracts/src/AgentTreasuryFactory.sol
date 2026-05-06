// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {AgentTreasury} from "./AgentTreasury.sol";

contract AgentTreasuryFactory is Ownable {
    error Unauthorized();
    error RegistryAlreadySet();

    event RegistrySet(address indexed registry);
    event TreasuryCreated(uint256 indexed tokenId, address indexed owner, address treasury);

    address public registry;
    address public immutable platformWallet;
    uint256 public immutable basePrice;
    uint256 public immutable slope;

    mapping(uint256 => address) public treasuryOf;

    constructor(address platformWallet_, uint256 basePrice_, uint256 slope_) Ownable(msg.sender) {
        platformWallet = platformWallet_;
        basePrice = basePrice_;
        slope = slope_;
    }

    function setRegistry(address registry_) external onlyOwner {
        if (registry != address(0)) revert RegistryAlreadySet();
        registry = registry_;
        emit RegistrySet(registry_);
    }

    function createTreasury(uint256 tokenId, address agentOwner, address orchestrator)
        external
        returns (address treasury)
    {
        if (msg.sender != registry) revert Unauthorized();
        treasury =
            address(new AgentTreasury(tokenId, agentOwner, platformWallet, registry, orchestrator, basePrice, slope));
        treasuryOf[tokenId] = treasury;
        emit TreasuryCreated(tokenId, agentOwner, treasury);
    }
}
