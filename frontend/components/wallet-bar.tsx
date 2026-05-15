"use client";

import { ConnectButton } from "@rainbow-me/rainbowkit";
import { useAccount, useBalance } from "wagmi";

export function WalletBar({ showBalance = false }: { showBalance?: boolean }) {
  const { address } = useAccount();
  const { data: balance } = useBalance({ address, query: { enabled: showBalance && !!address } });
  
  const buttonClass =
    "inline-flex min-h-10 max-w-full items-center justify-center rounded-2xl bg-white px-4 py-2 text-sm font-semibold text-black shadow-[0_8px_24px_rgba(0,0,0,0.18)] transition hover:translate-y-[-1px] sm:px-5";

  return (
    <div className="flex items-center gap-3">
      {showBalance && balance && (
        <div className="hidden flex-col items-end mr-2 sm:flex">
          <span className="text-[10px] font-black uppercase tracking-widest text-white/30">Your Balance</span>
          <span className="font-mono text-sm font-bold text-white">
            {Number(balance.formatted).toFixed(4)} {balance.symbol}
          </span>
        </div>
      )}
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
            <button
              onClick={openAccountModal}
              type="button"
              className={buttonClass}
            >
              <span className="truncate">{account.displayName}</span>
            </button>
          );
        }}
      </ConnectButton.Custom>
    </div>
  );
}
