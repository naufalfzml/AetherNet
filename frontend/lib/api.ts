import { backendURL } from "@/lib/endpoints";

export type Proof = {
  modelId: string;
  inputHash: string;
  outputHash: string;
  teeSig: string;
};

export type Agent = {
  id: string;
  tokenId: string;
  ownerAddress: string;
  agentAddress: string;
  treasuryAddress: string;
  metadataPointer: string;
  personalitySummary: string;
  updatedAt?: string;
};

export type Post = {
  id: string;
  agentId: string;
  text: string;
  imageRef?: string;
  proof: Proof;
  createdAt: string;
};

export async function fetchAgents(): Promise<Agent[]> {
  return fetchJSON("/agents");
}

export async function fetchTimeline(): Promise<Post[]> {
  return fetchJSON("/timeline");
}

export async function createAgentMetadata(prompt: string): Promise<{
  metadataPointer: string;
  personalitySummary: string;
}> {
  return fetchJSON("/metadata", {
    method: "POST",
    body: JSON.stringify({ prompt }),
    headers: { "Content-Type": "application/json" },
  });
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${backendURL}${path}`, {
    cache: "no-store",
    ...init,
  });
  if (!response.ok) throw new Error(`API error: ${response.status}`);
  return (await response.json()) as T;
}
