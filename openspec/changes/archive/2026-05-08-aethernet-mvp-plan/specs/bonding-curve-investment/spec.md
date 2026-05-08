## ADDED Requirements

### Requirement: Linear Bonding Curve Pricing

Each `AgentTreasury` SHALL price investor shares using a linear curve `price(n) = basePrice + slope * supply`, denominated in 0G token, where `supply` is the current total share count.

#### Scenario: Price increases with supply

- **WHEN** an investor queries `getBuyPrice(amount)` after N shares already minted
- **THEN** the returned price equals the integral of the linear curve from `N` to `N+amount`

### Requirement: Buy Investor Shares

The system SHALL allow any wallet to buy shares of an agent by transferring 0G token equal to the bonding-curve price; the treasury mints share tokens to the buyer.

#### Scenario: Successful buy

- **WHEN** caller sends `getBuyPrice(amount)` worth of 0G to `buyShares(amount, maxPrice)`
- **THEN** `amount` share tokens are minted to caller, `SharesBought` event emitted, and treasury supply increases

#### Scenario: Slippage protection

- **WHEN** the actual price exceeds `maxPrice` provided by caller
- **THEN** the transaction reverts with `SlippageExceeded`

### Requirement: Sell Investor Shares

The system SHALL allow share holders to sell back shares to the treasury at the current curve price, burning the shares.

#### Scenario: Successful sell

- **WHEN** holder calls `sellShares(amount, minPrice)` and curve price ≥ minPrice
- **THEN** shares are burned, holder receives 0G payout, and `SharesSold` is emitted
