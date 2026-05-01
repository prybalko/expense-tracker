import { test, expect, Page } from '@playwright/test';

// Helper to interact with the keypad
async function pressKeys(page: Page, keys: string) {
  for (const char of keys) {
    const key = char === '.' ? 'dot' : char;
    await page.getByTestId(`keypad-${key}`).click();
  }
}

// Format YYYY-MM-DD
function dayISO(day: number): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(day).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

test.describe('App E2E', () => {
  test.beforeEach(async ({ request }) => {
    // Clear the database via the test-only endpoint we added
    await request.post('/api/test/clear');
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  async function login(page: Page) {
    await expect(page.getByTestId('login-form')).toBeVisible();
    await page.getByTestId('login-username').fill('testuser');
    await page.getByTestId('login-password').fill('testpass123');
    await page.getByTestId('login-submit').click();
    await expect(page.getByTestId('feed-screen')).toBeVisible();
  }

  async function addExpense(page: Page, amount: string, note: string, categorySlug: string) {
    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();
    await pressKeys(page, amount);
    await page.getByTestId('entry-note').fill(note);
    await page.getByTestId('category-tile-' + categorySlug).click();
    await page.getByTestId('entry-submit').click();
    await expect(page.getByTestId('entry-form')).not.toBeVisible();
  }

  test('Complete User Flow', async ({ page }) => {
    await login(page);

    await expect(page.getByTestId('hero-label')).toHaveText('spent this month');

    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();

    await pressKeys(page, '12.50');
    await expect(page.getByTestId('entry-amount')).toHaveAttribute('data-amount', '12.50');

    await page.getByTestId('entry-note').fill('Lunch Test');
    await page.getByTestId('category-tile-groceries').click();
    await page.getByTestId('entry-submit').click();

    await expect(page.getByTestId('feed-screen')).toBeVisible();
    await expect(page.getByTestId('expense-row')).toHaveCount(1);

    const row = page.getByTestId('expense-row').first();
    await expect(row.getByTestId('expense-row-desc')).toHaveText('Lunch Test');
    await expect(row.getByTestId('expense-row-amount')).toContainText('12.50');
    await expect(row).toHaveAttribute('data-cat-slug', 'groceries');
  });

  test('Add Expense To Blank List', async ({ page }) => {
    await login(page);

    await expect(page.getByTestId('expense-row')).toHaveCount(0);
    await expect(page.getByTestId('hero-total')).toContainText('0.00');

    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();

    await pressKeys(page, '25.99');
    await expect(page.getByTestId('entry-amount')).toHaveAttribute('data-amount', '25.99');

    await page.getByTestId('entry-submit').click();

    await expect(page.getByTestId('expense-row')).toHaveCount(1);
    await expect(page.getByTestId('expense-row-amount').first()).toContainText('25.99');
    await expect(page.getByTestId('hero-total')).toContainText('25.99');

    // Default category is first (groceries)
    await expect(page.getByTestId('expense-row').first()).toHaveAttribute('data-cat-slug', 'groceries');
  });

  test('Edit Expense Flow', async ({ page }) => {
    await login(page);

    const todayDay = new Date().getDate();
    const createDay = 1;
    const editDay = todayDay;

    // 1. Create an expense to edit
    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();
    await page.getByTestId('category-tile-transport').click();

    await page.getByTestId('date-pill').click();
    await expect(page.getByTestId('date-sheet')).toBeVisible();
    await page.getByTestId('calendar-day-' + dayISO(createDay)).click();
    await page.getByTestId('date-sheet-done').click();
    await expect(page.getByTestId('date-sheet')).not.toBeVisible();

    await pressKeys(page, '50.00');
    await page.getByTestId('entry-note').fill('Original Expense');
    await page.getByTestId('entry-submit').click();

    await expect(page.getByTestId('entry-form')).not.toBeVisible();
    await expect(page.getByTestId('expense-row')).toHaveCount(1);

    const row = page.getByTestId('expense-row').first();
    await expect(row.getByTestId('expense-row-desc')).toHaveText('Original Expense');
    await expect(row.getByTestId('expense-row-amount')).toContainText('50.00');
    await expect(row).toHaveAttribute('data-cat-slug', 'transport');

    // 2. Open it for editing
    await row.click();
    await expect(page.getByTestId('entry-form')).toBeVisible();
    await expect(page.getByTestId('entry-note')).toHaveValue('Original Expense');

    // 3. Clear amount
    for (let i = 0; i < 5; i++) {
      await page.getByTestId('keypad-del').click();
    }
    await expect(page.getByTestId('entry-amount')).toHaveAttribute('data-amount', '0');

    await pressKeys(page, '40.00');
    await page.getByTestId('entry-note').fill('Updated Expense');

    // Edit date
    await page.getByTestId('date-pill').click();
    await expect(page.getByTestId('date-sheet')).toBeVisible();
    await page.getByTestId('calendar-day-' + dayISO(editDay)).click();
    await page.getByTestId('date-sheet-done').click();

    await page.getByTestId('entry-submit').click();
    await expect(page.getByTestId('entry-form')).not.toBeVisible();

    // 4. Verify edit
    const updated = page.getByTestId('expense-row').first();
    await expect(updated.getByTestId('expense-row-desc')).toHaveText('Updated Expense');
    await expect(updated.getByTestId('expense-row-amount')).toContainText('40.00');
    await expect(updated).toHaveAttribute('data-cat-slug', 'transport');
    await expect(page.getByTestId('hero-total')).toContainText('40.00');
  });

  test('Delete Expense Flow', async ({ page }) => {
    await login(page);

    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();

    await pressKeys(page, '99.99');
    await page.getByTestId('entry-note').fill('To Be Deleted');
    await page.getByTestId('entry-submit').click();
    await expect(page.getByTestId('entry-form')).not.toBeVisible();

    await expect(page.getByTestId('expense-row')).toHaveCount(1);
    await expect(page.getByTestId('hero-total')).toContainText('99.99');

    await page.getByTestId('expense-row').first().click();
    await expect(page.getByTestId('entry-delete')).toBeVisible();

    page.on('dialog', dialog => dialog.accept());
    await page.getByTestId('entry-delete').click();

    await expect(page.getByTestId('feed-screen')).toBeVisible();
    await expect(page.getByTestId('expense-row')).toHaveCount(0);
    await expect(page.getByTestId('hero-total')).toContainText('0.00');
  });

  test('Delete Button Not Visible On Create', async ({ page }) => {
    await login(page);

    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();

    await expect(page.getByTestId('entry-delete')).toHaveCount(0);
    await expect(page.getByTestId('keypad-del')).toBeVisible();
  });

  test('Category Details From Insights', async ({ page }) => {
    await login(page);

    await addExpense(page, '12.50', 'Bakery', 'groceries');
    await addExpense(page, '25.00', 'Supermarket', 'groceries');

    await expect(page.getByTestId('expense-row')).toHaveCount(2);

    await page.goto('/insights');
    
    const row = page.getByTestId('category-row-groceries');
    await expect(row).toBeVisible();
    await row.click();

    await expect(page.getByTestId('category-details')).toBeVisible();
    await expect(page.getByTestId('category-details-total')).toContainText('37');
    await expect(page.getByTestId('category-details-count')).toContainText('2 transactions');
    await expect(page.getByTestId('expense-row')).toHaveCount(2);

    await page.getByTestId('expense-row').first().click();
    await expect(page.getByTestId('entry-form')).toBeVisible();
  });
});
