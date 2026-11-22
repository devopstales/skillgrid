# Intermediate Zoom Out (Stage 8) — `blueprint.md`, @devopstales/pi-worktree

**Isolation attestation:** Part 1 (fresh context) and part 4 (no network/MCP) are enforced by the harness by construction; part 3 (no project instructions) is enforced by the dispatch prompt being my entire context. Part 2 (read-only) the harness cannot mechanically enforce — I enforced it by discipline: only `read`/`grep`/`glob` against the target and the explicit reading list, the sole write being this report file.

**Scope of this pass:** altitude drift + scope creep *introduced by the Stage 5/7 fixes* (R1-F001…F005, R2-F001…F009, R3-F001…F018), re-read against the whole document and the briefing's R1–R13.

**Overall:** The fix rounds net out to the right altitude. The two load-bearing Stage 5 fixes (branch-only restore, `observeWorktreeRoot`) *removed* complexity, the Stage 7 mechanical fixes (R3-F004…F017) are consistent, and every new device — the liveness-aware prune, the `skillgrid-constants` fence, `SKILLGRID_CONTRACT_PATH`, the `rebuildCache` dedup seeding, the `engines` field, the test resets — has a named, test-backed consumer and does not change architecture, interface, or dependency set (Self-Review item 6 "Additions" is accurate on all three). No scope creep: all additions trace to requirements R3/R4/MH-5 of the briefing's own R1–R13. Two of the fixes, however, leave behind a residue — one test that contradicts the fix's stated intent, and one whose stated invariant is unenforceable — plus three small weeds.

---

## Blockers

None.

## Majors

### M-1: The R3-F003 liveness-aware prune contradicts the blueprint's own "Per-worktree by construction" premise

**Affected Scope:** Task 4 (`pruneOtherSessions` + `PRUNE_MIN_AGE_MS`, ~lines 532–611); Global Constraints "Constants"; MH-5; Validation row R4; contract `_shared/work-unit-commits/README.md` § Scratch checkpoints ("Per-worktree by construction… prune at session start touches only the *current* worktree's refs").

**Simplicity Rationale:** The Stage 7 fix (R3-F003, user-chosen option (a)) adds `PRUNE_MIN_AGE_MS` = 1h and a two-session test on the ground that "in the fan-out scenario, 'other sessions' are concurrently-running sessions in sibling worktrees *sharing the same `refs/skillgrid/` namespace*" — i.e., that fan-out siblings share one ref namespace. But the contract (a reading-list document this plan implements verbatim) states the opposite premise for this design: *Per-worktree by construction — each worktree has its own ref namespace; prune at session start touches only the current worktree's refs.* That premise is load-bearing for the whole plan — the Key links bullet, the "Git repository selection" threat row, and the retired OD-2 rationale ("a checkpoint is only readable from the same repository") all rest on refs being per-worktree. Under that premise a fan-out sibling worktree never sees another session's refs, so the hazard the fix addresses is unreachable — and the 1h threshold is a new tunable constant (a knob) guarding an edge the architecture already excludes, contradicting the very design rationale (branch-only check = full safety, per-worktree refs) that the same review rounds settled. Two sub-problems follow. (a) The threshold rationale is false in the one case it claims to cover: a session interrupted >1h and restarted *is live from Pi's perspective* (resumable), yet its checkpoint — exactly the artifact the plugin exists to preserve — is pruned by its own session's restart. The comment "a live session's restart gap never crosses this" asserts a 1h gap bound with no source; it is a guessed number, and the plan's own testing discipline (falsifiers over assertions) applies to this invariant too. (b) Because the fan-out case is *unreachable* under the stated architecture, the two-session test cannot actually exercise it: both "sibling" and "dead" refs are created in the *same* repo (the test's `beforeEach` temp repo) and pruned purely by the injected clock — the test proves the arithmetic of `now - created > 1h`, not that a fresh *live* sibling's ref survives. If siblings truly shared a namespace, the test would need two repos; if they don't (the contract says they don't), the test and the constant are testing a non-edge. The fix should pick one premise: either the fan-out hazard is real (then the per-worktree premise needs a recorded requirement-level revision and the liveness signal should be something Pi actually knows — the original M-1's option (b), session liveness — not an age guess), or it isn't (then prune stays blind per the contract, the constant and the `now` injection parameter go away, and the "shared namespace" note in Task 4 is corrected).

**Cost of the fix:** One task's rework: drop or rebase `PRUNE_MIN_AGE_MS` + the `now` parameter on the chosen premise; correct the threat-matrix wording or record the revision; one line each in MH-5/Validation R4. If the per-worktree premise holds (it is the contract's and the plan's own settled rationale), net complexity *decreases*.

## Nits

### N-1: Test-count arithmetic is off by one again — the same class Stage 6 fixed (R2-F007)

**Affected Scope:** Task 4 Step 4, line 618 (`Expected: PASS (12 tests) …`).

**Simplicity Rationale:** Stage 6 corrected Task 3 to 9 tests, "confirmed by Task 4's 11"; the Stage 7 fix then added the third prune test (the liveness test) without touching Task 4 Step 4. The plan's own groups now count Task 2 = 4 + Task 3 = 5 + Task 4 prune group = 3 = 12 in `tests/scratch.test.ts`, so the literal is only right under one reading — and the document has now been caught with a stale test count twice in two rounds. An executor who verifies "Expected: PASS (11 tests)" at the expected-failure step gets a spurious mismatch. Restate the expected count once, derived from the actual groups shown.

**Cost of the fix:** One number.

### N-2: The `repoRoot(ctx.cwd)` helper is load-bearing but declared by no task

**Affected Scope:** Threat Matrix "Git repository selection" + "Untrusted input" rows (~lines 100–101, the R3-F005 rewrite); Task 5 Step 3 (four call sites, ~lines 655–672); File Structure.

**Simplicity Rationale:** The Stage 7 R3-F005 rewrite made `repoRoot(ctx.cwd)` a named component — it now carries the cwd→toplevel resolution that replaced the retired pair check — and Task 5's extension wiring calls it in every handler. But no task lists `repoRoot` in its Files, Interfaces, or Produces: Task 2 declares `git()` as the single git seam but not the toplevel resolver, and no shown test group exercises it (the threat-matrix test *names* it — "`repoRoot` from a worktree subdirectory resolves to the worktree toplevel" — yet Task 2's shown capture tests only ever call `createCheckpoint({root: repo})` with a pre-resolved root). This is the same phantom-symbol class as R3-F002 (`loadCheckpointFromRef`) — one iteration later, one layer down. It is a ~3-line `git rev-parse --show-toplevel` wrapper that the contract's shared snippet already provides (so it should be *reused*, not re-derived); the fix is to attach it to Task 2's Produces (or Task 5's Consumes) with the threat-matrix test attached, so the executor knows it must exist and is tested.

**Cost of the fix:** One interface line + one test-case reference; the helper itself is the contract's existing two-line snippet.

### N-3: `IGNORED_DIRS` is typed `Set` in Task 1's code but compared as an ordered list in its test and recorded as an ordered array in the contract fence

**Affected Scope:** Task 1 Step 1 (`expect([...c.IGNORED_DIRS]).toEqual([…])`), Task 1 Step 3 (`new Set([...])`); Task 11's `~~~skillgrid-constants` fence (`IGNORED_DIRS: ["node_modules", …]`); Task 9's drift-guard parse ("the fenced block is a key: value list").

**Simplicity Rationale:** A `Set` has no order. The import-scan test asserts the spread of the Set against an ordered literal with `toEqual` (order-sensitive), and the drift guard's fence records an ordered JSON array that the Task 9 test must parse into a value it then compares against an unordered structure. The comparison strategy (set-equality vs. ordered equality) is decided differently in the two test files, and one of them will silently pass when it should fail — or the guard needs a "normalize both sides" step that no task states. Pick one shape — array in the fence, Set in code, the guard comparing as sets — and state the comparison strategy once in Task 9 Step 1, which is the document that owns the comparison.

**Cost of the fix:** One sentence in Task 9 Step 1 + the one-line comparison strategy.

---

## No findings at remaining severities

No further altitude drift or scope creep at any severity: `supportsDirectiveProtocol`'s env-probe (pre-existing, not a fix artifact), the in-process mutex (recorded as accepted defense-in-depth), the `bun-types` devDep, the two smoke-script files, and the hand-written `references/wt-commands.md` all hold at the right altitude and stay inside the briefing's R1–R13.
