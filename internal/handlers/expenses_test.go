package handlers

import (
	"context"
	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ExpenseHandlerTestSuite provides a test suite for expense handler tests
type ExpenseHandlerTestSuite struct {
	suite.Suite
	db          *storage.DB
	templateDir string
}

// SetupTest runs before each test
func (s *ExpenseHandlerTestSuite) SetupTest() {
	db, err := storage.NewDB(":memory:")
	s.Require().NoError(err, "failed to create test database")
	s.db = db

	s.templateDir = "../../web/templates"
	if _, err := os.Stat(s.templateDir); os.IsNotExist(err) {
		s.T().Skip("Template directory not found, skipping handler integration test")
	}
}

// TearDownTest runs after each test
func (s *ExpenseHandlerTestSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *ExpenseHandlerTestSuite) addUserContext(req *http.Request) *http.Request {
	ctx := context.WithValue(req.Context(), UserContextKey, &models.User{ID: 1, Username: "testuser"})
	return req.WithContext(ctx)
}

func (s *ExpenseHandlerTestSuite) TestListExpenses() {
	h := NewHandlers(s.db, s.templateDir, false)

	req := httptest.NewRequest("GET", "/expenses", http.NoBody)
	req = s.addUserContext(req)
	w := httptest.NewRecorder()

	h.ListExpenses(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Simple content check - look for "Spent this month" which is in list.html
	body := w.Body.String()
	s.Contains(body, "Spent this month")
}

func (s *ExpenseHandlerTestSuite) TestListExpenses_Unauthorized() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Request without user context should return 401
	req := httptest.NewRequest("GET", "/expenses", http.NoBody)
	w := httptest.NewRecorder()

	h.ListExpenses(w, req)

	resp := w.Result()
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *ExpenseHandlerTestSuite) TestListExpenses_HighlightOtherUsersExpenses() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Create user 1 first (the current user)
	user1, err := s.db.CreateUser("testuser", "password123")
	s.Require().NoError(err)

	// Create user 2 (the other user)
	user2, err := s.db.CreateUser("otheruser", "password456")
	s.Require().NoError(err)

	// Create expenses for both users
	date := parseTestDate("2026-01-15T12:00:00")

	// Expense by user 1 (current user in context)
	err = s.db.CreateExpense(50.00, "My Expense", "groceries", date, user1.ID)
	s.Require().NoError(err)

	// Expense by user 2 (other user)
	err = s.db.CreateExpense(30.00, "Other User Expense", "transport", date.Add(time.Hour), user2.ID)
	s.Require().NoError(err)

	// Request as user 1
	req := httptest.NewRequest("GET", "/expenses", http.NoBody)
	ctx := context.WithValue(req.Context(), UserContextKey, user1) // Use the actual user1 from DB
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListExpenses(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()

	// Should contain both expenses
	s.Contains(body, "My Expense")
	s.Contains(body, "Other User Expense")

	// Should highlight the other user's expense with floralwhite background
	s.Contains(body, "floralwhite", "other user's expense should have floralwhite background")

	// Verify the pattern: article tag with floralwhite style attribute
	s.Contains(body, `style="background-color: floralwhite;"`, "should have floralwhite background style")

	// Verify that floralwhite appears in context with the other user's expense
	// Find the article tag that contains "Other User Expense"
	articleStart := strings.Index(body, `data-description="Other User Expense"`)
	s.Positive(articleStart, "should find Other User Expense in HTML")

	// Look backwards from that point to find the start of the article tag
	articleTagStart := strings.LastIndex(body[:articleStart], `<article class="expense-item"`)
	s.Positive(articleTagStart, "should find article tag start")

	// Check that floralwhite appears within this article element
	articleSection := body[articleTagStart : articleStart+300]
	s.Contains(articleSection, "floralwhite", "floralwhite should be in the Other User's expense article tag")
}

func (s *ExpenseHandlerTestSuite) TestCreateExpense() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Simulate form submission with current month's date
	form := url.Values{}
	form.Add("amount", "15.00")
	form.Add("description", "Lunch Test")
	form.Add("category", "food")
	// Use current month's date to ensure it appears in ListExpenses (which filters by current month)
	form.Add("date", "2026-01-09T12:00:00")

	req := httptest.NewRequest("POST", "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = s.addUserContext(req)
	w := httptest.NewRecorder()

	h.CreateExpense(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Check for HTMX redirect header
	expectedLoc := `{"path":"/expenses", "target":"#content"}`
	s.Equal(expectedLoc, resp.Header.Get("HX-Location"))

	// Verify DB insertion
	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 1, "expected exactly 1 expense")
	s.Equal("Lunch Test", expenses[0].Description)
	s.InDelta(15.00, expenses[0].Amount, 0.001)
}

func (s *ExpenseHandlerTestSuite) TestCreateExpense_LegacyFormat() {
	h := NewHandlers(s.db, s.templateDir, false)

	form := url.Values{}
	form.Add("amount", "20.00")
	form.Add("description", "Fallback Test")
	form.Add("category", "food")
	form.Add("date", "2026-01-09T12:30") // No seconds

	req := httptest.NewRequest("POST", "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = s.addUserContext(req)
	w := httptest.NewRecorder()

	h.CreateExpense(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 1)
	s.Equal("Fallback Test", expenses[0].Description)
}

func (s *ExpenseHandlerTestSuite) TestCreateExpense_MissingDate() {
	h := NewHandlers(s.db, "dummy_path", false)

	form := url.Values{}
	form.Add("amount", "15.00")
	form.Add("description", "No Date")
	form.Add("category", "food")
	// Missing date

	req := httptest.NewRequest("POST", "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = s.addUserContext(req)
	w := httptest.NewRecorder()

	h.CreateExpense(w, req)

	resp := w.Result()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *ExpenseHandlerTestSuite) TestStatistics_CurrentMonth() {
	h := NewHandlers(s.db, s.templateDir, false)

	// No query params should default to current month
	req := httptest.NewRequest("GET", "/statistics", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	s.Contains(body, "stats-screen", "should contain stats-screen class")
	s.Contains(body, "stat-label", "should contain stat labels")
}

func (s *ExpenseHandlerTestSuite) TestStatistics_WithExpenses() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Create test expenses for January 2026
	testExpenses := []struct {
		amount      float64
		description string
		category    string
		date        string
	}{
		{100.00, "Groceries", "groceries", "2026-01-15T12:00:00"},
		{50.00, "Bus", "transport", "2026-01-16T12:00:00"},
		{75.00, "Restaurant", "eating out", "2026-01-17T12:00:00"},
		{25.00, "More Groceries", "groceries", "2026-01-18T12:00:00"},
	}

	for _, exp := range testExpenses {
		form := url.Values{}
		form.Add("amount", strings.TrimSpace(strings.Split(strings.TrimPrefix(http.StatusText(int(exp.amount*100)), ""), " ")[0]))
		form.Add("amount", http.StatusText(int(exp.amount)))
		// Let's use a simpler approach
		err := s.db.CreateExpense(exp.amount, exp.description, exp.category, parseTestDate(exp.date), 1)
		s.Require().NoError(err, "failed to create test expense")
	}

	// Request statistics for January 2026
	req := httptest.NewRequest("GET", "/statistics?year=2026&month=1", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	s.Contains(body, "January", "should show month name")
	s.Contains(body, "2026", "should show year")
	s.Contains(body, ">250<", "should show total (100+50+75+25=250)")
	s.Contains(body, "groceries", "should show groceries category")
	s.Contains(body, "transport", "should show transport category")
	s.Contains(body, "eating out", "should show eating out category")
	s.Contains(body, "Groceries", "should show expense descriptions")
}

func (s *ExpenseHandlerTestSuite) TestStatistics_EmptyMonth() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Request statistics for a month with no expenses
	req := httptest.NewRequest("GET", "/statistics?year=2025&month=5", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	s.Contains(body, "May", "should show month name")
	s.Contains(body, "2025", "should show year")
	s.Contains(body, "No expenses recorded", "should show empty state message")
}

func (s *ExpenseHandlerTestSuite) TestStatistics_MonthNavigation() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Request statistics for November 2025 (a past month)
	req := httptest.NewRequest("GET", "/statistics?year=2025&month=11", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	// Should have previous month link (October 2025)
	s.Contains(body, "year=2025&month=10", "should have link to previous month")
	// Should have next month link (December 2025)
	s.Contains(body, "year=2025&month=12", "should have link to next month")
}

func (s *ExpenseHandlerTestSuite) TestStatistics_CategoryPercentages() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Create expenses with known percentages
	// Total will be 100, so percentages are easy to verify
	testExpenses := []struct {
		amount   float64
		category string
		date     string
	}{
		{50.00, "groceries", "2026-03-15T12:00:00"},  // 50%
		{30.00, "transport", "2026-03-16T12:00:00"},  // 30%
		{20.00, "eating out", "2026-03-17T12:00:00"}, // 20%
	}

	for _, exp := range testExpenses {
		err := s.db.CreateExpense(exp.amount, "Test", exp.category, parseTestDate(exp.date), 1)
		s.Require().NoError(err)
	}

	req := httptest.NewRequest("GET", "/statistics?year=2026&month=3", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	s.Contains(body, "50.0%", "should show 50% for groceries")
	s.Contains(body, "30.0%", "should show 30% for transport")
	s.Contains(body, "20.0%", "should show 20% for eating out")
}

func (s *ExpenseHandlerTestSuite) TestStatistics_InvalidMonth() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Request with invalid month should default to current month
	req := httptest.NewRequest("GET", "/statistics?year=2026&month=13", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	// Should still return OK, just with current month
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ExpenseHandlerTestSuite) TestBuildMonthView_CurrentMonth_MTDComparison() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Pretend "now" is April 19, 2026. 19 days elapsed in April.
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// Previous month (March 2026): $240 within the first 19 days, $100 after.
	// Only the first $240 should count against the comparison window.
	marchExpenses := []struct {
		amount float64
		date   time.Time
	}{
		{80.00, time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)},
		{80.00, time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)},
		{80.00, time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)},
		{100.00, time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)},
	}
	for _, exp := range marchExpenses {
		s.Require().NoError(s.db.CreateExpense(exp.amount, "Test", "other", exp.date, 1))
	}

	// Current month (April 2026, MTD through Apr 19): $200 total.
	aprilExpenses := []struct {
		amount float64
		date   time.Time
	}{
		{100.00, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)},
		{50.00, time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)},
		{50.00, time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)},
	}
	for _, exp := range aprilExpenses {
		s.Require().NoError(s.db.CreateExpense(exp.amount, "Test", "other", exp.date, 1))
	}

	vm := h.buildMonthView(2026, 4, now)

	s.True(vm.IsCurrentPeriod, "April 2026 should be the current period")
	s.InDelta(200.00, vm.Total, 0.001, "Total reflects MTD (Apr 1-19)")
	s.True(vm.HasChange)
	s.False(vm.IsIncrease, "MTD 200 < previous MTD 240, so it's a decrease")

	// (200 - 240) / 240 * 100 = -16.666...%, abs = 16.666...
	expectedPct := (240.0 - 200.0) / 240.0 * 100.0
	s.InDelta(expectedPct, vm.PercentageChange, 0.01, "+/- compares matching 19-day windows")

	// Average uses elapsed days (19), not the 30 days in April.
	s.InDelta(200.0/19.0, vm.AverageSpending, 0.01, "avg uses elapsed days for current month")
}

func (s *ExpenseHandlerTestSuite) TestBuildMonthView_CurrentMonth_CapsAtPrevMonthLength() {
	// Today is March 31 but February only has 28 days. Both windows should be
	// truncated to 28 days so the comparison stays apples-to-apples.
	h := NewHandlers(s.db, s.templateDir, false)
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	// Feb 1-28 (28 days, $100 each day-range we care about).
	s.Require().NoError(s.db.CreateExpense(100.00, "Feb", "other",
		time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC), 1))

	// March: $50 within first 28 days, $500 on Mar 30 (excluded from comparison).
	s.Require().NoError(s.db.CreateExpense(50.00, "Mar early", "other",
		time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), 1))
	s.Require().NoError(s.db.CreateExpense(500.00, "Mar late", "other",
		time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC), 1))

	vm := h.buildMonthView(2026, 3, now)

	s.True(vm.IsCurrentPeriod)
	s.InDelta(550.00, vm.Total, 0.001, "Total is MTD (all March)")
	s.True(vm.HasChange)
	s.False(vm.IsIncrease, "first 28 days of March ($50) < first 28 days of Feb ($100)")
	s.InDelta(50.0, vm.PercentageChange, 0.01, "both windows capped to 28 days")
	s.InDelta(550.0/31.0, vm.AverageSpending, 0.01, "avg uses 31 elapsed days")
}

func (s *ExpenseHandlerTestSuite) TestBuildMonthView_CompletedMonth_FullComparison() {
	h := NewHandlers(s.db, s.templateDir, false)
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// Full March ($300) vs full February ($200).
	marchDates := []time.Time{
		time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
	}
	for _, d := range marchDates {
		s.Require().NoError(s.db.CreateExpense(100.00, "Mar", "other", d, 1))
	}
	febDates := []time.Time{
		time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
	}
	for _, d := range febDates {
		s.Require().NoError(s.db.CreateExpense(100.00, "Feb", "other", d, 1))
	}

	vm := h.buildMonthView(2026, 3, now)

	s.False(vm.IsCurrentPeriod, "March 2026 is a past period when now is April")
	s.InDelta(300.00, vm.Total, 0.001)
	s.True(vm.HasChange)
	s.True(vm.IsIncrease, "$300 Mar > $200 Feb")
	s.InDelta(50.0, vm.PercentageChange, 0.001, "(300-200)/200 = 50%")
	s.InDelta(300.0/31.0, vm.AverageSpending, 0.01, "past month uses full month length (31 days)")
}

func (s *ExpenseHandlerTestSuite) TestBuildYearView_CurrentYear_YTDComparison() {
	h := NewHandlers(s.db, s.templateDir, false)

	// April 19, 2026 is YearDay 109. Compare [Jan 1, Apr 20) each year.
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// 2025: $300 in Jan-Mar (inside YTD window), $500 in Dec (outside).
	prevExpenses := []struct {
		amount float64
		date   time.Time
	}{
		{100.00, time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)},
		{200.00, time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)},
		{500.00, time.Date(2025, 12, 31, 10, 0, 0, 0, time.UTC)},
	}
	for _, exp := range prevExpenses {
		s.Require().NoError(s.db.CreateExpense(exp.amount, "Test", "other", exp.date, 1))
	}

	// 2026: $150 YTD
	currExpenses := []struct {
		amount float64
		date   time.Time
	}{
		{50.00, time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC)},
		{100.00, time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)},
	}
	for _, exp := range currExpenses {
		s.Require().NoError(s.db.CreateExpense(exp.amount, "Test", "other", exp.date, 1))
	}

	vm := h.buildYearView(2026, now)

	s.True(vm.IsCurrentPeriod)
	s.InDelta(150.00, vm.Total, 0.001, "Total is YTD")
	s.True(vm.HasChange)
	s.False(vm.IsIncrease, "YTD 150 < prev YTD 300")
	s.InDelta(50.0, vm.PercentageChange, 0.01, "(150-300)/300 = -50%")

	// April = month 4 elapsed (SPENT/MTH uses elapsed months for current year).
	s.InDelta(150.0/4.0, vm.AverageSpending, 0.01)
}

func (s *ExpenseHandlerTestSuite) TestBuildYearView_CompletedYear_FullComparison() {
	h := NewHandlers(s.db, s.templateDir, false)
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// 2024 full year: $1200. 2023 full year: $1500.
	s.Require().NoError(s.db.CreateExpense(1200.00, "2024", "other",
		time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC), 1))
	s.Require().NoError(s.db.CreateExpense(1500.00, "2023", "other",
		time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC), 1))

	vm := h.buildYearView(2024, now)

	s.False(vm.IsCurrentPeriod)
	s.InDelta(1200.00, vm.Total, 0.001)
	s.True(vm.HasChange)
	s.False(vm.IsIncrease)
	s.InDelta(20.0, vm.PercentageChange, 0.001, "(1200-1500)/1500 = -20%")
	s.InDelta(1200.0/12.0, vm.AverageSpending, 0.001, "completed year uses 12 months")
}

func (s *ExpenseHandlerTestSuite) TestStatistics_TransactionCount() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Create multiple expenses in same category
	for i := 1; i <= 3; i++ {
		err := s.db.CreateExpense(10.00, "Coffee", "eating out", parseTestDate("2026-04-15T12:00:00").Add(time.Duration(i)*time.Hour), 1)
		s.Require().NoError(err)
	}

	req := httptest.NewRequest("GET", "/statistics?year=2026&month=4", http.NoBody)
	w := httptest.NewRecorder()

	h.Statistics(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	s.Contains(body, "3 transactions", "should show transaction count")
}

func (s *ExpenseHandlerTestSuite) TestDeleteExpense() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Create an expense first
	err := s.db.CreateExpense(50.00, "To Delete", "food", parseTestDate("2026-01-10T12:00:00"), 1)
	s.Require().NoError(err)

	// Get the expense ID
	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 1)
	expenseID := expenses[0].ID

	// Send DELETE request
	req := httptest.NewRequest("DELETE", "/expenses/"+string(rune(expenseID+'0')), http.NoBody)
	req.SetPathValue("id", string(rune(expenseID+'0')))

	// Use a proper path value approach
	req = httptest.NewRequest("DELETE", "/expenses/1", http.NoBody)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	h.DeleteExpense(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Check for HTMX redirect header
	expectedLoc := `{"path":"/expenses", "target":"#content"}`
	s.Equal(expectedLoc, resp.Header.Get("HX-Location"))

	// Verify expense is deleted
	expenses, err = s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Empty(expenses, "expected expense to be deleted")
}

func (s *ExpenseHandlerTestSuite) TestDeleteExpense_NonExistent() {
	h := NewHandlers(s.db, s.templateDir, false)

	// Send DELETE request for non-existent expense
	req := httptest.NewRequest("DELETE", "/expenses/99999", http.NoBody)
	req.SetPathValue("id", "99999")
	w := httptest.NewRecorder()

	h.DeleteExpense(w, req)

	// Should still return OK (no-op for non-existent)
	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *ExpenseHandlerTestSuite) TestIsOtherUserLogic() {
	// Create two users
	user1, err := s.db.CreateUser("user1", "pass1")
	s.Require().NoError(err)

	user2, err := s.db.CreateUser("user2", "pass2")
	s.Require().NoError(err)

	// Create expenses for both users
	date := parseTestDate("2026-01-15T12:00:00")
	err = s.db.CreateExpense(50.00, "User1 Expense", "groceries", date, user1.ID)
	s.Require().NoError(err)

	err = s.db.CreateExpense(30.00, "User2 Expense", "transport", date.Add(time.Hour), user2.ID)
	s.Require().NoError(err)

	// Get all expenses
	expenses, err := s.db.ListExpenses(100, 0)
	s.Require().NoError(err)
	s.Require().Len(expenses, 2)

	// Test the IsOtherUser logic
	for _, e := range expenses {
		if e.Description == "User1 Expense" {
			s.Equal(user1.ID, *e.UserID, "User1 expense should belong to user1")
			isOtherUser := e.UserID != nil && *e.UserID != user1.ID
			s.False(isOtherUser, "From user1's perspective, their own expense should not be IsOtherUser")

			isOtherUserForUser2 := e.UserID != nil && *e.UserID != user2.ID
			s.True(isOtherUserForUser2, "From user2's perspective, user1's expense should be IsOtherUser")
		}
		if e.Description == "User2 Expense" {
			s.Equal(user2.ID, *e.UserID, "User2 expense should belong to user2")
			isOtherUser := e.UserID != nil && *e.UserID != user1.ID
			s.True(isOtherUser, "From user1's perspective, user2's expense should be IsOtherUser")
		}
	}
}

// Helper function to parse test dates
func parseTestDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05", dateStr)
	return t
}

// TestExpenseHandlerSuite runs the expense handler test suite
func TestExpenseHandlerSuite(t *testing.T) {
	suite.Run(t, new(ExpenseHandlerTestSuite))
}
