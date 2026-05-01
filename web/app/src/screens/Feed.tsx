import { useEffect, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { theme, FONT } from "../theme";
import { Hero } from "../components/Hero";
import { DayGroup } from "../components/DayGroup";
import { TabBar } from "../components/TabBar";
import { useExpenses } from "../hooks/useExpenses";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
import { useInsights } from "../hooks/useInsights";
import { dayLabel, groupByDay } from "../groupByDay";
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
  const lookup = useCategoryLookup();
  const slugFor = lookup.slugByLabel;

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
      data-testid="feed-screen"
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
