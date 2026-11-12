// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import type { ReactNode } from "react";
import { renderHook, waitFor, act, cleanup } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Expense } from "../types";

const api = vi.hoisted(() => ({
  listExpenses: vi.fn(),
  listChanges: vi.fn(),
  createExpense: vi.fn(),
  updateExpense: vi.fn(),
  deleteExpense: vi.fn(),
}));
vi.mock("../api/expenses", () => api);

import {
  expensesQueryKey,
  mergeChanges,
  useAllExpenses,
  useCreateExpense,
  useSyncExpenses,
} from "./useExpenses";

const T0 = "2026-11-12T10:00:00Z";
const T1 = "2026-11-12T10:01:00Z";
const T2 = "2026-11-12T10:02:00Z";
const T3 = "2026-11-12T10:03:00Z";

function row(id: number, userId: number, updatedAt: string): Expense {
  return {
    id,
    amount: 10,
    description: `row ${id}`,
    category: "Other",
    date: "2026-11-12T00:00:00Z",
    user_id: userId,
    updated_at: updatedAt,
  } as Expense;
}

function setup() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
  const hook = renderHook(
    () => ({
      list: useAllExpenses(),
      sync: useSyncExpenses(),
      create: useCreateExpense(),
    }),
    { wrapper },
  );
  return { qc, hook };
}

describe("delta sync cursor", () => {
  beforeEach(() => {
    for (const fn of Object.values(api)) fn.mockReset();
    api.listExpenses.mockResolvedValue({ items: [], nextCursor: null, serverTime: T0 });
    api.listChanges.mockResolvedValue({ updated: [], deletedIds: [], serverTime: T3 });
  });
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("an own write does not move the cursor past a peer's earlier row", async () => {
    const { qc, hook } = setup();
    await waitFor(() => expect(hook.result.current.list.isSuccess).toBe(true));

    api.createExpense.mockResolvedValue({ data: row(2, 1, T2), serverTime: T2 });
    await act(() => hook.result.current.create.mutateAsync({
      amount: 10, description: "mine", category: "Other",
    }));

    const peer = row(1, 2, T1);
    api.listChanges.mockImplementation(async (since: string) => ({
      updated: since < T1 ? [peer] : [],
      deletedIds: [],
      serverTime: T3,
    }));
    await act(() => hook.result.current.sync.mutateAsync());
    expect(api.listChanges).toHaveBeenCalledWith(T0);
    expect(qc.getQueryData<Expense[]>(expensesQueryKey)?.map((e) => e.id))
      .toEqual([2, 1]);
  });

  it("keeps the cursor after the default 5-minute gc window", async () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true });
    const { hook } = setup();
    await waitFor(() => expect(hook.result.current.list.isSuccess).toBe(true));

    act(() => {
      vi.advanceTimersByTime(6 * 60 * 1000);
    });

    await act(() => hook.result.current.sync.mutateAsync());
    expect(api.listChanges).toHaveBeenCalledWith(T0);
    expect(api.listExpenses).toHaveBeenCalledTimes(1);
  });
});

describe("mergeChanges", () => {
  it("drops a row that is in both the updated and the deleted bucket", () => {
    const r = row(1, 2, T1);
    expect(mergeChanges([r], [r], [1])).toEqual([]);
    expect(mergeChanges([], [r], [1])).toEqual([]);
  });
});
