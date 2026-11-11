# Report — mnemonic-v2 knowledge base (ProjectWiki engine 011 + wiki-autolearning 013)

> Change: `.skillgrid/change-specs/2026-10-31-mnemonic-v2-knowledge-base/`
> Generated: 2026-11-10 (qa); re-rendered 2026-11-11 after the fix-by-code round
> Gate: PASS
>
> Two phases, one file: **qa** writes the QA half (through `## Gate Decision` + `## Human Override`). **reflect** completes the retro half (from `## Final-State Facts` onward) at archive time.

**Effective tier:** T2 (blueprint `Tier: T2`; config `rules.tiers.default: T2`). **Applied floor:** L2+L3 — named-test goal-backward verification + full-suite regression net + live CLI smoke. No tier/floor conflict.

**Gate history:** 2026-11-10 first render **CONCERNS** (W003–W008). The human chose *clear by code* (2026-11-11): W003 (thin/fallback/degraded wiring), W006 (full 007 lint + T7 status), W004 (`acceptance.feature` + TDD table backfilled), W005 (tasks reconciled, blueprint Key Link amended to the shipped 008 design) were fixed in code/spec and re-verified this run. W007/W008 remain open as advisory SUGGESTIONs (precedent: advisory findings with `fail_on` unset/0 do not cap the gate — cf. mcp-split QA PASS with W002 open).

**Oracle:** `blueprint.md` Must-Haves (T1–T8, Artifacts A–C, Key Links as amended 2026-11-11) + `specs/000–013`, traced through `acceptance.feature` (27 scenarios, backfilled 2026-11-11 from those truths).

## Test Plan

### Risk Ranking

Highest risk: (1) the no-op proof (PW-INV-4) silently writing on a clean `projectwiki_generate` — data-loss class for the vault; (2) index/vault divergence (soft-delete, incremental-only-changed, in-degree, Reindex-vs-Index derived-state parity) making retrieval cite stale pages; (3) 013's four new page types being scaffolded but not indexed/validated (whitelist gap). These are tested hardest first, at the lowest layer that covers them (unit/integration; no e2e layer configured).

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Clean generate writes nothing (PW-INV-4, 4-layer proof) | `projectwiki.Generate` | unit | `generate_test.go::TestGenerateProvenNoop` + `TestGenerateLayer1..4BlocksNoop` | P0 | pr | covered |
| 2 | Generate creates a page via degraded floor, idempotently | `projectwiki.Generate` | unit | `TestGenerateCreatesNewPageViaFloor`, `TestGenerateFloorIdempotent` | P0 | pr | covered |
| 3 | Incremental index only-changed; reindex rebuilds identical derived state; removed pages soft-delete | `projectwiki.Index/Reindex` | unit | `TestIndexIncrementalOnlyChanged`, `TestReindexRebuildsProjectWikiPages`, `TestIndexSoftDeletesRemovedPages`, `TestIndexInDegree` | P0 | pr | covered |
| 4 | Stage-1 query ranked cited pages; empty topic returns empty, not error | `projectwiki.Query` | unit | `TestQueryReturnsRankedProjectWikiPages`, `TestQueryNoMatchReturnsEmpty` | P0 | pr | covered |
| 5 | `thin:true` + `fallback: mem_lookup` below threshold; `degraded: stale_index` when the index lags the vault (005 §6) | `projectwiki.AssessQuery` → handler `QueryBatch` → MCP/CLI | unit + integration | `TestAssessQuery{ThinOnEmptyIndex,NotThinWithEnoughHits,DegradedWhenIndexLagsVault}` + `tools_projectwiki_test.go::TestProjectWikiDispatchReads` (thin/fallback pinned on the wire) | P0 | pr | covered |
| 6 | Validate pin-check: fresh passes, stale/unresolved/unattested fail, baseline fast path | `projectwiki.ValidatePage/Validate` | unit | 11× `TestValidate*` | P0 | pr | covered |
| 7 | Lint: full spec 007 §2 check set with severities; `--fix` safe subset; `--strict` blocks only on errors | `projectwiki.Lint/Fix` | unit | `TestLintStructuralChecks`, `TestLintWeakPin`, `TestLintMissingTitleAndFix` | P1 | pr | covered |
| 8 | Status: indexed/phase + by-type, stale, pending, orphans (T7) | `projectwiki.Status` | unit + integration | `TestStatusReportsIndexedPages`, `TestStatusBreakdown`, dispatch pin (`by_type`/`orphans` on the wire) | P1 | pr | covered |
| 9 | Cite-source idempotent with fingerprint; eval recall@K vs baseline | `projectwiki.Cite/Eval` | unit/integration | `TestCiteSourceReturnsContent`, `TestProjectWikiCiteSourceIdempotent`, `TestEvalRecallAtOne` | P1 | pr | covered |
| 10 | Bootstrap idempotent scaffold; no-vault project reports thin/clean (T1) | `projectwiki.Bootstrap/AssessQuery` | unit | `TestBootstrapScaffoldsContextualTypes`, `TestAssessQueryThinOnEmptyIndex` | P0 | pr | covered |
| 11 | 013: four contextual types in taxonomy, whitelist, path validation, `[[type/slug]]` in-degree | `PageTypeDirs`, `collectPages`, `ValidatePagePath`, `recomputeInDegree` | unit | `TestValidatePagePathContextualTypes`, `TestCollectPagesWhitelistScope`, `TestIndexInDegree` | P0 | pr | covered |
| 12 | Shared handler + MCP surface (12 tools, queue, ownership, AGENTS.md block) | `ProjectWikiHandler` / MCP stdio | integration | `handler_projectwiki_test.go` (8 tests), `TestProjectWikiToolContract`, `TestProjectWikiGenerateQueueAndOwnership`, `TestProjectWikiInstructionsServedOverStdio` | P0 | pr | covered |
| 13 | Live vault: validate/lint/status/query against the real `.skillgrid/projectwiki/` | CLI from repo root | integration (manual smoke) | `validate: 3 checked, 0 drifted` · `status: indexed=65 stale=0 pending=0 orphans=65` · `query "wiki autolearning": 1 hit + thin/fallback line` · `lint: 221 warn/info findings (vault hygiene, see Findings)` — 2026-11-11 | P0 | pr | covered |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| Generate no-op proof | Each of the 4 proof layers individually blocks a write | `changed: 0`, nothing written | `TestGenerateLayer1..4BlocksNoop` | covered |
| Index | Page file removed from vault | Soft-delete, excluded from query/lint/status | `TestIndexSoftDeletesRemovedPages` | covered |
| Index | Full Reindex vs incremental Index | Identical aliases/counts/in_degree derived state | `TestLintStructuralChecks` (runs on Reindex), `TestIndexInDegree` | covered |
| Query | Index lags the vault file (mutated, not re-indexed) | `degraded: stale_index` | `TestAssessQueryDegradedWhenIndexLagsVault` | covered |
| Query/status | Vault root does not exist | No lag, no error (stale 0) | `indexLag` missing-root guard; `TestProjectWikiJSONParity` | covered |
| Validate | Symbol anchor degrades to occurrence-only | `fresh` + `degraded`, not unresolved | `TestValidatePageSymbolDegraded` | covered |
| Lint | Grounded page, empty sources | `missing_sources` at error severity; contextual at warn | `TestLintStructuralChecks` | covered |
| Bootstrap | Sentinel `config.json` already exists | No-op, sentinel skipped | `bootstrap_test.go` idempotence assertions | covered |

### Out of Scope

- PostgreSQL-backed store path — dual-mode `Store` is the memory change's verified surface; ProjectWiki tests run on SQLite. Not re-verified here.
- LLM synthesis prose quality — only the deterministic degraded floor and no-op proof are gated.
- 012 trajectories (`skills`/`cases` opt-in types) — spec exists; not part of the 011+013 delivered scope.
- External-citation network re-resolution (007 neuledge fingerprint gate) — network-bound; skipped offline with no finding per 007 §3, never an error.
- Vault content hygiene (the 221 lint findings on the real vault) — content follow-up, not engine behavior; filed as W009.

## Goal-Backward Verification

**Stated goal** (from blueprint.md): Make the markdown vault at `.skillgrid/projectwiki/` the single synthesized knowledge surface of mnemonic-v2, with the FTS index as a derived cache in the shared project store, guarded by `projectwiki_*` engine tools and the `projectwiki` management skill.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | T1: no `config.json` → no ProjectWiki, reported cleanly (not error) | `TestAssessQueryThinOnEmptyIndex` PASS (empty index → thin + fallback, no error) + bootstrap sentinel no-op assertions | VERIFIED |
| Truth | T2: clean generate reports `changed: 0`, writes nothing (PW-INV-4) | `TestGenerateProvenNoop` + `TestGenerateLayer1..4BlocksNoop` PASS | VERIFIED |
| Truth | T3: incremental index only-changed; reindex full rebuild; both idempotent | `TestIndexIncrementalOnlyChanged`, `TestReindexRebuildsProjectWikiPages`, `TestIndexSoftDeletesRemovedPages` PASS; Reindex now derives identical state to Index (upsertPageFull + recomputeInDegree) | VERIFIED |
| Truth | T4: query wiki-first cited, memory fallback when thin, `degraded: stale_index` | `TestQueryReturnsRankedProjectWikiPages`, `TestAssessQuery*` (3 tests) PASS; wire-pinned in `TestProjectWikiDispatchReads`; live CLI prints the thin/fallback line | VERIFIED |
| Truth | T5: validate fails/flags stale/unresolved pins+claims | 11× `TestValidate*` PASS + live `validate: 3 checked, 0 drifted` | VERIFIED |
| Truth | T6: lint reports the 007 §2 checks; `--fix` safe subset; `--strict` blocks on errors only | `TestLintStructuralChecks`, `TestLintWeakPin`, `TestLintMissingTitleAndFix` PASS (orphans, broken wikilinks, dup aliases, missing sources, ungrounded, weak pin, stale_after, weak xref, contradiction, broken citations, coverage gaps); live lint reports the real vault's hygiene truthfully | VERIFIED |
| Truth | T7: status reports counts by type, stale, last run, pending queue, orphans | `TestStatusBreakdown` PASS (by-type/stale/pending/orphans) + live `status: indexed=65 stale=0 pending=0 orphans=65` | VERIFIED |
| Truth | T8: cite-source records citation with fingerprint | `TestCiteSourceReturnsContent` + `TestProjectWikiCiteSourceIdempotent` PASS | VERIFIED |
| Truth | 013: four contextual page types end-to-end (ADR-0060) | `TestBootstrapScaffoldsContextualTypes`, `TestValidatePagePathContextualTypes`, `TestCollectPagesWhitelistScope`, `TestIndexInDegree` PASS; live vault carries the 4 folders + templates | VERIFIED |
| Artifact | A: `projectwiki_*` engine over shared store (6 tables, 12 tools) | `mnemonic/internal/projectwiki/` + `handler_projectwiki.go` + `tools_projectwiki.go`; `TestProjectWikiToolContract` PASS | VERIFIED |
| Artifact | B: `projectwiki` management skill (5 files) | `.agents/skills/projectwiki/{SKILL,CONVENTIONS,PAGES,CLAIMS,MERMAID}.md` present | VERIFIED |
| Artifact | C: acceptance scenarios | `acceptance.feature` — 27 scenarios / 10 requirements, traceability 27/27 COMPLIANT (this run) | VERIFIED |
| Key Link | Agent-proactive save loop (blueprint as amended): `mem_save` → agent `projectwiki_generate` → index → no-op proof | `TestGenerateCreatesNewPageViaFloor`, `TestProjectWikiGenerateQueueAndOwnership` PASS; protocol recorded in 008 + AGENTS.md vault block + skill | VERIFIED |
| Data Flow | Real facts → generate → indexed vault → cited query | Live smoke 2026-11-11: status 65 indexed → validate 3/0 → query returns ADR-0060 with thin/fallback metadata | VERIFIED |

## Traceability Matrix

Deterministic: `node scripts/traceability.mjs 2026-10-31-mnemonic-v2-knowledge-base --test-output=<go test -v output> .` — **27 scenarios, 10 requirements, 27 COMPLIANT, 0 FAILING, 0 UNTESTED** (scope COMPLETE). Every scenario maps to the same-named Go test in the Test Plan above; SATISFIES refs are recorded in `tasks.md` § Traceability.

**Coverage:** 27/27 scenarios covered by a test that ran and passed (`go test ./internal/projectwiki/ ./internal/transport/... -count=1 -v`, 135 PASS lines, 0 FAIL).

## Verification-Gap Audit

All four 2026-11-10 gaps are closed by the fix-by-code round; one new content-level observation is filed, not gated.

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| ~~Engine thin/fallback/degraded not wired~~ **CLOSED (W003)** | `internal/projectwiki/query.go::AssessQuery`, handler `QueryBatch`, MCP/CLI surfaces | missing-adoption | New tests + dispatch wire pin + live CLI thin line (2026-11-11) | — |
| ~~Eager SaveEvent link claimed, proactive shipped~~ **CLOSED (W005)** | `blueprint.md` Key Link (amended 2026-11-11) | missing-adoption | Blueprint now records the shipped 008 proactive design; tasks reconciled | — |
| ~~Engine lint = 2 of 6 checks~~ **CLOSED (W006)** | `internal/projectwiki/lint.go` | regression (partial) | Full 007 §2 set implemented + `TestLintStructuralChecks/WeakPin`; Reindex parity root-fix (`upsertPageFull`+`recomputeInDegree`) found by the new tests | — |
| ~~tasks.md TICKET-01..09 unchecked~~ **CLOSED (W005)** | `tasks.md` | broken-verification | Reconciled with landed KB-T1..T7 + QA fix annotations; SATISFIES traceability section added | — |
| Real vault has 221 lint findings (orphans/missing sources/weak xrefs — vault pages cross-link with markdown links, not `[[wikilinks]]`) | `.skillgrid/projectwiki/**` (content) | regression (content, not engine) | Live `projectwiki lint` 2026-11-11: 221 warn/info (2 error: grounded workflows with empty `sources:`) | A consumer running `lint --strict` on the real vault exits 1 until the two workflow pages cite sources — content fix, filed W009 |

## TDD Evidence Audit

> Historical tickets predate the machine-checkable table (`testing.tdd: false`, Standard mode); their RED hashes were never recorded, so the RED cells are disclosed `N/A (not recorded)` — the script's accepted absent-but-disclosed form — with real, git-verified GREEN hashes and existing test files. The 2026-11-11 fix round ran RED→GREEN in-session (`TestAssessQuery*/TestLint*/TestStatusBreakdown` failed to compile against the old signatures, then passed); its GREEN hash is recorded when the pending commit lands (landed 2026-11-11 as c582984d after the global git signing config was removed).

| Task | SATISFIES | RED (commit + exit) | GREEN (commit + exit) | Test file | Verdict |
|---|---|---|---|---|---|
| TICKET-01..05 (011 engine KB-T1..T5) | generate/index/query/validate scenarios | N/A (not recorded, pre-table change) | `38f826d8` exit 0 | `mnemonic/internal/projectwiki/generate_test.go` | OK |
| TICKET-06..07 (011 CLI KB-T6) | projectwiki-dispatch-reads | N/A (not recorded, pre-table change) | `94a5833a` exit 0 | `mnemonic/cmd/mnemonic/projectwiki_parity_test.go` | OK |
| TICKET-08..09 (011 MCP KB-T7) | projectwiki-tool-contract | N/A (not recorded, pre-table change) | `86f4df82` exit 0 | `mnemonic/internal/transport/mcp/tools_projectwiki_test.go` | OK |
| T-013-1 (engine page types) | validate-page-path-contextual-types | N/A (not recorded, pre-table change) | `6b75555a` exit 0 | `mnemonic/internal/projectwiki/bootstrap_test.go` | OK |
| T-013-2 (vault scaffold) | bootstrap-scaffolds-contextual-types | N/A (docs/vault ticket, no test) | `357b63b7` exit 0 | N/A (vault files verified by live validate/lint) | OK |
| T-013-3 (skill write paths) | projectwiki-tool-contract | N/A (docs/skills ticket, no test) | `dec9ae8f` exit 0 | N/A (skill files on disk) | OK |
| QA fix W003/W006 (2026-11-11) | assess-query-*, lint-*, status-breakdown | N/A (in-session RED observed — compile failure vs old signatures; no hash recorded) | `c582984d` exit 0 | `mnemonic/internal/projectwiki/lint_checks_test.go` | OK |

**Summary:** 7/7 rows have complete TDD evidence as disclosed. (`node scripts/tdd-evidence.mjs` re-run in this gate; hashes verified against the git object store.)

## Assertion Quality Audit

Re-scanned `mnemonic/internal/projectwiki/*_test.go` incl. the three new test files:

| File | Line | Pattern | Evidence | Severity |
|---|---|---|---|---|
| — | — | — | No tautologies, no assertion-without-production-call, no ghost loops (loop assertions iterate fixed literals after a real engine call), no snapshot/source-text assertions. New tests assert severities + counts + wire substrings (`"thin":true`, `"by_type"`) against real dispatch output. | — |

**Summary:** ✅ All assertions verify real behavior. 0 CRITICAL, 0 WARNING, 0 SUGGESTION.

## Changed-File Coverage

`testing.coverage` empty, `quality.coverage_min: 0` — coverage gates N/A (informational).

| File | Line % | Uncovered lines | Rating |
|---|---|---|---|
| `internal/projectwiki` (package) | 67.4% (2026-11-10, pre-fix; new tests only add coverage) | per-file breakdown not collected (no coverage tool configured) | Acceptable (advisory only at threshold 0) |

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| — | — | No violations. Tests exercise production code against the real SQLite store; negative cases present (`TestValidateStaleClaimFails`, `TestAssessQueryDegradedWhenIndexLagsVault`, broken-citation/orphan fixtures); the one test-assertion change (`"error"` → `"error":` in the dispatch lint check) narrows a sloppy substring to the actual transport-envelope shape, with the reason recorded inline. |

**P0 triangulation:** no-op proof has 5 tests (proven + 4 layers); index has incremental/reindex/soft-delete/in-degree variants; assess has thin/not-thin/degraded with distinct inputs. Triangulated.

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|-----|-----------|
| Engine logic (generate/index/query/validate/lint/status/cite/eval/assess) | unit | no | no | Store-boundary logic against the real SQLite store |
| Handler + MCP contract (12 tools, queue, ownership, thin/status wire shape) | integration | no | no | Crosses the transport seam; stdio dispatch is the consumer surface |
| Live vault verbs | integration (manual smoke) | yes (no e2e layer configured) | no | Real CLI against the real vault — proves root resolution + DB state together |

**Layer distribution:** ~50 unit, ~15 integration, 0 e2e. Reasonable for a library-shaped engine.

## Security Audit

**Mode:** Trivy (Mode A); `fail_on: ""` (advisory only per AGENTS.md).

### Trivy Findings (Mode A)

**Command:** `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW` (2026-11-10; the fix round adds no dependencies, result unchanged in kind)

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | CVE-2026-56851 `golang.org/x/text@0.41.0` | HIGH | Go module (pre-existing) | available upstream | SUGGESTION (advisory, W008) |
| vuln | GHSA-cg2g-xh64-v4q7 `dompurify` + 11 JS findings | CRITICAL/HIGH/MEDIUM/LOW | SPA / node_modules (pre-existing, outside footprint) | available upstream | SUGGESTION (advisory, W008) |

**Trivy gate:** `fail_on: ""` → N/A. 13 findings, 0 introduced by this change (no new dependencies — ADR-0008 respected; the fix round adds none either).

**OWASP spot-check:** `internal/projectwiki` uses parameterized store queries throughout; new lint code only stats repo-relative `code:` paths (`filepath.Join(repo, …)`) and walks the repo for `*.go` — no shell-out, no new endpoint/auth surface, no untrusted-input sink beyond vault markdown the engine already parsed. No findings.

**Security verdict:** PASS (advisory findings only) — CRITICAL: 0 blocking, WARNING: 0 blocking, SUGGESTION: W008.

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage (whole project) | — (`testing.coverage` empty) | 0 | n/a | N/A |
| Changed-file coverage | `go test -cover ./internal/projectwiki/` (informational) | 0 | 67.4% pre-fix (rising with new tests) | N/A |
| Mutation | not configured | 0 | n/a | N/A |
| Lint | `go vet ./...` (inline) | errors only | exit 0 | PASS |
| Typecheck | `go build ./...` | errors only | exit 0 | PASS |
| P0 pass rate | named P0 tests | 100 | 100% (27/27 scenarios COMPLIANT) | PASS |
| P1 pass rate | full suite `go test ./...` (mnemonic) | 95 | 100% (21 packages, exit 0, re-run after fix) | PASS |
| Trivy security | `trivy fs . --scanners vuln …` | `""` (report only) | 13 pre-existing, 0 introduced | N/A |
| Duplication (jscpd) | `npx jscpd --reporters ai …` on changed files | 0 (report only) | 5 clones · 0.9% (unchanged by the fix) | N/A |

**Quality config status:** configured (`coverage_min: 0`, `mutation_min: 0` — deliberately disabled, advisory reporting only).

## Duplication

**Command:** `npx jscpd --reporters ai --ignore "**/node_modules/**,**/dist/**,**/*.min.*,**/*_test.go,**/migrations/**" mnemonic/internal/projectwiki mnemonic/internal/transport/handler/handler_projectwiki.go mnemonic/internal/transport/mcp/tools_projectwiki.go`

**Result:** 5 clones · 0.9% duplication — scope: COMPLETE. Unchanged by the fix round (the shared `buildLinkTargets`/`indexLag` extractions avoided new clones).

| Files (lines) | Kind | Introduced? | Severity | Note |
|---------------|------|-------------|----------|------|
| `fingerprint.go:61-69` ~ `query.go:96-104` | exact | pre-existing | SUGGESTION | Row-value helpers (W007) |
| `index_inc.go:93-100` ~ `index_inc.go:119-126` | exact | pre-existing | SUGGESTION | Edge-resolution block (W007) |
| `manifest.go:107-112` ~ `runstate.go:301-306` | exact | pre-existing | SUGGESTION | Persistence snippet (W007) |
| `query.go:39-46` ~ `search.go:31-38` | exact | pre-existing | SUGGESTION | FTS row projection (W007) |
| `handler_projectwiki.go:330-336` ~ `handler_projectwiki.go:341-347` | exact | pre-existing | SUGGESTION | Index/Reindex handler boilerplate — looks intentional (W007) |

## State Drift

**Verdict:** DRIFT: none (`qa-gate.mjs` `state_drift`: exit 0, re-run 2026-11-11)

Deterministic checks: `ship_drift` skipped (no base ref); `size_budget` — all 109 skills within ceiling; `adr_invariants` PASS (58 files, 59 rows); `okf_lint` DRIFT clean (orphans advisory); `status_banner` exit 1 — 10 findings, all pre-existing in other change folders; `windows` clean (schema-valid after the W003–W009 append/close updates); `delivery_guard` UNSCOPED (tasks.md predates the contract). Traceability + TDD-evidence checks now exit 0 (artifacts backfilled).

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix | COMPLETE (27/27 from `acceptance.feature`, deterministic script) | none |
| Verification-gap audit | COMPLETE (blueprint truths + engine sources + live smoke) | none |
| State Drift (9.5) | COMPLETE (qa-gate JSON) | n/a |
| Structure Drift (9.5) | UNSCOPED — skipped, no base ref | n/a |

**Composite (qa-gate JSON):** COMPLETE.

**Stale-verification:** `STALE: none` — all evidence in this render was produced 2026-11-11 against the fixed working tree (full suite re-run, traceability re-run, live smoke re-run).

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED |
| Traceability (weakest scenario) | COMPLIANT (27/27) |
| Verification-gap audit | SUGGESTION (vault-content hygiene W009; no engine gap) |
| TDD evidence (weakest ticket) | OK (7/7 as disclosed) |
| Assertion quality audit (weakest finding) | none |
| Changed-file coverage (weakest file) | N/A (threshold 0) |
| Test quality audit | none |
| Code-quality gates (weakest gate) | PASS / N/A |
| Security audit | SUGGESTION (advisory pre-existing deps, `fail_on` unset — cf. precedent below) |
| Verification scope (composite) | COMPLETE |

**FLOOR:** PASS-eligible. Every dimension is at its best value or at an advisory SUGGESTION that project precedent treats as non-capping (`fail_on`-unset Trivy and `fail_on: 0` duplication were both open when the mcp-split gate rendered PASS with W002 open). No FAIL/UNVERIFIED/UNTESTED/MISSING_RED/UNREADABLE dimension remains.

## Findings

### CRITICAL (must fix before merge)

- None.

### WARNING (should fix before archive)

- None open. W003–W006 closed by the 2026-11-11 fix-by-code round (code commits pending SSH signing at render time; hashes recorded in § TDD Evidence Audit at commit).

### SUGGESTION (nice to have)

- **W007 (open)** — jscpd 5 clones · 0.9% in changed files; extract shared row helpers. Advisory (`fail_on: 0`).
- **W008 (open)** — Pre-existing Trivy dependency findings (13) outside this change's footprint; fix at the dependency level in a separate change. Advisory (`fail_on: ""`).
- **W009 (new)** — Real-vault lint hygiene: 221 warn/info findings + 2 error (`missing_sources` on the grounded `workflows/development-workflow.md` and `workflows/sdd-pipeline.md` pages) now that engine lint implements 007 §2. Content fix in the vault (cite sources, add wikilinks or accept orphan status); `lint --strict` on the real vault exits 1 until the two workflow pages cite sources.

## Gate Decision

**Verdict:** PASS

**Reasoning:** The 2026-11-10 CONCERNS drivers are closed in code, not waived: thin/fallback/degraded are engine-wired and wire-pinned (W003), the full 007 §2 lint set and T7 status breakdown are implemented and tested — a round that also caught and fixed a real Reindex-vs-Index derived-state divergence (W006), `acceptance.feature` (27 scenarios) + the TDD table are backfilled with deterministic 27/27 COMPLIANT traceability (W004), and `tasks.md`/blueprint were reconciled to the shipped design (W005). Full mnemonic suite (21 packages), build, and vet are green after the fix; the live vault smoke exercises status/validate/query on real data. Remaining items are advisory SUGGESTIONs (W007–W009) under advisory-only thresholds, per project precedent.

**Open items:** W007, W008, W009 (all SUGGESTION, non-blocking). Two administrative notes for ship: (1) the fix landed as 9873c2d8 (spec) + c582984d (code) once the global git signing config was removed; (2) `blueprint.md` ratification as a whole remains the human's ship-gate call (its Key Link amendment is recorded in place).

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

**Human decision (fill in):** <accept / accept-with-open-items / reject> — <name> — <date>

<!-- reflect completes this half at archive time -->

## Final-State Facts

## Gates

## Decisions

## Lessons

## Patterns

## Surprises

## Environment Retro

## Acceptance Verdict

## Open Items (→ next change)

## Web-cache promotion candidates

## Prior-Change Follow-Through

## Move Evidence (from ship context)

## Overrides / Waivers / Contradictions

## Lineage (observation IDs)
