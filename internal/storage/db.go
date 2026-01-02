package storage

import (
	"database/sql"
	"time"

	"expense-tracker/internal/models"

	// Import sqlite driver
	_ "modernc.org/sqlite"
)

// DB wraps a sql.DB connection.
type DB struct {
	conn *sql.DB
}

// NewDB opens a database connection and runs migrations.
func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS expenses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			amount REAL NOT NULL,
			description TEXT NOT NULL,
			category TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time) error {
	if date.IsZero() {
		date = time.Now()
	}
	_, err := db.conn.Exec(
		"INSERT INTO expenses (amount, description, category, created_at) VALUES (?, ?, ?, ?)",
		amount, description, category, date,
	)
	return err
}

// GetExpense retrieves a single expense by ID.
func (db *DB) GetExpense(id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		"SELECT id, amount, description, category, created_at FROM expenses WHERE id = ?",
		id,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.CreatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateExpense updates an existing expense in the database.
func (db *DB) UpdateExpense(e *models.Expense) error {
	_, err := db.conn.Exec(
		"UPDATE expenses SET amount = ?, description = ?, category = ?, created_at = ? WHERE id = ?",
		e.Amount, e.Description, e.Category, e.CreatedAt, e.ID,
	)
	return err
}

// ListExpenses retrieves all expenses from the database, ordered by date descending.
func (db *DB) ListExpenses() ([]models.Expense, error) {
	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, created_at FROM expenses ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.CreatedAt); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
