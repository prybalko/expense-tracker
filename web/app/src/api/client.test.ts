// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, NetworkError, request } from "./client";

function stubFetch(impl: (init: RequestInit) => Promise<Response>): void {
  vi.stubGlobal(
    "fetch",
    vi.fn((_url: string, init: RequestInit) => impl(init)),
  );
}

function hangUntilAborted(init: RequestInit): Promise<Response> {
  return new Promise((_resolve, reject) => {
    init.signal?.addEventListener("abort", () => {
      reject(new DOMException("aborted", "AbortError"));
    });
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("request", () => {
  it("maps a transport failure to NetworkError", async () => {
    stubFetch(() => Promise.reject(new TypeError("Load failed")));

    await expect(request("/api/expenses")).rejects.toBeInstanceOf(NetworkError);
  });

  it("keeps the underlying fetch failure as the cause", async () => {
    stubFetch(() => Promise.reject(new TypeError("Load failed")));

    const err = await request("/api/expenses").catch((e: unknown) => e);
    expect((err as NetworkError).cause).toBeInstanceOf(TypeError);
  });

  it("names the status when an error body carries no JSON error field", async () => {
    // Over HTTP/2 statusText is always "".
    stubFetch(() =>
      Promise.resolve(
        new Response("<html>502 Bad Gateway</html>", {
          status: 502,
          statusText: "",
          headers: { "content-type": "text/html" },
        }),
      ),
    );

    const err = await request("/api/expenses").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(502);
    expect((err as ApiError).message).toBe("HTTP 502");
  });

  it("prefers the server's own error copy over the status", async () => {
    stubFetch(() =>
      Promise.resolve(
        new Response(JSON.stringify({ error: "amount must be positive" }), {
          status: 400,
          headers: { "content-type": "application/json" },
        }),
      ),
    );

    const err = await request("/api/expenses", {
      method: "POST",
      body: {},
    }).catch((e: unknown) => e);
    expect((err as ApiError).message).toBe("amount must be positive");
  });

  it("reports a timeout as ApiError(0), not a transport failure", async () => {
    stubFetch(hangUntilAborted);

    const err = await request("/api/expenses", { timeoutMs: 5 }).catch(
      (e: unknown) => e,
    );
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(0);
    expect(err).not.toBeInstanceOf(NetworkError);
  });

  it("passes a caller-initiated abort through untouched", async () => {
    stubFetch(hangUntilAborted);

    const controller = new AbortController();
    const pending = request("/api/expenses", { signal: controller.signal });
    controller.abort();

    const err = await pending.catch((e: unknown) => e);
    expect(err).not.toBeInstanceOf(NetworkError);
    expect(err).not.toBeInstanceOf(ApiError);
    expect((err as DOMException).name).toBe("AbortError");
  });
});
