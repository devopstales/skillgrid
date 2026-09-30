Feature: Mnemonic Second Brain

  Background:
    Given a mnemonic store with a project "test-project"

  # C1 — Natural-language capture

  Scenario: mem_save infer fills empty type deterministically
    Given the text "we decided to use SQLite for the memory store"
    When I call mem_save with infer true and empty type
    Then the saved observation type is "decision"

  Scenario: mem_save infer fills empty topic_key
    Given the text "fixed the N+1 query in the user list"
    When I call mem_save with infer true and empty topic_key
    Then the saved observation has a non-empty topic_key

  Scenario: mem_save infer preserves agent-provided values
    Given the text "we decided to use SQLite"
    When I call mem_save with infer true, type "architecture" and topic_key "architecture/store"
    Then the saved observation type is "architecture"
    And the saved observation topic_key is "architecture/store"

  Scenario: mem_save infer does not run by default
    Given the text "we decided to use SQLite"
    When I call mem_save with empty type and infer unset
    Then the saved observation uses the default type
    And no inference heuristic ran

  Scenario: skill trigger phrase maps to mem_save
    Given the capture contract in the mnemonic-second-brain skill
    When the user says "remember this: we decided to use SQLite"
    Then a mem_save call is made with type "decision"
    And the content includes a What section

  Scenario: rephrased capture upserts via topic_key
    Given an observation already saved with topic_key "architecture/store"
    When I call mem_save with the same topic_key "architecture/store"
    Then no duplicate observation is created
    And the existing observation is updated

  # C2 — mem_ask synthesis

  Scenario: mem_ask cited mode works with no embedder
    Given 10 observations in "test-project" matching "auth"
    And no embedder is configured
    When I call mem_ask with query "auth" and mode cited for "test-project"
    Then the response contains a non-empty citations array
    And every citation has an id, title and type
    And the response degraded field is true
    And the response matched_via field is "keyword"

  Scenario: mem_ask cited mode is token-bounded
    Given 50 observations in "test-project" matching "auth"
    When I call mem_ask with query "auth" and mode cited and max_tokens 400
    Then the response is at most 400 tokens

  Scenario: mem_ask hybrid when embedder active
    Given an embedder is configured
    And 10 embedded observations in "test-project"
    When I call mem_ask with query "authentication" and mode cited for "test-project"
    Then the response degraded field is false
    And the response matched_via field is "hybrid"

  Scenario: mem_ask llm mode returns cited prose and fails open
    Given a reachable LLM
    And 5 observations in "test-project" matching "auth"
    When I call mem_ask with query "what did we decide about auth" and mode llm
    Then the response answer field is non-empty prose
    And the answer contains an [obs: reference
    And the response sources array lists observation ids

  Scenario: mem_ask llm mode fails open to cited
    Given an LLM that errors
    And 5 observations in "test-project" matching "auth"
    When I call mem_ask with query "auth" and mode llm
    Then the response is returned without error
    And the response contains a non-empty citations array

  Scenario: mem_ask is project-scoped by default
    Given 5 observations matching "auth" in "project-a" and 5 in "project-b"
    When I call mem_ask with query "auth" for "project-a" without all_projects
    Then every citation belongs to "project-a"

  Scenario: mem_ask spans projects with all_projects
    Given 5 observations matching "auth" in "project-a" and 5 in "project-b"
    When I call mem_ask with query "auth" for "project-a" with all_projects true
    Then citations may belong to both projects

  Scenario: mem_ask is registered as an MCP tool
    When I inspect the MCP server tools
    Then the tool "mem_ask" is registered

  # C3 — Knowledge lifecycle

  Scenario: mem_lifecycle health returns a report
    Given 20 observations in "test-project"
    When I call mem_lifecycle with action health for "test-project"
    Then the response contains counts by type
    And the response contains embedding coverage
    And the response contains an age distribution
    And the response contains a duplicate density value
    And the response contains a recommendations array

  Scenario: mem_lifecycle health is cached and never throws
    Given a lifecycle health computation that errors
    When I call mem_lifecycle with action health
    Then the response is returned without error
    And the recommendations array is empty

  Scenario: mem_lifecycle dedup scan returns clusters
    Given 10 observations with 3 near-duplicate pairs (cosine > 0.85)
    When I call mem_lifecycle with action dedup subaction scan and dry_run true
    Then the response contains a clusters array
    And the response contains a duplicate density
    And no observation was modified

  Scenario: mem_lifecycle dedup merge archives near-dupes with provenance
    Given 2 observations that are near-duplicates
    When I call mem_lifecycle with action dedup subaction merge keeping the canonical id
    Then the duplicate is soft-archived
    And the canonical is not archived
    And the canonical metadata contains consolidated_from with the duplicate id

  Scenario: mem_lifecycle dedup degrades to hash without embedder
    Given no embedder is configured
    And 10 observations with 2 identical-hash pairs
    When I call mem_lifecycle with action dedup subaction scan
    Then the response clusters are hash-based
    And the response is marked degraded

  Scenario: mem_lifecycle archive and restore
    Given an observation in "test-project"
    When I call mem_lifecycle with action archive subaction archive obs_ids [the id] reason "stale"
    Then the observation is archived with archive_reason "stale"
    When I call mem_lifecycle with action archive subaction restore obs_ids [the id]
    Then the observation is restored

  Scenario: mem_lifecycle archive finds stale
    Given 5 observations older than 90 days and 5 recent
    When I call mem_lifecycle with action archive subaction stale stale_days 90
    Then the response lists the 5 older observations

  Scenario: every mutating lifecycle op writes an audit row
    When I call mem_lifecycle with action archive subaction archive obs_ids [an id]
    Then a lifecycle_log row is written with status completed
    And the row has non-empty input_ids
    And the row has a completed_at timestamp

  Scenario: health warnings do not break search
    Given a lifecycle health that reports a HIGH duplicate warning
    When I call mem_search with query "auth"
    Then the response results are returned normally
    And the response _health_warnings array contains the HIGH warning
    And the result ordering is unaffected

  Scenario: health warnings are empty on computation error
    Given a lifecycle health that errors
    When I call mem_search with query "auth"
    Then the response _health_warnings array is empty
    And the response results are returned normally

  # C4 — MCP response-shape convention

  Scenario: errors are returned as values not thrown
    When I call mem_ask with a query for an unknown project "nope"
    Then the response contains an error field with actionable next-step text
    And no exception is thrown to the MCP transport

  Scenario: failed reads return an empty list
    When I call mem_lifecycle with action archive subaction list for a project with no archived obs
    Then the response list is empty
    And the response has no error field

  Scenario: meta fields are underscore-prefixed
    When I call mem_ask with query "auth"
    Then the response meta fields (timing, health warnings, source project) are underscore-prefixed
    And none of the meta fields collide with payload fields
