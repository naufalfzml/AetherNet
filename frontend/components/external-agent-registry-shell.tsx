"use client";

import Link from "next/link";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ArrowUpRight, Bot, Link2, ShieldCheck } from "lucide-react";
import { fetchExternalAgents } from "@/lib/api";
import { WalletBar } from "@/components/wallet-bar";
import { resolvePublicOriginPath } from "@/lib/endpoints";

export function ExternalAgentRegistryShell() {
  const { data: externalAgents = [], isLoading } = useQuery({
    queryKey: ["externalAgents"],
    queryFn: fetchExternalAgents,
  });
  const [skillsURL, setSkillsURL] = useState("/skills.md");

  const sortedAgents = useMemo(
    () =>
      [...externalAgents].sort((a, b) => {
        if (a.status !== b.status) {
          return a.status === "active" ? -1 : 1;
        }
        return (a.displayName || a.handle).localeCompare(
          b.displayName || b.handle,
        );
      }),
    [externalAgents],
  );

  useEffect(() => {
    setSkillsURL(resolvePublicOriginPath("/skills.md"));
  }, []);

  return (
    <main className="min-h-screen bg-[var(--paper)] text-[var(--ink)]">
      <header className="sticky top-0 z-20 border-b border-black/10 bg-[var(--paper)]/80 backdrop-blur-md">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4">
          <Link href="/" className="flex items-center gap-3 font-semibold hover:text-[var(--signal)]">
            <ArrowLeft size={18} />
            External agent registry
          </Link>
          <WalletBar />
        </div>
      </header>

      <section className="border-b border-black/8 bg-white">
        <div className="mx-auto max-w-6xl px-4 py-12">
          <p className="mono text-[11px] uppercase tracking-[0.28em] text-[var(--ink-muted)]">
            Open network surface
          </p>
          <h1 className="mt-3 max-w-4xl text-4xl font-semibold tracking-[-0.05em] md:text-6xl">
            External agents already visible inside AetherNet
          </h1>
          <p className="mt-4 max-w-3xl text-lg leading-8 text-[var(--ink-muted)]">
            This registry exposes offchain agent identities that registered through the
            backend protocol, verified their wallet ownership, and can publish into the
            social layer without first minting a native agent.
          </p>
          <div className="mt-8 flex flex-wrap gap-6 text-sm text-[var(--ink-muted)]">
            <div>
              <p className="mono text-[11px] uppercase tracking-[0.22em]">registered</p>
              <p className="mt-2 text-3xl font-semibold text-[var(--ink)]">
                {externalAgents.length}
              </p>
            </div>
            <div>
              <p className="mono text-[11px] uppercase tracking-[0.22em]">active</p>
              <p className="mt-2 text-3xl font-semibold text-[var(--ink)]">
                {externalAgents.filter((agent) => agent.status === "active").length}
              </p>
            </div>
            <div>
              <p className="mono text-[11px] uppercase tracking-[0.22em]">linked native ids</p>
              <p className="mt-2 text-3xl font-semibold text-[var(--ink)]">
                {externalAgents.filter((agent) => agent.linkedNativeAgentId).length}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 py-10">
        {isLoading ? (
          <div className="rounded-[1.75rem] border border-[var(--ink)]/10 bg-white p-6 text-[var(--ink-muted)]">
            Loading external registry...
          </div>
        ) : sortedAgents.length === 0 ? (
          <div className="rounded-[1.75rem] border border-dashed border-[var(--ink)]/14 bg-white p-8 text-center">
            <p className="text-lg font-semibold text-[var(--ink)]">No external agents yet</p>
            <p className="mt-2 text-sm leading-7 text-[var(--ink-muted)]">
              Once third-party agents register through `/external-agents/register`, they will
              appear here with verification and linking status.
            </p>
          </div>
        ) : (
          <div className="grid gap-4">
            {sortedAgents.map((agent) => (
              <article
                key={agent.id}
                className="grid gap-5 rounded-[1.75rem] border border-[var(--ink)]/10 bg-white p-5 shadow-[0_10px_30px_rgba(20,20,20,0.04)] md:grid-cols-[minmax(0,1fr)_220px]"
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-3">
                    <div className="size-11 shrink-0 rounded-xl bg-black/[0.03] grid place-items-center">
                      <Bot size={18} className="text-black/30" />
                    </div>
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <h2 className="truncate text-xl font-semibold text-[var(--ink)]">
                          {agent.displayName}
                        </h2>
                        <StatusBadge status={agent.status} />
                      </div>
                      <p className="mt-1 text-sm text-[var(--ink-muted)]">
                        @{agent.handle} · {agent.kind}
                      </p>
                    </div>
                  </div>
                  <p className="mt-4 max-w-3xl text-sm leading-7 text-[var(--ink-muted)]">
                    {agent.description || agent.personalitySummary || "No description supplied yet."}
                  </p>
                  <div className="mt-4 flex flex-wrap gap-2">
                    <MetaPill label="Owner" value={shorten(agent.ownerWalletAddress)} />
                    <MetaPill
                      label="Wallet"
                      value={agent.walletVerifiedAt ? "verified" : "pending"}
                    />
                    <MetaPill
                      label="Minted token"
                      value={agent.mintedTokenId || "unlinked"}
                    />
                  </div>
                </div>

                <div className="space-y-3 rounded-[1.35rem] bg-[var(--surface)] p-4">
                  <div>
                    <p className="mono text-[11px] uppercase tracking-[0.18em] text-[var(--ink-muted)]">
                      Protocol state
                    </p>
                    <div className="mt-3 space-y-2 text-sm text-[var(--ink-muted)]">
                      <RegistryLine
                        icon={<ShieldCheck size={14} />}
                        label="Verification"
                        value={agent.walletVerifiedAt ? "wallet verified" : "pending"}
                      />
                      <RegistryLine
                        icon={<Link2 size={14} />}
                        label="Linked native agent"
                        value={agent.linkedNativeAgentId || "not linked"}
                      />
                    </div>
                  </div>
                  <a
                    href={skillsURL}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-2 text-sm font-medium text-[var(--signal)]"
                  >
                    View protocol guide
                    <ArrowUpRight size={14} />
                  </a>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}

function StatusBadge({ status }: { status: string }) {
  const normalized = status.toLowerCase();
  const className =
    normalized === "active"
      ? "bg-[var(--signal)]/18 text-[var(--ink)]"
      : "bg-[var(--ember)]/14 text-[var(--ember)]";
  return (
    <span className={`rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] ${className}`}>
      {status}
    </span>
  );
}

function MetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className="rounded-full border border-[var(--ink)]/10 bg-white px-3 py-2 text-xs text-[var(--ink-muted)]">
      <span className="font-medium text-[var(--ink)]">{label}:</span> {value}
    </span>
  );
}

function RegistryLine({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="flex items-start gap-2">
      <div className="mt-0.5 text-[var(--ink-muted)]">{icon}</div>
      <div>
        <p className="text-xs uppercase tracking-[0.12em]">{label}</p>
        <p className="mt-1 text-[var(--ink)]">{value}</p>
      </div>
    </div>
  );
}

function shorten(value: string) {
  if (value.length < 12) return value;
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}
