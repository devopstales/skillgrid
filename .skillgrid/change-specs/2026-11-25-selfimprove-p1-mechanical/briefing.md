# Self-Improving Loop — Phase 1: Mechanical infrastructure — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-25)
> **Author:** devopstales
> **Intent frozen:** no (pending) ← the agent sets this to "yes (YYYY-MM-DD)" once the STATUS banner above shows `approved` (the user's hand-edit IS the approval — the agent never writes `approved` itself); after that, the frozen block below is read-only for agent steps

> **Phase 1 of the self-improving-skills plan.** Depends on Phase 0
> (`2026-11-25-selfimprove-p0-loop`) being shipped: it consumes the inbox and adds the three
> missing *machines* — the telemetry writer, the eval/scenario runner, and the runnable
> curation pass. Still no agents-in-the-loop (that is Phase 2): everything here is a script
> with an exit code or a skill with a procedure.

## Problem / Intent

Phase 0 closed the loop's *sink*; Phase 1 installs the loop's *sensors and repeater*. Three
gaps remain, all mechanical:

1. **No telemetry on disk.** `docs/04-lifecycle.md` and `skill-creator` specify the
   `.usage.json` sidecar (`use_count`, `view_count`, `patch_count`, `last_activity_at`,
   `state`, `pinned`), but no file exists and nothing writes it — so the curation
   transitions (90d stale / 180d archive) have no input and run "manual" by the doc's own
   fallback.
2. **No runner for the behavioral tests.** `evals/evals.json` (prose/trigger cases) and
   `evals/scenarios.yaml` (machine-checkable regex over replies, with the full format
   specified in `_shared/craft/scenario-evals.md`) exist for 11 skills, but **no script
   loads or executes either format** — RED/GREEN for a skill edit is currently a manual
   agent exercise, which Phase 2's paired-replay gate must not be.
3. **The curation interval is prose.** `docs/04-lifecycle.md` defines the procedure
   (status → review candidates → backup → transition → report) but no skill *is* that
   procedure — `skill-creator` mentions lifecycle curation in passing.

**Job to be done:** when a curation pass runs, it wants *numbers* (which skills are stale,
how often each was used), *evidence* (can I re-run a skill's RED scenario against the old
and the new version?), and *a procedure* it can follow — so that by the time Phase 2 adds
the improvement agent, every input and every check is already deterministic.

## Purpose & Success Criteria

- **Purpose:** make the self-improvement loop's inputs and checks mechanical: a harness
  hook that writes the telemetry sidecar, a runner that executes `scenarios.yaml` /
  `evals.json` with exit codes, and a curation pass that is an invokable skill.
- **Success criteria:**
  1. After a harness session that loads a skill, `.agents/skills/.usage.json` contains that
     skill's incremented `use_count` and current `last_activity_at`; a patch-only session
     (no load) updates `patch_count` / `last_activity_at` on the patched skill. The sidecar
     is committed (like `skill-size-budget.json`), not gitignored.
  2. `node .agents/skills/skill-creator/scripts/run-evals.mjs <skill>` exits 0 when all of a
     skill's scenarios pass and non-zero naming every failed assertion (with the
     `message` from `scenarios.yaml`); a skill with only `evals.json` runs the prose cases
     in *report mode* (LLM/human-judged, exit 0 with the report printed — the runner
     judges nothing by regex that the file didn't declare).
  3. Paired replay is one command: `run-evals.mjs --replay <skill> --against <git-ref-or-dir>`
     reports, per scenario, pass/fail on *both* versions — the exact output shape Phase 2's
     gate consumes (revised passes all; original fails ≥1).
  4. The curation pass is a skill invocation (a `skill-creator` mode, not a new skill —
     see Approaches): given the sidecar + inbox, it produces the doc's five-step output
     (status table, per-candidate decision list, backups, transition report) with
     `status: open` proposals from Phase 0's inbox folded into the candidate review.
  5. Degradation is explicit (the repo's degradation-floor concept): no harness hook →
     sidecar absent → the curation pass reports "telemetry: manual mode" and uses only
     `last_activity_at` from patches, exactly as `docs/04-lifecycle.md` already specifies.
- **Out of scope:** anything that *decides* an improvement (Phase 2's improvement pass);
  applying proposals (Phase 2); the user-experience ask (Phase 0 R4); per-project telemetry
  (the sidecar is library-level, in this repo — a *user* project's sidecar is Phase 3's
  concern, if at all); a background daemon (still never); changing the sidecar *format*
  (it is already the contract — this phase only writes it).

## Context

- **The sidecar format (already the contract):** `docs/04-lifecycle.md` § Usage Telemetry
  and `skill-creator/references/anatomy-and-checklist.md` § Usage telemetry (materialized
  by Phase 0 R6 into `skill-creator/references/skill-lifecycle.md`) — field set fixed:
  `use_count`, `view_count`, `patch_count`, `last_activity_at`, `state`, `pinned`.
- **The runner contract (already specified):** `_shared/craft/scenario-evals.md` —
  `scenarios.yaml` shape (`skill_file` relative to the skills root, `turns` with
  `user_input` + `assertions` of `required_pattern` / `forbidden_pattern` + `message`),
  and the RED-GREEN rule (a scenario must fail without the rule and pass with it).
  `_shared/craft/eval-assertions.md` covers the assertion semantics.
- **The plugin pattern to reuse:** `plugins/omp/` ships `pi-worktree` and
  `skillguard-guards` (ambient discovery via `<cwd>/.omp/hooks/pre/*.ts` re-exports, single
  source of truth in `plugins/` — see `.omp/hooks/pre/skillgrid-guards.ts`). A telemetry
  hook follows the identical shape: source in `plugins/omp/skill-telemetry/`, ambient
  re-export in `.omp/hooks/`.
- **The curation procedure (already written):** `docs/04-lifecycle.md` § The Curation
  Interval — five steps, deliberate (weekly/monthly, human-triggered), backup-before-
  archive invariant, pin exemption, agent-created-only scope.
- **Inbox (Phase 0):** `.skillgrid/skill-improvements/` + `check-proposal.mjs` — this phase
  makes the curation pass *read* it.
- **Upstream pattern:** Perplexity Brain — skill-load events as the evidence substrate;
  the replay job as a separate deterministic step from the reflection job. This phase
  builds exactly those two machines (events = sidecar; replay = `run-evals.mjs --replay`).

## User Context

- **Who:** the maintainer (runs the curation pass by hand, on a schedule) and, downstream,
  any project that installs skillgrid and wants its *own* library telemetry.
- **Direct evidence:** the curation interval's first step is "List every skill with its
  `state`, `last_activity_at`, `use_count`, provenance. The stale candidates jump out." —
  today that first step is impossible to run honestly (no `use_count` exists), so the
  interval degrades to a gut check.
- **Why it fails today:** the doc's fallback ("`last_activity_at` is the only honest field,
  update it on patch") hides *usage* — a skill used weekly but never patched looks identical
  to a dead one. The user's own cited sources (Perplexity) run the whole loop on exactly
  these two signals (load events + problem reports), so the gap is load events.

## Pre-mortem

- **Desirability:** the sidecar becomes cargo-cult numbers — incremented but never read.
  Countermeasure: the curation pass (R4) *is* the consumer and its status table is the
  deliverable; a sidecar nobody reads is a finding of the pass itself.
- **Feasibility:** the harness hook depends on the *consuming* harness (omp/Pi) exposing a
  "skill loaded" event. If the ambient hook cannot see skill loads (only session/turn
  events), the writer degrades to patch-time `last_activity_at` + a `use_count` grep over
  session transcripts where available. Countermeasure: R1's acceptance tests the hook
  against the hook events the harness *actually* exposes, and the degradation mode is the
  specified behavior, not a bug.
- **Viability:** a scenario runner that needs a live model per run is slow and flaky; if it
  becomes "nobody runs it because it takes 10 minutes," Phase 2's gate has no repeater.
  Countermeasure: the runner is a thin harness (load skill file → one turn → regex) with a
  per-run timeout, and `--replay` caches nothing by default (repeatability over speed);
  the eval set per skill is small by construction (the 10-rule audit caps it).

## Timebox

Target: ships within two weeks (2026-11-25 → 2026-12-09). If exceeded: ship the sidecar
writer (R1) + the curation pass (R4) first; the `--replay` mode (R3) becomes a follow-up
ticket of the same change — but R2 (the basic runner) never slips, it is Phase 2's floor.

## Approaches Considered

| Approach | Key trade-off | Verdict |
|----------|---------------|---------|
| Telemetry via an ambient omp hook plugin (`plugins/omp/skill-telemetry/` + `.omp/hooks/` re-export) | Tied to the omp harness; other harnesses get the documented fallback | **Chosen** — matches the existing `skillgrid-guards` pattern exactly |
| Telemetry via git hooks (post-commit `patch_count`, no `use_count`) | Zero harness coupling, but captures *patches only* — the doc's degraded mode, permanently | Rejected as the primary; it *is* the degradation path |
| A new top-level skill for the curation pass | Discoverable, but the pass is lifecycle tooling that `skill-creator` already claims (its description names "reviewing a skill library's lifecycle") | Rejected — a **mode** of `skill-creator` (a `references/curation-pass.md` + its SKILL.md getting a Curation section) keeps one owner for the library |
| Runner as a separate `evals/` npm package | Clean, but the repo's rule is scripts live with the skill that owns them; a new package is a dependency creation | Rejected — `skill-creator/scripts/run-evals.mjs` |

- **Chosen:** hook plugin + `run-evals.mjs` under `skill-creator/scripts/` + curation as a
  `skill-creator` mode.
- **Rejected by author:** standalone curation skill (two owners for the same lifecycle).

## Requirements

1. **R1 — Telemetry writer hook:** `plugins/omp/skill-telemetry/` (source of truth) +
   `.omp/hooks/` ambient re-export, following `skillgrid-guards`'s exact shape. On a skill
   load event: increment `use_count` and set `last_activity_at` (ISO 8601) in
   `.agents/skills/.usage.json` for that skill. On a `SKILL.md` patch (detectable from
   post-turn file state or the git diff at turn end): increment `patch_count` and update
   `last_activity_at`. The sidecar is **committed** (added to the repo, not gitignored) and
   the hook creates it lazily with the full skill inventory (every skill dir under
   `.agents/skills/`, `_shared` included, all fields zeroed/absent-per-contract). The hook
   never fails a turn (write errors are logged to stderr, exit 0 — the
   `pi-worktree` non-blocking discipline). Degradation: harness without the hook → no
   sidecar → the curation pass runs in the doc's manual mode (R4 reports it).
   - **Current:** no `.usage.json`, no writer anywhere (verified 2026-11-25).
   - **Target:** the plugin + re-export + a `bun test` suite (same seam as
     `plugins/omp/pi-worktree/tests`): sidecar created with correct inventory; load event
     increments exactly one skill; patch event increments `patch_count`; second identical
     load is counted again (no dedup — *use* is the signal); write error does not throw.
   - **Acceptance:** the `bun test` suite passes with no omp install (pure-core split,
     same as pi-worktree's import-scan rule); after one real session that loads `debug`,
     the committed sidecar shows `debug.use_count` incremented.
   - **Acceptance scenario:** `happy path load-event-increments-sidecar` → `acceptance.feature`
2. **R2 — Eval/scenario runner:** `skill-creator/scripts/run-evals.mjs <skill> [--replay
   <ref|dir>] [--json]`. Non-replay: loads the skill (per `scenarios.yaml`
   `system_prompt.skill_file`), runs each scenario's turns against a model (the harness's
   model — the runner takes a model-agnostic turn executor interface, so omp/Pi/CLI all
   plug in), asserts `required_pattern` / `forbidden_pattern` with the file's own
   `message` on failure; skills with only `evals.json` run in report mode (cases printed,
   judged by LLM/human, exit 0). Exit 0 = all asserted scenarios pass; exit 1 = ≥1 failed
   assertion, each named with its `scenario_id` + `message`.
   - **Current:** no script executes either format (verified 2026-11-25).
   - **Target:** the runner + `test-run-evals.mjs` (fixtures: a toy skill with one passing
     and one failing scenario; a scenario whose `skill_file` is missing → exit 2 with the
     path named — a distinct "broken harness" exit from a "failed assertion").
   - **Acceptance:** `test-run-evals.mjs` green; running the real `execute` scenarios
     (`evals/scenarios.yaml` exists for it) against the current skill body reports each
     assertion result — the pre-existing scenarios are the runner's first real corpus.
   - **Acceptance scenario:** `happy path runner-asserts-scenario` → `acceptance.feature`
3. **R3 — Paired replay mode:** `run-evals.mjs --replay <skill> --against <git-ref-or-dir>`
   runs the *same* scenario set against the skill at the given ref/dir and against the
   current one, and emits a per-scenario matrix (scenario × version → pass/fail) — JSON on
   `--json`, a table by default. It judges nothing; the *gate* (revised passes all,
   original fails ≥1, controls unchanged) is Phase 2's decision logic consuming this
   output. This separation is deliberate: the machine reports, the curation pass decides
   (the deterministic-boundary rule).
   - **Current:** RED-GREEN for a skill edit is a manual two-run exercise.
   - **Target:** the flag + the matrix output; `test-run-evals.mjs` covers it with a
     fixture skill where v1 fails a scenario and v2 passes it.
   - **Acceptance:** replay of a fixture pair produces the expected matrix (fail/pass);
     a skill with no scenario diff produces an all-pass matrix (the "controls unchanged"
     case).
   - **Acceptance scenario:** `happy path paired-replay-matrix` → `acceptance.feature`
4. **R4 — Runnable curation pass:** `skill-creator` gains a Curation mode (a
   `references/curation-pass.md` holding the five-step procedure, now lifted verbatim from
   `docs/04-lifecycle.md` — the doc keeps the human-facing summary and points at the
   reference, same split as Phase 0 R6). The pass: (1) reads the sidecar (or reports
   "telemetry: manual mode") + every skill's frontmatter (`created_by`, `pinned`, `state`)
   + the Phase 0 inbox (`status: proposed|deferred` proposals are **candidates**, each
   reviewed in the same sitting); (2) lists stale candidates per the 90/180-day rules
   (agent-created only, pinned exempt); (3) for each candidate and each open proposal,
   records a decision (keep+pin / keep+bump / →stale / →archived / proposal:
   apply→Phase 2 or reject/defer with a note); (4) takes the backup tarball for every
   archive *before* the move; (5) writes the one-line-per-transition report into the
   change's `report.md` (the pass runs as part of a change, or standalone with its report
   in `.skillgrid/work/`). The pass **never** edits a `created_by: human` skill and **never**
   deletes (archive is the ceiling) — both invariants from the doc, now enforced by the
   procedure the skill follows.
   - **Current:** the interval is prose in `docs/04-lifecycle.md`; `skill-creator`'s
     description claims lifecycle review but the procedure is not invokable.
   - **Target:** the reference + the SKILL.md Curation section (a Contents line + a short
     section pointing at the reference) + the pass's output contract (status table +
     decision list + backups + report lines).
   - **Acceptance:** running the pass on this repo's library produces a status table
     covering all 19 skills + `_shared` (every row: state, `last_activity_at`,
     `use_count` or "manual mode", provenance, pinned); with a seeded `PROP-001.md`
     present, the decision list names it; zero transitions on a fresh library (nothing is
     90 days stale) — the pass reports "no transitions," not nothing.
   - **Acceptance scenario:** `happy path curation-pass-status-table` → `acceptance.feature`
5. **R5 — Docs follow the machines:** `docs/04-lifecycle.md`'s "Skillgrid does not ship a
   curator daemon. The sidecar is optional and the format is the contract." is updated to
   name the writer hook and the curation pass (what is now shipped, and that the daemon
   prohibition stands); the curation-interval section points at
   `skill-creator/references/curation-pass.md`; `docs/03-tools.md` (the tools doc) lists
   `run-evals.mjs` and the telemetry hook with their exact invocations.
   - **Current:** the doc describes a system with no writer and no runnable pass.
   - **Target:** the doc is true; every invocation it names works as written (the
     cite-don't-restate rule: the *procedure* lives in the reference, the doc keeps the
     summary + the invariants).
   - **Acceptance:** `grep` for the three invocations in the docs returns working commands
     (each executed once from a clean shell during the change's test phase); no doc
     statement contradicts a shipped behavior.
   - **Acceptance scenario:** `happy path docs-name-the-machines` → `acceptance.feature`

## Implementation Decisions

- **Files to create:** `plugins/omp/skill-telemetry/` (`src/` core — pure, no harness
  imports — + `tests/` + `README.md` + package/Taskfile entries mirroring `pi-worktree`),
  `.omp/hooks/pre/skill-telemetry.ts` (ambient re-export, three lines),
  `.agents/skills/.usage.json` (lazy-created by the hook; committed),
  `skill-creator/scripts/run-evals.mjs` + `test-run-evals.mjs`,
  `skill-creator/references/curation-pass.md`.
- **Files to modify:** `skill-creator/SKILL.md` (Contents line + Curation section, small —
  size-budget check at execute time), `docs/04-lifecycle.md` (pointer + shipped-machinery
  note), `docs/03-tools.md` (two entries), `.skillgrid/config.yaml` only if the testing
  layers list needs the new plugin test dir (`plugins/**/tests` already covers it — expect
  no change).
- **Interfaces:** the sidecar JSON schema is frozen by the existing contract (R0, no new
  fields — if one is needed, it is a Phase-2 question, not a Phase-1 invention). The
  runner's turn-executor is an interface `{ runTurn(skillFile, userInput): Promise<string> }`
  with a default implementation for the omp harness and a `--mock` fixture executor for
  tests. `--json` output: `{ skill, version_current, version_against?, scenarios: [{ id,
  current: pass|fail, against?: pass|fail, message? }] }`.
- **Data flow:** harness events → hook → `.agents/skills/.usage.json` (committed at turn
  end only if changed, to avoid churn); `run-evals.mjs` reads `evals/` + the skill file
  (at the given ref) → matrix; curation pass reads sidecar + frontmatter + inbox → status
  table + decisions → (archive) backup tarball in `.skillgrid/artifacts/` + `report.md`
  lines.
- **Error handling:** hook write failure → stderr, exit 0 (never blocks a turn); runner
  with a broken `skill_file` → exit 2 (distinguished from exit 1 = failed assertion);
  curation pass with no sidecar → manual mode, reported, not an error; proposal whose
  `original_snippet` no longer matches → reported as *stale proposal* in the decision list
  (Phase 0's check already defines this state; the pass surfaces it).
- **Dependencies:** none new at the library level (the runner shells out to the harness's
  model CLI for real runs; tests use `--mock`). The hook is a sibling plugin — no shared
  code with `pi-worktree` (independent by the family pattern).

## Shootdown Findings

**Lens status:** all seven lenses **not run** — drafted from the Phase 0 plan + repo
investigation; Goldfish runs at the spec gate.

**Anchored hypothesis (as proposed, pending user confirmation):** the loop's sensors are
the binding constraint on Phase 2 — a curation pass with honest numbers and a one-command
replay makes the improvement agent's job *deciding*, not *measuring*.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| — | (not run) | — | pending spec gate | — |

**Overall verdict:** pending — drafted, not yet shot down.

## Testing Decisions

- **What makes a good test:** the pure cores (sidecar mutation logic, matrix assembly,
  pattern assertion) with fixtures; the harness boundary with the `--mock` executor. No
  test calls a real model in the unit suite (cost/flakiness); the R2 acceptance runs the
  real `execute` scenarios *once* as an integration check, not in the unit loop.
- **Modules to test:** `skill-telemetry` core (inventory bootstrap, load/patch mutation,
  write-error tolerance); `run-evals.mjs` (pass/fail/broken-file exits, mock scenarios,
  replay matrix, `--json` shape); curation pass (procedure test via the skill's own eval —
  a seeded library fixture: one agent-created stale skill, one pinned stale skill, one
  open proposal → expected table + decisions + one backup).
- **Prior art:** `plugins/omp/pi-worktree/tests` (pure-core split + import scan),
  `skill-creator/scripts/test-*.mjs` (script+test pairing), the seeded-fixture style of
  `scripts/test-state-lock.mjs`.
- **Edge cases:** sidecar and a new skill added in the same session (inventory grows
  lazily); replay `--against` a ref where the skill has *no* scenarios (matrix is
  "n/a", not a failure); a `forbidden_pattern` scenario where the word appears in the
  skill's own name (the craft doc's negative-lookbehind guidance applies — the runner
  asserts the file's pattern verbatim, never rewrites it); curation pass with `state:
  stale` already set but `last_activity_at` recently bumped (the bump wins — the doc's
  archived→active restore rule).

## Impact on Global Docs

- `.skillgrid/projectwiki/ASSUMPTIONS.md`: one new VERIFIED line at ship — "telemetry
  sidecar is committed library state; the writer is the omp hook; the daemon prohibition
  stands."
- `.skillgrid/projectwiki/ARCHITECTURE.md`: the `.skillgrid/` zone map and the
  `plugins/omp/` family gain the telemetry plugin (if the file exists; lazy).
- `.skillgrid/projectwiki/concepts/`: a `run-evals-contract.md` page if the turn-executor
  interface survives review unchanged (it is a new durable interface).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ✓      | three machines named, each with an owner |
| Boundary Clarity    | 0.85  | 0.70 | ✓      | decision logic explicitly deferred to Phase 2 |
| Constraint Clarity  | 0.85  | 0.65 | ✓      | frozen sidecar format, no-daemon, exit-code contract |
| Acceptance Criteria | 0.85  | 0.70 | ✓      | all five requirements falsifiable  |
| **Clarity**         | **0.14** | ≤0.20 | ✓ | from the Phase 0 plan + repo verification |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | (prior session) the loop needs sensors + a repeater before an agent can improve skills | telemetry + runner + runnable curation = Phase 1 |
| 2     | (this session) specs for all phases | curation is a `skill-creator` mode, not a new skill |

## Open Questions & Assumptions

| Type | Item | Owner | Due |
|------|------|-------|-----|
| Question | Which harness event carries a *skill load* (vs session/turn)? If none, the hook infers from transcript deltas — confirm the inference is acceptable telemetry. | user + harness maintainer | blueprint gate |
| Question | Real-model runs in CI or local-only? Default: local-only (CI runs `--mock`); a weekly local run is the curation pass's job. | user | spec gate |
| Assumption | The sidecar's format is closed (no new fields) — any needed field is an explicit Phase 2 change to the contract. | devopstales | until disproved |
| Assumption | One committed sidecar per repo (library-level); per-skill sidecars are a non-goal (the doc allows them but the repo ships one). | devopstales | until disproved |

## Decisions (ADR)

None clears the ADR bar yet (the turn-executor interface is the candidate; it is recorded
as a concept page at ship instead — see Impact). If the harness event question (Open
Questions) forces a fallback that is hard to reverse, file an ADR at that point.

## Terms

- **Paired replay** (new, from the Perplexity source): running the same scenario set
  against two skill versions and reporting the matrix. Glossary at ship.
- **Telemetry sidecar** (already in the lifecycle doc — the term is existing, not new):
  no glossary change.

## Retrospective

**Status:** pending

- **Intent met?** [filled by ship]
- **Pre-mortem score:** [filled by ship]
- **Assumptions:** [filled by ship]
- **Timebox:** [filled by ship]
- **Next time:** [filled by ship]
