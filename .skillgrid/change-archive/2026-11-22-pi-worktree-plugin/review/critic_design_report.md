# Critic Review (Stage 7) — blueprint.md, @devopstales/pi-worktree

Isolation attestation: fresh context; read-only (wrote this report file only); no project instructions beyond the persona brief, target, and the explicit reading list (findings.md, briefing.md, work-unit-commits/README.md); no network/MCP. Harness: omp task subagent, parts 1 & 4 by construction, part 2 by discipline, part 3 by prompt scope (R37).

The blueprint is self-contained, carries every required header, and maps R1–R13 onto tasks with falsifiable checks. Two design-level findings below.

## Blockers

None.

## Majors

### M-1: Cross-session prune deletes other sessions' checkpoint refs before those sessions can use them (shared-repo fan-out)

**Affected Scope:** Task 4 (`src/core/scratch-core.ts` — `pruneOtherSessions`), Task 5 (`src/scratch.ts` `session_start` handler), Validation row R4; contract `_shared/work-unit-commits/README.md` § Scratch checkpoints.

**Finding:** The contract mandates "at session start, prune refs from other sessions (background, non-blocking)" and the blueprint implements exactly that: `session_start` fires `void pruneOtherSessions(repoRoot(ctx.cwd), sessionId)`, which deletes every `refs/skillgrid/<id>` whose `sessionId` ≠ the starting session. But the same contract's fan-out scenario — "5 worktrees × 5 branches", one running Pi session per worktree, all sharing one repo's shared `refs/skillgrid/` namespace (the blueprint's own Key links admit "the ref itself lives in the shared `refs/skillgrid/` namespace") — means the "other sessions" whose refs get deleted are *concurrently running sessions in sibling worktrees*, not dead ones. When worktree B's session starts (or restarts after `/resume`), it silently deletes worktree A's still-live session's last-interruption checkpoint — exactly the ref the plugin exists to preserve. There is no liveness check (no comparison of the ref's `session`/`created` metadata against anything Pi exposes as "this session is alive"), and the contract's "Per-worktree by construction: each worktree has its own ref namespace" is false for refs (shared) even though it is true for worktrees — the blueprint inherits the contract sentence without noting the correction. The retired worktree-root half of the restore check (Stage 5 R1-F001, settled) was the mechanism that made a foreign-worktree ref *unrestorable*; with it gone, a ref deleted from under a live session is simply lost, unrecoverable, and undetectable. The plan's own validation only falsifies "a current-session ref pruned" and "51+ surviving" — nothing catches *another live session's* ref being pruned, because no test runs two live sessions against one repo.

**Recommendation:** Make prune liveness-aware. Either (a) prune only refs whose `created` timestamp is older than a threshold well beyond any live session's expected restart gap, or (b) skip pruning refs for sessions Pi reports as live/resumable (the session files Pi keeps are the liveness signal), or (c) at minimum demote cross-session prune to a background task that lists candidates and reports ("pruning N refs, sessions: …") instead of deleting blind — and add a two-session test to the prune group (start session B in the same temp repo, assert session A's ref survives while an explicitly dead session C's ref does not). If the owner rules that sibling-worktree sessions sharing a repo are out of scope for v1, state that scope boundary explicitly in Global Constraints instead of leaving the contract's fan-out sentence standing as the design.

## Nits

### N-1: Dedup `lastTree` is session-keyed only, so a resumed session's first turn re-checkpoints the unchanged tree

**Affected Scope:** Task 2 (`createCheckpoint` / module-level `Map<sessionId, worktreeTreeSha>`), Task 4 (`resetSessionState`), Task 5 (`session_start` → `rebuildCache`).

**Finding:** The dedup map is keyed by `sessionId` in-process, and `session_start` calls `resetSessionState()` then only `rebuildCache(ctx)` for the UI map. On resume (the exact recovery flow the plugin targets), the first `turn_end` after session start will find no entry for the session and will emit a new ref even though the worktree tree is identical to the last pre-interruption capture — violating the "no new ref for an unchanged worktree" invariant in the one case the feature exists for. This is bounded by `prune` (one extra ref per resume) and the contract's own "resume checkpoint" concept blurs the intent, so it is a nit — but `rebuildCache` already walks `refs/skillgrid/` at session start; seeding `lastTree` for the current session from that walk is one line and removes the edge.

**Recommendation:** In `rebuildCache`, when the session has existing refs, seed `lastTree.set(sessionId, <newest ref's worktreeTreeSha>)`. Add the case to the dedup group: capture, simulate session restart (reset state), capture again on unchanged tree → `skipped`.

### N-2: Threat-matrix "Git repository selection" row tests and names a check the task list never builds

**Affected Scope:** Threat Matrix row 2; Task 3 (`restoreCheckpoint`); Validation row R3.

**Finding:** The row commits the plan to "the pair check compares `--show-toplevel` outputs (realpath'd), never user-typed paths" and to a test "pair check from a subdirectory of the worktree (same toplevel → pass); from the primary checkout (different toplevel → refuse)". But after Stage 5 R1-F001 retired the worktree-root half, no task defines a pair-check function: Task 3's `restoreCheckpoint` checks `curBranch !== cp.branch` only, and the Key links bullet restates the branch-only rule. What actually handles the subdirectory case is `repoRoot(ctx.cwd)` (an untested, undefined helper) resolving cwd → toplevel, and what handles the primary-checkout case is `root` being passed per-worktree from the caller. The row as written specifies a mechanism (realpath'd toplevel comparison in the core) that no task produces, so an executor following the tasks literally will ship a matrix row with no matching code and no matching test — or invent one the blueprint didn't ask for.

**Recommendation:** Rewrite the row to match the surviving design: execution boundary = "every git call uses `{cwd: root}` where `root` is the caller-resolved toplevel; `loadCheckpoint` validates the ref id before use", test = "`repoRoot` from a worktree subdirectory resolves to the worktree toplevel (pass); a checkpoint ref from another worktree's session is refused by branch name (the primary-checkout case)". If a realpath'd toplevel comparison is genuinely wanted, name the function and the task that builds it.

## No findings at remaining severities

No additional blockers, majors, or nits: the remaining design risks are either already mitigated by the plan's own devices (directive-protocol internal-ness → min-version pin + `supportsDirectiveProtocol` + stay-put; dual marker writers → README note + install warning, per settled R1-F003; 150ms bench claim → pi-rewind's flat-latency bench to 182k files, flagged in findings R3) or outside this design's trust boundary and correctly scoped out (untracked-files `stat` cost inside `IGNORED_DIRS`-named directories is bounded by the 10MiB/200-file caps and the batch-100 mechanics; `commit-tree` orphan objects are normal git housekeeping, not a leak the plugin must manage; the `SKILLGRID_CONTRACT_PATH` requirement is a stated seam, not a gap).
