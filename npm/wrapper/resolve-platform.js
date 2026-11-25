'use strict';
const fs = require('fs');
const path = require('path');

const { version } = require('./package.json');

module.exports = function resolvePlatform(platform) {
  const candidates = [path.join(__dirname, 'node_modules', platform.pkg)];
  const scopeDir = path.dirname(__dirname);
  if (path.basename(scopeDir) === '@stripe') {
    candidates.push(path.join(scopeDir, path.basename(platform.pkg)));
  }

  for (const packageDir of candidates) {
    try {
      const metadata = JSON.parse(fs.readFileSync(path.join(packageDir, 'package.json'), 'utf8'));
      const binary = path.join(packageDir, 'bin', platform.bin);
      if (metadata.name === platform.pkg && metadata.version === version && fs.statSync(binary).isFile()) {
        return binary;
      }
    } catch {}
  }
  return null;
};
