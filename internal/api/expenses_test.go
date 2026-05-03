package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"
)

func TestHandleListExpenses(t *testing.T) {
	env := newTestEnv(t)

	// Seed 3 expenses with distinct dates so ordering is unambiguous.
	base := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	for i, d := range []string{"oldest", "middle", "newest"} {
		_, err := env.db.InsertExpense(float64((i+1)*10), d, "Other", base.Add(time.Duration(i)*time.Hour), env.user.ID)
		if err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}

	t.Run("returns every row latest first", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses", nil))
		rec := httptest.NewRecorder()
		env.server.handleListExpenses(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200", rec.Code)
		}
		var resp listExpensesResponse
		decodeBody(t, rec, &resp)
		if len(resp.Items) != 3 {
			t.Fatalf("items: got %d want 3", len(resp.Items))
		}
		if resp.Items[0].Description != "newest" || resp.Items[2].Description != "oldest" {
			t.Fatalf("ordering wrong: %+v", resp.Items)
		}
		// nextCursor stays nil forever — kept in the response shape so the
		// TS ExpensePage type doesn't churn during the move off pagination.
		if resp.NextCursor != nil {
			t.Fatalf("nextCursor: got %v want nil", *resp.NextCursor)
		}
	})

	t.Run("ignores legacy query params", func(t *testing.T) {
		// The handler used to interpret limit/before/year/month/category;
		// after the all-in refactor it returns the full list regardless.
		// A stale client sending these params should still get the full
		// dataset back rather than an error.
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?limit=1&year=2026&month=1&category=Other", nil))
		rec := httptest.NewRecorder()
		env.server.handleListExpenses(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		var resp listExpensesResponse
		decodeBody(t, rec, &resp)
		if len(resp.Items) != 3 {
			t.Fatalf("items: got %d want 3 (params should be ignored)", len(resp.Items))
		}
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		// Fresh env with no expenses.
		fresh := newTestEnv(t)
		req := fresh.withUser(buildRequest(t, http.MethodGet, "/api/expenses", nil))
		rec := httptest.NewRecorder()
		fresh.server.handleListExpenses(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d", rec.Code)
		}
		var resp listExpensesResponse
		decodeBody(t, rec, &resp)
		if resp.Items == nil {
			t.Fatalf("items should be non-nil empty slice, got nil (would marshal as null)")
		}
		if len(resp.Items) != 0 {
			t.Fatalf("expected 0 items, got %d", len(resp.Items))
		}
	})
}

func TestHandleGetExpense(t *testing.T) {
	env := newTestEnv(t)
	created, err := env.db.InsertExpense(42, "Find me", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses/"+itoa(created.ID), nil))
		req.SetPathValue("id", itoa(created.ID))
		rec := httptest.NewRecorder()
		env.server.handleGetExpense(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		var got models.Expense
		decodeBody(t, rec, &got)
		if got.ID != created.ID || got.Description != "Find me" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses/99999", nil))
		req.SetPathValue("id", "99999")
		rec := httptest.NewRecorder()
		env.server.handleGetExpense(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status: got %d want 404", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses/abc", nil))
		req.SetPathValue("id", "abc")
		rec := httptest.NewRecorder()
		env.server.handleGetExpense(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})
}

func TestHandleCreateExpense(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name: "happy path",
			body: map[string]any{
				"amount": 12.34, "description": "Coffee", "category": "Eating Out",
				"date": "2026-04-15T08:30:00Z",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing date is allowed",
			body:       map[string]any{"amount": 5, "description": "x", "category": "Other"},
			wantStatus: http.StatusCreated,
		},
		{
			name: "date-only YYYY-MM-DD is accepted",
			body: map[string]any{
				"amount": 5, "description": "x", "category": "Other",
				"date": "2026-04-15",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "amount must be positive",
			body:       map[string]any{"amount": 0, "category": "Other"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "category required",
			body:       map[string]any{"amount": 5, "category": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid date",
			body:       map[string]any{"amount": 5, "category": "Other", "date": "yesterday"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed json",
			body:       "{",
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			req := env.withUser(buildRequest(t, http.MethodPost, "/api/expenses", tc.body))
			rec := httptest.NewRecorder()
			env.server.handleCreateExpense(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status: got %d want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == http.StatusCreated {
				var got models.Expense
				decodeBody(t, rec, &got)
				if got.ID == 0 {
					t.Fatalf("expected non-zero id, got %+v", got)
				}
				if got.UserID == nil || *got.UserID != env.user.ID {
					t.Fatalf("expected user id %d, got %+v", env.user.ID, got.UserID)
				}
			}
		})
	}
}

func TestHandleCreateExpenseDuplicateReturns409(t *testing.T) {
	env := newTestEnv(t)

	// First insert succeeds.
	body := map[string]any{
		"amount": 12.34, "description": "Coffee", "category": "Eating Out",
		"date": "2026-04-15T08:30:00Z",
	}
	req := env.withUser(buildRequest(t, http.MethodPost, "/api/expenses", body))
	rec := httptest.NewRecorder()
	env.server.handleCreateExpense(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first insert: got %d want 201 (body=%q)", rec.Code, rec.Body.String())
	}

	// Replaying the same payload (e.g. an offline-queue retry whose original
	// already landed) must come back as 409 — not 500 — so the queued entry
	// is dropped instead of jammed behind a "retryable" 5xx.
	req = env.withUser(buildRequest(t, http.MethodPost, "/api/expenses", body))
	rec = httptest.NewRecorder()
	env.server.handleCreateExpense(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate insert: got %d want 409 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateExpenseRejectsOversizedFields(t *testing.T) {
	env := newTestEnv(t)
	long := strings.Repeat("a", maxDescriptionLength+1)
	body := map[string]any{
		"amount": 5, "description": long, "category": "Other",
	}
	req := env.withUser(buildRequest(t, http.MethodPost, "/api/expenses", body))
	rec := httptest.NewRecorder()
	env.server.handleCreateExpense(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized description: got %d want 400 (body=%q)", rec.Code, rec.Body.String())
	}

	body = map[string]any{
		"amount": 5, "description": "ok",
		"category": strings.Repeat("c", maxCategoryLength+1),
	}
	req = env.withUser(buildRequest(t, http.MethodPost, "/api/expenses", body))
	rec = httptest.NewRecorder()
	env.server.handleCreateExpense(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized category: got %d want 400 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateExpenseRejectsOversizedBody(t *testing.T) {
	env := newTestEnv(t)
	// Build a JSON body larger than maxRequestBody so MaxBytesReader trips.
	huge := strings.Repeat("a", maxRequestBody+1024)
	body := map[string]any{
		"amount": 5, "description": huge, "category": "Other",
	}
	req := env.withUser(buildRequest(t, http.MethodPost, "/api/expenses", body))
	rec := httptest.NewRecorder()
	env.server.handleCreateExpense(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized body: got %d want 400 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateExpenseUnauthenticated(t *testing.T) {
	env := newTestEnv(t)
	// No user in context.
	body := map[string]any{"amount": 5, "category": "Other"}
	req := buildRequest(t, http.MethodPost, "/api/expenses", body)
	rec := httptest.NewRecorder()
	env.server.handleCreateExpense(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d want 401", rec.Code)
	}
}

func TestHandleUpdateExpense(t *testing.T) {
	env := newTestEnv(t)
	created, err := env.db.InsertExpense(10, "Old", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("happy path partial update", func(t *testing.T) {
		body := map[string]any{"description": "New", "amount": 25.5}
		req := env.withUser(buildRequest(t, http.MethodPatch, "/api/expenses/"+itoa(created.ID), body))
		req.SetPathValue("id", itoa(created.ID))
		rec := httptest.NewRecorder()
		env.server.handleUpdateExpense(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		var got models.Expense
		decodeBody(t, rec, &got)
		if got.Description != "New" {
			t.Fatalf("description: got %q want New", got.Description)
		}
		if got.Amount != 25.5 {
			t.Fatalf("amount: got %v want 25.5", got.Amount)
		}
		if got.Category != "Other" {
			t.Fatalf("category should be unchanged, got %q", got.Category)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodPatch, "/api/expenses/99999", map[string]any{"amount": 1}))
		req.SetPathValue("id", "99999")
		rec := httptest.NewRecorder()
		env.server.handleUpdateExpense(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status: got %d want 404", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodPatch, "/api/expenses/abc", map[string]any{"amount": 1}))
		req.SetPathValue("id", "abc")
		rec := httptest.NewRecorder()
		env.server.handleUpdateExpense(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})

	t.Run("invalid amount", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodPatch, "/api/expenses/"+itoa(created.ID), map[string]any{"amount": -1.0}))
		req.SetPathValue("id", itoa(created.ID))
		rec := httptest.NewRecorder()
		env.server.handleUpdateExpense(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})
}

func TestHandleUpdateExpenseDuplicateReturns409(t *testing.T) {
	env := newTestEnv(t)

	base := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	first, err := env.db.InsertExpense(10, "Coffee", "Eating Out", base, env.user.ID)
	if err != nil {
		t.Fatalf("seed first: %v", err)
	}
	second, err := env.db.InsertExpense(20, "Lunch", "Eating Out", base.Add(time.Hour), env.user.ID)
	if err != nil {
		t.Fatalf("seed second: %v", err)
	}

	// Patch `second` so its (date, amount, description) collides with `first`.
	// A queued update whose collision is permanent must come back as 409 —
	// not 500 — so the offline replay path drops it instead of jamming the
	// queue behind a "retryable" 5xx.
	body := map[string]any{
		"amount":      first.Amount,
		"description": first.Description,
		"date":        first.Date.UTC().Format(time.RFC3339Nano),
	}
	req := env.withUser(buildRequest(t, http.MethodPatch, "/api/expenses/"+itoa(second.ID), body))
	req.SetPathValue("id", itoa(second.ID))
	rec := httptest.NewRecorder()
	env.server.handleUpdateExpense(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status: got %d want 409 (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteExpense(t *testing.T) {
	env := newTestEnv(t)
	created, err := env.db.InsertExpense(10, "Bye", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodDelete, "/api/expenses/"+itoa(created.ID), nil))
		req.SetPathValue("id", itoa(created.ID))
		rec := httptest.NewRecorder()
		env.server.handleDeleteExpense(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status: got %d want 204", rec.Code)
		}
		if _, err := env.db.GetExpense(env.user.ID, created.ID); err == nil {
			t.Fatalf("expected expense to be deleted")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodDelete, "/api/expenses/0", nil))
		req.SetPathValue("id", "0")
		rec := httptest.NewRecorder()
		env.server.handleDeleteExpense(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})
}

func itoa(i int64) string { return fmt.Sprintf("%d", i) }

// TestListExpensesIncludesServerTime pins the contract every client
// relies on: the full-list response carries a serverTime the client pins
// as its initial lastSyncAt. Without this field the Feed would have no
// baseline to hand to /api/expenses/changes?since=... and the delta-sync
// flow would degrade to a full refetch on every navigation.
func TestListExpensesIncludesServerTime(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.db.InsertExpense(5, "x", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses", nil))
	rec := httptest.NewRecorder()
	env.server.handleListExpenses(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var resp listExpensesResponse
	decodeBody(t, rec, &resp)
	if resp.ServerTime == "" {
		t.Fatalf("serverTime should be non-empty, got %q", resp.ServerTime)
	}
	if _, err := time.Parse(time.RFC3339Nano, resp.ServerTime); err != nil {
		t.Fatalf("serverTime must parse as RFC3339Nano, got %q (%v)", resp.ServerTime, err)
	}
	if got := rec.Header().Get(serverTimeHeader); got == "" {
		t.Fatalf("expected X-Server-Time header")
	}
}

// TestHandleListChanges exercises the delta endpoint's full contract.
func TestHandleListChanges(t *testing.T) {
	env := newTestEnv(t)

	// Seed a row that will be present BEFORE the cutoff; it should never
	// appear in the diff so long as nothing touches it.
	old, err := env.db.InsertExpense(10, "Old", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed old: %v", err)
	}

	cutoff := time.Now().UTC()
	time.Sleep(2 * time.Millisecond)

	// Insert a new row and edit the old one — both must come back in
	// `updated`.
	fresh, err := env.db.InsertExpense(20, "Fresh", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed fresh: %v", err)
	}
	old.Description = "Old edited"
	if err := env.db.UpdateExpense(env.user.ID, old); err != nil {
		t.Fatalf("update old: %v", err)
	}

	// Seed a row and soft-delete it — its id must come back in
	// `deletedIds`, not `updated`.
	doomed, err := env.db.InsertExpense(30, "Doomed", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed doomed: %v", err)
	}
	if err := env.db.DeleteExpense(env.user.ID, doomed.ID); err != nil {
		t.Fatalf("delete doomed: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		target := "/api/expenses/changes?since=" + cutoff.Format(time.RFC3339Nano)
		req := env.withUser(buildRequest(t, http.MethodGet, target, nil))
		rec := httptest.NewRecorder()
		env.server.handleListChanges(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		var resp changesResponse
		decodeBody(t, rec, &resp)
		if resp.ServerTime == "" {
			t.Fatalf("serverTime missing")
		}

		updatedIDs := map[int64]string{}
		for _, e := range resp.Updated {
			updatedIDs[e.ID] = e.Description
		}
		if updatedIDs[fresh.ID] != "Fresh" {
			t.Fatalf("expected fresh row in updated, got %+v", resp.Updated)
		}
		if updatedIDs[old.ID] != "Old edited" {
			t.Fatalf("expected edited old row in updated with new description, got %+v", resp.Updated)
		}
		if _, seen := updatedIDs[doomed.ID]; seen {
			t.Fatalf("deleted row must not appear in the updated bucket, got %+v", resp.Updated)
		}
		if len(resp.DeletedIDs) != 1 || resp.DeletedIDs[0] != doomed.ID {
			t.Fatalf("expected [%d] in deletedIds, got %+v", doomed.ID, resp.DeletedIDs)
		}
	})

	t.Run("missing since is 400", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses/changes", nil))
		rec := httptest.NewRecorder()
		env.server.handleListChanges(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})

	t.Run("malformed since is 400", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses/changes?since=yesterday", nil))
		rec := httptest.NewRecorder()
		env.server.handleListChanges(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})

	t.Run("empty diff returns empty arrays", func(t *testing.T) {
		future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses/changes?since="+future, nil))
		rec := httptest.NewRecorder()
		env.server.handleListChanges(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200", rec.Code)
		}
		var resp changesResponse
		decodeBody(t, rec, &resp)
		if resp.Updated == nil || resp.DeletedIDs == nil {
			t.Fatalf("empty arrays should be [] not null: %+v", resp)
		}
	})

	t.Run("returns 401 without user", func(t *testing.T) {
		req := buildRequest(t, http.MethodGet, "/api/expenses/changes?since=2000-01-01T00:00:00Z", nil)
		rec := httptest.NewRecorder()
		env.server.handleListChanges(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d want 401", rec.Code)
		}
	})
}

// TestWriteHandlersSetServerTime pins the X-Server-Time header on every
// mutation. The client's lastSyncAt depends on this header for POST (body
// is the row, no serverTime field) and DELETE (body is empty). Without it,
// two back-to-back writes + a Feed sync would re-emit the second write as
// a "change" and briefly show a duplicate row until React's reconciler
// caught up.
func TestWriteHandlersSetServerTime(t *testing.T) {
	env := newTestEnv(t)
	seeded, err := env.db.InsertExpense(10, "Seed", "Other", time.Now(), env.user.ID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	assertHeader := func(t *testing.T, rec *httptest.ResponseRecorder) {
		t.Helper()
		got := rec.Header().Get(serverTimeHeader)
		if got == "" {
			t.Fatalf("missing %s header", serverTimeHeader)
		}
		if _, err := time.Parse(time.RFC3339Nano, got); err != nil {
			t.Fatalf("%s must be RFC3339Nano, got %q (%v)", serverTimeHeader, got, err)
		}
	}

	t.Run("POST", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodPost, "/api/expenses",
			map[string]any{"amount": 5, "description": "New", "category": "Other"}))
		rec := httptest.NewRecorder()
		env.server.handleCreateExpense(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status: got %d want 201 (body=%q)", rec.Code, rec.Body.String())
		}
		assertHeader(t, rec)
	})

	t.Run("PATCH", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodPatch, "/api/expenses/"+itoa(seeded.ID),
			map[string]any{"description": "Edited"}))
		req.SetPathValue("id", itoa(seeded.ID))
		rec := httptest.NewRecorder()
		env.server.handleUpdateExpense(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
		}
		assertHeader(t, rec)
	})

	t.Run("DELETE", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodDelete, "/api/expenses/"+itoa(seeded.ID), nil))
		req.SetPathValue("id", itoa(seeded.ID))
		rec := httptest.NewRecorder()
		env.server.handleDeleteExpense(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status: got %d want 204", rec.Code)
		}
		assertHeader(t, rec)
	})
}

// TestExpenseEndpointsScopedToUser verifies the redesigned read paths return
// 404 / empty results when one authenticated user tries to access another
// user's data. The pre-redesign global queries plus a global
// (date, amount, description) unique index combined into a cross-user
// confidentiality + write-loss bug; this test pins the per-user-scoped
// behavior so it can't silently regress.
func TestExpenseEndpointsScopedToUser(t *testing.T) {
	env := newTestEnv(t)

	// Second user with their own session.
	hash, err := auth.HashPassword("hunter2")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	bob, err := env.db.CreateUser("bob", hash)
	if err != nil {
		t.Fatalf("create user bob: %v", err)
	}

	withBob := func(req *http.Request) *http.Request {
		ctx := context.WithValue(req.Context(), auth.UserContextKey, bob)
		return req.WithContext(ctx)
	}

	// Alice and Bob each insert one expense with the same business-key tuple
	// (date, amount, description). The per-user unique index must allow this.
	date := time.Date(2026, 4, 15, 8, 30, 0, 0, time.UTC)
	aliceExp, err := env.db.InsertExpense(4.50, "Coffee", "Eating Out", date, env.user.ID)
	if err != nil {
		t.Fatalf("alice insert: %v", err)
	}
	bobExp, err := env.db.InsertExpense(4.50, "Coffee", "Eating Out", date, bob.ID)
	if err != nil {
		t.Fatalf("bob insert: %v", err)
	}

	// Alice's list returns only Alice's row.
	req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses", nil))
	rec := httptest.NewRecorder()
	env.server.handleListExpenses(rec, req)
	var aliceList listExpensesResponse
	decodeBody(t, rec, &aliceList)
	if len(aliceList.Items) != 1 || aliceList.Items[0].ID != aliceExp.ID {
		t.Fatalf("alice list got %+v, want only alice row", aliceList.Items)
	}

	// Bob fetching Alice's row by id gets a 404, not Alice's data.
	req = withBob(buildRequest(t, http.MethodGet, "/api/expenses/"+itoa(aliceExp.ID), nil))
	req.SetPathValue("id", itoa(aliceExp.ID))
	rec = httptest.NewRecorder()
	env.server.handleGetExpense(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bob get alice's row: got %d want 404", rec.Code)
	}

	// Bob trying to update Alice's row gets a 404.
	req = withBob(buildRequest(t, http.MethodPatch, "/api/expenses/"+itoa(aliceExp.ID),
		map[string]any{"amount": 999.0}))
	req.SetPathValue("id", itoa(aliceExp.ID))
	rec = httptest.NewRecorder()
	env.server.handleUpdateExpense(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bob update alice's row: got %d want 404", rec.Code)
	}
	// Verify Alice's row is unchanged.
	got, err := env.db.GetExpense(env.user.ID, aliceExp.ID)
	if err != nil {
		t.Fatalf("alice re-read: %v", err)
	}
	if got.Amount != 4.50 {
		t.Fatalf("alice's row was mutated: %+v", got)
	}

	// Bob trying to delete Alice's row succeeds (204) but does not actually
	// remove it — DeleteExpense's WHERE filter scopes to bob.
	req = withBob(buildRequest(t, http.MethodDelete, "/api/expenses/"+itoa(aliceExp.ID), nil))
	req.SetPathValue("id", itoa(aliceExp.ID))
	rec = httptest.NewRecorder()
	env.server.handleDeleteExpense(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("bob delete alice's row: got %d want 204", rec.Code)
	}
	if _, err := env.db.GetExpense(env.user.ID, aliceExp.ID); err != nil {
		t.Fatalf("alice's row should still exist after bob's delete: %v", err)
	}

	// Bob's list returns only Bob's row — confirms ListExpensesAll is
	// scoped per user, so the client's all-in cache for one user can
	// never see another user's rows.
	req = withBob(buildRequest(t, http.MethodGet, "/api/expenses", nil))
	rec = httptest.NewRecorder()
	env.server.handleListExpenses(rec, req)
	var bobList listExpensesResponse
	decodeBody(t, rec, &bobList)
	if len(bobList.Items) != 1 || bobList.Items[0].ID != bobExp.ID {
		t.Fatalf("bob list got %+v, want only bob's row (alice's row must not leak)", bobList.Items)
	}

	// Sanity: bob's own row is reachable to him.
	req = withBob(buildRequest(t, http.MethodGet, "/api/expenses/"+itoa(bobExp.ID), nil))
	req.SetPathValue("id", itoa(bobExp.ID))
	rec = httptest.NewRecorder()
	env.server.handleGetExpense(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob get bob's row: got %d want 200", rec.Code)
	}
}
