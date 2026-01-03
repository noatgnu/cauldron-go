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

  const output = execSync('npm list --json --all', {
    cwd: frontendDir,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'ignore']
  });

  const data = JSON.parse(output);
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
  process.exit(0);
}
