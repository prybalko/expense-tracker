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
