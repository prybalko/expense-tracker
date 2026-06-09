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

func TestSPAHandler_MissingAssetReturns404(t *testing.T) {
	h, err := newSPAHandler(web.DistFS)
	require.NoError(t, err)

	// Hashed-asset-style URLs that don't exist must NOT receive the index
	// HTML — otherwise the browser tries to parse HTML as JavaScript/CSS.
	cases := []string{
		"/assets/index-abc12345.js",
		"/assets/styles-deadbeef.css",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, http.NoBody)
			req.Header.Set("Accept", "*/*")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}
