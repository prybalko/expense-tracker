// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, act, cleanup } from "@testing-library/react";

// Mock the data layer so the test exercises the real resume wiring
// (useSyncOnVisible + useStableCallback) against a fake sync mutation.
const { mockSync, mockExpenses } = vi.hoisted(() => ({
  mockSync: { mutate: vi.fn(), isPending: false },
  mockExpenses: { isSuccess: false },
}));

vi.mock("../hooks/useExpenses", () => ({
  useSyncExpenses: () => mockSync,
  useAllExpenses: () => mockExpenses,
}));
vi.mock("../hooks/useErrorBanner", () => ({
  useErrorBanner: () => ({ showError: vi.fn(), clear: vi.fn() }),
}));

import { ResumeSync } from "./ResumeSync";

function flushBurst() {
  vi.advanceTimersByTime(60);
}

describe("ResumeSync", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockSync.mutate.mockReset();
    mockSync.isPending = false;
    mockExpenses.isSuccess = false;
  });
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("fires a delta sync when the app resumes (on any route)", () => {
    render(<ResumeSync />);
    act(() => {
      // A resume that lands on a non-Feed screen still triggers the sync,
      // because ResumeSync lives above the router.
      window.dispatchEvent(new Event("focus"));
      flushBurst();
    });
    expect(mockSync.mutate).toHaveBeenCalledTimes(1);
  });

  it("does not start a second sync while one is in flight", () => {
    mockSync.isPending = true;
    render(<ResumeSync />);
    act(() => {
      window.dispatchEvent(new Event("focus"));
      flushBurst();
    });
    expect(mockSync.mutate).not.toHaveBeenCalled();
  });

  it("fires a one-shot catch-up sync after the initial load, only once", () => {
    mockExpenses.isSuccess = true;
    const { rerender } = render(<ResumeSync />);
    expect(mockSync.mutate).toHaveBeenCalledTimes(1);

    // A later re-render (e.g. the list query settling again) must not re-fire.
    rerender(<ResumeSync />);
    expect(mockSync.mutate).toHaveBeenCalledTimes(1);
  });

  it("does not run the catch-up sync until the initial load succeeds", () => {
    mockExpenses.isSuccess = false;
    render(<ResumeSync />);
    expect(mockSync.mutate).not.toHaveBeenCalled();
  });
});
