#!/usr/bin/env node
/**
 * test-status-banner.mjs — fixture-driven tests for scripts/status-banner.mjs.
 *
 * Usage: node scripts/test-status-banner.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const BANNER = join(HERE, "status-banner.mjs");

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

function makeProject({ specs = {} } = {}) {
  const root = mkdtempSync(join(tmpdir(), "sgbanner-"));
  mkdirSync(join(root, ".skillgrid", "specs"), { recursive: true });
  mkdirSync(join(root, ".skillgrid", "sdd"), { recursive: true });
  for (const [dir, files] of Object.entries(specs)) {
    const d = join(root, ".skillgrid", "specs", dir);
    mkdirSync(d, { recursive: true });
    for (const [f, c] of Object.entries(files)) writeFileSync(join(d, f), c);
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(root, ...args) {
  try {
    const stdout = execFileSync("node", [BANNER, ...args, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}

const BANNER_OK = "> **STATUS:** `draft` (2026-01-01)";
const BANNER_SLICED = "> **STATUS:** `sliced` (2026-01-01)";

// 1. Clean: briefing with valid draft banner at spec phase
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\n${BANNER_OK}\n` } } });
  const r = run(p.root, "check");
  assert("clean-spec: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("clean-spec: clean msg", r.stdout.includes("clean"), r.stdout);
  p.cleanup();
}

// 2. Missing briefing banner → exit 1
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\nno banner here\n` } } });
  const r = run(p.root, "check");
  assert("missing-banner: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("missing-banner: names briefing.md", r.stdout.includes("briefing.md"), r.stdout);
  p.cleanup();
}

// 3. Invalid value → exit 1
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\n> **STATUS:** \`bogus\` (2026-01-01)\n` } } });
  const r = run(p.root, "check");
  assert("bad-value: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("bad-value: names value", r.stdout.includes("bogus"), r.stdout);
  p.cleanup();
}

// 4. Phase inconsistency: briefing says `sliced` but phase is spec (no blueprint) → exit 1
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\n> **STATUS:** \`sliced\` (2026-01-01)\n` } } });
  const r = run(p.root, "check");
  assert("inconsistent: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("inconsistent: names phase", r.stdout.includes("derived phase"), r.stdout);
  p.cleanup();
}

// 5. tasks.md missing banner → exit 1
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\n${BANNER_OK}\n`, "tasks.md": `# Tasks\n\nno banner\n` } } });
  const r = run(p.root, "check");
  assert("tasks-missing: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("tasks-missing: names tasks.md", r.stdout.includes("tasks.md"), r.stdout);
  p.cleanup();
}

// 6. write: briefing at spec phase → banner set to draft
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\n> **STATUS:** \`stale\` (2020-01-01)\n` } } });
  const r = run(p.root, "write", "c1");
  assert("write: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  const after = readFileSync(join(p.root, ".skillgrid", "specs", "c1", "briefing.md"), "utf8");
  assert("write: banner now draft", /\*\*STATUS:\*\* `draft`/.test(after), after);
  p.cleanup();
}

// 7. write: execution phase (tasks [x] + ledger) → banner in-progress
{
  const p = makeProject({
    specs: { "c1": { "briefing.md": `# T\n`, "blueprint.md": `# B\n`, "tasks.md": `# Tasks\n\n- [x] t1\n` } },
  });
  mkdirSync(join(p.root, ".skillgrid", "sdd", "c1"), { recursive: true });
  writeFileSync(join(p.root, ".skillgrid", "sdd", "c1", "progress.md"), "- [x] t1\n");
  const r = run(p.root, "write", "c1");
  assert("write-exec: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  const after = readFileSync(join(p.root, ".skillgrid", "specs", "c1", "tasks.md"), "utf8");
  assert("write-exec: tasks banner in-progress", /\*\*STATUS:\*\* `in-progress`/.test(after), after);
  p.cleanup();
}

// 8. determinism: run check twice, identical
{
  const p = makeProject({ specs: { "c1": { "briefing.md": `# T\n\nno banner\n` } } });
  const a = run(p.root, "check").stdout;
  const b = run(p.root, "check").stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

// Summary
console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) {
  console.log("Failures:");
  for (const f of failures) console.log(`  - ${f}`);
}
process.exit(failed > 0 ? 1 : 0);
