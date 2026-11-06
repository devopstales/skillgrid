# Source: docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios <-> change.md DoD / Testing strategy; @p1 = important failure paths.
# Threat: Mnemonic tool surface — owning steps 02, 03, 04 (split per change.md Planned RED coverage).
# One Feature per step; tag each Feature with @step-NN matching tasks.md.

@step-01
Feature: Additive graph schema and 30-language extractors
  As a coding agent
  I want queryable symbols and edges after index
  So that navigation and blast-radius start from structured code intelligence

  @happy @p0
  Scenario: Indexed supported-language files yield symbols and edges
    Given a project with sources in a sample of the 30 supported languages
    When the code index runs
    Then symbols and edges for those files are queryable
    And graph tables exist without rewriting files or chunks

  @edge
  Scenario: Deleting a file or function prunes its whole footprint
    Given an indexed project with a file containing a function that has symbols edges and vectors
    When the file or function is removed and the code index runs again
    Then the removed file or function symbols edges and vectors are deleted in one pass
    And no orphan rows linger in the graph

  @edge
  Scenario: Oversized file is skipped and counted
    Given one file larger than the max file size threshold
    When the code index runs
    Then the file is skipped as part of the run stats
    And it is not treated as an error or a fallback

  @edge
  Scenario: Unsupported language uses regex fallback and index continues
    Given one file in a language outside the 30 supported
    When the code index runs
    Then the regex fallback is used for that file
    And the index completes for remaining files

  @failure @p1
  Scenario: Malformed file falls back and index continues
    Given one file that fails primary extraction
    When the code index runs
    Then fallback is used for that file
    And the index completes for the remaining files

  @failure @p1
  Scenario: A crashing grammar quarantines and the index continues
    Given a file whose syntax crashes its language grammar
    When the code index runs
    Then that file is quarantined and uses the regex fallback
    And the extraction worker is respawned up to a bound and a repeated grammar death trips its circuit breaker
    And the index run completes without the whole run failing

  @failure @p1
  Scenario: Store open preserves existing chunk index
    Given a store that already has files and chunks
    When the store opens with the new graph schema
    Then files and chunks remain intact
    And chunk search still returns prior hits

@step-02
Feature: Identifier-aware symbol search, structural grep, and orientation
  As a coding agent
  I want camelCase symbol lookup structural matching and orientation
  So that I find symbols chunk search misses and see the why behind code

  @happy @p0
  Scenario: Identifier FTS and orientation locate a symbol
    Given indexed camelCase and snake_case symbols
    When an agent searches symbols and requests signature TOC map list metadata
    Then identifier search finds symbols chunk code_search misses
    And orientation returns signature TOC map list metadata

  @happy @p0
  Scenario: Structural grep matches by example without an index
    Given a set of source files under the working directory and no index built
    When an agent runs structural search with a by-example pattern using metavariables
    Then matching syntax trees are returned per language
    And no store or embeddings are required to run the search

  @edge
  Scenario: Invalid structural pattern skips the language with a note
    Given a structural search pattern that is invalid for one language
    When an agent runs structural search across multiple languages
    Then files in that language are skipped with a clear note
    And files in the other languages still match
    And the run is not a silent no-match masquerading as success

  @edge
  Scenario: Rationale comments link to code
    Given indexed source with NOTE and WHY comments and ADR references
    When an agent requests orientation for the enclosing symbol
    Then rationale nodes are returned linked to the nearest enclosing symbol
    And each rationale node carries its source comment text

  @edge
  Scenario: Unknown symbol returns empty or not-found
    Given an indexed project
    When an agent looks up a missing symbol
    Then the response is empty or not-found with no fabricated symbol

  @failure @p1
  Scenario: code_search stable and bad orient args fail
    Given the Mnemonic tool surface is registered
    When an agent inspects code_search and calls orientation or structural search with missing args
    Then code_search name and required query schema are unchanged
    And the new tools are registered
    And the bad-args call is rejected clearly

@step-03
Feature: Call-graph traversal, composite code_explore, and risk-tiered impact
  As a coding agent
  I want callers callees a call path and blast radius in one call
  So that I judge blast radius before edits without re-grepping

  @happy @p0
  Scenario: Known symbol returns graph views with confidence
    Given an indexed known symbol with resolved edges
    When an agent requests callers callees dependents implementors hierarchy and tests-for
    Then each view returns edges for that symbol
    And every edge carries a confidence label

  @happy @p0
  Scenario: code_explore returns source call-flow and blast radius in one call
    Given an indexed project with a symbol and its dependents
    When an agent calls the composite explore tool for that symbol
    Then verbatim source grouped by file is returned
    And call paths between the returned symbols including inferred dynamic-dispatch hops are returned
    And a blast-radius summary is returned in the same call
    And the tool is the documented primary MCP tool with initialize guidance to answer directly and not re-grep

  @happy @p0
  Scenario: code_explore is the primary tool and menu tools re-enable
    Given the Mnemonic tool surface is registered
    When an agent inspects the MCP tool list
    Then the composite explore tool is listed as the primary tool
    And the narrow graph and orientation tools are unlisted by default
    And enabling them via configuration re-adds them to the surface
    And the status tool reports a per-language fair coverage field

  @edge
  Scenario: code_path traces the shortest connection or where the graph stops
    Given two symbols connected by edges in the graph
    When an agent requests the path between them
    Then the shortest path of edges is returned with each hop confidence-labeled
    When an agent requests the path between two symbols with no static edge path
    Then a where-the-graph-stops answer is returned naming the dispatch kind that ended the flow and its line
    And the refused name-only matches are returned with their confidence
    And no hop is fabricated

  @edge
  Scenario: code_explain explains a symbol
    Given an indexed symbol with multiple connections
    When an agent requests an explanation for it
    Then the symbol its degree and all connections ranked by degree are returned

  @edge
  Scenario: code_impact tiers risk and disambiguates a multi-symbol target
    Given a known symbol with direct and deeper dependents
    When an agent requests the impact of that symbol
    Then direct dependents are tiered as will-break and deeper dependents as likely-affected
    And every edge is confidence-tagged and low-confidence hops below the threshold are excluded
    When an agent requests the impact of a name that matches several symbols
    Then a ranked candidate list is returned instead of a silent pick
    And the candidate list can be narrowed by file uid or kind

  @edge
  Scenario: Ambiguous resolution is labeled not dropped
    Given a symbol with ambiguous edge resolution
    When an agent requests graph neighbors for it
    Then edges return with AMBIGUOUS confidence
    And those edges are not silently omitted

  @edge
  Scenario: code_status reports measured fair coverage per language
    Given an indexed project with symbol-bearing files in several languages
    When an agent checks the code status
    Then per-language fair coverage is reported as the measured share of symbol-bearing files with at least one resolved cross-file dependent

  @edge
  Scenario: maxTokens truncates the response and stays valid
    Given a composite explore response that exceeds the token budget
    When an agent calls explore with a maxTokens budget
    Then the formatted response is truncated with an ellipsis
    And the truncated response stays valid and well-formed

  @edge
  Scenario: MCP pool opens lazily and honors the repo param
    Given the MCP server has not yet opened any store
    When an agent issues the first query for a project
    Then the store is opened lazily on first use
    And idle stores are evicted after inactivity
    And an optional repo parameter selects the project

  @failure @p1
  Scenario: Unknown symbol graph query invents no edges
    Given an indexed project
    When an agent requests callers or impact for a missing symbol
    Then the response is empty or not-found
    And no invented edges are returned

@step-04
Feature: Offline hybrid code search with pluggable embeddings and doctor
  As a coding agent
  I want ranked hybrid hits with provenance and a working embedder
  So that search works offline and degrades gracefully without embeddings

  @happy @p0
  Scenario: Hybrid search ranks offline with provenance
    Given an indexed project with embeddings disabled
    When an agent runs hybrid code search and checks semantic and embedding status
    Then ranked hits include per-signal FTS and signal provenance
    And semantic code search and embedding status are available
    And hybrid code search is distinct from memory semantic search

  @happy @p0
  Scenario: ONNX default embeds and caches the model
    Given no embedder provider is configured
    When the code index embeds a code unit
    Then the ONNX nomic-embed-code provider is used by default
    And a 768-dim vector is produced
    And the model is downloaded to the local cache and reused on the next embed

  @happy @p0
  Scenario: doctor performs a functional embed round-trip on both sides
    Given an installed code intelligence setup
    When an operator runs the doctor command
    Then a real embed round-trip is performed under both the indexing and query parameters
    And the round-trip checks vector dimension and non-degeneracy
    And capability checks for grammars ONNX model WAL state and CGo-free are reported

  @happy @p0
  Scenario: Semantic search names the symbol
    Given an indexed project with symbol-level embeddings
    When an agent runs semantic code search for a concept
    Then symbol-level hits return the named symbol and its file and line
    And chunk-level hits return the line range for non-symbol code

  @happy @p0
  Scenario: CLI search returns hybrid hits with provenance
    Given an indexed project with embeddings enabled
    When an operator runs skillgrid search with a query
    Then hybrid search runs by default
    And a table is printed with a per-signal provenance column
    And --json emits machine-readable hits and the embedder
    And --fts and --semantic select the lexical or vector leg
    And search grep runs structural search and embedding-status reports the active provider and model

  @edge
  Scenario: Index embeds symbol-level and chunk-level eagerly and re-embeds on model swap
    Given an indexed project with embedded symbols and chunks
    When the code index runs again with unchanged files
    Then unchanged symbols and chunks are not re-embedded
    And a partial embedding run resumes without repeating completed units
    And a function spanning two chunks yields one symbol-level vector not two half-chunk vectors
    And chunk windows overlap so boundary context is captured
    And the embedder treats the corpus and the query with their separate parameter sets
    When the configured embedding model changes
    Then all symbol-level and chunk-level vectors are re-embedded with the new model

  @edge
  Scenario: External provider embeds via HTTP endpoint
    Given an external embedder provider is configured with a base URL and model
    When the code index embeds a code unit
    Then the vector is fetched from the configured OpenAI-compatible endpoint
    And the hybrid ranker blends it with FTS and signals

  @edge
  Scenario: Off provider is a Null Adapter
    Given the embedder is configured off
    When an agent runs hybrid code search
    Then the vector leg is absent and ranking uses FTS and signals
    And the run does not hard-fail for missing embeddings

  @failure @p1
  Scenario: Down embedder degrades to FTS and signals
    Given hybrid search available and the embedder down
    When an agent runs hybrid code search
    Then ranking uses FTS and signals without hard-failing for missing embeddings

  @failure @p1
  Scenario: Degenerate vector is reported and search degrades
    Given an embedder that produces a degenerate or dimension-mismatched vector
    When the code index embeds and an agent checks the doctor
    Then the doctor reports the degenerate vector
    And the index degrades to FTS and signals rather than storing a silent bad vector

  @failure @p1
  Scenario: Hybrid tool is distinct and rejects bad args
    Given the Mnemonic tool surface is registered
    When an agent compares hybrid code search to memory semantic search and sends bad args
    Then hybrid code search is distinct code_search is unchanged and bad args are rejected
