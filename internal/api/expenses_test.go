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

	t.Run("returns latest first with no cursor", func(t *testing.T) {
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
		if resp.NextCursor != nil {
			t.Fatalf("nextCursor: got %v want nil", *resp.NextCursor)
		}
	})

	t.Run("paginates with limit and before", func(t *testing.T) {
		// First page (limit=2)
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?limit=2", nil))
		rec := httptest.NewRecorder()
		env.server.handleListExpenses(rec, req)
		var page1 listExpensesResponse
		decodeBody(t, rec, &page1)
		if len(page1.Items) != 2 {
			t.Fatalf("page1 items: got %d want 2", len(page1.Items))
		}
		if page1.NextCursor == nil {
			t.Fatalf("expected nextCursor on page 1")
		}
		// Second page using cursor
		req = env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?limit=2&before="+*page1.NextCursor, nil))
		rec = httptest.NewRecorder()
		env.server.handleListExpenses(rec, req)
		var page2 listExpensesResponse
		decodeBody(t, rec, &page2)
		if len(page2.Items) != 1 {
			t.Fatalf("page2 items: got %d want 1", len(page2.Items))
		}
		if page2.Items[0].Description != "oldest" {
			t.Fatalf("page2: got %q want oldest", page2.Items[0].Description)
		}
		if page2.NextCursor != nil {
			t.Fatalf("page2 nextCursor: got %v want nil", *page2.NextCursor)
		}
	})

	t.Run("invalid cursor returns 400", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?before=abc", nil))
		rec := httptest.NewRecorder()
		env.server.handleListExpenses(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
		}
	})

	t.Run("invalid limit returns 400", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?limit=-1", nil))
		rec := httptest.NewRecorder()
		env.server.handleListExpenses(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", rec.Code)
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

func TestHandleListExpensesPaginationSurvivesAnchorDelete(t *testing.T) {
	env := newTestEnv(t)
	base := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	for i, d := range []string{"oldest", "second", "third", "newest"} {
		_, err := env.db.InsertExpense(float64((i+1)*10), d, "Other", base.Add(time.Duration(i)*time.Hour), env.user.ID)
		if err != nil {
			t.Fatalf("seed %s: %v", d, err)
		}
	}

	// Page 1 with limit=2 → returns ["newest", "third"], cursor anchored on "third".
	req := env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?limit=2", nil))
	rec := httptest.NewRecorder()
	env.server.handleListExpenses(rec, req)
	var page1 listExpensesResponse
	decodeBody(t, rec, &page1)
	if len(page1.Items) != 2 || page1.NextCursor == nil {
		t.Fatalf("page1 setup wrong: %+v", page1)
	}
	cursor := *page1.NextCursor

	// Delete the anchor row ("third"). With the old ID-only cursor scheme the
	// next page would silently come back empty.
	if err := env.db.DeleteExpense(env.user.ID, page1.Items[1].ID); err != nil {
		t.Fatalf("delete anchor: %v", err)
	}

	// Page 2 with the cached cursor must still surface "second" and "oldest".
	req = env.withUser(buildRequest(t, http.MethodGet, "/api/expenses?limit=2&before="+cursor, nil))
	rec = httptest.NewRecorder()
	env.server.handleListExpenses(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("page2 status: got %d (body=%q)", rec.Code, rec.Body.String())
	}
	var page2 listExpensesResponse
	decodeBody(t, rec, &page2)
	if len(page2.Items) != 2 {
		t.Fatalf("page2 items: got %d want 2 (%+v)", len(page2.Items), page2.Items)
	}
	if page2.Items[0].Description != "second" || page2.Items[1].Description != "oldest" {
		t.Fatalf("page2 ordering wrong: %+v", page2.Items)
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

	// Bob's insights for the same period only see Bob's totals.
	req = withBob(buildRequest(t, http.MethodGet, "/api/insights?view=month&year=2026&month=4", nil))
	rec = httptest.NewRecorder()
	env.server.handleInsights(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob insights: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
	}
	var bobInsights map[string]any
	decodeBody(t, rec, &bobInsights)
	if total, _ := bobInsights["total"].(float64); total != 4.50 {
		t.Fatalf("bob insights total: got %v want 4.50 (alice's row must not leak)", bobInsights["total"])
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
