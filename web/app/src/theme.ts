export const FONT = "'DM Sans', -apple-system, system-ui, sans-serif";

export type CategoryTone = {
  bg: string;
  ink: string;
};

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
  cat: Record<string, CategoryTone>;
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
  cat: {
    groceries: { bg: "#E5E2F0", ink: "#322B66" },
    travel: { bg: "#F2D7DA", ink: "#6E2730" },
    housing: { bg: "#DEE3EC", ink: "#26384D" },
    health: { bg: "#EFE0C5", ink: "#5C3D18" },
    eating: { bg: "#DAE6DC", ink: "#234633" },
    fashion: { bg: "#EBD7CB", ink: "#5C351E" },
    transport: { bg: "#D8E2DD", ink: "#28433D" },
    utilities: { bg: "#E5D6EA", ink: "#3E1F66" },
    gifts: { bg: "#F1D6DD", ink: "#612434" },
    fitness: { bg: "#D4E4D6", ink: "#214A2D" },
    entertainment: { bg: "#E0D6EE", ink: "#3D205C" },
    pets: { bg: "#EDDCC4", ink: "#5A3A18" },
    subs: { bg: "#DCDDEE", ink: "#2A2D5C" },
    kids: { bg: "#F0E5C9", ink: "#5A4218" },
    other: { bg: "#E0DCD4", ink: "#3A352E" },
  },
};

export const theme = linen;
