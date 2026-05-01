import { test, expect, Page } from '@playwright/test';

async function pressKeys(page: Page, keys: string) {
  for (const char of keys) {
    const key = char === '.' ? 'dot' : char;
    await page.getByTestId(`keypad-${key}`).click();
  }
}

test.describe('Offline Sync Queue', () => {
  test.beforeEach(async ({ request }) => {
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

  test('Optimistic update and background sync when coming online', async ({ page, context }) => {
    await login(page);

    await context.setOffline(true);

    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();
    await pressKeys(page, '75.50');
    await page.getByTestId('entry-note').fill('Offline Sync Test');
    await page.getByTestId('category-tile-groceries').click();
    
    await page.getByTestId('entry-submit').click();
    await expect(page.getByTestId('entry-form')).not.toBeVisible();

    await expect(page.getByTestId('expense-row')).toHaveCount(1);
    const row = page.getByTestId('expense-row').first();
    await expect(row.getByTestId('expense-row-desc')).toHaveText('Offline Sync Test');

    const requestPromise = page.waitForRequest(
      request => request.url().includes('/api/expenses') && request.method() === 'POST'
    );

    // Come back online
    await context.setOffline(false);
    // Playwright's setOffline doesn't fire the 'online' event, so we trigger it to wake the sync queue
    await page.evaluate(() => window.dispatchEvent(new Event('online')));

    const req = await requestPromise;
    expect(req.method()).toBe('POST');

    await page.waitForTimeout(500); 
    await page.reload();
    
    await expect(page.getByTestId('feed-screen')).toBeVisible();
    await expect(page.getByTestId('expense-row')).toHaveCount(1);
    await expect(page.getByTestId('expense-row').first().getByTestId('expense-row-desc')).toHaveText('Offline Sync Test');
  });

  test('Server 5xx error in foreground rolls back optimistic update', async ({ page }) => {
    await login(page);

    await page.route('**/api/expenses', async route => {
      if (route.request().method() === 'POST') {
        await route.fulfill({ status: 500, body: 'Internal Server Error' });
      } else {
        await route.continue();
      }
    });

    await page.getByTestId('fab-add').click();
    await expect(page.getByTestId('entry-form')).toBeVisible();
    await pressKeys(page, '55.00');
    await page.getByTestId('entry-note').fill('500 Test');
    await page.getByTestId('category-tile-transport').click();
    
    await page.getByTestId('entry-submit').click();
    
    // Form stays open on 500 error since it throws
    await expect(page.getByTestId('entry-form')).toBeVisible();

    // Close the form manually
    await page.getByRole('button', { name: 'Close' }).click();

    // The optimistic update should have been rolled back
    await expect(page.getByTestId('expense-row')).toHaveCount(0);
  });
});
