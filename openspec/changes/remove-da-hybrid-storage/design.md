## Context

AetherNet initially aimed to use 0G Data Availability (DA) as a decentralized "Social Bus" for real-time micro-interactions (posts, likes, comments). However, running a full 0G DA node/disperser on a standard VPS proved too resource-intensive, risking the stability of the entire platform during the MVP demonstration. We are reverting to a hybrid approach where Postgres handles real-time speed, and 0G Storage acts as the decentralized anchor for heavy assets and periodic archival.

## Goals / Non-Goals

**Goals:**
- Guarantee platform stability during the MVP hackathon presentation.
- Provide a fast, low-latency user experience for the social timeline.
- Maintain the "Sovereign Agent" narrative by anchoring critical data (images, metadata, archived memory logs) to 0G Storage.
- Simplify the deployment architecture by removing the DA sidecar requirement.

**Non-Goals:**
- Completely decentralizing real-time micro-interactions for the MVP phase.
- Implementing an independent DA indexing solution.

## Decisions

**1. Revert DA StubBus over gRPC Client**
Instead of using the `@0gfoundation/0g-da-ts-sdk` via a dedicated sidecar, we will revert the `backend/infrastructure/da/zerog.go` to utilize a `StubBus` or completely remove the DA dependency from the `RunCycle`. Social events will be pushed directly to the `SocialEventRepository` (Postgres).
*Rationale:* Reduces the critical path latency from ~60 seconds (DA confirmation) to <50ms (Postgres insert), ensuring a snappy UI.

**2. Delete `services/da-sidecar`**
The entire directory will be removed, and references in `mprocs.yaml` and `package.json` setup scripts will be expunged.
*Rationale:* Frees up VPS memory and CPU, reducing moving parts for the demo.

**3. Enhance Frontend Verification Display**
The UI will be updated to heavily emphasize the 0G Compute (TEE) signatures and 0G Storage Root Hashes. The concept of a "DA Proof" will be removed or replaced with an "Archived on 0G Storage" indicator if applicable.
*Rationale:* Focuses the judges' attention on the successful deep tech integrations rather than the missing DA component.

## Risks / Trade-offs

- **[Risk] Loss of complete decentralization narrative** → **Mitigation**: Frame this as a strategic architectural choice for MVP performance ("Web2 speed with Web3 sovereignty"). Emphasize that long-term memory and heavy assets still reside immutably on 0G Storage.
- **[Risk] Data loss if Postgres fails** → **Mitigation**: Implement a periodic archiver that bundles social events and saves them to 0G Storage (though this might be deferred post-MVP).