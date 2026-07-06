import { useState } from "react";
import { theme, FONT } from "../theme";
import { CalendarGrid } from "./CalendarGrid";
import { SectionLabel } from "./SectionLabel";

const sameYMD = (a: Date, b: Date): boolean =>
  a.getFullYear() === b.getFullYear() &&
  a.getMonth() === b.getMonth() &&
  a.getDate() === b.getDate();

const startOfDay = (d: Date): Date =>
  new Date(d.getFullYear(), d.getMonth(), d.getDate());

const addDays = (d: Date, n: number): Date => {
  const out = new Date(d);
  out.setDate(d.getDate() + n);
  return out;
};

function pillLabelFor(value: Date, today: Date): string {
  const v = startOfDay(value);
  const t = startOfDay(today);
  if (sameYMD(v, t)) return "Today";
  if (sameYMD(v, addDays(t, -1))) return "Yesterday";
  return v.toLocaleDateString("en-GB", {
    weekday: "short",
    day: "numeric",
    month: "short",
  });
}

type Props = {
  value: Date;
  onChange: (d: Date) => void;
  bare?: boolean;
  today?: Date;
};

export function DatePickerPill({
  value,
  onChange,
  bare = false,
  today: todayProp,
}: Props) {
  const t = theme;
  const today = todayProp ?? new Date();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Date>(value);
  const [viewMonth, setViewMonth] = useState<Date>(
    () => new Date(value.getFullYear(), value.getMonth(), 1),
  );

  const openSheet = () => {
    setDraft(value);
    setViewMonth(new Date(value.getFullYear(), value.getMonth(), 1));
    setOpen(true);
  };

  const button = (
    <button
      type="button"
      onClick={openSheet}
      data-testid="date-pill"
      style={{
        width: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: "14px 14px",
        borderRadius: 16,
        border: "none",
        background: t.card,
        color: t.ink,
        fontFamily: FONT,
        fontSize: 14,
        fontWeight: 500,
        cursor: "pointer",
        boxSizing: "border-box",
      }}
    >
      <span
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          minWidth: 0,
        }}
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeLinejoin="round"
          style={{ flex: "0 0 auto" }}
        >
          <rect x="3" y="5" width="18" height="16" rx="2" />
          <path d="M3 9h18M8 3v4M16 3v4" />
        </svg>
        <span
          style={{
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {pillLabelFor(value, today)}
        </span>
      </span>
      <svg
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        style={{ flex: "0 0 auto" }}
      >
        <path d="M3 5l3 3 3-3" />
      </svg>
    </button>
  );

  const sheet = open && (
    <div
      style={{
        position: "absolute",
        inset: 0,
        zIndex: 50,
        background: "rgba(0,0,0,0.4)",
        display: "flex",
        alignItems: "flex-end",
      }}
      onClick={() => setOpen(false)}
    >
      <div
        data-testid="date-sheet"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: "100%",
          background: t.bg,
          borderTopLeftRadius: 28,
          borderTopRightRadius: 28,
          padding: "14px 0 28px",
          boxShadow: "0 -8px 32px rgba(0,0,0,0.2)",
          animation: "datepicker-slideup .18s ease-out",
        }}
      >
        <div
          style={{
            width: 36,
            height: 4,
            background: t.rule,
            borderRadius: 2,
            margin: "4px auto 14px",
          }}
        />
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            padding: "0 18px 10px",
          }}
        >
          <button
            type="button"
            onClick={() => setOpen(false)}
            style={{
              background: "transparent",
              border: "none",
              color: t.ink2,
              fontSize: 14,
              fontFamily: FONT,
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
          <span style={{ fontSize: 14, fontWeight: 600 }}>Pick date</span>
          <button
            type="button"
            data-testid="date-sheet-done"
            onClick={() => {
              onChange(draft);
              setOpen(false);
            }}
            style={{
              background: "transparent",
              border: "none",
              color: t.accent,
              fontSize: 14,
              fontFamily: FONT,
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            Done
          </button>
        </div>
        <div
          style={{
            margin: "0 14px",
            background: t.card,
            borderRadius: 22,
            padding: "14px 14px 12px",
          }}
        >
          <CalendarGrid
            value={draft}
            onChange={setDraft}
            viewMonth={viewMonth}
            setViewMonth={setViewMonth}
            today={today}
          />
        </div>
        <style>{`
          @keyframes datepicker-slideup {
            from { transform: translateY(100%); }
            to { transform: translateY(0); }
          }
        `}</style>
      </div>
    </div>
  );

  if (bare) {
    return (
      <>
        {button}
        {sheet}
      </>
    );
  }

  return (
    <div style={{ padding: "6px 0 8px", fontFamily: FONT }}>
      <SectionLabel style={{ padding: "0 18px 8px" }}>Date</SectionLabel>
      <div style={{ padding: "0 14px" }}>{button}</div>
      {sheet}
    </div>
  );
}
