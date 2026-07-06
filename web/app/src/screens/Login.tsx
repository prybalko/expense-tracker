import { useState } from "react";
import type { FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { theme, FONT } from "../theme";
import { SectionLabel } from "../components/SectionLabel";
import { login } from "../api/auth";
import { ApiError } from "../api/client";

export function Login() {
  const t = theme;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const disabled = submitting || !username.trim() || !password;

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (submitting) return;
    setError(null);
    setSubmitting(true);
    try {
      await login(username.trim(), password);
      queryClient.clear();
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
        height: "100dvh",
        overflow: "auto",
        WebkitOverflowScrolling: "touch",
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
          <SectionLabel style={{ marginBottom: 6 }}>Username</SectionLabel>
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
              fontSize: 16,
              fontFamily: FONT,
              outline: "none",
              boxSizing: "border-box",
              marginBottom: 14,
            }}
          />
          <SectionLabel style={{ marginBottom: 6 }}>Password</SectionLabel>
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
              fontSize: 16,
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
            disabled={disabled}
            data-testid="login-submit"
            style={{
              marginTop: 18,
              width: "100%",
              padding: "14px",
              borderRadius: 999,
              background: disabled ? t.keyDisabled : t.accent,
              color: t.accentText,
              border: "none",
              fontSize: 14,
              fontWeight: 600,
              cursor: disabled ? "default" : "pointer",
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
