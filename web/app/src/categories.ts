import data from "./categories.json";
import extendedData from "./categoriesExtended.json";
import type { Category } from "./types";

export const categories: Category[] = data;

// Not offered in the picker; lookup still resolves them to a glyph and tone.
export const extendedCategories: Category[] = extendedData;
