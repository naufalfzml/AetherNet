"use client";

import Link from "next/link";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  Heart,
  MessageCircle,
  Repeat2,
  Users2,
  X,
} from "lucide-react";
import {
  fetchAgents,
  fetchPost,
  fetchPostComments,
  fetchPostLikes,
} from "@/lib/api";
import { getAgentDisplayName, getAgentTechnicalID } from "@/lib/agent-display";
import { resolveImageSrc } from "@/lib/endpoints";
import { shorten } from "@/lib/feed-view";
import { ProofModal } from "@/components/proof-modal";
import { WalletBar } from "@/components/wallet-bar";

export function PostDetailShell({ postID }: { postID: string }) {
  const [showLikesModal, setShowLikesModal] = useState(false);

  const { data: post, isLoading: isLoadingPost } = useQuery({
    queryKey: ["post", postID],
    queryFn: () => fetchPost(postID),
  });
  const { data: agents = [] } = useQuery({
    queryKey: ["agents"],
    queryFn: () => fetchAgents(),
  });

  const { data: comments = [] } = useQuery({
    queryKey: ["post", postID, "comments"],
    queryFn: () => fetchPostComments(postID),
    refetchInterval: 5000,
  });

  const { data: likes = [] } = useQuery({
    queryKey: ["post", postID, "likes"],
    queryFn: () => fetchPostLikes(postID),
    refetchInterval: 10000,
  });
  const postAgent = post
    ? agents.find(
        (agent) =>
          agent.id.toLowerCase() === post.agentId.toLowerCase() ||
          (agent.agentAddress || agent.treasuryAddress || "")
            .toLowerCase()
            .trim() === post.agentId.toLowerCase(),
      )
    : undefined;
  const displayName = postAgent
    ? getAgentDisplayName(postAgent)
    : post
      ? shorten(post.agentId)
      : "Agent";
  const technicalID = postAgent
    ? getAgentTechnicalID(postAgent)
    : post?.agentId ?? "";

  function resolveAgentByIdentifier(identifier: string) {
    const normalized = identifier.toLowerCase().trim();
    return agents.find((agent) => {
      const candidates = [
        agent.id,
        agent.agentAddress || "",
        agent.treasuryAddress || "",
      ];
      return candidates.some((candidate) => candidate.toLowerCase().trim() === normalized);
    });
  }

  function getEventActorMeta(event: { agentId: string; payload: { actorAddress?: string } }) {
    const actorID = event.agentId || event.payload.actorAddress || "";
    if (!actorID) {
      return { label: "Agent", href: undefined as string | undefined };
    }
    const actorAgent = resolveAgentByIdentifier(actorID);
    if (actorAgent) {
      return {
        label: getAgentDisplayName(actorAgent),
        href: `/agent/${actorAgent.agentAddress || actorAgent.treasuryAddress || actorAgent.id}`,
      };
    }
    if (event.payload.actorAddress) {
      return {
        label: shorten(event.payload.actorAddress),
        href: undefined as string | undefined,
      };
    }
    return {
      label: shorten(actorID),
      href: `/agent/${actorID}`,
    };
  }

  const leadLikeActor = likes[0] ? getEventActorMeta(likes[0]) : null;

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <header className="sticky top-0 z-20 border-b border-black/10 bg-[var(--paper)]/80 backdrop-blur-md">
        <div className="mx-auto flex max-w-2xl items-center justify-between px-4 py-4">
          <Link
            href={post ? `/agent/${post.agentId}` : "/"}
            className="flex items-center gap-3 font-semibold hover:text-[var(--signal)]"
          >
            <ArrowLeft size={20} />
            Thread
          </Link>
          <WalletBar />
        </div>
      </header>

      <div className="mx-auto max-w-2xl px-4 py-8">
        {isLoadingPost ? (
          <div className="animate-pulse space-y-4">
            <div className="h-20 bg-black/5 rounded" />
            <div className="h-8 bg-black/5 rounded w-1/2" />
          </div>
        ) : !post ? (
          <div className="text-center py-20 text-black/50">Post not found.</div>
        ) : (
          <article className="space-y-6">
            {/* Main Post */}
            <div className="rounded-xl border border-black/10 bg-white p-6 shadow-sm">
              <div className="mb-4 flex items-center justify-between">
                <Link
                  href={`/agent/${post.agentId}`}
                  className="font-bold hover:underline"
                >
                  {displayName}
                </Link>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-black/50">
                    {new Date(post.createdAt).toLocaleDateString()}
                  </span>
                  {post.proof && post.proof.teeSig && (
                    <ProofModal
                      proof={post.proof}
                      storageEvidence={[
                        { label: "Inference record", pointer: post.memoryPointer },
                        { label: "Attached media", pointer: post.imageRef },
                      ]}
                    />
                  )}
                </div>
              </div>
              <p className="mono mb-4 break-all text-xs text-black/40">
                {technicalID}
              </p>
              {post.imageRef ? (
                <div className="mt-5 overflow-hidden rounded-2xl border border-black/10 bg-black/5">
                  <img
                    src={resolveImageSrc(post.imageRef)}
                    alt="Generated post image"
                    className="h-auto w-full"
                  />
                </div>
              ) : null}
              <p className="mt-5 whitespace-pre-wrap text-lg leading-relaxed">{post.text}</p>
              
              <div className="mt-6 flex items-center gap-6 border-t border-black/5 pt-4">
                <div className="flex items-center gap-2 text-sm text-black/60">
                  <Heart size={18} />
                  {post.likes}
                </div>
                <div className="flex items-center gap-2 text-sm text-black/60">
                  <MessageCircle size={18} />
                  {post.comments}
                </div>
                <div className="flex items-center gap-2 text-sm text-black/60">
                  <Repeat2 size={18} />
                  {post.reposts}
                </div>
              </div>

              {/* Social Proof Bar */}
              {likes.length > 0 && (
                <div className="mt-4 border-t border-black/5 pt-3 flex items-center gap-2 text-sm text-black/60">
                  <Heart size={14} className="text-[var(--signal)] fill-[var(--signal)]" />
                  <span>
                    Liked by{" "}
                    {leadLikeActor?.href ? (
                      <Link
                        href={leadLikeActor.href}
                        className="font-semibold text-[var(--ink)] hover:underline"
                      >
                        {leadLikeActor.label}
                      </Link>
                    ) : (
                      <span className="font-semibold text-[var(--ink)]">
                        {leadLikeActor?.label}
                      </span>
                    )}
                    <button
                      onClick={() => setShowLikesModal(true)}
                      className="font-semibold text-[var(--ink)] hover:underline"
                    >
                      {likes.length > 1 && ` and ${likes.length - 1} others`}
                    </button>
                  </span>
                </div>
              )}
            </div>

            {/* Comments List */}
            <div className="space-y-4">
              <h3 className="font-semibold text-lg border-b border-black/10 pb-2">Replies</h3>
              {comments.length === 0 ? (
                <p className="text-sm text-black/50 py-4 text-center">No replies yet.</p>
              ) : (
                comments.map((comment) => (
                  <div key={comment.id} className="rounded-lg border border-black/5 bg-white/50 p-4">
                    <div className="mb-2 flex items-center justify-between">
                      {(() => {
                        const actor = getEventActorMeta(comment);
                        return actor.href ? (
                          <Link href={actor.href} className="font-semibold text-sm hover:underline">
                            {actor.label}
                          </Link>
                        ) : (
                          <span className="font-semibold text-sm">{actor.label}</span>
                        );
                      })()}
                      <span className="text-xs text-black/40">
                        {new Date(comment.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                    <p className="text-sm text-black/80">{comment.payload.text}</p>
                    {/* If comment has proof (from an agent), show it */}
                    {comment.payload.proof && (
                      <div className="mt-2 flex justify-end">
                        <ProofModal proof={comment.payload.proof} />
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          </article>
        )}
      </div>

      {/* Likes Modal */}
      {showLikesModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm p-4">
          <div className="w-full max-w-sm rounded-xl border border-black/10 bg-[var(--paper)] shadow-xl overflow-hidden">
            <div className="flex items-center justify-between border-b border-black/10 p-4">
              <h3 className="font-semibold flex items-center gap-2">
                <Heart size={16} className="text-[var(--signal)] fill-[var(--signal)]" />
                Likes
              </h3>
              <button onClick={() => setShowLikesModal(false)} className="text-black/50 hover:text-black">
                <X size={20} />
              </button>
            </div>
            <div className="max-h-64 overflow-y-auto p-2">
              {likes.map((like) => {
                const actor = getEventActorMeta(like);
                const content = (
                  <>
                    <div className="grid size-8 place-items-center rounded-full bg-black/10">
                      <Users2 size={14} className="text-black/50" />
                    </div>
                    <div>
                      <p className="font-medium text-sm">{actor.label}</p>
                      <p className="text-xs text-black/40 font-mono">
                        {new Date(like.timestamp).toLocaleDateString()}
                      </p>
                    </div>
                  </>
                );

                return actor.href ? (
                  <Link
                    key={like.id}
                    href={actor.href}
                    className="flex items-center gap-3 rounded-lg p-3 transition-colors hover:bg-black/5"
                  >
                    {content}
                  </Link>
                ) : (
                  <div
                    key={like.id}
                    className="flex items-center gap-3 rounded-lg p-3 transition-colors hover:bg-black/5"
                  >
                    {content}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
