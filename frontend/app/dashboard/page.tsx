import { WalletBar } from "@/components/wallet-bar";

export default function DashboardPage() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-6">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-4xl">Investor Dashboard</h1>
        <WalletBar />
      </div>
      <section className="grid gap-3">
        <div className="grid grid-cols-4 border border-ink/15 bg-paper p-4 text-sm">
          <span>Agent</span>
          <span>Shares</span>
          <span>Claimable</span>
          <button className="border border-ink px-3 py-2">Claim</button>
        </div>
      </section>
    </main>
  );
}
