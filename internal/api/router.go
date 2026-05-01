package api

import (
	"net/http"

	"expense-tracker/internal/storage"
)

// Server bundles the dependencies the JSON handlers need.
type Server struct {
	db           *storage.DB
	secureCookie bool
}

// NewServer constructs a Server.
func NewServer(db *storage.DB, secureCookie bool) *Server {
	return &Server{db: db, secureCookie: secureCookie}
}

// NewRouter wires every /api/* route. Public routes (login) bypass the auth
// middleware; everything else requires a valid session cookie.
func NewRouter(db *storage.DB, secureCookie bool) http.Handler {
	s := NewServer(db, secureCookie)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)

	protected := func(h http.HandlerFunc) http.Handler {
		return s.authMiddleware(h)
	}
	mux.Handle("POST /api/auth/logout", protected(s.handleLogout))
	mux.Handle("GET /api/auth/me", protected(s.handleMe))
	mux.Handle("GET /api/categories", protected(s.handleListCategories))
	mux.Handle("GET /api/expenses", protected(s.handleListExpenses))
	mux.Handle("POST /api/expenses", protected(s.handleCreateExpense))
	mux.Handle("PATCH /api/expenses/{id}", protected(s.handleUpdateExpense))
	mux.Handle("DELETE /api/expenses/{id}", protected(s.handleDeleteExpense))
	mux.Handle("GET /api/insights", protected(s.handleInsights))

	return mux
}
