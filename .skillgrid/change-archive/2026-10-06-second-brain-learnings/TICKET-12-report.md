# TICKET-12 report — Teaching graph page (single page, tour, locals)

**Status:** DONE
**SATISFIES:** @step-13 `teaching-graph-page` (G14)

## What was implemented

### Go (mnemonic/internal/http/mnemonic_graph.go)
1. **Node enrichment (additive):** `graphNode` now carries `lines` (1-based
   start/end from the stored `symbols.start_line`/`end_line`) and
   `signature` (stored column, `omitempty`). `loadGraph`'s SQL extended with
   `COALESCE(signature,''), start_line, end_line`. All pre-existing fields
   unchanged (asserted in test).
2. **Edge enrichment (additive):** `graphEdge` now carries `line` (the stored
   `edges.line`), feeding the inspector's per-edge source-line rows.
   (Discovered as a defect in pass 3: the UI rendered `L{n}` from a field the
   Go side never populated — fixed with `TestGraphEnrich_EdgeLines`.)
3. **Pure on-demand locals resolver:** `resolveSymbolLocals(st, root, name,
   lang, startLine, endLine) (*SymbolLocals, error)` — parses ONE symbol's
   file per click via gotreesitter (same registry as pdg; Go-only, other
   languages degrade to nil) and returns params + locals (first declaration,
   source order; loop-init locals included). **Read-only by construction:**
   only `st.DB.QueryRow` + `os.ReadFile` + tree parse. Degraded states
   (unknown language, missing file, parse failure, unlocatable body) →
   `(nil, nil)`, never an error.

**Co-location choice (recorded):** the resolver lives in
`mnemonic/internal/http/mnemonic_graph.go` next to `loadGraph` — it is a
per-click view of the same symbol rows the graph derive serves. Deletable as
one section.

### UI (skillgrid-ui/src/features/mnemonic/GraphPage.tsx)
- **Tree panel** from served node paths; **two-way sync** (select node → tree
  auto-expands its file; expand/click file → graph focuses on the file's hub
  node; symbol rows select the node).
- **Enriched inspector:** identity (type, language, degree, community,
  **lines**, **signature**, file) + **typed connection groups** (grouped by
  edge kind with counts, per-edge **source line** `L{n}`, direction verb) +
  **ephemeral locals** preview (signature box when a source range exists,
  else a degraded "locals unavailable" note — no endpoint, no fetch).
- **Search-to-focus:** type a symbol → graph centers on it (focus label
  shown).
- **Filters:** kind / community / layer selects + newcomer cap (uncategorized
  `community == -1` nodes capped; hidden count shown).
- **Concept toggle:** switch fetching the existing `/knowledge/graph`
  (TICKET-09) into an anchor overlay listing concepts with orphan/deprecated
  flags.
- **Client-side tour:** 3 stops — workflow (entry point) → decisions
  (highest-centrality community) → top-centrality (max degree). No endpoint.
- **Degraded states render:** no edges → "degraded: no edges" banner + served
  file list; empty filters → "no symbols match the filters"; missing locals →
  degraded note; concept fetch failure → overlay simply absent.

`graph/types.ts`: additive `line?: number` on `GraphEdge`.

## Test results

| Gate | Command | Result |
|---|---|---|
| G14 Go | `cd mnemonic && go test ./internal/http/ -run "NodeEnriched\|LocalsResolver\|GraphEnrich" -v` | **PASS** — TestGraphEnrich_Lines, TestLocalsResolver_Ephemeral, TestLocalsResolver_EphemeralUnlocatable, TestGraphEnrich_EdgeLines (4/4) |
| vet | `cd mnemonic && go vet ./...` | clean (exit 0) |
| G14 UI | `cd skillgrid-ui && npx vitest run GraphPage` | **PASS** — 16/16 (2 pre-existing + 11 teaching-page + …) |
| full UI suite | `npx vitest run` | 181/181 PASS |
| full Go http suite | `go test ./internal/http/` | PASS (186s) |
| route table | `git diff --stat <base>..HEAD -- mnemonic/internal/http/server.go` | **no diff** — no new endpoint |

Notes on the G14 gate names: the brief's ticket-level command names
(`TestNodeEnrichedLines`, `TestLocalsResolver_Pure`) differ from the
acceptance.feature G14 names (`TestGraphEnrich_Lines`,
`TestLocalsResolver_Ephemeral`). I implemented the G14 names (the feature is
the acceptance source) and the ticket's test intents. Both patterns covered:
`-run "NodeEnriched|LocalsResolver|GraphEnrich"` passes all four tests.

## TDD Evidence

**RED (Go):** wrote `mnemonic_graph_enrich_test.go` first →
`go test -run "TestGraphEnrich_Lines|TestLocalsResolver"` →
`undefined: dbContent / undefined: resolveSymbolLocals` (build failure, the
expected missing-symbol failure for a not-yet-existing resolver), then after
stub-less impl: fixture line mismatch (start 7 vs real 4) surfaced a test bug
before GREEN.

**RED (UI):** rewrote `GraphPage.test.tsx` with 11 new tests →
`npx vitest run GraphPage` → **9 failed / 2 passed** (all new tests failed:
`Unable to find an element with the text: demo/a.go` — tree/inspector/
filters/tour/degraded states absent from the old page).

**GREEN (Go):** enrichment + resolver implemented → 4/4 PASS.
**GREEN (UI):** GraphPage upgraded → 16/16 PASS; full suite 181/181.

**Triangulation:** `TestLocalsResolver_EphemeralUnlocatable` (different
input: unlocatable body → nil, no error, store still unchanged) and
`TestGraphEnrich_EdgeLines` (different input: the 5-node chain fixture with
4 edges at lines 10–13, all lines asserted present).

**BDD:** SATISFIES @step-13 `teaching-graph-page`. No cucumber runner is
installed in this repo (no cucumber-js in package.json; the feature file is
gate-annotated `# G14`-style, not Gherkin-executed) — the acceptance
scenario's Then-clauses are covered 1:1 by the unit tests above:
- "browse tree to canvas and canvas to tree" → tree two-way sync test
- "inspector shows identity with lines, connections grouped by edge kind
  with source lines, and a code preview" → inspector test
- "search centers a named symbol and the tour completes at least 3 stops"
  → search + tour tests
- "locals expand ephemerally with an empty store diff" →
  TestLocalsResolver_Ephemeral (store digest diff empty)
- "orphan, broken-link, deprecated, empty, and degraded states all render"
  → degraded + empty-filters + missing-locals + concept-overlay tests

## Refinement (4-pass loop)

- **Pass 1 — Implement complete:** all 8 ticket items present (enrichment,
  resolver, tree, inspector, search, filters+cap, concept toggle, tour,
  degraded states, TDD). No placeholders.
- **Pass 2 — Expert re-read:** names verified against the codebase (`lines`,
  `signature`, `SymbolLocals`, `resolveSymbolLocals`, `applyFilters`,
  `TOUR_STOPS`); removed the unused `adjacency` memo left over from the old
  page; removed a redundant field-index dance in `paramNameDecls` (identifier
  children of a `parameter_declaration` are the names, per the actual AST
  dump); doc comments state the co-location + degraded contract.
- **Pass 3 — Defect hunt:** found 1 real defect — `GraphEdge.line` was
  rendered by the UI but never populated by Go (dead per-edge source lines);
  fixed in `loadGraph` + `graphEdge` and proved by
  `TestGraphEnrich_EdgeLines`. Also fixed a test-fixture bug (seeded
  `start_line` 7 didn't match the fixture source's real line 4 — the
  resolver's line-anchored lookup was correct; the fixture was wrong).
  Edge cases: `lines` nil-tolerant on the UI (old payloads), `signature`
  `omitempty` (nil on old stores), newcomer cap of 0 works, tour completes
  to null, concept fetch failure degrades.
- **Pass 4 — Polish:** gofmt clean on touched Go files, tsc clean,
  test-output noise reviewed (vitest prints only standard summary). Final
  pass 2→4: clean.

## Lazy (ponytail ladder)

Stopped at **reuse + minimum**:
- Locals parse reuses the **same gotreesitter registry** pdg already uses —
  no new grammar, no CGo, no new Go dependency.
- Locals are derived **client-side from the served graph payload**
  (signature + lines) — the resolver exists server-side and is pure, but the
  page itself makes **zero extra requests** (no new endpoint, per the
  ticket).
- Tree, filters, tour are plain React state over the served payload — no
  new dependency.
- Concept overlay reuses the existing TICKET-09 `/knowledge/graph` route.

Ceiling comments left:
- `mnemonic_graph.go` resolver section: "ponytail: ceiling = Go only (one
  grammar family)… non-Go languages degrade to nil rather than shipping a
  per-language locals grammar matrix."

Nothing the ticket asked for was skipped.

## Gates (freshly run)

- **G14 Go:** `cd mnemonic && go test ./internal/http/ -run "NodeEnriched|LocalsResolver|GraphEnrich" -v -count=1` → exit 0, `PASS` (4 tests). The brief's literal pattern `NodeEnriched|LocalsResolver` matches a subset (TestLocalsResolver_*); the full G14 set including GraphEnrich also passes.
- **G14 vet:** `cd mnemonic && go vet ./...` → exit 0, clean.
- **G14 UI:** `cd skillgrid-ui && npx vitest run GraphPage` → `Test Files 2 passed (2)`, `Tests 16 passed (16)`.
- **Route-table diff:** `git diff --stat 858a3e13..HEAD -- mnemonic/internal/http/server.go` → empty (no new endpoint).

## Checkpoint

Commits (branch release/2):
- `dbe825e6` feat(mnemonic): enrich code graph nodes with source lines + on-demand locals
- `56b52d06` feat(mnemonic): serve stored edge source lines on the graph
- `16ecb68` feat(ui): teaching graph page (tree, inspector, tour, locals)

[skillgrid-context] block present in all three (change / ticket / phase).

**Concurrency note:** a parallel session committed `0f71c2e7` ("feat(mnemonic):
enrich graph nodes with symbol signatures and line ranges") ~5 minutes into my
work — it captured the same enrichment changes I was mid-flight on, plus
unrelated untracked files (google-styleguides skill folder, parallel-cleave-port
spec, UI dist). My `dbe825e6` is therefore a formatting-only delta (12 ins/12
del) on top of it. All Go content is present and tested in HEAD; the two
commits together constitute the ticket's Go work. Flagging so the controller
can decide whether to rebase/squash.

## Files changed

- `mnemonic/internal/http/mnemonic_graph.go` (enrichment + resolver + edge line)
- `mnemonic/internal/http/mnemonic_graph_enrich_test.go` (new; 4 tests)
- `skillgrid-ui/src/features/mnemonic/GraphPage.tsx` (teaching page)
- `skillgrid-ui/src/features/mnemonic/GraphPage.test.tsx` (11 new tests)
- `skillgrid-ui/src/features/mnemonic/graph/types.ts` (`GraphEdge.line`)

## Self-review findings / concerns

- **Parallel-commit collision** (above) — content is intact and tested, but
  history interleaves two sessions' commits for the same feature.
- The G14 gate's literal test names (`TestGraphEnrich_Lines` /
  `TestLocalsResolver_Ephemeral`) come from the acceptance feature; the
  ticket brief named them `TestNodeEnrichedLines` / `TestLocalsResolver_Pure`.
  I kept the feature's names (acceptance is authoritative) — the brief's
  intent (lines served; locals ephemeral with empty store diff) is fully
  covered.
- Locals preview in the inspector currently renders the **signature** (the
  served ephemeral payload) rather than the resolver's full param/local
  walk — the resolver is implemented + proved server-side and is the seam for
  a future on-demand route; adding a route would violate "no new endpoint."
  The UI degraded note ("locals unavailable") covers symbols without a source
  range.
