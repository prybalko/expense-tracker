// @vitest-environment jsdom
import { describe, it, expect, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";

import { ExpenseRow } from "./ExpenseRow";
import { currentUserQueryKey } from "../hooks/useExpenses";
import type { Expense, User } from "../types";

const ME: User = { id: 1, username: "me" };

function makeExpense(overrides: Partial<Expense>): Expense {
  return {
    id: 100,
    amount: 12.5,
    description: "Test expense",
    category: "Groceries",
    date: "2026-05-01",
    updated_at: "2026-05-01T00:00:00Z",
    ...overrides,
  };
}

// Pre-seed the auth/me cache so useCurrentUser resolves synchronously
// without firing a real fetch. retry:false keeps the cache strict if a
// test ever forgets to seed.
function renderWithUser(ui: ReactElement, me: User | null) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  if (me) qc.setQueryData(currentUserQueryKey, me);
  return render(
    <QueryClientProvider client={qc}>{ui}</QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
});

describe("ExpenseRow author tint", () => {
  it("does not tint when the expense is mine", () => {
    renderWithUser(
      <ExpenseRow expense={makeExpense({ user_id: ME.id })} slug="groceries" />,
      ME,
    );
    expect(screen.getByTestId("expense-row").getAttribute("data-not-mine"))
      .toBeNull();
  });

  it("does not tint legacy unowned rows (user_id == null)", () => {
    renderWithUser(
      <ExpenseRow expense={makeExpense({ user_id: null })} slug="groceries" />,
      ME,
    );
    expect(screen.getByTestId("expense-row").getAttribute("data-not-mine"))
      .toBeNull();
  });

  it("tints rows authored by someone else", () => {
    renderWithUser(
      <ExpenseRow expense={makeExpense({ user_id: 999 })} slug="groceries" />,
      ME,
    );
    expect(screen.getByTestId("expense-row").getAttribute("data-not-mine"))
      .toBe("true");
  });

  it("does not tint while the current user is still loading", () => {
    renderWithUser(
      <ExpenseRow expense={makeExpense({ user_id: 999 })} slug="groceries" />,
      null,
    );
    expect(screen.getByTestId("expense-row").getAttribute("data-not-mine"))
      .toBeNull();
  });
});
