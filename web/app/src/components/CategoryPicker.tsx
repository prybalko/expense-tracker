import { useMemo, useRef, useState } from "react";
import type { UIEvent } from "react";
import { theme, FONT } from "../theme";
import { categories } from "../categories";
import { CategoryGlyph } from "./CategoryGlyph";
import { SectionLabel } from "./SectionLabel";
import type { Category } from "../types";

type Props = {
  value: string;
  onChange: (label: string) => void;
  usageCounts?: Record<string, number>;
};

const PAGE_SIZE = 8;

function orderCategories(
  categories: Category[],
  usageCounts: Record<string, number>,
): Category[] {
  const declIdx = new Map<string, number>();
  categories.forEach((c, i) => declIdx.set(c.label, i));

  // Sort the entire list by usage (desc), with declaration order as the
  // tie-breaker. Every page — not just the first — surfaces the most-used
  // categories before less-used ones, so a never-used tile ends up at the
  // very end regardless of which page it falls on.
  return [...categories].sort((a, b) => {
    const ca = usageCounts[a.label] ?? 0;
    const cb = usageCounts[b.label] ?? 0;
    if (ca !== cb) return cb - ca;
    return (declIdx.get(a.label) ?? 0) - (declIdx.get(b.label) ?? 0);
  });
}

export function CategoryPicker({ value, onChange, usageCounts = {} }: Props) {
  const t = theme;
  const scrollerRef = useRef<HTMLDivElement | null>(null);
  const [page, setPage] = useState(0);

  // Re-sort when categories change, and once more when usage data first
  // becomes available — but not on every usage-count tick after that, so the
  // order doesn't shuffle mid-session as the user records new expenses.
  const hasUsageData = Object.keys(usageCounts).length > 0;
  const ordered = useMemo(
    () => orderCategories(categories, usageCounts),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [categories, hasUsageData],
  );

  const pages: Category[][] = [];
  for (let i = 0; i < ordered.length; i += PAGE_SIZE) {
    pages.push(ordered.slice(i, i + PAGE_SIZE));
  }

  const onScroll = (e: UIEvent<HTMLDivElement>) => {
    const el = e.currentTarget;
    const p = Math.round(el.scrollLeft / el.clientWidth);
    if (p !== page) setPage(p);
  };

  return (
    <div style={{ padding: "6px 0 4px", fontFamily: FONT }}>
      <SectionLabel style={{ padding: "0 18px 8px" }}>Category</SectionLabel>
      <div
        ref={scrollerRef}
        onScroll={onScroll}
        className="scroll-x"
        style={{
          display: "flex",
          overflowY: "hidden",
          scrollSnapType: "x mandatory",
        }}
      >
        {pages.map((pageCats, pi) => (
          <div
            key={pi}
            style={{
              flex: "0 0 100%",
              scrollSnapAlign: "start",
              padding: "0 14px",
              boxSizing: "border-box",
              display: "grid",
              gridTemplateColumns: "repeat(4, 1fr)",
              gridAutoRows: "min-content",
              gap: 8,
            }}
          >
            {pageCats.map((cat) => {
              const sel = cat.label === value;
              const tone = cat.color;
              return (
                <button
                  key={cat.label}
                  type="button"
                  onClick={() => onChange(cat.label)}
                  data-testid={`category-tile-${cat.slug}`}
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    gap: 6,
                    minWidth: 0,
                    padding: "12px 4px 10px",
                    borderRadius: 18,
                    background: sel ? tone.bg : t.card,
                    border: sel
                      ? `2px solid ${tone.ink}`
                      : "2px solid transparent",
                    color: sel ? tone.ink : t.ink,
                    cursor: "pointer",
                    fontFamily: FONT,
                    transition: "all .12s ease",
                  }}
                >
                  <div
                    style={{
                      width: 34,
                      height: 34,
                      borderRadius: 12,
                      background: sel ? t.cardAlt : tone.bg,
                      color: tone.ink,
                      display: "grid",
                      placeItems: "center",
                    }}
                  >
                    <CategoryGlyph icon={cat.icon} size={18} />
                  </div>
                  <span
                    title={cat.label}
                    style={{
                      fontSize: 13,
                      fontWeight: 500,
                      width: "100%",
                      textAlign: "center",
                      whiteSpace: "nowrap",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    }}
                  >
                    {cat.label}
                  </span>
                </button>
              );
            })}
          </div>
        ))}
      </div>
      {pages.length > 1 && (
        <div
          style={{
            display: "flex",
            justifyContent: "center",
            gap: 6,
            padding: "10px 0 2px",
          }}
        >
          {pages.map((_, pi) => (
            <span
              key={pi}
              style={{
                width: 6,
                height: 6,
                borderRadius: 3,
                background: pi === page ? t.ink : t.ink2,
                opacity: pi === page ? 1 : 0.35,
                transition: "all .18s ease",
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}
