package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ExpenseTestSuite provides a test suite for expense operations
type ExpenseTestSuite struct {
	suite.Suite
	db *DB
}

// SetupTest runs before each test
func (suite *ExpenseTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	require.NoError(suite.T(), err, "failed to create test database")
	suite.db = db
}

// TearDownTest runs after each test
func (suite *ExpenseTestSuite) TearDownTest() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *ExpenseTestSuite) TestCreateExpense() {
	err := suite.db.CreateExpense(10.50, "Lunch", "food", time.Now())
	assert.NoError(suite.T(), err)
}

func (suite *ExpenseTestSuite) TestCreateMultipleExpensesWithSameTimestamp() {
	now := time.Now()

	// First insert should succeed
	err := suite.db.CreateExpense(10.00, "First", "test", now)
	require.NoError(suite.T(), err)

	// Second insert with same timestamp (currently no unique constraint, so this will succeed)
	err = suite.db.CreateExpense(20.00, "Second", "test", now)
	assert.NoError(suite.T(), err)
}

func (suite *ExpenseTestSuite) TestListExpenses() {
	baseTime := time.Now().Add(time.Hour)

	// Create test expenses
	expenses := []struct {
		amount      float64
		description string
		category    string
		offset      time.Duration
	}{
		{20.00, "Bus", "transport", time.Minute},
		{5.00, "Coffee", "food", 2 * time.Minute},
		{15.00, "Snack", "food", 3 * time.Minute},
	}

	for _, exp := range expenses {
		err := suite.db.CreateExpense(exp.amount, exp.description, exp.category, baseTime.Add(exp.offset))
		require.NoError(suite.T(), err, "failed to create expense: %s", exp.description)
	}

	result, err := suite.db.ListExpenses()
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 3, "expected 3 expenses")

	// Check order (latest first). Snack was added last with latest timestamp
	if len(result) > 0 {
		assert.Equal(suite.T(), 15.00, result[0].Amount, "expected first expense to be Snack")
		assert.Equal(suite.T(), "Snack", result[0].Description)
	}
}

func (suite *ExpenseTestSuite) TestListExpensesCurrentMonth() {
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 15, 12, 0, 0, 0, now.Location())
	lastMonth := time.Date(now.Year(), now.Month()-1, 15, 12, 0, 0, 0, now.Location())
	twoMonthsAgo := time.Date(now.Year(), now.Month()-2, 15, 12, 0, 0, 0, now.Location())

	// Create expenses in different months
	testExpenses := []struct {
		amount      float64
		description string
		category    string
		date        time.Time
	}{
		{100.00, "Current Month 1", "food", currentMonth},
		{150.00, "Current Month 2", "transport", currentMonth.Add(24 * time.Hour)},
		{200.00, "Last Month", "food", lastMonth},
		{300.00, "Two Months Ago", "utilities", twoMonthsAgo},
	}

	for _, exp := range testExpenses {
		err := suite.db.CreateExpense(exp.amount, exp.description, exp.category, exp.date)
		require.NoError(suite.T(), err, "failed to create expense: %s", exp.description)
	}

	// List expenses should only return current month
	expenses, err := suite.db.ListExpenses()
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), expenses, 2, "expected only current month expenses")

	// Verify all returned expenses are from current month
	for _, exp := range expenses {
		assert.Equal(suite.T(), now.Month(), exp.Date.Month(), "expense month mismatch")
		assert.Equal(suite.T(), now.Year(), exp.Date.Year(), "expense year mismatch")
	}

	// Verify the expenses are the correct ones (ordered by date DESC)
	if assert.Len(suite.T(), expenses, 2) {
		assert.Equal(suite.T(), "Current Month 2", expenses[0].Description)
		assert.Equal(suite.T(), 150.00, expenses[0].Amount)
		assert.Equal(suite.T(), "Current Month 1", expenses[1].Description)
		assert.Equal(suite.T(), 100.00, expenses[1].Amount)
	}
}

// Test suite runner
func TestExpenseSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTestSuite))
}
