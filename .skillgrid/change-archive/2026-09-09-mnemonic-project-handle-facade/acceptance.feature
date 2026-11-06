# Source: docs/skillgrid/changes/007-mnemonic-project-handle-facade/change.md
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.

@step-01
Feature: Exported Project Handle seam
  As a maintainer
  I want a deep Project Handle for one project
  So that single-project work does not require the wide facade

  @happy @p0
  Scenario: Opened handle exposes memory and web
    Given a Mnemonic data directory and a resolvable project
    When a Project Handle is opened for that project
    Then the handle reports the project id
    And memory and web accessors are available on the handle
    And the store was opened once for that open

  @edge
  Scenario: Cross-store root ops remain
    Given a Service with one or more project stores
    When cross-store list or resolve runs on the root Service
    Then the operation completes without requiring a Project Handle
    And single-project open still works afterward

  @failure @p1
  Scenario: Invalid project id aborts open
    Given a Service pointed at a data directory
    When a Project Handle open is attempted with an empty or invalid project id
    Then the open aborts with a clear error
    And no usable handle is returned

@step-02
Feature: MCP single-open via Project Handle
  As an agent
  I want mem code and web tools to open once
  So that each tool call does not pay a double store open

  @happy @p0
  Scenario: mem_save opens store once
    Given an injected Mnemonic Service for tests
    When mem_save runs for the current project
    Then the observation is saved
    And the project store is opened exactly once for that call

  @happy @p0
  Scenario: mem_search result shape unchanged
    Given saved observations in the project store
    When mem_search runs with the same parameters as before
    Then matching observations are returned
    And the result field names remain contract-compatible

  @edge
  Scenario: code and web tools use handle
    Given an injected Mnemonic Service for tests
    When a code tool and a web cache tool each run once
    Then each call succeeds through the Project Handle path
    And tool names remain unchanged

  @edge
  Scenario: Injected service still works
    Given tests override the Service used by MCP handlers
    When a memory tool runs
    Then it uses the injected Service
    And it does not silently fall back to a different data directory

  @failure @p1 @security
  Scenario: Tool contract survives handle rewire
    Given MCP tools after the Project Handle rewire
    When tools are listed and mem_save is invoked
    Then mem_save code and web_cache tools remain registered under the same names
    And a missing or failing open returns a tool error without a partial write

@step-03
Feature: HTTP single-open via Project Handle
  As an HTTP client
  I want single-project routes to open once
  So that REST matches the MCP lifecycle

  @happy @p0
  Scenario: Observation routes open store once
    Given an HTTP server with a Mnemonic Service
    When an observation is created and recent observations are listed
    Then both succeed
    And each request opens the project store at most once

  @edge
  Scenario: Single-project HTTP uses handle
    Given an HTTP server with a Mnemonic Service
    When session search code or web single-project routes are exercised
    Then each uses the Project Handle path
    And JSON field names remain unchanged

  @edge
  Scenario: Migrate and merge stay on root
    Given two project stores under the data directory
    When projects migrate or merge is requested over HTTP
    Then the root Service cross-store path runs
    And single-project routes still use a Project Handle afterward

  @failure @p1
  Scenario: Bad project on HTTP aborts
    Given an HTTP server with a Mnemonic Service
    When a single-project route is called with a missing or invalid project
    Then the request fails with an error response
    And no partial observation write occurs

@step-04
Feature: Collapsed double-open wrappers
  As a maintainer
  I want no double-open production path
  So that store lifecycle locality stays in the Project Handle

  @happy @p0
  Scenario: Facade path cannot double-open
    Given adapters already use the Project Handle
    When a former single-project facade entry is exercised if it still exists
    Then it does not open the same project store twice for one logical operation

  @edge
  Scenario: Dead aliases neutralized
    Given unused or duplicate single-project aliases on the root Service
    When the facade surface is audited after adapter migration
    Then dead aliases are removed or redirected without a second open

  @failure @p1
  Scenario: Integration smoke still passes
    Given MCP and HTTP wired through the deepened seam
    When integration smoke for memory save and HTTP health or observation runs
    Then both adapters succeed
    And Global Constraints on contracts remain held
