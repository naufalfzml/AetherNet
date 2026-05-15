// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {AgentTreasury} from "./AgentTreasury.sol";

/// @title AetherNet Agent Treasury Factory
/// @notice Deploys one treasury per agent iNFT and stores the resulting address mapping.
/// @dev The registry is the only caller allowed to mint new treasuries through this factory.
contract AgentTreasuryFactory is Ownable {
    /// @notice Raised when a caller is not the configured registry.
    error Unauthorized();
    /// @notice Raised when attempting to set the registry more than once.
    error RegistryAlreadySet();

    /// @notice Emitted when the registry address is initialized.
    /// @param registry Registry address authorized to create treasuries.
    event RegistrySet(address indexed registry);
    /// @notice Emitted when a treasury is created for a token id.
    /// @param tokenId Agent token id whose treasury was deployed.
    /// @param owner Owner of the treasury and associated agent.
    /// @param treasury Deployed treasury contract address.
    event TreasuryCreated(uint256 indexed tokenId, address indexed owner, address treasury);

    /// @notice Registry allowed to create new treasuries.
    address public registry;
    /// @notice Platform wallet that receives platform revenue splits from new treasuries.
    address public immutable platformWallet;
    /// @notice Base share price applied to every newly deployed treasury.
    uint256 public immutable basePrice;
    /// @notice Linear share-price slope applied to every newly deployed treasury.
    uint256 public immutable slope;

    /// @notice Mapping from agent token id to deployed treasury address.
    mapping(uint256 => address) public treasuryOf;

    /// @param platformWallet_ Platform wallet used by newly deployed treasuries.
    /// @param basePrice_ Base marginal share price for newly deployed treasuries.
    /// @param slope_ Linear share-price slope for newly deployed treasuries.
    constructor(address platformWallet_, uint256 basePrice_, uint256 slope_) Ownable(msg.sender) {
        platformWallet = platformWallet_;
        basePrice = basePrice_;
        slope = slope_;
    }

    /// @notice Initializes the registry allowed to create treasuries.
    /// @param registry_ Registry address that will own treasury creation rights.
    function setRegistry(address registry_) external onlyOwner {
        if (registry != address(0)) revert RegistryAlreadySet();
        registry = registry_;
        emit RegistrySet(registry_);
    }

    /// @notice Deploys and records a treasury for a newly minted agent.
    /// @param tokenId Agent token id that will own the treasury.
    /// @param agentOwner Owner of the agent and treasury.
    /// @param orchestrator Initial orchestrator for the new treasury.
    /// @return treasury Address of the deployed treasury.
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
