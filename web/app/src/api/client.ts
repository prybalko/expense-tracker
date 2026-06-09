export class ApiError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  query?: Record<string, string | number | undefined>;
  signal?: AbortSignal;
  // Per-request timeout in milliseconds. Defaults to DEFAULT_TIMEOUT_MS;
  // pass null to disable (e.g. for an intentionally long-poll endpoint).
  timeoutMs?: number | null;
};

const LOGIN_PATH = "/login";

// Cap every request so a hung fetch (iOS PWA backgrounded mid-flight,
// Wi-Fi ↔ cellular handoff, TLS stall, server stuck) can't deadlock the UI.
// Without this the React Query mutation never settles and the form is stuck
// on "Adding…" forever.
const DEFAULT_TIMEOUT_MS = 15_000;

// SERVER_TIME_HEADER mirrors the Go handler's X-Server-Time header so the
// client can advance its lastSyncAt marker off every mutation response. The
// changes / full-list endpoints also echo serverTime in the JSON body, but
// DELETE returns 204 No Content and the other write bodies hold the row —
// the header is the single place the marker can advance without a
// follow-up round trip.
export const SERVER_TIME_HEADER = "X-Server-Time";

export type RequestMeta = {
  serverTime: string | null;
};

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  if (!query) return path;
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v === undefined || v === null) continue;
    params.set(k, String(v));
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

function redirectToLogin(): void {
  if (typeof window === "undefined") return;
  if (window.location.pathname === LOGIN_PATH) return;
  window.location.assign(LOGIN_PATH);
}

// requestWithMeta is the primitive; `request` is the convenience wrapper
// for call sites that only care about the parsed body. Callers that need to
// advance lastSyncAt off the X-Server-Time header (create / update / delete
// mutations) use this variant directly.
export async function requestWithMeta<T>(
  path: string,
  options: RequestOptions = {},
): Promise<{ data: T; meta: RequestMeta }> {
  const {
    method = "GET",
    body,
    query,
    signal: userSignal,
    timeoutMs = DEFAULT_TIMEOUT_MS,
  } = options;
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  let payload: BodyInit | undefined;
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }

  // Compose three abort sources into one signal handed to fetch:
  //   1. our internal timeout (the main reason this exists),
  //   2. the caller's optional signal,
  //   3. (implicit) controller.abort from the cleanup paths.
  // `timedOut` distinguishes our timer firing from a caller-initiated abort
  // so we can throw a clearly-typed error instead of a bare AbortError.
  const controller = new AbortController();
  let timedOut = false;
  const timer =
    timeoutMs !== null
      ? setTimeout(() => {
          timedOut = true;
          controller.abort();
        }, timeoutMs)
      : null;
  const onUserAbort = userSignal ? () => controller.abort() : null;
  if (userSignal && onUserAbort) {
    if (userSignal.aborted) {
      controller.abort();
    } else {
      userSignal.addEventListener("abort", onUserAbort);
    }
  }

  try {
    let res: Response;
    try {
      res = await fetch(buildUrl(path, query), {
        method,
        headers,
        body: payload,
        credentials: "include",
        signal: controller.signal,
      });
    } catch (err) {
      if (
        timedOut &&
        err instanceof DOMException &&
        err.name === "AbortError"
      ) {
        // Status 0 is the convention for "no HTTP response was received".
        // EntryForm's messageForWriteError special-cases this to show a
        // timeout-specific copy instead of the generic save-failed line.
        throw new ApiError(0, "request timed out", null);
      }
      throw err;
    }

    if (res.status === 401) {
      redirectToLogin();
      throw new ApiError(401, "unauthorized", null);
    }

    const meta: RequestMeta = {
      serverTime: res.headers.get(SERVER_TIME_HEADER),
    };

    if (res.status === 204) {
      return { data: undefined as T, meta };
    }

    const contentType = res.headers.get("content-type") ?? "";
    let parsed: unknown;
    if (contentType.includes("application/json")) {
      parsed = await res.json().catch(() => null);
    } else {
      const text = await res.text().catch(() => "");
      parsed = text || null;
    }

    if (!res.ok) {
      const message =
        (parsed && typeof parsed === "object" && "error" in parsed
          ? String((parsed as { error: unknown }).error)
          : null) ?? res.statusText;
      throw new ApiError(res.status, message, parsed);
    }

    return { data: parsed as T, meta };
  } finally {
    if (timer !== null) clearTimeout(timer);
    if (userSignal && onUserAbort) {
      userSignal.removeEventListener("abort", onUserAbort);
    }
  }
}

export async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { data } = await requestWithMeta<T>(path, options);
  return data;
}
