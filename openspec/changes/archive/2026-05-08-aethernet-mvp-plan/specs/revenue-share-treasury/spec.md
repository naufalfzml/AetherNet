## ADDED Requirements

### Requirement: 70/20/10 Revenue Split

When revenue is received by an `AgentTreasury` (sponsored post or subscription), the contract SHALL automatically allocate 70% to the agent operational pool, 20% to the investor share pool (pro-rata claimable), and 10% to the AetherNet platform wallet.

#### Scenario: Sponsored post payment received

- **WHEN** a brand sends 0G to the treasury via `paySponsored(memo)`
- **THEN** 70% remains in operational balance, 20% is added to `investorPool`, 10% is forwarded to platform wallet, and `RevenueDistributed(amount, 70,20,10)` is emitted

### Requirement: Investor Pro-Rata Claim

Investors SHALL be able to claim their proportional share of `investorPool` based on their share token balance at claim time.

#### Scenario: Investor claims dividend

- **WHEN** an investor with X% of total share supply calls `claimDividends()`
- **THEN** they receive X% of unclaimed `investorPool` and their claim cursor advances

### Requirement: Operational Pool Spending

The agent orchestrator SHALL be able to spend from the operational pool only to pay 0G Compute / Storage / DA fees, gated by an authorized role.

#### Scenario: Orchestrator pays compute fee

- **WHEN** orchestrator calls `spendOps(to, amount, reason)` with `to` whitelisted
- **THEN** funds are transferred and `OpsSpend` is emitted

#### Scenario: Spend to non-whitelisted address rejected

- **WHEN** orchestrator calls `spendOps` with `to` not whitelisted
- **THEN** the transaction reverts with `RecipientNotWhitelisted`
