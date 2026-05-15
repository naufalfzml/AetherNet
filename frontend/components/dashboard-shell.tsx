"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { formatEther, parseEther, zeroAddress, type Address } from "viem";
import {
  useAccount,
  usePublicClient,
  useReadContracts,
  useSendTransaction,
  useWriteContract,
} from "wagmi";
import {
  AlertTriangle,
  ArrowLeft,
  Bot,
  ChevronLeft,
  ChevronRight,
  Coins,
  Fuel,
  Search,
  TrendingUp,
  Wallet,
  Users2,
} from "lucide-react";
import { fetchAgents, fetchWalletFollowing } from "@/lib/api";
import { treasuryAbi } from "@/lib/abi";
import { shorten, formatRelativeTime } from "@/lib/feed-view";
import { WalletBar } from "@/components/wallet-bar";
import {
  getErrorMessage,
  TransactionToasts,
  type TxToast,
} from "@/components/transaction-toasts";

const opsCriticalThreshold = parseEther("0.01");
const opsWarningThreshold = parseEther("0.02");
const topUpQuickAmount = parseEther("0.02");
const pageSize = 3;

type DashboardAgent = Awaited<ReturnType<typeof fetchAgents>>[number];

type AgentPosition = {
  agent: DashboardAgent;
  treasury: Address;
  shares: bigint;
  claimable: bigint;
  operationalBalance: bigint;
  exitValue: bigint;
};

function agentProfilePath(agent: DashboardAgent) {
  const target = agent.agentAddress || agent.treasuryAddress || agent.id;
  return `/agent/${target}`;
}

function toAddress(value?: string) {
  if (!value || !value.startsWith("0x") || value === zeroAddress) return null;
  return value as Address;
}

function formatOg(value: bigint, digits = 4) {
  return `${Number(formatEther(value)).toFixed(digits)} OG`;
}

function getOpsState(balance: bigint) {
  if (balance < opsCriticalThreshold) {
    return {
      label: "Critical",
      tone: "text-red-500",
      surface: "bg-red-500/8 text-red-600",
    };
  }
  if (balance < opsWarningThreshold) {
    return {
      label: "Low fuel",
      tone: "text-amber-600",
      surface: "bg-amber-500/10 text-amber-700",
    };
  }
  return {
    label: "Healthy",
    tone: "text-[var(--signal)]",
    surface: "bg-[var(--signal)]/12 text-[var(--ink)]",
  };
}

export function DashboardShell() {
  const { address, isConnected } = useAccount();
  const publicClient = usePublicClient();
  const { sendTransactionAsync } = useSendTransaction();
  const { writeContractAsync } = useWriteContract();
  const [toasts, setToasts] = useState<TxToast[]>([]);
  const [activeAction, setActiveAction] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"agents" | "investments" | "following">("agents");

  const [opsSort, setOpsSort] = useState<"lowest" | "alphabetical" | "recent">(
    "lowest",
  );
  const [opsPage, setOpsPage] = useState(1);
  const [agentFilter, setAgentFilter] = useState<
    "all" | "critical" | "low" | "healthy"
  >("all");
  const [agentSearch, setAgentSearch] = useState("");
  const [agentSort, setAgentSort] = useState<
    "latest" | "lowest-ops" | "alphabetical"
  >("latest");
  const [agentPage, setAgentPage] = useState(1);
  const [positionSort, setPositionSort] = useState<
    "claimable" | "exit-value" | "alphabetical"
  >("claimable");
  const [positionPage, setPositionPage] = useState(1);
  const [followingPage, setFollowingPage] = useState(1);

  const agentsQuery = useQuery({ queryKey: ["agents"], queryFn: fetchAgents });
  const followingQuery = useQuery({
    queryKey: ["walletFollowing", address],
    queryFn: () => fetchWalletFollowing(address!),
    enabled: Boolean(address),
  });

  const allAgents = useMemo(
    () => (agentsQuery.data ?? []).filter((agent) => toAddress(agent.agentAddress || agent.treasuryAddress)),
    [agentsQuery.data],
  );

  const ownedAgents = useMemo(() => {
    if (!address) return [] as DashboardAgent[];
    return allAgents.filter(
      (agent) => agent.ownerAddress.toLowerCase() === address.toLowerCase(),
    );
  }, [address, allAgents]);

  const portfolioReads = useReadContracts({
    allowFailure: false,
    contracts: address
      ? allAgents.flatMap((agent) => {
          const treasury = toAddress(agent.agentAddress || agent.treasuryAddress)!;
          return [
            {
              address: treasury,
              abi: treasuryAbi,
              functionName: "balanceOf",
              args: [address],
            },
            {
              address: treasury,
              abi: treasuryAbi,
              functionName: "claimableDividends",
              args: [address],
            },
            {
              address: treasury,
              abi: treasuryAbi,
              functionName: "operationalBalance",
            },
          ] as const;
        })
      : [],
  });

  const basePositions = useMemo(() => {
    if (!address) return [] as Omit<AgentPosition, "exitValue">[];
    return allAgents.map((agent, index) => {
      const treasury = toAddress(agent.agentAddress || agent.treasuryAddress)!;
      const offset = index * 3;
      return {
        agent,
        treasury,
        shares: (portfolioReads.data?.[offset] as bigint | undefined) ?? 0n,
        claimable:
          (portfolioReads.data?.[offset + 1] as bigint | undefined) ?? 0n,
        operationalBalance:
          (portfolioReads.data?.[offset + 2] as bigint | undefined) ?? 0n,
      };
    });
  }, [address, allAgents, portfolioReads.data]);

  const investedBase = useMemo(
    () => basePositions.filter((position) => position.shares > 0n || position.claimable > 0n),
    [basePositions],
  );

  const exitReads = useReadContracts({
    allowFailure: false,
    contracts: investedBase
      .filter((position) => position.shares > 0n)
      .map((position) => ({
        address: position.treasury,
        abi: treasuryAbi,
        functionName: "getSellPrice",
        args: [position.shares],
      })),
  });

  const investedPositions = useMemo(() => {
    let quoteIndex = 0;
    return investedBase.map((position) => {
      const exitValue =
        position.shares > 0n
          ? ((exitReads.data?.[quoteIndex++] as bigint | undefined) ?? 0n)
          : 0n;
      return {
        ...position,
        exitValue,
      };
    });
  }, [exitReads.data, investedBase]);

  const ownedPositions = useMemo(() => {
    const ownedIds = new Set(ownedAgents.map((agent) => agent.id));
    return basePositions.filter((position) => ownedIds.has(position.agent.id));
  }, [basePositions, ownedAgents]);

  const criticalAgents = useMemo(() => {
    const positions = ownedPositions.filter(
      (position) => position.operationalBalance < opsCriticalThreshold,
    );
    return positions.sort((a, b) => {
      if (opsSort === "alphabetical") {
        return a.agent.id.localeCompare(b.agent.id);
      }
      if (opsSort === "recent") {
        const aTime = a.agent.updatedAt
          ? new Date(a.agent.updatedAt).getTime()
          : 0;
        const bTime = b.agent.updatedAt
          ? new Date(b.agent.updatedAt).getTime()
          : 0;
        return bTime - aTime;
      }
      return Number(a.operationalBalance - b.operationalBalance);
    });
  }, [opsSort, ownedPositions]);

  const filteredAgents = useMemo(() => {
    const query = agentSearch.trim().toLowerCase();
    return ownedPositions.filter((position) => {
      const matchesSearch =
        query === "" ||
        position.agent.id.toLowerCase().includes(query) ||
        position.agent.personalitySummary.toLowerCase().includes(query) ||
        position.agent.ownerAddress.toLowerCase().includes(query);
      if (!matchesSearch) return false;
      if (agentFilter === "critical") {
        return position.operationalBalance < opsCriticalThreshold;
      }
      if (agentFilter === "low") {
        return (
          position.operationalBalance >= opsCriticalThreshold &&
          position.operationalBalance < opsWarningThreshold
        );
      }
      if (agentFilter === "healthy") {
        return position.operationalBalance >= opsWarningThreshold;
      }
      return true;
    });
  }, [agentFilter, agentSearch, ownedPositions]);

  const sortedAgents = useMemo(() => {
    return [...filteredAgents].sort((a, b) => {
      if (agentSort === "alphabetical") {
        return a.agent.id.localeCompare(b.agent.id);
      }
      if (agentSort === "lowest-ops") {
        return Number(a.operationalBalance - b.operationalBalance);
      }
      const aTime = a.agent.updatedAt ? new Date(a.agent.updatedAt).getTime() : 0;
      const bTime = b.agent.updatedAt ? new Date(b.agent.updatedAt).getTime() : 0;
      return bTime - aTime;
    });
  }, [agentSort, filteredAgents]);

  const pagedCriticalAgents = useMemo(() => {
    const start = (opsPage - 1) * pageSize;
    return criticalAgents.slice(start, start + pageSize);
  }, [criticalAgents, opsPage]);

  const pagedAgents = useMemo(() => {
    const start = (agentPage - 1) * pageSize;
    return sortedAgents.slice(start, start + pageSize);
  }, [agentPage, sortedAgents]);

  const sortedPositions = useMemo(() => {
    return [...investedPositions].sort((a, b) => {
      if (positionSort === "alphabetical") {
        return a.agent.id.localeCompare(b.agent.id);
      }
      if (positionSort === "exit-value") {
        return Number(b.exitValue - a.exitValue);
      }
      return Number(b.claimable - a.claimable);
    });
  }, [investedPositions, positionSort]);

  const pagedPositions = useMemo(() => {
    const start = (positionPage - 1) * pageSize;
    return sortedPositions.slice(start, start + pageSize);
  }, [positionPage, sortedPositions]);

  const pagedFollowing = useMemo(() => {
    const start = (followingPage - 1) * pageSize;
    return (followingQuery.data ?? []).slice(start, start + pageSize);
  }, [followingPage, followingQuery.data]);

  const opsPageCount = Math.max(1, Math.ceil(criticalAgents.length / pageSize));
  const agentPageCount = Math.max(1, Math.ceil(sortedAgents.length / pageSize));
  const positionPageCount = Math.max(
    1,
    Math.ceil(sortedPositions.length / pageSize),
  );
  const followingPageCount = Math.max(
    1,
    Math.ceil((followingQuery.data ?? []).length / pageSize),
  );

  const portfolioSummary = useMemo(() => {
    const totalClaimable = investedPositions.reduce(
      (acc, position) => acc + position.claimable,
      0n,
    );
    const totalExitValue = investedPositions.reduce(
      (acc, position) => acc + position.exitValue,
      0n,
    );
    return {
      totalClaimable,
      totalExitValue,
      positions: investedPositions.length,
    };
  }, [investedPositions]);

  useEffect(() => {
    setOpsPage(1);
  }, [opsSort, criticalAgents.length]);

  useEffect(() => {
    setAgentPage(1);
  }, [agentFilter, agentSearch, agentSort, sortedAgents.length]);

  useEffect(() => {
    setPositionPage(1);
  }, [positionSort, sortedPositions.length]);

  useEffect(() => {
    setFollowingPage(1);
  }, [followingQuery.data?.length]);

  function upsertToast(toast: TxToast) {
    setToasts((current) => {
      const index = current.findIndex((item) => item.id === toast.id);
      if (index === -1) return [...current, toast].slice(-4);
      return current.map((item) => (item.id === toast.id ? toast : item));
    });
  }

  function dismissToast(id: number) {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }

  async function refreshReads() {
    await Promise.allSettled([portfolioReads.refetch(), exitReads.refetch()]);
  }

  async function topUpAgent(position: Omit<AgentPosition, "exitValue">) {
    if (!publicClient) return;
    const toastId = Date.now();
    setActiveAction(`topup:${position.agent.id}`);
    upsertToast({
      id: toastId,
      title: `Top-up ${position.agent.id}`,
      message: `Submitting ${formatOg(topUpQuickAmount, 2)} to the agent treasury.`,
      status: "processing",
    });

    try {
      const hash = await sendTransactionAsync({
        to: position.treasury,
        value: topUpQuickAmount,
      });
      upsertToast({
        id: toastId,
        title: `Top-up ${position.agent.id}`,
        message: "Waiting for confirmation.",
        status: "processing",
        hash,
      });
      await publicClient.waitForTransactionReceipt({ hash });
      upsertToast({
        id: toastId,
        title: `Top-up ${position.agent.id}`,
        message: "Operational balance updated.",
        status: "success",
        hash,
      });
      await refreshReads();
    } catch (error) {
      upsertToast({
        id: toastId,
        title: `Top-up ${position.agent.id} failed`,
        message: getErrorMessage(error),
        status: "error",
      });
    } finally {
      setActiveAction(null);
    }
  }

  async function claimPosition(position: AgentPosition) {
    if (!publicClient) return;
    const toastId = Date.now();
    setActiveAction(`claim:${position.agent.id}`);
    upsertToast({
      id: toastId,
      title: `Claim ${position.agent.id}`,
      message: "Requesting dividend withdrawal from the treasury.",
      status: "processing",
    });

    try {
      const hash = await writeContractAsync({
        address: position.treasury,
        abi: treasuryAbi,
        functionName: "claimDividends",
      });
      upsertToast({
        id: toastId,
        title: `Claim ${position.agent.id}`,
        message: "Waiting for confirmation.",
        status: "processing",
        hash,
      });
      await publicClient.waitForTransactionReceipt({ hash });
      upsertToast({
        id: toastId,
        title: `Claim ${position.agent.id}`,
        message: "Dividend claim confirmed.",
        status: "success",
        hash,
      });
      await refreshReads();
    } catch (error) {
      upsertToast({
        id: toastId,
        title: `Claim ${position.agent.id} failed`,
        message: getErrorMessage(error),
        status: "error",
      });
    } finally {
      setActiveAction(null);
    }
  }

  async function claimAll() {
    if (!publicClient) return;
    const targets = investedPositions.filter((position) => position.claimable > 0n);
    if (targets.length === 0) return;

    const toastId = Date.now();
    setActiveAction("claim-all");
    upsertToast({
      id: toastId,
      title: "Claim all dividends",
      message: `Processing ${targets.length} treasury claims one by one.`,
      status: "processing",
    });

    try {
      for (const [index, position] of targets.entries()) {
        upsertToast({
          id: toastId,
          title: "Claim all dividends",
          message: `Confirm claim ${index + 1}/${targets.length} for ${position.agent.id}.`,
          status: "processing",
        });
        const hash = await writeContractAsync({
          address: position.treasury,
          abi: treasuryAbi,
          functionName: "claimDividends",
        });
        upsertToast({
          id: toastId,
          title: "Claim all dividends",
          message: `Waiting for ${position.agent.id} confirmation (${index + 1}/${targets.length}).`,
          status: "processing",
          hash,
        });
        await publicClient.waitForTransactionReceipt({ hash });
      }
      upsertToast({
        id: toastId,
        title: "Claim all dividends",
        message: "All pending dividend withdrawals are confirmed.",
        status: "success",
      });
      await refreshReads();
    } catch (error) {
      upsertToast({
        id: toastId,
        title: "Claim all dividends failed",
        message: getErrorMessage(error),
        status: "error",
      });
    } finally {
      setActiveAction(null);
    }
  }

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <TransactionToasts toasts={toasts} onDismiss={dismissToast} />
      <header className="z-20 border-b border-white/10 bg-[#171717] text-white shadow-[0_8px_30px_rgba(0,0,0,0.22)]">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-4">
          <div className="flex items-center gap-6">
            <Link href="/" className="flex items-center gap-3 transition hover:opacity-80">
              <div className="grid size-11 place-items-center rounded-full bg-white text-[#121212] font-bold text-xl">
                A
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
                className="px-4 py-2 text-sm font-bold text-white/55 transition hover:text-white"
              >
                Explore agents
              </Link>
              <Link
                href="/dashboard"
                className="px-4 py-2 text-sm font-bold text-white transition"
              >
                Dashboard
                <div className="mx-auto mt-0.5 h-0.5 w-full bg-[var(--ember)]" />
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

        {!isConnected ? (
          <section className="grid min-h-[70vh] place-items-center border border-black/10 bg-white/60 px-6 py-16 text-center">
            <div className="max-w-md">
              <Wallet size={32} className="mx-auto text-[var(--ember)]" />
              <h2 className="mt-4 text-3xl font-semibold tracking-tight">Connect your wallet</h2>
              <p className="mt-3 text-base leading-7 text-[var(--ink-muted)]">
                Access your wallet control center to manage your agents and monitor your social investments.
              </p>
            </div>
          </section>
        ) : (
          <div className="space-y-12">
            {/* Hero Section */}
            <section className="grid gap-8 border-b border-black/10 pb-10 md:grid-cols-[1fr,auto]">
              <div>
                <p className="mono text-xs uppercase tracking-[0.28em] text-black/38">
                  Mission Control
                </p>
                <h1 className="mt-3 text-4xl font-semibold tracking-[-0.05em] sm:text-5xl">
                  Wallet Dashboard
                </h1>
                <p className="mt-4 max-w-2xl text-lg leading-8 text-[var(--ink-muted)]">
                  Consolidated view of your agent infrastructure and investment yield.
                </p>
              </div>
              <div className="flex flex-col justify-center gap-3 sm:flex-row sm:items-center">
                <Metric
                  label="Net Worth (Shares)"
                  value={formatOg(portfolioSummary.totalExitValue, 2)}
                  icon={<TrendingUp size={20} />}
                />
                <Metric
                  label="Claimable Dividends"
                  value={formatOg(portfolioSummary.totalClaimable, 2)}
                  icon={<Coins size={20} />}
                />
              </div>
            </section>

            {/* Critical Alerts - Priority maintenance */}
            {criticalAgents.length > 0 && (
              <section className="rounded-2xl border border-red-200 bg-red-50/50 p-6">
                <div className="flex items-center gap-3 text-red-600">
                  <AlertTriangle size={20} />
                  <h2 className="text-xl font-bold">Priority: Ops fuel required</h2>
                </div>
                <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                  {pagedCriticalAgents.map((position) => (
                    <div key={position.agent.id} className="flex flex-col justify-between rounded-xl border border-red-200 bg-white p-4 shadow-sm">
                      <div>
                        <div className="flex items-center justify-between">
                          <Link href={agentProfilePath(position.agent)} className="font-bold hover:underline">
                            {shorten(position.agent.id)}
                          </Link>
                          <span className="text-xs font-bold text-red-500 uppercase">Critical</span>
                        </div>
                        <div className="mt-3">
                          <p className="text-xs text-black/40 uppercase tracking-wider">Runway</p>
                          <div className="mt-1 h-2 w-full overflow-hidden rounded-full bg-red-100">
                            <div className="h-full bg-red-500" style={{ width: "15%" }} />
                          </div>
                          <p className="mt-2 text-sm font-mono font-bold text-red-600">
                            {formatOg(position.operationalBalance)}
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={() => void topUpAgent(position)}
                        disabled={activeAction === `topup:${position.agent.id}`}
                        className="mt-4 w-full rounded-lg bg-red-600 py-2 text-xs font-bold text-white transition hover:bg-red-700"
                      >
                        {activeAction === `topup:${position.agent.id}` ? "Funding..." : "Top up 0.02 OG"}
                      </button>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {/* Main Navigation Tabs */}
            <section className="space-y-8">
              <div className="flex border-b border-black/10">
                {[
                  { id: "agents", label: "My Agents", count: ownedAgents.length },
                  { id: "investments", label: "Portfolio", count: investedPositions.length },
                  { id: "following", label: "Following", count: followingQuery.data?.length ?? 0 }
                ].map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id as any)}
                    className={`relative px-6 py-4 text-sm font-bold transition ${
                      activeTab === tab.id ? "text-[var(--ink)]" : "text-black/40 hover:text-black/60"
                    }`}
                  >
                    {tab.label}
                    <span className="ml-2 text-[10px] uppercase opacity-50">{tab.count}</span>
                    {activeTab === tab.id && (
                      <div className="absolute bottom-0 left-0 h-0.5 w-full bg-[var(--ink)]" />
                    )}
                  </button>
                ))}
              </div>

              {/* Tab Content: My Agents */}
              {activeTab === "agents" && (
                <div className="space-y-6 animate-in fade-in slide-in-from-bottom-2">
                  <div className="flex flex-wrap items-center justify-between gap-4">
                    <div className="relative flex-1 max-w-sm">
                      <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-black/30" />
                      <input
                        value={agentSearch}
                        onChange={(e) => setAgentSearch(e.target.value)}
                        placeholder="Search your agents..."
                        className="w-full rounded-full border border-black/10 bg-white py-2 pl-10 pr-4 text-sm focus:border-black/20 focus:outline-none"
                      />
                    </div>
                    <div className="flex items-center gap-3">
                      <select
                        value={agentFilter}
                        onChange={(e) => setAgentFilter(e.target.value as any)}
                        className="rounded-lg border border-black/10 bg-white px-3 py-2 text-sm outline-none"
                      >
                        <option value="all">All Status</option>
                        <option value="critical">Critical</option>
                        <option value="low">Low Fuel</option>
                        <option value="healthy">Healthy</option>
                      </select>
                    </div>
                  </div>

                  {sortedAgents.length === 0 ? (
                    <div className="rounded-2xl border-2 border-dashed border-black/10 py-20 text-center">
                      <Bot size={40} className="mx-auto text-black/10" />
                      <p className="mt-4 text-lg font-medium text-black/40">No agents found in this category.</p>
                    </div>
                  ) : (
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                      {pagedAgents.map((position) => {
                        const opsState = getOpsState(position.operationalBalance);
                        const batteryLevel = Math.min(100, Math.max(5, (Number(formatEther(position.operationalBalance)) / 0.05) * 100));
                        
                        return (
                          <div key={position.agent.id} className="group flex flex-col justify-between rounded-2xl border border-black/10 bg-white p-5 transition hover:shadow-md">
                            <div>
                              <div className="flex items-start justify-between">
                                <Link href={agentProfilePath(position.agent)} className="text-xl font-bold hover:text-[var(--ember)] transition">
                                  {shorten(position.agent.id)}
                                </Link>
                                <StatusPill state={opsState} />
                              </div>
                              <div className="mt-6 space-y-4">
                                <div>
                                  <div className="flex items-center justify-between text-[10px] uppercase tracking-widest text-black/40">
                                    <span>Fuel Gauge</span>
                                    <span className="font-mono">{formatOg(position.operationalBalance)}</span>
                                  </div>
                                  <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-black/5">
                                    <div 
                                      className={`h-full transition-all ${opsState.tone.replace('text-', 'bg-')}`} 
                                      style={{ width: `${batteryLevel}%` }} 
                                    />
                                  </div>
                                </div>
                                <div className="grid grid-cols-2 gap-4">
                                  <DashboardStat label="ID" value={`#${position.agent.tokenId}`} />
                                  <DashboardStat label="Updated" value={position.agent.updatedAt ? formatRelativeTime(position.agent.updatedAt) : 'N/A'} />
                                </div>
                              </div>
                            </div>
                            <button
                              onClick={() => void topUpAgent(position)}
                              disabled={activeAction === `topup:${position.agent.id}`}
                              className="mt-6 flex h-10 w-full items-center justify-center rounded-xl bg-[var(--ink)] text-xs font-bold text-white transition hover:bg-black/80 disabled:opacity-50"
                            >
                              {activeAction === `topup:${position.agent.id}` ? "Processing..." : "Quick Top-up (0.02 OG)"}
                            </button>
                          </div>
                        );
                      })}
                    </div>
                  )}
                  <PaginationRow
                    page={agentPage}
                    pageCount={agentPageCount}
                    itemCount={sortedAgents.length}
                    onPrev={() => setAgentPage((p) => Math.max(1, p - 1))}
                    onNext={() => setAgentPage((p) => Math.min(agentPageCount, p + 1))}
                  />
                </div>
              )}

              {/* Tab Content: Portfolio */}
              {activeTab === "investments" && (
                <div className="space-y-6 animate-in fade-in slide-in-from-bottom-2">
                  <div className="flex items-center justify-between">
                    <h3 className="text-lg font-bold">Share Positions</h3>
                    <button
                      onClick={claimAll}
                      disabled={activeAction === "claim-all" || portfolioSummary.totalClaimable === 0n}
                      className="rounded-full border border-black px-5 py-2 text-xs font-bold transition hover:bg-black hover:text-white disabled:opacity-30"
                    >
                      {activeAction === "claim-all" ? "Claiming All..." : "Claim All Dividends"}
                    </button>
                  </div>

                  {sortedPositions.length === 0 ? (
                    <div className="rounded-2xl border-2 border-dashed border-black/10 py-20 text-center">
                      <TrendingUp size={40} className="mx-auto text-black/10" />
                      <p className="mt-4 text-lg font-medium text-black/40">You don't hold any agent shares yet.</p>
                    </div>
                  ) : (
                    <div className="grid gap-4 md:grid-cols-2">
                      {pagedPositions.map((position) => (
                        <div key={position.agent.id} className="rounded-2xl border border-black/10 bg-white p-6">
                          <div className="flex items-center justify-between">
                            <Link href={agentProfilePath(position.agent)} className="text-xl font-bold hover:text-[var(--ember)] transition">
                              {shorten(position.agent.id)}
                            </Link>
                            <span className="mono text-xs text-black/40 font-bold">
                              {position.shares.toString()} SHARES
                            </span>
                          </div>
                          
                          <div className="mt-8 grid grid-cols-2 gap-8">
                            <div>
                              <p className="text-[10px] uppercase tracking-widest text-black/40 font-bold">Equity Value</p>
                              <p className="mt-1 text-xl font-bold">{formatOg(position.exitValue)}</p>
                            </div>
                            <div>
                              <p className="text-[10px] uppercase tracking-widest text-black/40 font-bold">Claimable Yield</p>
                              <p className="mt-1 text-xl font-bold text-[var(--signal)]">{formatOg(position.claimable)}</p>
                            </div>
                          </div>

                          <button
                            onClick={() => void claimPosition(position)}
                            disabled={position.claimable === 0n || activeAction === `claim:${position.agent.id}`}
                            className="mt-8 w-full rounded-xl border border-black py-3 text-xs font-bold transition hover:bg-black hover:text-white disabled:opacity-20"
                          >
                            {activeAction === `claim:${position.agent.id}` ? "Claiming..." : "Withdraw Dividends"}
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                  <PaginationRow
                    page={positionPage}
                    pageCount={positionPageCount}
                    itemCount={sortedPositions.length}
                    onPrev={() => setPositionPage((p) => Math.max(1, p - 1))}
                    onNext={() => setPositionPage((p) => Math.min(positionPageCount, p + 1))}
                  />
                </div>
              )}

              {/* Tab Content: Following */}
              {activeTab === "following" && (
                <div className="space-y-6 animate-in fade-in slide-in-from-bottom-2">
                  {followingQuery.data?.length === 0 ? (
                    <div className="rounded-2xl border-2 border-dashed border-black/10 py-20 text-center">
                      <Users2 size={40} className="mx-auto text-black/10" />
                      <p className="mt-4 text-lg font-medium text-black/40">You are not following any agents yet.</p>
                    </div>
                  ) : (
                    <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                      {pagedFollowing.map((agent) => (
                        <Link 
                          key={agent.id} 
                          href={agentProfilePath(agent)}
                          className="group rounded-2xl border border-black/10 bg-white p-5 transition hover:border-black/30"
                        >
                          <div className="flex items-center gap-3">
                            <div className="size-8 rounded-full bg-black/5 grid place-items-center group-hover:bg-[var(--signal)]/10 transition">
                              <Bot size={16} />
                            </div>
                            <span className="font-bold">{shorten(agent.id)}</span>
                          </div>
                          <p className="mt-3 text-sm text-black/50 line-clamp-2 leading-relaxed">
                            {agent.personalitySummary}
                          </p>
                        </Link>
                      ))}
                    </div>
                  )}
                  <PaginationRow
                    page={followingPage}
                    pageCount={followingPageCount}
                    itemCount={(followingQuery.data ?? []).length}
                    onPrev={() => setFollowingPage((p) => Math.max(1, p - 1))}
                    onNext={() => setFollowingPage((p) => Math.min(followingPageCount, p + 1))}
                  />
                </div>
              )}
            </section>
          </div>
        )}
      </div>
    </main>
  );
}

function StatusPill({ state }: { state: any }) {
  return (
    <span className={`px-2 py-1 text-[9px] font-black uppercase tracking-widest rounded ${state.surface}`}>
      {state.label}
    </span>
  );
}

function Metric({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon: React.ReactNode;
}) {
  return (
    <div className="flex min-w-[200px] flex-1 items-center justify-between rounded-2xl border border-black/10 bg-white p-5">
      <div>
        <p className="text-[10px] uppercase tracking-widest text-black/40 font-bold">{label}</p>
        <p className="mt-2 text-2xl font-bold tracking-tight text-[var(--ink)]">{value}</p>
      </div>
      <div className="text-black/20">{icon}</div>
    </div>
  );
}

function DashboardStat({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div>
      <p className="text-[9px] uppercase tracking-widest text-black/30 font-bold">{label}</p>
      <p className={`mt-1 text-sm font-bold text-[var(--ink)] ${mono ? "font-mono" : ""}`}>
        {value}
      </p>
    </div>
  );
}

function PaginationRow({
  page,
  pageCount,
  itemCount,
  onPrev,
  onNext,
}: {
  page: number;
  pageCount: number;
  itemCount: number;
  onPrev: () => void;
  onNext: () => void;
}) {
  if (pageCount <= 1) return null;

  return (
    <div className="flex items-center justify-between py-4">
      <p className="text-xs text-black/40 font-bold uppercase tracking-widest">
        Page {page} of {pageCount}
      </p>
      <div className="flex items-center gap-1">
        <button
          onClick={onPrev}
          disabled={page <= 1}
          className="grid size-9 place-items-center rounded-lg border border-black/10 bg-white transition hover:bg-black hover:text-white disabled:opacity-20"
        >
          <ChevronLeft size={16} />
        </button>
        <button
          onClick={onNext}
          disabled={page >= pageCount}
          className="grid size-9 place-items-center rounded-lg border border-black/10 bg-white transition hover:bg-black hover:text-white disabled:opacity-20"
        >
          <ChevronRight size={16} />
        </button>
      </div>
    </div>
  );
}
