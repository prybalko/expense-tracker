import { describe, it, expect } from "vitest";
import { resolveSubmitDate } from "./entryDate";
import { amountToKeypadString, applyKeypadKey } from "./entryAmount";

describe("resolveSubmitDate", () => {
  const yesterday = new Date(2026, 5, 16); // the day a left-open form was opened
  const now = new Date(2026, 5, 17, 8, 30); // the day the user actually submits

  // The reported bug: a new expense created after an overnight resume gets
  // yesterday's date. For a brand-new, untouched expense we re-read the wall
  // clock at submit so the stored day is always the real current day.
  it("uses the current day for a new expense the user never touched", () => {
    expect(resolveSubmitDate(false, false, yesterday, now)).toBe(now);
  });

  it("keeps an explicit user pick on a new expense", () => {
    const picked = new Date(2026, 5, 14);
    expect(resolveSubmitDate(false, true, picked, now)).toBe(picked);
  });

  it("never overrides the date when editing, even if untouched", () => {
    expect(resolveSubmitDate(true, false, yesterday, now)).toBe(yesterday);
  });

  it("keeps an explicit pick when editing", () => {
    const picked = new Date(2026, 5, 10);
    expect(resolveSubmitDate(true, true, picked, now)).toBe(picked);
  });
});

describe("amountToKeypadString", () => {
  it("strips the whole decimal part when it is all zeros", () => {
    expect(amountToKeypadString(50)).toBe("50");
    expect(amountToKeypadString(100)).toBe("100");
  });

  it("strips only the trailing zero from x.x0 amounts", () => {
    expect(amountToKeypadString(12.5)).toBe("12.5");
    expect(amountToKeypadString(10.5)).toBe("10.5");
  });

  it("keeps both cents digits when they are significant", () => {
    expect(amountToKeypadString(12.34)).toBe("12.34");
    expect(amountToKeypadString(25.99)).toBe("25.99");
  });

  it("never eats zeros belonging to the integer part", () => {
    expect(amountToKeypadString(20)).toBe("20");
    expect(amountToKeypadString(200.1)).toBe("200.1");
  });

  it("handles sub-euro and zero amounts", () => {
    expect(amountToKeypadString(0.5)).toBe("0.5");
    expect(amountToKeypadString(0.05)).toBe("0.05");
    expect(amountToKeypadString(0)).toBe("0");
  });
});

describe("applyKeypadKey", () => {
  it("appends digits and caps at two decimals / eight chars", () => {
    expect(applyKeypadKey("16", "7")).toBe("167");
    expect(applyKeypadKey("1.23", "4")).toBe("1.23");
    expect(applyKeypadKey("12345678", "9")).toBe("12345678");
  });

  it("replaces a bare leading zero", () => {
    expect(applyKeypadKey("0", "5")).toBe("5");
  });

  it("starts cents on dot, at most one dot", () => {
    expect(applyKeypadKey("", ".")).toBe("0.");
    expect(applyKeypadKey("16", ".")).toBe("16.");
    expect(applyKeypadKey("16.7", ".")).toBe("16.7");
  });

  it("del removes the last character", () => {
    expect(applyKeypadKey("12.34", "del")).toBe("12.3");
    expect(applyKeypadKey("12", "del")).toBe("1");
  });

  it("del never strands a dangling dot", () => {
    expect(applyKeypadKey("16.7", "del")).toBe("16");
    expect(applyKeypadKey("16.", "del")).toBe("16");
  });

  it("del on empty is a no-op", () => {
    expect(applyKeypadKey("", "del")).toBe("");
  });
});
