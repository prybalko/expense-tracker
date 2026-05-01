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
// mixing zones across rows would silently break ORDER BY date and the
// (date, id) cursor in ListExpensesBefore (whose anchor is reconstructed in
// UTC).
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

// GetExpense retrieves a single expense owned by userID. Returns
// sql.ErrNoRows when the row does not exist or is owned by another user, so
// callers can map both cases to 404 without leaking existence to a different
// user.
func (db *DB) GetExpense(userID, id int64) (*models.Expense, error) {
	row := db.conn.QueryRow(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE id = ? AND user_id = ?",
		id, userID,
	)

	var e models.Expense
	if err := row.Scan(&e.ID, &e.Amount, &e.Description, &e.Category, &e.Date, &e.UserID); err != nil {
		return nil, err
	}
	return &e, nil
}

// UpdateExpense updates an existing expense owned by userID. If the new
// (user_id, date, amount, description) tuple collides with the unique index,
// it returns ErrDuplicateExpense so handlers can map it to 409 Conflict
// instead of a generic 500 — the offline-replay path treats 5xx as
// retryable, and a permanent collision would otherwise jam the queue.
//
// The date is normalised to UTC for the same reason as InsertExpense: the
// driver writes time.Time as RFC3339Nano text and SQLite compares it
// lexicographically, so mixed zones would corrupt ORDER BY and the
// cursor's (date, id) round-trip.
func (db *DB) UpdateExpense(userID int64, e *models.Expense) error {
	e.Date = e.Date.UTC()
	_, err := db.conn.Exec(
		"UPDATE expenses SET amount = ?, description = ?, category = ?, date = ? WHERE id = ? AND user_id = ?",
		e.Amount, e.Description, e.Category, e.Date, e.ID, userID,
	)
	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicateExpense
	}
	return err
}

// DeleteExpense removes an expense owned by userID. Rows owned by another
// user are silently ignored.
func (db *DB) DeleteExpense(userID, id int64) error {
	_, err := db.conn.Exec("DELETE FROM expenses WHERE id = ? AND user_id = ?", id, userID)
	return err
}

// ListExpenses retrieves expenses owned by userID, ordered by date descending.
// Supports pagination with limit and offset parameters. Used by storage tests
// only; the API uses ListExpensesBefore.
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

// ListExpensesBefore returns up to `limit` expenses owned by userID, ordered
// by (date DESC, id DESC). When beforeID > 0, only rows that come strictly
// after the (beforeDate, beforeID) pair in this ordering are returned. The
// caller passes both the date and id from the previous page's tail so that
// pagination keeps working even after the anchor is deleted — we never
// re-read the anchor row from the database. Edits that move the anchor's
// date are the standard keyset-pagination caveat: shifting earlier can cause
// a duplicate, shifting later can cause a skip.
func (db *DB) ListExpensesBefore(userID int64, limit int, beforeDate time.Time, beforeID int64) ([]models.Expense, error) {
	if beforeID > 0 {
		rows, err := db.conn.Query(
			`SELECT id, amount, description, category, date, user_id FROM expenses
			 WHERE user_id = ? AND (date < ? OR (date = ? AND id < ?))
			 ORDER BY date DESC, id DESC
			 LIMIT ?`,
			userID, beforeDate, beforeDate, beforeID, limit,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanExpenses(rows)
	}
	rows, err := db.conn.Query(
		`SELECT id, amount, description, category, date, user_id FROM expenses
		 WHERE user_id = ?
		 ORDER BY date DESC, id DESC
		 LIMIT ?`,
		userID, limit,
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

// ListExpensesByMonthCategory returns expenses owned by userID that fall in the
// given calendar month and match the given category, ordered by (date DESC,
// id DESC). Used by the per-category drill-down on the Insights screen, where
// the result set for one user × one month × one category is bounded enough to
// return without cursor pagination.
func (db *DB) ListExpensesByMonthCategory(userID int64, year, month int, category string) ([]models.Expense, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	rows, err := db.conn.Query(
		`SELECT id, amount, description, category, date, user_id FROM expenses
		 WHERE user_id = ? AND category = ? AND date >= ? AND date < ?
		 ORDER BY date DESC, id DESC`,
		userID, category, startOfMonth, endOfMonth,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExpenses(rows)
}

// GetExpensesByMonth retrieves expenses for a specific month, owned by userID.
func (db *DB) GetExpensesByMonth(userID int64, year, month int) ([]models.Expense, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE user_id = ? AND date >= ? AND date < ? ORDER BY date DESC",
		userID, startOfMonth, endOfMonth,
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

// GetCategoryTotalsByMonth retrieves spending totals by category for a specific
// month, scoped to userID.
func (db *DB) GetCategoryTotalsByMonth(userID int64, year, month int) ([]CategoryTotal, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	rows, err := db.conn.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count
		 FROM expenses
		 WHERE user_id = ? AND date >= ? AND date < ?
		 GROUP BY category
		 ORDER BY total DESC`,
		userID, startOfMonth, endOfMonth,
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

// GetMonthlyTotalsForYear retrieves spending totals by month for a specific
// year, scoped to userID.
func (db *DB) GetMonthlyTotalsForYear(userID int64, year int) ([]MonthlyTotal, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	// Use SUBSTR to extract month from ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)
	rows, err := db.conn.Query(
		`SELECT CAST(SUBSTR(date, 6, 2) AS INTEGER) as month, SUM(amount) as total
		 FROM expenses
		 WHERE user_id = ? AND date >= ? AND date < ?
		 GROUP BY SUBSTR(date, 6, 2)
		 ORDER BY month`,
		userID, startOfYear, endOfYear,
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

// GetDailyTotalsForMonth retrieves spending totals by day for a specific
// month, scoped to userID.
func (db *DB) GetDailyTotalsForMonth(userID int64, year, month int) ([]DailyTotal, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	// Use SUBSTR to extract day from ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ)
	rows, err := db.conn.Query(
		`SELECT CAST(SUBSTR(date, 9, 2) AS INTEGER) as day, SUM(amount) as total
		 FROM expenses
		 WHERE user_id = ? AND date >= ? AND date < ?
		 GROUP BY SUBSTR(date, 9, 2)
		 ORDER BY day`,
		userID, startOfMonth, endOfMonth,
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

// GetTotalForRange returns the sum of expense amounts in [start, end), scoped
// to userID.
func (db *DB) GetTotalForRange(userID int64, start, end time.Time) (float64, error) {
	var total float64
	err := db.conn.QueryRow(
		`SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE user_id = ? AND date >= ? AND date < ?`,
		userID, start, end,
	).Scan(&total)
	return total, err
}

// GetTotalForPeriod retrieves the total spending for a period, scoped to
// userID. If month is 0, it returns the total for the entire year. Otherwise,
// it returns the total for the specific month.
func (db *DB) GetTotalForPeriod(userID int64, year, month int) (float64, error) {
	var startDate, endDate time.Time

	if month == 0 {
		startDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(1, 0, 0)
	} else {
		startDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0)
	}

	return db.GetTotalForRange(userID, startDate, endDate)
}

// GetExpensesByYear retrieves all expenses for a specific year, owned by
// userID.
func (db *DB) GetExpensesByYear(userID int64, year int) ([]models.Expense, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	rows, err := db.conn.Query(
		"SELECT id, amount, description, category, date, user_id FROM expenses WHERE user_id = ? AND date >= ? AND date < ? ORDER BY date DESC",
		userID, startOfYear, endOfYear,
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

// GetCategoryTotalsByYear retrieves spending totals by category for a specific
// year, scoped to userID.
func (db *DB) GetCategoryTotalsByYear(userID int64, year int) ([]CategoryTotal, error) {
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endOfYear := startOfYear.AddDate(1, 0, 0)

	rows, err := db.conn.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count
		 FROM expenses
		 WHERE user_id = ? AND date >= ? AND date < ?
		 GROUP BY category
		 ORDER BY total DESC`,
		userID, startOfYear, endOfYear,
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
