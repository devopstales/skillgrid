---
name: work-unit-commits
description: >
  Use when committing work-unit changes during execution — the canonical commit
   protocol: conventional-commit format, atomic + independently-revertable
   sizing, when-to-commit, the [skillgrid-context] commit block, and the git-hook
   safety guards. Fully
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
`[skillgrid-context]` block, guarded by git hooks. The commit is the durable
record; its parsed block feeds a `commit` event on the session event stream,
so a fresh session resumes from git log plus the session's events.

**Announce at start:** "I'm using the skillgrid:work-unit-commits skill for commit discipline."

**Config:** Read `.skillgrid/config.yaml`. Use `conventions.commit_style`
(`conventional` by default) and `conventions.scratch_dir` (`.skillgrid/sdd`).

**Fully active by default** during `skillgrid:simple-execution` and
`skillgrid:subagent-execution` — every work-unit commit follows this protocol.
Not on-demand.

## When to Use

- Before committing after a work unit — any change during
  `skillgrid:simple-execution` or `skillgrid:subagent-execution`.
- When wiring git hooks into a repo (`skillgrid install`) or resuming from the
  session event stream.
- When checking that a commit is atomic, conventional, and independently
  revertable before it lands.

**When NOT to use:** for trivial throwaway commits on a scratch branch where
revertability doesn't matter.

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
| **Hook entrypoints (shims)** | `git-hooks/pre-commit.js`, `git-hooks/commit-msg.js`, `git-hooks/stop.js`, `git-hooks/gate-stop.js` → `~/.skillgrid/git-hooks/` | one-line shims that call the entrypoint |
| **Implementation** | `hooks/checkpoint-state.js` (subcommands `guard` / `guard-msg` / `post-check`), `hooks/precommit-guard.js`, `hooks/precommit-zone-guard.js`, `hooks/precommit-ignore-guard.js`, `hooks/gate-lint.js`, `hooks/gate-state.js`, `hooks/gate-stop.js`, `hooks/stop-tests.js` → `~/.skillgrid/hooks/` | the actual logic |

The shims resolve their implementation relative to themselves
(`../hooks/checkpoint-state.js`), so the two dirs stage together and work
identically from the repo checkout and from `~/.skillgrid/`.

## Wiring the git hooks into a repo

`skillgrid install` mirrors the whole repo tree to `~/.skillgrid/`
(remove-then-copy per top-level entry; `.git`/`node_modules` and the repo's own
`.skillgrid/` excluded, live `mnemonic/`/`repos/`/`config.d/` never touched)
and points git's global `core.hooksPath` at `~/.skillgrid/git-hooks`.
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
| `pre-commit` | **spec-zone XOR code-zone** (precommit-zone-guard.js) | BDD zone rule — commit `.skillgrid/specs/` before code, never both in one commit (simple-execution, subagent-execution, acceptance-test-authoring, qa, writing-blueprints) | FATAL + split-into-two RECOVERY |
| `pre-commit` | **generated/scratch not tracked** (precommit-ignore-guard.js) | `acceptance-tests/.extracted/` and the worktree dir stay untracked (acceptance-test-authoring, isolated-workspace) | auto-adds to `.gitignore` + unstages + FATAL |
| `commit-msg` | **conventional subject + no AI-attribution trailer** | commit format (work-unit-commits, branch-pr convention) | FATAL + RECOVERY |
| `pre-commit` | **gate blocks well-formed** (gate-lint.js) | every runnable `G<n>` carries CHECK+EXPECT, ABANDON has a reason, no happy-path requirement lacks a gate (acceptance-test-authoring, test-driven-verification) | WARNING (advisory, never blocks) |
| `stop` (agent hook, `--with-stop`) | **tests must pass** (stop-tests.js) | "commit after the gate is green" / "run the full suite before committing" — blocks the stop on a red `testing.runner` (execution + qa skills) | blocks stop + shows failures |
| `gate-stop` (agent hook, `--with-stop`) | **every gate met** (gate-stop.js → gate-state.js) | "no done-claim while any happy-path gate is unmet or abandoned" — fresh `--reverify`; blocks the stop on an unmet `G<n>` or a happy-path requirement with no gate; HANDOFF note when only `ABANDON` remain; 6-block loop guard (test-driven-verification) | blocks stop + lists outstanding gates |
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
(`guard` / `guard-msg` / `post-check`) is the stable seam.

## The commit event stream

Each work-unit commit carries its `[skillgrid-context]` block into the session
event stream as a `commit` event (parsed Task / Decisions / Remaining, keyed by
commit id). A commit with no block still records an event, with empty detail.

A fresh session resumes by reading git state plus the session's events:

```bash
git log --oneline -5
skillgrid session <session-id>
```

Resume rules (full decision tree in `references/state-schema.md`):
- No commits yet → fresh start.
- The latest commit's `Remaining:` names a partial unit — finish it first
  (`git status` remainder first), then move to the next task.
- `Remaining:` empty + tests green → unit done; move to the next task or
  `skillgrid:ship`.
- For another session's position, read its events via `skillgrid session
  <session-id>` (or the `session_changes` MCP tool): the events return in
  sequence order with the net commit range, and `to_commit` is the resume
  position.

## Commit events (no Hub, no markers)

Every commit is queryable per session from its `commit` event — there is no
separate change log to maintain and no named markers to place or verify.
Before a risky action, commit the current unit so the event stream holds the
position; to resume, read the events. See `skillgrid:resume` for the
events-first resume flow.

## Remember

- Conventional subject, no AI-attribution trailer — the `commit-msg` hook enforces it.
- Atomic + independently-revertable + ~100 lines.
- Spec commit before code commit (zone rule).
- Commit **after** a verified gate, **before** a long command.
- Every work-unit commit carries the `[skillgrid-context]` block (it feeds the
  session's `commit` events).
- Never commit on a protected ref or a detached HEAD — the `pre-commit` hook stops it.
- One-way-door decisions still get an ADR; the `Decisions:` line is the per-task record.

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
- `.skillgrid/specs/` and code committed in the same commit.
- Committing on `main`/`master`/`develop`/`trunk`/`release/*` or a detached HEAD.
- A broken test or mid-edit state committed as a "checkpoint".
- A work-unit commit with no `[skillgrid-context]` block.

## Verification

- [ ] `git log --oneline` shows atomic, independently-revertable commits.
- [ ] Each commit subject matches the conventional regex, with no AI attribution trailer.
- [ ] Each commit body carries the `[skillgrid-context]` block (Task / Decisions / Remaining / Tried).
- [ ] Git hooks are installed and fire on commit (`skillgrid install` ran; `pre-commit` + `commit-msg` shims present in `~/.skillgrid/git-hooks/`).
- [ ] `git log -1` shows the last work-unit commit with its `[skillgrid-context]` block intact.
