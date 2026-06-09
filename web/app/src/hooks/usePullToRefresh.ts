import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

// usePullToRefresh wires a touch-driven pull gesture on top of an existing
// scrollable element. The Feed previously auto-fired the delta-sync diff on
// every component mount, which meant every Feed↔Insights tab toggle paid a
// /api/expenses/changes round-trip even when the diff came back empty. The
// gesture moves that decision back to the user: pull down at the top of the
// list, the indicator fills as you drag, release past the threshold to fire
// the refresh. Same code path as the old auto-sync (useSyncExpenses), just
// triggered explicitly.
//
// Touch-only on purpose. The pointer events API would let us swipe with a
// mouse on desktop, but the breakpoint above 600px in index.css turns the
// app into a centered card and adding a fake "pull down with the mouse"
// affordance there only confuses the model. Desktop users have Cmd-R / F5.

// TRIGGER_DISTANCE_PX is the finger-travel distance past which a release
// commits to a refresh. Calibrated for iOS Safari PWA where the visible pull
// is RESISTANCE * travel — at the default 0.5 you have to drag your finger
// about 160 px (a comfortable half-screen on a 6.7" phone) before the
// indicator hits commit position. Lower values trigger refreshes on the
// flick that ends an upward scroll-to-top fling; higher values feel sluggish.
const TRIGGER_DISTANCE_PX = 80;

// MAX_PULL_DISTANCE_PX caps the visible pull regardless of how far the user
// drags. Without this, an aggressive yank produces a 400 px gap above the
// hero that snaps closed on release — visually jarring and out of step with
// the iOS-native pattern users have a model for.
const MAX_PULL_DISTANCE_PX = 120;

// RESISTANCE expresses pulled-distance ÷ finger-travel. iOS-native pull-to-
// refresh uses ~0.5 (you drag 200 px to see 100 px of pull); matching that
// makes the gesture feel familiar and prevents accidental triggers during a
// fast upward scroll.
const RESISTANCE = 0.5;

export type PullToRefreshState = {
  // pullDistance is the rendered displacement in CSS pixels (after RESISTANCE).
  // 0 when the user isn't actively pulling. Components translate the indicator
  // and/or the list content by this many pixels.
  pullDistance: number;
  // committed flips true the moment pullDistance crosses TRIGGER_DISTANCE_PX —
  // before release. Lets the UI swap the indicator into a "release to refresh"
  // affordance the same way iOS native does.
  committed: boolean;
  // isRefreshing is true from the moment the refresh promise starts until it
  // settles. Pull gestures are ignored while it's true so we don't queue
  // multiple parallel /changes calls on a fast double-pull.
  isRefreshing: boolean;
};

export function usePullToRefresh(
  scrollContainerRef: RefObject<HTMLElement | null>,
  onRefresh: () => Promise<void> | void,
): PullToRefreshState {
  const [pullDistance, setPullDistance] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // The touch handlers need to read live values without forcing an effect
  // re-run on every pixel of pull. Refs decouple "what the handlers see" from
  // "what triggers React renders".
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
      // Only initiate a pull at the very top of the scroll content. If the
      // user is mid-scroll, this is a normal flick and must not steal it.
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
        // Upward swipe (user pushed past the top, then scrolled back). Drop
        // the pull state so a subsequent downward pull starts fresh.
        if (pullDistanceRef.current !== 0) setPullDistance(0);
        return;
      }
      // Suppress the browser's elastic over-scroll while we're showing our
      // own indicator. The {passive: false} listener registration below is
      // what makes this preventDefault take effect on iOS Safari.
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
      // Hand off to the async refresh path; keep the indicator pinned at the
      // commit distance until the refresh settles so users see the spinner
      // instead of the gap snapping shut as soon as they let go.
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

    // touchmove must be non-passive so e.preventDefault() suppresses the
    // overscroll rubber-band. The others stay passive — they don't cancel
    // default behaviour and passive listeners are cheaper for the scheduler.
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

// useSyncOnVisible runs `callback` whenever the document becomes visible
// again. This covers the iOS PWA "resume from background" case the pull
// gesture can't reach: the user deletes an item on phone A, switches back
// to the PWA on phone B that was sitting idle, and the diff fires
// automatically as the page returns to the foreground. Same code path as
// the gesture — wired to useSyncExpenses at the call site.
//
// The handler is stored in a ref so the listener never needs to be torn
// down and re-installed on every parent render.
export function useSyncOnVisible(callback: () => void): void {
  const cbRef = useRef(callback);
  useEffect(() => {
    cbRef.current = callback;
  }, [callback]);

  useEffect(() => {
    if (typeof document === "undefined") return;
    const handler = () => {
      if (document.visibilityState === "visible") {
        cbRef.current();
      }
    };
    document.addEventListener("visibilitychange", handler);
    return () => document.removeEventListener("visibilitychange", handler);
  }, []);
}

// useStableCallback is a small helper for call sites that want to pass a
// fresh-every-render callback to a hook that registers DOM listeners. It
// returns a stable function whose body always invokes the latest captured
// `fn`. Useful for the Feed: the refresh handler closes over `sync.mutate`
// which is itself stable, but the showError-binding wrapper around it isn't,
// and we don't want to thrash listener registrations.
export function useStableCallback<TArgs extends unknown[], TRet>(
  fn: (...args: TArgs) => TRet,
): (...args: TArgs) => TRet {
  const ref = useRef(fn);
  useEffect(() => {
    ref.current = fn;
  }, [fn]);
  return useCallback((...args: TArgs) => ref.current(...args), []);
}
