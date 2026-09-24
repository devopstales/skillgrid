Feature: Bi-temporal observations and AUDN save classification
  As an agent using the Mnemonic memory system
  I want facts to carry a validity window and the save path to classify each write
  So that I can answer "what was true when?" and the system auto-supersedes stale facts

  Background:
    Given a project store with the bi-temporal migration applied
    And the observation table has valid_at, invalid_at, and superseded_by columns

  Rule: Bi-temporal columns on observations
    ### Requirement: New observations are valid from creation
    When I save a new observation
    Then its valid_at equals its created_at
    And its invalid_at is null
    And its superseded_by is null

    ### Requirement: Superseded observations are excluded from search
    Given observation A is active and valid
    And observation B supersedes A
    When I search for the shared keywords
    Then A does not appear in the results
    And B appears in the results

    ### Requirement: Time-travel query returns the fact valid at time T
    Given observation A was valid from T1 to T2 (A.valid_at = T1, A.invalid_at = T2)
    And observation B is valid from T2 onward (B.valid_at = T2, B.invalid_at = NULL)
    And T3 is a time after T2 (T2 <= T3)
    When I query ValidAtTime for the shared keywords at time T1
    Then A is returned
    And B is not returned (B.valid_at = T2 > T1)
    When I query ValidAtTime for the shared keywords at time T3
    Then B is returned
    And A is not returned (A.invalid_at = T2 <= T3)

  Rule: AUDN save classification
    ### Requirement: Save returns an action label
    When I save a new fact with no existing match
    Then the response includes action "add"
    When I save a fact that is a duplicate
    Then the response includes action "noop"
    When I save a fact that refines an existing one
    Then the response includes action "update"
    When I save a fact that supersedes an existing one
    Then the response includes action "delete"
    And the response includes superseded_id pointing to the old observation

    ### Requirement: Delete arm produces the supersede chain
    When I save a new fact that supersedes observation A
    Then A's invalid_at is set to the save time
    And A's superseded_by points to the new observation
    And A's status is "superseded"
    And a supersedes edge exists in memory_relations from A to the new observation

    ### Requirement: Deterministic floor without LLM
    Given the LLM dedup is disabled
    When I save a fact that hash-matches an existing one
    Then the response includes action "noop"
    And the existing observation's duplicate_count is incremented (BumpDuplicate)
    And the existing observation's last_seen_at is refreshed
    When I save a fact that does not match
    Then the response includes action "add"
    And a new observation row is inserted

    ### Requirement: LLM error degrades to deterministic floor
    Given the LLM dedup is enabled but the LLM call fails
    When I save a new fact that does not hash-match
    Then the response includes action "add"
    And a new observation row is inserted
    And the save succeeds (the LLM error is non-fatal, logged as a warning)
    When I save a fact that hash-matches an existing one
    Then the response includes action "noop"
    And the existing observation's duplicate_count is incremented

  Rule: Backward compatibility
    ### Requirement: Existing Save callers are unaffected
    When an existing caller invokes Save (not SaveWithAction)
    Then the return type is still (int64, error)
    And the observation is created with the same behavior as before

    ### Requirement: mem_save MCP response is additive
    When I call mem_save via MCP
    Then the response includes the existing id and project fields
    And the response includes the new action field
    And old consumers that ignore unknown fields are unaffected
