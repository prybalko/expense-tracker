import { theme, FONT } from "../theme";

type Option<T extends string> = { value: T; label: string };

type Props<T extends string> = {
  options: Option<T>[];
  value: T;
  onChange: (value: T) => void;
};

// Segmented control with a sliding thumb; generic so call sites stay type-safe.
export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: Props<T>) {
  const t = theme;
  const n = options.length;
  const idx = Math.max(
    0,
    options.findIndex((o) => o.value === value),
  );
  return (
    <div
      style={{
        position: "relative",
        flex: 1,
        display: "flex",
        background: t.card,
        borderRadius: 13,
        padding: 3,
        border: `1px solid ${t.rule}`,
      }}
    >
      <div
        aria-hidden
        style={{
          position: "absolute",
          top: 3,
          bottom: 3,
          width: `calc((100% - 6px) / ${n})`,
          left: `calc(3px + ${idx} * (100% - 6px) / ${n})`,
          background: t.accent,
          borderRadius: 10,
          transition: "left .26s cubic-bezier(.34,1.3,.5,1)",
          boxShadow: "0 1px 3px rgba(0,0,0,0.12)",
        }}
      />
      {options.map((o) => {
        const sel = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            data-testid={`segment-${o.value}`}
            aria-pressed={sel}
            onClick={() => onChange(o.value)}
            style={{
              position: "relative",
              zIndex: 1,
              flex: 1,
              border: "none",
              background: "transparent",
              cursor: "pointer",
              fontFamily: FONT,
              padding: "8px 6px",
              fontSize: 14,
              fontWeight: 600,
              color: sel ? t.accentText : t.ink2,
              transition: "color .2s ease",
            }}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}
