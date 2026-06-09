package storage

import (
	"database/sql"

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
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS expenses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			amount REAL NOT NULL,
			description TEXT NOT NULL,
			category TEXT NOT NULL,
			date DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return err
		}
	}

	// Add user_id column to expenses if it doesn't exist (for backwards compatibility)
	// We ignore the error here because the column might already exist
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id)`)

	// Add last_activity column to sessions for rolling sessions
	_, _ = db.conn.Exec(`ALTER TABLE sessions ADD COLUMN last_activity DATETIME DEFAULT CURRENT_TIMESTAMP`)

	// Add updated_at / deleted_at for delta sync. The client stores a
	// lastSyncAt marker and asks for everything with updated_at > it on
	// Feed entry; deletes flip deleted_at instead of removing the row so
	// the diff can report them as tombstones. Existing rows get a
	// backfilled updated_at so the very first post-migration diff call
	// doesn't blindly re-emit every historical row as "new".
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN updated_at DATETIME`)
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN deleted_at DATETIME`)
	_, _ = db.conn.Exec(`UPDATE expenses SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL`)

	// Partial unique index on (user_id, date, amount, description) that
	// only covers live (non-tombstoned) rows. The non-partial variant from
	// before soft-delete would prevent the user from legitimately recording
	// an identical tuple after deleting the original — now that deletes
	// don't remove the row, the index must scope itself to deleted_at IS
	// NULL to match the new semantics.
	_, _ = db.conn.Exec(`DROP INDEX IF EXISTS expenses_date_amount_description_uindex`)
	_, _ = db.conn.Exec(`DROP INDEX IF EXISTS expenses_user_date_amount_description_uindex`)
	_, _ = db.conn.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS expenses_user_date_amount_description_uindex ON expenses (user_id, date, amount, description) WHERE deleted_at IS NULL`)

	// Secondary index on updated_at to keep the delta-sync query O(log n)
	// on the pruned set. Without this the Feed diff scans the whole table.
	_, _ = db.conn.Exec(`CREATE INDEX IF NOT EXISTS expenses_user_updated_at_idx ON expenses (user_id, updated_at)`)
	_, _ = db.conn.Exec(`CREATE INDEX IF NOT EXISTS expenses_user_deleted_at_idx ON expenses (user_id, deleted_at) WHERE deleted_at IS NOT NULL`)
	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
