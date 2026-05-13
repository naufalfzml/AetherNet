## 1. Remove DA Sidecar Dependency

- [ ] 1.1 Delete the `services/da-sidecar` directory.
- [ ] 1.2 Remove `@aethernet/da-sidecar` from `mprocs.yaml`.
- [ ] 1.3 Ensure `pnpm run setup` scripts do not reference the removed sidecar.

## 2. Refactor Backend Go Client

- [ ] 2.1 Revert `backend/infrastructure/da/zerog.go` to use a `StubBus` or remove the HTTP Client logic targeting port 3003.
- [ ] 2.2 Update `backend/cmd/server/main.go` to remove the `DA_SIDECAR_URL` environment variable parsing and instantiation.

## 3. Enforce Hybrid Postgres Storage

- [ ] 3.1 Update `backend/usecase/orchestrator.go` `RunCycle` to bypass the DA publishing step and directly insert into `SocialEventRepository` if not already handled by the mock.
- [ ] 3.2 Ensure `RunReplyLoop` is strictly using database polling (hybrid approach) instead of DA subscriptions.

## 4. Update Frontend UI

- [ ] 4.1 Remove references to "DA Proof" or "Available on 0G DA" in the UI.
- [ ] 4.2 Emphasize "TEE Verified" for post text (from Compute sidecar).
- [ ] 4.3 Emphasize "Archived on 0G Storage" for images and metadata (from Storage sidecar).