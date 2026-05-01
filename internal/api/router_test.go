package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/internal/auth"
)

func TestRouterRequiresAuth(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"me", http.MethodGet, "/api/auth/me"},
		{"logout", http.MethodPost, "/api/auth/logout"},
		{"list expenses", http.MethodGet, "/api/expenses"},
		{"create expense", http.MethodPost, "/api/expenses"},
		{"insights", http.MethodGet, "/api/insights"},
	}
	for _, tc := range tests {
		t.Run("no cookie/"+tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			req := buildRequest(t, tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			env.router().ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status: got %d want 401", rec.Code)
			}
		})
		t.Run("with cookie/"+tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			req := buildRequest(t, tc.method, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: env.token})
			rec := httptest.NewRecorder()
			env.router().ServeHTTP(rec, req)
			if rec.Code == http.StatusUnauthorized {
				t.Fatalf("authed request to %s returned 401", tc.path)
			}
		})
	}
}

func TestRouterLoginIsPublic(t *testing.T) {
	env := newTestEnv(t)
	router := env.router()

	body := map[string]string{"username": "alice", "password": "hunter2"}
	req := buildRequest(t, http.MethodPost, "/api/auth/login", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestRouterEndToEndCreateAndList(t *testing.T) {
	env := newTestEnv(t)
	router := env.router()

	// Create
	body := map[string]any{
		"amount":      9.99,
		"description": "Test",
		"category":    "Other",
		"date":        "2026-04-15T08:00:00Z",
	}
	req := buildRequest(t, http.MethodPost, "/api/expenses", body)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: env.token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d (body=%q)", rec.Code, rec.Body.String())
	}

	// List
	req = buildRequest(t, http.MethodGet, "/api/expenses", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: env.token})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: got %d", rec.Code)
	}
	var resp listExpensesResponse
	decodeBody(t, rec, &resp)
	if len(resp.Items) != 1 {
		t.Fatalf("items: got %d want 1", len(resp.Items))
	}
}

func TestRouterRejectsBadCookie(t *testing.T) {
	env := newTestEnv(t)
	router := env.router()
	req := buildRequest(t, http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "not-a-real-token"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
}
