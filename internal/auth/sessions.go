package auth

import (
	"context"
	"net/http"
	"time"

	"expense-tracker/internal/models"
)

const (
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session"
	// SessionDuration is how long sessions last.
	SessionDuration = 30 * 24 * time.Hour
)

type contextKey string

// UserContextKey is the context key that holds the authenticated *models.User.
const UserContextKey contextKey = "user"

// UserFromContext returns the authenticated user attached by AuthMiddleware.
func UserFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(UserContextKey).(*models.User)
	return u, ok
}

// SetSessionCookie writes the session cookie with standard attributes.
func SetSessionCookie(w http.ResponseWriter, token string, secure bool) {
	//nolint:gosec // Secure flag is configurable via secure param
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the session cookie on the client.
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	//nolint:gosec // Secure flag is configurable via secure param
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
