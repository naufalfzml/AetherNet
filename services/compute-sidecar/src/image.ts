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

function getDefaultAssetPath() {
  return path.resolve(process.cwd(), "assets/default-avatar.jpg");
}

function getImageConfig() {
  const mode = (process.env.ZG_IMAGE_MODE ?? "mock") as ImageMode;
  const routerBaseURL = (process.env.ZG_ROUTER_BASE_URL ?? "").replace(
    /\/$/,
    "",
  );
  const routerAPIKey = process.env.ZG_ROUTER_API_KEY ?? "";
  const imageModel =
    process.env.ZG_IMAGE_MODEL ?? "qwen/qwen-image-edit-2511";
  const imageSize = process.env.ZG_IMAGE_SIZE ?? "1024x1024";
  const verifyTEE = (process.env.ZG_IMAGE_VERIFY_TEE ?? "true") === "true";
  const defaultAssetPath = getDefaultAssetPath();
  const mockAssetPath =
    process.env.ZG_IMAGE_MOCK_PATH?.trim() || defaultAssetPath;
  const seedImagePath =
    process.env.ZG_IMAGE_SEED_PATH?.trim() || mockAssetPath;
  const pollIntervalMS = Number(process.env.ZG_IMAGE_POLL_INTERVAL_MS ?? 1500);
  const pollTimeoutMS = Number(process.env.ZG_IMAGE_POLL_TIMEOUT_MS ?? 90_000);

  return {
    mode,
    routerBaseURL,
    routerAPIKey,
    imageModel,
    imageSize,
    verifyTEE,
    mockAssetPath,
    seedImagePath,
    pollIntervalMS,
    pollTimeoutMS,
  };
}

export function imageMode(): ImageMode {
  return getImageConfig().mode;
}

export async function runImageGen(req: ImageRequest): Promise<ImageResponse> {
  const mode = imageMode();
  switch (mode) {
    case "mock":
      return runMockImage(req);
    case "edit":
      return runRouterImageEdit(req);
    case "generate":
      return runRouterImageGenerate(req);
    default:
      throw new Error(`unsupported ZG_IMAGE_MODE: ${mode}`);
  }
}

async function runMockImage(req: ImageRequest): Promise<ImageResponse> {
  const { mockAssetPath } = getImageConfig();
  const bytes = await readFile(mockAssetPath);
  const base64 = bytes.toString("base64");
  const contentType = guessContentType(mockAssetPath);
  console.log(
    `compute /infer/image ok mode=mock agent=${req.agentId} bytes=${bytes.length}`,
  );
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
  const config = getImageConfig();
  assertRouterEnv(config);
  const modelId = req.modelId ?? config.imageModel;
  const seedBytes = await readFile(config.seedImagePath);
  const seedContentType = guessContentType(config.seedImagePath);

  const submitUrl = `${config.routerBaseURL}/async/images/edits${
    config.verifyTEE ? "?verify_tee=true" : ""
  }`;
  const form = new FormData();
  form.append("model", modelId);
  form.append("prompt", req.prompt);
  form.append("n", "1");
  form.append("size", req.size ?? config.imageSize);
  form.append("response_format", "b64_json");
  form.append(
    "image",
    new Blob([new Uint8Array(seedBytes)], { type: seedContentType }),
    path.basename(config.seedImagePath),
  );

  const submitRes = await fetch(submitUrl, {
    method: "POST",
    headers: { Authorization: `Bearer ${config.routerAPIKey}` },
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
    config,
  });
  console.log(
    `compute /infer/image ok mode=edit agent=${req.agentId} model=${modelId} jobId=${jobId} provider=${providerAddress} teeVerified=${polled.teeVerified}`,
  );
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
  config: ReturnType<typeof getImageConfig>;
}): Promise<{
  imageBase64: string;
  contentType: string;
  teeVerified: boolean;
}> {
  const { config } = args;
  const params = new URLSearchParams({
    provider_address: args.providerAddress,
    model: args.modelId,
  });
  if (config.verifyTEE) params.set("verify_tee", "true");
  const url = `${config.routerBaseURL}/async/jobs/${args.jobId}?${params}`;

  const deadline = Date.now() + config.pollTimeoutMS;
  while (Date.now() < deadline) {
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${config.routerAPIKey}` },
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
    await sleep(config.pollIntervalMS);
  }
  throw new Error(
    `image job ${args.jobId} timed out after ${config.pollTimeoutMS}ms`,
  );
}

async function runRouterImageGenerate(
  req: ImageRequest,
): Promise<ImageResponse> {
  const config = getImageConfig();
  assertRouterEnv(config);
  const modelId = req.modelId ?? config.imageModel;
  const url = `${config.routerBaseURL}/images/generations`;
  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${config.routerAPIKey}`,
    },
    body: JSON.stringify({
      model: modelId,
      prompt: req.prompt,
      n: 1,
      size: req.size ?? config.imageSize,
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
  console.log(
    `compute /infer/image ok mode=generate agent=${req.agentId} model=${modelId} bytes=${imageBase64.length}`,
  );
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

function assertRouterEnv(config: ReturnType<typeof getImageConfig>): void {
  if (!config.routerBaseURL || !config.routerAPIKey) {
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
