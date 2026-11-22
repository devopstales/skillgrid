# Work-Unit Commits (shared commit contract)

The single source of truth for commit discipline: conventional-commit format,
atomic + independently-revertable sizing, when-to-commit, the
`[skillgrid-context]` block, and the git-hook safety guards. Consumed by the
execution skills (`execute` — the subagent / inline / fan-out modes that
absorbed `subagent-execution` and `simple-execution`) and by `blueprint`
(spec-zone commit rule, phase-transition commits). **Fully active by default
during execution** — not on-demand. No consumer restates this contract; they
cite it.

**Config:** Read `.skillgrid/config.yaml`. Use `conventions.commit_style`
(`conventional` by default) and `conventions.scratch_dir`
(`.skillgrid/work`).

## Worktree root + detection (shared snippet)

The git worktree root is `<projectroot>/.worktree` (singular). Both `execute`
(isolated-workspace setup) and `ship` (integration) use this location and the
detection plumbing below. Reuse it; do not re-derive it.

```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
```

`GIT_DIR != GIT_COMMON` → linked worktree, **unless** you are inside a git
submodule — `git rev-parse --show-superproject-working-tree` printing a path
means a normal repo, not a worktree.

**Directory selection priority** (for `execute` 0c): (1) a worktree directory
declared in your instructions, (2) `conventions.worktree_dir` from config,
(3) the project-root `.worktree/` — the default when config is absent or
unset. The resolved directory MUST live **inside `<project-root>`** — never a
sibling of the checkout, never outside the repo root; a bare path with no
`/` is a sibling of the root, not a child of it. **MUST verify the
directory is ignored before creating** (`git check-ignore -q <dir>`) — if not
ignored, add it to `.gitignore` and commit first, or the whole tree gets
committed. Onboarding creates the root (gitignored, with a `.gitkeep`) at
project setup — `onboarding-write.md` Step 3 items 5/5b — so by execution
time it exists; a missing root at execute time is an onboarding gap, not a
reason to invent a new location.

**Tooling tiers** (who creates and manages worktrees; `execute` Step 0 and
`ship` Step 6 follow this order):

1. **Harness-native tool** — if the harness provides one (`EnterWorktree`,
   `WorktreeCreate`, a `/worktree` command, or a `--worktree` flag), it wins:
   it owns session state.
2. **Worktrunk (`wt`) — preferred when present.** Check `command -v wt` and
   that `wt list` succeeds in the repo. Worktrees MUST land in the standard
   root: the repo-root `.config/wt.toml` (or user config) MUST set
   `worktree-path = "{{ repo_path }}/.worktree/{{ branch | sanitize }}"`
   (branch `/` → `-`; dispatch briefs state the sanitized name so ledger paths
   resolve) — without the template, `wt` places worktrees in its own default
   layout *outside* `.worktree/`, which `git worktree list` does not flag.
   Verify the created worktree's path starts with `<project-root>/.worktree/`
   after every `wt switch --create`; a path outside the standard root is a
   placement error — record it, report it, do not silently work there. Fan-out
   branches stay on the per-agent allow-list (`worktree-wf_*` for fan-out
   domains, `agent-*` for single implementers) — worktrunk addresses by branch
   name and is agnostic to it. Project hooks (setup, pre-merge tests) live in
   the **committed** `.config/wt.toml`.
3. **`git worktree add` — the fallback.** Only when no tool from tier 1/2
   exists.

Worktrunk is optional: every rule in this contract has a plain-git equivalent,
and a machine without `wt` runs the git path with no other difference.

## The two layers

| Layer | Where | What | Who writes it |
|-------|-------|------|---------------|
| **Durable record** | the commit's body | `[skillgrid-context]` block (Task / Decisions / Remaining / Tried) | you, at commit time |
| **Commit event** | the session event stream | parsed Task / Decisions / Remaining from the block, queryable per session | the hooks, at commit time |

The `[skillgrid-context]` block is the source of truth for *decisions*; git
history is the source of truth for *state*. A fresh session resumes from
`git log` plus the session's events (see `skillgrid:resume`).

## Commit format

Conventional commits (per `conventions.commit_style`). Subject regex:

```
^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._-]+\))?!?: .+
```

- `type` required; `(scope)` optional lowercase; `!` marks a breaking change.
- **No `Co-Authored-By`, `Generated-By`, or any AI attribution trailer.**
- Body: bullet the key changes, then the `[skillgrid-context]` block:

```
feat(auth): add session refresh

- sliding-window refresh in auth/session.ts
- refresh middleware wired into the token route

[skillgrid-context]
Task: auth/session-refresh
Decisions: sliding window over absolute expiry (simpler, no re-login storm)
Remaining: wire refresh into the token middleware
Tried: absolute 15m expiry — caused re-login on long sessions
[/skillgrid-context]
```

The `commit-msg` hook rejects a non-conventional subject and any
`Co-Authored-By`/`Generated-By` trailer.

## Sizing: atomic + independently-revertable

- **One logical change per commit.** Never bundle a feature + a refactor + a
  config change — that's three commits.
- **~100 lines as the soft ceiling** for a single change (a vertical slice that
  must move together is fine; use the scope to name it).
- **Independently revertable:** prefer additive changes; do not delete something
  and replace it in the same commit — split into a delete commit and an add
  commit.
- **Spec before code (zone rule, BDD is always on):** commit
  `<specs_root>` (default `.skillgrid/change-specs/`) changes *before* the code
  that satisfies them. Never leave both uncommitted.
- **Commit first, then review.** Review diffs `merge-base...HEAD`; an uncommitted
  change is invisible to it. Commit, review, then amend or add a fixup.

## PR review budget (400 changed lines)

**400 changed lines (`additions + deletions`) per PR** is the review budget.
A PR above it needs one of:

1. **Chain it** — split into chained/stacked PRs (see the chain-strategy
   decision table below). This is the default path.
2. **Documented exception** — an explicit human-selected `size:exception`
   with a written over-budget rationale, recorded in the PR body.

Count authored additions plus deletions from the forge API after the PR is
created (or `git diff --stat <base>...<head>` locally, labeled *provisional*,
before). Generated goldens are excluded from the authored count but included
in snapshot identity checks.

Forecast the budget *before* implementing: if the work-unit table in
`tasks.md` forecasts a >400-line change, group the units into chained PR
slices before the first commit, not after the diff is big.

## Chain strategy decision

The project-level choice lives in `chain.strategy` (`.skillgrid/config.yaml`,
set at onboarding — `skillgrid:ship` Step 6 resolves the base branch from it).
When the slicer must pick a strategy for a specific change:

| Condition | Strategy |
|---|---|
| Every slice lands independently on the base | **Stacked PRs** — each slice is its own PR against the base; merge in order. |
| The feature must integrate as one atomic unit (slices do not work alone) | **Feature branch chain** — draft/no-merge tracker PR on the feature branch; slice PR #1 targets the tracker branch, later slices target the immediate parent branch. The tracker stays draft until all children merge. |
| A slice alone is >400 lines | Split the slice into smaller work units; a slice that cannot split takes the documented exception path. |

**Cross-fork limitation:** the forge does not support cross-fork base refs.
If the slice branches live only in a contributor's repo, only *stacked to
base* works contributor-side; the feature-branch chain needs the slice
branches pushed to the main repo, which only a maintainer can do. State which
side holds the branches before choosing.

## PR body honesty

Every `[x]` in a PR checklist is a **public claim requiring evidence**
(per `../rules/evidence-claims.md`):

- API-backed claims (labels, issue linkage, line counts) — `[x]` only when
  the forge API shows it. If the label is absent, leave the box unchecked and
  record the pending action.
- Test claims — `[x]` only for a command run in this session with captured
  output; "tests pass" is not a claim from memory.
- Counts follow the lifecycle: local `git diff --stat <base>...<head>` is
  *provisional* before the PR exists; reconcile against the forge API only
  after confirmed creation.
- Honest rewrites **add**, not remove: an unsupported claim becomes a
  `## Pending workflow actions` section naming the pending action, not a
  silently unchecked box.

## When to commit

Commit **after** a verified gate (a green `G<n>` from
`skillgrid:test-driven-verification` — not "I think it's done"), and **before**
a long-running install/build/test command (so an interruption lands on a
checkpoint, not mid-command). Never commit broken tests or mid-edit state.

Also:

- Before spawning parallel work or crossing a review gate.
- On a PreCompact / session-end hook — commit first, so the event stream holds
  the position.

## Scratch checkpoints (turn-level)

Commits are the durable record; **scratch refs** are the safety net *between*
commits — the uncommitted state of a turn that got interrupted, wrong, or
stopped. They are not commits: they never enter history, never carry a
`[skillgrid-context]` block, and are pruned.

- **Storage:** git refs under `refs/skillgrid/` in the *actual* repo (no shadow
  git). A ref points at a `commit-tree` object whose message carries the
  metadata: `branch <name>`, `head <sha>`, `index-tree <sha>`,
  `worktree-tree <sha>`, `session <id>`, `turn <n>`, `label <prompt ≤60 chars
  + mutating calls>`, `created <iso>`, `preexisting-untracked <json>`.
  Worktrunk's merge-time sweep uses `refs/wt-backup/<branch>` — separate
  namespace, coexists, never pruned by the scratch rules.
- **Timing:** once per *completed* turn (end of the response), never per tool.
  **Dedup:** skip when the worktree tree SHA is unchanged from the last
  checkpoint (a read-only turn creates no ref).
- **Capture:** temporary index — `GIT_INDEX_FILE` in a tempdir → `read-tree
  HEAD` → `add --all` in batches of 100 (filtered set) → `write-tree` =
  worktree tree. The real index is untouched; no commit is created.
- **Filtering (never snapshot):** `.gitignore`'d paths, plus the dir-name set
  `node_modules`, `.venv`, `venv`, `env`, `.env`, `dist`, `build`,
  `.pytest_cache`, `.mypy_cache`, `.cache`, `.tox`, `__pycache__`, `.worktree`;
  untracked files >10MiB; untracked dirs with >200 files. The pre-existing
  untracked list (post-filter) is recorded **in the ref metadata** — that list
  is the never-delete set at restore time.
- **Restore:** a *before-restore* checkpoint is taken first (automatic). Then:
  `reset --hard <head>` → `read-tree --reset -u <worktree-tree>` → restore the
  real index from `index-tree` → safe-clean: delete snapshot files that are
  neither pre-existing nor skipped. **Never delete:** ignored dirs, skipped
  large files/dirs, pre-existing untracked.
- **Branch + worktree safety (atomic pair):** restore is refused when the
  current branch **or** the current worktree root differs from the recorded
  pair. This is what keeps fan-out unambiguous: 5 worktrees × 5 branches,
  refs never cross.
- **Per-worktree by construction:** each worktree has its own ref namespace;
  prune at session start touches only the *current* worktree's refs.
- **Lifecycle:** max **50 refs per session** (prune oldest first); at session
  start, prune refs from other sessions (background, non-blocking).
- **Hook wiring:** `checkpoint-state.js` subcommands `scratch` (capture) and
  `scratch-restore <id>` (before-restore snapshot + pair check + diff preview
  on stdout). Same shim discipline as the existing subcommands; the git core
  stays testable without the harness.
- **Scope boundary:** no conversation-state restore (harness session layer
  owns that), no per-tool granularity, no auto-restore — restore is always a
  decision, files-only, noted in the ledger (`(scratch restore <label>)`).

Reference implementation: `@devopstales/pi-worktree` (`scratch-core.ts`); the constants in this section are its source of truth (its contract-drift guard test enforces the direction).

~~~skillgrid-constants
REF_BASE: refs/skillgrid
AGENT_BRANCH_RE: "^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$"
IGNORED_DIRS: ["node_modules", ".venv", "venv", "env", ".env", "dist", "build", ".pytest_cache", ".mypy_cache", ".cache", ".tox", "__pycache__", ".worktree"]
MAX_UNTRACKED_FILE_SIZE: 10485760
MAX_UNTRACKED_DIR_FILES: 200
MAX_PER_SESSION: 50
~~~

## Worktree removal + branch deletion (shared safety)

Applies to `ship` Step 6 cleanup and to any worktree the session created.

- **Branch deletion — no-added-changes rule:** never delete a worktree branch
  unless it would add **no changes** to the base. Check, in order (first
  failure stops): (1) branch HEAD equals base HEAD; (2) branch is an ancestor
  of base; (3) `git diff --quiet <base>...<branch>` (three-dot) is empty.
  Unmerged branch → keep it, report it — "branch kept is the rollback."
  Force-deleting an unmerged branch on your own initiative is a blocker, not a
  choice.
- **Worktree removal — force only with a backup ref:** `git worktree remove
  --force` (or `wt remove --force`) may be used only *after* uncommitted
  changes are saved — `git stash create` → `git update-ref
  refs/skillgrid/pre-remove-<branch> <sha>` — and the recovery path is in the
  report. Without worktrunk, also check for processes with cwd under the
  worktree before removal and report them; never kill a process holding an
  interactive terminal. With worktrunk: `wt remove --reap` (reaps detached
  processes by working directory, spares TTY-attached ones).
- **Reconciliation (wave end):** a worktree with a live agent process but no
  row in the fan-out ledger is a lost dispatch — report it; never silently
  reap it. With worktrunk installed, `wt list` (🤖/💬 activity markers) is the
  visual check.

## Worktree lifecycle (forking)

- **Fork before integration:** at any point, a worktree branch can be forked
  — `git worktree add .worktree/<branch>-alt -b <branch>-alt <branch>` (or
  `wt switch --create <branch>-alt --base <branch>`) — for "try option B
  while PR feedback iterates on A," or when the user keeps a branch and
  branches off it. The fork inherits the parent's commits; the new branch
  stays on the per-agent allow-list.
- **Scratch refs do NOT cross a fork** — a ref recorded in worktree A is not
  restorable in fork B (different worktree root: the pair check refuses).
  The fork starts its own checkpoint line from a fresh resume checkpoint.
- **Integration stays single-sink:** a worktree's branch integrates to the
  base exactly once (`ship` Step 6). A fork that also integrates is a
  *second* change with its own spec folder — two forks of one branch are two
  changes, not one change done twice.

## Where the hooks live

| Layer | Path (repo → staged) | What |
|-------|----------------------|------|
| **Hook entrypoints (shims)** | `git-hooks/pre-commit.js`, `git-hooks/commit-msg.js`, `git-hooks/prepare-commit-msg`, `git-hooks/pre-push`, `git-hooks/post-merge`, `git-hooks/stop.js`, `git-hooks/gate-stop.js` → `~/.skillgrid/git-hooks/` | one-line shims that call the entrypoint |
| **Implementation** | `hooks/checkpoint-state.js` (subcommands `guard` / `guard-msg` / `prepare-msg` / `pre-push` / `post-merge-check` / `post-check`), `hooks/precommit-guard.js`, `hooks/precommit-zone-guard.js`, `hooks/precommit-ignore-guard.js`, `hooks/ticketing.js`, `hooks/ticketing-guard.js`, `hooks/ticketing-pre-push.js`, `hooks/ticketing-post-merge.js`, `hooks/gate-lint.js`, `hooks/gate-state.js`, `hooks/gate-stop.js`, `hooks/stop-tests.js` → `~/.skillgrid/hooks/` | the actual logic |

The shims resolve their implementation relative to themselves
(`../hooks/checkpoint-state.js`), so the two dirs stage together and work
identically from the repo checkout and from `~/.skillgrid/`.

## Wiring the git hooks into a repo

`skillgrid install` mirrors the whole repo tree to `~/.skillgrid/`
(remove-then-copy per top-level entry; `.git`/`node_modules` and the repo's own
`.skillgrid/` excluded, live `mnemonic/`/`repos/`/`config.d/` never touched)
and points git's global `core.hooksPath` at `~/.skillgrid/git-hooks`. The
`git-hooks/` + `hooks/` tree is **not** part of the skill tree in this repo —
it ships with the `skillgrid` CLI that performs the install; the table above is
its contract, verifiable on any installed machine at `~/.skillgrid/hooks/`.
Consumers then read from the mirror: plugins from `~/.skillgrid/plugins`, git
hooks from `~/.skillgrid/git-hooks`, implementations via copy from
`~/.skillgrid/hooks`. Re-run install after the repo updates the hook tree.

What they enforce (see `hooks/precommit-guard.js`):

These guards serve **all** skillgrid skills — the commit-time invariants the
whole family states as prose. `guard` is a dispatcher over every guard; adding
one is a one-line change in `checkpoint-state.js`.

| Hook | Guard | Enforces (stated by) | Failure |
|------|-------|----------------------|---------|
| `pre-commit` | **cwd-drift** (precommit-guard.js) | a prior `cd` left the worktree root; sentinel catches it | FATAL + RECOVERY `cd` |
| `pre-commit` | **protected-ref / detached HEAD** (precommit-guard.js) | never commit on `main`/`master`/`develop`/`trunk`/`release/*` or detached; worktree branch must be per-agent (`agent-*`, `worktree-agent-*`, `worktree-wf_*`) | FATAL + RECOVERY |
| `pre-commit` | **spec-zone XOR code-zone** (precommit-zone-guard.js) | BDD zone rule — commit `<specs_root>` before code, never both in one commit (execution skills, `blueprint` Phase 5, `qa`) | FATAL + split-into-two RECOVERY |
| `pre-commit` | **generated/scratch not tracked** (precommit-ignore-guard.js) | `acceptance-tests/.extracted/` and the worktree dir stay untracked (`blueprint` Phase 5, `execute` Step 0) | auto-adds to `.gitignore` + unstages + FATAL |
| `commit-msg` | **conventional subject + no AI-attribution trailer** | commit format (this contract) | FATAL + RECOVERY |
| `commit-msg` | **tracker footer format** (ticketing) | a `Refs:`/`Closes:`/`Fixes:`/`Resolves:` footer must carry a parseable Tracker ID (`TASK-069`, `#42`, `PROJ-123`); footers stay optional, absence passes | FATAL + RECOVERY |
| `prepare-commit-msg` | **Refs injector seam** (ticketing) | appends `Refs: <id>` from `SKILLGRID_TRACKER_REF` (exported by the agent from the active tasks.md Tracker ID) when the message has no footer yet | no-op without the env var, never blocks |
| `pre-commit` | **backlog completeness** (ticketing-guard.js) | staged `<backlog_directory>/tasks/*.md` (dir from `ticketing.backlogmd.dir`) carry frontmatter `type:`, non-empty `references:`, and DoD items (ticketing completeness gate); missing Implementation Plan is WARNING-only (the CLI creates plan-less triage tasks) | FATAL + fix-via-CLI RECOVERY |
| `pre-push` | **outgoing footer check** (ticketing-pre-push.js) | no malformed `Closes:`/`Fixes:`/`Resolves:` in the outgoing range (local `git log` only, never network) | FATAL on Closes, WARNING on advisory `Refs:` |
| `post-merge` | **duplicate-ID scan** (ticketing-post-merge.js) | `backlog doctor` (read-only) after merges; gh/glab/jira have nothing local to reconcile | WARNING, never blocks |
| `pre-commit` | **gate blocks well-formed** (gate-lint.js) | every runnable `G<n>` carries CHECK+EXPECT, ABANDON has a reason, no happy-path requirement lacks a gate (`blueprint` Phase 5, `test-driven-verification`) | WARNING (advisory, never blocks) |
| `stop` (agent hook, `--with-stop`) | **tests must pass** (stop-tests.js) | "commit after the gate is green" / "run the full suite before committing" — blocks the stop on a red `testing.runner` (execution + qa skills) | blocks stop + shows failures |
| `gate-stop` (agent hook, `--with-stop`) | **every gate met** (gate-stop.js → gate-state.js) | "no done-claim while any happy-path gate is unmet or abandoned" — fresh `--reverify`; blocks the stop on an unmet `G<n>` or a happy-path requirement with no gate; HANDOFF note when only `ABANDON` remain; 6-block loop guard (`test-driven-verification`) | blocks stop + lists outstanding gates |
| (manual) `post-check` | **unexpected deletions** in HEAD | worth a glance, not a hard block | WARNING (document in the block) |

The `stop` and `gate-stop` hooks are **agent harness hooks** (Stop events), not
git hooks. `stop` needs a test runner and `gate-stop` needs the change's
`acceptance.feature`; both stage with the rest of `git-hooks/` and are wired
by the harness's Stop-hook config. `gate-stop` reads the harness JSON on stdin
for `session_id` (its loop-guard key); its shim does not drain stdin.

Run the deletion check after committing:

```bash
node hooks/checkpoint-state.js post-check
```

**CLI wiring:** the hooks are one-line shims over `hooks/checkpoint-state.js`,
staged to `~/.skillgrid/` by `skillgrid install`, which also points
`core.hooksPath` at `~/.skillgrid/git-hooks`. The subcommand contract
(`guard` / `guard-msg` / `prepare-msg` / `pre-push` / `post-merge-check` /
`post-check`) is the stable seam.

## The commit event stream

Each work-unit commit carries its `[skillgrid-context]` block into the session
event stream as a `commit` event (parsed Task / Decisions / Remaining, keyed by
commit id). A commit with no block still records an event, with empty detail.

A fresh session resumes by reading git state plus the session's events:

```bash
git log --oneline -5
skillgrid session <session-id>
```

Resume rules (full decision tree in [state-schema.md](state-schema.md)):
- No commits yet → fresh start.
- The latest commit's `Remaining:` names a partial unit — finish it first
  (`git status` remainder first), then move to the next task.
- `Remaining:` empty + tests green → unit done; move to the next task or
  `skillgrid:ship`.
- For another session's position, read its events via `skillgrid session
  <session-id>` (or the `session_changes` MCP tool): the events return in
  sequence order with the net commit range, and `to_commit` is the resume
  position.

## Remember

- Conventional subject, no AI-attribution trailer — the `commit-msg` hook enforces it.
- Atomic + independently-revertable + ~100 lines.
- Spec commit before code commit (zone rule).
- Commit **after** a verified gate, **before** a long command.
- Every work-unit commit carries the `[skillgrid-context]` block (it feeds the
  session's `commit` events).
- Never commit on a protected ref or a detached HEAD — the `pre-commit` hook stops it.
- One-way-door decisions still get an ADR (per `_shared/adr-records/README.md`);
  the `Decisions:` line is the per-task record.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "One big commit is fine" | One logical change per commit; a feature + refactor + config change is three commits, not one. |
| "I'll add Co-Authored-By" | No `Co-Authored-By`, `Generated-By`, or any AI attribution trailer — the `commit-msg` hook rejects it. |
| "I'll squash it all into one later" | Independently-revertable means you can revert *now*; a squashed bundle loses that the moment it lands. |
| "I'll commit when I'm really done" | Commit after a verified gate, before a long-running command — interruption lands on a checkpoint, not mid-command. |
| "The spec and code go together" | Zone rule: spec commit before code commit, never both in one — `precommit-zone-guard.js` enforces it. |

## Red Flags

- A commit mixing a feature, a refactor, and a config change.
- A `Co-Authored-By` or `Generated-By` trailer in a commit message.
- `<specs_root>` and code committed in the same commit.
- Committing on `main`/`master`/`develop`/`trunk`/`release/*` or a detached HEAD.
- A broken test or mid-edit state committed as a "checkpoint".
- A work-unit commit with no `[skillgrid-context]` block.

## Verification

- [ ] `git log --oneline` shows atomic, independently-revertable commits.
- [ ] Each commit subject matches the conventional regex, with no AI attribution trailer.
- [ ] Each commit body carries the `[skillgrid-context]` block (Task / Decisions / Remaining / Tried).
- [ ] Git hooks are installed and fire on commit (`skillgrid install` ran; `pre-commit` + `commit-msg` shims present in `~/.skillgrid/git-hooks/`).
- [ ] `git log -1` shows the last work-unit commit with its `[skillgrid-context]` block intact.
