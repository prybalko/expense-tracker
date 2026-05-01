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
	expenses, err := s.db.ListExpenses(1, 100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 1)
	expenseID := expenses[0].ID

	// Delete the expense
	err = s.db.DeleteExpense(1, expenseID)
	s.Require().NoError(err)

	// Verify it's gone
	expenses, err = s.db.ListExpenses(1, 100, 0)
	s.Require().NoError(err)
	s.Empty(expenses, "expected no expenses after deletion")
}

func (s *ExpenseTestSuite) TestDeleteExpense_NonExistent() {
	// Deleting a non-existent expense should not error (no-op)
	err := s.db.DeleteExpense(1, 99999)
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
	expenses, err := s.db.ListExpenses(1, 100, 0)
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

	err = s.db.DeleteExpense(1, lunchID)
	s.Require().NoError(err)

	// Verify only 2 remain and Lunch is gone
	expenses, err = s.db.ListExpenses(1, 100, 0)
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

	result, err := s.db.ListExpenses(1, 100, 0)
	s.Require().NoError(err)
	s.Len(result, 3, "expected 3 expenses")

	// Check order (latest first). Snack was added last with latest timestamp
	if len(result) > 0 {
		s.InDelta(15.00, result[0].Amount, 0.001, "expected first expense to be Snack")
		s.Equal("Snack", result[0].Description)
	}
}

// TestListExpensesAll covers the API's read path: every row visible to the
// caller, ordered by (date DESC, id DESC). Visible means owned by the caller
// OR unowned (user_id IS NULL — the legacy bucket from before per-user
// scoping). The client caches this array under one query key and derives
// every screen's view locally — leaking another user's owned rows here
// would mean they show up on the wrong account's Insights screen.
func (s *ExpenseTestSuite) TestListExpensesAll() {
	jan := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	seed := []struct {
		amount   float64
		desc     string
		category string
		date     time.Time
		user     int64
	}{
		{10.00, "Bakery", "Groceries", jan, 1},
		{25.00, "Supermarket", "Groceries", jan.Add(2 * time.Hour), 1},
		{40.00, "Feb shop", "Groceries", feb, 1},
		{99.00, "Other user row", "Groceries", jan, 2},
	}
	for _, e := range seed {
		_, err := s.db.InsertExpense(e.amount, e.desc, e.category, e.date, e.user)
		s.Require().NoError(err)
	}

	// Insert a pre-multi-user historical row directly (NULL user_id).
	// InsertExpense always sets a user, so we go around it.
	_, err := s.db.conn.Exec(
		`INSERT INTO expenses (amount, description, category, date, user_id)
		 VALUES (?, ?, ?, ?, NULL)`,
		5.00, "Legacy coffee", "Eating Out",
		time.Date(2018, 6, 15, 9, 0, 0, 0, time.UTC),
	)
	s.Require().NoError(err)

	got, err := s.db.ListExpensesAll(1)
	s.Require().NoError(err)
	s.Require().Len(got, 4, "expected 3 owned + 1 legacy NULL row for user 1")
	// (date DESC, id DESC) — Feb row first, then Jan rows in insert order
	// reversed (newer id first when dates tie), then the 2018 legacy row last.
	s.Equal("Feb shop", got[0].Description)
	s.Equal("Supermarket", got[1].Description)
	s.Equal("Bakery", got[2].Description)
	s.Equal("Legacy coffee", got[3].Description)
	s.Nil(got[3].UserID, "legacy row should still be unowned")

	// User 2 sees their own row + the same legacy NULL row.
	got2, err := s.db.ListExpensesAll(2)
	s.Require().NoError(err)
	s.Require().Len(got2, 2, "expected 1 owned + 1 legacy NULL row for user 2")
	s.Equal("Other user row", got2[0].Description)
	s.Equal("Legacy coffee", got2[1].Description)

	// An unknown user still sees the unowned legacy row (it's shared);
	// they just have nothing of their own.
	other, err := s.db.ListExpensesAll(999)
	s.Require().NoError(err)
	s.Require().Len(other, 1)
	s.Equal("Legacy coffee", other[0].Description)
}

// TestNullOwnedRowsAreEditableByAnyUser pins the policy that pre-multi-user
// historical rows can be updated and deleted by whoever's logged in — the
// alternative (immutable shared rows) would leave years of legacy data
// permanently miscategorised with no way to fix it from the UI.
func (s *ExpenseTestSuite) TestNullOwnedRowsAreEditableByAnyUser() {
	res, err := s.db.conn.Exec(
		`INSERT INTO expenses (amount, description, category, date, user_id)
		 VALUES (?, ?, ?, ?, NULL)`,
		7.50, "Old lunch", "Eating Out",
		time.Date(2017, 4, 1, 12, 0, 0, 0, time.UTC),
	)
	s.Require().NoError(err)
	id, err := res.LastInsertId()
	s.Require().NoError(err)

	// User 1 reads it.
	got, err := s.db.GetExpense(1, id)
	s.Require().NoError(err)
	s.Equal("Old lunch", got.Description)
	s.Nil(got.UserID)

	// User 1 updates it; the row stays unowned afterward.
	got.Description = "Old lunch (recategorised)"
	got.Category = "Groceries"
	s.Require().NoError(s.db.UpdateExpense(1, got))
	reread, err := s.db.GetExpense(2, id)
	s.Require().NoError(err, "user 2 should still see the legacy row")
	s.Equal("Old lunch (recategorised)", reread.Description)
	s.Equal("Groceries", reread.Category)
	s.Nil(reread.UserID, "update must not silently claim a shared row")

	// User 2 deletes it; gone for everyone.
	s.Require().NoError(s.db.DeleteExpense(2, id))
	_, err = s.db.GetExpense(1, id)
	s.ErrorContains(err, "no rows")
}

func (s *ExpenseTestSuite) TestListExpensesPagination() {
	// Create 5 expenses
	baseTime := time.Now()
	for i := 1; i <= 5; i++ {
		err := s.db.CreateExpense(float64(i*10), "Expense "+string(rune('0'+i)), "food", baseTime.Add(time.Duration(i)*time.Minute), 1)
		s.Require().NoError(err)
	}

	// Test limit
	expenses, err := s.db.ListExpenses(1, 2, 0)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses with limit=2")

	// Test offset
	expenses, err = s.db.ListExpenses(1, 2, 2)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses with limit=2, offset=2")

	// Test offset beyond data
	expenses, err = s.db.ListExpenses(1, 10, 10)
	s.Require().NoError(err)
	s.Empty(expenses, "expected 0 expenses with offset beyond data")
}

// Test suite runner
func TestExpenseSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTestSuite))
}
