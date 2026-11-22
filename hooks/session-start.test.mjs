import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { chmodSync, copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import os from 'node:os';
import path from 'node:path';

const HERE = fileURLToPath(new URL('.', import.meta.url));
const REPO_ROOT = path.resolve(HERE, '..');
const HOOK = path.join(REPO_ROOT, 'hooks', 'session-start.js');

function runHook(cwd, stdin, hook = HOOK) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [hook], { cwd, stdio: ['pipe', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => { stdout += d; });
    child.stderr.on('data', (d) => { stderr += d; });
    child.on('close', (code) => resolve({ code, stdout, stderr }));
    if (stdin === undefined) {
      child.stdin.end();
    } else {
      child.stdin.write(JSON.stringify(stdin));
      child.stdin.end();
    }
  });
}

function runMain(cwd) {
  return runHook(cwd, { session_id: 'test-session', cwd, hook_event_name: 'SessionStart', source: 'startup' });
}

test('in-repo main session: full router content injected (additionalContext), exit 0', async () => {
  const { code, stdout, stderr } = await runMain(REPO_ROOT);
  assert.equal(code, 0);
  const payload = JSON.parse(stdout);
  assert.equal(payload.hookSpecificOutput.hookEventName, 'SessionStart');
  const ctx = payload.hookSpecificOutput.additionalContext;
  assert.ok(ctx.includes('skillgrid loaded'));
  assert.ok(ctx.includes('Route through the using-skillgrid flow'));
  assert.ok(ctx.includes('using-skillgrid'));
  assert.equal(stderr, '');
});

test('non-skillgrid cwd (/tmp): no injection, INFO note on stderr, exit 0', async () => {
  const { code, stdout, stderr } = await runMain('/tmp');
  assert.equal(code, 0);
  assert.equal(stdout, '');
  assert.ok(stderr.includes('no skillgrid project detected'));
});

test('unreadable skillgrid config: treated as no project, no injection, exit 0', async () => {
  const dir = mkdtempSync('/tmp/skillgrid-session-start-test-');
  try {
    mkdirSync(path.join(dir, '.skillgrid'), { recursive: true });
    const config = path.join(dir, '.skillgrid', 'config.yaml');
    writeFileSync(config, 'schema: skillgrid/v1\n');
    chmodSync(config, 0o000);
    const { code, stdout, stderr } = await runMain(dir);
    assert.equal(code, 0);
    assert.equal(stdout, '');
    assert.ok(stderr.includes('skillgrid'));
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('child session (agent_kind sub) in skillgrid project: no injection, exit 0', async () => {
  const { code, stdout, stderr } = await runHook(REPO_ROOT, {
    session_id: 'child-session',
    cwd: REPO_ROOT,
    hook_event_name: 'SessionStart',
    source: 'startup',
    agent_kind: 'sub',
  });
  assert.equal(code, 0);
  assert.equal(stdout, '');
  assert.ok(stderr.includes('not a main-agent session'));
});

test('router missing at install location: no injection, not-found INFO note, exit 0', async () => {
  const dir = mkdtempSync('/tmp/skillgrid-session-start-test-');
  try {
    // The hook resolves ROUTER relative to its install location, so install a
    // copy in a bare tree where the router lookup misses.
    const hookCopy = path.join(dir, 'hooks', 'session-start.js');
    mkdirSync(path.join(dir, 'hooks'), { recursive: true });
    copyFileSync(HOOK, hookCopy);
    const proj = path.join(dir, 'proj');
    mkdirSync(path.join(proj, '.skillgrid'), { recursive: true });
    writeFileSync(path.join(proj, '.skillgrid', 'config.yaml'), 'schema: skillgrid/v1\n');
    const { code, stdout, stderr } = await runHook(proj, {
      session_id: 'router-missing-session',
      cwd: proj,
      hook_event_name: 'SessionStart',
      source: 'startup',
    }, hookCopy);
    assert.equal(code, 0);
    assert.equal(stdout, '');
    assert.ok(stderr.includes('using-skillgrid router not found'));
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('payload cwd wins over process.cwd(): non-project spawn cwd still injects', async () => {
  const dir = mkdtempSync('/tmp/skillgrid-session-start-test-');
  try {
    mkdirSync(path.join(dir, '.skillgrid'), { recursive: true });
    writeFileSync(path.join(dir, '.skillgrid', 'config.yaml'), 'schema: skillgrid/v1\n');
    // Spawn from a non-project dir; the stdin payload's cwd points at the project.
    const { code, stdout, stderr } = await runHook(path.join(os.tmpdir()), {
      session_id: 'payload-cwd-session',
      cwd: dir,
      hook_event_name: 'SessionStart',
      source: 'startup',
    });
    assert.equal(code, 0);
    const payload = JSON.parse(stdout);
    assert.equal(payload.hookSpecificOutput.hookEventName, 'SessionStart');
    const ctx = payload.hookSpecificOutput.additionalContext;
    assert.ok(ctx.includes('skillgrid loaded'));
    assert.ok(ctx.includes('using-skillgrid'));
    assert.equal(stderr, '');
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
