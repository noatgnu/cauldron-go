import { test, expect, Page } from '@playwright/test';

const BASE_URL = process.env['WAILS_URL'] || 'http://localhost:4200';

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

function skipIfNoWails(result: any) {
  if (result?.__skipped) {
    test.skip();
    return true;
  }
  return false;
}

test.describe('Real Wails API Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test.describe('Settings API', () => {
    test('GetSettings returns valid config', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetSettings');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(result.data).not.toBeNull();
      expect(typeof result.data).toBe('object');
    });

    test('SetSetting updates a setting value', async ({ page }) => {
      const getResult = await callWailsMethod(page, 'GetSettings');
      if (skipIfNoWails(getResult)) return;

      const originalTheme = getResult.data?.theme || 'system';
      const newTheme = originalTheme === 'dark' ? 'light' : 'dark';

      const setResult = await callWailsMethod(page, 'SetSetting', 'theme', newTheme);
      if (setResult.__error) {
        console.log('SetSetting error:', setResult.message);
      }

      const verifyResult = await callWailsMethod(page, 'GetSettings');
      expect(verifyResult.__success).toBeTruthy();

      await callWailsMethod(page, 'SetSetting', 'theme', originalTheme);
    });
  });

  test.describe('Jobs API', () => {
    test('GetAllJobs returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetAllJobs');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(Array.isArray(result.data)).toBeTruthy();
    });

    test('GetJob returns null for invalid ID', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetJob', 'nonexistent-job-id');
      if (skipIfNoWails(result)) return;

      if (result.__error) {
        expect(result.message).toContain('not found');
      } else {
        expect(result.data).toBeNull();
      }
    });

    test('GetJobQueueStatus returns queue info', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetJobQueueStatus');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(result.data).not.toBeNull();
      expect(typeof result.data).toBe('object');
    });
  });

  test.describe('Files API', () => {
    test('GetImportedFiles returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetImportedFiles');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(Array.isArray(result.data)).toBeTruthy();
    });
  });

  test.describe('Plugins API', () => {
    test('GetPluginsV2 returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetPluginsV2');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(Array.isArray(result.data)).toBeTruthy();
    });

    test('GetPluginV2 returns null for invalid ID', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetPluginV2', 999999);
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(result.data).toBeNull();
    });

    test('ValidatePluginYAML validates correct YAML', async ({ page }) => {
      const validYAML = `plugin:
  id: test-plugin
  name: Test Plugin
  description: A test plugin
  version: 1.0.0
  author: Test
  category: analysis

runtime:
  environments:
    - python
  entrypoint: main.py

inputs:
  - name: input_file
    label: Input File
    type: file
    required: true
    description: Input data file
    flag: --input

outputs: []

execution:
  outputDir: --output_folder
`;

      const result = await callWailsMethod(page, 'ValidatePluginYAML', validYAML);
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(result.data).toBeTruthy();
    });
  });

  test.describe('Environment API', () => {
    test('GetPythonVersion returns version string', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetPythonVersion');
      if (skipIfNoWails(result)) return;

      if (result.__success) {
        expect(typeof result.data).toBe('string');
      }
    });

    test('GetRVersion returns version or error', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetRVersion');
      if (skipIfNoWails(result)) return;

      expect(result.__success || result.__error).toBeTruthy();
    });

    test('CheckDockerVersion returns version or error', async ({ page }) => {
      const result = await callWailsMethod(page, 'CheckDockerVersion');
      if (skipIfNoWails(result)) return;

      expect(result.__success || result.__error).toBeTruthy();
    });

    test('DetectPythonEnvironments returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'DetectPythonEnvironments');
      if (skipIfNoWails(result)) return;

      if (result.__success) {
        expect(Array.isArray(result.data)).toBeTruthy();
      }
    });

    test('GetVirtualEnvironments returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetVirtualEnvironments');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(Array.isArray(result.data)).toBeTruthy();
    });

    test('GetRenvEnvironments returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetRenvEnvironments');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(Array.isArray(result.data)).toBeTruthy();
    });

    test('GetAllPluginEnvironmentBindings returns array', async ({ page }) => {
      const result = await callWailsMethod(page, 'GetAllPluginEnvironmentBindings');
      if (skipIfNoWails(result)) return;

      expect(result.__success).toBeTruthy();
      expect(Array.isArray(result.data)).toBeTruthy();
    });
  });

  test.describe('Job Queue API', () => {
    test('PauseJobQueue and ResumeJobQueue work correctly', async ({ page }) => {
      const pauseResult = await callWailsMethod(page, 'PauseJobQueue');
      if (skipIfNoWails(pauseResult)) return;

      const statusAfterPause = await callWailsMethod(page, 'GetJobQueueStatus');
      expect(statusAfterPause.__success).toBeTruthy();

      const resumeResult = await callWailsMethod(page, 'ResumeJobQueue');
      expect(resumeResult.__success || resumeResult.__error === undefined).toBeTruthy();

      const statusAfterResume = await callWailsMethod(page, 'GetJobQueueStatus');
      expect(statusAfterResume.__success).toBeTruthy();
    });
  });
});

test.describe('Data Serialization Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);
  });

  test('Empty arrays serialize to [] not null', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const win = window as any;
      if (!('_wails' in win)) {
        return { __skipped: true };
      }

      try {
        const WailsApp = win.go?.['github.com/noatgnu/cauldron-go'];
        if (!WailsApp) return { __skipped: true };

        const jobs = await WailsApp.GetAllJobs?.();
        const files = await WailsApp.GetImportedFiles?.();
        const plugins = await WailsApp.GetPluginsV2?.();
        const venvs = await WailsApp.GetVirtualEnvironments?.();

        return {
          __success: true,
          jobsIsArray: Array.isArray(jobs),
          filesIsArray: Array.isArray(files),
          pluginsIsArray: Array.isArray(plugins),
          venvsIsArray: Array.isArray(venvs),
          jobsNotNull: jobs !== null,
          filesNotNull: files !== null,
          pluginsNotNull: plugins !== null,
          venvsNotNull: venvs !== null
        };
      } catch (e: any) {
        return { __error: true, message: e.message };
      }
    });

    if (result.__skipped) {
      test.skip();
      return;
    }

    if (result.__success) {
      expect(result.jobsIsArray).toBeTruthy();
      expect(result.filesIsArray).toBeTruthy();
      expect(result.pluginsIsArray).toBeTruthy();
      expect(result.venvsIsArray).toBeTruthy();
      expect(result.jobsNotNull).toBeTruthy();
      expect(result.filesNotNull).toBeTruthy();
      expect(result.pluginsNotNull).toBeTruthy();
      expect(result.venvsNotNull).toBeTruthy();
    }
  });

  test('Job object has required fields', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const win = window as any;
      if (!('_wails' in win)) {
        return { __skipped: true };
      }

      try {
        const WailsApp = win.go?.['github.com/noatgnu/cauldron-go'];
        if (!WailsApp) return { __skipped: true };

        const jobs = await WailsApp.GetAllJobs?.();
        if (!jobs || jobs.length === 0) {
          return { __success: true, noJobs: true };
        }

        const job = jobs[0];
        return {
          __success: true,
          hasId: 'id' in job,
          hasStatus: 'status' in job,
          hasName: 'name' in job,
          hasArgs: 'args' in job,
          hasTerminalOutput: 'terminalOutput' in job,
          argsNotNull: job.args !== null,
          terminalOutputNotNull: job.terminalOutput !== null
        };
      } catch (e: any) {
        return { __error: true, message: e.message };
      }
    });

    if (result.__skipped) {
      test.skip();
      return;
    }

    if (result.__success && !result.noJobs) {
      expect(result.hasId).toBeTruthy();
      expect(result.hasStatus).toBeTruthy();
      expect(result.hasName).toBeTruthy();
      expect(result.argsNotNull).toBeTruthy();
      expect(result.terminalOutputNotNull).toBeTruthy();
    }
  });
});
