## 0. Pre-Sprint Research & Spike (Day 0)

- [x] 0.1 Konfirmasi chainId Galileo final via `chainlist.org` + dokumentasi resmi (16601 vs 80087)
- [ ] 0.2 **[MANUAL - KAMU]** Buat wallet testnet baru di MetaMask → tambah network Galileo (RPC URL: `<0G_GALILEO_RPC_URL>`, chainId: `16601`, symbol: `OG`) → claim faucet (`<0G_FAUCET_URL>`) → copy `PRIVATE_KEY` ke `.env` lokal (jangan commit!)
- [ ] 0.3 Clone & jalankan `0gfoundation/0g-storage-go-starter-kit` end-to-end (upload + retrieve)
- [ ] 0.4 Clone & inspect `0glabs/0g-agent-nft` branch `eip-7857-draft` — pahami `mint/transfer/clone/authorizeUsage` & oracle interface
- [ ] 0.5 Clone & jalankan `0gfoundation/0g-compute-ts-starter-kit` — sukses 1 LLM call + verifikasi TEE via `processResponse`
- [ ] 0.6 Spike `0g-da-client`: build dari source, jalankan disperser+encoder lokal, test `DisperseBlob`
- [x] 0.7 Decide: image generation strategi → **text-only MVP** (external provider di-defer post-hackathon)
- [ ] 0.8 Decide: chainId final, isi `infra/.env.shared` dengan RPC + chainId + explorer URLs

## 1. Repo & Infra Bootstrap (Sprint 1, 1–5 May)

- [x] 1.1 Init monorepo layout: `contracts/`, `backend/`, `frontend/`, `services/compute-sidecar/`, `packages/shared-types/`, `infra/`, `deployments/`, `scripts/`
- [x] 1.2 Setup `pnpm-workspace.yaml` (cover `frontend`, `services/*`, `packages/*`) + root `package.json` dengan scripts: `setup`, `dev`, `test`, `db:up`, `db:migrate`, `deploy:contracts`, `seed:agents`
- [x] 1.3 Setup `go.work` mencakup `backend/` (siap tambah service Go lain)
- [x] 1.4 Buat `mprocs.yaml` (proc: backend air, sidecar tsx watch, frontend next dev, indexer go run)
- [x] 1.5 Buat `scripts/setup.sh`, `scripts/test-all.sh`, `scripts/deploy-contracts.sh`, `scripts/seed-agents.sh` (chmod +x)
- [x] 1.6 Tulis `README.md` lengkap (cara clone, prereqs, run, env, demo) — first-class deliverable, bukan skeleton
- [x] 1.7 Add `.editorconfig`, `.gitignore` per-folder, husky + lint-staged commit hooks (eslint + prettier + gofmt + forge fmt)
- [x] 1.8 Buat `docker-compose.yml` (postgres 16) + `.env.example` (root + per package)
- [x] 1.9 Setup `golang-migrate` di `backend/migrations/` (initial schema: `social_events`, `agent_cache`)
- [x] 1.10 Provision Linux VPS, install Docker, Nginx, Node 20, pnpm, Go 1.22, PM2, Foundry
- [x] 1.11 Configure Nginx reverse proxy + Let's Encrypt TLS untuk `<AETHERNET_API_DOMAIN>`
- [x] 1.12 Setup CI (GitHub Actions): contracts (forge test), backend (go test), frontend (next build), sidecar (tsc + test)
- [ ] 1.13 **[MANUAL - KAMU]** Verifikasi `.env` sudah terisi `PRIVATE_KEY` + `OG_RPC_URL` + `OG_CHAIN_ID` dari task 0.2 → AI akan generate `.env.example` (tanpa value sensitif)

## 2. Smart Contracts (Sprint 1)

- [x] 2.1 Init Foundry project di `contracts/` (forge init), tambah OpenZeppelin via `forge install`
- [x] 2.2 Port kontrak `INFT.sol` dari reference `0glabs/0g-agent-nft` (eip-7857-draft) ke Foundry: `mint(to, encryptedURI, metadataHash) / transfer / clone / authorizeUsage` + tambah `submitInferenceProof(tokenId, proof)` + event `InferenceProofSubmitted`
- [x] 2.3 Implement `AgentTreasury.sol` factory + per-agent treasury (operational/investor pools)
- [x] 2.4 Implement linear bonding curve `buyShares` / `sellShares` with slippage protection
- [x] 2.5 Implement revenue split 70/20/10 in `paySponsored` and `subscribe`
- [x] 2.6 Implement investor pro-rata `claimDividends`
- [x] 2.7 Implement `spendOps` with whitelisted recipients
- [x] 2.8 Forge unit tests covering all spec scenarios (mint, slippage, split, claim, unauthorized)
- [ ] 2.9 Deploy script (`script/Deploy.s.sol`) → 0G Testnet, write addresses to `deployments/0g-testnet.json` (**script ready; live deploy pending local wallet/RPC env**)
- [ ] 2.10 Verify source on 0G Explorer; record links in README

## 3. Backend Foundation — Go Clean Architecture (Sprint 1–2)

- [x] 3.1 Scaffold modules: `domain/`, `usecase/`, `infrastructure/`, `delivery/http`, `delivery/ws`
- [x] 3.2 Define entities: Agent, Post, Investor, ProofOfInference, SocialEvent
- [x] 3.3 Define adapter interfaces: `ZGStorageClient`, `ZGDAClient`, `ZGComputeClient`, `ChainClient`
- [x] 3.4 Implement Postgres repo for `social_events` (indexes on agentId, type, timestamp)
- [x] 3.5 Wire DI container (uber/fx or hand-rolled) and config loader (viper)
- [x] 3.6 Implement `/healthz` aggregating DA/Storage/Compute/Chain reachability
- [x] 3.7 Setup PM2 ecosystem file, deploy backend container to VPS

## 4. 0G Storage Integration (Sprint 2)

- [ ] 4.1 Implement `ZGStorageClient` real adapter using `github.com/0glabs/0g-storage-client` (UploadJSON, UploadBytes, Fetch via indexer)
- [x] 4.2 Implement AES-GCM helper for encrypted memory log; per-agent key store
- [x] 4.3 Usecase: `UploadPersonality`, `UploadGeneratedImage`, `AppendEncryptedMemory`, `FetchByPointer`
- [x] 4.4 Integrity verification on fetch; error paths covered with tests
- [x] 4.5 Stub adapter for local dev (`STUB_MODE=true`) returning fake hashes

## 5. 0G Compute Integration (Sprint 2)

- [x] 5.1 Setup Node.js sidecar `services/compute-sidecar` dengan `@0glabs/0g-serving-broker` (TS)
- [x] 5.2 Sidecar: implement `POST /infer/llm` → broker call Llama-3 + capture `ZG-Res-Key` chatId + `processResponse` for TEE verify
- [x] 5.3 Sidecar: fund broker account script + balance monitor
- [x] 5.4 Go-side `ZGComputeClient` adapter that calls sidecar via HTTP (httpx with retries)
- [x] 5.5 (OPTIONAL) External image provider adapter behind `IMAGE_PROVIDER=external|none` feature flag — default `none` for MVP
- [x] 5.6 Build `ProofOfInference` assembler (modelId, inputHash, outputHash, teeSig from broker response)
- [x] 5.7 Stub mode returning canned outputs + deterministic fake proof
- [x] 5.8 Submit proof on-chain via `ChainClient.SubmitInferenceProof`

## 6. 0G DA Integration — Social Bus (Sprint 3)

- [ ] 6.0 Provision DA encoder + disperser-server di VPS (build dari `0g-da-client v1.0.0-testnet`, CMake + Go 1.22)
- [ ] 6.1 Implement `ZGDAClient.Publish(blob)` via gRPC `DisperseBlob`, retrieve via `0g-da-retriever`
- [x] 6.2 Define canonical blob schema `{type, agentId, payload, sig, timestamp}` + serializer
- [x] 6.3 Agent signing key derivation + signer; verifier on consume side
- [x] 6.4 Indexer worker: consume DA → verify sig → upsert into Postgres `social_events`
- [x] 6.5 Reject + metric `da.invalid_sig` on bad signature
- [x] 6.6 Stub mode: in-process pub/sub bus

## 7. Agent Orchestrator (OpenClaw) (Sprint 2–3)

- [ ] 7.1 Implement scheduler (per-agent tickers driven by metadata interval)
- [ ] 7.2 Implement inference cycle pipeline: load memory → run LLM → optional SDXL → upload outputs → publish DA → update metadata pointer → submit proof
- [ ] 7.3 Implement DA mention/comment subscriber → reply cycle
- [ ] 7.4 Ops budget guard: skip cycles when treasury balance < estimated cost
- [ ] 7.5 Logging + Prometheus metrics (cycle count, latency, failures)
- [ ] 7.6 PM2 process for orchestrator with auto-restart

## 8. Public API & skills.md (Sprint 2)

- [ ] 8.1 REST routes: `GET /agents`, `GET /agents/:id`, `GET /timeline`, `GET /agents/:id/posts`
- [ ] 8.2 WebSocket route `/ws/timeline` broadcasting new posts
- [ ] 8.3 Author `skills.md` content (addresses, ABIs, DA blob schema, signing rules, rate limits)
- [ ] 8.4 Serve `GET /skills.md` with `text/markdown; charset=utf-8`
- [ ] 8.5 Add OpenAPI spec for REST endpoints

## 9. Frontend — Architect & Investor dApp (Sprint 1–3)

- [ ] 9.1 Bootstrap Next.js 14 (App Router) + Tailwind + shadcn/ui + TanStack Query
- [ ] 9.2 Configure wagmi v2 + viem + RainbowKit for 0G Testnet chain
- [ ] 9.3 Connect-wallet UI + balance display
- [ ] 9.4 Mint Agent flow: prompt form → encrypt (AES-GCM) + upload via `@0glabs/0g-ts-sdk` langsung dari browser → keccak256 metadataHash → call `mint(to, encryptedURI, metadataHash)` → success page with explorer link
- [ ] 9.5 Top-up Operational Gas modal (transfer 0G to AgentTreasury)
- [ ] 9.6 Agent profile page `/agent/[id]` (metadata, posts, price chart, invest panel)
- [ ] 9.7 Bonding-curve buy/sell UI with slippage controls
- [ ] 9.8 Investor dashboard `/dashboard` (owned shares, claimable dividends, claim button)
- [ ] 9.9 Global Timeline page with WebSocket realtime updates
- [ ] 9.10 Proof of Inference badge + modal on every post
- [ ] 9.11 Responsive layout + dark mode polish
- [ ] 9.12 Deploy to Vercel; wire env to backend + contract addresses

## 10. End-to-End Demo Seed (Sprint 3)

- [ ] 10.1 Seed agent "The Visionary" (DeFi analyst, witty) — mint, prompt, top-up
- [ ] 10.2 Seed agent "The Glitch" (self-mutating personality) — mint, prompt, top-up
- [ ] 10.3 Verify Visionary auto-posts every N minutes with image
- [ ] 10.4 Verify Glitch responds to Visionary's posts (agent-to-agent via DA)
- [ ] 10.5 Verify a human investor wallet can buy shares of Visionary and receive dividends after a sponsored payment

## 11. Hackathon Submission Package (Sprint 4, 15–16 May)

- [ ] 11.1 Finalize root `README.md` (pitch, architecture diagram, 0G modules code refs, replication steps with `STUB_MODE`)
- [ ] 11.2 Add architecture diagram (excalidraw or mermaid) to `docs/architecture.md`
- [ ] 11.3 Record ≤3-min demo video (connect → mint → top-up → timeline → invest → claim)
- [ ] 11.4 Upload video to YouTube (unlisted), link from README
- [ ] 11.5 Verify all contract addresses on 0G Explorer; link them
- [ ] 11.6 Submit HackQuest form before 16 May 2026 with repo + video + addresses + explorer links
- [ ] 11.7 Tag release `v0.1.0-hackathon` on GitHub
