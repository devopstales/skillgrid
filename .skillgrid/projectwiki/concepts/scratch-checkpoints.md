---
type: Concept
title: "Scratch checkpoints"
---

# Scratch checkpoints

Turn-level safety net between commits. Contract source:
`.agents/skills/_shared/work-unit-commits/README.md` § Scratch
checkpoints (the plugin's constants must equal these — R13 drift guard).

**Storage:** refs under `refs/skillgrid/` in the actual repo. A ref points
at a `commit-tree` object; the message carries metadata: `branch`, `head`,
`index-tree`, `worktree-tree`, `session`, `turn`, `label` (prompt ≤60 +
mutating calls), `created`, `preexisting-untracked` (json, the
never-delete set). Coexists with `refs/wt-backup/` (worktrunk's, never
pruned by scratch rules).

**Timing + dedup:** once per completed turn (`turn_end`), never per tool.
Skip when the worktree tree SHA is unchanged from the last checkpoint — a
read-only turn creates no ref.

**Capture:** temp index (`GIT_INDEX_FILE` in tempdir → `read-tree HEAD` →
`add --all` in batches of 100 over the filtered set → `write-tree`). Real
index untouched; no commit created.

**Filtering (never snapshot):** gitignored paths; dir-name set
`node_modules`, `.venv`, `venv`, `env`, `.env`, `dist`, `build`,
`.pytest_cache`, `.mypy_cache`, `.cache`, `.tox`, `__pycache__`,
`.worktree`; untracked files >10MiB; untracked dirs with >200 files.
Pre-existing untracked (post-filter) is recorded in the ref metadata.

**Restore:** automatic before-restore checkpoint first; refuse when the
current branch **or** worktree root differs from the recorded atomic pair;
then `reset --hard <head>` → `read-tree --reset -u <worktree-tree>` →
restore real index from `index-tree` → safe-clean (delete snapshot files
that are neither pre-existing nor skipped). Never delete: ignored dirs,
skipped large files/dirs, pre-existing untracked.

**Lifecycle:** max 50 refs per session (prune oldest first); at session
start, prune other sessions' refs (background, non-blocking). Per-worktree
by construction — each worktree has its own ref namespace.

**Scope boundary:** no conversation-state restore (harness owns that), no
per-tool granularity, no auto-restore — restore is always a decision,
files-only, noted in the ledger (`(scratch restore <label>)`).
