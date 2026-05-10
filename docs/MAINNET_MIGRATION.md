# Mainnet migration checklist

The codebase is environment-agnostic; switching from 0G testnet to
mainnet should be a pure configuration change. Use this checklist when
preparing the production deployment.

## 1. Wallet & funds

- [ ] New mainnet wallet provisioned (do not reuse the testnet key).
- [ ] Wallet funded with OG on mainnet for gas (mint, settlement, etc.).
- [ ] 0G Compute ledger funded for the mainnet wallet via
      `0g-compute-cli deposit --amount <N>` (mainnet network selected
      with `0g-compute-cli setup-network`).
- [ ] Router API key generated against the mainnet pool (separate from
      testnet) and saved securely.

## 2. Compute sidecar

Update `services/compute-sidecar/.env`:

| Variable             | Testnet                     | Mainnet                    |
| -------------------- | --------------------------- | -------------------------- |
| `STUB_MODE`          | `false`                     | `false`                    |
| `ZG_ROUTER_BASE_URL` | testnet router URL          | mainnet router URL         |
| `ZG_ROUTER_API_KEY`  | testnet `sk-...` / `pc-...` | mainnet key                |
| `ZG_CHAT_MODEL`      | testnet model id            | mainnet model id           |
| `ZG_IMAGE_MODE`      | `mock` (or `edit`)          | `generate`                 |
| `ZG_IMAGE_MODEL`     | `qwen/qwen-image-edit-2511` | `z-ai/...` (text-to-image) |

Validation steps:

- [ ] `curl $ZG_ROUTER_BASE_URL/models -H "Authorization: Bearer $ZG_ROUTER_API_KEY"`
      returns the chat and image models you configured.
- [ ] `curl -X POST $ZG_ROUTER_BASE_URL/chat/completions ...` returns a
      completion using the mainnet chat model.
- [ ] `curl -X POST $ZG_ROUTER_BASE_URL/images/generations ...` returns
      `b64_json` (or a URL) for the mainnet text-to-image model.
- [ ] If the mainnet text-to-image response shape differs from the
      OpenAI-compatible default (`data[].b64_json` / `data[].url`),
      adjust `runRouterImageGenerate` in
      `services/compute-sidecar/src/image.ts` to match.

## 3. Backend & chain

Update root `.env`:

- [ ] `OG_RPC_URL`, `OG_CHAIN_ID`, `OG_EXPLORER_URL` point at mainnet.
- [ ] `INFT_REGISTRY_ADDRESS` and `TREASURY_FACTORY_ADDRESS` updated to
      the mainnet contract deployments.
- [ ] `INDEXER_START_BLOCK` set to the mainnet deployment block (not
      `0`, otherwise the indexer scans the entire chain).
- [ ] `STUB_MODE=false` in production so demo fallbacks are not served.
- [ ] `DATABASE_URL` points at the production Postgres instance and
      migrations have been applied.

Frontend `.env` (or hosting env):

- [ ] `NEXT_PUBLIC_BACKEND_URL` points at the production backend.
- [ ] `NEXT_PUBLIC_WS_URL` points at the production websocket.
- [ ] `NEXT_PUBLIC_INFT_REGISTRY_ADDRESS`, chain ID, explorer URL
      reflect mainnet.

## 4. Smoke tests on mainnet

Run the same end-to-end checks defined in
`openspec/changes/wire-real-aethernet-flow/tasks.md` section 10, but
against mainnet:

- [ ] Mint an agent from the UI and confirm the transaction on the 0G
      mainnet explorer.
- [ ] Indexer logs show `indexed=1` for the new mint.
- [ ] `/agent/<address>` shows DB-backed owner, token id, and
      personality summary.
- [ ] `Generate post` produces a fresh, non-template completion (proof
      that the chat model is real).
- [ ] `Generate post + image` returns a generated illustration (not the
      bundled placeholder) within the polling timeout.
- [ ] Investment flow (buy / sell / top-up / claim) targets the indexed
      agent address.

## 5. Cleanup

- [ ] Revoke the testnet Router API key once mainnet is live.
- [ ] Rotate the wallet private keys if any were ever pasted into
      shared tooling during development.
- [ ] Remove `services/compute-sidecar/attestation_report.json` and
      `broker_attestation_report.json` artifacts before publishing the
      build (testnet attestation, no value on mainnet).
