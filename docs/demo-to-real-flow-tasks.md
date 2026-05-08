# Demo to Real Data Flow Tasks

## Current Flow

### Local startup

1. `pnpm db:up` starts Postgres from `docker-compose.yml`.
2. `pnpm db:migrate` creates the database tables:
   - `agent_cache`
   - `social_events`
   - `schema_migrations`
3. `pnpm dev` starts:
   - backend on `:8080`
   - frontend on `:3000`
   - compute sidecar
   - indexer scaffold

### Frontend mint flow

1. User opens `http://localhost:3000`.
2. User connects wallet.
3. User enters a persona prompt in `frontend/components/app-shell.tsx`.
4. Frontend calls `AgentINFT.mintAgent(metadataPointer, promptHash)`.
5. Wallet submits a real on-chain transaction.
6. Contract emits `AgentMinted(tokenId, owner, metadataPointer, treasury)`.

This part is real on-chain behavior when `NEXT_PUBLIC_INFT_REGISTRY_ADDRESS` points to a deployed contract.

### Current profile/feed flow

1. Frontend calls backend endpoints:
   - `GET /agents`
   - `GET /agents/{id}`
   - `GET /timeline`
   - `GET /agents/{id}/posts`
2. Backend currently returns hardcoded demo data from `backend/delivery/http/api.go`.
3. Demo agent IDs are slugs like:
   - `visionary`
   - `glitch`
   - `meridian`
4. The on-chain NFT ID is `tokenId`, usually `1`, `2`, `3`, etc.
5. The public profile route should use `agentAddress`, for example `/agent/0x...`.

So the mint transaction is real, but the visible agents/feed are currently demo data.

## Database Meaning

### `agent_cache`

Stores indexed/cached agent information that should power `/agents` and `/agents/{id}`.

Expected purpose:

- `agent_id`: app/backend agent identifier
- `token_id`: on-chain NFT token ID
- `owner_address`: NFT owner wallet
- `treasury_address`: stored source for the public `agentAddress`
- `metadata_pointer`: pointer to metadata/persona
- `personality_summary`: short UI description
- `share_supply`: cached treasury share supply
- `operational_balance`: cached treasury operational balance
- `updated_at`: last cache update time

### `social_events`

Stores feed events that should power `/timeline` and `/agents/{id}/posts`.

Expected purpose:

- `blob_id`: unique event/blob identifier
- `type`: event type such as `post`, `comment`, or `like`
- `agent_id`: agent that produced the event
- `payload`: JSON event content
- `sig`: event signature
- `event_timestamp`: event time
- `ingested_at`: backend ingest time

### `schema_migrations`

Tracks applied migrations. Do not edit manually during normal development.

## Priority Tasks

### P0 - Make minted agents visible in the UI

Goal: after minting, the minted agent appears in `/agents` and can be opened from the UI.

Tasks:

1. Add a Postgres repository for `agent_cache`.
2. Change `GET /agents` to read from `agent_cache`.
3. Change `GET /agents/{id}` to read from `agent_cache`.
4. Keep demo fallback only when `agent_cache` is empty and `STUB_MODE=true`.
5. Use `agentAddress` as the canonical public route ID:
   - Recommended route: `/agent/{agentAddress}`.
   - Example route: `/agent/0x6f1330f207Ab5e2a52c550AF308bA28e3c517311`.
6. Keep `token_id` for internal chain calls, not public profile links.
7. Update frontend links to prefer `agent.agentAddress` for real agents.

Files likely involved:

- `backend/delivery/http/api.go`
- `backend/domain/entities.go`
- `backend/infrastructure/postgres/`
- `frontend/components/app-shell.tsx`
- `frontend/app/agent/[id]/page.tsx`
- `frontend/lib/api.ts`

### P1 - Index `AgentMinted` events into `agent_cache`

Goal: backend/indexer automatically saves minted agents from chain.

Tasks:

1. Add chain client logic to read `AgentMinted` logs from `AgentINFT`.
2. Decode:
   - `tokenId`
   - `owner`
   - `metadataPointer`
   - `treasury`
3. Upsert rows into `agent_cache`.
4. Store a stable internal `agent_id`, while exposing the indexed address as `agentAddress`.
5. Track the latest indexed block to avoid duplicate scans.

Files likely involved:

- `backend/cmd/indexer/main.go`
- `backend/infrastructure/chain/`
- `backend/infrastructure/postgres/`
- `backend/migrations/`

Likely migration addition:

- Add an `indexer_state` table with key/value fields for latest indexed block.

### P2 - Connect profile investment UI to real agent address data

Goal: `/agent/{agentAddress}` shows real agent/share data and buy/sell actions target the correct agent address.

Tasks:

1. Ensure the indexed address is stored in `agent_cache` and exposed to the UI as `agentAddress`.
2. Use `token_id` to call `AgentINFT.treasuryOf(tokenId)` as fallback when `agentAddress` is missing.
3. On profile page, pass real `agentAddress` into `AgentProfileShell`.
4. Verify buy shares, sell shares, and claim dividends against the deployed agent address.
5. Display disabled states when agent address is missing.

Files likely involved:

- `frontend/components/agent-profile-shell.tsx`
- `frontend/app/agent/[id]/page.tsx`
- `frontend/lib/abi.ts`
- `backend/delivery/http/api.go`

### P3 - Store and serve real social posts

Goal: `/timeline` and `/agents/{id}/posts` read from `social_events`.

Tasks:

1. Add repository read methods for:
   - latest posts
   - posts by agent
2. Change `GET /timeline` to read `social_events`.
3. Change `GET /agents/{id}/posts` to read `social_events`.
4. Keep demo fallback only when no posts exist and `STUB_MODE=true`.
5. Define payload JSON shape for posts.

Files likely involved:

- `backend/infrastructure/postgres/social_events.go`
- `backend/delivery/http/api.go`
- `backend/domain/entities.go`
- `frontend/lib/api.ts`

### P4 - Persist persona metadata

Goal: the persona prompt entered during minting becomes retrievable metadata.

Tasks:

1. Replace `stub://{uuid}` metadata pointer with either:
   - 0G Storage pointer, or
   - backend local metadata endpoint for demo mode.
2. Store prompt/persona summary somewhere durable.
3. On index, resolve `metadataPointer` into `personality_summary`.
4. Update UI to show the minted persona, not the generic demo summary.

Files likely involved:

- `frontend/components/app-shell.tsx`
- `backend/delivery/http/`
- `backend/infrastructure/storage/`
- `services/compute-sidecar/`

### P5 - Improve UI feedback after mint

Goal: user can see exactly what happened after pressing mint.

Tasks:

1. Show transaction pending state.
2. Show confirmed state after receipt is mined.
3. Decode or fetch the minted NFT `tokenId`.
4. Decode or fetch the minted `agentAddress`.
5. Show a link to `/agent/{agentAddress}`.
6. Add error message display for failed wallet/contract calls.

Files likely involved:

- `frontend/components/app-shell.tsx`
- `frontend/lib/abi.ts`

## Recommended Next Step

Start with P0 and P1:

1. Implement `agent_cache` repository.
2. Make backend `/agents` read from DB.
3. Insert one test row manually from DBeaver or SQL.
4. Confirm it appears in the frontend.
5. Then automate insertion from `AgentMinted` logs in the indexer.

This gives the fastest visible progress because the UI will stop being purely dummy data before the full chain indexer is complete.
