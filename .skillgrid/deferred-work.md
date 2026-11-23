# Deferred Work

> Findings deferred (not fixed) by a QA/review round, tracked here until dispositioned. IDs are `DW-<n>` (monotonic, project-wide). Each entry names the source round, the finding, why it's deferred, and the re-entry path.
>
> **Convention:** the *current change's* deferred work is recorded in `.skillgrid/change-specs/<id>/` (or `change-archive/<id>/`); entries here are cross-change or out-of-scope-for-the-current-change work that outlives that folder.

## Deferred entries

### DW-1 — real index not byte-identical after capture when staged (pi-worktree, R1)
- **Source:** review round 6 (2026-11-23), finding R6-B3.
- **What:** `git write-tree` on the real index (`src/core/scratch-core.ts:209`) persists the cache-tree extension when the index has staged changes, so the index *file* bytes differ (the tree SHA does not). Violates the R1 literal "byte-identical"; benign (git rewrites the cache-tree transparently).
- **Why deferred:** the common (unstaged) case is byte-identical; the staged case is a contract-literal nuance with no end-user consequence.
- **Re-entry:** capture the index tree from a temp index (`GIT_INDEX_FILE`), or compare tree SHA not file bytes, or document the staged-case caveat. Add a held-out test: stage a file, capture, assert index-file sha256 unchanged.

### DW-2 — safeClean staged-new file → AD state (pi-worktree, R3)
- **Source:** round 6, finding R6-C.
- **What:** `safeClean` set 1 (`git diff --name-only --diff-filter=AD HEAD <snapshot>`) includes a file that was staged-but-uncommitted at capture; the `read-tree --reset` re-stages it without restoring the worktree copy → `AD` after restore (`src/core/scratch-core.ts:337-361`).
- **Why deferred:** edge case (a file created *and* staged in a turn, then the tree restored).
- **Re-entry:** exclude staged-new (`A`) from the delete set, or restore the worktree copy after `read-tree`. Held-out test: capture a staged-new file → restore → worktree file present, no `AD`.

### DW-3 — recovery can land in a foreign recreated repo (pi-worktree, R6)
- **Source:** round 6, finding R6-F.
- **What:** `recover()` filters candidates by *recorded* identity; the survivor is validated by `stat` + `isExistingGitDir` but its **live** git-common-dir is never re-compared to `cwdIdentity` → if the removed dir is recreated as an unrelated repo before the next `worktrunk` call, the session recovers into it (`src/core/wt-core.ts:457-491`).
- **Why deferred:** edge case (dir recreated as a foreign repo in the window between cwd-removal and the next call).
- **Re-entry:** re-run `git rev-parse --git-common-dir` on the chosen survivor and compare to `cwdIdentity`; reject on mismatch. Held-out test: remove worktree, recreate dir as an unrelated repo, recover → no move into the foreign repo.

### DW-4 — directive outside-repo gate doesn't call validateDirective (pi-worktree, R5)
- **Source:** round 6, finding R6-G2.
- **What:** the "directive naming a path outside the repo is rejected" test asserts the `worktrunk` command exists, not that `validateDirective` runs and the session stays put.
- **Re-entry:** unit-test `validateDirective` directly (a `/tmp/...` path → error, cwd unchanged) and assert the tool's stay-put on a failed `wt list`.

### DW-5 — no-replay asserted on a constant (pi-worktree, R6)
- **Source:** round 6, finding R6-G5.
- **What:** `replayed` is always `false` (no replay implemented); the no-replay gate is asserted on that constant, and e2e step 5 greps the tool's self-report (not an independent proof the pending command was dropped).
- **Re-entry:** an independent assertion (e.g., a marker file the pending command would create, which must be *absent* after recovery).

### DW-6 — "51st prunes oldest" asserts count not identity/wiring (pi-worktree, R4)
- **Source:** round 6, finding R6-G6.
- **What:** the test asserts the ref count returns to 50, not that the *oldest* was the one pruned, nor that `createCheckpoint` → `pruneOldCheckpoints` is wired in the extension (it's not; pruning runs from `session_start`).
- **Re-entry:** assert the specific oldest ref is gone and the rest retained; or document that per-capture pruning is a design choice.

### DW-7 — e2e step-3 move assertion is a false-positive window (pi-worktree, R5)
- **Source:** round 6, finding R6-G7.
- **What:** e2e step 3 greps the output for the target path to prove the worktrunk *moved* — but wt's own `switch --create` stdout also contains the path (false-positive window).
- **Re-entry:** assert the session's cwd actually changed (e.g., a follow-up `git rev-parse --show-toplevel` from the tool) rather than grepping a path that appears in both.

### DW-8 — runDirectedWt swallows the spawn err (pi-worktree, robustness)
- **Source:** round 6, finding R6-J1.
- **What:** the `.catch` sets `code:-1` but drops the `err` (wt missing on PATH → empty `stderr`; inconsistent with `git()`/`wt()` which surface the error).
- **Re-entry:** surface `err` in the caught output so "wt not found" is diagnosable.

### DW-9 — git() inherits host GIT_* env (pi-worktree, robustness)
- **Source:** round 6, finding R6-J2.
- **What:** `git()` doesn't scrub inherited `GIT_*` env vars (only `wt()` does); a host `GIT_DIR`/`GIT_WORK_TREE` can steer the git calls.
- **Re-entry:** scrub `GIT_*` in `git()` too (or document why git is exempt).

### DW-10 — timeoutMs never set by the extension (pi-worktree, robustness)
- **Source:** round 6, finding R6-J3.
- **What:** `WtOptions.timeoutMs` defaults to 0 (no timeout) and the extension never sets it → a hung `wt` holds the module mutex.
- **Re-entry:** set a bounded `timeoutMs` in the extension's `runWt` call.

### DW-11 — loadCheckpoint doesn't validate 40-hex head (pi-worktree, robustness)
- **Source:** round 6, finding R6-J4.
- **What:** `loadCheckpoint` validates the path/branch but not that `head` is 40-hex; a corrupt JSON `head` (e.g. `""`) could flow to a `git diff HEAD <"">`.
- **Re-entry:** reject a checkpoint whose `head` isn't 40-hex (or null).

### DW-12 — ui.ts seeding comment (pi-worktree, cosmetic)
- **Source:** round 6, finding R6-K1.
- **What:** the comment claims implicit `seedCache` (now true after the D fix — re-verify the wording is accurate).
- **Re-entry:** align the comment with the post-D behavior.

### DW-13 — findCheckpoint hardcodes "skillgrid/" (pi-worktree, cosmetic)
- **Source:** round 6, finding R6-K2.
- **What:** `findCheckpoint` uses `refs/skillgrid/` as the literal, not the `REF_BASE` constant.
- **Re-entry:** derive it from `REF_BASE`.

### DW-14 — metadata keys camelCase vs kebab (pi-worktree, cosmetic)
- **Source:** round 6, finding R6-K4.
- **What:** `largeFiles`/`largeDirs` are camelCase while the sibling keys (`oversized-untracked`, `truncated`, `session-id`) are kebab-case.
- **Re-entry:** pick one convention (the contract's kebab) and align.

### DW-15 — minor (test name, comment numbering, footer flash) (pi-worktree, cosmetic)
- **Source:** round 6, findings R6-K6/K7/K8.
- **What:** the restore test name says "refuses" but asserts success; the safeClean comment says "three sets" but lists two; `setFooter` flashes the default text before the message.
- **Re-entry:** rename the test to the behavior it asserts; correct the comment; drop the pre-flash.

## Resolved at the `qa-verdict` gate (2026-11-23)

- **A (R7/R8/R9):** the programmer chose **wire** → applied in `f1b58b8` (`applyMergeDefaults` + `checkBranchAllowList` + `observeWorktreeRoot` wired into `runWorktrunk` via `isWorktreeContext`). No longer open.
- **E (R10):** the programmer chose **fix** → applied in `f1b58b8` (`probeUnapprovedHooks` detection + `storeApprovalsPty` PTY store + core `handleWtApproval`; the 2 approval gates now match). No longer open.
- **bench (R12):** the programmer chose **accept** given the 65–117ms idle carry (the 161/238ms runs are shared-load artifacts). Not a code defect; no DW entry.
