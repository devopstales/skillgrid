# Mnemonic memory improvements
#
# Source: .skillgrid/specs/2026-09-24-mnemonic-memory-improvements/
# Trace: briefing success criteria; blueprint SATISFIES lines; ADR-0018.

## Requirements

### Requirement: keyword-floor-search

`mem_search` SHALL keep BM25 order and report `matched_via=keyword` when no embedder is active. A private observation SHALL stay invisible to a different reader.

#### Scenario: mem_search without an embedder is keyword only

```gherkin
      Given observations saved by reader A and MNEMONIC_EMBED is unset
      When reader A calls mem_search
      Then each hit has matched_via "keyword"
      And signals.vector is 0
      And the order matches BM25
```

#### Scenario: owner blend hides another owner's private row

```gherkin
      Given a private observation owned by reader A
      When reader B calls mem_search with the same keywords
      Then reader A's observation is absent
```

#### Scenario: mem_search without a query is rejected

```gherkin
      Given an open memory service
      When mem_search is called with no query
      Then the tool result is an error
```

#### Gates

G1: keyword floor and private-row hide
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ -count=1 -run 'TestOwnerScopedBlendKeywordFloor|TestOwnerScopedBlendHidesPrivate|TestMemSearchSignalsKeyword'
  EXPECT: PASS
  EVIDENCE: pending
G2: missing query is an error
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -count=1 -run 'TestMemSearchRequiresQuery'
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: hybrid-rrf-signals

When an embedder is active and the query embeds, `mem_search` SHALL fuse the owner-scoped FTS leg and the vector leg with RRF k=60 and SHALL return per-signal scores in [0,1].

#### Scenario: mem_search with an embedder returns hybrid signals

```gherkin
      Given MNEMONIC_EMBED=1 and observations that have embeddings
      When mem_search runs for a query the embedder can embed
      Then at least one hit has matched_via "hybrid" or "vector"
      And every hit has signals.keyword, signals.vector, signals.recency, signals.entity, signals.decay, and signals.importance in [0,1]
```

#### Scenario: embedder error degrades to keyword

```gherkin
      Given an embedder that returns an error
      When mem_search runs
      Then results still return
      And matched_via is "keyword"
```

#### Gates

G3: hybrid signals when embedder is on
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ -count=1 -run 'TestOwnerScopedBlendHybrid|TestMemSearchSignalsHybrid'
  EXPECT: PASS
  EVIDENCE: pending
G4: embedder error stays keyword
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -count=1 -run 'TestOwnerScopedBlendEmbedderError'
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: reinforcement-decay

A frequently accessed observation SHALL rank above an equally relevant observation that is rarely accessed, unless decay is disabled. Immunity SHALL freeze the half-life term at 1 when importance ≥ 4 or retrieval_usage ≥ 3.

#### Scenario: high retrieval_usage outranks a cold twin

```gherkin
      Given two observations with the same BM25 rank and LastSeenAt 40 days ago
      And one has retrieval_usage 20 and the other has retrieval_usage 0
      When mem_search runs with decay enabled
      Then the high-usage observation is first
```

#### Scenario: decay disabled keeps BM25 order

```gherkin
      Given the same pair and mnemonic.decay.enabled false
      When mem_search runs
      Then BM25 order is unchanged
```

#### Gates

G5: reinforcement decay orders the hot row first
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -count=1 -run 'TestReinforcementDecayRanksHotRow|TestDecayDisabledKeepsBM25'
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: query-embedding-cache

A repeated identical query SHALL skip a second embedder call for 7 days. A different model or a row older than 7 days SHALL embed again.

#### Scenario: second identical query is a cache hit

```gherkin
      Given a query was embedded and stored in query_cache
      When the same query is embedded again within 7 days for the same model
      Then the embedder is not called
      And the cached vector is returned
```

#### Scenario: stale or other-model cache misses

```gherkin
      Given a cache row older than 7 days or stamped with another model
      When the query is embedded
      Then the embedder is called
```

#### Gates

G6: query cache hit and miss
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -count=1 -run 'TestQueryCacheHit|TestQueryCacheMiss'
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: entity-aliases

The code index SHALL resolve a case-insensitive alias to the symbol `qualified_name` before falling back to FTS.

#### Scenario: alias lookup finds the symbol

```gherkin
      Given entity_aliases maps "the renderer" to "comp_renderer.cpp"
      When code search looks up "The Renderer"
      Then the qualified name "comp_renderer.cpp" is returned
```

#### Scenario: unknown alias falls through

```gherkin
      Given no alias for "missing"
      When code search looks up "missing"
      Then the alias lookup returns no rows and FTS still runs
```

#### Gates

G7: alias hit and miss
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/codeindex/ ./internal/mnemonic/store/ -count=1 -run 'TestEntityAliasLookup|TestMigration045'
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: compact-hook

The `compact` hook SHALL save one continuity observation from `CompactionContext` and SHALL fail open within 3 seconds without blocking compaction.

#### Scenario: compact hook saves continuity

```gherkin
      Given a session with recent observations and hooks enabled
      When RunHook compact runs for that session
      Then one observation with topic_key compaction/<session> exists
```

#### Scenario: compact hook swallows a slow save

```gherkin
      Given the compact work exceeds 3 seconds
      When RunHook compact runs
      Then the caller receives a result and compaction is not blocked
```

#### Gates

G8: compact hook save and fail-open
  CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -count=1 -run 'TestCompactHookSaves|TestCompactHookFailOpen'
  EXPECT: PASS
  EVIDENCE: pending
