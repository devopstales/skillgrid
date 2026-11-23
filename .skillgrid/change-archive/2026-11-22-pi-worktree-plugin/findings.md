# Findings — @devopstales/pi-worktree

> Change: `2026-11-22-pi-worktree-plugin`
> Mode: single-pass (all primary sources fetched and read this session; paths
> local or cited). Per the research epistemics, every claim below traces to a
> source retrieved this run; nothing is asserted from training data.

## Research: what the upstream extensions actually implement

### R1 — pi-worktrunk (mavam) v0.13.2: the worktrunk runner + session movement

**Source:** clone at `github.com/mavam/pi-worktrunk` (read this session:
`README.md`, `worktrunk.ts` 1670 LOC, changelog releases v0.1.0 → v0.13.2).

- **Directive protocol (load-bearing for R5/R6).** The runner sets
  `WORKTRUNK_DIRECTIVE_CD_FILE` to a private temp file per invocation
  (`mkdtemp` + `0o600`), plus `WORKTRUNK_SHELL_CWD` = the caller's cwd (so
  worktrunk treats Pi as the caller, not a shell that launched Pi elsewhere),
  and `PWD`. After the child exits, the file's content is the requested
  directory (`directive`). [worktrunk.ts:34-83] **Confidence: high** (source
  read). The env var is a worktrunk-internal protocol, not documented public
  API — the README requires "current wt with the WORKTRUNK_DIRECTIVE_CD_FILE
  protocol." [README.md:Requirements]
- **Directive semantics.** Pi follows a valid directive even if a later hook
  fails (failure reported in the destination session); without a directive,
  Pi stays put as long as its cwd remains usable. Each invocation uses a
  private temp reply file, deleted afterward. [README.md]
- **Recovery (v0.12.1).** When the working directory is gone (removed by a
  `wt` command or by another process), the next `/wt` command or `worktrunk`
  tool call recovers the session to a surviving worktree in the same
  repository, preferring main. Recovery creates a linked session labeled
  "Session recovered." **A command requested after the directory became
  unusable is NOT run** — review it in the new worktree before retrying. If a
  command removed its own directory, its result is preserved and not replayed.
  [changelog v0.12.1 entry; README.md] **Confidence: high.**
- **Safety.** A requested directory is validated to belong to the original
  repository. Recovery only uses previously *recorded* worktrees whose
  repository identity still matches. Rejected directives and cancelled moves
  never trigger a fallback move. Worktrunk retains control of hooks,
  project-command approvals, dirty-worktree checks, force flags, branch
  deletion. [README.md:Safety]
- **Linked sessions.** Each move creates a linked Pi session in the
  destination; the source session remains available through `/resume`.
  Session movement requires TUI or RPC mode; in print/JSON mode a directory
  request *stops* continuation and reports where to restart. [README.md]
- **Markers (v0.1.0).** `session_start`→💬, `agent_start`→🤖, `agent_end`→💬,
  `session_shutdown`→clear, written to worktrunk's branch marker state. A
  stale marker can remain if the agent is killed before session-end;
  `wt config state marker clear` removes it manually. [changelog v0.1.0;
  README.md:Status markers]
- **Approvals (v0.10.0).** When a `wt` command needs project-command
  approval, Pi shows every unapproved command in a confirmation dialog;
  approving saves exactly those commands (as `wt config approvals add`) and
  retries once. Aliases ask before they run (a retry could repeat steps).
  The model cannot pass `-y`/`--yes`. Non-interactive mode: the command fails
  and lists what to approve. [changelog v0.10.0; README.md]
- **Tool surface (v0.13.0).** One `worktrunk` tool: `{command, args?}`; the
  description is compact (commands + global options + aliases only) with an
  on-demand `--help` reference (help calls return immediately, matching the
  *installed* binary). `executionMode: "sequential"`; calls must be
  sequential. Codemode: inspection commands (`list`, `config show`,
  `hook show`, `step diff`) return immediately; mutations return
  `status: "queued"` in TUI/RPC (run after the turn ends); print/JSON mode
  executes immediately, `status: "stopped"` means restart Pi in the reported
  directory. `promptGuidelines` include "Never run Worktrunk calls in
  parallel" and "Never pass -y/--yes." [worktrunk.ts:1489-1510; README.md]
- **Pi API surface used (extracted from source):**
  `pi.registerTool({name, label, description, promptSnippet,
  promptGuidelines, parameters, outputSchema, executionMode, execute})`,
  `pi.registerCommand("wt", {description, getArgumentCompletions, handler})`,
  `pi.exec("wt"|"git", args, {cwd, signal, timeout})`, `pi.on("session_start"
  | "agent_start" | "agent_end" | "session_shutdown")`,
  `ctx.sessionManager.getSessionFile()/getHeader()/getEntries()`,
  `ctx.mode` (`"tui"|"rpc"|"print"|"json"`), `ctx.ui.select/notify`,
  `ctx.cwd`. Requires **Pi ≥ 0.99.1**. [worktrunk.ts:16-21, 964-990,
  1237, 1641-1668; README.md:Requirements]
- **Cuts we do NOT take** (from the same sources): interactive picker TUI,
  OSC 7 terminal directory reporting, alias passthrough machinery,
  slash-command display styling, the full `pi-tui` widget surface.

### R2 — checkpoint-pi (prateekmedia) v1.0.5: the checkpoint shape

**Source:** package page `pi.dev/packages/checkpoint-pi` (fetched this
session; v1.0.5, published 2026-05-16).

- Saves the **full worktree (tracked + untracked) at the start of every
  turn**; stores snapshots as **git refs under `refs/pi-checkpoints/`**,
  persisting across sessions. Captures "HEAD, index, and worktree state."
  [package page:What it does / How it works]
- Creates a **"before restore" checkpoint automatically** to avoid losing
  current work. [package page]
- Restore options: files + conversation, conversation only, files only.
  [package page]
- **Smart filtering:** never snapshots 13 ignored dir patterns
  (`node_modules`, `.venv`, `venv`, `env`, `.env`, `dist`, `build`,
  `.pytest_cache`, `.mypy_cache`, `.cache`, `.tox`, `__pycache__`, +1);
  excludes untracked files **>10MiB** and untracked dirs with **>200
  files**. [package page:Smart Filtering]
- **Safe restore never deletes:** files in ignored directories, large
  files/dirs excluded from the snapshot, and **pre-existing untracked files
  that existed when the checkpoint was created**. [package page:Safe Restore]
- Architecture: `checkpoint.ts` (extension glue) + `checkpoint-core.ts`
  (core git operations, **no pi dependencies**) + tests for the core.
  [package page:File structure]

**Confidence: high** (first-party page; mechanics corroborated by the pi-rewind
source, R3).

### R3 — pi-rewind (arpagon): the capture mechanics + performance

**Source:** clone at `github.com/arpagon/pi-rewind` (read this session:
`src/core.ts` 723 LOC, `src/index.ts`, `README.md`, `docs/planning/
2026-02-24/implementation-plan.md`).

- **Timing + dedup (the converged design):** 1 resume checkpoint on session
  start; **1 checkpoint at `turn_end`** (after ALL tools in a response);
  label = user prompt + mutating tools that ran; **no per-tool or
  per-turn-start checkpoints** — "noisy, redundant." A turn with an
  unchanged worktree tree creates no checkpoint (smart dedup). [index.ts
  header comment; README.md:Features]
- **Capture mechanics (temp index):** `GIT_INDEX_FILE` pointed at a
  `mkdtemp` tempdir; seed with `read-tree <HEAD>` (skipped when HEAD is
  unborn — zero-SHA `ZEROS`); `add --all` filtered files **in batches of
  100**; `write-tree` → worktree tree SHA. The real index is untouched; no
  commit in history. The result is a `commit-tree` object (message =
  metadata block: id, sessionId, trigger, turn, branch, head, index-tree,
  worktree-tree, created, untracked JSON, largeFiles JSON, largeDirs JSON)
  stored at `update-ref refs/pi-checkpoints/<id>`. [core.ts:354-445]
- **Restore mechanics:** `reset --hard <headSha>` → `read-tree --reset -u
  <worktreeTreeSha>` → `read-tree --reset <indexTreeSha>` (real index
  restored) → `safeClean` (deletes files in the snapshot that are neither
  pre-existing-untracked nor skipped large). [core.ts:476-506]
- **Pruning:** 50 max checkpoints per session (oldest pruned); at startup,
  checkpoints from previous sessions are deleted (real-world test: 525 stale
  refs → 0), non-blocking/background. [core.ts; README.md]
- **Branch safety:** checkpoint records its `branch`; restore is blocked
  cross-branch ("avoids OpenCode bug"). **Records branch only — not the
  worktree root** (our atomic-pair delta). [README.md; core.ts CheckpointData]
- **Performance (bench, `tests/bench.ts`):** `createCheckpoint` 60–142ms
  constant from 20 files to a 182K-file / 87GB monorepo; `loadAll` (startup)
  8ms. Flat because the temp-index work is O(filtered files), not O(repo).
  [README.md:Performance]
- **Design history (why turn_end):** the planning doc ranked Cline #1 but the
  shipped code converged to one-per-turn at turn_end; per-tool was P1 and
  dropped. Feature-parity targets tracked (Claude Code `/rewind`, Gemini
  `/restore`, OpenCode `/undo /redo`) with the key open question being *what
  counts as a change* — pi-rewind/Cline track bash-induced changes;
  Claude/Gemini track AI edit-tool changes only. [docs/planning/
  2026-02-24/implementation-plan.md]

**Confidence: high** (source read; bench numbers are the project's own test
output, not independently re-run this session — flagged).

### R4 — worktrunk (max-sixty) CLI semantics the wrapper depends on

**Source:** clone at `github.com/max-sixty/worktrunk` (read this session:
`README.md`, `docs/src/content/docs/{config,hook,merge,remove,switch,
step,claude-code,tips-patterns}.md`).

- **Path template:** `worktree-path` in user config
  (`~/.config/worktrunk/config.toml`); variables `{{ repo_path }}`, `{{ repo
  }}`, `{{ branch }}`, `{{ branch | sanitize }}` (`/` and `\` → `-`).
  Inside-repo layout: `{{ repo_path }}/.worktrees/{{ branch | sanitize }}`.
  [config.md:Worktree path template] **Our standard root uses `.worktree`
  (singular) — the template is configured, not assumed.**
- **Merge pipeline:** commit → squash (default) → rebase → pre-merge hooks
  (after rebase, before merge; failure aborts) → fast-forward merge
  (**non-FF rejected**; `--no-ff` makes a merge commit) → pre-remove hooks →
  remove worktree + branch (`--no-remove` keeps) → post-* in background.
  `--no-squash` preserves commits; `--no-commit` skips the commit/squash step
  (rebase still runs unless `--no-rebase`). **`wt merge` merges source into
  target locally and never fetches.** [merge.md:Pipeline]
- **Branch deletion (six conditions):** same commit / ancestor / empty
  three-dot diff / matching tree SHA / simulated merge adds nothing /
  patch-id match — first failure stops; unmerged branches keep (or need
  `-D`). [remove.md:Branch cleanup]
- **Force + reaping:** `--force` (worktree) saves uncommitted changes to
  `refs/wt-backup/<branch>` first and prints a recovery command; `--reap`
  terminates detached processes by working directory (SIGTERM→SIGKILL),
  spares TTY-attached. [remove.md]
- **Hooks:** project config `.config/wt.toml` (**committed & shared**) —
  `[pre-start]` (blocking, at creation), `[post-start]` (background),
  `[pre-merge]` (blocking), etc. Template variables include `{{ branch }}`,
  `{{ worktree_path }}`, `{{ base }}`, `{{ target }}`. [hook.md; config.md]
- **Copy-on-write caches:** `wt step copy-ignored --from <donor>` copies
  gitignored files (build caches) between worktrees; reflinked on
  APFS/btrfs/XFS (14GB `target/`: 20s/~0 disk vs `cp -R` 2m/14GB), full copy
  on ext4/NTFS. [step.md:copy-ignored]
- **Agent plugins:** worktrunk ships its own Pi/oh-my-pi plugin — **activity
  markers only** (it needs worktree-lifecycle hooks that only Claude Code and
  OpenCode expose, so Pi users "invoke `wt switch --create` and `wt remove`
  directly"). **Confidence: high.** [claude-code.md] — *this is the
  marker-writer collision our plugin must handle (briefing shootdown 🟡).*

### R5 — the contract being implemented (this repo)

**Source:** read this session from `skillgrid-skills` (the working tree,
post the 2026-11-22 edits):

- `_shared/work-unit-commits/README.md` — **Tooling tiers** (harness-native →
  worktrunk preferred-when-present → git fallback; path template pinned to
  `{{ repo_path }}/.worktree/{{ branch | sanitize }}`; per-agent allow-list
  `agent-*` / `worktree-agent-*` / `worktree-wf_*`); **§ Scratch
  checkpoints (turn-level)** (`refs/skillgrid/`, turn_end + dedup, temp-index
  capture, filter set = R2's 13 dirs + `.worktree`, 10MiB/200-file caps,
  pre-existing-untracked never-delete, before-restore snapshot, **branch +
  worktree-root atomic pair**, 50/session + cross-session prune,
  `checkpoint-state.js` subcommands `scratch` / `scratch-restore <id>`, no
  auto-restore); **§ Worktree removal + branch deletion** (no-added-changes:
  same-commit → ancestor → empty three-dot diff; force only after a backup
  ref `refs/skillgrid/pre-remove-<branch>`; `--reap`; wave-end
  reconciliation); **§ Worktree lifecycle (forking)**.
- `execute/SKILL.md` — Step 0b tiers (Tier 2 requires explicit `--base <base>`
  — worktrunk's default base is the default branch, wrong for a
  `develop`-off feature); Step 0d (`.config/wt.toml` `[pre-start] setup` +
  `[pre-merge] test`, `wt step copy-ignored` first); fan-out (cwd binding,
  `worktree-wf_<domain>`, interrupted-worker scratch restore, `wt merge
  <parent> --no-squash --no-commit`).
- `ship/SKILL.md` — Step 6 Option 1 (`wt merge <base> --no-squash
  --no-commit`, cleanup per shared safety) and Option 4 (fork).

**Confidence: high** (this repo, current working tree).

## Synthesis — what the blueprint must decide from these facts

1. **Two proven cores to re-use by mechanics, not by code:** pi-rewind's
   temp-index capture/restore/prune (R3) is the scratch-core skeleton;
   pi-worktrunk's directive runner + recovery + linked-session move (R1) is
   the wt-core skeleton. Both are MIT; both were read as source this session.
   The skillgrid deltas (R5) are: `refs/skillgrid/` namespace, `.worktree`
   filter, **worktree-root half of the pair**, allow-list *at creation*,
   merge-default rewrite, standard-root verification, contract-drift guard.
2. **The single highest-risk fact** is the directive protocol (R1): it is a
   worktrunk-internal env var. Mitigations already in the briefing: min
   `wt` version pin, startup protocol check, stay-put-and-report fallback.
   The min version itself is the one fact NOT yet pinned — a build-step-2
   check records it (briefing Open Questions).
3. **Marker collision is real:** worktrunk's own Pi plugin writes the same
   markers (R4). Resolution: pi-worktree is the only marker writer when
   installed; warn at `session_start` if both are detected.
4. **Bench expectation to hold:** ≤150ms capture at 5k files (R3 measured
   60–142ms at up to 182k files) — becomes Must-Have MH-6.
5. **No new external dependencies** for the plugin runtime (R2/R3 both run
   zero-dep; R1's deps are the Pi packages + typebox only).
