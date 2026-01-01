package models

import "time"

// Expense represents a financial expense record.
type Expense struct {
	ID          int64     `json:"id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
}

