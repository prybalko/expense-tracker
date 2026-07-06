import type { CSSProperties, ReactNode } from "react";
import { theme } from "../theme";

type Props = {
  children: ReactNode;
  style?: CSSProperties;
};

export function SectionLabel({ children, style }: Props) {
  return (
    <div
      style={{
        fontSize: 11,
        fontWeight: 500,
        letterSpacing: "0.04em",
        textTransform: "uppercase",
        color: theme.ink2,
        ...style,
      }}
    >
      {children}
    </div>
  );
}
