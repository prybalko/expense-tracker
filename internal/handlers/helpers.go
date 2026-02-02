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
	for _, c := range categories {
		if c.Name == category {
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
	category = r.FormValue("category")
	desc = r.FormValue("description")
	if desc == "" {
		desc = category
	}
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
	// For fragment templates (partials), render them directly
	if viewName == "expense_groups.html" {
		filePath := filepath.Join(h.templateDir, viewName)
		tmpl, err := template.ParseFiles(filePath)
		if err != nil {
			log.Printf("Template parse error for %s: %v", filePath, err)
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "expense_groups", data); err != nil {
			log.Printf("Template execution error for %s: %v", viewName, err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	// Build list of template files to parse
	files := []string{
		filepath.Join(h.templateDir, "base.html"),
		filepath.Join(h.templateDir, viewName),
	}

	// Include expense_groups partial for list view
	if viewName == "list.html" {
		files = append(files, filepath.Join(h.templateDir, "expense_groups.html"))
	}

	tmpl, err := template.ParseFiles(files...)
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
