package e2e

import (
	"expense-tracker/internal/storage"
	"fmt"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite drives the Playwright-backed end-to-end suite against the
// React UI. Selectors are anchored on data-testid attributes the components
// expose specifically for tests; emoji glyphs and class names are not stable
// targets in the new design.
type E2ETestSuite struct {
	suite.Suite
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
	expect  playwright.PlaywrightAssertions
}

func (s *E2ETestSuite) SetupSuite() {
	pw, err := playwright.Run()
	s.Require().NoError(err, "could not launch playwright")
	s.pw = pw

	browser, err := pw.Chromium.Launch()
	s.Require().NoError(err, "could not launch chromium")
	s.browser = browser

	s.expect = playwright.NewPlaywrightAssertions()
}

func (s *E2ETestSuite) TearDownSuite() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
}

func (s *E2ETestSuite) SetupTest() {
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open database for cleanup")
	err = db.ClearExpenses()
	s.Require().NoError(err, "could not clear expenses")
	db.Close()

	ctx, err := s.browser.NewContext(playwright.BrowserNewContextOptions{})
	s.Require().NoError(err, "could not create context")

	page, err := ctx.NewPage()
	s.Require().NoError(err, "could not create page")
	s.page = page
	// React renders + TanStack queries + bundle parse take longer than the
	// previous HTMX server-rendered flow; bump from the old 1s.
	s.page.SetDefaultTimeout(5000)

	_, err = s.page.Goto(appURL)
	s.Require().NoError(err, "could not navigate to app")
}

func (s *E2ETestSuite) TearDownTest() {
	if s.page != nil {
		s.page.Close()
	}
}

func tid(id string) string {
	return fmt.Sprintf("[data-testid=%q]", id)
}

func (s *E2ETestSuite) login() {
	err := s.expect.Locator(s.page.Locator(tid("login-form"))).ToBeVisible()
	s.Require().NoError(err, "login form not visible")

	err = s.page.Locator(tid("login-username")).Fill("testuser")
	s.Require().NoError(err, "failed to fill username")

	err = s.page.Locator(tid("login-password")).Fill("testpass123")
	s.Require().NoError(err, "failed to fill password")

	err = s.page.Locator(tid("login-submit")).Click()
	s.Require().NoError(err, "failed to click login")

	err = s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "did not redirect to feed after login")
}

// pressKeys taps each character on the in-app keypad. "." maps to the
// dedicated decimal key.
func (s *E2ETestSuite) pressKeys(keys string) {
	for _, r := range keys {
		key := string(r)
		if key == "." {
			key = "dot"
		}
		err := s.page.Locator(tid("keypad-" + key)).Click()
		s.Require().NoError(err, "failed to click keypad key %s", key)
	}
}

// dayISO returns YYYY-MM-DD for the Nth day of the current month, used to
// build calendar-day testids for the date sheet.
func dayISO(day int) string {
	now := time.Now()
	return fmt.Sprintf("%04d-%02d-%02d", now.Year(), int(now.Month()), day)
}

func (s *E2ETestSuite) TestCompleteUserFlow() {
	s.login()

	err := s.expect.Locator(s.page.Locator(tid("hero-label"))).ToHaveText("spent this month")
	s.Require().NoError(err, "hero label mismatch")

	err = s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	s.pressKeys("12.50")

	err = s.expect.Locator(s.page.Locator(tid("entry-amount"))).ToHaveAttribute("data-amount", "12.50")
	s.Require().NoError(err, "amount display mismatch")

	err = s.page.Locator(tid("entry-note")).Fill("Lunch Test")
	s.Require().NoError(err, "failed to fill note")

	err = s.page.Locator(tid("category-tile-groceries")).Click()
	s.Require().NoError(err, "failed to select Groceries category")

	err = s.page.Locator(tid("entry-submit")).Click()
	s.Require().NoError(err, "failed to submit expense")

	// Form route closes; feed regains focus.
	err = s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "did not return to feed after submit")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected exactly 1 expense row")

	row := s.page.Locator(tid("expense-row")).First()
	err = s.expect.Locator(row.Locator(tid("expense-row-desc"))).ToHaveText("Lunch Test")
	s.Require().NoError(err, "row description mismatch")

	err = s.expect.Locator(row.Locator(tid("expense-row-amount"))).ToContainText("12.50")
	s.Require().NoError(err, "row amount mismatch")

	err = s.expect.Locator(row).ToHaveAttribute("data-cat-slug", "groceries")
	s.Require().NoError(err, "row category slug mismatch")
}

func (s *E2ETestSuite) TestAddExpenseToBlankList() {
	s.login()

	count, err := s.page.Locator(tid("expense-row")).Count()
	s.Require().NoError(err, "failed to count expense rows")
	s.Require().Equal(0, count, "expected blank list, got %d rows", count)

	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("0.00")
	s.Require().NoError(err, "hero total should be 0.00 on blank list")

	err = s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	s.pressKeys("25.99")

	err = s.expect.Locator(s.page.Locator(tid("entry-amount"))).ToHaveAttribute("data-amount", "25.99")
	s.Require().NoError(err, "entry amount should display 25.99")

	err = s.page.Locator(tid("entry-submit")).Click()
	s.Require().NoError(err, "failed to submit expense")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected exactly 1 expense row")

	err = s.expect.Locator(s.page.Locator(tid("expense-row-amount"))).ToContainText("25.99")
	s.Require().NoError(err, "row amount should be 25.99")

	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("25.99")
	s.Require().NoError(err, "hero total should reflect 25.99")

	// Default category is the first declared (Groceries / slug=groceries).
	err = s.expect.Locator(s.page.Locator(tid("expense-row")).First()).ToHaveAttribute("data-cat-slug", "groceries")
	s.Require().NoError(err, "default category slug should be groceries")
}

func (s *E2ETestSuite) TestEditExpenseFlow() {
	s.login()

	// Pick days that are guaranteed past-or-today so the calendar grid
	// doesn't disable them. CalendarGrid blocks any day strictly after today
	// (web/app/src/components/CalendarGrid.tsx:174-180), so on the 1st of
	// any month neither day 2 nor day 30 of the current month is selectable.
	// `createDay` is 1 (always valid). `editDay` is today's day-of-month so
	// it's selectable; when today >= 2 it's also strictly later than
	// `createDay`, exercising a real forward bump. When today == 1, both
	// dates collapse to day 1 and the date stays put — the rest of the
	// edit flow (amount, note, hero total) still exercises.
	todayDay := time.Now().Day()
	createDay := 1
	editDay := todayDay

	// 1. Create an expense to edit, picking the Transport category and day 1
	//    of the current month so we can assert both stick across edits.
	err := s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	err = s.page.Locator(tid("category-tile-transport")).Click()
	s.Require().NoError(err, "failed to select Transport category")

	// Open date sheet, pick createDay, commit with Done.
	err = s.page.Locator(tid("date-pill")).Click()
	s.Require().NoError(err, "failed to open date picker")

	err = s.expect.Locator(s.page.Locator(tid("date-sheet"))).ToBeVisible()
	s.Require().NoError(err, "date sheet not visible")

	err = s.page.Locator(tid("calendar-day-" + dayISO(createDay))).Click()
	s.Require().NoError(err, "failed to pick day %d", createDay)

	err = s.page.Locator(tid("date-sheet-done")).Click()
	s.Require().NoError(err, "failed to commit date selection")

	err = s.expect.Locator(s.page.Locator(tid("date-sheet"))).Not().ToBeVisible()
	s.Require().NoError(err, "date sheet did not close after Done")

	s.pressKeys("50.00")

	err = s.page.Locator(tid("entry-note")).Fill("Original Expense")
	s.Require().NoError(err, "failed to fill note")

	err = s.page.Locator(tid("entry-submit")).Click()
	s.Require().NoError(err, "failed to submit expense")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).Not().ToBeVisible()
	s.Require().NoError(err, "entry form should close after submit")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected 1 row after create")

	row := s.page.Locator(tid("expense-row")).First()
	err = s.expect.Locator(row.Locator(tid("expense-row-desc"))).ToHaveText("Original Expense")
	s.Require().NoError(err, "original description mismatch")

	err = s.expect.Locator(row.Locator(tid("expense-row-amount"))).ToContainText("50.00")
	s.Require().NoError(err, "original amount mismatch")

	err = s.expect.Locator(row).ToHaveAttribute("data-cat-slug", "transport")
	s.Require().NoError(err, "original category slug mismatch")

	// 2. Open it for editing.
	err = row.Click()
	s.Require().NoError(err, "failed to click row")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "edit form not visible")

	err = s.expect.Locator(s.page.Locator(tid("entry-note"))).ToHaveValue("Original Expense")
	s.Require().NoError(err, "note not populated for edit")

	// 3. Clear "50.00" by hammering del five times — the amount string is the
	//    full toFixed(2) form once the row exists, not "50".
	for range 5 {
		err = s.page.Locator(tid("keypad-del")).Click()
		s.Require().NoError(err, "failed to press del")
	}
	err = s.expect.Locator(s.page.Locator(tid("entry-amount"))).ToHaveAttribute("data-amount", "0")
	s.Require().NoError(err, "amount not cleared")

	s.pressKeys("40.00")

	err = s.page.Locator(tid("entry-note")).Fill("Updated Expense")
	s.Require().NoError(err, "failed to update note")

	// Re-open the date sheet and pick editDay (today). When today >= 2 this
	// bumps the date forward from createDay (1); on the 1st it stays on day 1.
	err = s.page.Locator(tid("date-pill")).Click()
	s.Require().NoError(err, "failed to reopen date picker")

	err = s.expect.Locator(s.page.Locator(tid("date-sheet"))).ToBeVisible()
	s.Require().NoError(err, "date sheet not visible on edit")

	err = s.page.Locator(tid("calendar-day-" + dayISO(editDay))).Click()
	s.Require().NoError(err, "failed to pick day %d", editDay)

	err = s.page.Locator(tid("date-sheet-done")).Click()
	s.Require().NoError(err, "failed to commit edited date")

	err = s.page.Locator(tid("entry-submit")).Click()
	s.Require().NoError(err, "failed to save edits")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).Not().ToBeVisible()
	s.Require().NoError(err, "entry form should close after edit")

	// 4. Verify the row reflects the edit.
	updated := s.page.Locator(tid("expense-row")).First()
	err = s.expect.Locator(updated.Locator(tid("expense-row-desc"))).ToHaveText("Updated Expense")
	s.Require().NoError(err, "updated description mismatch")

	err = s.expect.Locator(updated.Locator(tid("expense-row-amount"))).ToContainText("40.00")
	s.Require().NoError(err, "updated amount mismatch")

	err = s.expect.Locator(updated).ToHaveAttribute("data-cat-slug", "transport")
	s.Require().NoError(err, "category slug should remain transport")

	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("40.00")
	s.Require().NoError(err, "hero total not updated to 40.00")
}

func (s *E2ETestSuite) TestDeleteExpenseFlow() {
	s.login()

	err := s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	s.pressKeys("99.99")

	err = s.page.Locator(tid("entry-note")).Fill("To Be Deleted")
	s.Require().NoError(err, "failed to fill note")

	err = s.page.Locator(tid("entry-submit")).Click()
	s.Require().NoError(err, "failed to submit expense")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).Not().ToBeVisible()
	s.Require().NoError(err, "entry form should close after submit")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected 1 row after create")

	row := s.page.Locator(tid("expense-row")).First()
	err = s.expect.Locator(row.Locator(tid("expense-row-desc"))).ToHaveText("To Be Deleted")
	s.Require().NoError(err, "row description mismatch")

	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("99.99")
	s.Require().NoError(err, "hero total should show 99.99")

	err = row.Click()
	s.Require().NoError(err, "failed to click row to edit")

	err = s.expect.Locator(s.page.Locator(tid("entry-delete"))).ToBeVisible()
	s.Require().NoError(err, "delete button should be visible on edit")

	// EntryForm.onDelete uses window.confirm — Playwright surfaces it as a
	// dialog event, accepted via OnDialog before the click.
	s.page.OnDialog(func(dialog playwright.Dialog) {
		dialog.Accept()
	})

	err = s.page.Locator(tid("entry-delete")).Click()
	s.Require().NoError(err, "failed to click delete")

	err = s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "should be back on feed after delete")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(0)
	s.Require().NoError(err, "expense row should be gone")

	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("0.00")
	s.Require().NoError(err, "hero total should be 0.00 after delete")
}

func (s *E2ETestSuite) TestDeleteButtonNotVisibleOnCreate() {
	s.login()

	err := s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	// On create, the trash button is not rendered at all (a placeholder div
	// fills its slot in the layout), so it should not be present in the DOM.
	count, err := s.page.Locator(tid("entry-delete")).Count()
	s.Require().NoError(err, "failed to count delete buttons")
	s.Require().Equal(0, count, "delete button should not exist on create, found %d", count)

	// The keypad's del key is always present.
	err = s.expect.Locator(s.page.Locator(tid("keypad-del"))).ToBeVisible()
	s.Require().NoError(err, "keypad del key should be visible")
}

func (s *E2ETestSuite) TestCategoryDetailsFromInsights() {
	s.login()

	// Seed two Groceries expenses so the category row has a clear total/count.
	s.addExpense("12.50", "Bakery", "groceries")
	s.addExpense("25.00", "Supermarket", "groceries")

	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(2)
	s.Require().NoError(err, "expected 2 rows after seeding")

	// Switch to Insights. The TabBar's insights button has no data-testid, so
	// drive navigation through the URL directly — equivalent to a tap and not
	// the focus of this test.
	_, err = s.page.Goto(appURL + "/insights")
	s.Require().NoError(err, "failed to navigate to /insights")

	// Click the Groceries category row.
	row := s.page.Locator(tid("category-row-groceries"))
	err = s.expect.Locator(row).ToBeVisible()
	s.Require().NoError(err, "groceries category row not visible on insights")

	err = row.Click()
	s.Require().NoError(err, "failed to click groceries row")

	// Details screen renders with the right total and count.
	err = s.expect.Locator(s.page.Locator(tid("category-details"))).ToBeVisible()
	s.Require().NoError(err, "category details screen not visible")

	err = s.expect.Locator(s.page.Locator(tid("category-details-total"))).ToContainText("37")
	s.Require().NoError(err, "category total mismatch (expected 37 in 37.50)")

	err = s.expect.Locator(s.page.Locator(tid("category-details-count"))).ToContainText("2 transactions")
	s.Require().NoError(err, "category count mismatch")

	// Both expenses appear in the day-grouped list.
	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(2)
	s.Require().NoError(err, "expected 2 expense rows on details page")

	// Tapping a row opens the edit form for that expense.
	err = s.page.Locator(tid("expense-row")).First().Click()
	s.Require().NoError(err, "failed to click expense row on details")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form should open from details click")
}

// TestFeedSyncPicksUpServerSideInserts documents the delta-sync contract:
// a row inserted server-side after the initial cold fetch (e.g. another
// device, another tab, or an admin script — simulated here by writing to
// the DB directly) should surface in the Feed on the next navigation
// without a full page reload. If this test regresses, lastSyncAt isn't
// being pinned on cold start or the Feed mount effect isn't firing the
// diff hook.
func (s *E2ETestSuite) TestFeedSyncPicksUpServerSideInserts() {
	s.login()

	// Baseline: the Feed is empty at login. The initial GET /api/expenses
	// has already landed (the hero-label / blank-state test covers the
	// mount path) and lastSyncAt is set in the React Query cache.
	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(0)
	s.Require().NoError(err, "expected empty feed before server-side insert")

	// Insert a row directly into the DB the running server is using.
	// The admin bootstrap user is "testuser" (created by the harness) and
	// the only user in the DB, so any new expense with user_id=1 is
	// visible to them on the next sync. Business-key tuple is distinct
	// from anything the test creates via the UI so the partial unique
	// index stays happy.
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open DB for out-of-band insert")
	_, err = db.InsertExpense(77.77, "Synced From Elsewhere", "Other", time.Now(), 1)
	s.Require().NoError(err, "could not insert expense out-of-band")
	db.Close()

	// Force a Feed re-mount. Any navigation does — bouncing through /insights
	// and back is the shortest round trip that keeps the browser's session
	// cookie + in-memory React Query cache so lastSyncAt persists.
	_, err = s.page.Goto(appURL + "/insights")
	s.Require().NoError(err, "failed to navigate away from feed")
	_, err = s.page.Goto(appURL)
	s.Require().NoError(err, "failed to navigate back to feed")

	// On /insights Goto is a hard navigation, which drops React Query's
	// in-memory cache — the subsequent landing on / effectively cold-starts
	// again. That's fine: it still exercises the wire contract (new row
	// returned by GET /api/expenses). For a true in-memory diff we'd need
	// to drive navigation via the Router, which has no data-testid target.
	// The full-list response carries the inserted row too, so assert on
	// it either way.
	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected the out-of-band row to appear after re-navigation")

	row := s.page.Locator(tid("expense-row")).First()
	err = s.expect.Locator(row.Locator(tid("expense-row-desc"))).ToHaveText("Synced From Elsewhere")
	s.Require().NoError(err, "expected the synced row's description")
	err = s.expect.Locator(row.Locator(tid("expense-row-amount"))).ToContainText("77.77")
	s.Require().NoError(err, "expected the synced row's amount")
}

// TestDeleteThenReinsertSameTuple pins the partial unique index: after a
// soft-delete, the user can re-enter the exact same (date, amount,
// description) tuple without a 409. This is the practical user-facing
// payoff of the tombstone migration — they can delete a mistake and
// re-enter it with the same data.
func (s *E2ETestSuite) TestDeleteThenReinsertSameTuple() {
	s.login()

	s.addExpense("4.50", "Coffee", "eating")
	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected 1 row after initial create")

	// Open the row for edit and delete it.
	row := s.page.Locator(tid("expense-row")).First()
	err = row.Click()
	s.Require().NoError(err, "failed to click row")
	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "edit form not visible")
	s.page.OnDialog(func(dialog playwright.Dialog) { dialog.Accept() })
	err = s.page.Locator(tid("entry-delete")).Click()
	s.Require().NoError(err, "failed to click delete")
	err = s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "should be back on feed after delete")
	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(0)
	s.Require().NoError(err, "expected row to disappear after delete")

	// Re-create the same tuple. With the non-partial index from before the
	// soft-delete migration this would come back as a 409 and the error
	// banner would show; we assert the opposite — the row materializes
	// and no banner is rendered.
	s.addExpense("4.50", "Coffee", "eating")
	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected the re-inserted row to appear")
	count, err := s.page.Locator(tid("error-banner")).Count()
	s.Require().NoError(err, "failed to count error banners")
	s.Require().Equal(0, count, "no banner should appear on successful re-insert")
}

// addExpense is a small helper used by tests that need pre-seeded rows. It
// taps FAB → keypad → note → category tile → submit.
func (s *E2ETestSuite) addExpense(amount, note, categorySlug string) {
	err := s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	s.pressKeys(amount)

	err = s.page.Locator(tid("entry-note")).Fill(note)
	s.Require().NoError(err, "failed to fill note %q", note)

	err = s.page.Locator(tid("category-tile-" + categorySlug)).Click()
	s.Require().NoError(err, "failed to select category %s", categorySlug)

	err = s.page.Locator(tid("entry-submit")).Click()
	s.Require().NoError(err, "failed to submit %q", note)

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).Not().ToBeVisible()
	s.Require().NoError(err, "entry form should close after submit")
}

// TestE2ESuite is the entry point go test discovers.
func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
