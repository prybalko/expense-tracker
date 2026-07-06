import type { CSSProperties, ReactNode } from "react";
import { theme } from "../theme";

type Props = {
  children: ReactNode;
  card?: boolean;
  style?: CSSProperties;
};

export function StatusNote({ children, card = false, style }: Props) {
  return (
    <div
      style={{
        padding: "32px 16px",
        textAlign: "center",
        color: theme.ink2,
        fontSize: 13,
        ...(card ? { background: theme.card, borderRadius: 22 } : null),
        ...style,
      }}
    >
      {children}
    </div>
  );
}
