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
};

const LOGIN_PATH = "/login";

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

export async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", body, query, signal } = options;
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  let payload: BodyInit | undefined;
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }

  const res = await fetch(buildUrl(path, query), {
    method,
    headers,
    body: payload,
    credentials: "include",
    signal,
  });

  if (res.status === 401) {
    redirectToLogin();
    throw new ApiError(401, "unauthorized", null);
  }

  if (res.status === 204) {
    return undefined as T;
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

  return parsed as T;
}
