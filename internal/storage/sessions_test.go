package storage

import (
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SessionTestSuite provides a test suite for session operations
type SessionTestSuite struct {
	suite.Suite
	db   *DB
	user *models.User
}

// SetupTest runs before each test
func (suite *SessionTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	require.NoError(suite.T(), err, "failed to create test database")
	suite.db = db

	// Create a test user
	password, err := auth.HashPassword("testpass")
	require.NoError(suite.T(), err, "failed to hash password")

	user, err := suite.db.CreateUser("testuser", password)
	require.NoError(suite.T(), err, "failed to create test user")
	suite.user = user
}

// TearDownTest runs after each test
func (suite *SessionTestSuite) TearDownTest() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *SessionTestSuite) TestCreateAndValidateSession() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, expiresAt)
	require.NoError(suite.T(), err)

	// Validate the session
	sessionUser, err := suite.db.ValidateSession(token)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", sessionUser.Username)
}

func (suite *SessionTestSuite) TestValidateSessionWithInfo() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, expiresAt)
	require.NoError(suite.T(), err)

	// Get session info
	info, err := suite.db.ValidateSessionWithInfo(token)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", info.User.Username)

	// Check that last_activity is recent
	timeSinceActivity := time.Since(info.LastActivity)
	assert.Less(suite.T(), timeSinceActivity, 5*time.Second, "LastActivity should be recent")
}

func (suite *SessionTestSuite) TestRenewSession() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	originalExpiry := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, originalExpiry)
	require.NoError(suite.T(), err)

	// Wait a moment to ensure timestamps differ
	time.Sleep(10 * time.Millisecond)

	// Get original session info
	originalInfo, err := suite.db.ValidateSessionWithInfo(token)
	require.NoError(suite.T(), err)

	// Renew the session
	newExpiry := time.Now().Add(60 * 24 * time.Hour)
	err = suite.db.RenewSession(token, newExpiry)
	require.NoError(suite.T(), err)

	// Get updated session info
	updatedInfo, err := suite.db.ValidateSessionWithInfo(token)
	require.NoError(suite.T(), err)

	// Verify last_activity was updated
	assert.True(suite.T(), updatedInfo.LastActivity.After(originalInfo.LastActivity),
		"LastActivity should be updated after renewal")

	// Verify expires_at was updated
	assert.True(suite.T(), updatedInfo.ExpiresAt.After(originalInfo.ExpiresAt),
		"ExpiresAt should be extended after renewal")
}

func (suite *SessionTestSuite) TestDeleteSession() {
	token, err := auth.GenerateSessionToken()
	require.NoError(suite.T(), err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = suite.db.CreateSession(token, suite.user.ID, expiresAt)
	require.NoError(suite.T(), err)

	// Verify session exists
	_, err = suite.db.ValidateSession(token)
	require.NoError(suite.T(), err, "session should exist before deletion")

	// Delete session
	err = suite.db.DeleteSession(token)
	require.NoError(suite.T(), err)

	// Verify session is gone
	_, err = suite.db.ValidateSession(token)
	assert.Error(suite.T(), err, "expected error after deleting session")
}

// Test suite runner
func TestSessionSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
