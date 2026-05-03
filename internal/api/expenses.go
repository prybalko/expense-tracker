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

	// serverTimeHeader carries the authoritative server wall-clock on
	// every successful expenses-API response. The client mirrors it into
	// its lastSyncAt marker; the next `GET /api/expenses/changes?since=...`
	// call uses that marker to fetch only rows changed since the last
	// round-trip. Writes don't have a server-time field in the body
	// (POST returns the row, DELETE returns 204), so the header is the
	// single place the marker can advance from a mutation.
	serverTimeHeader = "X-Server-Time"
)

// parseAPIDate accepts RFC3339 or the date-only "2006-01-02" form sent by the
// PWA's date picker. The date-only form is interpreted as UTC midnight.
func parseAPIDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// formatServerTime is the single place that decides the textual form of
// updated_at / deleted_at / serverTime wire values. RFC3339Nano matches
// Go's default time.Time JSON encoding, so the field returned in the body
// and the X-Server-Time header are byte-identical, and the client can hand
// either one straight back to `?since=...` on the next diff call.
func formatServerTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// setServerTime stamps the response with the current wall-clock. Handlers
// call it just before writing their body so the client's lastSyncAt can
// advance off this request — even for 204 No Content deletes where the
// body carries no data.
func setServerTime(w http.ResponseWriter) time.Time {
	now := time.Now().UTC()
	w.Header().Set(serverTimeHeader, formatServerTime(now))
	return now
}

// listExpensesResponse carries the full dataset on cold start plus a
// `serverTime` the client pins as its initial lastSyncAt. `nextCursor`
// stays permanently null — the pagination it used to drive is gone, but
// removing the field would churn the TypeScript ExpensePage type for no
// benefit.
type listExpensesResponse struct {
	Items      []models.Expense `json:"items"`
	NextCursor *string          `json:"nextCursor"`
	ServerTime string           `json:"serverTime"`
}

// changesResponse is the delta-sync payload returned by
// GET /api/expenses/changes?since=<ts>. `updated` covers inserts and
// updates uniformly (the client upserts by id), `deletedIds` tombstones,
// and `serverTime` is the cursor the client should use on the next call.
type changesResponse struct {
	Updated    []models.Expense `json:"updated"`
	DeletedIDs []int64          `json:"deletedIds"`
	ServerTime string           `json:"serverTime"`
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
	setServerTime(w)
	writeJSON(w, http.StatusOK, exp)
}

// handleListExpenses returns every live expense owned by the authenticated
// user. The client caches the array under a single React Query key and
// derives Feed, Insights, and CategoryDetails views from it locally, so
// this endpoint takes no filter / pagination parameters by design. The
// response includes `serverTime` so the client can pin its initial
// lastSyncAt; subsequent Feed mounts call /api/expenses/changes?since=...
// with that value rather than refetching the whole list.
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
	now := setServerTime(w)
	writeJSON(w, http.StatusOK, listExpensesResponse{
		Items:      items,
		NextCursor: nil,
		ServerTime: formatServerTime(now),
	})
}

// handleListChanges powers the Feed-mount delta sync. The client supplies
// its lastSyncAt as `?since=<RFC3339>`; we return every row whose
// updated_at is strictly greater than that (inserts + updates) plus the
// ids of every row soft-deleted since then. The response's serverTime
// becomes the client's new lastSyncAt.
//
// `since` is required and must parse as RFC3339 (Go's time.Time default
// marshal) — any other format is a client bug, not a transient condition
// to retry, so we answer 400 instead of silently returning the full list.
func (s *Server) handleListChanges(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
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
	updated, deletedIDs, err := s.db.ListExpensesChangedSince(user.ID, since)
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
	now := setServerTime(w)
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
	setServerTime(w)
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
	setServerTime(w)
	w.WriteHeader(http.StatusNoContent)
}
