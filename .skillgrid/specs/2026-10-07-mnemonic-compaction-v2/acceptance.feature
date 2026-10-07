Feature: Mnemonic compaction v2
  Mnemonic compaction becomes smart, lossless, and instant: an advisory gate
  decides when it is safe to compact, a structured six-section prompt preserves
  what matters, a steering column lets the model manage its own context, and a
  proactive background build pre-summarizes so the real compaction is instant.
  All four capabilities are opt-in; a down LLM never blocks a compaction (fail-open).

  Background:
    Given a mnemonic store is reachable
    And the session events and observations tables are populated for session "s1"

  # --- Req 1: advisory timing gate ---

  Scenario: happy path advisory hint curve
    Given the adviser is enabled with budget 1000 chars
    And the session context is empty
    When I GET /compaction/advice for session "s1"
    Then the response floor is 0.90
    And the response hint is true when the LLM reports finished=true and hands_on=false
    When the session context fills to 1000 chars
    And I GET /compaction/advice for session "s1"
    Then the response floor is 0.50

  Scenario: happy path combined LLM call parses finished and hands_on
    Given the LLM returns JSON {"finished": true, "hands_on": true}
    When I GET /compaction/advice for session "s1"
    Then the score is 0.5
    And the hint is false when the floor exceeds 0.5

  Scenario: error path adviser fails open on LLM error
    Given the LLM is unreachable
    When I GET /compaction/advice for session "s1"
    Then the response is 200
    And the hint is false
    And the reason is "llm unavailable"
    And no compaction was blocked

  # --- Req 2: structured six-section prompt ---

  Scenario: happy path structured sections preserve corrections
    Given session "s1" has a user correction "use bcrypt not md5"
    And an error "connection refused on :5432"
    And an in-progress task "migrate users table"
    When I build the CompactionContext for session "s1"
    Then the section "Errors & Corrections" contains "use bcrypt not md5" verbatim
    And the section "Errors & Corrections" contains "connection refused on :5432"
    And the section "Active Work" contains "migrate users table"
    And the sections "User Intent", "Actions Succeeded", "Pending Tasks", and "Critical Details" are present

  Scenario: happy path prompt renders six sections in preserve order
    Given the CompactionContext has populated sections
    When I render the compaction prompt
    Then the prompt contains the sections "User Intent", "Actions Succeeded", "Errors & Corrections", "Active Work", "Pending Tasks", and "Critical Details"
    And corrections appear before completed work in the rendered order

  # --- Req 3: CLM steering column ---

  Scenario: happy path hookCompact appends a revision
    Given the adviser is enabled
    When I trigger hookCompact for session "s1"
    Then context_revisions has a row with revision 1 for session "s1"
    When I trigger hookCompact for session "s1" again
    Then context_revisions has a row with revision 2 for session "s1"
    And revision 1 is not overwritten

  Scenario: happy path steering re-injection
    Given context_revisions revision 1 for session "s1" has steering "keep all migration IDs"
    When I trigger hookCompact for session "s1"
    Then the compaction prompt input for revision 2 contains "keep all migration IDs"
    And the CompactionContext for session "s1" returns steering "keep all migration IDs"

  Scenario: happy path steering column migration is additive
    Given an existing context_revisions table from migration 052
    When migration 053 applies
    Then the steering column exists
    And all existing revision rows are preserved
    And the session-scoped purge (ADR-0028) still applies

  # --- Req 4: proactive instant compaction ---

  Scenario: happy path proactive pre-build on interval
    Given proactive is enabled with interval 50ms and budget 1000 chars
    And session "s1" context fraction is 0.4
    When 150ms elapse without a session.idle event
    Then a context_revisions row exists for session "s1"
    And the revision was built proactively

  Scenario: happy path proactive build skips below threshold
    Given proactive is enabled with interval 50ms
    And session "s1" context fraction is 0.2
    When 150ms elapse
    Then no context_revisions row was created for session "s1"
    And no build was attempted below the 0.3 threshold

  Scenario: error path proactive build never fails the request
    Given the proactive build LLM or store errors once
    When the proactive build runs
    Then a warning is logged
    And the request that triggered the check returned success
    And the next interval retries the build

  # --- Req 5: config + service wiring ---

  Scenario: happy path compaction config defaults off
    Given no mnemonic.compaction block exists
    When the config is loaded
    Then adviser_enabled is false
    And proactive is false
    And budget_chars defaults to 60000
    And interval defaults to 15m
    And max_revisions defaults to 10

  Scenario: happy path compaction config override
    Given mnemonic.compaction.adviser_enabled is true and budget_chars is 12000
    When the config is loaded
    Then the service compaction config has AdviserEnabled true
    And BudgetChars 12000
    And the adviser uses BudgetChars 12000 for its contextFraction

  # --- Req 6: HTTP routes + plugin wiring ---

  Scenario: happy path compaction advice routes
    Given a session "s1" exists
    When I GET /compaction/advice?session_id=s1
    Then the response is 200 with fields score, hint, floor, reason
    When I POST /compaction/advice with the structured six-section body
    Then a context_revisions row is appended
    And the response carries the new revision number

  Scenario: happy path mem_compact_advice tool
    Given the mnemonic-memory plugin is loaded
    When I call mem_compact_advice for session "s1"
    Then the tool returns fields score, hint, floor, reason

  Scenario: happy path skillgrid-compaction plugin injects the hint
    Given the skillgrid-compaction plugin is loaded
    When experimental.session.compacting fires for session "s1"
    Then the plugin GETs /compaction/advice
    And a one-line advisory hint is injected into the compaction prompt
    And the structured CompactionContext sections are POSTed to /compaction/advice

  Scenario: happy path skillgrid-events plugin reports context chars
    Given the skillgrid-events plugin is loaded
    When tool.execute.after fires
    Then the context char count is POSTed to the adviser input
    And the adviser contextFraction reflects the reported count

  Scenario: error path replace-first single-path wiring
    Given the retired shell hooks are removed (replace-hooks plan landed)
    When a compaction event fires
    Then the event is handled by the 5 new TS plugins only
    And no retired shell hook path is invoked
