const test = require('node:test');
const assert = require('node:assert/strict');
const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const readline = require('node:readline');

const BIN = path.join(__dirname, 'skill-pi.js');
const SKILL_PI_ROOT = path.join(__dirname, '..');

function tmpdir(name) {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'skill-pi-' + name + '-'));
}

function makeGitRoot(dir, extra) {
  fs.mkdirSync(path.join(dir, '.git'));
  if (extra) fs.writeFileSync(path.join(dir, extra), 'x');
  return dir;
}

function makeEnv(home, rootName, extra) {
  const binDir = path.join(home, 'node_modules', '.bin');
  fs.mkdirSync(binDir, { recursive: true });
  const bin = path.join(binDir, 'pi');
  const stub = [
    '#!/usr/bin/env node',
    'const fs = require("fs");',
    'try {',
    '  if (process.env.SKILL_PI_TEST_LOG) fs.appendFileSync(process.env.SKILL_PI_TEST_LOG, "1");',
    '  if (process.env.SKILL_PI_TEST_ARGV) fs.writeFileSync(process.env.SKILL_PI_TEST_ARGV, JSON.stringify({ argv: process.argv.slice(2), skip: process.env.PI_SKIP_VERSION_CHECK, agentDir: process.env.PI_CODING_AGENT_DIR }));',
    '} catch (e) {}',
    'if (process.env.SKILL_PI_TEST_EXIT) process.exit(parseInt(process.env.SKILL_PI_TEST_EXIT, 10)); else process.exit(0);',
  ].join('\n');
  fs.writeFileSync(bin, stub);
  fs.chmodSync(bin, 0o755);
  return {
    ...process.env,
    HOME: home,
    SKILL_PI_AGENT_DIR: home,
    SKILL_PI_PI_BIN: bin,
    SKILL_PI_TRUST_AUTO: '1',
    ...extra,
  };
}

function run(args, opts) {
  return spawnSync(process.execPath, [BIN, ...args], {
    encoding: 'utf8',
    cwd: opts.cwd,
    env: opts.env,
    input: opts.input,
    timeout: 20000,
  });
}

function launch(args, opts) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [BIN, ...args], {
      cwd: opts.cwd,
      env: opts.env,
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => (stdout += d.toString()));
    child.stderr.on('data', (d) => (stderr += d.toString()));
    const rl = readline.createInterface({ input: child.stdout });
    const respond = (line) => {
      if (line.includes('? (y/n)')) {
        child.stdin.write(opts.input ? 'n\n' : 'y\n');
      }
    };
    rl.on('line', respond);
    child.on('close', (code) => {
      rl.close();
      resolve({ code, stdout, stderr, pid: child.pid });
    });
    child.on('error', (err) => {
      rl.close();
      resolve({ code: null, stdout, stderr: stderr + String(err), pid: child.pid });
    });
  });
}

function logLines(home) {
  const f = path.join(home, 'spawns.log');
  try {
    return fs.readFileSync(f, 'utf8').split('1').length - 1;
  } catch (e) {
    return 0;
  }
}

test('exit-75 respawn: launcher re-spawns on 75, exits non-zero at cap, log > 1 line', async () => {
  const home = tmpdir('home-75');
  const root = makeGitRoot(tmpdir('root-75'), 'main.js');
  const env = makeEnv(home, path.basename(root), {
    SKILL_PI_TEST_LOG: path.join(home, 'spawns.log'),
    SKILL_PI_TEST_EXIT: '75',
  });

  const res = await launch([], { cwd: root, env });
  assert.notStrictEqual(res.code, 0, 'launcher must exit non-zero after the cap');
  assert.strictEqual(res.code, 75, 'launcher must exit with the child code after the cap');
  const lines = logLines(home);
  assert.ok(lines > 1, 'a second spawn must have happened, log has ' + lines + ' line(s)');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('exit-75 respawn: exit 0 child spawns exactly once, log has 1 line', async () => {
  const home = tmpdir('home-0');
  const root = makeGitRoot(tmpdir('root-0'), 'main.js');
  const env = makeEnv(home, path.basename(root), {
    SKILL_PI_TEST_LOG: path.join(home, 'spawns.log'),
  });

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0);
  const lines = logLines(home);
  assert.strictEqual(lines, 1, 'exit 0 must spawn exactly once, log has ' + lines + ' line(s)');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('buildShim: materializes piConfig.configDir .skill-pi and re-symlinks on rebuild', async () => {
  const home = tmpdir('home-shim');
  const root = makeGitRoot(tmpdir('root-shim'), 'main.js');
  const env = makeEnv(home, path.basename(root));

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0);

  const shim = path.join(home, 'cwd', path.basename(root));
  const pkg = JSON.parse(fs.readFileSync(path.join(shim, 'package.json'), 'utf8'));
  assert.strictEqual(pkg.piConfig.configDir, '.skill-pi');
  assert.ok(fs.existsSync(path.join(shim, 'main.js')), 'existing file must be linked into the shim');
  assert.ok(fs.lstatSync(path.join(shim, 'main.js')).isSymbolicLink(), 'shim entries are symlinks into the real root');

  fs.writeFileSync(path.join(root, 'fresh.js'), 'module.exports = 1;\n');
  const res2 = await launch([], { cwd: root, env });
  assert.strictEqual(res2.code, 0);
  assert.ok(fs.existsSync(path.join(shim, 'fresh.js')), 'new file in real root must be visible in the shim after rebuild');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('buildShim: missing root exits non-zero, no spawn', () => {
  const home = tmpdir('home-missing');
  const env = makeEnv(home, 'does-not-exist');
  const res = run(['--where', '--root', path.join(home, 'does-not-exist')], { cwd: home, env });
  assert.notStrictEqual(res.status, 0, 'missing root must exit non-zero');
  fs.rmSync(home, { recursive: true, force: true });
});

test('--where: emits 8-key JSON, configDir ends with /.skill-pi, agentDir contains .skill-pi', () => {
  const home = path.join(tmpdir('home-where'), '.skill-pi');
  fs.mkdirSync(home);
  const root = makeGitRoot(tmpdir('root-where'), 'main.js');
  const env = makeEnv(home, path.basename(root));
  const res = run(['--where', '--root', root], { cwd: root, env });
  assert.strictEqual(res.status, 0, '--where must exit 0');
  const where = JSON.parse(res.stdout);
  assert.deepStrictEqual(
    Object.keys(where).sort(),
    ['agentDir', 'configDir', 'globalConfig', 'piBin', 'piVersion', 'projectRoot', 'shimDir', 'skillPiRoot']
  );
  assert.ok(where.configDir.endsWith('/.skill-pi'), 'configDir must end with /.skill-pi, got ' + where.configDir);
  assert.ok(where.agentDir.includes('.skill-pi'), 'agentDir must contain .skill-pi, got ' + where.agentDir);
  assert.strictEqual(where.piVersion, '1.1.0');
  fs.rmSync(path.dirname(home), { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('version: prints pinned pi version and exits 0', () => {
  const home = tmpdir('home-version');
  const env = makeEnv(home, 'x');
  const res = run(['version'], { cwd: home, env });
  assert.strictEqual(res.status, 0, 'version must exit 0');
  assert.ok(res.stdout.includes('1.1.0'), 'stdout must contain 1.1.0, got: ' + res.stdout);
  fs.rmSync(home, { recursive: true, force: true });
});

test('pi flag forwarding: passthrough argv reaches the pi child', async () => {
  const home = tmpdir('home-fwd');
  const root = makeGitRoot(tmpdir('root-fwd'), 'main.js');
  const argvFile = path.join(home, 'argv.json');
  const env = makeEnv(home, path.basename(root), { SKILL_PI_TEST_ARGV: argvFile });

  const res = await launch(['--session', 'test123'], { cwd: root, env });
  assert.strictEqual(res.code, 0);
  const data = JSON.parse(fs.readFileSync(argvFile, 'utf8'));
  assert.ok(data.argv.includes('test123'), 'stub argv must contain test123, got: ' + JSON.stringify(data.argv));

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('spawn env + args: PI_SKIP_VERSION_CHECK=1 and owned args present', async () => {
  const home = tmpdir('home-env');
  const root = makeGitRoot(tmpdir('root-env'), 'main.js');
  const argvFile = path.join(home, 'argv.json');
  const env = makeEnv(home, path.basename(root), { SKILL_PI_TEST_ARGV: argvFile });

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0);
  const data = JSON.parse(fs.readFileSync(argvFile, 'utf8'));
  assert.strictEqual(data.skip, '1', 'PI_SKIP_VERSION_CHECK must be "1" in the child env');
  assert.strictEqual(data.agentDir, home, 'PI_CODING_AGENT_DIR must point at the agent dir');
  assert.ok(data.argv.includes('--no-skills'), 'owned arg --no-skills must be present');
  assert.ok(data.argv.includes('--no-prompt-templates'), 'owned arg --no-prompt-templates must be present');
  assert.ok(data.argv.includes('--no-themes'), 'owned arg --no-themes must be present');
  assert.ok(data.argv.includes('--extension'), 'owned arg --extension must be present');
  assert.ok(data.argv.includes(SKILL_PI_ROOT), '--extension value must be the skill-pi root');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('trust: fresh auto-accept writes trust.json keyed by shim realpath; second run needs no prompt', async () => {
  const home = tmpdir('home-trust');
  const root = makeGitRoot(tmpdir('root-trust'), 'main.js');
  const rootName = path.basename(root);
  const env = makeEnv(home, rootName);
  const trustFile = path.join(home, 'trust.json');
  const shim = path.join(home, 'cwd', rootName);

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0, 'auto-accept run must exit 0, stderr: ' + res.stderr.slice(0, 400));
  assert.ok(fs.existsSync(trustFile), 'trust.json must exist after the auto-accept run');
  const trust = JSON.parse(fs.readFileSync(trustFile, 'utf8'));
  const expectedKey = fs.realpathSync(shim);
  assert.ok(expectedKey in trust, 'trust.json must have a key matching the shim realpath, got: ' + JSON.stringify(Object.keys(trust)));
  assert.strictEqual(trust[expectedKey], true, 'trust value must be boolean true (pi readTrustFile contract)');

  const res2 = await launch([], { cwd: root, env });
  assert.strictEqual(res2.code, 0, 'second run must exit 0 without re-prompting');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});
