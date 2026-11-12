import { request, requestWithMeta } from "./client";
import type {
  CreateExpenseInput,
  Expense,
  ExpenseChanges,
  ExpensePage,
  UpdateExpenseInput,
} from "../types";

export function listExpenses(): Promise<ExpensePage> {
  return request<ExpensePage>("/api/expenses");
}

export function listChanges(since: string): Promise<ExpenseChanges> {
  return request<ExpenseChanges>("/api/expenses/changes", {
    query: { since },
  });
}

export function getExpense(id: number): Promise<Expense> {
  return request<Expense>(`/api/expenses/${id}`);
}

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
