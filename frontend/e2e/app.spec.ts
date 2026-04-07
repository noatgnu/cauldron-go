import { test, expect } from '@playwright/test';

const TEST_API_URL = process.env['TEST_API_URL'] || 'http://127.0.0.1:9245';

async function callTestAPI(path: string, method = 'GET', body?: object): Promise<any> {
  const options: RequestInit = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (body && method === 'POST') {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(`${TEST_API_URL}${path}`, options);
  return response.json();
}

async function waitForWindow(timeout = 30000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    const status = await callTestAPI('/test/window');
    if (status.mainWindow && status.ready) return true;
    await new Promise(r => setTimeout(r, 1000));
  }
  return false;
}

test.describe('Cauldron E2E Integration Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test.describe('TestAPI Health', () => {
    test('should return healthy status', async () => {
      const result = await callTestAPI('/test/health');
      expect(result.status).toBe('ok');
    });

    test('should confirm main window is available', async () => {
      const result = await callTestAPI('/test/window');
      expect(result.mainWindow).toBe(true);
      expect(result.ready).toBe(true);
    });
  });

  test.describe('Settings API', () => {
    test('should return settings', async () => {
      const result = await callTestAPI('/test/settings');
      expect(result).not.toHaveProperty('error');
      expect(result).toBeDefined();
    });
  });

  test.describe('Jobs API', () => {
    test('should return jobs list', async () => {
      const result = await callTestAPI('/test/jobs');
      expect(result).not.toHaveProperty('error');
      expect(result.jobs).toBeDefined();
      expect(Array.isArray(result.jobs)).toBe(true);
    });
  });

  test.describe('Plugins API', () => {
    test('should return plugins list', async () => {
      const result = await callTestAPI('/test/plugins');
      expect(result).not.toHaveProperty('error');
      expect(result.plugins).toBeDefined();
      expect(Array.isArray(result.plugins)).toBe(true);
    });
  });

  test.describe('Imported Files API', () => {
    test('should return imported files list', async () => {
      const result = await callTestAPI('/test/imported-files');
      expect(result).not.toHaveProperty('error');
      expect(result.files).toBeDefined();
      expect(Array.isArray(result.files)).toBe(true);
    });
  });

  test.describe('UI Navigation via TestAPI', () => {
    test('should navigate to settings page', async () => {
      const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/general' });
      expect(navResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 2000));
    });

    test('should navigate to jobs page', async () => {
      const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/jobs' });
      expect(navResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 2000));
    });

    test('should navigate to plugins page', async () => {
      const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugins' });
      expect(navResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 2000));
    });

    test('should navigate to files page', async () => {
      const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/files' });
      expect(navResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 2000));
    });

    test('should navigate to Python settings page', async () => {
      const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/python' });
      expect(navResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 2000));
    });

    test('should navigate to R settings page', async () => {
      const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/r' });
      expect(navResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 2000));
    });
  });

  test.describe('Environment Detection via TestAPI', () => {
    test('should detect Python environments', async () => {
      const result = await callTestAPI('/test/python-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
      expect(typeof result.count).toBe('number');
    });

    test('should detect R environments', async () => {
      const result = await callTestAPI('/test/r-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });

    test('should get virtual environments', async () => {
      const result = await callTestAPI('/test/virtual-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });

    test('should get renv environments', async () => {
      const result = await callTestAPI('/test/renv-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });
  });

  test.describe('Plugin Bindings via TestAPI', () => {
    test('should get plugin environment bindings', async () => {
      const result = await callTestAPI('/test/plugin-bindings');
      expect(result).not.toHaveProperty('error');
      expect(result.bindings).toBeDefined();
      expect(Array.isArray(result.bindings)).toBe(true);
    });
  });

  test.describe('Default Paths via TestAPI', () => {
    test('should get default venv path', async () => {
      const result = await callTestAPI('/test/default-venv-path?pluginId=test');
      expect(result).not.toHaveProperty('error');
      expect(result.path).toBeDefined();
      expect(typeof result.path).toBe('string');
    });
  });
});
