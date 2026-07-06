import { theme } from "../theme";

type Props = {
  isIncrease: boolean;
  percentageChange: number;
  prevLabel?: string;
};

export function DeltaPill({ isIncrease, percentageChange, prevLabel }: Props) {
  const t = theme;
  const arrow = isIncrease ? "↑" : "↓";
  const pct = Math.abs(percentageChange).toFixed(0);
  return (
    <span
      style={{
        background: isIncrease ? t.redSoft : t.greenSoft,
        color: isIncrease ? t.red : t.green,
        padding: "4px 10px",
        borderRadius: 999,
        fontSize: 12,
        fontWeight: 600,
        whiteSpace: "nowrap",
      }}
    >
      {arrow} {pct}%{prevLabel ? ` vs ${prevLabel}` : ""}
    </span>
  );
}
