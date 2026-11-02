# Mnemonic v2 — Code-Index Function
#
# The live symbol/edge graph of the project's working tree: tree-sitter extraction
# (with an optional LSP tier), dual-dialect storage, graph queries (callers/callees/
# impact/path), hybrid search (FTS + structural + semantic, RRF), and a live watch loop.
# The agent uses it to find files, find where a feature is implemented, see how a function
# connects to others, and know the blast-radius of a change — with every answer
# confidence-labeled and every degradation explicit. The knowledge-base + plugins consume
# it via the code_* surface; this change covers the shared machinery the KB index builds on
# (D-KB) but not the KB's kb_* tables.
#
# Source: .skillgrid/specs/2026-10-31-mnemonic-v2-code-index/specs/02-code-index/ (000–008)
# Trace: blueprint.md Must-Haves (T1–T21) + per-task SATISFIES lines + the four
# live-probe relevance truths (T-CI-1…4, findings.md). The scenarios below mirror the
# blueprint's SATISFIES names 1:1.

## Requirements

### Requirement: door check and additive store extension

The system SHALL extend the memory change's Store with the code-index methods (additive —
no existing signature changes), add the 7 code-index tables as additive migrations on the
shared project DB, mint cross-machine sync ids for files and symbols, and — as the door
check — write files+symbols+edges through the extended Store (SQLite, WAL) and find a saved
symbol by name via the FTS floor.

#### Scenario: door-check-code

```gherkin
      Given a clean data directory
      When I upsert a file internal/pay/pay.go and its symbol CapturePayment
      And I add a name-only calls edge to CapturePayment
      And I look up the symbol by the name CapturePayment
      Then exactly one symbol is returned with its stable uid
      And the file and symbol each carry a sync id in the fixed format
```

#### Scenario: store-extends-additive

```gherkin
      Given a store opened after the code-index extension
      When I save a memory observation and upsert a code-index file in the same store
      Then the observation round-trips with its obs- sync id
      And the code-index file is present in the same database
```

#### Scenario: single-node-table

```gherkin
      Given an edge from a route node to a function node
      When I read the edge
      Then both endpoints reference the single symbols table by id
      And no polymorphic id space or temporal window columns exist
```

#### Scenario: sync-id-format-files-symbols

```gherkin
      When I mint a sync id for the file and symbol prefixes
      Then every id matches its prefix followed by exactly 16 hex characters
      And no two mints in the same burst are equal
```

#### Gates
````
# G1: door check + additive extension + single node table + sync-id format
# CHECK: cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestDoorCheckCodeIndexWritesAndSearches|TestStoreExtensionAdditive|TestCodeIndexSyncIDMintFormat' -count=1
# EXPECT: ok  (go test success-only marker)
# EVIDENCE: 2026-11-02 G1 ran green (fee9db76): all three tests PASS — the sync-id scenario is covered by TestCodeIndexSyncIDMintFormat (the gate's original name TestNewSyncIDFormat never existed; R-3 re-points the gate, scenario unchanged)
````

### Requirement: migration parity and the reusable FTS unit

The system SHALL run the full code-index migration set from an empty DB (both dialects),
keep it additive across reopens, and — as the v1 bug-class regression (T-CI-3) — let a store
opened at each prior migration version run every code_* read without a SQL-column error. The
FTS rendering is a reusable unit the KB's kb_pages_fts re-uses verbatim (T21).

#### Scenario: migration-runs-from-empty

```gherkin
      Given a clean data directory
      When I open the store for project p and close it
      And I open it again
      Then the second open succeeds without re-running applied migrations
```

#### Scenario: migration-parity-all-versions

```gherkin
      Given a store opened at each prior migration version in the ordered set
      When I run every code_* read against each versioned store
      Then no read fails with a missing-column SQL error
      And a store at a lower schema version returns a version error, not no such column
```

#### Scenario: fts-stays-in-sync-code

```gherkin
      Given a chunk whose text mentions trigram recall
      When I search the chunk full-text for trigram
      Then the chunk is the only hit
      And when I update the chunk text to Bigger trigrams
      Then searching for Bigger returns it and searching for trigram returns nothing
```

#### Scenario: short-term-like-fallback-code

```gherkin
      Given a symbol named Go
      When I search for the two-rune term go
      Then the symbol is found via the substring fallback below the trigram minimum
```

#### Scenario: fts-unit-reusable

```gherkin
      When I render an external-content FTS table and its trigger set for the KB pages table
      Then exactly one FTS table and four sync triggers are produced
      And the same unit renders the code-index chunks table identically
```

#### Gates
````
# G2: migration parity (T-CI-3) + from-empty + FTS sync + LIKE fallback + reusable unit (T21)
# CHECK: cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestMigrationParityAllVersions|TestCodeMigrationRunsFromEmptyAndReopens|TestFtsUnitReusable|TestFTSStaysInSyncCode|TestShortTermLikeFallbackCode' -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: extraction ladder and the real extractor stamp

The system SHALL extract the working tree via the tree-sitter ladder (6 rich one-pass +
25 map + regex fallback), self-heal a panicking grammar via the per-language circuit
breaker, and — as the D-STAMP fix — compute the extractor stamp as a real content hash so a
changed extractor forces a full re-walk. The chunker is AST-first with a lines fallback, and
oversized files are a first-class outcome.

#### Scenario: extract-ladder-3-tiers

```gherkin
      Given a Go file with a method and a call
      When I extract it
      Then the tier is rich and the method symbol is emitted
      And a statically-bound call is EXTRACTED while a name-only call is kept with a null target and an unresolved row
      And a Python file extracts via the rich or map tier and an unmapped file falls to the regex tier
```

#### Scenario: self-healing-pool-circuit-breaker

```gherkin
      Given a pool configured with three respawns and trip-after-three
      When three go files panic the grammar
      Then the go circuit breaker trips
      And a fourth go file goes straight to the regex tier
```

#### Scenario: extractor-stamp-content-hash

```gherkin
      When I compute the extractor stamp for two different extractor configs
      Then the two stamps differ
      And the stamp is a content hash, not a constant
```

#### Scenario: chunker-ast-first

```gherkin
      Given a file with one long method and a config file with no symbols
      When I chunk both
      Then the method is one ast chunk under the target size and the config file is plain line chunks
```

#### Scenario: knowledge-node-classification

```gherkin
      Given a markdown file, a yaml file, a sql file, and an unmapped file
      When I classify each
      Then the markdown is a doc, the yaml is a config, the sql is a table
      And the unmapped file is skipped, not classified
```

#### Scenario: max-file-size-oversized

```gherkin
      Given a file larger than the configured max size
      When I extract it
      Then the file is not indexed and is counted as oversized
```

#### Gates
````
# G3: extraction ladder + circuit breaker + real stamp + chunker + knowledge nodes + oversized
# CHECK: cd mnemonic-v2 && go test ./internal/codeindex/extract/ -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: graph reads — impact, neighbors, resolve, packages, path, affected

The system SHALL answer the graph reads: risk-tiered blast radius, confidence-labeled
neighbor views, the NEW code_resolve (the KB pin-check contract), the NEW code_packages
(the KB coverage-gap contract), the shortest-path-or-graph-stops read, and the affected-tests
read (with the Java test-convention fix).

#### Scenario: impact-blast-radius

```gherkin
      Given a graph where C calls B calls A and D has a name-only edge to nothing
      When I compute the impact of A
      Then B is WILL BREAK and C is LIKELY AFFECTED
      And D is not in the radius because only stored edges are traversed
      And a higher min-confidence run counts the excluded hops rather than dropping them
```

#### Scenario: neighbors-confidence-labeled

```gherkin
      Given a symbol with callers and callees
      When I read its callers and callees
      Then every edge carries a confidence label and a resolved or name-only target
      And a name-only edge with multiple candidates returns one AMBIGUOUS edge per candidate
```

#### Scenario: code-resolve-contract

```gherkin
      Given a symbol CapturePayment that exists and is called from enterprise.go
      When I resolve CapturePayment
      Then the result has exists true, its file, line, sync_id, and called_from listing the enterprise.go file path
      And resolving a name with two symbols returns ambiguous true with two candidates
      And resolving a deleted symbol returns exists false
```

#### Scenario: code-packages-contract

```gherkin
      Given top-level Go packages internal/memory and internal/codeindex
      When I read the package map
      Then each package is reported with its path, file count, and symbol count
      And a sub-package is not reported as its own unit
```

#### Scenario: path-graph-stops

```gherkin
      Given a path from A to an unreachable symbol that stops at an interface dispatch
      When I compute the path
      Then the result is either a path or a graph-stops answer naming the interface dispatch kind
      And the dispatch kind is the boundary at the frontier, not a last-iterated value
```

#### Scenario: affected-tests-java

```gherkin
      Given a changed Java file Foo.java and its test FooTest.java
      When I compute the affected tests
      Then FooTest.java is reported as an affected test
```

#### Scenario: name-only-query-time-relabel

```gherkin
      Given a name-only edge whose target name matches two symbols
      When I read the callers of one of those symbols
      Then the ambiguous name-only edge is relabeled AMBIGUOUS at query time
```

#### Scenario: orient-map-navigable

```gherkin
      Given an indexed package internal/pay whose symbols call symbols in another file
      When I orient on the package with depth one
      Then the result is a bounded map with the symbol TOC and outbound edges carrying stable ids
      And the outbound edge targets are not expanded inline at depth one
      When I re-orient on an outbound edge target id
      Then I get that symbol's own neighborhood map
      And the cost stays flat as the codebase grows (one layer per query, never the whole vault)
```

#### Gates
````
# G4: graph reads — impact, neighbors, resolve, packages, path, affected + the orient map (T9b)
# CHECK: cd mnemonic-v2 && go test ./internal/codeindex/graph/ -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: hybrid search with the live-probe relevance fixes

The system SHALL fuse the FTS, structural, and semantic legs by RRF (K=60) with per-hit
provenance, and — as the T-CI-1/T-CI-2 live-probe fixes — return the real symbol for a real
symbol (real differs from fake), let a broken leg degrade out of the fusion, fix the
structural grep pattern language, redact secrets in code_read, and set degraded:no_vector
when the embedder is absent. It SHALL also narrow a search to a subtree via the `under`
package-prefix scope (the `ki search --under` context-rot defense), and code_read SHALL
offer a `content` drill-handle mode that returns a symbol's signature + callee/caller ids
without their bodies (the `ki get --type content` progressive-disclosure read).

#### Scenario: search-hybrid-rrf

```gherkin
      Given a query that matches a chunk, a symbol, and a semantic neighbor
      When I run the hybrid search
      Then the hit in the most legs ranks highest and every hit carries per-signal provenance
```

#### Scenario: search-real-vs-fake-divergence

```gherkin
      Given a real symbol wireGitHooks and a non-existent symbol capturePayment
      When I search for each
      Then the real symbol appears in the FTS leg
      And the real and fake searches do not return the same top hit
```

#### Scenario: broken-leg-degrades-out

```gherkin
      Given a semantic leg whose query errors
      When I run the hybrid search
      Then the result is degraded for that leg and the leg is dropped from the fusion
      And the FTS floor still answers with hits
```

#### Scenario: structural-grep-pattern-language

```gherkin
      Given a Go file
      When I grep with the node-type capture form and with the f shorthand
      Then both forms return the matching declarations
      And an invalid pattern for the language degrades to no_grammar for that language only
```

#### Scenario: code-read-secret-redaction

```gherkin
      Given a file containing a token and a password literal
      When I read the file span
      Then the secret values are masked and the keys remain
```

#### Scenario: search-under-scope

```gherkin
      Given a query retry that matches a chunk under internal/pay and a chunk under internal/billing
      When I search for retry with under set to internal/pay
      Then every hit is under the internal/pay prefix and none leaks outside it
      When I search for retry with under set to a path that matches nothing
      Then the search returns no hits and not an error
```

#### Scenario: code-read-content-mode

```gherkin
      Given a symbol CapturePayment that calls a symbol in another file
      When I read CapturePayment in content mode
      Then the result carries the signature and the callee and caller ids but none of their bodies
      When I read a file in content mode
      Then the result is the file TOC, not the file body
```

#### Scenario: degraded-no-vector

```gherkin
      Given no embedder configured
      When I run the hybrid search
      Then the result is degraded no_vector and the FTS floor still answers
```

#### Gates
````
# G5: hybrid search + T-CI-1 (divergence) + T-CI-2 (broken leg) + grep + read + no_vector
#      + the ki folds (under-scope, content-mode read)
# CHECK: cd mnemonic-v2 && go test ./internal/codeindex/search/ -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the live watch loop

The system SHALL keep the graph live via an fsnotify watcher with debounce, burst-collapse,
the metadata-only fingerprint re-check, and failure isolation — a trigger, not a gate: a
query during a re-index sees the last committed index and never blocks.

#### Scenario: watch-debounce-burst-collapse

```gherkin
      Given a watcher with a short debounce window
      When five edits land within the window
      Then exactly one re-index callback fires
```

#### Scenario: watch-fingerprint-recheck

```gherkin
      Given a file whose mtime changed but whose content did not
      When the watcher re-checks after the change
      Then no re-index is triggered because the file hash is unchanged
```

#### Scenario: watch-failure-isolation

```gherkin
      Given a re-index that fails with a store lock
      When the watcher processes the failure
      Then the watcher keeps running and the next change still fires a re-index
```

#### Scenario: watch-trigger-not-gate

```gherkin
      Given a re-index in flight
      When a freshness check runs during the re-index
      Then the check returns the last committed index immediately without blocking
```

#### Gates
````
# G6: watch loop — debounce/burst-collapse + re-check + failure isolation + trigger-not-gate
# CHECK: cd mnemonic-v2 && go test ./internal/codeindex/watch/ -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the code_* transport and the code_status report

The system SHALL expose the code_* family (16 agent-profile + the admin set) on the memory
MCP server's two-profile mechanism, queue code_index as a write (409 on a repeat), declare
the v2.1-deferred tools (reporting run code_index --flag), keep the CLI and MCP byte-
identical, and — as the T-CI-4 fix — have code_status report the 4-state freshness, the
per-language extractor tier, and the schema version.

#### Scenario: code-tool-contract

```gherkin
      When I list the registered code_* tools
      Then every agent tool name is present with its required params and return shape
      And code_resolve returns the pin-check shape with exists, called_from, and candidates
      And code_packages returns the package map with path, files, and symbols
```

#### Scenario: code-profile-membership

```gherkin
      When I list the tools for the agent profile and the all profile
      Then the all profile is a superset of the agent profile
      And admin-only tools like code_rename appear in all but not in agent
```

#### Scenario: code-status-reports-version-tier-coverage

```gherkin
      Given an indexed project with a stale file and per-language tiers
      When I read code_status
      Then the freshness is the 4-state value lag, not a boolean
      And the per-language extractor tier is reported for each language
      And the schema version and fair coverage are reported
```

#### Scenario: code-index-queued-write-409

```gherkin
      When I start a code_index run
      Then a run id and status are returned immediately
      And a second code_index while the first is active returns a 409 conflict
```

#### Scenario: v21-deferred-report-run-flag

```gherkin
      When I call a v2.1-deferred tool such as code_pdg_query
      Then it reports run code_index with the matching flag
      And it is declared in the surface, not missing
```

#### Scenario: mcp-cli-parity

```gherkin
      Given the same search query via the CLI and via the MCP tool
      When I compare the JSON output
      Then the CLI and MCP output are byte-identical
```

#### Gates
````
# G7: code_* transport + T-CI-4 (code_status report) + queued write + v2.1 deferred + parity
# CHECK: cd mnemonic-v2 && go test ./internal/transport/... ./cmd/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the semantic vector store and the advanced passes

The system SHALL ship a wired 3-index semantic store (BLOB + vec0 + HNSW + brute-force
floor, D-CI8) with the HNSW-preferred order and the model-swap guard, drop LSH + IVF,
provide the optional LSP tier (all-or-nothing per call site), and render the Postgres
dialect of the code-index tables.

#### Scenario: semantic-store-wired-3-index

```gherkin
      Given a 768-dim embedder and an embedded symbol
      When I run a semantic search
      Then HNSW is the preferred index and the result has hits
      And neither LSH nor IVF appears in the preference order
```

#### Scenario: vector-preference-order

```gherkin
      Given an embedded symbol with HNSW and then vec0 disabled
      When I run a semantic search
      Then it falls to vec0 and then to the brute-force floor, never a hard fail
```

#### Scenario: model-swap-guard

```gherkin
      Given vectors embedded under model A
      When the embedder swaps to model B
      Then all vectors and derived indexes are cleared
```

#### Scenario: lsp-all-or-nothing-per-call-site

```gherkin
      Given a member call site the static pass could not bind
      When the LSP tier resolves it
      Then the edge is LSP_RESOLVED
      And when the LSP server fails the static name-only edge stands, never a partial edge set
```

#### Scenario: postgres-dialect-render

```gherkin
  # Gated on MNEMONIC_PG_TEST_DSN (gate G-15)
      Given a Postgres store opened from an empty schema
      When I run the code-index migrations and a code_* read
      Then the read succeeds and the walker is dialect-agnostic
```

#### Gates
````
# G8: semantic store (D-CI8) + preference order + model swap + LSP + Postgres render
# CHECK: cd mnemonic-v2 && go test ./internal/codeindex/vectorstore/ ./internal/codeindex/extract/ -run 'Semantic|Vector|ModelSwap|LSP' -count=1
# EXPECT: ok  (PG test skips without MNEMONIC_PG_TEST_DSN)
# EVIDENCE: pending
````
