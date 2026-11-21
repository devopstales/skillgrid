#!/usr/bin/env node
// test-requirement-coverage.mjs — fixture tests for requirement-coverage.mjs.
//
// Usage: node test-requirement-coverage.mjs
// Exit: 0 all pass, 1 any fail.
//
// Case 4 is the pinned contract fixture: the same literal bytes a future
// second implementation (a Go checker or a host-specific twin) must parse to
// the same covered decision — positive AND negative half. If the fixture text
// drifts from the checker's extraction, this test fails.

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const CHECKER = join(HERE, "requirement-coverage.mjs");

let passed = 0;
let failed = 0;
const failures = [];

function check(name, cond) {
  if (cond) passed++;
  else {
    failed++;
    failures.push(name);
  }
}

function specDir({ briefing, blueprint, tasks, acceptance }) {
  const dir = mkdtempSync(join(tmpdir(), "sgreqcov-"));
  writeFileSync(join(dir, "briefing.md"), briefing);
  writeFileSync(join(dir, "blueprint.md"), blueprint);
  if (tasks !== undefined) writeFileSync(join(dir, "tasks.md"), tasks);
  if (acceptance !== undefined) writeFileSync(join(dir, "acceptance.feature"), acceptance);
  return dir;
}

function run(dir) {
  try {
    const out = execFileSync("node", [CHECKER, dir], { encoding: "utf8" });
    return { code: 0, out };
  } catch (e) {
    return { code: e.status, out: (e.stdout || "") + (e.stderr || "") };
  }
}

// --- 1. complete matrix -> exit 0 --------------------------------------------

const goodBriefing = `# B

#### Req 1 — one (locked)

#### Req 2 — two (locked)
`;
const goodBlueprint = `# BP

## Validation

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 1 | h | e | f | go test ./... | wrong output |
| 2 | h | e | f | go test ./... | wrong output |
`;

let d = specDir({ briefing: goodBriefing, blueprint: goodBlueprint });
let r = run(d);
check("complete matrix exits 0", r.code === 0);
check("complete matrix reports pass", r.out.includes('"status":"pass"') || r.out.includes('"status": "pass"'));
rmSync(d, { recursive: true, force: true });

// --- 2. missing row -> exit 1, names the requirement --------------------------

d = specDir({
  briefing: goodBriefing,
  blueprint: goodBlueprint.replace("| 2 | h | e | f | go test ./... | wrong output |\n", ""),
});
r = run(d);
check("missing row exits 1", r.code === 1);
check("missing row names req 2", r.out.includes('"2"'));
rmSync(d, { recursive: true, force: true });

// --- 3. usage error -> exit 2 --------------------------------------------------

r = run(join(tmpdir(), "no-such-change-dir-xyz"));
check("missing change dir exits 2", r.code === 2);

// --- 4. pinned contract fixture (positive + negative) --------------------------
// Literal bytes: three reqs, two locked. A twin implementation over these
// bytes must report the same covered decision both ways.

const contractBriefing = `# Briefing

> **STATUS:** \`draft\` (2026-11-21)
> **Topic:** 2026-11-21-contract-fixture

#### Req 19 — Technical validation plan (locked)

Some text.

#### Req 21 — Scope pushback (locked)

Some text.

#### Req 12 — Ablate always-loaded rules (named follow-up)

Follow-up text.
`;
const contractBlueprint = `# BP

## Validation

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 19 | h | e | f | go test ./... | wrong |
| 21 | h | e | f | go test ./... | wrong |
| 12 | h | e | f | go test ./... | wrong |
`;

d = specDir({ briefing: contractBriefing, blueprint: contractBlueprint });
r = run(d);
check("contract: exits 0 on the pinned fixture", r.code === 0);
check("contract: no findings on the pinned fixture", !r.out.includes('"kind"'));
rmSync(d, { recursive: true, force: true });

d = specDir({
  briefing: contractBriefing,
  blueprint: contractBlueprint.replace("| 21 | h | e | f | go test ./... | wrong |\n", ""),
});
r = run(d);
check(
  "contract: dropping req 21's row fails and names exactly 21",
  r.code === 1 && r.out.includes('"subject": "21"') && !r.out.includes('"subject": "19"')
);
rmSync(d, { recursive: true, force: true });

// --- 5. SATISFIES <-> acceptance.feature: both sides claim each other ----------

const satBriefing = `# B

#### Req 1 — one (locked)
`;
const satBlueprint = `# BP

## Validation

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 1 | h | e | f | go test ./... | wrong |
`;
const satAcceptance = `Feature: demo

#### Scenario: login-works

Given a user
When they log in
Then they are in

#### Scenario: logout-works

Given a user in
When they log out
Then they are out
`;
const satTasks = `# Tasks

### TICKET-01 — login
- **SATISFIES:** login-works

### TICKET-02 — logout
- **SATISFIES:** logout-works
`;

d = specDir({ briefing: satBriefing, blueprint: satBlueprint, tasks: satTasks, acceptance: satAcceptance });
r = run(d);
check("satifies: fully cross-referenced exits 0", r.code === 0);
check("satifies: reports satifies_checked true", r.out.includes('"satifies_checked": true'));
rmSync(d, { recursive: true, force: true });

// --- 6. SATISFIES names a scenario that does not exist -------------------------

d = specDir({
  briefing: satBriefing,
  blueprint: satBlueprint,
  tasks: satTasks.replace("logout-works", "logout-missing"),
  acceptance: satAcceptance,
});
r = run(d);
check(
  "satifies: dangling name fails and names it",
  r.code === 1 && r.out.includes('"kind": "satifies-dangling"') && r.out.includes('"subject": "logout-missing"')
);
rmSync(d, { recursive: true, force: true });

// --- 7. a scenario no ticket claims --------------------------------------------

d = specDir({
  briefing: satBriefing,
  blueprint: satBlueprint,
  tasks: satTasks.replace("### TICKET-02 — logout\n- **SATISFIES:** logout-works\n", ""),
  acceptance: satAcceptance,
});
r = run(d);
check(
  "satifies: unclaimed scenario fails and names it",
  r.code === 1 && r.out.includes('"kind": "scenario-unclaimed"') && r.out.includes('"subject": "logout-works"')
);
rmSync(d, { recursive: true, force: true });

// --- 8. no tasks.md / no acceptance.feature -> cross-check skipped --------------

d = specDir({ briefing: satBriefing, blueprint: satBlueprint, tasks: satTasks });
r = run(d);
check("satifies: absent acceptance.feature skips the check", r.code === 0 && r.out.includes('"satifies_checked": false'));
rmSync(d, { recursive: true, force: true });

d = specDir({ briefing: satBriefing, blueprint: satBlueprint, acceptance: satAcceptance });
r = run(d);
check("satifies: absent tasks.md skips the check", r.code === 0 && r.out.includes('"satifies_checked": false'));
rmSync(d, { recursive: true, force: true });

// --- report --------------------------------------------------------------------

console.log(`\nrequirement-coverage tests: ${passed} passed, ${failed} failed`);
if (failed > 0) {
  console.error(failures.join("\n"));
  process.exit(1);
}
