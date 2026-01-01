package storage

import (
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	// Use in-memory database for testing
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	t.Run("CreateExpense", func(t *testing.T) {
		err := db.CreateExpense(10.50, "Lunch", "food", time.Now())
		if err != nil {
			t.Errorf("Failed to create expense: %v", err)
		}
	})

	t.Run("ListExpenses", func(t *testing.T) {
		// Create a few more expenses
		db.CreateExpense(20.00, "Bus", "transport", time.Now())
		db.CreateExpense(5.00, "Coffee", "food", time.Now())

		expenses, err := db.ListExpenses()
		if err != nil {
			t.Errorf("Failed to list expenses: %v", err)
		}

		if len(expenses) != 3 {
			t.Errorf("Expected 3 expenses, got %d", len(expenses))
		}

		// Check order (latest first)
		if expenses[0].Amount != 5.00 { // Coffee was last
			t.Errorf("Expected first expense to be Coffee (5.00), got %.2f", expenses[0].Amount)
		}
	})
}

