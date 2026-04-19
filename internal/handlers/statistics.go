package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StatsCategoryItem represents a category with its spending statistics.
type StatsCategoryItem struct {
	Category      string
	Total         float64
	Count         int
	Percentage    float64
	CategoryStyle CategoryStyle
}

// ChartPoint represents a data point in the chart.
type ChartPoint struct {
	Label string
	Value float64
}

// StatsViewModel is the data passed to the statistics view template.
type StatsViewModel struct {
	ViewMode         string
	Year             int
	Month            int
	MonthName        string
	Total            float64
	PercentageChange float64
	IsIncrease       bool
	HasChange        bool
	AverageSpending  float64
	AverageLabel     string
	Categories       []StatsCategoryItem
	Expenses         []ExpenseItem
	ChartData        []ChartPoint
	MaxChartValue    float64
	PrevYear         int
	PrevMonth        int
	NextYear         int
	NextMonth        int
	IsCurrentPeriod  bool
}

// Statistics renders the statistics page.
func (h *Handlers) Statistics(w http.ResponseWriter, r *http.Request) {
	// Get view mode, year, and month from query params
	viewMode := r.URL.Query().Get("view")
	if viewMode == "" {
		viewMode = "month" // Default to month view
	}

	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	var viewModel StatsViewModel

	if viewMode == "year" {
		viewModel = h.buildYearView(year, now)
	} else {
		viewModel = h.buildMonthView(year, month, now)
	}

	h.render(w, r, "stats.html", viewModel)
}

// buildMonthView builds the view model for month view.
func (h *Handlers) buildMonthView(year, month int, now time.Time) StatsViewModel {
	// Get category totals
	categoryTotals, err := h.db.GetCategoryTotalsByMonth(year, month)
	if err != nil {
		log.Printf("GetCategoryTotalsByMonth error: %v", err)
		return StatsViewModel{}
	}

	// Get expenses for the month
	expenses, err := h.db.GetExpensesByMonth(year, month)
	if err != nil {
		log.Printf("GetExpensesByMonth error: %v", err)
		return StatsViewModel{}
	}

	// Get daily totals for chart
	dailyTotals, err := h.db.GetDailyTotalsForMonth(year, month)
	if err != nil {
		log.Printf("GetDailyTotalsForMonth error: %v", err)
	}

	// Check if this is the current month
	isCurrentPeriod := year == now.Year() && month == int(now.Month())

	currentStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	prevStart := currentStart.AddDate(0, -1, 0)

	// Calculate total (MTD for the current month)
	total, _ := h.db.GetTotalForPeriod(year, month)

	// For a fair +/- comparison, align the current and previous windows:
	// - completed month: compare full month vs full previous month
	// - current month: compare MTD vs same day-range of previous month,
	//   truncating both sides to the shorter of the two when necessary
	var prevTotal, compareTotal float64
	if isCurrentPeriod {
		nowUTC := now.UTC()
		daysElapsed := nowUTC.Day()
		daysInPrevMonth := int(currentStart.Sub(prevStart).Hours() / 24)
		daysCmp := min(daysElapsed, daysInPrevMonth)
		compareTotal, _ = h.db.GetTotalForRange(currentStart, currentStart.AddDate(0, 0, daysCmp))
		prevTotal, _ = h.db.GetTotalForRange(prevStart, prevStart.AddDate(0, 0, daysCmp))
	} else {
		compareTotal = total
		prevTotal, _ = h.db.GetTotalForPeriod(prevStart.Year(), int(prevStart.Month()))
	}

	// Calculate percentage change
	percentageChange := 0.0
	hasChange := false
	isIncrease := false
	if prevTotal > 0 {
		hasChange = true
		percentageChange = ((compareTotal - prevTotal) / prevTotal) * 100
		isIncrease = percentageChange > 0
		percentageChange = math.Abs(percentageChange)
	}

	// Calculate average spending per day: elapsed days for the current month,
	// full month length for completed months
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	avgDivisor := daysInMonth
	if isCurrentPeriod {
		avgDivisor = now.UTC().Day()
	}
	averageSpending := 0.0
	if avgDivisor > 0 {
		averageSpending = total / float64(avgDivisor)
	}

	// Build chart data
	chartData := make([]ChartPoint, 0)
	maxValue := 0.0

	// Create a map for quick lookup
	dailyMap := make(map[int]float64)
	for _, dt := range dailyTotals {
		dailyMap[dt.Day] = dt.Total
		if dt.Total > maxValue {
			maxValue = dt.Total
		}
	}

	// Fill in all days
	for day := 1; day <= daysInMonth; day++ {
		value := dailyMap[day]
		label := ""
		if day == 1 || day == 10 || day == 20 || day == daysInMonth {
			label = strconv.Itoa(day)
		}
		chartData = append(chartData, ChartPoint{
			Label: label,
			Value: value,
		})
	}

	// Prepare category items
	categoryItems := make([]StatsCategoryItem, 0, len(categoryTotals))
	for _, ct := range categoryTotals {
		percentage := 0.0
		if total > 0 {
			percentage = (ct.Total / total) * 100
		}
		categoryItems = append(categoryItems, StatsCategoryItem{
			Category:      ct.Category,
			Total:         ct.Total,
			Count:         ct.Count,
			Percentage:    percentage,
			CategoryStyle: getCategoryStyle(ct.Category),
		})
	}

	// Prepare expense items
	expenseItems := make([]ExpenseItem, 0, len(expenses))
	for _, e := range expenses {
		expenseItems = append(expenseItems, ExpenseItem{
			ID:            e.ID,
			Amount:        e.Amount,
			Description:   e.Description,
			Category:      e.Category,
			Time:          e.Date.Format("Jan 02, 15:04"),
			DateTime:      e.Date.Format("2006-01-02T15:04:05"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsIncome:      strings.Contains(e.Description, "[Income]"),
		})
	}

	nextDate := currentStart.AddDate(0, 1, 0)

	monthName := time.Month(month).String()

	return StatsViewModel{
		ViewMode:         "month",
		Year:             year,
		Month:            month,
		MonthName:        monthName,
		Total:            total,
		PercentageChange: percentageChange,
		IsIncrease:       isIncrease,
		HasChange:        hasChange,
		AverageSpending:  averageSpending,
		AverageLabel:     "SPENT/DAY",
		Categories:       categoryItems,
		Expenses:         expenseItems,
		ChartData:        chartData,
		MaxChartValue:    maxValue,
		PrevYear:         prevStart.Year(),
		PrevMonth:        int(prevStart.Month()),
		NextYear:         nextDate.Year(),
		NextMonth:        int(nextDate.Month()),
		IsCurrentPeriod:  isCurrentPeriod,
	}
}

// buildYearView builds the view model for year view.
func (h *Handlers) buildYearView(year int, now time.Time) StatsViewModel {
	// Get category totals for the year
	categoryTotals, err := h.db.GetCategoryTotalsByYear(year)
	if err != nil {
		log.Printf("GetCategoryTotalsByYear error: %v", err)
		return StatsViewModel{}
	}

	// Get expenses for the year
	expenses, err := h.db.GetExpensesByYear(year)
	if err != nil {
		log.Printf("GetExpensesByYear error: %v", err)
		return StatsViewModel{}
	}

	// Get monthly totals for chart
	monthlyTotals, err := h.db.GetMonthlyTotalsForYear(year)
	if err != nil {
		log.Printf("GetMonthlyTotalsForYear error: %v", err)
	}

	// Check if this is the current year
	isCurrentPeriod := year == now.Year()

	currentYearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	prevYearStart := time.Date(year-1, 1, 1, 0, 0, 0, 0, time.UTC)

	// Calculate total (YTD for the current year)
	total, _ := h.db.GetTotalForPeriod(year, 0)

	// For a fair +/- comparison, align the current and previous windows:
	// - completed year: compare full year vs full previous year
	// - current year: compare YTD vs same day-range of previous year,
	//   truncating to the shorter of the two when necessary (leap years)
	var prevTotal, compareTotal float64
	if isCurrentPeriod {
		nowUTC := now.UTC()
		daysElapsed := nowUTC.YearDay()
		daysInPrevYear := int(currentYearStart.Sub(prevYearStart).Hours() / 24)
		daysCmp := min(daysElapsed, daysInPrevYear)
		compareTotal, _ = h.db.GetTotalForRange(currentYearStart, currentYearStart.AddDate(0, 0, daysCmp))
		prevTotal, _ = h.db.GetTotalForRange(prevYearStart, prevYearStart.AddDate(0, 0, daysCmp))
	} else {
		compareTotal = total
		prevTotal, _ = h.db.GetTotalForPeriod(year-1, 0)
	}

	// Calculate percentage change
	percentageChange := 0.0
	hasChange := false
	isIncrease := false
	if prevTotal > 0 {
		hasChange = true
		percentageChange = ((compareTotal - prevTotal) / prevTotal) * 100
		isIncrease = percentageChange > 0
		percentageChange = math.Abs(percentageChange)
	}

	// Calculate average spending per month: elapsed months for the current year,
	// 12 for completed years
	monthsDivisor := 12
	if isCurrentPeriod {
		monthsDivisor = int(now.UTC().Month())
	}
	averageSpending := 0.0
	if monthsDivisor > 0 {
		averageSpending = total / float64(monthsDivisor)
	}

	// Build chart data
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	chartData := make([]ChartPoint, 12)
	maxValue := 0.0

	// Create a map for quick lookup
	monthlyMap := make(map[int]float64)
	for _, mt := range monthlyTotals {
		monthlyMap[mt.Month] = mt.Total
		if mt.Total > maxValue {
			maxValue = mt.Total
		}
	}

	// Fill in all months
	for i := range 12 {
		month := i + 1
		value := monthlyMap[month]
		chartData[i] = ChartPoint{
			Label: monthNames[i],
			Value: value,
		}
	}

	// Prepare category items
	categoryItems := make([]StatsCategoryItem, 0, len(categoryTotals))
	for _, ct := range categoryTotals {
		percentage := 0.0
		if total > 0 {
			percentage = (ct.Total / total) * 100
		}
		categoryItems = append(categoryItems, StatsCategoryItem{
			Category:      ct.Category,
			Total:         ct.Total,
			Count:         ct.Count,
			Percentage:    percentage,
			CategoryStyle: getCategoryStyle(ct.Category),
		})
	}

	// Prepare expense items
	expenseItems := make([]ExpenseItem, 0, len(expenses))
	for _, e := range expenses {
		expenseItems = append(expenseItems, ExpenseItem{
			ID:            e.ID,
			Amount:        e.Amount,
			Description:   e.Description,
			Category:      e.Category,
			Time:          e.Date.Format("Jan 02, 15:04"),
			DateTime:      e.Date.Format("2006-01-02T15:04:05"),
			CategoryStyle: getCategoryStyle(e.Category),
			IsIncome:      strings.Contains(e.Description, "[Income]"),
		})
	}

	return StatsViewModel{
		ViewMode:         "year",
		Year:             year,
		Month:            0,
		MonthName:        strconv.Itoa(year),
		Total:            total,
		PercentageChange: percentageChange,
		IsIncrease:       isIncrease,
		HasChange:        hasChange,
		AverageSpending:  averageSpending,
		AverageLabel:     "SPENT/MTH",
		Categories:       categoryItems,
		Expenses:         expenseItems,
		ChartData:        chartData,
		MaxChartValue:    maxValue,
		PrevYear:         year - 1,
		PrevMonth:        0,
		NextYear:         year + 1,
		NextMonth:        0,
		IsCurrentPeriod:  isCurrentPeriod,
	}
}
