import { ApiError } from "./client";

// Human copy for user-visible network failures. Keeps the banner wording in
// one place so write mutations, the cold-start list fetch, and the Feed-
// mount diff all speak with the same voice. The banner auto-dismisses
// after a few seconds, so the copy has to be short and self-contained.

// offlineCopy is the shared "your device isn't on a network" message.
// navigator.onLine is not a perfect signal — the OS can lie in weird ways
// (captive portals, iOS background fetch) — but when it does say false it's
// almost always right, so we prefer this framing over a generic "failed"
// line.
const OFFLINE_MESSAGE =
  "You're offline. Try again when you're back on a connection.";

function isOffline(): boolean {
  return typeof navigator !== "undefined" && navigator.onLine === false;
}

// messageForWriteError maps a create / update / delete failure to the
// banner copy. ApiError.status 0 is our convention for "timeout, no HTTP
// response" (see api/client.ts); 409 is the duplicate-row collision the
// offline-replay path relies on. Any other server-provided error message
// passes through so the backend's own phrasing (e.g. "amount must be
// positive") can reach the user unmangled.
export function messageForWriteError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 0) {
      if (isOffline()) return OFFLINE_MESSAGE;
      return "The request took too long. Check your connection and try again.";
    }
    if (err.status === 409) {
      return "An expense with the same date, amount, and description already exists.";
    }
    if (err.message) return err.message;
  }
  if (isOffline()) return OFFLINE_MESSAGE;
  return "Couldn't save. Please try again.";
}

// messageForReadError covers the cold-start list fetch and the Feed-mount
// diff. These are fire-and-forget from the user's perspective — they'll
// retry by navigating back or pulling the app forward — so the copy
// emphasises the fact that the data they're seeing may be slightly stale
// rather than asking them to do anything.
export function messageForReadError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 0) {
      if (isOffline()) return OFFLINE_MESSAGE;
      return "Couldn't refresh. Check your connection.";
    }
    if (err.message) return `Couldn't refresh: ${err.message}`;
  }
  if (isOffline()) return OFFLINE_MESSAGE;
  return "Couldn't refresh. Please try again soon.";
}
