import { test, expect } from '@playwright/test';
import * as mcp from './helpers/mcp-client';

test.describe('Cauldron E2E Integration Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test.describe('MCP Health', () => {
    test('should report app info with a visible window', async () => {
      const info = await mcp.appInfo();
      expect(info.windows?.[0]?.visible).toBe(true);
    });
  });

  test.describe('Settings', () => {
    test('should return settings', async () => {
      const result = await mcp.callBoundMethod('main.App.GetSettings');
      expect(result).not.toHaveProperty('error');
      expect(result).toBeDefined();
    });
  });

  test.describe('Jobs', () => {
    test('should return jobs list', async () => {
      const result = await mcp.callBoundMethod('main.App.GetAllJobs');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Plugins', () => {
    test('should return plugins list', async () => {
      const result = await mcp.callBoundMethod('main.App.GetPluginsV2');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Imported Files', () => {
    test('should return imported files list', async () => {
      const result = await mcp.callBoundMethod('main.App.GetImportedFiles');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('UI Navigation', () => {
    test('should navigate to settings page', async () => {
      await mcp.navigate('#/settings/general');
      expect(await mcp.getUrl()).toContain('#/settings/general');
    });

    test('should navigate to jobs page', async () => {
      await mcp.navigate('#/jobs');
      expect(await mcp.getUrl()).toContain('#/jobs');
    });

    test('should navigate to plugins page', async () => {
      await mcp.navigate('#/plugins');
      expect(await mcp.getUrl()).toContain('#/plugins');
    });

    test('should navigate to Python settings page', async () => {
      await mcp.navigate('#/settings/python');
      expect(await mcp.waitForElement('.python-settings')).toBe(true);
    });

    test('should navigate to R settings page', async () => {
      await mcp.navigate('#/settings/r');
      expect(await mcp.waitForElement('.r-settings')).toBe(true);
    });
  });

  test.describe('Environment Detection', () => {
    test('should detect Python environments', async () => {
      const result = await mcp.callBoundMethod('main.App.DetectPythonEnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('should detect R environments', async () => {
      const result = await mcp.callBoundMethod('main.App.DetectREnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('should get virtual environments', async () => {
      const result = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
      expect(Array.isArray(result)).toBe(true);
    });

    test('should get renv environments', async () => {
      const result = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Plugin Bindings', () => {
    test('should get plugin environment bindings', async () => {
      const result = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
      expect(Array.isArray(result)).toBe(true);
    });
  });

  test.describe('Default Paths', () => {
    test('should get default venv path', async () => {
      const result = await mcp.callBoundMethod('main.App.GetDefaultVenvPath', 'test');
      expect(typeof result).toBe('string');
    });
  });
});
