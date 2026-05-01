package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/internal/auth"
)

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantCookie bool
	}{
		{
			name:       "happy path",
			body:       map[string]string{"username": "alice", "password": "hunter2"},
			wantStatus: http.StatusOK,
			wantCookie: true,
		},
		{
			name:       "wrong password",
			body:       map[string]string{"username": "alice", "password": "nope"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown user",
			body:       map[string]string{"username": "bob", "password": "hunter2"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing fields",
			body:       map[string]string{"username": "alice"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "blank username",
			body:       map[string]string{"username": "   ", "password": "hunter2"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed json",
			body:       "{not json",
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			req := buildRequest(t, http.MethodPost, "/api/auth/login", tc.body)
			rec := httptest.NewRecorder()
			env.server.handleLogin(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status: got %d want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			gotCookie := false
			for _, c := range rec.Result().Cookies() {
				if c.Name == auth.SessionCookieName && c.Value != "" {
					gotCookie = true
				}
			}
			if gotCookie != tc.wantCookie {
				t.Fatalf("cookie set: got %v want %v", gotCookie, tc.wantCookie)
			}
			if tc.wantStatus == http.StatusOK {
				var resp userResponse
				decodeBody(t, rec, &resp)
				if resp.Username != "alice" {
					t.Fatalf("username: got %q want alice", resp.Username)
				}
				if resp.ID != env.user.ID {
					t.Fatalf("id: got %d want %d", resp.ID, env.user.ID)
				}
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	env := newTestEnv(t)
	req := env.authedRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	env.server.handleLogout(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	// Session should be deleted.
	if _, err := env.db.ValidateSession(env.token); err == nil {
		t.Fatalf("expected session to be deleted, but it still validates")
	}
	// Cookie should be cleared (MaxAge<0).
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatalf("session cookie was not cleared")
	}
}

func TestHandleMe(t *testing.T) {
	env := newTestEnv(t)
	req := env.withUser(buildRequest(t, http.MethodGet, "/api/auth/me", nil))
	rec := httptest.NewRecorder()
	env.server.handleMe(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var resp userResponse
	decodeBody(t, rec, &resp)
	if resp.ID != env.user.ID || resp.Username != env.user.Username {
		t.Fatalf("unexpected user response: %+v", resp)
	}
}

func TestHandleMeUnauthenticated(t *testing.T) {
	env := newTestEnv(t)
	// No user in context.
	req := buildRequest(t, http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	env.server.handleMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
}
