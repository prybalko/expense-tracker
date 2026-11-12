import { useEffect, useRef } from "react";
import { useAllExpenses, useSyncExpenses } from "../hooks/useExpenses";
import { useSyncOnVisible, useStableCallback } from "../hooks/usePullToRefresh";
import { useErrorBanner } from "../hooks/useErrorBanner";
import { messageForReadError } from "../api/errors";

// Mount once above <Routes>. Sync on resume events only, never on mount or
// navigation, or every tab switch fires a sync.
export function ResumeSync(): null {
  const sync = useSyncExpenses();
  const expenses = useAllExpenses();
  const { showError } = useErrorBanner();

  const syncIfIdle = useStableCallback(() => {
    if (sync.isPending) return;
    sync.mutate(undefined, {
      onError: (err) => showError(messageForReadError(err)),
    });
  });

  useSyncOnVisible(syncIfIdle);

  // The service worker can serve a stale cached list on cold launch, so catch
  // up once after it loads.
  const caughtUp = useRef(false);
  useEffect(() => {
    if (caughtUp.current || !expenses.isSuccess) return;
    caughtUp.current = true;
    syncIfIdle();
  }, [expenses.isSuccess, syncIfIdle]);

  return null;
}
