package handlers

import (
	"expense-tracker/internal/models"
	"log"
	"net/http"
	"sort"
	"strconv"
)

const pageSize = 50

// ListExpenses renders the list of expenses with infinite scroll support.
func (h *Handlers) ListExpenses(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse offset parameter for pagination
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Fetch one extra to check if there are more items
	expenses, err := h.db.ListExpenses(pageSize+1, offset)
	if err != nil {
		log.Printf("ListExpenses error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if there are more items
	hasMore := len(expenses) > pageSize
	if hasMore {
		expenses = expenses[:pageSize] // Trim to actual page size
	}

	// Group expenses by date
	groupsMap := make(map[string]*ExpenseGroup)
	for _, e := range expenses {
		dateStr := e.Date.Format("2006-01-02")
		if _, ok := groupsMap[dateStr]; !ok {
			groupsMap[dateStr] = &ExpenseGroup{Date: dateStr, Title: formatGroupTitle(e.Date)}
		}
		group := groupsMap[dateStr]
		group.Total += e.Amount

		// Check if this expense was created by a different user
		isOtherUser := e.UserID != nil && *e.UserID != user.ID

		group.Items = append(group.Items, ExpenseItem{
			ID:            e.ID,
			Amount:        e.Amount,
			Description:   e.Description,
			Category:      e.Category,
			Time:          e.Date.Format("15:04"),
			DateTime:      e.Date.Format("2006-01-02T15:04:05"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsOtherUser:   isOtherUser,
		})
	}

	groups := make([]ExpenseGroup, 0, len(groupsMap))
	for _, g := range groupsMap {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Date > groups[j].Date })

	// Calculate next offset
	nextOffset := 0
	if hasMore {
		nextOffset = offset + pageSize
	}

	viewModel := ListViewModel{
		Groups:     groups,
		NextOffset: nextOffset,
		HasMore:    hasMore,
	}

	// For HTMX requests loading more items, return only the fragment
	if offset > 0 && r.Header.Get("HX-Request") == "true" {
		h.render(w, r, "expense_groups.html", viewModel)
		return
	}

	// For full page load, get the current month total separately
	totalSpent, err := h.db.GetCurrentMonthTotal()
	if err != nil {
		log.Printf("GetCurrentMonthTotal error: %v", err)
		// Continue with 0 total rather than failing
	}
	viewModel.Total = totalSpent

	h.render(w, r, "list.html", viewModel)
}

// CreateExpenseForm renders the form to create a new expense.
func (h *Handlers) CreateExpenseForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "create.html", FormViewModel{
		IsEdit:     false,
		Categories: categories,
	})
}

// EditExpenseForm renders the form to edit an existing expense.
func (h *Handlers) EditExpenseForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if expense, err := h.db.GetExpense(id); err == nil {
		h.render(w, r, "create.html", FormViewModel{
			Expense:       expense,
			IsEdit:        true,
			FormattedDate: expense.Date.Format("2006-01-02T15:04:05"),
			Categories:    categories,
		})
	} else {
		http.Error(w, "Expense not found", http.StatusNotFound)
	}
}

// CreateExpense handles the creation of a new expense.
func (h *Handlers) CreateExpense(w http.ResponseWriter, r *http.Request) {
	amount, desc, cat, date, err := parseForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.db.CreateExpense(amount, desc, cat, date, user.ID); err != nil {
		log.Printf("CreateExpense error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}

// UpdateExpense handles the update of an existing expense.
func (h *Handlers) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	amount, desc, cat, date, err := parseForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.db.UpdateExpense(&models.Expense{
		ID: id, Amount: amount, Description: desc, Category: cat, Date: date,
	}); err != nil {
		log.Printf("UpdateExpense error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}

// DeleteExpense handles the deletion of an expense.
func (h *Handlers) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := h.db.DeleteExpense(id); err != nil {
		log.Printf("DeleteExpense error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", `{"path":"/expenses", "target":"#content"}`)
}
