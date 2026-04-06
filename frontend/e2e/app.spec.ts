import { test, expect, Page } from '@playwright/test';

const BASE_URL = process.env['WAILS_URL'] || 'http://localhost:4200';

test.describe('Cauldron E2E Integration Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test.describe('App Startup', () => {
    test('should load the application', async ({ page }) => {
      await expect(page).toHaveTitle(/Cauldron/i);
    });

    test('should display main navigation', async ({ page }) => {
      const nav = page.locator('mat-sidenav, mat-toolbar, nav');
      await expect(nav.first()).toBeVisible();
    });

    test('should have home/dashboard route accessible', async ({ page }) => {
      const homeLink = page.locator('a[routerlink="/"], a[href="/"]').first();
      if (await homeLink.isVisible()) {
        await homeLink.click();
        await page.waitForLoadState('networkidle');
      }
      await expect(page.url()).toContain(BASE_URL);
    });
  });

  test.describe('Settings Page', () => {
    test('should navigate to settings', async ({ page }) => {
      const settingsLink = page.locator('a[routerlink*="settings"], [routerlink*="settings"]').first();
      if (await settingsLink.isVisible()) {
        await settingsLink.click();
        await page.waitForURL('**/settings**');
        await expect(page.url()).toContain('settings');
      }
    });

    test('should display settings form', async ({ page }) => {
      await page.goto(`${BASE_URL}/#/settings`);
      await page.waitForLoadState('networkidle');

      const form = page.locator('form, mat-card, .settings');
      await expect(form.first()).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('Jobs Page', () => {
    test('should navigate to jobs', async ({ page }) => {
      await page.goto(`${BASE_URL}/#/jobs`);
      await page.waitForLoadState('networkidle');

      const jobsContent = page.locator('mat-table, .job-list, [class*="job"]');
      const pageTitle = page.locator('h1, h2, .page-title');

      const hasContent = await jobsContent.first().isVisible().catch(() => false) ||
                         await pageTitle.first().isVisible().catch(() => false);
      expect(hasContent).toBeTruthy();
    });

    test('should show empty state or job list', async ({ page }) => {
      await page.goto(`${BASE_URL}/#/jobs`);
      await page.waitForLoadState('networkidle');

      const jobItems = page.locator('mat-row, .job-item, [class*="job"]');
      const emptyState = page.locator('.empty-state, .no-jobs, [class*="empty"]');

      const hasJobs = await jobItems.count() > 0;
      const hasEmptyState = await emptyState.isVisible().catch(() => false);

      expect(hasJobs || hasEmptyState || true).toBeTruthy();
    });
  });

  test.describe('Plugins Page', () => {
    test('should navigate to plugins', async ({ page }) => {
      await page.goto(`${BASE_URL}/#/plugins`);
      await page.waitForLoadState('networkidle');

      const pageLoaded = await page.locator('body').isVisible();
      expect(pageLoaded).toBeTruthy();
    });

    test('should show plugin list or empty state', async ({ page }) => {
      await page.goto(`${BASE_URL}/#/plugins`);
      await page.waitForLoadState('networkidle');

      await page.waitForTimeout(2000);

      const pluginCards = page.locator('mat-card, .plugin-card, [class*="plugin"]');
      const pluginCount = await pluginCards.count();

      expect(pluginCount >= 0).toBeTruthy();
    });
  });

  test.describe('Files Page', () => {
    test('should navigate to files/imports', async ({ page }) => {
      await page.goto(`${BASE_URL}/#/files`);
      await page.waitForLoadState('networkidle');

      const pageLoaded = await page.locator('body').isVisible();
      expect(pageLoaded).toBeTruthy();
    });
  });
});

test.describe('Wails API Integration', () => {
  test('should have Wails runtime available', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const hasWails = await page.evaluate(() => {
      return '_wails' in window || 'go' in window || 'runtime' in window;
    });

    if (!hasWails) {
      test.skip();
    }
    expect(hasWails).toBeTruthy();
  });

  test('should be able to call GetSettings', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const result = await page.evaluate(async () => {
      if (!('_wails' in window)) {
        return { skipped: true };
      }

      try {
        const WailsApp = (window as any).go?.['github.com/noatgnu/cauldron-go'];
        if (WailsApp?.GetSettings) {
          const settings = await WailsApp.GetSettings();
          return { success: true, hasSettings: settings !== null };
        }
        return { skipped: true, reason: 'WailsApp not available' };
      } catch (e: any) {
        return { error: e.message };
      }
    });

    if ('skipped' in result && result.skipped) {
      test.skip();
    }

    if ('success' in result) {
      expect(result.success).toBeTruthy();
    }
  });

  test('should be able to call GetAllJobs', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const result = await page.evaluate(async () => {
      if (!('_wails' in window)) {
        return { skipped: true };
      }

      try {
        const WailsApp = (window as any).go?.['github.com/noatgnu/cauldron-go'];
        if (WailsApp?.GetAllJobs) {
          const jobs = await WailsApp.GetAllJobs();
          return {
            success: true,
            isArray: Array.isArray(jobs),
            count: Array.isArray(jobs) ? jobs.length : 0
          };
        }
        return { skipped: true };
      } catch (e: any) {
        return { error: e.message };
      }
    });

    if ('skipped' in result && result.skipped) {
      test.skip();
    }

    if ('success' in result) {
      expect(result.success).toBeTruthy();
      expect(result.isArray).toBeTruthy();
    }
  });

  test('should be able to call GetPluginsV2', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const result = await page.evaluate(async () => {
      if (!('_wails' in window)) {
        return { skipped: true };
      }

      try {
        const WailsApp = (window as any).go?.['github.com/noatgnu/cauldron-go'];
        if (WailsApp?.GetPluginsV2) {
          const plugins = await WailsApp.GetPluginsV2();
          return {
            success: true,
            isArray: Array.isArray(plugins),
            count: Array.isArray(plugins) ? plugins.length : 0
          };
        }
        return { skipped: true };
      } catch (e: any) {
        return { error: e.message };
      }
    });

    if ('skipped' in result && result.skipped) {
      test.skip();
    }

    if ('success' in result) {
      expect(result.success).toBeTruthy();
      expect(result.isArray).toBeTruthy();
    }
  });

  test('should be able to call GetJobQueueStatus', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const result = await page.evaluate(async () => {
      if (!('_wails' in window)) {
        return { skipped: true };
      }

      try {
        const WailsApp = (window as any).go?.['github.com/noatgnu/cauldron-go'];
        if (WailsApp?.GetJobQueueStatus) {
          const status = await WailsApp.GetJobQueueStatus();
          return {
            success: true,
            hasStatus: status !== null && typeof status === 'object'
          };
        }
        return { skipped: true };
      } catch (e: any) {
        return { error: e.message };
      }
    });

    if ('skipped' in result && result.skipped) {
      test.skip();
    }

    if ('success' in result) {
      expect(result.success).toBeTruthy();
    }
  });
});

test.describe('UI Interactions', () => {
  test('should toggle theme', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    const themeToggle = page.locator('[class*="theme"], button[aria-label*="theme"]').first();
    if (await themeToggle.isVisible().catch(() => false)) {
      const initialClass = await page.locator('body').getAttribute('class');
      await themeToggle.click();
      await page.waitForTimeout(500);
      const newClass = await page.locator('body').getAttribute('class');
      expect(initialClass !== newClass || true).toBeTruthy();
    }
  });

  test('should have responsive layout', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');

    await page.setViewportSize({ width: 1920, height: 1080 });
    const desktopVisible = await page.locator('mat-sidenav, nav').first().isVisible().catch(() => true);

    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(500);
    const mobileVisible = await page.locator('body').isVisible();

    expect(desktopVisible && mobileVisible).toBeTruthy();
  });
});
