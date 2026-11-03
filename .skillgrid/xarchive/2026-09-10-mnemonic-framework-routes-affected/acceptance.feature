# Source: docs/skillgrid/changes/010-mnemonic-framework-routes-affected/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# Threat: Mnemonic tool surface — owning steps 01, 02, 03; PR commands — 02;
#         Index freshness / concurrency — 03 (incl. fingerprint gate); Warm embedder lifecycle — 03.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.

@step-01
Feature: Framework route and navigation edges
  As a coding agent
  I want route and navigates edges in the graph
  So that URL to handler and screen to screen are one graph hop

  @happy @p0
  Scenario: Web-framework routing files produce route nodes
    Given an indexed web app with routing files for a supported framework
    When the agent queries the route bound to a view or controller
    Then a route node exists for the URL pattern
    And a references edge links the route to its handler
    And every new edge carries a confidence label

  @happy @p0
  Scenario: Router navigations produce navigates edges
    Given an indexed app with a recognized framework router
    When the agent queries where a sending function navigates
    Then a navigates edge links the sending function to the named screen
    And a literal destination is labeled EXTRACTED
    And a markup-written link is labeled INFERRED

  @edge
  Scenario: Computed or unserved destination stays unresolved
    Given a routing file with a computed non-literal destination and a path no route serves
    When the code index runs
    Then the computed destination is left unresolved not guessed
    And the unserved path is marked unresolved not dropped silently

  @edge
  Scenario: Ambiguous references are dropped not guessed
    Given a reference with no same-file match and no explicit specifier and no unique global match
    When the code index runs
    Then the reference is dropped at extraction not stored as an ambiguous edge
    And a warning is reported with a count and a sample not a silent discard
    And the ambiguous reference is excluded from downstream blast-radius math
    And an edge resolved only by a heuristic stays labeled AMBIGUOUS

  @edge
  Scenario: Malformed routing file falls back and index continues
    Given one routing file that fails pattern recognition
    When the code index runs
    Then no route node is fabricated for that file
    And a warning is emitted and indexing continues for the rest

  @edge
  Scenario: Framework with no routes reports none
    Given a supported framework with a routing file that declares no routes
    When the agent queries routes for that framework
    Then the result says there are no routes
    And no empty fabricated route list is returned

  @failure @p1
  Scenario: Route tools register and code_search stays stable
    Given the Mnemonic tool surface is registered
    When the agent inspects route tools and code_search and sends bad route args
    Then route tools are registered with distinct code_* names
    And code_search name and query schema are unchanged
    And 005 and 008 tools are intact
    And the bad route args are rejected clearly

@step-02
Feature: code_affected and code_rename grounded in the graph
  As a coding agent
  I want affected tests and a graph-grounded rename
  So that I know what to run and what a rename touches

  @happy @p0
  Scenario: code_affected returns affected test files
    Given an indexed project where changed sources are transitively imported
    When the agent runs code_affected on the changed files
    Then the test files that transitively import the changed symbols are returned
    And the result is paths and relationships from the existing graph with no invented edges

  @happy @p0
  Scenario: code_affected consumes a diff file list from stdin
    Given an indexed project and a diff file list on stdin
    When the agent runs code_affected with --stdin
    Then the test files affected by the diff are returned
    And --depth caps traversal and --filter restricts which test files are reported
    And --json and --quiet emit scriptable output

  @happy @p0
  Scenario: code_affected --base derives areas and owners
    Given an indexed project on a branch with a merge-base against a base ref
    When the agent runs code_affected with --base for that base ref
    Then the changed set is derived from the merge-base diff without stdin
    And the result is grouped into affected areas with per-symbol detail collapsed
    And git-history owners are named for each affected area as who to tag
    And --quiet and --json shape the output as a ready PR comment and CI gate

  @happy @p0
  Scenario: code_rename returns graph and text-search edit buckets
    Given an indexed project with a resolvable symbol
    When the agent runs code_rename for that symbol with a new name
    Then files_affected and total_edits are returned
    And graph edits are flagged high-confidence and text-search edits are flagged review carefully
    And every edit carries a confidence label
    And dry-run returns the plan without writing

  @edge
  Scenario: code_rename applies only listed files without committing
    Given a non-dry code_rename plan for a resolvable symbol
    When the rename is applied
    Then exactly the planned files are edited and no other file is touched
    And no commit or push is performed

  @edge
  Scenario: Ambiguous rename target returns candidates
    Given a rename target that matches several or no symbols
    When the agent runs code_rename
    Then a ranked candidate list or not-found is returned
    And no silent pick is made

  @edge
  Scenario: code_affected with no changed files returns empty
    Given an indexed project and no changed files
    When the agent runs code_affected on an empty input
    Then an empty result with a clear message is returned
    And it is not treated as an error

  @edge
  Scenario: Dropped references are excluded from blast radius
    Given an indexed project with a reference dropped at extraction and a resolved reference
    When the agent runs code_affected over the changed files
    Then only the resolved references are traversed
    And the dropped reference contributes nothing to the reported test-file radius

  @failure @p1
  Scenario: affected and rename tools register and stay stable
    Given the Mnemonic tool surface is registered
    When the agent inspects code_affected and code_rename and sends bad args
    Then code_affected and code_rename are registered with distinct code_* names
    And code_affected exposes stdin and base and depth and filter and json and quiet options
    And rename disambiguation returns candidates with no silent pick
    And dry_run touches nothing
    And 005 and 008 tools are still stable
    And the bad args are rejected clearly

@step-03
Feature: Auto-sync watcher, fingerprint gate, copy-and-swap, and warm embedder
  As a coding agent
  I want a fresh graph and a warm embedder while I edit
  So that I never read a torn index or a stale answer mid-session

  @happy @p0
  Scenario: Saving a source file triggers a debounced re-index
    Given a running MCP server with the watcher enabled
    When a source file is saved and rapid edits follow within the debounce window
    Then the burst collapses into one incremental re-index
    And only the changed surface is re-indexed
    And non-source files are ignored

  @happy @p0
  Scenario: Reader sees old or new index never torn
    Given a re-index publishing via copy-and-swap while a reader queries
    When the reader reads the index during publication
    Then it sees either the old or the new index never a partial one
    And on a sidecar-unsupported filesystem the in-place fallback retries pre-write and stops if it may have mutated the live index

  @happy @p0
  Scenario: Reader auto-reopens the new index
    Given a running MCP process and a newly published index
    When the next tool call is made
    Then the process reopens the new index within about five seconds
    And no restart is needed to see the just-made edit

  @happy @p0
  Scenario: Connect-time catch-up absorbs offline edits
    Given edits made while no MCP server was running
    When the MCP server reconnects
    Then a size mtime and content-hash reconciliation absorbs the offline edits
    And only genuinely changed files are re-indexed

  @happy @p0
  Scenario: Fingerprint gate re-indexes structurally with watcher off
    Given a running MCP server with the watcher disabled
    When a source file is edited and the agent makes a code query
    Then a stat-walk of the working tree detects drift against the index fingerprint
    And a structural-only incremental re-index runs before the answer under the writer lock
    And the query answers against the edited tree including uncommitted edits
    And the gate never invokes the embedder leg
    And the fingerprint is keyed by the extractor stamp so a stamp change invalidates it

  @happy @p0
  Scenario: Pending referenced file gets a staleness banner
    Given a still-pending file referenced by an MCP tool response
    When the response is returned during the debounce window
    Then a warning banner names the file and says to read it directly
    And un-referenced pending files surface as a footer
    And the banner clears after sync

  @edge
  Scenario: Second writer exits with a writer-lock error
    Given one serve or MCP process holds the writer lock for a project
    When a second process starts on the same project
    Then the second exits with a clear writer-lock error pointing at the live writer

  @edge
  Scenario: Reindex failure keeps the old index live
    Given a watcher re-index that fails after it may have mutated the live index
    When the failure is detected
    Then in-place mode stops the watcher and surfaces the failure
    And copy-and-swap mode keeps the old index live

  @edge
  Scenario: Watcher disabled means manual index
    Given a sandboxed environment with the watcher disabled
    When the agent edits a file and queries the index
    Then the index stays manual and the banner notes the watcher is off

  @edge
  Scenario: Absent model degrades to FTS and signals
    Given the embedder model is absent or fails to load warm
    When the agent runs a semantic search
    Then the search degrades to FTS and signals
    And no model-load error is surfaced to the agent

  @edge
  Scenario: Evicted embedder reloads on reuse
    Given a warm embedder that was evicted after idle
    When the agent reuses search
    Then the embedder reloads transparently on next use
    And the one-time load cost is not surfaced as an error

  @failure @p1
  Scenario: Warm embedder loads once and reuses
    Given an active session running repeated searches
    When the embedder is used across search calls
    Then the ONNX model loads once and is reused without per-call model load
    And it evicts after the idle timeout
    And an active MCP connection keeps it alive via heartbeat

  @failure @p1
  Scenario: Watcher keeps tool schemas stable
    Given the Mnemonic tool surface is registered with the watcher and banner active
    When the agent inspects tool schemas and sends bad watcher or status args
    Then no existing code_* tool schema is altered
    And the staleness banner is present on the response path
    And the bad args are rejected clearly
