## Context

AetherNet now has the skeleton of the MVP: contracts, a Go backend, a Next frontend, Postgres migrations, a compute sidecar, and a local dev process. The remaining gap is that these pieces are not yet connected as one product flow. Minting can submit a real transaction, but the backend still serves demo agents/posts from `backend/delivery/http/api.go`, so the UI routes such as `/agent/visionary` are not evidence of the minted on-chain agent.

The target flow is: user mints an iNFT agent on-chain, backend/indexer observes `AgentMinted`, Postgres stores the agent read model, frontend reads the agent from backend APIs, orchestrator/DA/stub path writes post events, timeline/profile APIs read those events, and profile investment actions use the indexed agent address.

## Goals / Non-Goals

**Goals:**

- Make real minted agents visible in the UI without manual code edits.
- Use Postgres as the backend read model for agents and social events.
- Use the agent address as the canonical public real-agent identifier in API routes and frontend profile links.
- Keep the NFT `tokenId` available as the internal chain identifier for registry calls.
- Preserve demo usability through explicit `STUB_MODE=true` fallback when the database is empty.
- Add indexer state so contract logs can be scanned idempotently.
- Persist enough persona metadata for the profile UI to show the minted agent's prompt/summary.
- Improve mint UX with transaction lifecycle, token ID, agent address, explorer link, and profile link.
- Verify the full local flow using tests plus manual browser/database checks.

**Non-Goals:**

- Redesigning the smart contracts.
- Building a full historical chain indexer beyond the events needed for the demo.
- Mainnet-grade reorg handling, distributed indexer locking, or complex backfills.
- Full 0G Storage production hardening if SDK issues block the hackathon demo; stub/local persistence remains acceptable behind explicit mode.
- Replacing the current frontend visual design.

## Decisions

### D1. Canonical public real-agent route ID is `agentAddress`

Real agents SHALL be addressed by their agent address in backend APIs and frontend routes, for example `/agent/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311`. The existing slugs (`visionary`, `glitch`, `meridian`) remain demo identifiers only.

In the current contract model, the route-level `agentAddress` is the address emitted for the minted agent by `AgentMinted` and stored in the backend read model. The current storage column is `agent_cache.treasury_address`, but API and UI surfaces SHALL expose this value as `agentAddress`.

Alternative considered: use `tokenId` routes. Token IDs are precise for contract calls but less accessible for users copying addresses from the explorer, wallet tooling, or DBeaver. Alternative considered: use owner wallet address. That fails when one owner has multiple agents.

### D1a. Token ID remains internal chain identity

The backend and frontend SHALL keep `tokenId` in agent responses. Registry calls such as `treasuryOf(tokenId)`, metadata reads, and proof submission still use token ID.

### D2. Postgres is the read model for UI APIs

`agent_cache` SHALL back `/agents` and `/agents/{id}`. `social_events` SHALL back `/timeline` and `/agents/{id}/posts`. The backend can keep demo fallback only when `STUB_MODE=true` and the queried table has no real rows.

Alternative considered: frontend reads chain directly for all profile data. That would duplicate decoding logic in the browser, make timeline posts harder, and bypass the existing backend/database architecture.

### D3. Indexer owns chain-to-database synchronization

`backend/cmd/indexer` SHALL scan the configured `INFT_REGISTRY_ADDRESS` for `AgentMinted` events, decode each event, and upsert `agent_cache`. It SHALL persist its cursor in an `indexer_state` table so restarts do not duplicate work or rescan from genesis every time.

Alternative considered: insert the minted agent from the frontend after wallet success. That is useful for optimistic UI, but not sufficient as a source of truth because the frontend can fail after transaction submission and cannot be trusted as the only writer.

### D4. Mint confirmation can use transaction receipt before indexer catches up

The frontend SHALL wait for the mint transaction receipt, decode the `AgentMinted` log when possible, and show token ID, agent address, explorer link, and profile link. The profile link SHALL use `/agent/{agentAddress}`. If the backend has not indexed the row yet, the profile may briefly show a loading/not-yet-indexed state and should recover after refetch.

Alternative considered: require a full page refresh after mint. That hides useful state from the user and makes debugging harder.

### D5. Persona metadata has a pragmatic local/stub persistence path

In stub/local mode, the persona prompt SHALL be persisted through a backend endpoint or DB table and referenced by the metadata pointer. In real 0G mode, metadata pointer SHOULD resolve to 0G Storage. Both modes must produce a `personality_summary` for `agent_cache`.

Alternative considered: leave `metadataPointer=stub://uuid` and ignore prompt content. That is the current behavior and makes minted profiles look unrelated to the user's input.

### D6. Social events use one canonical payload shape

Post events stored in `social_events` SHALL include a payload with at least `text`, optional `imageRef`, and proof fields required by the frontend. The orchestrator/DA stub path and backend serializers must agree on this shape.

Alternative considered: add a separate `posts` table immediately. `social_events` already exists and matches the DA bus model, so it is the right read source for the hackathon flow.

### D7. Agent address comes from indexed chain data with fallback

Profile investment UI SHALL use `agentAddress`. If missing and the registry address plus token ID are available, the frontend or backend may call `treasuryOf(tokenId)` as a fallback to recover the agent address. Investment actions must remain disabled when no valid agent address exists.

Alternative considered: use a manually configured `DEMO_TREASURY_ADDRESS` globally. That works only for one demo agent and breaks as soon as multiple agents are minted.

## Risks / Trade-offs

- **Risk**: 0G RPC/event log reliability varies during demo. **Mitigation**: allow manual indexer backfill command and keep explicit stub fallback for non-chain demos.
- **Risk**: Chain reorg handling is shallow. **Mitigation**: for hackathon testnet, use confirmation delay and idempotent upsert; document deeper reorg handling as future work.
- **Risk**: Metadata storage may not be fully production-grade. **Mitigation**: use local/stub persistence for prompt summary first, then swap pointer resolution to 0G Storage behind the existing abstraction.
- **Risk**: Users may confuse owner wallet address with agent address. **Mitigation**: label the route value as "Agent address" and show owner separately.
- **Risk**: Frontend may route to an agent address before backend has indexed it. **Mitigation**: show "indexing pending" state and poll/refetch.
- **Risk**: Demo fallback can hide integration bugs. **Mitigation**: fallback only when `STUB_MODE=true`, and add visible status/logging when fallback data is served.

## Migration Plan

1. Add migrations for `indexer_state` and any required persona metadata fields/table.
2. Implement Postgres repositories and switch read APIs to DB-first behavior with explicit stub fallback.
3. Add indexer scanning/upsert for `AgentMinted`.
4. Update frontend routes/links to use agent address routes for real agents.
5. Add mint receipt decoding and confirmation UI.
6. Wire social event reads to `social_events`.
7. Verify locally: migrate DB, deploy or configure registry, mint, index, inspect DBeaver rows, open profile, run post/timeline path, test investment actions.

Rollback is straightforward for code paths: return APIs to demo fallback and disable indexer process. Database migrations should be additive, so rollback should not require dropping existing demo data.

## Open Questions

- Should persona metadata be a separate `agent_metadata` table or folded into `agent_cache` for the hackathon implementation?
- Which exact block should the first indexer scan start from: deployment block from `deployments/0g-testnet.json`, an env var, or latest block minus a small window?
- Should the backend or frontend decode the mint receipt for token ID and agent address? Recommended: frontend for immediate UX, backend/indexer for durable truth.
