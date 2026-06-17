import { useEffect, useState } from "react";

// startOfDay strips the time-of-day so "today" is a pure calendar date. All
// day-label / current-period logic only cares about the y/m/d, and pinning to
// midnight keeps the returned reference stable across intra-day refreshes.
function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

// useToday returns the current calendar day and — crucially for the iOS PWA
// case — recomputes it when the app comes back to the foreground or the local
// day rolls over. A mount-time `useMemo(() => new Date(), [])` freezes at the
// day the app was first loaded: iOS resumes a backgrounded PWA by thawing the
// same JS heap WITHOUT remounting React, so the frozen value lingers and
// yesterday's expenses keep the "Today" header (and any "current period" math
// stays a day behind). This hook covers every resume path WebKit can take —
// visibilitychange / pageshow / focus — plus a self-rescheduling timer that
// fires just after the next local midnight for a foregrounded session.
//
// The setter is guarded by a date-only comparison: refresh events fire often,
// but the state (and therefore the returned reference) only changes when the
// calendar day actually changes, so consumers don't re-render or re-run
// `today`-keyed memos on every focus.
export function useToday(): Date {
  const [today, setToday] = useState<Date>(() => startOfDay(new Date()));

  useEffect(() => {
    if (typeof document === "undefined") return;

    let timer: ReturnType<typeof setTimeout> | undefined;

    const refresh = () => {
      setToday((prev) => {
        const next = startOfDay(new Date());
        return sameDay(prev, next) ? prev : next;
      });
    };

    // Fire just after the next local midnight, then reschedule. iOS throttles
    // timers while backgrounded, so this only reliably covers a session left
    // open in the foreground; the visibility/focus listeners handle resume.
    const scheduleMidnight = () => {
      const now = new Date();
      const nextMidnight = new Date(
        now.getFullYear(),
        now.getMonth(),
        now.getDate() + 1,
        0,
        0,
        1,
      );
      timer = setTimeout(() => {
        refresh();
        scheduleMidnight();
      }, nextMidnight.getTime() - now.getTime());
    };

    const onVisible = () => {
      if (document.visibilityState === "visible") refresh();
    };

    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("pageshow", refresh);
    window.addEventListener("focus", refresh);
    scheduleMidnight();

    return () => {
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("pageshow", refresh);
      window.removeEventListener("focus", refresh);
      if (timer) clearTimeout(timer);
    };
  }, []);

  return today;
}
