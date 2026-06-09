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

// expenseColumns is the canonical SELECT list for reading a whole row. Kept
// in one place so adding a column only requires bumping scanExpenses and the
// scanRow helper instead of auditing every query site.
const expenseColumns = "id, amount, description, category, date, user_id, updated_at"

// CreateExpense inserts a new expense into the database.
func (db *DB) CreateExpense(amount float64, description, category string, date time.Time, userID int64) error {
	_, err := db.InsertExpense(amount, description, category, date, userID)
	return err
}

// InsertExpense inserts a new expense and returns the persisted row, including
// the auto-generated id, resolved date, and updated_at timestamp. If the row
// collides with the partial unique index on (user_id, date, amount,
// description) WHERE deleted_at IS NULL, it returns ErrDuplicateExpense so
// callers can distinguish "already there" from a real server error.
//
// Dates are normalised to UTC before binding. The driver serializes time.Time
// as RFC3339Nano text and SQLite compares text rows lexicographically — so
// mixing zones across rows would silently break ORDER BY date.
//
// updated_at is set to wall-clock now() at insert time and returned to the
// caller. The handler echoes it back to the client so the client's lastSyncAt
// advances on every write without a follow-up round trip.
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

// GetExpense retrieves a single live (non-tombstoned) expense. The app is a
// shared-household ledger: every authenticated user can read every row
// regardless of who created it. Soft-deleted rows (deleted_at IS NOT NULL)
// are invisible to every code path above storage; callers see them as "not
// found" so the edit flow doesn't accidentally resurrect a tombstone.
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

// UpdateExpense updates an existing live expense. The SET clause does not
// touch user_id, so the original author survives any edit — whether it's
// the author editing their own row, a co-user editing someone else's, or
// an edit on a legacy NULL-owned historical row. updated_at advances to
// now() so the Feed diff picks this row up on the next sync.
//
// If the new (user_id, date, amount, description) tuple collides with the
// partial unique index, it returns ErrDuplicateExpense so handlers can map
// it to 409 Conflict. Attempts to update a soft-deleted row are silently
// no-ops at the SQL level; callers filter that earlier via GetExpense, which
// already hides tombstoned rows.
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

// DeleteExpense soft-deletes an expense. The row stays in the table with
// deleted_at set to now(); delta-sync uses that timestamp to emit a
// tombstone so every client drops the row from its cache on the next Feed
// diff. updated_at is also bumped so any code path that only tracks
// updated_at (rather than deleted_at separately) still sees the change.
//
// Rows already tombstoned are no-ops — deleting twice doesn't refresh the
// deleted_at cursor, which keeps tombstone ordering stable. Any authenticated
// user can delete any row (shared-household model).
func (db *DB) DeleteExpense(id int64) error {
	now := time.Now().UTC()
	_, err := db.conn.Exec(
		`UPDATE expenses SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	return err
}

// ListExpenses retrieves live expenses across all users, ordered by date
// descending. Soft-deleted rows are excluded. Used by storage tests only;
// the API uses ListExpensesAll.
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

// ListExpensesAll returns every live (non-tombstoned) expense across the
// whole household, ordered by (date DESC, id DESC). The app intentionally
// shares one ledger across every authenticated user — every account sees
// every row, including legacy NULL-owner historical data.
//
// The API ships the entire array to the client in one response and
// Insights / CategoryDetails derivations run off it locally — the dataset
// is bounded by household use, and avoiding per-screen aggregation queries
// removes the visual jump that came from coordinating three queries across
// a single page navigation.
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

// ListExpensesChangedSince powers the Feed-mount diff. It returns:
//
//   - updated: live rows whose updated_at strictly exceeds `since`. These are
//     upserted into the client cache (covers both "new since last sync" and
//     "edited since last sync" in one bucket since the client replaces by
//     id).
//   - deletedIDs: rows tombstoned (deleted_at IS NOT NULL) since `since`. The
//     client drops each id from its cache.
//
// The comparison is strictly greater than on purpose: the handler returns
// serverTime = now() when responding, and the client passes that back on the
// next call. Using `>=` would re-emit the last row observed on every diff.
//
// Visibility mirrors ListExpensesAll: every live row across the household.
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

// ClearExpenses deletes all expenses from the database (used for testing).
// Hard delete on purpose: tests and the e2e harness rely on a clean slate,
// not a per-row tombstone that would still surface through diff queries.
func (db *DB) ClearExpenses() error {
	_, err := db.conn.Exec("DELETE FROM expenses")
	return err
}
