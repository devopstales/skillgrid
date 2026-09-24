Feature: Session Context Injection

  Background:
    Given a mnemonic store with a project "test-project"
    And a completed session "sess-1" for "test-project" with 15 events

  Scenario: L1 summary is deterministic
    When I call DistillSummary for session "sess-1" with max_tokens 800 twice
    Then both summaries are byte-identical

  Scenario: L1 summary respects token cap
    Given a session "sess-2" with 60 events
    When I call DistillSummary for session "sess-2" with max_tokens 800
    Then the summary is at most 800 tokens

  Scenario: L1 summary excludes sensitive events
    Given a session "sess-3" with 5 normal events and 3 sensitive events touching ".env" files
    When I call DistillSummary for session "sess-3" with max_tokens 2000
    Then the summary does not contain ".env"

  Scenario: L1 summary is empty for few events
    Given a session "sess-4" with 2 events
    When I call DistillSummary for session "sess-4" with max_tokens 800
    Then the summary is empty

  Scenario: Auto-prepend on resume returns summary
    When I call AutoPrepend for project "test-project" with max_tokens 800
    Then the result contains "Prior Session Summary"

  Scenario: Auto-prepend on fresh session returns empty
    Given no completed session for project "fresh-project"
    When I call AutoPrepend for project "fresh-project" with max_tokens 800
    Then the result is empty

  Scenario: Hybrid retrieve BM25-only when no embedder
    Given no embedder is configured
    And 10 observations in "test-project"
    When I call HybridRetrieve with query "auth" for project "test-project"
    Then the result is degraded
    And the result contains items

  Scenario: Hybrid retrieve vector leg when embedder active
    Given an embedder is configured
    And 10 embedded observations in "test-project"
    When I call HybridRetrieve with query "authentication" for project "test-project"
    Then the result is not degraded
    And the result contains items

  Scenario: Hybrid retrieve is project-scoped
    Given 5 observations in "project-a" and 5 in "project-b"
    When I call HybridRetrieve with query "query" for project "project-a" without all_projects
    Then all items belong to "project-a"

  Scenario: Hybrid retrieve all projects
    Given 5 observations in "project-a" and 5 in "project-b"
    When I call HybridRetrieve with query "query" for project "project-a" with all_projects
    Then items may belong to both projects

  Scenario: mem_inject_session tool is registered
    When I inspect the MCP server tools
    Then the tool "mem_inject_session" is registered

  Scenario: mem_inject_session returns ranked items
    Given 10 observations in "test-project"
    When I call mem_inject_session with query "auth" for project "test-project"
    Then the response contains a "block" field
    And the response contains an "items" array
    And every item has a "token_cost" greater than 0

  Scenario: mem_inject_session all projects
    Given 5 observations in "project-a" and 5 in "project-b"
    When I call mem_inject_session with query "query" for project "project-a" and all_projects true
    Then the response items may include observations from both projects

  Scenario: mem_inject_session degraded flag
    Given no embedder is configured
    When I call mem_inject_session with query "query" for project "test-project"
    Then the response "degraded" field is true

  Scenario: Context block includes token costs
    Given a retrieval result with 2 items (50 tokens and 80 tokens)
    When I call RenderContextBlock
    Then the block contains "50 tokens"
    And the block contains "80 tokens"
    And the block contains "130 tokens total"

  Scenario: Context block shows BM25-only when degraded
    Given a retrieval result with degraded true
    When I call RenderContextBlock
    Then the block contains "BM25-only"
