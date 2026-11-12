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

// E2ETestSuite anchors selectors on data-testid; glyphs and class names are not stable targets.
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

	// CalendarGrid disables future days; day 1 and today are always selectable.
	todayDay := time.Now().Day()
	createDay := 1
	editDay := todayDay

	err := s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	err = s.page.Locator(tid("category-tile-transport")).Click()
	s.Require().NoError(err, "failed to select Transport category")

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

	err = row.Click()
	s.Require().NoError(err, "failed to click row")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "edit form not visible")

	err = s.expect.Locator(s.page.Locator(tid("entry-note"))).ToHaveValue("Original Expense")
	s.Require().NoError(err, "note not populated for edit")

	// Prefill strips padding zeros: "50", not "50.00".
	err = s.expect.Locator(s.page.Locator(tid("entry-amount"))).ToHaveAttribute("data-amount", "50")
	s.Require().NoError(err, "amount not prefilled in trailing-zero-stripped form")

	// Extra del presses on an empty amount are no-ops.
	for range 5 {
		err = s.page.Locator(tid("keypad-del")).Click()
		s.Require().NoError(err, "failed to press del")
	}
	err = s.expect.Locator(s.page.Locator(tid("entry-amount"))).ToHaveAttribute("data-amount", "0")
	s.Require().NoError(err, "amount not cleared")

	s.pressKeys("40.00")

	err = s.page.Locator(tid("entry-note")).Fill("Updated Expense")
	s.Require().NoError(err, "failed to update note")

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

	// EntryForm confirms delete with window.confirm.
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

	count, err := s.page.Locator(tid("entry-delete")).Count()
	s.Require().NoError(err, "failed to count delete buttons")
	s.Require().Equal(0, count, "delete button should not exist on create, found %d", count)

	err = s.expect.Locator(s.page.Locator(tid("keypad-del"))).ToBeVisible()
	s.Require().NoError(err, "keypad del key should be visible")
}

func (s *E2ETestSuite) TestCategoryDetailsFromInsights() {
	s.login()

	s.addExpense("12.50", "Bakery", "groceries")
	s.addExpense("25.00", "Supermarket", "groceries")

	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(2)
	s.Require().NoError(err, "expected 2 rows after seeding")

	_, err = s.page.Goto(appURL + "/insights")
	s.Require().NoError(err, "failed to navigate to /insights")

	row := s.page.Locator(tid("category-row-groceries"))
	err = s.expect.Locator(row).ToBeVisible()
	s.Require().NoError(err, "groceries category row not visible on insights")

	err = row.Click()
	s.Require().NoError(err, "failed to click groceries row")

	err = s.expect.Locator(s.page.Locator(tid("category-details"))).ToBeVisible()
	s.Require().NoError(err, "category details screen not visible")

	err = s.expect.Locator(s.page.Locator(tid("category-details-total"))).ToContainText("37")
	s.Require().NoError(err, "category total mismatch (expected 37 in 37.50)")

	err = s.expect.Locator(s.page.Locator(tid("category-details-count"))).ToContainText("2 transactions")
	s.Require().NoError(err, "category count mismatch")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(2)
	s.Require().NoError(err, "expected 2 expense rows on details page")

	err = s.page.Locator(tid("expense-row")).First().Click()
	s.Require().NoError(err, "failed to click expense row on details")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form should open from details click")
}

func (s *E2ETestSuite) goInsights() {
	err := s.page.Locator(tid("tab-insights")).Click()
	s.Require().NoError(err, "failed to switch to insights")
	err = s.expect.Locator(s.page.Locator(tid("insights-screen"))).ToBeVisible()
	s.Require().NoError(err, "insights screen not visible")
}

func (s *E2ETestSuite) TestInsightsYearViewAndDrilldown() {
	s.login()
	s.addExpense("12.50", "Bakery", "groceries")
	s.addExpense("30.00", "Train", "transport")

	s.goInsights()

	err := s.expect.Locator(s.page.Locator(tid("category-row-groceries"))).ToBeVisible()
	s.Require().NoError(err, "groceries treemap tile not visible in month view")

	err = s.page.Locator(tid("segment-year")).Click()
	s.Require().NoError(err, "failed to switch to year view")
	err = s.expect.Locator(s.page.Locator(tid("segment-year"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "year segment should be selected")

	transportRow := s.page.Locator(tid("category-row-transport"))
	err = s.expect.Locator(transportRow).ToBeVisible()
	s.Require().NoError(err, "transport category row not visible in year view")

	err = transportRow.Click()
	s.Require().NoError(err, "failed to open transport category")
	err = s.expect.Locator(s.page.Locator(tid("category-details"))).ToBeVisible()
	s.Require().NoError(err, "category details not visible")
	err = s.expect.Locator(s.page.Locator(tid("category-details-count"))).ToContainText("1 transaction")
	s.Require().NoError(err, "expected 1 transaction in transport detail")

	err = s.page.Locator(tid("category-details-back")).Click()
	s.Require().NoError(err, "failed to go back from category details")
	err = s.expect.Locator(s.page.Locator(tid("segment-year"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "should return to the year view")
}

func (s *E2ETestSuite) TestInsightsPeriodNavigation() {
	s.login()
	s.addExpense("20.00", "Lunch", "eating")

	s.goInsights()
	err := s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "eating tile not visible in current month")

	err = s.expect.Locator(s.page.Locator(tid("period-next"))).ToBeDisabled()
	s.Require().NoError(err, "next period should be disabled at current month")

	err = s.page.Locator(tid("period-prev")).Click()
	s.Require().NoError(err, "failed to step to previous month")
	err = s.expect.Locator(s.page.GetByText("No spending in this period.")).ToBeVisible()
	s.Require().NoError(err, "previous month should be empty")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToHaveCount(0)
	s.Require().NoError(err, "eating tile should be gone in the previous month")

	err = s.page.Locator(tid("period-next")).Click()
	s.Require().NoError(err, "failed to step forward")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "eating tile should return in the current month")
	err = s.expect.Locator(s.page.Locator(tid("period-next"))).ToBeDisabled()
	s.Require().NoError(err, "next period should be disabled again at current month")
}

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

	err = s.page.Locator(tid("category-row-eating")).Click()
	s.Require().NoError(err, "failed to open eating category")
	err = s.expect.Locator(s.page.Locator(tid("category-details"))).ToBeVisible()
	s.Require().NoError(err, "category details not visible")
	err = s.expect.Locator(s.page.Locator(tid("category-details-count"))).ToContainText("1 transaction")
	s.Require().NoError(err, "expected 1 transaction in eating detail under Mine")

	err = s.page.Locator(tid("category-details-back")).Click()
	s.Require().NoError(err, "failed to go back from category details")
	err = s.expect.Locator(s.page.Locator(tid("segment-mine"))).ToHaveAttribute("aria-pressed", "true")
	s.Require().NoError(err, "Mine should survive the details round-trip")
	err = s.expect.Locator(s.page.Locator(tid("category-row-eating"))).ToBeVisible()
	s.Require().NoError(err, "eating tile should still be visible under Mine after back")
}

func (s *E2ETestSuite) TestFeedSyncPicksUpServerSideInserts() {
	s.login()

	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(0)
	s.Require().NoError(err, "expected empty feed before server-side insert")

	// User 1 is the harness bootstrap user (testuser).
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open DB for out-of-band insert")
	_, err = db.InsertExpense(77.77, "Synced From Elsewhere", "Other", time.Now(), 1)
	s.Require().NoError(err, "could not insert expense out-of-band")
	db.Close()

	// Goto is a hard navigation, so this checks the full-list fetch, not the in-memory diff.
	_, err = s.page.Goto(appURL + "/insights")
	s.Require().NoError(err, "failed to navigate away from feed")
	_, err = s.page.Goto(appURL)
	s.Require().NoError(err, "failed to navigate back to feed")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected the out-of-band row to appear after re-navigation")

	row := s.page.Locator(tid("expense-row")).First()
	err = s.expect.Locator(row.Locator(tid("expense-row-desc"))).ToHaveText("Synced From Elsewhere")
	s.Require().NoError(err, "expected the synced row's description")
	err = s.expect.Locator(row.Locator(tid("expense-row-amount"))).ToContainText("77.77")
	s.Require().NoError(err, "expected the synced row's amount")
}

func (s *E2ETestSuite) TestFeedSyncPicksUpServerSideDeletes() {
	s.login()

	s.addExpense("33.33", "Synced Delete Target", "groceries")
	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected the seeded row before out-of-band delete")

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

	// Playwright already reports visibilityState "visible", so the event alone is enough.
	_, err = s.page.Evaluate(`() => document.dispatchEvent(new Event('visibilitychange'))`)
	s.Require().NoError(err, "failed to dispatch visibilitychange")

	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(0)
	s.Require().NoError(err, "row should disappear after server-side delete + sync")
	err = s.expect.Locator(s.page.Locator(tid("hero-total"))).ToContainText("0.00")
	s.Require().NoError(err, "hero total should drop to 0.00 once the row syncs out")
}

func (s *E2ETestSuite) TestFeedTabSwitchDoesNotFireSync() {
	s.login()

	// Install the counter only after the expected cold-start fetch.
	err := s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "feed should be visible after login")

	changesCalls := 0
	err = s.page.Route("**/api/expenses/changes**", func(route playwright.Route) {
		changesCalls++
		_ = route.Continue()
	})
	s.Require().NoError(err, "failed to install /changes route counter")

	err = s.page.Locator(tid("tab-insights")).Click()
	s.Require().NoError(err, "failed to switch to insights")
	err = s.expect.Locator(s.page.Locator(tid("insights-screen"))).ToBeVisible()
	s.Require().NoError(err, "insights screen not visible after tab switch")
	err = s.page.Locator(tid("tab-feed")).Click()
	s.Require().NoError(err, "failed to switch back to feed")
	err = s.expect.Locator(s.page.Locator(tid("feed-screen"))).ToBeVisible()
	s.Require().NoError(err, "feed screen not visible after returning")

	// Margin for a microtask-scheduled request to reach the counter.
	time.Sleep(150 * time.Millisecond)
	s.Require().Equal(0, changesCalls,
		"tab toggle must not call /api/expenses/changes — got %d call(s)", changesCalls)
}

func (s *E2ETestSuite) TestDeleteThenReinsertSameTuple() {
	s.login()

	s.addExpense("4.50", "Coffee", "eating")
	err := s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected 1 row after initial create")

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

	s.addExpense("4.50", "Coffee", "eating")
	err = s.expect.Locator(s.page.Locator(tid("expense-row"))).ToHaveCount(1)
	s.Require().NoError(err, "expected the re-inserted row to appear")
	count, err := s.page.Locator(tid("error-banner")).Count()
	s.Require().NoError(err, "failed to count error banners")
	s.Require().Equal(0, count, "no banner should appear on successful re-insert")
}

func (s *E2ETestSuite) TestFeedShowsOtherUsersRowTinted() {
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open DB to seed second user")
	hash, err := auth.HashPassword("partnerpw")
	s.Require().NoError(err, "could not hash partner password")
	partner, err := db.CreateUser("partner", hash)
	s.Require().NoError(err, "could not create partner user")

	// Insert before login so the cold-start fetch returns it.
	_, err = db.InsertExpense(11.11, "Roommate Beer", "Other", time.Now(), partner.ID)
	s.Require().NoError(err, "could not insert partner-owned expense")
	db.Close()

	s.login()

	partnerRow := s.page.Locator(tid("expense-row")).Filter(playwright.LocatorFilterOptions{
		HasText: "Roommate Beer",
	})
	err = s.expect.Locator(partnerRow).ToHaveCount(1)
	s.Require().NoError(err, "partner-owned row should be visible to testuser")
	err = s.expect.Locator(partnerRow).ToHaveAttribute("data-not-mine", "true")
	s.Require().NoError(err, "partner-owned row should be tagged data-not-mine=true")

	s.addExpense("4.50", "My Own Coffee", "eating")
	mineRow := s.page.Locator(tid("expense-row")).Filter(playwright.LocatorFilterOptions{
		HasText: "My Own Coffee",
	})
	err = s.expect.Locator(mineRow).ToHaveCount(1)
	s.Require().NoError(err, "self-authored row should appear on the feed")
	err = s.expect.Locator(mineRow).Not().ToHaveAttribute("data-not-mine", "true")
	s.Require().NoError(err, "self-authored row must not be marked data-not-mine")
}

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

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
