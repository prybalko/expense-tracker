import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { theme, FONT } from "../theme";
import { Hero } from "../components/Hero";
import { DayGroup } from "../components/DayGroup";
import { TabBar } from "../components/TabBar";
import { useAllExpenses, useInsightsFor } from "../hooks/useExpenses";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
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

const PAGE = 50;

export function Feed() {
  const t = theme;
  const navigate = useNavigate();
  const today = useMemo(() => new Date(), []);
  const insights = useInsightsFor(today.getFullYear(), today.getMonth() + 1);
  const expenses = useAllExpenses();
  const lookup = useCategoryLookup();
  const slugFor = lookup.slugByLabel;

  // Windowed render: bump `visible` as the sentinel scrolls into view.
  // Replaces the old useInfiniteQuery + cursor pagination — the data is
  // already fully in memory once the all-expenses query lands, so all we
  // need here is a slice + a counter.
  const [visible, setVisible] = useState(PAGE);
  const allItems = useMemo<Expense[]>(
    () => expenses.data ?? [],
    [expenses.data],
  );
  const visibleItems = useMemo(
    () => allItems.slice(0, visible),
    [allItems, visible],
  );
  const grouped = useMemo(() => groupByDay(visibleItems), [visibleItems]);
  const hasMore = visible < allItems.length;

  const sentinelRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || !hasMore) return;
    const obs = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setVisible((v) => v + PAGE);
          }
        }
      },
      { rootMargin: "200px" },
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [hasMore]);

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
        height: "100dvh",
        overflow: "hidden",
        background: t.bg,
        color: t.ink,
        fontFamily: FONT,
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div
        className="scroll-y"
        style={{
          flex: 1,
          padding: "calc(12px + env(safe-area-inset-top)) 16px 12px",
        }}
      >
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
