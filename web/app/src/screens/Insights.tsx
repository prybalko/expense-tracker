import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { theme, FONT } from "../theme";
import { TabBar } from "../components/TabBar";
import { CategoryGlyph } from "../components/CategoryGlyph";
import { useInsightsFor } from "../hooks/useExpenses";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
import { fmtEUR } from "../format";

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

export function Insights() {
  const t = theme;
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const today = useMemo(() => new Date(), []);
  // Initial period: prefer ?year=&month= (set by the back-arrow on
  // CategoryDetails so we land on the same month the user drilled in from),
  // otherwise default to "this month."
  const [period, setPeriod] = useState(() => {
    const y = parseInt(searchParams.get("year") ?? "", 10);
    const m = parseInt(searchParams.get("month") ?? "", 10);
    if (Number.isFinite(y) && y > 0 && Number.isFinite(m) && m >= 1 && m <= 12) {
      return { year: y, month: m };
    }
    return { year: today.getFullYear(), month: today.getMonth() + 1 };
  });
  // Derived from the cached all-expenses array — switching months recomputes
  // off cache instead of issuing a new request, so the totals/chart never
  // snap back to 0 between periods.
  const insights = useInsightsFor(period.year, period.month);
  const lookup = useCategoryLookup();

  const data = insights.data;
  const monthLabel = data?.monthName ?? MONTH_NAMES[period.month - 1] ?? "";
  const total = data?.total ?? 0;
  const avgPerDay = data?.averageSpending ?? 0;
  const avgLabel = data?.averageLabel ?? "Per day";
  const chart = data?.chart ?? [];
  const maxValue = data?.maxChartValue ?? 0;
  const cats = data?.categories ?? [];

  const todayDay = today.getDate();
  const todayYear = today.getFullYear();
  // Derive period navigation state from the local `period` rather than the
  // server response: when the query is still loading, `data` reflects the
  // *previous* period, so trusting `data.isCurrentPeriod` lets a fast
  // re-click step past today into a future month.
  const isCurrent =
    period.year === todayYear && period.month === today.getMonth() + 1;

  const prevPeriod =
    period.month === 1
      ? { year: period.year - 1, month: 12 }
      : { year: period.year, month: period.month - 1 };
  const nextPeriod =
    period.month === 12
      ? { year: period.year + 1, month: 1 }
      : { year: period.year, month: period.month + 1 };

  const prevMonthName = MONTH_NAMES[prevPeriod.month - 1] ?? "";

  // Show year in labels whenever we're not in the current calendar year —
  // without it, stepping back from January looks identical to stepping back
  // from December (both chips just say a month name) and the user can't
  // tell which December/November/etc. they're looking at.
  const showYear = period.year !== todayYear;
  const periodLabel = showYear ? `${monthLabel} ${period.year}` : monthLabel;
  // The vs-comparison crosses a year boundary when prev's year differs
  // from the current period's year (i.e. period is January). Show the
  // year then so "vs December" can never mean an ambiguous December.
  const prevLabel =
    prevPeriod.year === period.year
      ? prevMonthName
      : `${prevMonthName} ${prevPeriod.year}`;

  const goPrev = () => setPeriod(prevPeriod);
  const goNext = () => {
    if (isCurrent) return;
    setPeriod(nextPeriod);
  };

  return (
    <div
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
          padding: "calc(12px + env(safe-area-inset-top)) 16px 16px",
        }}
      >
        <div
          style={{
            padding: "6px 6px 14px",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <h1
            style={{
              fontSize: 28,
              fontWeight: 600,
              margin: 0,
              letterSpacing: "-0.02em",
            }}
          >
            Insights
          </h1>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 6,
              background: t.card,
              padding: "6px 8px",
              borderRadius: 999,
              fontSize: 12,
              fontWeight: 500,
            }}
          >
            <button
              type="button"
              onClick={goPrev}
              aria-label="Previous month"
              style={{
                background: "transparent",
                border: "none",
                color: t.ink,
                cursor: "pointer",
                padding: "2px 6px",
                fontSize: 14,
                lineHeight: 1,
              }}
            >
              ‹
            </button>
            <span>{periodLabel}</span>
            <button
              type="button"
              onClick={goNext}
              disabled={isCurrent}
              aria-label="Next month"
              style={{
                background: "transparent",
                border: "none",
                color: isCurrent ? t.ink2 : t.ink,
                opacity: isCurrent ? 0.4 : 1,
                cursor: isCurrent ? "default" : "pointer",
                padding: "2px 6px",
                fontSize: 14,
                lineHeight: 1,
              }}
            >
              ›
            </button>
          </div>
        </div>

        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
          <div
            style={{
              background: t.card,
              borderRadius: 22,
              padding: "18px 16px 16px",
            }}
          >
            <div style={{ fontSize: 11, color: t.ink2, fontWeight: 500 }}>
              Spent · {periodLabel}
            </div>
            <div
              style={{
                fontSize: 28,
                fontWeight: 600,
                letterSpacing: "-0.02em",
                marginTop: 4,
              }}
            >
              {fmtEUR(total, { cents: false })}
            </div>
            {data?.hasChange ? (
              <div style={{ fontSize: 12, color: t.ink2, marginTop: 4 }}>
                {data.isIncrease ? "↑" : "↓"}{" "}
                {Math.abs(data.percentageChange).toFixed(0)}% vs {prevLabel}
              </div>
            ) : null}
          </div>
          <div
            style={{
              background: t.card,
              borderRadius: 22,
              padding: "18px 16px 16px",
            }}
          >
            <div style={{ fontSize: 11, color: t.ink2, fontWeight: 500 }}>
              {avgLabel}
            </div>
            <div
              style={{
                fontSize: 28,
                fontWeight: 600,
                letterSpacing: "-0.02em",
                marginTop: 4,
              }}
            >
              {fmtEUR(avgPerDay, { cents: false })}
            </div>
          </div>
        </div>

        <div
          style={{
            background: t.card,
            borderRadius: 22,
            padding: "18px 18px 14px",
            marginTop: 10,
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: 14,
            }}
          >
            <span style={{ fontSize: 13, fontWeight: 500 }}>
              Daily spending
            </span>
            {isCurrent ? (
              <span style={{ fontSize: 11, color: t.ink2 }}>
                Day {todayDay} / {chart.length || 31}
              </span>
            ) : null}
          </div>
          <div
            style={{
              display: "flex",
              alignItems: "flex-end",
              gap: 3,
              height: 96,
            }}
          >
            {chart.map((point, i) => {
              const dayNum = i + 1;
              const isFuture = isCurrent && dayNum > todayDay;
              const v = point.value;
              const h = maxValue ? Math.max(3, (v / maxValue) * 100) : 3;
              const isToday = isCurrent && dayNum === todayDay;
              return (
                <div
                  key={i}
                  style={{
                    flex: 1,
                    height: isFuture ? 3 : `${h}%`,
                    borderRadius: 4,
                    background: isFuture
                      ? t.rule
                      : isToday
                        ? t.accent
                        : t.barOther,
                  }}
                />
              );
            })}
          </div>
        </div>

        <div style={{ marginTop: 16 }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: 500,
              padding: "4px 6px 10px",
            }}
          >
            By category
          </div>
          {cats.length === 0 ? (
            <div
              style={{
                background: t.card,
                borderRadius: 22,
                padding: "20px 16px",
                textAlign: "center",
                color: t.ink2,
                fontSize: 13,
              }}
            >
              No spending in this period.
            </div>
          ) : (
            <div
              style={{
                background: t.card,
                borderRadius: 22,
                overflow: "hidden",
              }}
            >
              {cats.map((c, i) => {
                const cat = lookup.byLabel(c.category) ?? lookup.fallback;
                const tone = cat.color;
                return (
                  <button
                    key={c.category}
                    type="button"
                    data-testid={`category-row-${cat.slug}`}
                    onClick={() =>
                      navigate(
                        `/insights/category/${cat.slug}?year=${period.year}&month=${period.month}`,
                      )
                    }
                    style={{
                      width: "100%",
                      textAlign: "left",
                      cursor: "pointer",
                      background: "transparent",
                      border: "none",
                      fontFamily: FONT,
                      color: t.ink,
                      padding: "14px 16px",
                      borderTop: i === 0 ? "none" : `1px solid ${t.rule}`,
                    }}
                  >
                    <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
                      <div
                        style={{
                          width: 38,
                          height: 38,
                          borderRadius: 12,
                          background: tone.bg,
                          color: tone.ink,
                          display: "grid",
                          placeItems: "center",
                          flex: "0 0 auto",
                        }}
                      >
                        <CategoryGlyph icon={cat.icon} size={18} />
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div
                          style={{
                            display: "flex",
                            justifyContent: "space-between",
                            alignItems: "baseline",
                          }}
                        >
                          <span style={{ fontSize: 14, fontWeight: 500 }}>
                            {c.category}
                          </span>
                          <span
                            style={{
                              fontSize: 14,
                              fontWeight: 600,
                              fontVariantNumeric: "tabular-nums",
                            }}
                          >
                            €{c.total.toFixed(2)}
                          </span>
                        </div>
                        <div
                          style={{
                            display: "flex",
                            justifyContent: "space-between",
                            marginTop: 4,
                            fontSize: 11,
                            color: t.ink2,
                          }}
                        >
                          <span>
                            {c.count}{" "}
                            {c.count === 1 ? "transaction" : "transactions"}
                          </span>
                          <span>{c.percentage.toFixed(1)}%</span>
                        </div>
                        <div
                          style={{
                            marginTop: 8,
                            height: 6,
                            borderRadius: 3,
                            background: t.bg,
                            overflow: "hidden",
                          }}
                        >
                          <div
                            style={{
                              width: `${c.percentage}%`,
                              height: "100%",
                              background: tone.ink,
                              opacity: 0.85,
                            }}
                          />
                        </div>
                      </div>
                      <svg
                        width="14"
                        height="14"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke={t.ink2}
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        style={{ flex: "0 0 auto", marginLeft: 4 }}
                      >
                        <path d="M9 6l6 6-6 6" />
                      </svg>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>
      <TabBar
        current="insights"
        onNavigate={(id) => navigate(id === "feed" ? "/" : "/insights")}
        onAdd={() => navigate("/add")}
      />
    </div>
  );
}
