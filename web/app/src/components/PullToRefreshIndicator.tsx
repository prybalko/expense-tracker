import { theme } from "../theme";

type Props = {
  // pullDistance is the rendered displacement coming out of usePullToRefresh
  // (already after RESISTANCE scaling). Used to position the indicator above
  // the content and fade it in as the user pulls.
  pullDistance: number;
  // committed flips true once the user has pulled past the trigger threshold,
  // before they release. The indicator switches to its filled "release to
  // refresh" state so the user knows their gesture will fire.
  committed: boolean;
  // isRefreshing is true from the start of the refresh promise to its
  // settlement. The indicator stays pinned at the commit position and the
  // arc switches into a continuous spin animation.
  isRefreshing: boolean;
};

// Diameter and stroke chosen to read at iPhone-pixel densities without
// looking like a system control — the Linen & Ink palette is warm and quiet,
// so the indicator uses the same ink color rather than the platform blue
// you'd expect from a webview control.
const SIZE = 28;
const STROKE = 2.5;
// FADE_DISTANCE_PX is the pull distance over which the indicator goes from
// fully transparent to fully opaque. Smaller than the trigger threshold so
// the user has visual feedback well before they're committed.
const FADE_DISTANCE_PX = 48;

export function PullToRefreshIndicator({
  pullDistance,
  committed,
  isRefreshing,
}: Props) {
  const t = theme;
  // Hide entirely when there's nothing to show. Keeping the element mounted
  // with opacity 0 would still intercept hit-testing on the hero area below.
  if (pullDistance === 0 && !isRefreshing) return null;

  const opacity = Math.min(pullDistance / FADE_DISTANCE_PX, 1);
  // Drop the indicator into the space the translated content vacated. The
  // gap below the safe-area inset is `pullDistance` px tall; placing the
  // glyph at half that distance centers it visually inside the gap.
  const top = `calc(env(safe-area-inset-top) + ${pullDistance / 2 - SIZE / 2}px)`;

  // Stroke arc fills proportionally up to commit, then stays full while
  // refreshing. Same geometry whether pulling or committed; only the visual
  // weight changes.
  const radius = (SIZE - STROKE) / 2;
  const circumference = 2 * Math.PI * radius;
  const fillFraction = isRefreshing
    ? 0.75
    : Math.min(pullDistance / 80, 1);
  const dashOffset = circumference * (1 - fillFraction);

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
        // Spin while refreshing — same direction & cadence as iOS native to
        // stay in users' muscle memory.
        animation: isRefreshing
          ? "ptr-spin 0.9s linear infinite"
          : undefined,
      }}
    >
      <svg
        width={SIZE}
        height={SIZE}
        viewBox={`0 0 ${SIZE} ${SIZE}`}
        fill="none"
        stroke="currentColor"
        strokeWidth={STROKE}
        strokeLinecap="round"
      >
        <circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={radius}
          opacity={0.18}
        />
        <circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={radius}
          strokeDasharray={circumference}
          strokeDashoffset={dashOffset}
          transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}
          style={{ transition: isRefreshing ? "none" : "stroke-dashoffset 80ms linear" }}
        />
      </svg>
      <style>{`
        @keyframes ptr-spin {
          from { transform: translateX(-50%) rotate(0deg); }
          to { transform: translateX(-50%) rotate(360deg); }
        }
      `}</style>
    </div>
  );
}
