import { test, expect } from '@playwright/test';
import * as mcp from './helpers/mcp-client';

test.describe('Backend API Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test.describe('Health and Status', () => {
    test('MCP app_info reports a visible window', async () => {
      const info = await mcp.appInfo();
      expect(info.windows?.[0]?.visible).toBe(true);
    });
  });

  test.describe('Settings API', () => {
    test('GetSettings returns valid config', async () => {
      const result = await mcp.callBoundMethod('main.App.GetSettings');
      expect(result).toBeDefined();
    });
  });

  test.describe('Jobs API', () => {
    test('GetAllJobs returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.GetAllJobs');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Files API', () => {
    test('GetImportedFiles returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.GetImportedFiles');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Plugins API', () => {
    test('GetPluginsV2 returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.GetPluginsV2');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Environment API', () => {
    test('DetectPythonEnvironments returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.DetectPythonEnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('DetectREnvironments returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.DetectREnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('GetVirtualEnvironments returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('GetRenvEnvironments returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('GetAllPluginEnvironmentBindings returns array', async () => {
      const result = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
      expect(Array.isArray(result)).toBe(true);
    });

    test('GetDefaultVenvPath returns path', async () => {
      const result = await mcp.callBoundMethod('main.App.GetDefaultVenvPath', 'test-plugin');
      expect(typeof result).toBe('string');
    });
  });
});

test.describe('Data Serialization Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('Jobs returns empty array not null', async () => {
    const result = await mcp.callBoundMethod('main.App.GetAllJobs');
    expect(result).not.toBeNull();
    expect(Array.isArray(result)).toBe(true);
  });

  test('Files returns empty array not null', async () => {
    const result = await mcp.callBoundMethod('main.App.GetImportedFiles');
    expect(result).not.toBeNull();
    expect(Array.isArray(result)).toBe(true);
  });

  test('Plugins returns empty array not null', async () => {
    const result = await mcp.callBoundMethod('main.App.GetPluginsV2');
    expect(result).not.toBeNull();
    expect(Array.isArray(result)).toBe(true);
  });

  test('Virtual environments returns empty array not null', async () => {
    const result = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
    expect(result).not.toBeNull();
    expect(Array.isArray(result)).toBe(true);
  });

  test('Renv environments returns empty array not null', async () => {
    const result = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
    expect(result).not.toBeNull();
    expect(Array.isArray(result)).toBe(true);
  });

  test('Plugin bindings returns empty array not null', async () => {
    const result = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
    expect(result).not.toBeNull();
    expect(Array.isArray(result)).toBe(true);
  });
});

test.describe('UI Control Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('Navigate to home', async () => {
    await mcp.navigate('#/');
    expect(await mcp.waitForElement('router-outlet, app-home, .home')).toBe(true);
  });

  test('Navigate to settings', async () => {
    await mcp.navigate('#/settings/general');
    expect(await mcp.getUrl()).toContain('#/settings/general');
  });

  test('Navigate to Python settings', async () => {
    await mcp.navigate('#/settings/python');
    expect(await mcp.waitForElement('.python-settings')).toBe(true);
  });

  test('Navigate to R settings', async () => {
    await mcp.navigate('#/settings/r');
    expect(await mcp.waitForElement('.r-settings')).toBe(true);
  });

  test('Navigate to jobs', async () => {
    await mcp.navigate('#/jobs');
    expect(await mcp.getUrl()).toContain('#/jobs');
  });

  test('Navigate to plugins', async () => {
    await mcp.navigate('#/plugins');
    expect(await mcp.getUrl()).toContain('#/plugins');
  });

  test('Navigate to plugin list', async () => {
    await mcp.navigate('#/plugin-list');
    expect(await mcp.getUrl()).toContain('#/plugin-list');
  });

  test('Execute custom JavaScript', async () => {
    const result = await mcp.callTool('js_eval', { js: 'return 1 + 1;' });
    expect(result).toBe(2);
  });
});
