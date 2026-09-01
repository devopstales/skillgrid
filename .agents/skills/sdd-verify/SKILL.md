---
name: sdd-verify
description: "Verify an SDD change: execute tests and prove the implementation matches the specs, design, and tasks. Use to run the quality gate after sdd-apply and before sdd-archive — building the spec-compliance matrix from real execution, auditing assertion quality (Strict TDD), and persisting a pass/fail verify-report. Marks nothing as done without runtime evidence. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  family: sdd
  phase-order: "tasks → apply → verify → archive"
  prev: [sdd-apply]
  next: [sdd-archive]
  artifact: verify-report
  delegate_only: true
---

# sdd-verify

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-verify` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-verify` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the VERIFY phase — an **independent quality gate**. You prove, with source inspection **plus real execution evidence**, that the implementation matches the specs (WHAT), the design (HOW), and the completed task set. You compare **specs first, design second, task completion third**, run the actual test and build commands, and produce a `verify-report` whose verdict is `PASS`, `PASS WITH WARNINGS`, or `FAIL`.

**You are the one independent requirements/runtime verification.** Your job is to judge, not to fix or to re-derive: a spec scenario is compliant **only when a covering test passed at runtime** — static analysis alone is never verification. A contradiction or a new failing check returns `FAIL` and hands back to the orchestrator. You never start a remediation/correction cycle, a refutation pass, or another phase on your own; the orchestrator decides the next step.

Phase order is `propose → design → spec → tasks → apply → verify → archive`. You run after `sdd-apply` and before `sdd-archive`.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case)
- **Structured status** (or enough to build it): the change folder root, the artifact paths, task-progress state, and the allowed verify scope. Use it before judging artifacts.
- **Strict TDD mode** (`true` | `false`) — if the orchestrator declares `STRICT TDD MODE IS ACTIVE`, treat it as authoritative. If not provided, resolve it in Step 3.
- Optional: a `## Skills to load before work` block.

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: writes `openspec/changes/{change-name}/verify-report.md` **and** persists the same report to Mnemonic under `sdd/{change-name}/verify-report`. A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout; `verify-report.md` lives in the change folder; `rules.verify` from `openspec/config.yaml`; the archive step later merges this into `openspec/specs/`.
- [references/strict-tdd.md](references/strict-tdd.md) — the apply-phase TDD cycle and assertion-quality rules you audit in Step 5 (local copy for a self-contained verify skill).
- [`references/report-format.md`](references/report-format.md) — the verify-report template, compliance statuses, and the self-check you run before persisting.
- [`references/strict-tdd-verify.md`](references/strict-tdd-verify.md) — the Strict TDD verify module; loaded **only** when Step 3 resolves Strict TDD as active.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover all artifacts from Mnemonic and the change folder (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/proposal")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/design")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; you count its actual requirements/scenarios.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; you read the `[x]` state.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/apply-progress")` → `..._mem_get_observation(id)` — the apply evidence (incl. the TDD Cycle Evidence table if Strict TDD was active).
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in `openspec/changes/{change-name}/`: `proposal.md`, `design.md`, `specs/{domain}/spec.md`, `tasks.md`, and any existing `verify-report.md`.
4. Read `openspec/config.yaml` if present — `context:`, `rules.verify` (test/build commands, coverage threshold), and the `strict_tdd` flag bind this phase.

## Status Guard

Before running tests, confirm readiness from the structured state (orchestrator-provided, or the `state.yaml` DAG state in the change folder):

- If state is not `all_done` (some assigned task still `[ ]` in `tasks.md`), **do not run the full suite** — return `blocked` naming the incomplete tasks. Focused single-unit checks remain an apply responsibility.
- If the change folder is missing its `tasks.md`, return `blocked` (see the Blocked Preflight shape in `references/report-format.md`) rather than guessing completion.
- If the verify scope is unsafe (you cannot run the tests for the change — no detectable runner, read-only workspace where the harness needs to write artifacts), STOP and report it.

## What to Do

### Step 1: Resolve Testing / TDD Mode

```
Read testing capabilities from:
├── Mnemonic: skillgrid-mnemonic_mem_search("sdd/{project}/testing-capabilities") → mem_get_observation(id)
├── openspec/config.yaml → rules.verify (test_command, build_command, coverage_threshold)
└── Fallback: detect from project files directly (package.json, go.mod, pyproject.toml, etc.)

OR the orchestrator already declared it:
├── "STRICT TDD MODE IS ACTIVE" in the launch → authoritative
Resolve:
├── strict_tdd: true AND a runner exists → STRICT TDD VERIFY (load references/strict-tdd-verify.md)
├── strict_tdd: false OR no runner → STANDARD VERIFY (skip TDD checks)
└── Cache the resolved mode for the report
```

If Strict TDD is **not** active, do **not** load or process `references/strict-tdd-verify.md` — no TDD sections appear in the report.

### Step 2: Compare Specs First

Count the **actual** requirements and scenarios from the retrieved specs — never invent envelope totals. For every spec requirement and every scenario, map it to:

1. A covering test (file + test name) — found via the code index or the repo.
2. The runtime result of that test (pass/fail) from Step 4 execution.

A scenario with **no covering test** is `UNTESTED` (CRITICAL for a required scenario). A scenario whose covering test **failed** is `FAILING` (CRITICAL). A test that passes but only partially covers the scenario is `PARTIAL` (WARNING). A test that exists and passed is `COMPLIANT`.

### Step 3: Check Design Coherence (second)

For each decision in the design, check it against the changed code:

- Decision **followed** → note it.
- Decision **deviated from** → WARNING, unless the deviation breaks a spec scenario (then CRITICAL, and it surfaces in Step 2's compliance matrix anyway).

If the design artifact is missing, **skip design coherence** and record that in the report (`Design coherence: skipped — no design artifact`). Do not fabricate a comparison.

### Step 4: Run Tests, Build, and Coverage

Execute the real commands — static analysis alone is never verification:

```
1. Run the test suite (rules.verify.test_command, or the runner's default).
   ├── Capture: command, exit code, pass/fail/skip counts, and the failure detail.
   └── Preserve the output (it is the execution evidence for the compliance matrix).
2. Run build / type-check (rules.verify.build_command, or the project's build step).
   └── Capture: command, exit code, relevant output.
3. IF a coverage tool is available AND rules.verify.coverage_threshold is set:
   ├── Run with coverage; filter to the files CHANGED by this change
   │   (file list from apply-progress "Files Changed" or the diff).
   └── Report per-file and aggregate; flag files below the threshold.
4. Quality metrics (linter / type checker) ONLY on changed files, ONLY if the tools
   exist — WARNING for errors, SUGGESTION for warnings, never CRITICAL.
```

Confirm the exact command you ran matches `rules.verify` / the runner — do not report the exit code of a command you did not actually run.

### Step 5: (Strict TDD only) Run the TDD Verify Module

Load `references/strict-tdd-verify.md` and run its checks **against the apply-progress TDD Cycle Evidence table**:

- **Step 5a — TDD Compliance + Assertion Quality Audit**: for each task row, verify the RED test file exists, the GREEN test actually passes now, triangulation was adequate for the scenario count, and the safety net was run for modified files. Then audit every test file changed by this change for trivial assertions (tautologies, orphan-empty, type-only, type-only-alone, ghost loops, smoke-only, implementation-detail coupling, mock-heavy). Tautologies are CRITICAL — they "pass" without proving anything.
- **Step 5b/5c — Test layer distribution + changed-file coverage**: classify every test file the change added/modified (unit / integration / E2E), cross-reference with the available tools, and report changed-file coverage if the tool exists.

The report gains the TDD Compliance, Test Layer Distribution, Changed File Coverage, Assertion Quality, and Quality Metrics sections (template in `references/strict-tdd-verify.md`).

### Step 6: Build the Report

Assemble the full report per [`references/report-format.md`](references/report-format.md):

- The YAML **envelope** first (`schema: skillgrid.verify-result/v1`, verdict, blockers, critical_findings, the **actual** `requirements` / `scenarios` totals you counted, and the exact test/build commands + exit codes).
- `## Verification Report` — completeness table, build/tests/coverage evidence, spec compliance matrix, correctness table, design coherence table, issues grouped **CRITICAL / WARNING / SUGGESTION**, and the final verdict.
- If Strict TDD is active, insert the extra sections from Step 5.
- For missing artifacts, include the skipped dimensions rather than an empty section (see Graceful Artifact Handling below).

### Step 7: Self-Check Before Persisting (replaces the external validator binary)

Run the checks in `references/report-format.md`; a failure means fix the report, don't persist an inconsistent one:

1. Every envelope field present, exactly once, non-contradictory (a `pass` with a `CRITICAL` finding is a `fail`).
2. `requirements` and `scenarios` totals **equal** the counts you actually retrieved in Step 2 — not a guessed or rounded number.
3. Every `CRITICAL` finding names a file / test / command / exit code that Step 4 ran (or the apply-progress table shows) — no finding floats free of evidence.
4. Every spec scenario lands in **exactly one** compliance status (COMPLIANT / PARTIAL / FAILING / UNTESTED); nothing is both tested and untested.
5. Unchecked tasks are `CRITICAL`, even if other artifacts are missing or the rest is warnings-only.
6. Verdict consistent with blockers: `FAIL` iff ≥1 CRITICAL or ≥1 unchecked task or a test/build command exited non-zero; `PASS` only if zero CRITICAL and all tasks complete and all required-test commands exited 0.
7. `test_exit_code` / `build_exit_code` are the **real** exit codes of the commands you ran (or `125` for commands that did not run in a preflight-blocked case), and `test_command` / `build_command` name them literally.

If any check fails, fix it before persisting; if you cannot, return `partial` and leave the prior `verify-report` untouched.

### Step 8: Persist Report (hybrid — MANDATORY, do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — `openspec/changes/{change-name}/verify-report.md` (already assembled in Step 6). If a prior `verify-report.md` exists, it is replaced by the newly admitted bytes (a new verdict supersedes the old one; the prior report is not kept as a second file).
2. **Mnemonic** — start one session, then save the same content:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/verify")

skillgrid-mnemonic_mem_save(
  title:      "sdd/{change-name}/verify-report",
  topic_key:  "sdd/{change-name}/verify-report",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{full verify-report markdown, envelope + body}"
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. `topic_key` upserts — re-running verify replaces the observation in place. A `fail` verdict is a **valid, persistable** result — persist it, do not discard a failed report.

### Step 9: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 8) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Verification Report
**Change**: {change-name}
**Version**: {spec version or N/A}
**Mode**: {Strict TDD | Standard}
**Location**: `openspec/changes/{change-name}/verify-report.md` · Mnemonic `sdd/{change-name}/verify-report` (hybrid)
**Status**: success (verified) | partial | blocked

### Verdict
{PASS | PASS WITH WARNINGS | FAIL}
{one-line reason}

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | {N} |
| Tasks complete | {N} |
| Tasks incomplete | {N} |

### Execution Evidence
- Tests: `{command}` → exit `{code}` · {N pass / M fail / K skip}
- Build: `{command}` → exit `{code}`
- Coverage: {N}% / threshold {N}% → {above | below | n/a}

### Spec Compliance
{N}/{total} scenarios compliant · {N} PARTIAL · {N} FAILING · {N} UNTESTED

### Issues
- **CRITICAL**: {list or None}
- **WARNING**: {list or None}
- **SUGGESTION**: {list or None}

{IF Strict TDD Mode → TDD Compliance · Test Layer Distribution · Changed File Coverage · Assertion Quality · Quality Metrics (from references/strict-tdd-verify.md)}

### Skipped Dimensions
{Design coherence / Spec compliance / TDD — list those with no artifact to compare against, or "None"}

**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-archive (PASS) | orchestrator decides the remediation path (FAIL)
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Graceful Artifact Handling

Verification degrades as artifacts are missing — never invent a comparison you cannot support:

- **Tasks only** (no specs/design): verify objective task completion only. If all tasks are checked and runtime evidence exists, `PASS WITH WARNINGS` for task completion is the ceiling. Do not claim spec correctness or design coherence.
- **Tasks + specs**: verify completeness **and** requirement/scenario correctness. Missing covering tests are CRITICAL for required scenarios unless `rules.verify` explicitly allows manual verification.
- **Tasks + specs + design (+ proposal)**: verify completeness, correctness, and coherence — the full matrix.
- **Any unchecked task**: always CRITICAL, even when other artifacts are missing and the rest is warnings-only. Unchecked tasks block a clean PASS regardless of what else looks fine.
- **Design missing**: skip design coherence and record it in Skipped Dimensions.
- **No runner**: report it in Evidence as a preflight block (see `references/report-format.md`) with the declared commands noted as not-executed — do not substitute source inspection for the missing runtime evidence.

## Rules

- ALWAYS read all available artifacts before judging — specs, design, tasks, and the apply evidence.
- ALWAYS run the real test and build commands; static analysis alone is never verification.
- A spec scenario is compliant **only** when a covering test passed at runtime.
- Compare **specs first, design second, task completion third**.
- Do NOT fix issues — report them for the orchestrator/user. Your verdict never triggers a repair.
- Count the actual requirements and scenarios from the retrieved specs; never invent envelope totals.
- Record the exact test/build commands, exit codes, and output in the envelope.
- Persist a `fail` report just like a `pass` — a failed verdict is a result, not a reason to discard the artifact.
- If Strict TDD is resolved active, load `references/strict-tdd-verify.md` and include its sections; if inactive, never load or reference it.
- **Hybrid is the only mode** — always write the filesystem `verify-report.md` AND persist to Mnemonic; never branch on `openspec` / `engram-compat` / `none`.
- No external binaries. Mnemonic (`mem_*`), the code index (`code_*`), and the project's own test/build/coverage commands are the only tools; no `gentle-ai sdd-verify-validate`, no `gentleman-ai`, no `sdd-phase-common.md` dispatcher, no separate admission-attestation binary.
- Model/provider/profile/effort selection stays user-owned; verification never changes them.
- Return envelope per Step 9 — final action is text, not a tool call.

## Gotchas

- **`mem_search` returns 300-char previews.** A preview of a 2000-char spec loses most of its scenarios — always `mem_get_observation(id)` before you count requirements/scenarios or map them to tests.
- **Envelopes with mismatched totals fail self-check.** A report that says `scenarios: 12/12` but the spec has 15 scenarios (or the report lists 9) is an inconsistent artifact — the totals must equal the actual counts you retrieved.
- **A `PASS` with a CRITICAL finding is a `FAIL`.** The verdict is derived from the blockers, not asserted: `FAIL` iff ≥1 CRITICAL, ≥1 unchecked task, or a required test/build command exited non-zero.
- **Coverage and quality metrics are informational.** They are WARNING/SUGGESTION at worst — never CRITICAL. A low coverage % or a linter warning does not, by itself, block a PASS.
- **Tautologies are worse than missing tests.** `expect(true).toBe(true)` "passes" and contributes nothing — if you find one in a changed test, flag it CRITICAL (it gives false confidence and does not count toward TDD coverage).
- **Ghost loops are tests that always pass.** An assertion inside a `for` over `queryAll`/`filter` results that could be empty never runs — audit each with a non-empty companion; flag CRITICAL.
- **Do not trust the apply-progress TDD table blindly.** Cross-reference each reported test file against existence (RED) and re-run it (GREEN) — a GREEN you did not re-execute is not a GREEN.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- A `fail` verdict is **persisted**, not skipped. `sdd-archive` should not run on a FAIL, but the report must be on disk and in Mnemonic so the orchestrator can hand the user the exact evidence of what failed.
- If you resolved Strict TDD as active and then cannot run the TDD audit because the apply phase left no TDD Cycle Evidence table, flag CRITICAL (the protocol was not followed) — do not silently fall back to a Standard-verify report.

## References

- [references/report-format.md](references/report-format.md) — the verify-report template, the YAML envelope, compliance statuses, the pre-persistence self-check, and the blocked-preflight recovery shape.
- [references/strict-tdd-verify.md](references/strict-tdd-verify.md) — the Strict TDD verify module (TDD Compliance audit, Assertion Quality audit, Test Layer Distribution, Changed-File Coverage, Quality Metrics, and its report-template extension). Load only when Step 1 resolves Strict TDD as active.
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) — upstream; its apply-progress artifact (incl. the TDD Cycle Evidence table and Work Unit Evidence) is the primary thing this phase audits.
- [references/strict-tdd.md](references/strict-tdd.md) — the apply-phase TDD cycle + assertion rules you are checking against in Step 5.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its scenarios are what the compliance matrix maps to.
- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its decisions are what the design-coherence table maps to.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder.
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — change-folder layout; `verify-report.md` placement; `rules.verify`; the archive step that later consumes this report.
