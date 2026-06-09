import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { theme, FONT } from "../theme";
import { TabBar } from "../components/TabBar";
import { CategoryGlyph } from "../components/CategoryGlyph";
import { Segmented } from "../components/Segmented";
import { PeriodNav } from "../components/PeriodNav";
import { MonthlyBars } from "../components/MonthlyBars";
import { Treemap, type TreemapCat } from "../components/Treemap";
import { useAllExpenses, useCurrentUser } from "../hooks/useExpenses";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
import { deriveInsights, deriveYearInsights } from "../insights/derive";
import { fmtEUR } from "../format";
import type { CategoryBreakdown } from "../types";

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

type Person = "all" | "mine";
type Period = "month" | "year";

// Compact category list for the year view (month view uses the treemap).
function YearCategoryRows({
  cats,
  onSelect,
}: {
  cats: TreemapCat[];
  onSelect: (slug: string) => void;
}) {
  const t = theme;
  return (
    <div style={{ background: t.card, borderRadius: 22, overflow: "hidden" }}>
      {cats.map((c, i) => (
        <button
          key={c.id}
          type="button"
          data-testid={`category-row-${c.slug}`}
          onClick={() => onSelect(c.slug)}
          style={{
            width: "100%",
            textAlign: "left",
            cursor: "pointer",
            background: "transparent",
            border: "none",
            fontFamily: FONT,
            color: t.ink,
            padding: "13px 16px",
            borderTop: i === 0 ? "none" : `1px solid ${t.rule}`,
            display: "flex",
            alignItems: "center",
            gap: 13,
          }}
        >
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: 11,
              background: c.color.bg,
              color: c.color.ink,
              display: "grid",
              placeItems: "center",
              flex: "0 0 auto",
            }}
          >
            <CategoryGlyph icon={c.icon} size={17} />
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "baseline",
              }}
            >
              <span style={{ fontSize: 14, fontWeight: 500 }}>{c.label}</span>
              <span
                style={{
                  fontSize: 14,
                  fontWeight: 600,
                  fontVariantNumeric: "tabular-nums",
                }}
              >
                {fmtEUR(c.amount, { cents: false })}
              </span>
            </div>
            <div
              style={{
                marginTop: 7,
                height: 5,
                borderRadius: 3,
                background: t.bg,
                overflow: "hidden",
              }}
            >
              <div
                style={{
                  width: `${c.pct * 100}%`,
                  height: "100%",
                  background: c.color.ink,
                  opacity: 0.85,
                }}
              />
            </div>
          </div>
        </button>
      ))}
    </div>
  );
}

export function Insights() {
  const t = theme;
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const today = useMemo(() => new Date(), []);
  const lookup = useCategoryLookup();

  // Seed from the URL so CategoryDetails' back-arrow restores the period/view.
  const [period, setPeriod] = useState<Period>(() =>
    searchParams.get("view") === "year" ? "year" : "month",
  );
  const [person, setPerson] = useState<Person>("all");
  const [cursor, setCursor] = useState(() => {
    const y = parseInt(searchParams.get("year") ?? "", 10);
    const m = parseInt(searchParams.get("month") ?? "", 10);
    let year = Number.isFinite(y) && y > 0 ? y : today.getFullYear();
    let month =
      Number.isFinite(m) && m >= 1 && m <= 12 ? m : today.getMonth() + 1;
    // Clamp a future (stale/hand-edited) URL to the current month.
    const todayAbs = today.getFullYear() * 12 + today.getMonth();
    if (year * 12 + (month - 1) > todayAbs) {
      year = today.getFullYear();
      month = today.getMonth() + 1;
    }
    return { year, month };
  });

  const all = useAllExpenses();
  const me = useCurrentUser().data;

  // "Mine" = legacy unowned rows + the signed-in user's (matches ExpenseRow).
  const filtered = useMemo(() => {
    const list = all.data ?? [];
    if (person === "all") return list;
    return list.filter(
      (e) => e.user_id == null || (me != null && e.user_id === me.id),
    );
  }, [all.data, person, me]);

  const monthView = useMemo(
    () => deriveInsights(filtered, cursor.year, cursor.month, today),
    [filtered, cursor.year, cursor.month, today],
  );
  const yearView = useMemo(
    () => deriveYearInsights(filtered, cursor.year, today),
    [filtered, cursor.year, today],
  );

  const isYear = period === "year";
  const total = isYear ? yearView.total : monthView.total;
  const avg = isYear ? yearView.averageSpending : monthView.averageSpending;
  const avgSuffix = isYear ? "per month" : "per day";
  const hasChange = isYear ? yearView.hasChange : monthView.hasChange;
  const isIncrease = isYear ? yearView.isIncrease : monthView.isIncrease;
  const pctChange = isYear
    ? yearView.percentageChange
    : monthView.percentageChange;

  const prevMonthName = MONTH_NAMES[(monthView.prevMonth - 1 + 12) % 12] ?? "";
  const monthPrevLabel =
    monthView.prevYear === cursor.year
      ? prevMonthName
      : `${prevMonthName} ${monthView.prevYear}`;
  const prevLabel = isYear ? String(yearView.prevYear) : monthPrevLabel;

  const monthName = MONTH_NAMES[cursor.month - 1] ?? "";
  const periodLabel = isYear
    ? String(cursor.year)
    : cursor.year === today.getFullYear()
      ? monthName
      : `${monthName} ${cursor.year}`;

  // Prev always allowed; next clamped at the present via absolute month index.
  const canPrev = true;
  const todayAbs = today.getFullYear() * 12 + today.getMonth();
  const cursorAbs = cursor.year * 12 + (cursor.month - 1);
  const canNext = isYear ? cursor.year < today.getFullYear() : cursorAbs < todayAbs;

  const step = (dir: -1 | 1) => {
    if (dir > 0 && !canNext) return;
    if (isYear) {
      setCursor((c) => ({ ...c, year: c.year + dir }));
    } else {
      setCursor((c) => {
        const abs = c.year * 12 + (c.month - 1) + dir;
        return { year: Math.floor(abs / 12), month: (abs % 12) + 1 };
      });
    }
  };

  const jumpToMonth = (monthIndex0: number) => {
    setCursor((c) => ({ year: c.year, month: monthIndex0 + 1 }));
    setPeriod("month");
  };

  const toCat = (c: CategoryBreakdown): TreemapCat => {
    const cat = lookup.byLabel(c.category) ?? lookup.fallback;
    return {
      // Raw string is unique per list; resolved slug can collapse onto "other".
      id: c.category,
      slug: cat.slug,
      label: cat.label,
      amount: c.total,
      pct: c.percentage / 100,
      color: cat.color,
      icon: cat.icon,
    };
  };
  const monthCats = monthView.categories.map(toCat);
  const yearCats = yearView.categories.map(toCat);

  const openCategory = (slug: string) => {
    if (isYear) {
      navigate(`/insights/category/${slug}?year=${cursor.year}&view=year`);
    } else {
      navigate(
        `/insights/category/${slug}?year=${cursor.year}&month=${cursor.month}&view=month`,
      );
    }
  };

  return (
    <div
      data-testid="insights-screen"
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
        <h1
          style={{
            fontSize: 26,
            fontWeight: 600,
            margin: "0 0 16px",
            letterSpacing: "-0.02em",
          }}
        >
          Insights
        </h1>

        {/* Controls — sliding pills + airy period navigator */}
        <div style={{ display: "flex", gap: 8 }}>
          <Segmented<Person>
            value={person}
            onChange={setPerson}
            options={[
              { value: "mine", label: "Mine" },
              { value: "all", label: "All" },
            ]}
          />
          <Segmented<Period>
            value={period}
            onChange={setPeriod}
            options={[
              { value: "month", label: "Month" },
              { value: "year", label: "Year" },
            ]}
          />
        </div>
        <PeriodNav
          label={periodLabel}
          canPrev={canPrev}
          canNext={canNext}
          onStep={step}
        />

        {/* Hero — total + average + delta pill */}
        <div
          style={{
            display: "flex",
            alignItems: "baseline",
            justifyContent: "space-between",
            padding: "16px 4px 16px",
          }}
        >
          <div>
            <div
              data-testid="insights-total"
              style={{
                fontSize: 40,
                fontWeight: 700,
                letterSpacing: "-0.03em",
                lineHeight: 1,
              }}
            >
              {fmtEUR(total, { cents: false })}
            </div>
            <div style={{ fontSize: 12, color: t.ink2, marginTop: 6 }}>
              {fmtEUR(avg, { cents: false })} {avgSuffix}
            </div>
          </div>
          {hasChange ? (
            <div
              style={{
                fontSize: 12,
                fontWeight: 600,
                padding: "5px 10px",
                borderRadius: 999,
                background: isIncrease ? "#F0DEDA" : "#E0EAE4",
                color: isIncrease ? t.red : t.green,
              }}
            >
              {isIncrease ? "↑" : "↓"} {pctChange.toFixed(0)}% vs {prevLabel}
            </div>
          ) : null}
        </div>

        {/* Period-aware main visual */}
        {isYear ? (
          <>
            <MonthlyBars
              series={yearView.series}
              elapsed={yearView.elapsedMonths}
              onMonth={jumpToMonth}
            />
            <div style={{ fontSize: 13, fontWeight: 500, padding: "20px 4px 10px" }}>
              On what
            </div>
            {yearCats.length ? (
              <YearCategoryRows cats={yearCats} onSelect={openCategory} />
            ) : (
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
            )}
          </>
        ) : (
          <Treemap cats={monthCats} onSelect={openCategory} />
        )}
      </div>
      <TabBar
        current="insights"
        onNavigate={(id) => navigate(id === "feed" ? "/" : "/insights")}
        onAdd={() => navigate("/add")}
      />
    </div>
  );
}
