# Source: docs/skillgrid/changes/013-mnemonic-layered-memory-governance/change.md
# Template: .agents/skills/_shared/templates/template-acceptance.feature
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.
# WHAT not HOW: no function names, file paths, or line numbers in Given/When/Then.

@step-01
Feature: Memory asset governance (owner, version, status, usage, visibility) — private by default
  As an operator
  I want every memory to be a governed asset with an owner, version history, status, usage, and visibility
  So that I can correct (not just delete) a wrong memory, control who/what can see what, and trust that sharing is an explicit act — never a leak

  # --- Threat: Mnemonic tool surface (01) ---
  @happy @p0
  Scenario: governance-tools-and-existing-schema-stable
    Given the governed memory surface is live
    When  the operator lists the available memory tools and inspects the existing memory tools' contracts
    Then  the share and governance query tools are registered
    And   every existing memory tool keeps its name and required parameters unchanged
    And   the new fields on existing responses are additive, not replacing existing fields

  @edge
  Scenario: bad-governance-args-rejected
    Given the governed memory surface is live
    When  the operator calls a governance tool with a missing or invalid argument
    Then  the call is rejected with a clear validation error
    And   the observation's visibility and governance fields are unchanged
    And   no governance data is invented

  # --- Threat: Data leak / visibility (01) ---
  @happy @p0
  Scenario: new-observation-private-by-default-with-owner
    Given a first owner saves a new observation without specifying a visibility
    When  the observation is created
    Then  its visibility is private
    And   its owner is the creating user/agent identity

  @happy @p0
  Scenario: private-observation-invisible-to-second-owner-until-share
    Given a first owner has a private observation
    When  a second owner searches for it before any share
    Then  the second owner's search does not return it
    When  the first owner explicitly shares it and the second owner searches again
    Then  the second owner's search returns it

  @edge @p0
  Scenario: restricted-acl-grant-enforced
    Given a first owner has an observation restricted to a named agent
    When  the granted agent reads it and a non-granted agent reads it
    Then  the granted agent can read it
    And   the non-granted agent sees it as absent (not found)

  @edge @p1
  Scenario: restricted-with-no-grants-is-owner-only
    Given a first owner sets an observation to restricted with no ACL grants
    When  the owner reads it and another owner reads it
    Then  the owner can read it
    And   another owner sees it as absent
    And   the condition is surfaced, not raised as an error

  @edge @p1
  Scenario: private-observation-absent-from-admin-cross-owner-list
    Given a first owner has a private observation
    When  an admin lists observations across all owners
    Then  the private observation is absent from that cross-owner list
    And   the admin's own observations are still listed

  @failure @p1
  Scenario: mem-share-unknown-target-rejected
    Given a first owner has a private observation
    When  the operator shares it to an unknown owner/agent/role
    Then  the share is rejected with a clear validation error
    And   the observation's visibility remains unchanged (still private)

  # --- Per-step WHAT bullets (01) ---
  @happy @p0
  Scenario: mem-update-appends-recoverable-version
    Given a first owner has an observation with known content
    When  the owner updates the observation's content
    Then  a new version row is appended instead of overwriting
    And   the prior content is still recoverable via the governance query
    And   the latest version is the read path
    And   the revision count advances

  @happy @p0
  Scenario: superseded-status-set-explicitly
    Given a first owner has a corrected atom that supersedes an old version
    When  the operator explicitly marks the old version's status
    Then  the old version's status is superseded (or archived)
    And   the status was set explicitly, never inferred from content

  @happy @p0
  Scenario: retrieval-usage-count-increments-on-search
    Given a first owner has an observation that a search will return
    When  a search returns the observation
    Then  the observation's retrieval usage count increments by one
    And   the count is distinct from the existing re-save (duplicate) counter

  @happy @p0
  Scenario: mem-governance-surfaces-asset-fields
    Given a first owner has an observation with an owner, a version history, a status, a usage count, and a visibility
    When  the operator runs the governance query for that observation
    Then  the result surfaces the owner, the version history, the status, the usage count, and the visibility

@step-02
Feature: Layered distillation (L0→L1 atoms→L2 scenario→L3 persona), provenance-linked
  As a coding agent
  I want a session's raw record to distill upward into atoms, a scenario, and a persona delta
  So that the next session bootstraps from stable layers instead of re-reading the raw record

  # --- Threat: Provenance integrity (02) ---
  @happy @p0
  Scenario: distilled-layer-carries-resolvable-l0-provenance
    Given a fixture session with a raw (L0) record containing distillable content
    When  the session-close distillation runs
    Then  each distilled L1/L2/L3 record carries a resolvable link to its L0 source

  @failure @p0
  Scenario: layer-with-unresolvable-source-not-created
    Given a distillation input whose L0 source cannot be resolved
    When  the distillation pass runs
    Then  no distilled record is created for that input
    And   a layer is never orphaned from its provenance

  @happy @p0
  Scenario: mem-layers-inspects-l0-to-l3-chain
    Given a session that was distilled into L1, L2, and L3 layers
    When  the operator inspects the layers for that session or topic
    Then  the result returns the L0→L1→L2→L3 chain
    And   each layer shows its provenance link to its L0 source

  @happy @p0
  Scenario: no-llm-floor-produces-provenance-ladder-offline
    Given a session with distillable content and no LLM available
    When  the distillation pass runs offline
    Then  the deterministic floor still produces L1 atoms
    And   the produced ladder is provenance-linked to the L0 source

  # --- Threat: Mnemonic tool surface (02) ---
  @happy @p0
  Scenario: mem-layers-registered-005-stable
    Given the layered memory surface is live
    When  the operator lists the available memory tools and inspects the 005 memory tools' contracts
    Then  the layers tool is registered
    And   every 005 memory tool keeps its name and required parameters unchanged

  @edge @p1
  Scenario: bad-layer-args-rejected
    Given the layered memory surface is live
    When  the operator calls the layers tool with a missing or invalid argument
    Then  the call is rejected with a clear validation error
    And   no layers are invented

  # --- Per-step WHAT bullets (02) ---
  @happy @p0
  Scenario: session-close-distill-l0-to-l1-l2-l3
    Given a session that is closed with opt-in distillation enabled
    When  the session-close distillation pass runs asynchronously
    Then  the L0 record is refined into L1 atoms (facts/preferences/constraints/events), L2 scenario block(s), and an L3 persona delta
    And   each produced layer is linked to its L0 source

  @edge @p0
  Scenario: distill-llm-cached-by-content-hash
    Given a session whose L0 content was already distilled
    When  the distillation runs again with an unchanged L0 source
    Then  the cached result is reused and no re-distillation occurs
    When  the L0 source changes and the distillation runs
    Then  re-distillation occurs

  @happy @p0
  Scenario: no-llm-floor-produces-l1-atoms
    Given a session with Key-Learnings/Lesson/Discovery-style content and no LLM available
    When  the distillation pass runs
    Then  L1 atoms are produced from the deterministic heuristics
    And   the ladder still works offline

  @edge
  Scenario: session-with-no-l1-content-is-noop
    Given a session with no new distillable content
    When  the session-close distillation pass runs
    Then  no empty atoms, scenarios, or persona delta are fabricated
    And   the pass is a no-op

  @happy @p1
  Scenario: l1-atom-correctable-with-traceable-provenance
    Given a distilled L1 atom with a provenance link to its L0 source
    When  the operator corrects the atom's content
    Then  the correction appends a version rather than deleting the atom
    And   the provenance link still traces the atom back to its L0 source to verify

@step-03
Feature: Layered retrieval (L2/L3-first, L1/L0 RRF fallback) with budgeted reads
  As a coding agent
  I want retrieval to bootstrap from stable layers and every read to be budgeted
  So that a 20-hit search can never drown the context window and I pull full content only on demand

  # --- Threat: Context-window flood (03) ---
  @happy @p0
  Scenario: twenty-hit-search-is-budgeted
    Given a store that would return 20 matching observations
    When  a search is run
    Then  the search returns 20 budgeted snippets, not 20 full-observation payloads
    And   each in-list snippet is truncated within the character budget
    And   the search completes within the context timeout

  @edge @p0
  Scenario: char-budget-truncates-with-explicit-omitted-count
    Given a read that would exceed the character budget
    When  the read is budgeted
    Then  the in-list snippet is truncated
    And   the truncation is explicit with an "N chars omitted" marker
    And   the truncation is never silent

  @failure @p1
  Scenario: context-timeout-returns-truncated-partial
    Given a read path that exceeds the context timeout
    When  the read is budgeted
    Then  a partial result is returned with a truncated flag set to true and a reason
    And   the agent is never hung

  # --- Threat: Mnemonic tool surface (03) ---
  @happy @p0
  Scenario: layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback
    Given a store with distilled L2/L3 layers and raw L1/L0 records
    When  a bootstrap read is run
    Then  L2/L3 results are returned first (cheap, stable)
    When  a specific-fact query is run
    Then  it falls back to L1/L0 via the existing ranked-fusion (RRF)

  @happy @p0
  Scenario: mem-get-observation-is-only-full-content-path
    Given a budgeted in-list result carrying an observation id
    When  the agent explicitly fetches that observation by id
    Then  full, untruncated content is returned
    And   no other read path returns full, untruncated content

  @edge @p0
  Scenario: budgeted-reads-005-stable
    Given the budgeted retrieval surface is live
    When  the operator inspects the 005 memory tools' contracts
    Then  every 005 memory tool keeps its name and required parameters unchanged
    And   the budgeted behavior is applied additively on the read paths

  @edge @p1
  Scenario: bad-retrieval-args-rejected
    Given the budgeted retrieval surface is live
    When  the operator calls a read path with a missing or invalid argument
    Then  the call is rejected with a clear validation error

  # --- Per-step WHAT bullets (03) ---
  @happy @p0
  Scenario: mem-context-search-timeline-budgeted
    Given the retrieval read paths are live
    When  the context, search, and timeline read paths are exercised
    Then  each is routed through the layered + budgeted path
    And   each enforces the item-count cap, character budget, and context timeout

  @happy @p0
  Scenario: every-inlist-result-carries-get-observation-id
    Given a search that returns multiple results
    When  the results are listed
    Then  every in-list result carries its full-content fetch id
    And   the agent can pull full content on demand

  @edge @p1
  Scenario: budget-is-tunable-via-config
    Given the retrieval budget has a default configuration
    When  the budget caps (item/char/timeout) are changed in configuration
    Then  the read paths honor the new caps

  @edge
  Scenario: cli-parity-for-layer-governance-share
    Given the CLI is live
    When  the operator invokes the layers, governance, and share commands
    Then  the CLI exposes parity with the corresponding memory tools
