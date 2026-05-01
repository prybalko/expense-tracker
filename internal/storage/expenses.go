package storage

import (
	"database/sql"
	"errors"
	"time"

	"expense-tracker/internal/models"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrDuplicateExpense indicates a row with the same
// (user_id, date, amount, description) already exists. Returned by
// InsertExpense/UpdateExpense when the unique index rejects the row, so
// callers can map it to a 409 Conflict instead of a generic 500. This
// matters for the offline replay path: a queued create whose original write
// already landed must be dropped, not retried indefinitely.
var ErrDuplicateExpense = errors.New("expense already exists")

func isUniqueConstraintError(err error) bool {
	var serr *sqlite.Error
	if !errors.As(err, &serr) {
		return false
	}
	code := serr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE ||
		code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY ||
		code == sqlite3.SQLITE_CONSTRAINT
}

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time, userID int64) error {
	_, err := db.InsertExpense(amount, description, category, date, userID)
	return err
}

// InsertExpense inserts a new expense and returns the persisted row, including
// the auto-generated id and resolved date. If the row collides with the
// existing unique index on (user_id, date, amount, description), it returns
// ErrDuplicateExpense so callers can distinguish "already there" from a real
// server error.
//
// Dates are normalised to UTC before binding. The driver serializes time.Time
// as RFC3339Nano text and SQLite compares text rows lexicographically — so
// mixing zones across rows would silently break ORDER BY date.
func (db *DB) InsertExpense(amount float64, description, category string, date time.Time, userID int64) (*models.Expense, error) {
	if date.IsZero() {
		date = time.Now()
	}
	date = date.UTC()
	res, err := db.conn.Exec(
		"INSERT INTO expenses (amount, description, category, date, user_id) VALUES (?, ?, ?, ?, ?)",
		amount, description, category, date, userID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDuplicateExpense
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	uid := userID
	return &models.Expense{
		ID:          id,
		Amount:      amount,
		Description: description,
		Category:    category,
		Date:        date,
		UserID:      &uid,
	}, nil
}

// GetExpense retrieves a single expense visible to userID. A row is visible
// when its user_id matches OR when it has no owner (user_id IS NULL) —
// the latter covers pre-multi-user historical data that predates user
// scoping. Returns sql.ErrNoRows when the row does not exist or is owned
// by a different user, so callers can map both to 404 without leaking
// cross-user existence.
func (db *DB) GetExpense(userID, id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		`SELECT id, amount, description, category, date, user_id FROM expenses
		 WHERE id = ? AND (user_id = ? OR user_id IS NULL)`,
		id, userID,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateExpense updates an existing expense visible to userID — either
// owned by them or unowned (user_id IS NULL, the legacy/shared case). The
// SET clause does not touch user_id, so updating a NULL-owned row keeps it
// NULL; we never silently "claim" a shared historical row for the editing
// user.
//
// If the new (user_id, date, amount, description) tuple collides with the
// unique index, it returns ErrDuplicateExpense so handlers can map it to
// 409 Conflict instead of a generic 500 — the offline-replay path treats
// 5xx as retryable, and a permanent collision would otherwise jam the
// queue.
//
// The date is normalised to UTC for the same reason as InsertExpense: the
// driver writes time.Time as RFC3339Nano text and SQLite compares it
// lexicographically, so mixed zones would corrupt ORDER BY.
func (db *DB) UpdateExpense(userID int64, e *models.Expense) error {
	e.Date = e.Date.UTC()
	_, err := db.conn.Exec(
		`UPDATE expenses SET amount = ?, description = ?, category = ?, date = ?
		 WHERE id = ? AND (user_id = ? OR user_id IS NULL)`,
		e.Amount, e.Description, e.Category, e.Date, e.ID, userID,
	)
	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicateExpense
	}
	return err
}

// DeleteExpense removes an expense visible to userID — either owned by
// them or unowned (user_id IS NULL). Rows owned by a different user are
// silently ignored, preserving cross-user isolation; unowned legacy rows
// are intentionally deletable so any logged-in user can prune them.
func (db *DB) DeleteExpense(userID, id int64) error {
	_, err := db.conn.Exec(
		`DELETE FROM expenses WHERE id = ? AND (user_id = ? OR user_id IS NULL)`,
		id, userID,
	)
	return err
}

// ListExpenses retrieves expenses owned by userID, ordered by date descending.
// Supports pagination with limit and offset parameters. Used by storage tests
// only; the API uses ListExpensesAll.
func (db *DB) ListExpenses(userID int64, limit, offset int) ([]models.Expense, error) {
	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE user_id = ? ORDER BY date DESC LIMIT ? OFFSET ?",
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// ListExpensesAll returns every expense visible to userID, ordered by
// (date DESC, id DESC). A row is visible when it's either owned by the
// user OR has no owner (user_id IS NULL). The NULL-owner branch surfaces
// pre-multi-user historical data — it predates user scoping and would
// otherwise be invisible to every account, hiding years of legacy
// expenses from Insights even though the rows are still in the table.
//
// The API ships the entire array to the client in one response and
// Insights / CategoryDetails derivations run off it locally — the dataset
// is bounded by personal use, and avoiding per-screen aggregation queries
// removes the visual jump that came from coordinating three queries across
// a single page navigation.
func (db *DB) ListExpensesAll(userID int64) ([]models.Expense, error) {
	rows, err := db.conn.Query(
		`SELECT id, amount, description, category, date, user_id FROM expenses
		 WHERE user_id = ? OR user_id IS NULL
		 ORDER BY date DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExpenses(rows)
}

func scanExpenses(rows *sql.Rows) ([]models.Expense, error) {
	var out []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ClearExpenses deletes all expenses from the database (used for testing).
func (db *DB) ClearExpenses() error {
	_, err := db.conn.Exec("DELETE FROM expenses")
	return err
}
