---
type: Spec
title: "2026-10-20-parallel-cleave-port — briefing"
status: draft

---

# Parallel Decomposition (Cleave Port) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-20)
>
> **Role:** this is the *what* — the validated design/spec: problem, intent,
> requirements, success criteria. The companion `blueprint.md` is the *how*
> (implementation plan). Do not rename either file.
>
> **Topic:** 2026-10-20-parallel-cleave-port
> **Date:** 2026-10-20
> **Classification:** standard (T2)
> **Build shape:** Smallest usable whole — three independent guardrail scripts
> + prose land as three separate usable increments; the first (`compute-waves`)
> is the thinnest whole that makes a parallel run deterministic.
> **Findings:** none (this change ports behavior from a known source — the
> omegon-pi `cleave` extension, read in-session; no research/prototype/sketch
> pre-gate was run. See Context for the source.)

---

## Problem / Intent

Skillgrid already decomposes a directive into parallel leaves
(`skillgrid:parallel-execution` → `concurrent-leaves.md`), but three guardrails
that make a parallel run safe and convergent live only as *prose an agent must
re-derive every time*, not as deterministic logic:

1. **Wave ordering.** A leaf is dispatchable only when its dependencies are
   satisfied — but the dispatch *order* (which leaves form wave 0, wave 1, …)
   is computed by the agent by hand each run. The dependency graph lives in the
   `tasks.md` `slicing` writes: per-ticket `- **Blocked by:** <ids>` bullets
   **and** a `## Dependency graph` mermaid block (the two must be read together —
   decision A+B — bullets are authoritative, mermaid is the consistency
   cross-check). Cleave does this deterministically
   (`computeDispatchWaves`, a Kahn topological sort) and also breaks
   dependency **cycles** (`detectAndBreakCycles`). Skillgrid has no equivalent.
2. **Dirty-tree preflight.** `skillgrid:isolated-workspace` requires a clean
   tree before worktrees, but it does not *classify* uncommitted files
   (related / unrelated / volatile) or offer a resolution action. Cleave's
   `runDirtyTreePreflight` classifies the tree and lets the operator pick
   checkpoint / stash-unrelated / stash-volatile / proceed / cancel, auto-
   resolving the volatile-only case.
3. **Review convergence.** Cleave's per-child review loop bails when a fix is
   *churning* — a fixed fraction of issues **reappears** after a fix round.
   Skillgrid's `concurrent-leaves.md` and `subagent-execution` fix loops have no
   churn breaker; a non-converging review can spin past its cap silently.

The intent: **port the three deterministic guardrails from Cleave into
skillgrid as scripts + skill prose** — *not* Cleave's runtime (no Rust
orchestrator, no OS-process RPC, no model-tier routing, no worktree lifecycle,
no dashboard `sharedState`). The SDD ledger stays the execution record
(ADR-0020); skillgrid's subagent model stays the execution model (in-harness,
not OS subprocesses).

## Purpose & Success Criteria

- **Purpose:** Make a skillgrid parallel run deterministic (wave order), safe
  (dirty-tree handled explicitly), and convergent (review churn is detected and
  bounded) — using the least machinery that fills each gap, per the
  `skill-write` rule (logic that repeats identically every run lives in a
  script the agent calls, not prose the agent re-interprets).
- **Success criteria:**
  - `scripts/compute-waves` reads a `tasks.md` graph — per-ticket `- **Blocked
    by:** <ids>` bullets (primary) + a `## Dependency graph` mermaid block
    (cross-check, decision A+B) — and emits the dispatch wave ordering (wave 0,
    wave 1, …) deterministically; a cyclic graph yields the cycle-named leaves
    broken and a warning, and a bullets-vs-mermaid disagreement yields a
    `MISMATCH:` line (bullets win), not an infinite loop.
  - `scripts/classify-dirty` reads `git status --porcelain` + the change's
    `Owns:` globs and emits the related / unrelated / volatile buckets with a
    suggested checkpoint message; a volatile-only tree is auto-resolvable.
  - The **churn breaker** is a named rule in `concurrent-leaves.md`: when ≥50%
    of a round's issues reappear in the next round, the loop stops, findings are
    parked in the ledger with a ruling, and it escalates.
  - Each script is exercised by a real invocation in a fixture repo (RED→GREEN),
    and each skill's prose names the script it now calls.
- **Out of scope:**
  - **`assess-complexity` / the Cleave complexity formula** — `brainstorming`
    already classifies trivial/standard/risky and `fast-track` already gates on
    it; a deterministic `systems × (1 + 0.5 × modifiers)` gate would be a
    duplicate gate. (Revisit if a future change needs a numeric decompose-vs-
    execute signal that the existing classification cannot express.)
  - Cleave's Rust `omegon-agent` orchestrator, RPC child event stream,
    `worktree.ts` lifecycle, model-tier routing (Ollama + provider
    `routingPolicy`), and dashboard `sharedState` mirror.
  - Any new Go route, new npm dependency, new runtime process, or migration.
  - Changing the SDD ledger record (ADR-0020) or the in-harness subagent model.
  - The `/cleave inspect` live view (stdout ring buffer + per-child git diff) —
    deferred to the Phase 2 dashboard as a feature, not part of this change.

## Context

- **Source being ported:** the `cleave` extension of `omegon-pi`
  (`styrene-lab/omegon-pi`, `extensions/cleave/`). Read in-session. The three
  pieces ported, and their source locations:
  - Wave computation: `planner.ts` → `computeDispatchWaves` (Kahn) +
    `detectAndBreakCycles` (Kahn cycle break, clears deps on cyclic nodes).
  - Dirty-tree preflight: `index.ts` → `runDirtyTreePreflight` +
    `classifyPreflightDirtyPaths` (in `workspace.ts`) — five actions
    (`checkpoint`, `stash-unrelated`, `stash-volatile`, `proceed-without-cleave`,
    `cancel`), volatile-only auto-resolve, max 3 resolution attempts.
  - Churn breaker: `review.ts` → the review loop's reappearing-issue fraction
    bail (default threshold 0.5), severity-gated fix iterations.
- **Existing skillgrid surfaces this extends (all present, verified in-repo):**
  - `.agents/skills/execution/parallel-execution/SKILL.md` +
    `references/concurrent-leaves.md` — the `Owns:`/`Needs:` contract, wave
    dispatch, branch integration gate, parent re-verify. **`Needs:` already
    exists** as a per-task ledger line in both skills; the wave *ordering*
    script consumes it.
  - `.agents/skills/execution/isolated-workspace/` — worktree isolation before
    execution; currently a clean-tree requirement, no classification.
  - `.agents/skills/execution/subagent-execution/` — the fix loop (rounds 1–5,
    breaker adjudication); the churn breaker composes with its existing cap.
  - `.skillgrid/config.yaml` `ticketing:` — Backlog.md tracker; task status
    already advances at wave transitions (no change needed here).
- **Locked decisions this change must respect** (`.skillgrid/ASSUMPTIONS.md`
  § LOCKED):
  - **ADR-0020** — the SDD ledger (`.skillgrid/sdd/<plan>/progress.md` /
    `parallel-ledger.md`) is the execution record; Mnemonic stores an index.
    These scripts are *inputs to* the ledger, never a second execution record.
  - **ADR-0016** — the skills/content zone is a governed capability layer; new
    skill scripts are content, not a new runtime component.
  - **ADR-0003** — single-binary, Go, cgo-free invariant. The scripts are
    Node/JS pipeline helpers (the established `scripts/*.mjs` + skill-`scripts/`
    pattern in `2026-10-02-skillgrid-workflow-updates` §4) — they run at
    *planning/execution* time, not inside the Go binary, so the invariant holds.
- **Operational rules** (`AGENTS.md` § Rules): Go 1.22+ floor (ADR-0031); **no
  new dependencies without an ADR** — the scripts must be **dependency-free**
  (Node stdlib only, or the repo's sole existing npm dep, `yaml`); serial
  development; conventional commits, no AI attribution; spec-zone commits
  before code-zone (the `pre-commit` zone guard enforces this).

## Approaches Considered

- **Chosen:** Port the three guardrails as **deterministic JS scripts** under
  the owning skills' `scripts/` dirs, each invoked by one line of skill prose,
  with RED→GREEN tests run against a fixture git repo. No Go, no new npm dep,
  no runtime, no ledger change. Rationale: each gap is logic that repeats
  identically every run (the `skill-write` criterion), the repo already runs
  JS pipeline helpers, and it keeps the single-binary + ledger-record invariants
  intact.
- **Rejected — port the Cleave runtime (Rust orchestrator + RPC children):**
  fights ADR-0003 (single Go binary), ADR-0020 (a second execution record), and
  ADR-0005 (MCP-spawn trust boundary — Cleave children `git commit` in detached
  OS processes). The runtime is omegon-pi-specific; only the guardrail logic is
  transferable.
- **Rejected — prose-only (no scripts):** cheapest, but re-introduces exactly the
  failure being fixed — an agent re-deriving wave order / dirty classification /
  churn each run and diverging. The scripts are the point.
- **Rejected — implement in Go as a `skillgrid` subcommand:** correct for
  production, but it is a larger change (a command group, an ADR, a test surface
  in the binary) and this change's value is the guardrails, not the language.
  Revisit if these become hot-path or cross-change infrastructure.

## Requirements

1. **R1 — Deterministic wave ordering:**
   - **Current:** `concurrent-leaves.md` says "A wave is a set of leaves whose
     `Needs:` are all satisfied" but the agent computes the order by hand; there
     is no cycle handling. The dependency graph is already in `tasks.md` as
     per-ticket `- **Blocked by:** <ids>` bullets + a `## Dependency graph`
     mermaid block — the agent just doesn't turn it into an order.
   - **Target:** A script reads a `tasks.md` carrying per-ticket `- **Blocked
     by:** <ids>` bullets (primary) **and** a `## Dependency graph` mermaid block
     (consistency cross-check — decision A+B: bullets authoritative, disagreement
     is a `MISMATCH:` line, not a silent pick) and emits an ordered list of waves
     (wave 0 = no unsatisfied deps, wave N = leaves whose deps are all in earlier
     waves), plus a `CYCLE:` note when the graph cannot fully order and a `MISMATCH:`
     line when the two sources disagree.
   - **Acceptance:** Running the script on a 3-ticket graph whose bullets +
     mermaid agree (T3 blocked by T2; T1, T2 free) emits exactly two waves with
     T3 in the later wave; running it on a 2-ticket cycle emits both tasks
     (broken) and a named-cycle warning; running it on a graph where a ticket's
     bullets disagree with its mermaid edges emits a `MISMATCH:` line and still
     exits 0 (bullets win); running it on an empty graph emits zero waves without
     error.
   - **Acceptance scenario:** `happy path compute-waves-orders-independent-and-dependent` → `acceptance.feature`

2. **R2 — Dirty-tree classification + preflight actions:**
   - **Current:** `isolated-workspace` requires a clean tree but does not
     classify uncommitted files or offer a resolution action.
   - **Target:** A script reads `git status --porcelain` plus the change's
     `Owns:` globs (from the plan/ledger) and classifies every dirty path into
     **related** (matches an `Owns:` glob), **volatile** (matches the built-in
     allowlist: build artifacts, caches, `node_modules`, `dist`, `*.log`, `.env`
     local, etc.), and **unrelated/unknown** (everything else); it emits a
     suggested checkpoint commit message covering the related set, and flags the
     tree **auto-resolvable** when only volatile paths are dirty.
   - **Acceptance:** In a fixture repo with 1 related, 1 volatile, 1 unrelated
     file dirty, the script reports the three in the correct buckets and a
     non-empty checkpoint message; with only a `dist/` file dirty it reports
     `auto-resolvable: true` and an empty related set.
   - **Acceptance scenario:** `happy path classify-dirty-buckets-and-autoresolve` → `acceptance.feature`

3. **R3 — Review churn breaker (named rule + skill prose):**
   - **Current:** `concurrent-leaves.md` parent re-verify + `subagent-execution`
     fix loop have a round cap (5) but no *convergence* signal — a fix can
     churn (same issues reappearing) up to the cap without being flagged.
   - **Target:** `concurrent-leaves.md` gains a **Churn breaker** rule: key each
     review issue by a stable identity (file + symbol + issue-type); if the
     fraction of issues from round R−1 that reappear in round R is ≥ the
     threshold (default **0.5**), stop the fix loop, park the still-open issues
     in the ledger each with a `Ruling:`, and escalate (do not start round R+1).
     `subagent-execution`'s fix loop references this rule so the per-task loop
     inherits it.
   - **Acceptance:** The `concurrent-leaves.md` section names the stable issue
     key, the 0.5 default, the stop-park-escalate action, and a pointer from
     `subagent-execution`'s task loop; the rule is expressible as a checkable
     condition (given two rounds' issue lists, the breaker fires iff the
     reappear fraction ≥ 0.5).
   - **Acceptance scenario:** `happy path churn-breaker-stops-reappearing-fix-loop` → `acceptance.feature`

4. **R4 — Skill prose names the scripts (the wiring):**
   - **Current:** None of the three skills reference a wave / classify /
     churn script.
   - **Target:** `parallel-execution`/`concurrent-leaves.md` name
     `scripts/compute-waves`; `isolated-workspace` names
     `scripts/classify-dirty`; `concurrent-leaves.md` (and a pointer in
     `subagent-execution`) name the churn breaker. Each reference is a one-line
     instruction: when to run it, what to do with its output.
   - **Acceptance:** `grep` for `compute-waves` in `concurrent-leaves.md`,
     `classify-dirty` in `isolated-workspace/SKILL.md`, and `churn` in both
     `concurrent-leaves.md` and `subagent-execution/` each returns ≥1 hit with
     the run-when + use-output instruction present.
   - **Acceptance scenario:** `happy path skills-reference-deterministic-guardrails` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - New scripts (dependency-free Node, run at plan/execution time, not in the
    Go binary):
    - `.agents/skills/execution/parallel-execution/scripts/compute-waves`
    - `.agents/skills/execution/isolated-workspace/scripts/classify-dirty`
    - `.agents/skills/execution/parallel-execution/scripts/churn-check`
  - New skill prose edits:
    - `concurrent-leaves.md` (wave step gated ≥4 leaves + Churn breaker section
      naming `churn-check`)
    - `isolated-workspace/SKILL.md` (preflight classify step)
    - `subagent-execution/` task-loop reference (pointer to the churn breaker)
  - Tests: one fixture-based test per script (RED→GREEN), plus a prose-presence
    check for R4.
- **Interfaces:**
  - `compute-waves` — input: a `tasks.md` carrying per-ticket `- **Blocked
    by:** <ids>` bullets (primary) + a `## Dependency graph` mermaid block
    (cross-check, decision A+B); output: a printed ordered wave list (`WAVE 0: …`,
    `WAVE 1: …`) plus `CYCLE:`/`DROPPED:`/`MISMATCH:`/`DUP:` lines and, on a
    genuinely unparseable graph, a `PARSE:` line + exit 2; a cycle is broken and
    named, not a crash.
  - `churn-check` — input: two files, one stable issue key (`file|symbol|
    issue-type`) per line (prev round, current round) + optional threshold;
    output: `REAPPEAR n/m (frac)` + `FIRES: true|false` (true iff frac ≥ 0.5).
    The reappear-fraction calculation lives here (deterministic, per
    `skill-write`); the prose states the stop-park-escalate decision.
    cycle is broken, a `CYCLE:` line naming the affected task ids. Exit 0 on a
    valid (possibly cycle-broken) graph; non-zero on unparseable input.
  - `classify-dirty` — input: repo path + one or more `Owns:` globs (plus the
    built-in volatile allowlist); output: three labeled sections
    (`RELATED:`, `VOLATILE:`, `UNRELATED/UNKNOWN:`), a `CHECKPOINT: <message>`
    line, and `AUTO_RESOLVABLE: true|false`. Exit 0 always (classification is
    advisory; the operator decides the action).
  - Both scripts: **Node stdlib only** (no `yaml` even, to keep them
    importable without a `node_modules`); they read `tasks.md`/`git status` as
    plain text.
- **Data flow:** The agent (running `subagent-execution` / `parallel-execution`)
  runs the script via the shell, reads the printed result, and records the
  outcome in the ledger (`parallel-ledger.md`) — the script has no state of its
  own and never writes a second execution record (ADR-0020).
- **Error handling:** Unparseable `tasks.md` → `compute-waves` prints the parse
  error + exit 2 (the agent falls back to hand-ordering and ledger-records why).
  A repo with no `git` / detached edge cases → `classify-dirty` degrades to
  "unrelated/unknown" for paths it cannot resolve and says so. Neither script
  throws an uncaught stack trace to the operator.
- **Dependencies:** None new. The scripts use Node stdlib. This is the explicit
  answer to AGENTS.md's "no new dependencies" rule: no `package.json` change, no
  `go.mod` change.
- **Prototype snippets:** The cycle-break algorithm is Cleave's
  `detectAndBreakCycles` (Kahn: in-degree per node, peel in-degree-0 nodes, the
  un-visited set is the cycle; clear `needs` on cyclic nodes and report them).
  The wave algorithm is Cleave's `computeDispatchWaves` (repeatedly emit all
  leaves whose `needs` are all in the satisfied set, until none remain; if a
  pass yields none but tasks remain, that residual is a deadlock/cycle).

## Testing Decisions

- **What makes a good test:** only external behavior — given a fixture task
  graph / fixture git tree, the script's *printed output and exit code*. No
  assertions on internal function names.
- **Modules to test:** `compute-waves` (linear, diamond, cycle, empty,
  unparseable) and `classify-dirty` (mixed buckets, volatile-only auto-resolve,
  all-related, no-git) at the CLI seam (subprocess, assert stdout + exit).
- **Prior art:** the repo's read-only Node verifiers
  `.agents/skills/verification/qa/scripts/state-drift-check.mjs` /
  `ship-drift-check.mjs` (each emits a machine-parseable `SCOPE:`/`DRIFT:` line)
  — model the output contract the same way: a stable line format the skill
  prose and the test both parse.
- **Edge cases:** a self-`Blocked by:` (a ticket depending on itself) is dropped
  before wave computation; a `Blocked by:` reference to an unknown ticket id is
  dropped and reported (`DROPPED:`); bullets-vs-mermaid disagreement is a
  `MISMATCH:` line (bullets win); a duplicate ticket id is a `DUP:` line (first
  wins); `Owns:` glob matching is case-sensitive on path, anchored to repo root,
  and a dirty submodule path is bucketed by its own path (not expanded); the
  volatile allowlist is a fixed const (not config) in v1.

## Impact on Global Docs

- `.skillgrid/reference/00-prd.md`: None (the PRD's engine scope is unchanged;
  this is the skills/content zone).
- `.skillgrid/ASSUMPTIONS.md`: One line under **VERIFIED** noting the three
  guardrail scripts exist and where (so a future session can find them), plus —
  **only if** the interview confirms the scripts are intended as cross-change
  infrastructure — a candidate ADR row. Default for this change: **no new ADR**
  (see Open Questions).
- `.skillgrid/ARCHITECTURE.md`: None (no new component; the scripts are skill
  content under `.agents/skills/…/scripts/`).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.80  | 0.75 | ✓      | Three named guardrails, each mapped to a Cleave source. |
| Boundary Clarity    | 0.80  | 0.70 | ✓      | Out-of-scope list is explicit (no runtime, no ledger change, no assess). |
| Constraint Clarity  | 0.75  | 0.65 | ✓      | Node-stdlib-only, no new dep, ADR-0003/0020 respected. |
| Acceptance Criteria | 0.80  | 0.70 | ✓      | Each requirement has Current/Target/Acceptance + a scenario. |
| **Clarity**         | 0.15  | ≤0.20| ✓      | |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 1 | Port the whole Cleave runtime or just the logic? | Logic only — three guardrails; no Rust/RPC/worktree/model-routing/dashboard. |
| 2 | Port the complexity-assess gate too? | No — `brainstorming`'s trivial/standard/risky + fast-track already gate decompose-vs-execute. |
| 3 | Scripts in JS or Go? | JS under the owning skills' `scripts/` (repo's pipeline-helper pattern); no new Go command, no ADR. |
| 4 | New ADR for the port? | Default no — skill content under ADR-0016; revisit only if promoted to cross-change infra. |
| 5 | What does `compute-waves` parse for the graph? | **A+B (user, post-critic):** the per-ticket `- **Blocked by:**` bullets are primary; the `## Dependency graph` mermaid block is a consistency cross-check; disagreement → `MISMATCH:` line (bullets win). The critic refuted the original `Needs:`-in-`tasks.md` premise (no `tasks.md` carries `Needs:` lines). |
| 6 | Is the churn fraction prose or a script? | A `churn-check` script (deterministic, per `skill-write`); the prose states the stop-park-escalate decision. |

## Open Questions & Assumptions

- **Assumption:** The volatile allowlist is a fixed const in `classify-dirty`
  for v1 (not read from `config.yaml`). Revisit if a project needs a custom
  volatile set.
- **Assumption:** No new ADR row is added for this change. If the review
  surfaces that these scripts are meant to be invoked by *multiple* changes
  (cross-change infrastructure), the reviewer should promote it to a candidate
  ADR row; until then ADR-0016 (skills content zone) governs.

## Decisions (ADR)

- **ADR-0020** (`.skillgrid/decisions/04-adr-0020-sdd-ledger-owns-execution.md`)
  — the scripts are inputs to the ledger, never a second execution record.
- **ADR-0016** (`.skillgrid/decisions/04-adr-0016-second-brain-capability-layer.md`)
  — new skill scripts are governed skills-content, not a new runtime component.
- **ADR-0003** (`.skillgrid/decisions/04-adr-0003-v1.0-line-engine-trustworthy.md`)
  — the scripts run at plan/execution time (JS pipeline helpers), not inside the
  single Go binary; the cgo-free single-binary invariant is untouched.
- No **new** ADR for this change (see Open Questions).

## Terms

- **wave** — a set of leaves whose `Needs:` are all satisfied by earlier
  waves; the unit of parallel dispatch. (Relies on the existing `Needs:` term.)
- **churn** — the reappear-fraction of review issues across two fix rounds;
  the convergence signal the breaker trips on. (New — sharpened this plan.)
- **auto-resolvable tree** — a dirty tree whose only uncommitted paths are
  volatile; `classify-dirty` flags it so the operator need not prompt.
  (New — this plan.)
