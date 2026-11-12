import { useMemo } from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";
import {
  createExpense,
  deleteExpense,
  listChanges,
  listExpenses,
  updateExpense,
  type ExpenseWriteResult,
} from "../api/expenses";
import { getMe } from "../api/auth";
import type {
  CreateExpenseInput,
  Expense,
  Insights,
  UpdateExpenseInput,
  User,
} from "../types";
import { deriveInsights } from "../insights/derive";

// The only expenses cache; every screen derives its view from it.
export const expensesQueryKey = ["expenses"] as const;

export const currentUserQueryKey = ["auth", "me"] as const;

// Infinity is safe: identity only changes on re-login, which clears the cache.
export function useCurrentUser(): UseQueryResult<User, Error> {
  return useQuery<User, Error>({
    queryKey: currentUserQueryKey,
    queryFn: () => getMe(),
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

// Rows with no user_id count as the user's own.
export function isOwnExpense(e: Expense, me: User | null | undefined): boolean {
  return e.user_id == null || (me != null && e.user_id === me.id);
}

// Server time (RFC3339Nano) of the last full or diff sync. Mutations must not
// advance it: their server time would skip rows other users wrote meanwhile.
export const lastSyncKey = ["expenses", "lastSyncAt"] as const;

// Must mirror the server's (date DESC, id DESC) order.
function sortExpenses(arr: Expense[]): Expense[] {
  const out = [...arr];
  out.sort((a, b) => {
    if (a.date !== b.date) return a.date < b.date ? 1 : -1;
    return b.id - a.id;
  });
  return out;
}

function upsertExpense(prev: Expense[] | undefined, next: Expense): Expense[] {
  const base = prev ?? [];
  const without = base.filter((e) => e.id !== next.id);
  return sortExpenses([...without, next]);
}

export function mergeChanges(
  prev: Expense[] | undefined,
  updated: Expense[],
  deletedIds: number[],
): Expense[] {
  const base = prev ?? [];
  const deletedSet = new Set(deletedIds);
  const updatedById = new Map(
    updated.filter((e) => !deletedSet.has(e.id)).map((e) => [e.id, e]),
  );
  const out: Expense[] = [];
  for (const e of base) {
    if (deletedSet.has(e.id)) continue;
    const replacement = updatedById.get(e.id);
    if (replacement) {
      out.push(replacement);
      updatedById.delete(e.id);
    } else {
      out.push(e);
    }
  }
  for (const fresh of updatedById.values()) {
    out.push(fresh);
  }
  return sortExpenses(out);
}

function advanceLastSync(qc: QueryClient, serverTime: string | null): void {
  if (!serverTime) return;
  // Nothing observes this key, so the default 5-minute gcTime would drop it
  // and silently turn every later sync into a no-op.
  qc.setQueryDefaults(lastSyncKey, { gcTime: Infinity });
  const current = qc.getQueryData<string>(lastSyncKey);
  // Only move forward; a late response must not rewind it. Both are RFC3339Nano UTC.
  if (!current || serverTime > current) {
    qc.setQueryData<string>(lastSyncKey, serverTime);
  }
}

// Fetched once per session; later refreshes go through useSyncExpenses.
export function useAllExpenses(): UseQueryResult<Expense[], Error> {
  const qc = useQueryClient();
  return useQuery<Expense[], Error>({
    queryKey: expensesQueryKey,
    queryFn: async () => {
      const page = await listExpenses();
      advanceLastSync(qc, page.serverTime);
      return sortExpenses(page.items);
    },
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

export type SyncExpensesMutation = UseMutationResult<void, Error, void, unknown>;

export function useSyncExpenses(): SyncExpensesMutation {
  const qc = useQueryClient();
  return useMutation<void, Error, void>({
    mutationKey: ["expenses", "sync"],
    mutationFn: async () => {
      const since = qc.getQueryData<string>(lastSyncKey);
      if (!since) return;
      const changes = await listChanges(since);
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        mergeChanges(prev, changes.updated, changes.deletedIds),
      );
      advanceLastSync(qc, changes.serverTime);
    },
  });
}

export function useInsightsFor(
  year: number,
  month: number,
  today: Date,
): { data: Insights | undefined } {
  const query = useAllExpenses();
  const data = useMemo(() => {
    if (!query.data) return undefined;
    return deriveInsights(query.data, year, month, today);
  }, [query.data, year, month, today]);
  return { data };
}

export function useCategoryView(
  year: number,
  month: number | null,
  label: string,
  mineOnly = false,
): {
  items: Expense[];
  total: number;
  count: number;
  monthTotal: number;
  pct: number;
  isLoading: boolean;
} {
  const query = useAllExpenses();
  const meQuery = useCurrentUser();
  const me = meQuery.data;
  const view = useMemo(() => {
    const all = mineOnly
      ? (query.data ?? []).filter((e) => isOwnExpense(e, me))
      : (query.data ?? []);
    const inScope = (iso: string): boolean => {
      if (!iso || iso.length < 10) return false;
      if (parseInt(iso.slice(0, 4), 10) !== year) return false;
      if (month != null && parseInt(iso.slice(5, 7), 10) !== month) {
        return false;
      }
      return true;
    };
    const items = all.filter((e) => e.category === label && inScope(e.date));
    let total = 0;
    for (const e of items) total += e.amount;
    let monthTotal = 0;
    for (const e of all) {
      if (inScope(e.date)) monthTotal += e.amount;
    }
    const pct = monthTotal > 0 ? (total / monthTotal) * 100 : 0;
    return { items, total, count: items.length, monthTotal, pct };
  }, [query.data, year, month, label, mineOnly, me]);
  // Mine-scoped numbers are wrong until the current user is known.
  return {
    ...view,
    isLoading: query.isLoading || (mineOnly && meQuery.isLoading),
  };
}

export type CreateExpenseMutation = UseMutationResult<
  ExpenseWriteResult<Expense>,
  Error,
  CreateExpenseInput,
  unknown
>;

export type UpdateExpenseMutation = UseMutationResult<
  ExpenseWriteResult<Expense>,
  Error,
  { id: number; patch: UpdateExpenseInput },
  unknown
>;

export type DeleteExpenseMutation = UseMutationResult<
  ExpenseWriteResult<void>,
  Error,
  number,
  unknown
>;

// No optimistic updates: the cache only holds rows the server confirmed.
export function useCreateExpense(): CreateExpenseMutation {
  const qc = useQueryClient();
  return useMutation<ExpenseWriteResult<Expense>, Error, CreateExpenseInput>({
    mutationKey: ["expenses", "create"],
    mutationFn: (input) => createExpense(input),
    onSuccess: ({ data }) => {
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        upsertExpense(prev, data),
      );
    },
  });
}

export function useUpdateExpense(): UpdateExpenseMutation {
  const qc = useQueryClient();
  return useMutation<
    ExpenseWriteResult<Expense>,
    Error,
    { id: number; patch: UpdateExpenseInput }
  >({
    mutationKey: ["expenses", "update"],
    mutationFn: ({ id, patch }) => updateExpense(id, patch),
    onSuccess: ({ data }) => {
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        upsertExpense(prev, data),
      );
    },
  });
}

export function useDeleteExpense(): DeleteExpenseMutation {
  const qc = useQueryClient();
  return useMutation<ExpenseWriteResult<void>, Error, number>({
    mutationKey: ["expenses", "delete"],
    mutationFn: (id) => deleteExpense(id),
    onSuccess: (_result, id) => {
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        (prev ?? []).filter((e) => e.id !== id),
      );
    },
  });
}
