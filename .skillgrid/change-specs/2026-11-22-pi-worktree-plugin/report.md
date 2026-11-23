# Test Report — 2026-11-22-pi-worktree-plugin

> Filled by `skillgrid:test`. Area order: intent → functional → code quality → security → review. Verdict rendered from the four-state gate and the floor.

## Header

- **Change:** 2026-11-22-pi-worktree-plugin — `@devopstales/pi-worktree`
- **Target URL:** (none) — a Pi agent plugin, no running web target; DAST + browser pass N/A
- **Effective tier:** T2 (blueprint `Tier:`; `rules.tiers.default` T1 does not raise it)
- **Applied floor:** L2+L3 (T2: full blueprint, two-axis review; L1+runtime satisfied)
- **Date:** 2026-11-23
- **Build under review:** plugin repo `.worktree/pi-worktree-wf_t5`, base `f548ef8` → review fixes `9376e19` (this report's fix pass)

## Test Plan

| Behavior / Truth | Source | Layer | Test | Cadence |
|---|---|---|---|---|
| capture-after-mutating-turn (1 ref, full metadata, index untouched) | R1 / scenario | unit + e2e | `scratch.test.ts` "mutating turn…", "index byte-identical"; e2e step 1 | pr |
| dedup-on-unchanged-tree (incl. resume) | R2 / scenario | unit | "read-only turn is skipped", "resume re-seeds the dedup baseline…" | pr |
| safe-restore (branch-only refusal, before-restore, safe-clean) | R3 / scenario | unit + e2e | "restore on matching branch…", "refused: branch mismatch"; e2e step 2 | pr |
| bounded-refs (50 cap, cross-session liveness) | R4 / scenario | unit | "exactly 50…", "51st capture prunes…", "liveness threshold" | pr |
| directive-follow (move, print stopped, outside-repo rejected) | R5 / scenario | unit + e2e | e2e step 3; `wt-core.test.ts` "directive naming a path outside…" | pr |
| no-replay-recovery | R6 / scenario | unit + e2e | `wt-core.test.ts` "recovery" group (real wt); e2e step 5 | pr |
| standard-root-observed | R7 / scenario | unit | `wt-core.test.ts` "observeWorktreeRoot" (function only — see A) | pr |
| branch-allow-list | R8 / scenario | unit | `wt-core.test.ts` "checkBranchAllowList" (function only — see A) | pr |
| merge-defaults | R9 / scenario | unit | `wt-core.test.ts` "applyMergeDefaults" (function only — see A) | pr |
| approval-passthrough | R10 / scenario | unit | `wt-core.test.ts` "approval" — **0 tests (vacuous, see E)** | pr |
| marker-lifecycle | R11 / scenario | unit | `markers.test.ts` lifecycle + stale (real wt) | pr |
| core-tests-without-pi | R12 / scenario | unit | `import-scan.test.ts` + full `bun test` | pr |
| contract-drift-guard | R13 / scenario | unit | `contract-drift.test.ts` (match/mutate/missing-fence) | pr |

Out of scope: conversation-state restore (Pi owns `/fork`/`/tree`), per-tool checkpoints, shadow git repos, auto-restore, worktrunk's picker TUI / OSC 7 / alias machinery (briefing § Out of scope).

## Intent

Tier: T2 (floor: L2+L3)

| Truth (briefing) | Level | Named test | Status |
|---|---|---|---|
| R1 mutating turn → 1 ref + full metadata, index untouched | Data Flow | "mutating turn creates exactly one ref…" + "index byte-identical" + e2e step 1 | VERIFIED (caveat B3: index bytes differ when staged) |
| R2 read-only turn → no ref | Data Flow | "read-only turn is skipped (dedup…)" | VERIFIED (fresh-session first turn creates a baseline ref — documented) |
| R2 resume → no duplicate | Data Flow | "resume re-seeds the dedup baseline…" (new) | VERIFIED (fixed this pass — D) |
| R3 restore on matching branch; refused cross-branch (named); before-restore first | Data Flow | "restore on matching branch…", "refused: branch mismatch", e2e step 2 | VERIFIED (caveat C: staged-new file → AD state) |
| R4 50/session cap + cross-session liveness prune | Data Flow | "exactly 50…", "51st capture…", "liveness threshold" | VERIFIED |
| R5 directive move / print stopped / outside-repo rejected | Data Flow | e2e step 3; "directive naming a path outside…" | VERIFIED (caveat G2: validateDirective not unit-asserted) |
| R6 cwd-removal recovery, no-replay | Data Flow | "recovery" group (real wt) + e2e step 5 | VERIFIED (caveats F, G5) |
| R7 standard root observed, foreign noted | Key Link | "observeWorktreeRoot" (parse fn only) | PRESENT_BEHAVIOR_UNVERIFIED — `observeWorktreeRoot` has no caller (A) |
| R8 branch allow-list at creation (worktree ctx) | Key Link | "checkBranchAllowList" (fn only) | PRESENT_BEHAVIOR_UNVERIFIED — not wired to the tool (A) |
| R9 merge `--no-squash --no-commit` default | Key Link | "applyMergeDefaults" (fn only) | PRESENT_BEHAVIOR_UNVERIFIED — not wired to the tool (A) |
| R10 approval passthrough, no `-y` injection | Key Link | `approval` gates — 0 tests | PRESENT_BEHAVIOR_UNVERIFIED — broken in piped runtime (E) |
| R11 marker lifecycle + stale | Data Flow | `markers.test.ts` (real wt) | VERIFIED |
| R12 core runs without Pi (import-scan) | Artifact | `import-scan.test.ts` + `bun test` (44/44) | VERIFIED |
| R13 contract-drift guard | Artifact | `contract-drift.test.ts` (3 halves) | VERIFIED |

Scenarios: 26 total, 22 covered+passed, 4 gaps (R7/R8/R9 runtime-wiring — finding A; R10 runtime — finding E).

## Functional

Suite: `bun test` → exit 0, **44 tests, 0 failing** (was 43; +1 resume-dedup test this pass)
E2E: `bash tests/e2e.sh` → exit 0, **6/6 steps** (step 6 clear-verify is a documented wt-state-cache SKIP)
P0: 100%/100   P1: 100%/95  (no P0/P1-labeled tests; whole suite is the floor)

| Behavior | Layer | Test | Ran? | Result |
|---|---|---|---|---|
| turn checkpoint + dedup + restore + prune | unit | `scratch.test.ts` (15) | yes | pass |
| wt runner / invariants / recovery | unit | `wt-core.test.ts` (real wt) | yes | pass |
| marker lifecycle + stale | unit | `markers.test.ts` | yes | pass |
| core purity + contract drift | unit | `import-scan`, `contract-drift` | yes | pass |
| full smoke (capture→restore→move→recovery→marker) | e2e | `e2e.sh` (real Pi + wt) | yes | pass (1 documented skip) |
| **bench 5k-file median ≤ 150ms** | perf | `bun run bench` | yes | **FAIL — 161.2ms / 238.5ms over two runs (shared load); recorded idle carry 65–117ms** |
| pack structure | build | `npm pack` (bun 1.4.5 has no `bun pack`) | yes | pass — 12 files, respects `files:["src/","references/"]` |

Browser pass: N/A — no reachable URL.
Regression: 0 behavioral deltas without a covering test (the 3 findings fixed this pass each got a new/updated test — see Fix Plans).

## Code Quality

jscpd: 1 pass (typescript, `--ignore node_modules,*.lock,*.md`) on src+tests → **1.66% dup lines** (49 lines; 6 clones, all test-fixture `rm`/`clearWorktreeRecord` boilerplate in `wt-core.test.ts`). `duplication.fail_on: 0` → advisory, PASS.
Health: no linter configured; no coverage/mutation (thresholds 0 → N/A).

| Gate | Command | Threshold | Result | Status |
|---|---|---|---|---|
| Coverage | — | 0 | — | N/A |
| Mutation | — | 0 | — | N/A |
| Lint | — (no `commands.lint`) | — | — | N/A |
| Typecheck | — (no `commands.typecheck`; `bun test` does not type-check) | — | — | N/A |
| jscpd duplication | `npx jscpd` | 0 (advisory) | 1.66% | PASS |
| P0 pass rate | `bun test` | 100 | 100% | PASS |
| P1 pass rate | `bun test` | 95 | 100% | PASS |

## Security

trivy: `trivy fs --scanners vuln,secret,misconfig --severity CRITICAL` → **0 findings** (0 vuln / 0 secret / 0 misconfig) — `fail_on: CRITICAL` PASS.
owasp (via the Security lens): no CRITICAL. Injection-clean (all git/wt spawn are argv-array, never a shell); the one `-y`/`--yes` the code adds is `wt config approvals add --yes` (a store subcommand) + the capability probe's `-y` — documented, not on the user's command path.
nuclei / wapiti / akca: N/A — no target URL.

| # | Sev | Title | Tool | Location | Confidence |
|---|---|---|---|---|---|
| — | none at/above CRITICAL | | | | |

## Findings

### CRITICAL
none (re-graded by consequence to the end user — no data loss, no git-state corruption, no security break). The most severe are WARNING-high (below).

### WARNING
- **A — Unwired invariants (merge-defaults, branch-allow-list, standard-root).** `applyMergeDefaults`, `checkBranchAllowList`, `observeWorktreeRoot` are exported + unit-tested but **never called on the extension path** (`runWorktrunk` passes `[command, ...args]` straight to `wt`). At runtime: `wt merge main` runs with wt's default (squash), `wt switch --create feature-x` in a worktree is accepted (no rejection/suggestion), foreign roots are never noted. Multi-confirmed by all 5 lenses. `src/wt.ts:222,292,326`. → **ESCALATE** (wire vs amend-spec).
- **E — Approval passthrough broken in the piped runtime.** wt is spawned with pipes (non-interactive); `wt config approvals add --yes` exits 1 "Cannot prompt for approval in non-interactive environment" even with `--yes` → a dialog approval can never be saved; `wt switch --create` piped with an unapproved hook exits 0 with no `[approval]` line → `isApprovalNeeded` never fires for the primary case. Both acceptance gates are vacuous (0 tests named "approval"). `src/wt.ts:154,186,83-85`. → **ESCALATE** (wt-CLI fix vs document as a runtime limitation).
- **B3 — Real index not byte-identical after capture when staged.** `git write-tree` on the real index (step 3) persists the cache-tree extension when the index has staged changes, so the index *file* differs (the tree SHA does not — the existing test only compares `write-tree` output). Violates the R1 literal "byte-identical". `src/core/scratch-core.ts:209`. → DEFER (benign; document or force a clean write-tree).
- **C — safeClean staged-new file → AD state.** `safeClean` set 1 (`--diff-filter=AD HEAD <snapshot>`) deletes a file that was staged-but-uncommitted at capture; `read-tree --reset` re-stages it without restoring the worktree copy → `AD` after restore. `src/core/scratch-core.ts:337-361`. → DEFER (edge case; needs a fix + held-out test).
- **F — Recovery can land in a foreign recreated repo.** `recover()` filters candidates by *recorded* identity; a surviving path is validated by `stat` + `isExistingGitDir` but its **live** git-common-dir is never re-compared to `cwdIdentity` → if the removed dir is recreated as an unrelated repo before the next call, the session recovers into it. `src/core/wt-core.ts:457-491`. → DEFER (edge case; re-verify common-dir of the survivor).

### SUGGESTION
- G2 directive-outside-repo gate doesn't call `validateDirective` (unverified stay-put). G5 no-replay is asserted on a constant (`replayed:false` always; e2e greps the tool's self-report). G6 "51st prunes oldest" asserts count not the "oldest" identity or the `createCheckpoint→prune` wiring. G7 e2e step-3 move assertion is a false-positive window (wt's own stdout contains the path). G10 5s flake (fixed this pass — `{timeout:30000}` on the 3 recovery tests).
- J1 `runDirectedWt` swallows the spawn `err` (wt missing → `code:-1`, empty stderr; inconsistent with `git()`/`wt()`). J2 `git()` inherits host `GIT_*` env. J3 `timeoutMs` never set by the extension → a hung `wt` holds the module mutex. J4 `loadCheckpoint` doesn't validate 40-hex `head`.
- K1 `ui.ts` rebuildCache comment claims implicit seeding (now true after D). K2 `findCheckpoint` hardcodes `"skillgrid/"` not `REF_BASE`. K4 metadata keys `largeFiles`/`largeDirs` camelCase vs kebab siblings. K6/K7/K8 minor (test name, comment numbering, footer flash).
- Fixed this pass (not open): B1 non-ASCII capture, B2 rename capture, B4 unborn-HEAD branch, D dedup seeding, H1/H2 before-restore outcome+catch, G1 `-t "resume"` gate, K3 dead `idOk`, K9 stale tgz.

## Gate Decision

- **Verdict:** **CONCERNS**
- **Floor (weakest dimension):** review → BACK-TO-APPLY with open **MEDIUM** entries (A, E) + the bench non-answer.
- **Criteria met:** intent (9/13 fully VERIFIED), functional (44/44 suite, 6/6 e2e, import-scan, drift guard, pack), code quality (jscpd advisory, P0/P1), security (trivy 0, owasp no CRITICAL).
- **Criteria unmet (the non-answers):** (1) the bench 5k gate (161/238ms under shared load vs 150ms; idle carry 65–117ms) — re-run path: `bun run bench` on an idle machine; (2) 4 acceptance requirements are PARTIAL at runtime — A (R7/R8/R9 not wired), E (R10 broken in the piped runtime).
- **Why CONCERNS not FAIL:** no CRITICAL after re-grade (no data loss / state corruption / security break); all hard gates PASS; the open items are MEDIUM (tolerable, should-fix) plus a load-sensitive bench — exactly the "human can clear by re-running / deciding" shape.
- **Human decision (if any):** pending at the `qa-verdict` gate — see Check-in (wire-vs-defer A, E, and the bench).

## Fix Plans (open items → `execute --gaps-only`)

| Item | Root cause | Fix | Verification |
|---|---|---|---|
| A (R7/R8/R9 unwired) | extension passes args to wt without calling the core invariants | **Wire** `applyMergeDefaults` + `checkBranchAllowList` (+ `observeWorktreeRoot` note) into `runWorktrunk` before `runWt`; **or** amend `acceptance.feature` to record the deferral (core provides, wt governs) | e2e: `wt merge main` argv carries `--no-squash --no-commit`; `switch --create feature-x` in a worktree is rejected with `agent-feature-x`; foreign root noted |
| E (R10 approval) | wt spawned piped/non-interactive; `--yes` store path fails; detection misses the piped exit-0 case | fix the approval detection for the piped exit-0 case + use an interactive-capable store (or a non-prompting store); add the 2 vacuous `approval` unit tests | a real unapproved hook → dialog → saved → retried once; declined → fails, approvals unchanged |
| B3 (index bytes) | `write-tree` on the real index persists cache-tree when staged | capture the index tree from a temp copy, or `git -c core.useIndexVersion=4 write-tree` / compare tree SHA not file bytes | index file sha256 unchanged with staged changes present |
| C (safeClean AD) | set 1 `--diff-filter=AD` includes staged-new files | exclude staged-new (`A`) from the delete set, or restore the worktree copy after `read-tree` | held-out test: capture a staged-new file → restore → worktree file present, no `AD` |
| F (foreign repo) | survivor not re-verified against live common-dir | re-run `git rev-parse --git-common-dir` on the chosen survivor and compare to `cwdIdentity` | held-out test: remove worktree, recreate dir as an unrelated repo, recover → no move into the foreign repo |
| bench (R12) | shared load, not a code regression (variance 161→238ms) | re-run idle; if still >150ms, raise the 5k gate or reduce git-call count | `bun run bench` on an idle machine ≤150ms |

## Project Wiki

- **Filed:** (pending ship — the review surfaced durable gotchas: "git C-quoting octal breaks JSON.parse" and "write-tree persists cache-tree on a dirty index" → `gotchas/`; the "wt piped runtime is non-interactive" approval limitation → `gotchas/`). Filed in Phase 7.
- **Index:** run in Phase 7.

## Human Override

- **Override:** none yet — at the `qa-verdict` gate.
