# QA Report — 2026-10-03-local-ollama-models

> QA half (Test Plan → Gate Decision → Human Override). Retro half (from `## Final-State Facts`) left empty for `skillgrid:reflect` at archive time.

**Change:** 2026-10-03-local-ollama-models
**Tier:** T2 (Beta) — from `blueprint.md` header `Tier: T2`; matches `rules.tiers.default: T2`. Applied floor = L2+L3 (T2).
**Effective floor (change-classification):** Standard change, multi-file behavior (catalog, floor gate, config merge, runtime default). Classification floor is met and exceeded (L2 tests + L3 named-test verification). Tier did not lower the floor.
**Verdict:** CONCERNS
**Finding counts:** 1 WARNING · 1 SUGGESTION · 0 CRITICAL

---

## Test Plan

**Layers available:** `[unit, integration]` (from `testing.layers`). This change is pure-Go library code with no server-dependent happy path in the acceptance scenarios; the acceptance gates are all `go test` at the unit/integration seam with package-var stubs (`tagsServer`, `ollamaVersion`, `runCmd`, `smokeProbe`). No E2E layer configured → highest available layer = unit (in-process, real `provider.go` code exercised through package-var seams).

### Scenario → Test matrix (traceability)

`acceptance.feature` has **6 scenarios** across 4 requirements (counted, not guessed).

| # | Scenario | Covering test | Layer | Cadence | P0/P1 | Status |
|---|----------|---------------|-------|---------|-------|--------|
| S1 | happy path catalog pull list | `TestCatalogPullList` | unit | pr | P0 | COMPLIANT |
| S2 | happy path version floor gates heavy models | `TestFloorGatesHeavyModels` | unit | pr | P0 | COMPLIANT |
| S3 | happy path version at or above floor pulls all | `TestAtFloorPullsAll` | unit | pr | P0 | COMPLIANT |
| S4 | happy path config merge writes new live-chat + embed models | `TestConfigMergeWritesNewModels` | unit | pr | P0 | COMPLIANT |
| S5 | happy path ollama default model | `TestDefaultOllamaModelIsEmbeddinggemma` | unit | pr | P1 | COMPLIANT |
| S6 | happy path glm vision tools excluded | `TestCatalogPullList` (asserts no glm in pull list) | unit | pr | P0 | COMPLIANT |

**Totals:** scenarios: 6/6 COMPLIANT · 0 PARTIAL · 0 FAILING · 0 UNTESTED · 0 pending.
**Gates G1–G6 all `EVIDENCE: PASS`** (recorded in `acceptance.feature`, commit `17d43abb`).

### Risk-ranked tests (behaviors not in the acceptance scenarios)

| Test | Behavior | Layer | Cadence | P0/P1 | Status |
|------|----------|-------|---------|-------|--------|
| `TestSmokeProbeNonFatal` | smokeProbe failing never aborts install (ADR-0016) + config merge still runs | unit | pr | P1 | COMPLIANT (non-acceptance edge/failure variant) |
| `TestOllamaVersionAtLeast` | pure version compare across 11 inputs (at/below/above/rc/empty/malformed) | unit | pr | P0 | COMPLIANT (T1 triangulation) |
| `TestSetupProviderLocal` (updated) | full local provider step pulls six-model catalog, floor seam at/above | unit | pr | P0 | COMPLIANT (legacy, updated in TICKET-02) |
| `TestSetupProviderEnsureSkipsPullWhenPresent` (updated) | ensure skips pulls when all 6 present | unit | pr | P1 | COMPLIANT (legacy, updated in TICKET-02) |

### Edge-Case Matrix (from blueprint tasks + scenario edge/failure variants)

| Edge / failure variant | Covered by | Status |
|------------------------|-----------|--------|
| version below floor → heavy gated + warning | `TestFloorGatesHeavyModels` | covered |
| version at/above floor → all pulled | `TestAtFloorPullsAll`, `TestCatalogPullList` | covered |
| empty/malformed version → treated as unknown → heavy gated | `TestOllamaVersionAtLeast` (`""`, `"not-a-version"`, `"v0.35.1"`) | covered |
| model already present → skip pull | `TestSetupProviderEnsureSkipsPullWhenPresent` | covered |
| pull failure → non-fatal (warn, continue) | `TestSmokeProbeNonFatal` (runCmd seam) | covered |
| smokeProbe failure → non-fatal, config merge still runs | `TestSmokeProbeNonFatal` | covered |
| unrelated operator keys (profile, dimension) preserved on merge | `TestConfigMergeWritesNewModels` | covered |

### Out of Scope (explicit)

- **Per-kind HTTP smoke calls (briefing req 3)** — chat/embed/systemone probes hitting the live Ollama, recording the embedder dimension, and emitting a mismatch warning. See Finding W1.
- **`mnemonic.ollama.models` list (briefing req 5)** — the six tags + kinds written to the home config. See Finding W1.
- **ONNX / `nomic-embed-code` non-Ollama default** — preserved, not changed by this diff (briefing states it stays).
- **cmd/skillgrid init/arch/eval + internal/mnemonic/project resolve_test.go** — pre-existing failures, NOT in this diff (confirmed unrelated; see Scope).

---

## Goal-Backward Verification (Step 3)

Force stance: assume the goal was NOT achieved. The executed goal (per `blueprint.md` `## Must-Haves`) is the six-model catalog pull, floor gate, config wiring, runtime default, and typo exclusion. Walked all four levels; each truth is `VERIFIED` by a named test that ran and passed.

| Truth (blueprint Must-Have) | Level | Evidence (named test, exit) | Status |
|-----------------------------|-------|------------------------------|--------|
| install pulls exactly the six catalog tags, no other | Truth+KeyLink+DataFlow | `TestCatalogPullList` PASS (pulledSet = 6 tags) | VERIFIED |
| llm.model=llama3.2:1b, embedder.model=embeddinggemma:300m (provider ollama) | Truth+Artifact | `TestConfigMergeWritesNewModels` PASS | VERIFIED |
| below floor → clef-flash+tev1 not pulled, warning names required version, install succeeds | Truth+DataFlow | `TestFloorGatesHeavyModels` PASS | VERIFIED |
| at/above floor → all six pulled | Truth+DataFlow | `TestAtFloorPullsAll` PASS | VERIFIED |
| DefaultOllamaModel == embeddinggemma:300m, embedder builds against it | Truth+Artifact | `TestDefaultOllamaModelIsEmbeddinggemma` PASS | VERIFIED |
| glm:vision-tools appears nowhere (pull/list/warning) | Truth+DataFlow | `TestCatalogPullList` PASS (glm absent) | VERIFIED |
| version parse 0.35.1 at-or-above / 0.35.0 below / malformed unknown | Truth | `TestOllamaVersionAtLeast` PASS (11 cases) | VERIFIED |

**Artifact presence (not behavior):** `localModels()`, `heavyModels()`, `ollamaVersion` seam, `pullMissingModels` gate, `mergeHomeProviderConfig`, `smokeProbe` package-var all present in `provider.go`; `DefaultOllamaModel` flipped in `ollama.go`. Wired: `pullMissingModels` iterates `localModels()`, floor gate consults `heavyModels()` + `ollamaVersion`, `smokeProbe` invoked post-pull.

**PRESENT_BEHAVIOR_UNVERIFIED (routes to human):** `smokeProbe` is a no-op placeholder (`provider.go:308-317`, `return true` always). It is wired and non-fatal (the ADR-0016 contract is verified by `TestSmokeProbeNonFatal`), but the *per-kind HTTP probe behavior* the briefing asked for is not implemented — the hook exists and never fails by construction. This is the substance of Finding W1.

**Verdict:** All 7 blueprint truths `VERIFIED`. No `UNVERIFIED`. One `PRESENT_BEHAVIOR_UNVERIFIED` (smoke probe body) → caps the floor at CONCERNS.

---

## Verification-Gap Audit (Step 4)

Question per changed behavior: *"If this behavior broke where it's actually used, would verification fail?"*

| Changed behavior | Would a break be caught? | Classification |
|------------------|--------------------------|----------------|
| catalog pull list = 6 tags | Yes — `TestCatalogPullList` + `TestSetupProviderLocal` assert the exact pulledSet | no gap |
| floor gate on heavy models | Yes — `TestFloorGatesHeavyModels` / `TestAtFloorPullsAll` (seam-driven, opposite inputs = triangulation) | no gap |
| version parse/compare | Yes — `TestOllamaVersionAtLeast` 11 inputs | no gap |
| config merge writes models + preserves keys | Yes — `TestConfigMergeWritesNewModels` | no gap |
| runtime default constant | Yes — `TestDefaultOllamaModelIsEmbeddinggemma` | no gap |
| smoke probe non-fatal contract | Yes — `TestSmokeProbeNonFatal` (stub a failing probe, assert setupProvider returns nil) | no gap (contract verified) |
| **per-kind HTTP smoke probe (briefing req 3)** | **No** — `smokeProbe` is a no-op `return true`; a break in the *intended* probe (which doesn't exist) would not fail any test | **missing-adoption gap → WARNING (W1)** |
| **dimension recording + mismatch warning (briefing req 3)** | **No** — no code records `mnemonic.embedder.dimension` or warns on mismatch; no test | **missing-adoption gap → WARNING (W1)** |
| **`mnemonic.ollama.models` list (briefing req 5)** | **No** — no code writes the list; no test | **missing-adoption gap → WARNING (W1)** |

**Missing-oracle check:** every acceptance scenario has a `G<n>` with `CHECK`+`EXPECT`+`EVIDENCE`. No happy-path scenario is oracle-less. ✓

**Summary:** 6 of 7 changed behaviors have covering tests that would catch a regression. The 3 briefing-scope behaviors (per-kind smoke, dimension recording, models list) were descoped in the blueprint to a no-op hook / omitted and have no tests — but they also have no acceptance scenarios, so they are not traceability gaps. They are a *scope-reconciliation* gap (W1).

---

## TDD Evidence Audit (Step 6)

`testing.tdd: false` (Standard mode) → strict-TDD branch machinery off. Basic TDD (failing test before code) is the baseline, but the strict evidence-table cells (separable RED commit hashes) are not machine-gated this change.

| Ticket | RED/GREEN commits | Notes |
|--------|-------------------|-------|
| TICKET-01 | b67500f (feat, impl+test same commit), c64d8fb (doc) | impl+test co-committed; no separable RED |
| TICKET-02 | 442f47ba (impl+tests), 1553b346 (review fixes) | impl+test co-committed |
| TICKET-03 | 38b0bced (one constant + test) | trivial one-way-door flip, human-approved |
| TICKET-04 | 3dc1dff (test-only), 5c15b65 (comment) | test-only gate |
| TICKET-05 | 17d43abb (evidence) | verification only |

**Summary line:** 0/5 tickets have a *separable* RED commit (impl and test co-committed in the same commit). Because `testing.tdd: false`, this is the established project pattern (see open W002 from the prior change — same pattern) and is recorded as a SUGGESTION-class note, not a `MISSING_RED` CRITICAL. The tests are genuine (they assert concrete pulledSet/config/constant values, not tautologies) and would fail if the code regressed.

**SATISFIES check:** blueprint task `SATISFIES` names match `acceptance.feature` scenario names (version floor, all-models, catalog pull, config merge, runtime default). ✓

---

## Assertion Quality Audit (Step 6a)

Re-scanned `provider_test.go` (G1–G6 + legacy) and `ollama_test.go` (default model).

| File | Pattern | Evidence | Severity |
|------|---------|----------|----------|
| provider_test.go | none of the banned patterns | pulledSet / config-map / constant assertions all bind to a concrete expected value; `TestSmokeProbeNonFatal` calls `setupProvider` then asserts it returns nil (real call + real assertion); no tautologies, no ghost loops, no smoke-only | — |
| ollama_test.go | none | `TestDefaultOllamaModelIsEmbeddinggemma` asserts `DefaultOllamaModel != want` → fail and `o.Model() != want` → fail; value assertion, not type-only | — |

Empty-collection-without-companion check: `TestCatalogPullList` asserts the pulledSet equals exactly 6 tags (non-empty) — the non-empty sibling is the same test. ✓
No CRITICAL-severity assertion patterns.

---

## Changed-File Coverage (Step 6b)

`testing.coverage: ""` → **Coverage: n/a — no coverage tool detected.** Not a failure. (Per config, coverage is advisory-only at `quality.coverage_min: 0`.)

---

## Test Quality Audit (Step 7)

| Contract | Result |
|----------|--------|
| Real code | Yes — tests call `setupProvider`/`pullMissingModels`/`mergeHomeProviderConfig`/`ollamaVersionAtLeast`/`DefaultOllamaModel` (production code), not the mocks |
| No vacuous truths | Yes — every assertion binds to a concrete expected set/value |
| No pass-always | Yes — remove the catalog loop and `TestCatalogPullList` fails; remove the floor gate and `TestFloorGatesHeavyModels` fails |
| Test claimed path | Yes — each test name matches the path it walks |
| Complete mocks | Yes — `tagsServer` returns a realistic `/api/tags` JSON shape; `ollamaVersion` seam returns a real version string |
| Counter-test (does-not-X) | Yes — glm exclusion (S6) is a negative assertion; floor-below is a negative assertion (heavy NOT pulled) |
| P0 triangulation | Yes — floor gate has opposite inputs (below vs at/above) across `TestFloorGatesHeavyModels` / `TestAtFloorPullsAll`; version compare has 11 inputs |

No P0 behavior is single-input. ✓

---

## Security Audit (Step 8)

Mode A (Trivy configured: `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`, `fail_on: ""`).

- **Trivy:** `fail_on: ""` → findings are **advisory only**, never block the gate. (No new dependency added — ADR-0023 stdlib only — so no new CVE surface in this diff; `go.mod` unchanged.)
- **Secrets:** no keys/tokens/PII introduced. The change hardcodes Ollama *model tags* (not secrets) and a localhost base URL.
- **OWASP spot-check (changed files):** `provider.go` handles an HTTP GET to `{base}/api/tags` and `/api/version` with a 5s timeout and parses JSON — no user-controlled input reaches the parse (base URL is operator-configured, model tags are static constants). No injection/CORS/SSRF surface newly introduced beyond the pre-existing `ollamaBaseURL` operator config. `ollama.go` is a constant flip.

**Classification:** no CRITICAL, no WARNING security findings from this diff. (Advisory Trivy run not blocking; `fail_on` empty.)

---

## Code Quality Gate (Step 9)

| Gate | Command | Threshold | Result |
|------|---------|-----------|--------|
| Build | `go build ./...` | exit 0 | PASS (exit 0) |
| Lint | `commands.lint` = "" (go vet inline) | errors only | N/A command → `go vet` run: exit 0, 0 errors |
| Typecheck | `commands.typecheck` = "" (Go compile) | errors only | N/A (build = typecheck) → PASS |
| Coverage | `testing.coverage` = "" | `coverage_min: 0` | N/A (threshold 0) |
| Mutation | `testing.mutation` = "" | `mutation_min: 0` | N/A (no tool, threshold 0) |
| P0 pass rate | G1,G2,G3,G4,G6 + TestOllamaVersionAtLeast + TestSetupProviderLocal | `p0_pass_rate: 100` | PASS (all ran, all pass → 100%) |
| P1 pass rate | G5 + TestSmokeProbeNonFatal + TestSetupProviderEnsureSkipsPullWhenPresent | `p1_pass_rate: 95` | PASS (all ran, all pass → 100%) |
| Trivy | `trivy fs . ...` | `fail_on: ""` | N/A (report-only, advisory) |

**P0/P1 evidence:** `go test ./internal/install/... ./internal/mnemonic/embedder/... -count=1` → both packages `ok`. Every P0/P1 test named in the plan actually ran (not skipped/filtered).

---

## Deterministic Gate Checks (Step 9.5)

`node .agents/skills/verification/qa/scripts/qa-gate.mjs .` →

```
state_drift:  exit 2  (parse error)
ship_drift:   exit 0  (skipped: no --anticipated paths)
size_budget:  exit 2  (script not found)
scopes.composite: COMPLETE
```

- **state_drift (exit 2):** `state.yaml` does not parse — line 10 `P26-10-06: 2026-10-06: 2026-10-02-mnemonic-llm-provider SHIPPED ...` is a **nested mapping in a compact mapping** (a `key:` inside the value). This is a **pre-existing** malformed key (a date-prefixed log key that itself contains a `:` + space), not introduced by this change. → SUGGESTION (S1). Never CRITICAL, never affects the four-state verdict.
- **ship_drift:** skipped (no `--anticipated` paths). Note in report.
- **size_budget:** "script not found" — pre-existing (W003/W007: qa-gate.mjs looks in `.agents/skills/qa/scripts` but helpers live under `verification/qa/scripts`). Advisory.
- **Review evidence:** `review.md` is absent (this is the first QA pass, not re-verification mode) → no staleness note.

**Scope (worst-scope-wins):** `COMPLETE`.

---

## Verification Scope + Staleness (Step 9.7)

| Derivation | SCOPE |
|------------|-------|
| Traceability matrix (all 6 scenarios read from `acceptance.feature`) | `COMPLETE` |
| Verification-gap audit (all changed behaviors enumerated from the diff: 5 files) | `COMPLETE` |
| State Drift (9.5) | `COMPLETE` (composite from qa-gate.mjs) |
| Structure Drift (9.5) | `COMPLETE` |
| **Composite** | `COMPLETE` |

**Staleness:**
- `STALE: none` — this is the first QA pass for the change; the evidence (named tests, `go build`/`go vet`/`go test` run this session, exit codes captured) is current. No code-zone change since the last verification because there was no prior verification.

All scopes `COMPLETE` and no staleness → the fail-closed rule is satisfied (the gate is not pulled off PASS by scope).

---

## Floor (Step 9.8)

The gate is the weakest dimension, never the average.

| Dimension | Weakest value |
|-----------|---------------|
| Goal-backward truths | all `VERIFIED` (best) |
| Scenario compliance | 6/6 `COMPLIANT` (best) |
| Verification-gap | one `WARNING` gap (W1: briefing-scope behaviors descoped) |
| TDD evidence | standard-mode, no separable RED (SUGGESTION-class, not CRITICAL) |
| Assertion quality | best (no banned patterns) |
| Test quality | best (all contracts met) |
| Security | best (no findings, advisory Trivy) |
| Code-quality gates | all PASS or N/A (best) |
| Deterministic checks | SUGGESTION (state.yaml parse, size-budget script path) |
| Verification scope | `COMPLETE` (best) |
| Staleness | `none` (best) |

Weakest dimension = the `WARNING` verification-gap (W1) + `PRESENT_BEHAVIOR_UNVERIFIED` smoke-probe body. No dimension is at a FAIL value.

**FLOOR: CONCERNS** (a `WARNING` gap + a `PRESENT_BEHAVIOR_UNVERIFIED` truth cap the floor at CONCERNS; no FAIL-valued dimension exists).

---

## Findings

### WARNING

**W1 — Scope-reconciliation gap: briefing requirements 3 & 5 descoped to a no-op / omitted in the executed blueprint.**
- **What:** `briefing.md` req 3 asks for *real* per-kind HTTP smoke calls (chat/embed/systemone probes) that record the embedder dimension into `mnemonic.embedder.dimension` and emit a mismatch warning; req 5 asks for a `mnemonic.ollama.models` list (six tags + kinds) in the home config. The executed `blueprint.md` scoped `smokeProbe` to a **non-fatal no-op placeholder** (`provider.go:308-317`, `return true` always) and **omitted** the dimension-recording and models-list work. `acceptance.feature` was narrowed to match the blueprint (no scenario covers req 3 or 5), so the gap is **invisible to the acceptance tests**.
- **Evidence:** `provider.go:308-317` (no-op `smokeProbe`); `grep "mnemonic.ollama.models" skillgrid-cli/` → no hit in code (only in briefing/tasks docs); no code records `mnemonic.embedder.dimension` for the Ollama path.
- **Why it's a WARNING and not a CRITICAL:** the as-built code **exactly matches the blueprint**, and all 7 blueprint truths are `VERIFIED` by named passing tests. This is a *scope* gap (intent vs. what was sliced), not a code defect — nothing in the shipped code is broken. It is a non-answer the human can clear by ratifying the scope (defer reqs 3/5 to a follow-up ticket) or by fixing now (add the probe body + dimension recording + models list, with tests).
- **Path to resolution:** (a) human defers reqs 3/5 to a follow-up ticket (record the deferral), or (b) implement the probe body + dimension recording + `mnemonic.ollama.models` list, add tests, re-run this gate.

### SUGGESTION

**S1 — `state.yaml` line 10 is malformed YAML (pre-existing), breaks the state-drift check.**
- `P26-10-06: 2026-10-06: 2026-10-02-mnemonic-llm-provider SHIPPED ...` — a `key:` inside a compact-mapping value is not allowed. `state-drift-check.mjs` exits 2 (parse error) on it.
- **Not introduced by this change** (the line predates it). **Fix:** quote the value or restructure the log key so the value has no unquoted `: `. Advisory — never CRITICAL, never affects the four-state verdict.
- Related pre-existing: `size_budget` "script not found" (W003/W007: qa-gate.mjs path mismatch).

### CRITICAL

(None.)

---

## Gate Decision

**Verdict: CONCERNS**

Four-state criteria check (thresholds from `config.yaml`: p0=100, p1=95, coverage_min=0, mutation_min=0, trivy fail_on=""):

1. All truths `VERIFIED` ✓
2. All scenarios covered by a test that ran and passed ✓ (6/6 COMPLIANT)
3. No CRITICAL findings from any audit ✓ (0 CRITICAL)
4. No `MISSING_RED` ✓ (standard-mode, no separable RED — SUGGESTION-class, not CRITICAL)
5. All code-quality gates PASS or N/A ✓
6. P0 pass rate 100% ≥ 100 ✓
7. P1 pass rate 100% ≥ 95 ✓
8. Coverage N/A (threshold 0) ✓
9. Mutation N/A (no tool, threshold 0) ✓
10. No Trivy finding ≥ `fail_on` (empty → advisory) ✓
11. Every verification-scope derivation `COMPLETE` ✓

All **hard** gates PASS. But there is **at least one WARNING** (W1) **and one `PRESENT_BEHAVIOR_UNVERIFIED`** truth (the smoke-probe body) → **CONCERNS** per the four-state table. The floor (`FLOOR: CONCERNS`) caps the verdict; the gate is not higher than its floor.

**This is not a FAIL** — no truth is `UNVERIFIED`, no scenario is `UNTESTED`/`FAILING`, no test ran and failed, no CRITICAL security/code-quality finding, no `MISSING_RED`. The gap is a named, scope-level non-answer the human can clear.

---

## Human Override

_Not yet exercised._ The machine verdict is **CONCERNS**. Two paths for the human:

- **Defer W1 (recommended for this change):** ratify that the *as-built* scope = the blueprint (catalog + floor + config merge + runtime default + typo exclusion), and file req 3 (per-kind smoke + dimension recording + mismatch warning) and req 5 (`mnemonic.ollama.models` list) as a **follow-up ticket**. Record the deferral. Gate → proceed to review (PASS-equivalent on the as-built scope).
- **Fix W1 now:** implement the `smokeProbe` body (per-kind HTTP probe), dimension recording into `mnemonic.embedder.dimension`, the mismatch warning, and the `mnemonic.ollama.models` list; add tests; re-run Steps 3–9; re-render.

S1 (state.yaml parse) is advisory and pre-existing — optionally fix the YAML quoting, or leave it to a maintenance pass.

---

## Re-run — 2026-10-06 (W1 resolved via human override: "Fix W1 now")

The human chose **Fix W1 now** rather than deferring. The three descoped behaviors were implemented and tested; the acceptance feature gained the missing scenarios. This section supersedes the first-pass `CONCERNS` verdict.

### What changed

- `smokeProbe` (was a no-op `return true`) now **dispatches per role** to the endpoint each role speaks:
  - `chat` → `POST /v1/chat/completions` (pass on non-empty first choice)
  - `embed` → `POST /api/embed` (pass on vector length > 0; **records the length**)
  - `systemone` → `POST /v1/systemone` (pass on a present `answer` field)
  - `research` → no-op pass (presence after pull is the check)
  - Still non-fatal (ADR-0016): a failing probe warns and the install returns nil.
- **Embed dimension recording (req 3):** the successful embed smoke records the vector length into `lastEmbedDimension`; `setupProviderLocal` reads the pre-merge home dimension and, on mismatch, warns with **both lengths** and that search is stale until reindex. The embedder model still switches (fail-open). The recorded length is written to `mnemonic.embedder.dimension`.
- **`mnemonic.ollama.models` list (req 5):** the home merge now writes `mnemonic.ollama.models` = six entries `{name, kind}` (kind ∈ `chat`/`embed`/`systemone`), `glm:vision-tools` excluded. Only the local provider publishes the list.

### New / updated tests (all PASS)

| Gate | Test | Covers |
|------|------|--------|
| G7 | `TestSmokeProbeDispatchByRole` | per-role dispatch: chat/embed/systemone pass on their endpoint; research no-op; embed 500 + unreachable fail the probe |
| G8 | `TestEmbedSmokeRecordsDimension` | smoke vector length 384 written to `mnemonic.embedder.dimension`, model = embeddinggemma:300m |
| G9 | `TestEmbedDimensionMismatchWarns` | fixture 768 vs smoke 512 → warning names both lengths, model still switches, dimension = 512 |
| G10 | `TestHomeMergeWritesOllamaModelsList` | `mnemonic.ollama.models` has six entries with correct kinds, `glm:vision-tools` absent, `profile` preserved |

`acceptance.feature` gained the three scenarios (smoke dispatch, dimension recording/mismatch, models list) — the gap that was invisible to the acceptance tests is now covered.

### Re-run results

- `go build ./...` → exit 0
- `go vet ./internal/install/... ./internal/mnemonic/embedder/...` → exit 0
- `go test ./internal/install/... ./internal/mnemonic/embedder/... -count=1` → `ok` (both packages)
- `gofmt -l internal/install/` → clean

### Revised verdict

**Verdict: PASS** (W1 resolved). The `PRESENT_BEHAVIOR_UNVERIFIED` smoke-probe body is now `VERIFIED` by `TestSmokeProbeDispatchByRole`; the verification-gap `WARNING` is closed. All hard gates remain PASS (P0 100%, P1 100%).

**Finding counts:** 0 CRITICAL · 0 WARNING (W1 closed) · 1 SUGGESTION (S1 — state.yaml parse, still open, pre-existing, advisory).

S1 is unchanged and remains the only open finding: `state.yaml` line 10 is malformed YAML (nested mapping in a compact mapping) and pre-dates this change. It does not gate this change. Route to `requesting-code-review`.

---

## Final-State Facts

_(left empty for `skillgrid:reflect` at archive time)_
