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

// TestListExpensesAll covers the API's read path: every live row regardless
// of owner, ordered by (date DESC, id DESC). The app is a shared-household
// ledger, so the list query has no per-user filter — Alice and Bob each
// see every row, including legacy NULL-owner historical rows.
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
	// InsertExpense always sets a user, so we go around it. updated_at is
	// required post-delta-sync — scanExpenses reads it as a non-nullable
	// field, and a NULL would also leave the legacy row invisible to the
	// diff endpoint until something bumped it.
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
	// (date DESC, id DESC). The two Jan rows by user 1 sit on different
	// timestamps (jan and jan+2h) so Supermarket (jan+2h) outranks both
	// Bakery (jan, id=1) and the partner's Other-user row (jan, id=4),
	// which itself outranks Bakery via the id tiebreak.
	s.Equal("Feb shop", got[0].Description)
	s.Equal("Supermarket", got[1].Description)
	s.Equal("Other user row", got[2].Description)
	s.Equal("Bakery", got[3].Description)
	s.Equal("Legacy coffee", got[4].Description)
	s.Nil(got[4].UserID, "legacy row should still be unowned")
}

// TestUpdatePreservesAuthor pins the invariant that any authenticated user
// can edit any row but the original author (user_id) is never overwritten.
// This matters in the shared-household model: when Alice tweaks Bob's row,
// it stays Bob's row in the ledger — the audit trail of who entered each
// expense survives every downstream edit.
func (s *ExpenseTestSuite) TestUpdatePreservesAuthor() {
	bob := int64(2)
	created, err := s.db.InsertExpense(7.50, "Old lunch", "Eating Out",
		time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), bob)
	s.Require().NoError(err)
	s.Require().NotNil(created.UserID)
	s.Equal(bob, *created.UserID, "fresh row should be owned by its creator")

	// Alice (user 1) edits Bob's row. UpdateExpense takes no userID
	// arg — visibility and mutation are global in the shared model.
	created.Description = "Old lunch (recategorised by Alice)"
	created.Category = "Groceries"
	s.Require().NoError(s.db.UpdateExpense(created))

	reread, err := s.db.GetExpense(created.ID)
	s.Require().NoError(err)
	s.Equal("Old lunch (recategorised by Alice)", reread.Description)
	s.Equal("Groceries", reread.Category)
	s.Require().NotNil(reread.UserID, "user_id must not be cleared on update")
	s.Equal(bob, *reread.UserID, "author must remain Bob even after Alice's edit")

	// And the same row remains visible+editable in both directions —
	// Bob can still delete what was originally his even though Alice
	// touched it last.
	s.Require().NoError(s.db.DeleteExpense(created.ID))
	_, err = s.db.GetExpense(created.ID)
	s.ErrorContains(err, "no rows")
}

// TestLegacyNullOwnedRowsAreEditable keeps coverage on the NULL-owner path
// that previously needed special handling. With the shared-household model
// the rule is trivial — any row, NULL-owner or not, is reachable to every
// authenticated user — but legacy rows are still the most likely real
// production data, so we pin that they survive the read/update/delete cycle.
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

// TestInsertBumpsUpdatedAt pins the invariant that InsertExpense always
// returns a non-zero updated_at — the client's lastSyncAt depends on every
// write emitting a monotonic cursor, so silently leaving this field zero
// would let a freshly-created row disappear from the Feed diff until the
// next full reload.
func (s *ExpenseTestSuite) TestInsertBumpsUpdatedAt() {
	before := time.Now().Add(-time.Second).UTC()
	created, err := s.db.InsertExpense(12.50, "Coffee", "Eating Out", time.Now(), 1)
	s.Require().NoError(err)
	s.Require().NotNil(created)
	s.False(created.UpdatedAt.IsZero(), "InsertExpense must set updated_at")
	s.True(created.UpdatedAt.After(before), "updated_at should be wall-clock now()")
}

// TestUpdateBumpsUpdatedAt mirrors the insert invariant for edits — without
// advancing updated_at, a PATCH would be invisible to the delta endpoint
// and the Feed diff would miss the user's own change made in another tab.
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

// TestDeleteExpense_SoftDeletes covers the tombstone behavior end-to-end:
// the row survives in the table with deleted_at set, GetExpense / ListAll
// filter it out, and ListExpensesChangedSince reports the id in the
// deletedIds bucket — that combination is what lets the PWA drop the row
// from its React Query cache on the next Feed sync without a full reload.
func (s *ExpenseTestSuite) TestDeleteExpense_SoftDeletes() {
	created, err := s.db.InsertExpense(42, "Bye", "Other", time.Now(), 1)
	s.Require().NoError(err)

	// Baseline lastSyncAt for the delta query. time.Sleep so the
	// deleted_at timestamp is strictly greater than `since`.
	since := time.Now().UTC()
	time.Sleep(2 * time.Millisecond)

	s.Require().NoError(s.db.DeleteExpense(created.ID))

	// Row no longer shows up in read paths.
	_, err = s.db.GetExpense(created.ID)
	s.Require().ErrorContains(err, "no rows")
	all, err := s.db.ListExpensesAll()
	s.Require().NoError(err)
	s.Empty(all)

	// But the changes endpoint reports the id in the deletion bucket.
	upd, deletedIDs, err := s.db.ListExpensesChangedSince(since)
	s.Require().NoError(err)
	s.Empty(upd, "deleted rows must not also surface in the updated bucket")
	s.Require().Len(deletedIDs, 1)
	s.Equal(created.ID, deletedIDs[0])

	// The raw row still exists — tombstone, not purge.
	var count int
	err = s.db.conn.QueryRow(
		`SELECT COUNT(*) FROM expenses WHERE id = ? AND deleted_at IS NOT NULL`,
		created.ID,
	).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1, count, "expected the row to remain as a tombstone")
}

// TestPartialUniqueIndexAllowsReuseAfterSoftDelete documents the index
// change that had to ship alongside soft-delete: recording "coffee €4 on
// Monday", deleting it, then recording it again must succeed. The old
// non-partial unique index on (user_id, date, amount, description) would
// reject the second insert as a duplicate and permanently block the user
// from re-entering any expense they had ever deleted.
func (s *ExpenseTestSuite) TestPartialUniqueIndexAllowsReuseAfterSoftDelete() {
	date := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	first, err := s.db.InsertExpense(4.50, "Coffee", "Eating Out", date, 1)
	s.Require().NoError(err)

	s.Require().NoError(s.db.DeleteExpense(first.ID))

	// Re-inserting the exact same tuple must succeed — the partial unique
	// index excludes the tombstone.
	second, err := s.db.InsertExpense(4.50, "Coffee", "Eating Out", date, 1)
	s.Require().NoError(err, "second insert after tombstone should not collide")
	s.NotEqual(first.ID, second.ID, "fresh row should get a new id")

	// But a second identical live row still collides — the uniqueness
	// guarantee holds for non-deleted rows.
	_, err = s.db.InsertExpense(4.50, "Coffee", "Eating Out", date, 1)
	s.ErrorIs(err, ErrDuplicateExpense)
}

// TestListExpensesChangedSince_InsertsAndUpdates checks both buckets of
// the diff query: rows whose updated_at moved past `since` land in
// `updated`, whether they're brand new or pre-existing edits. This is
// what the client upserts into the React Query cache on every Feed mount.
func (s *ExpenseTestSuite) TestListExpensesChangedSince_InsertsAndUpdates() {
	// Row 1 created before the cutoff — should not appear in the diff.
	old, err := s.db.InsertExpense(10, "Old", "Other", time.Now(), 1)
	s.Require().NoError(err)

	cutoff := time.Now().UTC()
	time.Sleep(2 * time.Millisecond)

	// Row 2 inserted after the cutoff — appears as a new entry.
	fresh, err := s.db.InsertExpense(20, "Fresh", "Other", time.Now(), 1)
	s.Require().NoError(err)

	// Row 1 edited after the cutoff — now appears as an update.
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

// TestListExpensesChangedSince_BoundaryIsStrict protects the cursor
// semantics: comparing updated_at > since (not >=) means that passing
// back the serverTime we just returned does not re-emit any row. If this
// became >= the Feed would re-process the last-seen row on every sync.
func (s *ExpenseTestSuite) TestListExpensesChangedSince_BoundaryIsStrict() {
	created, err := s.db.InsertExpense(10, "Row", "Other", time.Now(), 1)
	s.Require().NoError(err)
	upd, deletedIDs, err := s.db.ListExpensesChangedSince(created.UpdatedAt)
	s.Require().NoError(err)
	s.Empty(upd, "passing back the exact updated_at must not re-emit the row")
	s.Empty(deletedIDs)
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

// Test suite runner
func TestExpenseSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTestSuite))
}
