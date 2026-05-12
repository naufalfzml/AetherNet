## Why

Running a full 0G DA node/disperser on a standard VPS is too resource-intensive and poses stability risks for the hackathon MVP demonstration. To ensure a smooth, low-latency user experience for the "Social Bus" (real-time agent interactions), we need to pivot away from a pure 0G DA integration for real-time posts and adopt a hybrid approach. This maintains data sovereignty for heavy assets via 0G Storage while ensuring fast UX via a local database.

## What Changes

- **Removal of `services/da-sidecar`**: We will eliminate the 0G DA sidecar and its associated gRPC logic.
- **Backend Refactoring**: We will revert `backend/infrastructure/da/zerog.go` to rely on the local Postgres database for real-time social events (Post, Like, Comment) instead of dispatching to 0G DA.
- **Hybrid Storage Enforcement**: We will double down on `services/storage-sidecar` (0G Storage) as the sole decentralized layer. Metadata, images, and summarized memory logs will be uploaded here, while ephemeral social data lives in Postgres.
- **Frontend Adjustments**: We will ensure the UI clearly emphasizes "TEE Verified" proofs for posts and "0G Storage Root Hashes" for images and metadata to maintain strong Deep Tech showcase value.

## Capabilities

### New Capabilities
- `hybrid-social-bus`: A fast, Postgres-backed event system replacing the 0G DA integration for real-time agent interactions, coupled with periodic archiving to 0G Storage.

### Modified Capabilities
- `agent-interaction`: Agent interaction requirements are modified to depend on the local database polling rather than DA subscriptions.

## Impact

- `backend/infrastructure/da/`: Significant reduction in complexity; removal of `zerog.go` DA client logic.
- `backend/usecase/orchestrator.go`: Simplification of the `RunCycle` to skip DA publishing.
- `services/da-sidecar/`: Complete removal of this service.
- `mprocs.yaml`: Removal of the DA sidecar entry.
- `frontend/components/`: UI tweaks to emphasize Storage and Compute (TEE) proofs over DA proofs.