import { theme, FONT } from "../theme";
import { splitInt } from "../format";

type Props = {
  monthName: string;
  total: number;
  hasChange?: boolean;
  isIncrease?: boolean;
  percentageChange?: number;
  prevMonthName?: string;
};

export function Hero({
  monthName,
  total,
  hasChange = false,
  isIncrease = false,
  percentageChange = 0,
  prevMonthName,
}: Props) {
  const t = theme;
  const { int, dec } = splitInt(total);
  const arrow = isIncrease ? "↑" : "↓";
  const deltaText = prevMonthName
    ? `${arrow} ${Math.abs(percentageChange).toFixed(0)}% vs ${prevMonthName}`
    : `${arrow} ${Math.abs(percentageChange).toFixed(0)}%`;
  const deltaBg = isIncrease ? "#F2D7DA" : "#E0EAE4";
  const deltaColor = isIncrease ? t.red : t.green;

  return (
    <div
      style={{
        background: t.card,
        borderRadius: 28,
        padding: "24px 22px 20px",
        fontFamily: FONT,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          fontSize: 12,
          color: t.ink2,
        }}
      >
        <span>{monthName}</span>
        {hasChange ? (
          <span
            style={{
              background: deltaBg,
              color: deltaColor,
              padding: "4px 10px",
              borderRadius: 999,
              fontSize: 11,
              fontWeight: 500,
            }}
          >
            {deltaText}
          </span>
        ) : null}
      </div>
      <div
        data-testid="hero-total"
        style={{
          display: "flex",
          alignItems: "baseline",
          gap: 4,
          marginTop: 14,
        }}
      >
        <span style={{ fontSize: 22, color: t.ink2, fontWeight: 500 }}>€</span>
        <span
          style={{
            fontSize: 56,
            fontWeight: 600,
            letterSpacing: "-0.03em",
            lineHeight: 1,
          }}
        >
          {int}
        </span>
        <span style={{ fontSize: 24, color: t.ink2, fontWeight: 500 }}>
          .{dec}
        </span>
      </div>
      <div
        data-testid="hero-label"
        style={{ fontSize: 12, color: t.ink2, marginTop: 6 }}
      >
        spent this month
      </div>
    </div>
  );
}
