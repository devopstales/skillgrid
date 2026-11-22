# Report — Skillgrid host plugins (router auto-load at session start)

> Change: `.skillgrid/change-specs/2026-11-17-skillgrid-host-plugins/` (moves to `.skillgrid/change-archive/` at ship)
> Generated: 2026-11-22T~21:00Z (qa)
> Gate: PASS
>
> Two phases, one file: **qa** writes the QA half (through `## Gate Decision` + `## Human Override`). **ship** reads the verdict PRE-MOVE. **reflect** completes the retro half in the archive.
>
> Effective tier: **T2** (blueprint header `rigor: full (5 hosts, installer changes)`) → L2 + L3 audits. Change-classification floor met (live-system verification + installer integration tests).

## Test Plan

### Risk Ranking

Most likely to break in production: **silent non-injection** — the router never reaches the model's first-turn context (stale start-dir on opencode's daemon, a hook contract mismatch, an installer that skips the router files), and nothing errors. The plan therefore tests hardest: (1) each host's injector actually reads the live shared asset and injects it (per-host unit suites), (2) the installer distributes exactly the per-host file set + bootstrap copy (Go integration tests), (3) fail-open (no session-start breakage, static fallback).

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Live asset read by every pointer host | each host injector → `router-bootstrap.md` | unit | `plugins/opencode/test/skillgrid-router.test.ts` (14 checks), `.pi/extensions/skillgrid-router.test.ts` (4), `plugins/kilo/skillgrid-router.test.ts` (10) | P0 | pr | covered |
| 2 | Per-host session-start injection (5 hosts) | host native mechanism → first-turn context | unit + live smoke | the 5 suites + `node --test hooks/session-start.test.mjs` (4); live: `review/smoke-evidence.md` TICKET-07 | P0 | pr | covered |
| 3 | Project-conditional guard (R3) | cwd walk-up → `.skillgrid/config.yaml` | unit | non-project cases in all 5 suites | P0 | pr | covered |
| 4 | Main-agent only / subagent skip (R3) | agent-kind guard | unit + live | subagent cases in all suites; live omp subagent probe (TICKET-07) | P0 | pr | covered |
| 5 | Fail-open + static fallback (R4) | injector error path | unit | fallback cases in all 5 suites (kilo case5 hermetic after QA fix) | P0 | pr | covered |
| 6 | Installed-copy precedence (R4) | candidate order `[installed, checkout]` | unit | kilo case5 (HOME-isolated; precedence proven by pre-fix failure mode) | P1 | pr | covered |
| 7 | Installer distributes per-host injector + bootstrap (R5) | `skillgrid install <key>` | integration | `TestRunDistributesRouterPerHostKey` (skillgrid-cli/internal/install/router_distribution_test.go) | P0 | pr | covered |
| 8 | Unknown agent key errors, no writes (R5) | key validation | integration | `TestSetupAgentsUnknownKeyNamesKeyNoWrites` | P1 | pr | covered |
| 9 | sync-repo refresh + unchanged no-op (R5) | `sync-repo` | integration | `TestSyncRepoRefreshesRouterBootstrap`, `TestSyncRepoRouterBootstrapDryRun` | P1 | pr | covered |
| 10 | Bootstrap load-hint table content (R6) | `router-bootstrap.md` | unit (content oracle) | G6 gate command (see `acceptance.feature`); runtime fail-open in live smoke | P1 | pr | covered |
| 11 | opencode daemon-cwd start dir (T7 fix) | `resolveStartDir` | unit | `plugins/opencode/test/skillgrid-router.test.ts` (session.get location.directory cases) | P0 | pr | covered |

**Layer selection** — `testing.layers = [unit, integration]` (no e2e layer configured). Host-injection behavior is unit-tested at the plugin/hook seam (the host runtime is the e2e boundary, exercised by the live smoke, recorded as EVIDENCE). Installer behavior is integration (real fs, temp HOME). Duplicate Coverage Guard: the live smoke does not replace unit suites (it verifies the host runtime boundary the units cannot reach); units do not re-test installer distribution.

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| R1 | all four pointer hosts, one asset | identical bootstrap content | G1 suites (marker/first-action assertions) + live diff (TICKET-07) | covered |
| R1 | asset missing at session start | static fallback, no block | kilo case5, pi/omp/opencode fallback cases, hook test | covered |
| R3 | non-skillgrid directory | no injection | case4 (kilo), non-project cases (pi/omp/hook) | covered |
| R3 | subagent/child session | no injection | case3 (kilo), subagent cases (pi/omp/hook) + live probe | covered |
| R3 | unreadable config | treated as no project | hook test (fail-open), live /tmp runs (0/2) | covered |
| R4 | hook fails mid-run | session starts, no error | hook fail-open test; T7 live (INFO-to-stderr contract) | covered |
| R4 | both asset sources absent | static fallback | kilo case5 (hermetic) | covered |
| R4 | installed vs checkout marker | installed wins | candidate-order code + kilo case5 proof | covered |
| R5 | unknown key | error names key, no writes | `TestSetupAgentsUnknownKeyNamesKeyNoWrites` | covered |
| R5 | unchanged re-run | no rewrite | `TestSyncRepoRefreshesRouterBootstrap` (unchanged re-sync) | covered |
| R6 | mapping file deleted | bootstrap still injected | live smoke (pi env-limited — hint is inert text, no load dependency) | covered (ruling recorded) |

### Out of Scope

- **Marketplace distribution** (Cursor/Kilo/omp plugin marketplaces) — deferred by ratified OQ1 (installer-only v1); not in this change's scope.
- **kilo headless runtime** — kilo core `System message must be at the beginning` bug, plugin-independent (reproduced with empty plugin array, T5 + T7 re-confirmation); env-limited ruling recorded in the smoke evidence.
- **plain-pi runtime** — no runtime on this machine (pre-rename `@devopstales/skill-pi` npm package 404s); env-limited ruling.
- **ADR-0067 memory rework** of the mnemonic plugin trees (`plugins/opencode/` 5-plugin family relocation) — belongs to `2026-11-17-mnemonic-v2-host-memory-plugins`.

## Goal-Backward Verification

**Stated goal** (briefing.md): make every supported agent host deterministically auto-load the skillgrid router (`using-skillgrid`) at session start, so the router runs whether or not the agent notices the `AGENTS.md` entry line.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth R1 | one shared live asset, no reinstall for edits | G1 suites (live fs read per host); live: marker edit verified in TICKET-07 opencode run | VERIFIED |
| Truth R2 | all 5 hosts inject at session start | 5 unit suites + live smoke: opencode/omp/cursor PASS, kilo/pi env-limited (unit-verified) | VERIFIED |
| Truth R3 | project-conditional + main-agent only | guard cases in all suites; live /tmp negatives (0/2) + live omp subagent skip | VERIFIED |
| Truth R4 | fail-open + static fallback + installed precedence | fallback cases in all suites; hook contract test; kilo case5 | VERIFIED |
| Truth R5 | installer distributes per host + bootstrap | `TestRunDistributesRouterPerHostKey` + fresh 3-agent install green (19.3s) | VERIFIED |
| Truth R6 | tool-mapping hints for pi/opencode only | G6 content oracle + bootstrap asset read-back | VERIFIED |
| Artifact | `plugins/{opencode,kilo}/skillgrid-router.ts`, `.pi/extensions/skillgrid-router.ts`, `.omp/extensions/skillgrid-router-omp.ts`, `hooks/session-start.js` (guard), `router-bootstrap.md` | files on disk at release/2; each exercised by a named test above | VERIFIED |
| Key Link | installer → host config dirs → host runtime injection | integration tests + live smoke (installed copy served in first-turn context) | VERIFIED |
| Data Flow | edited asset → next session first-turn context | TICKET-07 live opencode run: proxy-captured request carries the edited bootstrap (len 118391) | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| R1 happy path — one-line edit reaches next session | G1 suites + live TICKET-07 opencode run | yes | pass |
| R1 edge — all four pointer hosts read identical asset | G1 suites (first-action/marker assertions) + live diff | yes | pass |
| R1 failure — asset missing → static fallback | kilo case5 + pi/omp/opencode/hook fallback cases | yes | pass |
| R2 happy path — cursor injects with server down | hook oracle (emits before mnemonic block) + live cursor run | yes | pass |
| R2 happy path — omp full router content | `.omp/extensions/skillgrid-router-omp.test.ts` (4) + live omp verbatim quote | yes | pass |
| R2 happy path — pi bootstrap | `.pi/extensions/skillgrid-router.test.ts` (4); runtime env-limited | yes | pass (unit) |
| R2 happy path — opencode bootstrap | `plugins/opencode/test/skillgrid-router.test.ts` (14) + live capture | yes | pass |
| R2 happy path — kilo bootstrap | `plugins/kilo/skillgrid-router.test.ts` (10); runtime env-limited | yes | pass (unit) |
| R2 edge — installer re-run no duplication | present-guards in all suites + `TestRunDistributesRouterPerHostKey` | yes | pass |
| R2 failure — cursor hook absent → no bootstrap | non-project case shape (no hook = no emission) + live /tmp negative | yes | pass |
| R3 happy path — project session receives bootstrap | G3 suites + live in-repo runs | yes | pass |
| R3 edge — non-skillgrid dir receives nothing | case4 (kilo) + non-project cases (pi/omp/hook) + live /tmp (0/2) | yes | pass |
| R3 edge — subagent skips bootstrap | case3 (kilo) + subagent cases + live omp subagent probe | yes | pass |
| R3 failure — unreadable config = no project | hook fail-open test + live /tmp runs | yes | pass |
| R4 failure — hook failure never blocks start | hook test (INFO + exit 0) + T7 live | yes | pass |
| R4 failure — neither source resolvable → fallback | kilo case5 (hermetic) | yes | pass |
| R4 edge — installed copy precedence | candidate order + kilo case5 proof | yes | pass |
| R5 happy path — host key distributes files + bootstrap | `TestRunDistributesRouterPerHostKey` | yes | pass |
| R5 happy path — sync-repo refreshes installed copy | `TestSyncRepoRefreshesRouterBootstrap` | yes | pass |
| R5 edge — unchanged re-run leaves copy unchanged | `TestSyncRepoRefreshesRouterBootstrap` (unchanged re-sync) | yes | pass |
| R5 failure — unknown key errors | `TestSetupAgentsUnknownKeyNamesKeyNoWrites` | yes | pass |
| R6 happy path — mapping hint for pi/opencode | G6 content oracle | yes | pass |
| R6 edge — plain hint for kilo/cursor/omp | G6 content oracle (negative grep) | yes | pass |
| R6 failure — mapping file missing → still routes | live smoke (pi env-limited; hint is inert text) | yes | pass (ruling) |

**Coverage:** 24/24 scenarios covered by a test that ran and passed (kilo/pi runtime-bound scenarios unit-covered + env-limited ruling recorded; live-bound scenarios live-verified in TICKET-07).

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| kilo case5 depended on `~/.skillgrid/` being empty | plugins/kilo/skillgrid-router.test.ts:91-116 (pre-QA-fix) | broken-verification | test passed pre-install, failed post-install (RED observed this QA run: `case5: static fallback used (not the live asset)`) | installed copy added → fallback test silently serves the live asset |
| installed-precedence edge has no dedicated marker test | all 5 injectors (`candidates = [installed, repo]`) | missing-oracle (P1, advisory) | code ordering + kilo case5 pre-fix failure mode + live runs (identical content, indistinguishable markers) | reorder candidates → no unit fails |

Both gaps resolved or adjudicated: the first fixed hermetically in this QA run (RED→GREEN recorded); the second recorded as a SUGGESTION (simple two-element candidate order, live-verified, unit-proven indirectly).

## TDD Evidence Audit

Standard mode (`testing.tdd: false`) — failing-test-before-code is the baseline; RED states recorded in `review/smoke-evidence.md` where the subagent ran the test pre-implementation.

| Task | SATISFIES | RED (record) | GREEN (commit) | TRIANGULATE | REFACTOR (suite exit) | Verdict |
|---|---|---|---|---|---|---|
| TICKET-01 | R1/R2/R3 cursor scenarios | live door-check: no bootstrap pre-hook (recorded 32395394) | `148370da` (+`34d04190` walk-up bound) | live cursor run (oracle 809ch) | 0 | OK |
| TICKET-02 | R3/R4 hook scenarios | `1 pass / 3 fail` pre-change (evidence L157) | `c8f7e0d5` | hook test 4/4 | 0 | OK |
| TICKET-03 | R2 pi scenario | MODULE_NOT_FOUND exit 1 (evidence L217) | `864c1e75` (+`6e6716a5`) | 4/4 | 0 | OK |
| TICKET-04 | R2 opencode scenario | module absent pre-`8fb76304` (TICKET-04 evidence) | `8fb76304` (+`60adab41` version-tolerance) | 14/14 | 0 | OK |
| TICKET-05 | R2 kilo scenario | MODULE_NOT_FOUND exit 1 (evidence L277) | `aafe9e21` | 10/10 | 0 | OK |
| TICKET-06 | R5 scenarios | distribution tests red pre-`197d46d3` (installer had no router step) | `197d46d3` (+`d5fc320d` ordering) | 4 router-distribution tests | 0 | OK |
| TICKET-07 | backstop (all live scenarios) | smoke RED: opencode daemon-cwd, hook contract, kilo core bug, pi-hooks dispatch order | `09350a03` `f8309b89` `0c542a07` `8fcb3039` | 5-host live matrix | 0 | OK |
| TICKET-08 | none (docs) | n/a | `1705b110` | read-back (3 acceptance checks) | n/a | OK |

**Summary:** 8/8 tickets with complete evidence (Standard mode: RED recorded + GREEN resolves in git + named tests re-run green in this QA pass).

## Assertion Quality Audit

| File | Line | Pattern | Evidence | Severity |
|---|---|---|---|---|
| plugins/kilo/skillgrid-router.test.ts | 91-116 | env-dependent assertion (pre-fix) | case5 asserted static fallback while the installed asset could satisfy the plugin — assertion passed only on an uninstalled machine | WARNING (fixed this QA run; hermetic via HOME isolation) |
| all 5 suites | — | no tautology / ghost-loop / no-code-call patterns | every assertion names a production path (transform/extension/hook invocation) and checks content markers, not source text | — |

**Summary:** 0 CRITICAL, 1 WARNING (resolved in-loop), 0 SUGGESTION.

## Changed-File Coverage

`testing.coverage` empty (advisory per config; `quality.coverage_min: 0`). Changed Go packages, this tree (release/2 @ 1705b110; code identical to the 645d0612 measurement except docs):

| File / package | Line % | Rating |
|---|---|---|
| `skillgrid-cli/internal/install` | 66.9% (measured 645d0612, code-identical) | Low (<80, advisory) |
| `skillgrid-cli/internal/setup` | 73.4% (measured 645d0612, code-identical) | Low (<80, advisory) |
| `plugins/opencode/skillgrid-router.ts`, `plugins/kilo/skillgrid-router.ts`, `.pi/extensions/*`, `.omp/extensions/*`, `hooks/session-start.js` | n/a (no JS coverage tooling configured) | n/a |

**Threshold:** `quality.coverage_min` = 0 → gate N/A; values reported informationally.

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| plugins/kilo/skillgrid-router.test.ts case5 (pre-fix) | no-vacuous (env-conditional pass-always-on-installed-machines) | failed this QA run after a fresh install; hermetic now |
| all other suites | none observed | assertions check injected content markers + negative cases (subagent/non-project/missing-asset) with real fs |

## Test Strategy Audit

**Available layers:** unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| host injection (5 hosts) | unit (plugin/hook seam) + live smoke | no | no | host runtime is the e2e boundary; units pin the seam, smoke pins the runtime |
| installer distribution | integration | no | no | real fs + temp HOME; unit layer would mock the seam under test |
| bootstrap content | unit (content oracle) | no | no | asset is the single source; content check is the oracle |

**Layer distribution:** 11 unit / 3 integration / 0 e2e. Reasonable: the change's risk concentrates at the host-runtime boundary, which the live smoke covers as recorded evidence (no e2e layer is configured in this repo).

## Security Audit

**Mode:** Trivy (config) — advisory only (`fail_on: ""`; repo rule: Trivy never blocks).

**Trivy gate:** `fail_on: ""` → N/A (findings reported, never blocking). No new third-party dependencies in this change (no `go.mod` change; host plugins are dependency-free TS; `@hsingjui/pi-hooks` pre-existing). No user-facing input surface added: the hooks read fs + stdin JSON with bounded walk-ups (fail-open). Secrets: none introduced (no credentials in any new file; `~/.skillgrid/` paths only).

**Security verdict:** N findings — CRITICAL: 0, WARNING: 0, SUGGESTION: 0.

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage (whole project) | not configured | 0 (advisory) | n/a | N/A |
| Changed-file coverage | not configured | 0 (advisory) | install 66.9% / setup 73.4% (informational) | N/A |
| Mutation | not configured | 0 | n/a | N/A |
| Lint | `go vet` (inline via pre-commit) + gate-lint | errors only | 0 errors (gate-lint clean after G-gates added) | PASS |
| Typecheck | `node --experimental-strip-types` (all suites) + `go build` | errors only | 0 errors | PASS |
| P0 pass rate | all P0 rows' tests | 100 | 100 (48 node checks + 4 Go tests + full suite exit 0) | PASS |
| P1 pass rate | all P1 rows' tests | 95 | 100 | PASS |
| Trivy security | `trivy fs .` | report only | n/a (advisory) | N/A |
| Duplication (jscpd) | `npx jscpd` (changed files) | 0 (advisory) | 1 clone · 2.21% TS (13 tokens) | PASS (advisory) |
| Full suite | `go test ./...` (skillgrid-cli/) | exit 0 | exit 0, 162.8s (install 125.9s, setup 155.5s) | PASS |
| gofmt (committed tree) | `gofmt -l` | clean | clean (sole working-tree flag is inside the user's uncommitted rename WIP) | PASS |

**Quality config status:** configured (thresholds as above).

**Duplication:** scoped to the 5 changed injectors + hook; 1 clone (13 tokens, 2.21%) — the shared `findProjectRoot` walk-up pattern, intentional: host plugins must be standalone (no shared runtime across host boundaries, blueprint Design contract). SUGGESTION-level at most.

## State Drift

**Verdict:** none material to this change. `changes.2026-11-17-skillgrid-host-plugins.phase` set to `qa` in this pass (state.yaml committed alongside). Router-reported drift on 5 other archived/parked change slugs predates this change and is out of scope (recorded, not blocking).

**Scope:** COMPLETE (phase-router run + state.yaml read in full this pass).

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix (24 scenarios) | COMPLETE | none (this run's tests) |
| Verification-gap audit | COMPLETE | none (changed paths re-read post-T8) |
| State Drift | COMPLETE | n/a |

**Stale-verification:** `STALE: none` — full suite + all 5 node suites re-run on release/2 @ 1705b110 in this pass (kilo RED→GREEN in-loop).

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED (all 6 truths) |
| Traceability (weakest scenario) | COMPLIANT (24/24 ran + passed) |
| Verification-gap audit | SUGGESTION (precedence edge, adjudicated) |
| TDD evidence (weakest ticket) | OK (8/8, Standard mode) |
| Assertion quality audit (weakest finding) | WARNING → resolved in-loop (hermetic kilo case5) |
| Changed-file coverage (weakest file) | N/A (threshold 0, advisory) |
| Test quality audit | resolved (kilo case5) |
| Code-quality gates (weakest gate) | PASS |
| Security audit | N/A (advisory, no new deps) |
| Verification scope (composite) | COMPLETE |

**FLOOR:** verification-gap audit — SUGGESTION (installed-precedence edge lacks a dedicated marker test; code-order + kilo case5 + live runs cover it). No dimension at FAIL/UNVERIFIED/MISSING_RED/UNREADABLE. Gate may render PASS.

## Findings

### CRITICAL (must fix before merge)

None.

### WARNING (should fix before archive)

None open. (kilo case5 env-dependence fixed in this QA run: RED observed → hermetic HOME isolation → GREEN.)

### SUGGESTION (nice to have)

- Installed-precedence edge (R4) has no dedicated marker-line unit test; covered by candidate-order code + kilo case5 proof + live runs. A marker test (installed asset carries marker X, checkout marker Y, expect X) would close it. → WINDOWS.md W-row appended.
- `install_test.go` rename WIP line (user's uncommitted work) is gofmt-misaligned; will need `gofmt -w` when the rename commits. Informational only — not part of the committed tree.

## Gate Decision

**Verdict:** PASS

**Reasoning:** All 6 truths VERIFIED by named tests + live smoke; 24/24 scenarios ran and passed on the integrated tree (release/2 @ 1705b110); full Go suite exit 0 (162.8s) + all 5 node suites green after one in-loop test fix (kilo case5, RED→GREEN recorded); no CRITICAL findings; all hard gates PASS/N-A per config; verification scope COMPLETE. Two env-limited host runtimes (kilo core bug, pi no-runtime) are recorded rulings with unit coverage, not verification gaps.

**Open items (if CONCERNS):** none.

**Waiver (if WAIVED):** n/a.

## Human Override

**Human decision (fill in):** — (machine verdict PASS; no override needed) — — —

<!-- reflect completes this half at archive time -->
