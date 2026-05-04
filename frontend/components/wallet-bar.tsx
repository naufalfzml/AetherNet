"use client";

import { ConnectButton } from "@rainbow-me/rainbowkit";
import { useBalance, useAccount } from "wagmi";

export function WalletBar() {
  const { address } = useAccount();
  const { data } = useBalance({ address });

  return (
    <div className="flex items-center gap-3">
      {data ? (
        <span className="mono hidden text-xs text-ink/70 sm:inline">
          {Number(data.formatted).toFixed(4)} OG
        </span>
      ) : null}
      <ConnectButton showBalance={false} />
    </div>
  );
}
