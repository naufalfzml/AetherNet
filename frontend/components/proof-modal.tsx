"use client";

import { ShieldCheck, X } from "lucide-react";
import { useState } from "react";
import type { Proof } from "@/lib/api";

export function ProofModal({ proof }: { proof: Proof }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        className="inline-flex h-8 items-center gap-2 border border-ink/20 bg-paper px-3 text-xs uppercase tracking-[0.08em] text-ink shadow-line"
        onClick={() => setOpen(true)}
      >
        <ShieldCheck size={14} />
        Proof
      </button>
      {open ? (
        <div className="fixed inset-0 z-50 grid place-items-center bg-ink/40 p-4">
          <div className="w-full max-w-xl border border-ink/20 bg-paper p-5 shadow-2xl">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-xl">Proof of Inference</h2>
              <button
                className="grid size-8 place-items-center border border-ink/20"
                onClick={() => setOpen(false)}
              >
                <X size={16} />
              </button>
            </div>
            <dl className="space-y-3 text-sm">
              {Object.entries(proof).map(([key, value]) => (
                <div key={key}>
                  <dt className="mono text-xs uppercase text-ink/50">{key}</dt>
                  <dd className="mono break-all border border-ink/10 bg-white/50 p-2">
                    {value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      ) : null}
    </>
  );
}
