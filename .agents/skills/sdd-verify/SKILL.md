---
name: sdd-verify
description: "Verify an SDD change by executing tests and proving the implementation matches change.md, change-level acceptance.feature (@step-NN), and completed tasks in docs/skillgrid/changes/<NNN-slug>/tasks.md. Use after sdd-apply and before sdd-archive — filling each ## NN-<name> → ### Verification block (Verdict + Evidence table with Run/Expected/Result) from real execution, auditing assertion quality (Strict TDD), and persisting PASS/FAIL verdicts. Does NOT write steps/.../verification.md. Marks nothing as done without runtime evidence. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "3.0"
  family: sdd
  part-of: skillgrid
  phase-order: "init → explore → propose → spec → apply → verify → archive"
  prev: [sdd-apply]
  next: [sdd-archive]
  artifact: verification-in-tasks
  delegate_only: true
---

# sdd-verify

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-verify` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-verify` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the VERIFY phase — an **independent quality gate**, run **per step section** inside change-level `tasks.md`. You prove, with source inspection **plus real execution evidence**, that the implementation matches change-level `acceptance.feature` Features tagged `@step-NN` (WHAT), `change.md` (HOW), and the completed task set under `## NN-<name>`. You compare **acceptance first, change.md second, task completion third**, run the actual test and build commands, and fill each step's `### Verification` block with Verdict `PASS`, `PASS WITH WARNINGS`, or `FAIL` plus an Evidence table (`Run` / `Expected` / `Result`).

**You do NOT write `steps/<NN-name>/verification.md`.** There is no `steps/` directory in v3. The per-step gate lives only in `docs/skillgrid/changes/<NNN-slug>/tasks.md` under `## NN-<name>` → `### Verification`.

**You are the one independent requirements/runtime verification per step.** Your job is to judge, not to fix or to re-derive: an acceptance scenario is compliant **only when a covering test passed at runtime** — static analysis alone is never verification. A contradiction or a new failing check returns `FAIL` for that step and hands back to the orchestrator. You never start a remediation/correction cycle, a refutation pass, or another phase on your own; the orchestrator decides the next step.

Phase order is `init → explore → propose → spec → apply → verify → archive`. You run after `sdd-apply` and before `sdd-archive`. The orchestrator may dispatch you per step, or once to walk every `## NN-<name>` — either way, the artifact is the Verification block inside change-level `tasks.md`.

**Gates that must hold for a step PASS:**

1. Every scenario under the Feature tagged `@step-NN` has a passing run.
2. **Global Constraints** in `tasks.md` are held.
3. No unchecked `- [ ]` under that step's `### Tasks`.
4. Required test/build commands exited 0 (unless preflight-blocked).

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`)
- **The step(s) to verify** — a single `NN-name` (per-step dispatch), or "all" (walk every `## NN-<name>` in `tasks.md`). For each step you verify, the change folder root, artifact paths, task-progress state, and allowed verify scope apply.
- **Strict TDD mode** (`true` | `false`) — if the orchestrator declares `STRICT TDD MODE IS ACTIVE`, treat it as authoritative. If not provided, resolve it in Step 2.
- Optional: a `## Skills to load before work` block.

**Artifact store mode is `hybrid` — the only mode for this phase.** For each step verified, the run does BOTH: fills `### Verification` inside `docs/skillgrid/changes/<NNN-slug>/tasks.md` **and** persists that step's report to Mnemonic under `sdd/<NNN-slug>/verification` (a single concatenated observation of all steps verified this run) plus an upsert of `sdd/<NNN-slug>/tasks` so memory matches disk. The filesystem write and the Mnemonic save are each their own obligations — the Mnemonic save does not stand in for the file. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout; Verification lives inside each `## NN-<name>` in `tasks.md`; `rules.verify` from `docs/skillgrid/config.yaml`; archive gates on every step Verdict in `tasks.md`.
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md) — canonical `### Verification` Evidence table shape (Verdict + Run/Expected/Result).
- [references/strict-tdd.md](references/strict-tdd.md) — the apply-phase TDD cycle and assertion-quality rules you audit in Step 6 (local copy for a self-contained verify skill).
- [`references/report-format.md`](references/report-format.md) — compliance statuses, self-check discipline, and blocked-preflight shape. **Adapt** the body into the `### Verification` block in `tasks.md` — do **not** create a separate `verification.md` file.
- [`references/strict-tdd-verify.md`](references/strict-tdd-verify.md) — the Strict TDD verify module; loaded **only** when Step 2 resolves Strict TDD as active.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover all artifacts from Mnemonic and the change folder (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/change")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; change-level `acceptance.feature` — count `@step-NN` scenarios.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; `[x]` state + Verification stubs.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")` → `..._mem_get_observation(id)` — the apply evidence (incl. the TDD Cycle Evidence table and per-step Step Evidence if Strict TDD was active).
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in `docs/skillgrid/changes/<NNN-slug>/`:
   - `change.md`
   - `tasks.md` (all `## NN-<name>` sections, Global Constraints, State)
   - `acceptance.feature` (all Features `@step-NN`)
4. Read `docs/skillgrid/config.yaml` if present — `context:`, `rules.verify` (test/build commands, coverage threshold), and the `tdd` flag bind this phase.

## Status Guard

Before running tests, confirm readiness from the structured state (orchestrator-provided, or `## State` in `tasks.md`):

- For each step to verify, if some assigned task under that `## NN-<name>` → `### Tasks` is still `[ ]`, **do not run the full suite for that step** — mark that step `blocked` naming the incomplete tasks. Focused single-unit checks remain an apply responsibility.
- If `acceptance.feature` or `tasks.md` is missing, mark the change `blocked` (see the Blocked Preflight shape in `references/report-format.md`) rather than guessing completion.
- If a step's `## NN-<name>` section is missing from `tasks.md`, mark that step `blocked`.
- If the verify scope is unsafe (you cannot run the tests for the step — no detectable runner, read-only workspace where the harness needs to write artifacts), STOP and report it.

## What to Do

### Step 1: Enumerate Steps to Verify

Determine the step set from change-level `tasks.md`:

- Single-step dispatch: that one `## NN-<name>`.
- "All" dispatch: every `## NN-<name>` section in NN order — **verify a step only if its `Depends on:` predecessors already have Verdict `PASS` or `PASS WITH WARNINGS`** in this run or on disk; otherwise mark the later step `blocked (dependency <NN> not PASS)`.

Map each `## NN-<name>` to Feature `@step-NN` in `acceptance.feature`.

### Step 2: Resolve Testing / TDD Mode

```
Read testing capabilities from:
├── Mnemonic: skillgrid-mnemonic_mem_search("sdd/{project}/testing-capabilities") → mem_get_observation(id)
├── docs/skillgrid/config.yaml → rules.verify (test_command, build_command, coverage_threshold)
└── Fallback: detect from project files directly (package.json, go.mod, pyproject.toml, etc.)

OR the orchestrator already declared it:
├── "STRICT TDD MODE IS ACTIVE" in the launch → authoritative
Resolve:
├── tdd: true AND a runner exists → STRICT TDD VERIFY (load references/strict-tdd-verify.md)
├── tdd: false OR no runner → STANDARD VERIFY (skip TDD checks)
└── Cache the resolved mode for the report
```

If Strict TDD is **not** active, do **not** load or process `references/strict-tdd-verify.md` — no TDD sections appear under Evidence.

### Step 3: Per Step — Compare Acceptance First

For **each** step, count the **actual** scenarios from the Feature tagged `@step-NN` in change-level `acceptance.feature` — never invent totals. For every scenario, map it to:

1. A covering test (file + test name) — found via the code index or the repo.
2. The runtime result of that test (pass/fail) from Step 5 execution.

A scenario with **no covering test** is `UNTESTED` (CRITICAL for a required scenario). A scenario whose covering test **failed** is `FAILING` (CRITICAL). A test that passes but only partially covers the scenario is `PARTIAL` (WARNING). A test that exists and passed is `COMPLIANT`.

A step cannot be `PASS` unless every `@step-NN` scenario is `COMPLIANT` (or an explicitly allowed manual verification per `rules.verify`).

### Step 4: Per Step — Check change.md Coherence (second)

For each decision in **`change.md`** that applies to this step (Architecture Decisions, per-step WHAT, Impacted Files), check it against the changed code:

- Decision **followed** → note it.
- Decision **deviated from** → WARNING, unless the deviation breaks an acceptance scenario (then CRITICAL, and it surfaces in Step 3's compliance matrix anyway).

Also confirm **Global Constraints** from `tasks.md` still hold for this step's changes — a violated Global Constraint is CRITICAL for that step.

If the change artifact is missing, **skip change.md coherence** and record that in the report (`change.md coherence: skipped — no change artifact`). Do not fabricate a comparison.

### Step 5: Per Step — Run Tests, Build, and Coverage

Execute the real commands — static analysis alone is never verification:

```
FOR EACH STEP:
1. Run the test suite (rules.verify.test_command, or the runner's default), filtered to
   the step's @step-NN Feature / scenarios where the runner supports tag/selector
   (e.g. cucumber --tags @step-01, jest -t, go test -run). Prefer Run: lines from
   the step's ### Tasks / Evidence stubs when present.
   ├── Capture: command, exit code, pass/fail/skip counts, and the failure detail.
   └── Preserve the output (it is the execution evidence for the Evidence table).
2. Run build / type-check (rules.verify.build_command, or the project's build step).
   └── Capture: command, exit code, relevant output.
3. IF a coverage tool is available AND rules.verify.coverage_threshold is set:
   ├── Run with coverage; filter to the files CHANGED by this step
   │   (file list from the step's apply-progress entry, tasks.md Files:, or the diff).
   └── Report per-file and aggregate; flag files below the threshold.
4. Quality metrics (linter / type checker) ONLY on changed files, ONLY if the tools
   exist — WARNING for errors, SUGGESTION for warnings, never CRITICAL.
```

Confirm the exact command you ran matches `rules.verify` / the runner / `Run:` lines — do not report the exit code of a command you did not actually run.

### Step 6: (Strict TDD only) Run the TDD Verify Module

Load `references/strict-tdd-verify.md` and run its checks **against the apply-progress TDD Cycle Evidence table** for the steps under verification:

- **Step 6a — TDD Compliance + Assertion Quality Audit**: for each task row, verify the RED test file exists, the GREEN test actually passes now, triangulation was adequate for the scenario count, and the safety net was run for modified files. Then audit every test file changed by this change for trivial assertions (tautologies, orphan-empty, type-only, type-only-alone, ghost loops, smoke-only, implementation-detail coupling, mock-heavy). Tautologies are CRITICAL — they "pass" without proving anything.
- **Step 6b/6c — Test layer distribution + changed-file coverage**: classify every test file the step's slice added/modified (unit / integration / E2E), cross-reference with the available tools, and report changed-file coverage if the tool exists.

Fold Strict TDD sections into Notes under the Evidence table (or an indented subsection under `### Verification`) — still inside `tasks.md`, never a separate file.

### Step 7: Fill `### Verification` in tasks.md

For each verified step, replace the stub under `## NN-<name>` → `### Verification` per [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md):

```markdown
### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `{cmd}` | PASS | PASS | |
| Acceptance `@step-NN` / scenario … | `{cmd}` | PASS | PASS | COMPLIANT |
| Runtime harness | `{cmd}` | PASS | PASS | |
| Rollback boundary | `{cmd or manual}` | PASS | PASS | |
| Global Constraints | — | held | held | |
| Build | `{cmd}` | PASS | PASS | exit 0 |
```

Also record (in Notes or a short prose block under Evidence, still in the same section):

- Completeness: tasks done/total for this step.
- Acceptance compliance matrix (scenario → status) using statuses from `references/report-format.md`.
- change.md coherence notes; CRITICAL / WARNING / SUGGESTION issues.
- If Strict TDD: compliance / assertion-quality summary.

**Do not** create `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/verification.md` or any other per-step file.

Bump `## State` (`phase: verify`, `current_step`, `status`, `updated`). Update the STATUS banner when useful (e.g. `N/M steps PASS`).

For missing artifacts, include the skipped dimensions rather than an empty section (see Graceful Artifact Handling below).

### Step 8: Self-Check Before Persisting (replaces the external validator binary)

Run these checks (adapted from `references/report-format.md`); a failure means fix the Verification block, don't persist an inconsistent one:

1. Every step verified has Verdict + Evidence table with Run/Expected/Result columns filled.
2. Scenario counts **equal** the count of scenarios in the `@step-NN` Feature — not a guessed or rounded number.
3. Every `CRITICAL` finding names a file / test / command / exit code that Step 5 ran (or the step's apply-progress shows) — no finding floats free of evidence.
4. Every scenario lands in **exactly one** compliance status (COMPLIANT / PARTIAL / FAILING / UNTESTED); nothing is both tested and untested.
5. Unchecked tasks in the step are `CRITICAL`, even if other artifacts are missing or the rest is warnings-only.
6. Verdict consistent with blockers: `FAIL` iff ≥1 CRITICAL or ≥1 unchecked task or a required test/build command exited non-zero or a Global Constraint is violated; `PASS` only if zero CRITICAL and all tasks complete and all required-test commands exited 0 and Global Constraints held; `PASS WITH WARNINGS` when WARNINGs exist but no CRITICAL.
7. Evidence `Result` values match the **real** exit codes / outcomes of the commands you ran.

If any check fails, fix it before persisting; if you cannot, return `partial` and leave the prior Verification content untouched for that step.

### Step 9: Persist Reports (hybrid — MANDATORY, do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — updated `docs/skillgrid/changes/<NNN-slug>/tasks.md` with filled `### Verification` blocks (Step 7). A new verdict supersedes the prior PENDING/old Verdict in place.
2. **Mnemonic** — start one session, then save the concatenated verification report AND upsert tasks:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/verify")

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/verification",
  topic_key:  "sdd/<NNN-slug>/verification",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "## Step 01-{name}\n{Verdict + Evidence + notes}\n\n## Step 02-{name}\n…"
)

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/tasks",
  topic_key:  "sdd/<NNN-slug>/tasks",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{full tasks.md including filled ### Verification blocks and ## State}"
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. `topic_key` upserts — re-running verify replaces the observation in place. A `FAIL` verdict is a **valid, persistable** result — persist it, do not discard a failed report.

The file must actually exist on disk at `docs/skillgrid/changes/<NNN-slug>/tasks.md` — a Mnemonic save without the file is incomplete. There must be **no** `steps/.../verification.md` written by this phase.

### Step 10: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 9) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Step Verification Report
**Change**: {NNN-slug}
**Steps verified**: {NN-…, …}   (or: single step {NN-…})
**Version**: {spec version or N/A}
**Mode**: {Strict TDD | Standard}
**Location**: `docs/skillgrid/changes/<NNN-slug>/tasks.md` (`## NN-<name>` → `### Verification`) · Mnemonic `sdd/<NNN-slug>/verification` + `sdd/<NNN-slug>/tasks` (hybrid)
**Status**: success (verified) | partial | blocked

### Per-Step Verdict
| Step | Verdict | Scenarios (pass/total) | Tasks (done/total) | Global Constraints |
|---|---|---|---|---|
| 01-{name} | {PASS \| PASS WITH WARNINGS \| FAIL} | {n}/{n} | {n}/{n} | held \| violated |
| 02-{name} | {PASS \| …} | {n}/{n} | {n}/{n} | … |

### Execution Evidence (per step)
- Step 01: Tests: `{command}` → exit `{code}` · Build: `{command}` → exit `{code}`
- Step 02: Tests: `{command}` → exit `{code}` · Build: `{command}` → exit `{code}`

### Issues (per step)
- **Step 01 — CRITICAL**: {list or None} · WARNING: {…} · SUGGESTION: {…}
- **Step 02 — CRITICAL**: {list or None} · WARNING: {…} · SUGGESTION: {…}

{IF Strict TDD Mode → per-step TDD Compliance · Test Layer Distribution · Changed File Coverage · Assertion Quality · Quality Metrics (from references/strict-tdd-verify.md)}

### Skipped Dimensions
{Step 01: change.md coherence skipped — no change artifact | Step 02: dependency 01 not PASS → blocked | …}

**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-archive (all steps PASS / PASS WITH WARNINGS) | orchestrator decides the remediation path (any step FAIL)
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Graceful Artifact Handling

Verification degrades per step as artifacts are missing — never invent a comparison you cannot support:

- **Tasks only** (no acceptance/change): verify objective task completion only. If all tasks are checked and runtime evidence exists, `PASS WITH WARNINGS` for task completion is the ceiling. Do not claim acceptance correctness or change.md coherence.
- **Tasks + acceptance**: verify completeness **and** scenario correctness. Missing covering tests are CRITICAL for required scenarios unless `rules.verify` explicitly allows manual verification.
- **Tasks + acceptance + change.md**: verify completeness, correctness, and coherence — the full per-step matrix.
- **Any unchecked task in a step**: always CRITICAL for that step, even when other artifacts are missing and the rest is warnings-only.
- **change.md missing**: skip change.md coherence for that step and record it in Skipped Dimensions.
- **Global Constraints violated**: CRITICAL for the step.
- **No runner**: report it in Evidence as a preflight block (see `references/report-format.md`) with the declared commands noted as not-executed — do not substitute source inspection for the missing runtime evidence.

## Rules

- ALWAYS read all available artifacts before judging — per step: `@step-NN` acceptance, `change.md`, `tasks.md` section, and the apply evidence.
- ALWAYS run the real test and build commands for each step; static analysis alone is never verification. This phase is the application of the `verification` Iron Law — no PASS claim without fresh execution evidence in the report.
- An acceptance scenario is compliant **only** when a covering test passed at runtime.
- Compare **acceptance first, change.md second, task completion third — per step.**
- Fill `### Verification` inside change-level `tasks.md`. A step without a Verdict is not verified.
- **NEVER** write `steps/<NN-name>/verification.md` or create a `steps/` directory.
- Do NOT fix issues — report them for the orchestrator/user. Your verdict never triggers a repair.
- Count the actual scenarios from each `@step-NN` Feature; never invent totals.
- Record the exact test/build commands, exit codes, and output per step in the Evidence table.
- Persist a `FAIL` report just like a `PASS` — a failed verdict is a result, not a reason to discard the artifact.
- If Strict TDD is resolved active, load `references/strict-tdd-verify.md` and include its sections under Verification; if inactive, never load or reference it.
- **Hybrid is the only mode** — always update filesystem `tasks.md` AND persist to Mnemonic.
- No external binaries. Mnemonic (`mem_*`), the code index (`code_*`), and the project's own test/build/coverage commands are the only tools; no `gentle-ai`, no `gentleman-ai`, no separate admission-attestation binary.
- Model/provider/profile/effort selection stays user-owned; verification never changes them.
- Return envelope per Step 10 — final action is text, not a tool call.

## Gotchas

- **`mem_search` returns 300-char previews.** A preview of a 2000-char change loses most of it — always `mem_get_observation(id)` before you count scenarios or map them to tests.
- **Per-step totals must match the `@step-NN` Feature.** A Verification block that says `scenarios: 4/4` but the Feature has 5 scenarios is an inconsistent artifact — totals equal the actual count for **that step**.
- **A `PASS` with a CRITICAL finding is a `FAIL`.** The step verdict is derived from the blockers, not asserted: `FAIL` iff ≥1 CRITICAL, ≥1 unchecked task in that step, a Global Constraint violated, or a required test/build command exited non-zero.
- **Coverage and quality metrics are informational.** WARNING/SUGGESTION at worst — never CRITICAL. A low coverage % or a linter warning does not, by itself, fail a step.
- **Tautologies are worse than missing tests.** `expect(true).toBe(true)` "passes" and contributes nothing — if you find one in a changed test, flag it CRITICAL (it does not count toward acceptance coverage).
- **Ghost loops are tests that always pass.** An assertion inside a `for` over empty results never runs — audit each with a non-empty companion; flag CRITICAL.
- **Do not trust the apply-progress TDD table blindly.** Cross-reference each reported test file against existence (RED) and re-run it (GREEN) — a GREEN you did not re-execute is not a GREEN.
- **A step's `Depends on` is a verify gate.** A later step whose predecessor is not PASS / PASS WITH WARNINGS is `blocked (dependency)`, not PASS — verify in NN order.
- **Mnemonic save rules**: `title == topic_key`, `scope: "project"`, active `session_id`. No `project:` parameter, no `capture_prompt` field. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- A `FAIL` verdict is **persisted**, not skipped. `sdd-archive` should not run while any step is `FAIL`, but the report must be in `tasks.md` and in Mnemonic so the orchestrator can hand the user the exact evidence of what failed **in that step**.
- If you resolved Strict TDD as active and then cannot run the TDD audit because the apply phase left no TDD Cycle Evidence table, flag CRITICAL (the protocol was not followed) — do not silently fall back to a Standard-verify report.
- **Legacy `verification.md` paths are retired.** If you see docs referring to `steps/.../verification.md`, ignore them for new work — v3 gates live only in `tasks.md`.

## References

- [references/report-format.md](references/report-format.md) — compliance statuses, YAML-envelope self-check discipline, and blocked-preflight recovery shape (adapt into `### Verification`; do not write a separate `verification.md`).
- [references/strict-tdd-verify.md](references/strict-tdd-verify.md) — the Strict TDD verify module (TDD Compliance audit, Assertion Quality audit, Test Layer Distribution, Changed-File Coverage, Quality Metrics). Load only when Step 2 resolves Strict TDD as active.
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) — upstream; its apply-progress artifact (incl. the TDD Cycle Evidence table and per-step Step Evidence) is the primary thing this phase audits.
- [references/strict-tdd.md](references/strict-tdd.md) — the apply-phase TDD cycle + assertion rules you are checking against in Step 6.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its change-level `acceptance.feature` (`@step-NN`) and `tasks.md` Verification stubs are what you fill and measure.
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md) — upstream; its `change.md` decisions and per-step WHAT are what the coherence check maps to.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout; Verification-in-tasks placement; `rules.verify`; the archive step that later consumes these verdicts.
- [`../_shared/templates/template-tasks.md`](../_shared/templates/template-tasks.md) — canonical Evidence table (Run / Expected / Result).
- [`../verification/SKILL.md`](../verification/SKILL.md) — the Iron Law this phase enforces: no completion claim without fresh execution evidence in the current report.
- [`../tdd/SKILL.md`](../tdd/SKILL.md) — the TDD discipline audited in Step 6; the RED/GREEN/REFACTOR cycle whose evidence trail is the assertion-quality input.
- [`../review-reception/SKILL.md`](../review-reception/SKILL.md) — when the verify-report surfaces findings, the receiving-side discipline: verify-first, push back with evidence, one at a time.
