import type { CSSProperties, ReactNode } from "react";
import { theme } from "../theme";

type Props = {
  onClick: () => void;
  children: ReactNode;
  size?: number;
  background?: string;
  color?: string;
  disabled?: boolean;
  style?: CSSProperties;
  "aria-label"?: string;
  "data-testid"?: string;
};

export function IconButton({
  onClick,
  children,
  size = 36,
  background = theme.card,
  color = theme.ink,
  disabled = false,
  style,
  ...rest
}: Props) {
  // iOS tap targets must be ≥ 44×44px even when the visual circle is
  // smaller; the negative margin keeps the layout footprint at `size` so
  // surrounding flex spacing doesn't shift.
  const hit = Math.max(size, 44);
  const inset = (hit - size) / 2;
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      {...rest}
      style={{
        width: hit,
        height: hit,
        margin: -inset,
        background: "transparent",
        border: "none",
        color,
        cursor: disabled ? "default" : "pointer",
        display: "grid",
        placeItems: "center",
        padding: 0,
        flex: "0 0 auto",
        ...style,
      }}
    >
      <span
        style={{
          width: size,
          height: size,
          borderRadius: size / 2,
          background,
          display: "grid",
          placeItems: "center",
        }}
      >
        {children}
      </span>
    </button>
  );
}
