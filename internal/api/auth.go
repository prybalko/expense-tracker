package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"expense-tracker/internal/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := s.db.GetUserByUsername(username)
	if err != nil || !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.GenerateSessionToken()
	if err != nil {
		log.Printf("api: generate session token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	expiresAt := time.Now().Add(auth.SessionDuration)
	if err := s.db.CreateSession(token, user.ID, expiresAt); err != nil {
		log.Printf("api: create session: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	auth.SetSessionCookie(w, token, s.secureCookie)
	writeJSON(w, http.StatusOK, userResponse{ID: user.ID, Username: user.Username})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie.Value != "" {
		if derr := s.db.DeleteSession(cookie.Value); derr != nil {
			log.Printf("api: delete session: %v", derr)
		}
	}
	auth.ClearSessionCookie(w, s.secureCookie)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, userResponse{ID: user.ID, Username: user.Username})
}
