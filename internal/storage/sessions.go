package storage

import (
	"errors"
	"time"

	"expense-tracker/internal/models"
)

// ErrSessionExpired reports a known session whose expiry has passed.
var ErrSessionExpired = errors.New("session expired")

// SessionInfo holds session validation data.
type SessionInfo struct {
	User         *models.User
	LastActivity time.Time
	ExpiresAt    time.Time
}

// Session times are bound as UTC: the driver stores time.Time as text, and local
// wall time would change meaning with the process timezone.

// CreateSession creates a new session for a user.
func (db *DB) CreateSession(token string, userID int64, expiresAt time.Time) error {
	now := time.Now()
	_, err := db.conn.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, last_activity) VALUES (?, ?, ?, ?)",
		token, userID, expiresAt.UTC(), now.UTC(),
	)
	return err
}

// ValidateSession checks if a session token is valid and returns the associated user.
func (db *DB) ValidateSession(token string) (*models.User, error) {
	info, err := db.ValidateSessionWithInfo(token)
	if err != nil {
		return nil, err
	}
	return info.User, nil
}

// ValidateSessionWithInfo returns sql.ErrNoRows for an unknown token and ErrSessionExpired
// for an expired one; any other error is a storage failure.
func (db *DB) ValidateSessionWithInfo(token string) (*SessionInfo, error) {
	row := db.conn.QueryRow(`
		SELECT u.id, u.username, u.password_hash, u.created_at, s.last_activity, s.expires_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token = ?
	`, token)

	var u models.User
	var lastActivity, expiresAt time.Time
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &lastActivity, &expiresAt); err != nil {
		return nil, err
	}
	// Expiry is checked in Go: comparing the stored text to CURRENT_TIMESTAMP depends on layout and timezone.
	if !expiresAt.After(time.Now()) {
		return nil, ErrSessionExpired
	}
	return &SessionInfo{
		User:         &u,
		LastActivity: lastActivity,
		ExpiresAt:    expiresAt,
	}, nil
}

// RenewSession updates the last_activity and expires_at for a session.
func (db *DB) RenewSession(token string, newExpiresAt time.Time) error {
	now := time.Now()
	_, err := db.conn.Exec(
		"UPDATE sessions SET last_activity = ?, expires_at = ? WHERE token = ?",
		now.UTC(), newExpiresAt.UTC(), token,
	)
	return err
}

// DeleteSession removes a session by token.
func (db *DB) DeleteSession(token string) error {
	_, err := db.conn.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// CleanExpiredSessions removes all expired sessions.
func (db *DB) CleanExpiredSessions() error {
	_, err := db.conn.Exec("DELETE FROM sessions WHERE expires_at <= ?", time.Now().UTC())
	return err
}
