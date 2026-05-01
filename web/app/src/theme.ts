import type { CategoryColor } from "./types";

export const FONT = "'DM Sans', -apple-system, system-ui, sans-serif";

export type CategoryTone = CategoryColor;

export type Theme = {
  label: string;
  swatches: [string, string, string];
  bg: string;
  card: string;
  cardAlt: string;
  ink: string;
  ink2: string;
  rule: string;
  accent: string;
  accentInk: string;
  accentText: string;
  accentSoft: string;
  green: string;
  red: string;
  barOther: string;
  keyDisabled: string;
};

export const linen: Theme = {
  label: "Linen & Ink",
  swatches: ["#F4F1EA", "#FBF9F4", "#1A1714"],
  bg: "#F4F1EA",
  card: "#FBF9F4",
  cardAlt: "#FFFFFF",
  ink: "#1A1714",
  ink2: "#7A736A",
  rule: "rgba(26,23,20,0.08)",
  accent: "#1A1714",
  accentInk: "#1A1714",
  accentText: "#FBF9F4",
  accentSoft: "#E5E0D5",
  green: "oklch(0.55 0.10 150)",
  red: "oklch(0.55 0.18 22)",
  barOther: "#C7BFB1",
  keyDisabled: "#CFC8BB",
};

export const theme = linen;
