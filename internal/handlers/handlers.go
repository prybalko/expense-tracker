package handlers

import (
	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
	"time"
)

// Context key type to avoid collisions.
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey contextKey = "user"
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session"
	// SessionDuration is how long sessions last (30 days).
	SessionDuration = 30 * 24 * time.Hour
)

// Handlers holds dependencies for HTTP handlers.
type Handlers struct {
	db           *storage.DB
	templateDir  string
	secureCookie bool
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *storage.DB, templateDir string, secureCookie bool) *Handlers {
	return &Handlers{db: db, templateDir: templateDir, secureCookie: secureCookie}
}

// CategoryDef defines the properties of a category.
type CategoryDef struct {
	ID    string
	Name  string
	Icon  string
	Color string
}

var categories = []CategoryDef{
	{"groceries", "Groceries", "🛒", "#60a5fa"},
	{"eating out", "Eating Out", "🍴", "#60a5fa"},
	{"transport", "Transport", "🚌", "#a78bfa"},
	{"entertainment", "Entertainment", "🎮", "#f472b6"},
	{"utilities", "Utilities", "💡", "#fbbf24"},
	{"housing", "Housing", "🏠", "#818cf8"},
	{"gifts", "Gifts", "🎁", "#fb7185"},
	{"other", "Other", "📦", "#94a3b8"},
}

// CategoryStyle defines the visual style for a category.
type CategoryStyle struct {
	Icon  string
	Color string
}

// ExpenseItem represents an expense in the list view.
type ExpenseItem struct {
	ID            int64
	Amount        float64
	Description   string
	Category      string
	Time          string
	CategoryStyle CategoryStyle
	IsIncome      bool
}

// ExpenseGroup groups expenses by date.
type ExpenseGroup struct {
	Title string
	Date  string
	Total float64
	Items []ExpenseItem
}

// ListViewModel is the data passed to the list view template.
type ListViewModel struct {
	Total  float64
	Groups []ExpenseGroup
}

// FormViewModel is the data passed to the create/edit form template.
type FormViewModel struct {
	Expense       *models.Expense
	IsEdit        bool
	FormattedDate string
	Categories    []CategoryDef
}

// LoginViewModel holds data for the login page.
type LoginViewModel struct {
	Error string
}
