// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSyncOnVisible } from "./usePullToRefresh";

function setVisibility(state: "visible" | "hidden") {
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    get: () => state,
  });
}

// Dispatch an event and let the burst-coalescing window elapse.
function flushBurst() {
  vi.advanceTimersByTime(60);
}

describe("useSyncOnVisible", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setVisibility("visible");
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("fires when the document becomes visible", () => {
    const cb = vi.fn();
    renderHook(() => useSyncOnVisible(cb));
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
      flushBurst();
    });
    expect(cb).toHaveBeenCalledTimes(1);
  });

  it("does not fire while the document is hidden", () => {
    const cb = vi.fn();
    renderHook(() => useSyncOnVisible(cb));
    setVisibility("hidden");
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
      flushBurst();
    });
    expect(cb).not.toHaveBeenCalled();
  });

  // The reason this hook exists: iOS doesn't fire visibilitychange on every
  // wake path, so the data sync must also catch pageshow / focus / online.
  it.each(["pageshow", "focus", "online"])("fires on %s", (evt) => {
    const cb = vi.fn();
    renderHook(() => useSyncOnVisible(cb));
    act(() => {
      window.dispatchEvent(new Event(evt));
      flushBurst();
    });
    expect(cb).toHaveBeenCalledTimes(1);
  });

  it("coalesces the burst iOS fires on a single resume into ONE call", () => {
    const cb = vi.fn();
    renderHook(() => useSyncOnVisible(cb));
    act(() => {
      // A single resume commonly emits all three together.
      window.dispatchEvent(new Event("pageshow"));
      window.dispatchEvent(new Event("focus"));
      document.dispatchEvent(new Event("visibilitychange"));
      flushBurst();
    });
    expect(cb).toHaveBeenCalledTimes(1);
  });

  it("fires again on a later, separate resume", () => {
    const cb = vi.fn();
    renderHook(() => useSyncOnVisible(cb));
    act(() => {
      window.dispatchEvent(new Event("focus"));
      flushBurst();
    });
    act(() => {
      window.dispatchEvent(new Event("focus"));
      flushBurst();
    });
    expect(cb).toHaveBeenCalledTimes(2);
  });

  it("always invokes the latest callback without re-registering listeners", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(({ cb }) => useSyncOnVisible(cb), {
      initialProps: { cb: first },
    });
    rerender({ cb: second });
    act(() => {
      window.dispatchEvent(new Event("focus"));
      flushBurst();
    });
    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });

  it("removes its listeners and timer on unmount", () => {
    const cb = vi.fn();
    const removeDoc = vi.spyOn(document, "removeEventListener");
    const removeWin = vi.spyOn(window, "removeEventListener");
    const { unmount } = renderHook(() => useSyncOnVisible(cb));

    unmount();

    expect(removeDoc).toHaveBeenCalledWith(
      "visibilitychange",
      expect.any(Function),
    );
    for (const evt of ["pageshow", "focus", "online"]) {
      expect(removeWin).toHaveBeenCalledWith(evt, expect.any(Function));
    }
    removeDoc.mockRestore();
    removeWin.mockRestore();
  });
});
