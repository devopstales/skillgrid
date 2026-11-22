---
type: Concept
title: "Worktree standard"
---

# Worktree standard

Contract source: `.agents/skills/_shared/work-unit-commits/README.md`.
The `pi-worktree` plugin implements the tier-1 slot of this standard.

**Worktree root:** `<projectroot>/.worktree` (singular). Must be
git-ignored before creating (`git check-ignore -q <dir>`).

**Tooling tiers** (execute Step 0, ship Step 6 follow this order):
1. Harness-native tool wins — it owns session state.
2. Worktrunk (`wt`) — preferred when present (`command -v wt` +
   `wt list` succeeds). User config:
   `worktree-path = "{{ repo_path }}/.worktree/{{ branch | sanitize }}"`
   (branch `/` → `-`). Fan-out branches: `worktree-wf_*` (domains),
   `agent-*` (single implementers). Project hooks in committed
   `.config/wt.toml` (`[pre-start]` setup, `[pre-merge]` test).
3. `git worktree add` — fallback.

**Branch allow-list** (pre-commit guard, FATAL on violation): worktree
branches must match `agent-*`, `worktree-agent-*`, or `worktree-wf_*`;
never commit on `main`/`master`/`develop`/`trunk`/`release/*` or
detached HEAD. The plugin's creation-time check (R8) uses the same regex
`^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$`.

**Merge default:** worktrunk's own default squashes; skillgrid wants
work-unit commits atomic + independently revertable, so the plugin's
merge invocation prepends `--no-squash --no-commit` unless the caller
explicitly passes either flag (R9 — the one place the plugin rewrites a
call).

**Removal + branch deletion (shared safety):** branch deletion only under
the no-added-changes rule (HEAD == base, or ancestor of base, or
three-dot diff empty); force-remove only after `git stash create` →
`git update-ref refs/skillgrid/pre-remove-<branch> <sha>`;
`wt remove --reap` reaps detached processes by cwd, spares TTY-attached.
Reconciliation: live process + no fan-out-ledger row = lost dispatch —
report, never silently reap; `wt list` (🤖/💬 markers) is the visual
check.

**Lifecycle (forking):** fork before integration
(`wt switch --create <branch>-alt --base <branch>`); scratch refs do NOT
cross a fork (different worktree root → pair check refuses); integration
stays single-sink — a second fork that integrates is a second change.
