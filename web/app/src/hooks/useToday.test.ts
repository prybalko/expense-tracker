// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useToday } from "./useToday";

// jsdom defines visibilityState as a getter; override it per-test so we can
// simulate the PWA being foreground/background.
function setVisibility(state: "visible" | "hidden") {
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    get: () => state,
  });
}

describe("useToday", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setVisibility("visible");
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns the current calendar day at start-of-day", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 9, 30, 0)); // 16 Jun 2026, 09:30 local
    const { result } = renderHook(() => useToday());

    expect(result.current.getFullYear()).toBe(2026);
    expect(result.current.getMonth()).toBe(5);
    expect(result.current.getDate()).toBe(16);
    expect(result.current.getHours()).toBe(0);
    expect(result.current.getMinutes()).toBe(0);
  });

  it("keeps the same Date reference across an intra-day visibilitychange (no re-render thrash)", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 9, 0, 0));
    const { result } = renderHook(() => useToday());
    const first = result.current;

    // Foreground a few hours later — still the same calendar day.
    vi.setSystemTime(new Date(2026, 5, 16, 14, 0, 0));
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // Identical reference => React bailed out, no downstream memo invalidation.
    expect(result.current).toBe(first);
  });

  // The reported bug: open the PWA in the morning and yesterday's data still
  // says "today". This is the fix — resuming after midnight advances `today`.
  it("advances to the new day when the PWA resumes after midnight (visibilitychange)", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 23, 0, 0)); // last night
    const { result } = renderHook(() => useToday());
    expect(result.current.getDate()).toBe(16);

    // Overnight background freeze, resumed the next morning.
    vi.setSystemTime(new Date(2026, 5, 17, 8, 0, 0));
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(result.current.getMonth()).toBe(5);
    expect(result.current.getDate()).toBe(17);
  });

  it("advances on window focus after the day rolls over", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 23, 0, 0));
    const { result } = renderHook(() => useToday());

    vi.setSystemTime(new Date(2026, 5, 17, 8, 0, 0));
    act(() => {
      window.dispatchEvent(new Event("focus"));
    });

    expect(result.current.getDate()).toBe(17);
  });

  it("advances on pageshow (bfcache restore) after the day rolls over", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 23, 0, 0));
    const { result } = renderHook(() => useToday());

    vi.setSystemTime(new Date(2026, 5, 17, 8, 0, 0));
    act(() => {
      window.dispatchEvent(new Event("pageshow"));
    });

    expect(result.current.getDate()).toBe(17);
  });

  it("rolls over via the midnight timer with no user interaction", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 23, 59, 30)); // 30s before midnight
    const { result } = renderHook(() => useToday());
    expect(result.current.getDate()).toBe(16);

    // Cross midnight by letting the scheduled timer fire (fake timers also
    // advance the mocked clock, so `new Date()` reads the 17th when it runs).
    act(() => {
      vi.advanceTimersByTime(35 * 1000);
    });

    expect(result.current.getDate()).toBe(17);
  });

  it("ignores a visibilitychange while the document is hidden", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 23, 0, 0));
    const { result } = renderHook(() => useToday());

    setVisibility("hidden");
    vi.setSystemTime(new Date(2026, 5, 17, 8, 0, 0));
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    expect(result.current.getDate()).toBe(16); // hidden => no refresh

    // ...and catches up once it becomes visible again.
    setVisibility("visible");
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    expect(result.current.getDate()).toBe(17);
  });

  it("removes its listeners and timer on unmount", () => {
    vi.setSystemTime(new Date(2026, 5, 16, 12, 0, 0));
    const removeSpy = vi.spyOn(document, "removeEventListener");
    const { unmount } = renderHook(() => useToday());

    unmount();

    expect(removeSpy).toHaveBeenCalledWith(
      "visibilitychange",
      expect.any(Function),
    );
    // No pending timers survive the unmount.
    expect(vi.getTimerCount()).toBe(0);
    removeSpy.mockRestore();
  });
});
