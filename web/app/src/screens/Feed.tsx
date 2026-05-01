import { useEffect, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { theme, FONT } from "../theme";
import { Hero } from "../components/Hero";
import { DayGroup } from "../components/DayGroup";
import { TabBar } from "../components/TabBar";
import { useExpenses } from "../hooks/useExpenses";
import { useCategories } from "../hooks/useCategories";
import { useInsights } from "../hooks/useInsights";
import type { Expense } from "../types";

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

function dayLabel(iso: string, today: Date): string {
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

function groupByDay(expenses: Expense[]): { day: string; items: Expense[] }[] {
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

export function Feed() {
  const t = theme;
  const navigate = useNavigate();
  const today = useMemo(() => new Date(), []);
  const insights = useInsights({
    view: "month",
    year: today.getFullYear(),
    month: today.getMonth() + 1,
  });
  const expenses = useExpenses(50);
  const { data: categories = [] } = useCategories();

  const slugFor = useMemo(() => {
    const map = new Map<string, string>();
    for (const c of categories) map.set(c.label, c.slug);
    return (label: string) => map.get(label) ?? "other";
  }, [categories]);

  const allItems: Expense[] = useMemo(
    () => expenses.data?.pages.flatMap((p) => p.items) ?? [],
    [expenses.data],
  );
  const grouped = useMemo(() => groupByDay(allItems), [allItems]);

  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const hasNextPage = expenses.hasNextPage;
  const isFetchingNextPage = expenses.isFetchingNextPage;
  const fetchNextPage = expenses.fetchNextPage;
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    if (!hasNextPage) return;
    const obs = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && !isFetchingNextPage) {
            fetchNextPage();
          }
        }
      },
      { rootMargin: "200px" },
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const heroMonth =
    insights.data?.monthName ?? MONTH_NAMES[today.getMonth()] ?? "";
  const heroTotal = insights.data?.total ?? 0;
  const prevMonthName =
    insights.data && typeof insights.data.prevMonth === "number"
      ? MONTH_NAMES[(insights.data.prevMonth - 1 + 12) % 12]
      : undefined;

  return (
    <div
      style={{
        minHeight: "100vh",
        background: t.bg,
        color: t.ink,
        fontFamily: FONT,
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div style={{ flex: 1, padding: "12px 16px 12px" }}>
        <Hero
          monthName={heroMonth}
          total={heroTotal}
          hasChange={insights.data?.hasChange ?? false}
          isIncrease={insights.data?.isIncrease ?? false}
          percentageChange={insights.data?.percentageChange ?? 0}
          prevMonthName={prevMonthName}
        />
        <div style={{ marginTop: 18 }}>
          {expenses.isLoading ? (
            <div
              style={{
                padding: "24px",
                textAlign: "center",
                color: t.ink2,
                fontSize: 13,
              }}
            >
              Loading...
            </div>
          ) : grouped.length === 0 ? (
            <div
              style={{
                padding: "32px 16px",
                textAlign: "center",
                color: t.ink2,
                fontSize: 13,
              }}
            >
              No expenses yet. Tap + to add your first one.
            </div>
          ) : (
            grouped.map(({ day, items }) => (
              <DayGroup
                key={day}
                label={dayLabel(day, today)}
                items={items}
                slugFor={slugFor}
                onItemClick={(e) => {
                  // Optimistic temp rows have negative ids and no server-side
                  // record yet — opening the edit form for them would route to
                  // /edit/-1 and the API rejects id <= 0. Wait for the create
                  // to land before allowing edits.
                  if (e.id < 0) return;
                  navigate(`/edit/${e.id}`);
                }}
              />
            ))
          )}
          <div ref={sentinelRef} style={{ height: 1 }} />
          {expenses.isFetchingNextPage ? (
            <div
              style={{
                padding: "16px",
                textAlign: "center",
                color: t.ink2,
                fontSize: 12,
              }}
            >
              Loading more...
            </div>
          ) : null}
        </div>
      </div>
      <TabBar
        current="feed"
        onNavigate={(id) => navigate(id === "feed" ? "/" : "/insights")}
        onAdd={() => navigate("/add")}
      />
    </div>
  );
}
