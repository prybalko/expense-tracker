// Hook + context + provider for the global error banner are colocated
// because they're only ever used together; the eslint rule would rather we
// had three separate modules so Fast Refresh can pinpoint a single
// component per file, but in this codebase "one error surface across the
// whole app" is a self-contained unit and splitting it would just churn
// imports without any observable benefit.
/* eslint-disable react-refresh/only-export-components */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

// Global error surface for the app. The delta-sync redesign collapses every
// user-visible network failure (list fetch, Feed diff, create / update /
// delete) onto one place: a top-of-screen banner driven by this hook. No
// Retry button on purpose — the sync is triggered on every Feed mount and
// every write runs wait-for-server, so the user can always re-try by
// repeating the gesture that failed.

const AUTO_DISMISS_MS = 5000;

type ErrorBannerContextValue = {
  message: string | null;
  showError: (msg: string) => void;
  clear: () => void;
};

const ErrorBannerContext = createContext<ErrorBannerContextValue | null>(null);

type ErrorBannerProviderProps = {
  children: ReactNode;
};

export function ErrorBannerProvider({ children }: ErrorBannerProviderProps) {
  const [message, setMessage] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const clear = useCallback(() => {
    clearTimer();
    setMessage(null);
  }, [clearTimer]);

  const showError = useCallback(
    (msg: string) => {
      // Latest error wins. A second failure while the first is still on
      // screen replaces the text and restarts the auto-dismiss timer so
      // the user gets the full dismiss window for the most recent error.
      clearTimer();
      setMessage(msg);
      timerRef.current = setTimeout(() => {
        timerRef.current = null;
        setMessage(null);
      }, AUTO_DISMISS_MS);
    },
    [clearTimer],
  );

  useEffect(() => {
    return () => {
      clearTimer();
    };
  }, [clearTimer]);

  const value: ErrorBannerContextValue = { message, showError, clear };
  return (
    <ErrorBannerContext.Provider value={value}>
      {children}
    </ErrorBannerContext.Provider>
  );
}

export function useErrorBanner(): ErrorBannerContextValue {
  const ctx = useContext(ErrorBannerContext);
  if (!ctx) {
    throw new Error(
      "useErrorBanner must be used within an ErrorBannerProvider",
    );
  }
  return ctx;
}
