package api

import (
	"context"
	"net/http"
	"time"

	"expense-tracker/internal/auth"
)

// authMiddleware validates the session cookie, attaches the authenticated
// user to the request context, and auto-renews sessions in the second half
// of their lifetime. On failure it returns a JSON 401 response.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		info, err := s.db.ValidateSessionWithInfo(cookie.Value)
		if err != nil {
			auth.ClearSessionCookie(w, s.secureCookie)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		now := time.Now()
		if info.ExpiresAt.Sub(now) < auth.SessionDuration/2 {
			newExpires := now.Add(auth.SessionDuration)
			if rerr := s.db.RenewSession(cookie.Value, newExpires); rerr == nil {
				auth.SetSessionCookie(w, cookie.Value, s.secureCookie)
			}
		}

		ctx := context.WithValue(r.Context(), auth.UserContextKey, info.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
