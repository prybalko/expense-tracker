package models

import "time"

// Expense is an expense row. UpdatedAt is the delta-sync cursor.
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
