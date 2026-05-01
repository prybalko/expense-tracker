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
	"expense-tracker/internal/storage"
)

const (
	// Caps on free-text fields to keep stored rows and the unique-index
	// (date, amount, description) bounded. The PWA's notes field is a
	// short caption — 200 chars is generous. The category is a freeform
	// string (canonical list lives on the frontend); 64 chars is well
	// over any realistic label. Aligned with the 64 KiB body cap in
	// json.go.
	maxDescriptionLength = 200
	maxCategoryLength    = 64
)

// parseAPIDate accepts RFC3339 or the date-only "2006-01-02" form sent by the
// PWA's date picker. The date-only form is interpreted as UTC midnight.
func parseAPIDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// listExpensesResponse intentionally keeps `nextCursor` as a permanent null
// so the TypeScript client's ExpensePage type doesn't churn during the move
// off pagination. The field exists, the value is always null.
type listExpensesResponse struct {
	Items      []models.Expense `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

func (s *Server) handleGetExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	exp, err := s.db.GetExpense(user.ID, id)
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

// handleListExpenses returns every expense owned by the authenticated user.
// The client caches the array under a single React Query key and derives
// Feed, Insights, and CategoryDetails views from it locally, so this
// endpoint takes no filter / pagination parameters by design.
func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := s.db.ListExpensesAll(user.ID)
	if err != nil {
		log.Printf("api: list expenses: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if items == nil {
		items = []models.Expense{}
	}
	writeJSON(w, http.StatusOK, listExpensesResponse{Items: items, NextCursor: nil})
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
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	if req.Category == "" {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}
	if len(req.Description) > maxDescriptionLength {
		writeError(w, http.StatusBadRequest, "description is too long")
		return
	}
	if len(req.Category) > maxCategoryLength {
		writeError(w, http.StatusBadRequest, "category is too long")
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
		if errors.Is(err, storage.ErrDuplicateExpense) {
			// A row with the same (date, amount, description) already exists.
			// Return 409 so the offline-queue replay path treats this as a
			// drop instead of a retryable 5xx that would block the queue.
			writeError(w, http.StatusConflict, "expense already exists")
			return
		}
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
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := s.db.GetExpense(user.ID, id)
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
	if err := decodeJSON(w, r, &req); err != nil {
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
		desc := strings.TrimSpace(*req.Description)
		if len(desc) > maxDescriptionLength {
			writeError(w, http.StatusBadRequest, "description is too long")
			return
		}
		existing.Description = desc
	}
	if req.Category != nil {
		cat := strings.TrimSpace(*req.Category)
		if cat == "" {
			writeError(w, http.StatusBadRequest, "category cannot be empty")
			return
		}
		if len(cat) > maxCategoryLength {
			writeError(w, http.StatusBadRequest, "category is too long")
			return
		}
		existing.Category = cat
	}
	if req.Date != nil {
		d, err := parseAPIDate(*req.Date)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid date (must be RFC3339 or YYYY-MM-DD)")
			return
		}
		existing.Date = d
	}

	if err := s.db.UpdateExpense(user.ID, existing); err != nil {
		if errors.Is(err, storage.ErrDuplicateExpense) {
			// New (date, amount, description) collides with another row.
			// Return 409 so the offline-queue replay path drops the entry
			// instead of retrying it as a "transient" 5xx and blocking
			// every later queued write.
			writeError(w, http.StatusConflict, "expense already exists")
			return
		}
		log.Printf("api: update expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.db.DeleteExpense(user.ID, id); err != nil {
		log.Printf("api: delete expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
