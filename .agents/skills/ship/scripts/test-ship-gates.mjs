#!/usr/bin/env node
// test-ship-gates.mjs — prove the pre-mutation gate reader: each gate state
// (PASS / CONCERNS / FAIL / WAIVED / unresolved-CRITICAL / missing verdict,
// and the review floor) maps to the right blocked/override verdict.
//
// Hermetic: builds throwaway change dirs under mkdtemp; torn down on exit.
//
// Usage: node test-ship-gates.mjs
// Exit: 0 all pass, 1 any fail.

import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const GATES = join(HERE, "ship-gates.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function check(name, cond, detail = "") {
  if (cond) passed++;
  else {
    failed++;
    failures.push(name);
    console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`);
  }
}

function changeDir({ report, review }) {
  const dir = mkdtempSync(join(tmpdir(), "sgshipgates-"));
  writeFileSync(join(dir, "report.md"), report);
  if (review !== undefined) writeFileSync(join(dir, "review.md"), review);
  return dir;
}

function run(dir) {
  try {
    const out = execFileSync("node", [GATES, dir], { encoding: "utf8" });
    return { code: 0, out, json: JSON.parse(out) };
  } catch (e) {
    const out = (e.stdout || "") + (e.stderr || "");
    let json = null;
    try { json = JSON.parse(e.stdout); } catch { /* usage errors: no JSON */ }
    return { code: e.status, out, json };
  }
}

const GATE_PASS = `# Report

## Gate Decision

**Verdict:** PASS

- [x] intent: all truths verified
- [x] functional: suite green

## After
`;
const GATE_CONCERNS = `# Report

## Gate Decision

**Verdict:** CONCERNS

Open items:
- P0 finding without triangulation

## After
`;
const GATE_FAIL = `# Report

## Gate Decision

**Verdict:** FAIL

- scenario with no covering test

## After
`;
const GATE_WAIVED = `# Report

## Gate Decision

**Verdict:** WAIVED

Waiver: functional browser pass (no target URL). Risk accepted by human.

## After
`;
const GATE_UNRESOLVED_CRITICAL = `# Report

## Gate Decision

**Verdict:** PASS

Findings:
- [ ] CRITICAL: SQLi in the login handler

## After
`;
const GATE_RESOLVED_CRITICAL = `# Report

## Gate Decision

**Verdict:** PASS

Findings:
- [x] CRITICAL: SQLi in the login handler — fixed in 0f3c2a1

## After
`;
const GATE_PLACEHOLDER = `# Report

## Gate Decision

**Verdict:** PASS / CONCERNS / FAIL / WAIVED

## After
`;
const REVIEW_MET = `# Review

## Verdict

**Verdict:** met

| Axis | Grade |
|---|---|
| Spec compliance | A |
| Code quality | B |
`;
const REVIEW_NOT_MET = `# Review

## Verdict

**Verdict:** not met

| Axis | Grade |
|---|---|
| Spec compliance | C |
| Code quality | B |
`;

// --- 1. PASS + met -> not blocked, exit 0 --------------------------------------
let d = changeDir({ report: GATE_PASS, review: REVIEW_MET });
let r = run(d);
check("pass+met: exit 0", r.code === 0, r.out);
check("pass+met: not blocked", r.json && r.json.blocked === false, r.out);
check("pass+met: qa is PASS", r.json && r.json.qa === "PASS", r.out);
check("pass+met: review is met", r.json && r.json.review === "met", r.out);
rmSync(d, { recursive: true, force: true });

// --- 2. FAIL -> blocked(qa-failed), exit 1 --------------------------------------
d = changeDir({ report: GATE_FAIL, review: REVIEW_MET });
r = run(d);
check("fail: exit 1", r.code === 1, r.out);
check("fail: blocked", r.json && r.json.blocked === true, r.out);
check("fail: reason qa-failed", r.json && r.json.block_reason === "qa-failed", r.out);
rmSync(d, { recursive: true, force: true });

// --- 3. unresolved CRITICAL blocks even a PASS ----------------------------------
d = changeDir({ report: GATE_UNRESOLVED_CRITICAL, review: REVIEW_MET });
r = run(d);
check("unresolved-critical: exit 1", r.code === 1, r.out);
check("unresolved-critical: blocked", r.json && r.json.blocked === true, r.out);
check("unresolved-critical: reason qa-critical-unresolved", r.json && r.json.block_reason === "qa-critical-unresolved", r.out);
check("unresolved-critical: names the finding", r.json && r.json.findings.some((f) => f.includes("SQLi")), r.out);
rmSync(d, { recursive: true, force: true });

// --- 4. resolved (checked) CRITICAL does not block ------------------------------
d = changeDir({ report: GATE_RESOLVED_CRITICAL, review: REVIEW_MET });
r = run(d);
check("resolved-critical: exit 0", r.code === 0, r.out);
check("resolved-critical: not blocked", r.json && r.json.blocked === false, r.out);
rmSync(d, { recursive: true, force: true });

// --- 5. CONCERNS -> not blocked, but override needed ----------------------------
d = changeDir({ report: GATE_CONCERNS, review: REVIEW_MET });
r = run(d);
check("concerns: exit 0 (advisory)", r.code === 0, r.out);
check("concerns: not blocked", r.json && r.json.blocked === false, r.out);
check("concerns: qa_needs_override true", r.json && r.json.qa_needs_override === true, r.out);
rmSync(d, { recursive: true, force: true });

// --- 6. WAIVED -> not blocked, no override flag ---------------------------------
d = changeDir({ report: GATE_WAIVED, review: REVIEW_MET });
r = run(d);
check("waived: exit 0", r.code === 0, r.out);
check("waived: not blocked", r.json && r.json.blocked === false, r.out);
check("waived: no override needed", r.json && r.json.qa_needs_override === false, r.out);
rmSync(d, { recursive: true, force: true });

// --- 7. placeholder verdict -> blocked(qa-unverifiable) --------------------------
d = changeDir({ report: GATE_PLACEHOLDER, review: REVIEW_MET });
r = run(d);
check("placeholder: exit 1", r.code === 1, r.out);
check("placeholder: blocked", r.json && r.json.blocked === true, r.out);
check("placeholder: reason qa-unverifiable", r.json && r.json.block_reason === "qa-unverifiable", r.out);
rmSync(d, { recursive: true, force: true });

// --- 8. review not met -> advisory (not blocked), override flagged --------------
d = changeDir({ report: GATE_PASS, review: REVIEW_NOT_MET });
r = run(d);
check("review-not-met: exit 0 (advisory)", r.code === 0, r.out);
check("review-not-met: not blocked", r.json && r.json.blocked === false, r.out);
check("review-not-met: review_needs_override true", r.json && r.json.review_needs_override === true, r.out);
check("review-not-met: names the Grade C axis", r.json && r.json.findings.some((f) => f.includes("Spec compliance")), r.out);
rmSync(d, { recursive: true, force: true });

// --- 9. review absent -> advisory, not blocked -----------------------------------
d = changeDir({ report: GATE_PASS });
r = run(d);
check("review-absent: exit 0", r.code === 0, r.out);
check("review-absent: not blocked", r.json && r.json.blocked === false, r.out);
check("review-absent: review is absent", r.json && r.json.review === "absent", r.out);
rmSync(d, { recursive: true, force: true });

// --- 10. no report.md -> usage exit 2 --------------------------------------------
d = changeDir({ report: GATE_PASS });
const empty = mkdtempSync(join(tmpdir(), "sgshipgates-empty-"));
r = run(empty);
check("no-report: exit 2", r.code === 2, r.out);
rmSync(empty, { recursive: true, force: true });
rmSync(d, { recursive: true, force: true });

console.log(`\nship-gates tests: ${passed} passed, ${failed} failed`);
if (failed > 0) {
  console.error(failures.join("\n"));
  process.exit(1);
}
