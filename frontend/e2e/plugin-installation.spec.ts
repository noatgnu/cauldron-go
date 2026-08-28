import { test, expect } from '@playwright/test';
import * as mcp from './helpers/mcp-client';

test.describe('Plugin Installation Workflow via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should list installed plugins', async () => {
    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    expect(Array.isArray(plugins)).toBe(true);
    expect(typeof plugins.length).toBe('number');
  });

  test('should navigate to plugin registry page via MCP', async () => {
    await mcp.navigate('#/plugin-registry');
    expect(await mcp.getUrl()).toContain('#/plugin-registry');
  });

  test('should navigate to plugins page via MCP', async () => {
    await mcp.navigate('#/plugins');
    expect(await mcp.getUrl()).toContain('#/plugins');
  });

  test('should have settings available', async () => {
    const settings = await mcp.callBoundMethod('main.App.GetSettings');
    expect(settings).toBeDefined();
  });
});

test.describe('Plugin Environment Management via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should get plugin bindings', async () => {
    const bindings = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
    expect(Array.isArray(bindings)).toBe(true);
  });

  test('should get virtual environments for plugin binding', async () => {
    const venvs = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
    expect(Array.isArray(venvs)).toBe(true);
  });

  test('should get renv environments for plugin binding', async () => {
    const renvs = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
    expect(Array.isArray(renvs)).toBe(true);
  });
});
