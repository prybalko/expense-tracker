import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

// Touch-only: desktop has no pull gesture.

// Rendered pull (after RESISTANCE), so about 160 px of finger travel.
// Lower values fire on the flick that ends a scroll-to-top fling.
const TRIGGER_DISTANCE_PX = 80;

const MAX_PULL_DISTANCE_PX = 120;

// Matches iOS-native pull-to-refresh (~0.5).
const RESISTANCE = 0.5;

export type PullToRefreshState = {
  // Rendered displacement in CSS px, after RESISTANCE.
  pullDistance: number;
  committed: boolean;
  isRefreshing: boolean;
};

export function usePullToRefresh(
  scrollContainerRef: RefObject<HTMLElement | null>,
  onRefresh: () => Promise<void> | void,
): PullToRefreshState {
  const [pullDistance, setPullDistance] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const onRefreshRef = useRef(onRefresh);
  const pullDistanceRef = useRef(0);
  const isRefreshingRef = useRef(false);

  useEffect(() => {
    onRefreshRef.current = onRefresh;
  }, [onRefresh]);
  useEffect(() => {
    pullDistanceRef.current = pullDistance;
  }, [pullDistance]);
  useEffect(() => {
    isRefreshingRef.current = isRefreshing;
  }, [isRefreshing]);

  useEffect(() => {
    const el = scrollContainerRef.current;
    if (!el) return;

    let startY: number | null = null;

    const reset = () => {
      startY = null;
      setPullDistance(0);
    };

    const onTouchStart = (e: TouchEvent) => {
      if (el.scrollTop > 0) return;
      if (isRefreshingRef.current) return;
      const t = e.touches[0];
      if (!t) return;
      startY = t.clientY;
    };

    const onTouchMove = (e: TouchEvent) => {
      if (startY === null) return;
      if (isRefreshingRef.current) return;
      const t = e.touches[0];
      if (!t) return;
      const delta = t.clientY - startY;
      if (delta <= 0) {
        if (pullDistanceRef.current !== 0) setPullDistance(0);
        return;
      }
      if (e.cancelable) e.preventDefault();
      const resisted = Math.min(delta * RESISTANCE, MAX_PULL_DISTANCE_PX);
      setPullDistance(resisted);
    };

    const onTouchEnd = () => {
      if (startY === null) return;
      const shouldFire = pullDistanceRef.current >= TRIGGER_DISTANCE_PX;
      if (!shouldFire) {
        reset();
        return;
      }
      startY = null;
      setIsRefreshing(true);
      setPullDistance(TRIGGER_DISTANCE_PX);
      Promise.resolve()
        .then(() => onRefreshRef.current())
        .finally(() => {
          setIsRefreshing(false);
          setPullDistance(0);
        });
    };

    // touchmove must be non-passive or iOS Safari ignores preventDefault()
    // and rubber-bands the overscroll.
    el.addEventListener("touchstart", onTouchStart, { passive: true });
    el.addEventListener("touchmove", onTouchMove, { passive: false });
    el.addEventListener("touchend", onTouchEnd, { passive: true });
    el.addEventListener("touchcancel", reset, { passive: true });

    return () => {
      el.removeEventListener("touchstart", onTouchStart);
      el.removeEventListener("touchmove", onTouchMove);
      el.removeEventListener("touchend", onTouchEnd);
      el.removeEventListener("touchcancel", reset);
    };
  }, [scrollContainerRef]);

  return {
    pullDistance,
    committed: pullDistance >= TRIGGER_DISTANCE_PX,
    isRefreshing,
  };
}

// Coalesces the several events iOS fires on one resume into one callback.
const SYNC_BURST_MS = 50;

// iOS does not fire visibilitychange on every wake path, so also listen to
// pageshow (bfcache), focus (app switcher) and online (wake with no network).
export function useSyncOnVisible(callback: () => void): void {
  const cbRef = useRef(callback);
  useEffect(() => {
    cbRef.current = callback;
  }, [callback]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    let burstTimer: ReturnType<typeof setTimeout> | undefined;

    const fire = () => {
      if (burstTimer) return;
      burstTimer = setTimeout(() => {
        burstTimer = undefined;
        cbRef.current();
      }, SYNC_BURST_MS);
    };
    const onVisible = () => {
      if (document.visibilityState === "visible") fire();
    };

    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("pageshow", fire);
    window.addEventListener("focus", fire);
    window.addEventListener("online", fire);
    return () => {
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("pageshow", fire);
      window.removeEventListener("focus", fire);
      window.removeEventListener("online", fire);
      if (burstTimer) clearTimeout(burstTimer);
    };
  }, []);
}

export function useStableCallback<TArgs extends unknown[], TRet>(
  fn: (...args: TArgs) => TRet,
): (...args: TArgs) => TRet {
  const ref = useRef(fn);
  useEffect(() => {
    ref.current = fn;
  }, [fn]);
  return useCallback((...args: TArgs) => ref.current(...args), []);
}
