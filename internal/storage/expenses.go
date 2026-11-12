package storage

import (
	"database/sql"
	"errors"
	"time"

	"expense-tracker/internal/models"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrDuplicateExpense reports a live row with the same (user_id, date, amount, description).
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

// Column order must match the Scan calls in GetExpense and scanExpenses.
const expenseColumns = "id, amount, description, category, date, user_id, updated_at"

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time, userID int64) error {
	_, err := db.InsertExpense(amount, description, category, date, userID)
	return err
}

// InsertExpense inserts an expense and returns the stored row, or ErrDuplicateExpense.
// Dates are bound as UTC because SQLite compares the time text lexicographically.
func (db *DB) InsertExpense(amount float64, description, category string, date time.Time, userID int64) (*models.Expense, error) {
	if date.IsZero() {
		date = time.Now()
	}
	date = date.UTC()
	now := time.Now().UTC()
	res, err := db.conn.Exec(
		"INSERT INTO expenses (amount, description, category, date, user_id, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		amount, description, category, date, userID, now,
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
		UpdatedAt:   now,
	}, nil
}

// GetExpense returns a live expense; tombstoned rows read as sql.ErrNoRows.
func (db *DB) GetExpense(id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		`SELECT `+expenseColumns+` FROM expenses
		 WHERE id = ? AND deleted_at IS NULL`,
		id,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID, &e.UpdatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateExpense updates a live expense, keeps its user_id, and returns ErrDuplicateExpense on collision.
func (db *DB) UpdateExpense(e *models.Expense) error {
	e.Date = e.Date.UTC()
	now := time.Now().UTC()
	_, err := db.conn.Exec(
		`UPDATE expenses SET amount = ?, description = ?, category = ?, date = ?, updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		e.Amount, e.Description, e.Category, e.Date, now, e.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateExpense
		}
		return err
	}
	e.UpdatedAt = now
	return nil
}

// DeleteExpense soft-deletes a live expense so delta sync can report it.
func (db *DB) DeleteExpense(id int64) error {
	now := time.Now().UTC()
	_, err := db.conn.Exec(
		`UPDATE expenses SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	return err
}

// ListExpenses returns a page of live expenses, newest first. Only tests use it.
func (db *DB) ListExpenses(limit, offset int) ([]models.Expense, error) {
	rows, err := db.conn.Query(
		"SELECT "+expenseColumns+" FROM expenses WHERE deleted_at IS NULL ORDER BY date DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// ListExpensesAll returns every live expense of all users, newest first.
func (db *DB) ListExpensesAll() ([]models.Expense, error) {
	rows, err := db.conn.Query(
		`SELECT ` + expenseColumns + ` FROM expenses
		 WHERE deleted_at IS NULL
		 ORDER BY date DESC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// ListExpensesChangedSince returns live rows updated after since and ids deleted after since.
// The comparison is strict: the client passes back serverTime, and >= would re-emit the last row.
func (db *DB) ListExpensesChangedSince(since time.Time) ([]models.Expense, []int64, error) {
	since = since.UTC()

	updRows, err := db.conn.Query(
		`SELECT `+expenseColumns+` FROM expenses
		 WHERE deleted_at IS NULL AND updated_at > ?
		 ORDER BY date DESC, id DESC`,
		since,
	)
	if err != nil {
		return nil, nil, err
	}
	defer updRows.Close()
	updated, err := scanExpenses(updRows)
	if err != nil {
		return nil, nil, err
	}

	delRows, err := db.conn.Query(
		`SELECT id FROM expenses
		 WHERE deleted_at IS NOT NULL AND deleted_at > ?`,
		since,
	)
	if err != nil {
		return nil, nil, err
	}
	defer delRows.Close()
	var deletedIDs []int64
	for delRows.Next() {
		var id int64
		if err := delRows.Scan(&id); err != nil {
			return nil, nil, err
		}
		deletedIDs = append(deletedIDs, id)
	}
	if err := delRows.Err(); err != nil {
		return nil, nil, err
	}
	return updated, deletedIDs, nil
}

func scanExpenses(rows *sql.Rows) ([]models.Expense, error) {
	var out []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ClearExpenses hard-deletes all expenses for tests.
func (db *DB) ClearExpenses() error {
	_, err := db.conn.Exec("DELETE FROM expenses")
	return err
}
