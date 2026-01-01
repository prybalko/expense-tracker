package handlers

import (
	"expense-tracker/internal/storage"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func setupTestDB(t *testing.T) *storage.DB {
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	return db
}

func TestListExpenses(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check if template dir exists (relative to test execution)
	templateDir := "../../web/templates"
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Skip("Template directory not found, skipping handler integration test")
	}

	h := NewHandlers(db, templateDir)

	req := httptest.NewRequest("GET", "/expenses", http.NoBody)
	w := httptest.NewRecorder()

	h.ListExpenses(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	// Simple content check - look for "Spent this month" which is in list.html
	body := w.Body.String()
	if !strings.Contains(body, "Spent this month") {
		t.Errorf("Expected body to contain 'Spent this month'")
	}
}

func TestCreateExpense(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	templateDir := "../../web/templates"
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Skip("Template directory not found, skipping handler integration test")
	}

	h := NewHandlers(db, templateDir)

	// Simulate form submission
	form := url.Values{}
	form.Add("amount", "15.00")
	form.Add("description", "Lunch Test")
	form.Add("category", "food")
	form.Add("date", "2023-10-27T12:00")

	req := httptest.NewRequest("POST", "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.CreateExpense(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	// Check for HTMX redirect header
	expectedLoc := `{"path":"/expenses", "target":"#content"}`
	if loc := resp.Header.Get("HX-Location"); loc != expectedLoc {
		t.Errorf("Expected HX-Location %s, got %s", expectedLoc, loc)
	}

	// Verify DB insertion
	expenses, _ := db.ListExpenses()
	if len(expenses) != 1 {
		t.Errorf("Expected 1 expense in DB, got %d", len(expenses))
	}
	if expenses[0].Description != "Lunch Test" {
		t.Errorf("Expected description 'Lunch Test', got '%s'", expenses[0].Description)
	}
}
