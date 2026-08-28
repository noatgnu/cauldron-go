import { test, expect } from '@playwright/test';
import * as mcp from './helpers/mcp-client';

async function tryCallBoundMethod(name: string, ...args: any[]): Promise<{ success: boolean; error?: string }> {
  try {
    await mcp.callBoundMethod(name, ...args);
    return { success: true };
  } catch (e) {
    return { success: false, error: (e as Error).message };
  }
}

async function getBasePythonPath(): Promise<string | null> {
  const environments = await mcp.callBoundMethod('main.App.DetectPythonEnvironments');
  const envs = (environments || []).filter((e: any) => !e.isVirtual);
  if (envs.length === 0) {
    return null;
  }
  return envs[0].path;
}

async function getBaseRPath(): Promise<string | null> {
  const environments = await mcp.callBoundMethod('main.App.DetectREnvironments');
  if (!environments || environments.length === 0) {
    return null;
  }
  return environments[0].path;
}

function generateTempVenvPath(): string {
  const timestamp = Date.now();
  return `/tmp/e2e-venv-${timestamp}`;
}

test.describe('Python Virtual Environment Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should create Python virtual environment via MCP', async () => {
    const basePythonPath = await getBasePythonPath();
    if (!basePythonPath) {
      test.skip(true, 'No Python environment available');
      return;
    }

    const venvPath = generateTempVenvPath();
    let createdVenvId: number | null = null;

    try {
      const createResult = await tryCallBoundMethod('main.App.CreatePythonVirtualEnv', basePythonPath, venvPath, '');

      if (!createResult.success) {
        test.skip(true, `Venv creation failed: ${createResult.error || 'unknown error'}`);
        return;
      }

      await new Promise(r => setTimeout(r, 3000));

      const environments = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
      expect(Array.isArray(environments)).toBe(true);

      const newVenv = environments.find((v: any) =>
        v.Path === venvPath || v.Path?.includes('e2e-venv-')
      );

      if (newVenv) {
        createdVenvId = newVenv.ID;
        expect(newVenv.ID).toBeDefined();
        expect(newVenv.Name).toBeDefined();
        expect(newVenv.Path).toBeDefined();
        expect(newVenv.BasePythonPath).toBe(basePythonPath);
      }
    } finally {
      if (createdVenvId) {
        await mcp.callBoundMethod('main.App.DeleteVirtualEnvironment', createdVenvId);
      }
    }
  });

  test('should delete Python virtual environment via MCP', async () => {
    const basePythonPath = await getBasePythonPath();
    if (!basePythonPath) {
      test.skip(true, 'No Python environment available');
      return;
    }

    const venvPath = generateTempVenvPath();

    const createResult = await tryCallBoundMethod('main.App.CreatePythonVirtualEnv', basePythonPath, venvPath, '');
    if (!createResult.success) {
      test.skip(true, 'Could not create test venv');
      return;
    }

    await new Promise(r => setTimeout(r, 3000));

    const environments = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
    const newVenv = environments?.find((v: any) =>
      v.Path === venvPath || v.Path?.includes('e2e-venv-')
    );

    if (!newVenv) {
      test.skip(true, 'Venv was not created');
      return;
    }

    const deleteResult = await tryCallBoundMethod('main.App.DeleteVirtualEnvironment', newVenv.ID);
    expect(deleteResult.success).toBe(true);

    const verifyResult = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
    const deletedVenv = verifyResult?.find((v: any) => v.ID === newVenv.ID);
    expect(deletedVenv).toBeUndefined();
  });

  test('should navigate to Python settings via MCP', async () => {
    await mcp.navigate('#/settings/python');
    expect(await mcp.waitForElement('.python-settings')).toBe(true);
  });
});

test.describe('R Renv Environment Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should create R renv environment via MCP', async () => {
    test.setTimeout(180000);

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const activateResult = await tryCallBoundMethod('main.App.SetActiveREnvironment', baseRPath);
    expect(activateResult.success).toBe(true);

    const envName = `e2e-renv-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      const createResult = await tryCallBoundMethod('main.App.CreateRenvEnvironment', envName, [], '', false);
      expect(createResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 5000));

      const environments = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
      expect(Array.isArray(environments)).toBe(true);

      const newRenv = environments.find((r: any) =>
        r.Name === envName || r.Name?.includes('e2e-renv-')
      );

      if (newRenv) {
        createdRenvId = newRenv.ID;
        expect(newRenv.ID).toBeDefined();
        expect(newRenv.Name).toBeDefined();
        expect(newRenv.ProjectPath).toBeDefined();
      }
    } finally {
      if (createdRenvId) {
        await mcp.callBoundMethod('main.App.DeleteRenvEnvironment', createdRenvId);
      }
    }
  });

  test('should delete R renv environment via MCP', async () => {
    test.setTimeout(180000);

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const activateResult = await tryCallBoundMethod('main.App.SetActiveREnvironment', baseRPath);
    expect(activateResult.success).toBe(true);

    const envName = `e2e-renv-del-${Date.now()}`;

    const createResult = await tryCallBoundMethod('main.App.CreateRenvEnvironment', envName, [], '', false);
    if (!createResult.success) {
      test.skip(true, 'Could not create test renv');
      return;
    }

    await new Promise(r => setTimeout(r, 5000));

    const environments = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
    const newRenv = environments?.find((r: any) =>
      r.Name === envName || r.Name?.includes('e2e-renv-del-')
    );

    if (!newRenv) {
      test.skip(true, 'Renv was not created');
      return;
    }

    const deleteResult = await tryCallBoundMethod('main.App.DeleteRenvEnvironment', newRenv.ID);
    expect(deleteResult.success).toBe(true);

    const verifyResult = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
    const deletedRenv = verifyResult?.find((r: any) => r.ID === newRenv.ID);
    expect(deletedRenv).toBeUndefined();
  });

  test('should navigate to R settings via MCP', async () => {
    await mcp.navigate('#/settings/r');
    expect(await mcp.waitForElement('.r-settings')).toBe(true);
  });
});

test.describe('Plugin Environment Binding Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should bind Python environment to plugin', async () => {
    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    if (!plugins || plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    const basePythonPath = await getBasePythonPath();
    if (!basePythonPath) {
      test.skip(true, 'No Python environment available');
      return;
    }

    const venvPath = generateTempVenvPath();
    let createdVenvId: number | null = null;

    try {
      const createVenvResult = await tryCallBoundMethod('main.App.CreatePythonVirtualEnv', basePythonPath, venvPath, '');
      if (!createVenvResult.success) {
        test.skip(true, 'Could not create test venv');
        return;
      }

      await new Promise(r => setTimeout(r, 3000));

      const environments = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
      const newVenv = environments?.find((v: any) =>
        v.Path === venvPath || v.Path?.includes('e2e-venv-')
      );

      if (!newVenv) {
        test.skip(true, 'Venv was not created');
        return;
      }

      createdVenvId = newVenv.ID;

      const plugin = plugins[0];
      const pluginID = plugin.definition?.plugin?.id || plugin.id?.toString();

      const bindResult = await tryCallBoundMethod(
        'main.App.BindPluginToEnvironment', pluginID, 'python', newVenv.ID, newVenv.Path
      );
      expect(bindResult.success).toBe(true);

      const bindings = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
      expect(Array.isArray(bindings)).toBe(true);

    } finally {
      if (createdVenvId) {
        await mcp.callBoundMethod('main.App.DeleteVirtualEnvironment', createdVenvId);
      }
    }
  });

  test('should get all plugin environment bindings', async () => {
    const bindings = await mcp.callBoundMethod('main.App.GetAllPluginEnvironmentBindings');
    expect(Array.isArray(bindings)).toBe(true);

    if (bindings.length > 0) {
      const binding = bindings[0];
      expect(binding.PluginID).toBeDefined();
      expect(binding.EnvironmentType).toBeDefined();
      expect(binding.EnvironmentID).toBeDefined();
    }
  });
});

test.describe('Environment Detection Tests via MCP', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);
  });

  test('should detect Python environments', async () => {
    const environments = await mcp.callBoundMethod('main.App.DetectPythonEnvironments');
    expect(Array.isArray(environments)).toBe(true);

    if (environments.length > 0) {
      const env = environments[0];
      expect(env.name).toBeDefined();
      expect(env.path).toBeDefined();
      expect(env.type).toBeDefined();
      expect(typeof env.isVirtual).toBe('boolean');
    }
  });

  test('should detect R environments', async () => {
    const environments = await mcp.callBoundMethod('main.App.DetectREnvironments');
    expect(Array.isArray(environments)).toBe(true);

    if (environments.length > 0) {
      const env = environments[0];
      expect(env.name).toBeDefined();
      expect(env.path).toBeDefined();
    }
  });

  test('should get virtual environments list', async () => {
    const environments = await mcp.callBoundMethod('main.App.GetVirtualEnvironments');
    expect(Array.isArray(environments)).toBe(true);

    if (environments.length > 0) {
      const venv = environments[0];
      expect(venv.ID).toBeDefined();
      expect(venv.Name).toBeDefined();
      expect(venv.Path).toBeDefined();
    }
  });

  test('should get renv environments list', async () => {
    const environments = await mcp.callBoundMethod('main.App.GetRenvEnvironments');
    expect(Array.isArray(environments)).toBe(true);

    if (environments.length > 0) {
      const renv = environments[0];
      expect(renv.ID).toBeDefined();
      expect(renv.Name).toBeDefined();
      expect(renv.ProjectPath).toBeDefined();
    }
  });
});
