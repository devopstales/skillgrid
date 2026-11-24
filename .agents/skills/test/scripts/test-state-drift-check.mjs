#!/usr/bin/env node
// test-state-drift-check.mjs — prove the state-drift guard: clean v2 and
// legacy v1 states exit 0 with a DRIFT + SCOPE line; phase / presence /
// completion drift exit 1 with the named table row; usage and parse errors
// exit 2; phase derivation (spec / blueprint / slicing / execution / qa) and
// the shippable-verdict detection behave per sdd-structure.md.
//
// Hermetic: builds throwaway project roots under mkdtemp; torn down on exit.
//
// Usage: node test-state-drift-check.mjs
// Exit code: 0 if all pass, 1 if any fail.
import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const script = join(here, "state-drift-check.mjs");

let pass = 0, fail = 0;
const work = mkdtempSync(join(tmpdir(), "skillgrid-state-drift-"));

function run(root) {
  try {
    const out = execFileSync("node", [script, root], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
    return { rc: 0, out };
  } catch (e) {
    return { rc: e.status ?? 1, out: `${e.stdout || ""}${e.stderr || ""}` };
  }
}

function check(label, haystack, needle) {
  if (haystack.includes(needle)) { console.log(`  ok: ${label}`); pass++; }
  else { console.log(`  FAIL: ${label} (missing: ${needle})`); fail++; }
}
function notcheck(label, haystack, needle) {
  if (haystack.includes(needle)) { console.log(`  FAIL: ${label} (unexpected: ${needle})`); fail++; }
  else { console.log(`  ok: ${label}`); pass++; }
}

function makeRoot(name) {
  const root = join(work, name);
  mkdirSync(join(root, ".skillgrid", "change-specs"), { recursive: true });
  return root;
}
function writeState(root, yaml) {
  writeFileSync(join(root, ".skillgrid", "state.yaml"), yaml);
}
function changeDir(root, slug, files = {}, zone = "change-specs") {
  const d = join(root, ".skillgrid", zone, slug);
  mkdirSync(d, { recursive: true });
  for (const [f, content] of Object.entries(files)) writeFileSync(join(d, f), content);
  return d;
}

const STATE_CLEAN = `schema: skillgrid/state/v2
changes:
  2026-01-01-alpha:
    phase: "spec"
    status: in_progress
    gate: ""
    autonomy: check-in
    genesis:
      independence:
        level: isolated-subagent
      decisions:
        - "a decision with a continuation
          line that looks like a key: yes it does"
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/projectwiki/ASSUMPTIONS.md   # pointer, not a copy
`;

const GATE_PASS = `# Test Report

## Test Plan

| Behavior | Test |
|---|---|

## Gate Decision

- **Verdict:** **PASS**
- **Floor (weakest dimension):** none

## Human Override

- **Override:** none
`;

const GATE_CONCERNS_OVERRIDE = `# Test Report

## Test Plan

## Gate Decision

- **Verdict:** CONCERNS
- **Human decision (applied at the qa-verdict gate):** fix later, proceed.

## Human Override

- **Override:** the human accepted the bench carry (2026-01-01)
`;

const GATE_CONCERNS_NO_OVERRIDE = `# Test Report

## Test Plan

## Gate Decision

- **Verdict:** CONCERNS
- **Floor (weakest dimension):** security

## Human Override

- **Override:** none
`;

try {
  // --- clean v2: state matches the spec zone --------------------------------
  {
    const root = makeRoot("clean");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n", "acceptance.feature": "### Scenario: s\n" });
    writeState(root, STATE_CLEAN);
    const r = run(root);
    if (r.rc === 0) { console.log("ok: clean v2 exit 0"); pass++; }
    else { console.log(`FAIL: clean v2 (exit ${r.rc})`); fail++; }
    check("clean v2: DRIFT verdict", r.out, "DRIFT: clean");
    check("clean v2: SCOPE line", r.out, "SCOPE: change-specs: 1 dir(s) enumerated, 1 map entry compared, 0 archive dir(s) scanned");
    notcheck("clean v2: no table", r.out, "| Check |");
  }

  // --- phase drift: blueprint.md present, state says spec --------------------
  {
    const root = makeRoot("phase-drift");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n", "blueprint.md": "# P\n" });
    writeState(root, STATE_CLEAN);
    const r = run(root);
    if (r.rc === 1) { console.log("ok: phase drift exit 1"); pass++; }
    else { console.log(`FAIL: phase drift (exit ${r.rc})`); fail++; }
    check("phase drift: hit count", r.out, "DRIFT: 1 hit(s)");
    check("phase drift: table row", r.out, "| phase | changes.2026-01-01-alpha.phase | spec | blueprint |");
  }

  // --- presence drift: dir without entry --------------------------------------
  {
    const root = makeRoot("presence-dir");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    changeDir(root, "2026-01-02-beta", { "briefing.md": "# B\n" });
    writeState(root, `schema: skillgrid/state/v2
changes:
  2026-01-02-beta:
    phase: "spec"
progress:
  completed_changes: 0
`);
    const r = run(root);
    if (r.rc === 1) { console.log("ok: presence (dir w/o entry) exit 1"); pass++; }
    else { console.log(`FAIL: presence dir (exit ${r.rc})`); fail++; }
    check("presence: dir row", r.out, "| presence | changes:2026-01-01-alpha | absent entry | dir present |");
    notcheck("presence: beta not flagged", r.out, "changes:2026-01-02-beta");
  }

  // --- presence drift: entry without dir --------------------------------------
  {
    const root = makeRoot("presence-entry");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    writeState(root, `schema: skillgrid/state/v2
changes:
  2026-01-01-alpha:
    phase: "spec"
  ghost-change:
    phase: "qa"
progress:
  completed_changes: 0
`);
    const r = run(root);
    if (r.rc === 1) { console.log("ok: presence (entry w/o dir) exit 1"); pass++; }
    else { console.log(`FAIL: presence entry (exit ${r.rc})`); fail++; }
    check("presence: entry row", r.out, "| presence | changes:ghost-change | entry present | dir absent |");
  }

  // --- completion drift: shippable report in archive, counter 0 ---------------
  {
    const root = makeRoot("completion-drift");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    changeDir(root, "2026-01-01-old", { "report.md": GATE_PASS }, "change-archive");
    writeState(root, STATE_CLEAN);
    const r = run(root);
    if (r.rc === 1) { console.log("ok: completion drift exit 1"); pass++; }
    else { console.log(`FAIL: completion drift (exit ${r.rc})`); fail++; }
    check("completion: row", r.out, "| completion | progress.completed_changes | 0 | shippable report in change-archive/2026-01-01-old |");
  }

  // --- shippable + counter 1 → clean ------------------------------------------
  {
    const root = makeRoot("completion-ok");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    changeDir(root, "2026-01-01-old", { "report.md": GATE_PASS }, "change-archive");
    writeState(root, STATE_CLEAN.replace("completed_changes: 0", "completed_changes: 1"));
    const r = run(root);
    if (r.rc === 0) { console.log("ok: shippable + counter 1 clean"); pass++; }
    else { console.log(`FAIL: shippable + counter 1 (exit ${r.rc})`); fail++; }
    check("completion ok: DRIFT clean", r.out, "DRIFT: clean");
  }

  // --- legacy v1: matching → clean --------------------------------------------
  {
    const root = makeRoot("legacy-clean");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    writeState(root, `schema: skillgrid/state/v1
pipeline:
  current_change: 2026-01-01-alpha
  current_phase: "spec"
`);
    const r = run(root);
    if (r.rc === 0) { console.log("ok: legacy v1 clean exit 0"); pass++; }
    else { console.log(`FAIL: legacy v1 clean (exit ${r.rc})`); fail++; }
    check("legacy: DRIFT clean", r.out, "DRIFT: clean");
    check("legacy: SCOPE notes legacy", r.out, "legacy v1 single-entry map");
    check("legacy: completion skipped", r.out, "completion check skipped");
  }

  // --- legacy v1: drifting phase ----------------------------------------------
  {
    const root = makeRoot("legacy-drift");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    writeState(root, `schema: skillgrid/state/v1
pipeline:
  current_change: 2026-01-01-alpha
  current_phase: "execution"
`);
    const r = run(root);
    if (r.rc === 1) { console.log("ok: legacy v1 drift exit 1"); pass++; }
    else { console.log(`FAIL: legacy v1 drift (exit ${r.rc})`); fail++; }
    check("legacy drift: row", r.out, "| phase | changes.2026-01-01-alpha.phase | execution | spec |");
  }

  // --- no state.yaml → exit 2 ---------------------------------------------------
  {
    const root = makeRoot("no-state");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    const r = run(root);
    if (r.rc === 2) { console.log("ok: no state.yaml exit 2"); pass++; }
    else { console.log(`FAIL: no state.yaml (exit ${r.rc})`); fail++; }
    check("no state: message", r.out, "cannot read");
  }

  // --- absent change-specs → clean with scope note ------------------------------
  {
    const root = join(work, "no-specs");
    mkdirSync(join(root, ".skillgrid"), { recursive: true });
    writeState(root, `schema: skillgrid/state/v2
changes:
progress:
  completed_changes: 0
`);
    const r = run(root);
    if (r.rc === 0) { console.log("ok: absent change-specs clean"); pass++; }
    else { console.log(`FAIL: absent change-specs (exit ${r.rc})`); fail++; }
    check("no specs: DRIFT clean", r.out, "DRIFT: clean");
    check("no specs: scope note", r.out, "change-specs absent");
  }

  // --- execution signal via tasks.md `[x]` ---------------------------------------
  {
    const root = makeRoot("exec-tasks");
    changeDir(root, "2026-01-01-alpha", {
      "tasks.md": "# Tasks\n- [ ] T1 pending\n- [x] T2 done\n",
    });
    writeState(root, STATE_CLEAN.replace('phase: "spec"', 'phase: "execution"'));
    let r = run(root);
    if (r.rc === 0) { console.log("ok: tasks [x] → execution matches state"); pass++; }
    else { console.log(`FAIL: tasks [x] clean (exit ${r.rc})`); fail++; }
    writeState(root, STATE_CLEAN.replace('phase: "spec"', 'phase: "slicing"'));
    r = run(root);
    if (r.rc === 1) { console.log("ok: tasks [x] vs slicing → drift"); pass++; }
    else { console.log(`FAIL: tasks [x] vs slicing (exit ${r.rc})`); fail++; }
    check("exec via tasks: row", r.out, "| phase | changes.2026-01-01-alpha.phase | slicing | execution |");
  }

  // --- execution signal via work/<slug>/progress.md `[x]` --------------------------
  {
    const root = makeRoot("exec-work");
    changeDir(root, "2026-01-01-alpha", { "tasks.md": "# Tasks\n- [ ] T1 pending\n" });
    const w = join(root, ".skillgrid", "work", "2026-01-01-alpha");
    mkdirSync(w, { recursive: true });
    writeFileSync(join(w, "progress.md"), "# Progress\n- [x] step one\n- [ ] step two\n");
    writeState(root, STATE_CLEAN.replace('phase: "spec"', 'phase: "execution"'));
    const r = run(root);
    if (r.rc === 0) { console.log("ok: work/progress [x] → execution matches state"); pass++; }
    else { console.log(`FAIL: work/progress [x] (exit ${r.rc})\n${r.out}`); fail++; }
  }

  // --- report.md with only a Test Plan: qa phase, NOT shippable ---------------------
  {
    const root = makeRoot("report-no-gate");
    const planOnly = "# Test Report\n\n## Test Plan\n\n| Behavior | Test |\n|---|---|\n| x | y |\n";
    changeDir(root, "2026-01-01-alpha", { "report.md": planOnly });
    changeDir(root, "2026-01-01-old", { "report.md": planOnly }, "change-archive");
    writeState(root, STATE_CLEAN.replace('phase: "spec"', 'phase: "qa"'));
    const r = run(root);
    if (r.rc === 0) { console.log("ok: Test Plan only → qa phase, not shippable"); pass++; }
    else { console.log(`FAIL: Test Plan only (exit ${r.rc})\n${r.out}`); fail++; }
    check("report-no-gate: DRIFT clean", r.out, "DRIFT: clean");
  }

  // --- CONCERNS + human override → shippable; without override → not -----------------
  {
    const root = makeRoot("concerns-override");
    changeDir(root, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    changeDir(root, "2026-01-01-old", { "report.md": GATE_CONCERNS_OVERRIDE }, "change-archive");
    writeState(root, STATE_CLEAN);
    let r = run(root);
    if (r.rc === 1) { console.log("ok: CONCERNS + override → completion drift"); pass++; }
    else { console.log(`FAIL: CONCERNS + override (exit ${r.rc})\n${r.out}`); fail++; }
    check("concerns + override: row", r.out, "completion");

    const root2 = makeRoot("concerns-no-override");
    changeDir(root2, "2026-01-01-alpha", { "briefing.md": "# B\n" });
    changeDir(root2, "2026-01-01-old", { "report.md": GATE_CONCERNS_NO_OVERRIDE }, "change-archive");
    writeState(root2, STATE_CLEAN);
    r = run(root2);
    if (r.rc === 0) { console.log("ok: CONCERNS without override → not shippable"); pass++; }
    else { console.log(`FAIL: CONCERNS without override (exit ${r.rc})\n${r.out}`); fail++; }
  }

  // --- undervivable dir: no derivable artifact → check 1 skipped, noted -------------
  {
    const root = makeRoot("undervivable");
    changeDir(root, "2026-01-01-alpha", { "proposal.md": "# P\n" });
    writeState(root, `schema: skillgrid/state/v2
changes:
  2026-01-01-alpha:
    phase: "intake"
progress:
  completed_changes: 0
`);
    const r = run(root);
    if (r.rc === 0) { console.log("ok: undervivable dir → check 1 skipped, clean"); pass++; }
    else { console.log(`FAIL: undervivable (exit ${r.rc})\n${r.out}`); fail++; }
    check("undervivable: DRIFT clean", r.out, "DRIFT: clean");
    notcheck("undervivable: no phase row", r.out, "| phase |");
    check("undervivable: scope note", r.out, "without a derivable artifact (check 1 skipped): 2026-01-01-alpha");
  }
} finally {
  rmSync(work, { recursive: true, force: true });
}

console.log("");
console.log(`state-drift-check: ${pass} passed, ${fail} failed`);
process.exit(fail === 0 ? 0 : 1);
