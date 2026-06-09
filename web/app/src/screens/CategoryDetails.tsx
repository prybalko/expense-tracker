import { useMemo } from "react";
import {
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { theme, FONT } from "../theme";
import { CategoryGlyph } from "../components/CategoryGlyph";
import { DayGroup } from "../components/DayGroup";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
import { useCategoryView } from "../hooks/useExpenses";
import { groupByDay, dayLabel } from "../groupByDay";
import { splitInt } from "../format";

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

export function CategoryDetails() {
  const t = theme;
  const navigate = useNavigate();
  const today = useMemo(() => new Date(), []);
  const { slug = "" } = useParams<{ slug: string }>();
  const [searchParams] = useSearchParams();

  const yearParam = parseInt(searchParams.get("year") ?? "", 10);
  const monthParam = parseInt(searchParams.get("month") ?? "", 10);
  const year =
    Number.isFinite(yearParam) && yearParam > 0
      ? yearParam
      : today.getFullYear();
  const month =
    Number.isFinite(monthParam) && monthParam >= 1 && monthParam <= 12
      ? monthParam
      : today.getMonth() + 1;

  // `view=year` → whole-year breakdown instead of a single month.
  const isYearScope = searchParams.get("view") === "year";

  const lookup = useCategoryLookup();
  const cat = lookup.bySlug(slug) ?? lookup.fallback;
  const tone = cat.color;

  // One selector returns all four header numbers in a single useMemo over
  // the same array, so `pct = total / monthTotal * 100` can never flicker
  // (no longer two independent queries resolving at different times).
  const view = useCategoryView(year, isYearScope ? null : month, cat.label);
  const { items, total, count, pct, isLoading } = view;

  const grouped = useMemo(() => groupByDay(items), [items]);
  const slugFor = lookup.slugByLabel;

  // Without the year, the screen says "spent this month" while showing
  // 2024 data — actively misleading once the user can navigate years.
  const isCurrentYear = year === today.getFullYear();
  const isCurrentMonth = isCurrentYear && month === today.getMonth() + 1;
  const monthName = MONTH_NAMES[month - 1] ?? "";
  const spentLabel = isYearScope
    ? isCurrentYear
      ? "spent this year"
      : `spent in ${year}`
    : isCurrentMonth
      ? "spent this month"
      : isCurrentYear
        ? `spent in ${monthName}`
        : `spent in ${monthName} ${year}`;

  const goBack = () => {
    const params = new URLSearchParams();
    if (isYearScope) {
      params.set("year", String(year));
      params.set("view", "year");
    } else if (year !== today.getFullYear() || month !== today.getMonth() + 1) {
      params.set("year", String(year));
      params.set("month", String(month));
    }
    const qs = params.toString();
    navigate(qs ? `/insights?${qs}` : "/insights");
  };

  const totalSplit = splitInt(total);

  return (
    <div
      data-testid="category-details"
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
        style={{
          padding: "calc(20px + env(safe-area-inset-top)) 16px 0",
          display: "flex",
          alignItems: "center",
          gap: 12,
          flexShrink: 0,
        }}
      >
        <button
          type="button"
          onClick={goBack}
          aria-label="Back"
          data-testid="category-details-back"
          style={{
            width: 36,
            height: 36,
            borderRadius: 18,
            background: t.card,
            border: "none",
            color: t.ink,
            cursor: "pointer",
            display: "grid",
            placeItems: "center",
          }}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M15 6l-6 6 6 6" />
          </svg>
        </button>
        <span style={{ fontSize: 14, fontWeight: 500, color: t.ink2 }}>
          Insights
        </span>
      </div>

      <div
        className="scroll-y"
        style={{
          flex: 1,
          padding: "14px 16px calc(24px + env(safe-area-inset-bottom, 0px))",
        }}
      >
        <div
          style={{
            background: t.card,
            borderRadius: 28,
            padding: "22px 20px 20px",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
            <div
              style={{
                width: 48,
                height: 48,
                borderRadius: 16,
                background: tone.bg,
                color: tone.ink,
                display: "grid",
                placeItems: "center",
                flex: "0 0 auto",
              }}
            >
              <CategoryGlyph icon={cat.icon} size={22} />
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: 20,
                  fontWeight: 600,
                  letterSpacing: "-0.01em",
                }}
              >
                {cat.label}
              </div>
              <div
                data-testid="category-details-count"
                style={{ fontSize: 12, color: t.ink2, marginTop: 2 }}
              >
                {count} {count === 1 ? "transaction" : "transactions"} ·{" "}
                {pct.toFixed(1)}% of total
              </div>
            </div>
          </div>
          <div
            style={{
              marginTop: 18,
              display: "flex",
              alignItems: "baseline",
              gap: 4,
            }}
          >
            <span
              style={{ fontSize: 22, color: t.ink2, fontWeight: 500 }}
            >
              €
            </span>
            <span
              data-testid="category-details-total"
              style={{
                fontSize: 44,
                fontWeight: 600,
                letterSpacing: "-0.03em",
                lineHeight: 1,
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {totalSplit.int}
            </span>
            <span
              style={{ fontSize: 20, color: t.ink2, fontWeight: 500 }}
            >
              .{totalSplit.dec}
            </span>
          </div>
          <div style={{ fontSize: 12, color: t.ink2, marginTop: 4 }}>
            {spentLabel}
          </div>
          <div
            style={{
              marginTop: 14,
              height: 6,
              borderRadius: 3,
              background: t.bg,
              overflow: "hidden",
            }}
          >
            <div
              style={{
                width: `${Math.min(100, Math.max(0, pct))}%`,
                height: "100%",
                background: tone.ink,
                opacity: 0.85,
              }}
            />
          </div>
        </div>

        {isLoading ? (
          <div
            style={{
              padding: "32px 16px",
              textAlign: "center",
              color: t.ink2,
              fontSize: 13,
            }}
          >
            Loading...
          </div>
        ) : count === 0 ? (
          <div
            style={{
              background: t.card,
              borderRadius: 22,
              padding: "32px 20px",
              marginTop: 16,
              textAlign: "center",
              color: t.ink2,
              fontSize: 13,
            }}
          >
            No transactions in this category yet.
          </div>
        ) : (
          <div style={{ marginTop: 18 }}>
            {grouped.map(({ day, items: dayItems }) => (
              <DayGroup
                key={day}
                label={dayLabel(day, today)}
                items={dayItems}
                slugFor={slugFor}
                onItemClick={(e) => {
                  if (e.id < 0) return;
                  navigate(`/edit/${e.id}`);
                }}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
