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

// The transport failed (fetch TypeError); the server may still have processed
// the request. Distinct from ApiError(0), our own timeout.
export class NetworkError extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = "NetworkError";
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  query?: Record<string, string | number | undefined>;
  signal?: AbortSignal;
  // null disables the timeout.
  timeoutMs?: number | null;
};

const LOGIN_PATH = "/login";

// iOS can leave a fetch hung forever (PWA backgrounded mid-flight, network
// handoff); without a cap the mutation never settles.
const DEFAULT_TIMEOUT_MS = 15_000;

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
        // Status 0 means timeout: no HTTP response was received.
        throw new ApiError(0, "request timed out", null);
      }
      // A caller abort is a DOMException, not a TypeError, and must pass
      // through so React Query treats it as a cancellation.
      if (err instanceof TypeError) {
        throw new NetworkError("could not reach the server", { cause: err });
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
      // statusText is always "" over HTTP/2.
      const message =
        (parsed && typeof parsed === "object" && "error" in parsed
          ? String((parsed as { error: unknown }).error)
          : null) ?? (res.statusText || `HTTP ${res.status}`);
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
