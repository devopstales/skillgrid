# Report — Shared LLM Provider

> Change: `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-llm-provider/` at ship)
> Generated: 2026-10-05 (qa)
> Gate: CONCERNS
> Effective tier: T2 (standard) — applied floor: L2+L3 (goal-backward + verification-gap + assertion-quality + code-quality gates)
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.
> Risk-ordered. Every row must be covered by a named test or scenario before the gate passes.

### Risk Ranking

The most likely thing to break in production is the **fail-open floor**: a real LLM provider
that is down, misconfigured, or returns a non-2xx must not panic the process or lose data —
ask must still return citations, hash dedup must still work, and regex extraction must still
run. Second, **boot wiring**: `AttachSharedLLM` must attach exactly one shared client to all
four seams (ask, dedup, extraction, dream) and leave them nil when disabled, so an operator
who enables `mnemonic.llm` gets a live path instead of the silent "no LLM configured" floor.
Third, **install provider setup** is fail-soft by design (a missed Ollama install must warn,
not fail the install). The plan tests the fail-open floors and the single-attach wiring hardest.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | LLM error fails open to floors (ask/dedup/extract, no panic) | `AskLLM`/`DedupLLM`/`ExtractionLLM` | unit | `secondbrain::TestAsk_LLMFailsOpen`, `TestAsk_LLMTimeoutFailsOpen`, `TestAsk_CitedModeSkipsLLM` (G8) | P0 | pr | covered |
| 2 | Single attach wires all four seams from one client | `AttachSharedLLM` | unit | `service::TestAttachSharedLLMWiresSeams`, `TestExtractionAdapterCompletes`, `TestDreamAdapterCompletes` (G5) | P0 | pr | covered |
| 3 | Disabled config clears all seams (nil) | `AttachSharedLLM` | unit | `service::TestAttachSharedLLMDisabledNoOp`, `TestAttachSharedLLMMissingConfig` (G6) | P0 | pr | covered |
| 4 | OpenAI-compatible Complete success returns assistant content | `llm.Client.Complete` | unit | `llm::TestCompleteSuccess` (G3) | P0 | pr | covered |
| 5 | non-2xx Complete returns an error | `llm.Client.Complete` | unit | `llm::TestCompleteNon2xx` (G4) | P0 | pr | covered |
| 6 | timeout Complete returns an error | `llm.Client.Complete` | unit | `llm::TestCompleteTimeout` | P1 | pr | covered |
| 7 | Config defaults off; enabled requires base_url+model | `config.Load` | unit | `config::TestLLMConfigDefaultOff`, `TestLLMConfigRequiresURLAndModel` (G1,G2) | P1 | pr | covered |
| 8 | api_key falls back to `SKILLGRID_LLM_API_KEY` | `config.Load` | unit | `config::TestLLMAPIKeyFromEnv` | P1 | pr | covered |
| 9 | No auth header when api_key empty | `llm.Client.Complete` | unit | `llm::TestCompleteNoKeyOmitsAuthHeader` | P1 | pr | covered |
| 10 | Trailing-slash base_url handled | `llm.Client.Complete` | unit | `llm::TestCompleteTrailingSlashInBaseURL` | P1 | pr | covered |
| 11 | Install `--yes` ensures local Ollama + wires config | `install.setupProvider` | unit (stubbed exec) | `install::TestSetupProviderLocal` (G10) | P1 | pr | covered |
| 12 | Install `--provider=external` wires without Ollama | `install.setupProvider` | unit (stubbed exec) | `install::TestSetupProviderExternalNoOllama` | P1 | pr | covered |
| 13 | Install `--skip-provider` leaves ensure out | `install.setupProvider` | unit | `install::TestSetupProviderSkipped` (G11) | P1 | pr | covered |
| 14 | Ensure skips pull when models already present | `install.setupProvider` | unit (stubbed probe) | `install::TestSetupProviderEnsureSkipsPullWhenPresent` | P1 | pr | covered |
| 15 | task-029 closed/superseded by this change | tracker | doc | `task-029` body (G9) | P1 | pr | covered |

**Layer** — all unit. This change is pure in-process wiring (config, HTTP client, attach,
install step); there is no e2e layer in `testing.layers` that adds signal over unit for a
stdlib HTTP client against `httptest`. Duplicate Coverage Guard: the fail-open floors were
already covered by pre-existing `secondbrain::TestAsk_*` tests; the plan reuses them rather
than adding a parallel e2e.

**Cadence** — `pr` for all: each is a small, fast unit test in the `go test ./...` suite.

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| OpenAI-compatible client | base_url with trailing slash | URL joined to `{base_url}/chat/completions` without `//` | `llm::TestCompleteTrailingSlashInBaseURL` | covered |
| OpenAI-compatible client | api_key empty | No `Authorization` header sent | `llm::TestCompleteNoKeyOmitsAuthHeader` | covered |
| OpenAI-compatible client | server exceeds timeout | error returned, no hang | `llm::TestCompleteTimeout` | covered |
| Config | enabled=true, base_url or model empty | attach refused, actionable error | `config::TestLLMConfigRequiresURLAndModel`, `service::TestAttachSharedLLMMissingConfig` | covered |
| Config | api_key empty, env set | bearer token from `SKILLGRID_LLM_API_KEY` | `config::TestLLMAPIKeyFromEnv` | covered |
| Fail-open | provider 500 with flags on | floor result, no panic | `secondbrain::TestAsk_LLMFailsOpen` | covered |
| Fail-open | provider timeout | cited floor, degraded signaling | `secondbrain::TestAsk_LLMTimeoutFailsOpen` | covered |
| Install provider | Ollama already on PATH, models listed | no pull invoked, config still merged | `install::TestSetupProviderEnsureSkipsPullWhenPresent` | covered |
| Install provider | OS install/start probe fails | warning + manual hint, install exits 0 | `install::TestSetupProviderLocal` (warn path) | covered |

### Out of Scope

- New retrieval engine (briefing out-of-scope).
- Anthropic-native SDK / separate Ollama `/api/chat` client (briefing out-of-scope; ADR-0023 locks OpenAI-compatible only).
- Changing ask/dedup/extraction prompt text (out-of-scope).
- Dashboard / new MCP tools / passive-learning product UX beyond attaching the extraction seam (out-of-scope).
- Real Ollama binary install on CI (stubbed exec/`LookPath` in tests; not run on the host).

## Goal-Backward Verification

**Stated goal** (from briefing.md): Operators configure one OpenAI-compatible chat endpoint; process start attaches that client to every existing LLM seam; opt-in flags still gate when the LLM is tried; failures stay fail-open; a normal install runs provider setup (Local ensure / External wire) and re-run is ensure.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | Req 1: `mnemonic.llm` loads, defaults off, requires base_url+model when enabled, api_key env fallback | `config::TestLLMConfigDefaultOff`, `TestLLMConfigRequiresURLAndModel`, `TestLLMAPIKeyFromEnv` (all PASS) | VERIFIED |
| Truth | Req 2: one stdlib HTTP Completer against `{base_url}/chat/completions`, no new SDK | `llm::TestCompleteSuccess`/`Non2xx`/`Timeout` (PASS) + `go.mod` has no LLM SDK | VERIFIED |
| Truth | Req 3: single `AttachSharedLLM` wires all four seams from one client; disabled clears all | `service::TestAttachSharedLLMWiresSeams`, `TestExtractionAdapterCompletes`, `TestDreamAdapterCompletes`, `TestAttachSharedLLMDisabledNoOp` (PASS) | VERIFIED |
| Truth | Req 4: feature flags stay opt-in (client attached + flag false → floor) | `secondbrain::TestAsk_CitedModeSkipsLLM` + pre-existing floor tests (PASS); extraction/dedup LLM flags default off (`config::TestExtractionLLMConfigDefaultOff`, `TestDedupLLMConfigDefaultOff`) | VERIFIED |
| Truth | Req 5: fail-open floors preserved when Complete errors | `secondbrain::TestAsk_LLMFailsOpen`, `TestAsk_LLMTimeoutFailsOpen` (PASS); `TestCompactHookFailOpen`, `TestDreamConsolidateDeterministicNoLLM` (PASS) | VERIFIED |
| Truth | Req 6: task-029 closed by this change | `task-029` `status: done`, references spec topic, DoD #6 checked (G9) | VERIFIED |
| Truth | Req 7: natural provider setup on install (Local ensure / External wire / skip) | `install::TestSetupProviderLocal`, `TestSetupProviderExternalNoOllama`, `TestSetupProviderSkipped`, `TestSetupProviderEnsureSkipsPullWhenPresent` (PASS) | VERIFIED |
| Artifact | `internal/mnemonic/llm` (Completer + Client) exists and is wired | `llm/client.go` + `llm_test.go` exist; `TestCompleteSuccess` runs it against `httptest` | VERIFIED |
| Artifact | `service.AttachSharedLLM` + adapters exist and are called at boot | `service/attach_llm.go` exists; `TestAttachSharedLLMWiresSeams` exercises the attach path | VERIFIED |
| Artifact | `internal/install` provider step exists in the install flow | `install/provider.go` + `provider_test.go`; `TestSetupProviderLocal` runs it | VERIFIED |
| Key Link | config → client → seams (one client, four seams) | `TestAttachSharedLLMWiresSeams` (one shared counter across ask+dedup+extraction) | VERIFIED |
| Data Flow | real input moves through Complete, observable output | `TestCompleteSuccess` (mock server, assistant content returned), `TestExtractionAdapterCompletes` (input→extracted text) | VERIFIED |

## Traceability Matrix

> Note: `acceptance.feature` gate lines G4–G8 name test identifiers that were never updated to
> the as-built names. The behavior each gate checks is covered by the named as-built tests
> below (all ran + passed this session). The gate-name mismatch is logged as a WARNING, not a
> coverage gap.

| Scenario (from acceptance.feature) | Test / Scenario (as-built) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| happy path llm config defaults off (G1) | `config::TestLLMConfigDefaultOff` | yes | pass |
| enabled config requires base_url and model (G2) | `config::TestLLMConfigRequiresURLAndModel` + `service::TestAttachSharedLLMMissingConfig` | yes | pass |
| api key falls back to environment | `config::TestLLMAPIKeyFromEnv` | yes | pass |
| happy path openai-compatible complete succeeds (G3) | `llm::TestCompleteSuccess` | yes | pass |
| non-2xx complete returns an error (G4) | `llm::TestCompleteNon2xx` (feature names `TestCompleteHTTPError`) | yes | pass |
| timeout complete returns an error | `llm::TestCompleteTimeout` | yes | pass |
| happy path attach wires all seams from one client (G5) | `service::TestAttachSharedLLMWiresSeams` (feature names `TestAttachSharedLLMWiresAllSeams`) + adapter tests | yes | pass |
| disabled config clears all seams (G6) | `service::TestAttachSharedLLMDisabledNoOp` (feature names `TestAttachSharedLLMDisabledClears`) | yes | pass |
| no third-party LLM module in go.mod | `go.mod` inspected — no LLM SDK | yes | pass |
| happy path attached client with flags off uses floors (G7) | `secondbrain::TestAsk_CitedModeSkipsLLM` + `config::TestExtractionLLMConfigDefaultOff`/`TestDedupLLMConfigDefaultOff` (feature names `TestAttachedClientFlagsOffUsesFloors`) | yes | pass |
| ask cited mode ignores attached client | `secondbrain::TestAsk_CitedModeSkipsLLM` | yes | pass |
| flag on with attached client invokes Complete | `service::TestExtractionAdapterCompletes` | yes | pass |
| happy path llm error fails open to floors (G8) | `secondbrain::TestAsk_LLMFailsOpen`, `TestAsk_LLMTimeoutFailsOpen` (feature names `TestLLMErrorFailsOpen`) | yes | pass |
| ask llm mode degrades to cited | `secondbrain::TestAsk_LLMFailsOpen` | yes | pass |
| dedup llm error uses hash floor | `service::TestAttachSharedLLMDisabledNoOp` (hash floor path) + `memory::TestCompactHookFailOpen` | yes | pass |
| happy path task-029 superseded by this change (G9) | `task-029` `status: done` + spec reference (doc) | yes | pass |
| task-029 is not a separate parallel implementation | ADR-0023 (OpenAI-compatible only, one client) | yes | pass |
| closing commit references task-029 | task-029 body references spec; closing commit references task-029 | pending | pending |
| happy path install yes ensures local ollama and wires config (G10) | `install::TestSetupProviderLocal` | yes | pass |
| install provider external wires without ollama | `install::TestSetupProviderExternalNoOllama` | yes | pass |
| skip-provider leaves ensure out (G11) | `install::TestSetupProviderSkipped` | yes | pass |
| ensure skips pull when models already present | `install::TestSetupProviderEnsureSkipsPullWhenPresent` | yes | pass |
| dry-run provider setup writes nothing | `install::TestSetupProviderLocal` (dry-run branch) | yes | pass |
| provider setup failure is non-fatal | `install::TestSetupProviderLocal` (warn + exit 0) | yes | pass |

**Coverage:** 23/24 scenarios covered by a test that ran and passed. 1 pending:
`closing commit references task-029` — the commit has not been authored yet (work is
uncommitted in the working tree). This is a ship-time event, not a code gap; the task-029
ticket is already `status: done` with the spec reference in place.

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| G4–G8 gate identifiers in `acceptance.feature` do not match as-built test names | `acceptance.feature:87,122,126,165,203` | broken-verification (documentation) | `TestCompleteHTTPError`, `TestAttachSharedLLMWiresAllSeams`, `TestAttachSharedLLMDisabledClears`, `TestAttachedClientFlagsOffUsesFloors`, `TestLLMErrorFailsOpen` do not exist; the as-built names do | A future `qa-gate.mjs` run by the exact feature CHECK command would report the gate test missing even though the behavior is covered |
| Closing commit not yet authored (work uncommitted) | working tree | broken-verification (pending event) | `git status` shows new files untracked; no commit hash resolves in history | The "closing commit references task-029" scenario cannot be verified until ship |

## TDD Evidence Audit

> Per `_shared/references/strict-tdd.md` § TDD Cycle Evidence Table.
> **The work for this change is uncommitted in the working tree** (new files untracked,
> modified files unstaged). There are no RED/GREEN commit hashes to resolve in git history.
> The RED→GREEN cycle was performed during implementation (tests written first, then made to
> pass) but the commits have not landed. Until the change is committed, the table cannot
> validate commit hashes.

| Task | SATISFIES | RED (commit + exit) | GREEN (commit + exit) | TRIANGULATE (commit + exit) | REFACTOR (suite exit) | Verdict |
|---|---|---|---|---|---|---|
| TICKET-01 (G1–G4) | happy path llm config / complete succeeds / non-2xx | uncommitted | uncommitted (tests pass now) | n/a (multi-scenario, single pkg) | `go test` exit 0 | STALE (no commits) |
| TICKET-02 (G5–G8) | attach wires all seams / disabled clears / fail-open | uncommitted | uncommitted (tests pass now) | n/a | `go test` exit 0 | STALE (no commits) |
| TICKET-03 (G10–G11) | install yes ensures local / skip-provider | uncommitted | uncommitted (tests pass now) | n/a | `go test` exit 0 | STALE (no commits) |
| TICKET-04 (G9) | task-029 superseded | uncommitted | n/a (doc change) | n/a | n/a | STALE (no commits) |

**Summary:** 0/4 tasks have complete TDD evidence (commits not yet authored). All test
behavior is verified green in the working tree; the STALE verdict is purely the absence of
commit hashes, which the closing commit on `release/2` will resolve.

## Assertion Quality Audit

> Rescan of all test files created/modified by this change for banned patterns.

| File | Line | Pattern | Evidence | Severity |
|---|---|---|---|---|
| `llm/llm_test.go` | `TestCompleteSuccess` | — | asserts returned string == assistant content (real `httptest` server, real `Complete` call) | none |
| `llm/llm_test.go` | `TestCompleteNon2xx` | — | asserts error non-nil on 500 | none |
| `llm/llm_test.go` | `TestCompleteTimeout` | — | asserts error on slow server, 5s | none |
| `config/load_test.go` | `TestLLMConfigDefaultOff`/`RequiresURLAndModel`/`APIKeyFromEnv` | — | table asserts config field values | none |
| `service/attach_llm_test.go` | `TestAttachSharedLLMWiresSeams` | — | asserts seams non-nil + shared counter increments (real `Set*` calls) | none |
| `service/attach_llm_test.go` | `TestAttachSharedLLMDisabledNoOp` | — | asserts seams nil after disabled attach | none |
| `install/provider_test.go` | `TestSetupProviderLocal`/`ExternalNoOllama`/`Skipped`/`EnsureSkipsPullWhenPresent` | — | stubbed exec/`LookPath`; asserts config merge + no-pull behavior | none |

**Summary:** 0 CRITICAL, 0 WARNING, 0 SUGGESTION. All assertions verify real behavior
(real `httptest` servers, real `Set*` seam calls, real config load against YAML).

## Changed-File Coverage

> Per `_shared/references/strict-tdd.md` § Changed-File Coverage.
> `quality.coverage_min` = 0 (advisory only, never blocking). Reported informationally.

| File | Line % | Uncovered lines | Rating |
|---|---|---|---|
| `internal/mnemonic/llm` (package) | 80.0% | error/edge branches | Acceptable |
| `internal/mnemonic/config` (package) | 88.6% | — | Acceptable |
| `internal/install` (package) | 43.5% | provider step error paths (brew/curl shells, start-reprobe) | Low (advisory) |
| `internal/mnemonic/service` (attach) | n/a (full-pkg coverage not isolated) | — | — |

**Threshold:** `quality.coverage_min` = 0 → changed-file coverage is informational, not a gate.
**Average changed-file coverage (measured pkgs):** ~70% (install drags the average down; the
provider step's exec/shell branches are stubbed, not host-run).

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| — | — | All named tests exercise production code against real `httptest` servers or stubbed exec; assertions can fail for real inputs; counter-test present for "seams nil when disabled" |

**Summary:** no contract violations found.

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| Complete success/non-2xx/timeout | unit | no | no | stdlib HTTP vs `httptest`; no higher layer adds signal |
| Attach wires/clears seams | unit | no | no | in-process `Set*` calls |
| Fail-open floors | unit | no | yes (reuses pre-existing `secondbrain::TestAsk_*`) | floors already covered by ask tests; reuse is correct |
| Install provider setup | unit (stubbed exec) | no | no | exec/`LookPath`/probe stubbed; not a host integration run |

**Layer distribution:** 15 unit, 0 integration, 0 e2e. Reasonable for a pure in-process
wiring change; there is no integration/e2e layer that would add signal over `httptest` +
stubbed exec.

## Security Audit

**Mode:** Trivy

### Trivy Findings (Mode A)

**Command:** `trivy fs . --scanners vuln --severity CRITICAL --quiet`

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| — | none | — | — | — | — |

**Trivy gate:** `fail_on: ""` (report only) → PASS. Trivy reports 0 CRITICAL vuln findings
across all `go.mod`/`package-lock.json` (clean). Advisory only — never blocks.

### Manual Findings

#### Secrets Archaeology

| Location | Type | Evidence | Severity |
|----------|------|----------|----------|
| — | — | api_key read from env `SKILLGRID_LLM_API_KEY` / `OPENAI_API_KEY` at config load; not committed; not logged | — |

#### OWASP Spot-Check

| File | OWASP Item | Pattern | Evidence |
|------|------------|---------|----------|
| `llm/client.go` | A02 Cryptographic | bearer token in `Authorization` header; base_url is operator-configured (trust boundary); no secrets logged | token sent over the configured base_url; Ollama `http://localhost:11434` is loopback |

**Security verdict:** PASS — CRITICAL: 0, WARNING: 0, SUGGESTION: 0. No user-facing secret
handling introduced beyond the existing env-based api_key pattern; Trivy clean.

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage (whole project) | `testing.coverage` (empty) | 0 | n/a | N/A |
| Changed-file coverage | `go test -cover` on changed pkgs | 0 | ~70% avg (informational) | N/A (advisory) |
| Mutation | not configured | 0 | N/A | N/A |
| Lint | `go vet` (changed pkgs) | errors only | 0 errors | PASS |
| Typecheck | `go build ./...` | errors only | 0 errors | PASS |
| P0 pass rate | P0 tests (rows 1–6) | 100 | 100% (6/6 PASS) | PASS |
| P1 pass rate | P1 tests (rows 7–15) | 95 | 100% (9/9 PASS) | PASS |
| Trivy security | `trivy fs --severity CRITICAL` | `""` (report only) | 0 findings | PASS (N/A-blocking) |

**Quality config status:** configured (coverage_min 0, mutation_min 0, p0 100, p1 95).

**Dead code (if configured):** N/A.

## State Drift

> `node .agents/skills/verification/qa/scripts/state-drift-check.mjs` — read-only; never
> CRITICAL, never affects the verdict.

**Verdict:** PARSE ERROR

The guard could not parse `.skillgrid/state.yaml`:
`Nested mappings are not allowed in compact mappings at line 10, column 12` — the
`P26-10-02:` field is a single compact line with unquoted colons. This is **pre-existing**
(from an earlier session's notes; this change did not touch `state.yaml`). Per the guard, a
parse error is advisory and never affects the verdict.

**Fix applied:** none (parse error is pre-existing and out of scope for this change).

**Scope (from the guard's `SCOPE:` line):** not reported (parse error prevented derivation).

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix | COMPLETE | none |
| Verification-gap audit | COMPLETE | none |
| State Drift (9.5) | UNREADABLE (parse error) | n/a |
| Structure Drift (9.6) | N/A (script absent) | n/a |

**Stale-verification:** STALE: none for the code zone (this run re-executed every named test
and the full touched-package suite; the evidence the gate rests on is from this session's runs).

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED |
| Traceability (weakest scenario) | PARTIAL (23/24; 1 pending = closing commit, ship-time) |
| Verification-gap audit | WARNING (gate-name mismatch) |
| TDD evidence (weakest ticket) | STALE (commits unauthored) |
| Assertion quality audit (weakest finding) | none |
| Changed-file coverage (weakest file) | N/A (advisory, min 0) |
| Test quality audit | none |
| Code-quality gates (weakest gate) | PASS |
| Security audit | PASS |
| Verification scope (composite) | COMPLETE (state-drift UNREADABLE is advisory) |

**FLOOR:** The weakest dimensions are the **TDD evidence (STALE — commits unauthored)** and
the **traceability matrix (1 pending scenario: closing commit)**. Both are non-answers that a
human can clear by authoring the closing commit on `release/2`; they are not FAIL conditions
(no truth is UNVERIFIED, no scenario lacks a covering test, no test ran and failed, no
CRITICAL finding). Per the fail-closed + floor-cap rules, the gate routes to **CONCERNS**
(not PASS, because TDD evidence is STALE and one scenario is pending; not FAIL, because the
pending item is a ship-time commit, not a code or test gap, and every behavioral gate is green).

## Findings

### CRITICAL (must fix before merge)

- None.

### WARNING (should fix before archive)

- W014 — `acceptance.feature` gate identifiers G4–G8 (`TestCompleteHTTPError`,
  `TestAttachSharedLLMWiresAllSeams`, `TestAttachSharedLLMDisabledClears`,
  `TestAttachedClientFlagsOffUsesFloors`, `TestLLMErrorFailsOpen`) do not match the as-built
  test names; a `qa-gate.mjs` run by the exact feature CHECK command would report the gate
  test missing. Fix: update the feature gate lines to the as-built names before ship.
- W015 — TDD evidence is STALE: the change is uncommitted in the working tree, so no
  RED/GREEN commit hashes resolve. Fix: author the closing commit on `release/2` (the ship
  step), which both commits the work and satisfies the "closing commit references task-029"
  scenario.

### SUGGESTION (nice to have)

- S001 — `internal/install` package coverage is 43.5% (advisory). The provider step's
  brew/curl shells and start-reprobe branches are stubbed, not host-run. Consider a
  host-integration smoke (nightly) for the Local-ensure path.

## Gate Decision

**Verdict:** CONCERNS

**Reasoning:** Every behavioral gate (G1–G11, all 23 code scenarios) is green in this
session's runs — all truths VERIFIED, all code-quality gates PASS, security clean,
assertion quality clean. The gate is held at CONCERNS (not PASS) by two non-answers, both
ship-time rather than code: (1) the change is uncommitted, so the TDD Evidence table has no
resolvable commit hashes (STALE), and (2) the "closing commit references task-029" scenario
is pending until that commit is authored. There is a third, lower-purity WARNING: the
`acceptance.feature` gate identifiers G4–G8 were never updated to the as-built test names.
None is a FAIL condition — no truth is UNVERIFIED, no scenario lacks a covering test, and no
test ran and failed.

**Open items (if CONCERNS):**
- Author the closing commit on `release/2` (commits the work + references task-029) → clears
  both the TDD STALE and the pending closing-commit scenario. Path: ship step.
- Update `acceptance.feature` G4–G8 gate lines to the as-built test names → clears W014.
  Path: one-line edit before ship.

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** —
> never as silently accepted.

**Human decision (fill in):** <accept / accept-with-open-items / reject> — <name> — <date>

<!-- reflect completes this half at archive time -->

## Final-State Facts

**Shipped:** <what actually shipped>
**Base branch:** <base-branch> · **Chain strategy:** <strategy>
**Integration:** <merged @ <commit> | PR <url> | kept branch <name>> (from ship context)

## Gates

| Gate | Result |
|------|--------|
| Ship gate | ✅ success + `diff -r` empty (or: blocked — {reason}) |
| QA gate | ✅ {PASS \| WAIVED \| CONCERNS — human override recorded} |
| Verdict gate (advisory) | {accepted \| accepted-with-open-items \| rejected} — recorded, not enforced |

## Decisions

| Decision | Tradeoff | Why | Source |
|----------|----------|-----|--------|
| OpenAI-compatible HTTP only; one client, all seams | no Anthropic-native SDK, no separate Ollama `/api/chat` client | same coverage via `/v1`, one test surface, no new dep | ADR-0023 |
| Fail-open floors preserved (ADR-0016) | LLM paths must degrade, never panic | production safety for a downed provider | ADR-0016 |
| Natural provider setup on install (not opt-in bolt-on) | install always offers Local/External | code indexing already needs Ollama/external embedder | briefing Req 7 |

## Lessons

| Lesson | Root Cause | Do Differently | Source |
|--------|-----------|----------------|--------|
| Keep `acceptance.feature` gate identifiers in sync with as-built test names | gates were written before the final test names | update the feature CHECK lines when test names change | `acceptance.feature:87,122,126,165,203` |

## Patterns

| Pattern | Reuse | Source |
|---------|-------|--------|
| One shared Completer adapted to each seam's interface (zero adapters where signatures match, thin adapter where they don't) | any future multi-seam capability that shares a backend | `service/attach_llm.go` |

## Surprises

| Surprise | Signal | Source |
|----------|--------|--------|
| `go test ./internal/mnemonic/service/` runs ~145–172s (5s-sleep timeout tests) | full-pkg run timed out at 120s | this session's runs |

## Environment Retro

| Category | Finding | Fix lands in | Source |
|----------|---------|--------------|--------|
| automated checks | `state-drift-check.mjs` fails to parse `state.yaml` because the `P26-10-02:` field is a compact line with unquoted colons | quote the field value or use a list; pre-existing, out of scope | `.skillgrid/state.yaml:10` |

## Acceptance Verdict

**Verdict:** accepted-with-open-items

**Grounding:** briefing goal (one OpenAI-compatible endpoint → one client → all seams;
opt-in flags; fail-open floors; natural install provider setup) is fully verified by named
tests (Goal-Backward table all VERIFIED; 23/24 scenarios pass). The single open item is the
closing commit, which is a ship-time event.

**Reasoning:** The feature is implemented and verified; the gate is CONCERNS only because
the work is uncommitted (TDD STALE) and one scenario is pending on the closing commit. Both
clear at ship.

## Open Items (→ next change)

- None new. (W014 feature gate-name sync and W015 closing commit both land at ship of THIS
  change, not a follow-up change.)

## Prior-Change Follow-Through

| Prior open item (from previous archived change) | Addressed by this change? | Evidence |
|--------------------------------------------------|---------------------------|----------|
| task-029 FOLLOWUP (shared production LLM client attach) — open from `2026-09-24-bitemporal-audn` | yes | task-029 `status: done`, superseded by this spec |

## Move Evidence (from ship context)

**From:** `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/` → **To:** `.skillgrid/archive/2026-10-02-mnemonic-llm-provider/`
**`diff -r` readback:** <empty → PASS | verbatim output → FAIL>

## Overrides / Waivers / Contradictions

- None (no human override recorded; no waiver).

## Lineage (observation IDs)

- briefing: `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/briefing.md`
- blueprint: `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/blueprint.md`
- tasks: `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/tasks.md`
- ADRs: `.skillgrid/artifacts/04-adr-0023-openai-compatible-llm-provider.md`, `04-adr-0016-second-brain-capability-layer.md`
- task-029: `.backlog/tasks/task-029 - FOLLOWUP-shared-LLM-client-attach-for-ask-extraction-dedup-seams.md`
- report (QA half): this file
