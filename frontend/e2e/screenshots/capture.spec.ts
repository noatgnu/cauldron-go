import { test } from '@playwright/test';
import { execFile } from 'child_process';
import { promisify } from 'util';
import * as fs from 'fs';
import * as path from 'path';
import { navigate, waitForElement, waitForWindow } from '../helpers/mcp-client';

const execFileAsync = promisify(execFile);

interface ScreenshotTarget {
  route: string;
  name: string;
  selector: string;
}

const OUTPUT_DIR = path.resolve(__dirname, '../../../docs/images');

// Mirrors the app's own quick-nav page list plus Plugin Registry.
const targets: ScreenshotTarget[] = [
  { route: '/home', name: 'home', selector: '.home-container' },
  { route: '/jobs', name: 'jobs', selector: '.jobs-container' },
  { route: '/plugin-registry', name: 'plugin-registry', selector: '.registry-container' },
  { route: '/settings/general', name: 'settings-general', selector: '.general-settings' },
  { route: '/settings/appearance', name: 'settings-appearance', selector: '.appearance-settings' },
  { route: '/gel-analysis', name: 'gel-analysis', selector: '.gel-analysis-container' },
  { route: '/table-browser', name: 'table-browser', selector: '.table-browser-container' },
  { route: '/about', name: 'about', selector: '.about-page' },
];

async function captureDisplay(outPath: string): Promise<void> {
  const display = process.env['SCREENSHOT_DISPLAY'] || process.env['DISPLAY'] || ':99';
  await execFileAsync('import', ['-display', display, '-window', 'root', outPath]);
}

test.beforeAll(async () => {
  const ready = await waitForWindow();
  if (!ready) {
    throw new Error('Cauldron app window did not become visible in time');
  }
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
});

for (const target of targets) {
  test(`capture ${target.name}`, async () => {
    await navigate(target.route);
    const found = await waitForElement(target.selector);
    if (!found) {
      throw new Error(`Timed out waiting for "${target.selector}" on route ${target.route}`);
    }
    // Let any route-transition animation settle before capturing.
    await new Promise(resolve => setTimeout(resolve, 500));
    await captureDisplay(path.join(OUTPUT_DIR, `${target.name}.png`));
  });
}
