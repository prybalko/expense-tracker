import { describe, it, expect } from "vitest";
import { resolveSubmitDate } from "./entryDate";

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
