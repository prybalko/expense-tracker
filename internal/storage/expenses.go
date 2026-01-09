package storage

import (
	"time"

	"expense-tracker/internal/models"
)

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time) error {
	if date.IsZero() {
		date = time.Now()
	}
	_, err := db.conn.Exec(
		"INSERT INTO expenses (amount, description, category, date) VALUES (?, ?, ?, ?)",
		amount, description, category, date,
	)
	return err
}

// GetExpense retrieves a single expense by ID.
func (db *DB) GetExpense(id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		"SELECT id, amount, description, category, date FROM expenses WHERE id = ?",
		id,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateExpense updates an existing expense in the database.
func (db *DB) UpdateExpense(e *models.Expense) error {
	_, err := db.conn.Exec(
		"UPDATE expenses SET amount = ?, description = ?, category = ?, date = ? WHERE id = ?",
		e.Amount, e.Description, e.Category, e.Date, e.ID,
	)
	return err
}

// ListExpenses retrieves expenses for the current month from the database, ordered by date descending.
func (db *DB) ListExpenses() ([]models.Expense, error) {
	// Calculate start of current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date FROM expenses WHERE date >= ? ORDER BY date DESC",
		startOfMonth,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

// ClearExpenses deletes all expenses from the database (used for testing).
func (db *DB) ClearExpenses() error {
	_, err := db.conn.Exec("DELETE FROM expenses")
	return err
}
