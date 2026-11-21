#!/usr/bin/env node
// test-trail.mjs — prove the hypothesis-trail state machine: init, append,
// list/open filters, the fix-attempt count, and the 3-fix architecture gate.
//
// Hermetic: builds throwaway trail dirs under mkdtemp; torn down on exit.
//
// Usage:
//   node .agents/skills/debug/scripts/test-trail.mjs          # run everything
//   node .agents/skills/debug/scripts/test-trail.mjs gate      # run only cases tagged gate
//
// Exit code: 0 if all pass, 1 if any fail.
import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync, appendFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const SCRIPTS_DIR = dirname(fileURLToPath(import.meta.url));
const TRAIL = join(SCRIPTS_DIR, "trail.mjs");
const NODE = process.execPath;
const FILTER = process.argv[2] || "";

let PASS = 0, FAIL = 0;
const RESULTS = [];

const TMP_ROOT = mkdtempSync(join(tmpdir(), "skillgrid-trail-"));
const D = join(TMP_ROOT, "debug", "2026-01-01-demo");

function run(args) {
  try {
    const out = execFileSync(NODE, [TRAIL, ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

function expectRC(name, want, got) {
  if (got === want) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (want rc=${want}, got rc=${got})`); }
}
function expectContains(name, haystack, needle) {
  if (haystack.includes(needle)) { PASS++; RESULTS.push(`PASS  ${name}`); }
  else { FAIL++; RESULTS.push(`FAIL  ${name} (missing: ${needle})`); }
}
function expectNotContains(name, haystack, needle) {
  if (haystack.includes(needle)) { FAIL++; RESULTS.push(`FAIL  ${name} (unexpected: ${needle})`); }
  else { PASS++; RESULTS.push(`PASS  ${name}`); }
}

// --- init -----------------------------------------------------------------------

function caseInit() {
  if (FILTER && FILTER !== "init") return;

  let r = run(["init", D]);
  expectRC("init: creates the trail", 0, r.rc);
  expectRC("init: state.md exists", 0, existsSync(join(D, "state.md")) ? 0 : 1);
  expectContains("init: header row present", readFileSync(join(D, "state.md"), "utf8"), "| hypothesis | source | evidence | verdict |");

  r = run(["init", D]);
  expectRC("init: idempotent on existing trail", 0, r.rc);
}

// --- append ---------------------------------------------------------------------

function caseAppend() {
  if (FILTER && FILTER !== "append") return;

  run(["init", D]);
  let r = run(["append", D, "config not loaded", "goldfish", "repro shows default used"]);
  expectRC("append: default verdict open", 0, r.rc);
  r = run(["append", D, "race in worker pool", "session", "log timestamps overlap", "ruled out"]);
  expectRC("append: explicit verdict", 0, r.rc);
  r = run(["append", D, "bad input", "session", "trace", "confirmed"]);
  expectRC("append: confirmed", 0, r.rc);

  r = run(["append", D, "x", "session", "y", "bogus"]);
  expectRC("append: bad verdict exits 2", 2, r.rc);
  r = run(["append", D]);
  expectRC("append: missing args exits 2", 2, r.rc);
}

// --- list / open -----------------------------------------------------------------

function caseList() {
  if (FILTER && FILTER !== "list") return;

  run(["init", D]);
  run(["append", D, "h1", "goldfish", "e1", "open"]);
  run(["append", D, "h2", "session", "e2", "ruled out"]);
  run(["append", D, "h3", "session", "e3", "confirmed"]);

  let all = run(["list", D]).out;
  expectContains("list: shows h1", all, "h1");
  expectContains("list: shows h2", all, "h2");
  expectContains("list: shows h3", all, "h3");

  const open = run(["open", D]).out;
  expectContains("open: shows only open rows", open, "h1");
  expectNotContains("open: hides ruled out", open, "h2");
  expectNotContains("open: hides confirmed", open, "h3");

  const r = run(["list", join(TMP_ROOT, "no-trail")]);
  expectRC("list: missing trail exits 2", 2, r.rc);
}

// --- count / status: the 3-fix gate -------------------------------------------------

function caseGate() {
  if (FILTER && FILTER !== "gate") return;

  run(["init", D]);
  run(["append", D, "h1", "session", "e1", "open"]);

  // zero fix attempts -> gate is go
  let status = run(["status", D]).out;
  expectContains("status: attempts 0", status, '"attempts":0');
  expectContains("status: gate go at 0", status, '"gate":"go"');

  appendFileSync(join(D, "state.md"), "fix-attempt: 1 — patch the timeout\n");
  appendFileSync(join(D, "state.md"), "fix-attempt: 2 — raise the limit\n");
  status = run(["status", D]).out;
  expectContains("status: attempts 2", status, '"attempts":2');
  expectContains("status: gate still go at 2", status, '"gate":"go"');

  appendFileSync(join(D, "state.md"), "fix-attempt: 3 — swap the provider\n");
  status = run(["status", D]).out;
  expectContains("status: attempts 3", status, '"attempts":3');
  expectContains("status: architecture-question at 3", status, '"gate":"architecture-question"');

  const count = run(["count", D]).out.trim();
  expectRC("count: reports 3", 0, count === "3" ? 0 : 1);
}

// --- run all (respect filter) -------------------------------------------------------

caseInit();
caseAppend();
caseList();
caseGate();

// --- report ------------------------------------------------------------------------

console.log("");
for (const line of RESULTS) console.log(line);
console.log("");
console.log("========================================");
console.log(`Results: ${PASS} passed, ${FAIL} failed`);
rmSync(TMP_ROOT, { recursive: true, force: true });
process.exit(FAIL === 0 ? 0 : 1);
