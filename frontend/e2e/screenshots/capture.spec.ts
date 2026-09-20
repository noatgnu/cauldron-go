import { test } from '@playwright/test';
import { execFile } from 'child_process';
import { promisify } from 'util';
import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import { navigate, waitForElement, waitForWindow, getUrl, callBoundMethod, click } from '../helpers/mcp-client';

const execFileAsync = promisify(execFile);

interface ScreenshotTarget {
  route: string;
  name: string;
  selector: string;
}

const OUTPUT_DIR = path.resolve(__dirname, '../../../docs/images');

const targets: ScreenshotTarget[] = [
  { route: '/home', name: 'home', selector: '.home-container.ready' },
  { route: '/jobs', name: 'jobs', selector: '.jobs-container' },
  { route: '/plugin-registry', name: 'plugin-registry', selector: '.registry-container' },
  { route: '/settings/general', name: 'settings-general', selector: '.general-settings' },
  { route: '/settings/appearance', name: 'settings-appearance', selector: '.appearance-settings' },
  { route: '/gel-analysis', name: 'gel-analysis', selector: '.gel-analysis-container' },
  { route: '/table-browser', name: 'table-browser', selector: '.table-browser-container' },
  { route: '/about', name: 'about', selector: '.about-page' },
];

function getDisplay(): string {
  return process.env['SCREENSHOT_DISPLAY'] || process.env['DISPLAY'] || ':99';
}

let cachedWindowId: string | null = null;

async function getCauldronWindowId(): Promise<string> {
  if (cachedWindowId) return cachedWindowId;
  const { stdout } = await execFileAsync('xwininfo', ['-root', '-tree', '-display', getDisplay()]);
  const match = stdout.match(/^\s*(0x[0-9a-f]+)\s+"Cauldron":/m);
  if (!match) {
    throw new Error('Could not find the Cauldron window via xwininfo');
  }
  cachedWindowId = match[1];
  return cachedWindowId;
}

async function captureDisplay(outPath: string): Promise<Buffer> {
  const windowId = await getCauldronWindowId();
  await execFileAsync('import', ['-display', getDisplay(), '-window', windowId, outPath]);
  return fs.readFileSync(outPath);
}

function hash(buf: Buffer): string {
  return crypto.createHash('sha256').update(buf).digest('hex');
}

async function waitForRoute(route: string, timeoutMs = 15000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const url = await getUrl();
    if (typeof url === 'string' && url.includes(route)) {
      return true;
    }
    await new Promise(resolve => setTimeout(resolve, 300));
  }
  return false;
}

test.beforeAll(async () => {
  const ready = await waitForWindow();
  if (!ready) {
    throw new Error('Cauldron app window did not become visible in time');
  }
  await callBoundMethod('main.App.SetSetting', 'autoCheckForUpdates', false);

  const dialogOpen = await waitForElement('.mat-mdc-dialog-container', 5000);
  if (dialogOpen) {
    await click('.mat-mdc-dialog-actions button:first-child');
  }

  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
});

let previousHash: string | null = null;

for (const target of targets) {
  test(`capture ${target.name}`, async () => {
    await navigate(target.route);

    const routeConfirmed = await waitForRoute(target.route);
    if (!routeConfirmed) {
      throw new Error(`Timed out waiting for the URL to reflect route ${target.route}`);
    }

    const found = await waitForElement(target.selector);
    if (!found) {
      throw new Error(`Timed out waiting for "${target.selector}" on route ${target.route}`);
    }

    const outPath = path.join(OUTPUT_DIR, `${target.name}.png`);

    let currentHash = '';
    let lastHash: string | null = null;
    const maxAttempts = 10;
    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      await new Promise(resolve => setTimeout(resolve, 1000));
      const buf = await captureDisplay(outPath);
      currentHash = hash(buf);
      const stable = currentHash === lastHash;
      const distinct = currentHash !== previousHash;
      lastHash = currentHash;
      if (stable && distinct) break;
      if (attempt === maxAttempts) {
        throw new Error(`Screenshot for ${target.name} never stabilized to a distinct frame after ${maxAttempts} attempts`);
      }
    }

    previousHash = currentHash;
  });
}
