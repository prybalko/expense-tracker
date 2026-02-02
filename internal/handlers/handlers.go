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
	Name  string
	Icon  string
	Color string
}

var categories = []CategoryDef{
	{"Groceries", "🛒", "#60a5fa"},
	{"Eating Out", "🍴", "#60a5fa"},
	{"Transport", "🚌", "#a78bfa"},
	{"Housing", "🏠", "#818cf8"},
	{"Utilities", "💡", "#fbbf24"},
	{"Sport", "🏋️‍♂️", "#fbbf24"},
	{"Health", "🚑", "#fbbf24"},
	{"Entertainment", "🎮", "#f472b6"},
	{"Travel", "✈️", "#f472b6"},
	{"Gifts", "🎁", "#fb7185"},
	{"Other", "📦", "#94a3b8"},
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
	DateTime      string // Full datetime for edit modal (2006-01-02T15:04:05)
	CategoryStyle CategoryStyle
	IsIncome      bool
	IsOtherUser   bool // True if this expense was created by a different user
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
	Total      float64
	Groups     []ExpenseGroup
	NextOffset int  // Offset for loading more items (0 means no more)
	HasMore    bool // Whether there are more items to load
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
