import { test, expect } from '@playwright/test';

const BASE_URL = process.env['WAILS_URL'] || 'http://localhost:4200';

test.describe('Plugin Installation Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should successfully install a plugin through the UI registry', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/plugin-registry`);
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText(/Plugin Registry/i);

    const pluginRow = page.locator('tr.plugin-row').filter({ hasText: /DIA-NN to CurtainPTM Converter/i }).first();
    await expect(pluginRow).toBeVisible({ timeout: 15000 });

    const installButton = pluginRow.locator('button[aria-label="Install plugin"]');
    await installButton.click();

    const dialog = page.locator('mat-dialog-container');
    await expect(dialog).toBeVisible({ timeout: 15000 });
    await expect(dialog.locator('h2')).toContainText(/Confirm Plugin Installation/i);

    const pythonCheckbox = dialog.locator('mat-checkbox:has-text("Create Python Virtual Environment")');
    if (await pythonCheckbox.isVisible() && await pythonCheckbox.locator('input').isChecked()) {
      await pythonCheckbox.click();
    }

    const rCheckbox = dialog.locator('mat-checkbox:has-text("Create R Environment")');
    if (await rCheckbox.isVisible() && await rCheckbox.locator('input').isChecked()) {
      await rCheckbox.click();
    }

    const confirmButton = dialog.locator('button:has-text("Install Plugin")');
    await confirmButton.click();

    const progressDialog = page.locator('mat-dialog-container:has-text("Installing Plugin")');
    await expect(progressDialog).toBeVisible({ timeout: 10000 });

    await expect(progressDialog).not.toBeVisible({ timeout: 60000 });

    await page.goto(`${BASE_URL}/#/plugins`);
    await page.waitForLoadState('networkidle');

    const pluginInList = page.locator('mat-card, tr').filter({ hasText: /DIA-NN to CurtainPTM Converter/i });
    await expect(pluginInList.first()).toBeVisible({ timeout: 10000 });
  });
});
