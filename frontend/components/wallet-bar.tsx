"use client";

import { ConnectButton } from "@rainbow-me/rainbowkit";
import { useBalance, useAccount } from "wagmi";

export function WalletBar() {
  const { address } = useAccount();
  const { data } = useBalance({ address });
  const buttonClass =
    "inline-flex h-10 items-center justify-center rounded-2xl bg-white px-5 text-sm font-semibold text-black shadow-[0_8px_24px_rgba(0,0,0,0.18)] transition hover:translate-y-[-1px]";

  return (
    <div className="flex items-center gap-3">
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
              <button onClick={openConnectModal} type="button" className={buttonClass}>
                Connect Wallet
              </button>
            );
          }

          if (chain.unsupported) {
            return (
              <button onClick={openChainModal} type="button" className={buttonClass}>
                Wrong Network
              </button>
            );
          }

          return (
            <button onClick={openAccountModal} type="button" className={buttonClass}>
              {account.displayName}
            </button>
          );
        }}
      </ConnectButton.Custom>
    </div>
  );
}
