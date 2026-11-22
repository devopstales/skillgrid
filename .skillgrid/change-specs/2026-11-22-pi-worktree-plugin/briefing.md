# @devopstales/pi-worktree — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-22)

## Problem / Intent

The skillgrid worktree standard (`_shared/work-unit-commits/README.md`: tooling
tiers, scratch checkpoints, branch+worktree atomicity, merge semantics) was
written for the *skill* layer, but the harness tier-1 slot it defines — "the
harness-native tool wins: it owns session state" (`execute` Step 0b) — has no
implementation for the Pi coding agent. Two community extensions cover pieces:
**pi-worktrunk** (mavam, v0.13.2) runs worktrunk in-Pi with linked-session
moves, cwd-removal recovery, and 🤖/💬 markers; **checkpoint-pi** (prateekmedia,
v1.0.5) plus its successor **pi-rewind** (arpagon) provide git-ref code
checkpoints with safe restore. Neither is skillgrid-aware: no
`refs/skillgrid/` namespace, no branch+worktree atomic pair, no per-agent
branch allow-list at creation, no `--no-squash --no-commit` merge default.
This change builds `@devopstales/pi-worktree` — one Pi plugin, two extension
files — that is the tier-1 harness implementation of the standard.

## Purpose & Success Criteria

- **Purpose:** Pi sessions in a skillgrid repo get (a) turn-level scratch
  checkpoints of the worktree with safe, decision-gated restore, and (b)
  worktrunk runs whose session state follows the worktree — both enforcing the
  shared contract instead of restating it.
- **Success criteria:**
  1. A dirty worktree survives a killed/interrupted turn: the last scratch
     checkpoint restores it (files-only, branch+worktree pair verified).
  2. A read-only turn creates no new ref (tree-SHA dedup) — ref count stable.
  3. `wt switch --create` from the plugin moves the Pi session into a linked
     session in the new worktree; the source session stays `/resume`-able.
  4. A worktree removed under a running session recovers the session to a
     surviving worktree (preferring main) *without replaying* a command that
     was requested after the directory died.
  5. `wt list` shows 🤖/💬 for live Pi sessions; stale markers are reportable
     and clearable.
  6. All scratch behavior is covered by tests that run without a Pi install
     (pure-git core) and by a bench ≤150ms at 5k files.
- **Out of scope:** conversation-state restore (Pi's `/fork`/`/tree` owns it);
  per-tool checkpoints; shadow git repos; auto-restore of any kind; LLM
  commit messages (`[skillgrid-context]` is the skill's job); worktrunk's
  interactive picker TUI, OSC 7 terminal reporting, and alias passthrough
  machinery; anything that makes the plugin the *only* place the worktree
  standard lives (the skills remain the SoT; the plugin implements it).

## Context

- **The contract being implemented** (all in this repo, all current):
  - `_shared/work-unit-commits/README.md` → **Tooling tiers** (harness-native →
    worktrunk preferred-when-present → `git worktree add`), **§ Scratch
    checkpoints (turn-level)** (`refs/skillgrid/`, turn_end timing, dedup,
    temp-index capture, filter set incl. `.worktree`, pre-existing-untracked
    never-delete set, before-restore snapshot, branch+worktree atomic pair,
    50/session cap + cross-session prune, `checkpoint-state.js` subcommands
    `scratch` / `scratch-restore`), **§ Worktree removal + branch deletion**
    (no-added-changes rule, force-only-with-backup-ref, `--reap`,
    reconciliation), **§ Worktree lifecycle** (fork rule, single-sink
    integration).
  - `execute/SKILL.md` → Step 0b tiers (Tier 2 = worktrunk, path template
    `{{ repo_path }}/.worktree/{{ branch | sanitize }}`, explicit `--base`),
    Step 0d (`.config/wt.toml` `[pre-start]` setup + `[pre-merge]` test,
    `wt step copy-ignored` cache reuse), fan-out (cwd binding,
    `worktree-wf_<domain>` branches, interrupted-worker scratch restore,
    `wt merge <parent> --no-squash --no-commit`).
  - `ship/SKILL.md` → Step 6 options (incl. Option 4 fork), cleanup per the
    shared safety section.
- **Upstream sources studied:**
  - `mavam/pi-worktrunk` v0.13.2 — `WORKTRUNK_DIRECTIVE_CD_FILE` runner
    (private temp reply file per invocation; follow valid directive even if a
    later hook fails; no directive → stay put), linked-session moves
    (`ctx.sessionManager`, `ctx.mode`), cwd-removal recovery with no-replay
    (changelog v0.12.1), marker lifecycle (v0.1: `session_start`→💬,
    `agent_start`→🤖, `agent_end`→💬, `session_shutdown`→clear),
    project-command approval dialog (model never passes `-y`), compact tool
    description + on-demand `--help` catalog (v0.13), `executionMode:
    "sequential"` + `queued` codemode semantics.
  - `prateekmedia/checkpoint-pi` v1.0.5 — checkpoint shape (HEAD + index tree
    + worktree tree as a ref), restore options, before-restore checkpoint,
    safe-restore never-delete set, smart filtering (13 dir patterns, >10MiB
    files, >200-file dirs).
  - `arpagon/pi-rewind` — temp-index capture mechanics
    (`GIT_INDEX_FILE` in tempdir → `read-tree HEAD` → `add --all` batches of
    100 → `write-tree`), turn_end timing + tree-SHA dedup, 50/session cap +
    stale-session prune, two-layer core/test split (723-LOC pure-git core,
    20 tests), flat-latency bench (60–142ms to 182k files).
- **Pi API surface used** (verified against both upstreams):
  `pi.registerTool` / `pi.registerCommand` / `pi.exec("wt"|"git", args,
  {cwd, signal, timeout})` / `pi.on("session_start"|"agent_start"|"agent_end"
  |"session_shutdown"|"turn_end")` / `ctx.sessionManager` (`getSessionFile`,
  `getHeader`, `getEntries`) / `ctx.mode` / `ctx.ui.select|notify` / `ctx.cwd`.
  **Pi floor: 0.99.1+** (linked-session API; matches pi-worktrunk).
- **Repo state:** this repo (`skillgrid-skills`) is a skills repo — the plugin
  ships as a *new sibling package* (`@devopstales/pi-worktree`), not code in
  this tree. This change spec governs the package; its code lives in
  `devopstales/pi-worktree` (new repo, same family, scoped npm).

## Approaches Considered

- **Chosen:** one npm package `@devopstales/pi-worktree` with **two Pi
  extension files in one manifest** (`src/scratch.ts` + `src/wt.ts`).
  Rationale: the family will have multiple `@devopstales/pi-*` plugins; the
  two concerns (checkpoints vs worktrunk) are independently useful — a git-only
  machine wants scratch without `wt` — so they split at the extension-file
  level (a user comments one manifest line) while sharing `markers.ts` /
  `ui.ts` and one version. This is also exactly how worktrunk's own Pi plugin
  split skills from markers to avoid approval re-prompting on update.
- **Rejected:** fork pi-worktrunk and bolt on scratch — its public surface
  (picker TUI, OSC 7, alias machinery, display styling) is most of its LOC,
  it is worktrunk-generic (no skillgrid invariants), and forking inherits an
  upstream we'd immediately diverge from on the merge-default and allow-list
  behavior.
- **Rejected:** two separate npm packages (`pi-worktree-scratch` +
  `pi-worktree-wt`) — over-versioning for v1.0; the shared marker/UI code and
  the joint fan-out story (markers + scratch + worktree in one wave) argue for
  one package. Revisit if the family's multi-plugin pattern demands it.
- **Rejected:** port pi-rewind wholesale for the checkpoint side — its
  checkpoint shape predates our contract (no worktree-root half of the pair,
  no `.worktree` filter, `refs/pi-checkpoints/` namespace); we take its
  *mechanics* (temp-index capture, dedup, prune, two-layer split) and write
  the contract-compliant core fresh.

## Requirements

1. **R1 — Scratch checkpoint capture:** a turn that ended with filesystem
   mutations produces exactly one ref under `refs/skillgrid/` for the current
   session, capturing HEAD, index tree, and worktree tree with the metadata
   block (`branch`, `head`, `index-tree`, `worktree-tree`, `session`, `turn`,
   `label`, `created`, `preexisting-untracked`, `skipped-large`).
   - **Current:** nothing implements `checkpoint-state.js` subcommands
     `scratch` / `scratch-restore`; no Pi plugin captures turn checkpoints in
     this namespace.
   - **Target:** `scratch-core.ts` `createCheckpoint({root, sessionId, turn,
     label})` implements the § Scratch checkpoints contract verbatim.
   - **Acceptance:** unit test — mutate a file, run capture, assert one new
     ref with all metadata fields; assert the real index is byte-identical
     before/after; assert a second capture of an unchanged tree returns
     `skipped`.
   - **Acceptance scenario:** `happy path checkpoint-after-mutating-turn` →
     `acceptance.feature`
2. **R2 — Read-only turn dedup:** a turn with no worktree change creates no
   ref.
   - **Current:** checkpoint-pi snapshots every turn; pi-rewind dedups but in
     its own namespace/contract.
   - **Target:** capture compares the candidate worktree-tree SHA against the
     last checkpoint's and returns `skipped` on equality.
   - **Acceptance:** unit test — read-only operations, capture, ref count
     unchanged; the `execute` eval scenario `scratch_dedup_readonly_turn`
     passes against the plugin's documented behavior.
   - **Acceptance scenario:** `happy path readonly-turn-no-ref` →
     `acceptance.feature`
3. **R3 — Safe restore with the atomic pair:** restore of a checkpoint is
   refused when the current branch **or** worktree root differs from the
   recorded pair; otherwise it takes a before-restore checkpoint, restores
   HEAD/index/worktree, and safe-cleans without deleting ignored dirs,
   skipped large files/dirs, or pre-existing untracked files.
   - **Current:** pi-rewind checks branch only; checkpoint-pi has no pair
     concept; neither knows the `.worktree` filter.
   - **Target:** `restoreCheckpoint(root, id)` enforces the pair, snapshots
     first, and restores.
   - **Acceptance:** unit tests — (a) restore succeeds on matching pair with
     a dirty tree (pre-existing untracked survives); (b) restore refused on
     branch mismatch with the mismatch named; (c) restore refused when the
     worktree root differs; (d) a deleted tracked file comes back, a new
     untracked file (post-checkpoint) is removed. The `execute` eval scenario
     `scratch_branch_safety_refused` passes.
   - **Acceptance scenario:** `happy path restore-on-matching-pair` +
     `happy path restore-refused-cross-branch` → `acceptance.feature`
4. **R4 — Bounded refs:** max 50 refs per session (oldest pruned); at session
   start, refs from other sessions are pruned in the background.
   - **Current:** pi-rewind has the caps; no skillgrid plugin does.
   - **Target:** `prune(root, sessionId)` + `pruneOtherSessions(root,
   current)` wired to `session_start`.
   - **Acceptance:** unit test — 55 captures → 50 refs, oldest gone; two
     sessions' refs → the older session's are gone after startup prune.
   - **Acceptance scenario:** `happy path session-start-prune` →
     `acceptance.feature`
5. **R5 — Worktrunk runs with session movement:** a `worktrunk` tool call
   (or `/wt` command) runs `wt` with the `WORKTRUNK_DIRECTIVE_CD_FILE`
   protocol; a valid directory directive moves the Pi session to a linked
   session in the destination worktree; the source session stays `/resume`-
   able; in print/json mode a move returns `stopped` with the restart
   directory instead of moving.
   - **Current:** pi-worktrunk does this generically; nothing does it with
     skillgrid invariants.
   - **Target:** `wt-core.ts` runner + linked-session move in `wt.ts`.
   - **Acceptance:** integration test (real `wt` binary, temp repo) —
     `wt switch --create agent-demo` followed by a session move leaves the
     session cwd in the new worktree and the old session resumable; print
     mode returns `stopped` + path.
   - **Acceptance scenario:** `happy path session-follows-worktree` →
     `acceptance.feature`
6. **R6 — Cwd-removal recovery without replay:** if the session's working
   directory no longer exists, the next `wt` call or `worktrunk` tool call
   recovers the session to a surviving worktree (repository identity
   re-verified; main preferred) and does **not** run a command that was
   requested after the directory died.
   - **Current:** pi-worktrunk v0.12.1 behavior; no skillgrid plugin.
   - **Target:** recovery path in `wt-core.ts`.
   - **Acceptance:** integration test — remove the worktree out from under a
     session, invoke a mutating command, assert recovery to the surviving
     worktree AND that the command did not execute.
   - **Acceptance scenario:** `happy path recovery-no-replay` →
     `acceptance.feature`
7. **R7 — Standard-root verification:** on first `wt` use, the plugin verifies
   worktrunk's path template places worktrees in `<projectroot>/.worktree/`
   (`worktree-path = "{{ repo_path }}/.worktree/{{ branch | sanitize }}"`);
   drift is reported (with the observed layout), never silently followed.
   - **Current:** `execute` Step 0b Tier 2 states the rule in prose; no tool
     enforces it.
   - **Target:** check in `wt-core.ts` startup; report on drift.
   - **Acceptance:** integration test — foreign template → reported drift
     message naming the observed path; standard template → silent pass.
   - **Acceptance scenario:** `happy path standard-root-verified` →
     `acceptance.feature`
8. **R8 — Per-agent branch allow-list at creation:** in a worktree context,
   `switch --create` with a branch not matching
   `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` reports the
   mismatch and the nearest allowed form instead of creating; the primary
   checkout is unconstrained.
   - **Current:** the `pre-commit` guard FATALs at commit time — after work
   has been done in the wrong-named branch.
   - **Target:** creation-time check in `wt-core.ts` (same regex constant as
     the hook, mirrored in `scratch-core.ts`).
   - **Acceptance:** integration test — `switch --create feature-x` in a
     worktree context → rejected with `agent-feature-x` / `worktree-wf_x`
     suggested; `switch --create feature-x` in the primary checkout → allowed.
   - **Acceptance scenario:** `happy path allow-list-at-creation` →
     `acceptance.feature`
9. **R9 — Merge default `--no-squash --no-commit`:** the plugin's `merge`
   invocation prepends `--no-squash --no-commit` unless the caller explicitly
   passes either flag (work-unit commits stay atomic and independently
   revertable).
   - **Current:** pi-worktrunk passes args straight through; worktrunk's
   default squashes.
   - **Target:** arg-rewrite rule in `wt-core.ts` (the one place the plugin
   rewrites a call).
   - **Acceptance:** integration test — `merge` call with no flags → the
   executed argv contains both; explicit `--squash` → no rewrite of that
   flag.
   - **Acceptance scenario:** `happy path merge-defaults` →
     `acceptance.feature`
10. **R10 — Project-command approval passthrough:** when `wt` needs
    project-command approval, the plugin shows the confirmation dialog
    (approving saves exactly those commands, as `wt config approvals add`
    would, and retries once); the model can never pass `-y`/`--yes`.
    - **Current:** pi-worktrunk behavior; worktrunk's own approvals.
    - **Target:** passthrough in `wt.ts`, no `-y` injection anywhere in the
      runner.
    - **Acceptance:** integration test — unapproved project hook → dialog
      shown, approve → command runs once, approvals file updated; no code path
      appends `-y`.
    - **Acceptance scenario:** `happy path approval-passthrough` →
      `acceptance.feature`
11. **R11 — Markers + reconciliation:** `session_start`→💬,
    `agent_start`→🤖, `agent_end`→💬, `session_shutdown`→clear, written via
    worktrunk's marker state; a stale marker (no live session) is reported
    and manually clearable from the plugin.
    - **Current:** pi-worktrunk v0.1 markers; our fan-out reconciliation rule
      (marker without a ledger row = lost dispatch) is prose-only.
    - **Target:** `markers.ts` shared by both extension files.
    - **Acceptance:** integration test — lifecycle events produce the
      expected marker sequence in `wt list`; kill-before-shutdown leaves a
      stale marker that the plugin reports and can clear.
    - **Acceptance scenario:** `happy path marker-lifecycle` →
      `acceptance.feature`
12. **R12 — Testable core without Pi:** `scratch-core.ts` and `wt-core.ts`
    import nothing from `@earendil-works/*`; their full test suites run with
    no Pi install; a bench measures capture latency per repo size.
    - **Current:** pi-rewind proved the split (723-LOC core, 20 tests, bench
      60–142ms flat to 182k files).
    - **Target:** two-layer architecture as a structural test (an import
      scan fails the build if a core file imports a Pi package).
    - **Acceptance:** `npm test` green in a bare Node/Bun environment with
      only git + `wt` on PATH; bench output ≤150ms at 5k files.
    - **Acceptance scenario:** `happy path core-tests-without-pi` →
      `acceptance.feature`
13. **R13 — Contract-drift guard:** the plugin's constants (branch regex,
    filter dir set, size caps, session cap) equal the values stated in
    `_shared/work-unit-commits/README.md`; a test detects divergence.
    - **Current:** nothing keeps an implementation honest against a prose
      contract.
    - **Target:** a deterministic test that greps the contract file for the
      constants and compares (fails with the diff on mismatch).
    - **Acceptance:** change one constant in the plugin → test fails naming
      the mismatch; restore → green.
    - **Acceptance scenario:** `happy path contract-drift-guard` →
      `acceptance.feature`

## Implementation Decisions

- **Modules to build:** new repo `devopstales/pi-worktree` (sibling to
  `skillgrid-skills`): `src/core/scratch-core.ts` (pure git),
  `src/core/wt-core.ts` (runner/directive/recovery/invariants, pure-ish —
  spawns `wt`/`git`, no Pi imports), `src/scratch.ts` + `src/wt.ts`
  (extension wiring), `src/markers.ts`, `src/ui.ts` (footer `◆ N checkpoints`,
  restore dialog: files-only / cancel), `tests/` (scratch, wt-core, bench),
  `evals/evals.json`, `references/` (on-demand model docs:
  `scratch-restore.md`, `wt-commands.md` generated catalog),
  `changelog/releases/` (per-release entries, pi-worktrunk's structure).
- **Interfaces:**
  - `scratch-core`: `createCheckpoint({root, sessionId, turn, label}) →
    {status: "created"|"skipped", id?}`; `restoreCheckpoint(root, id) →
    {status: "restored"|"refused", reason?}`; `listCheckpoints(root,
    sessionId?) → Checkpoint[]`; `prune(root, sessionId) → number`;
    `pruneOtherSessions(root, current) → number`. Constants:
    `AGENT_BRANCH_RE`, `IGNORED_DIRS` (13 upstream + `.worktree`),
    `MAX_UNTRACKED_FILE_SIZE = 10MiB`, `MAX_UNTRACKED_DIR_FILES = 200`,
    `MAX_PER_SESSION = 50`, `REF_BASE = "refs/skillgrid"`.
  - `wt-core`: `runDirectedWt(args, {cwd, signal}) → {stdout, stderr, code,
    directive?}`; `recover(cwd, recordedWorktrees) → {worktree?, replaySafe?}`;
    invariants: `verifyStandardRoot()`, `checkBranchAllowList(branch,
    inWorktree)`, `applyMergeDefaults(args)`.
  - Tools: `worktrunk` (`{command, args?}`, `executionMode: "sequential"`,
    small description + `--help` on-demand catalog, `promptGuidelines`
    incl. never pass `-y`); `scratch-restore` (`{id?}` — no id lists with
    diff preview). Command: `/wt` passthrough + `/checkpoint`.
- **Data flow:** `turn_end` event → `createCheckpoint` (dedup on worktree-tree
  SHA) → footer update. `worktrunk` tool / `/wt` → `runDirectedWt` (temp
  directive file per invocation) → on directive: linked-session move (source
  preserved) → on cwd-gone: recovery, no replay. `session_start` → rebuild
  checkpoint cache from refs + resume checkpoint + background
  `pruneOtherSessions`.
- **Error handling:** `wt` missing → scratch extension still loads; wt
  extension reports "worktrunk not installed — git fallback applies (skill
  Step 0c)" and stays inert. Directive pointing outside the original
  repository → rejected, stay put, report (pi-worktrunk's safety rule).
  Refused restore → reason surfaced (branch or worktree mismatch named).
  Print/json mode + move → `stopped` + restart directory, never a silent
  move.
- **Dependencies:** `wt` on PATH (current release line — required for the
  `WORKTRUNK_DIRECTIVE_CD_FILE` protocol; startup check prints protocol
  support, absence = stay-put + report, never guess); Pi ≥ 0.99.1; git. No
  other runtime deps (pi-rewind/checkpoint-pi both run zero-dependency).
- **Prototype snippets:** the ref metadata message shape (from pi-rewind's
  `commit-tree` approach, our fields):

  ```
  pi-worktree:<id>
  branch <name>
  head <sha>
  index-tree <sha>
  worktree-tree <sha>
  session <id>
  turn <n>
  label <prompt ≤60 + mutating calls>
  created <iso>
  untracked <json: pre-existing post-filter>
  largeFiles <json: skipped >10MiB>
  largeDirs <json: skipped >200 files>
  ```

## Shootdown Findings

**Lens status:** not run (design session conducted interactively with the
owner; all seven lenses named for the record: zoom-out, design-critic,
comprehension, implementation-readiness, codebase-details, structural,
inventive-catalyst — **all not run**, contribution capped at "not verified").
The owner-directed research (both upstreams cloned and read, Pi API surface
extracted) substitutes for the comprehension/codebase-details lenses on the
*upstream* side; the *plugin* codebase does not exist yet, so an
implementation-readiness pass is owed before build step 1 starts.

**Anchored hypothesis:** the skillgrid worktree standard is implementable as
one Pi plugin with two extension files whose cores are pure-git and
testable without Pi, reusing worktrunk's directive protocol and pi-rewind's
temp-index mechanics, with all skillgrid deltas (pair check, allow-list at
creation, merge defaults, standard-root verification, contract-drift guard)
enforced in the wrapper.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| 🟡 Major | implementation-readiness (not run — recorded from research) | `WORKTRUNK_DIRECTIVE_CD_FILE` is a worktrunk-internal env protocol, not a documented stable API | accepted-with-note | pin min `wt` version; startup check; absence → stay-put + report |
| 🟡 Major | design-critic (not run — recorded from research) | two marker writers if worktrunk's own Pi plugin is also installed | accepted-with-note | README: uninstall worktrunk's Pi plugin; plugin warns at `session_start` if both detected |
| 🟢 Nit | structural (not run — recorded) | two extension files in one manifest is non-standard vs. one file | resolved | precedent: worktrunk's own split of skills vs markers; users comment one manifest line |

**Overall verdict:** needs-revision — run the implementation-readiness lens
on `scratch-core.ts` once written (owed, not blocking the spec).

Gate: no undispositioned 🔴 Blocker.

## Testing Decisions

- **What makes a good test:** external behavior only — refs that exist,
  argv that was executed, session cwd after a move, marker state in `wt
  list`. Never assert on temp-file names or internal function order.
- **Modules to test:** `scratch-core` (capture, dedup, restore incl. all
  refusal paths, prune, pair check) at the git seam; `wt-core` (directive
  follow, no-replay recovery, allow-list, merge rewrite, standard-root check)
  at the spawned-`wt` seam with a real binary in a temp repo; extension
  wiring (`scratch.ts`, `wt.ts`, `markers.ts`) by `pi -e` smoke runs only.
- **Prior art:** pi-rewind's `tests/core.test.ts` (20 tests, pure git) and
  `bench.ts` (flat-latency table per repo size); pi-worktrunk's
  `worktrunk.test.ts` + per-feature test files (`recovery.test.ts`,
  `terminal-cwd.test.ts`, `approval.test.ts`) as the shape for
  directive/recovery/approval tests; the `skillgrid-skills` repo's
  `test-*.mjs` convention for the contract-drift guard.
- **Edge cases:** unborn HEAD (no commits yet — capture uses zero-SHA, as
  pi-rewind's `ZEROS`); detached HEAD in a worktree (pair check + restore
  behavior); the `.worktree` dir itself appearing in the untracked list
  (filtered, never snapshotted); a worktree removed *by another process*
  (recovery path, not the directive path); 55th checkpoint in a session
  (prune-oldest); two sessions sharing a repo (cross-session prune keeps the
  current session's refs); `wt` present but older than the directive
  protocol (stay-put + report).

## Impact on Global Docs

- `.skillgrid/projectwiki/reference/00-prd.md`: None (file does not yet exist
  in this repo; the plugin is a sibling package, not a product scope change).
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: None yet (file does not exist;
  create on first use if the repo adopts the vault).
- `.skillgrid/projectwiki/ARCHITECTURE.md`: None yet (file does not exist).
- **In-repo doc edits this change does make:**
  - `README.md` (Tool Dependency list): add `pi-worktree` line — "Pi plugin
    (`pi install npm:@devopstales/pi-worktree`): scratch checkpoints
    (`refs/skillgrid/`) + worktrunk runner with session movement (execute
    Step 0b tier 1/2, ship Step 6; optional — the git fallback path is
    unchanged without it)".
  - `_shared/work-unit-commits/README.md` § Scratch checkpoints: add one
    line — "Reference implementation: `@devopstales/pi-worktree`
    (`scratch-core.ts`); the constants here are its source of truth (the
    contract-drift guard test enforces the direction)".

## Clarity Report

Scores from the interactive design session (owner-directed; the questioning
loop ran as research rounds, not a formal frontier interview — recorded as
such):

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.85  | 0.75 | ok     | "tier-1 harness implementation of the standard" stated by owner |
| Boundary Clarity    | 0.80  | 0.70 | ok     | out-of-scope list explicit (no conversation restore, no picker, no auto-restore) |
| Constraint Clarity  | 0.75  | 0.65 | ok     | Pi 0.99.1+, wt directive protocol, zero-dep cores, contract drift guard |
| Acceptance Criteria | 0.80  | 0.70 | ok     | 13 requirements, each Current/Target/Acceptance + scenario |
| **Clarity**         | 0.17  | ≤0.20| pass   | 1 - avg(0.85, 0.80, 0.75, 0.80) |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|-----------------|-----------------|
| 1 | Which worktree tooling model for the skills? | worktrunk preferred-when-present, git fallback, no onboarding question; `conventions.worktree_tool` config key |
| 2 | What to learn from checkpoint-pi / pi-rewind? | turn_end + dedup + temp-index capture + safe restore + pruning; branch+worktree atomic pair as our delta |
| 3 | Build a Pi plugin? On what basis? | yes — one plugin implementing the skillgrid standard, built on pi-worktrunk (wt runner) + checkpoint-pi/pi-rewind (checkpoints) |
| 4 | Package naming (family will have multiple plugins)? | `@devopstales/pi-worktree`; repo `devopstales/pi-worktree`; sibling package, not code in skillgrid-skills |
| 5 | Write the briefing where? | `.skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md` in this repo (this file) |

## Open Questions & Assumptions

- **Question:** is the current `wt` release line (at spec time) the right
  minimum version pin for the directive protocol? — confirm the exact
  version at build step 2 and record it in the plugin README.
- **Question:** does the family want a shared `@devopstales/pi-*` manifest
  convention doc (first plugin sets the precedent: two-extension-file
  layout, changelog structure, evals format)? — propose with the first
  release.
- **Assumption:** the plugin's code lives in a *new* repo
  (`devopstales/pi-worktree`), published to npm scoped `@devopstales`; this
  repo's change spec governs it, and the in-repo doc edits above keep the
  skills honest in both directions.
- **Assumption:** the Goldfish shootdown's implementation-readiness lens runs
  after `scratch-core.ts` exists (owed per Shootdown Findings), before build
  step 2 starts — the ratchet from brainstorm's "hidden complexity" rule.

## Decisions (ADR)

- None yet — no decision for this change has been written to
  `.skillgrid/projectwiki/decisions/` (the vault is not initialized in this
  repo). Candidates that clear the bar once the vault exists:
  (1) two-extension-files-in-one-manifest over two npm packages;
  (2) wrapper-enforced invariants (merge default rewrite, allow-list at
  creation) over pass-through purity.

## Terms

- **Tier (tooling tier):** who creates/manages worktrees —
  harness-native → worktrunk → git fallback (`_shared/work-unit-commits` §
  Tooling tiers). Sharpened in round 1 of this change.
- **Scratch checkpoint:** a non-committing `refs/skillgrid/` snapshot of
  HEAD+index+worktree between commits (§ Scratch checkpoints). Distinguished
  from the *commit-as-checkpoint* rule: commits are the durable record,
  scratch refs are the in-between safety net and are pruned.
- **Atomic pair:** the branch+worktree-root pair recorded in a checkpoint;
  restore verifies both. New term introduced by this change's design (the
  hook's allow-list knows the branch; the pair is the checkpoint's
  contribution).
- **Linked session:** a Pi session created at a worktree's directory when
  worktrunk's directive moves the session; the source session remains
  resumable (pi-worktrunk v0.5 term, adopted).
- **No-replay recovery:** recovery to a surviving worktree that does not
  execute a command requested after the original directory died
  (pi-worktrunk v0.12.1 term, adopted).
