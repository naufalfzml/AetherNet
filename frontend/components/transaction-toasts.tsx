"use client";

import { AlertTriangle, CheckCircle2, Loader2, X } from "lucide-react";
import { explorerURL } from "@/lib/endpoints";

export type TxToast = {
  id: number;
  title: string;
  message: string;
  status: "processing" | "success" | "error";
  hash?: `0x${string}`;
};

export function TransactionToasts({
  toasts,
  onDismiss,
}: {
  toasts: TxToast[];
  onDismiss: (id: number) => void;
}) {
  if (toasts.length === 0) return null;

  return (
    <div className="fixed right-4 top-4 z-[80] flex w-[calc(100vw-2rem)] max-w-sm flex-col gap-3 sm:right-6 sm:top-6">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`border p-4 shadow-[0_18px_55px_rgba(0,0,0,0.22)] ${
            toast.status === "success"
              ? "border-[var(--signal)]/35 bg-[#10201d] text-white"
              : toast.status === "error"
                ? "border-red-400/35 bg-[#241111] text-white"
                : "border-white/15 bg-[#171717] text-white"
          }`}
        >
          <div className="flex items-start gap-3">
            <div className="mt-0.5 shrink-0">
              {toast.status === "processing" ? (
                <Loader2
                  size={18}
                  className="animate-spin text-[var(--signal)]"
                />
              ) : toast.status === "success" ? (
                <CheckCircle2 size={18} className="text-[var(--signal)]" />
              ) : (
                <AlertTriangle size={18} className="text-red-300" />
              )}
            </div>
            <div className="min-w-0 flex-1">
              <p className="font-semibold">{toast.title}</p>
              <p className="mt-1 text-sm leading-6 text-white/68">
                {toast.message}
              </p>
              {toast.hash ? (
                <a
                  href={
                    explorerURL ? `${explorerURL}/tx/${toast.hash}` : undefined
                  }
                  className="mono mt-2 block break-all text-xs text-[var(--signal)]"
                  target="_blank"
                  rel="noreferrer"
                >
                  {shorten(toast.hash)}
                </a>
              ) : null}
            </div>
            <button
              type="button"
              onClick={() => onDismiss(toast.id)}
              className="grid size-7 shrink-0 place-items-center border border-white/10 text-white/60 transition hover:text-white"
              aria-label="Dismiss notification"
            >
              <X size={14} />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

export function getErrorMessage(error: unknown) {
  if (error instanceof Error) {
    if (error.message.toLowerCase().includes("user rejected")) {
      return "Transaction rejected in wallet.";
    }
    return error.message;
  }
  return "Transaction failed.";
}

function shorten(value: string) {
  if (value.length < 12) return value;
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}
