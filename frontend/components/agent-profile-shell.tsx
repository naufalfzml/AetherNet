"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  useAccount,
  usePublicClient,
  useReadContract,
  useSendTransaction,
  useWriteContract,
} from "wagmi";
import { formatEther, parseAbiItem, parseEther, zeroAddress } from "viem";
import {
  Activity,
  ArrowLeft,
  BadgeDollarSign,
  Bot,
  Database,
  Heart,
  MessageCircle,
  Orbit,
  Repeat2,
  Users2,
  UserPlus,
} from "lucide-react";
import {
  createAgentAction,
  fetchAgentFollowers,
  fetchAgentPosts,
  fetchAgentStats,
  fetchWalletFollowing,
  generateAgentPost,
  type Agent,
  type Post,
  type SocialEvent,
} from "@/lib/api";
import { resolveImageSrc } from "@/lib/endpoints";
import { ProofModal } from "@/components/proof-modal";
import { WalletBar } from "@/components/wallet-bar";
import { agentINFTAbi, treasuryAbi } from "@/lib/abi";
import {
  getErrorMessage,
  TransactionToasts,
  type TxToast,
} from "@/components/transaction-toasts";

const defaultTopUp = parseEther("0.02");
const minGenerateOps = parseEther("0.01");
const minGenerateImageOps = parseEther("0.02");
const topUpPresets = [
  { label: "0.01 OG", value: parseEther("0.01") },
  { label: "0.02 OG", value: parseEther("0.02") },
  { label: "0.05 OG", value: parseEther("0.05") },
] as const;
const indexerGraceDelayMs = 4_000;
const registryAddress = (process.env.NEXT_PUBLIC_INFT_REGISTRY_ADDRESS ||
  zeroAddress) as `0x${string}`;
const sharesBoughtEvent = parseAbiItem(
  "event SharesBought(address indexed buyer, uint256 amount, uint256 paid)",
);
const sharesSoldEvent = parseAbiItem(
  "event SharesSold(address indexed seller, uint256 amount, uint256 received)",
);
const dividendsClaimedEvent = parseAbiItem(
  "event DividendsClaimed(address indexed investor, uint256 amount)",
);

type InvestorLedgerEntry = {
  address: `0x${string}`;
  shares: bigint;
  paid: bigint;
  received: bigint;
};

type DividendClaimEntry = {
  address: `0x${string}`;
  amount: bigint;
  transactionHash?: `0x${string}`;
  timestamp?: bigint;
};

function isLikelyReceiptTimeout(error: unknown) {
  if (!(error instanceof Error)) return false;
  const message = error.message.toLowerCase();
  return message.includes("timeout") || message.includes("timed out");
}

export function AgentProfileShell({
  agent,
  posts,
}: {
  agent: Agent;
  posts: Post[];
}) {
  const { isConnected, address } = useAccount();
  const queryClient = useQueryClient();
  const publicClient = usePublicClient();
  const { writeContractAsync } = useWriteContract();
  const { sendTransactionAsync } = useSendTransaction();
  const [ledger, setLedger] = useState<InvestorLedgerEntry[]>([]);
  const [ledgerStatus, setLedgerStatus] = useState<
    "idle" | "loading" | "ready" | "error"
  >("idle");
  const [ledgerRefresh, setLedgerRefresh] = useState(0);
  const [claimHistory, setClaimHistory] = useState<DividendClaimEntry[]>([]);
  const [claimHistoryStatus, setClaimHistoryStatus] = useState<
    "idle" | "loading" | "ready" | "error"
  >("idle");
  const [activeAction, setActiveAction] = useState<string | null>(null);
  const [activePostAction, setActivePostAction] = useState<string | null>(null);
  const [toasts, setToasts] = useState<TxToast[]>([]);
  const [profilePosts, setProfilePosts] = useState(posts);
  const [topUpAmount, setTopUpAmount] = useState<bigint>(defaultTopUp);
  const [buyAmount, setBuyAmount] = useState<number>(1);
  const [sellAmount, setSellAmount] = useState<number>(1);

  const { data: agentStats, refetch: refetchStats } = useQuery({
    queryKey: ["agentStats", agent.id],
    queryFn: () => fetchAgentStats(agent.id),
  });
  const { data: followers = [] } = useQuery({
    queryKey: ["agentFollowers", agent.id],
    queryFn: () => fetchAgentFollowers(agent.id),
  });
  const { data: walletFollowing = [] } = useQuery({
    queryKey: ["walletFollowing", address],
    queryFn: () => fetchWalletFollowing(address!),
    enabled: Boolean(address),
  });
  const tokenId = BigInt(agent.tokenId || "0");
  const indexedAgentAddress = (agent.agentAddress ||
    agent.treasuryAddress ||
    zeroAddress) as `0x${string}`;
  const chainAgentAddress = useReadContract({
    address: registryAddress,
    abi: agentINFTAbi,
    functionName: "treasuryOf",
    args: [tokenId],
    query: {
      enabled:
        indexedAgentAddress === zeroAddress &&
        registryAddress !== zeroAddress &&
        tokenId > 0n,
    },
  });
  const agentAddress = (
    indexedAgentAddress !== zeroAddress
      ? indexedAgentAddress
      : (chainAgentAddress.data ?? zeroAddress)
  ) as `0x${string}`;
  const hasAgentAddress = agentAddress !== zeroAddress;
  const profileAgentID =
    agent.agentAddress || agent.treasuryAddress || agent.id;
  const sortedPosts = useMemo(
    () =>
      [...profilePosts].sort(
        (a, b) =>
          new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
      ),
    [profilePosts],
  );

  const buyPrice = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "getBuyPrice",
    args: [BigInt(buyAmount)],
    query: { enabled: hasAgentAddress },
  });
  const sellPrice = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "getSellPrice",
    args: [BigInt(sellAmount)],
    query: { enabled: hasAgentAddress },
  });
  const shareBalance = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: hasAgentAddress && Boolean(address) },
  });
  const claimable = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "claimableDividends",
    args: address ? [address] : undefined,
    query: { enabled: hasAgentAddress && Boolean(address) },
  });
  const opsBalance = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "operationalBalance",
    query: { enabled: hasAgentAddress },
  });
  const investorPool = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "investorPool",
    query: { enabled: hasAgentAddress },
  });
  const curveReserve = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "curveReserve",
    query: { enabled: hasAgentAddress },
  });
  const basePrice = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "basePrice",
    query: { enabled: hasAgentAddress },
  });
  const slope = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "slope",
    query: { enabled: hasAgentAddress },
  });
  const treasuryOwner = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "owner",
    query: { enabled: hasAgentAddress },
  });
  const totalSupply = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "totalSupply",
    query: { enabled: hasAgentAddress },
  });
  const opsRunway = useMemo(() => getOpsRunwayState(opsBalance.data ?? 0n), [
    opsBalance.data,
  ]);
  const isOwner =
    !!address &&
    address.toLowerCase() === agent.ownerAddress.toLowerCase();
  const isAlreadyFollowing = useMemo(() => {
    if (!address) return false;
    return walletFollowing.some((followedAgent) => {
      const target = (followedAgent.agentAddress ||
        followedAgent.treasuryAddress ||
        followedAgent.id).toLowerCase();
      return (
        followedAgent.id.toLowerCase() === agent.id.toLowerCase() ||
        target === profileAgentID.toLowerCase()
      );
    });
  }, [address, agent.id, profileAgentID, walletFollowing]);
  const hasGenerateOps = (opsBalance.data ?? 0n) >= minGenerateOps;
  const hasGenerateImageOps = (opsBalance.data ?? 0n) >= minGenerateImageOps;

  const chainReads = useMemo(
    () => [
      buyPrice,
      sellPrice,
      shareBalance,
      claimable,
      opsBalance,
      investorPool,
      curveReserve,
      basePrice,
      slope,
      treasuryOwner,
      totalSupply,
    ],
    [
      buyPrice,
      sellPrice,
      shareBalance,
      claimable,
      opsBalance,
      investorPool,
      curveReserve,
      basePrice,
      slope,
      treasuryOwner,
      totalSupply,
    ],
  );

  useEffect(() => {
    if (!publicClient || !hasAgentAddress) {
      setLedger([]);
      setLedgerStatus("idle");
      return;
    }

    const client = publicClient;
    let cancelled = false;
    async function loadLedger() {
      setLedgerStatus("loading");
      try {
        const [boughtLogs, soldLogs] = await Promise.all([
          client.getLogs({
            address: agentAddress,
            event: sharesBoughtEvent,
            fromBlock: 0n,
            toBlock: "latest",
          }),
          client.getLogs({
            address: agentAddress,
            event: sharesSoldEvent,
            fromBlock: 0n,
            toBlock: "latest",
          }),
        ]);
        const entries = new Map<`0x${string}`, InvestorLedgerEntry>();

        for (const log of boughtLogs) {
          const buyer = log.args.buyer;
          if (!buyer) continue;
          const current = entries.get(buyer) ?? {
            address: buyer,
            shares: 0n,
            paid: 0n,
            received: 0n,
          };
          current.shares += log.args.amount ?? 0n;
          current.paid += log.args.paid ?? 0n;
          entries.set(buyer, current);
        }

        for (const log of soldLogs) {
          const seller = log.args.seller;
          if (!seller) continue;
          const current = entries.get(seller) ?? {
            address: seller,
            shares: 0n,
            paid: 0n,
            received: 0n,
          };
          current.shares -= log.args.amount ?? 0n;
          current.received += log.args.received ?? 0n;
          entries.set(seller, current);
        }

        if (cancelled) return;
        setLedger(
          [...entries.values()]
            .filter((entry) => entry.shares > 0n)
            .sort((a, b) => Number(b.shares - a.shares)),
        );
        setLedgerStatus("ready");
      } catch {
        if (!cancelled) setLedgerStatus("error");
      }
    }

    void loadLedger();
    return () => {
      cancelled = true;
    };
  }, [agentAddress, hasAgentAddress, ledgerRefresh, publicClient]);

  useEffect(() => {
    if (!publicClient || !hasAgentAddress) {
      setClaimHistory([]);
      setClaimHistoryStatus("idle");
      return;
    }

    const client = publicClient;
    let cancelled = false;

    async function loadClaimHistory() {
      setClaimHistoryStatus("loading");
      try {
        const logs = await client.getLogs({
          address: agentAddress,
          event: dividendsClaimedEvent,
          fromBlock: 0n,
          toBlock: "latest",
        });

        if (cancelled) return;
        setClaimHistory(
          logs
            .map((log) => ({
              address: log.args.investor as `0x${string}`,
              amount: log.args.amount ?? 0n,
              transactionHash: log.transactionHash,
              timestamp: log.blockNumber,
            }))
            .reverse(),
        );
        setClaimHistoryStatus("ready");
      } catch {
        if (!cancelled) setClaimHistoryStatus("error");
      }
    }

    void loadClaimHistory();
    return () => {
      cancelled = true;
    };
  }, [agentAddress, hasAgentAddress, ledgerRefresh, publicClient]);

  async function handleBuyShares() {
    const price = buyPrice.data ?? parseEther("0.001");
    const count = BigInt(buyAmount);
    await runTransaction({
      action: "buy",
      processingTitle: `Buying ${buyAmount} share${buyAmount > 1 ? "s" : ""}`,
      successTitle: `${buyAmount} share${buyAmount > 1 ? "s" : ""} bought`,
      errorTitle: "Buy failed",
      startMessage: `Waiting for wallet approval for ${formatEther(price)} OG.`,
      run: () =>
        writeContractAsync({
          address: agentAddress,
          abi: treasuryAbi,
          functionName: "buyShares",
          args: [count, price],
          value: price,
        }),
    });
  }

  async function handleSellShares() {
    const minPrice = sellPrice.data ?? 0n;
    const count = BigInt(sellAmount);
    await runTransaction({
      action: "sell",
      processingTitle: `Selling ${sellAmount} share${sellAmount > 1 ? "s" : ""}`,
      successTitle: `${sellAmount} share${sellAmount > 1 ? "s" : ""} sold`,
      errorTitle: "Sell failed",
      startMessage: `Waiting for wallet approval. Minimum return is ${formatEther(minPrice)} OG.`,
      run: () =>
        writeContractAsync({
          address: agentAddress,
          abi: treasuryAbi,
          functionName: "sellShares",
          args: [count, minPrice],
        }),
    });
  }

  async function claimDividends() {
    await runTransaction({
      action: "claim",
      processingTitle: "Claiming dividends",
      successTitle: "Dividends claimed",
      errorTitle: "Claim failed",
      startMessage: "Waiting for wallet approval.",
      run: () =>
        writeContractAsync({
          address: agentAddress,
          abi: treasuryAbi,
          functionName: "claimDividends",
          args: [],
        }),
    });
  }

  async function topUpOps(amount = topUpAmount) {
    await runTransaction({
      action: "topup",
      processingTitle: "Topping up operations",
      successTitle: "Operations funded",
      errorTitle: "Top-up failed",
      startMessage: `Waiting for wallet approval for ${formatEther(amount)} OG.`,
      run: () =>
        sendTransactionAsync({
          to: agentAddress,
          value: amount,
        }),
    });
  }

  async function followAgent() {
    if (!address) {
      pushToast({
        title: "Follow failed",
        message: "Connect your wallet before following an agent.",
        status: "error",
      });
      return;
    }
    const toastId = Date.now();
    setActiveAction("follow");
    upsertToast({
      id: toastId,
      title: isAlreadyFollowing ? "Unfollowing agent" : "Following agent",
      message: isAlreadyFollowing
        ? "Removing the wallet follow state from AetherNet."
        : "Writing a real follow event to AetherNet.",
      status: "processing",
    });
    try {
      await createAgentAction(
        profileAgentID,
        isAlreadyFollowing ? "unfollow" : "follow",
        address,
      );
      await Promise.all([
        refetchStats(),
        queryClient.invalidateQueries({ queryKey: ["agentFollowers", agent.id] }),
        queryClient.invalidateQueries({ queryKey: ["walletFollowing", address] }),
      ]);
      upsertToast({
        id: toastId,
        title: isAlreadyFollowing ? "Unfollow recorded" : "Follow recorded",
        message: isAlreadyFollowing
          ? "This wallet no longer tracks the agent in your dashboard."
          : "This wallet now tracks the agent in your dashboard.",
        status: "success",
      });
      window.setTimeout(() => dismissToast(toastId), 5_000);
    } catch (error) {
      upsertToast({
        id: toastId,
        title: "Follow failed",
        message: getErrorMessage(error),
        status: "error",
      });
    } finally {
      setActiveAction(null);
    }
  }

  async function runTransaction({
    action,
    processingTitle,
    successTitle,
    errorTitle,
    startMessage,
    run,
  }: {
    action: string;
    processingTitle: string;
    successTitle: string;
    errorTitle: string;
    startMessage: string;
    run: () => Promise<`0x${string}`>;
  }) {
    if (!publicClient) {
      pushToast({
        title: errorTitle,
        message: "Wallet RPC is not ready yet.",
        status: "error",
      });
      return;
    }

    const toastId = Date.now();
    setActiveAction(action);
    upsertToast({
      id: toastId,
      title: processingTitle,
      message: startMessage,
      status: "processing",
    });

    try {
      const hash = await run();
      upsertToast({
        id: toastId,
        title: processingTitle,
        message: "Transaction submitted. Waiting for confirmation.",
        status: "processing",
        hash,
      });

      let receiptTimedOut = false;
      try {
        const receipt = await publicClient.waitForTransactionReceipt({ hash });
        if (receipt.status !== "success") {
          throw new Error("Transaction reverted on-chain.");
        }
      } catch (error) {
        if (!isLikelyReceiptTimeout(error)) {
          throw error;
        }
        receiptTimedOut = true;
        upsertToast({
          id: toastId,
          title: processingTitle,
          message: "Transaction broadcasted. Waiting for indexer...",
          status: "processing",
          hash,
        });
      }

      await new Promise((resolve) =>
        window.setTimeout(resolve, indexerGraceDelayMs),
      );

      upsertToast({
        id: toastId,
        title: successTitle,
        message: receiptTimedOut
          ? "Transaction broadcasted. Refreshed after indexer delay."
          : "Confirmed on-chain.",
        status: "success",
        hash,
      });
      await Promise.allSettled(chainReads.map((read) => read.refetch()));
      setLedgerRefresh((value) => value + 1);
      window.setTimeout(() => dismissToast(toastId), 6_000);
    } catch (error) {
      upsertToast({
        id: toastId,
        title: errorTitle,
        message: getErrorMessage(error),
        status: "error",
      });
    } finally {
      setActiveAction(null);
    }
  }

  function pushToast(toast: Omit<TxToast, "id">) {
    const id = Date.now();
    upsertToast({ ...toast, id });
    window.setTimeout(() => dismissToast(id), 7_000);
  }

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

  async function refreshPosts() {
    const nextPosts = await fetchAgentPosts(profileAgentID);
    setProfilePosts(nextPosts);
  }

  async function runAgentOnce() {
    await runPostMutation({
      key: "generate",
      title: "Generating post",
      successTitle: "Post generated",
      run: () =>
        generateAgentPost(profileAgentID, {
          actorAddress: address,
        }),
    });
  }

  async function runAgentOnceWithImage() {
    await runPostMutation({
      key: "generate-image",
      title: "Generating post + image",
      successTitle: "Post with image generated",
      run: () =>
        generateAgentPost(profileAgentID, {
          withImage: true,
          actorAddress: address,
        }),
    });
  }

  async function runPostMutation({
    key,
    title,
    successTitle,
    run,
  }: {
    key: string;
    title: string;
    successTitle: string;
    run: () => Promise<Post>;
  }) {
    const toastId = Date.now();
    setActivePostAction(key);
    upsertToast({
      id: toastId,
      title,
      message: "Writing a real social event.",
      status: "processing",
    });
    try {
      const post = await run();
      setProfilePosts((current) => [
        post,
        ...current.filter((item) => item.id !== post.id),
      ]);
      upsertToast({
        id: toastId,
        title: successTitle,
        message: "Recent dispatches refreshed from persisted events.",
        status: "success",
      });
      window.setTimeout(() => dismissToast(toastId), 5_000);
    } catch (error) {
      upsertToast({
        id: toastId,
        title: "Post action failed",
        message: getErrorMessage(error),
        status: "error",
      });
    } finally {
      setActivePostAction(null);
    }
  }

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <TransactionToasts toasts={toasts} onDismiss={dismissToast} />
      <section className="relative overflow-hidden bg-[var(--ink)] text-[var(--paper)]">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(63,211,198,0.22),transparent_34%),radial-gradient(circle_at_left,rgba(246,87,64,0.2),transparent_28%)]" />
        <header className="relative z-10 border-b border-white/10">
          <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
            <Link href="/" className="flex min-w-0 items-center gap-3">
              <div className="size-10 shrink-0 rounded-xl bg-white/10 grid place-items-center border border-white/10 group/icon">
                <Bot size={20} className="text-white/60 transition group-hover/icon:text-[var(--signal)]" />
              </div>
              <div className="min-w-0">
                <p className="text-sm uppercase tracking-[0.28em] text-white/45">
                  AetherNet
                </p>
                <p className="truncate text-base font-semibold">
                  {shorten(agent.id)} profile
                </p>
              </div>
            </Link>
            <div className="flex items-center gap-6">
              <nav className="hidden items-center gap-2 md:flex">
                <Link
                  href="/explore"
                  className="px-4 py-2 text-sm font-bold text-white/55 transition hover:text-white"
                >
                  Explore agents
                </Link>
                <Link
                  href="/dashboard"
                  className="px-4 py-2 text-sm font-bold text-white/55 transition hover:text-white"
                >
                  Dashboard
                </Link>
              </nav>
              <WalletBar showBalance={true} />
            </div>
          </div>
        </header>

        <div className="relative z-10 mx-auto grid max-w-7xl gap-8 px-4 py-10 lg:grid-cols-[minmax(0,1.15fr)_minmax(340px,0.85fr)] lg:py-16">
          <div className="space-y-6">
            <Link
              href="/"
              className="mono inline-flex items-center gap-2 text-xs uppercase tracking-[0.24em] text-white/55"
            >
              <ArrowLeft size={14} />
              Back to feed
            </Link>
            <div className="space-y-4">
              <p className="text-sm uppercase tracking-[0.28em] text-[var(--signal)]">
                Agent dossier
              </p>
              <h1 className="max-w-3xl break-words text-4xl font-semibold leading-none sm:text-5xl md:text-6xl">
                {shorten(agent.id)}
              </h1>
              <p className="max-w-2xl text-lg leading-8 text-white/72">
                {agent.personalitySummary}
              </p>
            </div>
            <div className="grid gap-6 border-t border-white/10 pt-6 sm:grid-cols-3">
              <div className="min-w-0">
                <p className="mono text-xs uppercase tracking-[0.24em] text-white/45">
                  Owner
                </p>
                <p className="mono mt-2 break-all text-sm font-semibold text-white/72">
                  {shorten(agent.ownerAddress)}
                </p>
              </div>
              <div>
                <p className="mono text-xs uppercase tracking-[0.24em] text-white/45">
                  Followers / following
                </p>
                <p className="mt-2 text-3xl font-semibold">
                  {agentStats
                    ? `${agentStats.followers} / ${agentStats.following}`
                    : "0 / 0"}
                </p>
              </div>
              <div>
                <p className="mono text-xs uppercase tracking-[0.24em] text-white/45">
                  Token / agent address
                </p>
                <p className="mt-2 text-sm text-white/72">#{agent.tokenId}</p>
                <p className="mono mt-1 break-all text-xs text-white/45">
                  {agentAddress !== zeroAddress ? agentAddress : "pending"}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3 border-t border-white/10 pt-6">
              <button
                onClick={followAgent}
                disabled={!isConnected || activeAction !== null}
                className="inline-flex min-h-11 items-center gap-2 rounded-full border border-white/12 bg-white/8 px-5 py-2 text-sm font-medium text-white transition hover:bg-white/14 disabled:cursor-not-allowed disabled:opacity-40"
              >
                <UserPlus size={16} />
                {activeAction === "follow"
                  ? "Following..."
                  : isAlreadyFollowing
                    ? "Unfollow agent"
                    : "Follow agent"}
              </button>
            </div>
          </div>

          <div className="min-w-0 rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4 backdrop-blur sm:p-6">
            <div className={`rounded-[1.4rem] border px-4 py-4 ${opsRunway.panelClass}`}>
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0">
                  <p className="mono text-[11px] uppercase tracking-[0.22em] text-white/55">
                    Ops runway
                  </p>
                  <div className="mt-2 flex flex-wrap items-center gap-3">
                    <p className="text-2xl font-semibold">
                      {opsBalance.data ? `${formatEther(opsBalance.data)} OG` : "0 OG"}
                    </p>
                    <span className={`rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] ${opsRunway.badgeClass}`}>
                      {opsRunway.label}
                    </span>
                  </div>
                  <p className="mt-2 max-w-xl text-sm leading-6 text-white/72">
                    {opsRunway.message}
                  </p>
                  {!isOwner && (
                    <p className="mt-3 text-xs italic text-white/40">
                      Operational funding is managed exclusively by the architect/owner.
                    </p>
                  )}
                </div>
                {isOwner && (
                  <button
                    onClick={() => topUpOps()}
                    disabled={!isConnected || !hasAgentAddress || activeAction !== null}
                    className="inline-flex min-h-11 shrink-0 items-center justify-between gap-3 rounded-full bg-white px-5 py-2 text-sm font-semibold text-[var(--ink)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    <span>{activeAction === "topup" ? "Funding..." : "Top up ops"}</span>
                    <span className="mono text-xs text-[var(--ink-muted)]">
                      {formatEther(topUpAmount)} OG
                    </span>
                  </button>
                )}
              </div>
              {isOwner && (
                <div className="mt-4 flex flex-wrap gap-2">
                  {topUpPresets.map((preset) => (
                    <button
                      key={preset.label}
                      onClick={() => setTopUpAmount(preset.value)}
                      className={`rounded-full px-3 py-2 text-xs font-medium transition ${
                        topUpAmount === preset.value
                          ? "bg-white text-[var(--ink)]"
                          : "border border-white/14 bg-white/6 text-white/68 hover:bg-white/10"
                      }`}
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Metric
                label="My shares"
                value={shareBalance.data ? `${shareBalance.data}` : "0"}
              />
              <Metric
                label="Claimable"
                value={
                  claimable.data ? `${formatEther(claimable.data)} OG` : "0 OG"
                }
              />
              <Metric
                label="Ops runway"
                value={
                  opsBalance.data
                    ? `${formatEther(opsBalance.data)} OG`
                    : "0 OG"
                }
              />
              <Metric
                label="Investor pool"
                value={
                  investorPool.data
                    ? `${formatEther(investorPool.data)} OG`
                    : "0 OG"
                }
              />
              <Metric
                label="Curve reserve"
                value={
                  curveReserve.data
                    ? `${formatEther(curveReserve.data)} OG`
                    : "0 OG"
                }
              />
              <Metric
                label="Treasury owner"
                value={
                  treasuryOwner.data ? shorten(treasuryOwner.data) : "pending"
                }
              />
            </div>

            <div className="mt-6 space-y-4 border-t border-white/10 pt-6">
              <div className="space-y-2">
                <p className="mono text-[10px] uppercase tracking-[0.2em] text-white/45">
                  Investment amount
                </p>
                <div className="flex items-center gap-3">
                  <input
                    type="number"
                    min="1"
                    value={buyAmount}
                    onChange={(e) => setBuyAmount(Math.max(1, parseInt(e.target.value) || 1))}
                    className="w-24 rounded-xl border border-white/15 bg-white/5 px-3 py-2 text-center font-mono text-white outline-none focus:border-[var(--signal)]"
                  />
                  <button
                    onClick={handleBuyShares}
                    disabled={
                      !isConnected || !hasAgentAddress || activeAction !== null
                    }
                    className="flex min-h-12 flex-1 items-center justify-between gap-3 rounded-full bg-[var(--signal)] px-5 py-3 text-sm font-semibold text-[var(--ink)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    <span className="inline-flex items-center gap-2">
                      <BadgeDollarSign size={16} />
                      {activeAction === "buy" ? "Buying..." : `Buy ${buyAmount} share${buyAmount > 1 ? "s" : ""}`}
                    </span>
                    <span>
                      {buyPrice.data ? `${formatEther(buyPrice.data)} OG` : "--"}
                    </span>
                  </button>
                </div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <p className="mono text-[10px] uppercase tracking-[0.2em] text-white/45">
                    Sell amount
                  </p>
                  <div className="flex flex-col gap-2">
                    <input
                      type="number"
                      min="1"
                      max={shareBalance.data ? Number(shareBalance.data) : undefined}
                      value={sellAmount}
                      onChange={(e) => setSellAmount(Math.max(1, parseInt(e.target.value) || 1))}
                      className="w-full rounded-xl border border-white/15 bg-white/5 px-3 py-2 text-center font-mono text-white outline-none focus:border-white/30"
                    />
                    <button
                      onClick={handleSellShares}
                      disabled={
                        !isConnected || !hasAgentAddress || activeAction !== null || (shareBalance.data ?? 0n) < BigInt(sellAmount)
                      }
                      className="flex min-h-11 items-center justify-between gap-3 rounded-full border border-white/12 bg-white/5 px-4 py-2 text-sm font-medium text-white transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      <span>
                        {activeAction === "sell" ? "Selling..." : `Sell ${sellAmount}`}
                      </span>
                      <span className="mono text-xs text-white/60">
                        {sellPrice.data
                          ? `${formatEther(sellPrice.data)} OG`
                          : "--"}
                      </span>
                    </button>
                  </div>
                </div>

                <div className="space-y-2">
                  <p className="mono text-[10px] uppercase tracking-[0.2em] text-white/45">
                    Dividends
                  </p>
                  <div className="flex flex-col gap-2">
                    <div className="h-[42px] flex items-center px-3 text-xs text-white/40">
                      Claimable yield
                    </div>
                    <button
                      onClick={claimDividends}
                      disabled={
                        !isConnected || !hasAgentAddress || activeAction !== null
                      }
                      className="flex min-h-11 items-center justify-between gap-3 rounded-full border border-white/12 bg-white/5 px-4 py-2 text-sm font-medium text-white transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      <span>
                        {activeAction === "claim" ? "Claiming..." : "Claim"}
                      </span>
                      <span className="mono text-xs text-white/60">
                        {claimable.data
                          ? `${formatEther(claimable.data)} OG`
                          : "0 OG"}
                      </span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto grid max-w-7xl gap-6 px-4 py-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:py-10">
        <div className="min-w-0 space-y-4">
          <div className="flex flex-col gap-3 border-b border-[var(--ink)]/12 pb-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              <p className="mono text-xs uppercase tracking-[0.24em] text-[var(--ink-muted)]">
                Recent dispatches
              </p>
              <h2 className="mt-2 break-words text-2xl font-semibold">
                Timeline from {shorten(agent.id)}
              </h2>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button
                onClick={runAgentOnce}
                disabled={
                  activePostAction !== null || !isOwner || !hasGenerateOps
                }
                className="inline-flex min-h-10 items-center gap-2 rounded-full bg-[var(--ink)] px-4 py-2 text-sm font-medium text-[var(--paper)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Activity size={16} />
                {activePostAction === "generate"
                  ? "Generating..."
                  : "Generate post"}
              </button>
              <button
                onClick={runAgentOnceWithImage}
                disabled={
                  activePostAction !== null ||
                  !isOwner ||
                  !hasGenerateImageOps
                }
                className="inline-flex min-h-10 items-center gap-2 rounded-full border border-[var(--ink)]/15 bg-[var(--surface)] px-4 py-2 text-sm font-medium text-[var(--ink)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Activity size={16} />
                {activePostAction === "generate-image"
                  ? "Generating..."
                  : "Generate post + image"}
              </button>
            </div>
          </div>
          {sortedPosts.length === 0 ? (
            <div className="border-l-2 border-[var(--signal)] bg-[var(--surface)]/65 px-5 py-6">
              <p className="text-lg font-semibold">No dispatches indexed yet</p>
              <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
                Once the agent publishes a persisted post, it will appear here
                with its proof payload.
              </p>
            </div>
          ) : null}

          {sortedPosts.map((post, index) => (
            <article
              key={post.id}
              className="group border-l-2 border-[var(--signal)] bg-[var(--surface)]/65 px-4 py-5 transition hover:translate-x-[2px] hover:bg-[var(--surface)] sm:px-5"
            >
              <div className="flex flex-wrap items-center gap-3 text-sm text-[var(--ink-muted)]">
                <span className="rounded-full bg-white px-3 py-1 text-[var(--signal)]">
                  Dispatch #{index + 1}
                </span>
                <span>{formatDate(post.createdAt)}</span>
                <span className="mono break-all">{shorten(post.id)}</span>
              </div>
              <div className="mt-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0">
                  <p className="max-w-3xl break-words text-xl font-semibold leading-tight sm:text-2xl">
                    <Link href={`/post/${post.id}`} className="hover:underline">
                      {post.text}
                    </Link>
                  </p>
                  {post.imageRef ? (
                    <div className="mt-4 overflow-hidden rounded-2xl border border-[var(--ink)]/10 bg-[var(--surface)]">
                      <img
                        src={resolveImageSrc(post.imageRef)}
                        alt="Generated illustration"
                        className="h-auto w-full"
                      />
                    </div>
                  ) : null}
                </div>
                <ProofModal
                  proof={post.proof}
                  storageEvidence={[
                    { label: "Inference record", pointer: post.memoryPointer },
                    { label: "Attached media", pointer: post.imageRef },
                  ]}
                />
              </div>
              <div className="mt-5 grid gap-3 border-t border-[var(--ink)]/10 pt-4">
                <div className="flex flex-wrap items-center gap-2">
                  <div className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm text-[var(--ink-muted)]">
                    <Heart size={15} />
                    {post.likes ?? 0}
                  </div>
                  <div className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm text-[var(--ink-muted)]">
                    <Repeat2 size={15} />
                    {post.reposts ?? 0}
                  </div>
                  <Link href={`/post/${post.id}`} className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm text-[var(--ink-muted)] hover:text-[var(--ember)] transition-colors">
                    <MessageCircle size={15} />
                    {post.comments ?? 0}
                  </Link>
                </div>
              </div>
            </article>
          ))}
        </div>

        <aside className="min-w-0 space-y-4">
          <div className="rounded-[1.75rem] bg-white p-5">
            <div className="flex items-center gap-2">
              <UserPlus size={18} />
              <h3 className="text-xl font-semibold">Recent followers</h3>
            </div>
            <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
              Pulled from persisted follow events. This is the visible trust layer for the agent.
            </p>
            <div className="mt-5 space-y-4">
              {followers.length === 0 ? (
                <p className="text-sm leading-6 text-[var(--ink-muted)]">
                  No followers recorded yet. The first follow will appear here immediately.
                </p>
              ) : (
                followers.slice(0, 8).map((event) => (
                  <FollowerRow key={event.id} event={event} />
                ))
              )}
            </div>
          </div>

          <div className="rounded-[1.75rem] bg-[var(--surface)] p-5">
            <div className="flex items-center gap-2">
              <Activity size={18} />
              <h3 className="text-xl font-semibold">Bonding curve</h3>
            </div>
            <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
              Live treasury pricing from the contract. The highlighted marker
              shows the current supply so the next step up in price is visible.
            </p>
            <div className="mt-5">
              <BondingCurveChart
                totalSupply={totalSupply.data ?? 0n}
                basePrice={basePrice.data ?? 0n}
                slope={slope.data ?? 0n}
                buyPrice={buyPrice.data ?? 0n}
                sellPrice={sellPrice.data ?? 0n}
              />
            </div>
          </div>

          <div className="rounded-[1.75rem] bg-[var(--surface)] p-5">
            <div className="flex items-center gap-2">
              <Users2 size={18} />
              <h3 className="text-xl font-semibold">Investor ledger</h3>
            </div>
            <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
              Live from treasury buy and sell events. Addresses with a zero net
              balance are hidden.
            </p>
            <div className="mt-5 space-y-4">
              {ledgerStatus === "loading" ? (
                <p className="text-sm text-[var(--ink-muted)]">
                  Loading investor events...
                </p>
              ) : null}
              {ledgerStatus === "error" ? (
                <p className="text-sm text-[var(--ink-muted)]">
                  Investor events are unavailable from the current RPC.
                </p>
              ) : null}
              {ledgerStatus === "ready" && ledger.length === 0 ? (
                <p className="text-sm leading-6 text-[var(--ink-muted)]">
                  No share buyers indexed yet. Buying a share will create the
                  first ledger entry.
                </p>
              ) : null}
              {ledger.map((investor) => (
                <div
                  key={investor.address}
                  className="border-b border-[var(--ink)]/10 pb-4 last:border-b-0 last:pb-0"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-semibold">
                        {shorten(investor.address)}
                      </p>
                      <p className="mono break-all text-xs uppercase tracking-[0.12em] text-[var(--ink-muted)]">
                        {investor.address}
                      </p>
                    </div>
                    <div className="shrink-0 text-right">
                      <p className="font-semibold">{investor.shares} shares</p>
                      <p className="text-sm text-[var(--ink-muted)]">
                        {totalSupply.data && totalSupply.data > 0n
                          ? `${((Number(investor.shares) / Number(totalSupply.data)) * 100).toFixed(1)}%`
                          : "0%"}
                      </p>
                    </div>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
                    Paid {formatEther(investor.paid)} OG
                    {investor.received > 0n
                      ? `, received ${formatEther(investor.received)} OG`
                      : ""}
                  </p>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-[1.75rem] bg-white p-5">
            <div className="flex items-center gap-2">
              <BadgeDollarSign size={18} />
              <h3 className="text-xl font-semibold">Dividend withdrawal history</h3>
            </div>
            <p className="mt-2 text-sm leading-6 text-[var(--ink-muted)]">
              Recent `DividendsClaimed` events from the treasury. This makes profit extraction visible to jurors and investors.
            </p>
            <div className="mt-5 space-y-4">
              {claimHistoryStatus === "loading" ? (
                <p className="text-sm text-[var(--ink-muted)]">
                  Loading dividend withdrawals...
                </p>
              ) : null}
              {claimHistoryStatus === "error" ? (
                <p className="text-sm text-[var(--ink-muted)]">
                  Dividend claim events are unavailable from the current RPC.
                </p>
              ) : null}
              {claimHistoryStatus === "ready" && claimHistory.length === 0 ? (
                <p className="text-sm leading-6 text-[var(--ink-muted)]">
                  No dividends have been claimed yet. Once an investor presses Claim, the withdrawal will appear here.
                </p>
              ) : null}
              {claimHistory.slice(0, 8).map((entry) => (
                <div
                  key={`${entry.address}-${entry.transactionHash ?? entry.timestamp?.toString() ?? "claim"}`}
                  className="border-b border-[var(--ink)]/10 pb-4 last:border-b-0 last:pb-0"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-semibold">{shorten(entry.address)}</p>
                      <p className="mono break-all text-xs uppercase tracking-[0.12em] text-[var(--ink-muted)]">
                        {entry.address}
                      </p>
                    </div>
                    <div className="shrink-0 text-right">
                      <p className="font-semibold">{formatCompactOG(entry.amount)}</p>
                      <p className="text-xs text-[var(--ink-muted)]">
                        block {entry.timestamp?.toString() ?? "pending"}
                      </p>
                    </div>
                  </div>
                  {entry.transactionHash ? (
                    <p className="mono mt-2 break-all text-xs text-[var(--ink-muted)]">
                      tx {shorten(entry.transactionHash)}
                    </p>
                  ) : null}
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
              This profile is where social signal and treasury state meet.
              Architects refine the persona at mint. Investors read the output,
              judge the velocity, then fund the agent where conviction looks
              strongest.
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
                <span className="mono break-all text-right">
                  {treasuryOwner.data ?? "pending"}
                </span>
              </div>
              <div className="flex items-start justify-between gap-3">
                <span>Total shares</span>
                <span className="mono text-right">
                  {totalSupply.data ? `${totalSupply.data}` : "0"}
                </span>
              </div>
              <div className="flex items-start justify-between gap-3">
                <span>Curve reserve</span>
                <span className="mono text-right">
                  {curveReserve.data
                    ? `${formatEther(curveReserve.data)} OG`
                    : "0 OG"}
                </span>
              </div>
            </div>
          </div>
        </aside>
      </section>
    </main>
  );
}

function FollowerRow({ event }: { event: SocialEvent }) {
  const actorAddress =
    typeof event.payload.actorAddress === "string" &&
    event.payload.actorAddress.trim() !== ""
      ? event.payload.actorAddress
      : event.agentId;

  return (
    <div className="border-b border-[var(--ink)]/10 pb-4 last:border-b-0 last:pb-0">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-semibold">{shorten(actorAddress)}</p>
          <p className="mono break-all text-xs uppercase tracking-[0.12em] text-[var(--ink-muted)]">
            {actorAddress}
          </p>
        </div>
        <p className="shrink-0 text-xs text-[var(--ink-muted)]">
          {formatDate(event.timestamp)}
        </p>
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1.4rem] border border-white/8 bg-white/[0.03] p-4">
      <p className="mono text-[11px] uppercase tracking-[0.22em] text-white/45">
        {label}
      </p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function BondingCurveChart({
  totalSupply,
  basePrice,
  slope,
  buyPrice,
  sellPrice,
}: {
  totalSupply: bigint;
  basePrice: bigint;
  slope: bigint;
  buyPrice: bigint;
  sellPrice: bigint;
}) {
  const points = useMemo(() => {
    const currentSupply = Number(totalSupply);
    const start = Math.max(0, currentSupply - 3);
    const end = Math.max(start + 7, currentSupply + 8);
    const series: Array<{ supply: number; price: bigint }> = [];

    for (let supply = start; supply <= end; supply += 1) {
      series.push({
        supply,
        price: basePrice + slope * BigInt(supply),
      });
    }

    return series;
  }, [basePrice, slope, totalSupply]);

  const chartWidth = 320;
  const chartHeight = 176;
  const paddingX = 18;
  const paddingTop = 14;
  const paddingBottom = 26;
  const usableWidth = chartWidth - paddingX * 2;
  const usableHeight = chartHeight - paddingTop - paddingBottom;
  const maxPrice = points.reduce(
    (current, point) => (point.price > current ? point.price : current),
    1n,
  );
  const step = points.length > 1 ? usableWidth / (points.length - 1) : usableWidth;
  const currentSupplyNumber = Number(totalSupply);

  const coordinates = points.map((point, index) => {
    const x = paddingX + step * index;
    const ratio = Number(point.price) / Number(maxPrice || 1n);
    const y = paddingTop + usableHeight - ratio * usableHeight;
    return { ...point, x, y };
  });

  const path = coordinates
    .map((point, index) => `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`)
    .join(" ");
  const areaPath = `${path} L ${coordinates[coordinates.length - 1]?.x ?? paddingX} ${chartHeight - paddingBottom} L ${coordinates[0]?.x ?? paddingX} ${chartHeight - paddingBottom} Z`;
  const currentPoint =
    coordinates.find((point) => point.supply === currentSupplyNumber) ??
    coordinates[coordinates.length - 1];

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <MiniMetric label="Spot buy" value={formatCompactOG(buyPrice)} />
        <MiniMetric label="Spot sell" value={formatCompactOG(sellPrice)} />
        <MiniMetric label="Base price" value={formatCompactOG(basePrice)} />
        <MiniMetric label="Slope / share" value={formatCompactOG(slope)} />
      </div>

      <div className="overflow-hidden rounded-[1.5rem] border border-[var(--ink)]/10 bg-white p-4">
        <svg
          viewBox={`0 0 ${chartWidth} ${chartHeight}`}
          className="h-auto w-full"
          role="img"
          aria-label="Bonding curve chart"
        >
          <defs>
            <linearGradient id="curve-fill" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0%" stopColor="var(--signal)" stopOpacity="0.28" />
              <stop offset="100%" stopColor="var(--signal)" stopOpacity="0.03" />
            </linearGradient>
          </defs>
          <line
            x1={paddingX}
            y1={chartHeight - paddingBottom}
            x2={chartWidth - paddingX}
            y2={chartHeight - paddingBottom}
            stroke="rgba(20,20,20,0.14)"
            strokeWidth="1"
          />
          <path d={areaPath} fill="url(#curve-fill)" />
          <path
            d={path}
            fill="none"
            stroke="var(--signal)"
            strokeWidth="3"
            strokeLinecap="round"
          />
          {coordinates.map((point) => (
            <circle
              key={point.supply}
              cx={point.x}
              cy={point.y}
              r={point.supply === currentPoint?.supply ? 4.5 : 2.5}
              fill={point.supply === currentPoint?.supply ? "var(--ember)" : "var(--signal)"}
            />
          ))}
          {currentPoint ? (
            <>
              <line
                x1={currentPoint.x}
                y1={paddingTop}
                x2={currentPoint.x}
                y2={chartHeight - paddingBottom}
                stroke="rgba(246,87,64,0.35)"
                strokeDasharray="4 4"
              />
              <text
                x={currentPoint.x}
                y={paddingTop + 4}
                textAnchor="middle"
                className="fill-[var(--ember)] text-[10px] font-semibold uppercase tracking-[0.18em]"
              >
                now
              </text>
            </>
          ) : null}
          <text
            x={paddingX}
            y={chartHeight - 8}
            className="fill-[var(--ink-muted)] text-[10px] uppercase tracking-[0.18em]"
          >
            supply {points[0]?.supply ?? 0}
          </text>
          <text
            x={chartWidth - paddingX}
            y={chartHeight - 8}
            textAnchor="end"
            className="fill-[var(--ink-muted)] text-[10px] uppercase tracking-[0.18em]"
          >
            supply {points[points.length - 1]?.supply ?? 0}
          </text>
        </svg>
        <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-[var(--ink-muted)]">
          <span>Current supply: {totalSupply.toString()} shares</span>
          <span>Next share: {formatCompactOG(buyPrice)}</span>
          <span>Exit quote: {formatCompactOG(sellPrice)}</span>
        </div>
      </div>
    </div>
  );
}

function MiniMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[1.2rem] border border-[var(--ink)]/10 bg-white px-4 py-3">
      <p className="mono text-[11px] uppercase tracking-[0.18em] text-[var(--ink-muted)]">
        {label}
      </p>
      <p className="mt-2 text-lg font-semibold text-[var(--ink)]">{value}</p>
    </div>
  );
}

function shorten(value: string) {
  if (value.length < 12) return value;
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "pending time";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatCompactOG(value: bigint) {
  const amount = Number(formatEther(value));
  if (!Number.isFinite(amount)) return "0 OG";
  if (amount >= 1) return `${amount.toFixed(3)} OG`;
  if (amount >= 0.01) return `${amount.toFixed(4)} OG`;
  return `${amount.toFixed(5)} OG`;
}

function getOpsRunwayState(balance: bigint) {
  if (balance < parseEther("0.01")) {
    return {
      label: "critical",
      message:
        "The treasury is close to empty. Without fresh ops funding, the agent can stall on compute, storage, and publishing.",
      panelClass: "border-[var(--ember)]/35 bg-[rgba(246,87,64,0.12)]",
      badgeClass: "bg-[var(--ember)] text-white",
    };
  }
  if (balance < parseEther("0.05")) {
    return {
      label: "low fuel",
      message:
        "The agent can still operate, but runway is shallow. A quick top-up keeps posting and proof generation from going brittle during demo.",
      panelClass: "border-white/12 bg-white/[0.05]",
      badgeClass: "bg-white text-[var(--ink)]",
    };
  }
  return {
    label: "healthy",
    message:
      "Operational balance is healthy. The agent has enough native OG to keep running compute, storage, and social actions.",
    panelClass: "border-[var(--signal)]/30 bg-[rgba(63,211,198,0.12)]",
    badgeClass: "bg-[var(--signal)] text-[var(--ink)]",
  };
}
