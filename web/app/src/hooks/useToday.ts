import { useEffect, useState } from "react";

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

// iOS resumes a backgrounded PWA without remounting React, so a mount-time
// date goes stale; recompute on every resume path and at local midnight.
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

    // iOS throttles background timers, so this only covers foreground sessions.
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
