import { describe, it, expect } from 'vitest';
import { deriveInsights, expensesForCategory } from './derive';
import type { Expense } from '../types';

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
      { id: 1, amount: 10, category: 'Groceries', date: '2023-05-01', description: '' },
      { id: 2, amount: 20, category: 'Transport', date: '2023-05-02', description: '' },
      { id: 3, amount: 15, category: 'Groceries', date: '2023-05-03', description: '' },
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
      { id: 1, amount: 10, category: 'Groceries', date: '2023-05-01', description: '' },
      { id: 2, amount: 20, category: 'Transport', date: '2023-05-02', description: '' },
      { id: 3, amount: 15, category: 'Groceries', date: '2023-06-03', description: '' }, // wrong month
    ];

    const filtered = expensesForCategory(expenses, 2023, 5, 'Groceries');
    expect(filtered).toHaveLength(1);
    expect(filtered[0].id).toBe(1);
  });
});
