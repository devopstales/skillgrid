# Source: docs/skillgrid/changes/008-mnemonic-community-knowledge-graph/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# Threat: Mnemonic tool surface — owning steps 01, 02, 03 (005 tools must stay stable in each)
# Threat: Retrieval quality / evaluation — owning step 01 (significance-proven ranking, leak-free ground truth)
# Threat: Security boundary (output) — owning step 01 (output-time redaction + doctor --strict)
# One Feature per step; tag each Feature with @step-NN matching tasks.md.
# WHAT not HOW: no file paths or function names.

@step-01
Feature: Leiden community detection, god nodes, and a measured retrieval-quality layer
  As a coding agent
  I want subsystem-level orientation from clustered communities and hub symbols, plus search results whose ranking, confidence, and safety are measured and auditable
  So that I know the core modules before editing and can trust that a search hit is explainable, categorized, and free of raw secrets

  @happy @p0
  Scenario: Community detection returns labeled subsystems and existing search tools stay stable
    Given an indexed project whose symbols and edges form distinct subsystems
    When the community detection pass runs and the agent lists communities
    Then clustered subsystems are returned, each with an LLM-free label
    And the existing 005 code search tools keep their names and required parameters

  @happy @p0
  Scenario: God nodes rank hubs and hub exclusion suppresses utility symbols
    Given an indexed project with a few very highly-connected utility symbols
    When the agent requests god nodes with hubs excluded
    Then the most-connected symbols are ranked by degree
    And the utility super-hubs are suppressed from the ranking

  @happy @p0
  Scenario: Evaluation harness derives a leak-free query set and reports significance
    Given a project with git history whose commits each change known files
    When the evaluation harness mints its query set and runs the ablation
    Then the query set is derived from commit subjects to changed files with merge, revert, release, bump, formatting, changelog-like, and benchmark-touching commits dropped
    And exactly one index per corpus is shared by every variant
    And file-granularity recall, MRR, nDCG, useful-at-budget, tokens, duplicate percent, and p50 p95 p99 latency are reported
    And every non-baseline row carries a paired bootstrap 95 percent CI and a permutation p-value

  @happy @p0
  Scenario: Shipped ranking config is the significance winner
    Given the evaluation harness run over at least two pooled corpora
    When the shipped ranking configuration is compared against the 005 baseline
    Then the shipped config is non-negative across all pooled corpora
    And every ranking signal it ships survived significance

  @happy @p0
  Scenario: Search response carries confidence action rerank reasons and fallbacks
    Given an indexed project with symbols that match a search query
    When the agent runs a search
    Then each hit carries an explainable rerank reason from a named bounded factor
    And a candidate found by two retrievers at different locators surfaces as one strong file-level candidate
    And the response carries a categorical high or medium or low confidence mapped to an explicit agent action
    And a low-confidence response attaches fallback suggestions with ready patterns, likely paths, and a broaden-query hint

  @happy @p0
  Scenario: Snippets are skeletonized and secrets are redacted in output
    Given an indexed file containing a secret-like pattern and an unrelated long function body
    When the agent reads the search or read output for that file
    Then the snippet preserves imports, signatures, and the matched line with an exact read range
    And unrelated bodies are collapsed
    And the secret-like pattern is replaced and never emitted raw

  @happy @p0
  Scenario: Strict doctor reports redaction and freshness state
    Given an indexed project with redaction and freshness state
    When the operator runs strict doctor for CI
    Then redaction state and freshness state are reported
    And the command exits non-zero when a security or freshness violation exists

  @edge
  Scenario: Community explanation returns members and entry points
    Given a community returned by community detection
    When the agent requests an explanation for that community
    Then the community's member symbols and its key entry points are returned

  @edge
  Scenario: Tiny graph yields a single trivial community
    Given a graph with fewer than two connected symbols
    When the community detection pass runs
    Then a single trivial community is returned
    And the pass does not crash

  @edge
  Scenario: Community label falls back when no god node exists
    Given a community whose members have no dominant hub symbol
    When a community label is resolved for it
    Then a neutral community-N style label is returned
    And no label is fabricated from absent names

  @edge
  Scenario: Community partition is reproducible and cached by content-hash
    Given an indexed project with a stable graph
    When the community pass runs twice over an unchanged graph
    Then the partition is reproducible under the pinned seed and resolution
    And the result is cached by content-hash so an unchanged re-index is not recomputed

  @edge
  Scenario: Evaluation harness drops noise commits from the query set
    Given a project whose git history contains merges, reverts, releases, version bumps, formatting-only commits, changelog-like commits, and commits that touch the benchmark itself
    When the evaluation harness derives its query set
    Then none of those commits contribute a query
    And only substantive commits with changed files remain

  @edge
  Scenario: Evaluation corpus excludes the benchmark scaffolding
    Given a corpus that contains the evaluation benchmark itself
    When the evaluation harness builds the corpus it grades
    Then the benchmark scaffolding is excluded from the corpus
    And the retriever can never see the ground truth it is graded against

  @edge
  Scenario: Ablation shares one index so deltas measure ranking
    Given two ranking variants to compare on one corpus
    When the ablation runner executes both
    Then both variants query the single shared index for that corpus
    And the measured delta reflects ranking only, never indexing variance

  @edge
  Scenario: Failing ranking signal is removed or kept off with the decision recorded
    Given a candidate ranking signal whose pooled significance fails
    When the evaluation harness completes the ablation
    Then the signal is removed or kept off the shipped configuration
    And the decision with its CI and p-value is recorded in the harness report

  @failure @p1
  Scenario: Stale evaluation expectation fails the run loudly
    Given an evaluation query whose expected file no longer exists at HEAD
    When the evaluation harness validates its query set
    Then the run fails with a loud validation error
    And the score is not silently deflated

  @failure @p1
  Scenario: Community tools reject bad args clearly
    Given the community tools are registered on the Mnemonic tool surface
    When the agent calls a community tool with a missing or unknown community id
    Then a clear validation error is returned
    And no community is invented

  @failure @p1
  Scenario: Doctor strict exits non-zero on redaction violation
    Given an indexed project where a secret-like pattern would be emitted raw in search output
    When the operator runs strict doctor for CI
    Then the redaction state reports the violation
    And the command exits non-zero

@step-02
Feature: Precomputed process flows from entry points through call chains
  As a coding agent
  I want to see what a subsystem does end to end
  So that I can answer which flow a symbol participates in without reading every file

  @happy @p0
  Scenario: Process list returns precomputed flows from entry points
    Given an indexed project with entry points for routes handlers and CLI mains
    When the process pass runs and the agent lists processes
    Then complete execution flows are returned in a single call with no per-query traversal
    And each flow has named steps and a cross-community flag and an LLM label

  @happy @p0
  Scenario: Process detail returns the full step-by-step trace
    Given a named process from the process layer
    When the agent requests that process by name
    Then the full step-by-step trace is returned
    And each hop carries a confidence label

  @edge
  Scenario: Process labels are cached by content-hash and re-labeled only on change
    Given a process flow that has already been labeled
    When the index re-runs over an unchanged flow
    Then the cached label is reused and the LLM is not called again
    When the flow structure changes
    Then a new label is produced for the new content-hash

  @edge
  Scenario: Cross-community process is flagged
    Given a flow that passes through symbols in more than one community
    When the process pass traces it
    Then the process carries a cross-community flag

  @edge
  Scenario: Trace stops at a dispatch boundary with a note
    Given a call chain that reaches an interface to implementation or message bus or callback boundary
    When the process trace reaches that boundary
    Then the trace is truncated with a stops-at note naming the symbol and reason
    And the trace is not silently cut

  @edge
  Scenario: Symbol explanation surfaces process participation and existing search tools stay stable
    Given a symbol that participates in a precomputed process
    When the agent explains that symbol
    Then the processes the symbol participates in are surfaced with its step position
    And the existing 005 code tools keep their names and required parameters

  @edge
  Scenario: Untraceable entry point yields single-step or is skipped
    Given an entry point with no traceable call chain
    When the process pass seeds it
    Then it yields a single-step process or is skipped
    And no flow is fabricated

  @failure @p1
  Scenario: LLM down caches the flow unlabeled
    Given a process flow to label and the LLM is unavailable
    When the process pass labels it
    Then the flow structure is still cached
    And it is cached without a fabricated label

  @failure @p1
  Scenario: Process tools reject bad args clearly
    Given the process tools are registered on the Mnemonic tool surface
    When the agent requests an unknown process name with bad args
    Then a clear validation error is returned
    And no process is invented

@step-03
Feature: Knowledge-graph nodes for docs, configs, and SQL
  As a coding agent
  I want code linked to its docs, configs, and data schema in one graph
  So that I can see which code reads or writes a table and trace code to its documentation

  @happy @p0
  Scenario: Markdown links and wikilinks become references edges
    Given indexed markdown docs containing relative links and wikilinks
    When the knowledge extraction pass runs
    Then doc nodes are created with references edges between them
    And each new edge carries a confidence label

  @happy @p0
  Scenario: Config references become configures edges
    Given indexed config files in yaml toml and json
    When the knowledge extraction pass runs
    Then config nodes are created with configures edges to the code they configure
    And each new edge carries a confidence label

  @happy @p0
  Scenario: SQL DDL becomes table and column nodes with reads and writes
    Given indexed SQL schema defining tables and columns
    When the knowledge extraction pass runs
    Then table and column nodes are created
    And the code that references them gets reads and writes edges with confidence labels

  @edge
  Scenario: Path tool traces code to doc to config to table
    Given a graph that now spans code, docs, configs, and tables
    When the agent requests a path from a code symbol to a data table
    Then a single query traces through doc, config, and table nodes
    And the existing 005 path tool is unchanged

  @edge
  Scenario: Unresolvable config ref is ambiguous not dropped
    Given a config reference that resolves to no known symbol
    When the knowledge extraction pass runs
    Then the edge is kept and marked ambiguous
    And it is not silently dropped

  @edge
  Scenario: Malformed doc file falls back and indexes the rest
    Given one doc file with unparseable links among valid docs
    When the knowledge extraction pass runs
    Then the bad links are skipped and the rest is indexed
    And the index does not abort

  @edge
  Scenario: Malformed SQL statement is skipped and the rest is indexed
    Given one SQL file with a statement that fails to parse among valid DDL
    When the knowledge extraction pass runs
    Then that statement is skipped and the rest is indexed
    And the index does not abort

  @edge
  Scenario: Indexer hook runs community, process, and knowledge in one transaction
    Given an index run over a project
    When the indexer finishes code extraction
    Then the community, process, and knowledge passes run in the same incremental transaction

  @failure @p1
  Scenario: Knowledge tools register and reject bad args
    Given the knowledge tools are registered on the Mnemonic tool surface
    When the agent calls a knowledge tool with missing args
    Then a clear validation error is returned
    And the existing 005 code tools keep their names and required parameters
