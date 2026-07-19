package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/storage"
)

// findSessionCookie returns the last session cookie set on the response,
// or nil if none was set.
func findSessionCookie(cookies []*http.Cookie) *http.Cookie {
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			found = c
		}
	}
	return found
}

// TestAuthMiddlewareConcurrentAppOpen is the regression test for the
// intermittent-logout bug: a PWA open fires several concurrent authed
// requests (plus a write), and with the session inside its renewal window
// every one of them used to race the renewal UPDATE on a no-busy-timeout
// SQLite — losers got SQLITE_BUSY, which the middleware escalated to a 401
// that destroyed the still-valid session cookie. None of that may happen:
// every request must succeed and no response may clear the cookie.
func TestAuthMiddlewareConcurrentAppOpen(t *testing.T) {
	db, err := storage.NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	hash, err := auth.HashPassword("hunter2")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := db.CreateUser("alice", hash)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	// 14 days out: inside the renewal window, so validation reads race
	// the renewal write exactly like a real day-15+ app open.
	if err := db.CreateSession(token, user.ID, time.Now().Add(14*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	srv := httptest.NewServer(NewRouter(db, false))
	t.Cleanup(srv.Close)

	since := time.Now().UTC().Format(time.RFC3339)
	do := func(round int, method, path string, body string) (*http.Response, error) {
		var rdr io.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, srv.URL+path, rdr)
		if err != nil {
			return nil, err
		}
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		return http.DefaultClient.Do(req)
	}

	const rounds = 25
	for round := 0; round < rounds; round++ {
		type call struct {
			method, path, body string
		}
		calls := []call{
			{http.MethodGet, "/api/auth/me", ""},
			{http.MethodGet, "/api/expenses", ""},
			{http.MethodGet, "/api/expenses", ""},
			{http.MethodGet, "/api/expenses/changes?since=" + since, ""},
			// A concurrent write keeps SQLite's write lock contended the
			// way a real expense save during a background sync does.
			{http.MethodPost, "/api/expenses", fmt.Sprintf(
				`{"amount": 1.5, "description": "row %d", "category": "Other", "date": "2026-04-15T08:00:00Z"}`, round)},
		}

		var wg sync.WaitGroup
		// Each call can emit two errors (bad status AND cleared cookie);
		// undersizing the buffer deadlocks the test when it fails.
		errs := make(chan string, 2*len(calls))
		for _, c := range calls {
			wg.Add(1)
			go func(c call) {
				defer wg.Done()
				res, err := do(round, c.method, c.path, c.body)
				if err != nil {
					errs <- fmt.Sprintf("%s %s: %v", c.method, c.path, err)
					return
				}
				defer res.Body.Close()
				if res.StatusCode >= 400 {
					errs <- fmt.Sprintf("%s %s: status %d", c.method, c.path, res.StatusCode)
				}
				if sc := findSessionCookie(res.Cookies()); sc != nil && (sc.Value == "" || sc.MaxAge < 0) {
					errs <- fmt.Sprintf("%s %s: response cleared the session cookie", c.method, c.path)
				}
			}(c)
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			t.Errorf("round %d: %s", round, e)
		}
		if t.Failed() {
			break
		}
	}
}

// TestAuthMiddlewareStorageErrorDoesNotClearCookie: a storage failure must
// surface as a 500 and leave the client's cookie untouched — only a
// definitively invalid session may clear it.
func TestAuthMiddlewareStorageErrorDoesNotClearCookie(t *testing.T) {
	env := newTestEnv(t)
	router := env.router()
	env.db.Close() // force ValidateSessionWithInfo to fail with a non-ErrNoRows error

	req := env.authedRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want 500", rec.Code)
	}
	if sc := findSessionCookie(rec.Result().Cookies()); sc != nil {
		t.Fatalf("storage error must not touch the session cookie, got Set-Cookie %q (MaxAge=%d)", sc.Value, sc.MaxAge)
	}
}

// TestAuthMiddlewareExpiredSessionClearsCookie: a genuinely expired session
// is the case that SHOULD 401 and clear the cookie.
func TestAuthMiddlewareExpiredSessionClearsCookie(t *testing.T) {
	env := newTestEnv(t)
	token, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if err := env.db.CreateSession(token, env.user.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := buildRequest(t, http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	env.router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
	sc := findSessionCookie(rec.Result().Cookies())
	if sc == nil || sc.MaxAge >= 0 {
		t.Fatalf("expired session should clear the cookie, got %+v", sc)
	}
}

// TestAuthMiddlewareUnknownTokenClearsCookie mirrors the expired case for a
// token that was never issued.
func TestAuthMiddlewareUnknownTokenClearsCookie(t *testing.T) {
	env := newTestEnv(t)
	req := buildRequest(t, http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "never-issued"})
	rec := httptest.NewRecorder()
	env.router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
	sc := findSessionCookie(rec.Result().Cookies())
	if sc == nil || sc.MaxAge >= 0 {
		t.Fatalf("unknown token should clear the cookie, got %+v", sc)
	}
}

// TestAuthMiddlewareRefreshesCookieEveryRequest: the cookie's MaxAge is
// re-sent on every authenticated response so client-side expiry can never
// drift behind the server-side session.
func TestAuthMiddlewareRefreshesCookieEveryRequest(t *testing.T) {
	env := newTestEnv(t)
	req := env.authedRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	env.router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	sc := findSessionCookie(rec.Result().Cookies())
	if sc == nil {
		t.Fatalf("authed response did not re-send the session cookie")
	}
	if sc.Value != env.token {
		t.Fatalf("cookie value changed: got %q want %q", sc.Value, env.token)
	}
	if sc.MaxAge != int(auth.SessionDuration.Seconds()) {
		t.Fatalf("cookie MaxAge: got %d want %d", sc.MaxAge, int(auth.SessionDuration.Seconds()))
	}
}

// TestAuthMiddlewareRenewalExtendsSession: one request inside the renewal
// window must push the DB expiry back out to a full SessionDuration.
func TestAuthMiddlewareRenewalExtendsSession(t *testing.T) {
	env := newTestEnv(t)
	token, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if err := env.db.CreateSession(token, env.user.ID, time.Now().Add(10*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := buildRequest(t, http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	env.router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}

	info, err := env.db.ValidateSessionWithInfo(token)
	if err != nil {
		t.Fatalf("validate after renewal: %v", err)
	}
	if remaining := time.Until(info.ExpiresAt); remaining < 29*24*time.Hour {
		t.Fatalf("session was not renewed: %v remaining", remaining)
	}
}

// TestClaimRenewalSingleFlight: only one request per token per minute may
// perform the renewal write.
func TestClaimRenewalSingleFlight(t *testing.T) {
	env := newTestEnv(t)
	now := time.Now()

	if !env.server.claimRenewal("tok-a", now) {
		t.Fatalf("first claim for tok-a should win")
	}
	if env.server.claimRenewal("tok-a", now.Add(time.Second)) {
		t.Fatalf("second claim for tok-a within a minute should lose")
	}
	if !env.server.claimRenewal("tok-b", now) {
		t.Fatalf("claim for a different token should win")
	}
	if !env.server.claimRenewal("tok-a", now.Add(2*time.Minute)) {
		t.Fatalf("claim for tok-a after the window should win again")
	}
}

// TestHandleLogoutOverridesCookieRefresh: through the full router, logout
// runs after the middleware queued a cookie refresh — the response must
// carry exactly one session cookie, and it must be the clearing one.
func TestHandleLogoutOverridesCookieRefresh(t *testing.T) {
	env := newTestEnv(t)
	req := env.authedRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	env.router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var sessionCookies []*http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			sessionCookies = append(sessionCookies, c)
		}
	}
	if len(sessionCookies) != 1 {
		t.Fatalf("want exactly 1 session Set-Cookie, got %d", len(sessionCookies))
	}
	if sessionCookies[0].MaxAge >= 0 || sessionCookies[0].Value != "" {
		t.Fatalf("logout response must clear the cookie, got %+v", sessionCookies[0])
	}
}
