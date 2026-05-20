'use strict';
const https  = require('https');
const fs     = require('fs');
const path   = require('path');
const crypto = require('crypto');
const { execFileSync } = require('child_process');

const PKG       = require('../package.json');
const PLATFORMS = require('../platforms.json');
const VERSION   = PKG.version;
const BASE      = `https://github.com/stripe/stripe-cli/releases/download/v${VERSION}`;

const key      = `${process.platform}-${process.arch}`;
const platform = PLATFORMS[key];

if (!platform) {
  console.warn(`@stripe/cli: unsupported platform "${key}", skipping binary download.`);
  process.exit(0);
}

// If a platform package was already installed, nothing to do.
try {
  require.resolve(`${platform.pkg}/package.json`);
  process.exit(0);
} catch {}

const archive      = platform.archiveTemplate.replace('${VERSION}', VERSION);
const checksumFile = platform.checksums;
const vendorDir    = path.join(__dirname, '..', 'vendor', 'bin');

fs.mkdirSync(vendorDir, { recursive: true });

const archivePath  = path.join(vendorDir, '..', archive);
const checksumPath = path.join(vendorDir, '..', checksumFile);

console.log(`@stripe/cli: platform package not found, downloading from GitHub Releases...`);

async function main() {
  await download(`${BASE}/${checksumFile}`, checksumPath);
  await download(`${BASE}/${archive}`, archivePath);
  verifySha256(archivePath, checksumPath, archive);
  extractBinary(archivePath, vendorDir, platform.bin);
  // Clean up downloaded archives.
  fs.unlinkSync(archivePath);
  fs.unlinkSync(checksumPath);
  console.log(`@stripe/cli: binary installed successfully.`);
  process.exit(0);
}

main().catch(err => {
  console.error(`@stripe/cli: failed to download binary: ${err.message}`);
  process.exit(1);
});

// ---------------------------------------------------------------------------

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    function get(u) {
      https.get(u, res => {
        if (res.statusCode === 301 || res.statusCode === 302) {
          get(res.headers.location);
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
    get(url);
  });
}

function verifySha256(filePath, checksumPath, archiveName) {
  const content  = fs.readFileSync(checksumPath, 'utf8');
  const line     = content.split('\n').find(l => l.includes(archiveName));
  if (!line) {
    throw new Error(`checksum entry not found for ${archiveName}`);
  }
  const expected = line.trim().split(/\s+/)[0];
  const actual   = crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
  if (actual !== expected) {
    throw new Error(`SHA256 mismatch for ${archiveName}\n  expected: ${expected}\n  actual:   ${actual}`);
  }
}

function extractBinary(archivePath, outDir, binaryName) {
  if (archivePath.endsWith('.zip')) {
    execFileSync('powershell', [
      '-NoProfile', '-NonInteractive', '-Command',
      `$tmp = '${path.join(outDir, '..', '_extract')}'; ` +
      `Expand-Archive -Path '${archivePath}' -DestinationPath $tmp -Force; ` +
      `Move-Item -Path (Join-Path $tmp '${binaryName}') -Destination '${path.join(outDir, binaryName)}' -Force; ` +
      `Remove-Item -Recurse -Force $tmp`,
    ]);
  } else {
    execFileSync('tar', ['xzf', archivePath, '-C', outDir, binaryName]);
    fs.chmodSync(path.join(outDir, binaryName), 0o755);
  }
}
