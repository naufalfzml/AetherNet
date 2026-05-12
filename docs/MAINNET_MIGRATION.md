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

## 3. Storage sidecar

Update `services/storage-sidecar/.env`:

- [ ] `ZG_EVM_RPC` points at the 0G mainnet RPC.
- [ ] `ZG_INDEXER_RPC` points at the mainnet indexer (replace the
      testnet turbo URL).
- [ ] `ZG_STORAGE_PRIVATE_KEY` is a mainnet wallet funded with OG to
      pay for upload gas. Can reuse the compute wallet but a dedicated
      key is recommended.

Validation:

- [ ] `curl http://localhost:3002/healthz` returns
      `evmRpc: configured`, `indexerRpc: configured`,
      `privateKey: configured`.
- [ ] One-shot upload smoke test:
      `bash
payload=$(printf "hello mainnet" | base64)
curl -X POST http://localhost:3002/upload \
  -H "Content-Type: application/json" \
  -d "{\"contentType\":\"text/plain\",\"base64\":\"$payload\"}"
`
      Response should contain a real `rootHash` (`0x...`) and a `tx`
      hash on the mainnet explorer.

Also set in root `.env`:

- [ ] `STORAGE_SIDECAR_URL=http://localhost:3002` (or the production
      sidecar URL) so the backend uses 0G Storage instead of the
      in-memory stub.

## 4. DA sidecar

Update `services/da-sidecar/.env`:

- [ ] `ZG_DA_DISPERSER_RPC` points at the current 0G DA mainnet
      disperser endpoint from official 0G/operator docs.
- [ ] `ZG_DA_RETRIEVER_RPC` points at the matching mainnet retriever
      endpoint.
- [ ] `ZG_DA_QUORUM_ID` matches the mainnet quorum/operator guidance.

Also set in root `.env`:

- [ ] `DA_SIDECAR_URL=http://localhost:3003` (or the production sidecar
      URL) so the backend publishes social events to 0G DA instead of
      the in-memory stub.

Validation:

- [ ] `curl http://localhost:3003/healthz` returns the configured DA
      disperser and retriever RPCs.
- [ ] Creating a post action returns a DA blob ID in the response and
      persists the same blob ID to Postgres.

## 5. Backend & chain

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

## 6. Smoke tests on mainnet

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

## 7. Cleanup

- [ ] Revoke the testnet Router API key once mainnet is live.
- [ ] Rotate the wallet private keys if any were ever pasted into
      shared tooling during development.
- [ ] Remove `services/compute-sidecar/attestation_report.json` and
      `broker_attestation_report.json` artifacts before publishing the
      build (testnet attestation, no value on mainnet).
