// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest";
import { ApiError, NetworkError } from "./client";
import { messageForReadError, messageForWriteError } from "./errors";

function setOnLine(value: boolean): void {
  Object.defineProperty(navigator, "onLine", { value, configurable: true });
}

afterEach(() => setOnLine(true));

describe("messageForReadError", () => {
  it("says the server is unreachable when the device is online", () => {
    const msg = messageForReadError(new NetworkError("could not reach"));
    expect(msg).toContain("Can't reach the server");
  });

  it("blames the device when the device says it is offline", () => {
    setOnLine(false);
    expect(messageForReadError(new NetworkError("could not reach"))).toContain(
      "offline",
    );
  });

  it("surfaces an HTTP status rather than a bare retry prompt", () => {
    const msg = messageForReadError(new ApiError(502, "HTTP 502", null));
    expect(msg).toBe("Couldn't refresh: HTTP 502");
  });

  it("still distinguishes a timeout from an unreachable server", () => {
    expect(messageForReadError(new ApiError(0, "request timed out", null))).toBe(
      "Couldn't refresh. Check your connection.",
    );
  });
});

describe("messageForWriteError", () => {
  it("does not claim the write failed when the server is unreachable", () => {
    const msg = messageForWriteError(new NetworkError("could not reach"));
    expect(msg).toContain("Can't reach the server");
    expect(msg).toContain("Check the list");
    expect(msg).not.toMatch(/nothing was saved/i);
  });

  it("blames the device when the device says it is offline", () => {
    setOnLine(false);
    expect(messageForWriteError(new NetworkError("could not reach"))).toContain(
      "offline",
    );
  });

  it("keeps the duplicate-row copy, which means the server did answer", () => {
    setOnLine(false);
    expect(messageForWriteError(new ApiError(409, "conflict", null))).toContain(
      "already exists",
    );
  });

  it("passes a validation message through unmangled", () => {
    expect(
      messageForWriteError(new ApiError(400, "amount must be positive", null)),
    ).toBe("amount must be positive");
  });
});
