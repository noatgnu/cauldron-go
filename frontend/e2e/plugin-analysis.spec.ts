import { test, expect } from '@playwright/test';

const BASE_URL = process.env['WAILS_URL'] || 'http://localhost:4200';
const TEST_API_URL = process.env['TEST_API_URL'] || 'http://127.0.0.1:9245';

async function checkTestAPIAvailable(): Promise<boolean> {
  try {
    const response = await fetch(`${TEST_API_URL}/test/health`);
    if (response.ok) {
      const data = await response.json();
      return data.status === 'ok';
    }
    return false;
  } catch {
    return false;
  }
}

async function callTestAPI(endpoint: string): Promise<any> {
  try {
    const response = await fetch(`${TEST_API_URL}${endpoint}`);
    if (response.ok) {
      return await response.json();
    }
    return null;
  } catch {
    return null;
  }
}

test.describe('Plugin Analysis UI Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to plugin list page', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/plugin-list`);
    await page.waitForLoadState('networkidle');

    const pageLoaded = await page.locator('body').isVisible();
    expect(pageLoaded).toBeTruthy();

    const hasWails = await page.evaluate(() => '_wails' in window);
    if (!hasWails) {
      const testAPIAvailable = await checkTestAPIAvailable();
      if (testAPIAvailable) {
        const plugins = await callTestAPI('/test/plugins');
        expect(plugins).not.toBeNull();
        expect(Array.isArray(plugins?.plugins)).toBeTruthy();
      }
      return;
    }

    const pluginCard = page.locator('mat-card').filter({ hasText: /PCA Analysis/i }).first();
    await expect(pluginCard).toBeVisible({ timeout: 15000 });
  });

  test('should successfully run an analysis by clicking Load Example and Run Analysis', async ({ page }) => {
    const hasWails = await page.evaluate(() => '_wails' in window);
    if (!hasWails) {
      const testAPIAvailable = await checkTestAPIAvailable();
      expect(testAPIAvailable).toBeTruthy();

      const plugins = await callTestAPI('/test/plugins');
      expect(plugins).not.toBeNull();
      expect(Array.isArray(plugins?.plugins)).toBeTruthy();

      await page.goto(`${BASE_URL}/#/plugin-list`);
      await page.waitForLoadState('networkidle');
      const pageLoaded = await page.locator('body').isVisible();
      expect(pageLoaded).toBeTruthy();
      return;
    }

    await page.goto(`${BASE_URL}/#/plugin-list`);
    await page.waitForLoadState('networkidle');

    const pluginCard = page.locator('mat-card').filter({ hasText: /PCA Analysis/i }).first();
    await expect(pluginCard).toBeVisible({ timeout: 15000 });

    const openButton = pluginCard.locator('button:has-text("Open")');
    await openButton.click();

    await expect(page.url()).toContain('/plugin/');
    await expect(page.locator('mat-card-title')).toContainText(/PCA Analysis/i);

    const loadExampleButton = page.locator('button:has-text("Load Example")');
    await expect(loadExampleButton).toBeVisible();
    await loadExampleButton.click();

    const runButton = page.locator('button:has-text("Run Analysis")');
    await expect(runButton).toBeEnabled();
    await runButton.click();

    await expect(runButton).toContainText(/Running/i, { timeout: 5000 });

    const viewJobButton = page.locator('button:has-text("View Job")');
    await expect(viewJobButton).toBeVisible({ timeout: 15000 });
    await viewJobButton.click();

    await expect(page.url()).toContain('/job/');
    await expect(page.locator('h1, h2')).toContainText(/Job Details/i);

    const statusBadge = page.locator('.status-badge, [class*="status"]');
    await expect(statusBadge).toBeVisible();

    await expect(page.locator('body')).not.toContainText(/crash/i);
  });
});
