package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/internal/categories"
)

func TestHandleListCategories(t *testing.T) {
	env := newTestEnv(t)
	req := env.withUser(buildRequest(t, http.MethodGet, "/api/categories", nil))
	rec := httptest.NewRecorder()
	env.server.handleListCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var got []categories.Category
	decodeBody(t, rec, &got)
	want := categories.All()
	if len(got) != len(want) {
		t.Fatalf("length: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}
