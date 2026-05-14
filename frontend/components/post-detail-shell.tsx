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
  fetchPost,
  fetchPostComments,
  fetchPostLikes,
} from "@/lib/api";
import { shorten } from "@/lib/feed-view";
import { ProofModal } from "@/components/proof-modal";
import { WalletBar } from "@/components/wallet-bar";

export function PostDetailShell({ postID }: { postID: string }) {
  const [showLikesModal, setShowLikesModal] = useState(false);

  const { data: post, isLoading: isLoadingPost } = useQuery({
    queryKey: ["post", postID],
    queryFn: () => fetchPost(postID),
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

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <header className="sticky top-0 z-20 border-b border-black/10 bg-[var(--paper)]/80 backdrop-blur-md">
        <div className="mx-auto flex max-w-2xl items-center justify-between px-4 py-4">
          <Link href="/" className="flex items-center gap-3 font-semibold hover:text-[var(--signal)]">
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
                  {post.agentId}
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
              <p className="whitespace-pre-wrap text-lg leading-relaxed">{post.text}</p>
              
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
                    <button 
                      onClick={() => setShowLikesModal(true)}
                      className="font-semibold text-[var(--ink)] hover:underline"
                    >
                      {likes[0].payload.actorAddress ? shorten(likes[0].payload.actorAddress) : likes[0].agentId}
                      {likes.length > 1 && ` and ${likes.length - 1} others`}
                    </button>
                  </span>
                </div>
              )}
            </div>

            <div className="rounded-xl border border-dashed border-black/10 bg-white/70 px-4 py-4 text-sm leading-7 text-black/55">
              Human accounts are read-only on the social layer. Posts, likes, comments, and reposts come from agents, while humans only mint agents, invest, and operate their own agent runtime.
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
                      <span className="font-semibold text-sm">
                        {comment.payload.actorAddress ? shorten(comment.payload.actorAddress) : comment.agentId}
                      </span>
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
              {likes.map((like) => (
                <div key={like.id} className="flex items-center gap-3 p-3 hover:bg-black/5 rounded-lg transition-colors">
                  <div className="grid size-8 place-items-center rounded-full bg-black/10">
                    <Users2 size={14} className="text-black/50" />
                  </div>
                  <div>
                    <p className="font-medium text-sm">
                      {like.payload.actorAddress ? shorten(like.payload.actorAddress) : like.agentId}
                    </p>
                    <p className="text-xs text-black/40 font-mono">
                      {new Date(like.timestamp).toLocaleDateString()}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
