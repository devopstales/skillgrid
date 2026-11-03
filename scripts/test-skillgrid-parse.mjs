#!/usr/bin/env node
/**
 * test-skillgrid-parse.mjs — unit tests for scripts/lib/skillgrid-parse.mjs.
 *
 * Pure in-memory tests (no temp trees): feed parsed strings / fixture dirs and
 * assert the deterministic outputs. The phase derivation is exercised against
 * real temp change dirs (spec zone layouts) so the file-existence logic is
 * covered, not just the string parsers.
 *
 * Usage: node scripts/test-skillgrid-parse.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { derivePhase, worstScope, verdictFromReport, reviewFloor, retroFilled, parseStatusBanner, STATUS_VALUES, parseLedger, parseBlocked, normalizeVerdict } from "./lib/skillgrid-parse.mjs";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

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

// --- worstScope ---
{
  assert("worstScope: COMPLETE<COMPLETE", worstScope("COMPLETE", "COMPLETE") === "COMPLETE");
  assert("worstScope: TRUNCATED wins", worstScope("COMPLETE", "TRUNCATED") === "TRUNCATED");
  assert("worstScope: UNREADABLE worst", worstScope("UNSCOPED", "UNREADABLE") === "UNREADABLE");
  assert("worstScope: null passthrough", worstScope(null, "TRUNCATED") === "TRUNCATED");
}

// --- verdictFromReport ---
{
  assert("verdict: **Verdict:** PASS", verdictFromReport("## Gate Decision\n\n**Verdict:** PASS\n") === "PASS");
  assert("verdict: bare **CONCERNS**", verdictFromReport("## Gate Decision\n\n**CONCERNS**\n\nNo CRITICAL.") === "CONCERNS");
  assert("verdict: bare FAIL", verdictFromReport("## Gate Decision\n\n**FAIL**\n") === "FAIL");
  assert("verdict: absent → null", verdictFromReport("## Test Plan\n\n- nothing\n") === null);
  assert("verdict: first-word of `PASS / CONCERNS...` → PASS (resume doc behavior)", verdictFromReport("## Gate Decision\n\n**Verdict:** PASS / CONCERNS / FAIL / WAIVED\n") === "PASS");
  assert("normalizeVerdict: WAIVED", normalizeVerdict("waived") === "WAIVED");
  assert("normalizeVerdict: junk → null", normalizeVerdict("maybe") === null);
}

// --- reviewFloor ---
{
  assert("floor: met", reviewFloor("## Verdict\n\n- **Floor (decides):** met\n") === "met");
  assert("floor: met-with-fixes", reviewFloor("- **Floor (decides):** met-with-fixes\n") === "met-with-fixes");
  assert("floor: not met", reviewFloor("- **Floor (decides):** not met\n") === "not met");
  assert("floor: absent → null", reviewFloor("## Verdict\n\n- **Standards:** met\n") === null);
}

// --- retroFilled ---
{
  assert("retro: acceptance verdict chosen", retroFilled("## Acceptance Verdict\n\n**Verdict:** accepted\n") === true);
  assert("retro: Lessons table filled", retroFilled("## Lessons\n\n| Lesson | Why |\n|---|---|\n| x | y |\n") === true);
  assert("retro: empty report → false", retroFilled("## Final-State Facts\n\n- done\n") === false);
  assert("retro: Lessons empty table → false", retroFilled("## Lessons\n\n| Lesson | Why |\n|---|---|\n") === false);
}

// --- parseStatusBanner ---
{
  const b = parseStatusBanner("# T\n\n> **STATUS:** `sliced` (2026-10-24)\nrest\n");
  assert("banner: value sliced", b && b.value === "sliced", JSON.stringify(b));
  assert("banner: date", b && b.date === "2026-10-24", JSON.stringify(b));
  assert("banner: unbackticked", parseStatusBanner("> **STATUS:** draft (2026-01-01)\n")?.value === "draft");
  assert("banner: absent → null", parseStatusBanner("# T\n\nno banner\n") === null);
  assert("banner: allowed set has shipped", STATUS_VALUES.has("shipped"));
  assert("banner: allowed set has in-progress", STATUS_VALUES.has("in-progress"));
}

// --- parseLedger ---
{
  const l = parseLedger("# SDD ledger\nTask 1: complete\nTask 2: in-progress\nTask 3: pending\n");
  assert("ledger: done [1]", JSON.stringify(l.done) === JSON.stringify([1]), JSON.stringify(l));
  assert("ledger: next 2", l.next === 2, JSON.stringify(l));
  const l2 = parseLedger("Task 1: complete\nTask 2: complete\n");
  assert("ledger: all done → next null", l2.next === null, JSON.stringify(l2));
  const l3 = parseLedger("Task 5: in-progress\n");
  assert("ledger: single in-flight → next 5", l3.next === 5, JSON.stringify(l3));
}

// --- parseBlocked ---
{
  const b = parseBlocked("state\nblocked(execution): ticket T003 blocked by T001\nfoo\n");
  assert("blocked: phase", b.length === 1 && b[0].phase === "execution", JSON.stringify(b));
  assert("blocked: reason", b.length === 1 && b[0].reason.includes("T003"), JSON.stringify(b));
  assert("blocked: none → []", parseBlocked("no blockers\n").length === 0);
}

// --- derivePhase (real temp dirs) ---
function makeChange(files, { sdd = {} } = {}) {
  const root = mkdtempSync(join(tmpdir(), "sgparse-"));
  const dir = join(root, "change");
  mkdirSync(dir, { recursive: true });
  for (const [f, c] of Object.entries(files)) writeFileSync(join(dir, f), c);
  let sddDir = null;
  if (Object.keys(sdd).length) {
    sddDir = join(root, "sdd");
    mkdirSync(sddDir, { recursive: true });
    for (const [plan, content] of Object.entries(sdd)) {
      const p = join(sddDir, plan);
      mkdirSync(p, { recursive: true });
      writeFileSync(join(p, "progress.md"), content);
    }
  }
  return { dir, sddDir, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

const BRIEFING = "# Briefing\n";
const FEATURE = "Feature: x\n  Scenario: s\n";
const BLUEPRINT = "# Blueprint\n";
const TASKS_EMPTY = "# Tasks\n\n### TICKET-01\n- **Files:** a.go\n";
const TASKS_DONE = "# Tasks\n\n- [x] TICKET-01\n";
const REPORT_QA = "# Report\n\n## Test Plan\n\n- t\n";
const REPORT_VERDICT = "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n";
const REVIEW_MET = "## Verdict\n\n- **Floor (decides):** met\n";
const REVIEW_NOTMET = "## Verdict\n\n- **Floor (decides):** not met\n";

{
  const p = makeChange({ "briefing.md": BRIEFING });
  assert("phase: spec", derivePhase({ changeDir: p.dir }).phase === "spec");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT });
  assert("phase: blueprint", derivePhase({ changeDir: p.dir }).phase === "blueprint");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_EMPTY });
  assert("phase: slicing", derivePhase({ changeDir: p.dir }).phase === "slicing");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_DONE }, { "plan": "- [x] t" });
  assert("phase: execution (ledger [x])", derivePhase({ changeDir: p.dir, sddDir: p.sddDir }).phase === "execution");
  p.cleanup();
}
{
  // tasks.md itself has [x] (legacy fallback)
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_DONE });
  assert("phase: execution (tasks [x])", derivePhase({ changeDir: p.dir }).phase === "execution");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_EMPTY, "report.md": REPORT_QA });
  assert("phase: qa (report, no verdict)", derivePhase({ changeDir: p.dir }).phase === "qa");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_EMPTY, "report.md": REPORT_VERDICT, "review.md": REVIEW_MET });
  assert("phase: review (verdict + floor met)", derivePhase({ changeDir: p.dir }).phase === "review");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "tasks.md": TASKS_EMPTY, "report.md": REPORT_VERDICT, "review.md": REVIEW_NOTMET });
  assert("phase: qa (verdict + floor not met)", derivePhase({ changeDir: p.dir }).phase === "qa");
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "report.md": REPORT_QA });
  const r = derivePhase({ changeDir: p.dir, archived: true });
  assert("phase: reflect (xarchive, retro empty)", r.phase === "reflect", r.phase);
  p.cleanup();
}
{
  const p = makeChange({ "briefing.md": BRIEFING, "report.md": "# Report\n\n## Final-State Facts\n\n## Acceptance Verdict\n\n**Verdict:** accepted\n" });
  const r = derivePhase({ changeDir: p.dir, archived: true });
  assert("phase: complete (xarchive, retro filled)", r.phase === "complete", r.phase);
  p.cleanup();
}
{
  const p = makeChange({ "blueprint.md": BLUEPRINT });
  const r = derivePhase({ changeDir: p.dir });
  assert("phase: unknown (no briefing)", r.phase === "unknown", r.phase);
  p.cleanup();
}

// Summary
console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) {
  console.log("Failures:");
  for (const f of failures) console.log(`  - ${f}`);
}
process.exit(failed > 0 ? 1 : 0);
