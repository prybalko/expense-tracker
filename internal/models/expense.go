package models

import "time"

// Expense represents a financial expense record. UpdatedAt is the delta-sync
// cursor: it advances on every Insert/Update/Delete (soft) and is the
// field the client diffs against when asking "what changed since lastSyncAt".
// Storage keeps a deleted_at column for tombstones, but that never appears in
// the JSON payload — the API surfaces removals via a separate `deletedIds`
// list on the changes endpoint.
type Expense struct {
	ID          int64     `json:"id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
	UserID      *int64    `json:"user_id,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User represents a user account.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a user session.
type Session struct {
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
