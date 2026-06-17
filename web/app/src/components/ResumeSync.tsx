import { useEffect, useRef } from "react";
import { useAllExpenses, useSyncExpenses } from "../hooks/useExpenses";
import { useSyncOnVisible, useStableCallback } from "../hooks/usePullToRefresh";
import { useErrorBanner } from "../hooks/useErrorBanner";
import { messageForReadError } from "../api/errors";

// ResumeSync is the app-level, route-independent refresh trigger. It renders
// nothing and is mounted once above <Routes>, so a background→foreground
// resume fires the delta sync no matter which screen the user landed on — iOS
// restores the last route on resume, and previously only the Feed wired the
// visibility sync, so resuming on Insights/CategoryDetails/EntryForm refreshed
// nothing until the user navigated to Feed and pulled. Pull-to-refresh stays
// on the Feed as the manual touch affordance; this owns the automatic path.
//
// IMPORTANT: the sync must be driven by resume *events* (via useSyncOnVisible),
// never by mount or navigation — firing on mount would reintroduce the
// per-mount / tab-toggle sync storm the design deliberately removed.
export function ResumeSync(): null {
  const sync = useSyncExpenses();
  const expenses = useAllExpenses();
  const { showError } = useErrorBanner();

  // Guard against overlapping syncs: a resume can emit several events and the
  // user may wake the app repeatedly. mutate() would otherwise fan out into
  // parallel /changes calls. The delta merge is idempotent and lastSyncAt only
  // moves forward, so skipping while one is in flight is always safe.
  const syncIfIdle = useStableCallback(() => {
    if (sync.isPending) return;
    sync.mutate(undefined, {
      onError: (err) => showError(messageForReadError(err)),
    });
  });

  useSyncOnVisible(syncIfIdle);

  // One-shot catch-up after the initial full-list load. On a cold relaunch the
  // service worker can serve a stale cached list (StaleWhileRevalidate) and
  // seed lastSyncAt to an old server time; a single delta then pulls in
  // everything changed since, so the first screen isn't a launch behind. This
  // component mounts once for the session, so the ref makes it fire at most
  // once — it is not a per-mount sync.
  const caughtUp = useRef(false);
  useEffect(() => {
    if (caughtUp.current || !expenses.isSuccess) return;
    caughtUp.current = true;
    syncIfIdle();
  }, [expenses.isSuccess, syncIfIdle]);

  return null;
}
