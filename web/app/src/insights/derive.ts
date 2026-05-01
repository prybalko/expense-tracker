// Pure derivations from the in-memory expenses array. The all-in refactor
// moved aggregation off the server (was internal/insights) and into the
// client so that one fetch can drive Feed, Insights, and CategoryDetails
// without coordinating multiple queries on every navigation. Keep these
// functions pure and timezone-honest by parsing dates as YYYY-MM-DD
// calendar prefixes (matching the convention in EntryForm.toIsoDateTime
// and groupByDay.ts) rather than constructing Date objects, which would
// silently shift across midnight in non-UTC zones.

import type {
  CategoryBreakdown,
  ChartPoint,
  Expense,
  Insights,
} from "../types";

const MONTH_NAMES = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

type YMD = { year: number; month: number; day: number };

function parseYMD(iso: string): YMD | null {
  if (!iso || iso.length < 10) return null;
  const y = parseInt(iso.slice(0, 4), 10);
  const m = parseInt(iso.slice(5, 7), 10);
  const d = parseInt(iso.slice(8, 10), 10);
  if (!Number.isFinite(y) || !Number.isFinite(m) || !Number.isFinite(d)) {
    return null;
  }
  return { year: y, month: m, day: d };
}

// Days in a given calendar month. Date.UTC(year, monthIdx, 0) returns the
// last day of the previous month, so passing the 1-based month gives the
// last day of `month` itself (e.g. month=2 → Feb 29 in a leap year).
function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

function isInMonth(iso: string, year: number, month: number): boolean {
  const ymd = parseYMD(iso);
  return ymd !== null && ymd.year === year && ymd.month === month;
}

// Mirrors internal/insights.percentChange: returns the absolute percentage
// change, its direction, and a `hasChange` flag that is false whenever the
// previous period had no spending (avoids dividing by zero and avoids
// surfacing meaningless "+infinity%" deltas in the UI).
function percentChange(current: number, previous: number) {
  if (previous <= 0) return { pct: 0, isIncrease: false, hasChange: false };
  const raw = ((current - previous) / previous) * 100;
  return { pct: Math.abs(raw), isIncrease: raw > 0, hasChange: true };
}

export function expensesForCategory(
  expenses: Expense[],
  year: number,
  month: number,
  label: string,
): Expense[] {
  return expenses.filter(
    (e) => e.category === label && isInMonth(e.date, year, month),
  );
}

function sumForMonthUpToDay(
  expenses: Expense[],
  year: number,
  month: number,
  upToDayInclusive: number,
): number {
  let sum = 0;
  for (const e of expenses) {
    const ymd = parseYMD(e.date);
    if (!ymd) continue;
    if (ymd.year !== year || ymd.month !== month) continue;
    if (ymd.day > upToDayInclusive) continue;
    sum += e.amount;
  }
  return sum;
}

function sumForMonth(
  expenses: Expense[],
  year: number,
  month: number,
): number {
  let sum = 0;
  for (const e of expenses) {
    if (isInMonth(e.date, year, month)) sum += e.amount;
  }
  return sum;
}

// deriveInsights produces the same shape that the deleted Go endpoint
// (internal/insights.Month) used to return, so the consuming screens
// (Insights.tsx, Feed Hero) require no JSX changes. `now` is the caller's
// local "today" — Insights.tsx already memoises `new Date()` once per mount
// and we use the local day for the elapsed-day average and the
// "is current period" branch so the Insights chart's "today" highlight
// stays in sync with what the user sees on their device.
export function deriveInsights(
  expenses: Expense[],
  year: number,
  month: number,
  now: Date,
): Insights {
  const isCurrentPeriod =
    year === now.getFullYear() && month === now.getMonth() + 1;
  const days = daysInMonth(year, month);
  const todayDay = now.getDate();

  // Single pass: per-category totals, per-day totals, grand total.
  const categoryTotals = new Map<string, { total: number; count: number }>();
  const dailyMap = new Map<number, number>();
  let total = 0;
  for (const e of expenses) {
    const ymd = parseYMD(e.date);
    if (!ymd || ymd.year !== year || ymd.month !== month) continue;
    total += e.amount;
    const existing = categoryTotals.get(e.category);
    if (existing) {
      existing.total += e.amount;
      existing.count += 1;
    } else {
      categoryTotals.set(e.category, { total: e.amount, count: 1 });
    }
    dailyMap.set(ymd.day, (dailyMap.get(ymd.day) ?? 0) + e.amount);
  }

  const prevYear = month === 1 ? year - 1 : year;
  const prevMonth = month === 1 ? 12 : month - 1;
  const prevDays = daysInMonth(prevYear, prevMonth);

  // For the current period we compare month-to-date against the same number
  // of elapsed days in the previous month (clamped to that month's length).
  // Otherwise compare full month vs full previous month.
  let compareTotal = total;
  let prevTotal: number;
  if (isCurrentPeriod) {
    const cmpDays = Math.min(todayDay, prevDays);
    compareTotal = sumForMonthUpToDay(expenses, year, month, cmpDays);
    prevTotal = sumForMonthUpToDay(expenses, prevYear, prevMonth, cmpDays);
  } else {
    prevTotal = sumForMonth(expenses, prevYear, prevMonth);
  }

  const { pct, isIncrease, hasChange } = percentChange(compareTotal, prevTotal);

  const avgDivisor = isCurrentPeriod ? todayDay : days;
  const averageSpending = avgDivisor > 0 ? total / avgDivisor : 0;

  let maxChartValue = 0;
  const chart: ChartPoint[] = new Array(days);
  for (let day = 1; day <= days; day++) {
    const value = dailyMap.get(day) ?? 0;
    if (value > maxChartValue) maxChartValue = value;
    const label =
      day === 1 || day === 10 || day === 20 || day === days ? String(day) : "";
    chart[day - 1] = { label, value };
  }

  // Sort categories by total DESC to match the original SQL ORDER BY.
  const categories: CategoryBreakdown[] = Array.from(categoryTotals.entries())
    .map(([category, { total: catTotal, count }]) => ({
      category,
      total: catTotal,
      count,
      percentage: total > 0 ? (catTotal / total) * 100 : 0,
    }))
    .sort((a, b) => b.total - a.total);

  const nextYear = month === 12 ? year + 1 : year;
  const nextMonth = month === 12 ? 1 : month + 1;

  return {
    viewMode: "month",
    year,
    month,
    monthName: MONTH_NAMES[month - 1] ?? "",
    total,
    percentageChange: pct,
    isIncrease,
    hasChange,
    averageSpending,
    averageLabel: "SPENT/DAY",
    categories,
    chart,
    maxChartValue,
    isCurrentPeriod,
    prevYear,
    prevMonth,
    nextYear,
    nextMonth,
  };
}
