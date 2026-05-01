import type { CSSProperties } from "react";

type Props = {
  slug: string;
  size?: number;
};

const baseStyle = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.6,
  strokeLinecap: "round",
  strokeLinejoin: "round",
} satisfies CSSProperties;

export function CategoryGlyph({ slug, size = 18 }: Props) {
  const style: CSSProperties = { width: size, height: size, ...baseStyle };

  switch (slug) {
    case "groceries":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M3 5h2l2.4 10.5a2 2 0 0 0 2 1.5h7.6a2 2 0 0 0 2-1.5L21 8H6" />
          <circle cx="10" cy="20" r="1.2" />
          <circle cx="17" cy="20" r="1.2" />
        </svg>
      );
    case "travel":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M2 16l8-2 4-9 2 1-2 8 6-1.5 1.5 1.5-8 4-2 5-1.5-.5 1-5z" />
        </svg>
      );
    case "housing":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M3 11l9-7 9 7v9a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z" />
        </svg>
      );
    case "health":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M12 21s-7-4.5-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 11c0 5.5-7 10-7 10z" />
        </svg>
      );
    case "eating":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M7 3v8a2 2 0 0 0 2 2v8M7 3v6M11 3v6M9 3v6M16 3c-2 0-3 2-3 5s1 5 3 5v8" />
        </svg>
      );
    case "fashion":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M6 7l3-3h6l3 3-3 2v11H9V9z" />
        </svg>
      );
    case "transport":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <rect x="4" y="6" width="16" height="11" rx="2" />
          <path d="M4 12h16M8 17v2M16 17v2" />
          <circle cx="8.5" cy="14" r=".8" fill="currentColor" />
          <circle cx="15.5" cy="14" r=".8" fill="currentColor" />
        </svg>
      );
    case "utilities":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M13 3l-7 11h5l-1 7 7-11h-5z" />
        </svg>
      );
    case "gifts":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <rect x="3" y="8" width="18" height="4" />
          <path d="M5 12v9h14v-9M12 8v13M8 8a2.5 2.5 0 1 1 0-5c2 0 4 5 4 5s-2 0-4 0zM16 8a2.5 2.5 0 1 0 0-5c-2 0-4 5-4 5s2 0 4 0z" />
        </svg>
      );
    case "fitness":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M6 8v8M9 6v12M15 6v12M18 8v8M2 12h2M20 12h2M9 12h6" />
        </svg>
      );
    case "pets":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <circle cx="6" cy="9" r="1.6" />
          <circle cx="10" cy="6" r="1.6" />
          <circle cx="14" cy="6" r="1.6" />
          <circle cx="18" cy="9" r="1.6" />
          <path d="M12 11c-3 0-5 3-5 5a3 3 0 0 0 3 3c1 0 2-.5 2-.5s1 .5 2 .5a3 3 0 0 0 3-3c0-2-2-5-5-5z" />
        </svg>
      );
    case "subs":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <path d="M21 12a9 9 0 1 1-3-6.7M21 4v5h-5" />
        </svg>
      );
    case "kids":
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <circle cx="12" cy="6" r="3" />
          <path d="M6 21v-4a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v4M9 9c.5 1 1.5 2 3 2s2.5-1 3-2" />
        </svg>
      );
    case "other":
    default:
      return (
        <svg viewBox="0 0 24 24" style={style}>
          <circle cx="6" cy="12" r="1.4" fill="currentColor" />
          <circle cx="12" cy="12" r="1.4" fill="currentColor" />
          <circle cx="18" cy="12" r="1.4" fill="currentColor" />
        </svg>
      );
  }
}
