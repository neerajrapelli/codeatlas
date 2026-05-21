import { expect, test } from '@playwright/test';

const composeMode = process.env.E2E_COMPOSE === '1';

test.describe('Live API (docker compose)', () => {
  test.skip(!composeMode, 'Set E2E_COMPOSE=1 and run against http://127.0.0.1 (see make smoke-compose)');

  test('proxied GET /api/health returns ok', async ({ request, baseURL }) => {
    const res = await request.get(`${baseURL}/api/health`);
    expect(res.ok()).toBeTruthy();
    const body = (await res.json()) as { status?: string; service?: string };
    expect(body.status).toBe('ok');
    expect(body.service).toBe('codeatlas-api');
  });

  test('GET /api/repositories returns list', async ({ request, baseURL }) => {
    const res = await request.get(`${baseURL}/api/repositories`);
    expect(res.ok()).toBeTruthy();
    const body = (await res.json()) as { repositories?: unknown[] };
    expect(Array.isArray(body.repositories)).toBeTruthy();
  });

  test('UI shows API online after health poll', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('CodeAtlas')).toBeVisible();
    await expect(page.getByRole('status').getByText(/API online/i)).toBeVisible({
      timeout: 45_000,
    });
  });

  test('Settings reports API connected', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Settings' }).click();
    await expect(page.getByText('SETTINGS')).toBeVisible();
    await expect(page.getByText(/API connection:\s*Connected/i)).toBeVisible({
      timeout: 45_000,
    });
  });

  test('theme picker persists light mode', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Settings' }).click();
    await page.getByRole('radio', { name: 'Light' }).click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    await page.reload();
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  });
});
