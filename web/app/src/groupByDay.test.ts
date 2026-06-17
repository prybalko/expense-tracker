import { describe, it, expect } from "vitest";
import { dayLabel, groupByDay } from "./groupByDay";
import type { Expense } from "./types";

function exp(id: number, date: string): Expense {
  return {
    id,
    amount: 1,
    category: "Groceries",
    date,
    description: "",
    updated_at: `${date}T00:00:00Z`,
  };
}

describe("dayLabel", () => {
  const today = new Date(2026, 5, 17); // 17 Jun 2026

  it('labels the current calendar day "Today"', () => {
    expect(dayLabel("2026-06-17", today)).toBe("Today");
  });

  it('labels the prior calendar day "Yesterday"', () => {
    expect(dayLabel("2026-06-16", today)).toBe("Yesterday");
  });

  it("labels older days with a weekday/day/month string", () => {
    expect(dayLabel("2026-06-10", today)).toMatch(/Jun/);
  });

  it("returns the raw value for a malformed date", () => {
    expect(dayLabel("not-a-date", today)).toBe("not-a-date");
  });

  // Regression guard for the iOS-PWA stale-"today" bug. dayLabel is correct as
  // long as it is handed a fresh `today`; the bug was that callers froze
  // `today` at the load day via useMemo(() => new Date(), []). useToday() now
  // re-derives `today` on resume — here we assert the labelling actually flips
  // once the day advances, which is what the user sees fixed.
  it("re-labels the same expense as the day advances", () => {
    const dayOf = new Date(2026, 5, 16); // app loaded on the 16th
    const dayAfter = new Date(2026, 5, 17); // resumed on the 17th

    expect(dayLabel("2026-06-16", dayOf)).toBe("Today");
    expect(dayLabel("2026-06-16", dayAfter)).toBe("Yesterday");
  });
});

describe("groupByDay", () => {
  it("groups by calendar day, newest day first", () => {
    const grouped = groupByDay([
      exp(1, "2026-06-15"),
      exp(2, "2026-06-17"),
      exp(3, "2026-06-17"),
    ]);

    expect(grouped.map((g) => g.day)).toEqual(["2026-06-17", "2026-06-15"]);
    expect(grouped[0].items.map((e) => e.id)).toEqual([2, 3]);
  });

  it("slices the YYYY-MM-DD prefix off full timestamps", () => {
    const grouped = groupByDay([exp(1, "2026-06-17T09:30:00.000Z")]);
    expect(grouped[0].day).toBe("2026-06-17");
  });
});
