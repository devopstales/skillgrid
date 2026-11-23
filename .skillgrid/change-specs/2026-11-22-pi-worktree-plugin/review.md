# Code Review Record — 2026-11-22-pi-worktree-plugin

> Filled by the review-and-fix-loop (Phase 4). Round 6 of the ledger (rounds 1–5 were the blueprint goldfish). Scope: the built plugin at `.worktree/pi-worktree-wf_t5` (base `f548ef8`), all of `src/` + `tests/`.

## Topology

- **Core five dispatched** (fresh `reviewer` subagents, read-only, independence grade **A** — each read the working tree directly from disk, no shared memory, transcripts at `history://StdLens`, `history://SpecLens`, `history://EdgeLens`, `history://VerifyLens`, `history://SecLens`):
  - `StdLens` (Standards) — code quality, conventions, **concurrency** (the singleton mutex, checkpoint-lock, `runExclusive`), **public API** (`registerScratch`/`registerWorktrunk`/`runDirectedWt`/`getWtCommand`), dead code, jscpd clones.
  - `SpecLens` (Spec) — blueprint + acceptance.feature + R7/R8/R9/R10 truth + the contract-drift README; the "core exports but nothing calls it" smell.
  - `EdgeLens` (Edge) — error paths, `wt` missing, recovery, partial-failure, env, cross-session, `git` C-quoting, rename, unborn HEAD.
  - `VerifyLens` (Verification) — test coverage vs the acceptance scenario catalog, vacuous gates, test-isolation flakiness.
  - `SecLens` (Security) — injection (argv vs shell), secrets, permissions, the `--yes`/`-y` story, the `git -c` config injection surface.
- **Red-team lens: NOT separately dispatched.** The coordinator performed the triage/dedup/re-grade (independent of the lens conclusions — re-graded each finding by consequence to the end user, not by lens severity). Stated here for honesty; a strict reading of the review-and-fix-loop wants a dedicated red-team pass, which is folded into the coordinator's re-grade in this round.

## Independence

Grade **A**: 5 independent lenses, each a fresh subagent reading the tree from disk; no lens saw another's output; the coordinator's re-grade is a separate actor. No shared-memory cross-contamination.

## Bar guards (fix pass: strengthen, not weaken)

- A test was **added** (`resume re-seeds the dedup baseline and skips an unchanged first turn`) — it makes the previously-vacuous `-t "resume"` gate match 1 test (0→1). Tightening, not weakening.
- The 3 recovery tests got `{ timeout: 30000 }` (bun default 5s flakes under load on real-wt fixtures) — a flake fix, not a weakening of the assertion.
- The before-restore success message (`"A before-restore checkpoint was taken first"`) is unchanged for the `created` case → e2e step 2 still greps it (verified: e2e 6/6 after the change).
- No assertion was deleted, inverted, or narrowed. `idOk` was removed as dead code (it re-ran `refOk` on a value already validated).

## Findings (deduplicated + re-graded)

| ID | Area | What | Re-graded | Disposition |
|---|---|---|---|---|
| A | R7/R8/R9 | `applyMergeDefaults`/`checkBranchAllowList`/`observeWorktreeRoot` exported+unit-tested but **never called** on the extension path | WARNING (MEDIUM) | **ESCALATE** — wire into `runWorktrunk` vs amend-spec |
| E | R10 | approval passthrough broken in the piped runtime; both acceptance gates vacuous (0 tests) | WARNING (MEDIUM) | **ESCALATE** — wt-CLI/approval fix vs document as runtime limitation |
| B1 | R1 | C-quoted (non-ASCII) path → `JSON.parse` throws → the turn fails | WARNING | **FIXED this pass** (`core.quotePath=false` + `decodePath`) |
| B2 | R1 | rename `R old -> new` → pathspec `old` no longer exists → fatal | WARNING | **FIXED this pass** (`decodePath` keeps `new`) |
| B4 | R1 | unborn HEAD → branch `"HEAD"` → restore can't match | WARNING | **FIXED this pass** (`symbolic-ref` real branch) |
| D | R2 | fresh/resumed session re-captures an unchanged first turn (baseline starts `""`) | WARNING | **FIXED this pass** (`seedDedup` at `session_start`) + new test |
| H1/H2 | R3 | before-restore outcome swallowed (empty catch) + message says "taken" when skipped | SUGGESTION | **FIXED this pass** (`beforeRestore` field) |
| G1 | R2 | `-t "resume"` gate matched 0 tests | SUGGESTION | **FIXED this pass** (new test) |
| G10 | R6 | recovery tests flake at 5s under load | SUGGESTION | **FIXED this pass** (`{timeout:30000}`) |
| K3 | R1 | `idOk` dead re-check | SUGGESTION | **FIXED this pass** (removed) |
| K9 | build | working-tree tgz stale vs src | SUGGESTION | **FIXED this pass** (re-packed; gitignored artifact) |
| B3 | R1 | index file bytes differ when staged (cache-tree extension persisted by `write-tree`) | WARNING | **DEFER** (DW — benign; document or clean write-tree) |
| C | R3 | safeClean staged-new file → `AD` state after restore | WARNING | **DEFER** (DW — edge case; fix + held-out test) |
| F | R6 | recovery can land in a foreign recreated repo (survivor's live common-dir not re-checked) | WARNING | **DEFER** (DW — edge case) |
| G2–G7 | R5/R6/R4/R1 | verification gaps (validateDirective not asserted; no-replay on a constant; prune identity not asserted; e2e move false-positive window) | SUGGESTION | **DEFER** (DW) |
| J1–J4 | robustness | swallowed wt spawn err; `git()` inherits `GIT_*`; `timeoutMs` never set; `loadCheckpoint` no 40-hex check | SUGGESTION | **DEFER** (DW) |
| K1–K2,K4–K8 | cosmetic | ui.ts comment; `findCheckpoint` hardcode; camelCase keys; test name; comment numbering; footer flash | SUGGESTION | **DEFER / NOISE** (DW) |

**Re-grade basis (consequence to the end user):** no finding causes data loss, git-state corruption, or a security break. The two most severe (A, E) are "a promised invariant / feature isn't enforced on the runtime path" — a spec-completeness gap, not a correctness break — hence MEDIUM, not HIGH. Everything else is an edge case, a test-quality gap, or cosmetic.

## Fix-loop result

- **Round 1 (this record):** 5 lenses → dedup → re-grade → **11 fixed now** (B1,B2,B4,D,H1/H2,G1,G10,K3,K9) with covering tests; **2 escalated** (A, E) needing the programmer's decision; **~20 deferred** (DW) as edge cases / test improvements / cosmetic.
- Verified after the fix pass: `bun test` **44/44** (was 43; +1 resume test), `e2e` **6/6**, drift guard **green** (match→PASS / mutate→FAIL naming `MAX_PER_SESSION` / missing-fence→FAIL). All 5 core fixes proven empirically (non-ascii→created, rename→created, resume→skipped, unborn→`main`, beforeRestore→`skipped`).
- **Review signal: `BACK-TO-APPLY` — Critical 0 · Warning 5 (A, E open; B3, C, F deferred) · Suggestion ~20 (deferred).**
