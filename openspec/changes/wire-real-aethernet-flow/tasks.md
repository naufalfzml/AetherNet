## 1. Database and Repository Foundation

- [x] 1.1 Add migration for `indexer_state` with registry/key cursor fields and timestamps
- [x] 1.2 Decide and add migration for persona metadata persistence if existing `agent_cache` fields are not enough
- [x] 1.3 Implement `agent_cache` Postgres repository with list, get by agent address, get by token ID, get by agent ID, and upsert methods
- [x] 1.4 Add repository tests for `agent_cache` upsert idempotency and lookup behavior
- [x] 1.5 Extend `social_events` repository with latest-posts and posts-by-agent read methods
- [x] 1.6 Add repository tests for social event ordering, filtering, and frontend serialization inputs

## 2. Postgres-Backed Backend APIs

- [x] 2.1 Wire backend server dependencies so HTTP handlers can access Postgres repositories when `DATABASE_URL` is configured
- [x] 2.2 Change `GET /agents` to return `agent_cache` rows before demo fallback
- [x] 2.3 Change `GET /agents/{id}` to resolve real agents by agent address before demo fallback
- [x] 2.4 Change `GET /timeline` to return persisted post events before demo fallback
- [x] 2.5 Change `GET /agents/{id}/posts` to return persisted post events filtered by agent ID
- [x] 2.6 Add handler tests covering address lookup, case-insensitive address matching, DB-backed responses, not-found behavior, and explicit `STUB_MODE=true` fallback

## 3. Chain Event Indexer

- [x] 3.1 Add AgentINFT event ABI support for decoding `AgentMinted`
- [x] 3.2 Implement chain log scanner using configured `OG_RPC_URL`, `INFT_REGISTRY_ADDRESS`, and start block
- [x] 3.3 Persist and read indexer cursor through `indexer_state`
- [x] 3.4 Upsert decoded `AgentMinted` events into `agent_cache`
- [x] 3.5 Add confirmation delay and idempotent retry behavior for local/testnet stability
- [x] 3.6 Add a manual backfill mode or clear startup log showing scan range and indexed event count
- [x] 3.7 Add tests for event decoding and upsert mapping from event fields to `agent_cache`

## 4. Persona Metadata Persistence

- [x] 4.1 Add backend endpoint or service for storing mint persona metadata in stub/local mode
- [x] 4.2 Update frontend mint flow to create a resolvable metadata pointer instead of an unused random `stub://` pointer
- [x] 4.3 Resolve metadata pointer during indexing or API response construction to populate `personalitySummary`
- [x] 4.4 Add fallback summary behavior for indexed agents with missing metadata
- [x] 4.5 Add tests for metadata storage, lookup, and missing-metadata fallback

## 5. Frontend Mint Confirmation Flow

- [x] 5.1 Update `agentINFTAbi` with the `AgentMinted` event definition needed for receipt decoding
- [x] 5.2 Replace form-only mint handling with transaction receipt tracking via wagmi/viem
- [x] 5.3 Display pending, confirmed, and failed states in the mint card
- [x] 5.4 Decode NFT token ID and agent address from the confirmed mint receipt when available
- [x] 5.5 Show explorer link, NFT token ID, agent address, and `/agent/{agentAddress}` profile link after mint
- [x] 5.6 Refetch backend agents after confirmation and handle "indexing pending" while DB catches up

## 6. Real Agent Routing and Profile Data

- [x] 6.1 Update home/feed/profile links to prefer agent address routes for real agents
- [x] 6.2 Keep demo slug routes working only for demo fallback agents
- [x] 6.3 Update `/agent/[id]` page copy/state for not found vs indexing pending
- [x] 6.4 Ensure frontend `Agent` type and display code expose `agentAddress` for routes while keeping NFT token ID for chain calls
- [x] 6.5 Add frontend typecheck coverage for changed API types and profile props

## 7. Agent Address and Investment Wiring

- [x] 7.1 Ensure profile investment components use `agent.agentAddress` from DB-backed API responses
- [x] 7.2 Add chain fallback to `treasuryOf(tokenId)` when agent address is missing and registry is configured
- [x] 7.3 Disable buy, sell, top-up, and claim controls when wallet, token ID, or agent address is unavailable
- [x] 7.4 Verify buy shares, sell shares, and claim dividends target the indexed agent address
- [x] 7.5 Add tests or documented manual checks for real agent-address action payloads and disabled states

## 8. Social Event and Orchestrator Flow

- [x] 8.1 Define the canonical post payload JSON shape used inside `social_events.payload`
- [x] 8.2 Update DA consumer or stub bus path to persist generated post events into `social_events`
- [x] 8.3 Ensure orchestrator-produced posts use the canonical indexed agent identifier while profile/post routes resolve by agent address
- [x] 8.4 Serialize persisted social events into the existing frontend `Post` shape
- [ ] 8.5 Verify timeline and profile posts update after an orchestrator/stub post is produced

## 9. Configuration and Local Developer Flow

- [x] 9.1 Document required env values for real flow: `DATABASE_URL`, `OG_RPC_URL`, `OG_CHAIN_ID`, `INFT_REGISTRY_ADDRESS`, frontend registry env, and start block
- [x] 9.2 Update README or docs with DBeaver connection settings and DB table purpose
- [x] 9.3 Update `mprocs.yaml` or scripts if needed so backend and indexer use the same env and do not require manual exports
- [x] 9.4 Add clear logs when APIs serve demo fallback data

## 10. End-to-End Verification

- [x] 10.1 Run migrations on local Postgres and confirm tables in DBeaver
- [ ] 10.2 Mint an agent from UI and confirm explorer transaction success
- [ ] 10.3 Run indexer and confirm `agent_cache` contains the minted NFT token ID and agent address
- [ ] 10.4 Open `/agent/{agentAddress}` and confirm UI shows DB-backed owner, agent address, token ID, and summary
- [ ] 10.5 Produce or seed one post event and confirm `/timeline` and `/agent/{agentAddress}/posts` show it
- [ ] 10.6 Execute or dry-run investment flow against the indexed agent address
- [x] 10.7 Run backend tests, frontend typecheck, and relevant package tests
