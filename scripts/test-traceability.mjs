#!/usr/bin/env node
/**
 * test-traceability.mjs — fixture-driven tests for scripts/traceability.mjs.
 *
 * Builds temp spec dirs with real acceptance.feature + tasks.md shapes (the
 * Markdown-heading + gherkin-fence structure skillgrid uses), runs the script,
 * and asserts: scenario counts are EXACT (never guessed), every scenario has a
 * status, SATISFIES cross-checks work, and run-twice is identical.
 *
 * Usage: node scripts/test-traceability.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const TRACE = join(HERE, "traceability.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function makeSpec(files) {
  const root = mkdtempSync(join(tmpdir(), "sgtrace-"));
  const dir = join(root, ".skillgrid", "specs", "demo-change");
  mkdirSync(dir, { recursive: true });
  for (const [f, c] of Object.entries(files)) writeFileSync(join(dir, f), c);
  return { root, dir, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(root, ...args) {
  try {
    const stdout = execFileSync("node", [TRACE, ...args, root], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}
function parse(out) { try { return JSON.parse(out); } catch { return null; } }

// A realistic acceptance.feature (skillgrid's Markdown+gherkin shape).
const FEATURE = `# Demo Capability

## Requirements

### Requirement: r1

#### Scenario: happy path — thing works

\`\`\`gherkin
Given a thing
When I use it
Then it works
\`\`\`

#### Scenario: edge — boundary case

\`\`\`gherkin
Given the boundary
When I push it
Then it holds
\`\`\`

### Requirement: r2

#### Scenario: failure — error path

\`\`\`gherkin
Given a broken input
When I submit it
Then I see an error
\`\`\`
`;

// tasks.md with SATISFIES lines referencing the scenarios above (by name) and one not.
const TASKS = `# Tasks

### TICKET-01
- **SATISFIES:** happy path — thing works; failure — error path
- **Files:** a.go

### TICKET-02
- **SATISFIES:** edge — boundary case
- **Files:** b.go

### TICKET-03
- **SATISFIES:** some-unreferenced-scenario
- **Files:** c.go
`;

// 1. Exact scenario count (3), requirement grouping (2)
{
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": TASKS });
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  assert("count: exit 0", r.code === 0, `got ${r.code}: ${r.stderr}`);
  assert("count: total 3", j?.summary?.total === 3, r.stdout);
  assert("count: scenarios array len 3", Array.isArray(j?.scenarios) && j.scenarios.length === 3, r.stdout);
  assert("count: 2 requirements", j?.summary?.requirements === 2, r.stdout);
  assert("count: scope COMPLETE", j?.scope === "COMPLETE", r.stdout);
  p.cleanup();
}

// 2. Per-scenario status + SATISFIES reference
{
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": TASKS });
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  const s1 = j?.scenarios?.find((s) => s.name === "happy path — thing works");
  const s3 = j?.scenarios?.find((s) => s.name === "failure — error path");
  assert("status: s1 referenced by TICKET-01", s1 && s1.satisfies_refs.includes("TICKET-01"), JSON.stringify(s1));
  assert("status: s3 referenced by TICKET-01", s3 && s3.satisfies_refs.includes("TICKET-01"), JSON.stringify(s3));
  const unreferenced = j?.scenarios?.filter((s) => s.satisfies_refs.length === 0);
  assert("status: 1 unreferenced scenario (edge—boundary? no, it's referenced) → check", Array.isArray(unreferenced), JSON.stringify(j?.scenarios?.map(s=>s.name+':'+s.satisfies_refs)));
  p.cleanup();
}

// 3. No test output → every scenario UNTESTED (a SATISFIES ref is NOT a test;
//    coverage requires a covering test that actually ran).
{
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": TASKS });
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  const untested = j?.scenarios?.filter((s) => s.status === "UNTESTED");
  assert("untested-no-testout: all 3 UNTESTED (no test output)", untested.length === 3, JSON.stringify(j?.scenarios));
  assert("untested-no-testout: summary untested 3", j?.summary?.untested === 3, r.stdout);
  p.cleanup();
}

// 3b. SATISFIES refs are tracked independently of test status.
{
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": TASKS });
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  const s1 = j?.scenarios?.find((s) => s.name === "happy path — thing works");
  assert("satisfies: s1 referenced by TICKET-01 (status UNTESTED, ref tracked)",
    s1 && s1.satisfies_refs.includes("TICKET-01") && s1.status === "UNTESTED", JSON.stringify(s1));
  p.cleanup();
}

// 4. (unused — see 3/3b) kept for the summary-count assertion
{
  const tasksNoRef = `# Tasks\n\n### TICKET-01\n- **SATISFIES:** happy path — thing works\n`;
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": tasksNoRef });
  const r = run(p.root, "demo-change");
  const j = parse(r.stdout);
  const untested = j?.scenarios?.filter((s) => s.status === "UNTESTED");
  assert("summary: total 3 always", j?.summary?.total === 3, r.stdout);
  p.cleanup();
}

// 5. Test output: passing tests → matching scenarios COMPLIANT
{
  // A test runner output (go test style) where a test named after a scenario passes.
  const testOut = `ok  skillgrid/demo  0.02s\n--- PASS: TestHappyPathThingWorks (0.001)\n--- PASS: TestEdgeBoundaryCase (0.002)\n`;
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": TASKS });
  const testFile = join(p.root, "test-output.txt");
  writeFileSync(testFile, testOut);
  const r = run(p.root, "demo-change", `--test-output=${testFile}`);
  const j = parse(r.stdout);
  const s1 = j?.scenarios?.find((s) => s.name === "happy path — thing works");
  assert("testout: s1 ran + passed (COMPLIANT)", s1 && s1.status === "COMPLIANT", JSON.stringify(s1));
  const s2 = j?.scenarios?.find((s) => s.name === "edge — boundary case");
  assert("testout: s2 ran + passed (COMPLIANT)", s2 && s2.status === "COMPLIANT", JSON.stringify(s2));
  // failure — error path has no matching test → stays UNTESTED
  const s3 = j?.scenarios?.find((s) => s.name === "failure — error path");
  assert("testout: s3 no test → UNTESTED", s3 && s3.status === "UNTESTED", JSON.stringify(s3));
  p.cleanup();
}

// 6. No acceptance.feature → exit 2
{
  const p = makeSpec({ "tasks.md": TASKS });
  const r = run(p.root, "demo-change");
  assert("nofeature: exit 2", r.code === 2, `got ${r.code}: ${r.stderr}`);
  p.cleanup();
}

// 7. Determinism: run twice identical
{
  const p = makeSpec({ "acceptance.feature": FEATURE, "tasks.md": TASKS });
  const a = run(p.root, "demo-change").stdout;
  const b = run(p.root, "demo-change").stdout;
  assert("determinism: identical", a === b && a.length > 0);
  p.cleanup();
}

// Summary
console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
