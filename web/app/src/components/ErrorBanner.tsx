import { theme, FONT } from "../theme";
import { useErrorBanner } from "../hooks/useErrorBanner";

// Relies on #root having no transform; one would re-anchor position: fixed.
export function ErrorBanner() {
  const { message, clear } = useErrorBanner();
  if (!message) return null;
  return (
    <div
      data-testid="error-banner"
      role="alert"
      aria-live="polite"
      onClick={clear}
      style={{
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
        zIndex: 1000,
        padding:
          "calc(10px + env(safe-area-inset-top)) 16px 10px",
        display: "flex",
        justifyContent: "center",
        pointerEvents: "none",
      }}
    >
      <div
        style={{
          pointerEvents: "auto",
          maxWidth: 448,
          width: "100%",
          padding: "12px 16px",
          borderRadius: 14,
          background: theme.red,
          color: theme.accentText,
          fontFamily: FONT,
          fontSize: 13,
          fontWeight: 500,
          textAlign: "center",
          boxShadow: "0 8px 24px rgba(26,23,20,0.18)",
          cursor: "pointer",
        }}
      >
        {message}
      </div>
    </div>
  );
}
