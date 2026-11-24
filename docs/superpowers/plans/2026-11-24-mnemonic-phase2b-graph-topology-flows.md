# Mnemonic Phase 2B: Graph Topology, Leiden Communities & Execution Flows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver architectural graph topology intelligence (`communities`, `god_nodes`, `flows`, `cycles`, `shortest_path`) materialized inside `~/.skillgrid/mnemonic/<repo>.db`, promote `code_topology` into Eager Core MCP tools, attach affected execution flows to `code_affected`, and add the interactive TUI Topology view.

**Architecture:** Extends Phase 2A's `internal/code/csr.Graph` with deterministic Leiden modularity + refinement clustering (`seed = 0xc0de`), hub-PageRank community naming, cross-community `god_nodes` scoring, Tarjan strongly connected components (`cycles`), `-ln(confidence)` Dijkstra `shortest_path` with weakest-hop reporting, and ranked entry-point DFS execution `flows`. Materialized partitions and flows persist into `002_code_topology.sql` keyed by `code_index_meta.topology_generation` so cold `--standalone` CLI reads execute without recomputation.

**Tech Stack:** Go 1.26 (`CGO_ENABLED=0`), `modernc.org/sqlite`, `github.com/mark3labs/mcp-go`, `charmbracelet/bubbletea`, `internal/code/{api,store,csr,engine,mcp,http,client,cli,tui}` from Plan 2A.

## Global Constraints

- Prerequisite: Plan 2A (`docs/superpowers/plans/2026-11-24-mnemonic-phase2a-core-code-graph.md`) completed and green.
- Go module path: `mnemonic`; `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go test -race ./...` must pass.
- Every source file stays `<600` LOC; SQL lives strictly inside `internal/code/store`.
- Leiden clustering must be **100% deterministic** (`seed = 0xc0de`) regardless of map iteration or SQLite row insertion order, and exclude `AMBIGUOUS` edges when `len(g.SymIDs) > 10000`.
- Confidence string enum across `code_flows.min_confidence` and `shortest_path`: `LSP_RESOLVED | EXTRACTED | SYNTHESIZED | INFERRED | AMBIGUOUS`.
- Zero stubs: `code_topology` joins **Eager Core** (bringing Eager Core from `6` -> `7` tools; `agent` profile `9` -> `10`; `all` profile `11` -> `12`).

---

## File Structure (Phase 2B)

```text
internal/code/
├── api/{types.go,service.go}                 # Modify: add Community, Flow, Cycle, PathHop, TopologyRequest/Result + Service.Topology
├── store/
│   ├── migrations/002_code_topology.sql      # Create: code_communities, code_flows, code_flow_steps
│   ├── topology_io.go                        # Create: ReplaceTopology, QueryTopology, FlowsContainingSymbols
│   └── topology_test.go                      # Create: migration & persistence round-trip tests
├── csr/
│   ├── leiden.go                             # Create: deterministic Leiden (seed 0xc0de) + hub PageRank naming
│   ├── topology.go                           # Create: GodNodes, TarjanCycles, ConfidenceShortestPath
│   ├── flows.go                              # Create: entry-point scorer + bounded DFS (3..10) + terminal round-robin dedup
│   └── topology_test.go                      # Create: 10-shuffle determinism, Tarjan cycle, weakest-hop & flow tests
├── engine/{local.go,topology.go}             # Modify/Create: auto-refresh topology when topology_generation < graph_generation + enrich Affected
└── {mcp,http,client,cli,tui}/                # Modify: register Eager Core `code_topology`, `/v1/code/topology`, CLI `topology`, TUI `t` view
```

---

### Task 1: Topology Schema (`002_code_topology.sql`), API Contracts & Store Persistence

**Files:**
- Modify: `internal/code/api/types.go`, `internal/code/api/service.go`
- Create: `internal/code/store/migrations/002_code_topology.sql`, `internal/code/store/topology_io.go`
- Test: `internal/code/store/topology_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB`, `store.Init` (auto-applies `002_code_topology.sql` via `migrate.Apply`).
- Produces:
  - `api.Community`, `api.GodNode`, `api.ExecutionFlow`, `api.CycleGroup`, `api.PathHop`, `api.TopologyRequest`, `api.TopologyResult`, and `Service.Topology(ctx context.Context, req TopologyRequest) (TopologyResult, error)`.
  - `store.ReplaceTopology(ctx context.Context, tx *sql.Tx, gen int64, comms []api.Community, memberMap map[int64]int64, flows []FlowRecord) error`
  - `store.FlowsTouchingSymbols(ctx context.Context, db *sqlitepool.DB, symbolIDs []int64) ([]string, error)`.

- [ ] **Step 1: Write failing topology store migration & round-trip test**

`internal/code/store/topology_test.go`:

```go
package store

import (
	"context"
	"database/sql"
	"testing"

	"mnemonic/internal/code/api"
)

func TestReplaceTopologyAndFlowsTouchingSymbols(t *testing.T) {
	d := openTestRepoDB(t)
	ctx := context.Background()

	// Seed 3 symbols in 1 file.
	b := FileBundle{
		File: api.FileRecord{Path: "svc/order.go", Language: "go", Tier: "rich", Size: 80, MtimeNS: 1, ContentHash: "c1", ASTHash: "a1"},
		Symbols: []api.Symbol{
			{UID: "u-file", Name: "order.go", QualifiedName: "svc/order.go", Kind: "file", StartLine: 1, EndLine: 20, BodyHash: "b0"},
			{UID: "u-entry", Name: "HandleCreate", QualifiedName: "svc.HandleCreate", Kind: "function", IsExported: true, StartLine: 2, EndLine: 10, BodyHash: "b1"},
			{UID: "u-save", Name: "SaveOrder", QualifiedName: "svc.SaveOrder", Kind: "function", StartLine: 11, EndLine: 20, BodyHash: "b2"},
		},
	}
	if err := d.Write(ctx, func(tx *sql.Tx) error { return UpsertFileBundle(ctx, tx, b) }); err != nil {
		t.Fatal(err)
	}
	var entryID, saveID int64
	_ = d.R.QueryRow(`SELECT id FROM code_symbols WHERE uid='u-entry'`).Scan(&entryID)
	_ = d.R.QueryRow(`SELECT id FROM code_symbols WHERE uid='u-save'`).Scan(&saveID)

	comms := []api.Community{{ID: 1, Label: "svc/HandleCreate", HubSymbolUID: "u-entry", HubSymbolID: entryID, Size: 2, Cohesion: 1.0}}
	members := map[int64]int64{entryID: 1, saveID: 1}
	flows := []FlowRecord{{
		UID: "flow-1", Name: "HandleCreate -> SaveOrder", EntryID: entryID, TerminalID: saveID,
		MinConfidence: "EXTRACTED", CommunitiesSpanned: 1,
		Steps: []FlowStepRecord{{SymbolID: entryID, EdgeKind: "entry"}, {SymbolID: saveID, EdgeKind: "calls"}},
	}}
	if err := d.Write(ctx, func(tx *sql.Tx) error {
		return ReplaceTopology(ctx, tx, 1, comms, members, flows)
	}); err != nil {
		t.Fatal(err)
	}

	names, err := FlowsTouchingSymbols(ctx, d, []int64{saveID})
	if err != nil || len(names) != 1 || names[0] != "HandleCreate -> SaveOrder" {
		t.Fatalf("FlowsTouchingSymbols = %v err=%v", names, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/store/... -run TestReplaceTopology -v`
Expected: FAIL (`undefined: ReplaceTopology`).

- [ ] **Step 3: Add `002_code_topology.sql` (from Spec §3.2), `api` types, and `store/topology_io.go`**

Add to `internal/code/api/types.go` & `service.go`:

```go
type Community struct {
	ID           int64   `json:"id"`
	Label        string  `json:"label"`
	HubSymbolID  int64   `json:"-"`
	HubSymbolUID string  `json:"hub_symbol_uid"`
	Size         int     `json:"size"`
	Cohesion     float64 `json:"cohesion"`
	GodScoreMax  float64 `json:"god_score_max"`
}

type GodNode struct {
	SymbolUID           string  `json:"symbol_uid"`
	QualifiedName       string  `json:"qualified_name"`
	FilePath            string  `json:"file_path"`
	InDegree            int     `json:"in_degree"`
	OutDegree           int     `json:"out_degree"`
	CrossCommunityRatio float64 `json:"cross_community_ratio"`
	Score               float64 `json:"score"`
}

type ExecutionFlow struct {
	UID                string   `json:"uid"`
	Name               string   `json:"name"`
	Steps              []string `json:"steps"`
	MinConfidence      string   `json:"min_confidence"`
	CommunitiesSpanned int      `json:"communities_spanned"`
}

type CycleGroup struct {
	Kind    string   `json:"kind"` // symbol | file
	Members []string `json:"members"`
}

type PathHop struct {
	FromSymbol string `json:"from_symbol"`
	ToSymbol   string `json:"to_symbol"`
	EdgeKind   string `json:"edge_kind"`
	Confidence string `json:"confidence"`
}

type TopologyRequest struct {
	Project        string `json:"project,omitempty"`
	RootDir        string `json:"root_dir,omitempty"`
	Mode           string `json:"mode"` // overview | communities | god_nodes | flows | cycles | path
	From           string `json:"from,omitempty"`
	To             string `json:"to,omitempty"`
	Directed       bool   `json:"directed,omitempty"`
	IncludeTests   bool   `json:"include_tests,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type TopologyResult struct {
	Mode          string          `json:"mode"`
	Communities   []Community     `json:"communities,omitempty"`
	GodNodes      []GodNode       `json:"god_nodes,omitempty"`
	Flows         []ExecutionFlow `json:"flows,omitempty"`
	Cycles        []CycleGroup    `json:"cycles,omitempty"`
	Path          []PathHop       `json:"path,omitempty"`
	MinConfidence string          `json:"min_confidence,omitempty"`
}
```

Implement `store/topology_io.go` clearing and inserting `code_communities`, updating `code_symbols.community_id`, inserting `code_flows` + `code_flow_steps`, setting `code_index_meta.topology_generation = gen`, and implementing `FlowsTouchingSymbols`.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/store/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/api internal/code/store
git commit -m "feat(code): add Phase 2B topology schema, DTOs and persistence queries"
```

---

### Task 2: Deterministic Leiden Clustering (`seed = 0xc0de`) & Hub PageRank Labeling (`internal/code/csr/leiden.go`)

**Files:**
- Create: `internal/code/csr/leiden.go`
- Test: `internal/code/csr/topology_test.go`

**Interfaces:**
- Consumes: `csr.Graph`.
- Produces: `(g *Graph).DetectCommunities(resolution float64) (commByNode []int64, cohesion map[int64]float64)`.

- [ ] **Step 1: Write failing 10-run order-independence & two-clique separation test**

`internal/code/csr/topology_test.go`:

```go
package csr

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestLeidenSeparatesTwoDenseClustersDeterministically(t *testing.T) {
	edges := [][2]uint32{
		{0, 1}, {1, 2}, {2, 0}, {0, 2}, // Clique A
		{3, 4}, {4, 5}, {5, 3}, {3, 5}, // Clique B
		{2, 3}, // Single weak bridge
	}
	var baseline []int64
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 10; trial++ {
		shuffled := append([][2]uint32(nil), edges...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		b := NewBuilder(6)
		for i := uint32(0); i < 6; i++ {
			b.SetNodeMeta(i, int64(100+i), false, false)
		}
		for _, e := range shuffled {
			b.AddEdge(e[0], e[1], EdgeCalls, ConfExtracted)
		}
		comms, _ := b.Build(1).DetectCommunities(1.0)
		if comms[0] != comms[1] || comms[1] != comms[2] {
			t.Fatalf("trial %d split clique A: %v", trial, comms)
		}
		if comms[3] != comms[4] || comms[4] != comms[5] {
			t.Fatalf("trial %d split clique B: %v", trial, comms)
		}
		if comms[0] == comms[3] {
			t.Fatalf("trial %d merged separate cliques: %v", trial, comms)
		}
		if trial == 0 {
			baseline = comms
		} else if !reflect.DeepEqual(comms, baseline) {
			t.Fatalf("non-deterministic partition on trial %d: got %v want %v", trial, comms, baseline)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/csr/... -run TestLeiden -v`
Expected: FAIL (`undefined: DetectCommunities`).

- [ ] **Step 3: Implement deterministic Leiden modularity + local refinement in `csr/leiden.go`**

`DetectCommunities(resolution float64)`:
1. Canonicalizes node visit order by ascending `SymIDs[i]` perturbed with a fixed SplitMix64 PRNG seeded with `0xc0de` (never Go map iteration!).
2. Builds symmetrized weighted adjacency (dropping `ConfAmbiguous` edges when `len(g.SymIDs) > 10000`).
3. Executes Leiden fast local-move modularity optimization ($\Delta Q = k_{i,in} - \gamma \frac{\Sigma_{tot} k_i}{2m}$), followed by well-connected subset refinement within each super-community and aggregate graph compression (max 8 outer levels), then canonicalizes final community IDs `1..K` ordered by largest community size then lowest min `SymID`.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/csr/... -run TestLeiden -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/csr/leiden.go internal/code/csr/topology_test.go
git commit -m "feat(code): add deterministic Leiden community detection on CSR graph"
```

---

### Task 3: Architectural `god_nodes`, Tarjan `cycles` & Weakest-Hop `shortest_path` (`internal/code/csr/topology.go`)

**Files:**
- Create: `internal/code/csr/topology.go`
- Modify: `internal/code/csr/topology_test.go`

**Interfaces:**
- Consumes: `csr.Graph`, `commByNode []int64`.
- Produces:
  - `(g *Graph).RankGodNodes(commByNode []int64, includeTests bool, limit int) []GodNodeIdx`
  - `(g *Graph).TarjanCycles(maxCycles int) [][]uint32`
  - `(g *Graph).ConfidenceShortestPath(src, dst uint32, directed bool) (nodes []uint32, minConf uint8, ok bool)`.

- [ ] **Step 1: Write failing `god_nodes`, Tarjan SCC, and weakest-hop Dijkstra tests**

Append to `internal/code/csr/topology_test.go`:

```go
func TestTarjanCyclesAndWeakestHopShortestPath(t *testing.T) {
	b := NewBuilder(4)
	for i := uint32(0); i < 4; i++ {
		b.SetNodeMeta(i, int64(i+1), false, false)
	}
	b.AddEdge(0, 1, EdgeCalls, ConfExtracted)
	b.AddEdge(1, 0, EdgeCalls, ConfExtracted) // 2-node cycle {0,1}
	b.AddEdge(1, 2, EdgeCalls, ConfInferred)  // weaker hop
	b.AddEdge(2, 3, EdgeCalls, ConfExtracted)
	g := b.Build(1)

	cycles := g.TarjanCycles(10)
	if len(cycles) != 1 || len(cycles[0]) != 2 {
		t.Fatalf("cycles = %v, want 1 cycle of size 2", cycles)
	}
	path, minConf, ok := g.ConfidenceShortestPath(0, 3, true)
	if !ok || len(path) != 4 || minConf != ConfInferred {
		t.Fatalf("path = %v minConf=%d ok=%v, want [0 1 2 3] ConfInferred", path, minConf, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/csr/... -run TestTarjanCycles -v`
Expected: FAIL (`undefined: TarjanCycles`).

- [ ] **Step 3: Implement `topology.go`**

```go
package csr

import (
	"container/heap"
	"math"
	"sort"
)

type GodNodeIdx struct {
	Idx        uint32
	InDeg      int
	OutDeg     int
	CrossRatio float64
	Score      float64
}

func (g *Graph) RankGodNodes(commByNode []int64, includeTests bool, limit int) []GodNodeIdx {
	var out []GodNodeIdx
	for u := uint32(0); u < uint32(len(g.SymIDs)); u++ {
		if g.Generated[u] || (!includeTests && g.TestNode[u]) {
			continue
		}
		outDeg := int(g.FwdOffsets[u+1] - g.FwdOffsets[u])
		inDeg := int(g.RevOffsets[u+1] - g.RevOffsets[u])
		total := outDeg + inDeg
		if total == 0 {
			continue
		}
		cross := 0
		myComm := int64(0)
		if int(u) < len(commByNode) {
			myComm = commByNode[u]
		}
		for e := g.FwdOffsets[u]; e < g.FwdOffsets[u+1]; e++ {
			if v := g.FwdTargets[e]; int(v) < len(commByNode) && commByNode[v] != myComm {
				cross++
			}
		}
		for e := g.RevOffsets[u]; e < g.RevOffsets[u+1]; e++ {
			if v := g.RevSources[e]; int(v) < len(commByNode) && commByNode[v] != myComm {
				cross++
			}
		}
		ratio := float64(cross) / float64(total)
		score := math.Log1p(float64(inDeg)) * math.Log1p(float64(outDeg)) * (1.0 + ratio)
		if score > 0 {
			out = append(out, GodNodeIdx{Idx: u, InDeg: inDeg, OutDeg: outDeg, CrossRatio: ratio, Score: score})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
```

Plus iterative Tarjan's SCC (`len(scc) >= 2` or self-loop) and binary-heap Dijkstra minimizing `sum(-math.Log(ConfidenceWeight(conf)) + 0.05)` per hop while tracking the minimum confidence hop along the predecessor chain.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/csr/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/csr/topology.go internal/code/csr/topology_test.go
git commit -m "feat(code): add CSR god_nodes scorer, Tarjan SCC cycles, and confidence Dijkstra path"
```

---

### Task 4: Entry-Point Execution `flows` & Flow Blast Radius in `code_affected` (`internal/code/csr/flows.go`, `engine`)

**Files:**
- Create: `internal/code/csr/flows.go`, `internal/code/engine/topology.go`
- Modify: `internal/code/engine/local.go`, `internal/code/engine/local_test.go`

**Interfaces:**
- Consumes: `csr.Graph`, `store.ReplaceTopology`, `store.FlowsTouchingSymbols`.
- Produces:
  - `(g *Graph).ExtractExecutionFlows(names []string, exported []bool, commByNode []int64, maxFlows int) []TracedFlow`
  - `(l *Local).Topology(ctx context.Context, req api.TopologyRequest) (api.TopologyResult, error)`
  - Automatic population of `AffectedResult.AffectedFlows` in `(l *Local).Affected`.

- [ ] **Step 1: Write failing execution flow extraction & `code_affected` flow blast-radius test**

Append to `internal/code/engine/local_test.go`:

```go
func TestTopologyOverviewAndAffectedExecutionFlows(t *testing.T) {
	dataDir := t.TempDir()
	repoDir := writeFlowGitFixtureRepo(t) // main -> HandleCheckout -> ChargeCard -> SaveReceipt
	paths, _ := config.NewPaths(dataDir)
	pool := sqlitepool.NewPool()
	defer pool.Close()

	svc := NewLocal(paths, pool, nil)
	ctx := context.Background()
	if _, err := svc.Index(ctx, api.IndexRequest{Project: "flow-app", RootDir: repoDir, Full: true}); err != nil {
		t.Fatal(err)
	}
	topo, err := svc.Topology(ctx, api.TopologyRequest{Project: "flow-app", RootDir: repoDir, Mode: "overview"})
	if err != nil || len(topo.Communities) == 0 || len(topo.Flows) == 0 {
		t.Fatalf("Topology overview = %+v err=%v", topo, err)
	}

	// Edit leaf function SaveReceipt on disk -> Affected must surface the enclosing HandleCheckout flow!
	mutateLeafFunctionInGitRepo(t, repoDir)
	aff, err := svc.Affected(ctx, api.AffectedRequest{Project: "flow-app", RootDir: repoDir})
	if err != nil || len(aff.AffectedFlows) == 0 {
		t.Fatalf("expected AffectedFlows on leaf edit, got %+v err=%v", aff, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/engine/... -run TestTopologyOverview -v`
Expected: FAIL (`undefined: svc.Topology`).

- [ ] **Step 3: Implement `csr/flows.go` and `engine/topology.go`**

- `csr/flows.go`: Scores candidate entry points (`main`/`init`/`Run`/`Serve`/`Handle*` + exported callables with `in_degree == 0` and `out_degree >= 1`), runs cycle-free DFS (`depth 3..10`) over `calls`/`implements` edges (`conf >= ConfInferred`), and deduplicates traces round-robin by terminal node (`maxFlows = 100`).
- `engine/topology.go`: Checks `topology_generation < graph_generation`; if stale, computes CSR Leiden + hub PageRank labels + flows and persists via `store.ReplaceTopology` in a `WriteBatch` transaction, then serves `overview`, `communities`, `god_nodes`, `flows`, `cycles`, and `path`. Updates `engine/local.go`'s `Affected` to query `store.FlowsTouchingSymbols` for changed + impacted symbol IDs.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/engine/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/csr/flows.go internal/code/engine
git commit -m "feat(code): add execution flow tracer, auto-materialized topology, and affected flow blast radius"
```

---

### Task 5: Promote `code_topology` to Eager Core MCP, Daemon `/v1/code/topology`, CLI & TUI Topology View

**Files:**
- Modify: `internal/code/mcp/server.go`, `internal/code/mcp/server_test.go`
- Modify: `internal/code/http/handlers.go`, `internal/code/client/client.go`
- Modify: `internal/code/cli/cli.go`, `internal/code/cli/cli_test.go`
- Modify: `internal/code/tui/tab.go`

**Interfaces:**
- Consumes: `api.Service.Topology`.
- Produces:
  - **Eager Core** MCP tool `code_topology` (`readOnlyHint: true`, **no** `WithDeferLoading`, validating `mode` ∈ `overview|communities|god_nodes|flows|cycles|path`).
  - Daemon `POST /v1/code/topology`, `DaemonClient.Topology`, CLI `mnemonic code topology --mode ...`, and TUI `t` key Topology View (communities + `god_nodes` + `flows` inspector).

- [ ] **Step 1: Update MCP surface test asserting 7 Eager Core tools (`code_topology` eager)**

In `internal/code/mcp/server_test.go`:

```go
func TestPhase2BMCPPromotesCodeTopologyToEagerCore(t *testing.T) {
	srv := NewServer(nil, "agent")
	tools := srv.ListTools()
	if len(tools) != 10 { // 7 Eager Core (incl. code_topology) + 3 Deferred Extended
		t.Fatalf("agent tool count = %d, want 10", len(tools))
	}
	if !IsEagerTool(srv, "code_topology") {
		t.Fatal("code_topology must be registered in Eager Core without DeferLoading")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/mcp/... -v`
Expected: FAIL (`agent tool count = 9, want 10`).

- [ ] **Step 3: Wire `code_topology` across MCP, Daemon HTTP/Client, CLI, and `[2: Code]` TUI**

Register `code_topology` eagerly in `internal/code/mcp/server.go`, add `POST /v1/code/topology` in `http/handlers.go` + `client/client.go`, add `topology` subcommand in `cli/cli.go`, and add keybinding `t` in `tui/tab.go` toggling the 3-column Leiden Communities / God Nodes / Execution Flows browser.

- [ ] **Step 4: Run full workspace build & race test suite**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test -race ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/{mcp,http,client,cli,tui}
git commit -m "feat(code): expose Eager Core code_topology across MCP, daemon, CLI and TUI"
```
