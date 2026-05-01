package api

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// parseAPIDate accepts RFC3339 or the date-only "2006-01-02" form sent by the
// PWA's date picker. The date-only form is interpreted as UTC midnight.
func parseAPIDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

type listExpensesResponse struct {
	Items      []models.Expense `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

func (s *Server) handleGetExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	exp, err := s.db.GetExpense(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "expense not found")
			return
		}
		log.Printf("api: get expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, exp)
}

func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	limit := defaultPageSize
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		if n > maxPageSize {
			n = maxPageSize
		}
		limit = n
	}

	var beforeID int64
	if v := r.URL.Query().Get("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid 'before' cursor")
			return
		}
		beforeID = n
	}

	items, err := s.db.ListExpensesBefore(limit+1, beforeID)
	if err != nil {
		log.Printf("api: list expenses: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var nextCursor *string
	if len(items) > limit {
		items = items[:limit]
		c := strconv.FormatInt(items[len(items)-1].ID, 10)
		nextCursor = &c
	}
	if items == nil {
		items = []models.Expense{}
	}
	writeJSON(w, http.StatusOK, listExpensesResponse{Items: items, NextCursor: nextCursor})
}

type createExpenseRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Date        string  `json:"date,omitempty"`
}

func (s *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createExpenseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	if strings.TrimSpace(req.Category) == "" {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}

	var date time.Time
	if req.Date != "" {
		d, err := parseAPIDate(req.Date)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid date (must be RFC3339 or YYYY-MM-DD)")
			return
		}
		date = d
	}

	created, err := s.db.InsertExpense(req.Amount, req.Description, req.Category, date, user.ID)
	if err != nil {
		log.Printf("api: insert expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type updateExpenseRequest struct {
	Amount      *float64 `json:"amount,omitempty"`
	Description *string  `json:"description,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Date        *string  `json:"date,omitempty"`
}

func (s *Server) handleUpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := s.db.GetExpense(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "expense not found")
			return
		}
		log.Printf("api: get expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var req updateExpenseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Amount != nil {
		if *req.Amount <= 0 {
			writeError(w, http.StatusBadRequest, "amount must be positive")
			return
		}
		existing.Amount = *req.Amount
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Category != nil {
		if strings.TrimSpace(*req.Category) == "" {
			writeError(w, http.StatusBadRequest, "category cannot be empty")
			return
		}
		existing.Category = *req.Category
	}
	if req.Date != nil {
		d, err := parseAPIDate(*req.Date)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid date (must be RFC3339 or YYYY-MM-DD)")
			return
		}
		existing.Date = d
	}

	if err := s.db.UpdateExpense(existing); err != nil {
		log.Printf("api: update expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.db.DeleteExpense(id); err != nil {
		log.Printf("api: delete expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
