## ADDED Requirements

### Requirement: Profile uses indexed agent address

The agent profile SHALL use the agent address associated with the indexed token.

#### Scenario: Indexed agent address exists

- **WHEN** `/agent/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311` loads and `agent_cache` contains that agent address
- **THEN** investment reads and writes target that agent address

#### Scenario: Agent address is missing

- **WHEN** an agent has no valid agent address
- **THEN** buy, sell, top-up, and claim actions are disabled with a clear unavailable state

### Requirement: Agent address can be refreshed from chain

The system SHALL be able to derive or refresh agent address and share data from chain when needed.

#### Scenario: Agent cache lacks agent address but registry is configured

- **WHEN** token ID exists and `treasury_address` is empty
- **THEN** the system may call `treasuryOf(tokenId)` and update or use the returned address as the agent address

### Requirement: Investment actions preserve safety checks

The investment UI SHALL keep slippage and wallet connection checks when switching from demo data to a real agent address.

#### Scenario: User buys shares on a real agent address

- **WHEN** a connected user submits a buy with amount and max price
- **THEN** the contract call targets the indexed agent address and includes the max price slippage guard
