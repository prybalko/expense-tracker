package handlers

import (
	"context"
	"expense-tracker/internal/auth"
	"log"
	"net/http"
	"strings"
	"time"
)

// AuthMiddleware wraps handlers to require authentication.
// It also implements rolling sessions: if a session is past the halfway point
// of its lifetime, it automatically renews the session.
func (h *Handlers) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		sessionInfo, err := h.db.ValidateSessionWithInfo(cookie.Value)
		if err != nil {
			// Invalid or expired session, clear the cookie
			h.clearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Rolling session: renew if past halfway point
		// This keeps active users logged in while still expiring inactive sessions
		now := time.Now()
		timeUntilExpiry := sessionInfo.ExpiresAt.Sub(now)
		halfSessionDuration := SessionDuration / 2

		if timeUntilExpiry < halfSessionDuration {
			// Session is in the second half of its lifetime, renew it
			newExpiresAt := now.Add(SessionDuration)
			if err := h.db.RenewSession(cookie.Value, newExpiresAt); err == nil {
				// Update the cookie expiration too
				http.SetCookie(w, &http.Cookie{
					Name:     SessionCookieName,
					Value:    cookie.Value,
					Path:     "/",
					MaxAge:   int(SessionDuration.Seconds()),
					HttpOnly: true,
					Secure:   h.secureCookie,
					SameSite: http.SameSiteLaxMode,
				})
			}
			// If renewal fails, just continue with the current session
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, sessionInfo.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoginForm renders the login page.
func (h *Handlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to expenses
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		if _, err := h.db.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/expenses", http.StatusFound)
			return
		}
	}
	h.render(w, r, "login.html", LoginViewModel{})
}

// Login handles the login form submission.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render(w, r, "login.html", LoginViewModel{Error: "Invalid form submission"})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.render(w, r, "login.html", LoginViewModel{Error: "Username and password are required"})
		return
	}

	user, err := h.db.GetUserByUsername(username)
	if err != nil || !auth.CheckPassword(password, user.PasswordHash) {
		h.render(w, r, "login.html", LoginViewModel{Error: "Invalid username or password"})
		return
	}

	// Generate session token
	token, err := auth.GenerateSessionToken()
	if err != nil {
		log.Printf("Failed to generate session token: %v", err)
		h.render(w, r, "login.html", LoginViewModel{Error: "An error occurred. Please try again."})
		return
	}

	// Create session in database
	expiresAt := time.Now().Add(SessionDuration)
	if err := h.db.CreateSession(token, user.ID, expiresAt); err != nil {
		log.Printf("Failed to create session: %v", err)
		h.render(w, r, "login.html", LoginViewModel{Error: "An error occurred. Please try again."})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/expenses", http.StatusFound)
}

// Logout handles user logout.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.db.DeleteSession(cookie.Value); err != nil {
			log.Printf("Failed to delete session: %v", err)
		}
	}
	h.clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *Handlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}
