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
	// busy_timeout waits for a competing writer instead of failing with SQLITE_BUSY, and WAL
	// lets readers run during a commit. They sit on the DSN so every pooled connection gets them.
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	conn, err := sql.Open("sqlite", dsn)
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

	// Errors ignored: ALTER TABLE ADD COLUMN fails once the column exists.
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id)`)

	_, _ = db.conn.Exec(`ALTER TABLE sessions ADD COLUMN last_activity DATETIME DEFAULT CURRENT_TIMESTAMP`)

	// Backfill updated_at so the first post-migration diff doesn't re-emit every old row.
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN updated_at DATETIME`)
	_, _ = db.conn.Exec(`ALTER TABLE expenses ADD COLUMN deleted_at DATETIME`)
	_, _ = db.conn.Exec(`UPDATE expenses SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL`)

	// The unique index covers only live rows so a deleted tuple can be recorded again.
	_, _ = db.conn.Exec(`DROP INDEX IF EXISTS expenses_date_amount_description_uindex`)
	_, _ = db.conn.Exec(`DROP INDEX IF EXISTS expenses_user_date_amount_description_uindex`)
	_, _ = db.conn.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS expenses_user_date_amount_description_uindex ON expenses (user_id, date, amount, description) WHERE deleted_at IS NULL`)

	_, _ = db.conn.Exec(`CREATE INDEX IF NOT EXISTS expenses_user_updated_at_idx ON expenses (user_id, updated_at)`)
	_, _ = db.conn.Exec(`CREATE INDEX IF NOT EXISTS expenses_user_deleted_at_idx ON expenses (user_id, deleted_at) WHERE deleted_at IS NOT NULL`)
	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
