package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
)

type testEnv struct {
	t      *testing.T
	db     *storage.DB
	server *Server
	user   *models.User
	token  string
}

// newTestEnv spins up an in-memory DB, registers a user, creates a session,
// and returns a ready-to-use test environment.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db, err := storage.NewDB(":memory:")
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
	if err := db.CreateSession(token, user.ID, time.Now().Add(auth.SessionDuration)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	return &testEnv{
		t:      t,
		db:     db,
		server: NewServer(db, false),
		user:   user,
		token:  token,
	}
}

// authedRequest builds a request with the test session cookie set.
func (e *testEnv) authedRequest(method, target string, body any) *http.Request {
	e.t.Helper()
	req := buildRequest(e.t, method, target, body)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: e.token})
	return req
}

// withUser injects the user into the request context, simulating what the
// auth middleware does for protected handlers.
func (e *testEnv) withUser(req *http.Request) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserContextKey, e.user)
	return req.WithContext(ctx)
}

// router returns the live router (with middleware) for end-to-end-style tests.
func (e *testEnv) router() http.Handler {
	return NewRouter(e.db, false)
}

func buildRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			rdr = strings.NewReader(b)
		case []byte:
			rdr = bytes.NewReader(b)
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			rdr = bytes.NewReader(buf)
		}
	}
	req := httptest.NewRequest(method, target, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v (body=%q)", err, rec.Body.String())
	}
}
