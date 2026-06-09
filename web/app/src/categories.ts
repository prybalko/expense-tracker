import data from "./categories.json";
import extendedData from "./categoriesExtended.json";
import type { Category } from "./types";

// Active categories — the set the entry-form picker offers.
export const categories: Category[] = data;

// Staged extended palette — icons + tones for categories not yet in the
// active picker. Used by lookup so any expense whose category matches one
// of these slugs/labels (e.g. imported from elsewhere, or activated in a
// future change) renders with the right glyph and tone.
export const extendedCategories: Category[] = extendedData;
