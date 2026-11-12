package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ExpenseTestSuite struct {
	suite.Suite
	db *DB
}

func (s *ExpenseTestSuite) SetupTest() {
	db, err := NewDB(":memory:")
	s.Require().NoError(err, "failed to create test database")
	s.db = db
}

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
	err := s.db.CreateExpense(25.00, "Dinner", "food", time.Now(), 1)
	s.Require().NoError(err)

	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 1)
	expenseID := expenses[0].ID

	err = s.db.DeleteExpense(expenseID)
	s.Require().NoError(err)

	expenses, err = s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Empty(expenses, "expected no expenses after deletion")
}

func (s *ExpenseTestSuite) TestDeleteExpense_NonExistent() {
	err := s.db.DeleteExpense(99999)
	s.NoError(err, "deleting non-existent expense should not error")
}

func (s *ExpenseTestSuite) TestDeleteExpense_OnlyDeletesTarget() {
	baseTime := time.Now()

	err := s.db.CreateExpense(10.00, "Coffee", "food", baseTime, 1)
	s.Require().NoError(err)
	err = s.db.CreateExpense(20.00, "Lunch", "food", baseTime.Add(time.Minute), 1)
	s.Require().NoError(err)
	err = s.db.CreateExpense(30.00, "Dinner", "food", baseTime.Add(2*time.Minute), 1)
	s.Require().NoError(err)

	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 3)

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

	expenses, err = s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses after deletion")

	for _, e := range expenses {
		s.NotEqual("Lunch", e.Description, "Lunch expense should have been deleted")
	}
}

func (s *ExpenseTestSuite) TestListExpenses() {
	baseTime := time.Now().Add(time.Hour)

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

	if len(result) > 0 {
		s.InDelta(15.00, result[0].Amount, 0.001, "expected first expense to be Snack")
		s.Equal("Snack", result[0].Description)
	}
}

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

	// Raw insert because InsertExpense always sets user_id; updated_at must be non-NULL for scanExpenses.
	_, err := s.db.conn.Exec(
		`INSERT INTO expenses (amount, description, category, date, user_id, updated_at)
		 VALUES (?, ?, ?, ?, NULL, ?)`,
		5.00, "Legacy coffee", "Eating Out",
		time.Date(2018, 6, 15, 9, 0, 0, 0, time.UTC),
		time.Date(2018, 6, 15, 9, 0, 0, 0, time.UTC),
	)
	s.Require().NoError(err)

	got, err := s.db.ListExpensesAll()
	s.Require().NoError(err)
	s.Require().Len(got, 5, "expected every live row across both users plus the legacy NULL row")
	// Other user row (jan, id=4) outranks Bakery (jan, id=1) via the id tiebreak.
	s.Equal("Feb shop", got[0].Description)
	s.Equal("Supermarket", got[1].Description)
	s.Equal("Other user row", got[2].Description)
	s.Equal("Bakery", got[3].Description)
	s.Equal("Legacy coffee", got[4].Description)
	s.Nil(got[4].UserID, "legacy row should still be unowned")
}

func (s *ExpenseTestSuite) TestUpdatePreservesAuthor() {
	bob := int64(2)
	created, err := s.db.InsertExpense(7.50, "Old lunch", "Eating Out",
		time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), bob)
	s.Require().NoError(err)
	s.Require().NotNil(created.UserID)
	s.Equal(bob, *created.UserID, "fresh row should be owned by its creator")

	created.Description = "Old lunch (recategorised by Alice)"
	created.Category = "Groceries"
	s.Require().NoError(s.db.UpdateExpense(created))

	reread, err := s.db.GetExpense(created.ID)
	s.Require().NoError(err)
	s.Equal("Old lunch (recategorised by Alice)", reread.Description)
	s.Equal("Groceries", reread.Category)
	s.Require().NotNil(reread.UserID, "user_id must not be cleared on update")
	s.Equal(bob, *reread.UserID, "author must remain Bob even after Alice's edit")

	s.Require().NoError(s.db.DeleteExpense(created.ID))
	_, err = s.db.GetExpense(created.ID)
	s.ErrorContains(err, "no rows")
}

func (s *ExpenseTestSuite) TestLegacyNullOwnedRowsAreEditable() {
	res, err := s.db.conn.Exec(
		`INSERT INTO expenses (amount, description, category, date, user_id, updated_at)
		 VALUES (?, ?, ?, ?, NULL, ?)`,
		7.50, "Old lunch", "Eating Out",
		time.Date(2017, 4, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2017, 4, 1, 12, 0, 0, 0, time.UTC),
	)
	s.Require().NoError(err)
	id, err := res.LastInsertId()
	s.Require().NoError(err)

	got, err := s.db.GetExpense(id)
	s.Require().NoError(err)
	s.Equal("Old lunch", got.Description)
	s.Nil(got.UserID)

	got.Description = "Old lunch (recategorised)"
	got.Category = "Groceries"
	s.Require().NoError(s.db.UpdateExpense(got))
	reread, err := s.db.GetExpense(id)
	s.Require().NoError(err)
	s.Equal("Old lunch (recategorised)", reread.Description)
	s.Equal("Groceries", reread.Category)
	s.Nil(reread.UserID, "update must not silently stamp a user onto a NULL row")

	s.Require().NoError(s.db.DeleteExpense(id))
	_, err = s.db.GetExpense(id)
	s.ErrorContains(err, "no rows")
}

func (s *ExpenseTestSuite) TestInsertBumpsUpdatedAt() {
	before := time.Now().Add(-time.Second).UTC()
	created, err := s.db.InsertExpense(12.50, "Coffee", "Eating Out", time.Now(), 1)
	s.Require().NoError(err)
	s.Require().NotNil(created)
	s.False(created.UpdatedAt.IsZero(), "InsertExpense must set updated_at")
	s.True(created.UpdatedAt.After(before), "updated_at should be wall-clock now()")
}

func (s *ExpenseTestSuite) TestUpdateBumpsUpdatedAt() {
	created, err := s.db.InsertExpense(10, "Original", "Other", time.Now(), 1)
	s.Require().NoError(err)
	originalUpdated := created.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	created.Description = "Edited"
	s.Require().NoError(s.db.UpdateExpense(created))
	s.True(created.UpdatedAt.After(originalUpdated),
		"UpdateExpense must advance updated_at: %v not after %v",
		created.UpdatedAt, originalUpdated)
}

func (s *ExpenseTestSuite) TestDeleteExpense_SoftDeletes() {
	created, err := s.db.InsertExpense(42, "Bye", "Other", time.Now(), 1)
	s.Require().NoError(err)

	// Sleep so deleted_at is strictly after since.
	since := time.Now().UTC()
	time.Sleep(2 * time.Millisecond)

	s.Require().NoError(s.db.DeleteExpense(created.ID))

	_, err = s.db.GetExpense(created.ID)
	s.Require().ErrorContains(err, "no rows")
	all, err := s.db.ListExpensesAll()
	s.Require().NoError(err)
	s.Empty(all)

	upd, deletedIDs, err := s.db.ListExpensesChangedSince(since)
	s.Require().NoError(err)
	s.Empty(upd, "deleted rows must not also surface in the updated bucket")
	s.Require().Len(deletedIDs, 1)
	s.Equal(created.ID, deletedIDs[0])

	var count int
	err = s.db.conn.QueryRow(
		`SELECT COUNT(*) FROM expenses WHERE id = ? AND deleted_at IS NOT NULL`,
		created.ID,
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count, "expected the row to remain as a tombstone")
}

func (s *ExpenseTestSuite) TestPartialUniqueIndexAllowsReuseAfterSoftDelete() {
	date := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	first, err := s.db.InsertExpense(4.50, "Coffee", "Eating Out", date, 1)
	s.Require().NoError(err)

	s.Require().NoError(s.db.DeleteExpense(first.ID))

	second, err := s.db.InsertExpense(4.50, "Coffee", "Eating Out", date, 1)
	s.Require().NoError(err, "second insert after tombstone should not collide")
	s.NotEqual(first.ID, second.ID, "fresh row should get a new id")

	_, err = s.db.InsertExpense(4.50, "Coffee", "Eating Out", date, 1)
	s.ErrorIs(err, ErrDuplicateExpense)
}

func (s *ExpenseTestSuite) TestListExpensesChangedSince_InsertsAndUpdates() {
	old, err := s.db.InsertExpense(10, "Old", "Other", time.Now(), 1)
	s.Require().NoError(err)

	cutoff := time.Now().UTC()
	time.Sleep(2 * time.Millisecond)

	fresh, err := s.db.InsertExpense(20, "Fresh", "Other", time.Now(), 1)
	s.Require().NoError(err)

	old.Description = "Old edited"
	s.Require().NoError(s.db.UpdateExpense(old))

	upd, deletedIDs, err := s.db.ListExpensesChangedSince(cutoff)
	s.Require().NoError(err)
	s.Empty(deletedIDs)
	s.Require().Len(upd, 2)

	descs := map[string]bool{}
	for _, e := range upd {
		descs[e.Description] = true
	}
	s.True(descs["Fresh"], "expected the newly-inserted row")
	s.True(descs["Old edited"], "expected the edited row with the new description")
	_ = fresh
}

func (s *ExpenseTestSuite) TestListExpensesChangedSince_BoundaryIsStrict() {
	created, err := s.db.InsertExpense(10, "Row", "Other", time.Now(), 1)
	s.Require().NoError(err)
	upd, deletedIDs, err := s.db.ListExpensesChangedSince(created.UpdatedAt)
	s.Require().NoError(err)
	s.Empty(upd, "passing back the exact updated_at must not re-emit the row")
	s.Empty(deletedIDs)
}

func (s *ExpenseTestSuite) TestListExpensesPagination() {
	baseTime := time.Now()
	for i := 1; i <= 5; i++ {
		err := s.db.CreateExpense(float64(i*10), "Expense "+string(rune('0'+i)), "food", baseTime.Add(time.Duration(i)*time.Minute), 1)
		s.Require().NoError(err)
	}

	expenses, err := s.db.ListExpenses(2, 0)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses with limit=2")

	expenses, err = s.db.ListExpenses(2, 2)
	s.Require().NoError(err)
	s.Len(expenses, 2, "expected 2 expenses with limit=2, offset=2")

	expenses, err = s.db.ListExpenses(10, 10)
	s.Require().NoError(err)
	s.Empty(expenses, "expected 0 expenses with offset beyond data")
}

func TestExpenseSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTestSuite))
}
