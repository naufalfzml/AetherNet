## ADDED Requirements

### Requirement: Index minted agents from chain

The system SHALL index `AgentMinted` events from the configured AgentINFT registry into `agent_cache`.

#### Scenario: AgentMinted event is persisted

- **WHEN** the indexer observes an `AgentMinted(tokenId, owner, metadataPointer, treasury)` event from `INFT_REGISTRY_ADDRESS`
- **THEN** it upserts an `agent_cache` row containing the token ID, owner address, metadata pointer, agent address, and updated timestamp

#### Scenario: Re-running indexer is idempotent

- **WHEN** the indexer processes an already indexed `AgentMinted` event again
- **THEN** it updates the existing `agent_cache` row without creating a duplicate agent

### Requirement: Track indexer progress

The system SHALL persist chain indexing progress so restarts resume from the last processed block.

#### Scenario: Indexer restarts after progress was saved

- **WHEN** the indexer starts after a previous successful scan
- **THEN** it resumes from the stored cursor instead of scanning from genesis

#### Scenario: Indexer has no saved cursor

- **WHEN** no cursor exists for the AgentINFT registry
- **THEN** the indexer starts from the configured deployment block or a configured fallback block

### Requirement: Use agent address as public agent route identifier

The system SHALL use the indexed agent address as the public route identifier for real minted agents.

#### Scenario: Indexed agent is exposed with address route identity

- **WHEN** token ID `1` is indexed with agent address `0x6f1330f207Ab5e2a52c550AF308bA28e3c517311`
- **THEN** backend and frontend profile links identify the real agent as `/agent/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311`

### Requirement: Preserve token ID for chain operations

The system SHALL keep the NFT `tokenId` available for registry reads and other chain operations.

#### Scenario: Indexed agent is returned through API

- **WHEN** an indexed agent is returned by the backend
- **THEN** the response includes both `tokenId` and `agentAddress`
