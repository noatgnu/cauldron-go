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

async function getBasePythonPath(): Promise<string | null> {
  const result = await callTestAPI('/test/python-environments');
  if (result.error || !result.environments) {
    return null;
  }

  const envs = result.environments.filter((e: any) => !e.isVirtual);
  if (envs.length === 0) {
    return null;
  }

  return envs[0].path;
}

async function getBaseRPath(): Promise<string | null> {
  const result = await callTestAPI('/test/r-environments');
  if (result.error || !result.environments || result.environments.length === 0) {
    return null;
  }

  return result.environments[0].path;
}

function generateTempVenvPath(): string {
  const timestamp = Date.now();
  return `/tmp/e2e-venv-${timestamp}`;
}

test.describe('Python Virtual Environment Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should create Python virtual environment via TestAPI', async () => {
    const basePythonPath = await getBasePythonPath();
    if (!basePythonPath) {
      test.skip(true, 'No Python environment available');
      return;
    }

    const venvPath = generateTempVenvPath();
    let createdVenvId: number | null = null;

    try {
      const createResult = await callTestAPI('/test/create-venv', 'POST', {
        basePythonPath,
        venvPath,
        pluginId: ''
      });

      expect(createResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 3000));

      const listResult = await callTestAPI('/test/virtual-environments');
      expect(Array.isArray(listResult.environments)).toBe(true);

      const newVenv = listResult.environments.find((v: any) =>
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
        await callTestAPI('/test/delete-venv', 'POST', { id: createdVenvId });
      }
    }
  });

  test('should delete Python virtual environment via TestAPI', async () => {
    const basePythonPath = await getBasePythonPath();
    if (!basePythonPath) {
      test.skip(true, 'No Python environment available');
      return;
    }

    const venvPath = generateTempVenvPath();

    const createResult = await callTestAPI('/test/create-venv', 'POST', {
      basePythonPath,
      venvPath,
      pluginId: ''
    });

    if (!createResult.success) {
      test.skip(true, 'Could not create test venv');
      return;
    }

    await new Promise(r => setTimeout(r, 3000));

    const listResult = await callTestAPI('/test/virtual-environments');
    const newVenv = listResult.environments?.find((v: any) =>
      v.Path === venvPath || v.Path?.includes('e2e-venv-')
    );

    if (!newVenv) {
      test.skip(true, 'Venv was not created');
      return;
    }

    const deleteResult = await callTestAPI('/test/delete-venv', 'POST', { id: newVenv.ID });
    expect(deleteResult.success).toBe(true);

    const verifyResult = await callTestAPI('/test/virtual-environments');
    const deletedVenv = verifyResult.environments?.find((v: any) => v.ID === newVenv.ID);
    expect(deletedVenv).toBeUndefined();
  });

  test('should navigate to Python settings via TestAPI', async () => {
    const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/python' });
    expect(navResult.success).toBe(true);

    await new Promise(r => setTimeout(r, 2000));
  });
});

test.describe('R Renv Environment Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should create R renv environment via TestAPI', async () => {
    test.setTimeout(180000);

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const envName = `e2e-renv-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      const createResult = await callTestAPI('/test/create-renv', 'POST', {
        name: envName,
        packages: [],
        pluginId: '',
        useCache: false
      });

      expect(createResult.success).toBe(true);

      await new Promise(r => setTimeout(r, 5000));

      const listResult = await callTestAPI('/test/renv-environments');
      expect(Array.isArray(listResult.environments)).toBe(true);

      const newRenv = listResult.environments.find((r: any) =>
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
        await callTestAPI('/test/delete-renv', 'POST', { id: createdRenvId });
      }
    }
  });

  test('should delete R renv environment via TestAPI', async () => {
    test.setTimeout(180000);

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const envName = `e2e-renv-del-${Date.now()}`;

    const createResult = await callTestAPI('/test/create-renv', 'POST', {
      name: envName,
      packages: [],
      pluginId: '',
      useCache: false
    });

    if (!createResult.success) {
      test.skip(true, 'Could not create test renv');
      return;
    }

    await new Promise(r => setTimeout(r, 5000));

    const listResult = await callTestAPI('/test/renv-environments');
    const newRenv = listResult.environments?.find((r: any) =>
      r.Name === envName || r.Name?.includes('e2e-renv-del-')
    );

    if (!newRenv) {
      test.skip(true, 'Renv was not created');
      return;
    }

    const deleteResult = await callTestAPI('/test/delete-renv', 'POST', { id: newRenv.ID });
    expect(deleteResult.success).toBe(true);

    const verifyResult = await callTestAPI('/test/renv-environments');
    const deletedRenv = verifyResult.environments?.find((r: any) => r.ID === newRenv.ID);
    expect(deletedRenv).toBeUndefined();
  });

  test('should navigate to R settings via TestAPI', async () => {
    const navResult = await callTestAPI('/test/ui/navigate', 'POST', { route: '#/settings/r' });
    expect(navResult.success).toBe(true);

    await new Promise(r => setTimeout(r, 2000));
  });
});

test.describe('Plugin Environment Binding Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should bind Python environment to plugin', async () => {
    const pluginsResult = await callTestAPI('/test/plugins');
    if (pluginsResult.error || !pluginsResult.plugins || pluginsResult.plugins.length === 0) {
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
      await callTestAPI('/test/create-venv', 'POST', {
        basePythonPath,
        venvPath,
        pluginId: ''
      });
      await new Promise(r => setTimeout(r, 3000));

      const venvsResult = await callTestAPI('/test/virtual-environments');
      const newVenv = venvsResult.environments?.find((v: any) =>
        v.Path === venvPath || v.Path?.includes('e2e-venv-')
      );

      if (!newVenv) {
        test.skip(true, 'Venv was not created');
        return;
      }

      createdVenvId = newVenv.ID;

      const plugin = pluginsResult.plugins[0];
      const pluginID = plugin.definition?.plugin?.id || plugin.id?.toString();

      const bindResult = await callTestAPI('/test/bind-plugin-environment', 'POST', {
        pluginId: pluginID,
        envType: 'python',
        envId: newVenv.ID,
        envPath: newVenv.Path
      });

      expect(bindResult.success).toBe(true);

      const bindingsResult = await callTestAPI('/test/plugin-bindings');
      expect(Array.isArray(bindingsResult.bindings)).toBe(true);

    } finally {
      if (createdVenvId) {
        await callTestAPI('/test/delete-venv', 'POST', { id: createdVenvId });
      }
    }
  });

  test('should get all plugin environment bindings', async () => {
    const result = await callTestAPI('/test/plugin-bindings');
    expect(result).not.toHaveProperty('error');
    expect(Array.isArray(result.bindings)).toBe(true);

    if (result.bindings && result.bindings.length > 0) {
      const binding = result.bindings[0];
      expect(binding.PluginID).toBeDefined();
      expect(binding.EnvironmentType).toBeDefined();
      expect(binding.EnvironmentID).toBeDefined();
    }
  });
});

test.describe('Environment Detection Tests via TestAPI', () => {
  test.beforeAll(async () => {
    const ready = await waitForWindow();
    expect(ready).toBe(true);
  });

  test('should detect Python environments', async () => {
    const result = await callTestAPI('/test/python-environments');
    expect(result).not.toHaveProperty('error');
    expect(Array.isArray(result.environments)).toBe(true);

    if (result.environments && result.environments.length > 0) {
      const env = result.environments[0];
      expect(env.name).toBeDefined();
      expect(env.path).toBeDefined();
      expect(env.type).toBeDefined();
      expect(typeof env.isVirtual).toBe('boolean');
    }
  });

  test('should detect R environments', async () => {
    const result = await callTestAPI('/test/r-environments');
    expect(result).not.toHaveProperty('error');
    expect(Array.isArray(result.environments)).toBe(true);

    if (result.environments && result.environments.length > 0) {
      const env = result.environments[0];
      expect(env.name).toBeDefined();
      expect(env.path).toBeDefined();
    }
  });

  test('should get virtual environments list', async () => {
    const result = await callTestAPI('/test/virtual-environments');
    expect(result).not.toHaveProperty('error');
    expect(Array.isArray(result.environments)).toBe(true);

    if (result.environments && result.environments.length > 0) {
      const venv = result.environments[0];
      expect(venv.ID).toBeDefined();
      expect(venv.Name).toBeDefined();
      expect(venv.Path).toBeDefined();
    }
  });

  test('should get renv environments list', async () => {
    const result = await callTestAPI('/test/renv-environments');
    expect(result).not.toHaveProperty('error');
    expect(Array.isArray(result.environments)).toBe(true);

    if (result.environments && result.environments.length > 0) {
      const renv = result.environments[0];
      expect(renv.ID).toBeDefined();
      expect(renv.Name).toBeDefined();
      expect(renv.ProjectPath).toBeDefined();
    }
  });
});
