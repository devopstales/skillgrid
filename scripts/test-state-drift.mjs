#!/usr/bin/env node
/**
 * test-state-drift.mjs — fixture-driven tests for state-drift-check.mjs.
 *
 * Creates temp project trees (spec zone + state.yaml combos), runs the guard
 * via node, and asserts exit codes + drift output. Same shape as
 * test-hooks.sh: fixtures in, exit codes out.
 *
 * Usage: node scripts/test-state-drift.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const GUARD = join(HERE, "state-drift-check.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function makeProject({ stateYaml, specs = {}, sdd = {} } = {}) {
  const root = mkdtempSync(join(tmpdir(), "sgdrift-"));
  const sg = join(root, ".skillgrid");
  mkdirSync(sg, { recursive: true });
  mkdirSync(join(sg, "specs"), { recursive: true });
  mkdirSync(join(sg, "sdd"), { recursive: true });

  if (stateYaml !== null) {
    writeFileSync(join(sg, "state.yaml"), stateYaml);
  }

  for (const [dir, files] of Object.entries(specs)) {
    const d = join(sg, "specs", dir);
    mkdirSync(d, { recursive: true });
    for (const [fname, content] of Object.entries(files)) {
      writeFileSync(join(d, fname), content);
    }
  }

  for (const [dir, files] of Object.entries(sdd)) {
    const d = join(sg, "sdd", dir);
    mkdirSync(d, { recursive: true });
    for (const [fname, content] of Object.entries(files)) {
      writeFileSync(join(d, fname), content);
    }
  }

  return { root, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function runGuard(root) {
  try {
    const stdout = execFileSync("node", [GUARD, root], {
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
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

// --- State YAML fixtures ---

const STATE_CLEAN_SPEC = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "spec"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_CLEAN_BLUEPRINT = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "blueprint"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_CLEAN_SLICING = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "slicing"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_CLEAN_EXECUTION = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "execution"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_CLEAN_QA = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "qa"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_SHIPPED = `
schema: skillgrid/state/v1
pipeline:
  current_phase: ""
  current_change: ""
  status: idle
progress:
  completed_changes: 1
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_PHASE_LAG = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "blueprint"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_WRONG_CHANGE = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "spec"
  current_change: "2026-08-01-other-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_UNCOUNTED = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "qa"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 0
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

const STATE_COUNTED = `
schema: skillgrid/state/v1
pipeline:
  current_phase: "qa"
  current_change: "2026-09-01-test-change"
  status: in_progress
progress:
  completed_changes: 1
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md
notes: "test"
`;

// --- Spec zone fixtures ---

const SPEC_BRIEFING = {
  "2026-09-01-test-change": {
    "briefing.md": "# Briefing\ntest\n",
    "acceptance.feature": "Feature: test\n  Scenario: s1\n    Given x\n    When y\n    Then z\n",
  },
};

const SPEC_BLUEPRINT = {
  "2026-09-01-test-change": {
    "briefing.md": "# Briefing\ntest\n",
    "acceptance.feature": "Feature: test\n  Scenario: s1\n    Given x\n    When y\n    Then z\n",
    "blueprint.md": "# Blueprint\ntest\n",
  },
};

const SPEC_SLICING = {
  "2026-09-01-test-change": {
    "briefing.md": "# Briefing\ntest\n",
    "acceptance.feature": "Feature: test\n  Scenario: s1\n    Given x\n    When y\n    Then z\n",
    "blueprint.md": "# Blueprint\ntest\n",
    "tasks.md": "# Tasks\n\n## Tickets\n\n### TICKET-01 — Test ticket\n- **Scope:** test\n- **Acceptance:** test passes\n",
  },
};

const SPEC_EXECUTION = {
  "2026-09-01-test-change": {
    "briefing.md": "# Briefing\ntest\n",
    "acceptance.feature": "Feature: test\n  Scenario: s1\n    Given x\n    When y\n    Then z\n",
    "blueprint.md": "# Blueprint\ntest\n",
    "tasks.md": "# Tasks\n\n## Tickets\n\n### TICKET-01 — Test ticket\n- **Scope:** test\n- **Acceptance:** test passes\n",
  },
};

const SPEC_QA = {
  "2026-09-01-test-change": {
    "briefing.md": "# Briefing\ntest\n",
    "acceptance.feature": "Feature: test\n  Scenario: s1\n    Given x\n    When y\n    Then z\n",
    "blueprint.md": "# Blueprint\ntest\n",
    "tasks.md": "# Tasks\n\n## Tickets\n\n### TICKET-01 — Test ticket\n- **Scope:** test\n- **Acceptance:** test passes\n",
    "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n\n**Reasoning:** all good\n",
  },
};

const SDD_LEDGER = {
  "test-plan": {
    "progress.md": "# Progress\n- [x] TICKET-01 done\n",
  },
};

// --- Test cases ---

function run() {
  console.log("state-drift-check.mjs tests\n");

  // 1. Clean: spec zone at spec phase, state.yaml matches
  {
    const p = makeProject({ stateYaml: STATE_CLEAN_SPEC, specs: SPEC_BRIEFING });
    const r = runGuard(p.root);
    assert("clean-spec: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("clean-spec: DRIFT: none", r.stdout.includes("DRIFT: none"), r.stdout);
    assert("clean-spec: SCOPE: COMPLETE", /^SCOPE: COMPLETE$/m.test(r.stdout), r.stdout);
    p.cleanup();
  }

  // 2. Clean: spec zone at blueprint phase, state.yaml matches
  {
    const p = makeProject({ stateYaml: STATE_CLEAN_BLUEPRINT, specs: SPEC_BLUEPRINT });
    const r = runGuard(p.root);
    assert("clean-blueprint: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 3. Clean: spec zone at slicing phase, state.yaml matches
  {
    const p = makeProject({ stateYaml: STATE_CLEAN_SLICING, specs: SPEC_SLICING });
    const r = runGuard(p.root);
    assert("clean-slicing: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 4. Clean: spec zone at execution phase (ledger present), state.yaml matches
  {
    const p = makeProject({ stateYaml: STATE_CLEAN_EXECUTION, specs: SPEC_EXECUTION, sdd: SDD_LEDGER });
    const r = runGuard(p.root);
    assert("clean-execution: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 5. Clean: spec zone at qa phase, state.yaml matches, no report verdict yet (QA-half not written)
  {
    const specsNoVerdict = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
        "report.md": "# Report\n\n## Test Plan\n\n- test\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_CLEAN_QA, specs: specsNoVerdict });
    const r = runGuard(p.root);
    assert("clean-qa-no-verdict: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 6. Drift: phase lag (state says blueprint, spec zone has tasks.md → slicing)
  {
    const p = makeProject({ stateYaml: STATE_PHASE_LAG, specs: SPEC_SLICING });
    const r = runGuard(p.root);
    assert("phase-lag: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("phase-lag: names current_phase", r.stdout.includes("current_phase"), r.stdout);
    assert("phase-lag: names stale value", r.stdout.includes("blueprint"), r.stdout);
    assert("phase-lag: names derived value", r.stdout.includes("slicing"), r.stdout);
    p.cleanup();
  }

  // 7. Drift: wrong change dir
  {
    const p = makeProject({ stateYaml: STATE_WRONG_CHANGE, specs: SPEC_BRIEFING });
    const r = runGuard(p.root);
    assert("wrong-change: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("wrong-change: names current_change", r.stdout.includes("current_change"), r.stdout);
    assert("wrong-change: names stale value", r.stdout.includes("2026-08-01-other-change"), r.stdout);
    p.cleanup();
  }

  // 8. Drift: completion not counted (PASS verdict on shipped change, completed_changes=0)
  {
    const specs = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n\n**Reasoning:** all good\n",
      },
      "2026-08-01-shipped-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n\n**Reasoning:** shipped\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_UNCOUNTED, specs });
    const r = runGuard(p.root);
    assert("completion-uncounted: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("completion-uncounted: names completed_changes", r.stdout.includes("completed_changes"), r.stdout);
    p.cleanup();
  }

  // 9. Clean: completion counted (PASS verdict on shipped change, completed_changes=1)
  {
    const specs = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n\n**Reasoning:** all good\n",
      },
      "2026-08-01-shipped-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n\n**Reasoning:** shipped\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_COUNTED, specs });
    const r = runGuard(p.root);
    assert("completion-counted: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 10. state.yaml missing → exit 2
  {
    const p = makeProject({ stateYaml: null, specs: SPEC_BRIEFING });
    const r = runGuard(p.root);
    assert("state-missing: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 11. state.yaml unparseable → exit 2
  {
    const p = makeProject({ stateYaml: "not: valid: yaml: [", specs: SPEC_BRIEFING });
    const r = runGuard(p.root);
    assert("state-unparseable: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 12. No specs dir → exit 0 (no active change, nothing to check). The specs
  //     dir exists but is empty, so the enumeration is total → COMPLETE.
  {
    const p = makeProject({ stateYaml: STATE_SHIPPED, specs: {} });
    const r = runGuard(p.root);
    assert("no-specs: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("no-specs: SCOPE: COMPLETE (empty-but-present specs dir)", /^SCOPE: COMPLETE$/m.test(r.stdout), r.stdout);
    p.cleanup();
  }

  // 12b. specs dir genuinely missing (not just empty) → UNSCOPED: the
  //      "no active change" answer is a non-answer, not a clean bill.
  {
    const p = makeProject({ stateYaml: STATE_SHIPPED, specs: {} });
    rmSync(join(p.root, ".skillgrid", "specs"), { recursive: true, force: true });
    const r = runGuard(p.root);
    assert("specs-missing: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    assert("specs-missing: SCOPE: UNSCOPED", /^SCOPE: UNSCOPED$/m.test(r.stdout), r.stdout);
    p.cleanup();
  }

  // 12c. state names a change but specs dir is missing → drift + UNSCOPED.
  {
    const p = makeProject({ stateYaml: STATE_CLEAN_SPEC, specs: {} });
    rmSync(join(p.root, ".skillgrid", "specs"), { recursive: true, force: true });
    const r = runGuard(p.root);
    assert("specs-missing-drift: exit 1", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("specs-missing-drift: names current_change", r.stdout.includes("current_change"), r.stdout);
    assert("specs-missing-drift: SCOPE: UNSCOPED", /^SCOPE: UNSCOPED$/m.test(r.stdout), r.stdout);
    p.cleanup();
  }

  // 13. Shipped: state cleared, shipped change has PASS, counted → exit 0
  {
    const specs = {
      "2026-08-01-shipped-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** PASS\n\n**Reasoning:** shipped\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_SHIPPED, specs });
    const r = runGuard(p.root);
    assert("shipped: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 14. CONCERNS verdict without human override → not shippable → no completion drift
  {
    const specsConcerns = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** CONCERNS\n\n**Reasoning:** minor issue\n\n**Open items:**\n- fix it\n\n## Human Override\n\n**Human decision (fill in):** \n",
      },
    };
    const p = makeProject({ stateYaml: STATE_UNCOUNTED, specs: specsConcerns });
    const r = runGuard(p.root);
    assert("concerns-no-override: exit 0 (no completion drift)", r.code === 0, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 15. CONCERNS verdict WITH human override on shipped change → shippable → completion drift if uncounted
  {
    const specs = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
      },
      "2026-08-01-shipped-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** CONCERNS\n\n**Reasoning:** minor issue\n\n**Open items:**\n- fix it\n\n## Human Override\n\n**Human decision (fill in):** accept — Alice — 2026-09-24\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_UNCOUNTED, specs });
    const r = runGuard(p.root);
    assert("concerns-override: exit 1 (completion drift)", r.code === 1, `got ${r.code}: ${r.stderr}`);
    assert("concerns-override: names completed_changes", r.stdout.includes("completed_changes"), r.stdout);
    p.cleanup();
  }

  // 16. WAIVED verdict on shipped change → shippable → completion drift if uncounted
  {
    const specs = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
      },
      "2026-08-01-shipped-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** WAIVED\n\n**Reasoning:** acceptable\n\n## Human Override\n\n**Human decision (fill in):** accept — Bob — 2026-09-24\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_UNCOUNTED, specs });
    const r = runGuard(p.root);
    assert("waived: exit 1 (completion drift)", r.code === 1, `got ${r.code}: ${r.stderr}`);
    p.cleanup();
  }

  // 17. FAIL verdict → not shippable → no completion drift
  {
    const specsFail = {
      "2026-09-01-test-change": {
        "briefing.md": "# Briefing\n",
        "acceptance.feature": "Feature: test\n",
        "blueprint.md": "# Blueprint\n",
        "tasks.md": "# Tasks\n",
        "report.md": "# Report\n\n## Gate Decision\n\n**Verdict:** FAIL\n\n**Reasoning:** tests fail\n",
      },
    };
    const p = makeProject({ stateYaml: STATE_UNCOUNTED, specs: specsFail });
    const r = runGuard(p.root);
    assert("fail-verdict: exit 0 (no completion drift)", r.code === 0, `got ${r.code}: ${r.stderr}`);
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
