---
name: work-unit-commits
description: >
  Use when committing work-unit changes during execution — the canonical commit
  protocol: conventional-commit format, atomic + independently-revertable
  sizing, when-to-commit, the [skillgrid-context] commit block, the git-hook
  safety guards, and the .skillgrid/sdd/checkpoint.json resume handle. Fully
  active by default during simple-execution and subagent-execution.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: gstack (WIP context block + snapshot), gsd-core (commit guards), addyosmani (atomic sizing), gentleman-ai/branch-pr (conventional format), DeL-TaiseiOzaki/claude-code-orchestra (checkpointing) + orchestkit (checkpoint-resume)
---

# Work-Unit Commits

## Overview

Turn "commit after every task" into an *enforced* checkpoint: a conventional,
atomic, independently-revertable commit that carries a machine-readable
`[skillgrid-context]` block, guarded by git hooks, with a derived
`checkpoint.json` resume handle. The commit is the durable record; the JSON is
the resume pointer. They cannot drift because the JSON is derived from history.

**Announce at start:** "I'm using the skillgrid:work-unit-commits skill for commit discipline."

**Config:** Read `.skillgrid/config.yaml`. Use `conventions.commit_style`
(`conventional` by default) and `conventions.scratch_dir` (`.skillgrid/sdd`).
`checkpoint.state_file` (default `<scratch_dir>/checkpoint.json`) and
`checkpoint.enabled` (default `true`) control the resume handle.

**Fully active by default** during `skillgrid:simple-execution` and
`skillgrid:subagent-execution` — every work-unit commit follows this protocol.
Not on-demand.

## When to Use

- Before committing after a work unit — any change during
  `skillgrid:simple-execution` or `skillgrid:subagent-execution`.
- When wiring git hooks into a repo (`skillgrid install`) or resuming from a
  checkpoint handle.
- When checking that a commit is atomic, conventional, and independently
  revertable before it lands.

**When NOT to use:** for trivial throwaway commits on a scratch branch where
revertability doesn't matter.

## The two layers

| Layer | Where | What | Who writes it |
|-------|-------|------|---------------|
| **Durable record** | the commit's body | `[skillgrid-context]` block (Task / Decisions / Remaining / Tried) | you, at commit time |
| **Resume handle** | `.skillgrid/sdd/checkpoint.json` | current position, derived from `git log -1` + the block | `checkpoint-state.sh snapshot` (idempotent) |

The `[skillgrid-context]` block is the source of truth for *decisions*; git
history is the source of truth for *state*; `checkpoint.json` is a derived view
of both. Never hand-edit the JSON.

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
- **Spec before code (zone rule, BDD is always on):** commit `.skillgrid/specs/`
  changes *before* the code that satisfies them. Never leave both uncommitted.
- **Commit first, then review.** Review diffs `merge-base...HEAD`; an uncommitted
  change is invisible to it. Commit, review, then amend or add a fixup.

## When to commit

Commit **after** a verified gate (a green `G<n>` from
`skillgrid:test-driven-verification` — not "I think it's done"), and **before**
a long-running install/build/test command (so an interruption lands on a
checkpoint, not mid-command). Never commit broken tests or mid-edit state.

## Where the hooks live

| Layer | Path (repo → staged) | What |
|-------|----------------------|------|
| **Hook entrypoints (shims)** | `git-hooks/pre-commit`, `git-hooks/commit-msg`, `git-hooks/stop`, `git-hooks/gate-stop` → `~/.skillgrid/git-hooks/` | one-line shims that call the entrypoint |
| **Implementation** | `hooks/checkpoint-state.sh` (subcommands `guard` / `guard-msg` / `post-check` / `snapshot` / `restore`), `hooks/precommit-guard.sh`, `hooks/precommit-zone-guard.sh`, `hooks/precommit-ignore-guard.sh`, `hooks/gate-lint.sh`, `hooks/gate-state.sh`, `hooks/gate-stop.sh`, `hooks/stop-tests.sh` → `~/.skillgrid/hooks/` | the actual logic |

The shims resolve their implementation relative to themselves
(`../hooks/checkpoint-state.sh`), so the two dirs stage together and work
identically from the repo checkout and from `~/.skillgrid/`.

## Wiring the git hooks into a repo

`skillgrid install` mirrors the whole repo tree to `~/.skillgrid/`
(remove-then-copy per top-level entry; `.git`/`node_modules` and the repo's own
`.skillgrid/` excluded, live `mnemonic/`/`repos/`/`config.d/` never touched)
and points git's global `core.hooksPath` at `~/.skillgrid/git-hooks`.
Consumers then read from the mirror: plugins from `~/.skillgrid/plugins`, git
hooks from `~/.skillgrid/git-hooks`, implementations via copy from
`~/.skillgrid/hooks`. Re-run install after the repo updates the hook tree.

What they enforce (see `hooks/precommit-guard.sh`):

These guards serve **all** skillgrid skills — the commit-time invariants the
whole family states as prose. `guard` is a dispatcher over every guard; adding
one is a one-line change in `checkpoint-state.sh`.

| Hook | Guard | Enforces (stated by) | Failure |
|------|-------|----------------------|---------|
| `pre-commit` | **cwd-drift** (precommit-guard.sh) | a prior `cd` left the worktree root; sentinel catches it | FATAL + RECOVERY `cd` |
| `pre-commit` | **protected-ref / detached HEAD** (precommit-guard.sh) | never commit on `main`/`master`/`develop`/`trunk`/`release/*` or detached; worktree branch must be per-agent (`agent-*`, `worktree-agent-*`, `worktree-wf_*`) | FATAL + RECOVERY |
| `pre-commit` | **spec-zone XOR code-zone** (precommit-zone-guard.sh) | BDD zone rule — commit `.skillgrid/specs/` before code, never both in one commit (simple-execution, subagent-execution, acceptance-test-authoring, qa, writing-blueprints) | FATAL + split-into-two RECOVERY |
| `pre-commit` | **generated/scratch not tracked** (precommit-ignore-guard.sh) | `acceptance-tests/.extracted/` and the worktree dir stay untracked (acceptance-test-authoring, isolated-workspace) | auto-adds to `.gitignore` + unstages + FATAL |
| `commit-msg` | **conventional subject + no AI-attribution trailer** | commit format (work-unit-commits, branch-pr convention) | FATAL + RECOVERY |
| `pre-commit` | **gate blocks well-formed** (gate-lint.sh) | every runnable `G<n>` carries CHECK+EXPECT, ABANDON has a reason, no happy-path requirement lacks a gate (acceptance-test-authoring, test-driven-verification) | WARNING (advisory, never blocks) |
| `stop` (agent hook, `--with-stop`) | **tests must pass** (stop-tests.sh) | "commit after the gate is green" / "run the full suite before committing" — blocks the stop on a red `testing.runner` (execution + qa skills) | blocks stop + shows failures |
| `gate-stop` (agent hook, `--with-stop`) | **every gate met** (gate-stop.sh → gate-state.sh) | "no done-claim while any happy-path gate is unmet or abandoned" — fresh `--reverify`; blocks the stop on an unmet `G<n>` or a happy-path requirement with no gate; HANDOFF note when only `ABANDON` remain; 6-block loop guard (test-driven-verification) | blocks stop + lists outstanding gates |
| (manual) `post-check` | **unexpected deletions** in HEAD | worth a glance, not a hard block | WARNING (document in the block) |

The `stop` and `gate-stop` hooks are **agent harness hooks** (Stop events), not
git hooks. `stop` needs a test runner and `gate-stop` needs the change's
`acceptance.feature`; both stage with the rest of `git-hooks/` and are wired
by the harness's Stop-hook config. `gate-stop` reads the harness JSON on stdin
for `session_id` (its loop-guard key); its shim does not drain stdin.

Run the deletion check after committing:

```bash
bash hooks/checkpoint-state.sh post-check
```

**CLI wiring:** the hooks are one-line shims over `hooks/checkpoint-state.sh`,
staged to `~/.skillgrid/` by `skillgrid install`, which also points
`core.hooksPath` at `~/.skillgrid/git-hooks`. The subcommand contract
(`guard` / `guard-msg` / `post-check`) is the stable seam.

## The checkpoint resume handle

After each work-unit commit, snapshot (idempotent, cheap):

```bash
bash hooks/checkpoint-state.sh snapshot
```

A fresh session resumes by reading the handle + live git state:

```bash
bash hooks/checkpoint-state.sh restore
```

Resume rules (full decision tree in `references/state-schema.md`):
- No file → fresh start.
- File's `branch` == current branch → "Resuming from `<task>` — last commit
  `<short>` `<subject>`."; finish `git status` remainder first.
- File's `branch` differs → ask which branch to work on; don't guess.
- `remaining` empty + tests green → unit done; move to the next task or
  `skillgrid:ship`.

A human-readable view can be written from `templates/checkpoint.md` when a
handoff is wanted, but the JSON is the source of truth.

## The Handoff Hub (change snapshots + named checkpoints)

The `checkpoint.json` resume handle is the *per-change* pointer. The **Handoff
Hub** (change 015) is the *engine-backed* change log that makes the whole
history queryable from the admin UI and MCP. It has two layers on top of git:

| Layer | Where | What | Who writes it |
|-------|-------|------|---------------|
| **Change snapshots** | `change_snapshots` (SQLite) | a git-derived, append-only log of EVERY commit + its parsed `[skillgrid-context]` block | `checkpoint-state.sh snapshot` (best-effort `skillgrid handoff record`) or `skillgrid handoff backfill` (first run, last 100 commits) |
| **Named checkpoints** | `checkpoints` (SQLite) | intentional markers placed *before* risky actions, with drift verification | `skillgrid handoff checkpoint <name>` (you, before-apply / before-pause) |

The change snapshots are reconstructable by replaying `git log` — they are the
index, the commit is the durable record (same relationship as
`checkpoint.json`). They can never be the sole source of truth.

**Record a named checkpoint before a risky action** (before-apply-<change>,
before a long pause, before validation):

```bash
skillgrid handoff checkpoint before-apply-<change> --evidence "lint ok, tests 34/34"
```

**Drift-verify a checkpoint before resuming** (no reverts — it only reports):

```bash
skillgrid handoff verify before-apply-<change>
# -> continue | inspect-drift | refresh
```

The hook wiring is best-effort: `checkpoint-state.sh snapshot` already calls
`skillgrid handoff record` after writing `checkpoint.json`, so the hub stays
current with the resume handle. A missing `skillgrid` binary never blocks the
commit. See `resume` for the verify-first resume flow and `ship` for archiving
the change's checkpoints.

## Remember

- Conventional subject, no AI-attribution trailer — the `commit-msg` hook enforces it.
- Atomic + independently-revertable + ~100 lines.
- Spec commit before code commit (zone rule).
- Commit **after** a verified gate, **before** a long command.
- Every work-unit commit carries the `[skillgrid-context]` block.
- `snapshot` after each commit; `restore` to resume; never hand-edit the JSON.
- The snapshot also feeds the Handoff Hub change log (best-effort `skillgrid handoff record`).
- Place a named checkpoint (`skillgrid handoff checkpoint`) before a risky action; verify it (`skillgrid handoff verify`) before resuming.
- Never commit on a protected ref or a detached HEAD — the `pre-commit` hook stops it.
- One-way-door decisions still get an ADR; the `Decisions:` line is the per-task record.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "One big commit is fine" | One logical change per commit; a feature + refactor + config change is three commits, not one. |
| "I'll add Co-Authored-By" | No `Co-Authored-By`, `Generated-By`, or any AI attribution trailer — the `commit-msg` hook rejects it. |
| "I'll squash it all into one later" | Independently-revertable means you can revert *now*; a squashed bundle loses that the moment it lands. |
| "I'll commit when I'm really done" | Commit after a verified gate, before a long-running command — interruption lands on a checkpoint, not mid-command. |
| "The spec and code go together" | Zone rule: spec commit before code commit, never both in one — `precommit-zone-guard.sh` enforces it. |

## Red Flags

- A commit mixing a feature, a refactor, and a config change.
- A `Co-Authored-By` or `Generated-By` trailer in a commit message.
- `.skillgrid/specs/` and code committed in the same commit.
- Committing on `main`/`master`/`develop`/`trunk`/`release/*` or a detached HEAD.
- A broken test or mid-edit state committed as a "checkpoint".
- `checkpoint.json` hand-edited, or missing after a work-unit commit.

## Verification

- [ ] `git log --oneline` shows atomic, independently-revertable commits.
- [ ] Each commit subject matches the conventional regex, with no AI attribution trailer.
- [ ] Each commit body carries the `[skillgrid-context]` block (Task / Decisions / Remaining / Tried).
- [ ] Git hooks are installed and fire on commit (`skillgrid install` ran; `pre-commit` + `commit-msg` shims present in `~/.skillgrid/git-hooks/`).
- [ ] `checkpoint.json` exists and was derived by `checkpoint-state.sh snapshot` after the last commit.
