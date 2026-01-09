package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"expense-tracker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ExpenseHandlerTestSuite provides a test suite for expense handler tests
type ExpenseHandlerTestSuite struct {
	suite.Suite
	db          *storage.DB
	templateDir string
}

// SetupTest runs before each test
func (suite *ExpenseHandlerTestSuite) SetupTest() {
	db, err := storage.NewDB(":memory:")
	require.NoError(suite.T(), err, "failed to create test database")
	suite.db = db

	suite.templateDir = "../../web/templates"
	if _, err := os.Stat(suite.templateDir); os.IsNotExist(err) {
		suite.T().Skip("Template directory not found, skipping handler integration test")
	}
}

// TearDownTest runs after each test
func (suite *ExpenseHandlerTestSuite) TearDownTest() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *ExpenseHandlerTestSuite) TestListExpenses() {
	h := NewHandlers(suite.db, suite.templateDir, false)

	req := httptest.NewRequest("GET", "/expenses", http.NoBody)
	w := httptest.NewRecorder()

	h.ListExpenses(w, req)

	resp := w.Result()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Simple content check - look for "Spent this month" which is in list.html
	body := w.Body.String()
	assert.Contains(suite.T(), body, "Spent this month")
}

func (suite *ExpenseHandlerTestSuite) TestCreateExpense() {
	h := NewHandlers(suite.db, suite.templateDir, false)

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
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Check for HTMX redirect header
	expectedLoc := `{"path":"/expenses", "target":"#content"}`
	assert.Equal(suite.T(), expectedLoc, resp.Header.Get("HX-Location"))

	// Verify DB insertion
	expenses, err := suite.db.ListExpenses()
	require.NoError(suite.T(), err)
	require.Len(suite.T(), expenses, 1, "expected exactly 1 expense")
	assert.Equal(suite.T(), "Lunch Test", expenses[0].Description)
	assert.Equal(suite.T(), 15.00, expenses[0].Amount)
}

func (suite *ExpenseHandlerTestSuite) TestCreateExpense_MissingDate() {
	h := NewHandlers(suite.db, "dummy_path", false)

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
	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

// TestExpenseHandlerSuite runs the expense handler test suite
func TestExpenseHandlerSuite(t *testing.T) {
	suite.Run(t, new(ExpenseHandlerTestSuite))
}
