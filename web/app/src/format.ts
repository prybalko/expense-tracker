export function fmtEUR(
  n: number,
  opts: { sign?: boolean; cents?: boolean } = {},
): string {
  const { sign = false, cents = true } = opts;
  // Format the magnitude and prepend the sign explicitly — toFixed on a
  // negative would otherwise leak its ASCII "-" inside the euro symbol
  // ("−€-5.00" with sign, "€-5.00" without).
  const v = Math.abs(n).toFixed(cents ? 2 : 0);
  const grouped = v.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  const prefix = sign && n < 0 ? "−" : "";
  return `${prefix}€${grouped}`;
}

export function splitInt(n: number): { int: string; dec: string } {
  const [i, d] = n.toFixed(2).split(".");
  return { int: i.replace(/\B(?=(\d{3})+(?!\d))/g, ","), dec: d };
}
