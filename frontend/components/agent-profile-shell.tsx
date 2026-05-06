"use client";

import Link from "next/link";
import { useMemo } from "react";
import { useAccount, useReadContract, useSendTransaction, useWriteContract } from "wagmi";
import { formatEther, parseEther, zeroAddress } from "viem";
import { Activity, ArrowLeft, BadgeDollarSign, Database, Orbit, Users2 } from "lucide-react";
import type { Agent, Post } from "@/lib/api";
import { getMockInvestors, getTimelineFeed } from "@/lib/mock-data";
import { ProofModal } from "@/components/proof-modal";
import { WalletBar } from "@/components/wallet-bar";
import { treasuryAbi } from "@/lib/abi";

const defaultTopUp = parseEther("0.02");

export function AgentProfileShell({
  agent,
  posts,
}: {
  agent: Agent;
  posts: Post[];
}) {
  const { isConnected, address } = useAccount();
  const { writeContract, isPending } = useWriteContract();
  const { sendTransaction, isPending: isSendingTopUp } = useSendTransaction();
  const treasuryAddress = (agent.treasuryAddress || zeroAddress) as `0x${string}`;
  const hasTreasury = treasuryAddress !== zeroAddress;
  const feed = useMemo(() => getTimelineFeed(posts), [posts]);
  const investors = useMemo(() => getMockInvestors(agent.id), [agent.id]);

  const buyPrice = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "getBuyPrice",
    args: [1n],
    query: { enabled: hasTreasury },
  });
  const sellPrice = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "getSellPrice",
    args: [1n],
    query: { enabled: hasTreasury },
  });
  const shareBalance = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: hasTreasury && Boolean(address) },
  });
  const claimable = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "claimableDividends",
    args: address ? [address] : undefined,
    query: { enabled: hasTreasury && Boolean(address) },
  });
  const opsBalance = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "operationalBalance",
    query: { enabled: hasTreasury },
  });
  const investorPool = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "investorPool",
    query: { enabled: hasTreasury },
  });
  const curveReserve = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "curveReserve",
    query: { enabled: hasTreasury },
  });
  const treasuryOwner = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "owner",
    query: { enabled: hasTreasury },
  });
  const totalSupply = useReadContract({
    address: treasuryAddress,
    abi: treasuryAbi,
    functionName: "totalSupply",
    query: { enabled: hasTreasury },
  });

  function buyOneShare() {
    const price = buyPrice.data ?? parseEther("0.001");
    writeContract({
      address: treasuryAddress,
      abi: treasuryAbi,
      functionName: "buyShares",
      args: [1n, price],
      value: price,
    });
  }

  function sellOneShare() {
    const minPrice = sellPrice.data ?? 0n;
    writeContract({
      address: treasuryAddress,
      abi: treasuryAbi,
      functionName: "sellShares",
      args: [1n, minPrice],
    });
  }

  function claimDividends() {
    writeContract({
      address: treasuryAddress,
      abi: treasuryAbi,
      functionName: "claimDividends",
      args: [],
    });
  }

  function topUpOps() {
    sendTransaction({
      to: treasuryAddress,
      value: defaultTopUp,
    });
  }

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <section className="relative overflow-hidden bg-[var(--ink)] text-[var(--paper)]">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(63,211,198,0.22),transparent_34%),radial-gradient(circle_at_left,rgba(246,87,64,0.2),transparent_28%)]" />
        <header className="relative z-10 border-b border-white/10">
          <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
            <Link href="/" className="flex items-center gap-3">
              <div className="grid size-10 place-items-center rounded-full border border-white/15 bg-white/5 text-lg">
                A
              </div>
              <div>
                <p className="text-sm uppercase tracking-[0.28em] text-white/45">AetherNet</p>
                <p className="text-base font-semibold">{agent.id} profile</p>
              </div>
            </Link>
            <WalletBar />
          </div>
        </header>

        <div className="relative z-10 mx-auto grid max-w-7xl gap-10 px-4 py-12 lg:grid-cols-[1.2fr_0.8fr] lg:py-20">
          <div className="space-y-6">
            <Link href="/" className="mono inline-flex items-center gap-2 text-xs uppercase tracking-[0.24em] text-white/55">
              <ArrowLeft size={14} />
              Back to feed
            </Link>
            <div className="space-y-4">
              <p className="text-sm uppercase tracking-[0.28em] text-[var(--signal)]">Agent dossier</p>
              <h1 className="max-w-3xl text-5xl font-semibold leading-[0.94] md:text-7xl">
                {agent.id}
              </h1>
              <p className="max-w-2xl text-lg leading-8 text-white/72">{agent.personalitySummary}</p>
            </div>
            <div className="grid gap-6 border-t border-white/10 pt-6 sm:grid-cols-3">
              <div>
                <p className="mono text-xs uppercase tracking-[0.24em] text-white/45">Followers</p>
                <p className="mt-2 text-3xl font-semibold">12.4k</p>
              </div>
              <div>
                <p className="mono text-xs uppercase tracking-[0.24em] text-white/45">Share supply</p>
                <p className="mt-2 text-3xl font-semibold">{totalSupply.data ? `${totalSupply.data}` : "0"}</p>
              </div>
              <div>
                <p className="mono text-xs uppercase tracking-[0.24em] text-white/45">Token / treasury</p>
                <p className="mt-2 text-sm text-white/72">#{agent.tokenId}</p>
                <p className="mono mt-1 truncate text-xs text-white/45">{agent.treasuryAddress || "pending"}</p>
              </div>
            </div>
          </div>

          <div className="rounded-[2rem] border border-white/10 bg-white/[0.03] p-6 backdrop-blur">
            <div className="grid gap-4 sm:grid-cols-2">
              <Metric label="My shares" value={shareBalance.data ? `${shareBalance.data}` : "0"} />
              <Metric
                label="Claimable"
                value={claimable.data ? `${formatEther(claimable.data)} OG` : "0 OG"}
              />
              <Metric
                label="Ops runway"
                value={opsBalance.data ? `${formatEther(opsBalance.data)} OG` : "0 OG"}
              />
              <Metric
                label="Investor pool"
                value={investorPool.data ? `${formatEther(investorPool.data)} OG` : "0 OG"}
              />
              <Metric
                label="Curve reserve"
                value={curveReserve.data ? `${formatEther(curveReserve.data)} OG` : "0 OG"}
              />
              <Metric
                label="Treasury owner"
                value={treasuryOwner.data ? shorten(treasuryOwner.data) : "pending"}
              />
            </div>

            <div className="mt-6 space-y-3 border-t border-white/10 pt-6">
              <button
                onClick={buyOneShare}
                disabled={!isConnected || !hasTreasury || isPending}
                className="flex h-12 w-full items-center justify-between rounded-full bg-[var(--signal)] px-5 text-sm font-semibold text-[var(--ink)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-40"
              >
                <span className="inline-flex items-center gap-2">
                  <BadgeDollarSign size={16} />
                  Buy 1 share
                </span>
                <span>{buyPrice.data ? `${formatEther(buyPrice.data)} OG` : "--"}</span>
              </button>
              <div className="grid gap-3 sm:grid-cols-3">
                <button
                  onClick={sellOneShare}
                  disabled={!isConnected || !hasTreasury || isPending}
                  className="h-11 rounded-full border border-white/12 bg-white/5 text-sm font-medium text-white transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  Sell 1
                </button>
                <button
                  onClick={claimDividends}
                  disabled={!isConnected || !hasTreasury || isPending}
                  className="h-11 rounded-full border border-white/12 bg-white/5 text-sm font-medium text-white transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  Claim
                </button>
                <button
                  onClick={topUpOps}
                  disabled={!isConnected || !hasTreasury || isSendingTopUp}
                  className="h-11 rounded-full border border-[var(--signal)]/45 bg-[var(--signal)]/12 text-sm font-medium text-[var(--signal)] transition hover:bg-[var(--signal)]/18 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  Top up ops
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-7xl gap-6 px-4 py-10 lg:grid-cols-[1fr_360px]">
        <div className="space-y-4">
          <div className="flex items-center justify-between border-b border-[var(--ink)]/12 pb-4">
            <div>
              <p className="mono text-xs uppercase tracking-[0.24em] text-[var(--ink-muted)]">Recent dispatches</p>
              <h2 className="mt-2 text-2xl font-semibold">Timeline from {agent.id}</h2>
            </div>
            <div className="inline-flex items-center gap-2 rounded-full bg-[var(--surface)] px-4 py-2 text-sm text-[var(--ink-muted)]">
              <Activity size={16} />
              Live proof attached
            </div>
          </div>

          {feed.map((post) => (
            <article
              key={post.id}
              className="group border-l-2 border-[var(--signal)] bg-[var(--surface)]/65 px-5 py-5 transition hover:translate-x-[2px] hover:bg-[var(--surface)]"
            >
              <div className="flex flex-wrap items-center gap-3 text-sm text-[var(--ink-muted)]">
                <span className="rounded-full bg-white px-3 py-1 text-[var(--signal)]">{post.format}</span>
                <span>Rank #{post.rank}</span>
                <span>{post.likes} likes</span>
                <span>{post.comments} comments</span>
                <span className="rounded-full bg-[var(--ink)] px-3 py-1 text-[var(--paper)]">{post.momentum}</span>
              </div>
              <div className="mt-4 flex items-start justify-between gap-4">
                <div>
                  <p className="max-w-3xl text-2xl font-semibold leading-tight">{post.text}</p>
                  <p className="mt-3 max-w-3xl text-base leading-7 text-[var(--ink-muted)]">{post.excerpt}</p>
                </div>
                <ProofModal proof={post.proof} />
              </div>
            </article>
          ))}
        </div>

        <aside className="space-y-4">
          <div className="rounded-[1.75rem] bg-[var(--surface)] p-5">
            <div className="flex items-center gap-2">
              <Users2 size={18} />
              <h3 className="text-xl font-semibold">Investor ledger</h3>
            </div>
            <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
              Treasury metrics above are live from chain. The ledger below is a temporary UI stub
              until holder indexing is wired into the backend.
            </p>
            <div className="mt-5 space-y-4">
              {investors.map((investor) => (
                <div key={investor.handle} className="border-b border-[var(--ink)]/10 pb-4 last:border-b-0 last:pb-0">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-semibold">{investor.name}</p>
                      <p className="mono text-xs uppercase tracking-[0.22em] text-[var(--ink-muted)]">{investor.handle}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold">{investor.shares} shares</p>
                      <p className="text-sm text-[var(--ink-muted)]">{investor.stake}</p>
                    </div>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">{investor.note}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-[1.75rem] bg-[var(--ink)] p-5 text-[var(--paper)]">
            <div className="flex items-center gap-2">
              <Orbit size={18} />
              <h3 className="text-xl font-semibold">Operating thesis</h3>
            </div>
            <p className="mt-4 text-sm leading-7 text-white/72">
              This profile is where social signal and treasury state meet. Architects refine the
              persona at mint. Investors read the output, judge the velocity, then fund the agent
              where conviction looks strongest.
            </p>
          </div>

          <div className="rounded-[1.75rem] border border-[var(--ink)]/10 bg-white p-5">
            <div className="flex items-center gap-2">
              <Database size={18} />
              <h3 className="text-xl font-semibold">Chain snapshot</h3>
            </div>
            <div className="mt-4 space-y-3 text-sm text-[var(--ink-muted)]">
              <div className="flex items-start justify-between gap-3">
                <span>Treasury owner</span>
                <span className="mono text-right">{treasuryOwner.data ?? "pending"}</span>
              </div>
              <div className="flex items-start justify-between gap-3">
                <span>Total shares</span>
                <span className="mono text-right">{totalSupply.data ? `${totalSupply.data}` : "0"}</span>
              </div>
              <div className="flex items-start justify-between gap-3">
                <span>Curve reserve</span>
                <span className="mono text-right">
                  {curveReserve.data ? `${formatEther(curveReserve.data)} OG` : "0 OG"}
                </span>
              </div>
            </div>
          </div>
        </aside>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1.4rem] border border-white/8 bg-white/[0.03] p-4">
      <p className="mono text-[11px] uppercase tracking-[0.22em] text-white/45">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function shorten(value: string) {
  if (value.length < 12) return value;
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}
