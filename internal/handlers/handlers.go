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

type Handlers struct {
	db          *storage.DB
	templateDir string
}

func NewHandlers(db *storage.DB, templateDir string) *Handlers {
	return &Handlers{db: db, templateDir: templateDir}
}

// ... (keep CategoryStyle and getCategoryStyle as they are used in List) ...
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

type ExpenseItem struct {
	models.Expense
	Time          string
	CategoryStyle CategoryStyle
	IsIncome      bool
}

type ExpenseGroup struct {
	Title string
	Date  string
	Total float64
	Items []ExpenseItem
}

type ListViewModel struct {
	Total  float64
	Groups []ExpenseGroup
}

type FormViewModel struct {
	Expense       *models.Expense
	IsEdit        bool
	FormattedDate string
}

func (h *Handlers) ListExpenses(w http.ResponseWriter, r *http.Request) {
	expenses, err := h.db.ListExpenses()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	groupsMap := make(map[string]*ExpenseGroup)
	var totalSpent float64

	for _, e := range expenses {
		dateStr := e.CreatedAt.Format("2006-01-02")
		if _, ok := groupsMap[dateStr]; !ok {
			groupsMap[dateStr] = &ExpenseGroup{Date: dateStr, Title: formatGroupTitle(e.CreatedAt)}
		}
		group := groupsMap[dateStr]
		group.Total += e.Amount
		totalSpent += e.Amount

		group.Items = append(group.Items, ExpenseItem{
			Expense:       e,
			Time:          e.CreatedAt.Format("15:04"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsIncome:      strings.Contains(e.Description, "[Income]"),
		})
	}

	groups := make([]ExpenseGroup, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Date > groups[j].Date })

	h.render(w, r, "list.html", ListViewModel{Total: totalSpent, Groups: groups})
}

func (h *Handlers) CreateExpenseForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "create.html", FormViewModel{IsEdit: false})
}

func (h *Handlers) EditExpenseForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if expense, err := h.db.GetExpense(id); err == nil {
		h.render(w, r, "create.html", FormViewModel{
			Expense:       expense,
			IsEdit:        true,
			FormattedDate: expense.CreatedAt.Format("2006-01-02T15:04"),
		})
	} else {
		http.Error(w, "Expense not found", http.StatusNotFound)
	}
}

func (h *Handlers) CreateExpense(w http.ResponseWriter, r *http.Request) {
	amount, desc, cat, date, err := parseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	if err := h.db.CreateExpense(amount, desc, cat, date); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}

func (h *Handlers) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	amount, desc, cat, date, err := parseForm(r)
	if err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	if err := h.db.UpdateExpense(&models.Expense{
		ID: id, Amount: amount, Description: desc, Category: cat, CreatedAt: date,
	}); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}

func parseForm(r *http.Request) (float64, string, string, time.Time, error) {
	if err := r.ParseForm(); err != nil {
		return 0, "", "", time.Time{}, err
	}
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	desc := r.FormValue("description")
	if desc == "" {
		desc = "Expense"
	}
	dateStr := r.FormValue("date")
	date, err := time.Parse("2006-01-02T15:04", dateStr)
	if err != nil {
		date, _ = time.Parse(time.RFC3339, dateStr)
		if date.IsZero() {
			date = time.Now()
		}
	}
	return amount, desc, r.FormValue("category"), date, nil
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, viewName string, data any) {
	tmpl, err := template.ParseFiles(filepath.Join(h.templateDir, "base.html"), filepath.Join(h.templateDir, viewName))
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	target := "base.html"
	if r.Header.Get("HX-Request") == "true" {
		target = "content"
	}
	tmpl.ExecuteTemplate(w, target, data)
}

func formatGroupTitle(date time.Time) string {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if date.Truncate(24 * time.Hour).Equal(today) {
		return "TODAY"
	}
	if date.Truncate(24 * time.Hour).Equal(today.AddDate(0, 0, -1)) {
		return "YESTERDAY"
	}
	return strings.ToUpper(date.Format("Mon, 02 Jan '06"))
}
