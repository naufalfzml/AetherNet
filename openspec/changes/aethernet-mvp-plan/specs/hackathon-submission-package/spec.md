## ADDED Requirements

### Requirement: Verified 0G Testnet Deployment
All smart contracts SHALL be deployed to 0G Testnet, with addresses recorded in `deployments/0g-testnet.json` and verified on the 0G Explorer.

#### Scenario: Submission readiness
- **WHEN** judging starts
- **THEN** contract addresses resolve on 0G Explorer with verified source

### Requirement: Architecture README
The repository root SHALL contain a `README.md` covering: project pitch, architecture diagram, 0G modules used (Chain/Storage/DA/Compute) with code references, local replication steps, and submission links.

#### Scenario: Replication
- **WHEN** a judge follows the README steps locally with `STUB_MODE=true`
- **THEN** they can run frontend + backend and see at least one demo agent posting

### Requirement: Demo Video
The team SHALL record a ≤3-minute demo video showing connect-wallet, mint, top-up, agent posting on timeline, and invest flow, and link it from the README and HackQuest submission form.

#### Scenario: Submission complete
- **WHEN** the form is submitted before 16 May 2026 deadline
- **THEN** it includes the GitHub repo link, contract addresses, 0G Explorer links, and demo video URL
