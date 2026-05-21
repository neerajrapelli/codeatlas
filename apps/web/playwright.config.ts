import { defineConfig, devices } from '@playwright/test';

const composeMode = process.env.E2E_COMPOSE === '1';
const baseURL =
  process.env.E2E_BASE_URL ?? (composeMode ? 'http://127.0.0.1:80' : 'http://127.0.0.1:4173');

export default defineConfig({
  testDir: './e2e',
  fullyParallel: !composeMode,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: composeMode ? 1 : process.env.CI ? 1 : undefined,
  reporter: 'list',
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: composeMode
    ? undefined
    : {
        command: 'pnpm build && pnpm preview --host 127.0.0.1 --port 4173',
        url: 'http://127.0.0.1:4173',
        reuseExistingServer: !process.env.CI,
        timeout: 180_000,
      },
});
