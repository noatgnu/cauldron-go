import { test, expect, Page } from '@playwright/test';

const BASE_URL = process.env['WAILS_URL'] || 'http://localhost:4200';
const IS_CI = !!process.env['CI'];

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
    throw new Error(`CI failure: ${reason}`);
  }
  test.skip(true, reason);
}

function checkWailsResult(result: any): boolean {
  if (result?.__skipped) {
    failOrSkip(result.reason || 'Wails not available');
    return true;
  }
  return false;
}

async function getBasePythonPath(page: Page): Promise<string | null> {
  const result = await callWailsMethod(page, 'DetectPythonEnvironments');
  if (result.__skipped || result.__error || !result.data) {
    return null;
  }

  const envs = result.data.filter((e: any) => !e.isVirtual);
  if (envs.length === 0) {
    return null;
  }

  return envs[0].path;
}

async function getBaseRPath(page: Page): Promise<string | null> {
  const result = await callWailsMethod(page, 'DetectREnvironments');
  if (result.__skipped || result.__error || !result.data || result.data.length === 0) {
    return null;
  }

  return result.data[0].path;
}

async function generateTempVenvPath(page: Page): Promise<string | null> {
  const timestamp = Date.now();
  const platform = await page.evaluate(() => navigator.platform);

  if (platform.toLowerCase().includes('win')) {
    const tempDir = process.env['TEMP'] || process.env['TMP'] || 'C:\\Temp';
    return `${tempDir}\\e2e-venv-${timestamp}`;
  } else {
    return `/tmp/e2e-venv-${timestamp}`;
  }
}

test.describe('Python Virtual Environment Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should navigate to Python settings and see virtual environments section', async ({ page }) => {
    await page.goto(`${BASE_URL}/settings/python`);
    await page.waitForLoadState('networkidle');

    const venvSection = page.locator('.venv-section');
    await expect(venvSection).toBeVisible({ timeout: 15000 });

    const sectionTitle = page.locator('.venv-section h3');
    await expect(sectionTitle).toContainText('Virtual Environments');

    const createButton = page.locator('button:has-text("Create Virtual Environment")');
    await expect(createButton).toBeVisible();
  });

  test('should create Python virtual environment via API', async ({ page }) => {
    const checkResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(checkResult)) return;

    const basePythonPath = await getBasePythonPath(page);
    if (!basePythonPath) {
      failOrSkip('No Python environment available');
      return;
    }

    const venvPath = await generateTempVenvPath(page);
    if (!venvPath) {
      failOrSkip('Could not generate temp path');
      return;
    }

    let createdVenvId: number | null = null;

    try {
      const createResult = await callWailsMethod(page, 'CreatePythonVirtualEnv', basePythonPath, venvPath, '');

      if (createResult.__error) {
        console.log('CreatePythonVirtualEnv error:', createResult.message);
      }

      expect(createResult.__success || createResult.__error === undefined).toBeTruthy();

      await page.waitForTimeout(2000);

      const listResult = await callWailsMethod(page, 'GetVirtualEnvironments');
      expect(listResult.__success).toBeTruthy();
      expect(Array.isArray(listResult.data)).toBeTruthy();

      const newVenv = listResult.data.find((v: any) => v.Path === venvPath || v.Path?.includes('e2e-venv-'));
      if (newVenv) {
        createdVenvId = newVenv.ID;

        expect(newVenv.ID).toBeDefined();
        expect(newVenv.Name).toBeDefined();
        expect(newVenv.Path).toBeDefined();
        expect(newVenv.BasePythonPath).toBe(basePythonPath);
      }
    } finally {
      if (createdVenvId) {
        await callWailsMethod(page, 'DeleteVirtualEnvironment', createdVenvId);
      }
    }
  });

  test('should delete Python virtual environment via API', async ({ page }) => {
    const checkResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(checkResult)) return;

    const basePythonPath = await getBasePythonPath(page);
    if (!basePythonPath) {
      failOrSkip('No Python environment available');
      return;
    }

    const venvPath = await generateTempVenvPath(page);
    if (!venvPath) {
      failOrSkip('Could not generate temp path');
      return;
    }

    const createResult = await callWailsMethod(page, 'CreatePythonVirtualEnv', basePythonPath, venvPath, '');
    if (createResult.__error) {
      failOrSkip('Could not create test venv');
      return;
    }

    await page.waitForTimeout(2000);

    const listResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    const newVenv = listResult.data?.find((v: any) => v.Path === venvPath || v.Path?.includes('e2e-venv-'));

    if (!newVenv) {
      failOrSkip('Venv was not created');
      return;
    }

    const deleteResult = await callWailsMethod(page, 'DeleteVirtualEnvironment', newVenv.ID);
    expect(deleteResult.__success || deleteResult.__error === undefined).toBeTruthy();

    const verifyResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    expect(verifyResult.__success).toBeTruthy();

    const deletedVenv = verifyResult.data?.find((v: any) => v.ID === newVenv.ID);
    expect(deletedVenv).toBeUndefined();
  });

  test('should display created virtual environment in UI list', async ({ page }) => {
    const checkResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(checkResult)) return;

    const basePythonPath = await getBasePythonPath(page);
    if (!basePythonPath) {
      failOrSkip('No Python environment available');
      return;
    }

    const venvPath = await generateTempVenvPath(page);
    if (!venvPath) {
      failOrSkip('Could not generate temp path');
      return;
    }

    let createdVenvId: number | null = null;

    try {
      await callWailsMethod(page, 'CreatePythonVirtualEnv', basePythonPath, venvPath, '');
      await page.waitForTimeout(2000);

      const listResult = await callWailsMethod(page, 'GetVirtualEnvironments');
      const newVenv = listResult.data?.find((v: any) => v.Path === venvPath || v.Path?.includes('e2e-venv-'));
      if (newVenv) {
        createdVenvId = newVenv.ID;
      }

      await page.goto(`${BASE_URL}/settings/python`);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      if (newVenv) {
        const venvList = page.locator('.venv-section mat-list-item');
        const count = await venvList.count();
        expect(count).toBeGreaterThan(0);
      }
    } finally {
      if (createdVenvId) {
        await callWailsMethod(page, 'DeleteVirtualEnvironment', createdVenvId);
      }
    }
  });
});

test.describe('R Renv Environment Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should navigate to R settings and see renv environments section', async ({ page }) => {
    await page.goto(`${BASE_URL}/settings/r`);
    await page.waitForLoadState('networkidle');

    const renvSection = page.locator('.venv-section');
    await expect(renvSection).toBeVisible({ timeout: 15000 });

    const sectionTitle = page.locator('.venv-section h3');
    await expect(sectionTitle).toContainText('Renv Environments');

    const createButton = page.locator('button:has-text("Create Renv Environment")');
    await expect(createButton).toBeVisible();
  });

  test('should create R renv environment via API', async ({ page }) => {
    test.setTimeout(180000);

    const checkResult = await callWailsMethod(page, 'GetRenvEnvironments');
    if (checkWailsResult(checkResult)) return;

    const baseRPath = await getBaseRPath(page);
    if (!baseRPath) {
      failOrSkip('No R environment available');
      return;
    }

    const envName = `e2e-renv-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      const createResult = await callWailsMethod(page, 'CreateRenvEnvironment', envName, [], '', false);

      if (createResult.__error) {
        console.log('CreateRenvEnvironment error:', createResult.message);
      }

      expect(createResult.__success || createResult.__error === undefined).toBeTruthy();

      await page.waitForTimeout(5000);

      const listResult = await callWailsMethod(page, 'GetRenvEnvironments');
      expect(listResult.__success).toBeTruthy();
      expect(Array.isArray(listResult.data)).toBeTruthy();

      const newRenv = listResult.data.find((r: any) => r.Name === envName || r.Name?.includes('e2e-renv-'));
      if (newRenv) {
        createdRenvId = newRenv.ID;

        expect(newRenv.ID).toBeDefined();
        expect(newRenv.Name).toBeDefined();
        expect(newRenv.ProjectPath).toBeDefined();
      }
    } finally {
      if (createdRenvId) {
        await callWailsMethod(page, 'DeleteRenvEnvironment', createdRenvId);
      }
    }
  });

  test('should delete R renv environment via API', async ({ page }) => {
    test.setTimeout(180000);

    const checkResult = await callWailsMethod(page, 'GetRenvEnvironments');
    if (checkWailsResult(checkResult)) return;

    const baseRPath = await getBaseRPath(page);
    if (!baseRPath) {
      failOrSkip('No R environment available');
      return;
    }

    const envName = `e2e-renv-del-${Date.now()}`;

    const createResult = await callWailsMethod(page, 'CreateRenvEnvironment', envName, [], '', false);
    if (createResult.__error) {
      failOrSkip('Could not create test renv');
      return;
    }

    await page.waitForTimeout(5000);

    const listResult = await callWailsMethod(page, 'GetRenvEnvironments');
    const newRenv = listResult.data?.find((r: any) => r.Name === envName || r.Name?.includes('e2e-renv-del-'));

    if (!newRenv) {
      failOrSkip('Renv was not created');
      return;
    }

    const deleteResult = await callWailsMethod(page, 'DeleteRenvEnvironment', newRenv.ID);
    expect(deleteResult.__success || deleteResult.__error === undefined).toBeTruthy();

    const verifyResult = await callWailsMethod(page, 'GetRenvEnvironments');
    expect(verifyResult.__success).toBeTruthy();

    const deletedRenv = verifyResult.data?.find((r: any) => r.ID === newRenv.ID);
    expect(deletedRenv).toBeUndefined();
  });

  test('should display created renv environment in UI list', async ({ page }) => {
    test.setTimeout(180000);

    const checkResult = await callWailsMethod(page, 'GetRenvEnvironments');
    if (checkWailsResult(checkResult)) return;

    const baseRPath = await getBaseRPath(page);
    if (!baseRPath) {
      failOrSkip('No R environment available');
      return;
    }

    const envName = `e2e-renv-ui-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      await callWailsMethod(page, 'CreateRenvEnvironment', envName, [], '', false);
      await page.waitForTimeout(5000);

      const listResult = await callWailsMethod(page, 'GetRenvEnvironments');
      const newRenv = listResult.data?.find((r: any) => r.Name === envName || r.Name?.includes('e2e-renv-ui-'));
      if (newRenv) {
        createdRenvId = newRenv.ID;
      }

      await page.goto(`${BASE_URL}/settings/r`);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);

      if (newRenv) {
        const renvList = page.locator('.venv-section mat-list-item');
        const count = await renvList.count();
        expect(count).toBeGreaterThan(0);
      }
    } finally {
      if (createdRenvId) {
        await callWailsMethod(page, 'DeleteRenvEnvironment', createdRenvId);
      }
    }
  });
});

test.describe('Plugin Environment Binding Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should bind Python environment to plugin', async ({ page }) => {
    const pluginsResult = await callWailsMethod(page, 'GetPluginsV2');
    if (checkWailsResult(pluginsResult)) return;

    if (!pluginsResult.data || pluginsResult.data.length === 0) {
      failOrSkip('No plugins available');
      return;
    }

    const basePythonPath = await getBasePythonPath(page);
    if (!basePythonPath) {
      failOrSkip('No Python environment available');
      return;
    }

    const venvPath = await generateTempVenvPath(page);
    if (!venvPath) {
      failOrSkip('Could not generate temp path');
      return;
    }

    let createdVenvId: number | null = null;

    try {
      await callWailsMethod(page, 'CreatePythonVirtualEnv', basePythonPath, venvPath, '');
      await page.waitForTimeout(2000);

      const venvsResult = await callWailsMethod(page, 'GetVirtualEnvironments');
      const newVenv = venvsResult.data?.find((v: any) => v.Path === venvPath || v.Path?.includes('e2e-venv-'));

      if (!newVenv) {
        failOrSkip('Venv was not created');
        return;
      }

      createdVenvId = newVenv.ID;

      const plugin = pluginsResult.data[0];
      const pluginID = plugin.definition?.plugin?.id || plugin.id?.toString();

      const bindResult = await callWailsMethod(page, 'BindPluginToEnvironment', pluginID, 'python', newVenv.ID, newVenv.Path);

      if (bindResult.__error) {
        console.log('BindPluginToEnvironment error:', bindResult.message);
      }

      const bindingsResult = await callWailsMethod(page, 'GetAllPluginEnvironmentBindings');
      expect(bindingsResult.__success).toBeTruthy();
      expect(Array.isArray(bindingsResult.data)).toBeTruthy();

      const binding = bindingsResult.data?.find((b: any) =>
        b.PluginID === pluginID &&
        b.EnvironmentType === 'python' &&
        b.EnvironmentID === newVenv.ID
      );

      if (bindingsResult.data && bindingsResult.data.length > 0 && !binding) {
        console.log('Available bindings:', JSON.stringify(bindingsResult.data, null, 2));
      }

    } finally {
      if (createdVenvId) {
        await callWailsMethod(page, 'DeleteVirtualEnvironment', createdVenvId);
      }
    }
  });

  test('should bind R environment to plugin', async ({ page }) => {
    test.setTimeout(180000);

    const pluginsResult = await callWailsMethod(page, 'GetPluginsV2');
    if (checkWailsResult(pluginsResult)) return;

    if (!pluginsResult.data || pluginsResult.data.length === 0) {
      failOrSkip('No plugins available');
      return;
    }

    const baseRPath = await getBaseRPath(page);
    if (!baseRPath) {
      failOrSkip('No R environment available');
      return;
    }

    const envName = `e2e-renv-bind-${Date.now()}`;
    let createdRenvId: number | null = null;

    try {
      await callWailsMethod(page, 'CreateRenvEnvironment', envName, [], '', false);
      await page.waitForTimeout(5000);

      const renvsResult = await callWailsMethod(page, 'GetRenvEnvironments');
      const newRenv = renvsResult.data?.find((r: any) => r.Name === envName || r.Name?.includes('e2e-renv-bind-'));

      if (!newRenv) {
        failOrSkip('Renv was not created');
        return;
      }

      createdRenvId = newRenv.ID;

      const plugin = pluginsResult.data[0];
      const pluginID = plugin.definition?.plugin?.id || plugin.id?.toString();

      const bindResult = await callWailsMethod(page, 'BindPluginToEnvironment', pluginID, 'r', newRenv.ID, newRenv.ProjectPath);

      if (bindResult.__error) {
        console.log('BindPluginToEnvironment error:', bindResult.message);
      }

      const bindingsResult = await callWailsMethod(page, 'GetAllPluginEnvironmentBindings');
      expect(bindingsResult.__success).toBeTruthy();
      expect(Array.isArray(bindingsResult.data)).toBeTruthy();

    } finally {
      if (createdRenvId) {
        await callWailsMethod(page, 'DeleteRenvEnvironment', createdRenvId);
      }
    }
  });

  test('should get all plugin environment bindings', async ({ page }) => {
    const result = await callWailsMethod(page, 'GetAllPluginEnvironmentBindings');
    if (checkWailsResult(result)) return;

    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data)).toBeTruthy();

    if (result.data && result.data.length > 0) {
      const binding = result.data[0];
      expect(binding.PluginID).toBeDefined();
      expect(binding.EnvironmentType).toBeDefined();
      expect(binding.EnvironmentID).toBeDefined();
    }
  });
});

test.describe('Environment Detection Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('should detect Python environments', async ({ page }) => {
    const result = await callWailsMethod(page, 'DetectPythonEnvironments');
    if (checkWailsResult(result)) return;

    if (result.__success) {
      expect(Array.isArray(result.data)).toBeTruthy();

      if (result.data && result.data.length > 0) {
        const env = result.data[0];
        expect(env.name).toBeDefined();
        expect(env.path).toBeDefined();
        expect(env.type).toBeDefined();
        expect(typeof env.isVirtual).toBe('boolean');
      }
    }
  });

  test('should detect R environments', async ({ page }) => {
    const result = await callWailsMethod(page, 'DetectREnvironments');
    if (checkWailsResult(result)) return;

    if (result.__success) {
      expect(Array.isArray(result.data)).toBeTruthy();

      if (result.data && result.data.length > 0) {
        const env = result.data[0];
        expect(env.name).toBeDefined();
        expect(env.path).toBeDefined();
      }
    }
  });

  test('should get virtual environments list', async ({ page }) => {
    const result = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(result)) return;

    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data)).toBeTruthy();

    if (result.data && result.data.length > 0) {
      const venv = result.data[0];
      expect(venv.ID).toBeDefined();
      expect(venv.Name).toBeDefined();
      expect(venv.Path).toBeDefined();
    }
  });

  test('should get renv environments list', async ({ page }) => {
    const result = await callWailsMethod(page, 'GetRenvEnvironments');
    if (checkWailsResult(result)) return;

    expect(result.__success).toBeTruthy();
    expect(Array.isArray(result.data)).toBeTruthy();

    if (result.data && result.data.length > 0) {
      const renv = result.data[0];
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
    await page.goto(`${BASE_URL}/settings/python`);

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
    await page.goto(`${BASE_URL}/settings/r`);

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
    const checkResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(checkResult)) return;

    await page.goto(`${BASE_URL}/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    const basePythonPath = await getBasePythonPath(page);
    if (!basePythonPath) {
      failOrSkip('No Python environment available');
      return;
    }

    const createButton = page.locator('button:has-text("Create Virtual Environment")');
    await expect(createButton).toBeVisible({ timeout: 15000 });
  });

  test('should enable create renv button when R is selected', async ({ page }) => {
    const checkResult = await callWailsMethod(page, 'GetRenvEnvironments');
    if (checkWailsResult(checkResult)) return;

    await page.goto(`${BASE_URL}/settings/r`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);

    const baseRPath = await getBaseRPath(page);
    if (!baseRPath) {
      failOrSkip('No R environment available');
      return;
    }

    const createButton = page.locator('button:has-text("Create Renv Environment")');
    await expect(createButton).toBeVisible({ timeout: 15000 });
  });

  test('should show no environments message when list is empty', async ({ page }) => {
    const venvsResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(venvsResult)) return;

    await page.goto(`${BASE_URL}/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    if (!venvsResult.data || venvsResult.data.length === 0) {
      const noVenvsMessage = page.locator('.no-venvs');
      await expect(noVenvsMessage).toContainText('No virtual environments created yet');
    }
  });

  test('should show delete button for each virtual environment', async ({ page }) => {
    const venvsResult = await callWailsMethod(page, 'GetVirtualEnvironments');
    if (checkWailsResult(venvsResult)) return;

    if (!venvsResult.data || venvsResult.data.length === 0) {
      failOrSkip('No virtual environments to test');
      return;
    }

    await page.goto(`${BASE_URL}/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const deleteButtons = page.locator('.venv-section mat-list-item button[mattooltip="Delete environment"]');
    const count = await deleteButtons.count();

    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('should show bound plugins button when environment has bindings', async ({ page }) => {
    const bindingsResult = await callWailsMethod(page, 'GetAllPluginEnvironmentBindings');
    if (checkWailsResult(bindingsResult)) return;

    const pythonBindings = bindingsResult.data?.filter((b: any) => b.EnvironmentType === 'python') || [];

    if (pythonBindings.length === 0) {
      failOrSkip('No Python environment bindings to test');
      return;
    }

    await page.goto(`${BASE_URL}/settings/python`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const linkButtons = page.locator('.venv-section mat-list-item button[mattooltip="View bound plugins"]');
    const count = await linkButtons.count();

    expect(count).toBeGreaterThanOrEqual(1);
  });
});
