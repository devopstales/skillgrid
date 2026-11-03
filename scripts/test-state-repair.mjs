#!/usr/bin/env node
/**
 * test-state-repair.mjs — fixture-driven tests for scripts/state-repair.mjs.
 *
 * The drift guard (state-drift-check.mjs) is read-only by design (ADR-0008).
 * When it reports drift, the agent hand-edits state.yaml — the failure mode
 * the resume skill warns about ("Never reset changes or progress to a clean
 * value"). state-repair.mjs performs that edit deterministically: it reads the
 * drift, patches ONLY the drifted fields (changes.<slug>.phase, missing
 * entries, completed_changes), and prints before/after. It never rewrites the
 * whole map.
 *
 * Usage: node scripts/test-state-repair.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const REPAIR = join(HERE, "state-repair.mjs");

let passed = 0, failed = 0;
const failures = [];
function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function makeRoot({ state, specs = {} }) {
  const root = mkdtempSync(join(tmpdir(), "sgrep-"));
  const sg = join(root, ".skillgrid");
  mkdirSync(sg, { recursive: true });
  writeFileSync(join(sg, "state.yaml"), state);
  for (const [slug, files] of Object.entries(specs)) {
    const dir = join(sg, "specs", slug);
    mkdirSync(dir, { recursive: true });
    for (const [f, c] of Object.entries(files)) writeFileSync(join(dir, f), c);
  }
  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(root, ...args) {
  try {
    const stdout = execFileSync("node", [REPAIR, ...args, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}
function parse(out) { try { return JSON.parse(out); } catch { return null; } }

// A briefing.md makes the derived phase "spec".
const BRIEFING = "# Briefing\n\nSome intent.\n";
// A blueprint.md + acceptance.feature makes the phase "blueprint".
const BLUEPRINT = "# Blueprint\n\n## Intent\n\nx\n";
const FEATURE = "# Cap\n\n### Requirement: r\n\n#### Scenario: s\n";

// 1. Drift: state says phase: spec, but the dir has a blueprint → repair to blueprint
{
  const p = makeRoot({
    state: `schema: skillgrid/state/v2\nchanges:\n  demo:\n    phase: spec\n    status: in-progress\nprogress:\n  completed_changes: 0\n`,
    specs: { demo: { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "acceptance.feature": FEATURE } },
  });
  // The derived phase for a dir with briefing+blueprint+feature is "slicing" (has blueprint, has feature, no tasks).
  // We repair the phase to whatever derivePhase returns; assert it changed from "spec".
  const r = run(p.root, "--write");
  const j = parse(r.stdout);
  assert("phase-drift: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("phase-drift: reported a phase patch", j && j.patches.some((pt) => pt.field === "changes.demo.phase"), JSON.stringify(j?.patches));
  assert("phase-drift: new phase != spec", j && j.patches.find((pt) => pt.field === "changes.demo.phase")?.to !== "spec", JSON.stringify(j?.patches));
  // The file was actually rewritten.
  const after = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  assert("phase-drift: file no longer says phase: spec", !/phase: spec/.test(after), after);
  assert("phase-drift: file still has the other change fields", after.includes("schema: skillgrid/state/v2"), after);
  p.cleanup();
}

// 2. Missing entry: a specs dir with no changes.<slug> entry → add it.
//    (Both demo and other have specs dirs, so neither is stale.)
{
  const p = makeRoot({
    state: `schema: skillgrid/state/v2\nchanges:\n  other:\n    phase: spec\n    status: in-progress\nprogress:\n  completed_changes: 0\n`,
    specs: {
      "demo": { "briefing.md": BRIEFING },
      "other": { "briefing.md": BRIEFING },
    },
  });
  const r = run(p.root, "--write");
  const j = parse(r.stdout);
  assert("missing-entry: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("missing-entry: reported add changes.demo", j && j.patches.some((pt) => pt.field === "changes.demo"), JSON.stringify(j?.patches));
  const after = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  assert("missing-entry: file now has demo", after.includes("demo:"), after);
  assert("missing-entry: other entry preserved", after.includes("other:"), after);
  p.cleanup();
}

// 3. No drift → no patches, file unchanged
{
  // A dir with only a briefing derives "spec"; state says spec → no drift.
  const state = `schema: skillgrid/state/v2\nchanges:\n  demo:\n    phase: spec\n    status: in-progress\nprogress:\n  completed_changes: 0\n`;
  const p = makeRoot({ state, specs: { demo: { "briefing.md": BRIEFING } } });
  const before = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  const r = run(p.root, "--write");
  const j = parse(r.stdout);
  assert("nodrift: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("nodrift: 0 patches", j && j.patches.length === 0, JSON.stringify(j?.patches));
  const after = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  assert("nodrift: file unchanged", before === after, "file changed with no drift");
  p.cleanup();
}

// 4. Dry-run (no --write) → reports patches but does NOT modify the file
{
  const p = makeRoot({
    state: `schema: skillgrid/state/v2\nchanges:\n  demo:\n    phase: spec\n    status: in-progress\nprogress:\n  completed_changes: 0\n`,
    specs: { demo: { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "acceptance.feature": FEATURE } },
  });
  const before = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  const r = run(p.root); // no --write
  const j = parse(r.stdout);
  assert("dryrun: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("dryrun: would patch", j && j.patches.length > 0, JSON.stringify(j?.patches));
  assert("dryrun: wrote=false", j && j.wrote === false, JSON.stringify(j));
  const after = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  assert("dryrun: file unchanged", before === after, "file changed in dry-run");
  p.cleanup();
}

// 5. No state.yaml → exit 2
{
  const p = makeRoot({ state: "" });
  // Empty state file is parseable but has no changes map; repair of an empty
  // map is a no-op. Use a missing file instead:
  const root2 = mkdtempSync(join(tmpdir(), "sgrep2-"));
  mkdirSync(join(root2, ".skillgrid"), { recursive: true });
  const r = run(root2);
  assert("nostate: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
  rmSync(root2, { recursive: true, force: true });
  p.cleanup();
}

// 5b. Comment preservation: the repair must NOT drop YAML comments (the
//     parse/stringify round-trip used to — that's the "agent mis-edits the
//     YAML" failure this script exists to prevent).
{
  const state = `schema: skillgrid/state/v2\n# v2: the map IS the resume pointer.\nchanges:\n  demo:\n    phase: spec\n    status: in-progress\n  # 2026-01-01-old: shipped, archived\n  other:\n    phase: spec\n    status: in-progress\nprogress:\n  completed_changes: 1\n`;
  // demo has only a briefing → derives "spec" (matches). other has a briefing
  // too. Add a specs dir for demo with a blueprint so demo drifts to slicing.
  const p = makeRoot({
    state,
    specs: {
      "demo": { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "acceptance.feature": FEATURE },
      "other": { "briefing.md": BRIEFING },
    },
  });
  run(p.root, "--write");
  const after = readFileSync(join(p.root, ".skillgrid", "state.yaml"), "utf8");
  assert("comments: top comment preserved", after.includes("# v2: the map IS the resume pointer."), after);
  assert("comments: archived-comment preserved", after.includes("# 2026-01-01-old: shipped, archived"), after);
  assert("comments: demo phase updated", /phase: (slicing|blueprint)/.test(after), after);
  assert("comments: still valid YAML + 2 entries", (after.match(/^\s{2}\S.*:$/gm) || []).length >= 2, after);
  p.cleanup();
}

// 6. Determinism: dry-run run twice identical
{
  const p = makeRoot({
    state: `schema: skillgrid/state/v2\nchanges:\n  demo:\n    phase: spec\n    status: in-progress\nprogress:\n  completed_changes: 0\n`,
    specs: { demo: { "briefing.md": BRIEFING, "blueprint.md": BLUEPRINT, "acceptance.feature": FEATURE } },
  });
  const a = run(p.root).stdout;
  const b = run(p.root).stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
