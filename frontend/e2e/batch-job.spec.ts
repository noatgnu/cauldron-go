import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as mcp from './helpers/mcp-client';

const PLUGIN_ID = 'e2e-batch-lifecycle-plugin';

const PLUGIN_YAML = `plugin:
  id: "${PLUGIN_ID}"
  name: "E2E Batch Lifecycle Plugin"
  description: "Throwaway plugin used only by the batch job lifecycle E2E test"
  version: "1.0.0"
  category: "utilities"

runtime:
  environments:
    - python
  entrypoint: "main.py"

inputs:
  - name: "input_file"
    label: "Input File"
    type: "file"
    required: false
    accept: ".csv,.tsv,.txt"

  - name: "label"
    label: "Label"
    type: "text"
    required: true
    default: "sample"

  - name: "threshold"
    label: "Threshold"
    type: "number"
    required: false
    default: 1
`;

let pluginNumericId: number;
let pluginDir = '';
let batchId = '';

async function waitForRoute(fragment: string, timeoutMs = 15000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const url = await mcp.getUrl();
    if (typeof url === 'string' && url.includes(fragment)) return true;
    await new Promise(r => setTimeout(r, 300));
  }
  return false;
}

async function ensurePythonConfigured(): Promise<boolean> {
  const settings = await mcp.callBoundMethod('main.App.GetSettings');
  if (settings?.pythonPath) return true;

  const detected = await mcp.callBoundMethod('main.App.DetectPythonPath');
  const pythonPath = Array.isArray(detected) ? detected[0] : detected;
  if (!pythonPath) return false;

  await mcp.callBoundMethod('main.App.SetSetting', 'pythonPath', pythonPath);
  return true;
}

test.describe.serial('Batch Job UI lifecycle', () => {
  test.beforeAll(async () => {
    const ready = await mcp.waitForWindow();
    expect(ready).toBe(true);

    const pythonReady = await ensurePythonConfigured();
    if (!pythonReady) {
      test.skip(true, 'python3 not available on this machine, skipping batch lifecycle E2E test');
      return;
    }

    const plugins = await mcp.callBoundMethod('main.App.GetPluginsV2');
    const anyPlugin = (plugins || []).find((p: any) => p && p.folderPath);
    if (!anyPlugin) throw new Error('No installed plugin found to determine the plugins root directory');

    const pluginsRoot = path.dirname(anyPlugin.folderPath);
    pluginDir = path.join(pluginsRoot, PLUGIN_ID);

    fs.rmSync(pluginDir, { recursive: true, force: true });
    fs.mkdirSync(pluginDir, { recursive: true });
    fs.writeFileSync(path.join(pluginDir, 'plugin.yaml'), PLUGIN_YAML);
    fs.writeFileSync(path.join(pluginDir, 'main.py'), 'import sys\nsys.exit(0)\n');

    await mcp.callBoundMethod('main.App.ReloadPluginsV2');

    const reloaded = await mcp.callBoundMethod('main.App.GetPluginsV2');
    const plugin = (reloaded || []).find((p: any) => p?.definition?.plugin?.id === PLUGIN_ID);
    if (!plugin) throw new Error('Throwaway plugin did not load after ReloadPluginsV2');
    pluginNumericId = plugin.id;
  });

  test.afterAll(async () => {
    if (!pluginDir) return;
    fs.rmSync(pluginDir, { recursive: true, force: true });
    await mcp.callBoundMethod('main.App.ReloadPluginsV2');
  });

  test('navigates to the plugin execute page and opens the Batch tab', async () => {
    await mcp.navigate(`/plugin/${pluginNumericId}`);
    expect(await waitForRoute(`/plugin/${pluginNumericId}`)).toBe(true);
    expect(await mcp.waitForElement('.plugin-execute-container')).toBe(true);

    expect(await mcp.clickTabByLabel('Batch')).toBe(true);
    expect(await mcp.waitForElement('.batch-builder')).toBe(true);
  });

  test('builds three job rows and bulk-applies a shared field', async () => {
    expect(await mcp.waitForElement('.add-empty-job-btn')).toBe(true);
    await mcp.click('.add-empty-job-btn');
    await mcp.click('.add-empty-job-btn');

    const rows = await mcp.domQuery('.rows-table tbody tr, .rows-table tr[mat-row]', 10);
    expect(rows.count).toBe(3);

    expect(await mcp.clickAndWaitFor('.bulk-field-select', 'mat-option')).toBe(true);
    expect(await mcp.clickOptionByText('Label')).toBe(true);

    expect(await mcp.waitForElement('.bulk-field-value .form-field[data-field="label"] input')).toBe(true);
    await mcp.fill('.bulk-field-value .form-field[data-field="label"] input', 'e2e-bulk-label');

    await mcp.click('.apply-bulk-btn');
  });

  test('edits one row through the dynamic-form dialog', async () => {
    expect(await mcp.clickAndWaitFor('.rows-table .edit-row-btn', '.mat-mdc-dialog-container')).toBe(true);
    expect(await mcp.waitForElement('.mat-mdc-dialog-container .form-field[data-field="threshold"] input')).toBe(true);

    await mcp.fill('.mat-mdc-dialog-container .form-field[data-field="threshold"] input', '42');

    expect(await mcp.clickAndWaitForGone('.mat-mdc-dialog-container .save-btn', '.mat-mdc-dialog-container')).toBe(true);
  });

  test('creates the batch and tracks jobs to completion', async () => {
    await mcp.click('.create-batch-btn');
    expect(await waitForRoute('/job-batch/')).toBe(true);

    const url = await mcp.getUrl();
    const match = /\/job-batch\/([^/?#]+)/.exec(url);
    if (!match) throw new Error(`Could not extract batch id from URL: ${url}`);
    batchId = match[1];

    expect(await mcp.waitForElement('.jobs-table')).toBe(true);

    let finalStatus: any = null;
    const start = Date.now();
    while (Date.now() - start < 20000) {
      const status = await mcp.callBoundMethod('main.App.GetJobBatchStatus', batchId);
      if (status.totalJobs === 3 && status.completedCount + status.failedCount === 3) {
        finalStatus = status;
        break;
      }
      await new Promise(r => setTimeout(r, 500));
    }

    expect(finalStatus).not.toBeNull();
    expect(finalStatus.totalJobs).toBe(3);
    expect(finalStatus.completedCount).toBe(3);
    expect(finalStatus.failedCount).toBe(0);
  });

  test('lists the batch on the Job Batches page and navigates back to its detail', async () => {
    await mcp.navigate('/job-batches');
    expect(await waitForRoute('/job-batches')).toBe(true);
    expect(await mcp.waitForElement(`[data-batch-id="${batchId}"]`)).toBe(true);

    await mcp.click(`[data-batch-id="${batchId}"]`);
    expect(await waitForRoute(`/job-batch/${batchId}`)).toBe(true);
    expect(await mcp.waitForElement('.jobs-table')).toBe(true);
  });

  test('deletes the batch and confirms cascading removal', async () => {
    expect(await mcp.clickAndWaitFor('.delete-batch-btn', '.confirm-dialog-confirm-btn')).toBe(true);
    await mcp.click('.confirm-dialog-confirm-btn');

    expect(await waitForRoute('/job-batches')).toBe(true);

    const remaining = await mcp.callBoundMethod('main.App.GetAllJobBatches', 10, 0);
    const stillThere = (remaining || []).some((b: any) => b.id === batchId);
    expect(stillThere).toBe(false);
  });
});
