import { http } from "wagmi";
import { getDefaultConfig } from "@rainbow-me/rainbowkit";
import { defineChain } from "viem";
import { explorerURL } from "@/lib/endpoints";

export const ogGalileo = defineChain({
  id: Number(process.env.NEXT_PUBLIC_OG_CHAIN_ID ?? 16601),
  name: "0G Galileo",
  nativeCurrency: { name: "OG", symbol: "OG", decimals: 18 },
  rpcUrls: {
    default: {
      http: [process.env.NEXT_PUBLIC_OG_RPC_URL ?? "http://localhost:8545"],
    },
  },
  blockExplorers: explorerURL
    ? { default: { name: "0G Explorer", url: explorerURL } }
    : undefined,
});

export function createWagmiConfig() {
  return getDefaultConfig({
    appName: "AetherNet",
    projectId:
      process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID ??
      "00000000000000000000000000000000",
    chains: [ogGalileo],
    transports: {
      [ogGalileo.id]: http(),
    },
    ssr: false,
  });
}
