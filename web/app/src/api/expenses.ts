import { request } from "./client";
import type {
  CreateExpenseInput,
  Expense,
  ExpensePage,
  UpdateExpenseInput,
} from "../types";

export type ListExpensesParams = {
  limit?: number;
  before?: string | null;
};

export function listExpenses(
  params: ListExpensesParams = {},
): Promise<ExpensePage> {
  return request<ExpensePage>("/api/expenses", {
    query: {
      limit: params.limit,
      before: params.before ?? undefined,
    },
  });
}

export function getExpense(id: number): Promise<Expense> {
  return request<Expense>(`/api/expenses/${id}`);
}

export function createExpense(input: CreateExpenseInput): Promise<Expense> {
  return request<Expense>("/api/expenses", {
    method: "POST",
    body: input,
  });
}

export function updateExpense(
  id: number,
  input: UpdateExpenseInput,
): Promise<Expense> {
  return request<Expense>(`/api/expenses/${id}`, {
    method: "PATCH",
    body: input,
  });
}

export function deleteExpense(id: number): Promise<void> {
  return request<void>(`/api/expenses/${id}`, { method: "DELETE" });
}
