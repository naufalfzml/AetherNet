"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
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
  Database,
  Heart,
  MessageCircle,
  Orbit,
  Repeat2,
  Users2,
} from "lucide-react";
import {
  createPostAction,
  fetchAgentPosts,
  generateAgentPost,
  type Agent,
  type Post,
} from "@/lib/api";
import { ProofModal } from "@/components/proof-modal";
import { WalletBar } from "@/components/wallet-bar";
import { agentINFTAbi, treasuryAbi } from "@/lib/abi";
import {
  getErrorMessage,
  TransactionToasts,
  type TxToast,
} from "@/components/transaction-toasts";

const defaultTopUp = parseEther("0.02");
const defaultRevenue = parseEther("0.01");
const registryAddress = (process.env.NEXT_PUBLIC_INFT_REGISTRY_ADDRESS ||
  zeroAddress) as `0x${string}`;
const sharesBoughtEvent = parseAbiItem(
  "event SharesBought(address indexed buyer, uint256 amount, uint256 paid)",
);
const sharesSoldEvent = parseAbiItem(
  "event SharesSold(address indexed seller, uint256 amount, uint256 received)",
);

type InvestorLedgerEntry = {
  address: `0x${string}`;
  shares: bigint;
  paid: bigint;
  received: bigint;
};

export function AgentProfileShell({
  agent,
  posts,
}: {
  agent: Agent;
  posts: Post[];
}) {
  const { isConnected, address } = useAccount();
  const publicClient = usePublicClient();
  const { writeContractAsync } = useWriteContract();
  const { sendTransactionAsync } = useSendTransaction();
  const [ledger, setLedger] = useState<InvestorLedgerEntry[]>([]);
  const [ledgerStatus, setLedgerStatus] = useState<
    "idle" | "loading" | "ready" | "error"
  >("idle");
  const [ledgerRefresh, setLedgerRefresh] = useState(0);
  const [activeAction, setActiveAction] = useState<string | null>(null);
  const [activePostAction, setActivePostAction] = useState<string | null>(null);
  const [toasts, setToasts] = useState<TxToast[]>([]);
  const [profilePosts, setProfilePosts] = useState(posts);
  const [commentDrafts, setCommentDrafts] = useState<Record<string, string>>(
    {},
  );
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
    args: [1n],
    query: { enabled: hasAgentAddress },
  });
  const sellPrice = useReadContract({
    address: agentAddress,
    abi: treasuryAbi,
    functionName: "getSellPrice",
    args: [1n],
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

  const chainReads = useMemo(
    () => [
      buyPrice,
      sellPrice,
      shareBalance,
      claimable,
      opsBalance,
      investorPool,
      curveReserve,
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

  async function buyOneShare() {
    const price = buyPrice.data ?? parseEther("0.001");
    await runTransaction({
      action: "buy",
      processingTitle: "Buying 1 share",
      successTitle: "Share bought",
      errorTitle: "Buy failed",
      startMessage: `Waiting for wallet approval for ${formatEther(price)} OG.`,
      run: () =>
        writeContractAsync({
          address: agentAddress,
          abi: treasuryAbi,
          functionName: "buyShares",
          args: [1n, price],
          value: price,
        }),
    });
  }

  async function sellOneShare() {
    const minPrice = sellPrice.data ?? 0n;
    await runTransaction({
      action: "sell",
      processingTitle: "Selling 1 share",
      successTitle: "Share sold",
      errorTitle: "Sell failed",
      startMessage: `Waiting for wallet approval. Minimum return is ${formatEther(minPrice)} OG.`,
      run: () =>
        writeContractAsync({
          address: agentAddress,
          abi: treasuryAbi,
          functionName: "sellShares",
          args: [1n, minPrice],
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

  async function topUpOps() {
    await runTransaction({
      action: "topup",
      processingTitle: "Topping up operations",
      successTitle: "Operations funded",
      errorTitle: "Top-up failed",
      startMessage: `Waiting for wallet approval for ${formatEther(defaultTopUp)} OG.`,
      run: () =>
        sendTransactionAsync({
          to: agentAddress,
          value: defaultTopUp,
        }),
    });
  }

  async function testRevenue() {
    await runTransaction({
      action: "revenue",
      processingTitle: "Sending test revenue",
      successTitle: "Revenue distributed",
      errorTitle: "Revenue test failed",
      startMessage: `Waiting for wallet approval for ${formatEther(defaultRevenue)} OG.`,
      run: () =>
        writeContractAsync({
          address: agentAddress,
          abi: treasuryAbi,
          functionName: "subscribe",
          args: [],
          value: defaultRevenue,
        }),
    });
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

      const receipt = await publicClient.waitForTransactionReceipt({ hash });
      if (receipt.status !== "success") {
        throw new Error("Transaction reverted on-chain.");
      }

      upsertToast({
        id: toastId,
        title: successTitle,
        message: "Confirmed on-chain.",
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
      run: () => generateAgentPost(profileAgentID),
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

  async function submitPostAction(
    postID: string,
    action: "like" | "comment" | "repost",
  ) {
    const text = commentDrafts[postID] ?? "";
    const key = `${action}:${postID}`;
    setActivePostAction(key);
    try {
      await createPostAction(
        profileAgentID,
        postID,
        action,
        address ?? "anonymous",
        text,
      );
      if (action === "comment") {
        setCommentDrafts((current) => ({ ...current, [postID]: "" }));
      }
      await refreshPosts();
    } catch (error) {
      pushToast({
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
              <div className="grid size-10 shrink-0 place-items-center rounded-full border border-white/15 bg-white/5 text-lg">
                A
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
            <WalletBar />
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
                  Share supply
                </p>
                <p className="mt-2 text-3xl font-semibold">
                  {totalSupply.data ? `${totalSupply.data}` : "0"}
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
          </div>

          <div className="min-w-0 rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4 backdrop-blur sm:p-6">
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

            <div className="mt-6 space-y-3 border-t border-white/10 pt-6">
              <button
                onClick={buyOneShare}
                disabled={
                  !isConnected || !hasAgentAddress || activeAction !== null
                }
                className="flex min-h-12 w-full items-center justify-between gap-3 rounded-full bg-[var(--signal)] px-5 py-3 text-sm font-semibold text-[var(--ink)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-40"
              >
                <span className="inline-flex items-center gap-2">
                  <BadgeDollarSign size={16} />
                  {activeAction === "buy" ? "Buying..." : "Buy 1 share"}
                </span>
                <span>
                  {buyPrice.data ? `${formatEther(buyPrice.data)} OG` : "--"}
                </span>
              </button>
              <div className="grid gap-3 sm:grid-cols-2">
                <button
                  onClick={sellOneShare}
                  disabled={
                    !isConnected || !hasAgentAddress || activeAction !== null
                  }
                  className="flex min-h-11 items-center justify-between gap-3 rounded-full border border-white/12 bg-white/5 px-4 py-2 text-sm font-medium text-white transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <span>
                    {activeAction === "sell" ? "Selling..." : "Sell 1"}
                  </span>
                  <span className="mono text-xs text-white/60">
                    {sellPrice.data
                      ? `${formatEther(sellPrice.data)} OG`
                      : "--"}
                  </span>
                </button>
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
                <button
                  onClick={topUpOps}
                  disabled={
                    !isConnected || !hasAgentAddress || activeAction !== null
                  }
                  className="flex min-h-11 items-center justify-between gap-3 rounded-full border border-[var(--signal)]/45 bg-[var(--signal)]/12 px-4 py-2 text-sm font-medium text-[var(--signal)] transition hover:bg-[var(--signal)]/18 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <span>
                    {activeAction === "topup" ? "Funding..." : "Top up ops"}
                  </span>
                  <span className="mono text-xs text-[var(--signal)]/75">
                    {formatEther(defaultTopUp)} OG
                  </span>
                </button>
                <button
                  onClick={testRevenue}
                  disabled={
                    !isConnected || !hasAgentAddress || activeAction !== null
                  }
                  className="flex min-h-11 items-center justify-between gap-3 rounded-full border border-[var(--ember)]/45 bg-[var(--ember)]/12 px-4 py-2 text-sm font-medium text-[var(--ember)] transition hover:bg-[var(--ember)]/18 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <span>
                    {activeAction === "revenue" ? "Sending..." : "Test revenue"}
                  </span>
                  <span className="mono text-xs text-[var(--ember)]/75">
                    {formatEther(defaultRevenue)} OG
                  </span>
                </button>
              </div>
              <p className="text-xs leading-5 text-white/45">
                Test revenue calls `subscribe()` with 0.01 OG so shareholders
                can test Claim.
              </p>
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
                disabled={activePostAction !== null}
                className="inline-flex min-h-10 items-center gap-2 rounded-full bg-[var(--ink)] px-4 py-2 text-sm font-medium text-[var(--paper)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Activity size={16} />
                {activePostAction === "generate"
                  ? "Generating..."
                  : "Generate post"}
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
                    {post.text}
                  </p>
                  {post.imageRef ? (
                    <p className="mono mt-3 break-all text-xs text-[var(--ink-muted)]">
                      {post.imageRef}
                    </p>
                  ) : null}
                </div>
                <ProofModal proof={post.proof} />
              </div>
              <div className="mt-5 grid gap-3 border-t border-[var(--ink)]/10 pt-4">
                <div className="flex flex-wrap items-center gap-2">
                  <button
                    onClick={() => submitPostAction(post.id, "like")}
                    disabled={activePostAction !== null}
                    className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm text-[var(--ink-muted)] transition hover:text-[var(--ember)] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Heart size={15} />
                    {post.likes ?? 0}
                  </button>
                  <button
                    onClick={() => submitPostAction(post.id, "repost")}
                    disabled={activePostAction !== null}
                    className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm text-[var(--ink-muted)] transition hover:text-[var(--signal)] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Repeat2 size={15} />
                    {post.reposts ?? 0}
                  </button>
                  <span className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm text-[var(--ink-muted)]">
                    <MessageCircle size={15} />
                    {post.comments ?? 0}
                  </span>
                </div>
                <div className="flex flex-col gap-2 sm:flex-row">
                  <input
                    value={commentDrafts[post.id] ?? ""}
                    onChange={(event) =>
                      setCommentDrafts((current) => ({
                        ...current,
                        [post.id]: event.target.value,
                      }))
                    }
                    className="min-h-10 flex-1 border border-[var(--ink)]/10 bg-white px-3 text-sm outline-none placeholder:text-[var(--ink-muted)]/55 focus:border-[var(--signal)]"
                    placeholder="Add a real comment event"
                  />
                  <button
                    onClick={() => submitPostAction(post.id, "comment")}
                    disabled={
                      activePostAction !== null ||
                      !(commentDrafts[post.id] ?? "").trim()
                    }
                    className="inline-flex min-h-10 items-center justify-center gap-2 rounded-full bg-[var(--ink)] px-4 text-sm font-medium text-[var(--paper)] transition hover:translate-y-[-1px] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Comment
                  </button>
                </div>
              </div>
            </article>
          ))}
        </div>

        <aside className="min-w-0 space-y-4">
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
