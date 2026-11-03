#!/usr/bin/env node
/**
 * test-phase-router.mjs — fixture-driven tests for scripts/phase-router.mjs.
 *
 * Builds temp project trees (spec zone + state.yaml), runs the router via node,
 * and asserts the JSON output: derived phase, next skill, drift, ledger,
 * blocked, and the run-twice-identical determinism guarantee.
 *
 * Usage: node scripts/test-phase-router.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const ROUTER = join(HERE, "phase-router.mjs");

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

function makeProject({ stateYaml = null, specs = {}, sdd = {} } = {}) {
  const root = mkdtempSync(join(tmpdir(), "sgphase-"));
  const sg = join(root, ".skillgrid");
  mkdirSync(join(sg, "specs"), { recursive: true });
  mkdirSync(join(sg, "sdd"), { recursive: true });
  if (stateYaml !== null) writeFileSync(join(sg, "state.yaml"), stateYaml);
  for (const [dir, files] of Object.entries(specs)) {
    const d = join(sg, "specs", dir);
    mkdirSync(d, { recursive: true });
    for (const [f, c] of Object.entries(files)) writeFileSync(join(d, f), c);
  }
  for (const [dir, files] of Object.entries(sdd)) {
    const d = join(sg, "sdd", dir);
    mkdirSync(d, { recursive: true });
    for (const [f, c] of Object.entries(files)) writeFileSync(join(d, f), c);
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function runRouter(root, arg) {
  try {
    const stdout = execFileSync("node", [ROUTER, arg, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}

function parse(out) {
  try {
    return JSON.parse(out);
  } catch {
    return null;
  }
}

const BRIEFING = "# Briefing\n";
const BLUEPRINT = "# Blueprint\n";
const TASKS = "# Tasks\n\n### TICKET-01\n- **Files:** a.go\n";
const TASKS_DONE = "# Tasks\n\n- [x] TICKET-01\n- [ ] TICKET-02\n";

const STATE_SPEC = `schema: skillgrid/state/v2
changes:
  demo-change:
    phase: spec
    status: in_progress
progress:
  completed_changes: 0
`;
const STATE_LAG = `schema: skillgrid/state/v2
changes:
  demo-change:
    phase: blueprint
    status: in_progress
progress:
  completed_changes: 0
`;

// 1. spec phase, clean
{
  const p = makeProject({ stateYaml: STATE_SPEC, specs: { "demo-change": { "briefing.md": BRIEFING } } });
  const r = runRouter(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("spec: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("spec: phase spec", j?.phase === "spec", r.stdout);
  assert("spec: next writing-blueprints", j?.next === "writing-blueprints", r.stdout);
  assert("spec: no drift", j?.drift === null, r.stdout);
  assert("spec: scope COMPLETE", j?.scope === "COMPLETE", r.stdout);
  p.cleanup();
}

// 2. execution phase + ledger + drift (state says blueprint)
{
  const p = makeProject({
    stateYaml: STATE_LAG,
    specs: { "demo-change": { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_DONE } },
    sdd: { "demo-change": { "progress.md": "Task 1: complete\nTask 2: in-progress\n" } },
  });
  const r = runRouter(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("exec: exit 1 (drift)", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("exec: phase execution", j?.phase === "execution", r.stdout);
  assert("exec: next qa", j?.next === "qa", r.stdout);
  assert("exec: drift field", j?.drift?.field === "changes.demo-change.phase", r.stdout);
  assert("exec: drift stale blueprint", j?.drift?.stale === "blueprint", r.stdout);
  assert("exec: drift derived execution", j?.drift?.derived === "execution", r.stdout);
  assert("exec: ledger done [1]", Array.isArray(j?.ledger?.done) && j.ledger.done[0] === 1, r.stdout);
  assert("exec: ledger next 2", j?.ledger?.next === 2, r.stdout);
  p.cleanup();
}

// 3. blocked reason → next_recommended resolve-blockers
{
  const stateBlocked = `schema: skillgrid/state/v2
changes:
  demo-change:
    phase: execution
    status: in_progress
    blocked: "blocked(execution): ticket T002 blocked by T001"
progress:
  completed_changes: 0
`;
  const p = makeProject({
    stateYaml: stateBlocked,
    specs: { "demo-change": { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_DONE } },
    sdd: { "demo-change": { "progress.md": "- [x] t1" } },
  });
  const r = runRouter(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("blocked: phase execution", j?.phase === "execution", r.stdout);
  assert("blocked: next_recommended resolve-blockers", j?.next_recommended === "resolve-blockers", r.stdout);
  assert("blocked: blocked[0].phase execution", j?.blocked?.[0]?.phase === "execution", r.stdout);
  p.cleanup();
}

// 4. review phase (verdict + floor met)
{
  const stateReview = `schema: skillgrid/state/v2
changes:
  demo-change:
    phase: review
    status: in_progress
progress:
  completed_changes: 0
`;
  const p = makeProject({
    stateYaml: stateReview,
    specs: {
      "demo-change": {
        "briefing.md": BRIEFING,
        "blueprint.md": BLUEPRINT,
        "tasks.md": TASKS,
        "report.md": "## Gate Decision\n\n**Verdict:** PASS\n",
        "review.md": "## Verdict\n\n- **Floor (decides):** met\n",
      },
    },
  });
  const r = runRouter(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("review: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("review: phase review", j?.phase === "review", r.stdout);
  assert("review: next receiving-code-review", j?.next === "receiving-code-review", r.stdout);
  p.cleanup();
}

// 5. no state.yaml entry → presence drift
{
  const p = makeProject({
    stateYaml: "schema: skillgrid/state/v2\nchanges: {}\nprogress:\n  completed_changes: 0\n",
    specs: { "demo-change": { "briefing.md": BRIEFING } },
  });
  const r = runRouter(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("presence: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("presence: drift field changes", j?.drift?.field === "changes", r.stdout);
  p.cleanup();
}

// 6. `all` routes every change + reports drifted count
{
  const p = makeProject({
    stateYaml: STATE_LAG,
    specs: {
      "clean-change": { "briefing.md": BRIEFING },
      "demo-change": { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT },
    },
  });
  const r = runRouter(p.root, "all");
  const j = parse(r.stdout);
  assert("all: exit 1 (one drifted)", r.code === 1, `got ${r.code}: ${r.stderr}`);
  assert("all: 2 changes", Array.isArray(j?.changes) && j.changes.length === 2, r.stdout);
  assert("all: drifted count 1", j?.drifted === 1, r.stdout);
  p.cleanup();
}

// 7. determinism — run twice, identical output
{
  const p = makeProject({
    stateYaml: STATE_LAG,
    specs: { "demo-change": { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_DONE } },
    sdd: { "demo-change": { "progress.md": "- [x] t1" } },
  });
  const a = runRouter(p.root, "demo-change").stdout;
  const b = runRouter(p.root, "demo-change").stdout;
  assert("determinism: run twice identical", a === b && a.length > 0);
  p.cleanup();
}

// 8. usage: missing change dir → exit 2
{
  const p = makeProject({ stateYaml: STATE_SPEC, specs: { "demo-change": { "briefing.md": BRIEFING } } });
  const r = runRouter(p.root, "does-not-exist");
  assert("usage: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
  p.cleanup();
}

// Summary
console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) {
  console.log("Failures:");
  for (const f of failures) console.log(`  - ${f}`);
}
process.exit(failed > 0 ? 1 : 0);
