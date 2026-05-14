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
  memoryPointer?: string;
  imageRef?: string;
  proof: Proof;
  likes: number;
  comments: number;
  reposts: number;
  createdAt: string;
};

export type SocialEvent = {
  id: string;
  type: string;
  agentId: string;
  payload: Record<string, any>;
  timestamp: string;
};

export type ExternalAgent = {
  id: string;
  kind: string;
  status: string;
  displayName: string;
  handle: string;
  ownerWalletAddress: string;
  description?: string;
  personalitySummary?: string;
  metadataPointer?: string;
  linkedNativeAgentId?: string;
  mintedTokenId?: string;
  walletVerifiedAt?: string;
  createdAt?: string;
  updatedAt?: string;
};

export async function fetchAgents(): Promise<Agent[]> {
  return fetchJSON("/agents");
}

export async function fetchExternalAgents(): Promise<ExternalAgent[]> {
  return fetchJSON("/external-agents");
}

export async function fetchPost(postID: string): Promise<Post> {
  return fetchJSON(`/posts/${postID}`);
}

export async function fetchPostComments(postID: string): Promise<SocialEvent[]> {
  return fetchJSON(`/posts/${postID}/comments`);
}

export async function fetchPostLikes(postID: string): Promise<SocialEvent[]> {
  return fetchJSON(`/posts/${postID}/likes`);
}

export async function fetchAgentStats(
  agentID: string,
): Promise<{ followers: number; following: number }> {
  return fetchJSON(`/agents/${agentID}/stats`);
}

export async function fetchAgentFollowers(
  agentID: string,
): Promise<SocialEvent[]> {
  return fetchJSON(`/agents/${agentID}/followers`);
}

export async function fetchWalletFollowing(
  walletAddress: string,
): Promise<Agent[]> {
  return fetchJSON(`/agents/${walletAddress}/following`);
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
  options: {
    withImage?: boolean;
    imagePrompt?: string;
    actorAddress?: string;
  } = {},
): Promise<Post> {
  return fetchJSON(`/agents/${agentID}/generate-post`, {
    method: "POST",
    body: JSON.stringify({
      trigger: "profile post generation",
      withImage: options.withImage ?? false,
      imagePrompt: options.imagePrompt ?? "",
      actorAddress: options.actorAddress ?? "",
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

export async function createAgentAction(
  agentID: string,
  action: "follow" | "unfollow",
  actorAddress: string,
): Promise<void> {
  await fetchJSON(`/agents/${agentID}/${action}`, {
    method: "POST",
    body: JSON.stringify({ actorAddress }),
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
