package storage

import (
	"time"

	"expense-tracker/internal/models"
)

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time, userID int64) error {
	if date.IsZero() {
		date = time.Now()
	}
	_, err := db.conn.Exec(
		"INSERT INTO expenses (amount, description, category, date, user_id) VALUES (?, ?, ?, ?, ?)",
		amount, description, category, date, userID,
	)
	return err
}

// GetExpense retrieves a single expense by ID.
func (db *DB) GetExpense(id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE id = ?",
		id,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
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

// DeleteExpense removes an expense from the database by ID.
func (db *DB) DeleteExpense(id int64) error {
	_, err := db.conn.Exec("DELETE FROM expenses WHERE id = ?", id)
	return err
}

// ListExpenses retrieves expenses from the database, ordered by date descending.
// Supports pagination with limit and offset parameters.
func (db *DB) ListExpenses(limit, offset int) ([]models.Expense, error) {
	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date, user_id FROM expenses ORDER BY date DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

// GetCurrentMonthTotal returns the total spent in the current month.
func (db *DB) GetCurrentMonthTotal() (float64, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var total float64
	err := db.conn.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE date >= ?",
		startOfMonth,
	).Scan(&total)

	return total, err
}

// ClearExpenses deletes all expenses from the database (used for testing).
func (db *DB) ClearExpenses() error {
	_, err := db.conn.Exec("DELETE FROM expenses")
	return err
}

// GetExpensesByMonth retrieves expenses for a specific month.
func (db *DB) GetExpensesByMonth(year, month int) ([]models.Expense, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE date >= ? AND date < ? ORDER BY date DESC",
		startOfMonth, endOfMonth,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

// CategoryTotal represents spending total for a category.
type CategoryTotal struct {
	Category string
	Total    float64
	Count    int
}

// GetCategoryTotalsByMonth retrieves spending totals by category for a specific month.
func (db *DB) GetCategoryTotalsByMonth(year, month int) ([]CategoryTotal, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	rows, err := db.conn.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count 
		 FROM expenses 
		 WHERE date >= ? AND date < ? 
		 GROUP BY category 
		 ORDER BY total DESC`,
		startOfMonth, endOfMonth,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []CategoryTotal
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.Category, &ct.Total, &ct.Count); err != nil {
			return nil, err
		}
		totals = append(totals, ct)
	}

	return totals, rows.Err()
}

// MonthlyTotal represents spending total for a month.
type MonthlyTotal struct {
	Month int
	Total float64
}

// GetMonthlyTotalsForYear retrieves spending totals by month for a specific year.
func (db *DB) GetMonthlyTotalsForYear(year int) ([]MonthlyTotal, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	// Use SUBSTR to extract month from ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)
	rows, err := db.conn.Query(
		`SELECT CAST(SUBSTR(date, 6, 2) AS INTEGER) as month, SUM(amount) as total 
		 FROM expenses 
		 WHERE date >= ? AND date < ? 
		 GROUP BY SUBSTR(date, 6, 2) 
		 ORDER BY month`,
		startOfYear, endOfYear,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []MonthlyTotal
	for rows.Next() {
		var mt MonthlyTotal
		if err := rows.Scan(&mt.Month, &mt.Total); err != nil {
			return nil, err
		}
		totals = append(totals, mt)
	}

	return totals, rows.Err()
}

// DailyTotal represents spending total for a day.
type DailyTotal struct {
	Day   int
	Total float64
}

// GetDailyTotalsForMonth retrieves spending totals by day for a specific month.
func (db *DB) GetDailyTotalsForMonth(year, month int) ([]DailyTotal, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	// Use SUBSTR to extract day from ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)
	rows, err := db.conn.Query(
		`SELECT CAST(SUBSTR(date, 9, 2) AS INTEGER) as day, SUM(amount) as total 
		 FROM expenses 
		 WHERE date >= ? AND date < ? 
		 GROUP BY SUBSTR(date, 9, 2) 
		 ORDER BY day`,
		startOfMonth, endOfMonth,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []DailyTotal
	for rows.Next() {
		var dt DailyTotal
		if err := rows.Scan(&dt.Day, &dt.Total); err != nil {
			return nil, err
		}
		totals = append(totals, dt)
	}

	return totals, rows.Err()
}

// GetTotalForPeriod retrieves the total spending for a period.
// If month is 0, it returns the total for the entire year.
// Otherwise, it returns the total for the specific month.
func (db *DB) GetTotalForPeriod(year, month int) (float64, error) {
	var startDate, endDate time.Time

	if month == 0 {
		// Year total
		startDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(1, 0, 0)
	} else {
		// Month total
		startDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	}

	var total float64
	err := db.conn.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE date >= ? AND date < ?`,
		startDate, endDate,
	).Scan(&total)

	return total, err
}

// GetExpensesByYear retrieves all expenses for a specific year.
func (db *DB) GetExpensesByYear(year int) ([]models.Expense, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE date >= ? AND date < ? ORDER BY date DESC",
		startOfYear, endOfYear,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

// GetCategoryTotalsByYear retrieves spending totals by category for a specific year.
func (db *DB) GetCategoryTotalsByYear(year int) ([]CategoryTotal, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	rows, err := db.conn.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count 
		 FROM expenses 
		 WHERE date >= ? AND date < ? 
		 GROUP BY category 
		 ORDER BY total DESC`,
		startOfYear, endOfYear,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []CategoryTotal
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.Category, &ct.Total, &ct.Count); err != nil {
			return nil, err
		}
		totals = append(totals, ct)
	}

	return totals, rows.Err()
}
