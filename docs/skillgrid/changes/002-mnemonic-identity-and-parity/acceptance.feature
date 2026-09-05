# Source: docs/skillgrid/changes/002-mnemonic-identity-and-parity/change.md
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.

@step-01
Feature: Clone-private identity binding
  As an agent
  I want a stable, clone-private project identity
  So that memories are not scattered across invisible stores

  @happy @p0
  Scenario: Project binds to its clone
    Given a git repository checkout
    When the project is resolved
    Then the project id is bound to that clone and never re-derived from mutable git state

  @happy @p0
  Scenario: Worktree and main checkout share project id
    Given a linked worktree of the same clone as the main checkout
    When the project is resolved from each
    Then both resolutions yield the identical project id

  @happy @p0
  Scenario: Remote change and sibling path keep project id
    Given a repository whose remote URL was changed and whose checkout was copied to a sibling path
    When the project is resolved from the changed and copied locations
    Then the project id matches the original binding

  @edge
  Scenario: Single child auto-promotes
    Given a parent directory with exactly one child git repository
    When the project is resolved from that parent
    Then the child is promoted as the project with a soft warning

  @edge
  Scenario: Multi-repo parent returns AvailableProjects
    Given a parent directory with more than one child git repository
    When the project is resolved from that parent
    Then the result is ambiguous with AvailableProjects and no silent directory-hash store is written

  @edge
  Scenario: Config walk stops at repository root
    Given a path whose ancestors include config outside the enclosing repository root
    When the config is looked up during resolution
    Then the walk stops at the enclosing repository root

  @edge
  Scenario: Prior keys alias to canonical id
    Given a prior directory-hash or remote-key store for the same clone
    When a new binding is created
    Then prior keys route to the new canonical id via aliases

  @edge
  Scenario: MNEMONIC_PROJECT selects among candidates
    Given an ambiguous parent with AvailableProjects
    When MNEMONIC_PROJECT names one of the candidates
    Then resolution uses that selected project id

  @edge
  Scenario: Store open is idempotent under remapped id
    Given two working directories that map to the same project id
    When the store is opened for each
    Then both opens succeed without collision under the remapped id

  @failure @p1
  Scenario: Binding write failure does not fall through to path-hash
    Given the identity binding cannot be written because of permissions on the git common-dir
    When the project is resolved
    Then resolution aborts with a clear error and does not invent an unstable path-hash id

@step-02
Feature: Cross-store recall and alias unification
  As an agent
  I want recall across every store
  So that fragmented stores become one logical index

  @happy @p0
  Scenario: Recall spans every store
    Given data stored in multiple stores under the Mnemonic store directory
    When recalled with all projects enabled
    Then the results are merged and re-ranked across every store

  @happy @p0
  Scenario: all_projects search merges two stores
    Given two seeded stores with distinct observations
    When mem_search runs with all_projects true
    Then hits from both stores appear in the merged ranking

  @edge
  Scenario: Fragmented stores are one logical index
    Given stores that were previously fragmented and linked by aliases
    When queried or unified
    Then they are treated as one logical index

  @edge
  Scenario: mem_unify is idempotent on already-unified keys
    Given source keys that were already unified into the canonical project
    When mem_unify runs again for those keys
    Then the operation succeeds without a server error and records no harmful duplicate state

  @failure @p1
  Scenario: Missing data yields no result
    Given no data present across the stores being searched
    When recalled with all projects enabled
    Then an empty merged result is returned without a hard failure

@step-03
Feature: Observation lifecycle parity
  As an agent
  I want pinning, expiry, duplicate count, and recency honoured
  So that recall quality matches the reference

  @happy @p0
  Scenario: Lifecycle columns are honoured
    Given observations with pin, expiry, duplicate count, and recency state
    When context or search is evaluated
    Then pinning, expiry, duplicate count, and recency affect ordering and exclusion

  @happy @p0
  Scenario: Pin and unpin reorder context
    Given an unpinned observation that can be pinned
    When mem_pin then mem_unpin are applied
    Then context ordering reflects the pin boost and then returns to normal recency

  @edge
  Scenario: Expired entries are soft-excluded
    Given entries past their expires_at
    When context or search is evaluated
    Then they are no longer returned as live hits

  @edge
  Scenario: tool_name provenance is stored on save
    Given a save that provides tool_name
    When the observation is persisted
    Then tool_name provenance is stored on that observation

  @failure @p1
  Scenario: Invalid lifecycle state is rejected
    Given an invalid pin id or malformed expires_at
    When a lifecycle operation is attempted
    Then the request is rejected with a structured validation error and not a server error

@step-04
Feature: Optional embedding recall
  As an agent
  I want vector recall behind the flag
  So that semantic recall is available alongside keyword search

  @happy @p0
  Scenario: Vector recall is available behind the flag
    Given the embedding flag is enabled and vector data is present
    When searched
    Then vector recall is returned fused with keyword results via reciprocal-rank fusion

  @happy @p0
  Scenario: Flag on fuses vector and keyword results
    Given MNEMONIC_EMBED is set and embeddings exist for some observations
    When mem_search runs
    Then the ranking reflects fused keyword and vector order without requiring a cloud embedder always-on

  @edge
  Scenario: Keyword-only fallback when vectors are absent
    Given the embedding flag is enabled but no vector data is present
    When searched
    Then only keyword results are returned

  @edge
  Scenario: Missing embedder degrades to keyword-only
    Given the embedding flag is enabled but the embedder is unavailable
    When searched
    Then keyword-only results are returned without a hard failure

  @failure @p1
  Scenario: Disabled flag yields no vector recall
    Given the embedding flag is disabled
    When searched
    Then no vector recall path runs and only keyword results are returned
