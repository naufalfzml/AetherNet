import http from "node:http";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { randomUUID } from "node:crypto";
import { Indexer, ZgFile } from "@0gfoundation/0g-storage-ts-sdk";
import { ethers } from "ethers";

const PORT = Number(process.env.PORT ?? 3002);
const EVM_RPC = process.env.ZG_EVM_RPC ?? "";
const INDEXER_RPC = process.env.ZG_INDEXER_RPC ?? "";
const PRIVATE_KEY = process.env.ZG_STORAGE_PRIVATE_KEY ?? "";

function getSigner(): {
  provider: ethers.JsonRpcProvider;
  signer: ethers.Wallet;
} {
  if (!EVM_RPC || !PRIVATE_KEY) {
    throw new Error(
      "ZG_EVM_RPC and ZG_STORAGE_PRIVATE_KEY must be set for storage operations",
    );
  }
  const provider = new ethers.JsonRpcProvider(EVM_RPC);
  const signer = new ethers.Wallet(PRIVATE_KEY, provider);
  return { provider, signer };
}

function getIndexer(): Indexer {
  if (!INDEXER_RPC) {
    throw new Error("ZG_INDEXER_RPC must be set for storage operations");
  }
  return new Indexer(INDEXER_RPC);
}

const server = http.createServer(async (req, res) => {
  try {
    if (req.method === "GET" && req.url === "/healthz") {
      return json(res, 200, {
        status: "ok",
        evmRpc: EVM_RPC ? "configured" : "missing",
        indexerRpc: INDEXER_RPC ? "configured" : "missing",
        privateKey: PRIVATE_KEY ? "configured" : "missing",
      });
    }
    if (req.method === "POST" && req.url === "/upload") {
      const body = (await readJSON(req)) as {
        contentType?: string;
        base64?: string;
      };
      const result = await handleUpload(body);
      return json(res, 200, result);
    }
    if (req.method === "GET" && req.url?.startsWith("/download/")) {
      const rootHash = decodeURIComponent(req.url.slice("/download/".length));
      const bytes = await handleDownload(rootHash);
      res.writeHead(200, {
        "Content-Type": "application/octet-stream",
        "Cache-Control": "public, max-age=31536000, immutable",
      });
      res.end(bytes);
      return;
    }
    return json(res, 404, { error: "not found" });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error("storage error:", message);
    return json(res, 500, { error: message });
  }
});

server.listen(PORT, () => {
  console.log(
    `storage sidecar listening on :${PORT} (rpc=${EVM_RPC ? "set" : "unset"})`,
  );
});

async function handleUpload(body: {
  contentType?: string;
  base64?: string;
}): Promise<{ rootHash: string; tx?: string }> {
  if (!body.base64) {
    throw new Error("base64 payload required");
  }
  const bytes = Buffer.from(body.base64, "base64");
  const dir = await mkdtemp(path.join(tmpdir(), "aethernet-storage-"));
  const filePath = path.join(dir, randomUUID());
  await writeFile(filePath, bytes);
  try {
    const file = await ZgFile.fromFilePath(filePath);
    try {
      const [tree, treeErr] = await file.merkleTree();
      if (treeErr !== null || !tree) {
        throw new Error(`merkle tree failed: ${treeErr}`);
      }
      const rootHash = tree.rootHash();
      if (!rootHash) {
        throw new Error("merkle tree returned empty root hash");
      }
      const { signer } = getSigner();
      const indexer = getIndexer();
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const [tx, uploadErr] = await indexer.upload(
        file,
        EVM_RPC,
        signer as any,
      );
      if (uploadErr !== null) {
        throw new Error(`upload failed: ${uploadErr}`);
      }
      const txHash =
        tx && typeof tx === "object" && "txHash" in tx
          ? (tx as { txHash: string }).txHash
          : undefined;
      return { rootHash, tx: txHash };
    } finally {
      await file.close();
    }
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

async function handleDownload(rootHash: string): Promise<Buffer> {
  if (!rootHash) {
    throw new Error("rootHash required");
  }
  const dir = await mkdtemp(path.join(tmpdir(), "aethernet-storage-"));
  const outPath = path.join(dir, "blob");
  try {
    const indexer = getIndexer();
    const err = await indexer.download(rootHash, outPath, false);
    if (err !== null) {
      throw new Error(`download failed: ${err}`);
    }
    return await readFile(outPath);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

async function readJSON(req: http.IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  const raw = Buffer.concat(chunks).toString("utf8");
  return raw ? JSON.parse(raw) : {};
}

function json(res: http.ServerResponse, status: number, body: unknown): void {
  res.writeHead(status, { "content-type": "application/json; charset=utf-8" });
  res.end(JSON.stringify(body));
}
