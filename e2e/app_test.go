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

	// Workbox runtimeCaching uses StaleWhileRevalidate for /api/insights and
	// /api/expenses, so a freshly-revalidating fetch hands React Query the
	// stale body before the new one lands. The new body updates the SW cache
	// but doesn't trigger another React Query subscriber notification, so
	// the Hero stays on the old total. Blocking the SW per-context skips the
	// whole layer for tests.
	ctx, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
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

	// 1. Create an expense to edit, picking the Transport category and the 1st
	//    of the current month so we can assert both stick across edits.
	err := s.page.Locator(tid("fab-add")).Click()
	s.Require().NoError(err, "failed to click add button")

	err = s.expect.Locator(s.page.Locator(tid("entry-form"))).ToBeVisible()
	s.Require().NoError(err, "entry form not visible")

	err = s.page.Locator(tid("category-tile-transport")).Click()
	s.Require().NoError(err, "failed to select Transport category")

	// Open date sheet, pick the 1st, commit with Done.
	err = s.page.Locator(tid("date-pill")).Click()
	s.Require().NoError(err, "failed to open date picker")

	err = s.expect.Locator(s.page.Locator(tid("date-sheet"))).ToBeVisible()
	s.Require().NoError(err, "date sheet not visible")

	err = s.page.Locator(tid("calendar-day-" + dayISO(1))).Click()
	s.Require().NoError(err, "failed to pick day 1")

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

	// Bump the date forward by one day (calendar always has at least 2 days
	// since we created the expense on day 1; for day-1 runs the sheet allows
	// picking day 2 which is still in the past or today).
	err = s.page.Locator(tid("date-pill")).Click()
	s.Require().NoError(err, "failed to reopen date picker")

	err = s.expect.Locator(s.page.Locator(tid("date-sheet"))).ToBeVisible()
	s.Require().NoError(err, "date sheet not visible on edit")

	err = s.page.Locator(tid("calendar-day-" + dayISO(2))).Click()
	s.Require().NoError(err, "failed to pick day 2")

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

// TestE2ESuite is the entry point go test discovers.
func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
