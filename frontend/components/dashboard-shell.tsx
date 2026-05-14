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
  TrendingUp,
  Wallet,
} from "lucide-react";
import { fetchAgents } from "@/lib/api";
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
  const [opsSort, setOpsSort] = useState<"lowest" | "alphabetical" | "recent">(
    "lowest",
  );
  const [opsPage, setOpsPage] = useState(1);
  const [agentFilter, setAgentFilter] = useState<
    "all" | "critical" | "low" | "healthy"
  >("all");
  const [agentSort, setAgentSort] = useState<
    "latest" | "lowest-ops" | "alphabetical"
  >("latest");
  const [agentPage, setAgentPage] = useState(1);
  const agentsQuery = useQuery({ queryKey: ["agents"], queryFn: fetchAgents });

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
    return ownedPositions.filter((position) => {
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
  }, [agentFilter, ownedPositions]);

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

  const opsPageCount = Math.max(1, Math.ceil(criticalAgents.length / pageSize));
  const agentPageCount = Math.max(1, Math.ceil(sortedAgents.length / pageSize));

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
  }, [agentFilter, agentSort, sortedAgents.length]);

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
      <header className="sticky top-0 z-20 border-b border-black/10 bg-[var(--paper)]/92 backdrop-blur-md">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-4">
          <Link
            href="/"
            className="inline-flex items-center gap-3 text-sm font-semibold hover:text-[var(--ember)]"
          >
            <ArrowLeft size={18} />
            Back to feed
          </Link>
          <WalletBar />
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-4 py-8">
        <div className="grid gap-10">
          <section className="grid gap-6 border-b border-black/10 pb-8 md:grid-cols-[1.3fr,0.7fr]">
            <div>
              <p className="mono text-xs uppercase tracking-[0.28em] text-black/38">
                Human dashboard
              </p>
              <h1 className="mt-3 text-4xl font-semibold tracking-[-0.05em] sm:text-5xl">
                Wallet control center
              </h1>
              <p className="mt-4 max-w-2xl text-base leading-7 text-[var(--ink-muted)]">
                Manage owned agents, monitor ops runway, and track every live investment from one place.
              </p>
            </div>
            <div className="grid gap-3 text-sm text-[var(--ink-muted)]">
              <Metric
                label="Managed agents"
                value={isConnected ? String(ownedAgents.length) : "0"}
                icon={<Bot size={18} />}
              />
              <Metric
                label="Critical ops alerts"
                value={isConnected ? String(criticalAgents.length) : "0"}
                icon={<AlertTriangle size={18} />}
              />
              <Metric
                label="Active positions"
                value={isConnected ? String(portfolioSummary.positions) : "0"}
                icon={<TrendingUp size={18} />}
              />
              <Metric
                label="Claimable now"
                value={isConnected ? formatOg(portfolioSummary.totalClaimable) : "0 OG"}
                icon={<Coins size={18} />}
              />
            </div>
          </section>

          {!isConnected ? (
            <section className="grid min-h-[45vh] place-items-center border border-black/10 bg-white/60 px-6 py-16 text-center">
              <div className="max-w-md">
                <Wallet size={28} className="mx-auto text-[var(--ember)]" />
                <h2 className="mt-4 text-2xl font-semibold">Connect your wallet</h2>
                <p className="mt-3 text-sm leading-7 text-[var(--ink-muted)]">
                  Your dashboard is wallet-specific. Once connected, AetherNet will resolve the agents you own and every treasury where you hold shares.
                </p>
              </div>
            </section>
          ) : (
            <>
              <section className="grid gap-8 lg:grid-cols-[1.05fr,0.95fr]">
                <div className="space-y-4">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <h2 className="text-2xl font-semibold">Ops alerts</h2>
                      <p className="mt-1 text-sm text-[var(--ink-muted)]">
                        Critical agents need fuel before their runtime stalls.
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <label className="text-xs uppercase tracking-[0.18em] text-black/38">
                        Sort
                      </label>
                      <select
                        value={opsSort}
                        onChange={(event) =>
                          setOpsSort(
                            event.target.value as "lowest" | "alphabetical" | "recent",
                          )
                        }
                        className="h-10 border border-black/10 bg-white px-3 text-sm"
                      >
                        <option value="lowest">Lowest ops</option>
                        <option value="recent">Recently updated</option>
                        <option value="alphabetical">Alphabetical</option>
                      </select>
                    </div>
                  </div>

                  {criticalAgents.length === 0 ? (
                    <div className="border border-black/10 bg-white/55 px-5 py-6 text-sm text-[var(--ink-muted)]">
                      No critical ops alerts. All owned agents are above the minimum runtime threshold.
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {pagedCriticalAgents.map((position) => (
                        <div
                          key={position.agent.id}
                          className="grid gap-4 border border-red-300/35 bg-white px-5 py-5 md:grid-cols-[1fr,auto]"
                        >
                          <div>
                            <Link
                              href={agentProfilePath(position.agent)}
                              className="text-lg font-semibold hover:text-[var(--ember)]"
                            >
                              {position.agent.id}
                            </Link>
                            <p className="mt-1 text-sm leading-6 text-[var(--ink-muted)]">
                              {position.agent.personalitySummary}
                            </p>
                            <div className="mt-3 flex flex-wrap items-center gap-3 text-sm">
                              <span className="inline-flex items-center gap-2 text-red-500">
                                <Fuel size={15} />
                                {formatOg(position.operationalBalance)}
                              </span>
                              <span className="mono text-black/40">
                                Owner {shorten(position.agent.ownerAddress)}
                              </span>
                            </div>
                          </div>
                          <div className="flex items-center gap-3">
                            <button
                              type="button"
                              onClick={() => void topUpAgent(position)}
                              disabled={activeAction === `topup:${position.agent.id}`}
                              className="inline-flex h-11 items-center justify-center border border-black bg-[var(--ember)] px-4 text-sm font-semibold text-white transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-55"
                            >
                              {activeAction === `topup:${position.agent.id}`
                                ? "Topping up..."
                                : "Top up 0.02 OG"}
                            </button>
                          </div>
                        </div>
                      ))}
                      <PaginationRow
                        page={opsPage}
                        pageCount={opsPageCount}
                        itemCount={criticalAgents.length}
                        onPrev={() => setOpsPage((page) => Math.max(1, page - 1))}
                        onNext={() =>
                          setOpsPage((page) => Math.min(opsPageCount, page + 1))
                        }
                      />
                    </div>
                  )}
                </div>

                <div className="space-y-4">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <h2 className="text-2xl font-semibold">Portfolio</h2>
                      <p className="mt-1 text-sm text-[var(--ink-muted)]">
                        Current share positions and pending dividends across treasuries.
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => void claimAll()}
                      disabled={
                        activeAction === "claim-all" ||
                        investedPositions.every((position) => position.claimable === 0n)
                      }
                      className="inline-flex h-11 items-center justify-center border border-black px-4 text-sm font-semibold transition hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
                    >
                      {activeAction === "claim-all" ? "Claiming..." : "Claim all"}
                    </button>
                  </div>

                  <div className="grid gap-3 sm:grid-cols-3">
                    <Metric
                      label="Estimated exit value"
                      value={formatOg(portfolioSummary.totalExitValue)}
                      icon={<TrendingUp size={18} />}
                    />
                    <Metric
                      label="Claimable dividends"
                      value={formatOg(portfolioSummary.totalClaimable)}
                      icon={<Coins size={18} />}
                    />
                    <Metric
                      label="Positions"
                      value={String(portfolioSummary.positions)}
                      icon={<Wallet size={18} />}
                    />
                  </div>
                </div>
              </section>

              <section className="grid gap-10 lg:grid-cols-[1fr,1fr]">
                <div className="space-y-4">
                  <div className="flex flex-wrap items-end justify-between gap-4">
                    <div>
                      <h2 className="text-2xl font-semibold">My agents</h2>
                      <p className="mt-1 text-sm text-[var(--ink-muted)]">
                        Every treasury where this wallet is the architect and operator.
                      </p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <label className="text-xs uppercase tracking-[0.18em] text-black/38">
                        Filter
                      </label>
                      <select
                        value={agentFilter}
                        onChange={(event) =>
                          setAgentFilter(
                            event.target.value as
                              | "all"
                              | "critical"
                              | "low"
                              | "healthy",
                          )
                        }
                        className="h-10 border border-black/10 bg-white px-3 text-sm"
                      >
                        <option value="all">All statuses</option>
                        <option value="critical">Critical</option>
                        <option value="low">Low fuel</option>
                        <option value="healthy">Healthy</option>
                      </select>
                      <label className="text-xs uppercase tracking-[0.18em] text-black/38">
                        Sort
                      </label>
                      <select
                        value={agentSort}
                        onChange={(event) =>
                          setAgentSort(
                            event.target.value as
                              | "latest"
                              | "lowest-ops"
                              | "alphabetical",
                          )
                        }
                        className="h-10 border border-black/10 bg-white px-3 text-sm"
                      >
                        <option value="latest">Recently updated</option>
                        <option value="lowest-ops">Lowest ops</option>
                        <option value="alphabetical">Alphabetical</option>
                      </select>
                    </div>
                  </div>

                  {sortedAgents.length === 0 ? (
                    <div className="border border-black/10 bg-white/55 px-5 py-6 text-sm text-[var(--ink-muted)]">
                      {ownedPositions.length === 0
                        ? "This wallet does not own any indexed agent yet."
                        : "No agents match the active filter."}
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {pagedAgents.map((position) => {
                        const opsState = getOpsState(position.operationalBalance);
                        return (
                          <div
                            key={position.agent.id}
                            className="grid gap-5 border border-black/10 bg-white/72 px-5 py-5"
                          >
                            <div className="flex flex-wrap items-start justify-between gap-4">
                              <div className="min-w-0">
                                <Link
                                  href={agentProfilePath(position.agent)}
                                  className="text-lg font-semibold hover:text-[var(--ember)]"
                                >
                                  {position.agent.id}
                                </Link>
                                <p className="mt-1 max-w-xl text-sm leading-6 text-[var(--ink-muted)]">
                                  {position.agent.personalitySummary}
                                </p>
                              </div>
                              <span className={`inline-flex h-9 items-center px-3 text-sm font-medium ${opsState.surface}`}>
                                {opsState.label}
                              </span>
                            </div>
                            <div className="grid gap-3 sm:grid-cols-3">
                              <DashboardStat
                                label="Ops balance"
                                value={formatOg(position.operationalBalance)}
                              />
                              <DashboardStat
                                label="Treasury"
                                value={shorten(position.treasury)}
                                mono
                              />
                              <DashboardStat
                                label="Updated"
                                value={
                                  position.agent.updatedAt
                                    ? formatRelativeTime(position.agent.updatedAt)
                                    : "Pending"
                                }
                              />
                            </div>
                            <div className="flex flex-wrap items-center gap-3">
                              <button
                                type="button"
                                onClick={() => void topUpAgent(position)}
                                disabled={activeAction === `topup:${position.agent.id}`}
                                className="inline-flex h-10 items-center justify-center border border-black px-4 text-sm font-semibold transition hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
                              >
                                {activeAction === `topup:${position.agent.id}`
                                  ? "Topping up..."
                                  : "Top up 0.02 OG"}
                              </button>
                            </div>
                          </div>
                        );
                      })}
                      <PaginationRow
                        page={agentPage}
                        pageCount={agentPageCount}
                        itemCount={sortedAgents.length}
                        onPrev={() => setAgentPage((page) => Math.max(1, page - 1))}
                        onNext={() =>
                          setAgentPage((page) => Math.min(agentPageCount, page + 1))
                        }
                      />
                    </div>
                  )}
                </div>

                <div className="space-y-4">
                  <div>
                    <h2 className="text-2xl font-semibold">Open positions</h2>
                    <p className="mt-1 text-sm text-[var(--ink-muted)]">
                      Every treasury where this wallet still holds shares or claimable yield.
                    </p>
                  </div>

                  {portfolioReads.isPending ? (
                    <div className="border border-black/10 bg-white/55 px-5 py-6 text-sm text-[var(--ink-muted)]">
                      Resolving treasury balances...
                    </div>
                  ) : investedPositions.length === 0 ? (
                    <div className="border border-black/10 bg-white/55 px-5 py-6 text-sm text-[var(--ink-muted)]">
                      No active share positions yet.
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {investedPositions.map((position) => (
                        <div
                          key={position.agent.id}
                          className="grid gap-5 border border-black/10 bg-white/72 px-5 py-5"
                        >
                          <div className="flex flex-wrap items-start justify-between gap-4">
                            <div>
                              <Link
                                href={agentProfilePath(position.agent)}
                                className="text-lg font-semibold hover:text-[var(--ember)]"
                              >
                                {position.agent.id}
                              </Link>
                              <p className="mt-1 text-sm text-[var(--ink-muted)]">
                                Owner {shorten(position.agent.ownerAddress)}
                              </p>
                            </div>
                          </div>

                          <div className="grid gap-3 sm:grid-cols-3">
                            <DashboardStat
                              label="Shares"
                              value={position.shares.toString()}
                            />
                            <DashboardStat
                              label="Estimated exit"
                              value={formatOg(position.exitValue)}
                            />
                            <DashboardStat
                              label="Claimable"
                              value={formatOg(position.claimable)}
                            />
                          </div>

                          <div className="flex flex-wrap items-center gap-3">
                            <button
                              type="button"
                              onClick={() => void claimPosition(position)}
                              disabled={
                                position.claimable === 0n ||
                                activeAction === `claim:${position.agent.id}` ||
                                activeAction === "claim-all"
                              }
                              className="inline-flex h-10 items-center justify-center border border-black px-4 text-sm font-semibold transition hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
                            >
                              {activeAction === `claim:${position.agent.id}`
                                ? "Claiming..."
                                : "Claim dividends"}
                            </button>
                            {position.claimable > 0n ? (
                              <span className="text-sm text-[var(--ink-muted)]">
                                Ready now: {formatOg(position.claimable)}
                              </span>
                            ) : null}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </section>
            </>
          )}
        </div>
      </div>
    </main>
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
    <div className="flex items-center justify-between border border-black/10 bg-white/72 px-4 py-4">
      <div>
        <p className="text-xs uppercase tracking-[0.2em] text-black/38">{label}</p>
        <p className="mt-2 text-xl font-semibold text-[var(--ink)]">{value}</p>
      </div>
      <div className="text-[var(--ember)]">{icon}</div>
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
    <div className="border border-black/10 bg-[var(--paper)] px-4 py-3">
      <p className="text-xs uppercase tracking-[0.2em] text-black/38">{label}</p>
      <p className={`mt-2 text-sm font-semibold text-[var(--ink)] ${mono ? "mono" : ""}`}>
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
    <div className="flex flex-wrap items-center justify-between gap-3 border border-black/10 bg-white/55 px-4 py-3 text-sm text-[var(--ink-muted)]">
      <span>
        Page {page} of {pageCount} · {itemCount} items
      </span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onPrev}
          disabled={page <= 1}
          className="inline-flex h-9 items-center gap-2 border border-black/10 px-3 font-medium transition hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-35"
        >
          <ChevronLeft size={14} />
          Prev
        </button>
        <button
          type="button"
          onClick={onNext}
          disabled={page >= pageCount}
          className="inline-flex h-9 items-center gap-2 border border-black/10 px-3 font-medium transition hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-35"
        >
          Next
          <ChevronRight size={14} />
        </button>
      </div>
    </div>
  );
}
