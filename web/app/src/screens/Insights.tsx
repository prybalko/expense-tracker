import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { theme, FONT } from "../theme";
import { TabBar } from "../components/TabBar";
import { CategoryGlyph } from "../components/CategoryGlyph";
import { useInsights } from "../hooks/useInsights";
import { useCategories } from "../hooks/useCategories";
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
  const today = useMemo(() => new Date(), []);
  const [period, setPeriod] = useState({
    year: today.getFullYear(),
    month: today.getMonth() + 1,
  });
  const insights = useInsights({
    view: "month",
    year: period.year,
    month: period.month,
  });
  const { data: categories = [] } = useCategories();

  const slugFor = useMemo(() => {
    const map = new Map<string, string>();
    for (const c of categories) map.set(c.label, c.slug);
    return (label: string) => map.get(label) ?? "other";
  }, [categories]);

  const data = insights.data;
  const monthLabel = data?.monthName ?? MONTH_NAMES[period.month - 1] ?? "";
  const total = data?.total ?? 0;
  const avgPerDay = data?.averageSpending ?? 0;
  const avgLabel = data?.averageLabel ?? "Per day";
  const chart = data?.chart ?? [];
  const maxValue = data?.maxChartValue ?? 0;
  const cats = data?.categories ?? [];

  const todayDay = today.getDate();
  // Derive period navigation state from the local `period` rather than the
  // server response: when the query is still loading, `data` reflects the
  // *previous* period, so trusting `data.isCurrentPeriod` lets a fast
  // re-click step past today into a future month.
  const isCurrent =
    period.year === today.getFullYear() &&
    period.month === today.getMonth() + 1;

  const prevPeriod =
    period.month === 1
      ? { year: period.year - 1, month: 12 }
      : { year: period.year, month: period.month - 1 };
  const nextPeriod =
    period.month === 12
      ? { year: period.year + 1, month: 1 }
      : { year: period.year, month: period.month + 1 };

  const prevMonthName = MONTH_NAMES[prevPeriod.month - 1] ?? "";

  const goPrev = () => setPeriod(prevPeriod);
  const goNext = () => {
    if (isCurrent) return;
    setPeriod(nextPeriod);
  };

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
      <div style={{ flex: 1, padding: "12px 16px 16px" }}>
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
            <span>{monthLabel}</span>
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

        <div style={{ display: "grid", gridTemplateColumns: "1.4fr 1fr", gap: 10 }}>
          <div
            style={{
              background: t.accentSoft,
              borderRadius: 22,
              padding: "18px 18px 16px",
              color: t.accentInk,
            }}
          >
            <div style={{ fontSize: 11, fontWeight: 500, opacity: 0.75 }}>
              Spent · {monthLabel}
            </div>
            <div
              style={{
                fontSize: 36,
                fontWeight: 600,
                letterSpacing: "-0.02em",
                marginTop: 4,
              }}
            >
              {fmtEUR(total, { cents: false })}
            </div>
            {data?.hasChange ? (
              <div style={{ fontSize: 12, marginTop: 4, opacity: 0.8 }}>
                {data.isIncrease ? "↑" : "↓"}{" "}
                {Math.abs(data.percentageChange).toFixed(0)}% vs {prevMonthName}
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
                const slug = slugFor(c.category);
                const tone = t.cat[slug] ?? t.cat.other;
                return (
                  <div
                    key={c.category}
                    style={{
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
                        <CategoryGlyph slug={slug} size={18} />
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
                    </div>
                  </div>
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
