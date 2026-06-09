import { theme, FONT } from "../theme";
import { CategoryGlyph } from "./CategoryGlyph";
import { useCategoryLookup } from "../hooks/useCategoryLookup";
import { useCurrentUser } from "../hooks/useExpenses";
import type { Expense } from "../types";

type Props = {
  expense: Expense;
  slug: string;
  isFirst?: boolean;
  onClick?: (expense: Expense) => void;
};

export function ExpenseRow({ expense, slug, isFirst = false, onClick }: Props) {
  const t = theme;
  const lookup = useCategoryLookup();
  const cat = lookup.bySlug(slug) ?? lookup.fallback;
  const tone = cat.color;
  // Mark rows authored by someone other than the signed-in user with a
  // small dot next to the category. Legacy unowned rows (user_id == null)
  // and rows the signed-in user wrote both count as mine. While
  // /api/auth/me is still in flight we treat every row as mine to avoid
  // briefly flashing dots on every row during the cold-start render.
  const me = useCurrentUser().data;
  const isNotMine =
    me != null && expense.user_id != null && expense.user_id !== me.id;

  return (
    <button
      type="button"
      onClick={() => onClick?.(expense)}
      data-testid="expense-row"
      data-cat-slug={slug}
      data-not-mine={isNotMine ? "true" : undefined}
      aria-label={isNotMine ? "Added by someone else" : undefined}
      style={{
        width: "100%",
        display: "flex",
        alignItems: "center",
        gap: 14,
        padding: "12px 16px",
        textAlign: "left",
        background: "transparent",
        border: "none",
        cursor: "pointer",
        borderTop: isFirst ? "none" : `1px solid ${t.rule}`,
        fontFamily: FONT,
        color: t.ink,
      }}
    >
      <div
        style={{
          width: 42,
          height: 42,
          borderRadius: 14,
          background: tone.bg,
          color: tone.ink,
          display: "grid",
          placeItems: "center",
          flex: "0 0 auto",
        }}
      >
        <CategoryGlyph icon={cat.icon} size={20} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          data-testid="expense-row-desc"
          style={{ fontSize: 15, fontWeight: 500 }}
        >
          {expense.description || expense.category}
        </div>
        <div
          style={{
            fontSize: 12,
            color: t.ink2,
            marginTop: 2,
            display: "flex",
            alignItems: "center",
            gap: 6,
          }}
        >
          <span>{expense.category}</span>
          {isNotMine ? (
            <span
              aria-hidden="true"
              style={{
                width: 6,
                height: 6,
                borderRadius: "50%",
                background: t.ink2,
                flex: "0 0 auto",
              }}
            />
          ) : null}
        </div>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span
          data-testid="expense-row-amount"
          style={{
            fontSize: 15,
            fontWeight: 600,
            fontVariantNumeric: "tabular-nums",
          }}
        >
          −€{expense.amount.toFixed(2)}
        </span>
        <svg
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke={t.ink2}
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          style={{ flex: "0 0 auto" }}
        >
          <path d="M9 6l6 6-6 6" />
        </svg>
      </div>
    </button>
  );
}
