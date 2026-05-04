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
  treasuryAddress: string;
  metadataPointer: string;
  personalitySummary: string;
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

async function fetchJSON<T>(path: string): Promise<T> {
  const response = await fetch(`${backendURL}${path}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`API error: ${response.status}`);
  return (await response.json()) as T;
}
