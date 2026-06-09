import { request, requestWithMeta } from "./client";
import type {
  CreateExpenseInput,
  Expense,
  ExpenseChanges,
  ExpensePage,
  UpdateExpenseInput,
} from "../types";

// listExpenses returns every live expense owned by the authenticated user.
// The response carries `serverTime`; the hook pins this as the initial
// lastSyncAt. From then on the Feed calls listChanges(lastSyncAt) on mount
// to pick up anything modified elsewhere (another device, another tab).
export function listExpenses(): Promise<ExpensePage> {
  return request<ExpensePage>("/api/expenses");
}

// listChanges is the delta-sync endpoint. Client passes its lastSyncAt;
// server returns only rows changed since then plus the ids of soft-deleted
// rows. The included `serverTime` becomes the new lastSyncAt.
export function listChanges(since: string): Promise<ExpenseChanges> {
  return request<ExpenseChanges>("/api/expenses/changes", {
    query: { since },
  });
}

export function getExpense(id: number): Promise<Expense> {
  return request<Expense>(`/api/expenses/${id}`);
}

// ExpenseWriteResult is the shape every mutation returns to the React
// Query layer: the canonical row (or null for deletes) plus the
// X-Server-Time the backend stamped on this response. The sync hook
// advances lastSyncAt off this value so a later Feed diff doesn't re-emit
// rows that were just written locally.
export type ExpenseWriteResult<T> = {
  data: T;
  serverTime: string | null;
};

export async function createExpense(
  input: CreateExpenseInput,
): Promise<ExpenseWriteResult<Expense>> {
  const { data, meta } = await requestWithMeta<Expense>("/api/expenses", {
    method: "POST",
    body: input,
  });
  return { data, serverTime: meta.serverTime };
}

export async function updateExpense(
  id: number,
  input: UpdateExpenseInput,
): Promise<ExpenseWriteResult<Expense>> {
  const { data, meta } = await requestWithMeta<Expense>(
    `/api/expenses/${id}`,
    { method: "PATCH", body: input },
  );
  return { data, serverTime: meta.serverTime };
}

export async function deleteExpense(
  id: number,
): Promise<ExpenseWriteResult<void>> {
  const { data, meta } = await requestWithMeta<void>(`/api/expenses/${id}`, {
    method: "DELETE",
  });
  return { data, serverTime: meta.serverTime };
}
