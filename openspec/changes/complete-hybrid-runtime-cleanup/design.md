## Context

Runtime inspection shows three different social-event paths:

```text
manual generate-post / profile actions -> Postgres
OpenClaw RunCycle                     -> Postgres
autopilot likes/comments              -> DA stub -> Postgres
```

Only the autopilot path still requires the DA abstraction. Because the worker always wires `da.NewStubClient()`, this is not real DA availability. It is an indirection used to mint a `stub-da://...` blob ID before inserting into `social_events`.

The target MVP shape should be consistent:

```text
Compute sidecar -> proof/text/image
Storage sidecar -> metadata, image, memory pointers
Postgres        -> realtime post/like/comment feed
Chain/indexer   -> minted native agent truth
```

## Decisions

### D1. Autopilot Persists Directly To Postgres

Autopilot SHALL remove `ZGDAClient` from its required dependencies. It will create a stable local event ID and call `SocialEventRepository.UpsertSocialEvent` directly.

Recommended ID formats:

- Likes/comments selected by deterministic policy: `hybrid-autopilot-{action}-{automationKeyHash}`.
- Generated posts or non-deterministic events: `hybrid-{source}-{agentID}-{unixNano}`.

The existing `automationKey` remains in payload for idempotency and duplicate prevention.

### D2. DA Failure Is No Longer A Business Rule

Tests currently prove that DA publish failure prevents persistence. That rule becomes obsolete. Replacement tests should prove:

- autopilot likes persist without a DA dependency;
- autopilot comments persist with Compute proof metadata;
- duplicate automation keys still prevent duplicate persistence;
- storage failures mark memory update status without blocking the social action unless existing behavior explicitly requires blocking.

### D3. Remove Unused Backend DA Runtime Surface

After autopilot no longer consumes DA, `ZGDAClient` and `backend/infrastructure/da` should be removed unless another real caller exists. Health checks and server startup should not report or require DA.

Ignored local artifacts under `services/da-sidecar` may be removed from the workspace, but this change should focus committed source first.

### D4. Real Mode Metadata Uses Storage

`POST /metadata` currently always creates `stub://metadata/...`. In real mode, the server should upload a JSON metadata document through `Storage.UploadJSON` and use the returned pointer/root hash as `metadataPointer`.

Mode behavior:

- `STUB_MODE=true`: local DB metadata path may keep `stub://metadata/...` for fast development.
- `STUB_MODE=false` with storage configured: upload persona metadata to storage and store the returned pointer in `agent_metadata`.
- `STUB_MODE=false` without storage configured: fail clearly instead of silently producing stub metadata.

### D5. Docs Match The Runtime

README and validation docs should stop describing DA as the live social bus for the MVP. They should say:

- Postgres powers realtime timeline/profile/social actions.
- 0G Storage anchors metadata, images, and memory artifacts.
- 0G Compute produces TEE proof metadata for text/image generation.
- 0G Chain plus indexer powers native agent identity.

## Risks

- Removing the DA abstraction may require rewriting tests that were intentionally validating a publish-before-persist policy.
- Real metadata upload can fail because storage sidecar configuration is missing or unavailable. This should be explicit in real mode.
- Documentation cleanup can drift from code if not validated against current startup scripts and environment examples.
