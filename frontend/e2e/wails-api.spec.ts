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

test.describe('Backend API Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test.describe('Health and Status', () => {
    test('TestAPI health check returns ok', async () => {
      const result = await callTestAPI('/test/health');
      expect(result.status).toBe('ok');
    });

    test('Window status shows main window available', async () => {
      const result = await callTestAPI('/test/window');
      expect(result.mainWindow).toBe(true);
      expect(result.ready).toBe(true);
    });
  });

  test.describe('Settings API', () => {
    test('GetSettings returns valid config', async () => {
      const result = await callTestAPI('/test/settings');
      expect(result).not.toHaveProperty('error');
      expect(result).toBeDefined();
    });
  });

  test.describe('Jobs API', () => {
    test('GetAllJobs returns array', async () => {
      const result = await callTestAPI('/test/jobs');
      expect(result).not.toHaveProperty('error');
      expect(result.jobs).toBeDefined();
      expect(Array.isArray(result.jobs)).toBe(true);
    });
  });

  test.describe('Files API', () => {
    test('GetImportedFiles returns array', async () => {
      const result = await callTestAPI('/test/imported-files');
      expect(result).not.toHaveProperty('error');
      expect(result.files).toBeDefined();
      expect(Array.isArray(result.files)).toBe(true);
    });
  });

  test.describe('Plugins API', () => {
    test('GetPluginsV2 returns array', async () => {
      const result = await callTestAPI('/test/plugins');
      expect(result).not.toHaveProperty('error');
      expect(result.plugins).toBeDefined();
      expect(Array.isArray(result.plugins)).toBe(true);
    });
  });

  test.describe('Environment API', () => {
    test('DetectPythonEnvironments returns array', async () => {
      const result = await callTestAPI('/test/python-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });

    test('DetectREnvironments returns array', async () => {
      const result = await callTestAPI('/test/r-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });

    test('GetVirtualEnvironments returns array', async () => {
      const result = await callTestAPI('/test/virtual-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });

    test('GetRenvEnvironments returns array', async () => {
      const result = await callTestAPI('/test/renv-environments');
      expect(result).not.toHaveProperty('error');
      expect(result.environments).toBeDefined();
      expect(Array.isArray(result.environments)).toBe(true);
    });

    test('GetAllPluginEnvironmentBindings returns array', async () => {
      const result = await callTestAPI('/test/plugin-bindings');
      expect(result).not.toHaveProperty('error');
      expect(result.bindings).toBeDefined();
      expect(Array.isArray(result.bindings)).toBe(true);
    });

    test('GetDefaultVenvPath returns path', async () => {
      const result = await callTestAPI('/test/default-venv-path?pluginId=test-plugin');
      expect(result).not.toHaveProperty('error');
      expect(result.path).toBeDefined();
      expect(typeof result.path).toBe('string');
    });
  });
});

test.describe('Data Serialization Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('Jobs returns empty array not null', async () => {
    const result = await callTestAPI('/test/jobs');
    expect(result.jobs).not.toBeNull();
    expect(Array.isArray(result.jobs)).toBe(true);
  });

  test('Files returns empty array not null', async () => {
    const result = await callTestAPI('/test/imported-files');
    expect(result.files).not.toBeNull();
    expect(Array.isArray(result.files)).toBe(true);
  });

  test('Plugins returns empty array not null', async () => {
    const result = await callTestAPI('/test/plugins');
    expect(result.plugins).not.toBeNull();
    expect(Array.isArray(result.plugins)).toBe(true);
  });

  test('Virtual environments returns empty array not null', async () => {
    const result = await callTestAPI('/test/virtual-environments');
    expect(result.environments).not.toBeNull();
    expect(Array.isArray(result.environments)).toBe(true);
  });

  test('Renv environments returns empty array not null', async () => {
    const result = await callTestAPI('/test/renv-environments');
    expect(result.environments).not.toBeNull();
    expect(Array.isArray(result.environments)).toBe(true);
  });

  test('Plugin bindings returns empty array not null', async () => {
    const result = await callTestAPI('/test/plugin-bindings');
    expect(result.bindings).not.toBeNull();
    expect(Array.isArray(result.bindings)).toBe(true);
  });
});

test.describe('UI Control Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('Navigate to home', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to settings', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/general' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to Python settings', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/python' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to R settings', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/r' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to jobs', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/jobs' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to plugins', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugins' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to plugin list', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/plugin-list' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Navigate to files', async () => {
    const result = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/files' });
    expect(result.success).toBe(true);
    await new Promise(r => setTimeout(r, 1000));
  });

  test('Execute custom JavaScript', async () => {
    const result = await callTestAPI('/test/ui/exec-js', 'POST', {
      script: 'console.log("E2E test executed")'
    });
    expect(result.success).toBe(true);
  });
});
