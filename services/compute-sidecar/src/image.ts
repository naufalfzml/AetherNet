import { readFile } from "node:fs/promises";
import path from "node:path";
import { deterministicStubProof, type ProofOfInference } from "./proof.js";

export type ImageRequest = {
  agentId: string;
  personality?: string;
  prompt: string;
  size?: string;
  modelId?: string;
};

export type ImageResponse = {
  imageBase64: string;
  contentType: string;
  jobId: string;
  providerAddress: string;
  teeVerified: boolean;
  proof: ProofOfInference;
};

type ImageMode = "mock" | "edit" | "generate";

const MODE = (process.env.ZG_IMAGE_MODE ?? "mock") as ImageMode;
const ROUTER_BASE_URL = (process.env.ZG_ROUTER_BASE_URL ?? "").replace(
  /\/$/,
  "",
);
const ROUTER_API_KEY = process.env.ZG_ROUTER_API_KEY ?? "";
const IMAGE_MODEL = process.env.ZG_IMAGE_MODEL ?? "qwen/qwen-image-edit-2511";
const IMAGE_SIZE = process.env.ZG_IMAGE_SIZE ?? "1024x1024";
const VERIFY_TEE = (process.env.ZG_IMAGE_VERIFY_TEE ?? "true") === "true";
const DEFAULT_ASSET_PATH = path.resolve(
  process.cwd(),
  "assets/default-avatar.jpg",
);
const MOCK_ASSET_PATH =
  process.env.ZG_IMAGE_MOCK_PATH?.trim() || DEFAULT_ASSET_PATH;
const SEED_IMAGE_PATH =
  process.env.ZG_IMAGE_SEED_PATH?.trim() || MOCK_ASSET_PATH;
const POLL_INTERVAL_MS = Number(process.env.ZG_IMAGE_POLL_INTERVAL_MS ?? 1500);
const POLL_TIMEOUT_MS = Number(process.env.ZG_IMAGE_POLL_TIMEOUT_MS ?? 90_000);

export function imageMode(): ImageMode {
  return MODE;
}

export async function runImageGen(req: ImageRequest): Promise<ImageResponse> {
  switch (MODE) {
    case "mock":
      return runMockImage(req);
    case "edit":
      return runRouterImageEdit(req);
    case "generate":
      return runRouterImageGenerate(req);
    default:
      throw new Error(`unsupported ZG_IMAGE_MODE: ${MODE}`);
  }
}

async function runMockImage(req: ImageRequest): Promise<ImageResponse> {
  const bytes = await readFile(MOCK_ASSET_PATH);
  const base64 = bytes.toString("base64");
  const contentType = guessContentType(MOCK_ASSET_PATH);
  return {
    imageBase64: base64,
    contentType,
    jobId: "mock-job",
    providerAddress: "mock",
    teeVerified: false,
    proof: deterministicStubProof(
      req.modelId ?? "mock-image",
      JSON.stringify(req),
      base64.slice(0, 256),
    ),
  };
}

async function runRouterImageEdit(req: ImageRequest): Promise<ImageResponse> {
  assertRouterEnv();
  const modelId = req.modelId ?? IMAGE_MODEL;
  const seedBytes = await readFile(SEED_IMAGE_PATH);
  const seedContentType = guessContentType(SEED_IMAGE_PATH);

  const submitUrl = `${ROUTER_BASE_URL}/async/images/edits${
    VERIFY_TEE ? "?verify_tee=true" : ""
  }`;
  const form = new FormData();
  form.append("model", modelId);
  form.append("prompt", req.prompt);
  form.append("n", "1");
  form.append("size", req.size ?? IMAGE_SIZE);
  form.append("response_format", "b64_json");
  form.append(
    "image",
    new Blob([new Uint8Array(seedBytes)], { type: seedContentType }),
    path.basename(SEED_IMAGE_PATH),
  );

  const submitRes = await fetch(submitUrl, {
    method: "POST",
    headers: { Authorization: `Bearer ${ROUTER_API_KEY}` },
    body: form,
  });
  if (!submitRes.ok) {
    throw new Error(
      `image submit failed ${submitRes.status}: ${await submitRes.text()}`,
    );
  }
  const submission = (await submitRes.json()) as {
    jobId?: string;
    job_id?: string;
    status?: string;
    provider_address?: string;
    providerAddress?: string;
  };
  const jobId = submission.jobId ?? submission.job_id ?? "";
  const providerAddress =
    submission.provider_address ?? submission.providerAddress ?? "";
  if (!jobId || !providerAddress) {
    throw new Error(
      `image submit missing jobId/provider_address: ${JSON.stringify(submission)}`,
    );
  }

  const polled = await pollImageJob({
    jobId,
    providerAddress,
    modelId,
  });
  return {
    imageBase64: polled.imageBase64,
    contentType: polled.contentType,
    jobId,
    providerAddress,
    teeVerified: polled.teeVerified,
    proof: deterministicStubProof(
      modelId,
      JSON.stringify(req),
      polled.imageBase64.slice(0, 256),
    ),
  };
}

async function pollImageJob(args: {
  jobId: string;
  providerAddress: string;
  modelId: string;
}): Promise<{
  imageBase64: string;
  contentType: string;
  teeVerified: boolean;
}> {
  const params = new URLSearchParams({
    provider_address: args.providerAddress,
    model: args.modelId,
  });
  if (VERIFY_TEE) params.set("verify_tee", "true");
  const url = `${ROUTER_BASE_URL}/async/jobs/${args.jobId}?${params}`;

  const deadline = Date.now() + POLL_TIMEOUT_MS;
  while (Date.now() < deadline) {
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${ROUTER_API_KEY}` },
    });
    if (!res.ok) {
      throw new Error(`image poll failed ${res.status}: ${await res.text()}`);
    }
    const body = (await res.json()) as {
      status?: string;
      result?: {
        data?: Array<{ b64_json?: string; url?: string }>;
      };
      data?: Array<{ b64_json?: string; url?: string }>;
      trace?: { tee_verified?: boolean };
      tee_verified?: boolean;
    };
    const status = body.status ?? "";
    if (status === "completed" || status === "succeeded") {
      const imageData = body.result?.data?.[0] ?? body.data?.[0] ?? {};
      const teeVerified = Boolean(
        body.trace?.tee_verified ?? body.tee_verified,
      );
      if (imageData.b64_json) {
        return {
          imageBase64: imageData.b64_json,
          contentType: "image/png",
          teeVerified,
        };
      }
      if (imageData.url) {
        const fetched = await fetch(imageData.url);
        const buffer = Buffer.from(await fetched.arrayBuffer());
        return {
          imageBase64: buffer.toString("base64"),
          contentType: fetched.headers.get("content-type") ?? "image/png",
          teeVerified,
        };
      }
      throw new Error("image job completed without data");
    }
    if (status === "failed" || status === "error") {
      throw new Error(`image job failed: ${JSON.stringify(body)}`);
    }
    await sleep(POLL_INTERVAL_MS);
  }
  throw new Error(
    `image job ${args.jobId} timed out after ${POLL_TIMEOUT_MS}ms`,
  );
}

async function runRouterImageGenerate(
  req: ImageRequest,
): Promise<ImageResponse> {
  assertRouterEnv();
  const modelId = req.modelId ?? IMAGE_MODEL;
  const url = `${ROUTER_BASE_URL}/images/generations`;
  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ROUTER_API_KEY}`,
    },
    body: JSON.stringify({
      model: modelId,
      prompt: req.prompt,
      n: 1,
      size: req.size ?? IMAGE_SIZE,
      response_format: "b64_json",
    }),
  });
  if (!res.ok) {
    throw new Error(`image generate failed ${res.status}: ${await res.text()}`);
  }
  const body = (await res.json()) as {
    id?: string;
    data?: Array<{ b64_json?: string; url?: string }>;
  };
  const item = body.data?.[0] ?? {};
  let imageBase64 = item.b64_json ?? "";
  let contentType = "image/png";
  if (!imageBase64 && item.url) {
    const fetched = await fetch(item.url);
    const buffer = Buffer.from(await fetched.arrayBuffer());
    imageBase64 = buffer.toString("base64");
    contentType = fetched.headers.get("content-type") ?? "image/png";
  }
  if (!imageBase64) {
    throw new Error("image generate response missing image data");
  }
  return {
    imageBase64,
    contentType,
    jobId: body.id ?? "",
    providerAddress: "router",
    teeVerified: false,
    proof: deterministicStubProof(
      modelId,
      JSON.stringify(req),
      imageBase64.slice(0, 256),
    ),
  };
}

function assertRouterEnv(): void {
  if (!ROUTER_BASE_URL || !ROUTER_API_KEY) {
    throw new Error(
      "ZG_ROUTER_BASE_URL and ZG_ROUTER_API_KEY must be set when ZG_IMAGE_MODE != mock",
    );
  }
}

function guessContentType(filePath: string): string {
  const ext = path.extname(filePath).toLowerCase();
  if (ext === ".png") return "image/png";
  if (ext === ".webp") return "image/webp";
  if (ext === ".gif") return "image/gif";
  return "image/jpeg";
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
