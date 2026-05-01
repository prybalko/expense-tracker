// Package insights computes spending statistics from the storage layer
// without any HTML or HTTP concerns. The JSON API consumes these structs
// directly.
package insights

import (
	"math"
	"strconv"
	"time"

	"expense-tracker/internal/storage"
)

// CategoryBreakdown is a per-category rollup for a period.
type CategoryBreakdown struct {
	Category   string  `json:"category"`
	Total      float64 `json:"total"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ChartPoint is a single bar in the daily/monthly time series.
type ChartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Insights is the combined statistics payload for a month or year.
type Insights struct {
	ViewMode         string              `json:"viewMode"`
	Year             int                 `json:"year"`
	Month            int                 `json:"month"`
	MonthName        string              `json:"monthName"`
	Total            float64             `json:"total"`
	PercentageChange float64             `json:"percentageChange"`
	IsIncrease       bool                `json:"isIncrease"`
	HasChange        bool                `json:"hasChange"`
	AverageSpending  float64             `json:"averageSpending"`
	AverageLabel     string              `json:"averageLabel"`
	Categories       []CategoryBreakdown `json:"categories"`
	Chart            []ChartPoint        `json:"chart"`
	MaxChartValue    float64             `json:"maxChartValue"`
	IsCurrentPeriod  bool                `json:"isCurrentPeriod"`
	PrevYear         int                 `json:"prevYear"`
	PrevMonth        int                 `json:"prevMonth"`
	NextYear         int                 `json:"nextYear"`
	NextMonth        int                 `json:"nextMonth"`
}

var monthNames = []string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

// Month computes Insights for the given (year, month) using `now` as the
// "current time" reference (so the caller can inject a clock for tests).
func Month(db *storage.DB, year, month int, now time.Time) (Insights, error) {
	categoryTotals, err := db.GetCategoryTotalsByMonth(year, month)
	if err != nil {
		return Insights{}, err
	}

	dailyTotals, err := db.GetDailyTotalsForMonth(year, month)
	if err != nil {
		return Insights{}, err
	}

	isCurrentPeriod := year == now.Year() && month == int(now.Month())
	currentStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	prevStart := currentStart.AddDate(0, -1, 0)

	total, err := db.GetTotalForPeriod(year, month)
	if err != nil {
		return Insights{}, err
	}

	var prevTotal, compareTotal float64
	if isCurrentPeriod {
		nowUTC := now.UTC()
		daysElapsed := nowUTC.Day()
		daysInPrevMonth := int(currentStart.Sub(prevStart).Hours() / 24)
		daysCmp := min(daysElapsed, daysInPrevMonth)
		compareTotal, err = db.GetTotalForRange(currentStart, currentStart.AddDate(0, 0, daysCmp))
		if err != nil {
			return Insights{}, err
		}
		prevTotal, err = db.GetTotalForRange(prevStart, prevStart.AddDate(0, 0, daysCmp))
		if err != nil {
			return Insights{}, err
		}
	} else {
		compareTotal = total
		prevTotal, err = db.GetTotalForPeriod(prevStart.Year(), int(prevStart.Month()))
		if err != nil {
			return Insights{}, err
		}
	}

	percentageChange, isIncrease, hasChange := percentChange(compareTotal, prevTotal)

	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	avgDivisor := daysInMonth
	if isCurrentPeriod {
		avgDivisor = now.UTC().Day()
	}
	averageSpending := 0.0
	if avgDivisor > 0 {
		averageSpending = total / float64(avgDivisor)
	}

	chart, maxValue := buildDailyChart(dailyTotals, daysInMonth)
	categories := buildCategories(categoryTotals, total)

	nextDate := currentStart.AddDate(0, 1, 0)

	return Insights{
		ViewMode:         "month",
		Year:             year,
		Month:            month,
		MonthName:        time.Month(month).String(),
		Total:            total,
		PercentageChange: percentageChange,
		IsIncrease:       isIncrease,
		HasChange:        hasChange,
		AverageSpending:  averageSpending,
		AverageLabel:     "SPENT/DAY",
		Categories:       categories,
		Chart:            chart,
		MaxChartValue:    maxValue,
		IsCurrentPeriod:  isCurrentPeriod,
		PrevYear:         prevStart.Year(),
		PrevMonth:        int(prevStart.Month()),
		NextYear:         nextDate.Year(),
		NextMonth:        int(nextDate.Month()),
	}, nil
}

// Year computes Insights for an entire year, using monthly buckets as the
// chart series.
func Year(db *storage.DB, year int, now time.Time) (Insights, error) {
	categoryTotals, err := db.GetCategoryTotalsByYear(year)
	if err != nil {
		return Insights{}, err
	}

	monthlyTotals, err := db.GetMonthlyTotalsForYear(year)
	if err != nil {
		return Insights{}, err
	}

	isCurrentPeriod := year == now.Year()
	currentYearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	prevYearStart := time.Date(year-1, 1, 1, 0, 0, 0, 0, time.UTC)

	total, err := db.GetTotalForPeriod(year, 0)
	if err != nil {
		return Insights{}, err
	}

	var prevTotal, compareTotal float64
	if isCurrentPeriod {
		nowUTC := now.UTC()
		daysElapsed := nowUTC.YearDay()
		daysInPrevYear := int(currentYearStart.Sub(prevYearStart).Hours() / 24)
		daysCmp := min(daysElapsed, daysInPrevYear)
		compareTotal, err = db.GetTotalForRange(currentYearStart, currentYearStart.AddDate(0, 0, daysCmp))
		if err != nil {
			return Insights{}, err
		}
		prevTotal, err = db.GetTotalForRange(prevYearStart, prevYearStart.AddDate(0, 0, daysCmp))
		if err != nil {
			return Insights{}, err
		}
	} else {
		compareTotal = total
		prevTotal, err = db.GetTotalForPeriod(year-1, 0)
		if err != nil {
			return Insights{}, err
		}
	}

	percentageChange, isIncrease, hasChange := percentChange(compareTotal, prevTotal)

	monthsDivisor := 12
	if isCurrentPeriod {
		monthsDivisor = int(now.UTC().Month())
	}
	averageSpending := 0.0
	if monthsDivisor > 0 {
		averageSpending = total / float64(monthsDivisor)
	}

	chart, maxValue := buildMonthlyChart(monthlyTotals)
	categories := buildCategories(categoryTotals, total)

	return Insights{
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
		Categories:       categories,
		Chart:            chart,
		MaxChartValue:    maxValue,
		IsCurrentPeriod:  isCurrentPeriod,
		PrevYear:         year - 1,
		PrevMonth:        0,
		NextYear:         year + 1,
		NextMonth:        0,
	}, nil
}

func percentChange(current, previous float64) (pct float64, isIncrease, hasChange bool) {
	if previous <= 0 {
		return 0, false, false
	}
	pct = ((current - previous) / previous) * 100
	isIncrease = pct > 0
	return math.Abs(pct), isIncrease, true
}

func buildDailyChart(dailyTotals []storage.DailyTotal, daysInMonth int) (chart []ChartPoint, maxValue float64) {
	dailyMap := make(map[int]float64, len(dailyTotals))
	for _, dt := range dailyTotals {
		dailyMap[dt.Day] = dt.Total
		if dt.Total > maxValue {
			maxValue = dt.Total
		}
	}

	chart = make([]ChartPoint, 0, daysInMonth)
	for day := 1; day <= daysInMonth; day++ {
		label := ""
		if day == 1 || day == 10 || day == 20 || day == daysInMonth {
			label = strconv.Itoa(day)
		}
		chart = append(chart, ChartPoint{Label: label, Value: dailyMap[day]})
	}
	return chart, maxValue
}

func buildMonthlyChart(monthlyTotals []storage.MonthlyTotal) (chart []ChartPoint, maxValue float64) {
	monthlyMap := make(map[int]float64, len(monthlyTotals))
	for _, mt := range monthlyTotals {
		monthlyMap[mt.Month] = mt.Total
		if mt.Total > maxValue {
			maxValue = mt.Total
		}
	}

	chart = make([]ChartPoint, 12)
	for i := range 12 {
		chart[i] = ChartPoint{Label: monthNames[i], Value: monthlyMap[i+1]}
	}
	return chart, maxValue
}

func buildCategories(totals []storage.CategoryTotal, grandTotal float64) []CategoryBreakdown {
	out := make([]CategoryBreakdown, 0, len(totals))
	for _, ct := range totals {
		percentage := 0.0
		if grandTotal > 0 {
			percentage = (ct.Total / grandTotal) * 100
		}
		out = append(out, CategoryBreakdown{
			Category:   ct.Category,
			Total:      ct.Total,
			Count:      ct.Count,
			Percentage: percentage,
		})
	}
	return out
}
