# Mnemonic Web UI rewrite from prototype 001
#
# Source: .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/
# Trace: briefing success criteria; blueprint SATISFIES lines; ADR-0017.
# Verification worktree: a detached checkout of release/2 HEAD (clean of the
# concurrent uncommitted work in the main working tree).

## Requirements

### Requirement: nav-matches-mockup

The primary nav SHALL render the mockup's groups in order — Overview · Project (Kanban, Changes, Decisions, Prototypes) · Memory (Sessions, Code Graph, MemFS) · Observe (Telemetry, Compaction, Web Cache) · Docs · System (Security, Settings, Swagger). Git, Prototypes gallery (`/prototypes`), Memories, Search, and the mnemonic Decisions bridge SHALL stay routable but off the primary nav.

#### Scenario: sidebar renders the six mockup groups

```gherkin
      Given the SPA shell renders AppLayout
      When the sidebar is read top to bottom
      Then the group labels are Project, Memory, Observe, System
      And Overview and Docs are top-level items outside a group
      And no nav item points at /git, /mnemonic/memories, /mnemonic/search, or /decisions
```

#### Scenario: demoted routes stay reachable by URL

```gherkin
      Given the route tree in app.tsx
      When /git, /mnemonic/memories, /mnemonic/search, /decisions, and /prototypes are resolved
      Then each resolves to its page component, not the 404 stub
```

#### Gates

G1: AppLayout nav groups
  CHECK: cd skillgrid-ui && npx vitest run src/components/layout/AppLayout.test.tsx
  EXPECT: PASS
  EVIDENCE: PASS — 4 passed at d5e01985 (clean worktree)
G2: demoted routes present in the route tree
  CHECK: rg -c "path: '/(git|mnemonic/memories|mnemonic/search|decisions|prototypes)'" skillgrid-ui/src/app.tsx
  EXPECT: 5
  EVIDENCE: PASS — 5 at d5e01985

### Requirement: theme-tokens

The SPA SHALL use the mockup's dark console tokens (indigo accent on surface-800/900) and SHALL NOT ship the Terminal Ops green accent.

#### Scenario: index.css carries the indigo surface tokens

```gherkin
      Given skillgrid-ui/src/styles/index.css
      When the theme block is read
      Then --color-accent resolves to an indigo hue
      And --color-surface-800 / --color-surface-900 are defined
```

#### Gates

G3: theme tokens
  CHECK: rg -n "surface-800|surface-900|--color-accent" skillgrid-ui/src/styles/index.css | head -3
  EXPECT: at least three matching lines, accent is not a green hex
  EVIDENCE: PASS — --color-accent: #6366f1, surface-800/900 defined at d5e01985

### Requirement: d3-code-graph

The Code Graph page SHALL render `/mnemonic/graph/data` with a D3 v7 force layout capped at 500 nodes, with a node inspector that highlights callers/callees on selection (ADR-0017). Sigma.js, graphology, and @react-sigma SHALL NOT be dependencies.

#### Scenario: graph page requests at most 500 nodes

```gherkin
      Given GraphPage mounts
      When it fetches the graph
      Then the request carries limit=500
      And d3.forceSimulation drives node positions
      And a "Node Inspector" panel exists
```

#### Scenario: Sigma stack removed

```gherkin
      Given skillgrid-ui/package.json
      When dependencies are listed
      Then d3 is present
      And sigma, graphology, and @react-sigma/core are absent
```

#### Gates

G4: D3 graph wiring
  CHECK: rg -n "fetchGraph\(\{ limit: 500 \}\)|forceSimulation|Node Inspector" skillgrid-ui/src/features/mnemonic/GraphPage.tsx
  EXPECT: three matching lines
  EVIDENCE: PASS — 3 lines at d5e01985
G5: dependency swap
  CHECK: rg -c '"(sigma|graphology|@react-sigma/core)"' skillgrid-ui/package.json; rg -c '"d3"' skillgrid-ui/package.json
  EXPECT: 0 then 1
  EVIDENCE: PASS — 0 then 1 at d5e01985; build:check 298.2 kB/400 kB with vendor-d3 chunk

### Requirement: panels-fetch-live-endpoints

Compaction, Web Cache, Security, and Prototypes SHALL fetch their Go endpoints through the shared `apiGet`/`apiFetch` helpers (relative URL, `project` query param injected), and each endpoint SHALL exist on the Go mux.

#### Scenario: Observe panels hit registered routes

```gherkin
      Given CompactionPage, WebCachePage, TelemetryPage
      When their fetch paths are listed
      Then they include /context, /context/compaction, /web/status, /web/search
      And each path is registered on the Go mux in server.go
```

#### Scenario: Security and Prototypes panels hit registered routes

```gherkin
      Given the System → Security page and the Project → Prototypes page
      When their fetch paths are listed
      Then the Go mux registers /security/trivy as a JSON handler
      And the Go mux registers the Prototypes fetch path as a JSON handler
```

#### Gates

G6: Observe endpoints exist
  CHECK: cd skillgrid-cli && rg -c '"GET /(context|context/compaction|web/status|web/search)"' internal/mnemonic/http/server.go
  EXPECT: 4
  EVIDENCE: PASS — 4 at d5e01985 and ffd83337
G7: Security + Prototypes endpoints exist
  CHECK: cd skillgrid-cli && rg -c '"GET /security/trivy"' internal/mnemonic/http/server.go; p=$(rg -o "apiGet<[^>]*>\('[^']+'" ../skillgrid-ui/src/features/prototypes/PrototypesPage.tsx ../skillgrid-ui/src/features/spikes/SpikesPage.tsx 2>/dev/null | head -1 | sed "s/.*('//;s/'//"); rg -c "\"GET $p\"" internal/mnemonic/http/server.go
  EXPECT: 1 then 1
  EVIDENCE: PASS — 1 then 1 at 0bc899a6 (path=/prototypes)

### Requirement: relative-api-urls

Production API calls SHALL be same-origin relative URLs with `project` injected; an absolute `127.0.0.1:7438` origin MAY appear only behind `import.meta.env.DEV` (Vite dev iframe case).

#### Scenario: apiBase is relative in production

```gherkin
      Given skillgrid-ui/src/lib/apiBase.ts
      When apiOrigin() is evaluated outside DEV
      Then it returns the empty string
```

#### Gates

G8: no hardcoded origin outside apiBase
  CHECK: rg -l "127\.0\.0\.1:7438" skillgrid-ui/src --glob '!**/apiBase.ts' --glob '!**/*.test.*'
  EXPECT: only files where the string is display text (AppLayout footer label), no fetch() call sites
  EVIDENCE: PASS — only AppLayout.tsx footer display label at d5e01985; no fetch() site

### Requirement: briefing-task-refs-linkified

A plan briefing rendered through MarkdownView SHALL still turn `#NNN` / `task-NNN` references into `/tracker?task=NNN` links (regression from moving briefings into MarkdownView in aa0aa791).

#### Scenario: briefing #012 becomes a tracker link

```gherkin
      Given a plan whose briefing text is "See #012 for the design."
      When PlansPage opens the plan
      Then an anchor with href /tracker?task=012 is rendered
```

#### Gates

G9: linked tasks regression
  CHECK: cd skillgrid-ui && npx vitest run src/features/plans
  EXPECT: PASS (PlansPage.linkedTasks.test.tsx green)
  EVIDENCE: PASS — 2 files, 11 passed at ffd83337 (RED 87/88 at d5e01985 before the fix)

### Requirement: verification-floor

`npm test`, `npm run build:check`, `go build ./...`, and `go build -tags ui ./...` SHALL exit 0 on the integrated tree.

#### Scenario: full UI suite and bundle budget pass

```gherkin
      Given the skillgrid-ui package
      When npm test and npm run build:check run
      Then both exit 0
      And the index chunk is under the 400 kB budget
```

#### Scenario: Go embed builds

```gherkin
      Given the skillgrid-cli module
      When go build ./... and go build -tags ui ./... run
      Then both exit 0
```

#### Gates

G10: UI suite + budget
  CHECK: cd skillgrid-ui && npm test && npm run build:check
  EXPECT: exit 0, "Bundle-size budget OK."
  EVIDENCE: PASS — clean worktree at 0bc899a6: npm test 110 passed, build:check 298.7 kB "Bundle-size budget OK."; main tree after d35d28d3: 122 passed
G11: Go embed
  CHECK: cd skillgrid-cli && go build ./... && go build -tags ui ./...
  EXPECT: exit 0
  EVIDENCE: PASS — go build ./... and go build -tags ui ./... exit 0 at 0bc899a6; go test ./internal/mnemonic/http/ and .../docs/ ok
