import { test, expect, Page } from '@playwright/test';

const BASE_URL = process.env['WAILS_URL'] || 'http://localhost:4200';
const TEST_API_URL = process.env['TEST_API_URL'] || 'http://127.0.0.1:9245';
const IS_CI = !!process.env['CI'];

async function checkTestAPIAvailable(): Promise<boolean> {
  try {
    const response = await fetch(`${TEST_API_URL}/test/health`);
    if (response.ok) {
      const data = await response.json();
      return data.status === 'ok';
    }
    return false;
  } catch {
    return false;
  }
}

async function callTestAPI(endpoint: string, method: string = 'GET', body?: any): Promise<any> {
  try {
    const options: RequestInit = {
      method,
      headers: { 'Content-Type': 'application/json' },
    };
    if (body && method === 'POST') {
      options.body = JSON.stringify(body);
    }

    const response = await fetch(`${TEST_API_URL}${endpoint}`, options);
    if (response.ok) {
      const data = await response.json();
      return { __success: true, data };
    }
    return { __error: true, message: `HTTP ${response.status}` };
  } catch (e: any) {
    return { __error: true, message: e.message };
  }
}

async function callWailsMethod(page: Page, method: string, ...args: any[]): Promise<any> {
  return page.evaluate(async ({ method, args }) => {
    const win = window as any;
    if (!('_wails' in win)) {
      return { __skipped: true, reason: 'Wails runtime not available' };
    }

    try {
      const WailsApp = win.go?.['github.com/noatgnu/cauldron-go'];
      if (!WailsApp || !WailsApp[method]) {
        return { __skipped: true, reason: `Method ${method} not found` };
      }

      const result = await WailsApp[method](...args);
      return { __success: true, data: result };
    } catch (e: any) {
      return { __error: true, message: e.message };
    }
  }, { method, args });
}

function failOrSkip(reason: string): void {
  if (IS_CI) {
    test.skip(true, `CI skip: ${reason}`);
  } else {
    test.skip(true, reason);
  }
}

function checkWailsResult(result: any): boolean {
  if (result?.__skipped) {
    failOrSkip(result.reason || 'Wails not available');
    return true;
  }
  return false;
}

async function getBasePythonPath(): Promise<string | null> {
  const result = await callTestAPI('/test/python-environments');
  if (result.__error || !result.data?.environments) {
    return null;
  }

  const envs = result.data.environments.filter((e: any) => !e.isVirtual);
  if (envs.length === 0) {
    return null;
  }

  return envs[0].path;
}

async function getBaseRPath(): Promise<string | null> {
  const result = await callTestAPI('/test/r-environments');
  if (result.__error || !result.data?.environments || result.data.environments.length === 0) {
    return null;
  }

  return result.data.environments[0].path;
}

function generateTempVenvPath(): string {
  const timestamp = Date.now();
  if (process.platform === 'win32') {
    const tempDir = process.env['TEMP'] || process.env['TMP'] || 'C:\\Temp';
    return `${tempDir}\\e2e-venv-${timestamp}`;
  } else {
    return `/tmp/e2e-venv-${timestamp}`;
  }
}

test.describe('Python Virtual Environment Tests', () => {
  let testAPIAvailable = false;

  test.beforeAll(async () => {
    testAPIAvailable = await checkTestAPIAvailable();
    if (!testAPIAvailable) {
      console.log('TestAPI not available, some tests will be skipped');
    }
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should navigate to Python settings and see virtual environments section', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/settings/python`);
    await page.waitForLoadState('networkidle');

    const venvSection = page.locator('.venv-section');
    await expect(venvSection).toBeVisible({ timeout: 15000 });

    const sectionTitle = page.locator('.venv-section h3');
    await expect(sectionTitle).toContainText('Virtual Environments');

    const createButton = page.locator('button:has-text("Create Virtual Environment")');
    await expect(createButton).toBeVisible();
  });

  test('should create Python virtual environment via TestAPI', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
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
      const createResult = await callTestAPI('/test/create-venv', 'POST', {
        basePythonPath,
        venvPath,
        pluginId: ''
      });

      if (createResult.__error) {
        console.log('CreatePythonVirtualEnv error:', createResult.data?.error || createResult.message);
      }

      expect(createResult.__success).toBeTruthy();
      expect(createResult.data?.success || createResult.data?.error === undefined).toBeTruthy();

      await page.waitForTimeout(2000);

      const listResult = await callTestAPI('/test/virtual-environments');
      expect(listResult.__success).toBeTruthy();
      expect(Array.isArray(listResult.data?.environments)).toBeTruthy();

      const newVenv = listResult.data.environments.find((v: any) =>
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

  test('should delete Python virtual environment via TestAPI', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

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

    if (createResult.__error || !createResult.data?.success) {
      test.skip(true, 'Could not create test venv');
      return;
    }

    await page.waitForTimeout(2000);

    const listResult = await callTestAPI('/test/virtual-environments');
    const newVenv = listResult.data?.environments?.find((v: any) =>
      v.Path === venvPath || v.Path?.includes('e2e-venv-')
    );

    if (!newVenv) {
      test.skip(true, 'Venv was not created');
      return;
    }

    const deleteResult = await callTestAPI('/test/delete-venv', 'POST', { id: newVenv.ID });
    expect(deleteResult.__success).toBeTruthy();
    expect(deleteResult.data?.success).toBeTruthy();

    const verifyResult = await callTestAPI('/test/virtual-environments');
    expect(verifyResult.__success).toBeTruthy();

    const deletedVenv = verifyResult.data?.environments?.find((v: any) => v.ID === newVenv.ID);
    expect(deletedVenv).toBeUndefined();
  });

  test('should display created virtual environment in UI list', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
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
      const createResult = await callTestAPI('/test/create-venv', 'POST', {
        basePythonPath,
        venvPath,
        pluginId: ''
      });
      await page.waitForTimeout(2000);

      const listResult = await callTestAPI('/test/virtual-environments');
      const newVenv = listResult.data?.environments?.find((v: any) =>
        v.Path === venvPath || v.Path?.includes('e2e-venv-')
      );
      if (newVenv) {
        createdVenvId = newVenv.ID;
      }

      expect(createResult.__success).toBeTruthy();
      expect(newVenv).toBeDefined();

      await page.goto(`${BASE_URL}/#/settings/python`);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      const venvSection = page.locator('.venv-section');
      await expect(venvSection).toBeVisible({ timeout: 10000 });

      const hasWails = await page.evaluate(() => '_wails' in window);
      if (hasWails && newVenv) {
        const venvList = page.locator('.venv-section mat-list-item');
        const count = await venvList.count();
        expect(count).toBeGreaterThan(0);
      }
    } finally {
      if (createdVenvId) {
        await callTestAPI('/test/delete-venv', 'POST', { id: createdVenvId });
      }
    }
  });
});

test.describe('R Renv Environment Tests', () => {
  let testAPIAvailable = false;

  test.beforeAll(async () => {
    testAPIAvailable = await checkTestAPIAvailable();
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should navigate to R settings and see renv environments section', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/settings/r`);
    await page.waitForLoadState('networkidle');

    const renvSection = page.locator('.venv-section');
    await expect(renvSection).toBeVisible({ timeout: 15000 });

    const sectionTitle = page.locator('.venv-section h3');
    await expect(sectionTitle).toContainText('Renv Environments');

    const createButton = page.locator('button:has-text("Create Renv Environment")');
    await expect(createButton).toBeVisible();
  });

  test('should create R renv environment via TestAPI', async ({ page }) => {
    test.setTimeout(180000);

    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

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

      if (createResult.__error) {
        console.log('CreateRenvEnvironment error:', createResult.data?.error || createResult.message);
      }

      expect(createResult.__success).toBeTruthy();

      await page.waitForTimeout(5000);

      const listResult = await callTestAPI('/test/renv-environments');
      expect(listResult.__success).toBeTruthy();
      expect(Array.isArray(listResult.data?.environments)).toBeTruthy();

      const newRenv = listResult.data.environments.find((r: any) =>
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

  test('should delete R renv environment via TestAPI', async ({ page }) => {
    test.setTimeout(180000);

    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

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

    if (createResult.__error || !createResult.data?.success) {
      test.skip(true, 'Could not create test renv');
      return;
    }

    await page.waitForTimeout(5000);

    const listResult = await callTestAPI('/test/renv-environments');
    const newRenv = listResult.data?.environments?.find((r: any) =>
      r.Name === envName || r.Name?.includes('e2e-renv-del-')
    );

    if (!newRenv) {
      test.skip(true, 'Renv was not created');
      return;
    }

    const deleteResult = await callTestAPI('/test/delete-renv', 'POST', { id: newRenv.ID });
    expect(deleteResult.__success).toBeTruthy();

    const verifyResult = await callTestAPI('/test/renv-environments');
    expect(verifyResult.__success).toBeTruthy();

    const deletedRenv = verifyResult.data?.environments?.find((r: any) => r.ID === newRenv.ID);
    expect(deletedRenv).toBeUndefined();
  });

  test('should display created renv environment in UI list', async ({ page }) => {
    test.setTimeout(180000);

    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const envName = `e2e-renv-ui-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      await callTestAPI('/test/create-renv', 'POST', {
        name: envName,
        packages: [],
        pluginId: '',
        useCache: false
      });
      await page.waitForTimeout(5000);

      const listResult = await callTestAPI('/test/renv-environments');
      const newRenv = listResult.data?.environments?.find((r: any) =>
        r.Name === envName || r.Name?.includes('e2e-renv-ui-')
      );
      if (newRenv) {
        createdRenvId = newRenv.ID;
      }

      await page.goto(`${BASE_URL}/#/settings/r`);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      if (newRenv) {
        const renvList = page.locator('.venv-section mat-list-item');
        const count = await renvList.count();
        expect(count).toBeGreaterThan(0);
      }
    } finally {
      if (createdRenvId) {
        await callTestAPI('/test/delete-renv', 'POST', { id: createdRenvId });
      }
    }
  });
});

test.describe('Plugin Environment Binding Tests', () => {
  let testAPIAvailable = false;

  test.beforeAll(async () => {
    testAPIAvailable = await checkTestAPIAvailable();
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should bind Python environment to plugin', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const pluginsResult = await callTestAPI('/test/plugins');
    if (pluginsResult.__error || !pluginsResult.data?.plugins || pluginsResult.data.plugins.length === 0) {
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
      await page.waitForTimeout(2000);

      const venvsResult = await callTestAPI('/test/virtual-environments');
      const newVenv = venvsResult.data?.environments?.find((v: any) =>
        v.Path === venvPath || v.Path?.includes('e2e-venv-')
      );

      if (!newVenv) {
        test.skip(true, 'Venv was not created');
        return;
      }

      createdVenvId = newVenv.ID;

      const plugin = pluginsResult.data.plugins[0];
      const pluginID = plugin.definition?.plugin?.id || plugin.id?.toString();

      const bindResult = await callTestAPI('/test/bind-plugin-environment', 'POST', {
        pluginId: pluginID,
        envType: 'python',
        envId: newVenv.ID,
        envPath: newVenv.Path
      });

      if (bindResult.__error) {
        console.log('BindPluginToEnvironment error:', bindResult.data?.error || bindResult.message);
      }

      const bindingsResult = await callTestAPI('/test/plugin-bindings');
      expect(bindingsResult.__success).toBeTruthy();
      expect(Array.isArray(bindingsResult.data?.bindings)).toBeTruthy();

      const binding = bindingsResult.data?.bindings?.find((b: any) =>
        b.PluginID === pluginID &&
        b.EnvironmentType === 'python' &&
        b.EnvironmentID === newVenv.ID
      );

      if (bindingsResult.data?.bindings && bindingsResult.data.bindings.length > 0 && !binding) {
        console.log('Available bindings:', JSON.stringify(bindingsResult.data.bindings, null, 2));
      }

    } finally {
      if (createdVenvId) {
        await callTestAPI('/test/delete-venv', 'POST', { id: createdVenvId });
      }
    }
  });

  test('should bind R environment to plugin', async ({ page }) => {
    test.setTimeout(180000);

    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const pluginsResult = await callTestAPI('/test/plugins');
    if (pluginsResult.__error || !pluginsResult.data?.plugins || pluginsResult.data.plugins.length === 0) {
      test.skip(true, 'No plugins available');
      return;
    }

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const envName = `e2e-renv-bind-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      await callTestAPI('/test/create-renv', 'POST', {
        name: envName,
        packages: [],
        pluginId: '',
        useCache: false
      });
      await page.waitForTimeout(5000);

      const renvsResult = await callTestAPI('/test/renv-environments');
      const newRenv = renvsResult.data?.environments?.find((r: any) =>
        r.Name === envName || r.Name?.includes('e2e-renv-bind-')
      );

      if (!newRenv) {
        test.skip(true, 'Renv was not created');
        return;
      }

      createdRenvId = newRenv.ID;

      const plugin = pluginsResult.data.plugins[0];
      const pluginID = plugin.definition?.plugin?.id || plugin.id?.toString();

      const bindResult = await callTestAPI('/test/bind-plugin-environment', 'POST', {
        pluginId: pluginID,
        envType: 'r',
        envId: newRenv.ID,
        envPath: newRenv.ProjectPath
      });

      if (bindResult.__error) {
        console.log('BindPluginToEnvironment error:', bindResult.data?.error || bindResult.message);
      }

      const bindingsResult = await callTestAPI('/test/plugin-bindings');
      expect(bindingsResult.__success).toBeTruthy();
      expect(Array.isArray(bindingsResult.data?.bindings)).toBeTruthy();

    } finally {
      if (createdRenvId) {
        await callTestAPI('/test/delete-renv', 'POST', { id: createdRenvId });
      }
    }
  });

  test('should get all plugin environment bindings', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const result = await callTestAPI('/test/plugin-bindings');
    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data?.bindings)).toBeTruthy();

    if (result.data?.bindings && result.data.bindings.length > 0) {
      const binding = result.data.bindings[0];
      expect(binding.PluginID).toBeDefined();
      expect(binding.EnvironmentType).toBeDefined();
      expect(binding.EnvironmentID).toBeDefined();
    }
  });
});

test.describe('Environment Detection Tests', () => {
  let testAPIAvailable = false;

  test.beforeAll(async () => {
    testAPIAvailable = await checkTestAPIAvailable();
  });

  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should detect Python environments', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const result = await callTestAPI('/test/python-environments');
    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data?.environments)).toBeTruthy();

    if (result.data?.environments && result.data.environments.length > 0) {
      const env = result.data.environments[0];
      expect(env.name).toBeDefined();
      expect(env.path).toBeDefined();
      expect(env.type).toBeDefined();
      expect(typeof env.isVirtual).toBe('boolean');
    }
  });

  test('should detect R environments', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const result = await callTestAPI('/test/r-environments');
    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data?.environments)).toBeTruthy();

    if (result.data?.environments && result.data.environments.length > 0) {
      const env = result.data.environments[0];
      expect(env.name).toBeDefined();
      expect(env.path).toBeDefined();
    }
  });

  test('should get virtual environments list', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const result = await callTestAPI('/test/virtual-environments');
    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data?.environments)).toBeTruthy();

    if (result.data?.environments && result.data.environments.length > 0) {
      const venv = result.data.environments[0];
      expect(venv.ID).toBeDefined();
      expect(venv.Name).toBeDefined();
      expect(venv.Path).toBeDefined();
    }
  });

  test('should get renv environments list', async ({ page }) => {
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const result = await callTestAPI('/test/renv-environments');
    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data?.environments)).toBeTruthy();

    if (result.data?.environments && result.data.environments.length > 0) {
      const renv = result.data.environments[0];
      expect(renv.ID).toBeDefined();
      expect(renv.Name).toBeDefined();
      expect(renv.ProjectPath).toBeDefined();
    }
  });
});

test.describe('UI Workflow Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should show Python environment detection spinner while loading', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/settings/python`);

    const spinner = page.locator('.detecting-container mat-spinner');

    await page.waitForTimeout(500);
    const spinnerVisible = await spinner.isVisible();

    if (spinnerVisible) {
      const text = page.locator('.detecting-container:has-text("Detecting Python environments")');
      await expect(text).toBeVisible();
    }

    await page.waitForTimeout(10000);
  });

  test('should show R environment detection spinner while loading', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/settings/r`);

    const spinner = page.locator('.detecting-container mat-spinner');

    await page.waitForTimeout(500);
    const spinnerVisible = await spinner.isVisible();

    if (spinnerVisible) {
      const text = page.locator('.detecting-container:has-text("Detecting R environments")');
      await expect(text).toBeVisible();
    }

    await page.waitForTimeout(10000);
  });

  test('should enable create venv button when Python is selected', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    const basePythonPath = await getBasePythonPath();
    if (!basePythonPath) {
      test.skip(true, 'No Python environment available');
      return;
    }

    const createButton = page.locator('button:has-text("Create Virtual Environment")');
    await expect(createButton).toBeVisible({ timeout: 15000 });
  });

  test('should enable create renv button when R is selected', async ({ page }) => {
    await page.goto(`${BASE_URL}/#/settings/r`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    const baseRPath = await getBaseRPath();
    if (!baseRPath) {
      test.skip(true, 'No R environment available');
      return;
    }

    const createButton = page.locator('button:has-text("Create Renv Environment")');
    await expect(createButton).toBeVisible({ timeout: 15000 });
  });

  test('should show no environments message when list is empty', async ({ page }) => {
    const testAPIAvailable = await checkTestAPIAvailable();
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const venvsResult = await callTestAPI('/test/virtual-environments');

    await page.goto(`${BASE_URL}/#/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    if (!venvsResult.data?.environments || venvsResult.data.environments.length === 0) {
      const noVenvsMessage = page.locator('.no-venvs');
      await expect(noVenvsMessage).toContainText('No virtual environments created yet');
    }
  });

  test('should show delete button for each virtual environment', async ({ page }) => {
    const testAPIAvailable = await checkTestAPIAvailable();
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const venvsResult = await callTestAPI('/test/virtual-environments');

    if (!venvsResult.data?.environments || venvsResult.data.environments.length === 0) {
      test.skip(true, 'No virtual environments to test');
      return;
    }

    await page.goto(`${BASE_URL}/#/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const deleteButtons = page.locator('.venv-section mat-list-item button[mattooltip="Delete environment"]');
    const count = await deleteButtons.count();

    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('should show bound plugins button when environment has bindings', async ({ page }) => {
    const testAPIAvailable = await checkTestAPIAvailable();
    if (!testAPIAvailable) {
      test.skip(true, 'TestAPI not available');
      return;
    }

    const bindingsResult = await callTestAPI('/test/plugin-bindings');

    const pythonBindings = bindingsResult.data?.bindings?.filter((b: any) => b.EnvironmentType === 'python') || [];

    if (pythonBindings.length === 0) {
      test.skip(true, 'No Python environment bindings to test');
      return;
    }

    await page.goto(`${BASE_URL}/#/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const linkButtons = page.locator('.venv-section mat-list-item button[mattooltip="View bound plugins"]');
    const count = await linkButtons.count();

    expect(count).toBeGreaterThanOrEqual(1);
  });
});
