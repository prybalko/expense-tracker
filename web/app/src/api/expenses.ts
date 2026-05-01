import { request } from "./client";
import type {
  CreateExpenseInput,
  Expense,
  ExpensePage,
  UpdateExpenseInput,
} from "../types";

// listExpenses returns every expense owned by the authenticated user. The
// server takes no filter / pagination params: the client caches the array
// under a single React Query key and Feed / Insights / CategoryDetails all
// derive their views from it locally. ExpensePage is preserved as the
// response wrapper (with a permanently null nextCursor) so the type didn't
// have to churn during the move off pagination.
export function listExpenses(): Promise<ExpensePage> {
  return request<ExpensePage>("/api/expenses");
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
