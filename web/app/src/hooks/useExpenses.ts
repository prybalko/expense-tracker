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

// Single canonical cache for every read in the app. Feed shows a windowed
// slice of this array; Insights and CategoryDetails derive their views via
// useMemo selectors. There is no separate insights cache, so mutations only
// touch this one key — and screens never coordinate two queries that
// resolve at different times (the source of the old "number jumps" UX).
export const expensesQueryKey = ["expenses"] as const;

// currentUserQueryKey holds the signed-in user fetched via /api/auth/me.
// ExpenseRow reads this to decide whether each row was authored by someone
// else (and should be tinted). It stays in cache for the lifetime of the
// session and is wiped by Login.tsx's queryClient.clear() on a fresh login.
export const currentUserQueryKey = ["auth", "me"] as const;

// useCurrentUser returns the signed-in user. ExpenseRow calls it once per
// row, but React Query dedupes by key — there's only ever one in-flight
// /api/auth/me request and the result is reused. staleTime/gcTime are
// Infinity because the identity doesn't change without re-login (which
// clears the whole cache via Login.tsx).
export function useCurrentUser(): UseQueryResult<User, Error> {
  return useQuery<User, Error>({
    queryKey: currentUserQueryKey,
    queryFn: () => getMe(),
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

// lastSyncKey holds the server's wall-clock at the moment the client last
// successfully synced. The full-list query populates it on cold start; the
// diff hook advances it on every successful /changes call; every mutation
// advances it off X-Server-Time. The value is whatever string the server
// handed back (RFC3339Nano) — the client never derives or compares it to
// its own clock. Stays in the query cache rather than a module variable so
// React Query devtools + multi-tab cache inspection surface it the same
// way as the rest of the sync state.
export const lastSyncKey = ["expenses", "lastSyncAt"] as const;

// sortExpenses mirrors the server's (date DESC, id DESC) ordering so that
// upserts from mutations / diffs don't visually reshuffle the Feed in ways
// the server wouldn't. Keeping the comparator lexical on `date` relies on
// the RFC3339Nano wire format being sortable as a plain string.
function sortExpenses(arr: Expense[]): Expense[] {
  const out = [...arr];
  out.sort((a, b) => {
    if (a.date !== b.date) return a.date < b.date ? 1 : -1;
    return b.id - a.id;
  });
  return out;
}

// upsertExpense replaces a row with the same id if present, otherwise
// inserts. Used for the happy path of create / update mutations where the
// server hands back exactly one row. Re-sorts so the Feed doesn't briefly
// render the new row in insertion order before the next render.
function upsertExpense(prev: Expense[] | undefined, next: Expense): Expense[] {
  const base = prev ?? [];
  const without = base.filter((e) => e.id !== next.id);
  return sortExpenses([...without, next]);
}

// mergeChanges applies a delta-sync payload to the cache in one pass:
// deletions pruned, updates upserted (covers both "new since lastSyncAt"
// and "edited since lastSyncAt"), everything else preserved. One re-sort
// at the end keeps the rendered order canonical.
export function mergeChanges(
  prev: Expense[] | undefined,
  updated: Expense[],
  deletedIds: number[],
): Expense[] {
  const base = prev ?? [];
  const deletedSet = new Set(deletedIds);
  const updatedById = new Map(updated.map((e) => [e.id, e]));
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
  const current = qc.getQueryData<string>(lastSyncKey);
  // Only move the marker forward. A stale mutation response (slow network
  // overtaken by a later diff) could otherwise rewind lastSyncAt and cause
  // the next diff to re-emit rows we already have. String compare is safe
  // because both values are RFC3339Nano UTC.
  if (!current || serverTime > current) {
    qc.setQueryData<string>(lastSyncKey, serverTime);
  }
}

// useAllExpenses owns the first fetch. staleTime + gcTime: Infinity pin
// the cache for the lifetime of the session so React Query never refetches
// the full list behind our back; every post-initial refresh lives in
// useSyncExpenses, which calls the diff endpoint and merges. That way a
// five-minute-long navigation session never repays the full-list cost,
// and newly-edited rows still show up on Feed entry.
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

// useSyncExpenses exposes a manually-triggered delta fetch. The Feed
// screen calls this from its mount effect; on success the cache is
// merged in place and lastSyncAt advances. The hook short-circuits
// gracefully when lastSyncAt isn't set yet (the cold-start fetch is
// still in flight) so navigating to Feed mid-load doesn't double-request.
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

// useInsightsFor reproduces the shape the deleted /api/insights endpoint
// used to return, derived in O(n) from the cached array. Switching months
// on the Insights screen recomputes from cache — zero network, zero
// loading state — so the rendered totals never snap back to 0 between
// query keys.
export function useInsightsFor(
  year: number,
  month: number,
): { data: Insights | undefined } {
  const query = useAllExpenses();
  const data = useMemo(() => {
    if (!query.data) return undefined;
    return deriveInsights(query.data, year, month, new Date());
  }, [query.data, year, month]);
  return { data };
}

// useCategoryView returns CategoryDetails' four header numbers from a
// single useMemo over one array on one render — `pct` can never flicker
// because the numerator (filtered total) and denominator (month total)
// are computed in the same pass.
export function useCategoryView(
  year: number,
  month: number | null,
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
    // Single month, or the whole year when `month` is null.
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
  }, [query.data, year, month, label]);
  return { ...view, isLoading: query.isLoading };
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

// useCreateExpense POSTs to the server and only touches the cache on
// success. Optimistic updates were removed when the app moved to
// wait-for-server writes: the user sees "Adding..." → banner-or-row, with
// no phantom temp-id row in between. Insert copy has to absorb the full
// round-trip latency now, but the cache never shows a row the server
// hasn't confirmed — which is the behaviour the error banner assumes.
export function useCreateExpense(): CreateExpenseMutation {
  const qc = useQueryClient();
  return useMutation<ExpenseWriteResult<Expense>, Error, CreateExpenseInput>({
    mutationKey: ["expenses", "create"],
    mutationFn: (input) => createExpense(input),
    onSuccess: ({ data, serverTime }) => {
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        upsertExpense(prev, data),
      );
      advanceLastSync(qc, serverTime);
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
    onSuccess: ({ data, serverTime }) => {
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        upsertExpense(prev, data),
      );
      advanceLastSync(qc, serverTime);
    },
  });
}

export function useDeleteExpense(): DeleteExpenseMutation {
  const qc = useQueryClient();
  return useMutation<ExpenseWriteResult<void>, Error, number>({
    mutationKey: ["expenses", "delete"],
    mutationFn: (id) => deleteExpense(id),
    onSuccess: ({ serverTime }, id) => {
      qc.setQueryData<Expense[]>(expensesQueryKey, (prev) =>
        (prev ?? []).filter((e) => e.id !== id),
      );
      advanceLastSync(qc, serverTime);
    },
  });
}
