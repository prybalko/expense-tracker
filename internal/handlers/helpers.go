package handlers

import (
	"errors"
	"expense-tracker/internal/models"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetUserFromContext retrieves the authenticated user from request context.
func GetUserFromContext(r *http.Request) *models.User {
	if user, ok := r.Context().Value(UserContextKey).(*models.User); ok {
		return user
	}
	return nil
}

func getCategoryStyle(category string) CategoryStyle {
	catLower := strings.ToLower(category)
	for _, c := range categories {
		if c.ID == catLower {
			return CategoryStyle{Icon: c.Icon, Color: c.Color}
		}
	}
	return CategoryStyle{Icon: "📦", Color: "#94a3b8"}
}

func parseForm(r *http.Request) (amount float64, desc, category string, date time.Time, err error) {
	if err := r.ParseForm(); err != nil {
		return 0, "", "", time.Time{}, err
	}
	amount, _ = strconv.ParseFloat(r.FormValue("amount"), 64)
	desc = r.FormValue("description")
	if desc == "" {
		desc = "Expense"
	}
	category = r.FormValue("category")
	dateStr := r.FormValue("date")
	if dateStr == "" {
		return 0, "", "", time.Time{}, errors.New("date is required")
	}
	date, err = time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		// Fallback to minutes if seconds are missing
		date, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			return 0, "", "", time.Time{}, err
		}
	}
	return amount, desc, category, date, nil
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
	if err := tmpl.ExecuteTemplate(w, target, data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func formatGroupTitle(date time.Time) string {
	dateStr := date.Format("2006-01-02")
	nowStr := time.Now().Format("2006-01-02")

	if dateStr == nowStr {
		return "TODAY"
	}
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if dateStr == yesterdayStr {
		return "YESTERDAY"
	}
	return strings.ToUpper(date.Format("Mon, 02 Jan '06"))
}
