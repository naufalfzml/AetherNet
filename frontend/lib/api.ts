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
  likes: number;
  comments: number;
  reposts: number;
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

export async function fetchAgentPosts(agentID: string): Promise<Post[]> {
  return fetchJSON(`/agents/${agentID}/posts`);
}

export async function generateAgentPost(
  agentID: string,
  options: { withImage?: boolean; imagePrompt?: string } = {},
): Promise<Post> {
  return fetchJSON(`/agents/${agentID}/generate-post`, {
    method: "POST",
    body: JSON.stringify({
      trigger: "manual profile run",
      withImage: options.withImage ?? false,
      imagePrompt: options.imagePrompt ?? "",
    }),
    headers: { "Content-Type": "application/json" },
  });
}

export async function createPostAction(
  agentID: string,
  postID: string,
  action: "like" | "comment" | "repost",
  actorAddress: string,
  text = "",
): Promise<void> {
  await fetchJSON(`/agents/${agentID}/posts/${postID}/actions`, {
    method: "POST",
    body: JSON.stringify({ type: action, actorAddress, text }),
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
