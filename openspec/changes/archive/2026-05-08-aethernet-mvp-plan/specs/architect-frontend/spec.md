## ADDED Requirements

### Requirement: Wallet Connect

The frontend SHALL allow users to connect a Web3 wallet (MetaMask via RainbowKit/wagmi) configured for 0G Testnet chainId.

#### Scenario: Connect wallet

- **WHEN** user clicks "Connect Wallet" and approves in MetaMask
- **THEN** the app shows their address, balance in 0G, and gates Architect actions to authenticated state

### Requirement: Mint Agent Flow

The frontend SHALL provide a multi-step Architect flow: input personality prompt → upload to 0G Storage → call `mintAgent` → display new tokenId and treasury address.

#### Scenario: Successful mint

- **WHEN** user submits a valid prompt and signs the mint tx
- **THEN** UI shows transaction hash, new tokenId, and a deep link to 0G Explorer

### Requirement: Top-Up Operational Gas

The frontend SHALL allow agent owners to top-up their agent's `AgentTreasury` operational balance with 0G token.

#### Scenario: Successful top-up

- **WHEN** owner submits a top-up amount and signs
- **THEN** the treasury operational balance reflects the new amount in UI within one block

### Requirement: Investor Buy Shares Flow

The frontend SHALL display the current bonding-curve price and allow any user to buy shares with slippage control.

#### Scenario: Buy with slippage protection

- **WHEN** user enters share amount and slippage %
- **THEN** UI computes `maxPrice` and submits `buyShares(amount, maxPrice)`

### Requirement: Investor Dashboard

The frontend SHALL render an investor dashboard showing owned agents, share balances, claimable dividends, and a button to call `claimDividends`.

#### Scenario: Claim dividends

- **WHEN** a user with claimable dividends clicks "Claim"
- **THEN** the tx is sent and on success the claimable balance is reset to zero
