import type { CSSProperties, ReactNode } from "react";

type Props = {
  icon: string;
  size?: number;
};

const baseStyle = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.6,
  strokeLinecap: "round",
  strokeLinejoin: "round",
} satisfies CSSProperties;

const ICONS: Record<string, ReactNode> = {
  groceries: (
    <>
      <path d="M3 5h2l2.4 10.5a2 2 0 0 0 2 1.5h7.6a2 2 0 0 0 2-1.5L21 8H6" />
      <circle cx="10" cy="20" r="1.2" />
      <circle cx="17" cy="20" r="1.2" />
    </>
  ),
  travel: (
    <path d="M2 16l8-2 4-9 2 1-2 8 6-1.5 1.5 1.5-8 4-2 5-1.5-.5 1-5z" />
  ),
  housing: (
    <path d="M3 11l9-7 9 7v9a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1z" />
  ),
  health: (
    <path d="M12 21s-7-4.5-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 11c0 5.5-7 10-7 10z" />
  ),
  eating: (
    <path d="M7 3v8a2 2 0 0 0 2 2v8M7 3v6M11 3v6M9 3v6M16 3c-2 0-3 2-3 5s1 5 3 5v8" />
  ),
  transport: (
    <>
      <rect x="4" y="6" width="16" height="11" rx="2" />
      <path d="M4 12h16M8 17v2M16 17v2" />
      <circle cx="8.5" cy="14" r=".8" fill="currentColor" />
      <circle cx="15.5" cy="14" r=".8" fill="currentColor" />
    </>
  ),
  utilities: <path d="M13 3l-7 11h5l-1 7 7-11h-5z" />,
  gifts: (
    <>
      <rect x="3" y="8" width="18" height="4" />
      <path d="M5 12v9h14v-9M12 8v13M8 8a2.5 2.5 0 1 1 0-5c2 0 4 5 4 5s-2 0-4 0zM16 8a2.5 2.5 0 1 0 0-5c-2 0-4 5-4 5s2 0 4 0z" />
    </>
  ),
  fitness: <path d="M6 8v8M9 6v12M15 6v12M18 8v8M2 12h2M20 12h2M9 12h6" />,
  entertainment: (
    <>
      <rect x="3" y="6" width="18" height="12" rx="2" />
      <path d="M10 10l5 2.5L10 15z" fill="currentColor" stroke="none" />
    </>
  ),
  fashion: <path d="M6 7l3-3h6l3 3-3 2v11H9V9z" />,
  pets: (
    <>
      <circle cx="6" cy="9" r="1.6" />
      <circle cx="10" cy="6" r="1.6" />
      <circle cx="14" cy="6" r="1.6" />
      <circle cx="18" cy="9" r="1.6" />
      <path d="M12 11c-3 0-5 3-5 5a3 3 0 0 0 3 3c1 0 2-.5 2-.5s1 .5 2 .5a3 3 0 0 0 3-3c0-2-2-5-5-5z" />
    </>
  ),
  subs: <path d="M21 12a9 9 0 1 1-3-6.7M21 4v5h-5" />,
  kids: (
    <>
      <circle cx="12" cy="6" r="3" />
      <path d="M6 21v-4a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v4M9 9c.5 1 1.5 2 3 2s2.5-1 3-2" />
    </>
  ),
  coffee: (
    <>
      <path d="M4 8h13v6a4 4 0 0 1-4 4H8a4 4 0 0 1-4-4z" />
      <path d="M17 10h2a2 2 0 0 1 0 4h-2" />
      <path d="M8 4c0 1 1 1 1 2s-1 1-1 2M12 4c0 1 1 1 1 2s-1 1-1 2" />
    </>
  ),
  drinks: (
    <>
      <path d="M6 3h12l-1.5 7a4 4 0 0 1-4 3h-1a4 4 0 0 1-4-3z" />
      <path d="M12 13v7M9 21h6" />
    </>
  ),
  fuel: (
    <>
      <rect x="4" y="3" width="10" height="18" rx="1.5" />
      <path d="M4 11h10M14 8l3 3v7a2 2 0 0 0 2 2v-9l-3-3M8 6h2" />
    </>
  ),
  parking: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="3" />
      <path d="M9 17V7h4a3 3 0 0 1 0 6H9" />
    </>
  ),
  taxi: (
    <>
      <path d="M5 11l1.5-4a2 2 0 0 1 2-1.5h7a2 2 0 0 1 2 1.5L19 11" />
      <rect x="3" y="11" width="18" height="6" rx="1.5" />
      <path d="M6 17v2M18 17v2M9 5h6" />
      <circle cx="7.5" cy="14.5" r=".8" fill="currentColor" />
      <circle cx="16.5" cy="14.5" r=".8" fill="currentColor" />
    </>
  ),
  flights: (
    <path d="M21 12c0-.7-.4-1.3-1-1.6L14 8 11 2H9l2 7-4 1-2-2H4l1 4-1 4h1l2-2 4 1-2 7h2l3-6 6-2.4c.6-.3 1-.9 1-1.6z" />
  ),
  hotel: (
    <>
      <path d="M3 21V8h18v13M3 13h18" />
      <circle cx="8" cy="11" r="1.5" />
      <path d="M11 11h7M3 8l9-5 9 5" />
    </>
  ),
  rent: (
    <>
      <path d="M3 11l9-7 9 7v9a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1z" />
      <path d="M9 21v-6h6v6" />
      <circle cx="12" cy="11" r="1.5" />
    </>
  ),
  internet: (
    <>
      <path d="M2 9a14 14 0 0 1 20 0M5 13a9 9 0 0 1 14 0M8.5 16.5a4.5 4.5 0 0 1 7 0" />
      <circle cx="12" cy="20" r="1.2" fill="currentColor" />
    </>
  ),
  phone: (
    <>
      <rect x="6" y="2" width="12" height="20" rx="2.5" />
      <path d="M10 5h4" />
      <circle cx="12" cy="18.5" r="1" fill="currentColor" />
    </>
  ),
  insurance: (
    <>
      <path d="M12 2l8 3v7c0 5-3.5 8.5-8 10-4.5-1.5-8-5-8-10V5z" />
      <path d="M9 12l2.5 2.5L16 10" />
    </>
  ),
  taxes: (
    <>
      <rect x="4" y="3" width="16" height="18" rx="1.5" />
      <path d="M8 8h8M8 12h8M8 16h5" />
    </>
  ),
  education: (
    <>
      <path d="M2 9l10-5 10 5-10 5z" />
      <path d="M6 11v5c0 1.5 2.7 3 6 3s6-1.5 6-3v-5" />
      <path d="M22 9v5" />
    </>
  ),
  books: (
    <>
      <path d="M4 4h6a3 3 0 0 1 3 3v13a2 2 0 0 0-2-2H4z" />
      <path d="M20 4h-6a3 3 0 0 0-3 3v13a2 2 0 0 1 2-2h7z" />
    </>
  ),
  music: (
    <>
      <path d="M9 18V6l11-2v12" />
      <circle cx="6" cy="18" r="3" />
      <circle cx="17" cy="16" r="3" />
    </>
  ),
  movies: (
    <>
      <rect x="3" y="5" width="18" height="14" rx="1.5" />
      <path d="M3 9h18M3 15h18M7 5v14M17 5v14" />
    </>
  ),
  games: (
    <>
      <path d="M6 8h12a4 4 0 0 1 4 4v2a3 3 0 0 1-5.5 1.7L15 14H9l-1.5 1.7A3 3 0 0 1 2 14v-2a4 4 0 0 1 4-4z" />
      <path d="M7 11v2M6 12h2" />
      <circle cx="16" cy="11" r=".9" fill="currentColor" />
      <circle cx="18" cy="13" r=".9" fill="currentColor" />
    </>
  ),
  hobby: (
    <path d="M12 3l2.5 5.5L20 9l-4 4 1 6-5-3-5 3 1-6-4-4 5.5-.5z" />
  ),
  beauty: (
    <>
      <path d="M9 3h6v5l2 2v9a2 2 0 0 1-2 2h-6a2 2 0 0 1-2-2v-9l2-2z" />
      <path d="M9 12h6" />
    </>
  ),
  laundry: (
    <>
      <rect x="4" y="3" width="16" height="18" rx="2" />
      <circle cx="12" cy="13" r="4.5" />
      <circle cx="7.5" cy="6.5" r=".8" fill="currentColor" />
      <circle cx="10" cy="6.5" r=".8" fill="currentColor" />
    </>
  ),
  charity: (
    <>
      <path d="M12 21s-7-4.5-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 11c0 5.5-7 10-7 10z" />
      <path d="M9 11l2 2 4-4" />
    </>
  ),
  tools: (
    <>
      <path d="M14.5 5.5a3.5 3.5 0 0 0 4.5 4.5l-9 9a2.5 2.5 0 0 1-3.5-3.5z" />
      <path d="M5 19l1.5 1.5" />
    </>
  ),
  garden: (
    <>
      <path d="M12 21V10" />
      <path d="M12 10c0-3-2-5-5-5 0 3 2 5 5 5zM12 10c0-3 2-5 5-5 0 3-2 5-5 5z" />
      <path d="M5 21h14" />
    </>
  ),
  cash: (
    <>
      <rect x="3" y="6" width="18" height="12" rx="1.5" />
      <circle cx="12" cy="12" r="2.5" />
      <path d="M6 9v.01M18 15v.01" />
    </>
  ),
  fees: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M9 9l6 6M15 9l-6 6" />
    </>
  ),
  work: (
    <>
      <rect x="3" y="7" width="18" height="13" rx="2" />
      <path d="M9 7V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2M3 13h18" />
    </>
  ),
  other: (
    <>
      <circle cx="6" cy="12" r="1.4" fill="currentColor" />
      <circle cx="12" cy="12" r="1.4" fill="currentColor" />
      <circle cx="18" cy="12" r="1.4" fill="currentColor" />
    </>
  ),
};

export function CategoryGlyph({ icon, size = 18 }: Props) {
  const style: CSSProperties = { width: size, height: size, ...baseStyle };
  const body = ICONS[icon] ?? ICONS.other;
  return (
    <svg viewBox="0 0 24 24" style={style}>
      {body}
    </svg>
  );
}
