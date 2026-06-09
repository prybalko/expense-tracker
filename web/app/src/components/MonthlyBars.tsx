import { theme, FONT } from "../theme";

const LETTERS = ["J", "F", "M", "A", "M", "J", "J", "A", "S", "O", "N", "D"];

type Props = {
  series: number[]; // 12 monthly totals, index 0 = January
  elapsed: number; // months elapsed (12 for a past year, current month count for this year)
  onMonth: (monthIndex: number) => void; // 0-based month index
};

// Year trend: 12 month bars; future months flatten, last elapsed is accented.
export function MonthlyBars({ series, elapsed, onMonth }: Props) {
  const t = theme;
  const max = Math.max(...series, 1);
  return (
    <div
      style={{
        background: t.card,
        borderRadius: 22,
        padding: "18px 16px 14px",
        fontFamily: FONT,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <span style={{ fontSize: 13, fontWeight: 500 }}>Monthly spending</span>
        <span style={{ fontSize: 11, color: t.ink2 }}>tap a month</span>
      </div>
      <div style={{ display: "flex", alignItems: "flex-end", gap: 6, height: 120 }}>
        {series.map((v, i) => {
          const future = i >= elapsed;
          const h = max ? Math.max(4, (v / max) * 100) : 4;
          const isLast = i === elapsed - 1;
          return (
            <button
              key={i}
              type="button"
              onClick={() => !future && onMonth(i)}
              disabled={future}
              data-testid={`month-bar-${i}`}
              aria-label={`${LETTERS[i]}: ${Math.round(v)}`}
              style={{
                flex: 1,
                display: "flex",
                flexDirection: "column",
                justifyContent: "flex-end",
                height: "100%",
                border: "none",
                background: "transparent",
                padding: 0,
                cursor: future ? "default" : "pointer",
              }}
            >
              <div
                style={{
                  height: future ? 4 : `${h}%`,
                  borderRadius: 5,
                  background: future
                    ? t.rule
                    : isLast
                      ? t.accent
                      : t.barOther,
                }}
              />
            </button>
          );
        })}
      </div>
      <div style={{ display: "flex", gap: 6, marginTop: 8 }}>
        {LETTERS.map((l, i) => (
          <span
            key={i}
            style={{
              flex: 1,
              textAlign: "center",
              fontSize: 10,
              color: i < elapsed ? t.ink2 : t.rule,
              fontWeight: i === elapsed - 1 ? 700 : 400,
            }}
          >
            {l}
          </span>
        ))}
      </div>
    </div>
  );
}
