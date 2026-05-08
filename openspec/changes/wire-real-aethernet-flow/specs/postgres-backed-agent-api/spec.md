## ADDED Requirements

### Requirement: List agents from Postgres

The backend SHALL serve `GET /agents` from `agent_cache` when real agent rows exist.

#### Scenario: Database contains indexed agents

- **WHEN** `agent_cache` contains one or more agents
- **THEN** `GET /agents` returns those agents instead of hardcoded demo agents

#### Scenario: Database is empty in stub mode

- **WHEN** `agent_cache` is empty and `STUB_MODE=true`
- **THEN** `GET /agents` may return demo fallback agents and indicate fallback behavior in logs

### Requirement: Fetch agent detail from Postgres

The backend SHALL serve `GET /agents/{id}` from `agent_cache`, resolving valid EVM address params by indexed agent address for real agents.

#### Scenario: Agent address exists

- **WHEN** `GET /agents/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311` is requested and that agent address exists in `agent_cache`
- **THEN** the backend returns that agent with owner, agent address, metadata pointer, and summary fields

#### Scenario: Agent address casing differs

- **WHEN** the requested address casing differs from the stored agent address casing
- **THEN** the backend still returns the matching agent

#### Scenario: Agent does not exist

- **WHEN** `GET /agents/0x0000000000000000000000000000000000000001` is requested and no matching row exists
- **THEN** the backend returns a not-found response unless explicit stub fallback applies

### Requirement: Keep API response shape stable

The backend SHALL preserve the frontend `Agent` response shape while changing the data source to Postgres.

#### Scenario: Frontend fetches real agent data

- **WHEN** the frontend calls `GET /agents`
- **THEN** every returned item includes `id`, `tokenId`, `ownerAddress`, `agentAddress`, `metadataPointer`, and `personalitySummary`
