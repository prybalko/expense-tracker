import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
  type InfiniteData,
  type UseMutationResult,
} from "@tanstack/react-query";
import {
  createExpense,
  deleteExpense,
  listExpenses,
  updateExpense,
  type ListExpensesParams,
} from "../api/expenses";
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
  queueId: number;
  previous: ExpensesData | undefined;
};

type UpdateContext = {
  queueId: number;
  previous: ExpensesData | undefined;
};

type DeleteContext = {
  queueId: number;
  previous: ExpensesData | undefined;
};

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
      const queueId = await enqueueWrite({
        op: "create",
        payload: { ...input, tempId },
      });
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
        await removeQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
        await qc.invalidateQueries({ queryKey: ["insights"] });
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await removeQueued(ctx.queueId);
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
      const queueId = await enqueueWrite({
        op: "update",
        payload: { id, patch },
      });
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
        await removeQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
        await qc.invalidateQueries({ queryKey: ["insights"] });
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await removeQueued(ctx.queueId);
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
      const queueId = await enqueueWrite({
        op: "delete",
        payload: { id },
      });
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
        await removeQueued(ctx.queueId);
        await qc.invalidateQueries({ queryKey: expensesQueryKey });
        await qc.invalidateQueries({ queryKey: ["insights"] });
      }
    },
    onError: async (_err, _vars, ctx) => {
      if (!ctx) return;
      await removeQueued(ctx.queueId);
      qc.setQueryData<ExpensesData>(expensesQueryKey, ctx.previous);
    },
  });
}
