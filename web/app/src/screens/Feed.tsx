import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { theme, FONT } from "../theme";
import { Hero } from "../components/Hero";
import { DayGroup } from "../components/DayGroup";
import { TabBar } from "../components/TabBar";
import { PullToRefreshIndicator } from "../components/PullToRefreshIndicator";
import {
  useAllExpenses,
  useInsightsFor,
  useSyncExpenses,
} from "../hooks/useExpenses";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
import { useErrorBanner } from "../hooks/useErrorBanner";
import { messageForReadError } from "../api/errors";
import { dayLabel, groupByDay } from "../groupByDay";
import { MONTH_NAMES } from "../dates";
import { StatusNote } from "../components/StatusNote";
import { usePullToRefresh } from "../hooks/usePullToRefresh";
import { useToday } from "../hooks/useToday";
import type { Expense } from "../types";

const PAGE = 50;

export function Feed() {
  const t = theme;
  const navigate = useNavigate();
  const today = useToday();
  const insights = useInsightsFor(today.getFullYear(), today.getMonth() + 1, today);
  const expenses = useAllExpenses();
  const sync = useSyncExpenses();
  const { showError } = useErrorBanner();
  const lookup = useCategoryLookup();
  const slugFor = lookup.slugByLabel;

  // Delta sync no longer fires on every Feed mount. The sync
  // is user-driven (pull-to-refresh) plus a visibilitychange handler
  // for the PWA-resumed-from-background case.
  const syncMutate = sync.mutate;
  const triggerSync = useCallback(() => {
    return new Promise<void>((resolve) => {
      let isSettled = false;
      const timer = setTimeout(() => {
        if (!isSettled) {
          isSettled = true;
          showError("Couldn't refresh. The request took too long.");
          resolve();
        }
      }, 10000); // 10s timeout for the pull-to-refresh spinner

      syncMutate(undefined, {
        onSuccess: () => {
          if (isSettled) return;
          isSettled = true;
          clearTimeout(timer);
          resolve();
        },
        onError: (err) => {
          if (isSettled) return;
          isSettled = true;
          clearTimeout(timer);
          showError(messageForReadError(err));
          resolve();
        },
      });
    });
  }, [syncMutate, showError]);

  const scrollRef = useRef<HTMLDivElement | null>(null);
  const pull = usePullToRefresh(scrollRef, triggerSync);
  // Resume-driven sync now lives app-wide in <ResumeSync/>; the Feed keeps
  // only the manual pull gesture.

  // Surface cold-start fetch failures the same way. Without this the Feed
  // would quietly sit on an empty state and the user would have no signal
  // that the app couldn't load their data.
  useEffect(() => {
    if (expenses.isError && expenses.error) {
      showError(messageForReadError(expenses.error));
    }
  }, [expenses.isError, expenses.error, showError]);

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
        ref={scrollRef}
        className="scroll-y"
        style={{
          flex: 1,
          padding: "calc(12px + env(safe-area-inset-top)) 16px 12px",
          transform: `translateY(${pull.pullDistance}px)`,
          transition: pull.pullDistance === 0 ? "transform 220ms ease-out" : "none",
        }}
      >
        <PullToRefreshIndicator
          pullDistance={pull.pullDistance}
          committed={pull.committed}
          isRefreshing={pull.isRefreshing}
        />
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
            <StatusNote>Loading...</StatusNote>
          ) : grouped.length === 0 ? (
            <StatusNote>No expenses yet. Tap + to add your first one.</StatusNote>
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
