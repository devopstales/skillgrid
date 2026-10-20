#!/usr/bin/env node
/**
 * test-okf-lint.mjs — fixture-driven tests for okf-lint.mjs.
 *
 * Runs the lint over the committed fixture bundles (test/fixtures/okf-clean,
 * test/fixtures/okf-drift) plus temp-project trees for layout and telemetry
 * cases, and asserts exit codes + finding-class names + JSONL events.
 *
 * Usage: node scripts/test-okf-lint.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync, readFileSync, existsSync, readdirSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = join(HERE, "..");
const LINT = join(ROOT, ".agents", "skills", "verification", "qa", "scripts", "okf-lint.mjs");
const CLEAN = join(ROOT, "test", "fixtures", "okf-clean");
const DRIFT = join(ROOT, "test", "fixtures", "okf-drift");

let passed = 0;
let failed = 0;
const failures = [];

function runLint(root, env = {}) {
  try {
    const stdout = execFileSync("node", [LINT, root], {
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env, ...env },
    });
    return { code: 0, stdout, stderr: "" };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}

function readSinkLines(sink) {
  if (!existsSync(sink)) return [];
  return readFileSync(sink, "utf8")
    .split("\n")
    .filter(Boolean)
    .map((l) => JSON.parse(l));
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

// --- Clean bundle: all nine checks pass ---

{
  const env = { OKF_METRICS_SINK: join(tmpdir(), "okf-t1-metrics.jsonl") };
  const r = runLint(CLEAN, env);
  assert("clean: exit 0", r.code === 0, `got ${r.code}: ${r.stdout}\n${r.stderr}`);
  assert("clean: DRIFT: clean", r.stdout.includes("DRIFT: clean"), r.stdout);
  assert("clean: SCOPE: COMPLETE", /^SCOPE: COMPLETE$/m.test(r.stdout), r.stdout);
}

// --- Drifted bundle: all four finding classes named ---

{
  const env = { OKF_METRICS_SINK: join(tmpdir(), "okf-t2-metrics.jsonl") };
  const r = runLint(DRIFT, env);
  assert("drift: exit 1", r.code === 1, `got ${r.code}: ${r.stdout}\n${r.stderr}`);
  assert("drift: names frontmatter-invalid", r.stdout.includes("frontmatter-invalid"), r.stdout);
  assert("drift: names index-incomplete", r.stdout.includes("index-incomplete"), r.stdout);
  assert("drift: names link-broken", r.stdout.includes("link-broken"), r.stdout);
  assert("drift: names provenance-missing", r.stdout.includes("provenance-missing"), r.stdout);
  assert("drift: SCOPE: COMPLETE", /^SCOPE: COMPLETE$/m.test(r.stdout), r.stdout);
}

// --- Read-only: the lint never writes into the fixture dirs ---

{
  const env = { OKF_METRICS_SINK: join(tmpdir(), "okf-t3-metrics.jsonl") };
  const before = snapshot(CLEAN);
  runLint(CLEAN, env);
  runLint(DRIFT, env);
  assert("read-only: clean fixture unmodified", snapshot(CLEAN) === before, "fixture mtime changed");
}

function snapshot(dir) {
  const entries = [];
  (function walk(d) {
    for (const entry of readdirSync(d).sort()) {
      const p = join(d, entry);
      const st = statSync(p);
      if (st.isDirectory()) walk(p);
      else entries.push(`${p}:${st.mtimeMs}`);
    }
  })(dir);
  return entries.join("\n");
}

// --- Telemetry: one JSONL line per run, false_drift_rate computed ---

{
  const sink = join(mkdtempSync(join(tmpdir(), "okf-t4-")), "okf-metrics.jsonl");
  const env = { OKF_METRICS_SINK: sink };
  runLint(DRIFT, env);
  assert("telemetry: first run appends one line", countLines(sink) === 1, `lines=${countLines(sink)}`);
  const first = readSinkLines(sink)[0];
  assert("telemetry: event is lint", first?.event === "lint", JSON.stringify(first));
  assert("telemetry: drift is a number", typeof first?.drift === "number", JSON.stringify(first));
  assert("telemetry: findings is an array", Array.isArray(first?.findings), JSON.stringify(first));
  assert("telemetry: false_drift_rate numeric", typeof first?.false_drift_rate === "number", JSON.stringify(first));
  assert("telemetry: first run rate is 0.0", first?.false_drift_rate === 0.0, JSON.stringify(first));
  assert("telemetry: iso-utc timestamp", /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(first?.at ?? ""), JSON.stringify(first));

  runLint(DRIFT, env);
  assert("telemetry: second run appends (2 lines)", countLines(sink) === 2, `lines=${countLines(sink)}`);
  const lines = readSinkLines(sink);
  // Identical findings both runs -> every prior finding present again -> rate 0.
  assert("telemetry: repeated findings rate 0.0", lines[1]?.false_drift_rate === 0.0, JSON.stringify(lines[1]));

  // Different findings on the second run (clean bundle, zero drift) ->
  // every prior finding absent -> rate 1.0.
  runLint(CLEAN, env);
  const l3 = readSinkLines(sink)[2] ?? {};
  assert("telemetry: prior findings gone -> rate 1.0", l3.false_drift_rate === 1.0, JSON.stringify(l3));
}

function countLines(p) {
  if (!existsSync(p)) return 0;
  return readFileSync(p, "utf8").split("\n").filter(Boolean).length;
}

// --- Telemetry: default sink is <bundle root>/.skillgrid/okf-metrics.jsonl,
//     auto-created, and the write is the only side effect ---

{
  const root = mkdtempSync(join(tmpdir(), "okf-t5-"));
  mkdirSync(join(root, ".skillgrid", "knowledge"), { recursive: true });
  // Minimal bundle: a root index.md is enough for a scoped, clean run.
  writeFileSync(join(root, ".skillgrid", "knowledge", "index.md"), [
    "---",
    'okf_version: "0.2"',
    "title: minimal",
    "status: stable",
    "generated:",
    "  by: skillgrid/fixture",
    "  at: 2026-10-18T00:00:00Z",
    "---",
    "",
    "# minimal",
  ].join("\n"));
  const r = runLint(root);
  assert("default-sink: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  const sink = join(root, ".skillgrid", "okf-metrics.jsonl");
  assert("default-sink: file created under .skillgrid/", existsSync(sink));
  assert("default-sink: one line", existsSync(sink) && countLines(sink) === 1);
  rmSync(root, { recursive: true, force: true });
}

// --- Telemetry write failure is non-fatal ---

{
  const sink = join(tmpdir(), "okf-t6-dir", "sub", "metrics.jsonl");
  // Parent is a FILE, not a directory -> mkdir recursive fails -> write fails.
  const t6 = mkdtempSync(join(tmpdir(), "okf-t6-"));
  const blocker = join(t6, "blocker");
  writeFileSync(blocker, "i am a file");
  const r = runLint(DRIFT, { OKF_METRICS_SINK: join(blocker, "metrics.jsonl") });
  assert("telemetry-failure: exit code unchanged (1)", r.code === 1, `got ${r.code}`);
  assert("telemetry-failure: warns to stderr", /okf-metrics|telemetry/i.test(r.stderr), r.stderr);
  rmSync(t6, { recursive: true, force: true });
}

// --- Layout: neither knowledge/ nor artifacts/ => UNSCOPED, still exit 0 ---

{
  const root = mkdtempSync(join(tmpdir(), "okf-t7-"));
  mkdirSync(join(root, ".skillgrid"), { recursive: true });
  const env = { OKF_METRICS_SINK: join(root, "metrics.jsonl") };
  const r = runLint(root, env);
  assert("layout-unscopecd: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("layout-unscopecd: SCOPE: UNSCOPED", /^SCOPE: UNSCOPED$/m.test(r.stdout), r.stdout);
  assert("layout-unscopecd: not a clean bill", !r.stdout.includes("DRIFT: clean"), r.stdout);
  rmSync(root, { recursive: true, force: true });
}

// --- Layout: pre-R6 artifacts/ fallback is linted ---

{
  const root = mkdtempSync(join(tmpdir(), "okf-t8-"));
  const art = join(root, ".skillgrid", "artifacts");
  mkdirSync(art, { recursive: true });
  // A stable concept with broken YAML frontmatter -> the fallback root must be checked.
  writeFileSync(
    join(art, "04-adr-0001-x.md"),
    ["---", "type: TechnicalDecision", "status: stable", "generated: { by: x", "---", "", "# x"].join("\n"),
  );
  writeFileSync(
    join(art, "README.md"),
    ["---", 'okf_version: "0.2"', "---", "", "# catalog", "", "- [04-adr-0001-x.md](04-adr-0001-x.md) — x"].join("\n"),
  );
  const env = { OKF_METRICS_SINK: join(root, "metrics.jsonl") };
  const r = runLint(root, env);
  assert("layout-artifacts: exit 1 (frontmatter drift in fallback)", r.code === 1, `got ${r.code}: ${r.stdout}\n${r.stderr}`);
  assert("layout-artifacts: names frontmatter-invalid", r.stdout.includes("frontmatter-invalid"), r.stdout);
  assert("layout-artifacts: SCOPE: COMPLETE", /^SCOPE: COMPLETE$/m.test(r.stdout), r.stdout);
  rmSync(root, { recursive: true, force: true });
}

// --- Usage: missing arg / unreadable bundle -> exit 2 ---

{
  try {
    execFileSync("node", [LINT], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    assert("usage: no arg exits 2", false, "no exit");
  } catch (e) {
    assert("usage: no arg exits 2", e.status === 2, `got ${e.status}`);
  }
}

// Summary
console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) {
  console.log("Failures:");
  for (const f of failures) console.log(`  - ${f}`);
}
process.exit(failed > 0 ? 1 : 0);
