const test = require('node:test');
const assert = require('node:assert');
const { spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const readline = require('node:readline');

const BIN = path.join(__dirname, 'skill-pi.js');
const SKILL_PI_ROOT = path.join(__dirname, '..');

const STUB_SCRIPT = `
const fs = require('fs');
const stateFile = process.env.SKILL_PI_TEST_STATE;
let n = 0;
if (stateFile && fs.existsSync(stateFile)) {
  try { n = parseInt(fs.readFileSync(stateFile, 'utf8'), 10); } catch (e) {}
}
n += 1;
if (stateFile) fs.writeFileSync(stateFile, String(n));
fs.writeFileSync(process.argv[2] + '.spawn', '');
process.exit(n);
`;

const EXIT_CODES = [75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75];

function tmpdir(name) {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'skill-pi-' + name + '-'));
}

function makeGitRoot(dir, extra) {
  fs.mkdirSync(path.join(dir, '.git'));
  if (extra) fs.writeFileSync(path.join(dir, extra), 'x');
  return dir;
}

function stubPiBin(dir, lines) {
  const binDir = path.join(dir, 'node_modules', '.bin');
  fs.mkdirSync(binDir, { recursive: true });
  const bin = path.join(binDir, 'pi');
  fs.writeFileSync(bin, '#!/usr/bin/env node\n' + lines.join('\n') + '\n');
  fs.chmodSync(bin, 0o755);
  return bin;
}

function makeEnv(home) {
  return {
    ...process.env,
    SKILL_PI_AGENT_DIR: home,
    SKILL_PI_PI_BIN: stubPiBin(home, STUB_SCRIPT.trim().split('\n')),
    SKILL_PI_TRUST_AUTO: '1',
  };
}

function run(args, opts) {
  const res = spawnSync(process.execPath, [BIN, ...args], {
    encoding: 'utf8',
    cwd: opts.cwd,
    env: opts.env,
    input: opts.input,
    timeout: 20000,
  });
  return res;
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
      if (line.includes('Trust this project?')) {
        child.stdin.write('y\n');
        return;
      }
      if (line.includes('skill-pi-trust-prompt')) {
        child.stdin.write(opts.input || 'n\n');
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

function spawnLog(home) {
  const f = path.join(home, 'pi.spawn');
  try {
    return fs.readFileSync(f, 'utf8').split('\n').filter(Boolean);
  } catch (e) {
    return [];
  }
}

function countSpawns(home) {
  return fs.existsSync(path.join(home, 'pi.spawn')) ? 1 : 0;
}

function shimDirFor(home, rootName) {
  return path.join(home, 'cwd', rootName);
}

test('edge spawn-respawn-loop: exit 75 re-spawns, exit 0 does not, cap at 10', async () => {
  for (const code of [0, 1, 75]) {
    const home = tmpdir('home');
    const root = makeGitRoot(tmpdir('root'), 'main.js');
    const state = path.join(home, 'counter');
    const env = makeEnv(home);
    delete env.SKILL_PI_PI_BIN;
    const binDir = path.join(home, 'node_modules', '.bin');
    fs.mkdirSync(binDir, { recursive: true });
    const bin = path.join(binDir, 'pi');
    fs.writeFileSync(
      bin,
      '#!/usr/bin/env node\n' +
        'const fs = require("fs");\n' +
        'const stateFile = process.argv[1];\n' +
        'let n = 0;\n' +
        'if (fs.existsSync(stateFile)) { try { n = parseInt(fs.readFileSync(stateFile, "utf8"), 10); } catch (e) {} }\n' +
        'n += 1;\n' +
        'fs.writeFileSync(stateFile, String(n));\n' +
        'process.exit(' + code + ');\n'
    );
    fs.chmodSync(bin, 0o755);
    env.SKILL_PI_PI_BIN = bin;
    env.SKILL_PI_TEST_STATE = state;

    const expectedSpawns = code === 75 ? 11 : 1;
    const t0 = Date.now();
    const res = await launch([], { cwd: root, env });
    const elapsed = Date.now() - t0;

    let spawns = 0;
    try {
      spawns = parseInt(fs.readFileSync(state, 'utf8'), 10);
    } catch (e) {}
    assert.strictEqual(spawns, expectedSpawns, 'expected ' + expectedSpawns + ' spawns for exit ' + code + ', got ' + spawns + ' in ' + elapsed + 'ms');
    if (code === 75) {
      assert.strictEqual(res.code, 75, 'launcher must exit with the child code after the cap');
    } else {
      assert.strictEqual(res.code, code, 'launcher must exit with the child code');
    }
    fs.rmSync(home, { recursive: true, force: true });
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('happy path cwd-shim: buildShim materializes piConfig and re-symlinks on rebuild', async () => {
  const home = tmpdir('home-shim');
  const root = makeGitRoot(tmpdir('root-shim'), 'main.js');
  const env = makeEnv(home);
  const rootName = path.basename(root);

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0, 'launch must exit 0 when the child exits 0');

  const shim = shimDirFor(home, rootName);
  const pkg = JSON.parse(fs.readFileSync(path.join(shim, 'package.json'), 'utf8'));
  assert.strictEqual(pkg.piConfig.configDir, '.skill-pi');
  assert.strictEqual(pkg.name, 'skill-pi-cwd');
  assert.strictEqual(pkg.version, '0.0.0');

  fs.writeFileSync(path.join(root, 'fresh.js'), 'module.exports = 1;\n');
  const res2 = await launch([], { cwd: root, env });
  assert.strictEqual(res2.code, 0);
  const fresh = path.join(shim, 'fresh.js');
  assert.ok(fs.existsSync(fresh), 'new file in real root must be visible through the shim after rebuild');
  assert.ok(fs.lstatSync(fresh).isSymbolicLink(), 'shim entries are symlinks into the real root');

  const stale = path.join(shim, 'removed.js');
  fs.writeFileSync(path.join(root, 'removed.js'), 'x');
  fs.rmSync(path.join(root, 'removed.js'));
  await launch([], { cwd: root, env });
  assert.ok(!fs.existsSync(stale), 'rebuild is from scratch: removed files must not linger in the shim');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('edge missing-root: exits non-zero, no spawn', () => {
  const home = tmpdir('home-missing');
  const env = makeEnv(home);
  const res = run(['--where'], { cwd: home, env });
  assert.notStrictEqual(res.status, 0, 'missing root must exit non-zero');
  assert.ok(!fs.existsSync(path.join(home, 'pi.spawn')), 'no spawn may happen when the root is missing');
  fs.rmSync(home, { recursive: true, force: true });
});

test('happy path where-json: emits the 8-key JSON and exits 0', () => {
  const home = tmpdir('home-where');
  const root = makeGitRoot(tmpdir('root-where'), 'main.js');
  const env = makeEnv(home);
  const res = run(['--where'], { cwd: root, env });
  assert.strictEqual(res.status, 0, '--where must exit 0');
  const where = JSON.parse(res.stdout);
  const keys = Object.keys(where).sort();
  assert.deepStrictEqual(keys, [
    'agentDir',
    'configDir',
    'globalConfig',
    'piBin',
    'piVersion',
    'projectRoot',
    'shimDir',
    'skillPiRoot',
  ]);
  assert.strictEqual(where.configDir, path.join(root, '.skill-pi'));
  assert.strictEqual(where.agentDir, home);
  assert.strictEqual(where.globalConfig, home);
  assert.strictEqual(where.shimDir, path.join(home, 'cwd', path.basename(root)));
  assert.strictEqual(where.projectRoot, root);
  assert.strictEqual(where.skillPiRoot, SKILL_PI_ROOT);
  assert.strictEqual(where.piVersion, '1.1.0');
  assert.strictEqual(where.piBin, path.join(home, 'node_modules', '.bin', 'pi'));
  assert.ok(!fs.existsSync(path.join(home, 'pi.spawn')), 'no spawn may happen for --where');
  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('happy path owned-subcommands: version prints the pinned pi version without the TUI', () => {
  const home = tmpdir('home-version');
  const env = makeEnv(home);
  const res = run(['version'], { cwd: home, env });
  assert.strictEqual(res.status, 0, 'version must exit 0');
  assert.strictEqual(res.stdout.trim(), '1.1.0');
  assert.ok(!fs.existsSync(path.join(home, 'pi.spawn')), 'version must not spawn pi');
  fs.rmSync(home, { recursive: true, force: true });
});

test('happy path owned-subcommands: init materializes <project>/.skill-pi from the global config', () => {
  const home = tmpdir('home-init');
  const root = makeGitRoot(tmpdir('root-init'), 'main.js');
  fs.writeFileSync(path.join(home, 'settings.json'), '{"a":1}\n');
  fs.writeFileSync(path.join(home, 'mcp.json'), '{"mcpServers":{}}\n');
  const env = makeEnv(home);
  const res = run(['init'], { cwd: root, env });
  assert.strictEqual(res.status, 0, 'init must exit 0');
  const projectConfig = path.join(root, '.skill-pi');
  assert.ok(fs.existsSync(projectConfig), 'init must materialize <project>/.skill-pi/');
  assert.deepStrictEqual(JSON.parse(fs.readFileSync(path.join(projectConfig, 'settings.json'), 'utf8')), { a: 1 });
  assert.deepStrictEqual(JSON.parse(fs.readFileSync(path.join(projectConfig, 'mcp.json'), 'utf8')), { mcpServers: {} });
  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('happy path owned-subcommands: sync copies skills from ~/.agents/skills into the pi skills location', () => {
  const home = tmpdir('home-sync');
  const root = makeGitRoot(tmpdir('root-sync'), 'main.js');
  const srcSkill = path.join(home, '.agents', 'skills', 'demo-skill');
  fs.mkdirSync(srcSkill, { recursive: true });
  fs.writeFileSync(path.join(srcSkill, 'SKILL.md'), '# demo\n');
  const env = makeEnv(home);
  const res = run(['sync'], { cwd: root, env });
  assert.strictEqual(res.status, 0, 'sync must exit 0');
  const dest = path.join(home, '.skill-pi', 'agent', 'skills', 'demo-skill');
  assert.ok(fs.existsSync(path.join(dest, 'SKILL.md')), 'sync must copy the skill into the pi skills location');
  assert.strictEqual(
    fs.readFileSync(path.join(dest, 'SKILL.md'), 'utf8'),
    fs.readFileSync(path.join(srcSkill, 'SKILL.md'), 'utf8')
  );
  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('happy path quiet-startup: spawn env + owned argv + pi flag forwarding', async () => {
  const home = tmpdir('home-spawn');
  const root = makeGitRoot(tmpdir('root-spawn'), 'main.js');
  const bin = stubPiBin(
    home,
    [
      'process.exit(0)',
      'process.stdout.write("never")',
    ]
  );
  const probe = stubPiBin(
    home,
    [
      'const fs = require("fs");',
      'fs.writeFileSync(process.argv[1], JSON.stringify({ argv: process.argv.slice(2), env: { PI_CODING_AGENT_DIR: process.env.PI_CODING_AGENT_DIR, PI_SKIP_VERSION_CHECK: process.env.PI_SKIP_VERSION_CHECK }, cwd: process.cwd() }));',
      'process.exit(0);',
    ]
  );
  const out = path.join(home, 'probe.json');
  const env = makeEnv(home);
  env.SKILL_PI_PI_BIN = probe;

  const res = await launch(['--session', 'test123'], { cwd: root, env });
  assert.strictEqual(res.code, 0);
  const data = JSON.parse(fs.readFileSync(out, 'utf8'));
  assert.strictEqual(data.env.PI_CODING_AGENT_DIR, home, 'PI_CODING_AGENT_DIR must point at the agent dir');
  assert.strictEqual(data.env.PI_SKIP_VERSION_CHECK, '1', 'PI_SKIP_VERSION_CHECK must be 1 in the spawn env');
  assert.strictEqual(data.cwd, path.join(home, 'cwd', path.basename(root)), 'the child must run with the shim as cwd');
  assert.deepStrictEqual(data.argv.slice(0, 6), [
    '--no-skills',
    '--no-prompt-templates',
    '--no-themes',
    '--extension',
    SKILL_PI_ROOT,
    '--session',
  ]);
  assert.strictEqual(data.argv[6], 'test123', 'pi flags must be forwarded verbatim after the owned args');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('happy path trust-first-run: fresh run auto-accepts, trust.json keyed by shim realpath, second run no prompt', async () => {
  const home = tmpdir('home-trust');
  const root = makeGitRoot(tmpdir('root-trust'), 'main.js');
  const rootName = path.basename(root);
  const env = makeEnv(home);
  const trustFile = path.join(home, 'trust.json');
  const shim = shimDirFor(home, rootName);

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0, 'auto-accept run must exit 0');
  assert.ok(fs.existsSync(trustFile), 'trust.json must be written after an accepted trust prompt');
  const trust = JSON.parse(fs.readFileSync(trustFile, 'utf8'));
  const expectedKey = fs.realpathSync(shim);
  assert.ok(expectedKey in trust, 'trust.json must have a key equal to the shim realpath, got keys: ' + JSON.stringify(Object.keys(trust)));
  assert.strictEqual(trust[expectedKey].trusted, true);
  assert.ok(!Number.isNaN(Date.parse(trust[expectedKey].since)), 'since must be an ISO timestamp');

  const home2 = home;
  const res2 = await launch([], { cwd: root, env });
  assert.strictEqual(res2.code, 0, 'second run must exit 0');

  const probe = stubPiBin(
    home2,
    [
      'process.stdout.write("skill-pi-trust-prompt")',
      'process.exit(0)',
    ]
  );
  const env3 = makeEnv(home2);
  env3.SKILL_PI_PI_BIN = probe;
  const res3 = await launch([], { cwd: root, env3 });
  assert.strictEqual(res3.code, 0);
  assert.ok(res3.stdout.includes('skill-pi-trust-prompt'), 'the child ran on the trusted second run');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});

test('happy path trust-first-run: interactive prompt accepts and persists, decline exits without spawn', async () => {
  const home = tmpdir('home-trust-interactive');
  const root = makeGitRoot(tmpdir('root-trust-interactive'), 'main.js');
  const rootName = path.basename(root);
  const bin = stubPiBin(
    home,
    [
      'process.stdout.write("pi-ran")',
      'process.exit(0)',
    ]
  );
  const env = {
    ...process.env,
    SKILL_PI_AGENT_DIR: home,
    SKILL_PI_PI_BIN: bin,
  };
  const trustFile = path.join(home, 'trust.json');
  const shim = shimDirFor(home, rootName);

  const res = await launch([], { cwd: root, env });
  assert.strictEqual(res.code, 0, 'accepted run must exit 0');
  assert.ok(res.stdout.includes('pi-ran'), 'pi must run after trust is accepted');
  assert.ok(fs.existsSync(trustFile), 'trust.json must be written after an accepted interactive prompt');
  const trust = JSON.parse(fs.readFileSync(trustFile, 'utf8'));
  assert.ok(fs.realpathSync(shim) in trust, 'trust key must be the shim realpath');

  const home2 = tmpdir('home-trust-decline');
  const bin2 = stubPiBin(home2, ['process.stdout.write("pi-ran")', 'process.exit(0)']);
  const env2 = {
    ...process.env,
    SKILL_PI_AGENT_DIR: home2,
    SKILL_PI_PI_BIN: bin2,
  };
  const res2 = await launch([], { cwd: root, env: env2, input: 'n' });
  assert.ok(res2.code !== 0, 'declining trust must exit non-zero');
  assert.ok(!res2.stdout.includes('pi-ran'), 'pi must not run when trust is declined');
  assert.ok(!fs.existsSync(path.join(home2, 'trust.json')), 'no trust entry may be written on decline');

  fs.rmSync(home, { recursive: true, force: true });
  fs.rmSync(home2, { recursive: true, force: true });
  fs.rmSync(root, { recursive: true, force: true });
});
