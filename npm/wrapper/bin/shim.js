#!/usr/bin/env node
'use strict';
const { spawnSync } = require('child_process');
const path = require('path');

const PLATFORMS = require('../platforms.json');

const key = `${process.platform}-${process.arch}`;
const platform = PLATFORMS[key];
if (!platform) {
  console.error(`@stripe/cli: unsupported platform "${key}". See https://docs.stripe.com/stripe-cli`);
  process.exit(1);
}

let binPath;
try {
  const pkgDir = path.dirname(require.resolve(`${platform.pkg}/package.json`));
  binPath = path.join(pkgDir, 'bin', platform.bin);
} catch {
  binPath = path.join(__dirname, '..', 'vendor', 'bin', platform.bin);
}

const isNpx = !!process.env.npm_config_user_agent;
const result = spawnSync(binPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: { ...process.env, STRIPE_INSTALL_METHOD: isNpx ? 'npx' : 'npm' },
});
process.exit(result.status ?? 1);
