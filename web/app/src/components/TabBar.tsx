import type { ReactNode } from "react";
import { theme, FONT } from "../theme";
import { vibrate } from "../utils/haptics";

export type TabId = "feed" | "insights";

type Props = {
  current: TabId;
  onNavigate: (id: TabId) => void;
  onAdd: () => void;
};

type BtnProps = {
  id: TabId;
  label: string;
  icon: ReactNode;
  current: TabId;
  onNavigate: (id: TabId) => void;
};

function TabButton({ id, label, icon, current, onNavigate }: BtnProps) {
  const t = theme;
  const sel = current === id;
  return (
    <button
      type="button"
      data-testid={`tab-${id}`}
      onClick={() => {
        vibrate();
        onNavigate(id);
      }}
      style={{
        flex: 1,
        background: "transparent",
        border: 0,
        padding: "10px 0 4px",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        gap: 2,
        color: sel ? t.ink : t.ink2,
        cursor: "pointer",
        fontFamily: FONT,
      }}
    >
      <div
        style={{
          width: 44,
          height: 26,
          borderRadius: 13,
          background: sel ? t.accentSoft : "transparent",
          display: "grid",
          placeItems: "center",
        }}
      >
        {icon}
      </div>
      <span style={{ fontSize: 13, fontWeight: 500 }}>{label}</span>
    </button>
  );
}

export function TabBar({ current, onNavigate, onAdd }: Props) {
  const t = theme;
  return (
    <div
      style={{
        background: t.card,
        borderTop: `1px solid ${t.rule}`,
        padding: "6px 14px max(6px, env(safe-area-inset-bottom))",
        display: "flex",
        alignItems: "center",
        gap: 8,
        fontFamily: FONT,
        flexShrink: 0,
        position: "relative",
        zIndex: 10,
      }}
    >
      <TabButton
        id="feed"
        label="Feed"
        current={current}
        onNavigate={onNavigate}
        icon={
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
          >
            <path d="M4 6h16M4 12h16M4 18h10" />
          </svg>
        }
      />
      <button
        type="button"
        onClick={() => {
          vibrate();
          onAdd();
        }}
        aria-label="Add expense"
        data-testid="fab-add"
        style={{
          width: 56,
          height: 56,
          borderRadius: 28,
          background: t.accent,
          color: t.accentText,
          border: 0,
          cursor: "pointer",
          boxShadow: `0 8px 22px ${t.accent}66`,
          display: "grid",
          placeItems: "center",
          padding: 0,
        }}
      >
        <svg
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M12 5v14M5 12h14" />
        </svg>
      </button>
      <TabButton
        id="insights"
        label="Insights"
        current={current}
        onNavigate={onNavigate}
        icon={
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M4 19V8M10 19V4M16 19v-7M22 19H2" />
          </svg>
        }
      />
    </div>
  );
}
