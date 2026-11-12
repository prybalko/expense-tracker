package storage

import (
	"testing"
	"time"

	"expense-tracker/internal/auth"
	"expense-tracker/internal/models"

	"github.com/stretchr/testify/suite"
)

type SessionTestSuite struct {
	suite.Suite
	db   *DB
	user *models.User
}

func (s *SessionTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	s.Require().NoError(err, "failed to create test database")
	s.db = db

	password, err := auth.HashPassword("testpass")
	s.Require().NoError(err, "failed to hash password")

	user, err := s.db.CreateUser("testuser", password)
	s.Require().NoError(err, "failed to create test user")
	s.user = user
}

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

	info, err := s.db.ValidateSessionWithInfo(token)
	s.Require().NoError(err)
	s.Equal("testuser", info.User.Username)

	timeSinceActivity := time.Since(info.LastActivity)
	s.Less(timeSinceActivity, 5*time.Second, "LastActivity should be recent")
}

func (s *SessionTestSuite) TestRenewSession() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	originalExpiry := time.Now().Add(30 * 24 * time.Hour)
	err = s.db.CreateSession(token, s.user.ID, originalExpiry)
	s.Require().NoError(err)

	time.Sleep(10 * time.Millisecond)

	originalInfo, err := s.db.ValidateSessionWithInfo(token)
	s.Require().NoError(err)

	newExpiry := time.Now().Add(60 * 24 * time.Hour)
	err = s.db.RenewSession(token, newExpiry)
	s.Require().NoError(err)

	updatedInfo, err := s.db.ValidateSessionWithInfo(token)
	s.Require().NoError(err)

	s.True(updatedInfo.LastActivity.After(originalInfo.LastActivity),
		"LastActivity should be updated after renewal")

	s.True(updatedInfo.ExpiresAt.After(originalInfo.ExpiresAt),
		"ExpiresAt should be extended after renewal")
}

func (s *SessionTestSuite) TestValidateExpiredSession() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	err = s.db.CreateSession(token, s.user.ID, time.Now().Add(-time.Hour))
	s.Require().NoError(err)

	_, err = s.db.ValidateSessionWithInfo(token)
	s.Require().ErrorIs(err, ErrSessionExpired)
}

func (s *SessionTestSuite) TestDeleteSession() {
	token, err := auth.GenerateSessionToken()
	s.Require().NoError(err)

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	err = s.db.CreateSession(token, s.user.ID, expiresAt)
	s.Require().NoError(err)

	_, err = s.db.ValidateSession(token)
	s.Require().NoError(err, "session should exist before deletion")

	err = s.db.DeleteSession(token)
	s.Require().NoError(err)

	_, err = s.db.ValidateSession(token)
	s.Error(err, "expected error after deleting session")
}

func TestSessionSuite(t *testing.T) {
	suite.Run(t, new(SessionTestSuite))
}
