// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC721} from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {AgentTreasuryFactory} from "./AgentTreasuryFactory.sol";
import {AgentTreasury} from "./AgentTreasury.sol";

/// @title AetherNet Agent iNFT Registry
/// @notice Mints and manages agent identity NFTs together with their per-agent treasuries.
/// @dev The registry is the source of truth for agent ownership, metadata pointers, proof hashes, and treasury links.
contract AgentINFT is ERC721, Ownable, ReentrancyGuard {
    /// @notice Raised when a mint call does not send the required mint fee.
    error InsufficientMintFee();
    /// @notice Raised when the caller is not authorized for the requested action.
    error Unauthorized();
    /// @notice Raised when a referenced token does not exist.
    error TokenMissing();

    /// @notice Emitted when a new agent iNFT and treasury are created.
    /// @param tokenId Newly minted agent token id.
    /// @param owner Wallet that owns the agent iNFT.
    /// @param metadataPointer Initial offchain metadata pointer for the agent.
    /// @param treasury Treasury contract deployed for the agent.
    event AgentMinted(uint256 indexed tokenId, address indexed owner, string metadataPointer, address treasury);
    /// @notice Emitted when an agent metadata pointer is updated.
    /// @param tokenId Agent token id whose metadata changed.
    /// @param oldPointer Previous metadata pointer.
    /// @param newPointer New metadata pointer.
    event MetadataUpdated(uint256 indexed tokenId, string oldPointer, string newPointer);
    /// @notice Emitted when the orchestrator submits a proof for an inference cycle.
    /// @param tokenId Agent token id associated with the proof.
    /// @param proofHash Hash of the submitted proof payload.
    /// @param proof Full serialized proof payload.
    event InferenceProofSubmitted(uint256 indexed tokenId, bytes32 proofHash, bytes proof);
    /// @notice Emitted when the registry-level orchestrator is rotated.
    /// @param orchestrator New orchestrator address.
    event OrchestratorUpdated(address indexed orchestrator);
    /// @notice Emitted when accrued mint fees are withdrawn from the registry.
    /// @param recipient Address receiving the withdrawn fees.
    /// @param amount Amount of native token withdrawn.
    event MintFeesWithdrawn(address indexed recipient, uint256 amount);

    /// @notice Metadata tracked for each minted agent.
    /// @param pointer Offchain metadata pointer, typically a 0G storage reference.
    /// @param promptHash Hash of the initial personality prompt used to seed the agent.
    /// @param treasury Treasury contract deployed for the agent.
    struct AgentMetadata {
        string pointer;
        bytes32 promptHash;
        address treasury;
    }

    /// @notice Fixed native-token fee required to mint a new agent.
    uint256 public immutable mintFee;
    /// @notice Factory used to deploy a treasury for every newly minted agent.
    AgentTreasuryFactory public immutable treasuryFactory;

    /// @notice Address allowed to submit inference proofs and update metadata.
    address public orchestrator;
    /// @notice Monotonic counter used to assign the next token id.
    uint256 public nextTokenId = 1;

    /// @notice Per-agent metadata and treasury mapping keyed by token id.
    mapping(uint256 => AgentMetadata) public agentMetadata;
    /// @notice Latest submitted proof hash keyed by token id.
    mapping(uint256 => bytes32) public latestProof;

    /// @param mintFee_ Fixed mint fee charged for every new agent.
    /// @param orchestrator_ Initial orchestrator allowed to manage inference-related actions.
    /// @param treasuryFactory_ Factory used to deploy per-agent treasuries.
    constructor(uint256 mintFee_, address orchestrator_, AgentTreasuryFactory treasuryFactory_)
        ERC721("AetherNet Agent iNFT", "AINFT")
        Ownable(msg.sender)
    {
        mintFee = mintFee_;
        orchestrator = orchestrator_;
        treasuryFactory = treasuryFactory_;
    }

    /// @notice Mints a new agent iNFT and deploys its dedicated treasury.
    /// @param initialMetadataPointer Initial offchain metadata pointer for the agent.
    /// @param initialPromptHash Hash of the initial prompt that defines the agent persona.
    /// @return tokenId Newly minted agent token id.
    /// @return treasury Treasury address created for the new agent.
    function mintAgent(string calldata initialMetadataPointer, bytes32 initialPromptHash)
        external
        payable
        nonReentrant
        returns (uint256 tokenId, address treasury)
    {
        if (msg.value < mintFee) revert InsufficientMintFee();

        tokenId = nextTokenId++;
        _safeMint(msg.sender, tokenId);

        treasury = treasuryFactory.createTreasury(tokenId, msg.sender, orchestrator);
        agentMetadata[tokenId] =
            AgentMetadata({pointer: initialMetadataPointer, promptHash: initialPromptHash, treasury: treasury});

        emit AgentMinted(tokenId, msg.sender, initialMetadataPointer, treasury);
    }

    /// @notice Rotates the registry orchestrator and synchronizes every deployed treasury.
    /// @param nextOrchestrator New orchestrator address.
    function setOrchestrator(address nextOrchestrator) external onlyOwner nonReentrant {
        orchestrator = nextOrchestrator;

        for (uint256 tokenId = 1; tokenId < nextTokenId; tokenId++) {
            address treasury = agentMetadata[tokenId].treasury;
            if (treasury != address(0)) {
                AgentTreasury(payable(treasury)).syncOrchestrator(nextOrchestrator);
            }
        }

        emit OrchestratorUpdated(nextOrchestrator);
    }

    /// @notice Updates the metadata pointer for an existing agent.
    /// @dev Callable by the agent owner or the configured orchestrator.
    /// @param tokenId Agent token id to update.
    /// @param newPointer New offchain metadata pointer.
    function setMetadataPointer(uint256 tokenId, string calldata newPointer) external {
        _requireExisting(tokenId);
        if (msg.sender != ownerOf(tokenId) && msg.sender != orchestrator) revert Unauthorized();

        string memory oldPointer = agentMetadata[tokenId].pointer;
        agentMetadata[tokenId].pointer = newPointer;
        emit MetadataUpdated(tokenId, oldPointer, newPointer);
    }

    /// @notice Submits an inference proof for an existing agent.
    /// @dev Callable only by the configured orchestrator.
    /// @param tokenId Agent token id associated with the proof.
    /// @param proof Full serialized proof payload.
    function submitInferenceProof(uint256 tokenId, bytes calldata proof) external {
        _requireExisting(tokenId);
        if (msg.sender != orchestrator) revert Unauthorized();

        bytes32 proofHash = keccak256(proof);
        latestProof[tokenId] = proofHash;
        emit InferenceProofSubmitted(tokenId, proofHash, proof);
    }

    /// @notice Withdraws accumulated mint fees from the registry.
    /// @param recipient Address receiving the native-token withdrawal.
    /// @param amount Amount of native token to withdraw.
    function withdrawMintFees(address payable recipient, uint256 amount) external onlyOwner nonReentrant {
        (bool ok,) = recipient.call{value: amount}("");
        require(ok, "withdraw failed");
        emit MintFeesWithdrawn(recipient, amount);
    }

    /// @notice Returns the current metadata pointer for a minted agent.
    /// @param tokenId Agent token id to inspect.
    /// @return Current offchain metadata pointer.
    function metadataPointer(uint256 tokenId) external view returns (string memory) {
        _requireExisting(tokenId);
        return agentMetadata[tokenId].pointer;
    }

    /// @notice Returns the initial prompt hash for a minted agent.
    /// @param tokenId Agent token id to inspect.
    /// @return Hash of the initial agent prompt.
    function promptHash(uint256 tokenId) external view returns (bytes32) {
        _requireExisting(tokenId);
        return agentMetadata[tokenId].promptHash;
    }

    /// @notice Returns the treasury address for a minted agent.
    /// @param tokenId Agent token id to inspect.
    /// @return Treasury contract address for the agent.
    function treasuryOf(uint256 tokenId) external view returns (address) {
        _requireExisting(tokenId);
        return agentMetadata[tokenId].treasury;
    }

    /// @notice Reverts when a token id has not been minted.
    /// @param tokenId Agent token id to validate.
    function _requireExisting(uint256 tokenId) internal view {
        if (_ownerOf(tokenId) == address(0)) revert TokenMissing();
    }
}
