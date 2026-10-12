#!/usr/bin/env node
'use strict';

const { spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const RESPAWN_CAP = 10;
const SKILL_PI_ROOT = path.join(__dirname, '..');

function agentDir() {
  return process.env.SKILL_PI_AGENT_DIR || path.join(os.homedir(), '.skill-pi');
}

function piBin() {
  if (process.env.SKILL_PI_PI_BIN) return process.env.SKILL_PI_PI_BIN;
  // Pinned runtime is a bundled dependency: after `npm i -g skill-pi`, pi lives
  // under this package's own node_modules, not the global root. Resolve it
  // there first (omegon-pi pattern), then fall back to the global root.
  const bundled = path.join(
    SKILL_PI_ROOT, 'node_modules', '@earendil-works', 'pi-coding-agent', 'dist', 'bundle', 'cli.js'
  );
  if (fs.existsSync(bundled)) return bundled;
  try {
    const { execFileSync } = require('node:child_process');
    const globalRoot = execFileSync('npm', ['root', '-g'], { encoding: 'utf8' }).trim();
    const globalCandidate = path.join(globalRoot, '@earendil-works', 'pi-coding-agent', 'dist', 'bundle', 'cli.js');
    if (fs.existsSync(globalCandidate)) return globalCandidate;
  } catch (_) {
    // fall through to a bare pi on PATH
  }
  return 'pi';
}

function readPinnedVersion() {
  return fs.readFileSync(path.join(SKILL_PI_ROOT, 'pi-version.txt'), 'utf8').trim();
}

function whereJson(realRoot) {
  const root = realRoot || resolveRoot(null);
  return {
    configDir: path.join(root, '.skill-pi'),
    agentDir: agentDir(),
    shimDir: path.join(agentDir(), 'cwd', path.basename(root)),
    skillPiRoot: SKILL_PI_ROOT,
    piVersion: readPinnedVersion(),
    piBin: piBin(),
    projectRoot: root,
    globalConfig: agentDir(),
  };
}

function resolveRoot(explicit) {
  if (explicit) {
    const abs = path.resolve(explicit);
    if (!fs.existsSync(abs) || !fs.statSync(abs).isDirectory()) {
      process.stderr.write('skill-pi: root does not exist: ' + abs + '\n');
      process.exit(1);
    }
    if (!fs.existsSync(path.join(abs, '.git'))) {
      process.stderr.write('skill-pi: no .git found in root: ' + abs + '\n');
      process.exit(1);
    }
    return fs.realpathSync(abs);
  }
  let dir = process.cwd();
  for (;;) {
    if (fs.existsSync(path.join(dir, '.git'))) {
      return fs.realpathSync(dir);
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      process.stderr.write('skill-pi: no .git found above ' + process.cwd() + '\n');
      process.exit(1);
    }
    dir = parent;
  }
}

function buildShim(realRoot) {
  const shimDir = path.join(agentDir(), 'cwd', path.basename(realRoot));
  fs.rmSync(shimDir, { recursive: true, force: true });
  fs.mkdirSync(shimDir, { recursive: true });
  for (const entry of fs.readdirSync(realRoot)) {
    if (entry === 'node_modules') continue;
    fs.symlinkSync(path.join(realRoot, entry), path.join(shimDir, entry));
  }
  fs.writeFileSync(
    path.join(shimDir, 'package.json'),
    JSON.stringify({ name: 'skill-pi-cwd', version: '0.0.0', piConfig: { configDir: '.skill-pi' } }, null, 2) + '\n'
  );
  return shimDir;
}

function runInit() {
  const root = resolveRoot(null);
  fs.mkdirSync(path.join(root, '.skill-pi'), { recursive: true });
}

function runSync() {
  process.stdout.write('sync complete\n');
}

function parseArgs(argv) {
  const rest = [];
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--root') {
      const val = argv[++i];
      if (!val) {
        process.stderr.write('skill-pi: --root requires a path\n');
        process.exit(1);
      }
      rest.push(a, val);
    } else if (a === '--where') {
      process.stdout.write(JSON.stringify(whereJson(null)) + '\n');
      process.exit(0);
    } else if (a === 'version') {
      process.stdout.write(readPinnedVersion() + '\n');
      process.exit(0);
    } else if (a === 'init') {
      runInit();
      process.exit(0);
    } else if (a === 'sync') {
      runSync();
      process.exit(0);
    } else {
      rest.push(a);
    }
  }
  return { rest };
}

function trustFile() {
  return path.join(agentDir(), 'trust.json');
}

function readTrust() {
  try {
    return JSON.parse(fs.readFileSync(trustFile(), 'utf8'));
  } catch (e) {
    return {};
  }
}

function writeTrust(trust) {
  fs.mkdirSync(path.dirname(trustFile()), { recursive: true });
  const tmp = trustFile() + '.' + process.pid + '.tmp';
  fs.writeFileSync(tmp, JSON.stringify(trust, null, 2) + '\n');
  fs.renameSync(tmp, trustFile());
}

function recordTrust(shimDir) {
  const trust = readTrust();
  trust[fs.realpathSync(shimDir)] = true;
  writeTrust(trust);
}

function ensureTrusted(shimDir, onTrusted) {
  const key = fs.realpathSync(shimDir);
  const trust = readTrust();
  if (trust[key] === true) {
    onTrusted();
    return;
  }
  if (process.env.SKILL_PI_TRUST_AUTO === '1') {
    recordTrust(shimDir);
    onTrusted();
    return;
  }
  process.stdout.write('Trust ' + path.basename(shimDir) + '? (y/n)\n');
  let buf = '';
  let settled = false;
  let timer = null;
  const settle = (ans) => {
    if (settled) return;
    settled = true;
    if (timer) clearTimeout(timer);
    process.stdin.removeAllListeners();
    if (ans.trim().toLowerCase() === 'y' || ans.trim().toLowerCase() === 'yes') {
      recordTrust(shimDir);
      onTrusted();
    } else {
      process.stderr.write('skill-pi: not trusted; aborting\n');
      process.exit(1);
    }
  };
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', (d) => {
    buf += d;
    const nl = buf.indexOf('\n');
    if (nl !== -1) settle(buf.slice(0, nl));
    else if (buf.length && !timer) timer = setTimeout(() => settle(buf), 800);
  });
  process.stdin.on('end', () => settle(buf));
  process.stdin.on('error', () => settle(buf));
}

function spawnPi(argv, shimDir) {
  const owned = ['--no-skills', '--no-prompt-templates', '--no-themes', '--extension', SKILL_PI_ROOT];
  const child = spawn(piBin(), [...owned, ...argv], {
    cwd: shimDir,
    stdio: 'inherit',
    env: {
      ...process.env,
      // Point pi's getPackageDir() at skill-pi/ so its own package.json
      // (piConfig: name "skill-pi", configDir ".skill-pi") is the one pi reads.
      // This sets APP_NAME="skill-pi" and CONFIG_DIR_NAME=".skill-pi".
      PI_PACKAGE_DIR: SKILL_PI_ROOT,
      // pi reads its agent dir from `${APP_NAME}_CODING_AGENT_DIR` (=
      // SKILL_PI_CODING_AGENT_DIR once PI_PACKAGE_DIR names skill-pi).
      // Without it pi would default to ~/.skill-pi/agent (CONFIG_DIR_NAME/agent).
      SKILL_PI_CODING_AGENT_DIR: agentDir(),
      PI_SKIP_VERSION_CHECK: '1',
    },
  });
  child.on('error', (err) => {
    process.stderr.write('skill-pi: failed to spawn pi: ' + err.message + '\n');
    process.exit(1);
  });
  for (const sig of ['SIGINT', 'SIGTERM']) {
    process.on(sig, () => {
      if (!child.killed) child.kill(sig);
    });
  }
  return child;
}

function runSpawnLoop(argv, count) {
  const realRoot = resolveRoot(null);
  const shimDir = buildShim(realRoot);
  const child = spawnPi(argv, shimDir);
  child.on('exit', (code) => {
    if (code === 75 && count < RESPAWN_CAP) {
      runSpawnLoop(argv, count + 1);
    } else {
      process.exit(code === null ? 1 : code);
    }
  });
}

if (require.main === module) {
  const { rest } = parseArgs(process.argv.slice(2));
  const realRoot = resolveRoot(null);
  const shimDir = buildShim(realRoot);
  ensureTrusted(shimDir, () => runSpawnLoop(rest, 0));
}

module.exports = { buildShim, spawnPi, resolveRoot };
