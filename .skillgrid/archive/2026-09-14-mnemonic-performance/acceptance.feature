# Source: docs/skillgrid/changes/014-mnemonic-performance/change.md
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.

@step-01
Feature: Store handle caching with reference counting and WAL lock retry
  As a agent orchestrator
  I want store handles to be cached and reused with safe reference counting
  So that repeated opens do not create redundant connections and WAL writes do not contend

  @happy @p0
  Scenario: Cached handle reuse on second open
    Given a project store is open and tracked in the handle cache
    When I open the same project store again
    Then the same cached handle is returned
    And no new underlying connection is created

  @happy @p0
  Scenario: Handle closes only when reference count reaches zero
    Given a cached store handle with a reference count of one
    When I open the same project store a second time
    And I close the first handle
    Then the underlying connection remains open
    When I close the second handle
    Then the underlying connection is closed and evicted from the cache

  @edge
  Scenario: Cache disabled by environment variable
    Given the environment variable disabling the store cache is set
    When I open a project store twice
    Then each open creates a new underlying connection
    And no handle is stored in the cache

  @failure @p1
  Scenario: WAL lock retry with exponential backoff
    Given two concurrent writers contend on the same store
    When a write encounters a WAL lock
    Then the write retries at 50ms, 100ms, and 200ms intervals
    And the write succeeds after a successful retry
    And no panic is raised during retry

@step-02
Feature: FTS5 trigram and prefix match modes for search
  As a operator
  I want full-text search to support trigram and prefix matching
  So that I can find partial identifiers and use wildcard queries

  @happy @p0
  Scenario: Trigram mode splits query into three-character trigrams
    Given an observation containing the text "authentication middleware"
    When I search with trigram mode for "auth"
    Then the observation is returned in the results

  @happy @p0
  Scenario: Prefix mode appends wildcard to each search term
    Given an observation containing the text "authentication middleware"
    When I search with prefix mode for "auth"
    Then the observation is returned in the results

  @happy @p0
  Scenario: Default search mode remains phrase-only for backward compatibility
    Given an observation containing the text "authentication middleware"
    When I search with the default mode for the phrase "authentication middleware"
    Then the observation is returned in the results
    And partial word matches are not returned

  @edge
  Scenario: Trigram query with fewer than three characters
    Given an observation containing the text "go"
    When I search with trigram mode for "g"
    Then the search completes without error
    And the result set reflects the matching behavior for short inputs

  @failure @p1
  Scenario: Wildcard query falls back to LIKE for unsupported patterns
    Given an observation containing the text "handler"
    When I search with a wildcard pattern that cannot be expressed in FTS
    Then the query falls back to a LIKE-based match
    And the observation is returned in the results

@step-03
Feature: Concurrent cross-project search with bounded goroutines
  As a operator
  I want cross-project search to run in parallel with bounded concurrency
  So that searching many projects does not block sequentially

  @happy @p0
  Scenario: Cross-project search runs in parallel and merges by rank
    Given three project stores each containing matching observations
    When I search all projects in parallel
    Then all matching observations are returned
    And results are ordered by the highest rank across all stores
    And the search completes faster than sequential iteration

  @happy @p0
  Scenario: Concurrent goroutines are bounded by available CPUs
    Given ten project stores
    When I search all projects in parallel
    Then the number of concurrent store searches never exceeds the CPU count
    And all results are still returned

  @edge
  Scenario: Missing store is skipped with a warning
    Given three project stores where one is unavailable
    When I search all projects in parallel
    Then results from the two available stores are returned
    And a warning is emitted for the missing store
    And the search does not fail

  @failure @p1
  Scenario: Resource exhaustion under fifty stores
    Given fifty project stores
    When I search all projects in parallel
    Then the search completes without exceeding the concurrency bound
    And no goroutine leak is observed
    And all matching observations are returned

@step-04
Feature: Automatic TTL expiry with scheduled cleanup
  As a operator
  I want observations to expire automatically by default
  So that the store stays clean without manual intervention

  @happy @p0
  Scenario: Save sets expiry to seven days when not explicitly provided
    Given I save a new observation without an explicit expiry
    When the save completes
    Then the observation carries an expiry timestamp of now plus seven days

  @happy @p0
    Scenario: TTL cleanup retires expired observations
    Given an observation whose expiry timestamp is in the past
    When I run TTL expiry across all projects
    Then the expired observation is retired
    And non-expired observations remain active

  @happy @p0
  Scenario: Expired observations are excluded from search results
    Given an observation that has passed its expiry timestamp
    When I search for the observation
    Then the observation is not returned

  @edge
  Scenario: Explicit expiry overrides the default
    Given I save a new observation with an explicit expiry of one day
    When the save completes
    Then the observation carries the one-day expiry, not the seven-day default

  @failure @p1
  Scenario: TTL expiry does not retire observations that are not yet expired
    Given an observation whose expiry timestamp is in the future
    When I run TTL expiry across all projects
    Then the observation remains active and searchable
    And no data is lost

@step-05
Feature: LLM-backed passive extraction with regex fallback
  As a agent orchestrator
  I want passive extraction to use an LLM for higher quality results
  So that learnings are captured reliably even from free-form text

  @happy @p0
  Scenario: LLM extraction returns valid structured learnings
    Given a block of session text containing a key learnings section
    When I capture the text passively
    Then the LLM extracts structured learnings
    And each learning is shaped into the standard passive item format

  @happy @p0
  Scenario: Regex fallback activates when LLM is unavailable
    Given the LLM backend is unavailable
    And a block of session text containing a key learnings section
    When I capture the text passively
    Then the regex fallback extracts the learnings
    And each learning is shaped into the standard passive item format

  @edge
  Scenario: LLM returns malformed output
    Given the LLM backend returns a response that does not match the expected schema
    When I capture the text passively
    Then the system falls back to regex extraction
    And the capture completes without error

  @failure @p1
  Scenario: LLM extraction error triggers fallback without data loss
    Given the LLM backend returns an error during extraction
    When I capture the text passively
    Then the regex fallback produces the learnings
    And no session text is lost
    And the capture result contains the same quality or better than regex alone

@step-06
Feature: Multiple embedder providers via configuration
  As a operator
  I want to choose between local and remote embedding providers
  So that I can run offline or use a dedicated embedding service

  @happy @p0
  Scenario: Ollama provider embeds via the local Ollama API
    Given the configuration selects the Ollama embedder
    When I request an embedding for a text input
    Then the embedding is produced by the Ollama service
    And the result has the expected vector dimension

  @happy @p0
  Scenario: Local provider embeds via an ONNX model
    Given the configuration selects the local embedder
    And an ONNX model exists in the models directory
    When I request an embedding for a text input
    Then the embedding is produced by the local ONNX model
    And the result has the expected vector dimension

  @edge
  Scenario: Configuration falls back to null embedder when provider is unknown
    Given the configuration specifies an unrecognized embedder provider
    When I request an embedding
    Then the null embedder is returned
    And the embedding is an empty vector

  @failure @p1
  Scenario: Ollama service unreachable
    Given the configuration selects the Ollama embedder
    And the Ollama service is not running
    When I request an embedding
    Then the error is propagated clearly
    And no partial or corrupt embedding is stored

@step-07
Feature: Cross-linked relational, vector, and graph stores
  As a agent orchestrator
  I want observations to carry graph references and embeddings
  So that I can traverse from an observation to related code symbols and back

  @happy @p0
  Scenario: Cross-link query traverses observation to symbol to embedding to related observations
    Given an observation linked to a code symbol with an embedding
    And another observation also linked to a symbol with a similar embedding
    When I run a cross-link query starting from the first observation
    Then the query returns the second observation through the symbol-to-embedding bridge
    And the traversal path is observation to symbol to embedding to related observations

  @happy @p0
  Scenario: Every observation has a graph reference or a null default
    Given I save a new observation
    When the save completes
    Then the observation carries either a code symbol reference or a null default
    And no broken references exist in the store

  @edge
  Scenario: Observation has no matching code symbol
    Given I save an observation that does not reference any code symbol
    When the save completes
    Then the graph reference is null
    And the observation is still queryable through the relational store
    And existing query paths are unchanged

  @failure @p1
  Scenario: Orphaned symbol embedding is detected
    Given a symbol embedding exists without a corresponding code symbol
    When I run a cross-link query
    Then the orphaned embedding does not produce a false match
    And the join is correct with no dangling references

@step-08
Feature: Self-improvement feedback loop based on retrieval usage
  As a agent orchestrator
  I want frequently retrieved observations to rank higher over time
  So that the most useful memories surface first

  @happy @p0
  Scenario: High-usage observation is boosted in search ranking
    Given an observation with retrieval usage above the boost threshold
    And a similar observation with zero retrieval usage
    When I search for a query matching both
    Then the high-usage observation appears before the low-usage one

  @happy @p0
  Scenario: Never-accessed observation decays in rank over time
    Given an observation with zero retrieval usage and age beyond the TTL window
    And a recently accessed observation
    When I search for a query matching both
    Then the never-accessed observation appears after the recently accessed one

  @edge
  Scenario: Improve runs transparently before each search
    Given an observation whose usage has changed since the last search
    When I run a search
    Then the improve step is applied before ranking
    And the caller sees no additional API surface

  @failure @p1
  Scenario: Improve does not regress existing search results
    Given a set of observations with known baseline rankings
    When I enable the improve loop
    And I run the same search
    Then no observation that previously matched is dropped
    And relative ordering changes are only due to usage signals

@step-09
Feature: Session-to-graph auto-promotion
  As a agent orchestrator
  I want completed sessions with summaries to become permanent graph nodes
  So that session knowledge is queryable after the session ends

  @happy @p0
  Scenario: Session end with summary creates a permanent graph node
    Given a session ends with a populated summary
    When the session end handler runs
    Then a permanent graph node is created in the code index
    And the node links to all observations from the session

  @happy @p0
  Scenario: Promoted graph node is queryable through search and graph traversal
    Given a session was promoted to a graph node
    When I search for a term from the session summary
    Then the promoted node appears in the results
    And the node is traversable through the code index graph

  @edge
  Scenario: Session end with empty summary does not promote
    Given a session ends with an empty summary
    When the session end handler runs
    Then no graph node is created
    And the session observations remain in their original store

  @edge
  Scenario: Session end with summary below quality threshold does not promote
    Given a session ends with a summary that is below the quality threshold
    When the session end handler runs
    Then no graph node is created

  @failure @p1
  Scenario: No duplicate promotion on repeated session end
    Given a session was already promoted to a graph node
    When the session end handler runs again for the same session
    Then no duplicate graph node is created
    And the original node remains intact

@step-10
Feature: Temporal knowledge graph edges with validity bounds
  As a agent orchestrator
  I want graph edges to carry temporal validity bounds
  So that I can see which relationships are current and which are historical

  @happy @p0
  Scenario: Active edges are visible in graph queries
    Given a symbol relationship is recorded with a validity start in the past and no validity end
    When I query the graph for active edges
    Then the edge is returned

  @happy @p0
  Scenario: Expired edges are hidden from normal queries but preserved for history
    Given a symbol relationship has a validity end in the past
    When I query the graph for active edges
    Then the expired edge is not returned
    Given I query the graph for all edges including historical
    Then the expired edge is returned

  @edge
  Scenario: Edge with null validity end is treated as active
    Given a symbol relationship is recorded with no validity end
    When I query the graph for active edges
    Then the edge is returned as active

  @failure @p1
  Scenario: Temporal edge filtering regression is prevented
    Given a set of edges with mixed validity states
    When I query the graph for active edges
    Then only edges where the validity start is before now and the validity end is after now or null are returned
    And no active edge is incorrectly hidden
    And no expired edge is incorrectly shown

@step-11
Feature: Portable JSON export of observations, graph, and embeddings
  As a operator
  I want to export a project's full memory state as portable JSON
  So that I can move data between skillgrid instances

  @happy @p0
  Scenario: Export produces JSON with observations, graph edges, and embeddings
    Given a project with observations, graph edges, and embeddings
    When I export the project
    Then the output contains the observations array with full content and metadata
    And the output contains the graph edges array
    And the output contains the embeddings array with base64-encoded vectors

  @happy @p0
  Scenario: Export is portable and can be imported into another instance
    Given a project exported to JSON
    When I import the JSON into a new instance
    Then the observations, edges, and embeddings are restored
    And the data is queryable in the new instance

  @edge
  Scenario: Export to file and to stdout
    Given a project with data
    When I export with a file target
    Then the JSON is written to the file
    When I export without a file target
    Then the JSON is written to standard output

  @failure @p1
  Scenario: Export roundtrip preserves all data
    Given a project with observations, edges, and embeddings
    When I export and then import into a fresh instance
    Then every observation id, content, and metadata field matches the original
    And every edge and embedding is present
    And no data is lost or corrupted in the roundtrip

@step-12
Feature: Dream executor decomposes distillation into consolidate, synthesize, and prune
  As a agent orchestrator
  I want distillation to run as a decomposable dream with locking and rollback
  So that memory consolidation is safe and atomic

  @happy @p0
  Scenario: Dream executor runs consolidate, synthesize, and prune in sequence
    Given a project with multiple observations
    When I trigger a dream
    Then the consolidate phase merges facts from multiple observations
    And the synthesize phase creates higher-level summaries
    And the prune phase removes low-importance observations

  @happy @p0
  Scenario: Dream lock prevents concurrent distillation
    Given a dream is in progress on a project
    When a second dream is triggered on the same project
    Then the second dream is blocked until the first completes or times out

  @edge
  Scenario: Dream lock timeout auto-releases
    Given a dream is in progress on a project
    When the dream exceeds the five-minute lock timeout
    Then the lock is automatically released
    And a new dream can be triggered

  @failure @p1
  Scenario: Dream rollback reverts observations on error
    Given a dream is in progress and an error occurs mid-phase
    When the dream fails
    Then all observations are reverted to their pre-distill state
    And no partial consolidation is visible

@step-13
Feature: Importance scoring with recency decay in search ranking
  As a agent orchestrator
  I want observations to carry importance scores that decay with age
  So that the most relevant and recent memories rank higher

  @happy @p0
  Scenario: Importance score is computed from retrieval count and recency
    Given an observation with high retrieval count and recent access
    When the importance score is computed
    Then the score is high
    Given an observation with low retrieval count and old access
    When the importance score is computed
    Then the score is low

  @happy @p0
  Scenario: Importance score boosts ranking at query time
    Given two observations matching a query where one has a higher importance score
    When I search for the query
    Then the higher-importance observation ranks first

  @edge
  Scenario: Maturity tier transitions based on age and retrieval
    Given a freshly created observation
    When the maturity tier is evaluated
    Then the tier is fresh
    Given an old observation with high retrieval
    When the maturity tier is evaluated
    Then the tier is mature
    Given a very old observation with low retrieval
    When the maturity tier is evaluated
    Then the tier is archival

  @failure @p1
  Scenario: Importance scoring does not skew results incorrectly
    Given a set of observations with known importance scores
    When I search for a query
    Then the ranking reflects the importance scores
    And no observation with a valid match is dropped due to scoring
    And the decay rate is configurable

@step-14
Feature: Explicit typed relations between observations
  As a operator
  I want to annotate relationships between observations with typed edges
  So that I can query which memories relate to which and how

  @happy @p0
  Scenario: Relation edge is created between two observations
    Given two observations
    When I create a relation edge with type depends-on from the first to the second
    Then the relation is stored with source, target, type, and confidence
    And the relation is queryable

  @happy @p0
  Scenario: Relation query returns all related observations with types
    Given an observation with multiple incoming and outgoing relations
    When I query the relations for that observation
    Then all related observations are returned with their relation types and confidence values

  @edge
  Scenario: Self-relation is allowed and queryable
    Given an observation
    When I create a self-relation with type mentions
    Then the relation is stored
    And the relation query returns the observation as related to itself

  @failure @p1
  Scenario: Relation with low confidence is filterable
    Given a relation edge with a confidence below the filter threshold
    When I query relations with a minimum confidence filter
    Then the low-confidence relation is excluded from the results
    And the relation still exists in the store

@step-15
Feature: Provenance chain metadata on observations
  As a agent orchestrator
  I want every observation to carry a provenance chain
  So that I can trace why a memory was stored and what produced it

  @happy @p0
  Scenario: Provenance chain is set during curation
    Given I save an observation through the curation pipeline
    When the save completes
    Then the observation carries a provenance chain with session id, curate command, source files, and LLM reasoning
    And the provenance is visible in the observation list output

  @happy @p0
  Scenario: Provenance is queryable by observation id
    Given an observation with a provenance chain
    When I query the provenance for that observation
    Then the full provenance chain is returned

  @edge
  Scenario: Provenance is immutable after save
    Given an observation with a provenance chain
    When I attempt to modify the provenance after save
    Then the provenance remains unchanged
    And no update is applied

  @failure @p1
  Scenario: Provenance chain is preserved through TTL expiry
    Given an observation with a provenance chain that has expired
    When I query the provenance for the expired observation
    Then the provenance chain is still intact and readable
    And no data is lost

@step-16
Feature: Federated cross-project query with importance ranking
  As a operator
  I want cross-project search to use a federated pipeline with importance ranking
  So that results are ranked by both relevance and importance across projects

  @happy @p0
  Scenario: Federated query merges results from multiple stores by rank and importance
    Given two project stores each with matching observations at different importance scores
    When I run a federated cross-project search
    Then the results are merged using cross-store rank and importance score
    And higher-importance results appear first

  @happy @p0
  Scenario: Federated query deduplicates by observation id
    Given the same observation is present in two project stores
    When I run a federated cross-project search
    Then the observation appears only once in the results
    And the deduplicated result carries the highest importance score

  @edge
  Scenario: Federated query with a single store
    Given one project store with matching observations
    When I run a federated cross-project search
    Then the results are returned with importance scores
    And the pipeline behaves the same as a single-store search

  @failure @p1
  Scenario: Federated query correctness under mixed importance
    Given three project stores with observations at varying importance scores
    When I run a federated cross-project search
    Then the results are ordered by cross-store rank plus importance score
    And no observation is dropped
    And the deduplication is correct by observation id

@step-17
Feature: Distill lock service with rollback on failure
  As a agent orchestrator
  I want distillation to be protected by a per-project lock with automatic rollback
  So that concurrent distillations cannot corrupt the store

  @happy @p0
  Scenario: Distill lock is acquired before distillation and released after
    Given a project with no active distill lock
    When I start a distillation
    Then the lock is acquired
    When the distillation completes
    Then the lock is released

  @happy @p0
  Scenario: Distill rollback reverts observations on failure
    Given a distillation is in progress and fails
    When the failure is detected
    Then all observations are reverted to their pre-distill state
    And the lock is released

  @edge
  Scenario: Distill lock timeout auto-releases after five minutes
    Given a distill lock is held and the distillation exceeds five minutes
    When the timeout is reached
    Then the lock is automatically released
    And a new distillation can proceed

  @failure @p1
  Scenario: Concurrent distill is blocked by active lock
    Given a distill lock is active on a project
    When a second distillation is triggered on the same project
    Then the second distillation is blocked
    And the first distillation completes without interference

@step-18
Feature: Typed memory categories with LLM dedup and async two-phase commit
  As a agent orchestrator
  I want observations to be typed into memory categories and deduplicated before write
  So that the store is organized and free of semantic duplicates

  @happy @p0
  Scenario: Observation is assigned a memory type
    Given I save a new observation
    When the save completes
    Then the observation carries a memory type from the typed categories

  @happy @p0
  Scenario: LLM dedup detects semantic duplicates before write
    Given an existing observation in the store
    When I save a new observation that is semantically similar to the existing one
    Then the LLM dedup detects the similarity
    And the new observation is merged or linked rather than duplicated

  @happy @p0
  Scenario: Async two-phase commit writes synchronously and extracts asynchronously
    Given I save a new observation
    When the sync phase completes
    Then the observation is immediately queryable
    When the async phase completes
    Then the LLM extraction result is written to the memory diff audit file

  @edge
  Scenario: Memory type filter in list command
    Given observations of multiple memory types
    When I list observations filtered by a specific memory type
    Then only observations of that type are returned

  @failure @p1
  Scenario: Async extraction failure does not lose the synchronous write
    Given I save a new observation
    And the async extraction phase fails
    When the failure is detected
    Then the synchronously written observation remains intact
    And the memory diff audit file records the failure
    And a retry is attempted

  @failure @p1
  Scenario: LLM dedup false negative falls back to hash comparison
    Given the LLM dedup is unavailable
    When I save a new observation that is a near-exact duplicate of an existing one
    Then the hash-based fallback detects the duplicate
    And the duplicate is merged or linked

@step-19
Feature: Directory-level recursive retrieval with drill-down and trajectory
  As a operator
  I want search to locate the best directory first and then drill down
  So that results are contextual and the retrieval path is observable

  @happy @p0
  Scenario: Retrieval finds the highest-scoring directory first
    Given a hierarchical directory structure with observations at multiple levels
    When I search for a query
    Then the highest-scoring directory is identified first
    And the search drills down from that directory to find specific observations

  @happy @p0
  Scenario: Retrieval trajectory is preserved and queryable
    Given I run a search that traverses multiple directories
    When the search completes
    Then the directory-browsing path is stored in the retrieval trails
    And I can view the trajectory through the search trajectory flag

  @edge
  Scenario: Intent analysis classifies the query before retrieval
    Given a query that is clearly a debugging question
    When I run a search
    Then the intent is classified as debugging before retrieval begins
    And the retrieval strategy is adjusted accordingly

  @failure @p1
  Scenario: Deep hierarchy retrieval respects depth limit
    Given a directory structure with very deep nesting
    When I run a search
    Then the retrieval does not exceed the configured depth limit
    And results are returned from within the depth limit
    And no infinite recursion occurs

@step-20
Feature: Multi-version snapshots with transaction locking
  As a operator
  I want to create and roll back point-in-time store snapshots
  So that I can safely experiment with changes to the store

  @happy @p0
  Scenario: Snapshot creates a point-in-time view of the store
    Given a store with a set of observations
    When I create a snapshot
    Then the snapshot captures the current state of the store
    And the snapshot is stored with a version identifier

  @happy @p0
  Scenario: Rollback restores the store to a snapshot state
    Given a snapshot was created
    And observations have been modified since the snapshot
    When I roll back to the snapshot
    Then the store is restored to the snapshot state
    And the modified observations are reverted

  @edge
  Scenario: Row-level locking prevents concurrent writes to the same observation
    Given two concurrent writers targeting the same observation
    When both attempt to write
    Then one write succeeds and the other waits
    And no write is lost or corrupted

  @failure @p1
  Scenario: Snapshot storage does not bloat unboundedly
    Given I create multiple snapshots over time
    When the snapshot count exceeds the retention threshold
    Then old snapshots are automatically pruned
    And the store size does not grow unboundedly

@step-21
Feature: Prefix and delta handoff artifacts for agent-to-agent context
  As a agent orchestrator
  I want to generate a handoff artifact with stable prefix and dynamic delta
  So that another agent can pick up context without re-briefing

  @happy @p0
  Scenario: Handoff artifact contains stable prefix and dynamic delta
    Given a project with hub summaries, file counts, and metadata
    And recent changes and events
    When I generate a handoff
    Then the handoff contains the stable prefix with hub summaries, file counts, and project metadata
    And the handoff contains the dynamic delta with changed file stubs, risk files, and recent events

  @happy @p0
  Scenario: Handoff allows switching between agents without re-briefing
    Given a handoff artifact was generated
    When a new agent reads the handoff
    Then the new agent has the stable context from the prefix
    And the new agent has the recent changes from the delta
    And no re-briefing is needed

  @edge
  Scenario: Handoff with no recent changes
    Given a project with no recent changes
    When I generate a handoff
    Then the prefix is present
    And the delta is empty or minimal
    And the handoff is still valid

  @failure @p1
  Scenario: Handoff delta freshness
    Given a handoff was generated and then changes occur
    When I generate a new handoff
    Then the delta reflects only the changes since the last handoff
    And the prefix remains stable
    And no stale delta entries are carried forward

@step-22
Feature: Working set tracking, intent classification, and context envelope
  As a agent orchestrator
  I want a universal context envelope combining working set, intent, and skills
  So that any agent can get a complete context snapshot in one call

  @happy @p0
  Scenario: Context envelope contains project metadata, working set, and matched skills
    Given a session with edited files and a detected intent
    When I generate the context envelope
    Then the envelope contains the project metadata
    And the envelope contains the working set with edited files, edit counts, and net line deltas
    And the envelope contains the matched skills for the detected intent

  @happy @p0
  Scenario: Intent classification identifies the session intent
    Given a session where the agent is debugging
    When I classify the intent
    Then the intent is identified as debugging
    And the intent is included in the context envelope

  @edge
  Scenario: Context envelope with no matched skills
    Given a session with an intent that has no matching skills
    When I generate the context envelope
    Then the envelope is still valid
    And the matched skills section is empty
    And the working set and project metadata are present

  @failure @p1
  Scenario: Context envelope size is bounded
    Given a project with a very large working set
    When I generate the context envelope
    Then the envelope size is within the configured limit
    And field filtering is applied to keep the envelope manageable
    And no critical fields are dropped

@step-23
Feature: Hub file identification and impact analysis with risk scores
  As a agent orchestrator
  I want to identify hub files and score the risk of changes to them
  So that I can prioritize review for high-impact changes

  @happy @p0
  Scenario: Hub files are identified by import count threshold
    Given a code index with symbols where some have three or more importers
    When I run hub analysis
    Then the hub files are identified
    And each hub file has a hub score reflecting its import count

  @happy @p0
  Scenario: Impact analysis identifies hub files among changed files
    Given a set of changed files including a hub file
    When I run impact analysis on the changes
    Then the hub file is flagged in the impact analysis
    And the risk score for observations related to the hub file is high

  @edge
  Scenario: Non-hub file changes have low risk score
    Given a set of changed files that are not hub files
    When I run impact analysis on the changes
    Then the risk score for observations related to those files is low
    And no hub flags are raised

  @failure @p1
  Scenario: Hub analysis accuracy under large codebase
    Given a large codebase with many files and symbols
    When I run hub analysis
    Then the hub files are identified correctly
    And the impact analysis is accurate
    And the analysis completes within a reasonable time

@step-24
Feature: Skills framework and lifecycle hooks for context injection
  As a agent orchestrator
  I want to manage skills and inject context through lifecycle hooks
  So that relevant guidance is available at the right moments

  @happy @p0
  Scenario: Skills are stored and matched by intent
    Given skills stored as observations with the skill memory type
    When I match skills for a debugging intent
    Then relevant debugging skills are returned
    And irrelevant skills are not returned

  @happy @p0
  Scenario: Lifecycle hook injects context at session start
    Given the session-start hook is configured
    When a new session begins
    Then the hook injects relevant memories into the session context

  @happy @p0
  Scenario: Pre-edit hook injects risk analysis
    Given the pre-edit hook is configured
    When an edit is about to be made
    Then the hook injects risk analysis for the files being edited

  @edge
  Scenario: Hooks are opt-in per project
    Given a project with no hooks configured
    When a session begins
    Then no hook context is injected
    And the session proceeds normally

  @failure @p1
  Scenario: Hook timeout does not block the session
    Given a hook is configured and the hook processing exceeds the timeout
    When the hook is triggered
    Then the hook times out gracefully
    And the session continues without the hook context
    And no error is raised

@step-25
Feature: Virtual filesystem for structural browsing of memories
  As a operator
  I want to browse memories using a filesystem interface
  So that I can explore the store structurally alongside semantic search

  @happy @p0
  Scenario: Directory listing returns observations in a scope
    Given a project with observations in the preferences scope
    When I list the preferences scope
    Then the observations in that scope are returned
    And the listing reflects the directory structure

  @happy @p0
  Scenario: Tree view shows hierarchical structure of memory scopes
    Given a project with observations across multiple scopes
    When I request a tree view
    Then the hierarchical structure of scopes is displayed
    And observations are shown under their respective scopes

  @happy @p0
  Scenario: Pattern search finds observations by filesystem-style matching
    Given a project with observations in various scopes
    When I search for a pattern matching a scope path
    Then the matching observations are returned
    And the search does not replace semantic search

  @happy @p0
  Scenario: URI resolution maps virtual paths to stored observations
    Given a virtual URI pointing to a project scope
    When I resolve the URI
    Then the observations in that project and scope are returned
    And the resolution maps the URI to the correct SQL filter

  @edge
  Scenario: Scope management creates and lists scopes
    Given a project with no custom scopes
    When I create a new scope
    Then the scope is available for use
    When I list scopes
    Then the new scope is included in the list

  @failure @p1
  Scenario: Virtual filesystem performance under ten thousand observations
    Given a project with ten thousand or more observations
    When I list, tree, or find across scopes
    Then the operations complete within acceptable latency
    And no memory exhaustion occurs
    And results are correct

  @failure @p1
  Scenario: Virtual filesystem coexists with existing flat list without conflict
    Given a project with observations
    When I use the flat list command
    And I use the virtual filesystem listing
    Then both return the same underlying observations
    And there is no conflict or duplication between the two views

@step-26
Feature: Full test coverage for all mnemonic performance steps
  As a operator
  I want comprehensive unit and integration tests for every step
  So that all behavior is verified and regressions are caught early

  @happy @p0
  Scenario: All test suites pass
    Given the full test suite for the skillgrid CLI
    When I run the test suite
    Then all tests pass
    And no package has failing tests

  @happy @p0
  Scenario: Step 01 tests verify cached handle reuse, reference counting, and WAL retry
    Given the store pooling tests
    When I run them
    Then cached handle reuse is verified
    And reference counting is verified
    And WAL retry is verified

  @happy @p0
  Scenario: Step 02 tests verify trigram and prefix mode queries
    Given the FTS trigram tests
    When I run them
    Then trigram queries return correct results
    And prefix queries return correct results
    And default phrase queries are unchanged

  @happy @p0
  Scenario: Step 03 tests verify concurrent search matches sequential results
    Given the parallel search tests
    When I run them
    Then concurrent search completes
    And results match sequential search results

  @happy @p0
  Scenario: Step 04 tests verify automatic expiry and TTL retire
    Given the TTL tests
    When I run them
    Then automatic expiry is verified
    And TTL retire only retires expired observations

  @happy @p0
  Scenario: Step 05 tests verify LLM extraction and regex fallback
    Given the extraction tests
    When I run them
    Then LLM extraction is verified
    And regex fallback is verified
    And error handling is verified

  @happy @p0
  Scenario: Step 06 tests verify embedder providers and config-driven selection
    Given the embedder tests
    When I run them
    Then the Ollama provider is verified
    And the local provider is verified
    And config-driven selection is verified

  @happy @p0
  Scenario: Step 07 tests verify cross-link queries and symbol embedding integrity
    Given the triple-store tests
    When I run them
    Then cross-link queries are verified
    And symbol embedding integrity is verified

  @happy @p0
  Scenario: Step 08 tests verify ranking boost and decay
    Given the improve loop tests
    When I run them
    Then high-usage observation ranking boost is verified
    And decay for never-accessed observations is verified

  @happy @p0
  Scenario: Step 09 tests verify session promotion to graph node
    Given the session promotion tests
    When I run them
    Then L0 and L1 sessions create permanent graph nodes

  @happy @p0
  Scenario: Step 10 tests verify temporal edge filtering
    Given the temporal graph tests
    When I run them
    Then validity start and end edge filtering is verified

  @happy @p0
  Scenario: Step 11 tests verify export JSON completeness
    Given the export tests
    When I run them
    Then the JSON output contains observations, edges, and embeddings

  @happy @p0
  Scenario: Step 12 tests verify dream executor phases and lock and rollback
    Given the dream executor tests
    When I run them
    Then consolidate, synthesize, and prune are verified
    And the dream lock is verified
    And dream rollback is verified

  @happy @p0
  Scenario: Step 13 tests verify importance scoring and recency decay
    Given the importance scoring tests
    When I run them
    Then importance scoring is verified
    And recency decay is verified
    And query-time ranking boost is verified

  @happy @p0
  Scenario: Step 14 tests verify relation annotations and queries
    Given the explicit relation tests
    When I run them
    Then relation annotations are verified
    And relation queries are verified

  @happy @p0
  Scenario: Step 15 tests verify provenance chain and immutability
    Given the provenance tests
    When I run them
    Then the provenance chain is verified
    And provenance immutability is verified

  @happy @p0
  Scenario: Step 16 tests verify federated query, importance ranking, and dedup
    Given the federated query tests
    When I run them
    Then federated query results are verified
    And importance ranking is verified
    And deduplication is verified

  @happy @p0
  Scenario: Step 17 tests verify distill lock, rollback, and timeout
    Given the distill lock tests
    When I run them
    Then the distill lock is verified
    And distill rollback is verified
    And lock timeout is verified

  @happy @p0
  Scenario: Step 18 tests verify memory types, LLM dedup, and async commit
    Given the memory types tests
    When I run them
    Then memory type assignment is verified
    And LLM dedup is verified
    And async two-phase commit is verified

  @happy @p0
  Scenario: Step 19 tests verify directory retrieval, drill-down, and trajectory
    Given the directory retrieval tests
    When I run them
    Then directory retrieval is verified
    And drill-down is verified
    And trajectory preservation is verified

  @happy @p0
  Scenario: Step 20 tests verify snapshots and row-level locking
    Given the snapshot tests
    When I run them
    Then snapshot creation and rollback are verified
    And row-level locking is verified

  @happy @p0
  Scenario: Step 21 tests verify handoff artifact prefix and delta
    Given the handoff tests
    When I run them
    Then the handoff artifact prefix and delta are verified
    And handoff generation is verified

  @happy @p0
  Scenario: Step 22 tests verify context envelope, working set, and intent classification
    Given the context envelope tests
    When I run them
    Then the context envelope is verified
    And the working set is verified
    And intent classification is verified

  @happy @p0
  Scenario: Step 23 tests verify hub identification, impact analysis, and risk score
    Given the hub impact tests
    When I run them
    Then hub file identification is verified
    And impact analysis is verified
    And risk score is verified

  @happy @p0
  Scenario: Step 24 tests verify skills framework and lifecycle hooks
    Given the skills and hooks tests
    When I run them
    Then the skills framework is verified
    And lifecycle hooks are verified

  @happy @p0
  Scenario: Step 25 tests verify virtual filesystem commands and URI resolution
    Given the memfs tests
    When I run them
    Then the listing, tree, and find commands are verified
    And URI resolution is verified
    And scope management is verified

  @edge
  Scenario: Test coverage is maintained across all steps
    Given the full test suite
    When I run it
    Then every step from 01 to 25 has at least one test
    And no step is missing test coverage

  @failure @p1
  Scenario: Test failures are isolated to the failing step
    Given the full test suite
    When a single step's test fails
    Then only that step's test is reported as failing
    And other steps' tests still pass
    And the failure is attributable to the specific step
