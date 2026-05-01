import type { Expense } from "./types";

export function dayLabel(iso: string, today: Date): string {
  const [y, m, d] = iso.split("-").map((s) => parseInt(s, 10));
  if (!y || !m || !d) return iso;
  const date = new Date(y, m - 1, d);
  const t0 = new Date(today.getFullYear(), today.getMonth(), today.getDate());
  const diffDays = Math.round((t0.getTime() - date.getTime()) / 86400000);
  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "Yesterday";
  return date.toLocaleDateString("en-GB", {
    weekday: "short",
    day: "numeric",
    month: "short",
  });
}

export function groupByDay(
  expenses: Expense[],
): { day: string; items: Expense[] }[] {
  const map = new Map<string, Expense[]>();
  for (const e of expenses) {
    const day = (e.date || "").slice(0, 10);
    const arr = map.get(day);
    if (arr) arr.push(e);
    else map.set(day, [e]);
  }
  const days = Array.from(map.keys()).sort((a, b) => (a < b ? 1 : -1));
  return days.map((day) => ({ day, items: map.get(day)! }));
}
