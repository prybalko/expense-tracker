package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"expense-tracker/internal/storage"

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
	form.Add("date", "2026-01-09T12:00")

	req := httptest.NewRequest("POST", "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func (s *ExpenseHandlerTestSuite) TestCreateExpense_MissingDate() {
	h := NewHandlers(s.db, "dummy_path", false)

	form := url.Values{}
	form.Add("amount", "15.00")
	form.Add("description", "No Date")
	form.Add("category", "food")
	// Missing date

	req := httptest.NewRequest("POST", "/expenses", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.CreateExpense(w, req)

	resp := w.Result()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

// TestExpenseHandlerSuite runs the expense handler test suite
func TestExpenseHandlerSuite(t *testing.T) {
	suite.Run(t, new(ExpenseHandlerTestSuite))
}
