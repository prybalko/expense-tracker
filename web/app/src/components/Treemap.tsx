import { useLayoutEffect, useRef, useState } from "react";
import { theme, FONT } from "../theme";
import { CategoryGlyph } from "./CategoryGlyph";
import { fmtEUR } from "../format";
import type { CategoryColor } from "../types";

export type TreemapCat = {
  id: string; // stable, unique React key — the raw category string from the breakdown
  slug: string;
  label: string;
  amount: number;
  pct: number; // fraction 0..1
  color: CategoryColor;
  icon: string;
};

type Sized = TreemapCat & { a: number };
type Rect = TreemapCat & { x: number; y: number; w: number; h: number };

// Squarified treemap: block area ∝ spend share, aspect ratios kept near 1.
function squarify(items: TreemapCat[], W: number, H: number): Rect[] {
  const total = items.reduce((s, i) => s + i.amount, 0) || 1;
  const scale = (W * H) / total;
  const data: Sized[] = items
    .map((i) => ({ ...i, a: i.amount * scale }))
    .sort((p, q) => q.a - p.a);

  const worst = (row: Sized[], side: number): number => {
    const sum = row.reduce((s, r) => s + r.a, 0);
    const mx = Math.max(...row.map((r) => r.a));
    const mn = Math.min(...row.map((r) => r.a));
    const s2 = sum * sum;
    return Math.max((side * side * mx) / s2, s2 / (side * side * mn));
  };

  const out: Rect[] = [];
  let x = 0;
  let y = 0;
  let w = W;
  let h = H;
  let i = 0;
  while (i < data.length) {
    const side = Math.min(w, h);
    let row: Sized[] = [data[i]];
    let best = worst(row, side);
    let j = i + 1;
    while (j < data.length) {
      const cand = row.concat(data[j]);
      const r = worst(cand, side);
      if (r <= best) {
        row = cand;
        best = r;
        j++;
      } else break;
    }
    const rowArea = row.reduce((s, r) => s + r.a, 0);
    if (w >= h) {
      const colW = rowArea / h;
      let oy = y;
      row.forEach((r) => {
        const rh = r.a / colW;
        out.push({ ...r, x, y: oy, w: colW, h: rh });
        oy += rh;
      });
      x += colW;
      w -= colW;
    } else {
      const rowH = rowArea / w;
      let ox = x;
      row.forEach((r) => {
        const rw = r.a / rowH;
        out.push({ ...r, x: ox, y, w: rw, h: rowH });
        ox += rw;
      });
      y += rowH;
      h -= rowH;
    }
    i = j;
  }
  return out;
}

type Props = {
  cats: TreemapCat[];
  onSelect: (slug: string) => void;
};

// Keep the design's 358×320 proportion regardless of device width.
const ASPECT = 320 / 358;

export function Treemap({ cats, onSelect }: Props) {
  const t = theme;
  const ref = useRef<HTMLDivElement | null>(null);
  const [width, setWidth] = useState(0);

  // Measure width (absolute tiles need concrete px); keep it in sync on resize.
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const update = () => setWidth(el.clientWidth);
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const hasCats = cats.length > 0;
  const H = Math.round(width * ASPECT);
  const rects = hasCats && width > 0 ? squarify(cats, width, H) : [];

  // Ref div is always mounted so width stays measured across empty↔populated
  // transitions (early-returning before it would strand the measurement at 0).
  return (
    <div style={{ fontFamily: FONT }}>
      <div
        ref={ref}
        style={{
          position: "relative",
          width: "100%",
          height: hasCats ? H || undefined : undefined,
        }}
      >
        {!hasCats ? (
          <div
            style={{
              background: t.card,
              borderRadius: 22,
              padding: "40px 20px",
              textAlign: "center",
              color: t.ink2,
              fontSize: 13,
            }}
          >
            No spending in this period.
          </div>
        ) : (
          rects.map((r) => {
          const big = r.w > 78 && r.h > 52;
          const med = r.h > 34;
          return (
            <button
              key={r.id}
              type="button"
              data-testid={`category-row-${r.slug}`}
              aria-label={`${r.label}: ${fmtEUR(r.amount, { cents: false })}`}
              onClick={() => onSelect(r.slug)}
              style={{
                position: "absolute",
                left: r.x + 1.5,
                top: r.y + 1.5,
                width: Math.max(0, r.w - 3),
                height: Math.max(0, r.h - 3),
                background: r.color.bg,
                color: r.color.ink,
                border: "none",
                borderRadius: 10,
                cursor: "pointer",
                padding: big ? 10 : 6,
                textAlign: "left",
                fontFamily: FONT,
                overflow: "hidden",
                display: "flex",
                flexDirection: "column",
                justifyContent: "space-between",
              }}
            >
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "flex-start",
                }}
              >
                {(big || med) && (
                  <CategoryGlyph icon={r.icon} size={big ? 18 : 14} />
                )}
                {big && (
                  <span style={{ fontSize: 11, fontWeight: 600, opacity: 0.7 }}>
                    {Math.round(r.pct * 100)}%
                  </span>
                )}
              </div>
              {(big || med) && (
                <div>
                  {big && (
                    <div
                      style={{ fontSize: 13, fontWeight: 600, lineHeight: 1.1 }}
                    >
                      {r.label}
                    </div>
                  )}
                  <div
                    style={{
                      fontSize: big ? 15 : 12,
                      fontWeight: 700,
                      fontVariantNumeric: "tabular-nums",
                    }}
                  >
                    {fmtEUR(r.amount, { cents: false })}
                  </div>
                </div>
              )}
            </button>
          );
          })
        )}
      </div>
      {hasCats && (
        <div
          style={{
            fontSize: 11,
            color: t.ink2,
            textAlign: "center",
            padding: "12px 0 0",
          }}
        >
          Block size = share of spend · tap to open
        </div>
      )}
    </div>
  );
}
