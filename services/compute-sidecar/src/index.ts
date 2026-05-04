import http from "node:http";
import { deterministicStubProof, type ProofOfInference } from "./proof.js";

type InferRequest = {
  agentId: string;
  personality: string;
  memory?: string;
  trigger: string;
  modelId?: string;
};

type InferResponse = {
  outputText: string;
  chatId: string;
  proof: ProofOfInference;
};

const port = Number(process.env.PORT ?? 3001);
const stubMode = (process.env.STUB_MODE ?? "true") === "true";
const defaultModel = process.env.ZG_COMPUTE_MODEL ?? "llama-3-8b";

const server = http.createServer(async (req, res) => {
  try {
    if (req.method === "GET" && req.url === "/healthz") {
      return json(res, 200, { status: "ok", stubMode });
    }
    if (req.method === "POST" && req.url === "/infer/llm") {
      const body = (await readJSON(req)) as InferRequest;
      const response = stubMode
        ? await runStubLLM(body)
        : await runBrokerLLM(body);
      return json(res, 200, response);
    }
    return json(res, 404, { error: "not found" });
  } catch (error) {
    return json(res, 500, {
      error: error instanceof Error ? error.message : String(error),
    });
  }
});

server.listen(port, () => {
  console.log(`compute sidecar listening on :${port}`);
});

async function runStubLLM(req: InferRequest): Promise<InferResponse> {
  const modelId = req.modelId ?? defaultModel;
  const input = canonicalInput(req);
  const outputText = `AetherNet stub response for ${req.agentId}: ${req.trigger}`;
  return {
    outputText,
    chatId: "stub-chat",
    proof: deterministicStubProof(modelId, input, outputText),
  };
}

async function runBrokerLLM(req: InferRequest): Promise<InferResponse> {
  const modelId = req.modelId ?? defaultModel;
  const input = canonicalInput(req);

  const brokerModule = (await import("@0glabs/0g-serving-broker")) as Record<
    string,
    unknown
  >;
  const Broker = brokerModule.ServingBroker ?? brokerModule.default;
  if (typeof Broker !== "function") {
    throw new Error("0G serving broker export not found");
  }

  const BrokerCtor = Broker as new (
    args: Record<string, string | undefined>,
  ) => {
    chat: (args: {
      model: string;
      messages: Array<{ role: string; content: string }>;
    }) => Promise<{
      text?: string;
      outputText?: string;
      headers?: Record<string, string>;
    }>;
    processResponse?: (
      chatId: string,
    ) => Promise<{ signature?: string; teeSig?: string }>;
  };

  const broker = new BrokerCtor({
    privateKey: process.env.ZG_SERVING_BROKER_PRIVATE_KEY,
    rpcUrl: process.env.OG_RPC_URL,
  });

  const brokerResponse = await broker.chat({
    model: modelId,
    messages: [{ role: "user", content: input }],
  });
  const outputText = brokerResponse.outputText ?? brokerResponse.text ?? "";
  const chatId =
    brokerResponse.headers?.["ZG-Res-Key"] ??
    brokerResponse.headers?.["zg-res-key"] ??
    "";
  const processed =
    broker.processResponse && chatId
      ? await broker.processResponse(chatId)
      : undefined;
  const proof = deterministicStubProof(modelId, input, outputText);

  return {
    outputText,
    chatId,
    proof: {
      ...proof,
      teeSig: processed?.teeSig ?? processed?.signature ?? proof.teeSig,
    },
  };
}

function canonicalInput(req: InferRequest): string {
  return JSON.stringify({
    agentId: req.agentId,
    personality: req.personality,
    memory: req.memory ?? "",
    trigger: req.trigger,
  });
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
