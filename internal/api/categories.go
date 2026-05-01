package api

import (
	"net/http"

	"expense-tracker/internal/categories"
)

func (s *Server) handleListCategories(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, categories.All())
}
