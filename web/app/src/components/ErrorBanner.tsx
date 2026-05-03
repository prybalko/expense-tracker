import { theme, FONT } from "../theme";
import { useErrorBanner } from "../hooks/useErrorBanner";

// ErrorBanner renders the currently-active message from useErrorBanner as a
// top-fixed pill, stacked above every screen. It sits inside the 480px
// mobile-card clip (position: fixed relative to the nearest containing
// block is fine here because #root has no transform) so the banner lines up
// with the card on desktop and respects the iOS notch / dynamic island
// safe area on the phone. Tap to dismiss explicitly; the provider also
// auto-hides after ~5s.
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
