'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const { Readable } = require('node:stream');
const { createRequire } = require('node:module');
const { execFileSync, spawnSync } = require('node:child_process');
const { test } = require('node:test');

const source = path.resolve(__dirname, '../wrapper');
const platforms = require('../wrapper/platforms.json');
const { version } = require('../wrapper/package.json');
const platform = platforms[`${process.platform}-${process.arch}`];
assert(platform, 'tests require a supported platform');

function fixture(t, scoped = true) {
  const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'cli-resolution-')));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const wrapper = scoped
    ? path.join(root, 'project', 'node_modules', '@stripe', 'cli')
    : path.join(root, 'project', 'wrapper');
  fs.cpSync(source, wrapper, { recursive: true });
  return {
    root,
    wrapper,
    nested: path.join(wrapper, 'node_modules', platform.pkg),
    sibling: path.join(path.dirname(wrapper), path.basename(platform.pkg)),
    ancestor: path.join(root, 'node_modules', platform.pkg),
    vendor: path.join(wrapper, 'vendor', 'bin', platform.bin),
    resolve: () => require(path.join(wrapper, 'resolve-platform.js'))(platform),
  };
}

function write(file, content) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
}

function packageAt(directory, metadata = {}, binary = true) {
  write(path.join(directory, 'package.json'), JSON.stringify({ name: platform.pkg, version, ...metadata }));
  const file = path.join(directory, 'bin', platform.bin);
  if (binary) write(file, 'fixture binary');
  return file;
}

function runScript(wrapper, relative, overrides, processOverrides = {}) {
  const file = path.join(wrapper, relative);
  const localRequire = createRequire(file);
  vm.runInNewContext(fs.readFileSync(file, 'utf8'), {
    __dirname: path.dirname(file),
    require: name => Object.hasOwn(overrides, name) ? overrides[name] : localRequire(name),
    process: {
      platform: process.platform,
      arch: process.arch,
      argv: ['node', file, '--help', 'argument with spaces'],
      env: { TEST_FORWARDING: 'preserved' },
      exit: () => {},
      ...processOverrides,
    },
    console: { log() {}, warn() {}, error() {} },
  }, { filename: file });
}

function launch(f, env = {}) {
  const calls = [];
  const exits = [];
  const stop = new Error('exit');
  try {
    runScript(f.wrapper, 'bin/shim.js', {
      child_process: { spawnSync: (...args) => { calls.push(args); return { status: 7 }; } },
    }, {
      env: { TEST_FORWARDING: 'preserved', ...env },
      exit: status => { exits.push(status); throw stop; },
    });
  } catch (err) {
    if (err !== stop) throw err;
  }
  return { calls, exits };
}

async function install(f, validChecksum = true) {
  const downloads = [];
  const extractions = [];
  const archive = platform.archiveTemplate.replace('${VERSION}', version);
  const body = Buffer.from('fixture archive');
  const digest = crypto.createHash('sha256').update(body).digest('hex');
  const checksum = `${validChecksum ? digest : '0'.repeat(64)}  ${archive}\n`;
  const stop = new Error('exit');
  const status = await new Promise(resolve => {
    try {
      runScript(f.wrapper, 'scripts/postinstall.js', {
        https: {
          get: (url, callback) => {
            downloads.push(url);
            const response = Readable.from([url.endsWith(platform.checksums) ? checksum : body]);
            response.statusCode = 200;
            callback(response);
            return response;
          },
        },
        child_process: {
          execFileSync: (...args) => {
            extractions.push(args);
            write(f.vendor, 'downloaded binary');
          },
        },
      }, { exit: status => {
        resolve(status);
        if (downloads.length === 0) throw stop;
      } });
    } catch (err) {
      if (err !== stop) throw err;
    }
  });
  return { status, downloads, extractions };
}

for (const location of ['nested', 'sibling']) {
  test(`accepts a valid ${location} package in both callers`, async t => {
    const f = fixture(t);
    const binary = packageAt(f[location]);
    assert.equal(launch(f).calls[0][0], binary);
    const result = await install(f);
    assert.equal(result.status, 0);
    assert.deepEqual(result.downloads, []);
    assert.equal(fs.existsSync(f.vendor), false);
  });

  for (const invalid of ['name', 'version', 'missing binary', 'binary directory', 'invalid JSON', 'null JSON']) {
    test(`rejects ${location} package with ${invalid}`, async t => {
      const f = fixture(t);
      const metadata = invalid === 'name' ? { name: '@stripe/not-the-cli' }
        : invalid === 'version' ? { version: '0.0.0' } : {};
      const binary = packageAt(f[location], metadata, invalid !== 'missing binary');
      if (invalid === 'binary directory') {
        fs.unlinkSync(binary);
        fs.mkdirSync(binary);
      }
      if (invalid === 'invalid JSON' || invalid === 'null JSON') {
        write(path.join(f[location], 'package.json'), invalid === 'null JSON' ? 'null' : '{');
      }
      assert.deepEqual(launch(f), { calls: [], exits: [1] });
      const result = await install(f);
      assert.equal(result.status, 0);
      assert.equal(result.downloads.length, 2);
      assert.equal(result.extractions.length, 1);
    });
  }
}

test('nested package takes precedence and an invalid nested package allows a valid sibling', t => {
  const f = fixture(t);
  const sibling = packageAt(f.sibling);
  const nested = packageAt(f.nested);
  assert.equal(f.resolve(), nested);
  packageAt(f.nested, { version: '0.0.0' });
  assert.equal(f.resolve(), sibling);
});

test('does not treat an unscoped wrapper parent as an installation scope', t => {
  const f = fixture(t, false);
  packageAt(f.sibling);
  assert.equal(f.resolve(), null);
  const binary = packageAt(f.nested);
  assert.equal(f.resolve(), binary);
});

test('ancestor package present before installation cannot suppress the download', async t => {
  const f = fixture(t);
  packageAt(f.ancestor);
  // Both permitted optional-dependency locations must actually be absent.
  assert.equal(fs.existsSync(f.nested), false);
  assert.equal(fs.existsSync(f.sibling), false);
  assert.deepEqual(launch(f), { calls: [], exits: [1] });
  const result = await install(f);
  assert.equal(result.status, 0);
  assert.equal(result.downloads.length, 2);
  assert.equal(result.extractions.length, 1);
  assert.equal(launch(f).calls[0][0], f.vendor);
});

test('ancestor package added after installation cannot replace the downloaded binary', async t => {
  const f = fixture(t);
  assert.equal((await install(f)).status, 0);
  packageAt(f.ancestor);
  assert.equal(launch(f).calls[0][0], f.vendor);
  fs.unlinkSync(f.vendor);
  assert.deepEqual(launch(f), { calls: [], exits: [1] });
});

test('vendor binary takes precedence over valid platform packages', t => {
  const f = fixture(t);
  packageAt(f.nested);
  packageAt(f.sibling);
  write(f.vendor, 'vendor binary');
  assert.equal(launch(f).calls[0][0], f.vendor);
});

test('launcher preserves arguments, environment, invocation method, and exit status', t => {
  const f = fixture(t);
  packageAt(f.nested);
  for (const [env, method] of [
    [{}, 'npm_global'],
    [{ npm_config_user_agent: 'npm' }, 'npx'],
    [{ npm_config_user_agent: 'npm', npm_lifecycle_event: 'test' }, 'npm_run'],
  ]) {
    const result = launch(f, env);
    const [, args, options] = result.calls[0];
    assert.deepEqual(Array.from(args), ['--help', 'argument with spaces']);
    assert.equal(options.stdio, 'inherit');
    assert.equal(options.env.TEST_FORWARDING, 'preserved');
    assert.equal(options.env.STRIPE_INSTALL_METHOD, method);
    assert.deepEqual(result.exits, [7]);
  }
});

test('fallback still rejects a checksum mismatch before extraction', async t => {
  const f = fixture(t);
  const result = await install(f, false);
  assert.equal(result.status, 1);
  assert.equal(result.downloads.length, 2);
  assert.equal(result.extractions.length, 0);
  assert.equal(fs.existsSync(f.vendor), false);
});

test('published archive contains the locator and runs both callers from an isolated installation', async t => {
  const f = fixture(t);
  const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm';
  const destination = process.platform === 'win32' ? `"${f.root}"` : f.root;
  const [packed] = JSON.parse(execFileSync(npm, ['pack', '--json', '--ignore-scripts', '--pack-destination', destination], {
    cwd: source,
    encoding: 'utf8',
    shell: process.platform === 'win32',
    env: { ...process.env, npm_config_cache: path.join(f.root, 'cache') },
  }));
  assert(packed.files.some(file => file.path === 'resolve-platform.js'));
  const unpack = path.join(f.root, 'unpack');
  fs.mkdirSync(unpack);
  execFileSync('tar', ['xzf', path.join(f.root, packed.filename), '-C', unpack]);
  fs.rmSync(f.wrapper, { recursive: true });
  fs.renameSync(path.join(unpack, 'package'), f.wrapper);
  assert.equal(fs.existsSync(f.nested), false);
  assert.equal(fs.existsSync(f.sibling), false);
  const fallback = await install(f);
  assert.equal(fallback.status, 0);
  assert.equal(fallback.downloads.length, 2);
  fs.unlinkSync(f.vendor);
  const binary = packageAt(f.sibling);
  assert.deepEqual((await install(f)).downloads, []);
  fs.copyFileSync(process.execPath, binary);
  fs.chmodSync(binary, 0o755);
  const script = 'process.stdout.write(JSON.stringify({args:process.argv.slice(1),method:process.env.STRIPE_INSTALL_METHOD,env:process.env.TEST_FORWARDING}));process.exitCode=7';
  const result = spawnSync(process.execPath, [path.join(f.wrapper, 'bin/shim.js'), '-e', script, 'argument with spaces'], {
    encoding: 'utf8',
    env: { ...process.env, TEST_FORWARDING: 'preserved', npm_config_user_agent: '', npm_lifecycle_event: '' },
  });
  assert.equal(result.status, 7, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout), {
    args: ['argument with spaces'], method: 'npm_global', env: 'preserved',
  });
});
