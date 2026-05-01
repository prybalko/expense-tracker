import type { QueryClient } from "@tanstack/react-query";
import { drainQueue } from "./queue";
import {
  createExpense,
  updateExpense,
  deleteExpense,
} from "../api/expenses";
import { expensesQueryKey } from "../hooks/useExpenses";

function isOnline(): boolean {
  if (typeof navigator === "undefined") return true;
  return navigator.onLine !== false;
}

export async function syncQueue(queryClient: QueryClient): Promise<void> {
  if (!isOnline()) return;
  const result = await drainQueue(async (entry) => {
    if (entry.op === "create") {
      const { tempId: _ignored, ...input } = entry.payload;
      void _ignored;
      await createExpense(input);
    } else if (entry.op === "update") {
      await updateExpense(entry.payload.id, entry.payload.patch);
    } else {
      await deleteExpense(entry.payload.id);
    }
  });
  if (result.processed > 0) {
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
