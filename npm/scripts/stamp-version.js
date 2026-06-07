/**
 * CI helper: stamp the release version into all npm package.json files.
 * Replaces the placeholder "0.0.0" with the actual release version.
 *
 * Usage: node npm/scripts/stamp-version.js <version>
 *   e.g. node npm/scripts/stamp-version.js 1.40.9
 */
'use strict';
const fs   = require('fs');
const path = require('path');

const version = process.argv[2];
if (!version || !/^\d+\.\d+\.\d+/.test(version)) {
  console.error('stamp-version: usage: node stamp-version.js <version>  (e.g. 1.40.9)');
  process.exit(1);
}

const PACKAGE_DIRS = [
  'wrapper',
  'darwin-arm64',
  'darwin-x64',
  'linux-x64',
  'linux-arm64',
  'win32-x64',
];

const repoRoot = path.resolve(__dirname, '..', '..');

for (const dir of PACKAGE_DIRS) {
  const pkgPath = path.join(repoRoot, 'npm', dir, 'package.json');
  const raw     = fs.readFileSync(pkgPath, 'utf8');
  const pkg     = JSON.parse(raw);

  pkg.version = version;

  // Also update the version pins in optionalDependencies (wrapper only).
  if (pkg.optionalDependencies) {
    for (const dep of Object.keys(pkg.optionalDependencies)) {
      pkg.optionalDependencies[dep] = version;
    }
  }

  // Preserve trailing newline and 2-space indent to match the existing style.
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
  console.log(`  stamped ${dir}/package.json → ${version}`);
}

console.log(`\nAll package.json files stamped with version ${version}.`);
