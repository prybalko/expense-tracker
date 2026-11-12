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
	// Bound stored rows and the (date, amount, description) unique index.
	maxDescriptionLength = 200
	maxCategoryLength    = 64

	serverTimeHeader = "X-Server-Time"

	// changesOverlap re-sends writes stamped before but committed after a
	// cursor (they wait on the SQLite writer lock). Best effort, not a bound.
	changesOverlap = 10 * time.Second
)

func parseAPIDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// Must match time.Time's JSON encoding so body and header values are
// interchangeable as ?since=.
func formatServerTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func setServerTime(w http.ResponseWriter) time.Time {
	now := time.Now().UTC()
	w.Header().Set(serverTimeHeader, formatServerTime(now))
	return now
}

// NextCursor is always null; it stays so the client's ExpensePage type is stable.
type listExpensesResponse struct {
	Items      []models.Expense `json:"items"`
	NextCursor *string          `json:"nextCursor"`
	ServerTime string           `json:"serverTime"`
}

type changesResponse struct {
	Updated    []models.Expense `json:"updated"`
	DeletedIDs []int64          `json:"deletedIds"`
	ServerTime string           `json:"serverTime"`
}

func (s *Server) handleGetExpense(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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
	setServerTime(w)
	writeJSON(w, http.StatusOK, exp)
}

// All users share one ledger, so there is intentionally no per-user filter.
func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Stamped before the read so the cursor never passes a row it missed.
	now := setServerTime(w)
	items, err := s.db.ListExpensesAll()
	if err != nil {
		log.Printf("api: list expenses: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if items == nil {
		items = []models.Expense{}
	}
	writeJSON(w, http.StatusOK, listExpensesResponse{
		Items:      items,
		NextCursor: nil,
		ServerTime: formatServerTime(now),
	})
}

func (s *Server) handleListChanges(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sinceRaw := r.URL.Query().Get("since")
	if sinceRaw == "" {
		writeError(w, http.StatusBadRequest, "missing since")
		return
	}
	since, err := time.Parse(time.RFC3339Nano, sinceRaw)
	if err != nil {
		if t, err2 := time.Parse(time.RFC3339, sinceRaw); err2 == nil {
			since = t
		} else {
			writeError(w, http.StatusBadRequest, "invalid since (must be RFC3339)")
			return
		}
	}
	now := setServerTime(w)
	updated, deletedIDs, err := s.db.ListExpensesChangedSince(since.Add(-changesOverlap))
	if err != nil {
		log.Printf("api: list changes: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if updated == nil {
		updated = []models.Expense{}
	}
	if deletedIDs == nil {
		deletedIDs = []int64{}
	}
	writeJSON(w, http.StatusOK, changesResponse{
		Updated:    updated,
		DeletedIDs: deletedIDs,
		ServerTime: formatServerTime(now),
	})
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
			// 409, not 5xx, so the offline queue drops the replay instead of blocking.
			writeError(w, http.StatusConflict, "expense already exists")
			return
		}
		log.Printf("api: insert expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	setServerTime(w)
	writeJSON(w, http.StatusCreated, created)
}

type updateExpenseRequest struct {
	Amount      *float64 `json:"amount,omitempty"`
	Description *string  `json:"description,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Date        *string  `json:"date,omitempty"`
}

func (s *Server) handleUpdateExpense(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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

	if err := s.db.UpdateExpense(existing); err != nil {
		if errors.Is(err, storage.ErrDuplicateExpense) {
			// 409, not 5xx, so the offline queue drops the replay instead of blocking.
			writeError(w, http.StatusConflict, "expense already exists")
			return
		}
		log.Printf("api: update expense: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	setServerTime(w)
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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
	setServerTime(w)
	w.WriteHeader(http.StatusNoContent)
}
