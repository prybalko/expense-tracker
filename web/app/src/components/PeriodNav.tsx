import { theme } from "../theme";

type Props = {
  label: string;
  canPrev: boolean;
  canNext: boolean;
  onStep: (dir: -1 | 1) => void;
};

type ChevProps = { dir: -1 | 1; off: boolean; onStep: (dir: -1 | 1) => void };

// Chevron that dims when it can't move in its direction.
function Chev({ dir, off, onStep }: ChevProps) {
  const t = theme;
  return (
    <button
      type="button"
      onClick={() => !off && onStep(dir)}
      disabled={off}
      aria-label={dir < 0 ? "Previous period" : "Next period"}
      data-testid={dir < 0 ? "period-prev" : "period-next"}
      style={{
        width: 36,
        height: 36,
        borderRadius: 999,
        border: "none",
        cursor: off ? "default" : "pointer",
        background: "transparent",
        color: t.ink,
        opacity: off ? 0.18 : 0.6,
        display: "grid",
        placeItems: "center",
        flex: "0 0 auto",
        transition: "opacity .15s",
      }}
    >
      <svg
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d={dir < 0 ? "M15 6l-6 6 6 6" : "M9 6l6 6-6 6"} />
      </svg>
    </button>
  );
}

export function PeriodNav({ label, canPrev, canNext, onStep }: Props) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        marginTop: 16,
      }}
    >
      <Chev dir={-1} off={!canPrev} onStep={onStep} />
      <span
        data-testid="period-label"
        style={{ fontSize: 18, fontWeight: 600, letterSpacing: "-0.01em" }}
      >
        {label}
      </span>
      <Chev dir={1} off={!canNext} onStep={onStep} />
    </div>
  );
}
