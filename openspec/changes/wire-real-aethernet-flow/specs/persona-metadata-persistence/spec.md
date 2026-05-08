## ADDED Requirements

### Requirement: Persist mint persona input

The system SHALL persist the persona prompt entered during minting so it can be resolved for the agent profile.

#### Scenario: User mints with a prompt

- **WHEN** the user submits a persona prompt during mint
- **THEN** the system stores retrievable metadata associated with the minted token or metadata pointer

### Requirement: Resolve personality summary

The backend SHALL expose a `personalitySummary` for indexed agents using persisted metadata when available.

#### Scenario: Metadata exists for indexed token

- **WHEN** an indexed agent has stored persona metadata
- **THEN** `GET /agents/{id}` returns a personality summary derived from that metadata

#### Scenario: Metadata is missing

- **WHEN** an indexed agent has no resolvable metadata
- **THEN** `GET /agents/{id}` returns an explicit fallback summary rather than unrelated demo copy

### Requirement: Support stub and real metadata pointers

The metadata persistence flow SHALL support local/stub pointers for development and 0G Storage pointers for real mode.

#### Scenario: Stub mode metadata is used

- **WHEN** `STUB_MODE=true` and the user mints an agent
- **THEN** the metadata pointer resolves through the local/stub metadata path

#### Scenario: Real storage pointer is used

- **WHEN** real storage mode is configured and upload succeeds
- **THEN** the metadata pointer resolves to the 0G Storage-backed metadata
