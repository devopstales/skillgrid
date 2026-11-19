import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { chmodSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const HERE = fileURLToPath(new URL('.', import.meta.url));
const REPO_ROOT = path.resolve(HERE, '..');
const HOOK = path.join(REPO_ROOT, 'hooks', 'session-start.js');

function runHook(cwd, stdin) {
  return new Promise((resolve) => {
    const child = spawn(process.execPath, [HOOK], { cwd, stdio: ['pipe', 'pipe', 'pipe'] });
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

test('in-repo main session: full router content injected (IMPORTANT), exit 0', async () => {
  const { code, stdout, stderr } = await runMain(REPO_ROOT);
  assert.equal(code, 0);
  const payload = JSON.parse(stdout);
  assert.equal(payload.hookSpecificName, 'SessionStart');
  assert.equal(payload.priority, 'IMPORTANT');
  assert.ok(payload.message.includes('skillgrid loaded'));
  assert.ok(payload.message.includes('Route through the using-skillgrid flow'));
  assert.ok(payload.message.includes('using-skillgrid'));
  assert.equal(stderr, '');
});

test('non-skillgrid cwd (/tmp): no router injection, INFO note, exit 0', async () => {
  const { code, stdout, stderr } = await runMain('/tmp');
  assert.equal(code, 0);
  const payload = JSON.parse(stdout);
  assert.equal(payload.priority, 'INFO');
  assert.ok(!payload.message.includes('skillgrid loaded'));
  assert.equal(stderr, '');
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
    const payload = JSON.parse(stdout);
    assert.equal(payload.priority, 'INFO');
    assert.ok(!payload.message.includes('skillgrid loaded'));
    assert.equal(stderr, '');
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
  const payload = JSON.parse(stdout);
  assert.equal(payload.priority, 'INFO');
  assert.ok(!payload.message.includes('skillgrid loaded'));
  assert.equal(stderr, '');
});
