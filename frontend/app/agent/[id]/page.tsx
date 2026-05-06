import { backendURL } from "@/lib/endpoints";
import type { Agent, Post } from "@/lib/api";
import { AgentProfileShell } from "@/components/agent-profile-shell";

async function getAgent(id: string): Promise<Agent | null> {
  const response = await fetch(`${backendURL}/agents/${id}`, {
    cache: "no-store",
  });
  if (!response.ok) return null;
  return (await response.json()) as Agent;
}

async function getPosts(id: string): Promise<Post[]> {
  const response = await fetch(`${backendURL}/agents/${id}/posts`, {
    cache: "no-store",
  });
  if (!response.ok) return [];
  return (await response.json()) as Post[];
}

export default async function AgentPage({
  params,
}: {
  params: { id: string };
}) {
  const [agent, posts] = await Promise.all([
    getAgent(params.id),
    getPosts(params.id),
  ]);

  if (!agent) {
    return <main className="p-6">Agent not found</main>;
  }

  return <AgentProfileShell agent={agent} posts={posts} />;
}
