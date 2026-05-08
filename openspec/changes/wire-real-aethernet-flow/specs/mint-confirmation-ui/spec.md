## ADDED Requirements

### Requirement: Show transaction lifecycle

The mint form SHALL show pending, confirmed, and failed states for the wallet transaction.

#### Scenario: Transaction is pending

- **WHEN** the user submits the mint form and the wallet transaction is waiting for confirmation
- **THEN** the UI disables duplicate submission and shows a pending state

#### Scenario: Transaction fails

- **WHEN** the wallet or contract call fails
- **THEN** the UI shows a readable error message and allows the user to retry

### Requirement: Decode minted agent details

The mint flow SHALL display the minted token ID and agent address after the mint transaction is confirmed.

#### Scenario: Receipt includes AgentMinted event

- **WHEN** the confirmed transaction receipt includes an `AgentMinted` event
- **THEN** the UI displays the token ID, agent address, explorer link, and profile link

#### Scenario: Event decoding is unavailable

- **WHEN** the transaction is confirmed but the UI cannot decode the event
- **THEN** the UI still displays the transaction hash and tells the user indexing may take time

### Requirement: Link to real profile by address

The mint confirmation SHALL route to the real agent address profile.

#### Scenario: Minted agent address is known

- **WHEN** agent address `0x6f1330f207Ab5e2a52c550AF308bA28e3c517311` is decoded from the receipt
- **THEN** the UI provides a link to `/agent/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311`

### Requirement: Distinguish token ID and agent address

The mint confirmation UI SHALL label token ID and agent address separately.

#### Scenario: Mint details are shown

- **WHEN** mint confirmation displays both values
- **THEN** the UI labels token ID as the NFT token ID and address value as the agent address
