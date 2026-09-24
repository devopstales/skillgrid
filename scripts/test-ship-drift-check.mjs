#!/usr/bin/env node
/**
 * test-ship-drift-check.mjs — fixture-driven tests for ship-drift-check.mjs.
 *
 * Creates temp git repos to test drift detection.
 *
 * Usage: node scripts/test-ship-drift-check.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execSync, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const DRIFT = join(HERE, "ship-drift-check.mjs");

let passed = 0;
let failed = 0;
const failures = [];

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

function git(repo, ...args) {
  return execSync(`git ${args.join(" ")}`, { cwd: repo, encoding: "utf8", stdio: "pipe" });
}

function gitCommit(repo, msg) {
  return execSync(`git commit --no-verify -m ${JSON.stringify(msg)}`, { cwd: repo, encoding: "utf8", stdio: "pipe" });
}

function makeRepo() {
  const root = mkdtempSync(join(tmpdir(), "sgdrift-"));
  git(root, "init");
  git(root, "config", "user.email", "test@test.com");
  git(root, "config", "user.name", "Test");
  // Base commit
  writeFileSync(join(root, "base.txt"), "base\n");
  git(root, "add", ".");
  gitCommit(root, "chore: base");
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function runDrift(args) {
  const r = spawnSync("node", [DRIFT, ...args], { encoding: "utf8" });
  return { code: r.status ?? -1, stdout: r.stdout ?? "", stderr: r.stderr ?? "" };
}

function run() {
  console.log("ship-drift-check.mjs tests\n");

  // 1. Clean repo (no changes after base) → exit 0, DRIFT: none
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    const r = runDrift(["check", base, p.root]);
    assert("clean: exit 0", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    assert("clean: DRIFT: none", r.stdout.includes("DRIFT: none"), r.stdout);
    assert("clean: SCOPE: COMPLETE (diff ran)", /^SCOPE: COMPLETE$/m.test(r.stdout), r.stdout);
    p.cleanup();
  }

  // 1b. Bad base ref → git diff fails → empty set is UNSCOPED, not a clean bill
  {
    const p = makeRepo();
    const r = runDrift(["check", "nonexistent-ref-xyz", p.root]);
    assert("bad-base: exit 0 (no files to drift on)", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    assert("bad-base: SCOPE: UNSCOPED (diff did not run)", /^SCOPE: UNSCOPED$/m.test(r.stdout), r.stdout);
    p.cleanup();
  }

  // 2. Changed file in anticipated set → exit 0 (no drift)
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    mkdirSync(join(p.root, "src"), { recursive: true });
    writeFileSync(join(p.root, "src", "foo.go"), "package main\n");
    git(p.root, "add", ".");
    gitCommit(p.root, "add foo");
    const r = runDrift(["check", base, "--anticipated", "src/", p.root]);
    assert("anticipated: exit 0", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    assert("anticipated: DRIFT: none", r.stdout.includes("DRIFT: none"), r.stdout);
    p.cleanup();
  }

  // 3. Changed file NOT in anticipated set → exit 1, drift
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    writeFileSync(join(p.root, "unexpected.txt"), "drift\n");
    git(p.root, "add", ".");
    gitCommit(p.root, "unexpected");
    const r = runDrift(["check", base, "--anticipated", "src/", p.root]);
    assert("drift: exit 1", r.code === 1, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    assert("drift: DRIFT DETECTED", r.stdout.includes("DRIFT DETECTED"), r.stdout);
    assert("drift: names file", r.stdout.includes("unexpected.txt"), r.stdout);
    p.cleanup();
  }

  // 4. Lock file excluded by default → no drift
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    writeFileSync(join(p.root, "package-lock.json"), "{}");
    git(p.root, "add", ".");
    gitCommit(p.root, "lock");
    const r = runDrift(["check", base, "--anticipated", "src/", p.root]);
    assert("lockfile-excluded: exit 0", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    p.cleanup();
  }

  // 5. Custom exclude pattern
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    writeFileSync(join(p.root, "generated.out"), "data");
    git(p.root, "add", ".");
    gitCommit(p.root, "gen");
    const r = runDrift(["check", base, "--anticipated", "src/", "--exclude", "*.out", p.root]);
    assert("custom-exclude: exit 0", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    p.cleanup();
  }

  // 6. Multiple anticipated dirs
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    mkdirSync(join(p.root, "src"), { recursive: true });
    mkdirSync(join(p.root, "lib"), { recursive: true });
    writeFileSync(join(p.root, "src", "a.go"), "a\n");
    writeFileSync(join(p.root, "lib", "b.go"), "b\n");
    git(p.root, "add", ".");
    gitCommit(p.root, "two dirs");
    const r = runDrift(["check", base, "--anticipated", "src/", "lib/", p.root]);
    assert("multi-anticipated: exit 0", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    p.cleanup();
  }

  // 7. No anticipated set → any change is drift
  {
    const p = makeRepo();
    const base = git(p.root, "rev-parse", "HEAD").trim();
    writeFileSync(join(p.root, "anything.txt"), "x");
    git(p.root, "add", ".");
    gitCommit(p.root, "anything");
    const r = runDrift(["check", base, p.root]);
    assert("no-anticipated: exit 1", r.code === 1, `got ${r.code}: ${r.stdout} ${r.stderr}`);
    assert("no-anticipated: names file", r.stdout.includes("anything.txt"), r.stdout);
    p.cleanup();
  }

  // 8. Deleted file in anticipated set → no drift
  {
    const p = makeRepo();
    mkdirSync(join(p.root, "src"), { recursive: true });
    writeFileSync(join(p.root, "src", "old.go"), "old\n");
    git(p.root, "add", ".");
    gitCommit(p.root, "add old");
    const base = git(p.root, "rev-parse", "HEAD").trim();
    execSync(`rm ${join(p.root, "src", "old.go")}`, { stdio: "pipe" });
    git(p.root, "add", ".");
    gitCommit(p.root, "remove old");
    const r = runDrift(["check", base, "--anticipated", "src/", p.root]);
    assert("deleted-anticipated: exit 0", r.code === 0, `got ${r.code}: ${r.stdout} ${r.stderr}`);
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
