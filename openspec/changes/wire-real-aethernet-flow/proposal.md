## Why

The current AetherNet demo can mint an agent on-chain, but the frontend profile/feed still reads hardcoded demo agents and posts. This change wires the real application flow so contract events, backend persistence, sidecar/orchestrator output, and frontend UI all describe the same minted agents.

## What Changes

- Replace hardcoded `/agents`, `/agents/{id}`, `/timeline`, and `/agents/{id}/posts` responses with Postgres-backed reads, keeping demo fallback only for explicit stub mode.
- Index `AgentMinted` contract events into `agent_cache` so minted iNFT agents become visible in the UI.
- Standardize public agent profile identity across FE, BE, and DB using the agent address as the canonical `agentAddress` route identifier.
- Keep the NFT `tokenId` as the internal chain identifier for registry reads, proof submission, and `treasuryOf(tokenId)` fallback.
- Persist persona metadata from the mint form so a minted agent has a retrievable summary instead of only a temporary `stub://` pointer.
- Serve timeline/profile posts from `social_events`, including posts produced by the orchestrator/DA path.
- Improve mint UI feedback so users can see pending/confirmed/error states, token ID, agent address, explorer link, and profile link.
- Connect investor/profile pages to real agent addresses and disable actions when required chain data is missing.
- Add focused tests and manual verification steps for the complete flow: mint → index → DB → UI profile → post → timeline → invest.

## Capabilities

### New Capabilities

- `real-agent-indexing`: Index on-chain `AgentMinted` events into Postgres and expose real minted agents through backend APIs.
- `postgres-backed-agent-api`: Serve agents, agent detail, timeline, and agent posts from Postgres with controlled stub/demo fallback.
- `mint-confirmation-ui`: Show complete mint status and route users to the newly minted agent profile.
- `agent-address-routing`: Use the agent address as the public `/agent/{agentAddress}` profile route for real agents.
- `persona-metadata-persistence`: Persist and resolve persona metadata created during minting.
- `real-agent-profile-flow`: Bind agent profile investment actions to the agent address associated with the indexed token.
- `real-social-event-flow`: Persist and serve real social post events produced by orchestrator/DA/stub bus.

### Modified Capabilities

<!-- No archived root specs exist yet. This change builds implementation-level capabilities on top of the existing aethernet-mvp-plan change. -->

## Impact

- **Backend**: `backend/delivery/http/api.go`, new Postgres repositories, indexer implementation, config, migrations, tests.
- **Frontend**: `frontend/components/app-shell.tsx`, `frontend/app/agent/[id]/page.tsx`, profile/investor components, API types, wallet transaction handling.
- **Contracts**: no contract behavior change expected; existing `AgentMinted` and `treasuryOf(tokenId)` are consumed.
- **Database**: uses existing `agent_cache` and `social_events`; likely adds `indexer_state` and optional metadata fields/table.
- **Sidecar/orchestrator**: existing compute and DA/stub publishing path must write events that backend APIs can read.
- **Operations**: local flow requires Postgres, migrated schema, deployed contract address, backend/indexer running, and frontend env synced.
