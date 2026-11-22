# @devopstales/pi-worktree — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:execute (subagent mode recommended, or inline mode) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread

**Goal:** Build the `@devopstales/pi-worktree` Pi plugin — two extension files (scratch checkpoints + worktrunk runner) whose pure-git cores implement the skillgrid worktree standard (`refs/skillgrid/` scratch layer, directive-based worktrunk runs with linked-session movement, skillguard invariants enforced in the wrapper).

**Architecture:** Two-layer split throughout: `src/core/` is pure (scratch-core imports only `node:*`; wt-core spawns `wt`/`git` but imports no Pi package), independently testable without a Pi install; `src/*.ts` extension files wire Pi events to the cores. The scratch core re-uses pi-rewind's temp-index mechanics with the skillgrid deltas (refs namespace, `.worktree` filter, atomic pair); the wt core re-uses pi-worktrunk's directive protocol with the skillguard invariants (standard-root check, allow-list at creation, merge-default rewrite). Per `findings.md` R1–R5.

**Tech Stack:** TypeScript (Node/Bun), Pi extension API (`@earendil-works/pi-coding-agent`, Pi ≥ 0.99.1), `@earendil-works/pi-tui` (dialog/footer widgets only), `typebox` (tool schemas), git (plain, via `node:child_process` spawn — no shell), worktrunk `wt` on PATH (current release line; min version pinned in Task 7). Zero other runtime dependencies.

**Spec:** `.skillgrid/change-specs/2026-11-22-pi-worktree-plugin/briefing.md` (R1–R13)

**Findings:** `.skillgrid/change-specs/2026-11-22-pi-worktree-plugin/findings.md` (R1 directive protocol + linked sessions; R2 checkpoint shape; R3 capture mechanics + bench; R4 worktrunk CLI semantics; R5 the in-repo contract)

**Prototype:** skipped (user opt-out — pi-rewind's core is a proven reference implementation; the directive risk is covered by the min-version pin, startup check, and Task 7's integration tests)

**Wiki:** None — the projectwiki vault is not initialized in this repo (recorded in briefing § Impact on Global Docs; the durable pages are owed to the plugin repo's own vault at its first ship).

## Terms

Pointers only (canonical terms live in the glossary files; none exist yet in this repo — the briefing's § Terms section is the interim record): **tooling tier** (who creates worktrees: harness-native → worktrunk → git fallback), **scratch checkpoint** (`refs/skillgrid/` non-committing snapshot), **atomic pair** (branch + worktree-root recorded per checkpoint; restore verifies both), **linked session** (Pi session created at a worktree move; source stays resumable), **no-replay recovery** (recovery that does not execute a command requested after the directory died). All five defined in `briefing.md` § Terms.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- MH-1: A turn that mutated the worktree produces exactly one new ref under `refs/skillgrid/` for the session, with the full metadata block (branch, head, index-tree, worktree-tree, session, turn, label, created, untracked, largeFiles, largeDirs); the real git index is byte-identical before/after capture.
- MH-2: A turn with no worktree change produces no new ref (worktree-tree SHA dedup; `createCheckpoint` returns `skipped`).
- MH-3: `restoreCheckpoint` on a matching atomic pair restores a dirty tree (deleted tracked files return, post-checkpoint untracked files are removed, pre-existing untracked files survive, ignored/large skipped items are never deleted) and always takes a before-restore checkpoint first.
- MH-4: `restoreCheckpoint` is refused — with the mismatch named — when the current branch OR the current worktree root differs from the recorded pair.
- MH-5: Refs are bounded: the 51st capture in a session prunes the oldest (50 max); at session start, other sessions' refs are pruned in the background.
- MH-6: `createCheckpoint` completes in ≤150ms on a 5,000-file repo (bench output, per pi-rewind's flat-latency result in `findings.md` R3).
- MH-7: A `wt` invocation whose directive names a directory inside the original repository moves the Pi session to a linked session in that directory; the source session stays `/resume`-able; in print/json mode the result is `stopped` + restart directory, no move.
- MH-8: When the session's cwd is gone, the next wt call recovers to a surviving worktree (repository identity re-verified, main preferred) and does NOT run a command requested after the directory died.
- MH-9: First `wt` use verifies the standard root (path template lands worktrees under `<projectroot>/.worktree/`); a foreign layout is reported with the observed path, never silently followed.
- MH-10: In a worktree context, `switch --create` with a branch outside `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` is rejected with the nearest allowed form suggested; the primary checkout is unconstrained.
- MH-11: `merge` calls execute with `--no-squash --no-commit` prepended unless the caller explicitly passed either flag; the executed argv is observable in the result.
- MH-12: A `wt` call needing project-command approval shows the dialog; approving saves exactly those commands (approvals file updated) and retries once; no code path injects `-y`/`--yes`.
- MH-13: Markers follow the lifecycle (`session_start`→💬, `agent_start`→🤖, `agent_end`→💬, `session_shutdown`→clear) and are visible in `wt list`; a stale marker is reported and manually clearable.
- MH-14: `npm test` passes in a bare Node/Bun environment with only git + `wt` on PATH (no Pi install); an import-scan test fails the build if any `src/core/` file imports `@earendil-works/*`.
- MH-15: The contract-drift guard test fails when any plugin constant diverges from the value stated in `_shared/work-unit-commits/README.md`, naming the mismatch.

**Artifacts** (files that must exist with real implementation, not stubs):
- [`src/core/scratch-core.ts` — capture/restore/prune/list, temp-index mechanics, all constants]
- [`src/core/wt-core.ts` — directive runner, recovery, standard-root/allow-list/merge-default invariants]
- [`src/scratch.ts` — scratch extension: turn_end wiring, `scratch-restore` tool, `/checkpoint` command, footer]
- [`src/wt.ts` — wt extension: `worktrunk` tool, `/wt` command, linked-session moves, approval dialog]
- [`src/markers.ts` — marker lifecycle shared by both extensions]
- [`src/ui.ts` — footer status + restore dialog (files-only / cancel)]
- [`tests/scratch.test.ts`, `tests/wt-core.test.ts`, `tests/import-scan.test.ts`, `tests/contract-drift.test.ts`, `tests/bench.ts`]
- [`package.json` — `pi.extensions` manifest with both extension files; zero runtime deps beyond Pi packages + typebox]
- [`evals/evals.json` — the two ported execute scratch scenarios + marker/approval scenarios]
- [`references/scratch-restore.md`, `references/wt-commands.md` (generated catalog)]
- [`changelog/releases/v0.1.0/` — per-release entry structure]
- In-repo (skillgrid-skills): `README.md` tool-list line + `_shared/work-unit-commits/README.md` reference-implementation pointer (Task 11).

**Key links** (critical connections that must work together):
- `scratch.ts`'s `turn_end` handler MUST call `scratch-core.createCheckpoint` with the session id from `ctx.sessionManager` — the ref id encodes it, and `pruneOtherSessions` at `session_start` keys off it.
- `restoreCheckpoint`'s pair check reads the *current* worktree root via `git rev-parse --show-toplevel` — it must be called with the session's cwd, not the repo root, or the worktree-root half of the check is vacuous.
- `wt.ts`'s directive follow MUST reuse `wt-core.recover`'s recorded-worktree list: recovery is only allowed for worktrees recorded *before* removal (R1 safety rule).
- `wt-core.applyMergeDefaults` runs BEFORE `runDirectedWt` assembles argv — the rewrite is on the command, not on the result.
- `markers.ts` is the only writer of worktrunk marker state in the plugin; both extension files import it, neither writes markers directly.

**One-way-door decisions** (hard to reverse — explicit approval requested at Task 1):
- **OD-1: ref namespace `refs/skillgrid/`** — once sessions checkpoint into it, pruning policies and the contract-drift guard reference it; renaming later orphans existing refs. (Briefing R1; contract § Scratch checkpoints.)
- **OD-2: restore refusal on worktree-root mismatch** (the second half of the atomic pair beyond pi-rewind's branch check) — stricter than both upstreams; loosening later is easy, but shipping lenient and tightening surprises restorers.
- **OD-3: merge-default argv rewrite** (`--no-squash --no-commit` prepended) — the one place the plugin mutates the user's command; a wrong default silently changes history shape for every `wt merge` through the tool.

## Validation

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| R1 | mutating turn → 1 ref, full metadata, index untouched | unborn HEAD (zero-SHA capture) | capture throws mid-batch (no partial ref) | `bun test tests/scratch.test.ts` (capture group) | a second ref for one turn, or a mutated real index |
| R2 | read-only turn → `skipped`, ref count stable | turn with only ignored-file changes (tree unchanged) | dedup compares wrong SHAs | `bun test` (dedup group) | a ref appears for a read-only turn |
| R3 | restore on matching pair: deletes return, new untracked removed, pre-existing survive | before-restore snapshot itself is a ref | branch OR root mismatch → refused with named reason | `bun test` (restore group) | a restored file deleted, or a cross-branch restore completing |
| R4 | 51st capture prunes oldest; startup prunes other sessions | exactly 50 refs (no prune) | prune during active restore | `bun test` (prune group) | 51+ session refs surviving, or a current-session ref pruned |
| R5 | directive move → linked session, source resumable | print mode → `stopped` + path, no move | directive outside original repo → rejected, stay put | `bun test tests/wt-core.test.ts` + `pi -e` smoke | session not moved in TUI, or moved in print mode |
| R6 | cwd gone → recover to survivor (main preferred), pending command NOT run | removed by another process (no directive) | no surviving worktree → report restart location | `bun test` (recovery group) | the pending command executing in the new worktree |
| R7 | standard template → silent pass | foreign template → drift reported with observed path | template set to a relative path | `bun test` (root-check group) | foreign layout silently followed |
| R8 | `switch --create agent-x` allowed in worktree context | `worktree-wf_auth` allowed; `feature-x` rejected with suggestion | primary checkout `feature-x` → allowed | `bun test` (allow-list group) | a rejected branch created anyway |
| R9 | `merge` → argv has both flags | explicit `--squash` → not rewritten for that flag | `--no-commit` present, `--no-squash` absent → only the missing one prepended | `bun test` (merge-defaults group) | a squash commit where work-unit commits were expected |
| R10 | unapproved hook → dialog, approve → runs once, approvals file updated | decline → command fails, approvals unchanged | non-interactive mode → fail + list what to approve | `bun test` (approval group) | `-y` appearing in any executed argv |
| R11 | lifecycle → 💬/🤖/💬/clear in `wt list` | kill before shutdown → stale marker reported | manual clear removes it | `bun test` (marker group) + `wt list` | a marker left after `session_shutdown` |
| R12 | `npm test` green without Pi installed | core import scan: a Pi import in `src/core/` fails the build | bench at 5k files > 150ms | `bun test` (import-scan) + `bun run bench` | any core test requiring a Pi install |
| R13 | constants match the contract file | a contract value changed → guard fails naming the diff | contract file moved (path fallback error, not silent pass) | `bun test tests/contract-drift.test.ts` | a diverged constant with a green guard |
| out-of-scope | — | — | — | — | conversation restore, per-tool checkpoints, auto-restore, picker TUI, OSC 7, alias machinery are NOT proven by this plan |

## Threat Matrix

Per the threat matrix convention (`_shared/references/threat-matrix.md` in the
skillgrid-skills repo): one row per applicable class, each with a test.

| Threat class | Applicable | Execution boundary | Test |
|---|---|---|---|
| Untrusted input | **Applicable** — worktrunk stdout/stderr and the directive file content are external (a malicious or buggy `wt` write could name any path); checkpoint ref metadata is read back from git on restore | Directive path must resolve inside the original repository's worktree set (R1 safety rule); ref ids validated against `^[\w-]+$` before use in `update-ref`/`for-each-ref`; ref metadata fields parsed, never interpolated into a shell (spawn with arg arrays, no shell) | `tests/wt-core.test.ts`: directive naming a path outside the repo → rejected, stay put; ref id with `/` or `..` → rejected at `loadCheckpointFromRef` |
| Git repository selection | **Applicable** — the plugin runs in worktrees, the primary checkout, and across moves; `git -C` vs relative cwd errors are the classic failure | every git call in the cores is `spawn("git", [...], {cwd: root})` with `root` from `git rev-parse --show-toplevel` captured per operation; the pair check compares `--show-toplevel` outputs (realpath'd), never user-typed paths | `tests/scratch.test.ts`: pair check from a subdirectory of the worktree (same toplevel → pass); from the primary checkout (different toplevel → refuse) |
| Commit state | **Applicable** — capture runs against unborn HEAD, detached HEAD, dirty index, partial staging | unborn HEAD → zero-SHA head, `read-tree` seed skipped (R3 `ZEROS`); detached HEAD in a worktree → captured, restore refused by the branch half of the pair (named) | `tests/scratch.test.ts`: capture on a fresh `git init` repo succeeds with zero-SHA; detached-HEAD restore refused with "branch mismatch: (detached)" |
| Push state | N/A — the plugin never pushes; `wt merge` is local-only (R4: "never fetches"), pushes stay with `ship` Step 6 | — | — |
| PR commands | N/A — no forge interaction in the plugin | — | — |
| Documentation-like paths | N/A — no dependency manifests or executable docs are parsed; `package.json` is written, not parsed-for-behavior | — | — |
| Concurrency | **Applicable** — the `worktrunk` tool is `executionMode: "sequential"` (R1); two `wt` calls in flight would race the directive file and the session state | the tool schema enforces sequential execution; `wt-core` holds an in-process mutex around directive-file creation/reading anyway (defense in depth for codemode queued calls) | `tests/wt-core.test.ts`: two concurrent `runDirectedWt` calls → the second waits (mutex), both complete, both directives honored in order |
| Filesystem boundaries | **Applicable** — temp directive file and temp index live in `os.tmpdir()`; the worktree root is user-configured (template) | temp files are `mkdtemp` + `0o600`, deleted in `finally`; the standard-root check (R7) runs before any creation; `.worktree` is in the snapshot filter so worktrees never snapshot each other | `tests/wt-core.test.ts`: directive file deleted after failed invocation (no temp litter); `tests/scratch.test.ts`: an untracked `.worktree/` dir is absent from the captured tree |

## Global Constraints

- **Pi floor: 0.99.1+** (linked-session API; matches pi-worktrunk). `package.json` `engines` / README state it.
- **`wt` requirement:** current release line on PATH (the `WORKTRUNK_DIRECTIVE_CD_FILE` protocol is worktrunk-internal — `findings.md` R1). Startup check prints protocol support; absence → wt extension inert + "git fallback applies (skill Step 0c)" report. Min version pinned in the README at Task 7 (the one open fact from the briefing).
- **Zero runtime dependencies** beyond `@earendil-works/pi-coding-agent`, `@earendil-works/pi-tui`, `typebox` (all provided by the Pi ecosystem). No shell in any git/wt invocation — `spawn(cmd, args[], {cwd, env, stdio})` only (threat matrix: untrusted input).
- **Core purity:** nothing in `src/core/` imports `@earendil-works/*` (enforced by `tests/import-scan.test.ts`, MH-14).
- **Ref namespace:** `refs/skillgrid/` (OD-1). Ref ids: `^[\w-]+$`.
- **Constants are the contract's values, verbatim** (`findings.md` R5): `AGENT_BRANCH_RE = /^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$/`; `IGNORED_DIRS` = the 13 upstream dir names (`node_modules`, `.venv`, `venv`, `env`, `.env`, `dist`, `build`, `.pytest_cache`, `.mypy_cache`, `.cache`, `.tox`, `__pycache__`) + `.worktree`; `MAX_UNTRACKED_FILE_SIZE = 10 * 1024 * 1024`; `MAX_UNTRACKED_DIR_FILES = 200`; `MAX_PER_SESSION = 50`. Enforced by the contract-drift guard (R13).
- **Branch naming:** worktree-context creation restricted to the allow-list; primary checkout unconstrained (R8).
- **Merge default:** `--no-squash --no-commit` prepended unless explicitly overridden (R9, OD-3).
- **Standard root:** worktrees under `<projectroot>/.worktree/` via `worktree-path = "{{ repo_path }}/.worktree/{{ branch | sanitize }}"`; drift reported, never silently followed (R7).
- **No auto-restore, ever:** every restore is a decision with diff preview (briefing out-of-scope; scope boundary in the contract).
- **No `-y`/`--yes` injection** anywhere in the wt runner (R10).
- **Markers:** `markers.ts` is the sole marker writer when this plugin is installed; warn at `session_start` if worktrunk's own Pi plugin is also detected (briefing shootdown 🟡).
- **Commit discipline** (per `_shared/work-unit-commits`): conventional commits, no AI-attribution trailer, `[skillgrid-context]` block in the body of each work-unit commit; spec-zone edits (this repo's `README.md` + `work-unit-commits/README.md`) commit before code-zone.
- **Package:** `@devopstales/pi-worktree` on npm; repo `devopstales/pi-worktree`; MIT.

## File Structure

New repo `devopstales/pi-worktree` (sibling of `skillgrid-skills`):

- `package.json` — name/version/license; `pi.extensions: ["./src/scratch.ts", "./src/wt.ts"]`; scripts (`test`, `bench`); zero runtime deps beyond the Pi packages + typebox.
- `src/core/scratch-core.ts` — **pure git, zero Pi imports.** Capture (temp-index mechanics per `findings.md` R3), restore (pair check + before-restore snapshot + safe-clean), list/load, prune/pruneOtherSessions. Exports the constants block (Global Constraints) as named exports so the drift guard and the UI can reference them.
- `src/core/wt-core.ts` — spawns `wt`/`git`, no Pi imports. `runDirectedWt` (directive protocol per R1), `recover` (recorded-worktree list, identity re-check, main preference, no-replay), `verifyStandardRoot`, `checkBranchAllowList`, `applyMergeDefaults`, `supportsDirectiveProtocol` (startup check), in-process mutex.
- `src/scratch.ts` — extension file 1. `turn_end` → capture + footer; `session_start` → cache rebuild + resume checkpoint + background prune; registers `scratch-restore` tool (`{id?}`) and `/checkpoint` command; restore dialog via `ui.ts`.
- `src/wt.ts` — extension file 2. `worktrunk` tool (`{command, args?}`, sequential, compact description + on-demand `--help` catalog per R1) and `/wt` command; directive follow → linked-session move (print/json → `stopped`); recovery hook; approval dialog passthrough.
- `src/markers.ts` — marker lifecycle (R11); sole marker writer.
- `src/ui.ts` — footer `◆ N checkpoints`; restore dialog (files-only / cancel).
- `tests/scratch.test.ts` — capture/dedup/restore/prune/pair-check/unborn-HEAD/detached groups (R1–R4).
- `tests/wt-core.test.ts` — directive follow, no-replay recovery, standard-root, allow-list, merge-defaults, approval, mutex groups (R5–R10, R12); runs against a real `wt` binary in a temp repo (skipped with a named reason when `wt` is absent — the pure-parse groups like `applyMergeDefaults` still run).
- `tests/import-scan.test.ts` — fails if any `src/core/**` imports `@earendil-works/*` (R12/MH-14).
- `tests/contract-drift.test.ts` — greps the skillgrid-skills contract file for the constants and diffs against the exported values (R13/MH-15); the contract path is an env override (`SKILLGRID_CONTRACT_PATH`) so the plugin repo's CI can point at a pinned copy.
- `tests/bench.ts` — capture latency at 20 / 500 / 5,000 / 50,000 files (MH-6 table).
- `evals/evals.json` — ported `scratch_dedup_readonly_turn` + `scratch_branch_safety_refused` (from skillgrid-skills `execute/evals/scenarios.yaml`) + marker-lifecycle + approval scenarios.
- `references/scratch-restore.md` — on-demand model doc: the restore protocol, refusal reasons, the pair.
- `references/wt-commands.md` — generated compact command catalog (regenerated by `bun run catalog` from the installed `wt --help` output; committed so the tool description stays small per R1).
- `changelog/releases/v0.1.0/{manifest.yaml,entries/*.md,notes.md}` — pi-worktrunk's per-release structure.
- `README.md` — install, requirements (Pi ≥ 0.99.1, wt min version), the two-extension-file manifest note, the marker-writer warning, git-fallback behavior.

In-repo (skillgrid-skills), Task 11:
- `README.md` — Tool Dependency list: the `pi-worktree` line.
- `_shared/work-unit-commits/README.md` § Scratch checkpoints — the reference-implementation pointer line.


