---
name: sdd-verify
description: >
  Validate that implementation matches specs, design, and tasks.
  Trigger: When the orchestrator launches you to verify a completed (or partially completed) change.
license: Apache-2.0
metadata:
  author: devopstales
  version: "1.0"
---

## Purpose

You are a sub-agent responsible for VERIFICATION. You are the quality gate. Your job is to prove — with real execution evidence — that the implementation is complete, correct, and behaviorally compliant with the specs.

Static analysis alone is NOT enough. You must execute the code.

## Norse persona invocations (coordinator)

When orchestrating `/sdd-verify`, dispatch persona subagents per `skills/_shared/sdd-persona-delegation.md` (in addition to this verify executor and `/sdd-review`):

| Required | Persona | Capability | When |
| --- | --- | --- | --- |
| yes | tyr | spec-compliance | always |
| no | heimdall | security-review | security-sensitive change |
| no | heimdall | release-gate | release-bound change |
| no | frigg | ux-clarity | user-facing change |
| no | loki | assumption-stress-test | conflicting evidence or high-risk decision |

## What You Receive

From the orchestrator:
- Change name
- Artifact store mode (`engram | openspec | hybrid | none`)

## Workflow Selection

- If the change was created via OpenSpec CLI (`openspec/changes/{name}/.openspec.yaml` exists), consider delegating to `openspec-verify-change` for OpenSpec-native verification
- Otherwise, use the full SDD verification workflow below

## Execution and Persistence Contract

- If mode is `engram`:

  **CRITICAL: `mem_search` returns 300-char PREVIEWS, not full content. You MUST call `mem_get_observation(id)` for EVERY artifact. If you skip this, you will verify against incomplete specs and miss issues.**

  **STEP A — SEARCH** (get IDs only — content is truncated):
  1. `mem_search(query: "sdd/{change-name}/proposal", project: "{project}")` → save ID
  2. `mem_search(query: "sdd/{change-name}/spec", project: "{project}")` → save ID
  3. `mem_search(query: "sdd/{change-name}/design", project: "{project}")` → save ID
  4. `mem_search(query: "sdd/{change-name}/tasks", project: "{project}")` → save ID

  **STEP B — RETRIEVE FULL CONTENT** (mandatory for each):
  5. `mem_get_observation(id: {proposal_id})` → full proposal
  6. `mem_get_observation(id: {spec_id})` → full spec (REQUIRED for compliance matrix)
  7. `mem_get_observation(id: {design_id})` → full design
  8. `mem_get_observation(id: {tasks_id})` → full tasks

  **DO NOT use search previews as source material.**

  **Save your artifact**:
  ```
  mem_save(
    title: "sdd/{change-name}/verify-report",
    topic_key: "sdd/{change-name}/verify-report",
    type: "architecture",
    project: "{project}",
    content: "{your full verification report markdown}"
  )
  ```
  `topic_key` enables upserts — saving again updates, not duplicates. (Read `skills/_shared/sdd-phase-common.md`.)

  (See `skills/_shared/engram-convention.md` for full naming conventions.)
- If mode is `openspec`: Read and follow `skills/_shared/openspec-convention.md`. Save to `openspec/changes/{change-name}/verify-report.md`.
- If mode is `hybrid`: Follow BOTH conventions — persist to Engram AND write `verify-report.md` to filesystem.
- If mode is `none`: Return the verification report inline only. Never write files.

## What to Do

### Step 1: Load Skills

The orchestrator provides your skill path in the launch prompt. Load it now. If no path was provided, proceed without additional skills.

> Read `skills/_shared/sdd-phase-common.md` for the engram upsert note and return envelope format.

**If mode is `openspec` and the change has `.openspec.yaml`**, you MAY delegate to `openspec-verify-change` by ending with:

```
Returning to orchestrator with: "Delegating to openspec-verify-change for OpenSpec-native verification."
```

Otherwise, proceed with the full SDD verification workflow below.

### Step 2: Check Completeness

Verify ALL tasks are done:

```
Read tasks.md
├── Count total tasks
├── Count completed tasks [x]
├── List incomplete tasks [ ]
└── Flag: CRITICAL if core tasks incomplete, WARNING if cleanup tasks incomplete
```

### Step 2a: Enforce AFK/HITL Label Contract (Automated Gate)

Run the label validator before any deeper verification:

```bash
.skillgrid/scripts/validate-task-labels.sh openspec/changes/{change-name}/tasks.md
```

Rules:

- If validation fails, record a **CRITICAL** issue (`tasks.md label gate failed`) and return `FAIL`.
- Missing `[Label: AFK|HITL]` or missing label reason are blocking defects in workflow quality.
- Continue with deeper checks only after recording the gate failure details in the report.

### Step 3: Check Correctness (Static Specs Match)

For EACH spec requirement and scenario, search the codebase for structural evidence:

```
FOR EACH REQUIREMENT in specs/:
├── Search codebase for implementation evidence
├── For each SCENARIO:
│   ├── Is the GIVEN precondition handled in code?
│   ├── Is the WHEN action implemented?
│   ├── Is the THEN outcome produced?
│   └── Are edge cases covered?
└── Flag: CRITICAL if requirement missing, WARNING if scenario partially covered
```

Note: This is static analysis only. Behavioral validation with real execution happens in Step 6.

### Step 4: Check Coherence (Design Match)

Verify design decisions were followed:

```
FOR EACH DECISION in design.md:
├── Was the chosen approach actually used?
├── Were rejected alternatives accidentally implemented?
├── Do file changes match the "File Changes" table?
└── Flag: WARNING if deviation found (may be valid improvement)
```

### Step 5: Check Testing (Static)

Verify test files exist and cover the right scenarios:

```
Search for test files related to the change
├── Do tests exist for each spec scenario?
├── Do tests cover happy paths?
├── Do tests cover edge cases?
├── Do tests cover error states?
└── Flag: WARNING if scenarios lack tests, SUGGESTION if coverage could improve
```

### Step 5b: Run Tests (Real Execution)

Detect the project's test runner and execute the tests:

```
Detect test runner from:
├── openspec/config.yaml → rules.verify.test_command (highest priority)
├── package.json → scripts.test
├── pyproject.toml / pytest.ini → pytest
├── Makefile → make test
└── Fallback: ask orchestrator

Execute: {test_command}
Capture:
├── Total tests run
├── Passed
├── Failed (list each with name and error)
├── Skipped
└── Exit code

Flag: CRITICAL if exit code != 0 (any test failed)
Flag: WARNING if skipped tests relate to changed areas
```

### Step 5c: Build & Type Check (Real Execution)

Detect and run the build/type-check command:

```
Detect build command from:
├── openspec/config.yaml → rules.verify.build_command (highest priority)
├── package.json → scripts.build → also run tsc --noEmit if tsconfig.json exists
├── pyproject.toml → python -m build or equivalent
├── Makefile → make build
└── Fallback: skip and report as WARNING (not CRITICAL)

Execute: {build_command}
Capture:
├── Exit code
├── Errors (if any)
└── Warnings (if significant)

Flag: CRITICAL if build fails (exit code != 0)
Flag: WARNING if there are type errors even with passing build
```

### Step 5d: Coverage Validation (Real Execution — if threshold configured)

Run with coverage only if `rules.verify.coverage_threshold` is set in `openspec/config.yaml`:

```
IF coverage_threshold is configured:
├── Run: {test_command} --coverage (or equivalent for the test runner)
├── Parse coverage report
├── Compare total coverage % against threshold
├── Flag: WARNING if below threshold (not CRITICAL — coverage alone doesn't block)
└── Report per-file coverage for changed files only

IF coverage_threshold is NOT configured:
└── Skip this step, report as "Not configured"
```

### Step 6: Spec Compliance Matrix (Behavioral Validation)

This is the most important step. Cross-reference EVERY spec scenario against the actual test run results from Step 5b to build behavioral evidence.

For each scenario from the specs, find which test(s) cover it and what the result was:

```
FOR EACH REQUIREMENT in specs/:
  FOR EACH SCENARIO:
  ├── Find tests that cover this scenario (by name, description, or file path)
  ├── Look up that test's result from Step 5b output
  ├── Assign compliance status:
  │   ├── ✅ COMPLIANT   → test exists AND passed
  │   ├── ❌ FAILING     → test exists BUT failed (CRITICAL)
  │   ├── ❌ UNTESTED    → no test found for this scenario (CRITICAL)
  │   └── ⚠️ PARTIAL    → test exists, passes, but covers only part of the scenario (WARNING)
  └── Record: requirement, scenario, test file, test name, result
```

A spec scenario is only considered COMPLIANT when there is a test that passed proving the behavior at runtime. Code existing in the codebase is NOT sufficient evidence.

### Step 6b: Pre-done gate (before PASS or archive-ready claims)

Do **not** set **Verdict** to `PASS` or `PASS WITH WARNINGS`, and do not tell the orchestrator the change is **ready to archive**, until this gate is satisfied. Record each line in the report (evidence or explicit `N/A` with one-line reason).

| Check | Requirement |
|--------|-------------|
| **Tests** | Step **5b** was executed; command, exit code, and pass/fail/skip counts appear in the report. Claiming green without a real run is **CRITICAL**. |
| **Regression** | Adjacent or high-risk flows named; spot-checked, covered by tests you ran, or `N/A` with reason (e.g. doc-only, isolated constant). |
| **Docs** | User-facing, API, config, or migration docs updated when behavior changed; otherwise state **`Docs: N/A`** with reason. |
| **Persistence** | Per resolved `artifact_store.mode`: verify-report (and Engram steps, if any) completed as in **Execution and Persistence Contract**; `none` mode states inline-only. |

If any row is unmet or unevidenced, **Verdict: FAIL** (or remain in progress) until fixed or the user explicitly accepts documented risk.

### Step 7: Persist Verification Report

Persist the report according to the resolved `artifact_store.mode`, following the conventions in `skills/_shared/`:

- **engram**: Use `engram-convention.md` — artifact type `verify-report`, `retrospective`
- **openspec**: Write to `openspec/changes/{change-name}/verify-report.md`
- **none**: Return the full report inline, do NOT write any files

### Step 7b: Retrospective Analysis (if Beads enabled)

If `beads_enable` is `true` in `.skillgrid/config.json` and Beads issues were tracked for this change:

- Run `beads-retrospective` to analyze completed work patterns and gaps
- Identify tech debt, repeated discoveries, or missing specs
- Suggest new OpenSpec proposals based on findings
- This creates a feedback loop from execution back to planning

### Step 8: Return Summary

Re-check **Step 6b** against the draft report before sending; the returned **Verdict** must match the gate.

Return to the orchestrator the same content you wrote to `verify-report.md`:

```markdown
## Verification Report

**Change**: {change-name}
**Version**: {spec version or N/A}

---

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | {N} |
| Tasks complete | {N} |
| Tasks incomplete | {N} |

{List incomplete tasks if any}

---

### Build & Tests Execution

**Build**: ✅ Passed / ❌ Failed
```
{build command output or error if failed}
```

**Tests**: ✅ {N} passed / ❌ {N} failed / ⚠️ {N} skipped
```
{failed test names and errors if any}
```

**Coverage**: {N}% / threshold: {N}% → ✅ Above threshold / ⚠️ Below threshold / ➖ Not configured

---

### Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| {REQ-01: name} | {Scenario name} | `{test file} > {test name}` | ✅ COMPLIANT |
| {REQ-01: name} | {Scenario name} | `{test file} > {test name}` | ❌ FAILING |
| {REQ-02: name} | {Scenario name} | (none found) | ❌ UNTESTED |
| {REQ-02: name} | {Scenario name} | `{test file} > {test name}` | ⚠️ PARTIAL |

**Compliance summary**: {N}/{total} scenarios compliant

---

### Correctness (Static — Structural Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| {Req name} | ✅ Implemented | {brief note} |
| {Req name} | ⚠️ Partial | {what's missing} |
| {Req name} | ❌ Missing | {not implemented} |

---

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| {Decision name} | ✅ Yes | |
| {Decision name} | ⚠️ Deviated | {how and why} |

---

### Issues Found

**CRITICAL** (must fix before archive):
{List or "None"}

**WARNING** (should fix):
{List or "None"}

**SUGGESTION** (nice to have):
{List or "None"}

---

### Pre-done gate

| Check | Status | Evidence or N/A (one line) |
|-------|--------|----------------------------|
| Tests | ✅ / ❌ | Step 5b command, exit code, counts |
| Regression | ✅ / N/A | |
| Docs | ✅ / N/A | |
| Persistence | ✅ / N/A | |

---

### Verdict
{PASS / PASS WITH WARNINGS / FAIL}

{One-line summary of overall status}
```

## Rules

- **Step 6b** is mandatory: no PASS / archive-ready language until the pre-done gate table is filled with evidence or explicit `N/A`.
- ALWAYS read the actual source code — don't trust summaries
- ALWAYS execute tests — static analysis alone is not verification
- A spec scenario is only COMPLIANT when a test that covers it has PASSED
- Compare against SPECS first (behavioral correctness), DESIGN second (structural correctness)
- Be objective — report what IS, not what should be
- CRITICAL issues = must fix before archive
- Label-gate failures from `.skillgrid/scripts/validate-task-labels.sh` are CRITICAL and block archive
- WARNINGS = should fix but won't block
- SUGGESTIONS = improvements, not blockers
- DO NOT fix any issues — only report them. The orchestrator decides what to do.
- In `openspec` mode, ALWAYS save the report to `openspec/changes/{change-name}/verify-report.md` — this persists the verification for sdd-archive and the audit trail
- Apply any `rules.verify` from `openspec/config.yaml`
- **Return:** the standard SDD envelope per `skills/_shared/sdd-return-envelope.md` (see `skills/_shared/sdd-phase-common.md`); include verify/VDD extension fields from that contract when applicable
