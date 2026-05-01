import { useMemo } from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  createExpense,
  deleteExpense,
  listExpenses,
  updateExpense,
} from "../api/expenses";
import { ApiError } from "../api/client";
import type {
  CreateExpenseInput,
  Expense,
  Insights,
  UpdateExpenseInput,
} from "../types";
import { enqueueWrite, removeQueued } from "../offline/queue";
import { deriveInsights, expensesForCategory } from "../insights/derive";

// Single canonical cache for every read in the app. Feed shows a windowed
// slice of this array; Insights and CategoryDetails derive their views via
// useMemo selectors. There is no separate insights cache, so mutations only
// touch this one key — and screens never coordinate two queries that
// resolve at different times (the source of the old "number jumps" UX).
export const expensesQueryKey = ["expenses"] as const;

export function useAllExpenses(): UseQueryResult<Expense[], Error> {
  return useQuery<Expense[], Error>({
    queryKey: expensesQueryKey,
    queryFn: async () => {
      const page = await listExpenses();
      return page.items;
    },
    // Mutations explicitly invalidate this key after a successful write, so
    // we can treat the cached array as fresh between writes. Without this,
    // every screen mount on the Feed → Insights → CategoryDetails path would
    // refetch the full list and bring back the "loading flash" we just
    // removed.
    staleTime: 5 * 60 * 1000,
  });
}

// useInsightsFor reproduces the shape the deleted /api/insights endpoint
// used to return, derived in O(n) from the cached array. Switching months
// on the Insights screen recomputes from cache — zero network, zero
// loading state — so the rendered totals never snap back to 0 between
// query keys.
export function useInsightsFor(
  year: number,
  month: number,
): { data: Insights | undefined; isLoading: boolean } {
  const query = useAllExpenses();
  const data = useMemo(() => {
    if (!query.data) return undefined;
    return deriveInsights(query.data, year, month, new Date());
  }, [query.data, year, month]);
  return { data, isLoading: query.isLoading };
}

// useCategoryView returns CategoryDetails' four header numbers from a
// single useMemo over one array on one render — `pct` can never flicker
// because the numerator (filtered total) and denominator (month total)
// are computed in the same pass.
export function useCategoryView(
  year: number,
  month: number,
  label: string,
): {
  items: Expense[];
  total: number;
  count: number;
  monthTotal: number;
  pct: number;
  isLoading: boolean;
} {
  const query = useAllExpenses();
  const view = useMemo(() => {
    const all = query.data ?? [];
    const items = expensesForCategory(all, year, month, label);
    let total = 0;
    for (const e of items) total += e.amount;
    let monthTotal = 0;
    for (const e of all) {
      const ymd = e.date;
      if (
        ymd &&
        ymd.length >= 10 &&
        parseInt(ymd.slice(0, 4), 10) === year &&
        parseInt(ymd.slice(5, 7), 10) === month
      ) {
        monthTotal += e.amount;
      }
    }
    const pct = monthTotal > 0 ? (total / monthTotal) * 100 : 0;
    return { items, total, count: items.length, monthTotal, pct };
  }, [query.data, year, month, label]);
  return { ...view, isLoading: query.isLoading };
}

let _tempIdCounter = -1;
function nextTempId(): number {
  return _tempIdCounter--;
}

function isNetworkError(err: unknown): boolean {
  if (err instanceof TypeError) return true;
  if (typeof navigator !== "undefined" && navigator.onLine === false) {
    return true;
  }
  return false;
}

function todayIso(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

type CreateContext = {
  tempId: number;
  queueId: number | null;
  previous: Expense[] | undefined;
};

type UpdateContext = {
  queueId: number | null;
  previous: Expense[] | undefined;
};

type DeleteContext = {
  queueId: number | null;
  previous: Expense[] | undefined;
};

async function dropQueued(queueId: number | null): Promise<void> {
  if (queueId === null) return;
  try {
    await removeQueued(queueId);
  } catch {
    // queue already gone or IDB unavailable — nothing to do
  }
}

export type CreateExpenseMutation = UseMutationResult<
  Expense | null,
  Error,
  CreateExpenseInput,
  CreateContext
>;

export type UpdateExpenseMutation = UseMutationResult<
  Expense | null,
  Error,
  { id: number; patch: UpdateExpenseInput },
  UpdateContext
>;

export type DeleteExpenseMutation = UseMutationResult<
  boolean,
  Error,
  number,
  DeleteContext
>;

export function useCreateExpense(): CreateExpenseMutation {
  const qc = useQueryClient();
  return useMutation<Expense | null, Error, CreateExpenseInput, CreateContext>({
    mutationKey: ["expenses", "create"],
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey: expensesQueryKey });
      const previous = qc.getQueryData<Expense[]>(expensesQueryKey);
      const tempId = nextTempId();
      const optimistic: Expense = {
        id: tempId,
        amount: input.amount,
        description: input.description,
        category: input.category,
        date: input.date ?? todayIso(),
      };
      qc.setQueryData<Expense[]>(expensesQueryKey, (old) =>
        old ? [optimistic, ...old] : [optimistic],
      );
      let queueId: number | null = null;
      try {
        queueId = await enqueueWrite({
          op: "create",
          payload: { ...input, tempId },
        });
      } catch {
        // IDB unavailable (private browsing / quota) — proceed without offline
        // persistence. The network call below is the source of truth.
      }
      return { tempId, queueId, previous };
    },
    mutationFn: async (input) => {
      try {
        return await createExpense(input);
      } catch (err) {
        if (isNetworkError(err)) return null;
        throw err;
      }
    },
    onSuccess: async (result, _vars, ctx) => {
      if (!ctx) return;
      if (result) {
        await dropQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
      } else if (ctx.queueId === null) {
        // Network failed AND the offline queue refused the write (IDB
        // unavailable). The optimistic row has nowhere to be persisted —
        // roll back so the user can see and retry.
        qc.setQueryData<Expense[]>(expensesQueryKey, ctx.previous);
      }
    },
    onError: async (err, _vars, ctx) => {
      if (!ctx) return;
      if (err instanceof ApiError && err.status === 409) {
        // The server already has this row — either the offline drain raced
        // ahead and replayed our queued entry (foreground POST then comes
        // back 409), or the (date, amount, description) unique key collides
        // with a pre-existing row. Drop the queue entry and refetch so the
        // canonical row replaces the optimistic one; do NOT roll back, or
        // we'd hide a row the user really did add.
        await dropQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
        return;
      }
      await dropQueued(ctx.queueId);
      qc.setQueryData<Expense[]>(expensesQueryKey, ctx.previous);
    },
  });
}

export function useUpdateExpense(): UpdateExpenseMutation {
  const qc = useQueryClient();
  return useMutation<
    Expense | null,
    Error,
    { id: number; patch: UpdateExpenseInput },
    UpdateContext
  >({
    mutationKey: ["expenses", "update"],
    onMutate: async ({ id, patch }) => {
      await qc.cancelQueries({ queryKey: expensesQueryKey });
      const previous = qc.getQueryData<Expense[]>(expensesQueryKey);
      qc.setQueryData<Expense[]>(expensesQueryKey, (old) =>
        old?.map((e) =>
          e.id === id
            ? {
                ...e,
                amount: patch.amount ?? e.amount,
                description: patch.description ?? e.description,
                category: patch.category ?? e.category,
                date: patch.date ?? e.date,
              }
            : e,
        ),
      );
      let queueId: number | null = null;
      try {
        queueId = await enqueueWrite({
          op: "update",
          payload: { id, patch },
        });
      } catch {
        // IDB unavailable — proceed without offline persistence.
      }
      return { queueId, previous };
    },
    mutationFn: async ({ id, patch }) => {
      try {
        return await updateExpense(id, patch);
      } catch (err) {
        if (isNetworkError(err)) return null;
        throw err;
      }
    },
    onSuccess: async (result, _vars, ctx) => {
      if (!ctx) return;
      if (result) {
        await dropQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
      } else if (ctx.queueId === null) {
        // Same recovery as create: no offline queue + no network = roll back.
        qc.setQueryData<Expense[]>(expensesQueryKey, ctx.previous);
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await dropQueued(ctx.queueId);
      qc.setQueryData<Expense[]>(expensesQueryKey, ctx.previous);
    },
  });
}

export function useDeleteExpense(): DeleteExpenseMutation {
  const qc = useQueryClient();
  return useMutation<boolean, Error, number, DeleteContext>({
    mutationKey: ["expenses", "delete"],
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: expensesQueryKey });
      const previous = qc.getQueryData<Expense[]>(expensesQueryKey);
      qc.setQueryData<Expense[]>(expensesQueryKey, (old) =>
        old?.filter((e) => e.id !== id),
      );
      let queueId: number | null = null;
      try {
        queueId = await enqueueWrite({
          op: "delete",
          payload: { id },
        });
      } catch {
        // IDB unavailable — proceed without offline persistence.
      }
      return { queueId, previous };
    },
    mutationFn: async (id) => {
      try {
        await deleteExpense(id);
        return true;
      } catch (err) {
        if (isNetworkError(err)) return false;
        throw err;
      }
    },
    onSuccess: async (delivered, _vars, ctx) => {
      if (!ctx) return;
      if (delivered) {
        await dropQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
      } else if (ctx.queueId === null) {
        // Same recovery as create: no offline queue + no network = roll back.
        qc.setQueryData<Expense[]>(expensesQueryKey, ctx.previous);
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await dropQueued(ctx.queueId);
      qc.setQueryData<Expense[]>(expensesQueryKey, ctx.previous);
    },
  });
}
