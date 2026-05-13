## Why

The current runtime is only partially hybrid. Timeline reads, manual generated posts, and the orchestrator already persist social events directly to Postgres, but the autopilot worker still depends on a DA abstraction and publishes through a stub DA client before persistence. The mint metadata path also still creates `stub://metadata/...` pointers even when the rest of the stack is configured for real services.

This creates ambiguity during demos and reviews: the product looks hybrid, but some runtime paths still claim DA semantics or stub pointers. We need one clear MVP architecture: Postgres is the realtime social bus, 0G Storage anchors metadata/assets/memory, 0G Compute provides TEE-backed inference, and 0G Chain/indexer provides agent truth.

## What Changes

- Make autopilot likes/comments persist directly to `social_events` without `ZGDAClient` or `stub-da://` blob IDs.
- Remove the backend DA runtime surface once it has no production caller.
- Keep Postgres as the source for `/timeline`, `/agents/{id}/posts`, external feed reads, and autopilot action counting.
- Update metadata creation so real mode uploads persona metadata to 0G Storage and stores a storage pointer, while `stub://metadata/...` remains only for explicit local stub mode.
- Clean up stale DA-facing docs and runtime references that imply a required DA sidecar.
- Add real-flow verification tasks for mint, index, profile, fresh posts, autopilot actions, and investment targeting.

## Capabilities

### New Capabilities

- `hybrid-autopilot-runtime`: Autopilot social actions use the same Postgres-backed social bus as manual and orchestrator-generated events.
- `real-mode-metadata-storage`: Mint persona metadata can be anchored through 0G Storage in real mode.
- `real-flow-verification`: The chain-to-UI flow is verified end-to-end with real indexed agents and fresh timeline content.

### Modified Capabilities

- Existing social event behavior is tightened so DA is not on the runtime critical path for MVP realtime interactions.

## Impact

- Backend: `backend/usecase/autopilot.go`, `backend/usecase/ports.go`, `backend/cmd/worker/main.go`, tests, and unused DA infrastructure.
- Backend HTTP: metadata creation in `backend/delivery/http/api.go` and related tests.
- Storage: existing storage sidecar/client abstractions are reused for real metadata upload.
- Docs: README and validation docs should describe hybrid Postgres social events, 0G Storage anchoring, and Compute proof behavior accurately.
- Operations: real mode requires `STUB_MODE=false`, `DATABASE_URL`, chain/indexer env, compute sidecar URL, and storage sidecar URL.

## Non-Goals

- Reintroducing a real 0G DA sidecar or DA indexer for MVP realtime events.
- Changing smart contract behavior.
- Building production-grade archival batching beyond storing metadata/assets/memory through 0G Storage.
