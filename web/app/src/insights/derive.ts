// Parse dates as YYYY-MM-DD prefixes, never via Date, which shifts across
// midnight in non-UTC zones.

import type {
  CategoryBreakdown,
  ChartPoint,
  Expense,
  Insights,
  YearInsights,
} from "../types";
import { MONTH_NAMES } from "../dates";

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

// Day 0 of the 0-based month index is the last day of the 1-based `month`.
function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

function isInMonth(iso: string, year: number, month: number): boolean {
  const ymd = parseYMD(iso);
  return ymd !== null && ymd.year === year && ymd.month === month;
}

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

  // Current period: month-to-date vs the same elapsed days of last month.
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

function sumForYear(expenses: Expense[], year: number): number {
  let sum = 0;
  for (const e of expenses) {
    const ymd = parseYMD(e.date);
    if (ymd && ymd.year === year) sum += e.amount;
  }
  return sum;
}

function sumForYearUpToMonth(
  expenses: Expense[],
  year: number,
  upToMonthInclusive: number,
): number {
  let sum = 0;
  for (const e of expenses) {
    const ymd = parseYMD(e.date);
    if (!ymd || ymd.year !== year) continue;
    if (ymd.month > upToMonthInclusive) continue;
    sum += e.amount;
  }
  return sum;
}

export function deriveYearInsights(
  expenses: Expense[],
  year: number,
  now: Date,
): YearInsights {
  const isCurrentYear = year === now.getFullYear();
  const elapsedMonths = isCurrentYear ? now.getMonth() + 1 : 12;

  const series = new Array<number>(12).fill(0);
  const categoryTotals = new Map<string, { total: number; count: number }>();
  let total = 0;
  for (const e of expenses) {
    const ymd = parseYMD(e.date);
    if (!ymd || ymd.year !== year) continue;
    total += e.amount;
    if (ymd.month >= 1 && ymd.month <= 12) series[ymd.month - 1] += e.amount;
    const existing = categoryTotals.get(e.category);
    if (existing) {
      existing.total += e.amount;
      existing.count += 1;
    } else {
      categoryTotals.set(e.category, { total: e.amount, count: 1 });
    }
  }

  const prevYear = year - 1;
  // Current year: day-clamp the trailing in-progress month on both sides so it
  // can't drag the YoY delta down. Past year: full vs full.
  const todayDay = now.getDate();
  const prevTotal = isCurrentYear
    ? sumForYearUpToMonth(expenses, prevYear, elapsedMonths - 1) +
      sumForMonthUpToDay(expenses, prevYear, elapsedMonths, todayDay)
    : sumForYear(expenses, prevYear);
  const { pct, isIncrease, hasChange } = percentChange(total, prevTotal);

  const averageSpending = elapsedMonths > 0 ? total / elapsedMonths : 0;

  const categories: CategoryBreakdown[] = Array.from(categoryTotals.entries())
    .map(([category, { total: catTotal, count }]) => ({
      category,
      total: catTotal,
      count,
      percentage: total > 0 ? (catTotal / total) * 100 : 0,
    }))
    .sort((a, b) => b.total - a.total);

  return {
    year,
    total,
    averageSpending,
    series,
    elapsedMonths,
    categories,
    percentageChange: pct,
    isIncrease,
    hasChange,
    isCurrentYear,
    prevYear,
  };
}
