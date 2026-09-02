---
name: sdd-verify
description: "Verify an SDD change step by step: execute each step's tests and prove the implementation matches that step's acceptance.feature, plan, and completed tasks. Use to run the quality gate after sdd-apply and before sdd-archive — writing one verification.md per step (the per-step gate) from real execution, auditing assertion quality (Strict TDD), and persisting per-step PASS/FAIL verdicts. Marks nothing as done without runtime evidence. Uses Mnemonic memory + code index; no external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "tasks → apply → verify → archive"
  prev: [sdd-apply]
  next: [sdd-archive]
  artifact: verification (per step)
  delegate_only: true
---

# sdd-verify

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-verify` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-verify` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the VERIFY phase — an **independent quality gate**, run **per step**. You prove, with source inspection **plus real execution evidence**, that the implementation matches the step's `acceptance.feature` (WHAT), the plan (HOW), and the completed task set for **that step**. You compare **acceptance first, plan second, task completion third**, run the actual test and build commands, and produce one `verification.md` per step — each carrying a verdict of `PASS`, `PASS WITH WARNINGS`, or `FAIL`.

**You are the one independent requirements/runtime verification per step.** Your job is to judge, not to fix or to re-derive: an acceptance scenario is compliant **only when a covering test passed at runtime** — static analysis alone is never verification. A contradiction or a new failing check returns `FAIL` for that step and hands back to the orchestrator. You never start a remediation/correction cycle, a refutation pass, or another phase on your own; the orchestrator decides the next step.

Phase order is `… → apply → verify → archive`. You run after `sdd-apply` and before `sdd-archive`, **once per step** (the orchestrator may dispatch you per step, or dispatch you once to walk every step — either way, the artifact is one `verification.md` per step folder).

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`)
- **The step(s) to verify** — a single `NN-name` (per-step dispatch), or "all" (walk every step folder). For each step you verify, the change folder root, artifact paths, task-progress state, and allowed verify scope apply.
- **Strict TDD mode** (`true` | `false`) — if the orchestrator declares `STRICT TDD MODE IS ACTIVE`, treat it as authoritative. If not provided, resolve it in Step 2.
- Optional: a `## Skills to load before work` block.

**Artifact store mode is `hybrid` — the only mode for this phase.** For each step verified, the run does BOTH: writes `steps/<NN-name>/verification.md` **and** persists that step's report to Mnemonic under `sdd/<NNN-slug>/verification` (a single concatenated observation of all steps verified this run). A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout; `verification.md` lives inside each `steps/<NN-name>/`; `rules.verify` from `docs/skillgrid/config.yaml`; the archive step gates on every step's `verification.md`.
- [references/strict-tdd.md](references/strict-tdd.md) — the apply-phase TDD cycle and assertion-quality rules you audit in Step 6 (local copy for a self-contained verify skill).
- [`references/report-format.md`](references/report-format.md) — the **per-step** verification template, compliance statuses, and the self-check you run before persisting.
- [`references/strict-tdd-verify.md`](references/strict-tdd-verify.md) — the Strict TDD verify module; loaded **only** when Step 2 resolves Strict TDD as active.

## Skill Loading

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise recover all artifacts from Mnemonic and the change folder (previews are not enough — always fetch full content):
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/intent")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/plan")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; you count each step's actual scenarios.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**; you read each step's `[x]` state.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")` → `..._mem_get_observation(id)` — the apply evidence (incl. the TDD Cycle Evidence table and per-step Step Evidence if Strict TDD was active).
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts (stack, testing, tracker).
3. Read the filesystem primary copies in `docs/skillgrid/changes/<NNN-slug>/`: `intent.md`, `plan.md`, and for each step `steps/<NN-name>/tasks.md`, `steps/<NN-name>/acceptance.feature`, and any existing `steps/<NN-name>/verification.md`.
4. Read `docs/skillgrid/config.yaml` if present — `context:`, `rules.verify` (test/build commands, coverage threshold), and the `tdd` flag bind this phase.

## Status Guard

Before running tests, confirm readiness from the structured state (orchestrator-provided, or the `state.yaml` DAG state in the change folder):

- For each step to verify, if some assigned task in that step is still `[ ]` in `steps/<NN-name>/tasks.md`, **do not run the full suite for that step** — mark that step `blocked` naming the incomplete tasks. Focused single-unit checks remain an apply responsibility.
- If a step is missing its `acceptance.feature` or `tasks.md`, mark that step `blocked` (see the Blocked Preflight shape in `references/report-format.md`) rather than guessing completion.
- If the verify scope is unsafe (you cannot run the tests for the step — no detectable runner, read-only workspace where the harness needs to write artifacts), STOP and report it.

## What to Do

### Step 1: Enumerate Steps to Verify

Determine the step set:

- Single-step dispatch: that one `NN-name`.
- "All" dispatch: every folder under `steps/`, in NN order — **verify a step only if its `Depends on:` predecessors already have a PASS verdict** in this run or on disk; otherwise the earlier step's `verification.md` is the gate, and mark the later step `blocked (dependency <NN> not PASS)`.

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

If Strict TDD is **not** active, do **not** load or process `references/strict-tdd-verify.md` — no TDD sections appear in the report.

### Step 3: Per Step — Compare Acceptance First

For **each** step, count the **actual** scenarios from `steps/<NN-name>/acceptance.feature` — never invent totals. For every scenario, map it to:

1. A covering test (file + test name) — found via the code index or the repo.
2. The runtime result of that test (pass/fail) from Step 5 execution.

A scenario with **no covering test** is `UNTESTED` (CRITICAL for a required scenario). A scenario whose covering test **failed** is `FAILING` (CRITICAL). A test that passes but only partially covers the scenario is `PARTIAL` (WARNING). A test that exists and passed is `COMPLIANT`.

### Step 4: Per Step — Check Plan Coherence (second)

For each decision in the **plan** that applies to this step, check it against the changed code:

- Decision **followed** → note it.
- Decision **deviated from** → WARNING, unless the deviation breaks an acceptance scenario (then CRITICAL, and it surfaces in Step 3's compliance matrix anyway).

If the plan artifact is missing, **skip plan coherence** and record that in the report (`Plan coherence: skipped — no plan artifact`). Do not fabricate a comparison.

### Step 5: Per Step — Run Tests, Build, and Coverage

Execute the real commands — static analysis alone is never verification:

```
FOR EACH STEP:
1. Run the test suite (rules.verify.test_command, or the runner's default), filtered to the step's acceptance scenarios where the runner supports tag/selector (e.g. cucumber --tags, jest -t, go test -run).
   ├── Capture: command, exit code, pass/fail/skip counts, and the failure detail.
   └── Preserve the output (it is the execution evidence for the compliance matrix).
2. Run build / type-check (rules.verify.build_command, or the project's build step).
   └── Capture: command, exit code, relevant output.
3. IF a coverage tool is available AND rules.verify.coverage_threshold is set:
   ├── Run with coverage; filter to the files CHANGED by this step
   │   (file list from the step's apply-progress entry or the diff).
   └── Report per-file and aggregate; flag files below the threshold.
4. Quality metrics (linter / type checker) ONLY on changed files, ONLY if the tools
   exist — WARNING for errors, SUGGESTION for warnings, never CRITICAL.
```

Confirm the exact command you ran matches `rules.verify` / the runner — do not report the exit code of a command you did not actually run.

### Step 6: (Strict TDD only) Run the TDD Verify Module

Load `references/strict-tdd-verify.md` and run its checks **against the apply-progress TDD Cycle Evidence table** for the steps under verification:

- **Step 6a — TDD Compliance + Assertion Quality Audit**: for each task row, verify the RED test file exists, the GREEN test actually passes now, triangulation was adequate for the scenario count, and the safety net was run for modified files. Then audit every test file changed by this change for trivial assertions (tautologies, orphan-empty, type-only, type-only-alone, ghost loops, smoke-only, implementation-detail coupling, mock-heavy). Tautologies are CRITICAL — they "pass" without proving anything.
- **Step 6b/6c — Test layer distribution + changed-file coverage**: classify every test file the step's slice added/modified (unit / integration / E2E), cross-reference with the available tools, and report changed-file coverage if the tool exists.

The report gains the TDD Compliance, Test Layer Distribution, Changed File Coverage, Assertion Quality, and Quality Metrics sections (template in `references/strict-tdd-verify.md`).

### Step 7: Build the Per-Step Report

Assemble each step's `verification.md` per [`references/report-format.md`](references/report-format.md):

- The YAML **envelope** first (`schema: skillgrid.verify-result/v1`, the **step id**, verdict, blockers, critical_findings, the **actual** `scenarios` total you counted for THIS step, and the exact test/build commands + exit codes).
- `## Verification: {NNN-slug} — Step {NN}-{name}` — completeness table (this step's tasks), build/tests/coverage evidence, **acceptance compliance matrix** (one row per scenario), correctness table, plan-coherence table, issues grouped **CRITICAL / WARNING / SUGGESTION**, and the step verdict.
- If Strict TDD is active, insert the extra sections from Step 6.
- For missing artifacts, include the skipped dimensions rather than an empty section (see Graceful Artifact Handling below).

### Step 8: Self-Check Before Persisting (replaces the external validator binary)

Run the checks in `references/report-format.md`; a failure means fix the report, don't persist an inconsistent one:

1. Every envelope field present, exactly once, non-contradictory (a `pass` with a `CRITICAL` finding is a `fail`) — **per step**.
2. `scenarios` totals **equal** the count of scenarios in that step's `acceptance.feature` — not a guessed or rounded number.
3. Every `CRITICAL` finding names a file / test / command / exit code that Step 5 ran (or the step's apply-progress shows) — no finding floats free of evidence.
4. Every scenario lands in **exactly one** compliance status (COMPLIANT / PARTIAL / FAILING / UNTESTED); nothing is both tested and untested.
5. Unchecked tasks in the step are `CRITICAL`, even if other artifacts are missing or the rest is warnings-only.
6. Verdict consistent with blockers: `FAIL` iff ≥1 CRITICAL or ≥1 unchecked task or a test/build command exited non-zero; `PASS` only if zero CRITICAL and all tasks complete and all required-test commands exited 0.
7. `test_exit_code` / `build_exit_code` are the **real** exit codes of the commands you ran (or `125` for commands that did not run in a preflight-blocked case), and `test_command` / `build_command` name them literally.

If any check fails, fix it before persisting; if you cannot, return `partial` and leave the prior `verification.md` untouched.

### Step 9: Persist Reports (hybrid — MANDATORY, do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = BOTH writes:

1. **Filesystem** — one `steps/<NN-name>/verification.md` per step verified (already assembled in Step 7). If a prior `verification.md` exists for a step, it is replaced by the newly admitted bytes (a new verdict supersedes the old one; the prior report is not kept as a second file).
2. **Mnemonic** — start one session, then save the concatenated report (one observation covering every step verified this run):

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/verify")

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/verification",
  topic_key:  "sdd/<NNN-slug>/verification",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "## Step 01-{name}\n{envelope + body}\n\n## Step 02-{name}\n{envelope + body}\n..."
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. `topic_key` upserts — re-running verify replaces the observation in place. A `FAIL` verdict is a **valid, persistable** result — persist it, do not discard a failed report.

### Step 10: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 9) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Step Verification Report
**Change**: {NNN-slug}
**Steps verified**: {NN-…, …}   (or: single step {NN-…})
**Version**: {spec version or N/A}
**Mode**: {Strict TDD | Standard}
**Location**: `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/verification.md` (per step) · Mnemonic `sdd/<NNN-slug>/verification` (hybrid)
**Status**: success (verified) | partial | blocked

### Per-Step Verdict
| Step | Verdict | Scenarios (pass/total) | Tasks (done/total) |
|---|---|---|---|
| 01-{name} | {PASS \| PASS WITH WARNINGS \| FAIL} | {n}/{n} | {n}/{n} |
| 02-{name} | {PASS \| …} | {n}/{n} | {n}/{n} |

### Execution Evidence (per step)
- Step 01: Tests: `{command}` → exit `{code}` · Build: `{command}` → exit `{code}`
- Step 02: Tests: `{command}` → exit `{code}` · Build: `{command}` → exit `{code}`

### Issues (per step)
- **Step 01 — CRITICAL**: {list or None} · WARNING: {…} · SUGGESTION: {…}
- **Step 02 — CRITICAL**: {list or None} · WARNING: {…} · SUGGESTION: {…}

{IF Strict TDD Mode → per-step TDD Compliance · Test Layer Distribution · Changed File Coverage · Assertion Quality · Quality Metrics (from references/strict-tdd-verify.md)}

### Skipped Dimensions
{Step 01: plan coherence skipped — no plan artifact | Step 02: dependency 01 not PASS → blocked | …}

**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: sdd-archive (all steps PASS) | orchestrator decides the remediation path (any step FAIL)
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Graceful Artifact Handling

Verification degrades per step as artifacts are missing — never invent a comparison you cannot support:

- **Tasks only** (no acceptance/plan): verify objective task completion only. If all tasks are checked and runtime evidence exists, `PASS WITH WARNINGS` for task completion is the ceiling. Do not claim acceptance correctness or plan coherence.
- **Tasks + acceptance**: verify completeness **and** scenario correctness. Missing covering tests are CRITICAL for required scenarios unless `rules.verify` explicitly allows manual verification.
- **Tasks + acceptance + plan (+ intent)**: verify completeness, correctness, and coherence — the full per-step matrix.
- **Any unchecked task in a step**: always CRITICAL for that step, even when other artifacts are missing and the rest is warnings-only.
- **Plan missing**: skip plan coherence for that step and record it in Skipped Dimensions.
- **No runner**: report it in Evidence as a preflight block (see `references/report-format.md`) with the declared commands noted as not-executed — do not substitute source inspection for the missing runtime evidence.

## Rules

- ALWAYS read all available artifacts before judging — per step: acceptance, plan, tasks, and the apply evidence.
- ALWAYS run the real test and build commands for each step; static analysis alone is never verification.
- An acceptance scenario is compliant **only** when a covering test passed at runtime.
- Compare **acceptance first, plan second, task completion third — per step.**
- One `verification.md` per step. A step without a verdict is not verified.
- Do NOT fix issues — report them for the orchestrator/user. Your verdict never triggers a repair.
- Count the actual scenarios from each step's `acceptance.feature`; never invent envelope totals.
- Record the exact test/build commands, exit codes, and output per step.
- Persist a `FAIL` report just like a `PASS` — a failed verdict is a result, not a reason to discard the artifact.
- If Strict TDD is resolved active, load `references/strict-tdd-verify.md` and include its sections; if inactive, never load or reference it.
- **Hybrid is the only mode** — always write each step's `verification.md` AND persist to Mnemonic; never branch on `openspec` / `engram-compat` / `none`.
- No external binaries. Mnemonic (`mem_*`), the code index (`code_*`), and the project's own test/build/coverage commands are the only tools; no `gentle-ai`, no `gentleman-ai`, no separate admission-attestation binary.
- Model/provider/profile/effort selection stays user-owned; verification never changes them.
- Return envelope per Step 10 — final action is text, not a tool call.

## Gotchas

- **`mem_search` returns 300-char previews.** A preview of a 2000-char plan loses most of it — always `mem_get_observation(id)` before you count scenarios or map them to tests.
- **Per-envelope totals must match the step's acceptance.** A step report that says `scenarios: 4/4` but the `acceptance.feature` has 5 scenarios is an inconsistent artifact — totals equal the actual count for **that step**.
- **A `PASS` with a CRITICAL finding is a `FAIL`.** The step verdict is derived from the blockers, not asserted: `FAIL` iff ≥1 CRITICAL, ≥1 unchecked task in that step, or a required test/build command exited non-zero.
- **Coverage and quality metrics are informational.** WARNING/SUGGESTION at worst — never CRITICAL. A low coverage % or a linter warning does not, by itself, fail a step.
- **Tautologies are worse than missing tests.** `expect(true).toBe(true)` "passes" and contributes nothing — if you find one in a changed test, flag it CRITICAL (it does not count toward acceptance coverage).
- **Ghost loops are tests that always pass.** An assertion inside a `for` over empty results never runs — audit each with a non-empty companion; flag CRITICAL.
- **Do not trust the apply-progress TDD table blindly.** Cross-reference each reported test file against existence (RED) and re-run it (GREEN) — a GREEN you did not re-execute is not a GREEN.
- **A step's `Depends on` is a verify gate.** A later step whose predecessor is not PASS is `blocked (dependency)`, not PASS — verify in NN order.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- A `FAIL` verdict is **persisted**, not skipped. `sdd-archive` should not run while any step is `FAIL`, but the report must be on disk and in Mnemonic so the orchestrator can hand the user the exact evidence of what failed **in that step**.
- If you resolved Strict TDD as active and then cannot run the TDD audit because the apply phase left no TDD Cycle Evidence table, flag CRITICAL (the protocol was not followed) — do not silently fall back to a Standard-verify report.

## References

- [references/report-format.md](references/report-format.md) — the **per-step** verification template, the YAML envelope, compliance statuses, the pre-persistence self-check, and the blocked-preflight recovery shape.
- [references/strict-tdd-verify.md](references/strict-tdd-verify.md) — the Strict TDD verify module (TDD Compliance audit, Assertion Quality audit, Test Layer Distribution, Changed-File Coverage, Quality Metrics, and its report-template extension). Load only when Step 2 resolves Strict TDD as active.
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) — upstream; its apply-progress artifact (incl. the TDD Cycle Evidence table and per-step Step Evidence) is the primary thing this phase audits.
- [references/strict-tdd.md](references/strict-tdd.md) — the apply-phase TDD cycle + assertion rules you are checking against in Step 6.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its per-step `acceptance.feature` scenarios are what the compliance matrix maps to.
- [`../sdd-design/SKILL.md`](../sdd-design/SKILL.md) — upstream; its per-step WHAT and decisions are what the plan-coherence table maps to.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — change-folder layout; `verification.md` placement per step; `rules.verify`; the archive step that later consumes these reports.
