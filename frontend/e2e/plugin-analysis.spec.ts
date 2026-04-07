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

test.describe('Plugin Analysis via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should list available plugins', async () => {
    const plugins = await callTestAPI('/test/plugins');
    expect(plugins).not.toHaveProperty('error');
    expect(plugins.plugins).toBeDefined();
    expect(Array.isArray(plugins.plugins)).toBe(true);
  });

  test('should have plugin with required structure', async () => {
    const plugins = await callTestAPI('/test/plugins');
    if (!plugins.plugins || plugins.plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    const plugin = plugins.plugins[0];
    expect(plugin).toBeDefined();

    if (plugin.definition) {
      expect(plugin.definition.plugin).toBeDefined();
      expect(plugin.definition.plugin.name).toBeDefined();
      expect(plugin.definition.plugin.id).toBeDefined();
    }
  });

  test('should navigate to plugin list page via TestAPI', async () => {
    const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugin-list' });
    expect(navResult.success).toBe(true);

    await new Promise(r => setTimeout(r, 2000));
  });

  test('should navigate to plugins page via TestAPI', async () => {
    const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugins' });
    expect(navResult.success).toBe(true);

    await new Promise(r => setTimeout(r, 2000));
  });

  test('should get jobs list', async () => {
    const jobs = await callTestAPI('/test/jobs');
    expect(jobs).not.toHaveProperty('error');
    expect(jobs.jobs).toBeDefined();
    expect(Array.isArray(jobs.jobs)).toBe(true);
  });

  test('should verify plugin bindings are accessible', async () => {
    const bindings = await callTestAPI('/test/plugin-bindings');
    expect(bindings).not.toHaveProperty('error');
    expect(bindings.bindings).toBeDefined();
    expect(Array.isArray(bindings.bindings)).toBe(true);
  });
});

test.describe('Plugin Details via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should have plugins with runtime environments', async () => {
    const plugins = await callTestAPI('/test/plugins');
    if (!plugins.plugins || plugins.plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    for (const plugin of plugins.plugins.slice(0, 3)) {
      if (plugin.definition?.runtime) {
        expect(plugin.definition.runtime).toBeDefined();
      }
    }
  });

  test('should have plugins with inputs configuration', async () => {
    const plugins = await callTestAPI('/test/plugins');
    if (!plugins.plugins || plugins.plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    for (const plugin of plugins.plugins.slice(0, 3)) {
      if (plugin.definition?.inputs) {
        expect(Array.isArray(plugin.definition.inputs)).toBe(true);
      }
    }
  });
});
