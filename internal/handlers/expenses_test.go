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
	w := httptest.NewRecorder()

	h.ListExpenses(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Simple content check - look for "Spent this month" which is in list.html
	body := w.Body.String()
	s.Contains(body, "Spent this month")
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
	expenses, err := s.db.ListExpenses()
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

	expenses, err := s.db.ListExpenses()
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
	s.Contains(body, "Statistics", "should contain Statistics heading")
	s.Contains(body, "Total spent", "should contain total spent label")
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
	s.Contains(body, "250.00", "should show total (100+50+75+25=250)")
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
		{50.00, "groceries", "2026-03-15T12:00:00"},    // 50%
		{30.00, "transport", "2026-03-16T12:00:00"},    // 30%
		{20.00, "eating out", "2026-03-17T12:00:00"},   // 20%
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
	expenses, err := s.db.ListExpenses()
	s.Require().NoError(err)
	s.Require().Len(expenses, 1)
	expenseID := expenses[0].ID

	// Send DELETE request
	req := httptest.NewRequest("DELETE", "/expenses/"+string(rune(expenseID+'0')), http.NoBody)
	req.SetPathValue("id", string(rune(expenseID + '0')))
	w := httptest.NewRecorder()

	// Use a proper path value approach
	req = httptest.NewRequest("DELETE", "/expenses/1", http.NoBody)
	req.SetPathValue("id", "1")
	w = httptest.NewRecorder()

	h.DeleteExpense(w, req)

	resp := w.Result()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Check for HTMX redirect header
	expectedLoc := `{"path":"/expenses", "target":"#content"}`
	s.Equal(expectedLoc, resp.Header.Get("HX-Location"))

	// Verify expense is deleted
	expenses, err = s.db.ListExpenses()
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

// Helper function to parse test dates
func parseTestDate(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05", dateStr)
	return t
}

// TestExpenseHandlerSuite runs the expense handler test suite
func TestExpenseHandlerSuite(t *testing.T) {
	suite.Run(t, new(ExpenseHandlerTestSuite))
}
