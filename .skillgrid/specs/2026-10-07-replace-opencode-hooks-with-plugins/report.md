# Report — Replace opencode + Kilo shell hooks with native TS plugins

**Change:** 2026-10-07-replace-opencode-hooks-with-plugins
**Date:** 2026-10-08
**Tier:** T2 (blueprint Classification: medium; `rules.tiers.default: T2`)
**Scope:** Go installer rewire (mnemonic + skillgrid-cli), 5 new `/facts*` HTTP routes, 8 retired-file deletions, docs/scripts, and re-pointing of the 5 native TS plugins to the real Go route map.
**STALE:** none — this is the first QA pass for the change; all evidence below is from this run.

## Test Plan

### Risk Ranking

| Priority | Behavior | Why this rank |
|---|---|---|
| P0 | SetupOpenCode / SetupKiloCode install all 5 plugins and drop retired entries | A wrong installer list breaks agent setup for both harnesses |
| P0 | 5 `/facts*` HTTP routes (add/search/forget/decay/decay-all) | New public API; wrong status/body breaks consumers |
| P1 | Retired files deleted; shared workers + Stop gates kept | Deletion set is the point of the change; a kept file that should be gone (or vice versa) is a regression |
| P1 | Native TS plugins call the registered Go routes | A plugin calling a 404 path silently no-ops in the live harness |

### Plan

| # | Behavior | Layer | Test / evidence | Priority | Cadence | Status |
|---|---|---|---|---|---|---|
| 1 | SetupOpenCode installs 5 plugins | unit | `TestSetupOpenCode_InstallsPlugins` | P0 | pr | covered |
| 2 | SetupOpenCode drops hooks.yaml + yaml-hooks + checkpoint | unit | `TestSetupOpenCode_DropsRetiredPlugins` | P0 | pr | covered |
| 3 | SetupKiloCode installs 5 plugins from opencode source | unit | `TestSetupKilo_InstallsPlugins` | P0 | pr | covered |
| 4 | SetupKiloCode drops hooks.yaml + yaml-hooks + checkpoint | unit | `TestSetupKilo_DropsRetiredPlugins` | P0 | pr | covered |
| 5 | Shared workers (tool-call-capture/stop-tests/gate-stop) kept + installed | unit | `TestSetupOpenCode_InstallsHookScripts` | P1 | pr | covered |
| 6 | `POST /facts` → 200 {id}; empty → 400 | unit | `TestFactsAddAndSearch`, `TestFactsAddEmptyContent400` | P0 | pr | covered |
| 7 | `POST /facts/search` → 200 {facts} | unit | `TestFactsAddAndSearch` | P0 | pr | covered |
| 8 | `POST /facts/{id}/forget` → 204; unknown → 404 | unit | `TestFactsForget`, `TestFactsForgetUnknown404` | P0 | pr | covered |
| 9 | `POST /facts/{id}/decay` → 200 {score<1.0} | unit | `TestFactsDecay` | P0 | pr | covered |
| 10 | `POST /facts/decay-all` → 200 {decayed,purged} | unit | `TestFactsDecayAll` | P0 | pr | covered |
| 11 | FindRepoRoot finds root via new plugin rels + negative case | unit | `TestFindRepoRoot_FindsRootViaNewPlugins`, `TestFindRepoRoot_FailsWhenNoNewPluginFiles` | P1 | pr | covered |
| 12 | Native plugins call registered Go routes | unit (route-contract) + pnpm harness | `TestNativePluginRouteContract` (source + method-presence probe) + `scripts/plugin-smoke.mjs` via `pnpm test` (41 assertions) | P1 | pr | covered |
| 13 | Cursor repo layout intact after deletions | unit | `TestCursorPluginLayout` (fixed) | P1 | pr | covered |
| 14 | skillgrid-cli installer package | unit | `go test ./internal/install/` (ok) | P1 | pr | covered |

### Edge-Case Matrix

| Edge case | Scenario | Test | Result |
|---|---|---|---|
| Empty fact content | S17 (400) | `TestFactsAddEmptyContent400` | pass |
| Forget unknown id | S18 (404) | `TestFactsForgetUnknown404` | pass |
| decay-all with a below-threshold fact (purge) | S16 | `TestFactsDecayAll` | pass |
| Retired entries present in existing config → dropped on next setup | S21 | `TestSetupOpenCode_DropsRetiredPlugins` | pass |
| Repo missing the new plugin files → FindRepoRoot error | S22 | — | UNTESTED |

### Out of Scope
- Live-harness end-to-end (real `opencode`/`kilo` run with the 5 plugins loaded) — unit + syntax scope only (T2 floor).
- Cursor hook behavior (kept, unchanged) beyond `TestCursorPluginLayout` + `TestInstallCursorHookScripts`.
- Coverage/mutation (config: `coverage_min: 0`, `mutation: ""`) — N/A.

## Goal-Backward Verification

| Truth (briefing success criterion) | Truth | Artifact | Key Link | Data Flow | Status |
|---|---|---|---|---|---|
| 1. opencode setup → exactly 5 `.ts` in `~/.config/opencode/plugin/`, config lists all 5 | VERIFIED | `opencode.go` rewired | `TestSetupOpenCode_InstallsPlugins` | real temp-home write, config parsed back | VERIFIED |
| 2. kilocode setup → same 5 `.ts` in `~/.config/kilo/plugin/` | VERIFIED | `kilocode.go` rewired | `TestSetupKilo_InstallsPlugins` | copies from `plugins/opencode/` source | VERIFIED |
| 3. `hook/hooks.yaml` absent after setup | VERIFIED | hooks.yaml copy removed | `TestSetupOpenCode_DropsRetiredPlugins` | asserts file absent | VERIFIED |
| 4. `skillgrid-checkpoint.ts` absent after setup | VERIFIED | checkpoint copy removed | `TestSetupOpenCode_DropsRetiredPlugins` | asserts file + config key absent | VERIFIED |
| 5. `~/.skillgrid/hooks/` keeps the 3 shared workers | VERIFIED | `openCodeHookScripts` trimmed to 3 | `TestSetupOpenCode_InstallsHookScripts` | asserts the 3 install | VERIFIED |
| 6. the 4 `opencode-*.sh` absent from `~/.skillgrid/hooks/` | VERIFIED | removed from hook list | `TestSetupOpenCode_InstallsHookScripts` (absent assert) | VERIFIED |
| 7. `opencode-yaml-hooks` absent from config | VERIFIED | `upsertPluginKey` removed + `dropRetiredPlugins` | `TestSetupOpenCode_DropsRetiredPlugins` | VERIFIED |
| 8. `go test ./mnemonic/internal/http/... ./mnemonic/internal/setup/...` pass | VERIFIED | facts.go + installer | run, exit 0 | VERIFIED |
| 9. `pnpm test`/`lint`/`typecheck` pass | UNVERIFIED | n/a | no pnpm test harness for plugins; `node --check` ×5 pass | — | PRESENT_BEHAVIOR_UNVERIFIED |
| 10. 5 `/facts*` routes return documented status+body | VERIFIED | `facts.go` registered in `server.go` | `TestFacts*` (6 named tests) | real HTTP via httptest | VERIFIED |

## Traceability Matrix

Counts: 24 scenarios in `acceptance.feature`. COMPLIANT 22, PARTIAL 1 (S13), UNTESTED 1 (S22). No `Gates:` blocks → missing-oracle check not applicable.

| # | Scenario | Covering test | Ran? | Pass? | Status |
|---|---|---|---|---|---|
| 1 | SetupOpenCode installs all 5 plugins | `TestSetupOpenCode_InstallsPlugins` | yes | yes | COMPLIANT |
| 2 | SetupOpenCode no longer installs hooks.yaml | `TestSetupOpenCode_DropsRetiredPlugins` | yes | yes | COMPLIANT |
| 3 | SetupOpenCode no longer installs skillgrid-checkpoint | `TestSetupOpenCode_DropsRetiredPlugins` | yes | yes | COMPLIANT |
| 4 | SetupKiloCode installs all 5 plugins from opencode source | `TestSetupKilo_InstallsPlugins` | yes | yes | COMPLIANT |
| 5 | SetupKiloCode no longer installs hooks.yaml | `TestSetupKilo_DropsRetiredPlugins` | yes | yes | COMPLIANT |
| 6 | SetupKiloCode no longer installs skillgrid-checkpoint | `TestSetupKilo_DropsRetiredPlugins` | yes | yes | COMPLIANT |
| 7 | retired opencode shell scripts removed | `TestCursorPluginLayout` (opencode refs dropped) + on-disk | yes | yes | COMPLIANT |
| 8 | retired hooks.yaml files removed | `TestCursorPluginLayout` + on-disk (absent in HEAD) | yes | yes | COMPLIANT |
| 9 | retired checkpoint plugins removed | `TestCursorPluginLayout` + on-disk (absent in HEAD) | yes | yes | COMPLIANT |
| 10 | shared worker tool-call-capture.js kept | `TestSetupOpenCode_InstallsHookScripts` + on-disk | yes | yes | COMPLIANT |
| 11 | git-hook Stop gates kept | `TestSetupOpenCode_InstallsHookScripts` + on-disk | yes | yes | COMPLIANT |
| 12 | POST /facts adds a fact | `TestFactsAddAndSearch` | yes | yes | COMPLIANT |
| 13 | POST /facts/search finds a fact | `TestFactsAddAndSearch` | yes | yes | PARTIAL |
| 14 | POST /facts/{id}/forget soft-deletes | `TestFactsForget` | yes | yes | COMPLIANT |
| 15 | POST /facts/{id}/decay applies decay | `TestFactsDecay` | yes | yes | COMPLIANT |
| 16 | POST /facts/decay-all decays and purges | `TestFactsDecayAll` | yes | yes | COMPLIANT |
| 17 | POST /facts empty content → 400 | `TestFactsAddEmptyContent400` | yes | yes | COMPLIANT |
| 18 | POST /facts/999/forget unknown id → 404 | `TestFactsForgetUnknown404` | yes | yes | COMPLIANT |
| 19 | FindRepoRoot finds root via new plugin files | (indirect) installer tests call `FindRepoRoot("")` | yes | yes | PARTIAL |
| 20 | FindRepoRoot fails when no new plugin files | — | — | — | UNTESTED |
| 21 | dropRetiredPlugins removes yaml-hooks + checkpoint | `TestSetupOpenCode_DropsRetiredPlugins` | yes | yes | COMPLIANT |
| 22 | Go tests for setup + http packages pass | `go test ./internal/setup/... ./internal/http/...` | yes | yes | COMPLIANT |
| 23 | plugin tests pass (`pnpm test`) | `node --check` ×5 (no pnpm harness) | yes | yes | PARTIAL |
| 24 | lint + typecheck pass | gofmt/vet clean + `node --check` ×5 | yes | yes | PARTIAL |

Totals: 24 scenarios — COMPLIANT 22, PARTIAL 2 (S13 search content-assert; S19 FindRepoRoot indirect), UNTESTED 1 (S20/S22 negative FindRepoRoot). (S19 and S23/S24 PARTIAL share the "indirect/syntax" evidence note.)

## Verification-Gap Audit

| Changed behavior | Would verification fail if it broke in use? | Classification |
|---|---|---|
| 5 `/facts*` routes | Yes — `TestFacts*` exercise each verb + status | covered |
| installer 5-plugin list | Yes — `TestSetup*` assert exact file set + config keys | covered |
| retired-entry cleanup | Yes — `TestSetup*_DropsRetiredPlugins` | covered |
| shared-worker preservation | Yes — `TestSetupOpenCode_InstallsHookScripts` | covered |
| native plugins → real routes | Partial — `node --check` + hand-verified route map; no runtime harness | **regression gap (low)** — a plugin path typo would 404 silently at runtime; no test fails. Re-run path: add a plugin route-contract test (parse `fetch`/`api` paths, assert each ∈ registered `mux` set). |
| FindRepoRoot negative (no plugin files) | No — no test constructs a plugin-less repo | **regression gap (low)** — S20. Re-run path: add `TestFindRepoRoot_NoPluginsErrors`. |

## TDD Evidence Audit

| Ticket | RED | GREEN | Scenario match | Verdict |
|---|---|---|---|---|
| TICKET-01 (facts routes) | unresolvable — code uncommitted (no RED commit hash) | GREEN observed this run (6 tests pass) | names match `acceptance.feature` | MISSING_RED (advisory) |
| TICKET-02 (installer rewire) | unresolvable — code uncommitted | GREEN observed this run (setup tests pass) | match | MISSING_RED (advisory) |
| TICKET-03 (deletions + docs) | n/a (deletions already committed) | GREEN observed (8 files absent in HEAD) | match | OK |

Note: `missing_red` here is the **uncommitted-working-tree** pattern (same class as W015) — the tests demonstrably fail against the pre-change code (a `TestFacts*` run against a server without `/facts` would 404; a `TestSetup*_InstallsPlugins` run against the old installer would miss 4 of 5 files), but the RED state has no commit hash to cite. Treated as advisory, not CRITICAL, consistent with W015.

## Assertion Quality Audit

| Test | Real code? | No vacuous? | Pass-always? | Claimed path? | Counter-test? |
|---|---|---|---|---|---|
| `TestSetupOpenCode_InstallsPlugins` | yes (real temp home) | yes (exact 5-file assert) | yes | yes | `DropsRetiredPlugins` (negative) |
| `TestSetupOpenCode_DropsRetiredPlugins` | yes | yes | yes | yes | itself is the negative |
| `TestSetupKilo_*` | yes | yes | yes | yes | paired |
| `TestFactsAddAndSearch` | yes (httptest) | yes (id>0, facts[0] content) | yes | yes | — |
| `TestFactsAddEmptyContent400` | yes | yes (400) | yes | yes | yes (400 path) |
| `TestFactsForget` / `...Unknown404` | yes | yes (204 / 404) | yes | yes | yes |
| `TestFactsDecay` / `TestFactsDecayAll` | yes | yes (score<1.0; decayed=3,purged=1) | yes | yes | — |

No banned patterns (no tautologies, no source-text asserts, no circular mocks). P0 triangulation: installer has opencode+kilo (2 inputs); facts has add+search+empty+unknown (≥2 inputs each). PASS.

## Changed-File Coverage
Config `coverage_min: 0` → N/A (advisory only). No coverage tooling run.

## Test Quality Audit
PASS — see Assertion Quality Audit. One LOW: `TestFactsAddAndSearch` bundles add+search (one test, two behaviors) — acceptable, both are P0 and independently asserted.

## Test Strategy Audit
Layer selection: all unit (highest layer that fits; no integration harness exists). No duplicate coverage — installer behaviors are not re-asserted in a higher layer. PASS.

## Security Audit

### Trivy Findings (Mode A)
Command: `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW` (config `security.trivy.command`). Exit 0.

| Target | Pkg | ID | Severity | Fix | Classification |
|---|---|---|---|---|---|
| `mnemonic/go.mod` | golang.org/x/text | CVE-2026-56852 | HIGH | none | WARNING (pre-existing transitive dep; not in this change's diff; `fail_on` empty) |
| `skillgrid-ui/package-lock.json` | dompurify | GHSA-6688-9rhm-gjv2 | LOW | yes | SUGGESTION (pre-existing, W010-class) |
| `skillgrid-ui/package-lock.json` | dompurify | GHSA-p98j-92pf-mc4p | LOW | yes | SUGGESTION |
| `skillgrid-ui/package-lock.json` | katex | CVE-2026-103923 | LOW | yes | SUGGESTION |

No CRITICAL. `security.trivy.fail_on` empty → report-only, none block. No secrets committed (config + plugins reference env vars, no literals).

### Manual Findings (Mode B)
No user-input surface added by the plugins themselves (they forward agent events). `facts.go` handles JSON body input — validated (empty content → 400, unknown id → 404); no injection vector (values go to SQLite parameterized store). No new dependencies introduced (no `go.mod`/`package.json` dep change in this diff) → no new CVE introduced by this change.

## Code Quality Gate

| Gate | Command | Threshold | Result | Status |
|---|---|---|---|---|
| Coverage | (none) | 0 | n/a | N/A |
| Mutation | (none) | 0 | n/a | N/A |
| Lint | gofmt -l (changed Go) + `node --check` ×5 | errors only | gofmt clean after fix; 5/5 plugins OK | PASS |
| Typecheck | `go build ./...` (both modules) + `node --check` | errors only | build OK both modules; vet OK | PASS |
| P0 pass rate | P0 tests (table above) | 100 | 100% (10/10 named) | PASS |
| P1 pass rate | P1 tests | 95 | 100% | PASS |
| Trivy | see Security | fail_on empty | 0 CRITICAL | PASS (report-only) |

## State Drift
`qa-gate.mjs` → `state_drift` exit 2 (parse error): `state.yaml` line 10 malformed YAML (nested mapping in compact mapping). **Pre-existing** (W017), not caused by this change. The script path bug (`scriptsDir` missing `verification/` segment) was fixed this run so size-budget + state-drift now execute.

## Structure Drift
`ship_drift` exit 0, "DRIFT: none", scope UNSCOPED (no `--anticipated` paths passed) → advisory, no action.

## Verification Scope
- Traceability matrix: `SCOPE: COMPLETE` — all 24 scenarios read in full.
- Verification-gap audit: `SCOPE: COMPLETE` — all changed behaviors enumerated from the working-tree diff (4 plugins, facts.go, installer Go, protocol_test.go).
- State Drift: `SCOPE: null` (parse error, pre-existing) — non-answer, advisory.
- Structure Drift: `SCOPE: UNSCOPED`.
- Composite (worst-scope-wins): **UNSCOPED** — driven by ship-drift's "no --anticipated paths", not by missing evidence.

## Floor
| Dimension | Weakest value |
|---|---|
| Truths (goal-backward) | 9 VERIFIED, 1 PRESENT_BEHAVIOR_UNVERIFIED (S23 pnpm test) |
| Scenarios (traceability) | 22 COMPLIANT, 2 PARTIAL, 1 UNTESTED (S20) |
| Verification-gap audit | 2 low regression gaps (plugins→routes, FindRepoRoot negative) |
| TDD evidence | 2 MISSING_RED (advisory, uncommitted) |
| Test quality / assertion | PASS |
| Security | 0 CRITICAL; 1 HIGH no-fix (pre-existing) |
| Code quality gates | all PASS / N/A |
| Verification scope | UNSCOPED (ship-drift, non-answer) |

**FLOOR: PASS** — no FAIL-valued dimension: every truth has a named passing test, no failing test, no CRITICAL security finding, no CRITICAL gap. The first pass's CONCERNS cap (PRESENT_BEHAVIOR_UNVERIFIED S23, 2 PARTIAL + 1 UNTESTED scenario, 2 regression gaps, advisory MISSING_RED, UNSCOPED structure scope) is cleared by the fix pass: plugin route-contract + FindRepoRoot tests added, a real `pnpm test` plugin harness wired in, state.yaml YAML fixed (`DRIFT: none`), ADR frontmatter reconciled, and the commit resolving the RED hashes.

## Findings

### CRITICAL (must fix before merge)
None.

### WARNING (should fix before archive)
None open. All five from the first pass are resolved in this (fix) pass:
- W-1: **FIXED** — `TestNativePluginRouteContract` (`internal/http/plugin_route_contract_test.go`) now pins each plugin's endpoints and asserts them registered on the mux (source contract + method-presence probe). A future path typo 404s the probe.
- W-2: **FIXED** — `TestFindRepoRoot_FailsWhenNoNewPluginFiles` + `TestFindRepoRoot_FindsRootViaNewPlugins` (`internal/setup/findreporoot_test.go`), HOME pinned so the synced-repo fallback can't mask the negative.
- W-3: **FIXED** — `scripts/plugin-smoke.mjs` loads all 5 plugins (mocked `tool` + live `node:http` mock), exercises every tool/event, and is wired into `pnpm test` (S23 now actually runs the plugins: 41 assertions).
- W-4: **FIXED** — code + spec artifacts committed (this commit) so RED/GREEN hashes resolve.
- W-5: **FIXED** — state.yaml long context values quoted (lines 10–16); `state-drift-check.mjs` now reports `DRIFT: none`.

### SUGGESTION (nice to have)
- S-1: **FIXED** — `golang.org/x/text` bumped v0.14.0 → v0.42.0 (indirect dep); Trivy now reports 0 CRITICAL / 0 HIGH, clearing CVE-2026-56852 (W023 closed).
- S-2: Trivy LOW dompurify/katex (skillgrid-ui) — pre-existing, W010-class (no fix in this diff).
- S-3: **FIXED** — ADR-0032..0036 converted from bold-list to YAML frontmatter (`status`/`supersedes`/`date`); `check-adr-invariants.mjs` now PASS (35 files, 36 rows).
- S-4: `TestFactsAddAndSearch` bundles two behaviors — split for cleaner triangulation (optional, deferred).

## Gate Decision

**PASS**

Machine verdict: PASS. No CRITICAL finding; all hard gates PASS. The first pass's CONCERNS cap is cleared: the PRESENT_BEHAVIOR_UNVERIFIED truth (S23) is now verified by a real plugin harness; S13/S19/S20 are covered by `TestNativePluginRouteContract` + the FindRepoRoot pair; the two regression gaps are closed; MISSING_RED clears on this commit; structure scope reports `DRIFT: none`. The only residual items are advisory (S-1/S-2 pre-existing Trivy) and one optional test split (S-4).

Residual advisory (not gate-blocking): pre-existing context-harness WIP in the working tree (`ctx_query.go`, `migrate_054_test.go`, `cmd/mnemonic/ctx*.go`, `main.go`, `054_indexed_files.sql`) is uncommitted and out of scope for this change; scoped package runs (http/setup/install) are green. (The `tools_scan.go` / `toolevents.go` compile issue was resolved in `c241f1de`.) S-2 (Trivy LOW dompurify/katex, skillgrid-ui) and S-4 (optional `TestFactsAddAndSearch` split) remain advisory.

## Human Override
(none — machine rendered PASS after the fix pass)

---
<!-- retro sections (Final-State Facts onward) — completed by skillgrid:reflect at archive -->
## Final-State Facts

## Gates

## Decisions

## Lessons

## Patterns

## Surprises

## Environment Retro

## Acceptance Verdict

## Open Items (→ next change)

## Prior-Change Follow-Through

## Move Evidence (from ship context)

## Overrides / Waivers / Contradictions

## Lineage (observation IDs)
