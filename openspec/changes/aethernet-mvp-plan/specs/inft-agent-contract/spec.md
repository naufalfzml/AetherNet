## ADDED Requirements

### Requirement: Mint iNFT Agent
The system SHALL allow any wallet to mint a new agent as an iNFT (ERC-7857-compatible) on 0G Chain by paying a base mint fee and providing an initial metadata pointer (0G Storage hash) plus a personality prompt hash.

#### Scenario: Successful mint
- **WHEN** a connected wallet calls `mintAgent(metadataPointer, promptHash)` with the required fee
- **THEN** a new tokenId is assigned, ownership recorded, an `AgentTreasury` is deployed for that token, and an `AgentMinted(tokenId, owner, metadataPointer)` event is emitted

#### Scenario: Mint with insufficient fee
- **WHEN** the caller sends less than `mintFee`
- **THEN** the transaction reverts with `InsufficientMintFee`

### Requirement: Dynamic Metadata Pointer
The system SHALL allow the agent owner (or an authorized orchestrator role) to update the agent's metadata pointer to a new 0G Storage root hash, representing evolved personality/memory state.

#### Scenario: Owner updates metadata pointer
- **WHEN** the agent owner calls `setMetadataPointer(tokenId, newPointer)`
- **THEN** the on-chain pointer is updated and a `MetadataUpdated` event is emitted with old and new pointer

#### Scenario: Unauthorized update rejected
- **WHEN** a non-owner / non-orchestrator calls `setMetadataPointer`
- **THEN** the transaction reverts with `Unauthorized`

### Requirement: Proof of Inference Verification Hook
The contract SHALL accept a `submitInferenceProof(tokenId, proof)` call from the orchestrator role, store the latest proof hash, and emit it for off-chain verification.

#### Scenario: Orchestrator submits proof
- **WHEN** orchestrator calls `submitInferenceProof(tokenId, proof)`
- **THEN** `latestProof[tokenId]` is set and `InferenceProofSubmitted` is emitted
