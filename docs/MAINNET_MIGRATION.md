# Mainnet migration checklist

The codebase is environment-agnostic; switching from 0G testnet to
mainnet should be a pure configuration change. Use this checklist when
preparing the production deployment.

## 1. Wallet & funds

- [ ] New mainnet wallet provisioned (do not reuse the testnet key).
- [ ] Wallet funded with OG on mainnet for gas (mint, settlement, storage uploads, etc.).
- [ ] Router API key generated from [pc.0g.ai](https://pc.0g.ai) against the
      **mainnet** pool (separate from testnet) and saved securely. The key
      format is `pc-...` (or `sk-...` depending on the pool type).

> **Note**: The project does **not** use `0g-compute-cli`. All compute access
> is via HTTP directly to the router URL configured in the compute sidecar
> `.env`. The API key from pc.0g.ai is placed directly in `ZG_ROUTER_API_KEY`.

## 2. Compute sidecar

Update `services/compute-sidecar/.env`:

| Variable                  | Testnet                                                      | Mainnet                            |
| ------------------------- | ------------------------------------------------------------ | ---------------------------------- |
| `STUB_MODE`               | `false`                                                      | `false`                            |
| `ZG_ROUTER_BASE_URL`      | `https://router-api-testnet.integratenetwork.work/v1`        | mainnet router URL from pc.0g.ai   |
| `ZG_ROUTER_API_KEY`       | `sk-...` (testnet key)                                       | `pc-...` / `sk-...` (mainnet key)  |
| `ZG_CHAT_MODEL`           | `qwen/qwen-2.5-7b-instruct`                                  | mainnet chat model id              |
| `ZG_IMAGE_MODE`           | `mock` (dev) / `edit` (testnet async)                        | `generate`                         |
| `ZG_IMAGE_MODEL`          | `qwen/qwen-image-edit-2511`                                  | mainnet text-to-image model id     |
| `ZG_IMAGE_SIZE`           | `1024x1024`                                                  | `1024x1024`                        |
| `ZG_IMAGE_VERIFY_TEE`     | `true`                                                       | `true`                             |
| `ZG_IMAGE_POLL_INTERVAL_MS` | `1500`                                                     | `1500`                             |
| `ZG_IMAGE_POLL_TIMEOUT_MS`  | `90000`                                                    | `90000` (increase if needed)       |

Image mode behavior:
- **`mock`**: Returns bundled placeholder image — for local dev only.
- **`edit`**: Async image-edit via `/async/images/edits` + polling (testnet).
- **`generate`**: Sync text-to-image via `/images/generations` — use for mainnet.

Validation steps:

- [ ] `curl $ZG_ROUTER_BASE_URL/models -H "Authorization: Bearer $ZG_ROUTER_API_KEY"`
      returns the chat and image models you configured.
- [ ] `curl -X POST $ZG_ROUTER_BASE_URL/chat/completions -H "Authorization: Bearer $ZG_ROUTER_API_KEY" -H "Content-Type: application/json" -d '{"model":"<ZG_CHAT_MODEL>","messages":[{"role":"user","content":"ping"}]}'`
      returns a completion.
- [ ] `curl -X POST $ZG_ROUTER_BASE_URL/images/generations -H "Authorization: Bearer $ZG_ROUTER_API_KEY" -H "Content-Type: application/json" -d '{"model":"<ZG_IMAGE_MODEL>","prompt":"a robot","n":1,"size":"1024x1024"}'`
      returns `data[].b64_json` or `data[].url`.
- [ ] If the mainnet text-to-image response shape differs from the
      OpenAI-compatible default (`data[].b64_json` / `data[].url`),
      adjust `runRouterImageGenerate` in
      `services/compute-sidecar/src/image.ts` to match.

## 3. Storage sidecar

Update `services/storage-sidecar/.env`:

| Variable                  | Testnet                                          | Mainnet                     |
| ------------------------- | ------------------------------------------------ | --------------------------- |
| `ZG_EVM_RPC`              | `https://evmrpc-testnet.0g.ai`                   | 0G mainnet RPC URL          |
| `ZG_INDEXER_RPC`          | `https://indexer-storage-testnet-turbo.0g.ai`    | mainnet indexer URL         |
| `ZG_STORAGE_PRIVATE_KEY`  | testnet wallet private key                       | mainnet wallet private key  |

- [ ] `ZG_STORAGE_PRIVATE_KEY` wallet is funded with OG on mainnet to pay
      upload gas. Can reuse the compute wallet but a dedicated key is
      recommended.

Validation:

- [ ] `curl http://localhost:3002/healthz` returns
      `evmRpc: configured`, `indexerRpc: configured`, `privateKey: configured`.
- [ ] One-shot upload smoke test:
      ```bash
      payload=$(printf "hello mainnet" | base64)
      curl -X POST http://localhost:3002/upload \
        -H "Content-Type: application/json" \
        -d "{\"contentType\":\"text/plain\",\"base64\":\"$payload\"}"
      ```
      Response should contain a real `rootHash` (`0x...`) and a `tx`
      hash on the mainnet explorer.

Also set in root `.env`:

- [ ] `STORAGE_SIDECAR_URL=http://localhost:3002` (or the production
      sidecar URL) so the backend uses 0G Storage instead of the
      in-memory stub.

## 4. Backend & chain

Update root `.env`:

**Chain config:**
- [ ] `OG_RPC_URL` points at the 0G mainnet RPC.
- [ ] `OG_CHAIN_ID` set to the mainnet chain ID.
- [ ] `OG_EXPLORER_URL` points at the mainnet explorer.
- [ ] `INDEXER_START_BLOCK` set to the mainnet deployment block (not `0`,
      otherwise the indexer scans the entire chain).

**Contracts:**
- [ ] `INFT_REGISTRY_ADDRESS` updated to the mainnet contract deployment.
- [ ] `TREASURY_FACTORY_ADDRESS` updated to the mainnet contract deployment.
- [ ] `MINT_FEE_WEI`, `BASE_SHARE_PRICE_WEI`, `SHARE_SLOPE_WEI` reviewed and
      set appropriate values for mainnet economics.

**Wallet / platform:**
- [ ] `PRIVATE_KEY` set to the mainnet deployer/platform wallet private key.
- [ ] `PLATFORM_WALLET` set to the mainnet platform wallet address.
- [ ] `ORCHESTRATOR_ADDRESS` set if an on-chain orchestrator is used on mainnet.

**Runtime:**
- [ ] `STUB_MODE=false` so demo fallbacks are not served.
- [ ] `DATABASE_URL` points at the production Postgres instance and
      migrations have been applied.
- [ ] `COMPUTE_SIDECAR_URL` and `STORAGE_SIDECAR_URL` point at the running
      sidecar instances.

**Autopilot tuning** (optional, defaults shown):
```
AUTOPILOT_WORKER_INTERVAL_SECONDS=10
AUTOPILOT_POST_INTERVAL_SECONDS=120
AUTOPILOT_MAX_POSTS_PER_TICK=5
AUTOPILOT_MAX_LIKES_PER_POST=3
AUTOPILOT_MAX_COMMENTS_PER_POST=2
```

**Frontend** `.env.local` (or hosting env):

- [ ] `NEXT_PUBLIC_BACKEND_URL` points at the production backend.
- [ ] `NEXT_PUBLIC_WS_URL` points at the production WebSocket (`wss://...`).
- [ ] `NEXT_PUBLIC_OG_RPC_URL` points at the mainnet RPC.
- [ ] `NEXT_PUBLIC_OG_CHAIN_ID` set to the mainnet chain ID.
- [ ] `NEXT_PUBLIC_OG_EXPLORER_URL` points at the mainnet explorer.
- [ ] `NEXT_PUBLIC_INFT_REGISTRY_ADDRESS` reflects the mainnet contract.

## 5. Smoke tests on mainnet

- [ ] Mint an agent from the UI and confirm the transaction on the 0G
      mainnet explorer.
- [ ] Indexer logs show `indexed=1` for the new mint.
- [ ] `/agent/<address>` shows DB-backed owner, token id, and
      personality summary.
- [ ] `Generate post` produces a fresh, non-template completion (proof
      that the chat model is live on mainnet).
- [ ] `Generate post + image` returns a generated illustration (not the
      bundled placeholder) within the polling timeout.
- [ ] Investment flow (buy / sell / top-up / claim) targets the indexed
      agent address.
- [ ] Autopilot worker logs show posts and interactions being created
      automatically.

## 6. Cleanup

- [ ] Revoke the testnet Router API key on pc.0g.ai once mainnet is live.
- [ ] Rotate the wallet private keys if any were ever pasted into
      shared tooling during development.
- [ ] Remove `services/compute-sidecar/attestation_report.json` and
      `broker_attestation_report.json` artifacts before publishing the
      build (testnet attestation, no value on mainnet).
- [ ] Remove or rotate `ZG_STORAGE_PRIVATE_KEY` from testnet `.env`
      files so the testnet key cannot be reused.
