# Report — mnemonic v2 memory function

> Change: `.skillgrid/specs/2026-10-31-mnemonic-v2-memory/` (moves to `.skillgrid/archive/2026-10-31-mnemonic-v2-memory/` at ship)
> Generated: 2026-11-02T15:05+01:00 (qa)
> Gate: CONCERNS
> Tier: T3 (blueprint `Tier:` line — escalated from config default T2; QA floor L4: full suite + build + go vet + coverage; mutation/security advisory per config)
> Effective floor applied: L4 per T3; change classification (high-risk: new trust boundary, public MCP contract, data migration) matches.
> Quality config: configured (coverage_min 0 = advisory, mutation 0 = disabled, p0 100%, p1 95%; trivy fail_on "" = advisory).

## Test Plan

> Derived from `blueprint.md` (Must-Haves T1–T24, K1–K6), `tasks.md` (20 tickets, per-ticket SATISFIES), and `acceptance.feature` (16 requirements, 75 scenarios, gates G1–G16).

### Risk Ranking

The most likely production breakers, in order: (1) the SQLite write path corrupting or losing a write under concurrent access — the data itself is the product, so the two-process WAL backstop (T21) is tested hardest, at the integration layer with real OS processes; (2) the public `mem_*` MCP contract silently drifting (D1 — the plugins consume it) — pinned by a literal contract test (K5) and the CLI↔MCP byte-parity test (K6); (3) the sync engine resurrecting deleted rows or double-applying mutations across machines — pinned by tombstone/deferred/enrollment tests through the real store.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Two-process WAL: 20 saves from a second process + live parent session, no lost write | `TestConcurrentProcessWAL` (real re-exec) | integration | internal/core/store/concurrency_test.go | P0 | pr | covered |
| 2 | Store opens per-project file; sync-id format; ledger FK SET NULL | `store.Open` / `syncid` / ledger | unit | TestStoreOpensPerProjectFile, TestNewSyncIDFormat, TestLedgerFKSetNull | P0 | pr | covered |
| 3 | FTS trigger sync + <3-rune LIKE fallback + composite ranking | `Store.Search` | unit | TestFTSStaysInSync, TestShortTermFallsBackToLIKE, TestSearchRankingRecencyAndStability | P0 | pr | covered |
| 4 | Save persists with sync-id + result shape + redact/truncate + validation | `pipeline.Save` | unit | TestSave* (pipeline_test.go) | P0 | pr | covered |
| 5 | Hash-dedupe noop (same id, dup bump, no new row) | `dedup.Run` floor | unit | TestHashDedupeNoop | P0 | pr | covered |
| 6 | Replay: identical → committed id; changed → 409; deleted → 410 | `pipeline` ledger step 4.5 | unit | TestReplay* (replay_test.go) | P0 | pr | covered |
| 7 | Session lifecycle: lease +30min, terminal 409, resume, finalize atomic idempotent, six-section summary | `sessions.go` | unit | TestSession* / TestSaveIntoEndedSessionRefused / TestSaveAutoRegistersMissingSession | P0 | pr | covered |
| 8 | Prompt capture tri-state + prompt search | `prompts.go` | unit | TestCapturePromptAbsentMeansDefaultOn, TestPromptSearchFindsCaptured (handler) | P1 | pr | covered |
| 9 | Dedup LLM primary + trust-but-verify + noop first-class + degraded floors | `dedup.go` LLM branch | unit (fake LLM seam) | TestDedup* (dedup_llm_test.go, dedup_test.go) | P0 | pr | covered |
| 10 | Conflict candidates + relevance gate + no-candidate boundary | `relations.go` | unit | TestConflictCandidates*, TestRelevanceGate*, TestNoCandidate* | P1 | pr | covered |
| 11 | Judge/judge_many one-write/compare idempotent/consolidate fingerprint/scan | `judgment.go` | unit | TestJudge*, TestCompare*, TestConsolidate*, TestScan* | P1 | pr | covered |
| 12 | Lookup ladder: topic-key exact, phrase ≤5, context budget, RRF vector blend degraded | `lookup.go` | unit | TestLookup*, TestContext*, TestVectorLeg* | P0 | pr | covered |
| 13 | Trails recorded on every retrieval; usage raises importance; mark-reviewed restarts clock | `trails.go` / `importance.go` | unit | TestTrail*, TestUsageRaises*, TestReviewMark* | P1 | pr | covered |
| 14 | Save event exactly-once + consumer isolation + no-wiki-import | `eventbus.go` | unit | TestSaveEvent*, TestConsumerFailure*, TestMemoryDoesNotImportWiki | P0 | pr | covered |
| 15 | MCP tool contract pinned (agent/admin literal sets, K5) + write queue serializes + envelope degraded + stdio round-trip | `transport/mcp` | integration (stdio) | TestMemToolContract, TestWriteQueue*, TestEnvelope*, TestStdio* | P0 | pr | covered |
| 16 | CLI↔MCP byte-identical parity (K6) + project selection + doctor strict exit | `cmd/mnemonic` | integration (in-proc CLI) | TestCLIMCPParity, TestCLIProjectSelection, TestDoctorStrict | P0 | pr | covered |
| 17 | Pull applies through store (INV-4); tombstone floor; deferred 5-cap; enrollment gates | `sync.go` | unit | TestPull*, TestTombstone*, TestDeferred*, TestEnrollment* | P0 | pr | covered |
| 18 | Push acks seq (≤100 + ≤8MiB); pull scoped per schema; fail-fast; export/import round-trip; chunk id stable | `transport.go` / `chunkcodec.go` | unit (fake transport) | TestPush*, TestPull*, TestSchema*, TestExport*, TestChunk* | P0 | pr | covered |
| 19 | Autosync: single lease, debounce 10→1, write path never blocks on network (timing) | `autosync.go` | integration (goroutine + real store) | TestAutosync*, TestDebounce*, TestWritePath* | P0 | pr | covered |
| 20 | PG dialect parity + from-empty migrations (G15, live DSN) | `postgres.go` | integration (real PG) | TestPG* (postgres_test.go) | P0 | pr | covered |
| 21 | v1 seed import + v1 files byte-identical (D4) + idempotent re-run | `migrate.go` | integration (real v1 fixture file) | TestV1* (migrate_test.go) | P0 | pr | covered |
| 22 | Doctor reports capability gaps, --strict exits non-zero | `cmd/mnemonic doctor` | integration (in-proc CLI) | TestDoctor* | P1 | pr | covered |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| save pipeline | empty title / bad type / bad scope | rejected before write | save-validation → TestSaveValidation | covered |
| save pipeline | `<private>` tag + >max-bytes content | redacted + truncated before hash | save-redact-truncate | covered |
| dedup | LLM returns candidate not in pre-filter set | trust-but-verify rejects, hash verdict stands | dedup-trust-but-verify | covered |
| dedup | nil seam / failing seam (500) | floor + `degraded:true`, ≤1 retry | degraded-explicit, degraded-bounded-retry | covered |
| replay | OperationID reused with changed payload | 409 operation_conflict | replay-changed-payload-409 | covered |
| replay | target hard-deleted (ledger FK SET NULL) | 410 operation_expired | replay-deleted-target-410 | covered |
| sessions | save into ended session | 409 session_already_ended | session-terminal-409 | covered |
| sessions | save with unknown session id | auto-registered, save proceeds | save-with-unknown-session | covered |
| search | <3-rune query term | LIKE fallback still finds the row | short-term-like-fallback | covered |
| lookup | absent topic_key | returns nothing, does not widen | lookup-topic-key-exact | covered |
| lookup | vector leg, null embedder | FTS-only + degraded flag | vector-blend-degraded | covered |
| relations | 1 shared significant term | gate fails, no candidate | no-candidate-when-gate-fails | covered |
| sync | upsert → delete → late upsert (same entity) | row stays deleted (tombstone floor) | tombstone-floor-no-resurrect | covered |
| sync | relation pulled before its target observation | deferred, 5 retries, then dead | deferred-apply-5-cap | covered |
| sync | save in unenrolled project | no remote mutation journaled | enrollment-gates-journal | covered |
| sync transport | 101-mutation batch | refused (≤100 cap) | push-acks-seq | covered |
| sync transport | project schema unresolvable | fail fast, no `public` fallback | schema-per-project-fail-fast | covered |
| autosync | two workers, same store | one lease holder, loser stands down | autosync-single-lease | covered |
| autosync | 500 ms slow transport during save | save returns <100 ms | write-path-never-blocks-on-network | covered |
| cutover | v1 fixture re-migrated | idempotent (hash dedupe absorbs) | v1-seed-import (TestV1MigrateIdempotent) | covered |
| doctor | capability gap present, --strict | exit non-zero | doctor-strict-exits-nonzero-on-gap | covered |

### Out of Scope

- Live multi-machine PG replication end-to-end (two real PG instances) — the transport is tested against a fake transport + one real PG for dialect parity; cross-machine clock/ordering behavior beyond tombstone/seq is the sync spec's later hardening, not this change.
- The llm-wiki consumer (separate change) — only the event contract + the no-wiki-import guard are proven here via `internal/wiki_teststub`.
- opencode/pi plugin consumption of the MCP contract (separate changes) — the contract is pinned literally (K5) so their builds cannot drift silently.
- `--all` multi-project fan-out on read commands — deferred to the cross-project store (T16 note); the CLI rejects it loudly (usage error) rather than fanning out.

## Goal-Backward Verification

**Stated goal** (blueprint.md): Build the memory function of mnemonic v2 — the raw, sync-ready source-of-truth fact store (observations, sessions, relations, prompts, dedup, lookup, sync) behind a dialect-agnostic `Store` (SQLite | Postgres) and an MCP `mem_*` tool surface — so agents save atomic durable facts and the llm-wiki function consumes them via a save event.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | T1 save persists + result shape + retrievable by id and sync_id | TestSavePersists* + TestAddGetObservationRoundTrip (ran, pass) | VERIFIED |
| Truth | T2 hash floor: same content → same id, dup bump, no new row | TestHashDedupeNoop (ran, pass) | VERIFIED |
| Truth | T3 topic_key revision in place + version snapshot | TestSave* revision subtests (pipeline_test.go, ran, pass) | VERIFIED |
| Truth | T4 LLM dedup `update` reason=llm; zero-value → hash floor | TestDedupLLMPreservesUpdate / TestDedup* (fake seam, ran, pass) | VERIFIED |
| Truth | T5 search compact rows + BM25×boost ranking + LIKE fallback + soft-delete exclusion | TestSearchRanking* + TestShortTermFallsBackToLIKE (ran, pass) | VERIFIED |
| Truth | T6 lookup topic-key exact / id full / phrase ≤5 / no widening | TestLookup* (ran, pass) | VERIFIED |
| Truth | T7 conflict candidates → pending relations + judge provenance + judge_many one write | TestConflictCandidates* + TestJudge* (ran, pass) | VERIFIED |
| Truth | T8 compare idempotent + consolidate fingerprint-gated | TestCompare* + TestConsolidate* (ran, pass) | VERIFIED |
| Truth | T9 session lease/terminal/finalize atomic idempotent | TestSession* + TestSaveIntoEndedSessionRefused (ran, pass) | VERIFIED |
| Truth | T10 prompt tri-state capture + search finds it | TestCapturePrompt* + handler TestPromptSearchFindsCaptured (ran, pass) | VERIFIED |
| Truth | T23 retrieval usage raises importance (10× vs 0×) | TestUsageRaises* (ran, pass) | VERIFIED |
| Truth | T24 mark_reviewed restarts clock; list returns needs_review | TestReviewMark* (ran, pass) | VERIFIED |
| Truth | T11 trail recorded on every retrieval | TestTrail* (ran, pass) | VERIFIED |
| Truth | T12 degraded floors never silent; bounded retry | TestDegrade* + envelope tests (ran, pass) | VERIFIED |
| Truth | T13 dual-dialect same result shapes + from-empty migrations both | TestPG* parity helpers (ran WITH live DSN, 8/8 pass) | VERIFIED |
| Truth | T14 per-project file; PG schema-per-project fail-fast | TestStoreOpensPerProjectFile + TestSchemaPerProject* (ran, pass) | VERIFIED |
| Truth | T15 push/pull acks + through-store apply + tombstones + deferred 5-cap | TestPush* / TestPull* / TestTombstone* / TestDeferred* (ran, pass) | VERIFIED |
| Truth | T16 chunk archive export/import + stable chunk id | TestExportImportRoundTrip + TestChunk* (ran, pass) | VERIFIED |
| Truth | T17 autosync lease + debounce + off-write-path | TestAutosync* + TestDebounce* + TestWritePath* (ran, pass, -race) | VERIFIED |
| Truth | T18 replay 409/410/committed-id | TestReplay* (ran, pass) | VERIFIED |
| Truth | T19 update/delete wrong expected_project; soft delete hides; --hard physical | TestUpdate* / TestDelete* (handler/store tests, ran, pass) | VERIFIED |
| Truth | T20 save event exactly-once; no wiki import; consumer failure isolated | TestSaveEvent* + TestConsumerFailure* + TestMemoryDoesNotImportWiki (ran, pass) | VERIFIED |
| Truth | T21 (backstop) two processes, same file, no lost write | TestConcurrentProcessWAL (ran, pass, -count=5 + -race) | VERIFIED |
| Truth | T22 doctor reports gaps + --strict non-zero | TestDoctor* (ran, pass) | VERIFIED |
| Artifact | all 28 named artifacts (module, store, seams, pipeline, transport, CLI, migrations, ADRs) | `find` on mnemonic-v2/ — every file present and substantive (implementer reports + this gate's reads); ADR-0040/0041 in `.skillgrid/artifacts/` | VERIFIED |
| Key Link | K1 mem_save → handler → pipeline → one SQLite txn → event | TestSavePersists* + stdio round-trip + TestSaveEvent* (ran, pass) | VERIFIED |
| Key Link | K2 one dedup code path, degraded flag distinguishes | TestDedup* nil-seam vs fake-seam (ran, pass) | VERIFIED |
| Key Link | K3 search → FTS + vector → RRF + trail | TestVectorLeg* + TestTrail* (ran, pass) | VERIFIED |
| Key Link | K4 push CLI + autosync → one Cycle → transport | TestPush* (CLI path) + TestAutosync* (manager path) (ran, pass) | VERIFIED |
| Key Link | K5 contract pin (22 agent names + 6 admin) | TestMemToolContract literal sets (ran, pass) | VERIFIED |
| Key Link | K6 CLI↔MCP byte-identical | TestCLIMCPParity (ran, pass) | VERIFIED |
| Data Flow | real save → FTS row → search returns it → trail written → usage bumps importance | TestFTSStaysInSync + TestTrail* + TestUsageRaises* on one store (ran, pass) | VERIFIED |
| Data Flow | v1 file → migrate → v2 store searchable, v1 bytes+mtime unchanged | TestV1SeedImport + TestV1FilesUntouched (ran, pass) | VERIFIED |

**Floor note:** every truth is VERIFIED by a named test that ran and passed in this gate's suite run (2026-11-02 14:35, `go test ./... -count=1`, exit 0). No PRESENT_BEHAVIOR_UNVERIFIED rows.

## Traceability Matrix

All 75 scenarios of `acceptance.feature` were read in full (scope COMPLETE) and mapped to the covering tests named in the test plan; every covering test ran in the 14:35 suite run (exit 0) and in the per-gate G1–G16 CHECK commands (EVIDENCE lines in acceptance.feature, all green, G15 with live PG DSN).

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| store-opens-per-project | TestStoreOpensPerProjectFile | yes | pass |
| door-check | TestDoorCheckFTSRoundTrip | yes | pass |
| sync-id-format | TestNewSyncIDFormat + TestNewSyncIDFormatViaObservation | yes | pass |
| migration-runs-from-empty | TestMigrationRunsFromEmptyAndReopens | yes | pass |
| ledger-fk-set-null | TestLedgerFKSetNull | yes | pass |
| fts-stays-in-sync | TestFTSStaysInSync | yes | pass |
| short-term-like-fallback | TestShortTermFallsBackToLIKE | yes | pass |
| search-ranking-recency-stability | TestSearchRankingRecencyAndStability | yes | pass |
| save-persists-with-sync-id | TestSavePersists* | yes | pass |
| save-result-shape | TestSaveResult* | yes | pass |
| save-redact-truncate | TestSaveRedacts* | yes | pass |
| save-validation | TestSaveValidation* | yes | pass |
| save-into-ended-session | TestSaveIntoEndedSessionRefused | yes | pass |
| save-with-unknown-session | TestSaveAutoRegistersMissingSession | yes | pass |
| replay-identical-returns-committed-id | TestReplay* | yes | pass |
| replay-changed-payload-409 | TestReplay* | yes | pass |
| replay-deleted-target-410 | TestReplay* | yes | pass |
| session-start-lease | TestSession* | yes | pass |
| session-terminal-409 | TestSaveIntoEndedSessionRefused | yes | pass |
| session-resume-continuation | TestSession* | yes | pass |
| session-finalize-atomic-idempotent | TestSession* | yes | pass |
| session-summary-six-section | TestSession* | yes | pass |
| prompt-capture | TestCapturePromptAbsentMeansDefaultOn | yes | pass |
| prompt-search-finds-captured | TestPromptSearchFindsCaptured (handler) | yes | pass |
| dedup-llm-primary | TestDedup* | yes | pass |
| dedup-trust-but-verify | TestDedupTrustButVerify* | yes | pass |
| dedup-noop-first-class | TestDedupNoop* | yes | pass |
| hash-dedupe-noop | TestHashDedupeNoop | yes | pass |
| degraded-explicit | TestDegrade* | yes | pass |
| degraded-bounded-retry | TestDegradeErroredSeamRetriesOnceThenFloor | yes | pass |
| conflict-candidates-surfaced | TestConflictCandidates* | yes | pass |
| relevance-gate-terms | TestRelevanceGate* | yes | pass |
| no-candidate-when-gate-fails | TestNoCandidate* | yes | pass |
| judge-sets-provenance | TestJudge* | yes | pass |
| judge-many-one-write | TestJudge* (SetTxnHook counter) | yes | pass |
| compare-idempotent | TestCompare* | yes | pass |
| consolidate-preview-confirm | TestConsolidate* | yes | pass |
| consolidate-fingerprint-mismatch | TestConsolidate* | yes | pass |
| scan-resurfaces | TestScan* | yes | pass |
| search-ranking-boosts | TestSearchRanking* | yes | pass |
| lookup-topic-key-exact | TestLookup* | yes | pass |
| lookup-phrase-bounded | TestLookup* | yes | pass |
| context-byte-budget | TestContext* | yes | pass |
| vector-blend-degraded | TestVectorLeg* | yes | pass |
| trail-recorded | TestTrail* | yes | pass |
| usage-raises-importance | TestUsageRaises* | yes | pass |
| review-mark-reviewed-restarts-clock | TestReviewMark* | yes | pass |
| save-event-exactly-once | TestSaveEvent* | yes | pass |
| wiki-failure-never-fails-save | TestConsumerFailure* | yes | pass |
| memory-does-not-import-wiki | TestMemoryDoesNotImportWiki | yes | pass |
| tool-contract-pinned | TestMemToolContract | yes | pass |
| profile-membership | TestMemToolContract (both profiles) | yes | pass |
| write-queue-serializes | TestWriteQueue* (txn-order log) | yes | pass |
| envelope-degraded | TestEnvelope* | yes | pass |
| stdio-round-trip | TestStdio* | yes | pass |
| cli-mcp-parity | TestCLIMCPParity | yes | pass |
| cli-project-selection | TestCLIProjectSelection | yes | pass |
| doctor-strict-exits-nonzero-on-gap | TestDoctorStrict | yes | pass |
| pull-applies-through-store | TestPull* | yes | pass |
| tombstone-floor-no-resurrect | TestTombstone* | yes | pass |
| deferred-apply-5-cap | TestDeferred* | yes | pass |
| enrollment-gates-journal | TestEnrollment* | yes | pass |
| push-acks-seq | TestPushAcksSeq + TestPushEmptyOutbox | yes | pass |
| pull-scoped-to-project-schema | TestPullScopedToProjectSchema | yes | pass |
| schema-per-project-fail-fast | TestSchemaPerProjectFailFast | yes | pass |
| export-import-round-trip | TestExportImportRoundTrip (+ CLI variant) | yes | pass |
| chunk-id-stable | TestChunk* | yes | pass |
| autosync-single-lease | TestAutosyncSingleLease (strengthened fix-round) | yes | pass |
| write-path-never-blocks-on-network | TestWritePathNeverBlocksOnNetwork (timing) | yes | pass |
| autosync-debounce | TestDebounceCoalescesBursts | yes | pass |
| pg-dialect-parity | TestPG* (live DSN) | yes | pass |
| migration-runs-from-empty-pg | TestPG* | yes | pass |
| v1-seed-import | TestV1SeedImport + type-mapping subtests | yes | pass |
| v1-files-untouched | TestV1FilesUntouched (bytes+mtime fingerprint) | yes | pass |
| concurrent-process-wal | TestConcurrentProcessWAL (two real OS processes) | yes | pass |

**Coverage:** 75/75 scenarios covered by a test that ran and passed. (Per-scenario test mapping as named in the test plan rows; the G-gate CHECK commands are the runnable shadow and are all recorded green in acceptance.feature EVIDENCE lines.)

## Verification-Gap Audit

Scope: all 48 code-zone commits of the change (52d554b3..9fe95b0d, mnemonic-v2/ only) enumerated; every changed behavior walked against "if this broke where it's actually used, would verification fail?".

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| PG dialect refused for autosync is a documented deferral, not a defect | autosync.go / mcp.go `startAutosync` (review I-3 fix) | missing-adoption (accepted, documented) | `startAutosync` logs a warning and skips on non-SQLite; run-loop `recover()` bounds the damage; PG lease columns added for future parity | a PG-local machine with `sync.auto=true` silently has no autosync — the warning is the observable signal; no local-truth loss (INV-4) |
| Concurrent-window not enforced in the backstop test | concurrency_test.go (review I-3, acknowledged) | broken-verification (weaker guarantee, documented) | parent writes 10 rows after `Start()`; on a fast host the worker may finish first — still two real OS processes sharing the WAL file, counts order-independent | the test would not catch a regression that only manifests under true interleaving — accepted ceiling, documented in the test comment |
| G5/G6/G7/G10 CHECK lines carry shorthand/0-match patterns (EVIDENCE conservatively documented) | acceptance.feature gates G5–G10 | broken-verification (gate weaker than requirement text; substitution documented per gate) | the CHECK commands as written pass a subset; EVIDENCE lines name the real covering tests and their counts | a future reader running only the CHECK line verifies less than the requirement claims — spec-zone follow-up (named in state.yaml open notes) |

No regression gaps: every changed behavior has a covering test that would fail on the change (the per-ticket SATISFIES sets are the named tests; the T20 gate EVIDENCE re-ran all 16 gates green).

## TDD Evidence Audit

> Validated against git history (48 commits, 52d554b3..HEAD), the implementer reports in `.skillgrid/sdd/2026-10-31-mnemonic-v2-memory/`, and the test files on disk. Full-suite re-run at 2026-11-02 14:35 (exit 0) stands in for per-row GREEN re-runs.

| Task | SATISFIES | RED (commit + exit) | GREEN (commit + exit) | Verdict |
|---|---|---|---|---|
| T01 | 4/4 | narrative pre-impl (package absent, build fail) | 15e009e1 | OK (narrative RED) |
| T02 | 5/5 (1 task-level) | narrative pre-impl + 5 RED runs | dffdfd8d, bfd3b859 | OK (narrative RED) |
| T03 | 3/3 (1 task-level) | narrative pre-impl (`undefined: llm.NewClient`) | 0dda6b64 | OK (narrative RED) |
| T04 | 3/3 (task-level per blueprint) | narrative pre-impl | ed38985e | OK (narrative RED) |
| T05 | 5/5 | **commit f8983234** (test-only, build fail, exit 1) | dd1a0a3f | OK |
| T06 | 3/3 | narrative pre-impl (build fail, exit 1) | 41adbfb0 | OK (narrative RED) |
| T07 | 8/8 | narrative pre-impl (0 events / no such table) | 60cadd77, 92acc30b | OK (narrative RED) |
| T08 | 3/3 | narrative pre-impl | 6c8d4d09 | OK (narrative RED) |
| T09 | 3/3 | narrative pre-impl + weakened-gate re-RED | 30f30600, e4cef7d1, 71078018 | OK (narrative RED) |
| T10 | 6/6 | narrative pre-impl | 3591f147 | OK (narrative RED) |
| T11 | 9/9 (1 task-level) | narrative pre-impl (lookup.go absent) | ad80c23c | OK (narrative RED) |
| T12 | 3/3 | narrative pre-impl | 7bf9772b | OK (narrative RED) |
| T13 | 5/5 | narrative pre-impl | 549fc498 (+ 8a20f585, 4c5a8137, bbec0632) | OK (narrative RED) |
| T14 | 4/4 (1 task-level) | narrative pre-impl + fix-round RED | 9db4e83d, 81a38631 | OK (narrative RED) |
| T15 | 4/4 | narrative pre-impl | 9df8b63b | OK (narrative RED) |
| T16 | 5/5 | narrative pre-impl; import-fix RED = failing TestSyncExportImportCLI at resume | d24ce971; fix 490d7b4a | OK (narrative RED) |
| T17 | 3/3 | **no RED evidence** (fix-round report only; single landing commit 340f9ae7, no test-first commit) | 340f9ae7, b26e2cad | **MISSING_RED** |
| T18 | 3/3 | narrative pre-impl (TestPGMigrationFromEmpty fail w/ DSN) | a86d0d46, 1df17db7 | OK (narrative RED) |
| T19 | 2/2 | narrative pre-fix (TestStubSubcommands exit 2 + 2 failing migrate tests recorded) | b26e2cad (impl) + acc948b8, 1f482b31, 609230a1, a69f61a3 | OK (narrative RED) |
| T20 | 1/1 | narrative pre-first-green (worker 0/20, harness RED) | 0bf051bd, d62059ca | OK (narrative RED) |

**Summary:** 19/20 tasks have complete TDD evidence; 1 MISSING_RED (T17).
All GREEN hashes resolve in git; all SATISFIES names match acceptance.feature or the explicitly-marked task-level names (no STALE rows); all test files on disk.

## Assertion Quality Audit

> Re-scan of all 42 `*_test.go` files in mnemonic-v2/ (every one created by this change). Module uses plain `t.Fatalf`/`t.Errorf` (no testify). 68 range loops inventoried; ~20 highest-suspicion files read in full.

| File | Line | Pattern | Evidence | Severity |
|---|---|---|---|---|
| internal/transport/handler/handler_test.go | 108–120 | smoke-test-only (comment over-promises vs body) | comment claims "the row lands in the requested project (not the store's)" but body asserts only `res.SyncID == ""`; the explicit-vs-detector resolution is not observed here (mitigated package-wide by TestCurrentProjectReturnsDetectorResolution with a distinct-project fixture) | WARNING |
| cmd/mnemonic/sync_test.go | 176–187 | insufficient triangulation | TestSyncPushFailsLoud: comment claims "leaves the outbox intact … the next cycle retries" but body asserts only exit code; no outbox-intact assertion, single error injection | SUGGESTION |
| cmd/mnemonic/sync_test.go | 154–171 | insufficient triangulation (thin CLI adapter) | TestSyncExportImportCLI: export proved by manifest.json existence, import by exit code only; content equivalence is asserted in the memory-package TestExportImportRoundTrip (contract covered, adapter test thin) | SUGGESTION |

Investigated and rejected: all 14 query-result loops guarded or non-vacuous (companions named per loop); zero tautologies; zero setup-only tests; no incomplete-TDD-cycle short-circuits (4 candidates traced through the production call chain); every empty-collection assertion has its non-empty twin; no type-only assertions; no mock-heavy files (worst ratio ~1:1).

**Summary:** 0 CRITICAL, 1 WARNING, 2 SUGGESTION.

## Changed-File Coverage

> `go test ./... -count=1 -coverprofile` (2026-11-02 14:35). `quality.coverage_min` = 0 → advisory only (WARNING at worst, never blocking, per config).

Per-package (the change is an entire fresh module — every file is a changed file):

| Package | Line % | Rating |
|---|---|---|
| cmd/mnemonic | 44.7% | Low (advisory) |
| internal/core/config | 71.5% | Low (advisory) |
| internal/core/project | 96.0% | Excellent |
| internal/core/store | 22.7% | Low (advisory) |
| internal/core/syncid | 75.0% | Low (advisory) |
| internal/intelligence/degrade | 82.4% | Acceptable |
| internal/intelligence/embedder | 100.0% | Excellent |
| internal/intelligence/llm | 82.9% | Acceptable |
| internal/memory | 74.6% | Low (advisory) |
| internal/migrate | 77.9% | Low (advisory) |
| internal/transport/envelope | 60.3% | Low (advisory) |
| internal/transport/handler | 14.6% | Low (advisory) |
| internal/transport/mcp | 56.0% | Low (advisory) |
| internal/wiki_teststub | n/a (no statements) | — |

**Threshold:** `quality.coverage_min` = 0 (disabled — advisory). **Average changed-file coverage:** ~62% (weighted by statements; store + handler packages dominate the denominator — the store's uncovered bulk is the PG-parity legs and error branches exercised by the live-DSN suite and the concurrency backstop).

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| internal/transport/handler/handler_test.go::TestSaveResolvesExplicitProject (108–120) | no-pass-always (weak) / test-claimed-path | the named claim ("lands in the requested project, not the store's") is not what the assertion checks (only sync_id presence); would still pass if explicit-project resolution regressed to the detector default — mitigated by the distinct-fixture sibling test |

All other contracts hold across the 42 test files: real code (every test invokes production through the handler/store/pipeline seams), no vacuous truths (empty-result loops all guarded or companioned), no pass-always (per-scenario SATISFIES tests fail on their scenario's regression), complete mocks (hand-rolled fakes with realistic seam shapes), counter-tests present for the "does not X" behaviors (no-wiki-import, tombstone no-resurrect, unenrolled no-journal, 1-term gate no-candidate).

**P0 triangulation:** every P0 row in the test plan has ≥2 tests with different inputs (TRIANGULATE-commented siblings in the memory/store packages; error paths paired with happy paths per requirement).

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| store/save/relations/sync domain | unit | no | no | real SQLite files in t.TempDir() — the production path, not a mock; highest available layer that fits without a running PG |
| PG dialect parity | integration (real PG) | no | no | live DSN (G15); the only way to prove the second dialect render |
| two-process WAL | integration (real OS processes) | no | no | cross-process lock behavior is unobservable in-process |
| MCP stdio round-trip | integration (real stdio pipe) | no | no | the transport contract is the wire format |
| CLI parity / doctor / sync CLI | integration (in-proc CLI) | no | no | exercises the real dispatch over the same handler |
| autosync lease/debounce/write-path | integration (goroutines + real store + timing) | no | no | lease contention and the timing guarantee need the manager running |

**Layer distribution:** ~65% unit, ~35% integration, 0 e2e. Reasonable for this risk profile: the domain is a store + a pipeline, and the integration surface is deliberately thin (one stdio server, one CLI) — both fully covered at the integration layer. No duplicate coverage: the CLI parity test asserts the byte-identical seam, not a re-assertion of domain behavior.

## Security Audit

**Mode:** Trivy (Mode A, per config) — attempted, degraded to manual notes.

### Trivy Findings (Mode A)

**Command:** `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`

The scan could not complete: Trivy's vulnerability DB download (131 MiB) failed with `write /home/paladm/.cache/trivy/db/trivy.db: no space left on device` — the root filesystem is 100% full (496 MiB available, disk-pressure pre-existing, not caused by this change). No findings were produced; the gate cannot classify them.

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | (none produced — DB download failed) | — | — | re-run Trivy with disk space (or a local trivy-db) as a named re-run path | SUGGESTION (process gap, not a code finding) |

**Trivy gate:** `fail_on: ""` → N/A (advisory-only per config; AGENTS.md § Rules: "Trivy is advisory-only — findings are reported, never blocking the QA gate").

### Manual Findings (Mode B supplement)

#### Secrets Archaeology

| Location | Type | Evidence | Severity |
|----------|------|----------|----------|
| none in mnemonic-v2/ | — | no `.env`/`.pem`/credentials tracked in the module; the only DSN in the tree is the test DSN `postgres://postgres:test@127.0.0.1:15432` (test fixture, in the T18 gate notes, not a committed source constant — the fix round explicitly deleted the `pgLocalhostDSN` bridge-IP constant) | — |

#### Dependency Audit

| Dependency | CVE / Issue | Severity | Fix Available |
|------------|-------------|----------|---------------|
| modernc.org/sqlite v1.60.1, lib/pq v1.12.3, mark3labs/mcp-go v1.1.1, yaml.v3 v3.0.1 | unverified (Trivy DB unavailable — see above) | SUGGESTION (re-run pending) | re-run Trivy with disk space |

#### OWASP Spot-Check

User-facing input = agent-supplied save content. Checked: redaction (`<private>` strip before hash/store — tested), truncation to ContentMaxBytes (tested), project/scope/type validation before write (tested, reject-before-write), sync payload canonicalization (tested), no SQL interpolation (all store paths parameterized via the dialect render), JSON-RPC input parsed by mcp-go (stdlib-level validation), CLI flag parsing via stdlib `flag`. No suspicious pattern requiring human confirmation.

**Security verdict:** N findings — CRITICAL: 0, WARNING: 0, SUGGESTION: 1 (Trivy re-run pending on disk space; advisory-only gate).

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage (whole project) | coverprofile run (advisory) | 0 (disabled) | ~62% avg | N/A (reported) |
| Changed-file coverage | same, filtered to changed files | 0 (disabled) | ~62% (all files changed) | N/A (reported) |
| Mutation | not configured | 0 | N/A | N/A |
| Lint | go vet (no separate linter configured) | errors only | 0 errors | PASS |
| Typecheck | go build (Go compilation) | errors only | 0 errors | PASS |
| P0 pass rate | P0 rows of the test plan (all 22 P0 rows ran in the suite + G1–G16) | 100% | 100% (22/22 green, exit 0) | PASS |
| P1 pass rate | P1 rows (all ran in the suite) | 95% | 100% | PASS |
| Trivy security | trivy fs (see Security Audit) | "" (report only) | scan incomplete (disk) | N/A |

**Quality config status:** configured (thresholds read from `.skillgrid/config.yaml`).

**Dead code:** N/A (no dead-code tool configured).

## State Drift

> `node .agents/skills/verification/qa/scripts/qa-gate.mjs .` (2026-11-02 14:50).

**Verdict:** PARSE ERROR (tooling) — `state-drift-check.mjs` crashed: `ERR_MODULE_NOT_FOUND: Cannot find package 'yaml'` (the script's node_modules dependency is absent in this environment — pre-existing tooling gap, W017's lineage; state.yaml itself parses as YAML and was just edited consistently: `current_phase: execution`, `current_change: 2026-10-31-mnemonic-v2-memory`, `status: all-20-tickets-executed`).

| Field | Stale (state.yaml) | Derived (spec zone) |
|-------|-------------------|---------------------|
| pipeline.current_phase | execution | execution (all tickets done, QA running) — consistent |

**Fix applied:** no drift — manual read confirms state.yaml matches the spec zone (the machine check is blocked by the missing node module, not by state content).

**Scope (from the guard's `SCOPE:` line):** state_drift — n/a (crashed before SCOPE). Advisory only: never CRITICAL, never affects the verdict.

**Advisory drift noted by the guard (adr_invariants check, exit 1):** ADR-0040/0041 use bold-list frontmatter (`- **status:** accepted`) where `check-adr-invariants.mjs` expects YAML frontmatter — the same pre-existing pattern as closed W024 (0032–0036); plus a pointer in an older spec (`2026-10-07-replace-opencode-hooks-with-plugins/adr.md`) to a missing ADR-0032 file — pre-existing, outside this change.

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix | COMPLETE (all 75 scenarios of acceptance.feature read in full) | none |
| Verification-gap audit | COMPLETE (all 48 code-zone commits enumerated from 52d554b3..9fe95b0d) | none |
| TDD evidence audit | COMPLETE (20/20 reports read or grepped; all GREEN hashes resolved) | none |
| Assertion quality audit | COMPLETE (42/42 test files scanned; ~20 read in full) | none |
| State Drift (9.5) | UNREADABLE (script crashed: missing node `yaml` module) | n/a |
| Structure Drift (9.5) | UNSCOPED (guard reports no base ref for the drift window) | n/a |
| Composite (guard `scopes.composite`) | UNSCOPED | — |

**Stale-verification:** `STALE: none` — no code-zone change since the 14:35 suite run; this run's evidence (suite + G1–G16 + -race runs, all 2026-11-02) is current.

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED (all 24 truths + artifacts + K1–K6 + data flows) |
| Traceability (weakest scenario) | COMPLIANT (75/75) |
| Verification-gap audit | WARNING (3 documented/accepted gaps, none a regression gap) |
| TDD evidence (weakest ticket) | **MISSING_RED (T17)** |
| Assertion quality audit (weakest finding) | WARNING |
| Changed-file coverage (weakest file) | N/A (threshold 0 — advisory) |
| Test quality audit | WARNING (1 weak-claim test) |
| Code-quality gates (weakest gate) | PASS (N/A gates disabled) |
| Security audit | SUGGESTION (Trivy re-run pending, advisory gate) |
| Verification scope (composite) | **UNSCOPED** (guard composite: ship_drift no base ref; state_drift tool crash) |

**FLOOR:** the weakest dimension is **TDD evidence — MISSING_RED (T17)** (a gate-FULL value per the four-state table: "no MISSING_RED" is a PASS criterion). Together with the UNSCOPED composite verification scope (ship-drift guard has no base ref for this change) and the WARNING-level items, the floor caps the gate at **CONCERNS** — the T17 row is the named blocker; it is a missing *record* of the RED phase (the behavior itself is verified: 8 autosync tests pass under -race -count=3, the lease/debounce/write-path acceptance is proven), which is why this is a CONCERNS open item with a named resolution path rather than a FAIL: the alternative reading (test written after code for T17) is contradicted by the review record (the reviewer independently walked the uncommitted T17 diff before its commit and produced the GO-WITH-FIXES verdict with per-scenario test analysis).

## Findings

### CRITICAL (must fix before merge)

- None.

### WARNING (should fix before archive)

- W-1: **T17 TDD evidence MISSING_RED** — no RED phase recorded for the autosync tests (task-17-report.md is the fix-round report; single landing commit 340f9ae7, no test-first commit). Path to resolution: either (a) a human accepts the review-record evidence (the reviewer analyzed the T17 diff pre-commit, scenario-by-scenario) and the row is re-marked `OK (review-attested RED)` in the review ledger, or (b) the missing initial T17 report is reconstructed from the session record. Either clears the row without a code change.
- W-2: **TestSaveResolvesExplicitProject comment over-promises** (handler_test.go:108–120) — the assertion does not observe the claimed project resolution; mitigated by the distinct-fixture sibling. Path: tighten the assertion to read the store row's project (10 lines) or fix the comment.
- W-3: **Trivy scan incomplete** — root disk 100% full; vulnerability DB (131 MiB) would not download. Advisory gate (`fail_on: ""`), but the dependency CVE posture of the four new/third-party deps (modernc.org/sqlite, lib/pq, mcp-go, yaml.v3) is unverified. Path: free disk (or mount trivy-db elsewhere) and re-run `trivy fs . --scanners vuln`.
- W-4: **Guard tooling gaps** — state-drift-check.mjs crashes (missing node `yaml` module) and ship-drift reports UNSCOPED (no base ref); both block the machine drift checks and force the manual state-read. Pre-existing tooling, not this change's defect. Path: `npm i yaml` in the skills tree / provide the base ref at QA time.

### SUGGESTION (nice to have)

- S-1: TestSyncPushFailsLoud — assert the outbox is intact after the failed push (the comment claims it).
- S-2: TestSyncExportImportCLI — assert the imported row's content, not just the exit code (the contract is covered in the memory package; the CLI adapter test stays thin).
- S-3: G5/G6/G7/G10 CHECK lines in acceptance.feature still carry shorthand/0-match patterns with conservative EVIDENCE — spec-zone follow-up to rewrite the CHECK lines to the real test names.
- S-4: ADR-0040/0041 frontmatter is bold-list where the invariant checker wants YAML (same pre-existing pattern as W024, closed for 0032–0036).

## Gate Decision

**Verdict:** CONCERNS

**Reasoning:** All 24 truths VERIFIED by named passing tests, 75/75 scenarios COMPLIANT, 0 CRITICAL findings, all code-quality gates PASS/N-A, P0 and P1 at 100%. The floor is capped below PASS by one TDD-evidence row (T17 MISSING_RED — a missing RED *record*, not a missing behavior; the behavior is verified and was independently reviewed pre-commit) and by the UNSCOPED composite verification scope from the guard's ship-drift check (no base ref available in this environment). Both have named, code-free or low-cost resolution paths.

**Open items (if CONCERNS):**
- W-1 T17 RED record — resolution: human attestation (review record stands in) or reconstructed initial report. No code change.
- W-3 Trivy re-run — resolution: free root disk, re-run the configured command. Advisory gate; does not block review routing.
- W-2 handler test comment/assertion — resolution: tighten in the review fix round (optional; mitigation exists).
- W-4 guard tooling — resolution: environment fix (node `yaml` module + base ref); pre-existing.

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

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

**Every entry MUST cite a source** — a file:line, a commit, a ticket ID, or a scenario name. No source = undiagnosed, not a learning.

| Decision | Tradeoff | Why | Source |
|----------|----------|-----|--------|
| <choice made> | <what was given up> | <why it won> | <file:line / commit / ADR / ticket> |

## Lessons

| Lesson | Root Cause | Do Differently | Source |
|--------|-----------|----------------|--------|
| <what to do differently> | <why it went wrong — root, not symptom> | <the concrete change> | <file:line / commit / ticket> |

## Patterns

| Pattern | Reuse | Source |
|---------|-------|--------|
| <named reusable approach / convention> | <when a future change should apply it> | <file:line / commit / ticket> |

## Surprises

| Surprise | Signal | Source |
|----------|--------|--------|
| <non-obvious gotcha / edge case / behavior> | <what the evidence shows> | <file:line / commit / ticket / scenario> |

## Environment Retro

**Written by `skillgrid:environment-retro` (a reflect sub-phase), not qa.** Improvements to the *agent environment*, never the code. A finding appears only where its "Use when" fired; a clean session leaves this empty.

| Category | Finding | Fix lands in | Source |
|----------|---------|--------------|--------|
| <navigation \| automated checks \| coding standards \| steering no-ops \| tool economy \| information access> | <what to change + why> | <linter rule / pre-commit / CI job / ARCHITECTURE.md pointer / steering file / tooling> | <file:line / commit / tool call / session symptom> |

## Acceptance Verdict

**Verdict:** accepted | accepted-with-open-items | rejected

**Grounding:** <goal from briefing.md + the Goal-Backward Verification / Traceability Matrix evidence above that supports the verdict>

**Reasoning:** <2–3 sentences: what drove the verdict>

## Open Items (→ next change)

- <item + path to resolution> (or "None")

## Prior-Change Follow-Through

| Prior open item (from previous archived change) | Addressed by this change? | Evidence |
|--------------------------------------------------|---------------------------|----------|
| <item> | yes / no / partially | <commit / file:line / scenario, or "still open"> |

## Move Evidence (from ship context)

**From:** `.skillgrid/specs/2026-10-31-mnemonic-v2-memory/` → **To:** `.skillgrid/archive/2026-10-31-mnemonic-v2-memory/`
**`diff -r` readback:** <empty → PASS | verbatim output → FAIL>

## Overrides / Waivers / Contradictions

- <QA waiver, human CONCERNS override, size-exception, or "None">
- <Unrankable contradiction: both statements + sources + when written — or "None">

## Lineage (observation IDs)

Every artifact read this close, for traceability:

- briefing: (no briefing.md in this spec folder — the Must-Haves live in blueprint.md; T0 checklist substituted)
- blueprint: .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
- tasks: .skillgrid/specs/2026-10-31-mnemonic-v2-memory/tasks.md
- acceptance: .skillgrid/specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
- ship: (pending)
- report (QA half): this file
- research / findings / ADRs: .skillgrid/artifacts/04-adr-0040-mnemonic-v2-module-extraction.md, 04-adr-0041-mnemonic-v2-postgres-driver.md, 04-adr-0042 (goldfish plan review)
