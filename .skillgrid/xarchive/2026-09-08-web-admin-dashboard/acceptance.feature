# Source: .skillgrid/specs/2026-09-08-web-admin-dashboard/briefing.md
# Trace: briefing.md ## Goal + ## Definition of Done + Per-phase WHAT; tasks.md @phase-NN verify lines.
# Mapping: @p0 scenarios ↔ briefing.md DoD / Testing strategy; @p1 = important failure paths.
# One Feature per phase; tag each Feature with @phase-NN matching tasks.md.
# Execution runs 1 → 7 in order; each phase is independently shippable
# (shipped views work, future views show as disabled stubs).
# Soft dependencies: 005/008 (graph edge set) and 013 (memory governance) —
# the graph degrades to node-only + file-list and the governance views render
# forward-compat placeholders when the data is absent; both stay fully interactive.

@phase-1
Feature: Vite/React SPA embedded via go:embed serves the shell with every view as a stub
  As an operator of a skillgrid-installed machine
  I want the dashboard at / to be a real React SPA compiled into the Go binary
  So that every later phase has an embedded foundation to extend one view at a time

  @happy @p0
  Scenario: GET / serves the embedded SPA shell
    Given a running skillgrid serve with the SPA built into ui/dist/
    When  I open http://127.0.0.1:7438/
    Then  I receive 200 with the SPA index.html
    And  I see the sidebar nav (Tracker, Mnemonic, Docs, Plans, Activity, Git, Prototypes, Settings)
    And  the dark theme is applied (background #0a0a0b)

  @happy @p0
  Scenario: Hashed JS and CSS assets are served from the binary
    Given the SPA build produced assets/index-[hash].js and assets/index-[hash].css
    When  I request GET /assets/index-[hash].js
    Then  I receive 200 with the JS asset content (not index.html)
    And  the same holds for the CSS asset

  @happy @p0
  Scenario: SPA fallback serves index.html for non-API, non-asset routes
    Given a running skillgrid serve
    When  I open http://127.0.0.1:7438/tracker
    Then  I receive 200 with the SPA index.html (client router takes over)
    And  the same holds for /docs, /plans, /activity, /git, and /prototypes

  @failure @p1
  Scenario: API-prefix routes are not shadowed by the SPA fallback
    Given a running skillgrid serve
    When  I request GET /mnemonic/nope (an API prefix with no handler yet)
    Then  I receive 404 with a JSON error
    And  I do NOT receive the SPA index.html

  @happy @p0
  Scenario: swagger and openapi are preserved
    Given a running skillgrid serve
    When  I request GET /openapi.yaml
    Then  I receive 200 with content-type application/yaml
    And  when I open /swagger/
    Then  the Swagger UI loads (no 404 or JS error)

  @edge
  Scenario: Future views render as labeled disabled stubs, never dead links
    Given only Phase 1 is shipped
    When  I view the sidebar nav
    Then  Tracker, Docs, Plans, Activity, Git, and Prototypes show as labeled disabled stubs
    And  clicking a stub never navigates to a blank page or 404

  @edge
  Scenario: Dashboard works offline (no external CDN assets)
    Given I am on the dashboard
    When  I inspect the page source and loaded resources
    Then  there are no external CDN links (all assets are served from the binary)

  @happy @p0
  Scenario: Project selector persists to localStorage
    Given I have opened the dashboard
    When  I select project "my-project" in the selector
    Then  the selection is persisted to localStorage
    And  when I reload the page, the same project is selected

@phase-2
Feature: Multi-provider Kanban over the Go TicketProvider registry
  As an operator of a skillgrid-installed machine
  I want the Tracker to expose the repo's active tracker (Backlog.md, GitHub, GitLab, or Jira) as one board
  So that I can view and manage items without leaving the browser, regardless of tracker

  @happy @p0
  Scenario: Provider detection resolves the active tracker without guessing
    Given the issue-tracker doc names "backlogmd" and SKILLGRID_TRACKER is unset
    When  I send GET /tracker/providers
    Then  I receive 200 with provider "backlogmd"
    And  setting SKILLGRID_TRACKER to "github" switches the active provider to "github"

  @failure @p1
  Scenario: Unknown provider or missing Jira project key returns 501, never a guess
    Given SKILLGRID_TRACKER is set to "not-a-tracker"
    When  I send GET /tracker/providers
    Then  I receive 501 with a reason
    And  given the tracker is Jira with no project key in the issue-tracker doc
    When  I send GET /tracker/tasks?provider=jira
    Then  I receive 501 (project key is never guessed)

  @happy @p0
  Scenario: Backlog.md GET /tracker/tasks parses .backlog/tasks/*.md frontmatter
    Given the tracker is Backlog.md with 3 tasks in .backlog/tasks/
    When  I send GET /tracker/tasks?provider=backlogmd
    Then  I receive 200 with normalized UnifiedTask items (id, title, status, priority, assignee, labels, dependencies, milestone, board column, provider "backlogmd")
    And  no backlog CLI invocation was required (file-based parse)

  @happy @p0
  Scenario: Backlog.md GET /tracker/tasks/{id} returns one item; unknown id returns 404
    Given the tracker is Backlog.md and task TASK-001 exists
    When  I send GET /tracker/tasks/TASK-001
    Then  I receive 200 with the normalized item details
    And  sending GET /tracker/tasks/TASK-999 returns 404

  @happy @p0
  Scenario: Backlog.md PATCH /tracker/tasks/{id} runs task edit (write-gated)
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is Backlog.md and task TASK-001 has status "needs-triage"
    When  I send PATCH /tracker/tasks/TASK-001 with body {"status": "ready-for-agent"} and bearer token "secret"
    Then  I receive 200
    And  the task file on disk now has status "ready-for-agent"

  @failure @p1
  Scenario: PATCH /tracker/tasks/{id} without token returns 401
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    When  I send PATCH /tracker/tasks/TASK-001 without a bearer token
    Then  I receive 401

  @failure @p1
  Scenario: PATCH /tracker/tasks/{id} with invalid status returns 400
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is Backlog.md
    When  I send PATCH /tracker/tasks/TASK-001 with body {"status": "not-a-real-status"} and bearer token "secret"
    Then  I receive 400 listing the valid statuses

  @happy @p0
  Scenario: GitHub/GitLab GET /tracker/tasks returns normalized issues
    Given the tracker is GitHub (gh CLI authenticated, repo from git remote)
    When  I send GET /tracker/tasks?provider=github
    Then  I receive 200 with normalized items (number as id, title, open/closed status, labels, provider "github")
    And  the same normalized shape holds for GitLab (glab CLI, iid as id, provider "gitlab")

  @happy @p0
  Scenario: Jira GET /tracker/tasks returns normalized issues for the project key
    Given the tracker is Jira with project key PROJ in the issue-tracker doc
    When  I send GET /tracker/tasks?provider=jira
    Then  I receive 200 with normalized items (issue key as id, summary as title, workflow status, provider "jira")

  @happy @p0
  Scenario: GitHub/GitLab PATCH maps to close/reopen; Jira PATCH runs issue move (write-gated)
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is GitHub and issue 12 is open
    When  I send PATCH /tracker/tasks/12 with body {"status": "done"} and bearer token "secret"
    Then  I receive 200 and the issue is closed
    And  for Jira with a "Done" transition, PATCH runs issue move and returns 200

  @failure @p1
  Scenario: PATCH with unsupported transition returns 501
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  the tracker is GitHub
    When  I send PATCH /tracker/tasks/12 with body {"status": "needs-triage"} and bearer token "secret"
    Then  I receive 501 listing what GitHub supports (close/reopen)

  @failure @p1
  Scenario: CLI-backed provider missing returns 503 with provider name
    Given the active provider is GitHub and the gh CLI is not on PATH
    When  I send GET /tracker/tasks?provider=github
    Then  I receive 503 with {"error": "gh CLI not found", "provider": "github"}
    And  the Backlog.md (file-based) provider still returns 200

  @failure @p1
  Scenario: CLI-backed provider non-zero exit / auth failure / timeout returns 502
    Given the active tracker CLI exits 1 with stderr "boom: something broke" (or auth failure)
    When  I send GET /tracker/tasks?provider=github
    Then  I receive 502 with "boom: something broke" in the JSON error (truncated to 200 chars) and the provider name
    And  given the CLI takes 11 seconds to respond, I receive 502 (timeout)

  @failure @p1
  Scenario: CLI-backed provider unparsable output returns 502
    Given the active tracker CLI prints "not json" (or unparsable Jira output)
    When  I send GET /tracker/tasks?provider=github
    Then  I receive 502 with a reason

  @happy @p0
  Scenario: Kanban view renders a board with provider tabs and drag-and-drop
    Given I am on the Tracker view
    And  the active tracker has 5 items in 3 different provider statuses
    When  I view the Tracker view
    Then  I see a provider tab per connected provider and a board grouped by the canonical columns
    And  I can drag a card to another column and the status changes (round-trip via the provider)

  @happy @p0
  Scenario: Kanban view switches providers and filters
    Given I am on the Tracker view with provider backlogmd
    When  I click the GitHub tab
    Then  the board reloads from GET /tracker/tasks?provider=github
    And  an unconnected provider shows a "not connected" empty state, not a spinner forever
    And  typing "login" in the filter box narrows to matching items (title, id, or label)

  @happy @p0
  Scenario: Kanban view click opens a detail drawer with dependency mini-graph
    Given I am on the Tracker view viewing an item card
    When  I click the item card
    Then  I see a drawer with the markdown description, metadata grid, and a dependency mini-graph
    And  the URL carries ?provider=&task= for deep-linking

  @happy @p0
  Scenario: Kanban view updates live from /tracker/stream
    Given I am on the Tracker view (Backlog.md)
    And  a .backlog/tasks/*.md file is modified on disk
    When  the fsnotify event fires
    Then  the board reflects the change without a manual reload (SSE push)

  @edge
  Scenario: Kanban view degraded state when a provider is missing or unconfigured
    Given I am on the Tracker view
    And  the active CLI-backed provider is not on PATH (503) or unresolvable (501)
    When  I view that provider's tab
    Then  I see the provider name plus the reason ("CLI not found" / "tracker not configured")
    And  interactions are disabled for that provider only

  @failure @p1
  Scenario: Per-widget error isolation — a 500 on /tracker/tasks leaves other views interactive
    Given I am on the Tracker view
    And  GET /tracker/tasks returns 500
    When  I view the Tracker view
    Then  I see an error state in the Tracker view
    And  I navigate to the Docs view
    Then  the Docs view still renders

@phase-3
Feature: Docs view renders repo markdown with Mermaid diagrams
  As an operator of a skillgrid-installed machine
  I want the Docs view to show rendered markdown (with Mermaid) for any doc in the repo
  So that I can read .skillgrid/sdd, openspec, .backlog, and docs content without leaving the browser

  @happy @p0
  Scenario: GET /docs/tree returns the grouped doc tree
    Given the repo has docs under .skillgrid/sdd/, openspec/, .backlog/tasks/, and docs/
    When  I send GET /docs/tree?root=all
    Then  I receive 200 with the tree grouped by root (path, title, updatedAt, frontmatter status)

  @happy @p0
  Scenario: GET /docs/content returns markdown + frontmatter + relatedPlans
    Given a doc at .skillgrid/sdd/014-mnemonic-performance/briefing.md exists
    When  I send GET /docs/content?path=.skillgrid/sdd/014-mnemonic-performance/briefing.md
    Then  I receive 200 with the markdown body, the frontmatter, and relatedPlans

  @failure @p1
  Scenario: Path traversal on GET /docs/content returns 400
    Given I send GET /docs/content?path=../secret
    When  the request reaches the server
    Then  I receive 400 with a JSON error
    And  sending GET /docs/content?path=/etc/passwd also returns 400
    And  sending GET /docs/content?path=docs/../x also returns 400

  @failure @p1
  Scenario: Unknown doc path returns 404
    Given no doc at the requested path exists
    When  I send GET /docs/content?path=nope/missing.md
    Then  I receive 404

  @happy @p0
  Scenario: Docs view shows the tree and renders markdown with GFM + anchors
    Given I am on the Docs view
    When  I expand the .skillgrid/sdd/ root and click 014-mnemonic-performance
    Then  I see the rendered markdown with headings (anchor links), GFM tables, task lists, and syntax-highlighted code
    And  the frontmatter chips (status, author, updated) are shown
    And  the on-this-page TOC is sticky with scroll-spy

  @happy @p0
  Scenario: Docs view renders Mermaid blocks to sanitized SVG
    Given a doc containing a ```mermaid graph TD block
    When  I open the doc in the Docs view
    Then  I see the Mermaid diagram rendered as an SVG
    And  the SVG is DOMPurify-sanitized (no raw script elements)

  @failure @p1
  Scenario: Untrusted markdown with an inline script is sanitized
    Given a doc whose markdown body contains a <script>alert(1)</script> element
    When  I open the doc in the Docs view
    Then  the rendered output contains no raw <script> element (rehype-sanitize applied)

  @happy @p0
  Scenario: Docs view search returns matches with snippets
    Given the repo has docs mentioning "mnemonic"
    When  I send GET /docs/search?q=mnemonic
    Then  I receive 200 with matches (path + snippet)
    And  clicking a match opens that doc in the Docs view

  @happy @p0
  Scenario: Docs view cross-links to SPA routes and to the Plans view
    Given I am viewing a doc in the Docs view
    When  I click a cross-link to another markdown file
    Then  I land on that doc (SPA route, no full reload)
    And  I see a "View plan progress →" link that targets the Plans view

  @edge
  Scenario: GET /docs/render returns rendered HTML (SSR fallback)
    Given a doc at a known path
    When  I send GET /docs/render?path=<known path>
    Then  I receive 200 with rendered HTML

  @edge
  Scenario: Docs view copy/download/print actions work
    Given I am viewing a rendered doc
    When  I click "Copy", "Download", or "Print"
    Then  the markdown is copied / downloaded as .md / printed respectively

@phase-4
Feature: Mnemonic vector graph renders the memory graph with GitNexus parity
  As an operator of a skillgrid-installed machine
  I want the Mnemonic graph to visualize the memory graph as an interactive vector graph
  So that I can explore how memories, files, functions, and concepts relate

  @happy @p0
  Scenario: GET /mnemonic/graph returns nodes and edges
    Given the code index and memory have nodes and edges
    When  I send GET /mnemonic/graph?node_id=<root>&depth=2
    Then  I receive 200 with nodes (id, label, type, path, degree, community) and edges (source, target, type, weight)

  @happy @p0
  Scenario: GET /mnemonic/graph/nodes caps the node count
    Given the graph has more than 500 nodes
    When  I send GET /mnemonic/graph/nodes?limit=500
    Then  I receive 200 with at most 500 nodes

  @failure @p1
  Scenario: Depth filter caps the neighborhood and sets truncated
    Given a graph that is deeper than 2 hops from the root
    When  I send GET /mnemonic/graph?node_id=<root>&depth=2
    Then  I receive 200 with only the 2-hop neighborhood
    And  the response carries truncated: true
    And  the UI shows a "depth capped" hint

  @failure @p1
  Scenario: Edges whose endpoints are absent are dropped by the converter
    Given a graph payload where an edge references a node not in the node set
    When  the graph is converted to graphology (mnemonicGraphToGraphology)
    Then  the edge with the absent endpoint is not present in the resulting graph

  @happy @p0
  Scenario: VectorGraph renders force/tree/circles layouts with community coloring
    Given I am on the Mnemonic > Graph view
    When  I switch the layout between force, tree, and circles
    Then  the graph re-renders with the selected layout
    And  nodes are colored by Louvain community
    And  node size reflects degree centrality

  @happy @p0
  Scenario: Depth slider and semantic search highlight nodes
    Given I am on the Graph view
    When  I drag the depth slider
    Then  the visible neighborhood updates
    And  when I run a semantic search that matches nodes
    Then  the matched nodes are highlighted

  @happy @p0
  Scenario: Graph controls show legend, zoom, pan, and hover tooltips
    Given I am on the Graph view
    When  I hover a node
    Then  I see a tooltip with node type, path, degree, and community
    And  I can zoom, pan, and toggle the legend

  @happy @p0
  Scenario: Graph exports a lightweight standalone HTML
    Given I am on the Graph view
    When  I click "Export HTML"
    Then  I receive a standalone HTML file that loads Sigma.js from CDN and renders the graph

  @edge
  Scenario: Graph degrades to node-only + file-list fallback when 005/008 edge data is absent
    Given the store has nodes but no edge/community data (005/008 not landed)
    When  I send GET /mnemonic/graph?node_id=<root>&depth=2
    Then  I receive 200 with nodes, empty edges, and degraded: true
    And  the UI renders the node-only view with a file-list fallback and a labeled placeholder

  @failure @p1
  Scenario: A failed graph endpoint shows an error in the view, not a blank page
    Given I am on the Graph view
    And  GET /mnemonic/graph returns 500
    When  I view the Graph view
    Then  I see an error message in the view (not a blank canvas)
    And  the rest of the dashboard is still interactive

@phase-5
Feature: Mnemonic files, memories, sessions, and search (with 013 governance)
  As an operator of a skillgrid-installed machine
  I want the Mnemonic suite to browse the file tree, memories, sessions, and search semantically
  So that I can inspect and govern memory without leaving the browser

  @happy @p0
  Scenario: GET /mnemonic/files/tree returns the memfs tree
    Given the memfs has nested nodes
    When  I send GET /mnemonic/files/tree?path=/
    Then  I receive 200 with the tree (node icon, name, memory-count badge, last-indexed)

  @happy @p0
  Scenario: GET /mnemonic/files/content returns L0/L1/L2 tiers
    Given a file at mnemonic://...
    When  I send GET /mnemonic/files/content?uri=mnemonic://...
    Then  I receive 200 with the L0 (abstract), L1 (overview), and L2 (details) tiers
    And  an unknown uri returns 404

  @happy @p0
  Scenario: GET /mnemonic/memories is paginated and GET /mnemonic/memories/{id} returns full content
    Given the store has 60 memories
    When  I send GET /mnemonic/memories?limit=50&offset=0
    Then  I receive 200 with 50 memories + a total
    And  sending GET /mnemonic/memories/<id> returns the full content + 013 governance fields (when present)
    And  an unknown id returns 404

  @happy @p0
  Scenario: GET /mnemonic/sessions returns the session list
    Given there are 3 sessions
    When  I send GET /mnemonic/sessions
    Then  I receive 200 with the session list (id, title, started_at, count, extracted entities)

  @happy @p0
  Scenario: GET /mnemonic/audit returns the hash-chained log
    Given the audit log has entries
    When  I send GET /mnemonic/audit
    Then  I receive 200 with the hash-chained verification log

  @happy @p0
  Scenario: GET /mnemonic/search returns ranked hybrid results
    Given the store has memories matching "auth"
    When  I send GET /mnemonic/search?q=auth&mode=hybrid
    Then  I receive 200 with ranked results + relevance scores
    And  an empty query returns 200 with an empty list

  @happy @p0
  Scenario: OpenViking file tree renders with L0/L1/L2 content and breadcrumbs
    Given I am on the Mnemonic > Files view
    When  I expand a node in the tree
    Then  I see the content panel with L0/L1/L2 tiers
    And  the mnemonic:// breadcrumbs reflect the current path
    And  scoped search finds memories within the directory

  @happy @p0
  Scenario: Memory content view shows grid, timeline, and detail modal
    Given I am on the Mnemonic > Memories view
    When  I view the memory card grid
    Then  each card shows title, source, timestamp, tags, relevance, and a preview
    And  the timeline groups memories by day/session
    And  clicking a card opens a detail modal with full content, source context, vector-similar memories, and edit/delete

  @happy @p0
  Scenario: Session browser groups by session with summary and entities
    Given I am on the Mnemonic > Sessions view
    When  I view the session browser
    Then  I see sessions grouped by session id with summary, count, timeline, and extracted entities

  @failure @p1
  Scenario: Governance mutations (edit, share, status) are write-gated
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  I am viewing a memory in the detail modal
    When  I edit it in place, share it, or change its status without a bearer token
    Then  each governance mutation returns 401
    And  with the bearer token "secret", the same mutations return 200

  @failure @p1
  Scenario: In-place edit appends a 013 version rather than overwriting
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  I am viewing an atom with prior content "v1"
    When  I edit it in place to "v2" with the bearer token
    Then  the current content is "v2"
    And  the prior content "v1" is still re-readable in the version history view

  @failure @p1
  Scenario: Share is idempotent and returns 400 on an unknown target
    Given SKILLGRID_HTTP_TOKEN is set to "secret"
    And  I am viewing a memory
    When  I share it to "team" with the token, then share to "team" again
    Then  both return 200 and the visibility stays "team" (idempotent)
    And  when I share it to an unknown target
    Then  I receive 400 and the visibility is unchanged

  @edge
  Scenario: With 013 absent every governance view renders a forward-compat placeholder
    Given I am on the Mnemonic > Memories view against a pre-013 store (no owner/version/status/usage/visibility)
    When  I view the asset metadata, layer drill-down, share, in-place edit, and review/status
    Then  each renders a labeled collapsed placeholder plus the flat pre-013 view
    And  the Memories view's search, detail, and actions stay fully interactive

  @failure @p1
  Scenario: A failed or absent 013 field kills that widget only (per-widget isolation)
    Given I am on the Mnemonic > Memories view against a 013-provisioned store
    And  the layer-drill-down field for one memory is missing or returns 500
    When  I view that memory
    Then  the layer drill-down widget shows an error or placeholder
    And  the asset grid, share, in-place edit, review/status, and loadout widgets remain fully interactive

@phase-6
Feature: Activity stream, SDD Plans, and Git views
  As an operator of a skillgrid-installed machine
  I want the dashboard to show a live activity stream, SDD plan progress, and git history
  So that I can watch the machine, track plan progress, and inspect commits without leaving the browser

  @happy @p0
  Scenario: GET /activity/events returns the event log newest-first
    Given the event log has 150 events
    When  I send GET /activity/events?limit=100
    Then  I receive 200 with the 100 most recent events (type, source, actor, severity, relatedIds)

  @happy @p0
  Scenario: GET /activity/stats returns the counters
    Given the event log has tasks, memories, searches, and errors
    When  I send GET /activity/stats
    Then  I receive 200 with the active-tasks, memories-stored, searches-today, and errors-last-hour counters

  @happy @p0
  Scenario: GET /activity/stream emits new events live (SSE)
    Given a subscribed SSE client on /activity/stream
    When  a new event occurs (e.g. a task update)
    Then  the client receives the event without a manual refresh
    And  when the client disconnects, the server cleans up the stream (no leaked goroutine)

  @happy @p0
  Scenario: Activity view renders a live feed with filters, stats, and alerts
    Given I am on the Activity view
    When  I view the feed
    Then  I see an infinite-scroll live feed (newest first) with compact cards (icon, timestamp, summary, severity border)
    And  I can filter by type, source, severity, actor, and time range
    And  I see a stats bar and an agent health panel
    And  errors/warnings show as alert banners

  @happy @p0
  Scenario: GET /plans returns SDD plans with status and progress
    Given the repo has a .skillgrid/sdd/ change with frontmatter + step files
    When  I send GET /plans
    Then  I receive 200 with the plan (id, title, status, progress)

  @happy @p0
  Scenario: GET /plans/{id} returns plan detail with steps and links
    Given a plan with steps and linked files/tasks/commits
    When  I send GET /plans/<id>
    Then  I receive 200 with the step checklist, acceptance criteria, and linked files/tasks/commits
    And  an unknown id returns 404

  @happy @p0
  Scenario: GET /specs/{path} returns the spec markdown
    Given a spec at a known path
    When  I send GET /specs/<path>
    Then  I receive 200 with the spec markdown
    And  an unknown path returns 404

  @happy @p0
  Scenario: Plans view renders cards, detail, dependency DAG, and spec viewer
    Given I am on the Plans view
    When  I view the plan cards
    Then  I see status and a progress bar per plan
    And  clicking a plan shows the step checklist, acceptance, linked files/tasks/commits, and a dependency DAG
    And  the spec viewer renders markdown + Mermaid (reusing the Docs rendering)
    And  every plan has a "Read in Docs →" link

  @happy @p0
  Scenario: GET /git/commits returns the commit list
    Given a fixture git repo with 20 commits
    When  I send GET /git/commits?limit=50
    Then  I receive 200 with commits (sha, message, author, date, +/- stats, branch lane)

  @happy @p0
  Scenario: GET /git/commits/{sha} and /git/diff/{sha} return the commit and its diff
    Given a known commit sha
    When  I send GET /git/commits/<sha>
    Then  I receive 200 with the commit metadata
    And  sending GET /git/diff/<sha> returns the diff
    And  an unknown sha returns 404

  @happy @p0
  Scenario: GET /git/file-history and /git/blame return history and blame
    Given a file with history
    When  I send GET /git/file-history?path=<path>
    Then  I receive 200 with the file history
    And  sending GET /git/blame?path=<path>&line=<n> returns the blame for that line
    And  an unknown path returns 404

  @happy @p0
  Scenario: Git view renders commit graph, list, detail, and blame
    Given I am on the Git view
    When  I view the commit graph
    Then  I see branch lanes and merge nodes with author avatars
    And  the commit list shows SHA (copyable), message, author, date, and +/- stats
    And  clicking a commit shows the unified/split diff + changed-file tree
    And  the blame view shows line-by-line attribution
    And  commits link to Kanban tasks via commit message conventions

  @failure @p1
  Scenario: Git view on a non-repo returns 503
    Given the working directory is not a git repository
    When  I send GET /git/commits?limit=50
    Then  I receive 503 with a reason

  @failure @p1
  Scenario: A failed activity/git endpoint shows an error in the view, not a blank page
    Given I am on the Activity view
    And  GET /activity/events returns 500
    When  I view the Activity view
    Then  I see an error message in the feed (not a blank page)
    And  the rest of the dashboard is still interactive

@phase-7
Feature: Prototypes gallery and dashboard polish
  As an operator of a skillgrid-installed machine
  I want the dashboard to show the .stitch/ prototypes and be polished and performant
  So that I can preview design prototypes and use the whole dashboard smoothly

  @happy @p0
  Scenario: GET /prototypes returns the .stitch/ gallery
    Given the repo has prototypes under .stitch/
    When  I send GET /prototypes
    Then  I receive 200 with the gallery (id, title, thumbnail, updatedAt)

  @happy @p0
  Scenario: GET /prototypes/{id} and /prototypes/{id}/html return the prototype
    Given a prototype with a known id
    When  I send GET /prototypes/<id>
    Then  I receive 200 with the prototype metadata
    And  sending GET /prototypes/<id>/html returns the prototype HTML
    And  an unknown id returns 404

  @failure @p1
  Scenario: Path traversal on /prototypes returns 400
    Given I send GET /prototypes/../secret
    When  the request reaches the server
    Then  I receive 400 with a JSON error
    And  only paths under .stitch/ resolve

  @happy @p0
  Scenario: Prototypes view renders gallery, sandboxed iframe, and code viewer
    Given I am on the Prototypes view
    When  I view the gallery grid
    Then  I see each prototype as a card
    And  clicking one opens a sandboxed iframe preview (sandbox attribute set)
    And  I can toggle the device viewport (desktop/tablet/mobile)
    And  I see a code viewer (HTML/CSS/JS side-by-side)
    And  I can export as standalone HTML / copy the code

  @happy @p0
  Scenario: Prototypes view shows version history with diffs
    Given a prototype with 2 versions
    When  I open its version history
    Then  I see both versions with a diff between them

  @edge
  Scenario: Dashboard is responsive and supports density modes
    Given I am on the dashboard
    When  I resize between desktop, tablet, and mobile
    Then  the layout adapts (no horizontal scroll, nav collapses appropriately)
    And  I can switch between comfortable and compact density
    And  the activity stream animates new events from the top (150ms ease)

  @happy @p0
  Scenario: Heavy features are code-split and the bundle stays within budget
    Given the SPA is built
    When  I inspect the build output
    Then  the graph and git features are lazy-loaded (separate chunks)
    And  the total bundle size is within the configured budget (reported in the build)

  @happy @p0
  Scenario: openapi.yaml documents all new routes with examples
    Given a running skillgrid serve
    When  I fetch GET /openapi.yaml
    Then  I see the tracker routes (providers, tasks CRUD, deps, stream), mnemonic routes (graph, files, memories, sessions, audit, search), activity routes (stream, events, stats), docs routes (tree, content, search, render), plans/specs routes, git routes, and prototypes routes — each with request/response examples

  @happy @p0
  Scenario: swagger loads and can exercise each new route
    Given a running skillgrid serve
    When  I open /swagger/
    Then  I see all new routes listed
    And  I can try GET /mnemonic/memories/{id} with a valid id
    Then  I receive a 200 response in the swagger try panel
    And  the old routes are still documented

  @edge
  Scenario: User-manual documents the views and tracker-CLI dependency
    Given I read the user-manual serve section
    When  I look for the dashboard documentation
    Then  I see all views described (Tracker, Mnemonic, Docs, Plans, Activity, Git, Prototypes)
    And  I see the per-provider tracker-CLI dependency documented (install/auth + what happens when the CLI is missing)

  @happy @p0
  Scenario: Full DoD smoke passes
    Given a running skillgrid serve with the dashboard deployed
    When  I run npm run build
    Then  the SPA builds into ui/dist/
    And  I run go test ./...
    Then  all tests pass
    And  I run go vet ./...
    Then  no issues reported
    And  I run tsc --noEmit
    Then  no type errors
    And  I open the dashboard in a browser
    Then  all views render (shipped views live, none left as stubs)

  @failure @p1
  Scenario: swagger still serves when new routes are added
    Given a running skillgrid serve
    When  I open /swagger/
    Then  it loads (no 404 or JS error)
    And  the old routes are still documented

# Rules:
# - >=1 @happy + @edge + @failure per phase
# - Every briefing.md per-phase WHAT bullet -> a Scenario
# - Every applicable threat-matrix row -> a Scenario in its owning @phase-NN
# - Scenario names unique and referenceable from tasks.md `Run:` / Expected lines
# - @p0 must cover briefing.md Definition of Done user-visible criteria
