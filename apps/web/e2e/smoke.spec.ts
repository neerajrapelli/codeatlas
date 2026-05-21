import { expect, test } from '@playwright/test';

test.describe('CodeAtlas shell', () => {
  test('loads workspace chrome', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('CodeAtlas')).toBeVisible();
    await expect(page.getByLabel('Repository')).toBeVisible();
  });

  test('shows repository sidebar', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.primary-sidebar')).toBeVisible();
  });
});
