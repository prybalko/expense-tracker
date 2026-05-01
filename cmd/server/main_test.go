package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/web"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSPAHandler_ServesEmbeddedIndex(t *testing.T) {
	h, err := newSPAHandler(web.DistFS)
	require.NoError(t, err)

	tests := []struct {
		name string
		path string
	}{
		{"root", "/"},
		{"unknown deep path falls back to index", "/expenses/123"},
		{"unknown sibling falls back to index", "/insights"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
			body := w.Body.String()
			assert.True(t, strings.HasPrefix(strings.TrimSpace(body), "<!doctype html>"),
				"expected SPA fallback to return HTML doc, got: %q", body)
		})
	}
}
