---
name: work-unit-commits
description: >
  Use when committing work-unit changes during execution — the canonical commit
  protocol: conventional-commit format, atomic + independently-revertable
  sizing, when-to-commit, the [skillgrid-context] commit block, the git-hook
  safety guards, and the .skillgrid/sdd/checkpoint.json resume handle. Fully
  active by default during simple-execution and subagent-execution.
  # based on gstack (WIP context block + snapshot), gsd-core (commit guards),
  # addyosmani (atomic sizing), gentleman-ai/branch-pr (conventional format),
  # DeL-TaiseiOzaki/claude-code-orchestra (checkpointing) + orchestkit
  # (checkpoint-resume)
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

| Layer | Path | What |
|-------|------|------|
| **Hook entrypoints (shims)** | `.agents/git-hooks/pre-commit`, `.agents/git-hooks/commit-msg`, `.agents/git-hooks/stop`, `.agents/git-hooks/gate-stop` | one-line shims that call the entrypoint |
| **Implementation** | `.agents/hooks/checkpoint-state.sh` (subcommands `guard` / `guard-msg` / `post-check` / `snapshot` / `restore`), `.agents/hooks/precommit-guard.sh`, `.agents/hooks/precommit-zone-guard.sh`, `.agents/hooks/precommit-ignore-guard.sh`, `.agents/hooks/gate-lint.sh`, `.agents/hooks/gate-state.sh`, `.agents/hooks/gate-stop.sh`, `.agents/hooks/stop-tests.sh` | the actual logic |

The shims resolve `checkpoint-state.sh` relative to themselves
(`.agents/git-hooks/../hooks/checkpoint-state.sh`), so the two dirs stay
together and move as a unit.

## Wiring the git hooks into a repo

Install once per repo (idempotent) via the installer:

```bash
bash scripts/install-hooks.sh
```

It writes per-repo shims into the repo's *active* hooks dir (honoring an
existing `core.hooksPath`, e.g. a global `~/.aiskillgrid/git-hooks`) and points
them at the absolute path of `checkpoint-state.sh`. Re-run it after the skill
moves. A future `skillgrid-cli` replaces this per-repo copy with a single
`core.hooksPath` pointing at `.agents/git-hooks`.

What they enforce (see `.agents/hooks/precommit-guard.sh`):

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
`acceptance.feature`, so `install-hooks.sh --with-stop` installs both separately.
`gate-stop` reads the harness JSON on stdin for `session_id` (its loop-guard key);
its shim does not drain stdin.

Run the deletion check after committing:

```bash
bash .agents/hooks/checkpoint-state.sh post-check
```

**CLI-ready:** the hooks are one-line shims over
`.agents/hooks/checkpoint-state.sh`. `skillgrid-cli` later rebinds those shims
(or points `core.hooksPath` at `.agents/git-hooks`) — the subcommand contract
(`guard` / `guard-msg` / `post-check`) is the stable seam, so no repo change is
needed when the CLI lands.

## The checkpoint resume handle

After each work-unit commit, snapshot (idempotent, cheap):

```bash
bash .agents/hooks/checkpoint-state.sh snapshot
```

A fresh session resumes by reading the handle + live git state:

```bash
bash .agents/hooks/checkpoint-state.sh restore
```

Resume rules (full decision tree in `references/state-schema.md`):
- No file → fresh start.
- File's `branch` == current branch → "Resuming from `<task>` — last commit
  `<short>` `<subject>`."; finish `git status` remainder first.
- File's `branch` differs → ask which branch to work on; don't guess.
- `remaining` empty + tests green → unit done; move to the next task or
  `skillgrid:finishing-a-development-branch`.

A human-readable view can be written from `templates/checkpoint.md` when a
handoff is wanted, but the JSON is the source of truth.

## Remember

- Conventional subject, no AI-attribution trailer — the `commit-msg` hook enforces it.
- Atomic + independently-revertable + ~100 lines.
- Spec commit before code commit (zone rule).
- Commit **after** a verified gate, **before** a long command.
- Every work-unit commit carries the `[skillgrid-context]` block.
- `snapshot` after each commit; `restore` to resume; never hand-edit the JSON.
- Never commit on a protected ref or a detached HEAD — the `pre-commit` hook stops it.
- One-way-door decisions still get an ADR; the `Decisions:` line is the per-task record.
