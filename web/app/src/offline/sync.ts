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

export async function syncQueue(queryClient: QueryClient): Promise<void> {
  if (!isOnline()) return;
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
