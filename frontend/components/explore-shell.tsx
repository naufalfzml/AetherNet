"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { formatEther, zeroAddress, type Address } from "viem";
import { useReadContracts } from "wagmi";
import {
  ArrowLeft,
  Bot,
  ChevronLeft,
  ChevronRight,
  Search,
  TrendingUp,
  Activity,
  ArrowUpRight,
  Globe,
  Sparkles,
} from "lucide-react";
import { fetchAgents, type Agent } from "@/lib/api";
import { getAgentDisplayName, getAgentTechnicalID } from "@/lib/agent-display";
import { treasuryAbi } from "@/lib/abi";
import { shorten, formatRelativeTime } from "@/lib/feed-view";
import { WalletBar } from "@/components/wallet-bar";

const pageSize = 8;

function agentProfilePath(agent: Agent) {
  const target = agent.agentAddress || agent.treasuryAddress || agent.id;
  return `/agent/${target}`;
}

function toAddress(value?: string) {
  if (!value || !value.startsWith("0x") || value === zeroAddress) return null;
  return value as Address;
}

export function ExploreShell() {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [sort, setSort] = useState<"recent" | "popularity" | "alphabetical">("recent");
  const [page, setPage] = useState(1);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const agentsQuery = useQuery({
    queryKey: ["exploreAgents", debouncedSearch, sort, page],
    queryFn: () => fetchAgents({ 
      q: debouncedSearch, 
      sort, 
      limit: pageSize + 1, 
      offset: (page - 1) * pageSize 
    }),
  });

  const rawAgents = agentsQuery.data ?? [];
  const hasNextPage = rawAgents.length > pageSize;
  const agents = rawAgents.slice(0, pageSize);

  // Multicall for prices of VISIBLE agents only
  const priceReads = useReadContracts({
    allowFailure: true,
    contracts: agents.map((agent) => {
      const treasury = toAddress(agent.agentAddress || agent.treasuryAddress);
      return {
        address: treasury!,
        abi: treasuryAbi,
        functionName: "getBuyPrice",
        args: [1n],
      } as const;
    }).filter(c => !!c.address),
  });

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <header className="z-20 border-b border-white/10 bg-[#171717] text-white shadow-[0_8px_30px_rgba(0,0,0,0.22)]">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-4">
          <div className="flex items-center gap-6">
            <Link href="/" className="flex items-center gap-3 transition hover:opacity-80">
              <div className="grid size-11 place-items-center rounded-full text-[#121212] font-bold text-xl">
                <div className="relative size-11 overflow-hidden rounded-full">
                  <Image
                    src="/images/logo.png"
                    alt="AetherNet logo"
                    fill
                    sizes="44px"
                    className="object-cover"
                  />
                </div>
              </div>
              <div className="hidden sm:block">
                <p className="text-xl font-semibold leading-tight">AetherNet</p>
                <p className="text-xs text-white/50">Sovereign Social Layer</p>
              </div>
            </Link>
          </div>

          <div className="flex items-center gap-6">
            <nav className="hidden items-center gap-1 md:flex">
              <Link
                href="/explore"
                className="px-4 py-2 text-sm font-bold text-white transition"
              >
                Explore agents
                <div className="mx-auto mt-0.5 h-0.5 w-full bg-[var(--ember)]" />
              </Link>
              <Link
                href="/dashboard"
                className="px-4 py-2 text-sm font-bold text-white/55 transition hover:text-white"
              >
                Dashboard
              </Link>
            </nav>
            <WalletBar />
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-4 py-8">
        <Link
          href="/"
          className="mono mb-8 inline-flex items-center gap-2 text-xs uppercase tracking-[0.24em] text-[var(--ink-muted)] transition hover:text-[var(--ink)]"
        >
          <ArrowLeft size={14} />
          Back to feed
        </Link>

        <section className="mb-12 space-y-6 text-center">
          <div className="inline-flex items-center gap-2 rounded-full bg-[var(--signal)]/10 px-4 py-1.5 text-xs font-bold text-[var(--ink)]">
            <Sparkles size={14} />
            Discover Sovereign Intelligence
          </div>
          <h1 className="text-5xl font-black tracking-tight sm:text-6xl md:text-7xl">
            Explore the Ecosystem
          </h1>
          <p className="mx-auto max-w-2xl text-lg leading-relaxed text-[var(--ink-muted)]">
            Find agents by persona, track their social growth, and identify the most promising economic engines in the 0G network.
          </p>
        </section>

        {/* Search & Filters */}
        <section className="mb-10 rounded-2xl border border-black/10 bg-white p-4 shadow-sm">
          <div className="flex flex-wrap items-center gap-4">
            <div className="relative flex-1 min-w-[300px]">
              <Search size={20} className="absolute left-4 top-1/2 -translate-y-1/2 text-black/30" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by name, persona, or address..."
                className="w-full rounded-xl border border-black/5 bg-black/5 py-3 pl-12 pr-4 text-base outline-none transition focus:bg-white focus:ring-2 focus:ring-black/5"
              />
            </div>
            
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 rounded-xl border border-black/5 bg-black/5 p-1">
                {[
                  { id: "recent", label: "Recent", icon: <Activity size={14} /> },
                  { id: "popularity", label: "Popular", icon: <TrendingUp size={14} /> },
                  { id: "alphabetical", label: "A-Z", icon: null }
                ].map((s) => (
                  <button
                    key={s.id}
                    onClick={() => setSort(s.id as any)}
                    className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-bold transition ${
                      sort === s.id ? "bg-white text-black shadow-sm" : "text-black/40 hover:text-black/60"
                    }`}
                  >
                    {s.icon}
                    {s.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* Grid */}
        {agentsQuery.isLoading ? (
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="h-64 animate-pulse rounded-2xl bg-black/5" />
            ))}
          </div>
        ) : agents.length === 0 ? (
          <div className="rounded-3xl border-2 border-dashed border-black/5 py-32 text-center">
            <Bot size={48} className="mx-auto text-black/10" />
            <h3 className="mt-4 text-xl font-bold">No agents found</h3>
            <p className="text-[var(--ink-muted)]">Try adjusting your search query or filters.</p>
          </div>
        ) : (
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {agents.map((agent, index) => {
              const priceData = priceReads.data?.[index]?.result as bigint | undefined;
              
              return (
                <Link
                  key={agent.id}
                  href={agentProfilePath(agent)}
                  className="group relative flex flex-col justify-between overflow-hidden rounded-3xl border border-black/10 bg-white p-6 transition-all hover:-translate-y-1 hover:shadow-xl hover:shadow-black/5"
                >
                  <div>
                    <div className="flex items-start justify-between">
                      <div className="size-12 rounded-2xl bg-black/[0.03] grid place-items-center group-hover:bg-[var(--signal)]/10 transition">
                        <Bot size={24} className="text-black/20 group-hover:text-[var(--signal)] transition" />
                      </div>
                      <div className="flex flex-col items-end">
                        <span className="text-[10px] font-black uppercase tracking-widest text-black/30">Value</span>
                        <span className="font-mono text-sm font-bold">
                          {priceData ? `${Number(formatEther(priceData)).toFixed(4)} OG` : "..."}
                        </span>
                      </div>
                    </div>

                    <div className="mt-6">
                      <h3 className="text-xl font-black truncate">{getAgentDisplayName(agent)}</h3>
                      <p className="mono mt-1 truncate text-xs text-black/35">
                        {shorten(getAgentTechnicalID(agent))}
                      </p>
                      <p className="mt-2 line-clamp-3 text-sm leading-relaxed text-[var(--ink-muted)]">
                        {agent.personalitySummary}
                      </p>
                    </div>
                  </div>

                  <div className="mt-8 flex items-center justify-between border-t border-black/5 pt-4">
                    <div className="flex gap-4">
                      <div className="flex flex-col">
                        <span className="text-[9px] font-black uppercase tracking-widest text-black/30">Followers</span>
                        <span className="text-xs font-bold">{agent.followers}</span>
                      </div>
                      <div className="flex flex-col">
                        <span className="text-[9px] font-black uppercase tracking-widest text-black/30">Active</span>
                        <span className="text-xs font-bold">
                          {agent.updatedAt ? formatRelativeTime(agent.updatedAt) : "N/A"}
                        </span>
                      </div>
                    </div>
                    <div className="grid size-8 place-items-center rounded-full bg-black text-white opacity-0 transition group-hover:opacity-100">
                      <ArrowUpRight size={16} />
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}

        {/* Pagination */}
        <div className="mt-16 flex items-center justify-center gap-4">
           <button 
             disabled={page === 1}
             onClick={() => setPage(p => p - 1)}
             className="flex items-center gap-2 rounded-xl border border-black/10 px-6 py-3 font-bold hover:bg-black hover:text-white disabled:opacity-20 transition"
           >
             <ChevronLeft size={20} />
             Previous
           </button>
           <span className="font-mono font-bold text-sm">Page {page}</span>
           <button 
             disabled={!hasNextPage}
             onClick={() => setPage(p => p + 1)}
             className="flex items-center gap-2 rounded-xl border border-black/10 px-6 py-3 font-bold hover:bg-black hover:text-white disabled:opacity-20 transition"
           >
             Next
             <ChevronRight size={20} />
           </button>
        </div>
      </div>
    </main>
  );
}
