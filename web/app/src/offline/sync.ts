import type { QueryClient } from "@tanstack/react-query";
import { drainQueue, type DrainOutcome } from "./queue";
import {
  createExpense,
  updateExpense,
  deleteExpense,
} from "../api/expenses";
import { ApiError } from "../api/client";
import { expensesQueryKey } from "../hooks/useExpenses";

function isOnline(): boolean {
  if (typeof navigator === "undefined") return true;
  return navigator.onLine !== false;
}

// Login.tsx sets `queueBlocked=1` when it cannot clear the queue after
// detecting a different user. Until a subsequent login successfully clears
// the queue, draining is unsafe — the queue may contain a previous user's
// writes that would replay under the current session. The mount-time drain
// in setupOnlineSync and the `online` listener both reach syncQueue without
// going through Login.tsx, so the gate has to live here too.
//
// In-memory mirror of the same flag, used when Web Storage is unavailable
// (Safari private mode etc.). Login.tsx maintains it alongside the
// localStorage value so subsequent syncQueue calls in this page session
// still see the blocked state when the persistent write failed.
let inMemoryQueueBlocked = false;

export function setInMemoryQueueBlocked(blocked: boolean): void {
  inMemoryQueueBlocked = blocked;
}

function isQueueBlocked(): boolean {
  try {
    return localStorage.getItem("queueBlocked") === "1";
  } catch {
    // Web Storage unavailable — fall back to the in-memory flag. If
    // Login.tsx couldn't clear the queue this session it set the flag,
    // so the drain stays blocked until a successful clear in a future
    // login. Across page reloads with broken storage we have no signal
    // to recover, so the default is unblocked.
    return inMemoryQueueBlocked;
  }
}

function classifyError(err: unknown): DrainOutcome {
  if (err instanceof TypeError) return "retry";
  if (err instanceof ApiError) {
    // 4xx (except 401) are permanent — server rejected the payload, retrying
    // will not help. 401 should retry once auth is restored. 5xx is transient.
    if (err.status === 401) return "retry";
    if (err.status >= 400 && err.status < 500) return "drop";
    return "retry";
  }
  return "retry";
}

let inFlight: Promise<void> | null = null;

export async function syncQueue(queryClient: QueryClient): Promise<void> {
  if (!isOnline()) return;
  if (isQueueBlocked()) return;
  // Coalesce concurrent invocations. Without this, an `online` event firing
  // while a mutation is mid-flight could replay the same queued entry through
  // both paths and double-write.
  if (inFlight) return inFlight;
  inFlight = (async () => {
    try {
      const result = await drainQueue(async (entry): Promise<DrainOutcome> => {
        try {
          if (entry.op === "create") {
            const { tempId: _ignored, ...input } = entry.payload;
            void _ignored;
            await createExpense(input);
          } else if (entry.op === "update") {
            await updateExpense(entry.payload.id, entry.payload.patch);
          } else {
            await deleteExpense(entry.payload.id);
          }
          return "ok";
        } catch (err) {
          return classifyError(err);
        }
      });
      if (result.processed > 0 || result.dropped > 0) {
        await queryClient.invalidateQueries({ queryKey: expensesQueryKey });
        await queryClient.invalidateQueries({ queryKey: ["insights"] });
      }
    } finally {
      inFlight = null;
    }
  })();
  return inFlight;
}

let detach: (() => void) | null = null;

export function setupOnlineSync(queryClient: QueryClient): () => void {
  if (typeof window === "undefined") return () => {};
  if (detach) return detach;

  const handler = () => {
    void syncQueue(queryClient);
  };
  window.addEventListener("online", handler);
  if (isOnline()) {
    void syncQueue(queryClient);
  }
  detach = () => {
    window.removeEventListener("online", handler);
    detach = null;
  };
  return detach;
}
