// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC721} from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {AgentTreasuryFactory} from "./AgentTreasuryFactory.sol";

contract AgentINFT is ERC721, Ownable, ReentrancyGuard {
    error InsufficientMintFee();
    error Unauthorized();
    error TokenMissing();

    event AgentMinted(uint256 indexed tokenId, address indexed owner, string metadataPointer, address treasury);
    event MetadataUpdated(uint256 indexed tokenId, string oldPointer, string newPointer);
    event InferenceProofSubmitted(uint256 indexed tokenId, bytes32 proofHash, bytes proof);
    event UsageAuthorized(uint256 indexed tokenId, address indexed user, bool allowed);
    event AgentCloned(uint256 indexed sourceTokenId, uint256 indexed cloneTokenId, address indexed owner);
    event OrchestratorUpdated(address indexed orchestrator);

    struct AgentMetadata {
        string pointer;
        bytes32 promptHash;
        address treasury;
    }

    uint256 public immutable mintFee;
    AgentTreasuryFactory public immutable treasuryFactory;

    address public orchestrator;
    uint256 public nextTokenId = 1;

    mapping(uint256 => AgentMetadata) public agentMetadata;
    mapping(uint256 => bytes32) public latestProof;
    mapping(uint256 => mapping(address => bool)) public usageAuthorized;

    constructor(uint256 mintFee_, address orchestrator_, AgentTreasuryFactory treasuryFactory_)
        ERC721("AetherNet Agent iNFT", "AINFT")
        Ownable(msg.sender)
    {
        mintFee = mintFee_;
        orchestrator = orchestrator_;
        treasuryFactory = treasuryFactory_;
    }

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

    function setOrchestrator(address nextOrchestrator) external onlyOwner {
        orchestrator = nextOrchestrator;
        emit OrchestratorUpdated(nextOrchestrator);
    }

    function setMetadataPointer(uint256 tokenId, string calldata newPointer) external {
        _requireExisting(tokenId);
        if (msg.sender != ownerOf(tokenId) && msg.sender != orchestrator) revert Unauthorized();

        string memory oldPointer = agentMetadata[tokenId].pointer;
        agentMetadata[tokenId].pointer = newPointer;
        emit MetadataUpdated(tokenId, oldPointer, newPointer);
    }

    function submitInferenceProof(uint256 tokenId, bytes calldata proof) external {
        _requireExisting(tokenId);
        if (msg.sender != orchestrator) revert Unauthorized();

        bytes32 proofHash = keccak256(proof);
        latestProof[tokenId] = proofHash;
        emit InferenceProofSubmitted(tokenId, proofHash, proof);
    }

    function authorizeUsage(uint256 tokenId, address user, bool allowed) external {
        _requireExisting(tokenId);
        if (msg.sender != ownerOf(tokenId)) revert Unauthorized();

        usageAuthorized[tokenId][user] = allowed;
        emit UsageAuthorized(tokenId, user, allowed);
    }

    function clone(uint256 sourceTokenId, address to)
        external
        payable
        nonReentrant
        returns (uint256 cloneTokenId, address treasury)
    {
        _requireExisting(sourceTokenId);
        if (msg.sender != ownerOf(sourceTokenId) && !usageAuthorized[sourceTokenId][msg.sender]) revert Unauthorized();
        if (msg.value < mintFee) revert InsufficientMintFee();

        AgentMetadata memory source = agentMetadata[sourceTokenId];
        cloneTokenId = nextTokenId++;
        _safeMint(to, cloneTokenId);
        treasury = treasuryFactory.createTreasury(cloneTokenId, to, orchestrator);
        agentMetadata[cloneTokenId] =
            AgentMetadata({pointer: source.pointer, promptHash: source.promptHash, treasury: treasury});

        emit AgentCloned(sourceTokenId, cloneTokenId, to);
        emit AgentMinted(cloneTokenId, to, source.pointer, treasury);
    }

    function metadataPointer(uint256 tokenId) external view returns (string memory) {
        _requireExisting(tokenId);
        return agentMetadata[tokenId].pointer;
    }

    function promptHash(uint256 tokenId) external view returns (bytes32) {
        _requireExisting(tokenId);
        return agentMetadata[tokenId].promptHash;
    }

    function treasuryOf(uint256 tokenId) external view returns (address) {
        _requireExisting(tokenId);
        return agentMetadata[tokenId].treasury;
    }

    function _requireExisting(uint256 tokenId) internal view {
        if (_ownerOf(tokenId) == address(0)) revert TokenMissing();
    }
}
