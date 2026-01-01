package main

import (
	"expense-tracker/internal/handlers"
	"expense-tracker/internal/storage"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSetupRouter(t *testing.T) {
	// Setup dependencies
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Use relative paths for tests running in cmd/server
	h := handlers.NewHandlers(db, "../../web/templates")
	
	// Ensure template directory exists, otherwise skip handler initialization if it panics (handlers might check for templates)
	if _, err := os.Stat("../../web/templates"); os.IsNotExist(err) {
		t.Skip("Template directory not found, skipping router test")
	}

	// Create router - this triggers the panic if routing conflict exists
	mux := setupRouter(h, "../../web/static")

	// Verify routes
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "Root redirects to /expenses",
			method:     "GET",
			path:       "/",
			wantStatus: http.StatusFound,
		},
		{
			name:       "Static file access",
			method:     "GET",
			path:       "/static/style.css", // Assuming style.css exists or we check for 404 handled by FileServer
			// If file doesn't exist, FileServer returns 404, but not 405 or 500 panic.
			// The main thing we are testing is that it DOES NOT PANIC during registration.
			wantStatus: http.StatusOK, // Or 404 if file missing, but let's assume it might be missing in test env if not moved
		},
		{
			name: "List Expenses",
			method: "GET",
			path: "/expenses",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			w := httptest.NewRecorder()
			
			mux.ServeHTTP(w, req)

			// Special case for static files: they might return 404 if file not found in test env
			if tt.path == "/static/style.css" {
				if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
					t.Errorf("GET /static/style.css returned %v, expected 200 or 404", w.Code)
				}
				return
			}

			if w.Code != tt.wantStatus {
				t.Errorf("%s %s returned %v, expected %v", tt.method, tt.path, w.Code, tt.wantStatus)
			}
		})
	}
}

