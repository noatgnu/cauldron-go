#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const OUTPUT_FILE = path.join(__dirname, '..', 'resources', 'licenses', 'npm-licenses.json');

function getPackageLicense(packagePath) {
  try {
    const packageJsonPath = path.join(packagePath, 'package.json');
    if (fs.existsSync(packageJsonPath)) {
      const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
      return {
        license: packageJson.license || packageJson.licenses?.[0]?.type || 'Unknown',
        repository: packageJson.repository?.url || packageJson.repository || null
      };
    }
  } catch (error) {
    return { license: 'Unknown', repository: null };
  }
  return { license: 'Unknown', repository: null };
}

try {
  const outputDir = path.dirname(OUTPUT_FILE);
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  console.log('Generating NPM package license information...');

  const frontendDir = path.join(__dirname, '..', 'frontend');
  const nodeModulesDir = path.join(frontendDir, 'node_modules');

  if (!fs.existsSync(nodeModulesDir)) {
    console.error('Error: node_modules not found. Run "npm install" in frontend/ first.');
    fs.writeFileSync(OUTPUT_FILE, '[]');
    process.exit(1);
  }

  let output;
  try {
    output = execSync('npm list --json --depth=0', {
      cwd: frontendDir,
      encoding: 'utf8',
      maxBuffer: 50 * 1024 * 1024
    });
  } catch (execError) {
    output = execError.stdout || '{}';
  }

  let data;
  try {
    data = JSON.parse(output);
  } catch (parseError) {
    console.error('Failed to parse npm list output, using empty list');
    data = { dependencies: {} };
  }

  const dependencies = data.dependencies || {};

  const licenses = Object.entries(dependencies).map(([name, info]) => {
    const packagePath = path.join(nodeModulesDir, name);
    const licenseInfo = getPackageLicense(packagePath);

    return {
      name,
      version: info.version || 'unknown',
      license: licenseInfo.license,
      repository: licenseInfo.repository
    };
  });

  const uniqueLicenses = Array.from(
    new Map(licenses.map(item => [item.name, item])).values()
  ).sort((a, b) => a.name.localeCompare(b.name));

  fs.writeFileSync(OUTPUT_FILE, JSON.stringify(uniqueLicenses, null, 2));
  console.log(`NPM licenses generated at ${OUTPUT_FILE}`);
} catch (error) {
  console.error('Error generating NPM licenses:', error.message);
  fs.writeFileSync(OUTPUT_FILE, '[]');
}
