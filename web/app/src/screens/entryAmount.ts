import type { KeypadKey } from "../components/Keypad";

// Keypad prefill: strip padding zeros ("50.00" → "50", "12.50" → "12.5") so
// the first ⌫ press removes a digit the user can see.
export function amountToKeypadString(amount: number): string {
  return amount.toFixed(2).replace(/\.?0+$/, "");
}

export function applyKeypadKey(a: string, k: KeypadKey): string {
  if (k === "del") {
    const next = a.slice(0, -1);
    return next.endsWith(".") ? next.slice(0, -1) : next;
  }
  if (k === ".") {
    if (a.includes(".")) return a;
    if (a === "") return "0.";
    return a + ".";
  }
  if (a === "0") return k;
  const [, decPart] = a.split(".");
  if (decPart && decPart.length >= 2) return a;
  return (a + k).slice(0, 8);
}
