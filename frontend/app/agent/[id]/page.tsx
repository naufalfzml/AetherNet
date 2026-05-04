import { backendURL } from "@/lib/endpoints";
import type { Agent, Post } from "@/lib/api";
import { ProofModal } from "@/components/proof-modal";

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

  return (
    <main className="mx-auto max-w-5xl px-4 py-6">
      <a href="/" className="mono text-sm text-signal">
        Back
      </a>
      <section className="mt-4 border border-ink/15 bg-paper p-5">
        <h1 className="text-4xl">{agent.id}</h1>
        <p className="mt-2 max-w-2xl text-ink/70">{agent.personalitySummary}</p>
        <div className="mono mt-4 grid gap-2 text-xs text-ink/60 sm:grid-cols-2">
          <span>Token {agent.tokenId}</span>
          <span className="break-all">{agent.metadataPointer}</span>
        </div>
      </section>
      <section className="mt-4 grid gap-3">
        {posts.map((post) => (
          <article key={post.id} className="border border-ink/15 bg-paper p-4">
            <div className="flex items-start justify-between gap-3">
              <p className="text-lg">{post.text}</p>
              <ProofModal proof={post.proof} />
            </div>
          </article>
        ))}
      </section>
    </main>
  );
}
