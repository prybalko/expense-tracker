import { theme, FONT } from "../theme";
import { splitInt } from "../format";
import { DeltaPill } from "./DeltaPill";

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
          fontSize: 13,
          color: t.ink2,
        }}
      >
        <span>{monthName}</span>
        {hasChange ? (
          <DeltaPill
            isIncrease={isIncrease}
            percentageChange={percentageChange}
            prevLabel={prevMonthName}
          />
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
        <span style={{ fontSize: 24, color: t.ink2, fontWeight: 500 }}>€</span>
        <span
          style={{
            fontSize: 64,
            fontWeight: 600,
            letterSpacing: "-0.03em",
            lineHeight: 1,
          }}
        >
          {int}
        </span>
        <span style={{ fontSize: 26, color: t.ink2, fontWeight: 500 }}>
          .{dec}
        </span>
      </div>
      <div
        data-testid="hero-label"
        style={{ fontSize: 13, color: t.ink2, marginTop: 6 }}
      >
        spent this month
      </div>
    </div>
  );
}
