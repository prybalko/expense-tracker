import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  createExpense,
  deleteExpense,
  listExpenses,
  updateExpense,
  type ListExpensesParams,
} from "../api/expenses";
import { ApiError } from "../api/client";
import type {
  CreateExpenseInput,
  Expense,
  ExpensePage,
  UpdateExpenseInput,
} from "../types";
import { enqueueWrite, removeQueued } from "../offline/queue";

export const expensesQueryKey = ["expenses"] as const;

type ExpensesData = InfiniteData<ExpensePage, string | null>;

export function useExpenses(limit?: number) {
  return useInfiniteQuery<
    ExpensePage,
    Error,
    ExpensesData,
    typeof expensesQueryKey,
    string | null
  >({
    queryKey: expensesQueryKey,
    initialPageParam: null,
    queryFn: ({ pageParam }) => {
      const params: ListExpensesParams = { limit };
      if (pageParam) {
        params.before = pageParam;
      }
      return listExpenses(params);
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
  });
}

// Per-category drill-down on Insights: returns the unpaginated list of
// expenses for the given user × month × category. The cache key starts with
// `expenses` so it's invalidated alongside the main feed by create/update/
// delete mutations (TanStack's invalidateQueries does prefix matching).
export function useExpensesForCategory(
  year: number,
  month: number,
  category: string,
): UseQueryResult<ExpensePage, Error> {
  return useQuery<ExpensePage, Error>({
    queryKey: ["expenses", "by-category", year, month, category],
    queryFn: () => listExpenses({ year, month, category }),
    enabled: Boolean(category) && year > 0 && month >= 1 && month <= 12,
  });
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

function mapPages(
  data: ExpensesData | undefined,
  fn: (items: Expense[]) => Expense[],
): ExpensesData | undefined {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((p) => ({ ...p, items: fn(p.items) })),
  };
}

type CreateContext = {
  tempId: number;
  queueId: number | null;
  previous: ExpensesData | undefined;
};

type UpdateContext = {
  queueId: number | null;
  previous: ExpensesData | undefined;
};

type DeleteContext = {
  queueId: number | null;
  previous: ExpensesData | undefined;
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
  return useMutation<
    Expense | null,
    Error,
    CreateExpenseInput,
    CreateContext
  >({
    mutationKey: ["expenses", "create"],
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey: expensesQueryKey });
      const previous = qc.getQueryData<ExpensesData>(expensesQueryKey);
      const tempId = nextTempId();
      const optimistic: Expense = {
        id: tempId,
        amount: input.amount,
        description: input.description,
        category: input.category,
        date: input.date ?? todayIso(),
      };
      qc.setQueryData<ExpensesData>(expensesQueryKey, (old) => {
        if (!old) {
          return {
            pages: [{ items: [optimistic], nextCursor: null }],
            pageParams: [null],
          };
        }
        return {
          ...old,
          pages: old.pages.map((p, i) =>
            i === 0 ? { ...p, items: [optimistic, ...p.items] } : p,
          ),
        };
      });
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
        await qc.invalidateQueries({ queryKey: ["insights"] });
      } else if (ctx.queueId === null) {
        // Network failed AND the offline queue refused the write (IDB
        // unavailable). The optimistic row has nowhere to be persisted —
        // roll back so the user can see and retry.
        qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
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
        await qc.invalidateQueries({ queryKey: ["insights"] });
        return;
      }
      await dropQueued(ctx.queueId);
      qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
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
      const previous = qc.getQueryData<ExpensesData>(expensesQueryKey);
      qc.setQueryData<ExpensesData>(expensesQueryKey, (old) =>
        mapPages(old, (items) =>
          items.map((e) =>
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
        await qc.invalidateQueries({ queryKey: ["insights"] });
      } else if (ctx.queueId === null) {
        // Same recovery as create: no offline queue + no network = roll back.
        qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await dropQueued(ctx.queueId);
      qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
    },
  });
}

export function useDeleteExpense(): DeleteExpenseMutation {
  const qc = useQueryClient();
  return useMutation<boolean, Error, number, DeleteContext>({
    mutationKey: ["expenses", "delete"],
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: expensesQueryKey });
      const previous = qc.getQueryData<ExpensesData>(expensesQueryKey);
      qc.setQueryData<ExpensesData>(expensesQueryKey, (old) =>
        mapPages(old, (items) => items.filter((e) => e.id !== id)),
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
        await qc.invalidateQueries({ queryKey: ["insights"] });
      } else if (ctx.queueId === null) {
        // Same recovery as create: no offline queue + no network = roll back.
        qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await dropQueued(ctx.queueId);
      qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
    },
  });
}
