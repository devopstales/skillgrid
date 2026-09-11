# Source: docs/skillgrid/changes/009-web-admin-dashboard/change.md
# Trace: change.md ## Goal + ## Definition of Done; tasks.md @step-NN verify lines.
# Mapping: @p0 scenarios ↔ change.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per step; tag each Feature with @step-NN matching tasks.md.
# Execution runs P1 → P6 in order; each phase is independently shippable
# (shipped entries work, future entries show as disabled stubs).
# Note: @step-04 renders 013's governance/layer data as a forward-compat
# placeholder when 013 is absent (soft dependency — the Memory entry stays
# fully interactive with or without 013, per-widget error isolation).

@step-01
Feature: Dashboard shell with menu bar, static Welcome page, and swagger passthrough
  As an operator of a skillgrid-installed machine
  I want the dashboard at / to open on a menu shell with a Welcome page
  So that every later phase has a foundation to extend one entry at a time

  @happy @p0
  Scenario: GET / serves the shell with menu bar and path routing
    Given a running skillgrid serve
    When  I open http://127.0.0.1:7438/
    Then  I see a menu bar (Welcome, Tracker, Docs, Memory, Code, Sessions)
    And  the URL path is /welcome (default entry, served with the shell)

  @happy @p0
  Scenario: Every menu path serves the shell for the client router
    Given a running skillgrid serve
    When  I open http://127.0.0.1:7438/tracker
    Then  I receive 200 with the dashboard shell (menu bar present)
    And  the same holds for /welcome, /docs, /memory, /code, and /sessions

  @happy @p0
  Scenario: Welcome page is static with swagger links
    Given I am on the Welcome entry
    When  I view the page
    Then  I see what the dashboard is and each entry's status (live vs coming)
    And  I see working links to /swagger-ui and /openapi.yaml
    And  the page issues zero data fetches

  @happy @p0
  Scenario: Project selector persists to localStorage
    Given I have opened the dashboard
    When  I select project "my-project" in the selector
    Then  the selection is persisted to localStorage
    And  when I reload the page, the same project is selected

  @edge
  Scenario: Future entries render as disabled stubs, never dead links
    Given only P1 is shipped
    When  I view the menu bar
    Then  Tracker, Docs, Memory, Code, Sessions show as labeled disabled stubs ("coming in P2" etc.)
    And  clicking a stub never navigates to a blank page or 404

  @failure @p1
  Scenario: Swagger-ui is reachable from the menu
    Given a running skillgrid serve
    When  I click the Swagger UI menu link
    Then  /swagger-ui loads (no 404 or JS error)

  @edge
  Scenario: Dashboard works offline (no external CDN assets)
    Given I am on the dashboard
    When  I inspect the page source
    Then  there are no external CDN links (all assets are served from the binary)

@step-02
Feature: Tracker menu entry works on the active tracker with graceful degradation
  As an operator of a skillgrid-installed machine
  I want the Tracker entry to expose the repo's active tracker (Backlog.md, GitHub, GitLab, or Jira)
  So that I can view and manage items without leaving the browser, regardless of tracker

  @happy @p0
  Scenario: Provider detection resolves the active tracker without guessing
    Given docs/skillgrid/agents/issue-tracker.md names "backlogmd" and SKILLGRID_TRACKER is unset
    When  I send GET /tracker/config
    Then  I receive 200 with provider "backlogmd"
    And  setting SKILLGRID_TRACKER to "github" switches the provider to "github"

  @failure @p1
  Scenario: Unknown provider or missing Jira project key returns 501, never a guess
    Given SKILLGRID_TRACKER is set to "not-a-tracker"
    When  I send GET /tracker/config
    Then  I receive 501 with a reason
    And  given the tracker is Jira with no project key in issue-tracker.md
    When  I send GET /tracker/tasks
    Then  I receive 501 (project key is never guessed)

  @happy @p0
  Scenario: Backlog.md GET /tracker/config parses text config list
    Given the tracker is Backlog.md (backlog CLI available)
    When  I send GET /tracker/config
    Then  I receive 200 with statuses, types, and priorities parsed from `backlog config list` text (which has no --json)

  @happy @p0
  Scenario: Backlog.md GET /tracker/tasks returns the normalized task-list DTO
    Given the tracker is Backlog.md with 3 tasks
    When  I send GET /tracker/tasks
    Then  I receive 200 with normalized items (id, title, status, type, priority, assignees, labels, AC progress, isReady, provider "backlogmd")

  @happy @p0
  Scenario: Backlog.md GET /tracker/tasks/{id} returns one item
    Given the tracker is Backlog.md and task TASK-001 exists
    When  I send GET /tracker/tasks/TASK-001
    Then  I receive 200 with the normalized item details

  @failure @p1
  Scenario: GET /tracker/tasks/{id} returns 404 when item not found
    Given the tracker is Backlog.md
    And  no item with id TASK-999 exists
    When  I send GET /tracker/tasks/TASK-999
    Then  I receive 404

  @happy @p0
  Scenario: GitHub GET /tracker/tasks returns normalized issues
    Given the tracker is GitHub (gh CLI authenticated, repo from git remote)
    When  I send GET /tracker/tasks
    Then  I receive 200 with normalized items (number as id, title, open/closed status, labels, provider "github")

  @happy @p0
  Scenario: GitLab GET /tracker/tasks returns normalized issues
    Given the tracker is GitLab (glab CLI authenticated, repo from git remote)
    When  I send GET /tracker/tasks
    Then  I receive 200 with normalized items (iid as id, title, opened/closed status, labels, provider "gitlab")

  @happy @p0
  Scenario: Jira GET /tracker/tasks returns normalized issues for the project key
    Given the tracker is Jira with project key PROJ in issue-tracker.md
    When  I send GET /tracker/tasks
    Then  I receive 200 with normalized items (issue key as id, summary as title, workflow status, provider "jira")

  @happy @p0
  Scenario: Backlog.md POST /tracker/tasks/{id}/status runs task edit (write-gated)
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is Backlog.md and task TASK-001 has status "needs-triage"
    When  I send POST /tracker/tasks/TASK-001/status with body {"status": "ready-for-agent"} and bearer token "secret"
    Then  I receive 200
    And  the task status is now "ready-for-agent" (via `backlog task edit TASK-001 -s ready-for-agent`)

  @happy @p0
  Scenario: GitHub/GitLab POST /tracker/tasks/{id}/status maps to close/reopen (write-gated)
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is GitHub and issue 12 is open
    When  I send POST /tracker/tasks/12/status with body {"status": "closed"} and bearer token "secret"
    Then  I receive 200 and the issue is closed

  @happy @p0
  Scenario: Jira POST /tracker/tasks/{id}/status runs issue move (write-gated)
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is Jira with project key PROJ and PROJ-1 has a "Done" transition
    When  I send POST /tracker/tasks/PROJ-1/status with body {"status": "Done"} and bearer token "secret"
    Then  I receive 200 (via `jira issue move PROJ-1 Done`)

  @failure @p1
  Scenario: POST /tracker/tasks/{id}/status without token returns 401
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    When  I send POST /tracker/tasks/TASK-001/status without a bearer token
    Then  I receive 401

  @failure @p1
  Scenario: POST /tracker/tasks/{id}/status with invalid status returns 400
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is Backlog.md
    When  I send POST /tracker/tasks/TASK-001/status with body {"status": "not-a-real-status"} and bearer token "secret"
    Then  I receive 400 listing the valid statuses

  @failure @p1
  Scenario: POST /tracker/tasks/{id}/status with unsupported transition returns 501
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is GitHub
    When  I send POST /tracker/tasks/12/status with body {"status": "needs-triage"} and bearer token "secret"
    Then  I receive 501 listing what GitHub supports (close/reopen)

  @happy @p0
  Scenario: Tracker entry renders a fixed 4-column board
    Given I am on the Tracker entry
    And  the active tracker has 5 items in 3 different provider statuses
    When  I view the Tracker entry
    Then  I see exactly 4 columns (To Do, In Progress, Blocked, Done) fed by each item's canonical board column
    And  I see rich cards (priority badge, label chips, assignee initials, doc-ref count, relative time)

  @happy @p0
  Scenario: Tracker entry switches providers without reload
    Given I am on the Tracker entry with provider backlogmd
    When  I click the GitHub tab in the provider switcher
    Then  the board reloads from GET /tracker/tasks?provider=github
    And  an unconnected provider shows a "not connected" empty state, not a spinner forever

  @happy @p0
  Scenario: Tracker entry filters by search text and priority
    Given I am on the Tracker entry with 5 items
    When  I type "login" in the filter box
    Then  only matching items (title, id, or label) remain
    And  selecting priority "high" narrows the board further

  @happy @p0
  Scenario: Tracker entry click opens a slide-over detail panel
    Given I am on the Tracker entry viewing an item card
    When  I click the item card
    Then  I see a right-side panel (status, priority, assignee, labels, description, linked docs, created/updated footer)
    And  pressing Escape or the backdrop closes it
    And  the URL carries ?provider=&task= for deep-linking

  @happy @p0
  Scenario: Tracker entry click opens detail pane and status change works
    Given I am viewing an item in the slide-over panel
    When  I select the provider-native "Move to…" action and apply
    Then  the status changes (round-trip via the active CLI) and the board re-renders

  @happy @p0
  Scenario: Tracker entry shows provider banner
    Given I am on the Tracker entry
    And  the tracker is Backlog.md (CLI version 1.2.3, schemaVersion 1)
    When  I view the Tracker entry
    Then  I see a banner with provider "backlogmd", "CLI v1.2.3 (schema v1)"

  @edge
  Scenario: Tracker entry warns on unknown Backlog.md schemaVersion
    Given I am on the Tracker entry
    And  the tracker is Backlog.md responding with schemaVersion 99
    When  I view the Tracker entry
    Then  I see a warning banner (not a hard error)

  @edge
  Scenario: Tracker entry degraded state when CLI is missing or tracker unconfigured
    Given I am on the Tracker entry
    And  the active tracker CLI is not on PATH (503) or the provider is unresolvable (501)
    When  I view the Tracker entry
    Then  I see the provider name plus the reason ("CLI not found" / "tracker not configured")
    And  interactions are disabled

  @edge
  Scenario: /backlog/* alias mirrors /tracker/* on Backlog.md repos
    Given the tracker is Backlog.md
    When  I send GET /backlog/tasks
    Then  I receive the same 200 normalized DTO as GET /tracker/tasks

  @failure @p1
  Scenario: GET /tracker/tasks returns 502 with stderr excerpt when CLI exits non-zero or auth fails
    Given the active tracker CLI exits 1 with stderr "boom: something broke" (or auth failure)
    When  I send GET /tracker/tasks
    Then  I receive 502 with "boom: something broke" in the JSON error (truncated to 200 chars) and the provider name

  @failure @p1
  Scenario: GET /tracker/tasks returns 502 when CLI times out
    Given the active tracker CLI takes 11 seconds to respond
    When  I send GET /tracker/tasks
    Then  I receive 502 (timeout) with the provider name

  @failure @p1
  Scenario: GET /tracker/tasks returns 502 when CLI output is unparsable
    Given the active tracker CLI prints "not json" (or unparsable Jira output)
    When  I send GET /tracker/tasks
    Then  I receive 502 with a reason

  @edge
  Scenario: Backlog.md GET /tracker/tasks returns 502 when schemaVersion is unknown
    Given the tracker is Backlog.md and the CLI prints JSON with schemaVersion 99
    When  I send GET /tracker/tasks
    Then  I receive 502 with a reason

  @failure @p1
  Scenario: Per-widget error isolation — a 500 on /tracker/tasks leaves other entries interactive
    Given I am on the Tracker entry
    And  GET /tracker/tasks returns 500
    When  I view the Tracker entry
    Then  I see an error state in the Tracker entry
    And  I switch to the Welcome entry
    Then  the Welcome entry still renders

@step-03
Feature: Docs menu entry renders SDD docs with two-way tracker links
  As an operator of a skillgrid-installed machine
  I want the Docs entry to show change.md and tasks.md for any change, linked to its tracker item
  So that I can read the plan behind a task without leaving the browser

  @happy @p0
  Scenario: GET /docs/changes returns the change list with ticket ids
    Given the repo has changes 009-web-admin-dashboard (Ticket: TASK-001) and 004-hermes-memory (Ticket: none)
    When  I send GET /docs/changes
    Then  I receive 200 with both changes (name, status, ticket id or none)

  @happy @p0
  Scenario: GET /docs/changes/{name} returns change.md + tasks.md text
    Given change 009-web-admin-dashboard exists
    When  I send GET /docs/changes/009-web-admin-dashboard
    Then  I receive 200 with the change.md text, the tasks.md text, and ticket id TASK-001

  @happy @p0
  Scenario: Docs entry shows change list and file view
    Given I am on the Docs entry
    When  I view the entry
    Then  I see the change list
    And  I click 009-web-admin-dashboard
    Then  I see its change.md and tasks.md rendered as text

  @happy @p0
  Scenario: Docs entry links to the tracker item via Ticket:
    Given I am viewing 009-web-admin-dashboard in the Docs entry (Ticket: TASK-001)
    When  I click "Open tracker item"
    Then  I land on the Tracker entry with TASK-001 open

  @happy @p0
  Scenario: Tracker detail links back to SDD docs
    Given I am viewing TASK-001 in the Tracker entry and it references docs/skillgrid/changes/009-web-admin-dashboard/change.md
    When  I click "View SDD docs"
    Then  I land on the Docs entry with 009-web-admin-dashboard open

  @edge
  Scenario: Docs entry with no ticket shows no tracker link, not an error
    Given I am viewing 004-hermes-memory in the Docs entry (Ticket: none)
    When  I view the links area
    Then  I see no tracker link and no error

  @failure @p1
  Scenario: Path traversal on GET /docs/changes/{name} returns 400
    Given I send GET /docs/changes/../secret
    When  the request reaches the server
    Then  I receive 400 with a JSON error
    And  sending GET /docs/changes//etc/passwd also returns 400

  @failure @p1
  Scenario: Unknown change name returns 404
    Given no change named "nope" exists
    When  I send GET /docs/changes/nope
    Then  I receive 404

@step-04
Feature: Memory menu entry browses memory and governs it (013 data with forward-compat placeholders)
  As an operator of a skillgrid-installed machine
  I want the Memory entry to browse, correct, share, and review memory
  So that I can govern memory — not just browse it — and P4 stays usable even before 013 lands

  @happy @p0
  Scenario: GET /observations/{id} returns full untruncated content
    Given a saved observation with id 42 and content "the full body"
    When  I send GET /observations/42
    Then  I receive 200 with the full content "the full body" (same shape as mem_get_observation)

  @edge
  Scenario: GET /observations/{id} returns 404 for unknown id
    Given no observation exists with id 999
    When  I send GET /observations/999
    Then  I receive 404 with a JSON error

  @failure @p1
  Scenario: Pin without token returns 401 when SKILLGRID_HTTP_TOKEN is set
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  an observation with id 42
    When  I send POST /memory/observations/42/pin without a bearer token
    Then  I receive 401

  @failure @p1
  Scenario: Pin with token returns 200 and is idempotent
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  an observation with id 42
    When  I send POST /memory/observations/42/pin with bearer token "secret"
    Then  I receive 200
    And  I send POST /memory/observations/42/pin again with the same token
    Then  I receive 200 (idempotent)

  @happy @p0
  Scenario: GET /observations/{id} is open (no token required)
    Given an observation with id 42
    When  I send GET /observations/42 without a bearer token
    Then  I receive 200 (read routes are open)

  @happy @p0
  Scenario: Memory entry search shows results and click opens detail pane
    Given I am on the Memory entry
    And  there is an observation matching "auth"
    When  I type "auth" in the search box (debounced)
    Then  I see a results list with the matching observation
    And  I click the observation
    Then  I see a detail pane with the full content

  @happy @p0
  Scenario: Memory entry pin/unpin and soft-delete actions work
    Given I am viewing an observation in the detail pane
    When  I click "Pin"
    Then  the observation is pinned (confirmed by re-fetching)
    And  I click "Unpin"
    Then  the observation is unpinned
    And  I click "Delete" and confirm
    Then  the observation is soft-deleted and removed from the list

  @happy @p0
  Scenario: Memory entry shows relation drill-down with confidence badges
    Given I am viewing an observation that has 2 relations (1 EXTRACTED, 1 INFERRED)
    When  I view the detail pane
    Then  I see a relations list with confidence badges (EXTRACTED, INFERRED)
    And  I click the related observation
    Then  I navigate to its detail pane

  @edge
  Scenario: Memory entry empty state shows suggested prompts
    Given I am on the Memory entry with no search results
    And  there are recent session prompts in the context
    When  I view the empty state
    Then  I see recent session prompts as clickable suggestions (not a blank "no results")

  @happy @p0
  Scenario: Every data widget has a show-numbers table twin
    Given I am viewing any data widget (status card, search results, relation list)
    When  I toggle "show numbers"
    Then  I see the raw JSON/table behind the visual

  @edge
  Scenario: Web-cache view is preserved in Memory entry
    Given I am on the Memory entry
    When  I navigate to the web-cache sub-section
    Then  I see the web-cache search and results (equivalent to the old viewer)

  @failure @p1
  Scenario: A failed observation detail endpoint shows an error in the detail pane, not a blank page
    Given I am on the Memory entry
    And  GET /observations/{id} returns 500
    When  I click an observation in the results list
    Then  I see an error message in the detail pane (not a blank pane or console-only failure)
    And  the rest of the dashboard (search, Tracker entry) is still interactive

  # --- 013-present (happy path) ---

  @happy @p0
  Scenario: Asset library shows owner, version, status, usage, and visibility
    Given I am on the Memory entry against a 013-provisioned store
    And  there is an observation with owner "op-1", 3 versions, status "active", retrieval usage 12, visibility "team"
    When  I view the Memory list
    Then  each row shows owner, version count, status, retrieval usage, and a visibility badge
    And  toggling "show numbers" on the asset list shows the raw JSON/table

  @happy @p0
  Scenario: Layer drill-down renders the L0 to L3 chain lazy-loaded per layer
    Given I am viewing an observation that has a distilled L1 atom
    When  I expand the layer drill-down
    Then  I see the L0 -> L1 -> L2 -> L3 chain, loaded one layer at a time on expand
    And  each layer shows its provenance link
    And  the distilled L1 atom links back to its L0 source

  @happy @p0
  Scenario: Explicit share widens visibility by an explicit click
    Given I am viewing an observation with visibility "private"
    When  I select visibility "team" and confirm the share
    Then  the observation is now shared to "team" (an explicit action, never a default)
    And  a confirmation dialog is shown before the visibility change

  @happy @p0
  Scenario: In-place edit corrects an atom and appends a version
    Given I am viewing an L1 atom
    When  I edit it in place and save
    Then  the new content is current
    And  the prior content is recoverable in the version history view

  @happy @p0
  Scenario: Review status is visible and changeable
    Given I am viewing an observation with status "active"
    When  I change the status to "superseded"
    Then  the status is now "superseded" (the personal -> shared gate is a UI action)

  @happy @p0
  Scenario: Read-only agent loadout shows equipped assets
    Given I am on the Memory entry
    And  agent "alpha" is equipped with 2 assets (visibility "agent")
    When  I open the agent loadout panel for "alpha"
    Then  I see the 2 assets equipped to "alpha" (read-only)

  # --- forward-compat (013 absent — soft dependency) ---

  @edge
  Scenario: With 013 absent every governance and layer view renders a forward-compat placeholder
    Given I am on the Memory entry against a pre-013 store (no owner/version/status/usage/visibility or layer data)
    When  I view the asset library, layer drill-down, share, in-place edit, review/status, and loadout
    Then  each renders a labeled collapsed placeholder plus the flat pre-013 view
    And  the Memory entry's search, detail, and actions stay fully interactive

  # --- threat: Governance mutation / soft-dep (write-gated + per-widget isolation) ---

  @failure @p1
  Scenario: Governance mutations (edit, share, status) are write-gated
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  I am viewing an observation in the Memory entry
    When  I edit it in place, share it, or change its status without a bearer token
    Then  each governance mutation returns 401
    And  with the bearer token "secret", the same mutations return 200

  @failure @p1
  Scenario: In-place edit appends a 013 version rather than overwriting
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  I am viewing an L1 atom with prior content "v1"
    When  I edit it in place to "v2" with the bearer token
    Then  the current content is "v2"
    And  the prior content "v1" is still re-readable in the version history view

  @failure @p1
  Scenario: Share is idempotent and returns 400 on an unknown target
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  I am viewing an observation
    When  I share it to "team" with the token, then share to "team" again
    Then  both return 200 and the visibility stays "team" (idempotent)
    And  when I share it to an unknown target
    Then  I receive 400 and the visibility is unchanged

  @failure @p1
  Scenario: A failed or absent 013 field kills that widget only (per-widget isolation)
    Given I am on the Memory entry against a 013-provisioned store
    And  the layer-drill-down field for one observation is missing or returns 500
    When  I view that observation
    Then  the layer drill-down widget shows an error or placeholder
    And  the asset library, share, in-place edit, review/status, and loadout widgets remain fully interactive

@step-05
Feature: Code menu entry shows index health and searches code on existing routes
  As an operator of a skillgrid-installed machine
  I want the Code entry to show freshness, status, search, and source view
  So that I can check code-index health without leaving the browser (no new backend)

  @happy @p0
  Scenario: Code entry shows freshness banner with last-indexed and stale flag
    Given I am on the Code entry
    And  the code index was last indexed at 2026-09-08T08:00:00Z
    When  I view the Code entry
    Then  I see a top banner with "Last indexed: 2026-09-08T08:00:00Z" and the stale flag
    And  I see a "Re-index" action in the banner

  @happy @p0
  Scenario: Code entry search shows results and click opens source view
    Given I am on the Code entry
    And  the code index has a file matching "handler"
    When  I type "handler" in the search box
    Then  I see a results list with the matching chunk
    And  I click the result
    Then  I see a source view with the code

  @happy @p0
  Scenario: Code entry forward-compat graph placeholder is collapsed
    Given I am on the Code entry
    When  I view the Code entry
    Then  I see a collapsed panel labeled "Code graph (coming in 010)"
    And  expanding it shows a file-list fallback

  @edge
  Scenario: Code entry Re-index without token returns 401 when token is set
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    When  I click "Re-index" without a bearer token
    Then  the re-index call returns 401 (existing write-gate, unchanged)

  @failure @p1
  Scenario: A failed code search shows an error in the entry, not a blank page
    Given I am on the Code entry
    And  GET /code/search returns 500
    When  I type "handler" in the search box
    Then  I see an error message in the results area (not a blank page)
    And  the rest of the dashboard (Memory, Tracker entries) is still interactive

@step-06
Feature: Sessions menu entry lists sessions and summaries; docs and polish match the shipped surface
  As an operator of a skillgrid-installed machine
  I want the Sessions entry to list sessions with summaries
  So that I can review sessions without leaving the browser, with documented APIs

  @happy @p0
  Scenario: GET /sessions returns the session list
    Given two started sessions with titles "session A" and "session B"
    When  I send GET /sessions
    Then  I receive 200 with both sessions (id, title, started_at, status)

  @happy @p0
  Scenario: GET /sessions/{id}/summary returns the session summary
    Given a session that was ended with a summary
    When  I send GET /sessions/{id}/summary
    Then  I receive 200 with the session summary

  @edge
  Scenario: GET /sessions/{id}/summary returns 404 for unknown session
    Given no session exists with id "nope"
    When  I send GET /sessions/nope/summary
    Then  I receive 404

  @happy @p0
  Scenario: Sessions entry shows session list, context, and summary
    Given I am on the Sessions entry
    And  there are 3 sessions with summaries
    When  I view the Sessions entry
    Then  I see a session list (title, started_at, status)
    And  I see a recent context section
    And  I click a session
    Then  I see a summary pane

  @edge
  Scenario: Sessions entry 404 on unknown session shows in-pane error
    Given I am on the Sessions entry
    And  GET /sessions/{id}/summary returns 404
    When  I click a session with an unknown id
    Then  I see an error message in the summary pane (not a blank pane)

  @happy @p0
  Scenario: openapi.yaml documents all new routes with examples
    Given a running skillgrid serve
    When  I fetch GET /openapi.yaml
    Then  I see the P2 tracker routes (GET /tracker/config, GET /tracker/tasks, GET /tracker/tasks/{id}, POST /tracker/tasks/{id}/status, plus the /backlog/* alias) with request/response examples
    And  I see the P3 docs routes (GET /docs/changes, GET /docs/changes/{name}) with request/response examples
    And  I see the P4 memory routes (GET /observations/{id}, POST .../pin, POST .../unpin) with request/response examples
    And  I see the P6 session routes (GET /sessions, GET /sessions/{id}/summary) with request/response examples

  @happy @p0
  Scenario: swagger-ui loads and can exercise each new route
    Given a running skillgrid serve
    When  I open /swagger-ui
    Then  I see all new routes listed
    And  I can try GET /observations/{id} with a valid id
    Then  I receive a 200 response in the swagger-ui try panel

  @edge
  Scenario: User-manual documents the menu entries and tracker-CLI dependency
    Given I read the user-manual serve section
    When  I look for the dashboard documentation
    Then  I see all 6 entries described (Welcome, Tracker, Docs, Memory, Code, Sessions)
    And  I see the per-provider tracker-CLI dependency documented (install/auth + what happens when the CLI is missing)

  @happy @p0
  Scenario: Full DoD smoke passes
    Given a running skillgrid serve with the dashboard deployed
    When  I run go test ./...
    Then  all tests pass
    And  I run go vet ./...
    Then  no issues reported
    And  I open the dashboard in a browser
    Then  all 6 menu entries render (shipped entries live, none left as stubs)

  @failure @p1
  Scenario: swagger-ui still serves when new routes are added
    Given a running skillgrid serve
    When  I open /swagger-ui
    Then  it loads (no 404 or JS error)
    And  the old routes are still documented

# Rules:
# - ≥1 @happy + @edge + @failure per step
# - Every change.md per-step WHAT bullet → a Scenario
# - Every applicable threat-matrix row → a Scenario in its owning @step-NN
# - Scenario names unique and referenceable from tasks.md `Run:` / Expected lines
# - @p0 must cover change.md Definition of Done user-visible criteria
