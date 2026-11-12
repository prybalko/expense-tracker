import { theme } from "../theme";

type Props = {
  // Already scaled by RESISTANCE.
  pullDistance: number;
  committed: boolean;
  isRefreshing: boolean;
};

const SIZE = 28;
const PETALS = Array.from({ length: 12 }, (_, index) => index);
const FADE_DISTANCE_PX = 48;

export function PullToRefreshIndicator({
  pullDistance,
  committed,
  isRefreshing,
}: Props) {
  const t = theme;
  if (pullDistance === 0 && !isRefreshing) return null;

  const opacity = isRefreshing ? 1 : Math.min(pullDistance / FADE_DISTANCE_PX, 1);
  const top = `calc(env(safe-area-inset-top) + ${pullDistance / 2 - SIZE / 2}px)`;
  const progress = committed || isRefreshing ? 1 : Math.min(pullDistance / 80, 1);

  return (
    <div
      data-testid="pull-to-refresh-indicator"
      data-committed={committed ? "true" : "false"}
      data-refreshing={isRefreshing ? "true" : "false"}
      style={{
        position: "absolute",
        top,
        left: "50%",
        transform: "translateX(-50%)",
        width: SIZE,
        height: SIZE,
        opacity,
        pointerEvents: "none",
        color: committed || isRefreshing ? t.ink : t.ink2,
      }}
    >
      <svg
        width={SIZE}
        height={SIZE}
        viewBox="0 0 32 32"
        fill="none"
        stroke="currentColor"
        strokeWidth={2.5}
        strokeLinecap="round"
        aria-hidden="true"
      >
        {PETALS.map((index) => (
          <path
            key={index}
            className="ptr-petal"
            d="M16 3v5"
            transform={`rotate(${index * 30} 16 16)`}
            style={{
              opacity: isRefreshing
                ? 1
                : 0.16 + 0.84 * Math.max(0, Math.min(progress * 12 - index, 1)),
              animationDelay: `${index / 12 - 1}s`,
            }}
          />
        ))}
      </svg>
      <style>{`
        [data-refreshing="true"] .ptr-petal {
          animation: ptr-petal-fade 1s linear infinite;
        }
        @keyframes ptr-petal-fade {
          from { opacity: 1; }
          to { opacity: 0.16; }
        }
        @media (prefers-reduced-motion: reduce) {
          [data-refreshing="true"] .ptr-petal { animation: none; }
        }
      `}</style>
    </div>
  );
}
