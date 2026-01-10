package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ExpenseTestSuite provides a test suite for expense operations
type ExpenseTestSuite struct {
	suite.Suite
	db *DB
}

// SetupTest runs before each test
func (s *ExpenseTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	s.Require().NoError(err, "failed to create test database")
	s.db = db
}

// TearDownTest runs after each test
func (s *ExpenseTestSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *ExpenseTestSuite) TestCreateExpense() {
	err := s.db.CreateExpense(10.50, "Lunch", "food", time.Now())
	s.NoError(err)
}

func (s *ExpenseTestSuite) TestCreateMultipleExpensesWithSameTimestamp() {
	now := time.Now()

	// First insert should succeed
	err := s.db.CreateExpense(10.00, "First", "test", now)
	s.Require().NoError(err)

	// Second insert with same timestamp (currently no unique constraint, so this will succeed)
	err = s.db.CreateExpense(20.00, "Second", "test", now)
	s.NoError(err)
}

func (s *ExpenseTestSuite) TestListExpenses() {
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
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, baseTime.Add(exp.offset))
		s.Require().NoError(err, "failed to create expense: %s", exp.description)
	}

	result, err := s.db.ListExpenses()
	s.Require().NoError(err)
	s.Len(result, 3, "expected 3 expenses")

	// Check order (latest first). Snack was added last with latest timestamp
	if len(result) > 0 {
		s.InDelta(15.00, result[0].Amount, 0.001, "expected first expense to be Snack")
		s.Equal("Snack", result[0].Description)
	}
}

func (s *ExpenseTestSuite) TestListExpensesCurrentMonth() {
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
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, exp.date)
		s.Require().NoError(err, "failed to create expense: %s", exp.description)
	}

	// List expenses should only return current month
	expenses, err := s.db.ListExpenses()
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected only current month expenses")

	// Verify all returned expenses are from current month
	for _, exp := range expenses {
		s.Equal(now.Month(), exp.Date.Month(), "expense month mismatch")
		s.Equal(now.Year(), exp.Date.Year(), "expense year mismatch")
	}

	// Verify the expenses are the correct ones (ordered by date DESC)
	if s.Len(expenses, 2) {
		s.Equal("Current Month 2", expenses[0].Description)
		s.InDelta(150.00, expenses[0].Amount, 0.001)
		s.Equal("Current Month 1", expenses[1].Description)
		s.InDelta(100.00, expenses[1].Amount, 0.001)
	}
}

// Test suite runner
func TestExpenseSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTestSuite))
}
