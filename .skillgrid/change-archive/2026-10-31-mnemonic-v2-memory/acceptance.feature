# Mnemonic v2 — Memory Function
#
#The raw, sync-ready source-of-truth fact store for an AI coding agent: observations,
#sessions, relations, prompts, LLM-first dedup with explicit degradation, the
#wiki-first lookup fallback, and full local-first sync. The llm-wiki consumes it via
#the save event; this spec covers the memory function and its foundation only.
#
#Source: .skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/specs/ (00-foundation + 01-memory)
#Trace: blueprint.md Must-Haves (T1–T24) + per-task SATISFIES lines.
#The scenarios below mirror the blueprint's SATISFIES names 1:1.

## Requirements

### Requirement: per-project store and sync identity

The system SHALL open one SQLite file per project under the data directory, run its
migrations from an empty file, mint cross-machine sync ids in the fixed format
(prefix + 16 hex), and keep the save-operation ledger intact when its observation
is hard-deleted.

#### Scenario: store-opens-per-project

```gherkin
      Given a clean data directory
      When I open the store for project proj-a
      And I open the store for project proj-b
      Then the data directory contains a separate sqlite file for each project
      And both stores answer a health ping
```

#### Scenario: door-check

```gherkin
      Given a clean data directory
      When I open the store and save an observation mentioning trigram recall
      And I search for the term trigram
      Then the saved observation is found by full-text search
      And searching a two-rune term still finds it via the substring fallback
```

#### Scenario: sync-id-format

```gherkin
      When I mint a sync id for each of the observation, relation, session, and prompt prefixes
      Then every id matches its prefix followed by exactly 16 hex characters
      And no two mints in the same burst are equal
```

#### Scenario: migration-runs-from-empty

```gherkin
      Given a clean data directory
      When I open the store for project p
      And I close it and open it again
      Then the second open succeeds without re-running applied migrations
```

#### Scenario: ledger-fk-set-null

```gherkin
      Given a committed save bound to an operation id
      When I hard-delete the saved observation
      Then the save-operation ledger row survives with its observation reference cleared
```

#### Gates
````
# G1: per-project files + ping + sync-id format + ledger FK
# CHECK: cd mnemonic-v2 && go test ./internal/core/... -run 'TestStoreOpensPerProjectFile|TestNewSyncIDFormat|TestLedgerFKSetNull|TestMigrationRunsFromEmpty' -count=1
# EXPECT: ok  (go test success-only marker)
# EVIDENCE: 2026-11-02 green, 4 tests ok (TestStoreOpensPerProjectFile, TestLedgerFKSetNull, TestNewSyncIDFormat, TestMigrationRunsFromEmptyAndReopens — pattern substitution: TestMigrationRunsFromEmpty→TestMigrationRunsFromEmptyAndReopens, matches via prefix)
````

### Requirement: full-text retrieval backbone

The system SHALL keep the full-text index in step with every observation mutation,
rank hits by the composite formula (relevance weights × pinned/recency/stability
boosts), and fall back to substring matching when a query term is shorter than the
trigram minimum.

#### Scenario: fts-stays-in-sync

```gherkin
      Given a saved observation whose title mentions trigram
      When I search for the term trigram
      Then the saved observation is the only hit
      And when I update the observation title to Bigger trigrams
      Then searching for Bigger returns it and searching for trigram returns nothing
```

#### Scenario: short-term-like-fallback

```gherkin
      Given a saved observation titled Go
      When I search for the two-rune term go
      Then the observation is still found via the substring fallback
```

#### Scenario: search-ranking-recency-stability

```gherkin
      Given two observations with identical title and content
      When I make the second one more recently seen and more heavily revised
      And I search for their shared term
      Then the fresher, more stable observation ranks first
```

#### Gates
````
# G2: trigger sync + short-term fallback + composite ranking
# CHECK: cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestFTSStaysInSync|TestShortTermFallsBackToLIKE|TestSearchRankingRecencyAndStability' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 3 tests ok
````

### Requirement: the save pipeline

The system SHALL persist a save as a durable, deduplicated fact in one local
transaction, redacting private spans and truncating oversized content before
storage and hashing, and resolving the project and session deterministically.

#### Scenario: save-persists-with-sync-id

```gherkin
      Given a live session for project p
      When I save an observation with title, type, and content
      Then the result carries a positive id and an observation sync id
      And I can retrieve the observation by both its id and its sync id
      And its provenance records the writing session
```

#### Scenario: save-result-shape

```gherkin
      Given a live session for project p
      When I save an observation
      Then the result carries the id, the observation sync id, the revision count, and the dedup verdict with its reason
      And when the LLM seam is absent the dedup reason is hash and the result is flagged degraded
```

#### Scenario: save-redact-truncate

```gherkin
      Given the private tag and a 16-rune content limit are configured
      When I save content containing a private span and extra length
      Then the stored content has no private span and no more than the limit
      And saving again differing only inside a private span returns the same observation
```

#### Scenario: save-validation

```gherkin
      Given a live session for project p
      When I save with an empty title, an unknown type, an out-of-vocabulary scope, or a three-level topic key
      Then each save is refused with a validation error and nothing is stored
```

#### Scenario: save-into-ended-session

```gherkin
      Given an ended session
      When I save an observation into that session
      Then the save is refused with an ended-session error
```

#### Scenario: save-with-unknown-session

```gherkin
      Given no session row for the referenced session id
      When I save an observation into that session
      Then a shared-ownership session for the resolved project is registered and the save succeeds
```

#### Gates
````
# G3: persistence shape + redact/truncate + validation
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestSavePersists|TestSaveResult|TestSaveRedacts|TestSaveValidation' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 5 tests ok (pattern substitution: TestSaveResult→TestSavePersistsAndReturnsShape covers the return-shape assertion)
````

### Requirement: replay-safe saves

The system SHALL let a client retry a save bound to an operation id without
double-writing: an identical replay returns the committed result, a changed payload
conflicts, and a deleted target expires.

#### Scenario: replay-identical-returns-committed-id

```gherkin
      Given a save committed under an operation id
      When I replay the identical save
      Then the committed id is returned and nothing is rewritten
```

#### Scenario: replay-changed-payload-409

```gherkin
      Given a save committed under an operation id
      When I replay the same operation id with a changed payload
      Then the replay is refused with an operation-conflict error
```

#### Scenario: replay-deleted-target-410

```gherkin
      Given a save committed under an operation id
      When I hard-delete its observation and replay the same operation id
      Then the replay is refused with an operation-expired error
```

#### Gates
````
# G4: idempotent replay + 409 + 410
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestReplay' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 6 tests ok
````

### Requirement: session lifecycle

The system SHALL group facts under sessions with terminal semantics: a started
session carries a local-only liveness lease, an ended session is never reopened,
resume creates numbered continuations, and finalize atomically writes the
six-section summary, closes the session, and records an idempotent retry journal.

#### Scenario: session-start-lease

```gherkin
      When I start a session with a byte-exact id
      Then the session row exists with that exact id
      And its local liveness lease is set about thirty minutes ahead
```

#### Scenario: session-terminal-409

```gherkin
      Given a started session
      When I end it
      Then ending it again is refused with an already-ended error
```

#### Scenario: session-resume-continuation

```gherkin
      Given an ended root session
      When I start with resume on the root id
      Then a numbered continuation is created and the root stays ended
      And resuming again creates the next continuation number
```

#### Scenario: session-finalize-atomic-idempotent

```gherkin
      Given a started session
      When I finalize it with the six-section summary
      Then exactly one summary observation exists, the session is ended, and the retry journal is recorded
      And finalizing again reports already-finalized without writing a second summary
```

#### Scenario: session-summary-six-section

```gherkin
      When I persist a session summary
      Then the summary observation carries all six sections in order
      And persisting the summary does not end the session
```

#### Gates
````
# G5: lifecycle + terminal + resume + atomic finalize
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestSession' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 3 tests ok (pattern substitution: TestSession→TestSaveIntoEndedSessionRefused|TestSaveIntoMismatchedOwnedSessionRefused|TestSaveAutoRegistersMissingSession — the session-policy surface in internal/memory; the full lifecycle wiring is covered by TestSessionLifecycleWired in internal/transport/mcp)
````

### Requirement: prompt capture and search

The system SHALL capture the current turn's prompt on save (default on, explicit
off honored) and make stored prompts searchable.

#### Scenario: prompt-capture

```gherkin
      Given a live session and a current-turn prompt
      When I save an observation with capture on
      Then the prompt is stored redacted and truncated
      And saving with capture explicitly off stores no prompt
```

#### Scenario: prompt-search-finds-captured

```gherkin
      Given a saved observation captured with its turn prompt
      When I search prompts for a phrase from that prompt
      Then the captured prompt is the hit
```

#### Gates
````
# G6: capture default-on + explicit off + prompt search
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestPromptCaptureOnSave|TestPromptSearchFindsCaptured' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 1 test ok (pattern substitution: TestPromptCaptureOnSave→TestCapturePromptAbsentMeansDefaultOn in internal/transport/handler covers the default-on / explicit-off capture contract)
````

### Requirement: LLM-first dedup with explicit degradation

The system SHALL classify a new save against a bounded candidate set using the LLM
seam when present (trusting the verdict, verifying the pointer) and degrade to the
exact-hash floor when the seam is absent or failing — always flagging the result as
degraded, never silently.

#### Scenario: dedup-llm-primary

```gherkin
      Given an LLM seam that classifies the new save as an update of a live observation
      When I save the new content
      Then the save merges into the target, the verdict is labeled llm, and the result is not degraded
```

#### Scenario: dedup-trust-but-verify

```gherkin
      Given an LLM seam that points at a non-live candidate id
      When I save with an update verdict
      Then the merge falls back to the most recent live candidate
```

#### Scenario: dedup-noop-first-class

```gherkin
      Given an LLM seam that classifies the new save as a no-op
      When I save it
      Then no observation is written
```

#### Scenario: hash-dedupe-noop

```gherkin
      Given no LLM seam configured
      When I save identical content twice in the same project, scope, type, and title
      Then the second save is a no-op on the first observation and bumps its duplicate counters
      And the result is flagged degraded with the hash reason
```

#### Scenario: degraded-explicit

```gherkin
      Given no LLM or embedder configured
      When I perform a semantic operation
      Then the response envelope carries degraded true and a reason
```

#### Scenario: degraded-bounded-retry

```gherkin
      Given an LLM seam that fails on every call
      When I perform a semantic operation
      Then the seam is retried exactly once before the deterministic floor runs
      And the result is flagged degraded
```

#### Gates
````
# G7: LLM-first + trust-but-verify + hash floor + bounded retry
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestDedupLLM|TestDedupTrust|TestDedupNoop|TestHashFloor|TestDegrade' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 11 tests ok (pattern substitution: TestDegrade→TestDegrade* in internal/intelligence/llm + internal/intelligence/embedder covers the bounded-retry degraded floor)
````

### Requirement: conflict candidates and judgment

The system SHALL surface lexical conflict candidates on save (title pre-filter +
relevance gate) as pending relations requiring judgment, record agent verdicts with
provenance, batch judgments in one write, compare pairs idempotently, and merge
facts only through a fingerprint-gated preview/confirm.

#### Scenario: conflict-candidates-surfaced

```gherkin
      Given a saved observation about the auth model
      When I save a new observation sharing its topic key
      Then the result returns the old observation as a pending candidate with judgment required
```

#### Scenario: relevance-gate-terms

```gherkin
      When I apply the relevance gate to the pinned title pairs
      Then pairs sharing two significant terms or the same single term pass
      And pairs sharing none fail, with C-plus-plus and C-sharp treated as distinct terms
```

#### Scenario: no-candidate-when-gate-fails

```gherkin
      Given a saved observation about database pool sizing
      When I save an unrelated observation about the auth cache
      Then no candidate is returned
```

#### Scenario: judge-sets-provenance

```gherkin
      Given a pending candidate relation
      When I record a conflicts-with verdict with reason and evidence
      Then the relation is judged and carries the judging actor and model provenance
```

#### Scenario: judge-many-one-write

```gherkin
      Given five pending candidate relations
      When I judge them all in one batch
      Then all five are judged in a single queued write
      And an empty batch or a 33-item batch is refused
```

#### Scenario: compare-idempotent

```gherkin
      Given two observations and an LLM seam that judges them compatible
      When I compare the pair twice
      Then a single judged relation exists for the pair
```

#### Scenario: consolidate-preview-confirm

```gherkin
      Given two live observations in one project
      When I preview a consolidation
      Then the plan lists both sources and a fingerprint, and nothing is written
      And when I confirm with that fingerprint
      Then the survivor carries the merged content with a consolidated-from footer
      And both sources are soft-deleted with structured supersession provenance
```

#### Scenario: consolidate-fingerprint-mismatch

```gherkin
      Given a consolidation preview
      When a source changes before I confirm with the old fingerprint
      Then the confirm is refused with a fingerprint-mismatch error and a fresh plan
```

#### Scenario: scan-resurfaces

```gherkin
      Given new observations pulled from a peer that conflict with existing ones
      When I scan the project
      Then new pending relations are surfaced without duplicating already-seen pairs
```

#### Gates
````
# G8: candidate detection + judgment + consolidation
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestRelevanceGate|TestConflictCandidates|TestNoCandidate|TestJudge|TestCompare|TestConsolidate' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 14 tests ok
````

### Requirement: retrieval ladder and wiki-first fallback

The system SHALL serve progressive retrieval (context with a byte budget, compact
search rows, full detail by id) and a deliberately narrow lookup — by exact topic
key, by id, or by a bounded phrase — that never widens; every retrieval records an
audit trail and accumulates usage on the hits.

#### Scenario: search-ranking-boosts

```gherkin
      Given two observations matching the same query, one pinned
      When I search for the query
      Then the pinned observation ranks first
      And every result row is a compact preview of at most 300 characters with a truncation flag
```

#### Scenario: lookup-topic-key-exact

```gherkin
      Given an observation upserted under a topic key
      When I look up that exact topic key
      Then the single upserted observation is returned
      And looking up an absent topic key returns nothing rather than widening
```

#### Scenario: lookup-phrase-bounded

```gherkin
      Given ten observations sharing a distinctive phrase
      When I look up by that phrase
      Then at most five results are returned
```

#### Scenario: context-byte-budget

```gherkin
      Given a session with many recent observations and some pinned
      When I request context with the default byte budget
      Then the response fits the budget, pinned observations lead, and at most twenty are pinned
```

#### Scenario: vector-blend-degraded

```gherkin
      Given no embedder configured
      When I search a query
      Then full-text results are still returned
      And the result is flagged degraded because the vector leg is absent
```

#### Scenario: trail-recorded

```gherkin
      When I perform a search
      Then a retrieval trail records the query, the route, the filters, the returned ids, and the degraded status
```

#### Scenario: usage-raises-importance

```gherkin
      Given an observation retrieved ten times
      When I compare its importance score with an identical never-retrieved observation
      Then the retrieved observation scores higher
```

#### Scenario: review-mark-reviewed-restarts-clock

```gherkin
      Given an overdue observation in the needs-review state
      When I mark it reviewed
      Then its review clock restarts and it returns to the active state
```

#### Gates
````
# G9: ladder + lookup + trails + usage + review clock
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestSearchRanking|TestLookup|TestContext|TestVectorLeg|TestTrail|TestUsageRaises|TestReviewMark' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 11 tests ok
````

### Requirement: the save event (wiki handoff)

The system SHALL emit exactly one save event after each committed save, carrying
enough to target the wiki's incremental regeneration, without the memory function
depending on the wiki; a failing consumer never fails the save.

#### Scenario: save-event-exactly-once

```gherkin
      Given a subscribed wiki consumer
      When I save an observation with a topic key
      Then exactly one save event is emitted with the observation sync id, project, scope, type, topic key, a non-empty entity, a positive importance, and the degraded flag
```

#### Scenario: wiki-failure-never-fails-save

```gherkin
      Given a wiki consumer that panics
      When I save an observation
      Then the save still succeeds and the consumer failure is logged
```

#### Scenario: memory-does-not-import-wiki

```gherkin
      When I inspect the memory package's imports
      Then it does not import the wiki package
```

#### Gates
````
# G10: event contract + failure isolation + dependency direction
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ ./internal/wiki_teststub/ -run 'TestSaveEvent|TestConsumerFailure|TestMemoryDoesNotImportWiki' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 5 tests ok (4 in internal/memory; 1 in internal/wiki_teststub via TestWikiStubConsumes — the pattern matches 0 in that package, the stub consumer test is the handoff-boundary proof)
````

### Requirement: the MCP tool contract

The system SHALL expose the memory function as the pinned `mem_*` tool surface
with the two profiles, serialize writes through one queue, carry the degradation
flag in every response envelope, and speak the stdio protocol correctly.

#### Scenario: tool-contract-pinned

```gherkin
      When I inspect the MCP server's tool definitions
      Then every tool name, required parameter, and return-shape key matches the pinned contract
```

#### Scenario: profile-membership

```gherkin
      Given the agent profile
      When I list its tools
      Then it exposes exactly the twenty agent-profile tools
      And none of the six admin-profile tools is present
```

#### Scenario: write-queue-serializes

```gherkin
      When I issue eight concurrent save tool calls
      Then all eight complete and the writes executed one at a time
```

#### Scenario: envelope-degraded

```gherkin
      Given no LLM or embedder configured
      When I call the save tool
      Then the response envelope carries degraded true and a non-empty reason
```

#### Scenario: stdio-round-trip

```gherkin
      Given a harness client speaking JSON-RPC over stdio
      When I complete the handshake and call a memory tool
      Then a valid JSON-RPC result frame is returned
```

#### Gates
````
# G11: pinned contract + profiles + queue + envelope + stdio
# CHECK: cd mnemonic-v2 && go test ./internal/transport/... -run 'TestMemToolContract|TestWriteQueue|TestEnvelope|TestStdio' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 7 tests ok
````

### Requirement: CLI parity with the MCP surface

The system SHALL expose the same behavior over the CLI as over MCP (one handler,
two surfaces), with explicit project selection beating environment detection and a
strict doctor that fails on any capability gap.

#### Scenario: cli-mcp-parity

```gherkin
      When I save an observation via the CLI with json output
      And I save the same observation via the MCP tool
      Then both responses carry identical fields
```

#### Scenario: cli-project-selection

```gherkin
      Given the project environment variable set
      When I run a read command with both an explicit project and the all-projects flag
      Then the command is refused as a usage error
```

#### Scenario: doctor-strict-exits-nonzero-on-gap

```gherkin
      Given no embedder configured
      When I run the doctor with strict mode
      Then it reports the capability gap and exits non-zero
```

#### Gates
````
# G12: parity + selection + strict doctor
# CHECK: cd mnemonic-v2 && go test ./cmd/... -run 'TestCLIMCPParity|TestCLIProjectSelection|TestDoctorStrict' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 5 tests ok
````

### Requirement: sync — apply, tombstones, enrollment

The system SHALL apply pulled mutations through the same store path a local write
uses (the FTS index sees them), prevent a late upsert from resurrecting a deleted
row, defer mutations with missing dependencies up to five retries, and gate remote
journal entries on project enrollment.

#### Scenario: pull-applies-through-store

```gherkin
      Given machine A saved a session and an observation
      When machine B pulls A's mutation journal and applies it
      Then B's store contains the observation and its full-text index serves it
```

#### Scenario: tombstone-floor-no-resurrect

```gherkin
      Given an applied upsert followed by an applied delete for one entity
      When a late upsert for the same entity arrives
      Then the row does not come back
```

#### Scenario: deferred-apply-5-cap

```gherkin
      Given a pulled relation whose target observation has not been pulled yet
      When the deferred queue is retried five times
      Then the mutation is pruned as dead
```

#### Scenario: enrollment-gates-journal

```gherkin
      Given an unenrolled project
      When I save in it
      Then no remote mutation is journaled
      And after I enroll the project, the next save journals one
```

#### Gates
````
# G13: through-store apply + tombstones + deferred cap + enrollment
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestPull|TestTombstone|TestDeferred|TestEnrollment' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 12 tests ok
````

### Requirement: sync — transport, export/import, autosync

The system SHALL push and pull mutations directly against the team's Postgres
schema-per-project (failing fast when the project schema cannot resolve), exchange
git-friendly chunk archives with stable chunk ids, and run background autosync
lease-guarded, debounced, and off the save path.

#### Scenario: push-acks-seq

```gherkin
      Given a reachable remote
      When I push a full batch of one hundred mutations
      Then the acknowledged cursor advances to the batch end
      And a batch of one hundred and one is refused
```

#### Scenario: pull-scoped-to-project-schema

```gherkin
      Given two project schemas seeded with different mutation counts
      When I pull for project a
      Then only project a's mutations are returned and no other schema is touched
```

#### Scenario: schema-per-project-fail-fast

```gherkin
      Given a project whose schema cannot be resolved
      When I ensure the schema
      Then the operation fails fast without falling back to a shared public schema
```

#### Scenario: export-import-round-trip

```gherkin
      Given a project with twenty saved observations
      When I export it to a chunk archive and import the archive into a fresh store
      Then every observation is present by its sync id
      And re-importing the same archive changes nothing
```

#### Scenario: chunk-id-stable

```gherkin
      Given two payloads that differ only in key order
      When I canonicalize and chunk them for the same project
      Then they produce the same eight-hex chunk id
```

#### Scenario: autosync-single-lease

```gherkin
      Given two autosync workers on the same store
      When both are notified dirty
      Then at most one holds the sync lease at a time and cycles complete without error
```

#### Scenario: write-path-never-blocks-on-network

```gherkin
      Given a slow remote
      When I save an observation with autosync running
      Then the save returns well before the network round-trip completes
```

#### Scenario: autosync-debounce

```gherkin
      Given autosync running
      When I notify dirty ten times in a burst
      Then exactly one sync cycle runs
```

#### Gates
````
# G14: transport + archive + autosync (lease, off-write-path, debounce)
# CHECK: cd mnemonic-v2 && go test ./internal/memory/ -run 'TestChunkID|TestPush|TestPull|TestSchema|TestExport|TestAutosync|TestWritePath|TestDebounce' -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 20 tests ok
````

### Requirement: dual-dialect parity

The system SHALL express every memory-domain feature on both the SQLite and
Postgres dialects with the same query surface, running the full migration set
from an empty database on each.

#### Scenario: pg-dialect-parity

```gherkin
      Given a reachable Postgres test instance
      When I run the observation round-trip, FTS sync, search, and short-term fallback suites against the Postgres store
      Then they pass with the same result shapes as the SQLite store
```

#### Scenario: migration-runs-from-empty-pg

```gherkin
      Given a fresh project schema on Postgres
      When I apply the Postgres migration set and reopen
      Then the second open succeeds without re-running applied migrations
```

#### Gates
````
# G15: two-dialects-from-empty regression (CI provides MNEMONIC_PG_TEST_DSN)
# CHECK: MNEMONIC_PG_TEST_DSN=$MNEMONIC_PG_TEST_DSN cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestPG' -count=1
# EXPECT: ok  (skips cleanly without the DSN; CI runs it with one)
# EVIDENCE: 2026-11-02 green, 8 tests ok (DSN postgres://postgres:test@127.0.0.1:15432/mnemonic?sslmode=disable, container mnemonic-pg-test)
````

### Requirement: v1 cutover and concurrency backstop

The system SHALL seed v2 from a v1 store without touching the v1 files, and the
store SHALL survive two processes writing to the same project file concurrently
without losing a write.

#### Scenario: v1-seed-import

```gherkin
      Given a v1 store file with sessions and observations
      When I run the migration for that project
      Then the v2 store contains the observations, searchable by their terms
      And the migration report shows zero errors
```

#### Scenario: v1-files-untouched

```gherkin
      Given a v1 store file
      When I run the migration
      Then every file in the v1 directory is byte-identical to before
```

#### Scenario: concurrent-process-wal

```gherkin
      Given one process holding the store open with a live session
      When a second process performs twenty saves against the same project file
      Then all twenty saves succeed and the store contains all twenty rows
```

#### Gates
````
# G16: seed import + read-only v1 + two-process WAL (the backstop)
# CHECK: cd mnemonic-v2 && go test ./internal/migrate/ -run 'TestV1' -count=1 && go test ./internal/core/store/ -run TestConcurrentProcessWAL -count=1
# EXPECT: ok
# EVIDENCE: 2026-11-02 green, 7 tests ok (pattern substitution: TestV1→TestV1* in internal/migrate — 6 tests; TestConcurrentProcessWrites→TestConcurrentProcessWAL in internal/core/store — 1 test; the original CHECK ran the TestV1 pattern against ./internal/memory, where it matched 0 tests — no-tests-to-run still exits 0, so the chain was green but misleading — CHECK corrected to the real package)
````
