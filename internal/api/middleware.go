package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/storage"
)

// authMiddleware validates the session cookie, attaches the authenticated
// user to the request context, and auto-renews sessions in the second half
// of their lifetime. Only a definitively missing or expired session earns a
// 401 (and clears the cookie); a storage failure is a 500 that leaves the
// client's still-valid cookie alone.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		info, err := s.db.ValidateSessionWithInfo(cookie.Value)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrSessionExpired) {
				auth.ClearSessionCookie(w, s.secureCookie)
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			// Transient failures (e.g. SQLITE_BUSY under concurrent
			// requests) must never destroy a valid session.
			log.Printf("api: validate session: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		now := time.Now()
		if info.ExpiresAt.Sub(now) < auth.SessionDuration/2 && s.claimRenewal(cookie.Value, now) {
			if rerr := s.db.RenewSession(cookie.Value, now.Add(auth.SessionDuration)); rerr != nil {
				log.Printf("api: renew session: %v", rerr)
			}
		}

		// Re-send the cookie on every authenticated response so its
		// client-side expiry can never drift behind the server-side
		// session (a lost renewal response used to cost 15 days).
		auth.SetSessionCookie(w, cookie.Value, s.secureCookie)

		ctx := context.WithValue(r.Context(), auth.UserContextKey, info.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// claimRenewal reports whether this request should perform the renewal
// write for the token. The PWA opens with several concurrent requests that
// all cross the renewal threshold together; without this guard each of
// them issues the same UPDATE and they contend for SQLite's write lock.
func (s *Server) claimRenewal(token string, now time.Time) bool {
	s.renewMu.Lock()
	defer s.renewMu.Unlock()
	if last, ok := s.lastRenewal[token]; ok && now.Sub(last) < time.Minute {
		return false
	}
	if len(s.lastRenewal) > 1024 {
		for k, v := range s.lastRenewal {
			if now.Sub(v) >= time.Minute {
				delete(s.lastRenewal, k)
			}
		}
	}
	s.lastRenewal[token] = now
	return true
}
