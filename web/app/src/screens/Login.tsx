import { useState } from "react";
import type { FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { theme, FONT } from "../theme";
import { login } from "../api/auth";
import { ApiError } from "../api/client";
import { clearQueue } from "../offline/queue";
import { setInMemoryQueueBlocked, syncQueue } from "../offline/sync";

export function Login() {
  const t = theme;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (submitting) return;
    setError(null);
    setSubmitting(true);
    try {
      const user = await login(username.trim(), password);
      // Queued writes belong to whichever user was signed in when they were
      // enqueued. Preserve them only when we can *prove* the prior owner is
      // the same user (same-user re-auth after a 401 — the queue must replay).
      // Anything else has unknown ownership and must be dropped:
      //   - no `lastUser` key yet (first login after upgrade — queue may
      //     contain another user's writes from before this code shipped)
      //   - `lastUser` belongs to a different user
      //   - `queueBlocked` is set — a previous clear failed, ownership is
      //     undetermined and must be re-established by a successful clear
      //   - localStorage unavailable (Safari private mode etc.) — we can't
      //     prove ownership, so we can't safely replay
      // localStorage access is isolated from the outer catch: throwing here
      // after the session cookie is already set would surface as
      // "login failed" and desync UI from auth state.
      let sameUser = false;
      try {
        const blocked = localStorage.getItem("queueBlocked") === "1";
        sameUser =
          !blocked && localStorage.getItem("lastUser") === user.username;
      } catch {
        // Web Storage unavailable — treat as unknown ownership.
      }
      // queueOwned tracks whether we can safely replay the queue under this
      // user. Starts true for same-user re-auth; for unknown ownership we
      // must successfully clear the queue before flipping it back to true.
      let queueOwned = sameUser;
      if (!sameUser) {
        try {
          await clearQueue();
          queueOwned = true;
          setInMemoryQueueBlocked(false);
          try {
            localStorage.removeItem("queueBlocked");
          } catch {
            // Best-effort.
          }
        } catch {
          // Clear failed — ownership remains unknown. Persist that so the
          // background sync paths in setupOnlineSync (mount-time drain,
          // `online` listener) refuse to drain stale entries under this
          // session. A later login retries the clear. The in-memory mirror
          // covers the case where localStorage.setItem also fails — without
          // it, syncQueue would fall through to its `false` default and
          // replay stale entries on the next online tick.
          setInMemoryQueueBlocked(true);
          try {
            localStorage.setItem("queueBlocked", "1");
          } catch {
            // Web Storage unavailable — best-effort only.
          }
        }
      }
      if (queueOwned) {
        try {
          localStorage.setItem("lastUser", user.username);
        } catch {
          // Web Storage unavailable — best-effort only.
        }
      }
      queryClient.clear();
      // After a same-user re-auth (e.g. 401 → /login → sign in again), the
      // app is already mounted and connectivity hasn't changed, so neither
      // App.tsx's mount-time drain nor the `online` listener will fire. Kick
      // off a sync now so any writes queued under the expired session land —
      // but only when we know the queue belongs to this user.
      if (queueOwned) {
        void syncQueue(queryClient);
      }
      navigate("/", { replace: true });
    } catch (err) {
      const message =
        err instanceof ApiError
          ? err.status === 401
            ? "Invalid username or password"
            : err.message
          : "Could not sign in";
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      style={{
        minHeight: "100vh",
        background: t.bg,
        color: t.ink,
        fontFamily: FONT,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        padding: "24px",
        boxSizing: "border-box",
      }}
    >
      <div
        style={{
          maxWidth: 380,
          width: "100%",
          margin: "0 auto",
          background: t.card,
          borderRadius: 28,
          padding: "28px 24px",
        }}
      >
        <h1
          style={{
            margin: 0,
            fontSize: 28,
            fontWeight: 600,
            letterSpacing: "-0.02em",
          }}
        >
          Expenses
        </h1>
        <p style={{ margin: "6px 0 22px", fontSize: 13, color: t.ink2 }}>
          Sign in to continue
        </p>
        <form onSubmit={onSubmit} data-testid="login-form">
          <label
            style={{
              display: "block",
              fontSize: 11,
              fontWeight: 500,
              letterSpacing: "0.04em",
              textTransform: "uppercase",
              color: t.ink2,
              marginBottom: 6,
            }}
          >
            Username
          </label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            autoCapitalize="none"
            autoCorrect="off"
            required
            data-testid="login-username"
            style={{
              width: "100%",
              padding: "14px",
              borderRadius: 16,
              border: "none",
              background: t.cardAlt,
              color: t.ink,
              fontSize: 15,
              fontFamily: FONT,
              outline: "none",
              boxSizing: "border-box",
              marginBottom: 14,
            }}
          />
          <label
            style={{
              display: "block",
              fontSize: 11,
              fontWeight: 500,
              letterSpacing: "0.04em",
              textTransform: "uppercase",
              color: t.ink2,
              marginBottom: 6,
            }}
          >
            Password
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
            data-testid="login-password"
            style={{
              width: "100%",
              padding: "14px",
              borderRadius: 16,
              border: "none",
              background: t.cardAlt,
              color: t.ink,
              fontSize: 15,
              fontFamily: FONT,
              outline: "none",
              boxSizing: "border-box",
            }}
          />
          {error ? (
            <div
              style={{
                marginTop: 14,
                fontSize: 13,
                color: t.red,
              }}
            >
              {error}
            </div>
          ) : null}
          <button
            type="submit"
            disabled={submitting || !username.trim() || !password}
            data-testid="login-submit"
            style={{
              marginTop: 18,
              width: "100%",
              padding: "14px",
              borderRadius: 999,
              background:
                submitting || !username.trim() || !password
                  ? t.keyDisabled
                  : t.accent,
              color: t.accentText,
              border: "none",
              fontSize: 14,
              fontWeight: 600,
              cursor:
                submitting || !username.trim() || !password
                  ? "default"
                  : "pointer",
              fontFamily: FONT,
            }}
          >
            {submitting ? "Signing in..." : "Sign in"}
          </button>
        </form>
      </div>
    </div>
  );
}
