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
	err := s.db.CreateExpense(10.50, "Lunch", "food", time.Now(), 1)
	s.NoError(err)
}

func (s *ExpenseTestSuite) TestDeleteExpense() {
	// Create an expense
	err := s.db.CreateExpense(25.00, "Dinner", "food", time.Now(), 1)
	s.Require().NoError(err)

	// Get the expense to find its ID
	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 1)
	expenseID := expenses[0].ID

	// Delete the expense
	err = s.db.DeleteExpense(expenseID)
	s.Require().NoError(err)

	// Verify it's gone
	expenses, err = s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Empty(expenses, "expected no expenses after deletion")
}

func (s *ExpenseTestSuite) TestDeleteExpense_NonExistent() {
	// Deleting a non-existent expense should not error (no-op)
	err := s.db.DeleteExpense(99999)
	s.NoError(err, "deleting non-existent expense should not error")
}

func (s *ExpenseTestSuite) TestDeleteExpense_OnlyDeletesTarget() {
	baseTime := time.Now()

	// Create multiple expenses
	err := s.db.CreateExpense(10.00, "Coffee", "food", baseTime, 1)
	s.Require().NoError(err)
	err = s.db.CreateExpense(20.00, "Lunch", "food", baseTime.Add(time.Minute), 1)
	s.Require().NoError(err)
	err = s.db.CreateExpense(30.00, "Dinner", "food", baseTime.Add(2*time.Minute), 1)
	s.Require().NoError(err)

	// Get all expenses
	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 3)

	// Find the "Lunch" expense and delete it
	var lunchID int64
	for _, e := range expenses {
		if e.Description == "Lunch" {
			lunchID = e.ID
			break
		}
	}
	s.Require().NotZero(lunchID, "could not find Lunch expense")

	err = s.db.DeleteExpense(lunchID)
	s.Require().NoError(err)

	// Verify only 2 remain and Lunch is gone
	expenses, err = s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses after deletion")

	for _, e := range expenses {
		s.NotEqual("Lunch", e.Description, "Lunch expense should have been deleted")
	}
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
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, baseTime.Add(exp.offset), 1)
		s.Require().NoError(err, "failed to create expense: %s", exp.description)
	}

	result, err := s.db.ListExpenses(100, 0)
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
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, exp.date, 1)
		s.Require().NoError(err, "failed to create expense: %s", exp.description)
	}

	// List expenses should return all expenses (no longer filtered by month)
	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Len(expenses, 4, "expected all expenses")

	// Verify the expenses are ordered by date DESC
	if s.Len(expenses, 4) {
		s.Equal("Current Month 2", expenses[0].Description)
		s.InDelta(150.00, expenses[0].Amount, 0.001)
		s.Equal("Current Month 1", expenses[1].Description)
		s.InDelta(100.00, expenses[1].Amount, 0.001)
	}
}

func (s *ExpenseTestSuite) TestListExpensesPagination() {
	// Create 5 expenses
	baseTime := time.Now()
	for i := 1; i <= 5; i++ {
		err := s.db.CreateExpense(float64(i*10), "Expense "+string(rune('0'+i)), "food", baseTime.Add(time.Duration(i)*time.Minute), 1)
		s.Require().NoError(err)
	}

	// Test limit
	expenses, err := s.db.ListExpenses(2, 0)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses with limit=2")

	// Test offset
	expenses, err = s.db.ListExpenses(2, 2)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses with limit=2, offset=2")

	// Test offset beyond data
	expenses, err = s.db.ListExpenses(10, 10)
	s.Require().NoError(err)
	s.Empty(expenses, "expected 0 expenses with offset beyond data")
}

func (s *ExpenseTestSuite) TestGetExpensesByMonth() {
	// Create expenses in different months
	jan2026 := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	feb2026 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	dec2025 := time.Date(2025, 12, 15, 12, 0, 0, 0, time.UTC)

	testExpenses := []struct {
		amount      float64
		description string
		category    string
		date        time.Time
	}{
		{100.00, "January Expense 1", "groceries", jan2026},
		{150.00, "January Expense 2", "transport", jan2026.Add(24 * time.Hour)},
		{200.00, "February Expense", "eating out", feb2026},
		{300.00, "December Expense", "utilities", dec2025},
	}

	for _, exp := range testExpenses {
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, exp.date, 1)
		s.Require().NoError(err, "failed to create expense: %s", exp.description)
	}

	// Test getting January 2026 expenses
	janExpenses, err := s.db.GetExpensesByMonth(2026, 1)
	s.Require().NoError(err)
	s.Len(janExpenses, 2, "expected 2 expenses in January 2026")

	// Verify expenses are ordered by date DESC
	if s.Len(janExpenses, 2) {
		s.Equal("January Expense 2", janExpenses[0].Description)
		s.InDelta(150.00, janExpenses[0].Amount, 0.001)
		s.Equal("January Expense 1", janExpenses[1].Description)
		s.InDelta(100.00, janExpenses[1].Amount, 0.001)
	}

	// Test getting February 2026 expenses
	febExpenses, err := s.db.GetExpensesByMonth(2026, 2)
	s.Require().NoError(err)
	s.Len(febExpenses, 1, "expected 1 expense in February 2026")
	if s.Len(febExpenses, 1) {
		s.Equal("February Expense", febExpenses[0].Description)
		s.InDelta(200.00, febExpenses[0].Amount, 0.001)
	}

	// Test getting December 2025 expenses
	decExpenses, err := s.db.GetExpensesByMonth(2025, 12)
	s.Require().NoError(err)
	s.Len(decExpenses, 1, "expected 1 expense in December 2025")
	if s.Len(decExpenses, 1) {
		s.Equal("December Expense", decExpenses[0].Description)
		s.InDelta(300.00, decExpenses[0].Amount, 0.001)
	}

	// Test getting a month with no expenses
	novExpenses, err := s.db.GetExpensesByMonth(2025, 11)
	s.Require().NoError(err)
	s.Empty(novExpenses, "expected 0 expenses in November 2025")
}

func (s *ExpenseTestSuite) TestGetCategoryTotalsByMonth() {
	// Create expenses in different months and categories
	jan2026 := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	testExpenses := []struct {
		amount      float64
		description string
		category    string
		date        time.Time
	}{
		{100.00, "Groceries 1", "groceries", jan2026},
		{150.00, "Groceries 2", "groceries", jan2026.Add(time.Hour)},
		{200.00, "Bus", "transport", jan2026.Add(2 * time.Hour)},
		{50.00, "Taxi", "transport", jan2026.Add(3 * time.Hour)},
		{75.00, "Restaurant", "eating out", jan2026.Add(4 * time.Hour)},
		// February expenses (should not be included)
		{300.00, "Feb Groceries", "groceries", time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)},
	}

	for _, exp := range testExpenses {
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, exp.date, 1)
		s.Require().NoError(err, "failed to create expense: %s", exp.description)
	}

	// Test getting category totals for January 2026
	totals, err := s.db.GetCategoryTotalsByMonth(2026, 1)
	s.Require().NoError(err)
	s.Len(totals, 3, "expected 3 categories in January 2026")

	// Build a map for easier verification (since groceries and transport have same total)
	categoryMap := make(map[string]CategoryTotal)
	for _, ct := range totals {
		categoryMap[ct.Category] = ct
	}

	// Verify groceries: 100 + 150 = 250
	s.Contains(categoryMap, "groceries")
	s.InDelta(250.00, categoryMap["groceries"].Total, 0.001)
	s.Equal(2, categoryMap["groceries"].Count)

	// Verify transport: 200 + 50 = 250
	s.Contains(categoryMap, "transport")
	s.InDelta(250.00, categoryMap["transport"].Total, 0.001)
	s.Equal(2, categoryMap["transport"].Count)

	// Verify eating out: 75 (should be last since it has lowest total)
	s.Contains(categoryMap, "eating out")
	s.InDelta(75.00, categoryMap["eating out"].Total, 0.001)
	s.Equal(1, categoryMap["eating out"].Count)

	// The last item should be eating out (lowest total)
	s.Equal("eating out", totals[2].Category)

	// Test getting category totals for February 2026
	febTotals, err := s.db.GetCategoryTotalsByMonth(2026, 2)
	s.Require().NoError(err)
	s.Len(febTotals, 1, "expected 1 category in February 2026")
	if s.Len(febTotals, 1) {
		s.Equal("groceries", febTotals[0].Category)
		s.InDelta(300.00, febTotals[0].Total, 0.001)
		s.Equal(1, febTotals[0].Count)
	}

	// Test getting category totals for a month with no expenses
	novTotals, err := s.db.GetCategoryTotalsByMonth(2025, 11)
	s.Require().NoError(err)
	s.Empty(novTotals, "expected 0 categories in November 2025")
}

func (s *ExpenseTestSuite) TestGetCategoryTotalsByMonth_SingleCategory() {
	// Test when all expenses are in one category
	jan2026 := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	expenses := []struct {
		amount float64
		desc   string
	}{
		{10.00, "Coffee"},
		{20.00, "Lunch"},
		{30.00, "Dinner"},
	}

	for _, exp := range expenses {
		err := s.db.CreateExpense(exp.amount, exp.desc, "eating out", jan2026.Add(time.Hour), 1)
		jan2026 = jan2026.Add(time.Hour)
		s.Require().NoError(err)
	}

	totals, err := s.db.GetCategoryTotalsByMonth(2026, 1)
	s.Require().NoError(err)
	s.Len(totals, 1, "expected 1 category")
	if s.Len(totals, 1) {
		s.Equal("eating out", totals[0].Category)
		s.InDelta(60.00, totals[0].Total, 0.001)
		s.Equal(3, totals[0].Count)
	}
}

func (s *ExpenseTestSuite) TestGetExpensesByMonth_EdgeCases() {
	// Test month boundaries
	// Last day of January
	jan31 := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	// First day of February
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	err := s.db.CreateExpense(100.00, "End of January", "groceries", jan31, 1)
	s.Require().NoError(err)
	err = s.db.CreateExpense(200.00, "Start of February", "groceries", feb1, 1)
	s.Require().NoError(err)

	// Get January expenses
	janExpenses, err := s.db.GetExpensesByMonth(2026, 1)
	s.Require().NoError(err)
	s.Len(janExpenses, 1, "expected 1 expense in January")
	if s.Len(janExpenses, 1) {
		s.Equal("End of January", janExpenses[0].Description)
	}

	// Get February expenses
	febExpenses, err := s.db.GetExpensesByMonth(2026, 2)
	s.Require().NoError(err)
	s.Len(febExpenses, 1, "expected 1 expense in February")
	if s.Len(febExpenses, 1) {
		s.Equal("Start of February", febExpenses[0].Description)
	}
}

// Test suite runner
func TestExpenseSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTestSuite))
}
