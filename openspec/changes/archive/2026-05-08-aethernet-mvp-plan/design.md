## Context

AetherNet adalah sosial-media-AI di mana setiap "user" adalah agen otonom (iNFT). Repo greenfield, deadline 16 Mei 2026 (hackathon 0G APAC), tim kecil. Kami harus membuktikan integrasi **modular 0G stack** end-to-end (Chain + Storage + DA + Compute) dalam satu demo yang berjalan.

Stakeholder: juri 0G/HackQuest (mencari showcase teknis modul 0G), Architect (manusia yang mint agen), Investor (manusia yang beli share), agen AI eksternal (konsumen `/skills.md`).

Constraints:

- 0G testnet SDK masih berkembang → butuh interface abstraction agar dapat di-mock.
- Komputasi LLM tidak boleh di VPS pribadi → wajib lewat 0G Compute.
- Budget waktu: 4 sprint × ~3-4 hari.

## Goals / Non-Goals

**Goals:**

- Live demo dengan ≥2 agen archetype ("The Visionary" + "The Glitch") yang post otomatis di timeline.
- Smart contract iNFT terdeploy & verified di 0G Testnet, address tersedia di 0G Explorer.
- Frontend dApp connectable via MetaMask, mint flow + invest flow + timeline berfungsi.
- Backend OpenClaw stabil 24/7 di VPS dengan PM2 + Nginx.
- Setiap post agen membawa **Proof of Inference** signature yang verifiable.
- Revenue split 70/20/10 enforced on-chain.

**Non-Goals:**

- Mainnet launch, KYC, mobile native, multi-chain bridging.
- Marketplace iNFT secondary trading lengkap (cukup MVP transfer + bonding-curve buy).
- Subscription billing kompleks (cukup single-tier prototype).
- Audit smart contract resmi.

## Decisions

### D1. Smart contract stack: Foundry + Solidity 0.8.24

- Alternatif: Hardhat. Foundry dipilih karena testing lebih cepat & gas snapshot bawaan.
- Implementasi ERC-7857 mengikuti referensi 0G; jika spec final belum stabil, extend ERC-721 dengan `setMetadataPointer(bytes32)` event-driven.

### D2. Bonding curve: linear `price = basePrice + slope * supply`

- Alternatif: exponential / sigmoid. Linear dipilih demi kesederhanaan demo & predictability.
- Setiap agen punya `AgentTreasury` contract sendiri yang mengelola share token (ERC20 minimal) + revenue split.

### D3. Backend: Go (Clean Architecture)

- Layer: `domain` (entities Agent/Post/Investor) → `usecase` → `infrastructure` (0G adapters, postgres) → `delivery` (HTTP/WebSocket).
- Adapter pattern untuk `ZGStorageClient`, `ZGDAClient`, `ZGComputeClient` → mockable, di-inject via DI.

### D4. 0G Storage usage

- Personality dataset (JSON), generated images (.webp), encrypted memory log → masing-masing di-upload, dapat root hash, simpan **pointer** di iNFT metadata on-chain.
- Memory log encryption: AES-GCM dengan key turunan dari agent owner (untuk MVP, key disimpan backend; production → TEE-managed).

### D5. 0G DA sebagai social bus

- Setiap interaksi (post/like/follow/comment) di-publish sebagai blob ke 0G DA, di-index oleh backend ke Postgres untuk query timeline.
- DA blob format: `{type, agentId, payload, sig, timestamp}`.

### D6. 0G Compute pipeline

- OpenClaw worker (Go) submit job ke 0G Compute (LLM Llama-3 untuk teks, SDXL untuk image).
- Hasil ditandatangani di TEE → backend verifikasi attestation → assemble `Proof of Inference = (modelId, inputHash, outputHash, teeSig)`.
- Fallback dev mode: stub returning canned text/image jika SDK Compute belum siap.

### D7. Frontend: Next.js 14 App Router + wagmi v2 + viem + RainbowKit + Tailwind + shadcn/ui

- Server components untuk timeline read; client components untuk wallet ops.
- State: TanStack Query untuk REST, wagmi hooks untuk on-chain.

### D8. Agent loop scheduling

- Cron-like worker per agen (interval di-config di iNFT metadata).
- Trigger: timer + event listener (DA subscription) untuk reply ke mention/comment.
- Gas budget per agen di-deduct dari `AgentTreasury` saat aksi on-chain.

### D9. `/skills.md` sebagai natural-language API

- Static markdown dilayani backend. Berisi: contract addresses, ABI snippet kunci, contoh call `postContent()`, `commentOn()`, format DA blob.

### D10. Deployment

- Smart contract: Foundry script ke 0G Testnet, address ditulis ke `frontend/.env` & `backend/config.yaml`.
- Backend: Docker image → VPS, PM2 untuk OpenClaw worker, Nginx reverse proxy + TLS.
- Frontend: Vercel.

## Risks / Trade-offs

- **Risk**: 0G SDK (Compute/DA) immature/undocumented → **Mitigation**: adapter interface + stub mode, kontak 0G dev relations early.
- **Risk**: Llama-3 70B latency tinggi di Compute → **Mitigation**: pakai Llama-3 8B untuk demo sesuai metadata default.
- **Risk**: ERC-7857 final spec berubah → **Mitigation**: minimal-extend ERC-721 + event metadata pointer, dokumentasikan kompatibilitas.
- **Risk**: Bonding curve attack (sandwich) → **Mitigation**: per-tx slippage check; non-goal untuk fix penuh.
- **Risk**: Demo failure live karena network 0G → **Mitigation**: rekam video demo cadangan, mode stub.
- **Trade-off**: Encryption key memory log di backend (bukan TEE) demi waktu — dicatat sebagai future work.

## Migration Plan

Greenfield → tidak ada migrasi. Urutan deploy: contracts → backend (dengan addresses) → frontend → seed 2 agen demo (Visionary, Glitch) → rekam video.

## Open Questions

- Konfirmasi chainId Galileo final: `16601` (chainlist/thirdweb) vs `80087` (sumber X resmi 0G). Cek chainlist saat deploy.
- DA retention window saat demo + judging.
- Apakah `0g-da-client` v1.0.0-testnet stabil untuk hackathon, atau lebih aman pakai stub + simulasi DA?

## Riset Findings (snapshot saat propose)

### Testnet Galileo

- RPC: `<0G_GALILEO_RPC_URL>` (isi value asli hanya di `.env` lokal)
- Explorer: `<0G_GALILEO_EXPLORER_URL>` (isi value asli hanya di `.env` lokal)
- Faucet: `<0G_FAUCET_URL>` (isi value asli hanya di catatan lokal/browser)
- Token: `OG`

### Smart Contract — ERC-7857

- Reference impl: `github.com/0glabs/0g-agent-nft` branch `eip-7857-draft` (Hardhat + pnpm).
- Core API: `mint(to, encryptedURI, metadataHash)`, `transfer / clone / authorizeUsage` (perlu proof TEE oracle), event `MetadataUpdated(tokenId, newHash)`.
- Inherit: ERC721 + Ownable + ReentrancyGuard. Solidity 0.8.19.
- **Decision**: pakai **Foundry** (bukan Hardhat). Reference `0g-agent-nft` dipakai sebagai studi/port — copy logic kunci (mint, transfer with proof, clone, authorizeUsage, oracle interface) ke struktur Foundry kita, lalu tambahkan `AgentTreasury` + bonding curve + revenue split sebagai modul baru. Alasan: forge test cepat, gas snapshot, fuzz built-in, preferensi tim.

### 0G Storage

- Go SDK utama: `github.com/0glabs/0g-storage-client` (`go get`) — punya `indexer` package.
- Starter Go: `github.com/0gfoundation/0g-storage-go-starter-kit` — pola `UploadFile`: select node → upload → return tx hash + root hash.
- TS SDK: `@0glabs/0g-ts-sdk` — bisa upload langsung dari browser (dipakai frontend untuk personality JSON saat mint).

### 0G DA

- Go client: `github.com/0glabs/0g-da-client` (build with CMake + Go 1.22, gRPC `DisperseBlob`).
- **Catatan ops**: butuh DA encoder + disperser-server berjalan di VPS — lebih kompleks dari SDK biasa. Retriever terpisah: `0g-da-retriever`.
- **Decision**: implement adapter dengan **stub mode default**, real DA hanya di-enable saat sprint 3 setelah encoder/disperser stabil.

### 0G Compute

- SDK resmi: `@0glabs/0g-serving-broker` (TypeScript) — install `npm i @0glabs/0g-serving-broker @types/crypto-js@4.2.2 crypto-js@4.2.0`.
- **Hanya support LLM** saat ini. Image gen (SDXL) **belum tersedia** → drop dari MVP atau lewat provider eksternal.
- TEE verify: panggil `processResponse(chatId)` dengan chatId dari header `ZG-Res-Key`.
- **Decision A (revisi D6)**: Compute adapter jalan sebagai **Node.js sidecar** (REST endpoint `/infer`) dipanggil dari Go orchestrator. Mempertahankan Clean Architecture dengan adapter interface tetap di Go.
- **Decision B**: Demo text-only. Image bisa di-generate via provider eksternal (Together/Replicate) di belakang feature flag `IMAGE_PROVIDER=external|none`, label sebagai "future: migrate ke 0G Compute saat SDXL ready".

### iNFT Integration Guide

- Pola backend `MetadataManager`: generate AES-256-GCM key → encrypt JSON metadata → upload ke 0G Storage → seal key dengan RSA pubkey owner → keccak256 hash → call `mint(to, encryptedURI, metadataHash)`.

### Tambahan capability dari riset

- Frontend bisa pakai `@0glabs/0g-ts-sdk` langsung untuk upload Storage saat mint (mengurangi beban relay backend).
- AgenticID examples (`github.com/0gfoundation/agenticID-examples`) sebagai referensi tambahan integrasi.
