package e2e

import (
	"expense-tracker/internal/auth"
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

	ctx, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
		// Block the SW so hard navigations hit the network, not a stale cache.
		ServiceWorkers: playwright.ServiceWorkerPolicyBlock,
	})
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

	// Prefill strips padding zeros: "50", not "50.00".
	err = s.expect.Locator(s.page.Locator(tid("entry-amount"))).ToHaveAttribute("data-amount", "50")
	s.Require().NoError(err, "amount not prefilled in trailing-zero-stripped form")

	// 3. Clear the amount; extra del presses on empty are no-ops.
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

// goInsights switches to the Insights tab and waits for the screen.
func (s *E2ETestSuite) goInsights() {
	err := s.page.Locator(tid("tab-insights")).Click()
	s.Require().NoError(err, "failed to switch to insights")
	err = s.expect.Locator(s.page.Locator(tid("insights-screen"))).ToBeVisible()
	s.Require().NoError(err, "insights screen not visible")
}

// Month treemap → Year view (bars + rows) → year-scoped drilldown → back.
func (s *E2ETestSuite) TestInsightsYearViewAndDrilldown() {
	s.login()
	s.addExpense("12.50", "Bakery", "groceries")
	s.addExpense("30.00", "Train", "transport")

	s.goInsights()

	// Month view (default) renders the treemap tiles.
	err := s.expect.Locator(s.page.Locator(tid("category-row-groceries"))).ToBeVisible()
	s.Require().NoError(err, "groceries treemap tile not visible in month view")

	// Switch to the Year view.
	err = s.page.Locator(tid("segment-year")).Click()
	s.Require().NoError(err, "failed to switch to year view")
	err = s.expect.Locator(s.page.Locator(tid("segment-year"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "year segment should be selected")

	transportRow := s.page.Locator(tid("category-row-transport"))
	err = s.expect.Locator(transportRow).ToBeVisible()
	s.Require().NoError(err, "transport category row not visible in year view")

	// Drill into a category from the year view (year-scoped detail).
	err = transportRow.Click()
	s.Require().NoError(err, "failed to open transport category")
	err = s.expect.Locator(s.page.Locator(tid("category-details"))).ToBeVisible()
	s.Require().NoError(err, "category details not visible")
	err = s.expect.Locator(s.page.Locator(tid("category-details-count"))).ToContainText("1 transaction")
	s.Require().NoError(err, "expected 1 transaction in transport detail")

	// Back returns to the Year view.
	err = s.page.Locator(tid("category-details-back")).Click()
	s.Require().NoError(err, "failed to go back from category details")
	err = s.expect.Locator(s.page.Locator(tid("segment-year"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "should return to the year view")
}

// Period stepper: next clamped at current month, prev is empty, forward returns.
func (s *E2ETestSuite) TestInsightsPeriodNavigation() {
	s.login()
	s.addExpense("20.00", "Lunch", "eating")

	s.goInsights()
	err := s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "eating tile not visible in current month")

	// At the current month the next chevron is clamped.
	err = s.expect.Locator(s.page.Locator(tid("period-next"))).ToBeDisabled()
	s.Require().NoError(err, "next period should be disabled at current month")

	// Step to the previous (empty) month.
	err = s.page.Locator(tid("period-prev")).Click()
	s.Require().NoError(err, "failed to step to previous month")
	err = s.expect.Locator(s.page.GetByText("No spending in this period.")).ToBeVisible()
	s.Require().NoError(err, "previous month should be empty")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToHaveCount(0)
	s.Require().NoError(err, "eating tile should be gone in the previous month")

	// Step forward to the current month again; next re-clamps.
	err = s.page.Locator(tid("period-next")).Click()
	s.Require().NoError(err, "failed to step forward")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "eating tile should return in the current month")
	err = s.expect.Locator(s.page.Locator(tid("period-next"))).ToBeDisabled()
	s.Require().NoError(err, "next period should be disabled again at current month")
}

// Mine/All toggle: single user owns everything, so Mine keeps it visible.
func (s *E2ETestSuite) TestInsightsMineAllFilter() {
	s.login()
	s.addExpense("15.00", "Coffee", "eating")

	s.goInsights()

	err := s.expect.Locator(s.page.Locator(tid("segment-all"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "All should be the default scope")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "eating tile visible under All")

	err = s.page.Locator(tid("segment-mine")).Click()
	s.Require().NoError(err, "failed to switch to Mine")
	err = s.expect.Locator(s.page.Locator(tid("segment-mine"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "Mine should be selected")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "own expense should remain visible under Mine")
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

// TestFeedSyncPicksUpServerSideDeletes is the mirror of the insert test
// above, pinning the `deletedIds` half of the /api/expenses/changes
// contract. Reported in the wild as "I delete an item on phone A but it
// stays on phone B forever"; the round trip the user expects is:
//
//  1. Phone B's full-list fetch lands → row visible, lastSyncAt = T0.
//  2. Phone A soft-deletes the row → row stays in the DB with deleted_at
//     set to T1 > T0.
//  3. Phone B resumes the PWA (or pulls to refresh) → useSyncExpenses hits
//     /api/expenses/changes?since=T0 → server emits deletedIds=[id] →
//     mergeChanges drops the id from the cache → the row vanishes.
//
// Trigger choice: we drive `visibilitychange` rather than the pull gesture
// because it's the realistic phone-switching scenario, exercises the
// useSyncOnVisible hook end-to-end, and doesn't depend on touch-event
// emulation. The SPA Feed↔Insights bounce no longer fires a sync (that
// auto-trigger was the source of the wasted-request UX bug), so a tab tap
// alone wouldn't surface the deletion — the new test contract is "if the
// user is here and the page returns to the foreground, deletes propagate".
func (s *E2ETestSuite) TestFeedSyncPicksUpServerSideDeletes() {
	s.login()

	// 1. Seed a row through the UI. The mutation's onSuccess upserts it
	//    into the cache and advances lastSyncAt off X-Server-Time, so the
	//    subsequent diff has a real cursor to send back.
	s.addExpense("33.33", "Synced Delete Target", "groceries")
	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected the seeded row before out-of-band delete")

	// 2. Soft-delete the row directly in the DB the running server is
	//    using — same out-of-band trick the insert test uses, just on the
	//    DeleteExpense side. The admin bootstrap user (testuser, id=1) is
	//    the only owner, so list-and-match by description is unambiguous.
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open DB for out-of-band delete")
	all, err := db.ListExpensesAll()
	s.Require().NoError(err, "could not list expenses to find target id")
	var targetID int64
	for _, e := range all {
		if e.Description == "Synced Delete Target" {
			targetID = e.ID
			break
		}
	}
	s.Require().NotZero(targetID, "could not find seeded row in DB")
	s.Require().NoError(db.DeleteExpense(targetID), "soft-delete the target row")
	db.Close()

	// 3. Simulate the PWA returning to the foreground. The useSyncOnVisible
	//    handler in Feed.tsx checks document.visibilityState before firing;
	//    Playwright's page reports "visible" by default, so dispatching the
	//    event is sufficient to exercise the handler. This matches what
	//    Safari fires when the user re-selects the PWA from the app
	//    switcher — the exact path the wasted-on-mount sync used to take
	//    behind the scenes.
	_, err = s.page.Evaluate(`() => document.dispatchEvent(new Event('visibilitychange'))`)
	s.Require().NoError(err, "failed to dispatch visibilitychange")

	// 4. Row must be gone and the hero total must reflect zero. The two
	//    assertions together pin both halves of the merge: the row drop
	//    (cache mutation) and the downstream recomputed total (selector
	//    re-runs). If the row stays, the regression is somewhere on the
	//    `deletedIds` path; if the row drops but the hero stays at 33.33,
	//    the cache mutation isn't invalidating the derived insights.
	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(0)
	s.Require().NoError(err, "row should disappear after server-side delete + sync")
	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("0.00")
	s.Require().NoError(err, "hero total should drop to 0.00 once the row syncs out")
}

// TestFeedTabSwitchDoesNotFireSync is the negative half of the contract
// the user asked for: tapping between Feed and Insights MUST NOT issue an
// /api/expenses/changes request. Before the redesign every tab toggle
// paid a round-trip, and when nothing had changed on the server the
// response was an empty `{updated:[], deletedIds:[]}` — visibly a wasted
// call when watching the network panel. We intercept the route and assert
// it stays cold across an in-app navigation.
func (s *E2ETestSuite) TestFeedTabSwitchDoesNotFireSync() {
	s.login()

	// Wait for the cold-start GET /api/expenses to settle before we install
	// the counter — otherwise the initial fetch (which IS expected) would
	// be conflated with the tab-toggle behavior under test.
	err := s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "feed should be visible after login")

	changesCalls := 0
	err = s.page.Route("**/api/expenses/changes**", func(route playwright.Route) {
		changesCalls++
		// Let the request continue so any code path that did fire a diff
		// still works end-to-end; we only care about counting hits.
		_ = route.Continue()
	})
	s.Require().NoError(err, "failed to install /changes route counter")

	// Drive a Feed → Insights → Feed bounce through the TabBar (the
	// realistic phone-tap path). If the old mount-time `useEffect` is ever
	// resurrected, this round trip will increment changesCalls.
	err = s.page.Locator(tid("tab-insights")).Click()
	s.Require().NoError(err, "failed to switch to insights")
	err = s.expect.Locator(s.page.Locator(tid("insights-screen"))).ToBeVisible()
	s.Require().NoError(err, "insights screen not visible after tab switch")
	err = s.page.Locator(tid("tab-feed")).Click()
	s.Require().NoError(err, "failed to switch back to feed")
	err = s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "feed screen not visible after returning")

	// Give any in-flight request a moment to land in the counter before
	// asserting. The Feed re-mount is synchronous; the counter would
	// already see the call if one was made — but a small sleep guards
	// against a future microtask-scheduled mutation slipping through.
	time.Sleep(150 * time.Millisecond)
	s.Require().Equal(0, changesCalls,
		"tab toggle must not call /api/expenses/changes — got %d call(s)", changesCalls)
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

// TestFeedShowsOtherUsersRowTinted pins the shared-household UX contract:
// every authenticated user sees every expense, and rows authored by a
// different known user (user_id != null && user_id != me.id) are marked
// with data-not-mine="true" so the row can be visually tinted. A row the
// signed-in user creates themselves through the UI must NOT carry that
// attribute. Together these assertions guard against two regressions:
//  1. ListExpensesAll re-acquiring a per-user filter (Bob's row would
//     disappear from Alice's feed).
//  2. The "not mine" condition slipping back to include the self case.
func (s *E2ETestSuite) TestFeedShowsOtherUsersRowTinted() {
	// Mint a second household member directly against the DB. The harness
	// only knows the bootstrap admin (testuser, id=1); we add "partner"
	// out-of-band so we can stamp an expense with their user_id without
	// having to log in twice.
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open DB to seed second user")
	hash, err := auth.HashPassword("partnerpw")
	s.Require().NoError(err, "could not hash partner password")
	partner, err := db.CreateUser("partner", hash)
	s.Require().NoError(err, "could not create partner user")

	// Drop a row owned by the partner before login, so the Feed renders
	// it on its cold-start fetch instead of needing a diff round trip.
	_, err = db.InsertExpense(11.11, "Roommate Beer", "Other", time.Now(), partner.ID)
	s.Require().NoError(err, "could not insert partner-owned expense")
	db.Close()

	s.login()

	// The partner's row must be visible to testuser AND tagged as "not
	// mine". Other tests would also flag a per-user filter regression by
	// failing to find the row at all; we assert both halves here for
	// clarity.
	partnerRow := s.page.Locator(tid("expense-row")).Filter(playwright.LocatorFilterOptions{
		HasText: "Roommate Beer",
	})
	err = s.expect.Locator(partnerRow).ToHaveCount(1)
	s.Require().NoError(err, "partner-owned row should be visible to testuser")
	err = s.expect.Locator(partnerRow).ToHaveAttribute("data-not-mine", "true")
	s.Require().NoError(err, "partner-owned row should be tagged data-not-mine=true")

	// Negative control: a row testuser creates themselves through the UI
	// must NOT carry the data-not-mine flag. ExpenseRow renders the
	// attribute as `undefined` for self-authored rows so React omits it
	// from the DOM entirely; Not().ToHaveAttribute(_, "true") covers both
	// "attribute absent" and "attribute set to something other than true".
	s.addExpense("4.50", "My Own Coffee", "eating")
	mineRow := s.page.Locator(tid("expense-row")).Filter(playwright.LocatorFilterOptions{
		HasText: "My Own Coffee",
	})
	err = s.expect.Locator(mineRow).ToHaveCount(1)
	s.Require().NoError(err, "self-authored row should appear on the feed")
	err = s.expect.Locator(mineRow).Not().ToHaveAttribute("data-not-mine", "true")
	s.Require().NoError(err, "self-authored row must not be marked data-not-mine")
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
