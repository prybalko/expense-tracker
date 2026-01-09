package e2e

import (
	"expense-tracker/internal/storage"
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
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
func (suite *E2ETestSuite) SetupSuite() {
	pw, err := playwright.Run()
	require.NoError(suite.T(), err, "could not launch playwright")
	suite.pw = pw

	browser, err := pw.Chromium.Launch()
	require.NoError(suite.T(), err, "could not launch chromium")
	suite.browser = browser

	suite.expect = playwright.NewPlaywrightAssertions()
}

// TearDownSuite runs once after all tests
func (suite *E2ETestSuite) TearDownSuite() {
	if suite.browser != nil {
		suite.browser.Close()
	}
	if suite.pw != nil {
		suite.pw.Stop()
	}
}

// SetupTest runs before each test
func (suite *E2ETestSuite) SetupTest() {
	// Clear the database before each test
	db, err := storage.NewDB(dbPath)
	require.NoError(suite.T(), err, "could not open database for cleanup")
	err = db.ClearExpenses()
	require.NoError(suite.T(), err, "could not clear expenses")
	db.Close()

	page, err := suite.browser.NewPage()
	require.NoError(suite.T(), err, "could not create page")
	suite.page = page

	_, err = suite.page.Goto(appURL)
	require.NoError(suite.T(), err, "could not navigate to app")
}

// TearDownTest runs after each test
func (suite *E2ETestSuite) TearDownTest() {
	if suite.page != nil {
		suite.page.Close()
	}
}

func (suite *E2ETestSuite) login() {
	// Wait for login form
	err := suite.expect.Locator(suite.page.Locator(".login-form")).ToBeVisible()
	require.NoError(suite.T(), err, "login form not visible")

	// Fill in credentials
	err = suite.page.Locator("input[name=username]").Fill("testuser")
	require.NoError(suite.T(), err, "failed to fill username")

	err = suite.page.Locator("input[name=password]").Fill("testpass123")
	require.NoError(suite.T(), err, "failed to fill password")

	// Submit login
	err = suite.page.Locator(".login-btn").Click()
	require.NoError(suite.T(), err, "failed to click login")

	// Wait for redirect to expenses page
	err = suite.expect.Locator(suite.page.Locator(".list-screen")).ToBeVisible()
	require.NoError(suite.T(), err, "did not redirect to expenses page after login")
}

func (suite *E2ETestSuite) TestCompleteUserFlow() {
	// Login
	suite.login()

	// Verify Homepage
	err := suite.expect.Locator(suite.page.Locator(".summary small")).ToHaveText("Spent this month")
	require.NoError(suite.T(), err, "homepage assertion failed")

	// Create Expense - Click add button
	err = suite.page.Locator(".fab-add").Click()
	require.NoError(suite.T(), err, "failed to click add button")

	// Wait for form
	err = suite.expect.Locator(suite.page.Locator("#expense-form")).ToBeVisible()
	require.NoError(suite.T(), err, "expense form not visible")

	// Enter Amount: 12.50 using keypad
	keys := []string{"1", "2", ".", "5", "0"}
	for _, key := range keys {
		err = suite.page.Locator("button:text-is('" + key + "')").Click()
		require.NoError(suite.T(), err, "failed to click key %s", key)
	}

	// Verify amount display
	err = suite.expect.Locator(suite.page.Locator("#display-amount")).ToHaveText("12.50")
	require.NoError(suite.T(), err, "amount display mismatch")

	// Fill description
	err = suite.page.Locator("input[name=description]").Fill("Lunch Test")
	require.NoError(suite.T(), err, "failed to fill description")

	// Select category
	_, err = suite.page.Locator("select[name=category]").SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"food"},
	})
	require.NoError(suite.T(), err, "failed to select category")

	// Submit
	err = suite.page.Locator("button.submit").Click()
	require.NoError(suite.T(), err, "failed to submit expense")

	// Verify in List - Wait for expense item to appear
	err = suite.expect.Locator(suite.page.Locator(".expense-item")).ToHaveCount(1)
	require.NoError(suite.T(), err, "expense item count mismatch")

	item := suite.page.Locator(".expense-item").First()
	err = suite.expect.Locator(item.Locator(".expense-details strong")).ToHaveText("Lunch Test")
	require.NoError(suite.T(), err, "description mismatch")

	err = suite.expect.Locator(item.Locator(".expense-amount")).ToContainText("12.50")
	require.NoError(suite.T(), err, "amount mismatch")
}

func (suite *E2ETestSuite) TestAddExpenseToBlankList() {
	// Login
	suite.login()

	// Verify the list is blank initially (no expense items)
	count, err := suite.page.Locator(".expense-item").Count()
	require.NoError(suite.T(), err, "failed to count expense items")
	require.Equal(suite.T(), 0, count, "expected list to be blank, but found %d items", count)

	// Verify total is 0.00
	err = suite.expect.Locator(suite.page.Locator(".total")).ToContainText("0.00")
	require.NoError(suite.T(), err, "expected total to be 0.00")

	// Click the "+" button to add an expense
	err = suite.page.Locator(".fab-add").Click()
	require.NoError(suite.T(), err, "failed to click add button")

	// Wait for the expense form to appear
	err = suite.expect.Locator(suite.page.Locator("#expense-form")).ToBeVisible()
	require.NoError(suite.T(), err, "expense form not visible")

	// Set the amount to 25.99 using the keypad
	keys := []string{"2", "5", ".", "9", "9"}
	for _, key := range keys {
		err = suite.page.Locator("button:text-is('" + key + "')").Click()
		require.NoError(suite.T(), err, "failed to click key %s", key)
	}

	// Verify the amount is displayed correctly
	err = suite.expect.Locator(suite.page.Locator("#display-amount")).ToHaveText("25.99")
	require.NoError(suite.T(), err, "amount display should show 25.99")

	// Save the expense by clicking the submit button (✓)
	err = suite.page.Locator("button.submit").Click()
	require.NoError(suite.T(), err, "failed to submit expense")

	// Verify the expense is visible in the list
	err = suite.expect.Locator(suite.page.Locator(".expense-item")).ToHaveCount(1)
	require.NoError(suite.T(), err, "expected exactly 1 expense item in the list")

	// Verify the amount is displayed in the list
	err = suite.expect.Locator(suite.page.Locator(".expense-amount")).ToContainText("25.99")
	require.NoError(suite.T(), err, "expense amount should be visible in the list")

	// Verify the total is updated
	err = suite.expect.Locator(suite.page.Locator(".total")).ToContainText("25.99")
	require.NoError(suite.T(), err, "total should be updated to 25.99")
}

// TestE2ESuite runs the e2e test suite
func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
