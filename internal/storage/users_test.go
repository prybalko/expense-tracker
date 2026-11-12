package storage

import (
	"testing"

	"expense-tracker/internal/auth"

	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	suite.Suite
	db *DB
}

func (s *UserTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	s.Require().NoError(err, "failed to create test database")
	s.db = db
}

func (s *UserTestSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *UserTestSuite) TestCreateUser() {
	passwordHash, err := auth.HashPassword("testpassword")
	s.Require().NoError(err)

	user, err := s.db.CreateUser("johndoe", passwordHash)
	s.Require().NoError(err)
	s.NotNil(user)
	s.Positive(user.ID)
	s.Equal("johndoe", user.Username)
	s.Equal(passwordHash, user.PasswordHash)
	s.False(user.CreatedAt.IsZero())
}

func (s *UserTestSuite) TestCreateUserDuplicateUsername() {
	passwordHash, err := auth.HashPassword("testpassword")
	s.Require().NoError(err)

	_, err = s.db.CreateUser("johndoe", passwordHash)
	s.Require().NoError(err)

	_, err = s.db.CreateUser("johndoe", passwordHash)
	s.Error(err, "expected error when creating user with duplicate username")
}

func (s *UserTestSuite) TestGetUserByID() {
	passwordHash, err := auth.HashPassword("testpassword")
	s.Require().NoError(err)

	createdUser, err := s.db.CreateUser("janedoe", passwordHash)
	s.Require().NoError(err)

	retrievedUser, err := s.db.GetUserByID(createdUser.ID)
	s.Require().NoError(err)
	s.Equal(createdUser.ID, retrievedUser.ID)
	s.Equal(createdUser.Username, retrievedUser.Username)
	s.Equal(createdUser.PasswordHash, retrievedUser.PasswordHash)
}

func (s *UserTestSuite) TestGetUserByIDNotFound() {
	_, err := s.db.GetUserByID(99999)
	s.Error(err, "expected error when getting non-existent user")
}

func (s *UserTestSuite) TestGetUserByUsername() {
	passwordHash, err := auth.HashPassword("testpassword")
	s.Require().NoError(err)

	createdUser, err := s.db.CreateUser("bobsmith", passwordHash)
	s.Require().NoError(err)

	retrievedUser, err := s.db.GetUserByUsername("bobsmith")
	s.Require().NoError(err)
	s.Equal(createdUser.ID, retrievedUser.ID)
	s.Equal(createdUser.Username, retrievedUser.Username)
	s.Equal(createdUser.PasswordHash, retrievedUser.PasswordHash)
}

func (s *UserTestSuite) TestGetUserByUsernameNotFound() {
	_, err := s.db.GetUserByUsername("nonexistent")
	s.Error(err, "expected error when getting non-existent user")
}

func (s *UserTestSuite) TestUserCount() {
	passwordHash, err := auth.HashPassword("testpassword")
	s.Require().NoError(err)

	count, err := s.db.UserCount()
	s.Require().NoError(err)
	s.Equal(0, count)

	_, err = s.db.CreateUser("user1", passwordHash)
	s.Require().NoError(err)

	count, err = s.db.UserCount()
	s.Require().NoError(err)
	s.Equal(1, count)

	_, err = s.db.CreateUser("user2", passwordHash)
	s.Require().NoError(err)

	count, err = s.db.UserCount()
	s.Require().NoError(err)
	s.Equal(2, count)

	_, err = s.db.CreateUser("user3", passwordHash)
	s.Require().NoError(err)

	count, err = s.db.UserCount()
	s.Require().NoError(err)
	s.Equal(3, count)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
