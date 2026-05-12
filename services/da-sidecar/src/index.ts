import http from "node:http";
import "dotenv/config";
import { DisperserClient } from "./disperser.js";
import { RetrieverClient } from "./retriever.js";

const PORT = Number(process.env.PORT ?? 3003);
const DISPERSER_RPC =
  process.env.ZG_DA_DISPERSER_RPC ?? "disperser-testnet.0g.ai:51001";
const RETRIEVER_RPC =
  process.env.ZG_DA_RETRIEVER_RPC ?? "retriever-testnet.0g.ai:34000";

const disperserClient = new DisperserClient(DISPERSER_RPC);
const retrieverClient = new RetrieverClient(RETRIEVER_RPC);

const server = http.createServer(async (req, res) => {
  try {
    if (req.method === "GET" && req.url === "/healthz") {
      return json(res, 200, {
        status: "ok",
        disperserRpc: DISPERSER_RPC,
        retrieverRpc: RETRIEVER_RPC,
      });
    }

    if (req.method === "POST" && req.url === "/publish") {
      const body = await readJSON(req);
      console.log(
        "Received publish request. Body size:",
        JSON.stringify(body).length,
      );

      const payloadStr = JSON.stringify(body);
      const data = Buffer.from(payloadStr, "utf8");
      const quorumId = Number(process.env.ZG_DA_QUORUM_ID ?? 0);

      console.log("Dispersing blob...");
      const disperseRes = await disperserClient.disperseBlob(data, quorumId);
      const requestId = Buffer.from(disperseRes.request_id);
      console.log(`Blob dispersed. Request ID: ${requestId.toString("hex")}`);

      let status = disperseRes.result;
      let batchHeaderHash = "";
      let blobIndex = 0;
      let referenceBlockNumber = 0;

      console.log("Polling for blob confirmation...");
      for (let i = 0; i < 40; i++) {
        // Poll for up to 80 seconds
        await new Promise((r) => setTimeout(r, 2000));
        const statusRes = await disperserClient.getBlobStatus(requestId);
        status = statusRes.status;
        console.log(`Blob status [${i}]: ${status}`);

        if (status === "CONFIRMED" || status === "FINALIZED") {
          const info = statusRes.info.blob_verification_proof;
          batchHeaderHash = Buffer.from(
            info.batch_metadata.batch_header_hash,
          ).toString("hex");
          blobIndex = info.blob_index;
          referenceBlockNumber =
            info.batch_metadata.batch_header.reference_block_number;
          break;
        }
        if (status === "FAILED" || status === "INSUFFICIENT_SIGNATURES") {
          throw new Error(`Blob dispersion failed with status: ${status}`);
        }
      }

      if (!batchHeaderHash) {
        throw new Error("Timeout waiting for blob to be confirmed");
      }

      // Format: 0xBatchHeaderHash:BlobIndex:ReferenceBlockNumber
      const blobId = `0x${batchHeaderHash}:${blobIndex}:${referenceBlockNumber}`;
      console.log(`Blob confirmed! ID: ${blobId}`);

      return json(res, 200, { blobId });
    }

    if (req.method === "GET" && req.url?.startsWith("/retrieve/")) {
      const blobId = decodeURIComponent(req.url.slice("/retrieve/".length));
      console.log("Received retrieve request for:", blobId);

      const parts = blobId.split(":");
      if (parts.length < 3) {
        throw new Error(
          "Invalid blobId format. Expected Hash:Index:BlockNumber",
        );
      }

      const batchHeaderHash = parts[0];
      const blobIndex = parseInt(parts[1], 10);
      const referenceBlockNumber = parseInt(parts[2], 10);
      const quorumId = Number(process.env.ZG_DA_QUORUM_ID ?? 0);

      console.log(
        `Retrieving blob... Hash: ${batchHeaderHash}, Index: ${blobIndex}, Block: ${referenceBlockNumber}`,
      );
      const retrieveRes = await retrieverClient.retrieveBlob(
        batchHeaderHash,
        blobIndex,
        referenceBlockNumber,
        quorumId,
      );

      const retrievedData = Buffer.from(retrieveRes.data).toString("utf8");

      let parsedData;
      try {
        parsedData = JSON.parse(retrievedData);
      } catch (e) {
        parsedData = { raw: retrievedData };
      }

      return json(res, 200, parsedData);
    }

    return json(res, 404, { error: "not found" });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error("DA sidecar error:", message);
    return json(res, 500, { error: message });
  }
});

server.listen(PORT, () => {
  console.log(`DA sidecar listening on :${PORT}`);
  console.log(`Disperser: ${DISPERSER_RPC}`);
  console.log(`Retriever: ${RETRIEVER_RPC}`);
});

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
