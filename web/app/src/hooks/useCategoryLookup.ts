import { categories } from "../categories";
import type { Category, CategoryColor } from "../types";

const FALLBACK: Category = {
  slug: "other",
  label: "Other",
  icon: "other",
  color: { bg: "#E0DCD4", ink: "#3A352E" },
};

const slugMap = new Map<string, Category>();
const labelMap = new Map<string, Category>();
for (const c of categories) {
  slugMap.set(c.slug, c);
  labelMap.set(c.label, c);
}

const bySlug = (slug: string) => slugMap.get(slug);
const byLabel = (label: string) => labelMap.get(label);

export type CategoryLookup = {
  bySlug: (slug: string) => Category | undefined;
  byLabel: (label: string) => Category | undefined;
  toneBySlug: (slug: string) => CategoryColor;
  toneByLabel: (label: string) => CategoryColor;
  iconBySlug: (slug: string) => string;
  iconByLabel: (label: string) => string;
  slugByLabel: (label: string) => string;
  fallback: Category;
};

const lookup: CategoryLookup = {
  bySlug,
  byLabel,
  toneBySlug: (slug) => (bySlug(slug) ?? FALLBACK).color,
  toneByLabel: (label) => (byLabel(label) ?? FALLBACK).color,
  iconBySlug: (slug) => (bySlug(slug) ?? FALLBACK).icon,
  iconByLabel: (label) => (byLabel(label) ?? FALLBACK).icon,
  slugByLabel: (label) => (byLabel(label) ?? FALLBACK).slug,
  fallback: FALLBACK,
};

export function useCategoryLookup(): CategoryLookup {
  return lookup;
}
