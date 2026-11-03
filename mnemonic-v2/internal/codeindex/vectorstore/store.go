// Package vectorstore is the dialect-agnostic semantic vector store
// (TICKET-08a, D-CI8). The Engine (M2 — the layering decided) holds the
// Store (never a *sql.DB, the S-2 rule holds) and renders the vector SQL in
// the adapters. The preference order is HNSW → vec0 → brute-force (the v1
// live path, lifted; PG collapses to brute-force, degraded: brute_force).
// LSH + IVF are DROPPED (D-CI8 — not shipped; v1's lsh_buckets is
// schema-only, IVF is an adapter not wired).
package vectorstore

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
)

// Result is one top-K semantic search hit (the symbol/chunk id + the cosine
// distance/similarity, so the semantic leg can report Provenance.Sim).
type Result struct {
	ID   int64
	Dist float64 // cosine distance in [0,2]
	Sim  float64 // 1 - Dist
}

// Engine is the dialect-agnostic vector store over the Store (M2 — the
// Engine holds the Store, never a *sql.DB). Vector storage goes through the
// Store's vector methods (EmbedSymbol/EmbedChunk/ClearVectors, TICKET-01's
// additive methods); the ANN indexes (HNSW/vec0, SQLite-only) are rendered
// by the SQLite adapter via the store's RawExec/RawQuery seams.
type Engine struct {
	st       store.Store
	emb      embedder.Embedder
	mu       sync.Mutex
	disabled map[string]bool // the DisableIndexForTest seam (disable HNSW/vec0)
	// hnsw is the in-memory HNSW graph (SQLite-only; nil on PG).
	hnsw *hnswGraph
}

// NewEngine builds the Engine over the Store + the embedder seam (M2). The
// embedder is the 768-dim opt-in + the hash-64 default (the floor). A nil
// embedder is the "absent" state (the vector leg degrades to no_vector).
func NewEngine(st store.Store, emb embedder.Embedder) *Engine {
	return &Engine{
		st:       st,
		emb:      emb,
		disabled: map[string]bool{},
	}
}

// EmbedSymbol embeds the symbol's text (name + signature) and writes the
// vector via the Store's vector method (the BLOB source of truth) + the
// SQLite ANN tables (HNSW/vec0, when the dim activates them). fileID is the
// file row the symbol belongs to (the FK reference). uid is the symbol's
// local upsert key (the lookup for the symbol id).
func (e *Engine) EmbedSymbol(ctx context.Context, fileID int64, uid, text string) error {
	if e.emb == nil {
		return nil // absent embedder → no vector (the floor)
	}
	vecs, err := e.emb.Embed(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("vectorstore: embed symbol: %w", err)
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil // degenerate embedding → no vector
	}
	vec := vecs[0]
	// Resolve the symbol id by uid (the local upsert key, INV-6).
	sym, err := e.st.SymbolByUID(ctx, uid)
	if err != nil {
		return fmt.Errorf("vectorstore: symbol %s: %w", uid, err)
	}
	if err := e.st.EmbedSymbol(ctx, sym.ID, e.emb.Model(), vec); err != nil {
		return fmt.Errorf("vectorstore: write symbol vector: %w", err)
	}
	// The SQLite ANN dual-write (HNSW/vec0) — only when the dim activates
	// the ANN indexes (the 768-dim gating).
	if annActive(e.emb) {
		e.dualWriteSymbol(ctx, sym.ID, vec)
	}
	return nil
}

// EmbedChunk embeds the chunk's text and writes the vector via the Store's
// vector method + the SQLite ANN tables. fileID is the file row; chunkIdx is
// the chunk's position (the chunk id lookup).
func (e *Engine) EmbedChunk(ctx context.Context, fileID int64, chunkIdx int, text string) error {
	if e.emb == nil {
		return nil
	}
	vecs, err := e.emb.Embed(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("vectorstore: embed chunk: %w", err)
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil
	}
	vec := vecs[0]
	// Resolve the chunk id by position (the chunk the driver wrote).
	chunkID, err := e.chunkIDForIndex(fileID, chunkIdx)
	if err != nil || chunkID == 0 {
		return nil // no chunk (the position is out of range) → no vector
	}
	if err := e.st.EmbedChunk(ctx, chunkID, e.emb.Model(), vec); err != nil {
		return fmt.Errorf("vectorstore: write chunk vector: %w", err)
	}
	if annActive(e.emb) {
		e.dualWriteChunk(ctx, chunkID, vec)
	}
	return nil
}

// Search runs the preference order HNSW → vec0 → brute-force and returns the
// top-K results + the index used ("hnsw"/"vec0"/"brute_force"). A missing
// index falls to the next, never a hard fail (the brute-force floor always
// answers). On PG (no ANN tables) the order collapses to brute-force
// (degraded: brute_force).
func (e *Engine) Search(ctx context.Context, queryVec []float64, limit int) ([]Result, string, error) {
	if limit <= 0 {
		limit = 10
	}
	qv := toFloat32(queryVec)

	// 1. HNSW (SQLite-only; the dim must activate it).
	if e.emb != nil && annActive(e.emb) && !e.disabled["hnsw"] {
		if hits, err := e.hnswSearch(ctx, qv, limit); err == nil && len(hits) > 0 {
			return hits, "hnsw", nil
		}
		// HNSW missing/empty/degraded → fall to vec0.
	}
	// 2. vec0 (SQLite-only; the dim must activate it).
	if e.emb != nil && annActive(e.emb) && !e.disabled["vec0"] {
		if hits, err := e.vec0Search(ctx, qv, limit); err == nil && len(hits) > 0 {
			return hits, "vec0", nil
		}
		// vec0 missing/empty/degraded → fall to brute-force.
	}
	// 3. brute-force (the floor — always answers, SQLite + PG).
	return e.bruteForceSearch(ctx, qv, limit), "brute_force", nil
}

// CountVectors returns the number of stored vectors (the embeddings +
// chunk_embeddings rows). The test seam (the guard's clear assertion).
func (e *Engine) CountVectors() int {
	n, _ := e.countRows(ctx0(), `SELECT COUNT(*) FROM embeddings`)
	m, _ := e.countRows(ctx0(), `SELECT COUNT(*) FROM chunk_embeddings`)
	return n + m
}

// IndexBuilt reports whether the named ANN index is present + non-empty
// (the test seam). "hnsw" → the hnsw_nodes rows; "vec0" → the vec_symbols
// rows. A missing table (pre-migration) reports false.
func (e *Engine) IndexBuilt(name string) bool {
	switch name {
	case "hnsw":
		n, _ := e.countRows(ctx0(), `SELECT COUNT(*) FROM hnsw_nodes`)
		return n > 0
	case "vec0":
		n, _ := e.countRows(ctx0(), `SELECT COUNT(*) FROM vec_symbols`)
		return n > 0
	}
	return false
}

// ANNTablesDropped reports how many of the dim-shaped ANN tables were dropped
// (the M9 dim-change assertion). It returns the count of the annTables that
// are ABSENT (dropped). A PG store (no ANN tables) returns 0 (none present to
// drop). The test asserts the count is > 0 (the dim change dropped the
// tables) — not all of them, since the Engine may only populate a subset
// (hnsw_nodes, not hnsw_links).
func (e *Engine) ANNTablesDropped() int {
	rq, ok := e.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return 0
	}
	dropped := 0
	for _, t := range annTables {
		rows, err := rq.RawQuery(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, t)
		if err != nil {
			continue // a missing table-check seam (a PG store) → not counted
		}
		exists := false
		if len(rows) > 0 && len(rows[0]) > 0 {
			if n, ok := rows[0][0].(int64); ok {
				exists = n > 0
			}
		}
		if !exists {
			dropped++
		}
	}
	return dropped
}

// DisableIndexForTest disables the named ANN index (the test seam — disable
// HNSW/vec0 to force the fallback). "hnsw"/"vec0".
func (e *Engine) DisableIndexForTest(name string) {
	e.mu.Lock()
	e.disabled[name] = true
	e.mu.Unlock()
}

// ── the preference-order legs ───────────────────────────────────────────────

// hnswSearch runs the HNSW graph search (SQLite-only). It returns a nil
// slice (the caller falls to vec0) when the graph is absent/empty (the
// hnsw_nodes table is missing or has no rows).
func (e *Engine) hnswSearch(ctx context.Context, qv []float32, limit int) ([]Result, error) {
	rq, ok := e.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("hnsw: no RawQuery seam")
	}
	// Count the graph nodes (the warm-load trigger + the empty check).
	rows, err := rq.RawQuery(`SELECT COUNT(*) FROM hnsw_nodes`)
	if err != nil {
		return nil, err // the table is missing (pre-migration) → fall to vec0
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return nil, nil
	}
	if n, ok := rows[0][0].(int64); !ok || n == 0 {
		return nil, nil // empty graph → fall to vec0
	}
	// Warm-load the graph from the BLOB table (the in-memory HNSW, the v1
	// pattern — the graph is rebuilt in-memory over the BLOB source of truth).
	graph, err := e.loadHnswGraph(ctx, rq)
	if err != nil {
		return nil, err
	}
	if len(graph.nodes) == 0 {
		return nil, nil
	}
	graph.build()
	hits := graph.search(qv, limit)
	out := make([]Result, len(hits))
	for i, h := range hits {
		out[i] = Result{ID: h.id, Dist: 1 - h.sim, Sim: h.sim}
	}
	return out, nil
}

// vec0Search runs the vec0 virtual table search (SQLite-only). It returns a
// nil slice (the caller falls to brute-force) when the vec_symbols table is
// missing or empty (the vec0 vtab is absent pre-migration).
func (e *Engine) vec0Search(ctx context.Context, qv []float32, limit int) ([]Result, error) {
	rq, ok := e.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("vec0: no RawQuery seam")
	}
	qb := encodeFloat32BLOB(qv)
	rows, err := rq.RawQuery(
		`SELECT rowid, vec_distance_cosine(embedding, vec_f32(?)) FROM vec_symbols
		 ORDER BY vec_distance_cosine(embedding, vec_f32(?)) LIMIT ?`,
		qb, qb, limit)
	if err != nil {
		return nil, err // the table is missing (pre-migration) → fall to brute-force
	}
	out := make([]Result, 0, limit)
	for _, r := range rows {
		id, _ := r[0].(int64)
		dist, _ := r[1].(float64)
		out = append(out, Result{ID: id, Dist: dist, Sim: 1 - dist})
	}
	return out, nil
}

// bruteForceSearch is the floor (SQLite + PG): a cosine scan over the
// embeddings BLOB table (the source of truth). It always answers (an empty
// corpus → an empty result, not an error).
func (e *Engine) bruteForceSearch(ctx context.Context, qv []float32, limit int) []Result {
	rq, ok := e.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil
	}
	rows, err := rq.RawQuery(`SELECT symbol_id, vector FROM embeddings`)
	if err != nil {
		return nil
	}
	type scored struct {
		id  int64
		sim float64
	}
	var all []scored
	for _, r := range rows {
		id, _ := r[0].(int64)
		var blob []byte
		switch b := r[1].(type) {
		case []byte:
			blob = b
		default:
			continue
		}
		vec, ok := decodeFloat32BLOB(blob)
		if !ok || len(vec) != len(qv) {
			continue // a corrupt/dim-mismatched row is skipped (the floor tolerates it)
		}
		all = append(all, scored{id: id, sim: cosine(qv, vec)})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].sim != all[j].sim {
			return all[i].sim > all[j].sim
		}
		return all[i].id < all[j].id
	})
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([]Result, len(all))
	for i, s := range all {
		out[i] = Result{ID: s.id, Dist: 1 - s.sim, Sim: s.sim}
	}
	return out
}

// ── the SQLite ANN dual-write (HNSW/vec0, when the dim activates them) ─────

// dualWriteSymbol mirrors the symbol vector into the vec0 table + the HNSW
// graph (the v1 dual-write contract: the BLOB is the source of truth; the ANN
// tables mirror it). A dim-mismatch (the embedder's dim != the vec0 table's)
// skips the mirror (the v1 indexer's dimension guard).
func (e *Engine) dualWriteSymbol(ctx context.Context, symbolID int64, vec []float32) {
	re, ok := e.st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return
	}
	vb := encodeFloat32BLOB(vec)
	// vec0: the DELETE + INSERT (the vec0 vtab does not implement UPSERT).
	if !e.disabled["vec0"] {
		_ = re.RawExec(`DELETE FROM vec_symbols WHERE rowid=?`, symbolID)
		_ = re.RawExec(`INSERT INTO vec_symbols(rowid, embedding) VALUES (?, vec_f32(?))`, symbolID, vb)
	}
	// HNSW: the in-memory graph (the persistent hnsw_nodes is rebuilt on
	// warm-load from the BLOB — the v1 pattern; the in-memory graph is the
	// hot path).
	if !e.disabled["hnsw"] {
		e.hnswUpsert(symbolID, vec)
	}
}

// dualWriteChunk mirrors the chunk vector into vec_chunks (the HNSW graph is
// symbol-only — the semantic leg searches symbols, the v1 HnswChunks is the
// chunk partition but the Engine's semantic leg is the symbol leg).
func (e *Engine) dualWriteChunk(ctx context.Context, chunkID int64, vec []float32) {
	re, ok := e.st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return
	}
	vb := encodeFloat32BLOB(vec)
	if !e.disabled["vec0"] {
		_ = re.RawExec(`DELETE FROM vec_chunks WHERE rowid=?`, chunkID)
		_ = re.RawExec(`INSERT INTO vec_chunks(rowid, embedding) VALUES (?, vec_f32(?))`, chunkID, vb)
	}
}

// chunkIDForIndex resolves the chunk id by position (the chunk the driver
// wrote at chunkIdx for fileID).
func (e *Engine) chunkIDForIndex(fileID int64, chunkIdx int) (int64, error) {
	rq, ok := e.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return 0, fmt.Errorf("no RawQuery seam")
	}
	rows, err := rq.RawQuery(`SELECT id FROM chunks WHERE file_id = ? ORDER BY id`, fileID)
	if err != nil {
		return 0, err
	}
	if len(rows) <= chunkIdx || len(rows[chunkIdx]) == 0 {
		return 0, nil
	}
	id, _ := rows[chunkIdx][0].(int64)
	return id, nil
}

// ── the HNSW graph (the in-memory pure-Go HNSW, the v1 hnsw.go lift) ───────

// hnswGraph is the in-memory HNSW graph over the embeddings BLOB table (the
// v1 hnsw.go:26-37 params: M=16/M0=32/efConstruction=64/efSearch=96, ml=0.5).
// It is rebuilt in-memory from the BLOB on warm-load (the v1 ensureLoaded
// pattern — the BLOB is the source of truth, the graph is a derived index).
// A corrupt/empty graph degrades to brute force (never a hard fail).
type hnswGraph struct {
	nodes    map[int64]*hnswNode
	top      int64
	topLevel int
	dim      int
	M, M0    int
	efCon    int
	efSearch int
	ml       float64
}

// hnswNode is one in-memory graph node: the vector + per-layer adjacency.
type hnswNode struct {
	id   int64
	vec  []float32
	adj  [][]int64
	tomb bool
}

const (
	hnswM        = 16  // max connections per node at layers > 0 (the v1 ADR-0037)
	hnswM0       = 32  // max connections at layer 0 (2*M)
	hnswEfCon    = 64  // candidate beam during insert
	hnswEfSearch = 96  // candidate beam during query
	hnswML       = 0.5 // level distribution: P(level=l) = (1-ml)*ml^l
)

func newHnswGraph(dim int) *hnswGraph {
	return &hnswGraph{
		nodes:    map[int64]*hnswNode{},
		dim:      dim,
		M:        hnswM,
		M0:       hnswM0,
		efCon:    hnswEfCon,
		efSearch: hnswEfSearch,
		ml:       hnswML,
	}
}

// build connects the graph (the v1 connectInMemory, in insertion order — the
// deterministic build). It is idempotent (a built graph is not rebuilt).
func (g *hnswGraph) build() {
	if g.top != 0 || len(g.nodes) == 0 {
		return
	}
	// Insert in id order (deterministic — the v1 levelFor uses a process RNG;
	// here the level is deterministic from the id so the build is reproducible
	// across process restarts, the v1 warm-load contract).
	ids := make([]int64, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		g.insertNode(id)
	}
}

// insertNode adds a node + connects it to the graph (the v1 connectInMemory).
func (g *hnswGraph) insertNode(id int64) {
	node := g.nodes[id]
	lvl := g.levelFor(id)
	for len(node.adj) <= lvl {
		node.adj = append(node.adj, nil)
	}
	if len(g.nodes) == 1 {
		g.top = id
		g.topLevel = lvl
		return
	}
	g.connect(id, lvl)
	if lvl > g.topLevel {
		g.topLevel = lvl
	}
	if lvl > g.maxLevel(g.top) {
		g.top = id
	}
}

// levelFor returns the HNSW level for a node (deterministic from the id —
// the v1 levelFor's geometric distribution, made reproducible: a hash of the
// id seeds the geometric draw so a warm-load rebuilds the same graph).
func (g *hnswGraph) levelFor(id int64) int {
	h := uint64(id)
	l := 0
	for l < 32 {
		// A deterministic geometric draw (P(level=l) = (1-ml)*ml^l, ml=0.5).
		// The id's low bits seed the draw (reproducible across processes).
		h = h*6364136223846793005 + 1442695040888963407
		bit := (h >> 63) & 1
		if bit == 1 {
			break
		}
		l++
	}
	return l
}

// connect runs the v1 connectInMemory (the neighbor selection + the bidir
// links, capped at M/M0).
func (g *hnswGraph) connect(id int64, lvl int) {
	node := g.nodes[id]
	ep := g.top
	for l := g.topLevel; l > lvl; l-- {
		ep = g.greedyDescend(ep, node.vec, l)
	}
	for l := lvl; l >= 0; l-- {
		maxConn := g.M
		if l == 0 {
			maxConn = g.M0
		}
		res := g.searchLayer(ep, node.vec, l, g.efCon)
		kept := g.prune(id, res, maxConn)
		node.adj[l] = kept
		for _, k := range kept {
			nn := g.nodes[k]
			if nn == nil || nn.tomb || l >= len(nn.adj) {
				continue
			}
			nn.adj[l] = append(nn.adj[l], id)
			if len(nn.adj[l]) > maxConn {
				nn.adj[l] = g.prune(k, nn.adj[l], maxConn)
			}
		}
	}
}

// greedyDescend moves the entry point to the best neighbor at layer l (the
// v1 greedyDescend).
func (g *hnswGraph) greedyDescend(ep int64, q []float32, l int) int64 {
	cur := g.nodes[ep]
	if cur == nil {
		return ep
	}
	best := ep
	bestD := cosine(cur.vec, q)
	if l >= len(cur.adj) {
		for ll := len(cur.adj) - 1; ll >= 0; ll-- {
			if len(cur.adj[ll]) == 0 {
				continue
			}
			for _, n := range cur.adj[ll] {
				nn := g.nodes[n]
				if nn == nil || nn.tomb {
					continue
				}
				d := cosine(nn.vec, q)
				if d > bestD {
					best = n
					bestD = d
				}
			}
			break
		}
		return best
	}
	for _, n := range cur.adj[l] {
		nn := g.nodes[n]
		if nn == nil || nn.tomb {
			continue
		}
		d := cosine(nn.vec, q)
		if d > bestD {
			best = n
			bestD = d
		}
	}
	return best
}

// searchLayer runs the beam search at a single layer (the v1
// searchLayerInMem, the max-heap candidates + the min-heap results capped at
// k).
func (g *hnswGraph) searchLayer(ep int64, q []float32, l, k int) []int64 {
	if ep == 0 {
		return nil
	}
	epNode := g.nodes[ep]
	if epNode == nil || l >= len(epNode.adj) {
		return nil
	}
	visited := map[int64]struct{}{ep: {}}
	// candidates: best-first (max-heap, largest d first).
	cands := []hnswCand{{d: cosine(epNode.vec, q), id: ep}}
	// results: worst-first capped at k (min-heap, the root is the worst kept).
	results := []hnswCand{{d: cosine(epNode.vec, q), id: ep}}
	for len(cands) > 0 {
		c, rest := popMax(cands)
		cands = rest
		if len(results) >= k && c.d < minD(results) {
			break
		}
		for _, n := range g.nodes[c.id].adj[l] {
			if _, ok := visited[n]; ok {
				continue
			}
			visited[n] = struct{}{}
			nn := g.nodes[n]
			if nn == nil || nn.tomb {
				continue
			}
			d := cosine(nn.vec, q)
			cands = append(cands, hnswCand{d: d, id: n})
			if len(results) < k {
				results = append(results, hnswCand{d: d, id: n})
			} else if d > minD(results) {
				replaceMin(results, hnswCand{d: d, id: n})
			}
		}
	}
	// Drain worst-first + reverse to sim-desc.
	out := make([]hnswCand, 0, len(results))
	for len(results) > 0 {
		c, rest := popMin(results)
		out = append(out, c)
		results = rest
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	ids := make([]int64, len(out))
	for i, c := range out {
		ids[i] = c.id
	}
	return ids
}

// prune applies the HNSW neighbor-selection heuristic (alpha=1.0 diversity
// pruning, the v1 pruneInMem).
func (g *hnswGraph) prune(cur int64, ids []int64, maxConn int) []int64 {
	if len(ids) <= maxConn {
		return ids
	}
	cv := g.nodes[cur].vec
	type scored struct {
		id int64
		d  float64
	}
	var sc []scored
	for _, id := range ids {
		nn := g.nodes[id]
		if nn == nil || nn.tomb {
			continue
		}
		sc = append(sc, scored{id: id, d: cosine(nn.vec, cv)})
	}
	sort.Slice(sc, func(i, j int) bool { return sc[i].d > sc[j].d })
	cached := make(map[int64]map[int64]float64, len(sc))
	var kept []int64
	for _, s := range sc {
		if len(kept) == maxConn {
			break
		}
		m, ok := cached[s.id]
		if !ok {
			m = make(map[int64]float64, len(kept))
			cached[s.id] = m
		}
		okKeep := true
		for _, k := range kept {
			d, hit := m[k]
			if !hit {
				d = cosine(g.nodes[s.id].vec, g.nodes[k].vec)
				m[k] = d
			}
			if d > s.d {
				okKeep = false
				break
			}
		}
		if okKeep {
			kept = append(kept, s.id)
		}
	}
	for _, s := range sc {
		if len(kept) == maxConn {
			break
		}
		dup := false
		for _, k := range kept {
			if k == s.id {
				dup = true
				break
			}
		}
		if !dup {
			kept = append(kept, s.id)
		}
	}
	return kept
}

// maxLevel returns the highest layer index for a node.
func (g *hnswGraph) maxLevel(id int64) int {
	if id == 0 {
		return -1
	}
	nn := g.nodes[id]
	if nn == nil || len(nn.adj) == 0 {
		return -1
	}
	return len(nn.adj) - 1
}

// search runs the HNSW descent + the layer-0 beam (the v1 searchGraph).
func (g *hnswGraph) search(q []float32, limit int) []struct {
	id  int64
	sim float64
} {
	ep := g.top
	for l := g.topLevel; l >= 1; l-- {
		ep = g.greedyDescend(ep, q, l)
	}
	if g.nodes[ep].tomb {
		started := false
		for _, n := range g.nodes[ep].adj[0] {
			if !g.nodes[n].tomb {
				ep = n
				started = true
				break
			}
		}
		if !started {
			return nil
		}
	}
	ef := g.efSearch
	if ef < limit {
		ef = limit
	}
	ids := g.searchLayer(ep, q, 0, ef)
	hits := make([]struct {
		id  int64
		sim float64
	}, 0, len(ids))
	for _, id := range ids {
		hits = append(hits, struct {
			id  int64
			sim float64
		}{id: id, sim: cosine(g.nodes[id].vec, q)})
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// loadHnswGraph rebuilds the in-memory HNSW graph from the embeddings BLOB
// table (the v1 ensureLoaded pattern — the BLOB is the source of truth).
func (e *Engine) loadHnswGraph(ctx context.Context, rq interface {
	RawQuery(query string, args ...any) ([][]any, error)
}) (*hnswGraph, error) {
	rows, err := rq.RawQuery(`SELECT symbol_id, vector FROM embeddings`)
	if err != nil {
		return nil, err
	}
	var dim int
	for _, r := range rows {
		var blob []byte
		switch b := r[1].(type) {
		case []byte:
			blob = b
		default:
			continue
		}
		vec, ok := decodeFloat32BLOB(blob)
		if !ok || len(vec) == 0 {
			continue
		}
		if dim == 0 {
			dim = len(vec)
		}
	}
	g := newHnswGraph(dim)
	for _, r := range rows {
		id, _ := r[0].(int64)
		var blob []byte
		switch b := r[1].(type) {
		case []byte:
			blob = b
		default:
			continue
		}
		vec, ok := decodeFloat32BLOB(blob)
		if !ok || len(vec) == 0 {
			continue
		}
		g.nodes[id] = &hnswNode{id: id, vec: vec}
	}
	return g, nil
}

// hnswUpsert adds a node to the HNSW graph (the dual-write hot path): the
// in-memory graph node + the hnsw_nodes row (the persistence — the warm-load
// rebuilds the graph from it on the next process, the v1 ensureLoaded
// pattern). The in-memory graph is built lazily (the first search triggers
// the connect).
func (e *Engine) hnswUpsert(id int64, vec []float32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hnsw == nil {
		e.hnsw = newHnswGraph(len(vec))
	}
	if _, ok := e.hnsw.nodes[id]; !ok {
		e.hnsw.nodes[id] = &hnswNode{id: id, vec: vec}
	} else {
		e.hnsw.nodes[id].vec = vec
		e.hnsw.nodes[id].tomb = false
	}
	// Persist the node to hnsw_nodes (the warm-load's source of truth). The
	// level is the deterministic geometric draw (the build's levelFor).
	lvl := e.hnsw.levelFor(id)
	re, ok := e.st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return
	}
	vb := encodeFloat32BLOB(vec)
	_ = re.RawExec(`INSERT OR REPLACE INTO hnsw_nodes (source, node_id, level, vector) VALUES ('symbol', ?, ?, ?)`, id, lvl, vb)
}

// ── the vector codec + cosine (the store's little-endian float32 BLOB) ─────

// encodeFloat32BLOB encodes a vector as the store's little-endian float32
// BLOB (the spec 006 §4.1 encoding — mirrors the store's encodeFloat32BLOB).
func encodeFloat32BLOB(v []float32) []byte {
	b := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

// decodeFloat32BLOB decodes the store's little-endian float32 BLOB.
func decodeFloat32BLOB(b []byte) ([]float32, bool) {
	if len(b) == 0 || len(b)%4 != 0 {
		return nil, false
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v, true
}

// cosine is the cosine similarity (dot/norm; 0 on a dim mismatch / zero vec).
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		d := float64(a[i])
		e := float64(b[i])
		dot += d * e
		na += d * d
		nb += e * e
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// toFloat32 converts a []float64 query vector to []float32 (the query leg's
// type — the embedder's Embed returns []float32, the Search query is
// []float64 per the brief).
func toFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, f := range v {
		out[i] = float32(f)
	}
	return out
}

// countRows returns the COUNT(*) of a query (the test seam's counter). A
// missing table returns 0 (the seam's absence is a no-op).
func (e *Engine) countRows(ctx context.Context, q string) (int, error) {
	rq, ok := e.st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return 0, nil
	}
	rows, err := rq.RawQuery(q)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0, nil
	}
	n, _ := rows[0][0].(int64)
	return int(n), nil
}

// ctx0 is the no-context for the test-seam counters (the seams do no I/O
// that needs a context).
func ctx0() context.Context { return context.Background() }

// hnswCand is one candidate in the HNSW beam search (sim + id).
type hnswCand struct {
	d  float64
	id int64
}

// popMax removes + returns the largest-d candidate (the best-first expansion).
func popMax(cands []hnswCand) (hnswCand, []hnswCand) {
	maxI := 0
	for i := 1; i < len(cands); i++ {
		if cands[i].d > cands[maxI].d {
			maxI = i
		}
	}
	out := cands[maxI]
	cands[maxI] = cands[len(cands)-1]
	return out, cands[:len(cands)-1]
}

func minD(c []hnswCand) float64 {
	minI := 0
	for i := 1; i < len(c); i++ {
		if c[i].d < c[minI].d {
			minI = i
		}
	}
	return c[minI].d
}

func popMin(c []hnswCand) (hnswCand, []hnswCand) {
	minI := 0
	for i := 1; i < len(c); i++ {
		if c[i].d < c[minI].d {
			minI = i
		}
	}
	out := c[minI]
	c[minI] = c[len(c)-1]
	return out, c[:len(c)-1]
}

func replaceMin(c []hnswCand, x hnswCand) {
	minI := 0
	for i := 1; i < len(c); i++ {
		if c[i].d < c[minI].d {
			minI = i
		}
	}
	c[minI] = x
}
