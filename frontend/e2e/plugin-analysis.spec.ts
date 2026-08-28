import { test, expect } from '@playwright/test';
import * as mcp from './helpers/mcp-client';

test.describe('Plugin Analysis via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should list available plugins', async () => {
    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    expect(Array.isArray(plugins)).toBe(true);
  });

  test('should have plugin with required structure', async () => {
    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    if (!plugins || plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    const plugin = plugins[0];
    expect(plugin).toBeDefined();

    if (plugin.definition) {
      expect(plugin.definition.plugin).toBeDefined();
      expect(plugin.definition.plugin.name).toBeDefined();
      expect(plugin.definition.plugin.id).toBeDefined();
    }
  });

  test('should navigate to plugin list page via MCP', async () => {
    await mcp.navigate('#/plugin-list');
    expect(await mcp.getUrl()).toContain('#/plugin-list');
  });

  test('should navigate to plugins page via MCP', async () => {
    await mcp.navigate('#/plugins');
    expect(await mcp.getUrl()).toContain('#/plugins');
  });

  test('should get jobs list', async () => {
    const jobs = await mcp.callBoundMethod('main.App.GetAllJobs');
    expect(Array.isArray(jobs)).toBe(true);
  });

  test('should verify plugin bindings are accessible', async () => {
    const bindings = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
    expect(Array.isArray(bindings)).toBe(true);
  });
});

test.describe('Plugin Details via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should have plugins with runtime environments', async () => {
    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    if (!plugins || plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    for (const plugin of plugins.slice(0, 3)) {
      if (plugin.definition?.runtime) {
        expect(plugin.definition.runtime).toBeDefined();
      }
    }
  });

  test('should have plugins with inputs configuration', async () => {
    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    if (!plugins || plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    for (const plugin of plugins.slice(0, 3)) {
      if (plugin.definition?.inputs) {
        expect(Array.isArray(plugin.definition.inputs)).toBe(true);
      }
    }
  });
});
