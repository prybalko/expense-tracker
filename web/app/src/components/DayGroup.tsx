import { theme } from "../theme";
import { fmtEUR } from "../format";
import { ExpenseRow } from "./ExpenseRow";
import type { Expense } from "../types";

type Props = {
  label: string;
  items: Expense[];
  slugFor: (categoryLabel: string) => string;
  onItemClick?: (expense: Expense) => void;
};

export function DayGroup({ label, items, slugFor, onItemClick }: Props) {
  const t = theme;
  const sum = items.reduce((s, e) => s + e.amount, 0);

  return (
    <div style={{ marginBottom: 14 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          padding: "4px 8px 8px",
          fontSize: 12,
          fontWeight: 600,
          letterSpacing: "0.04em",
          color: t.ink2,
          textTransform: "uppercase",
        }}
      >
        <span>{label}</span>
        <span>−{fmtEUR(sum)}</span>
      </div>
      <div
        style={{
          background: t.card,
          borderRadius: 22,
          overflow: "hidden",
        }}
      >
        {items.map((expense, i) => (
          <ExpenseRow
            key={expense.id}
            expense={expense}
            slug={slugFor(expense.category)}
            isFirst={i === 0}
            onClick={onItemClick}
          />
        ))}
      </div>
    </div>
  );
}
