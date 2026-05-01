import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:8081',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'cd ../../ && go build -o server ./cmd/server && PORT=8081 ADMIN_USER=testuser ADMIN_PASSWORD=testpass123 DB_PATH=./test_expenses.db ./server',
    port: 8081,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
