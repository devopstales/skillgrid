# Task 01 Report — Community Detection (008 step 01)

## Status: DONE_WITH_CONCERNS

All 9 tasks (01.1–01.9) implemented, tested, and committed. Full relevant suite
passes; `go build ./...` succeeds. One concern: a pre-existing 005 latent bug
(stale `fileFirstSymbol` cache) surfaced and was mitigated with a test-only
reset — see "Issues & Concerns."

## What I implemented (each file)

### New files
- `skillgrid-cli/internal/mnemonic/store/migrations/012_community_knowledge_graph.sql`
  Additive 012 migration: `communities` (id, symbol_id UNIQUE) +
  `community_meta` (id, label, symbol_count, god_nodes, cache_key). 005
  symbols/edges untouched; `011` left owned by 005.
- `skillgrid-cli/internal/mnemonic/community/leiden.go`
  The pure-Go Leiden adapter. `Options` (Seed/Resolution/MaxIterations/NumRuns/
  Limit), `Community` (ID/Label/Members/GodNodes), `Result`
  (Communities/CacheKey/FromCache/Warning/Modularity/Updated). `Detect`
  consults the content-hash cache, else runs the seeded Leiden pass and writes
  the communities table.
- `skillgrid-cli/internal/mnemonic/community/godnodes.go`
  `RankGodNodes(db, symbolIDs, excludeHubs)` — degree ranking (distinct-neighbor
  count, matching the graph package's notion) with `excludeHubs` suppressing
  utility super-hubs (`hubSymbols`: incoming edges from >= `HubFileThreshold`=5
  distinct source files).
- `skillgrid-cli/internal/mnemonic/community/labels.go`
  LLM-free label derivation: top god-node name(s) + path dir; falls back to
  `community-N` when no god node (degree-0 or empty) — never fabricated.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_community.go`
  Three MCP tools (`code_communities`, `code_god_nodes` with `exclude_hubs` +
  `limit`, `code_explain_community` with required `id`) + `registerCommunityTools`
  + `numericArg` validation helper. Bad/missing args → clear validation error
  (no invented communities).
- Tests: `community/leiden_test.go`, `community/godnodes_test.go`,
  `community/labels_test.go`, `mcp/tools_code_community_test.go`,
  `mcp/tools_code_schema_test.go` (rewritten), `store/community_schema_test.go`,
  `service/community_test.go`, `cmd/skillgrid/code_community_test.go`.

### Modified files
- `skillgrid-cli/internal/mnemonic/mcp/server.go` — register
  `registerCommunityTools` in `Start`/`NewServer`.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_explore.go` — added the three
  community tools to the `menuCodeTools` unlisted-by-default list.
- `skillgrid-cli/internal/mnemonic/mcp/server_test.go` — `TestAllToolsRegistered`
  now expects the three new tools (count + names).
- `skillgrid-cli/internal/mnemonic/service/service.go` — thin community facade:
  `CodeCommunities`, `CodeGodNodes`, `CodeExplainCommunity` + DTO aliases
  (`CommunityResult`, `CommunityOptions`, `GodNode`, `CommunityExplanation`).
- `skillgrid-cli/cmd/skillgrid/code_intel.go` — CLI parity: `communities`,
  `god-nodes --exclude-hubs --limit`, `explain-community ID` subcommands.
- `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` — added exported
  `ResetFileFirstSymbol()` (test-only mitigation; see Concerns).
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_explore_test.go`,
  `tools_code_impact_test.go` — call `codeindex.ResetFileFirstSymbol()` before
  RunCodeIndex.
- `skillgrid-cli/go.mod` / `go.sum` — added `github.com/bluuewhale/loom`
  (pure-Go, zero deps, no CGo; `go 1.26`).

## How I wired the loom adapter

- **NodeRegistry mapping**: `loadGraph` reads 005 `symbols` (id, uid, name) and
  registers each with loom's `NodeRegistry` using the symbol's `uid` (preferred)
  or `name` as the key. A parallel `nodeRef` slice + `loomToSymbol` map
  (built via `reg.ID(key)`) invert the partition back to symbol ids.
- **LeidenOptions**: `graph.NewLeiden(graph.LeidenOptions{Seed, Resolution,
  MaxIterations, NumRuns})`. I default `Seed=1` (a fixed non-zero seed → loom
  runs a single deterministic pass, NumRuns ignored) for reproducibility;
  `Resolution=1`.
- **Seed**: the RNG is seeded via `LeidenOptions.Seed`; the cache key is a
  content-hash of the edge set so the partition is pinned + cached (01.8).
- **Graph build**: edges are treated **undirected** for Leiden (a call clusters
  the same either way). Only edges with both endpoints live (`to_id` set) build
  connectivity; name-only dangling edges are ignored (they would fabricate
  nodes). Self-loops and duplicate undirected pairs are de-duped.
- **< 2 nodes**: `symCount < 2` → one trivial community, `Warning` set
  (warn+continue), no crash (01.6).

## The 005 tool-surface baseline test (01.1)

`TestToolSurfaceBaseline` (in `tools_code_schema_test.go`) sweeps an
`expected005ToolSurface` map of all 18 005 `code_*` tools (name → required
param list) through the live `NewServer().ListTools()` and asserts each keeps
its exact name + required params. This is the baseline lock: any later change
that renames a 005 tool or changes its required params fails it. I also defined
the pre-existing-but-missing `assertCodeToolStable` helper (referenced by
`tools_code_hybrid_test.go` but never defined — a 005 build break), and kept the
original `TestCodeSearchSchemaStable`.

## Test results

Focused (brief's exact commands), all PASS:
- 01.1 `go test ./internal/mnemonic/mcp/... -run ToolSurfaceBaseline -count=1` → ok
- 01.2 `go test ./internal/mnemonic/community/... ./internal/mnemonic/mcp/... -run Communities -count=1` → ok
- 01.3 `go test ./internal/mnemonic/mcp/... -run CommunityToolsArgs -count=1` → ok
- 01.4 `go test ./internal/mnemonic/community/... -run GodNodes -count=1` → ok
- 01.5 `go test ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... -count=1` → ok
- 01.6 `go test ./internal/mnemonic/community/... -run TinyGraph -count=1` → ok
- 01.7 `go test ./internal/mnemonic/community/... -run "LabelFallsBack|LabelDerived" -count=1` → ok
- 01.8 `go test ./internal/mnemonic/community/... -run Reproducible -count=1` → ok
- 01.9 `go test ./cmd/skillgrid/... -run Parity -count=1` → ok

Full relevant suite (the brief's final gate), all PASS:
```
go test ./internal/mnemonic/community/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... ./internal/mnemonic/store/... ./cmd/skillgrid/... -count=1
ok  community  0.694s
ok  mcp        25.242s
ok  service    18.828s
ok  store       5.005s
ok  cmd        11.992s
```
`go build ./...` → BUILD_OK. I re-ran the mcp suite 5× consecutively to confirm
the cache-reset fix is stable (all ok).

## TDD evidence

### 01.1 `[RED]` — 005 tool-surface baseline
- RED: I started from the existing `tools_code_schema_test.go` which referenced
  `assertCodeToolStable` without defining it → `undefined: assertCodeToolStable`
  (build fail). That is the "red until baseline lock exists" state: the lock
  helper didn't exist.
- GREEN: defined `assertCodeToolStable` + `TestToolSurfaceBaseline` →
  `go test -run "ToolSurfaceBaseline|CodeSearchSchemaStable|HybridToolsDistinct" -count=1` → ok.

### 01.2 `[RED]` — Leiden communities
- RED: wrote `leiden_test.go` (Detect/Options/Result) before `leiden.go` existed
  → `undefined: Detect`, `undefined: Options`, `undefined: Result` (build fail,
  feature missing). Also wrote `tools_code_community_test.go` (code_communities)
  → `undefined: handleCodeCommunities`.
- GREEN: added `012_*.sql`, `leiden.go`, `labels.go`, `codeindex` dep,
  `tools_code_community.go` (code_communities), service facade →
  `go test ./internal/mnemonic/community/... ./internal/mnemonic/mcp/... -run Communities -count=1` → ok.

### 01.3 `[RED]` — community tool arg validation
- RED: `TestCommunityToolsArgs` written with the `code_god_nodes` non-numeric
  limit case → the handler didn't validate `limit` (silent `GetFloat` fallback)
  → `code_god_nodes with a non-numeric limit should be a validation error`.
- GREEN: added `numericArg` validation in `tools_code_community.go` →
  `go test -run CommunityToolsArgs -count=1` → ok.

### 01.4 `[RED]` — god nodes + exclude-hubs
- RED: `godnodes_test.go` (TestGodNodesRankByDegree) → `excludeHubs should
  suppress the utility super-hub 'helper'` (the `hubSymbols` query initially
  counted the edge's own file_id instead of the source file).
- GREEN: fixed `hubSymbols` to join the source symbol's file →
  `go test -run GodNodes -count=1` → ok.

### 01.5–01.9 `[AFK]`
- 01.5 `service/community_test.go` (CodeExplainCommunity members + entry
  points) → PASS on first run (facade existed from 01.2).
- 01.6 `leiden_test.go` TinyGraph → PASS (implemented in 01.2).
- 01.7 `labels_test.go` (fallback + derived) → PASS (labels.go from 01.2).
- 01.8 `leiden_test.go` ReproducibleAndCached → PASS (cache from 01.2).
- 01.9 `cmd/skillgrid/code_community_test.go` (CLI parity) → PASS (CLI
  subcommands added in 01.9).

## Files changed
See "What I implemented" — 8 new files, 7 modified, +2021/-19 lines vs the
pre-task tree. 005's `011_hybrid_code_intel.sql`, `011b_edges_file_id.sql`, and
the `extract/` package are byte-identical (rollback boundary held).

## Self-review findings
- **CGo-free**: verified loom has no `import "C"` and zero deps (go.mod is just
  the module + go directive).
- **Additive**: 005 symbols/edges/extractors untouched; only the 012 migration
  adds tables. No new edges introduced (the community pass reads, never writes,
  005's edges), so the "every new edge carries a confidence label" constraint is
  vacuously satisfied.
- **LLM-free labels**: `labels.go` derives from god-node names + paths; no API
  call. Fallback is `community-N`, never fabricated.
- **Seeded + reproducible**: fixed Seed=1 + content-hash cache key; 01.8 proves
  two Detect calls yield the identical partition and the second is a cache hit.
- **Bad args → abort**: `code_explain_community` with a non-existent/missing
  id returns a validation error (no invented community); `code_god_nodes` with
  a non-numeric limit returns a validation error.
- **005 tools stable**: `TestToolSurfaceBaseline` + `TestAllToolsRegistered`
  both pass; no 005 tool name/param changed.
- **gofmt**: all changed files gofmt-clean (a few pre-existing files in the
  repo are unformatted, but I didn't touch them).

## Issues & Concerns

1. **Pre-existing 005 latent bug (mitigated, not root-fixed):** the codeindex
   package keeps a process-global `fileFirstSymbol map[int64]int64` (keyed by
   file id, not by store). When multiple mcp fixtures each open a fresh temp
   store and run RunCodeIndex, file ids collide across stores, so a stale entry
   points at a non-existent symbol id → `upsert edge imports: FOREIGN KEY
   constraint failed (787)` on `edges.from_id`. This is a 005 bug that my new
   fixtures (which run RunCodeIndex in fresh stores) surfaced by changing test
   ordering. I added an exported `codeindex.ResetFileFirstSymbol()` and call it
   from the three mcp fixtures before RunCodeIndex (test-only; production
   single-process single-store is unaffected). The codeindex package's own tests
   already reset via an unexported helper. I chose a test-only mitigation over a
   root fix (e.g. keying the cache by store or clearing it at end of Run) to keep
   this change additive to 005 — but the root cause is in 005's indexer and a
   future non-test multi-store process could hit it. Flagging for a follow-up.

2. **`communities.id` has no primary key** in the 012 migration (it's a
   logical column, with `symbol_id` UNIQUE). I re-number communities 0..N-1 in
   stable order (sorted by smallest member) on every Detect and renumber on
   cache read, so the id is a stable, deterministic index. This is fine for the
   advisory partition, but if a future step wants stable cross-run community
   ids (beyond the cache key), the schema may need a natural key.

3. **`code_god_nodes` ranks the whole project** (not per-community) — the tool
   has no community scope. That matches the brief ("the most-connected
   symbols"); per-community god nodes are attached to each community in
   `code_communities` output and surfaced by `code_explain_community`.

4. **loom version**: added `github.com/bluuewhale/loom
   v0.0.0-20260402142357-ae7e94ccfead` (a pseudo-version; loom has no tagged
   release). It bumped go.mod's `go` directive from 1.25.5 to 1.26. loom's API
   matched the brief's assumptions exactly (`NewLeiden`, `LeidenOptions{Seed,
   Resolution, MaxIterations, NumRuns}`, `NodeRegistry`, `CommunityResult.
   Partition`).
