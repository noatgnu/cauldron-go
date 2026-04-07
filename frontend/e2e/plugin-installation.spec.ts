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

test.describe('Plugin Installation Workflow via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should list installed plugins', async () => {
    const plugins = await callTestAPI('/test/plugins');
    expect(plugins).not.toHaveProperty('error');
    expect(plugins.plugins).toBeDefined();
    expect(Array.isArray(plugins.plugins)).toBe(true);
    expect(typeof plugins.count).toBe('number');
  });

  test('should navigate to plugin registry page via TestAPI', async () => {
    const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugin-registry' });
    expect(navResult.success).toBe(true);

    await new Promise(r => setTimeout(r, 2000));
  });

  test('should navigate to plugins page via TestAPI', async () => {
    const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugins' });
    expect(navResult.success).toBe(true);

    await new Promise(r => setTimeout(r, 2000));
  });

  test('should have settings available', async () => {
    const settings = await callTestAPI('/test/settings');
    expect(settings).not.toHaveProperty('error');
    expect(settings).toBeDefined();
  });
});

test.describe('Plugin Environment Management via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should get plugin bindings', async () => {
    const bindings = await callTestAPI('/test/plugin-bindings');
    expect(bindings).not.toHaveProperty('error');
    expect(bindings.bindings).toBeDefined();
    expect(Array.isArray(bindings.bindings)).toBe(true);
  });

  test('should get virtual environments for plugin binding', async () => {
    const venvs = await callTestAPI('/test/virtual-environments');
    expect(venvs).not.toHaveProperty('error');
    expect(venvs.environments).toBeDefined();
    expect(Array.isArray(venvs.environments)).toBe(true);
  });

  test('should get renv environments for plugin binding', async () => {
    const renvs = await callTestAPI('/test/renv-environments');
    expect(renvs).not.toHaveProperty('error');
    expect(renvs.environments).toBeDefined();
    expect(Array.isArray(renvs.environments)).toBe(true);
  });
});
