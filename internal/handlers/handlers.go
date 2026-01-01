package handlers

import (
	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Handlers struct holds the database connection and template directory.
type Handlers struct {
	db          *storage.DB
	templateDir string
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *storage.DB, templateDir string) *Handlers {
	return &Handlers{
		db:          db,
		templateDir: templateDir,
	}
}

// CategoryStyle defines the icon and background color for a category.
type CategoryStyle struct {
	Icon string
	Bg   string
}

func getCategoryStyle(category string) CategoryStyle {
	styles := map[string]CategoryStyle{
		"food":          {"🍽️", "cat-food"},
		"transport":     {"🚌", "cat-transport"},
		"entertainment": {"🎮", "cat-entertainment"},
		"utilities":     {"💡", "cat-utilities"},
		"housing":       {"🏠", "cat-housing"},
		"gifts":         {"🎁", "cat-gifts"},
		"other":         {"📦", "cat-other"},
	}
	if style, ok := styles[strings.ToLower(category)]; ok {
		return style
	}
	return styles["other"]
}

// ExpenseItem represents a single expense item in the view.
type ExpenseItem struct {
	models.Expense
	Time          string
	CategoryStyle CategoryStyle
	IsIncome      bool
}

// ExpenseGroup represents a group of expenses for a specific date.
type ExpenseGroup struct {
	Title string
	Date  string
	Total float64
	Items []ExpenseItem
}

// ListViewModel is the data model for the list view.
type ListViewModel struct {
	Total  float64
	Groups []ExpenseGroup
}

// ListExpenses handles the request to list expenses.
func (h *Handlers) ListExpenses(w http.ResponseWriter, r *http.Request) {
	expenses, err := h.db.ListExpenses()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Grouping logic
	groupsMap := make(map[string]*ExpenseGroup)
	var totalSpent float64

	for _, e := range expenses {
		dateStr := e.CreatedAt.Format("2006-01-02")
		
		if _, ok := groupsMap[dateStr]; !ok {
			groupsMap[dateStr] = &ExpenseGroup{
				Date: dateStr,
				Title: formatGroupTitle(e.CreatedAt),
			}
		}
		
		group := groupsMap[dateStr]
		group.Total += e.Amount
		totalSpent += e.Amount

		item := ExpenseItem{
			Expense:       e,
			Time:          e.CreatedAt.Format("15:04"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsIncome:      strings.Contains(e.Description, "[Income]"),
		}
		group.Items = append(group.Items, item)
	}

	// Convert map to slice and sort
	groups := make([]ExpenseGroup, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}
	
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Date > groups[j].Date
	})

	data := ListViewModel{
		Total:  totalSpent,
		Groups: groups,
	}

    h.render(w, r, "list.html", data)
}

// CreateExpenseForm renders the form to create a new expense.
func (h *Handlers) CreateExpenseForm(w http.ResponseWriter, r *http.Request) {
    h.render(w, r, "create.html", nil)
}

// CreateExpense handles the creation of a new expense.
func (h *Handlers) CreateExpense(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	description := r.FormValue("description")
	category := r.FormValue("category")
	dateStr := r.FormValue("date") // Expecting ISO string or similar

    if description == "" {
        description = "Expense"
    }

	date, err := time.Parse("2006-01-02T15:04", dateStr) // datetime-local format usually
    if err != nil {
        // try ISO if JS sends full ISO
        date, _ = time.Parse(time.RFC3339, dateStr)
        if date.IsZero() {
            date = time.Now()
        }
    }

	if err := h.db.CreateExpense(amount, description, category, date); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

    // After creation, redirect to list.
    // Use JSON to specify target explicitly so the #content wrapper is preserved
    w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
    w.WriteHeader(http.StatusOK)
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, viewName string, data any) {
	// Updated path to templates
	basePath := filepath.Join(h.templateDir, "base.html")
	viewPath := filepath.Join(h.templateDir, viewName)
	
	tmpl, err := template.ParseFiles(basePath, viewPath)
	if err != nil {
		log.Printf("Template parsing error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	target := "base.html"
	if r.Header.Get("HX-Request") == "true" {
		target = "content"
	}

	err = tmpl.ExecuteTemplate(w, target, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}


func formatGroupTitle(date time.Time) string {
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    yesterday := today.AddDate(0, 0, -1)
    
    check := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
    
    if check.Equal(today) {
        return "TODAY"
    }
    if check.Equal(yesterday) {
        return "YESTERDAY"
    }
    
    // TUE, 30 DEC '25
    return strings.ToUpper(date.Format("Mon, 02 Jan '06"))
}
