package storage

import (
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"

	"github.com/stretchr/testify/suite"
)

// SessionTestSuite provides a test suite for session operations
type SessionTestSuite struct {
	suite.Suite
	db   *DB
	user *models.User
}

// SetupTest runs before each test
func (s *SessionTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	s.Require().NoError(err, "failed to create test database")
	s.db = db

	// Create a test user
	password, err := auth.HashPassword("testpass")
	s.Require().NoError(err, "failed to hash password")

	user, err := s.db.CreateUser("testuser", password)
	s.Require().NoError(err, "failed to create test user")
	s.user = user
}

// TearDownTest runs after each test
func (s *SessionTestSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *SessionTestSuite) TestCreateAndValidateSession() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = s.db.CreateSession(token, s.user.ID, expiresAt)
	s.Require().NoError(err)

	// Validate the session
	sessionUser, err := s.db.ValidateSession(token)
	s.Require().NoError(err)
	s.Equal("testuser", sessionUser.Username)
}

func (s *SessionTestSuite) TestValidateSessionWithInfo() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = s.db.CreateSession(token, s.user.ID, expiresAt)
	s.Require().NoError(err)

	// Get session info
	info, err := s.db.ValidateSessionWithInfo(token)
	s.Require().NoError(err)
	s.Equal("testuser", info.User.Username)

	// Check that last_activity is recent
	timeSinceActivity := time.Since(info.LastActivity)
	s.Less(timeSinceActivity, 5*time.Second, "LastActivity should be recent")
}

func (s *SessionTestSuite) TestRenewSession() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	originalExpiry := time.Now().Add(30 * 24 * time.Hour)
	err = s.db.CreateSession(token, s.user.ID, originalExpiry)
	s.Require().NoError(err)

	// Wait a moment to ensure timestamps differ
	time.Sleep(10 * time.Millisecond)

	// Get original session info
	originalInfo, err := s.db.ValidateSessionWithInfo(token)
	s.Require().NoError(err)

	// Renew the session
	newExpiry := time.Now().Add(60 * 24 * time.Hour)
	err = s.db.RenewSession(token, newExpiry)
	s.Require().NoError(err)

	// Get updated session info
	updatedInfo, err := s.db.ValidateSessionWithInfo(token)
	s.Require().NoError(err)

	// Verify last_activity was updated
	s.True(updatedInfo.LastActivity.After(originalInfo.LastActivity),
		"LastActivity should be updated after renewal")

	// Verify expires_at was updated
	s.True(updatedInfo.ExpiresAt.After(originalInfo.ExpiresAt),
		"ExpiresAt should be extended after renewal")
}

func (s *SessionTestSuite) TestDeleteSession() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = s.db.CreateSession(token, s.user.ID, expiresAt)
	s.Require().NoError(err)

	// Verify session exists
	_, err = s.db.ValidateSession(token)
	s.Require().NoError(err, "session should exist before deletion")

	// Delete session
	err = s.db.DeleteSession(token)
	s.Require().NoError(err)

	// Verify session is gone
	_, err = s.db.ValidateSession(token)
	s.Error(err, "expected error after deleting session")
}

// Test suite runner
func TestSessionSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
