#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');

const OUTPUT_FILE = path.join(__dirname, '..', 'resources', 'licenses', 'go-licenses.json');

function detectLicenseType(licenseText) {
  const text = licenseText.toUpperCase();

  if (text.includes('MIT LICENSE') || (text.includes('MIT') && text.includes('PERMISSION IS HEREBY GRANTED'))) return 'MIT';
  if (text.includes('APACHE LICENSE') && text.includes('VERSION 2.0')) return 'Apache-2.0';
  if (text.includes('APACHE LICENSE, VERSION 2.0')) return 'Apache-2.0';

  if (text.includes('BSD 3-CLAUSE')) return 'BSD-3-Clause';
  if (text.includes('BSD 2-CLAUSE')) return 'BSD-2-Clause';

  if (text.includes('REDISTRIBUTION') && text.includes('THIS SOFTWARE IS PROVIDED')) {
    if (text.includes('NEITHER THE NAME')) return 'BSD-3-Clause';
    if (text.includes('REDISTRIBUTIONS OF SOURCE CODE') && text.includes('REDISTRIBUTIONS IN BINARY FORM')) return 'BSD-2-Clause';
  }

  if (text.includes('MOZILLA PUBLIC LICENSE')) return 'MPL-2.0';
  if (text.includes('GNU GENERAL PUBLIC LICENSE') && text.includes('VERSION 3')) return 'GPL-3.0';
  if (text.includes('GNU LESSER GENERAL PUBLIC LICENSE')) return 'LGPL';
  if (text.includes('ISC LICENSE') || (text.includes('ISC') && text.includes('PERMISSION TO USE'))) return 'ISC';
  if (text.includes('UNLICENSE') || text.includes('THIS IS FREE AND UNENCUMBERED SOFTWARE')) return 'Unlicense';
  if (text.includes('CREATIVE COMMONS')) return 'CC';

  return 'See module repository';
}

function fetchUrl(url) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith('https') ? https : http;
    const req = protocol.get(url, {
      headers: { 'User-Agent': 'cauldron-go-license-fetcher' },
      timeout: 5000
    }, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        return fetchUrl(res.headers.location).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`HTTP ${res.statusCode}`));
      }

      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(data));
    });
    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Timeout'));
    });
  });
}

async function resolveGoImportRedirect(modulePath) {
  try {
    const url = `https://${modulePath}?go-get=1`;
    const html = await fetchUrl(url);

    const match = html.match(/<meta\s+name="go-import"\s+content="([^"]+)"/i);
    if (match && match[1]) {
      const parts = match[1].split(/\s+/);
      if (parts.length >= 3) {
        const repoUrl = parts[2];

        const githubMatch = repoUrl.match(/github\.com\/([^\/]+\/[^\/\.]+)/);
        if (githubMatch) {
          return githubMatch[0];
        }

        const gitlabMatch = repoUrl.match(/gitlab\.com\/([^\/]+\/[^\/\.]+)/);
        if (gitlabMatch) {
          return gitlabMatch[0];
        }

        const bitbucketMatch = repoUrl.match(/bitbucket\.org\/([^\/]+\/[^\/\.]+)/);
        if (bitbucketMatch) {
          return bitbucketMatch[0];
        }
      }
    }
  } catch (e) {
    return null;
  }
  return null;
}

async function fetchLicenseFromWeb(repoPath) {
  const licenseFiles = ['LICENSE', 'License', 'LICENSE.txt', 'License.txt', 'LICENSE.md', 'License.md', 'COPYING', 'COPYRIGHT', 'LICENCE', 'Licence'];
  const branches = ['main', 'master'];

  if (!repoPath.startsWith('github.com/') && !repoPath.startsWith('gitlab.com/') && !repoPath.startsWith('bitbucket.org/')) {
    const redirectedRepo = await resolveGoImportRedirect(repoPath);
    if (redirectedRepo) {
      repoPath = redirectedRepo;
    }
  }

  if (repoPath.startsWith('github.com/')) {
    const [, owner, repo] = repoPath.split('/');

    for (const branch of branches) {
      for (const fileName of licenseFiles) {
        const url = `https://raw.githubusercontent.com/${owner}/${repo}/${branch}/${fileName}`;
        try {
          const content = await fetchUrl(url);
          if (content && content.length > 50 && content.length < 100000) {
            const detected = detectLicenseType(content);
            if (detected !== 'See module repository') {
              return detected;
            }
          }
        } catch (e) {
          continue;
        }
      }
    }
  } else if (repoPath.startsWith('gitlab.com/')) {
    const pathParts = repoPath.replace('gitlab.com/', '').split('/');
    const repoSlug = pathParts.slice(0, 2).join('/');

    for (const branch of branches) {
      for (const fileName of licenseFiles) {
        const url = `https://gitlab.com/${repoSlug}/-/raw/${branch}/${fileName}`;
        try {
          const content = await fetchUrl(url);
          if (content && content.length > 50 && content.length < 100000) {
            const detected = detectLicenseType(content);
            if (detected !== 'See module repository') {
              return detected;
            }
          }
        } catch (e) {
          continue;
        }
      }
    }
  } else if (repoPath.startsWith('bitbucket.org/')) {
    const [, owner, repo] = repoPath.split('/');

    for (const branch of branches) {
      for (const fileName of licenseFiles) {
        const url = `https://bitbucket.org/${owner}/${repo}/raw/${branch}/${fileName}`;
        try {
          const content = await fetchUrl(url);
          if (content && content.length > 50 && content.length < 100000) {
            const detected = detectLicenseType(content);
            if (detected !== 'See module repository') {
              return detected;
            }
          }
        } catch (e) {
          continue;
        }
      }
    }
  }

  return null;
}

function getRepositoryRoot(modulePath) {
  const parts = modulePath.split('/');

  if (parts[0] === 'github.com' || parts[0] === 'gitlab.com' || parts[0] === 'bitbucket.org') {
    return parts.slice(0, 3).join('/');
  }

  if (parts[0] === 'golang.org' && parts[1] === 'x') {
    return `github.com/golang/${parts[2]}`;
  }

  if (parts[0] === 'google.golang.org') {
    if (parts[1] === 'protobuf') {
      return 'github.com/protocolbuffers/protobuf-go';
    }
    return `github.com/golang/${parts[1]}`;
  }

  if (parts[0] === 'gopkg.in') {
    const pkgName = parts[1];
    if (pkgName.includes('.v')) {
      const name = pkgName.split('.v')[0];
      if (name.includes('/')) {
        const [owner, repo] = name.split('/');
        return `github.com/${owner}/${repo}`;
      }
      return `github.com/go-${name}/${name}`;
    }
    if (pkgName.includes('/')) {
      const [owner, repo] = pkgName.split('/');
      return `github.com/${owner}/${repo}`;
    }
    return `github.com/go-${pkgName}/${pkgName}`;
  }

  if (parts[0] === 'gorm.io') {
    if (parts.length > 2) {
      return `github.com/go-gorm/${parts[2]}`;
    }
    return `github.com/go-gorm/${parts[1]}`;
  }

  if (parts[0].includes('.')) {
    return parts.slice(0, 2).join('/');
  }

  return modulePath;
}

function findLicenseInPath(searchPath) {
  const licenseFiles = [
    'LICENSE', 'LICENSE.txt', 'LICENSE.md', 'LICENSE.MIT', 'LICENSE.APACHE',
    'COPYING', 'COPYING.txt', 'COPYRIGHT', 'COPYRIGHT.txt',
    'license', 'license.txt', 'license.md'
  ];

  for (const fileName of licenseFiles) {
    const licensePath = path.join(searchPath, fileName);
    if (fs.existsSync(licensePath)) {
      try {
        const stats = fs.statSync(licensePath);
        if (stats.size > 100000) continue;

        const licenseText = fs.readFileSync(licensePath, 'utf8');
        const detected = detectLicenseType(licenseText);
        if (detected !== 'See module repository') {
          return detected;
        }
      } catch (error) {
        continue;
      }
    }
  }

  return null;
}

async function getModuleLicense(mod, goModCache) {
  const repoRoot = getRepositoryRoot(mod.Path);

  const webLicense = await fetchLicenseFromWeb(repoRoot);
  if (webLicense) return webLicense;

  if (mod.Dir && fs.existsSync(mod.Dir)) {
    const license = findLicenseInPath(mod.Dir);
    if (license) return license;
  }

  if (mod.Version) {
    const modCachePath = path.join(goModCache, mod.Path + '@' + mod.Version);
    if (fs.existsSync(modCachePath)) {
      const license = findLicenseInPath(modCachePath);
      if (license) return license;
    }

    if (repoRoot !== mod.Path) {
      const repoCachePath = path.join(goModCache, repoRoot + '@' + mod.Version);
      if (fs.existsSync(repoCachePath)) {
        const license = findLicenseInPath(repoCachePath);
        if (license) return license;
      }
    }
  }

  const allVersions = fs.existsSync(goModCache) ? fs.readdirSync(goModCache) : [];
  const repoVersions = allVersions.filter(dir => dir.startsWith(repoRoot + '@'));

  for (const versionDir of repoVersions) {
    const versionPath = path.join(goModCache, versionDir);
    const license = findLicenseInPath(versionPath);
    if (license) return license;
  }

  return 'See module repository';
}

async function main() {
  try {
    const outputDir = path.dirname(OUTPUT_FILE);
    if (!fs.existsSync(outputDir)) {
      fs.mkdirSync(outputDir, { recursive: true });
    }

    console.log('Generating Go module license information...');

    const output = execSync('go list -m -json all', { encoding: 'utf8' });

    const modules = output
      .trim()
      .split('\n}\n')
      .filter(line => line.trim())
      .map(line => {
        try {
          const json = line.endsWith('}') ? line : line + '}';
          return JSON.parse(json);
        } catch (e) {
          return null;
        }
      })
      .filter(mod => mod && !mod.Replace);

    const goModCache = execSync('go env GOMODCACHE', { encoding: 'utf8' }).trim();

    console.log(`Processing ${modules.length} modules...`);

    const licenses = await Promise.all(modules.map(async (mod, index) => {
      if (index % 20 === 0) {
        process.stdout.write(`\rProcessed ${index}/${modules.length} modules...`);
      }

      const license = await getModuleLicense(mod, goModCache);
      const repoRoot = getRepositoryRoot(mod.Path);

      let repository = null;
      if (mod.Path.startsWith('github.com/') || mod.Path.startsWith('gitlab.com/') || mod.Path.startsWith('bitbucket.org/')) {
        repository = `https://${repoRoot}`;
      }

      return {
        name: mod.Path,
        version: mod.Version || 'unknown',
        license: license,
        repository: repository
      };
    }));

    console.log(`\rProcessed ${modules.length}/${modules.length} modules.`);

    const uniqueLicenses = Array.from(
      new Map(licenses.map(item => [item.name, item])).values()
    );

    fs.writeFileSync(OUTPUT_FILE, JSON.stringify(uniqueLicenses, null, 2));
    console.log(`Go licenses generated at ${OUTPUT_FILE}`);
  } catch (error) {
    console.error('Error generating Go licenses:', error.message);
    fs.writeFileSync(OUTPUT_FILE, '[]');
    process.exit(0);
  }
}

main();
