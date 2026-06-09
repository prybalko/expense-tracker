import { describe, it, expect } from 'vitest';
import { deriveInsights, deriveYearInsights, expensesForCategory } from './derive';
import type { Expense } from '../types';

// Compact factory so the year-aggregation fixtures stay readable.
function exp(
  id: number,
  amount: number,
  category: string,
  date: string,
): Expense {
  return { id, amount, category, date, description: '', updated_at: `${date}T00:00:00Z` };
}

describe('deriveInsights', () => {
  it('should correctly derive insights from an empty list of expenses', () => {
    const expenses: Expense[] = [];
    const now = new Date('2023-05-15T12:00:00Z');
    
    const insights = deriveInsights(expenses, 2023, 5, now);
    
    expect(insights.total).toBe(0);
    expect(insights.categories).toEqual([]);
    expect(insights.monthName).toBe('May');
    expect(insights.isCurrentPeriod).toBe(true);
    expect(insights.hasChange).toBe(false);
  });

  it('should calculate category totals correctly', () => {
    const expenses: Expense[] = [
      { id: 1, amount: 10, category: 'Groceries', date: '2023-05-01', description: '', updated_at: '2023-05-01T00:00:00Z' },
      { id: 2, amount: 20, category: 'Transport', date: '2023-05-02', description: '', updated_at: '2023-05-02T00:00:00Z' },
      { id: 3, amount: 15, category: 'Groceries', date: '2023-05-03', description: '', updated_at: '2023-05-03T00:00:00Z' },
    ];
    const now = new Date('2023-05-15T12:00:00Z');
    
    const insights = deriveInsights(expenses, 2023, 5, now);
    
    expect(insights.total).toBe(45);
    expect(insights.categories).toHaveLength(2);
    // Should be sorted by total descending
    expect(insights.categories[0].category).toBe('Groceries');
    expect(insights.categories[0].total).toBe(25);
    expect(insights.categories[1].category).toBe('Transport');
    expect(insights.categories[1].total).toBe(20);
  });
});

describe('expensesForCategory', () => {
  it('should filter expenses by category and month', () => {
    const expenses: Expense[] = [
      { id: 1, amount: 10, category: 'Groceries', date: '2023-05-01', description: '', updated_at: '2023-05-01T00:00:00Z' },
      { id: 2, amount: 20, category: 'Transport', date: '2023-05-02', description: '', updated_at: '2023-05-02T00:00:00Z' },
      { id: 3, amount: 15, category: 'Groceries', date: '2023-06-03', description: '', updated_at: '2023-06-03T00:00:00Z' }, // wrong month
    ];

    const filtered = expensesForCategory(expenses, 2023, 5, 'Groceries');
    expect(filtered).toHaveLength(1);
    expect(filtered[0].id).toBe(1);
  });
});

describe('deriveYearInsights', () => {
  it('aggregates a full past year: total, monthly series, sorted categories', () => {
    const now = new Date('2026-06-08T12:00:00Z'); // current period is 2026
    const expenses: Expense[] = [
      exp(1, 100, 'Groceries', '2025-01-15'),
      exp(2, 50, 'Transport', '2025-03-10'),
      exp(3, 25, 'Groceries', '2025-12-20'),
    ];

    const y = deriveYearInsights(expenses, 2025, now);

    expect(y.year).toBe(2025);
    expect(y.isCurrentYear).toBe(false);
    expect(y.elapsedMonths).toBe(12);
    expect(y.total).toBe(175);
    expect(y.averageSpending).toBeCloseTo(175 / 12);
    expect(y.series[0]).toBe(100); // Jan
    expect(y.series[2]).toBe(50); // Mar
    expect(y.series[11]).toBe(25); // Dec
    expect(y.categories[0].category).toBe('Groceries'); // sorted desc
    expect(y.categories[0].total).toBe(125);
    expect(y.categories[1].category).toBe('Transport');
    // No 2024 data, so the year-over-year delta is suppressed.
    expect(y.hasChange).toBe(false);
  });

  it('uses elapsed months for the current year and divides the average by them', () => {
    const now = new Date('2026-06-08T12:00:00Z'); // June => 6 elapsed months
    const expenses: Expense[] = [
      exp(1, 200, 'Groceries', '2026-01-10'),
      exp(2, 100, 'Transport', '2026-06-05'),
    ];

    const y = deriveYearInsights(expenses, 2026, now);

    expect(y.isCurrentYear).toBe(true);
    expect(y.elapsedMonths).toBe(6);
    expect(y.total).toBe(300);
    expect(y.averageSpending).toBeCloseTo(300 / 6);
    expect(y.series[0]).toBe(200);
    expect(y.series[5]).toBe(100);
  });

  it('day-clamps the trailing in-progress month when comparing year over year', () => {
    const now = new Date('2026-06-08T12:00:00Z'); // June 8 => clamp prev June to day 8
    const expenses: Expense[] = [
      // Current year (on/before today)
      exp(1, 200, 'Groceries', '2026-01-10'),
      exp(2, 100, 'Transport', '2026-06-05'),
      // Previous year
      exp(3, 300, 'Groceries', '2025-01-10'), // completed month -> counted in full
      exp(4, 100, 'Transport', '2025-06-05'), // June day 5 (<= 8) -> counted
      exp(5, 500, 'Travel', '2025-06-25'), // June day 25 (> 8) -> clamped OUT
    ];

    const y = deriveYearInsights(expenses, 2026, now);

    // current=300 vs prev=300+100=400 (the day-25 Travel is excluded), so -25%.
    // Without the day-clamp prev would be 900 and the delta would read -66.7%.
    expect(y.total).toBe(300);
    expect(y.hasChange).toBe(true);
    expect(y.isIncrease).toBe(false);
    expect(y.percentageChange).toBeCloseTo(25);
  });
});
