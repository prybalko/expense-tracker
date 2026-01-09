package storage

import "expense-tracker/internal/models"

// CreateUser creates a new user with the given username and password hash.
func (db *DB) CreateUser(username, passwordHash string) (*models.User, error) {
	result, err := db.conn.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetUserByID(id)
}

// GetUserByID retrieves a user by ID.
func (db *DB) GetUserByID(id int64) (*models.User, error) {
	row := db.conn.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE id = ?",
		id,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByUsername retrieves a user by username.
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	row := db.conn.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE username = ?",
		username,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// UserCount returns the number of users in the database.
func (db *DB) UserCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}
