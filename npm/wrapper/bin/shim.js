#!/usr/bin/env node
'use strict';
const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const resolvePlatform = require('../resolve-platform');

const PLATFORMS = require('../platforms.json');

const key = `${process.platform}-${process.arch}`;
const platform = PLATFORMS[key];
if (!platform) {
  console.error(`@stripe/cli: unsupported platform "${key}". See https://docs.stripe.com/stripe-cli`);
  process.exit(1);
}

const vendorPath = path.join(__dirname, '..', 'vendor', 'bin', platform.bin);
const binPath = fs.existsSync(vendorPath) ? vendorPath : resolvePlatform(platform);
if (!binPath) {
  console.error('@stripe/cli: binary not found. Reinstall @stripe/cli to download it.');
  process.exit(1);
}

// Detect invocation method using npm-injected env vars:
// - npm_config_user_agent is set by npm/npx at invocation time but never by a direct shell exec,
//   so its absence means the user ran `stripe` directly (global install, PATH symlink).
// - npm_lifecycle_event is set to the script name during `npm run` but not by npx,
//   so its presence distinguishes `npm run` scripts from a bare `npx @stripe/cli` call.
let installMethod;
if (!process.env.npm_config_user_agent) {
  installMethod = 'npm_global';
} else if (process.env.npm_lifecycle_event) {
  installMethod = 'npm_run';
} else {
  installMethod = 'npx';
}
const result = spawnSync(binPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: { ...process.env, STRIPE_INSTALL_METHOD: installMethod },
});
process.exit(result.status ?? 1);
