package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker/internal/insights"
)

func TestHandleInsights(t *testing.T) {
	env := newTestEnv(t)
	// Seed some expenses in March 2026.
	march := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, exp := range []struct {
		amount float64
		cat    string
	}{
		{100, "Groceries"},
		{50, "Transport"},
		{25, "Eating Out"},
	} {
		if _, err := env.db.InsertExpense(exp.amount, exp.cat, exp.cat, march, env.user.ID); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	t.Run("month view", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/insights?view=month&year=2026&month=3", nil))
		rec := httptest.NewRecorder()
		env.server.handleInsights(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d (body=%q)", rec.Code, rec.Body.String())
		}
		var got insights.Insights
		decodeBody(t, rec, &got)
		if got.ViewMode != "month" || got.Year != 2026 || got.Month != 3 {
			t.Fatalf("unexpected period: %+v", got)
		}
		if got.Total != 175 {
			t.Fatalf("total: got %v want 175", got.Total)
		}
		if len(got.Categories) != 3 {
			t.Fatalf("categories: got %d want 3", len(got.Categories))
		}
	})

	t.Run("year view", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/insights?view=year&year=2026", nil))
		rec := httptest.NewRecorder()
		env.server.handleInsights(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d", rec.Code)
		}
		var got insights.Insights
		decodeBody(t, rec, &got)
		if got.ViewMode != "year" || got.Year != 2026 {
			t.Fatalf("unexpected period: %+v", got)
		}
		if len(got.Chart) != 12 {
			t.Fatalf("chart: got %d want 12", len(got.Chart))
		}
	})

	t.Run("default view is month", func(t *testing.T) {
		req := env.withUser(buildRequest(t, http.MethodGet, "/api/insights", nil))
		rec := httptest.NewRecorder()
		env.server.handleInsights(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d", rec.Code)
		}
		var got insights.Insights
		decodeBody(t, rec, &got)
		if got.ViewMode != "month" {
			t.Fatalf("default view: got %q want month", got.ViewMode)
		}
	})

	tests := []struct {
		name string
		url  string
		want int
	}{
		{"invalid view", "/api/insights?view=decade", http.StatusBadRequest},
		{"invalid year", "/api/insights?view=month&year=abc", http.StatusBadRequest},
		{"invalid month", "/api/insights?view=month&year=2026&month=13", http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := env.withUser(buildRequest(t, http.MethodGet, tc.url, nil))
			rec := httptest.NewRecorder()
			env.server.handleInsights(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status: got %d want %d", rec.Code, tc.want)
			}
		})
	}
}
