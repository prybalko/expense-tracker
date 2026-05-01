import { theme, FONT } from "../theme";

export type KeypadKey =
  | "0"
  | "1"
  | "2"
  | "3"
  | "4"
  | "5"
  | "6"
  | "7"
  | "8"
  | "9"
  | "."
  | "del";

const KEYS: KeypadKey[] = [
  "1",
  "2",
  "3",
  "4",
  "5",
  "6",
  "7",
  "8",
  "9",
  ".",
  "0",
  "del",
];

type Props = {
  onPress: (key: KeypadKey) => void;
};

export function Keypad({ onPress }: Props) {
  const t = theme;
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(3, 1fr)",
        gap: 6,
        marginTop: 10,
      }}
    >
      {KEYS.map((k) => (
        <button
          key={k}
          type="button"
          onClick={() => onPress(k)}
          style={{
            padding: "12px 0",
            background: t.cardAlt,
            border: "none",
            borderRadius: 14,
            fontSize: 22,
            fontWeight: 500,
            color: t.ink,
            cursor: "pointer",
            fontFamily: FONT,
          }}
        >
          {k === "del" ? "⌫" : k}
        </button>
      ))}
    </div>
  );
}
