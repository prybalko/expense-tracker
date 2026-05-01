package insights

import (
	"math"
	"testing"
	"time"

	"expense-tracker/internal/storage"
)

func newTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seed(t *testing.T, db *storage.DB, amount float64, desc, cat string, date time.Time) {
	t.Helper()
	if err := db.CreateExpense(amount, desc, cat, date, 1); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}
}

func TestMonthCompletedPeriod(t *testing.T) {
	db := newTestDB(t)

	// February 2024 (completed): 100 in Groceries, 50 in Transport.
	seed(t, db, 100, "Feb groceries", "Groceries", time.Date(2024, 2, 5, 12, 0, 0, 0, time.UTC))
	seed(t, db, 50, "Feb transit", "Transport", time.Date(2024, 2, 10, 12, 0, 0, 0, time.UTC))
	// January 2024 (previous): 100 total.
	seed(t, db, 100, "Jan groceries", "Groceries", time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))

	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC) // far in the future
	got, err := Month(db, 2024, 2, now)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}

	if got.ViewMode != "month" {
		t.Errorf("ViewMode = %q, want month", got.ViewMode)
	}
	if got.Total != 150 {
		t.Errorf("Total = %v, want 150", got.Total)
	}
	if got.IsCurrentPeriod {
		t.Error("IsCurrentPeriod = true, want false (Feb 2024 with now in Jun 2024)")
	}
	if !got.HasChange || !got.IsIncrease {
		t.Errorf("HasChange=%v IsIncrease=%v, want true,true (150 vs 100)", got.HasChange, got.IsIncrease)
	}
	// 150 vs 100 is +50%; the struct stores the absolute value.
	if math.Abs(got.PercentageChange-50) > 1e-9 {
		t.Errorf("PercentageChange = %v, want 50", got.PercentageChange)
	}
	// February 2024 has 29 days (leap year).
	wantAvg := 150.0 / 29.0
	if math.Abs(got.AverageSpending-wantAvg) > 1e-9 {
		t.Errorf("AverageSpending = %v, want %v", got.AverageSpending, wantAvg)
	}
	if got.AverageLabel != "SPENT/DAY" {
		t.Errorf("AverageLabel = %q, want SPENT/DAY", got.AverageLabel)
	}
	if len(got.Chart) != 29 {
		t.Errorf("len(Chart) = %d, want 29", len(got.Chart))
	}
	if got.MaxChartValue != 100 {
		t.Errorf("MaxChartValue = %v, want 100", got.MaxChartValue)
	}
	if len(got.Categories) != 2 {
		t.Errorf("len(Categories) = %d, want 2", len(got.Categories))
	}
	if got.PrevYear != 2024 || got.PrevMonth != 1 {
		t.Errorf("Prev = %d/%d, want 2024/1", got.PrevYear, got.PrevMonth)
	}
	if got.NextYear != 2024 || got.NextMonth != 3 {
		t.Errorf("Next = %d/%d, want 2024/3", got.NextYear, got.NextMonth)
	}
}

func TestMonthCurrentPeriodMTDCompare(t *testing.T) {
	db := newTestDB(t)

	// Current month: March 2024, observed up to day 10.
	// Day 5: 60, Day 9: 40 → MTD total = 100 (both within first 10 days).
	seed(t, db, 60, "Mar a", "Groceries", time.Date(2024, 3, 5, 9, 0, 0, 0, time.UTC))
	seed(t, db, 40, "Mar b", "Transport", time.Date(2024, 3, 9, 9, 0, 0, 0, time.UTC))
	// Previous month February 2024, first 10 days = 50; later days ignored for MTD compare.
	seed(t, db, 50, "Feb in window", "Groceries", time.Date(2024, 2, 3, 9, 0, 0, 0, time.UTC))
	seed(t, db, 999, "Feb late", "Groceries", time.Date(2024, 2, 25, 9, 0, 0, 0, time.UTC))

	now := time.Date(2024, 3, 10, 23, 30, 0, 0, time.UTC)
	got, err := Month(db, 2024, 3, now)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if !got.IsCurrentPeriod {
		t.Error("IsCurrentPeriod = false, want true")
	}
	if got.Total != 100 {
		t.Errorf("Total = %v, want 100", got.Total)
	}
	// Average over elapsed days (10), not 31.
	wantAvg := 100.0 / 10.0
	if math.Abs(got.AverageSpending-wantAvg) > 1e-9 {
		t.Errorf("AverageSpending = %v, want %v", got.AverageSpending, wantAvg)
	}
	// Compare 100 (current MTD) vs 50 (prev month, first 10 days) → +100%.
	if !got.HasChange || !got.IsIncrease {
		t.Errorf("HasChange=%v IsIncrease=%v, want true,true", got.HasChange, got.IsIncrease)
	}
	if math.Abs(got.PercentageChange-100) > 1e-9 {
		t.Errorf("PercentageChange = %v, want 100", got.PercentageChange)
	}
}

func TestMonthNoPriorData(t *testing.T) {
	db := newTestDB(t)
	seed(t, db, 25, "Solo", "Groceries", time.Date(2024, 4, 5, 9, 0, 0, 0, time.UTC))

	got, err := Month(db, 2024, 4, time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if got.HasChange {
		t.Error("HasChange = true, want false when prev period had nothing")
	}
	if got.PercentageChange != 0 || got.IsIncrease {
		t.Errorf("Pct=%v IsIncrease=%v, want 0,false", got.PercentageChange, got.IsIncrease)
	}
}

func TestYearCompletedPeriod(t *testing.T) {
	db := newTestDB(t)

	// 2023 — completed year.
	seed(t, db, 200, "March", "Groceries", time.Date(2023, 3, 1, 9, 0, 0, 0, time.UTC))
	seed(t, db, 100, "September", "Transport", time.Date(2023, 9, 1, 9, 0, 0, 0, time.UTC))
	// 2022 — previous year.
	seed(t, db, 100, "Old", "Groceries", time.Date(2022, 6, 1, 9, 0, 0, 0, time.UTC))

	now := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	got, err := Year(db, 2023, now)
	if err != nil {
		t.Fatalf("Year: %v", err)
	}

	if got.ViewMode != "year" {
		t.Errorf("ViewMode = %q, want year", got.ViewMode)
	}
	if got.Total != 300 {
		t.Errorf("Total = %v, want 300", got.Total)
	}
	if got.IsCurrentPeriod {
		t.Error("IsCurrentPeriod = true, want false")
	}
	if !got.HasChange || !got.IsIncrease {
		t.Errorf("HasChange=%v IsIncrease=%v, want true,true (300 vs 100)", got.HasChange, got.IsIncrease)
	}
	if math.Abs(got.PercentageChange-200) > 1e-9 {
		t.Errorf("PercentageChange = %v, want 200", got.PercentageChange)
	}
	if got.AverageLabel != "SPENT/MTH" {
		t.Errorf("AverageLabel = %q, want SPENT/MTH", got.AverageLabel)
	}
	wantAvg := 300.0 / 12.0
	if math.Abs(got.AverageSpending-wantAvg) > 1e-9 {
		t.Errorf("AverageSpending = %v, want %v", got.AverageSpending, wantAvg)
	}
	if len(got.Chart) != 12 {
		t.Errorf("len(Chart) = %d, want 12", len(got.Chart))
	}
	// March (index 2) should hold 200, September (index 8) 100.
	if got.Chart[2].Value != 200 || got.Chart[8].Value != 100 {
		t.Errorf("Chart Mar=%v Sep=%v, want 200,100", got.Chart[2].Value, got.Chart[8].Value)
	}
	if got.MaxChartValue != 200 {
		t.Errorf("MaxChartValue = %v, want 200", got.MaxChartValue)
	}
	if got.PrevYear != 2022 || got.NextYear != 2024 {
		t.Errorf("nav = %d/%d, want 2022/2024", got.PrevYear, got.NextYear)
	}
}

func TestCategoryPercentages(t *testing.T) {
	db := newTestDB(t)
	seed(t, db, 75, "A", "Groceries", time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC))
	seed(t, db, 25, "B", "Transport", time.Date(2024, 5, 2, 9, 0, 0, 0, time.UTC))

	got, err := Month(db, 2024, 5, time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(got.Categories) != 2 {
		t.Fatalf("len(Categories) = %d, want 2", len(got.Categories))
	}
	// First entry is the bigger category (sorted by total DESC at the SQL layer).
	if got.Categories[0].Category != "Groceries" || math.Abs(got.Categories[0].Percentage-75) > 1e-9 {
		t.Errorf("top category = %+v, want Groceries 75%%", got.Categories[0])
	}
	if got.Categories[1].Category != "Transport" || math.Abs(got.Categories[1].Percentage-25) > 1e-9 {
		t.Errorf("second category = %+v, want Transport 25%%", got.Categories[1])
	}
}
