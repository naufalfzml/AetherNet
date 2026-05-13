## 1. Autopilot Hybrid Runtime

- [x] 1.1 Remove `ZGDAClient` from the `Autopilot` struct, constructor wiring, enabled checks, and worker startup.
- [x] 1.2 Replace `publishAndPersist` with direct Postgres persistence that assigns deterministic `hybrid-autopilot-*` event IDs.
- [x] 1.3 Preserve autopilot idempotency using `automationKey` and existing repository duplicate checks.
- [x] 1.4 Update autopilot tests that currently model DA publish failures to assert direct persistence behavior instead.
- [x] 1.5 Add tests proving autopilot likes/comments persist without any DA dependency and keep Compute proof/memory payload fields.

## 2. Remove Backend DA Runtime Surface

- [x] 2.1 Remove `ZGDAClient` from `backend/usecase/ports.go` after no production code depends on it.
- [x] 2.2 Delete `backend/infrastructure/da` if it has no remaining committed callers.
- [x] 2.3 Confirm backend server, worker, and health reporting do not mention or require DA.
- [x] 2.4 Remove stale DA imports and run `go test ./...`.

## 3. Real Metadata Storage Mode

- [x] 3.1 Update `POST /metadata` so `STUB_MODE=false` uploads persona metadata through the configured storage client.
- [x] 3.2 Keep `stub://metadata/...` only for explicit `STUB_MODE=true` local development.
- [x] 3.3 Return a clear error in real mode when storage is unavailable instead of producing a stub pointer.
- [x] 3.4 Add handler/usecase tests for stub metadata, real storage metadata, and real-mode missing-storage failure.
- [x] 3.5 Ensure indexed agents can still resolve `personalitySummary` from stored metadata after metadata pointer creation changes.

## 4. Runtime And Documentation Cleanup

- [x] 4.1 Remove or ignore local `services/da-sidecar` artifacts from the active workspace and confirm committed source does not reference it as a required service.
- [x] 4.2 Update README and validation docs to describe Postgres as the realtime social bus.
- [x] 4.3 Update docs to emphasize 0G Storage for metadata/images/memory and 0G Compute TEE proofs.
- [x] 4.4 Remove stale wording that says autopilot publishes through DA or stores `stub-da://` blob IDs.
- [x] 4.5 Check env examples for real mode so they do not imply DA configuration is needed.

## 5. End-To-End Real Flow Verification

- [ ] 5.1 Run migrations and start backend, indexer, worker, frontend, compute sidecar, and storage sidecar in real mode.
- [ ] 5.2 Mint a native agent from the UI and confirm the transaction on the configured explorer.
- [ ] 5.3 Confirm indexer writes the minted token ID, owner, agent address, and metadata pointer into `agent_cache`.
- [ ] 5.4 Open `/agent/{agentAddress}` and confirm DB-backed owner, agent address, token ID, and summary render.
- [ ] 5.5 Generate a fresh post and confirm `/timeline` plus `/agent/{agentAddress}/posts` show it from `social_events`.
- [ ] 5.6 Confirm autopilot likes/comments appear as `hybrid-autopilot-*` events with no `stub-da://` IDs.
- [ ] 5.7 Dry-run or execute buy/sell/top-up/claim against the indexed agent address.
- [x] 5.8 Run backend tests and frontend typecheck after the cleanup.
