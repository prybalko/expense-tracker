package e2e

import (
	"expense-tracker/internal/storage"
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite provides a test suite for end-to-end tests
type E2ETestSuite struct {
	suite.Suite
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
	expect  playwright.PlaywrightAssertions
}

// SetupSuite runs once before all tests
func (s *E2ETestSuite) SetupSuite() {
	pw, err := playwright.Run()
	s.Require().NoError(err, "could not launch playwright")
	s.pw = pw

	browser, err := pw.Chromium.Launch()
	s.Require().NoError(err, "could not launch chromium")
	s.browser = browser

	s.expect = playwright.NewPlaywrightAssertions()
}

// TearDownSuite runs once after all tests
func (s *E2ETestSuite) TearDownSuite() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
}

// SetupTest runs before each test
func (s *E2ETestSuite) SetupTest() {
	// Clear the database before each test
	db, err := storage.NewDB(dbPath)
	s.Require().NoError(err, "could not open database for cleanup")
	err = db.ClearExpenses()
	s.Require().NoError(err, "could not clear expenses")
	db.Close()

	page, err := s.browser.NewPage()
	s.Require().NoError(err, "could not create page")
	s.page = page

	_, err = s.page.Goto(appURL)
	s.Require().NoError(err, "could not navigate to app")
}

// TearDownTest runs after each test
func (s *E2ETestSuite) TearDownTest() {
	if s.page != nil {
		s.page.Close()
	}
}

func (s *E2ETestSuite) login() {
	// Wait for login form
	err := s.expect.Locator(s.page.Locator(".login-form")).ToBeVisible()
	s.Require().NoError(err, "login form not visible")

	// Fill in credentials
	err = s.page.Locator("input[name=username]").Fill("testuser")
	s.Require().NoError(err, "failed to fill username")

	err = s.page.Locator("input[name=password]").Fill("testpass123")
	s.Require().NoError(err, "failed to fill password")

	// Submit login
	err = s.page.Locator(".login-btn").Click()
	s.Require().NoError(err, "failed to click login")

	// Wait for redirect to expenses page
	err = s.expect.Locator(s.page.Locator(".list-screen")).ToBeVisible()
	s.Require().NoError(err, "did not redirect to expenses page after login")
}

func (s *E2ETestSuite) TestCompleteUserFlow() {
	// Login
	s.login()

	// Verify Homepage
	err := s.expect.Locator(s.page.Locator(".summary small")).ToHaveText("Spent this month")
	s.Require().NoError(err, "homepage assertion failed")

	// Create Expense - Click add button
	err = s.page.Locator(".fab-add").Click()
	s.Require().NoError(err, "failed to click add button")

	// Wait for form
	err = s.expect.Locator(s.page.Locator("#expense-form")).ToBeVisible()
	s.Require().NoError(err, "expense form not visible")

	// Enter Amount: 12.50 using keypad
	keys := []string{"1", "2", ".", "5", "0"}
	for _, key := range keys {
		err = s.page.Locator("button:text-is('" + key + "')").Click()
		s.Require().NoError(err, "failed to click key %s", key)
	}

	// Verify amount display
	err = s.expect.Locator(s.page.Locator("#display-amount")).ToHaveText("12.50")
	s.Require().NoError(err, "amount display mismatch")

	// Fill description
	err = s.page.Locator("input[name=description]").Fill("Lunch Test")
	s.Require().NoError(err, "failed to fill description")

	// Select category
	_, err = s.page.Locator("select[name=category]").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"food"},
	})
	s.Require().NoError(err, "failed to select category")

	// Submit
	err = s.page.Locator("button.submit").Click()
	s.Require().NoError(err, "failed to submit expense")

	// Verify in List - Wait for expense item to appear
	err = s.expect.Locator(s.page.Locator(".expense-item")).ToHaveCount(1)
	s.Require().NoError(err, "expense item count mismatch")

	item := s.page.Locator(".expense-item").First()
	err = s.expect.Locator(item.Locator(".expense-details strong")).ToHaveText("Lunch Test")
	s.Require().NoError(err, "description mismatch")

	err = s.expect.Locator(item.Locator(".expense-amount")).ToContainText("12.50")
	s.Require().NoError(err, "amount mismatch")
}

func (s *E2ETestSuite) TestAddExpenseToBlankList() {
	// Login
	s.login()

	// Verify the list is blank initially (no expense items)
	count, err := s.page.Locator(".expense-item").Count()
	s.Require().NoError(err, "failed to count expense items")
	s.Require().Equal(0, count, "expected list to be blank, but found %d items", count)

	// Verify total is 0.00
	err = s.expect.Locator(s.page.Locator(".total")).ToContainText("0.00")
	s.Require().NoError(err, "expected total to be 0.00")

	// Click the "+" button to add an expense
	err = s.page.Locator(".fab-add").Click()
	s.Require().NoError(err, "failed to click add button")

	// Wait for the expense form to appear
	err = s.expect.Locator(s.page.Locator("#expense-form")).ToBeVisible()
	s.Require().NoError(err, "expense form not visible")

	// Set the amount to 25.99 using the keypad
	keys := []string{"2", "5", ".", "9", "9"}
	for _, key := range keys {
		err = s.page.Locator("button:text-is('" + key + "')").Click()
		s.Require().NoError(err, "failed to click key %s", key)
	}

	// Verify the amount is displayed correctly
	err = s.expect.Locator(s.page.Locator("#display-amount")).ToHaveText("25.99")
	s.Require().NoError(err, "amount display should show 25.99")

	// Save the expense by clicking the submit button (✓)
	err = s.page.Locator("button.submit").Click()
	s.Require().NoError(err, "failed to submit expense")

	// Verify the expense is visible in the list
	err = s.expect.Locator(s.page.Locator(".expense-item")).ToHaveCount(1)
	s.Require().NoError(err, "expected exactly 1 expense item in the list")

	// Verify the amount is displayed in the list
	err = s.expect.Locator(s.page.Locator(".expense-amount")).ToContainText("25.99")
	s.Require().NoError(err, "expense amount should be visible in the list")

	// Verify the total is updated
	err = s.expect.Locator(s.page.Locator(".total")).ToContainText("25.99")
	s.Require().NoError(err, "total should be updated to 25.99")
}

// TestE2ESuite runs the e2e test suite
func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
