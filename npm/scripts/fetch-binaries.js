/**
 * CI helper: download platform binaries from the GitHub Release and place
 * each one in the correct npm/<platform>/bin/ directory.
 *
 * Requires env:
 *   VERSION       - release version without leading "v" (e.g. "1.40.9")
 *   GITHUB_TOKEN  - optional, for authenticated GitHub API requests
 *
 * Usage: node npm/scripts/fetch-binaries.js
 */
'use strict';
const https  = require('https');
const fs     = require('fs');
const path   = require('path');
const crypto = require('crypto');
const { execFileSync } = require('child_process');

const PLATFORMS = require('../wrapper/platforms.json');
const VERSION      = process.env.VERSION;
const GITHUB_TOKEN = process.env.GITHUB_TOKEN;

if (!VERSION) {
  console.error('fetch-binaries: VERSION environment variable is required');
  process.exit(1);
}

const BASE    = `https://github.com/stripe/stripe-cli/releases/download/v${VERSION}`;
const repoRoot = path.resolve(__dirname, '..', '..');
const tmpDir   = path.join(repoRoot, 'npm', '_tmp');
fs.mkdirSync(tmpDir, { recursive: true });

const downloadedChecksums = new Set();

async function main() {
  for (const [platformKey, config] of Object.entries(PLATFORMS)) {
    const archive      = config.archiveTemplate.replace('${VERSION}', VERSION);
    const checksumFile = config.checksums;
    const binaryName   = config.bin;

    console.log(`\n[${platformKey}] ${archive}`);

    const archivePath  = path.join(tmpDir, archive);
    const checksumPath = path.join(tmpDir, checksumFile);
    const outDir       = path.join(repoRoot, 'npm', platformKey, 'bin');
    fs.mkdirSync(outDir, { recursive: true });

    if (!downloadedChecksums.has(checksumFile)) {
      console.log(`  downloading ${checksumFile}...`);
      await download(`${BASE}/${checksumFile}`, checksumPath);
      downloadedChecksums.add(checksumFile);
    }

    console.log(`  downloading ${archive}...`);
    await download(`${BASE}/${archive}`, archivePath);

    console.log(`  verifying SHA256...`);
    verifySha256(archivePath, checksumPath, archive);

    console.log(`  extracting ${binaryName}...`);
    if (archive.endsWith('.zip')) {
      execFileSync('unzip', ['-q', '-o', archivePath, binaryName, '-d', outDir]);
    } else {
      execFileSync('tar', ['xzf', archivePath, '-C', outDir, binaryName]);
      fs.chmodSync(path.join(outDir, binaryName), 0o755);
    }

    console.log(`  -> npm/${platformKey}/bin/${binaryName}`);
  }

  fs.rmSync(tmpDir, { recursive: true, force: true });
  console.log('\nDone.');
  process.exit(0);
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const headers = { 'User-Agent': 'stripe-cli-npm-publish' };
    if (GITHUB_TOKEN) headers['Authorization'] = `Bearer ${GITHUB_TOKEN}`;

    function get(u, withAuth) {
      const opts = withAuth ? { headers } : { headers: { 'User-Agent': headers['User-Agent'] } };
      https.get(u, opts, res => {
        if (res.statusCode === 301 || res.statusCode === 302) {
          get(res.headers.location, false);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode} for ${u}`));
          return;
        }
        res.pipe(file);
        file.on('finish', () => file.close(resolve));
      }).on('error', reject);
    }

    get(url, true);
  });
}

function verifySha256(filePath, checksumPath, archiveName) {
  const content  = fs.readFileSync(checksumPath, 'utf8');
  const line     = content.split('\n').find(l => l.includes(archiveName));
  if (!line) {
    console.error(`  checksum entry not found for ${archiveName}`);
    process.exit(1);
  }
  const expected = line.trim().split(/\s+/)[0];
  const actual   = crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
  if (actual !== expected) {
    console.error(`  checksum mismatch: expected ${expected}, got ${actual}`);
    process.exit(1);
  }
}

main().catch(err => {
  console.error('fetch-binaries failed:', err.message);
  process.exit(1);
});
