"use client";

import Link from "next/link";
import { ConnectButton } from "@rainbow-me/rainbowkit";
import { useBalance, useAccount } from "wagmi";

export function WalletBar() {
  const { address } = useAccount();
  const { data } = useBalance({ address });
  const buttonClass =
    "inline-flex min-h-10 max-w-full items-center justify-center rounded-2xl bg-white px-4 py-2 text-sm font-semibold text-black shadow-[0_8px_24px_rgba(0,0,0,0.18)] transition hover:translate-y-[-1px] sm:px-5";
  const secondaryClass =
    "inline-flex min-h-10 max-w-full items-center justify-center rounded-2xl border border-black/10 bg-white px-4 py-2 text-sm font-semibold text-black shadow-[0_8px_24px_rgba(0,0,0,0.08)] transition hover:translate-y-[-1px] sm:px-5";

  return (
    <div className="flex max-w-full flex-wrap items-center gap-3">
      {data ? (
        <span className="mono hidden text-xs text-ink/70 sm:inline">
          {Number(data.formatted).toFixed(4)} OG
        </span>
      ) : null}
      <ConnectButton.Custom>
        {({
          account,
          chain,
          mounted,
          openAccountModal,
          openChainModal,
          openConnectModal,
        }) => {
          const ready = mounted;
          const connected = ready && account && chain;

          if (!connected) {
            return (
              <button
                onClick={openConnectModal}
                type="button"
                className={buttonClass}
              >
                Connect Wallet
              </button>
            );
          }

          if (chain.unsupported) {
            return (
              <button
                onClick={openChainModal}
                type="button"
                className={buttonClass}
              >
                Wrong Network
              </button>
            );
          }

          return (
            <>
              <Link href="/dashboard" className={secondaryClass}>
                Dashboard
              </Link>
              <button
                onClick={openAccountModal}
                type="button"
                className={buttonClass}
              >
                <span className="truncate">{account.displayName}</span>
              </button>
            </>
          );
        }}
      </ConnectButton.Custom>
    </div>
  );
}
