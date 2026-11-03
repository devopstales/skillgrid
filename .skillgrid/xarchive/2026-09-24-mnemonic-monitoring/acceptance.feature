Feature: Monitoring — Track Every Tool Call

  Background:
    Given a mnemonic store with a project "test-project"
    And a registered session "sess-1" for "test-project"

  Scenario: tool-calls route writes a session_events row
    When I POST to /sessions/sess-1/tool-calls with tool_name "bash" and command "go test ./..."
    Then a session_events row exists with action_type "command_exec", tool_name "bash", command "go test ./..."

  Scenario: tool-calls route returns 404 for unknown session
    When I POST to /sessions/no-such/tool-calls with tool_name "bash"
    Then the response status is 404
    And no session_events row is written

  Scenario: tool-calls route flags sensitive paths
    When I POST to /sessions/sess-1/tool-calls with tool_name "read" and path "/home/u/.env"
    Then a session_events row exists with action_type "file_read" and is_sensitive true

  Scenario: plugin hook fires for non-Task tools
    When the opencode plugin tool.execute.after fires for tool "bash"
    Then a POST is made to /sessions/{id}/tool-calls with tool_name "bash"

  Scenario: plugin hook does not fire the tool-call POST for Task
    When the opencode plugin tool.execute.after fires for tool "Task"
    Then no POST is made to /sessions/{id}/tool-calls
    And the existing Task passive-capture path runs

  Scenario: plugin hook strips private spans
    When the plugin captures tool output containing "<private>secret123</private>"
    Then the stored content_preview does not contain "secret123"

  Scenario: plugin hook strips always-private tool output
    Given SKILLGRID_MNEMONIC_PRIVATE_TOOLS is "mem_save"
    When the plugin captures tool "mem_save" with output "full body here"
    Then the stored content_preview does not contain "full body here"

  Scenario: plugin hook is fail-open
    When the POST to /sessions/{id}/tool-calls fails (server down)
    Then the tool call is not blocked or failed

  Scenario: mem_query_events filters by action
    Given session "sess-1" has command_exec and file_read events
    When I call mem_query_events with action "command_exec" and session "sess-1"
    Then all returned events have action_type "command_exec"

  Scenario: mem_query_events excludes sensitive by default
    Given session "sess-1" has sensitive and non-sensitive events
    When I call mem_query_events with session "sess-1"
    Then no returned event has is_sensitive true

  Scenario: mem_query_events includes sensitive with flag
    Given session "sess-1" has sensitive events
    When I call mem_query_events with session "sess-1" and sensitive true
    Then a returned event has is_sensitive true

  Scenario: mem_query_events count-only
    Given session "sess-1" has 10 events
    When I call mem_query_events with session "sess-1" and count true
    Then the events array is empty
    And the count is 10

  Scenario: mem_export_events returns JSONL
    Given session "sess-1" has 5 events
    When I call mem_export_events with session "sess-1"
    Then the response jsonl field has 5 lines
    And each line is valid JSON

  Scenario: hooks default to enabled
    Given a config with mnemonic enabled and no hooks section
    When the config is loaded
    Then mnemonic.hooks.enabled is true

  Scenario: retention days default to 90
    Given a config with mnemonic enabled and no retention_days
    When the config is loaded
    Then mnemonic.retention_days is 90

  Scenario: retention prunes old events
    Given session_events with 5 events 100 days old and 5 recent
    When I run PruneOldEvents with days 90
    Then 5 events are deleted
    And no remaining event is older than 90 days

  Scenario: retention keeps recent events
    Given session_events with only recent events
    When I run PruneOldEvents with days 90
    Then 0 events are deleted
