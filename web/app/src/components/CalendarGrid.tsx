import { theme, FONT } from "../theme";

const WEEKDAYS_MIN = ["S", "M", "T", "W", "T", "F", "S"];
const MONTHS_LONG = [
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

const sameYMD = (a: Date, b: Date): boolean =>
  a.getFullYear() === b.getFullYear() &&
  a.getMonth() === b.getMonth() &&
  a.getDate() === b.getDate();

const startOfDay = (d: Date): Date =>
  new Date(d.getFullYear(), d.getMonth(), d.getDate());

type Props = {
  value: Date;
  onChange: (d: Date) => void;
  viewMonth: Date;
  setViewMonth: (d: Date) => void;
  today?: Date;
};

export function CalendarGrid({
  value,
  onChange,
  viewMonth,
  setViewMonth,
  today: todayProp,
}: Props) {
  const t = theme;
  const today = startOfDay(todayProp ?? new Date());
  const selected = startOfDay(value);

  const y = viewMonth.getFullYear();
  const m = viewMonth.getMonth();
  const first = new Date(y, m, 1);
  const startWeekday = first.getDay();
  const daysInMonth = new Date(y, m + 1, 0).getDate();

  const cells: (Date | null)[] = [];
  for (let i = 0; i < 42; i++) {
    const dayNum = i - startWeekday + 1;
    if (dayNum < 1 || dayNum > daysInMonth) {
      cells.push(null);
      continue;
    }
    cells.push(new Date(y, m, dayNum));
  }

  const isCurrentMonth =
    y === today.getFullYear() && m === today.getMonth();
  const canNext = !isCurrentMonth;
  const cellSize = 36;
  const cellGap = 4;

  return (
    <div style={{ fontFamily: FONT }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "0 4px 10px",
        }}
      >
        <button
          type="button"
          onClick={() => setViewMonth(new Date(y, m - 1, 1))}
          style={{
            width: 30,
            height: 30,
            borderRadius: 15,
            background: "transparent",
            border: "none",
            cursor: "pointer",
            color: t.ink,
            display: "grid",
            placeItems: "center",
          }}
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
          >
            <path d="M15 6l-6 6 6 6" />
          </svg>
        </button>
        <span style={{ fontSize: 14, fontWeight: 600 }}>
          {MONTHS_LONG[m]} {y}
        </span>
        <button
          type="button"
          onClick={() => setViewMonth(new Date(y, m + 1, 1))}
          disabled={!canNext}
          style={{
            width: 30,
            height: 30,
            borderRadius: 15,
            background: "transparent",
            border: "none",
            cursor: canNext ? "pointer" : "default",
            color: t.ink,
            display: "grid",
            placeItems: "center",
            opacity: canNext ? 1 : 0.25,
          }}
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
          >
            <path d="M9 6l6 6-6 6" />
          </svg>
        </button>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7, 1fr)",
          gap: cellGap,
          padding: "0 0 6px",
        }}
      >
        {WEEKDAYS_MIN.map((w, i) => (
          <div
            key={i}
            style={{
              textAlign: "center",
              fontSize: 10,
              fontWeight: 600,
              color: t.ink2,
              letterSpacing: "0.05em",
            }}
          >
            {w}
          </div>
        ))}
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(7, 1fr)",
          gap: cellGap,
        }}
      >
        {cells.map((d, i) => {
          if (!d) return <div key={i} style={{ height: cellSize }} />;
          const sel = sameYMD(d, selected);
          const isToday = sameYMD(d, today);
          const isFuture = d > today;
          return (
            <button
              key={i}
              type="button"
              disabled={isFuture}
              onClick={() => onChange(d)}
              style={{
                height: cellSize,
                borderRadius: cellSize / 2,
                background: sel ? t.accent : "transparent",
                color: sel ? t.accentText : isFuture ? t.ink2 : t.ink,
                border:
                  !sel && isToday
                    ? `1.5px solid ${t.ink}`
                    : "1.5px solid transparent",
                fontSize: 13,
                fontWeight: sel || isToday ? 600 : 500,
                fontFamily: FONT,
                cursor: isFuture ? "default" : "pointer",
                opacity: isFuture ? 0.3 : 1,
                fontVariantNumeric: "tabular-nums",
                transition: "background .12s, color .12s",
              }}
            >
              {d.getDate()}
            </button>
          );
        })}
      </div>
    </div>
  );
}
