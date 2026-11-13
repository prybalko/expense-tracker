import { ApiError, NetworkError } from "./client";

// navigator.onLine can report true when offline, but false is reliable.
const OFFLINE_MESSAGE =
  "You're offline. Try again when you're back on a connection.";

function isOffline(): boolean {
  return typeof navigator !== "undefined" && navigator.onLine === false;
}

// NetworkError while online: the app loaded, but the server, a proxy or the
// connection to it failed.
const UNREACHABLE_READ = "Can't reach the server. Data may be out of date.";
// The connection can drop after the server saved the row, so don't claim it wasn't.
const UNREACHABLE_WRITE = "Can't reach the server. Check the list before you try again.";

// ApiError status 0 is a timeout (see client.ts).
export function messageForWriteError(err: unknown): string {
  if (err instanceof NetworkError) {
    return isOffline() ? OFFLINE_MESSAGE : UNREACHABLE_WRITE;
  }
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

export function messageForReadError(err: unknown): string {
  if (err instanceof NetworkError) {
    return isOffline() ? OFFLINE_MESSAGE : UNREACHABLE_READ;
  }
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
