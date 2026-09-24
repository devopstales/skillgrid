#!/usr/bin/env node
/**
 * test-state-lock.mjs — fixture-driven tests for state-lock.mjs.
 *
 * Usage: node scripts/test-state-lock.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync, readFileSync, existsSync, utimesSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const LOCK = join(HERE, "state-lock.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function makeProject() {
  const root = mkdtempSync(join(tmpdir(), "sglock-"));
  const sg = join(root, ".skillgrid");
  mkdirSync(sg, { recursive: true });
  return { root, lockPath: join(sg, ".lock"), cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function runLock(args) {
  const r = spawnSync("node", [LOCK, ...args], { encoding: "utf8" });
  return { code: r.status ?? -1, stdout: r.stdout ?? "", stderr: r.stderr ?? "" };
}

function assert(name, cond, detail = "") {
  if (cond) {
    passed++;
    console.log(`  ok   ${name}`);
  } else {
    failed++;
    failures.push(name);
    console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`);
  }
}

function setLockAge(p, hours) {
  // Backdate both the file mtime AND the acquired_at field in the JSON
  const lock = JSON.parse(readFileSync(p.lockPath, "utf8"));
  lock.acquired_at = new Date(Date.now() - hours * 3600 * 1000).toISOString();
  writeFileSync(p.lockPath, JSON.stringify(lock));
}

function run() {
  console.log("state-lock.mjs tests\n");

  // 1. Acquire on clean tree → creates .lock, exit 0
  {
    const p = makeProject();
    const r = runLock(["acquire", "2026-09-24-test", "sess-1", p.root]);
    assert("acquire-clean: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("acquire-clean: lock exists", existsSync(p.lockPath));
    const lock = JSON.parse(readFileSync(p.lockPath, "utf8"));
    assert("acquire-clean: lock has change", lock.change === "2026-09-24-test", JSON.stringify(lock));
    assert("acquire-clean: lock has session", lock.session_id === "sess-1", JSON.stringify(lock));
    p.cleanup();
  }

  // 2. Acquire with active lock, same change → exit 0 (idempotent)
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-24-test", session_id: "sess-1", acquired_at: new Date().toISOString() }));
    const r = runLock(["acquire", "2026-09-24-test", "sess-2", p.root]);
    assert("acquire-same: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 3. Acquire with active lock, different change → exit 1, conflict
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-01-other", session_id: "sess-9", acquired_at: new Date().toISOString() }));
    const r = runLock(["acquire", "2026-09-24-test", "sess-1", p.root]);
    assert("acquire-different: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("acquire-different: conflict message", r.stderr.includes("CONFLICT"), r.stderr);
    p.cleanup();
  }

  // 4. Acquire with stale lock (>2h) → overwrites, exit 0, stale warning
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-01-old", session_id: "sess-0", acquired_at: new Date().toISOString() }));
    setLockAge(p, 3);
    const r = runLock(["acquire", "2026-09-24-test", "sess-1", p.root]);
    assert("acquire-stale: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("acquire-stale: stale warning", r.stderr.includes("STALE"), r.stderr);
    const lock = JSON.parse(readFileSync(p.lockPath, "utf8"));
    assert("acquire-stale: overwritten", lock.change === "2026-09-24-test", JSON.stringify(lock));
    p.cleanup();
  }

  // 5. Release matching → removes .lock, exit 0
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-24-test", session_id: "sess-1", acquired_at: new Date().toISOString() }));
    const r = runLock(["release", "2026-09-24-test", p.root]);
    assert("release-matching: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("release-matching: lock removed", !existsSync(p.lockPath));
    p.cleanup();
  }

  // 6. Release non-matching → no-op, exit 0
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-01-other", session_id: "sess-9", acquired_at: new Date().toISOString() }));
    const r = runLock(["release", "2026-09-24-test", p.root]);
    assert("release-nonmatching: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("release-nonmatching: lock still exists", existsSync(p.lockPath));
    p.cleanup();
  }

  // 7. Release with no lock → no-op, exit 0
  {
    const p = makeProject();
    const r = runLock(["release", "2026-09-24-test", p.root]);
    assert("release-nolock: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 8. Check: no lock → exit 0
  {
    const p = makeProject();
    const r = runLock(["check", p.root]);
    assert("check-nolock: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 9. Check: active lock → exit 1
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-24-test", session_id: "sess-1", acquired_at: new Date().toISOString() }));
    const r = runLock(["check", p.root]);
    assert("check-active: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("check-active: LOCKED message", r.stdout.includes("LOCKED"), r.stdout);
    p.cleanup();
  }

  // 10. Check: stale lock → exit 2
  {
    const p = makeProject();
    writeFileSync(p.lockPath, JSON.stringify({ change: "2026-09-24-test", session_id: "sess-1", acquired_at: new Date().toISOString() }));
    setLockAge(p, 3);
    const r = runLock(["check", p.root]);
    assert("check-stale: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
    assert("check-stale: STALE message", r.stdout.includes("STALE"), r.stdout);
    p.cleanup();
  }

  // Summary
  console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
  if (failures.length) {
    console.log("Failures:");
    for (const f of failures) console.log(`  - ${f}`);
  }
  process.exit(failed > 0 ? 1 : 0);
}

run();
