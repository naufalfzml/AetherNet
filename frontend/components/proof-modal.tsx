"use client";

import { ExternalLink, ShieldCheck, X } from "lucide-react";
import { useState } from "react";
import type { Proof } from "@/lib/api";
import { resolveStoragePointerSrc } from "@/lib/endpoints";

type StorageEvidence = {
  label: string;
  pointer?: string;
};

export function ProofModal({
  proof,
  storageEvidence = [],
}: {
  proof: Proof;
  storageEvidence?: StorageEvidence[];
}) {
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
            {storageEvidence.filter((item) => item.pointer).length > 0 ? (
              <div className="mt-5 border-t border-ink/10 pt-4">
                <p className="mono text-xs uppercase text-ink/50">
                  0G storage evidence
                </p>
                <div className="mt-3 space-y-3">
                  {storageEvidence
                    .filter((item) => item.pointer)
                    .map((item) => {
                      const pointer = item.pointer as string;
                      const rootHash = extractRootHash(pointer);
                      return (
                        <div key={`${item.label}-${pointer}`} className="border border-ink/10 bg-white/50 p-3">
                          <div className="flex items-start justify-between gap-3">
                            <div className="min-w-0">
                              <p className="font-medium">{item.label}</p>
                              <p className="mono mt-2 break-all text-xs text-ink/60">
                                Root Hash: {rootHash}
                              </p>
                              <p className="mono mt-2 break-all text-xs text-ink/45">
                                Pointer: {pointer}
                              </p>
                            </div>
                            <a
                              href={resolveStoragePointerSrc(pointer)}
                              target="_blank"
                              rel="noreferrer"
                              className="inline-flex shrink-0 items-center gap-2 border border-ink/15 bg-paper px-3 py-2 text-xs uppercase tracking-[0.08em] text-ink transition hover:border-ink/30"
                            >
                              View raw data
                              <ExternalLink size={12} />
                            </a>
                          </div>
                        </div>
                      );
                    })}
                </div>
              </div>
            ) : null}
          </div>
        </div>
      ) : null}
    </>
  );
}

function extractRootHash(pointer: string): string {
  if (!pointer) return "";
  const schemeIndex = pointer.indexOf("://");
  if (schemeIndex >= 0) {
    return pointer.slice(schemeIndex + 3) || pointer;
  }
  return pointer;
}
