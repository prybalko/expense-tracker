package storage

import (
	"testing"

	"expense-tracker/internal/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// UserTestSuite provides a test suite for user operations
type UserTestSuite struct {
	suite.Suite
	db *DB
}

// SetupTest runs before each test
func (suite *UserTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	require.NoError(suite.T(), err, "failed to create test database")
	suite.db = db
}

// TearDownTest runs after each test
func (suite *UserTestSuite) TearDownTest() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *UserTestSuite) TestCreateUser() {
	passwordHash, err := auth.HashPassword("testpassword")
	require.NoError(suite.T(), err)

	user, err := suite.db.CreateUser("johndoe", passwordHash)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Greater(suite.T(), user.ID, int64(0))
	assert.Equal(suite.T(), "johndoe", user.Username)
	assert.Equal(suite.T(), passwordHash, user.PasswordHash)
	assert.False(suite.T(), user.CreatedAt.IsZero())
}

func (suite *UserTestSuite) TestCreateUserDuplicateUsername() {
	passwordHash, err := auth.HashPassword("testpassword")
	require.NoError(suite.T(), err)

	// Create first user
	_, err = suite.db.CreateUser("johndoe", passwordHash)
	require.NoError(suite.T(), err)

	// Try to create second user with same username
	_, err = suite.db.CreateUser("johndoe", passwordHash)
	assert.Error(suite.T(), err, "expected error when creating user with duplicate username")
}

func (suite *UserTestSuite) TestGetUserByID() {
	passwordHash, err := auth.HashPassword("testpassword")
	require.NoError(suite.T(), err)

	// Create a user
	createdUser, err := suite.db.CreateUser("janedoe", passwordHash)
	require.NoError(suite.T(), err)

	// Retrieve the user by ID
	retrievedUser, err := suite.db.GetUserByID(createdUser.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), createdUser.ID, retrievedUser.ID)
	assert.Equal(suite.T(), createdUser.Username, retrievedUser.Username)
	assert.Equal(suite.T(), createdUser.PasswordHash, retrievedUser.PasswordHash)
}

func (suite *UserTestSuite) TestGetUserByIDNotFound() {
	// Try to get a user that doesn't exist
	_, err := suite.db.GetUserByID(99999)
	assert.Error(suite.T(), err, "expected error when getting non-existent user")
}

func (suite *UserTestSuite) TestGetUserByUsername() {
	passwordHash, err := auth.HashPassword("testpassword")
	require.NoError(suite.T(), err)

	// Create a user
	createdUser, err := suite.db.CreateUser("bobsmith", passwordHash)
	require.NoError(suite.T(), err)

	// Retrieve the user by username
	retrievedUser, err := suite.db.GetUserByUsername("bobsmith")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), createdUser.ID, retrievedUser.ID)
	assert.Equal(suite.T(), createdUser.Username, retrievedUser.Username)
	assert.Equal(suite.T(), createdUser.PasswordHash, retrievedUser.PasswordHash)
}

func (suite *UserTestSuite) TestGetUserByUsernameNotFound() {
	// Try to get a user that doesn't exist
	_, err := suite.db.GetUserByUsername("nonexistent")
	assert.Error(suite.T(), err, "expected error when getting non-existent user")
}

func (suite *UserTestSuite) TestUserCount() {
	passwordHash, err := auth.HashPassword("testpassword")
	require.NoError(suite.T(), err)

	// Initially should have 0 users
	count, err := suite.db.UserCount()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, count)

	// Create first user
	_, err = suite.db.CreateUser("user1", passwordHash)
	require.NoError(suite.T(), err)

	count, err = suite.db.UserCount()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, count)

	// Create second user
	_, err = suite.db.CreateUser("user2", passwordHash)
	require.NoError(suite.T(), err)

	count, err = suite.db.UserCount()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, count)

	// Create third user
	_, err = suite.db.CreateUser("user3", passwordHash)
	require.NoError(suite.T(), err)

	count, err = suite.db.UserCount()
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, count)
}

// Test suite runner
func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}
