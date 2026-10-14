@context-orchestrator
# Context Orchestrator (CTX) — v1.11 delta
# Prerequisite plan: .skillgrid/artifacts/context-orchestrator-plan.md v1.11
# Gates on: 2026-10-06-context-harness-clm (ADR-0025..0028) + 2026-10-07-replace-opencode-hooks-with-plugins (ADR-0032)

Feature: CTX module layout and DB ownership
  As an operator
  I want CTX to be a thin 3rd Go module that reads Mnemonic's store
  So that CTX orchestrates without re-implementing storage or search

  @step-01
  Scenario: ctx-module-builds
    Given a go.work with the mnemonic and skillgrid-cli modules
    When I add the ctx module and go.work entry and build the workspace
    Then the workspace builds with three modules
    And the ctx module imports github.com/devopstales/skillgrid/mnemonic for store access
    And the ctx module opens no handle of its own to the shared store
    And the ctx module creates no table that is not prefixed ctx_

Feature: CTX CLI surface
  As an operator
  I want the CTX verbs under a mnemonic subcommand group
  So that I run one binary I already know

  @step-02
  Scenario: ctx-subcommand-group
    Given the mnemonic binary
    When I run mnemonic ctx stats
    Then it returns the unified CTX stats surface
    When I run mnemonic stats
    Then it returns the per-agent activity rollup unchanged
    And the existing mnemonic stats command is distinct from mnemonic ctx stats
    When I run mnemonic ctx --help
    Then it lists the CTX verbs proxy wrap stats search index purge metrics traffic learn judge checkpoint

  @step-03
  Scenario: ctx-stats-includes-clm-row-counts
    Given a store with tool_outputs and indexed_files rows
    When I run mnemonic ctx stats
    Then the output includes the tool_outputs row count as a sub-slice
    And the output includes the indexed_files row count as a sub-slice

Feature: CTX proxy surface
  As an agent client
  I want a transparent loopback proxy that applies the CTX pipeline
  So that I integrate with zero code change

  @step-04
  Scenario: proxy-three-modes
    Given a proxy bound to 127.0.0.1:8787 in cache mode
    When I send an Anthropic-compatible request through the proxy
    Then the request completes with savings headers present
    And X-CTX-Savings is accurate
    And the response is forwarded back to me
    When I send an OpenAI-compatible request through the proxy
    Then the request completes with savings headers present
    When a compression fault occurs mid-request
    Then the request still completes in passthrough

  @step-05
  Scenario: proxy-protocol-passthrough
    Given a proxy in passthrough mode
    When I send HTTP/1.1, HTTP/2, SSE, and WebSocket traffic through it
    Then each stream survives the proxy intact

  @step-06
  Scenario: compress-endpoint-loopback
    Given a proxy bound to 127.0.0.1:8787
    When I POST a message list to /v1/compress from loopback
    Then it returns a compressed message list
    And no upstream completion is generated
    When I POST the same to /v1/compress from a non-loopback origin
    Then it returns 404
    And it does not return 403

  @step-07
  Scenario: mode-switch-emitted
    Given a proxy running in cache mode
    When I switch the mode to token at runtime
    Then a row is written to ctx_proxy_mode_history with the old and new mode
    And a proxy-mode-changed event is emitted
    And the current mode token is exposed in GET /stats
    When I set the mode via CTX_PROXY_MODE and restart
    Then the proxy starts in that mode

  @step-08
  Scenario: proxy-fail-open-provider-timeout
    Given a proxy configured with provider_retry_backoff_ms
    When the upstream provider times out
    Then the proxy retries with the configured backoff
    And after the final retry it returns the provider error verbatim

Feature: CTX two-tier integration
  As an operator
  I want to launch a named agent through the proxy with one command
  So that I do not hand-edit agent config

  @step-09
  Scenario: wrap-core4-launches
    Given a machine with claude installed
    When I run mnemonic ctx wrap claude
    Then the proxy starts
    And claude's config is injected to point at the proxy
    And the previous config is backed up
    And claude is launched
    When claude runs a turn through the proxy
    Then the turn completes with savings headers present
    And claude's code is not modified

  @step-10
  Scenario: wrap-optin-agent
    Given a machine with aider installed
    And the wrap config listing aider as opt-in
    When I run mnemonic ctx wrap aider
    Then the proxy starts and aider's config is injected
    And aider is launched through the proxy

Feature: CTX observability
  As an operator
  I want read-only observability on the existing UI server
  So that I can see savings without a second process or port

  @step-11
  Scenario: observability-endpoints
    Given the mnemonic serve UI server
    When I GET /stats
    Then it returns live session metrics as JSON
    When I GET /stats-history
    Then it returns hourly daily weekly and monthly rollups
    And the rollups survive a server restart
    When I GET /metrics
    Then it returns Prometheus-parseable metrics
    When I GET /api/session-summaries
    Then it returns paginated per-session wide events

  @step-12
  Scenario: session-summary-aggregated
    Given a session with tool events and proxy savings
    When the session ends
    Then a session_summary_aggregate job is enqueued
    And exactly one row is written to ctx_session_summaries
    And the row counts match the session's events
    And the SessionEnd hook completes within its budget
    And the agent is not blocked by the aggregation

  @step-13
  Scenario: history-snapshot-durable
    Given rollups have been written to ctx_stats_history
    When the snapshot is written
    Then .skillgrid/ctx/stats_history.json matches the DB rollups
    When the project is re-indexed
    Then the snapshot file is intact
    And it still matches the DB

  @step-14
  Scenario: run-mode-is-observability-signal
    Given a proxy in cache mode
    When I GET /stats
    Then the current mode cache is shown
    When the mode changes
    Then the Live view header reflects the new mode

Feature: CTX real-time traffic learning
  As an operator
  I want reusable memories mined from the session-event stream during the session
  So that the agent learns without LLM cost

  @step-15
  Scenario: traffic-learner-extracts
    Given a session where a tool call fails then succeeds on retry
    When the traffic learner runs over the session-event stream
    Then an error_recovery memory is stored in ctx_traffic_memories
    And the memory carries its evidence_json
    And no LLM call is made during extraction

  @step-16
  Scenario: traffic-learner-four-categories
    Given sessions exhibiting error recovery, stable facts, repeated preferences, and file-structure patterns
    When the traffic learner runs
    Then it extracts memories in all four categories
    And each memory is below the max_memories bound
    And a memory below min_confidence is not stored
    And the oldest low-confidence memory is evicted first when the store is full

  @step-17
  Scenario: traffic-promotion-human-gate
    Given a high-confidence traffic memory
    When no human has promoted it
    Then it is not applied as a rule
    When I run mnemonic ctx learn apply on the rule id
    Then the rule is active
    When I run mnemonic ctx learn revoke on the rule id
    Then the rule is removed
    And a traffic memory never contradicts the offline ctx learn loop

Feature: CTX judgment layer
  As the pipeline
  I want per-unit truncate decisions from a small local model
  So that context stays within budget without deleting or rewriting

  @step-18
  Scenario: judgment-truncate-only
    Given a context unit above the judgment threshold
    When the judgment layer evaluates it
    Then the unit is truncated
    And the unit is not deleted
    And no stored message is rewritten

  @step-19
  Scenario: judgment-fingerprint-cached
    Given a context unit already judged
    When the same unit is presented again
    Then the decision is served from the fingerprint cache
    And no model call is made

  @step-20
  Scenario: judgment-fail-open
    Given the local Ollama endpoint is unavailable
    When a context unit needs a judgment
    Then the deterministic pre-filter still runs
    And nothing blocks the request
    And the unit is not dropped

Feature: CTX config
  As an operator
  I want CTX config in the config.d pattern
  So that per-machine and per-project settings coexist

  @step-21
  Scenario: ctx-config-precedence
    Given a machine-local ~/.skillgrid/config.d/ctx.yaml
    And a repo-local .skillgrid/config.d/ctx.yaml with the same key
    When CTX loads its config
    Then the repo-local value wins for that key
    When the repo-local file is absent
    Then the machine-local value is used
    When neither file is present
    Then all defaults are used
    And the clm block is still read from .skillgrid/config.yaml

Feature: CLM absorption amendment
  As the project
  I want CLM's unapplied migrations renumbered and its row counts in ctx stats
  So that CTX's tables start clean at 056

  @step-22
  Scenario: clm-renumber
    Given the CLM spec naming migrations 050 tool_outputs, 051 indexed_files, 052 context_revisions
    And those migrations are not yet applied
    When the CTX change lands
    Then CLM's migrations are renumbered to 053 tool_outputs, 054 indexed_files, 055 context_revisions
    And no CLM migration is numbered 050, 051, or 052
    And CTX's first migration is 056
    And the ctx_ tables 056+ apply cleanly after 053/054/055

  @step-23
  Scenario: clm-stats-subslice
    Given the CLM tables tool_outputs and indexed_files exist
    When I run mnemonic ctx stats
    Then the output includes the CLM row counts as a sub-slice
    And the existing mnemonic stats per-agent rollup is unchanged
