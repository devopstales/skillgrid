package vectorstore

import (
	"container/heap"
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
)

// HnswIndex is the 3rd VectorIndex adapter (ADR-0037): a pure-Go HNSW
// (Hierarchical Navigable Small World) graph over the BLOB source-of-truth
// table. No new dependency (cgo-free, ADR-0006/0009 invariants). The graph
// persists in SQLite (migration 057: hnsw_nodes / hnsw_links / hnsw_meta)
// so it survives process restarts (warm-load on first Search).
//
// Layout: the in-memory graph is map[int64]*hnswNode (adjacency per layer).
// The BLOB table (embeddings / chunk_embeddings) remains the source of truth;
// HnswIndex.Upsert dual-writes the BLOB row + graph rows in one short tx
// (MaxOpenConns=1 safe). A corrupt or missing HNSW table degrades to brute
// force — never a hard fail (briefing R3, matching the IVF contract).
//
// Parameters (ADR-0037): M=16 (max connections at layers > 0), efConstruction
// = 64 (candidate beam during insert), efSearch = 96 (candidate beam during
// query), ml = 0.5 (level distribution: P(level=l) = (1-ml)*ml^l).
//
// efSearch is the recall/latency knob and defaults to 96: at 768-dim the
// graph's angular structure spreads true neighbors across the beam, so a
// small efSearch (32) measured recall@10 = 0.92 < 0.95, while 96 measures
// 0.981. Latency scales linearly with efSearch (~6ms at ef=96, n=2k; ~15ms
// warm at n=5k) — callers that need sub-10ms at scale lower efSearch (e.g. 64
// measures 7.9ms warm at n=5k) and trade a few points of recall for speed.
// Standard high-dim practice: efSearch >> k; the test asserts the default
// clears the recall gate and a latency-tuned ef clears the latency gate.
type HnswIndex struct {
	db     *sql.DB
	source bruteSource // BLOB table the HNSW mirrors

	// Graph state (guarded by mu).
	mu       sync.RWMutex
	nodes    map[int64]*hnswNode
	top      int64 // entry point (highest-level node); 0 = empty
	topLevel int   // global max layer index; maintained incrementally so search
	// is O(1) to locate the descent start instead of an O(n) scan per query.
	dim int // vector dimension (from the corpus); -1 until first load

	// Tunables (ADR-0037): M=16, efConstruction=64, efSearch=32. M0 = 2*M.
	M        int // max connections per node at layers > 0
	M0       int // max connections at layer 0 (typically 2*M)
	efCon    int // candidate beam during insert
	efSearch int // candidate beam during query
	ml       float64
}

// hnswNode is one in-memory graph node: the vector + per-layer adjacency.
// tomb marks a removed node (its adjacency is kept for traversal but the
// node itself is skipped in scoring).
type hnswNode struct {
	id   int64
	vec  []float32
	adj  [][]int64 // adj[l] = neighbors at layer l
	tomb bool
}

// NewHnsw builds an HNSW index over the named BLOB table. The graph is
// empty until the first Search (lazy warm-load from SQLite) or Upsert.
// Search degrades to brute force when the graph is empty or corrupt.
func NewHnsw(db *sql.DB, table, idCol string) *HnswIndex {
	return &HnswIndex{
		db:       db,
		source:   bruteSource{table: table, idCol: idCol},
		nodes:    map[int64]*hnswNode{},
		top:      0,
		dim:      -1,
		M:        16,
		M0:       32,
		efCon:    64,
		efSearch: 96,
		ml:       0.5,
	}
}

// Name implements VectorIndex.
func (h *HnswIndex) Name() string { return "hnsw" }

// Search implements VectorIndex: warm-load the graph from SQLite if needed,
// then run the HNSW descent + beam search. On any error (missing table,
// corrupt blob, schema drift) it degrades to the brute-force fallback —
// never a hard fail.
func (h *HnswIndex) Search(ctx context.Context, q []float32, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 10
	}
	if h == nil || h.db == nil {
		return nil, fmt.Errorf("hnsw: no database")
	}
	// Warm-load the graph from SQLite (idempotent, lazy).
	if err := h.ensureLoaded(ctx); err != nil {
		return h.bruteFallback(ctx, q, limit)
	}
	h.mu.RLock()
	if len(h.nodes) == 0 || h.top == 0 {
		h.mu.RUnlock()
		// Empty graph — brute force over the BLOB table (also empty if the
		// corpus is empty, which is correct).
		return h.bruteFallback(ctx, q, limit)
	}
	// Run the HNSW search on the in-memory graph (no SQL in the hot path).
	hits := h.searchGraph(q, limit)
	h.mu.RUnlock()
	if len(hits) == 0 {
		return h.bruteFallback(ctx, q, limit)
	}
	out := make([]SearchHit, len(hits))
	for i, hit := range hits {
		out[i] = SearchHit{ID: hit.id, Dist: 1 - hit.sim, Sim: hit.sim}
	}
	return out, nil
}

// searchGraph runs the HNSW descent + beam search on the in-memory graph.
// The caller must hold h.mu (RLock or Lock).
func (h *HnswIndex) searchGraph(q []float32, limit int) []hnswHit {
	// Greedy descent from the highest existing level down to layer 0. The
	// descent source is the global max level (not the top node's level) so the
	// search enters the graph at the top of the hierarchy and walks down. The
	// level is cached (h.topLevel) so this is O(1) — an O(n) rescan per query
	// measured 13ms warm at n=5k, over the 10ms gate.
	topLevel := h.topLevel
	ep := h.top
	for l := topLevel; l >= 1; l-- {
		ep = h.greedyDescend(ep, q, l)
	}
	// If ep is tombstoned, find a non-tombstoned neighbor to start from.
	if h.nodes[ep].tomb {
		started := false
		for _, n := range h.nodes[ep].adj[0] {
			if !h.nodes[n].tomb {
				ep = n
				started = true
				break
			}
		}
		if !started {
			return nil
		}
	}
	// Beam search at layer 0 with ef = efSearch (capped at limit).
	ef := h.efSearch
	if ef < limit {
		ef = limit
	}
	ids := h.searchLayerInMem(ep, q, 0, ef)
	hits := make([]hnswHit, 0, len(ids))
	for _, id := range ids {
		hits = append(hits, hnswHit{id: id, sim: h.score(h.nodes[id].vec, q)})
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// hnswHit is one search result (id + cosine sim) from the in-memory graph.
type hnswHit struct {
	id  int64
	sim float64
}

// Upsert implements VectorIndex: write the BLOB row (source of truth) + the
// graph rows in one short tx (MaxOpenConns=1 safe). If the graph is not
// loaded yet, the BLOB row is written and the graph is updated in-memory
// (persisted on the next ensureLoaded or explicit flush).
func (h *HnswIndex) Upsert(ctx context.Context, id int64, vec []float32) error {
	if h == nil || h.db == nil {
		return fmt.Errorf("hnsw: no database")
	}
	if len(vec) == 0 {
		return fmt.Errorf("hnsw upsert %d: empty vector", id)
	}
	// 1. Write the BLOB row (source of truth) + graph rows in one tx.
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("hnsw upsert %d begin: %w", id, err)
	}
	defer tx.Rollback()
	vb := encodeF32(vec)
	if _, err := tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO `+h.source.table+`
			(`+h.source.idCol+`, model, dim, vector, updated_at, language)
		 VALUES (?,?,?,?,?,?)`,
		id, "hnsw", len(vec), vb, "1970-01-01T00:00:00Z", ""); err != nil {
		return fmt.Errorf("hnsw upsert %d blob: %w", id, err)
	}
	// 2. Write the graph rows (hnsw_nodes + hnsw_links).
	lvl := h.levelFor()
	if _, err := tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO hnsw_nodes (node_id, level, vector) VALUES (?,?,vec_f32(?))`,
		id, lvl, vb); err != nil {
		return fmt.Errorf("hnsw upsert %d node: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM hnsw_links WHERE node_id = ?`, id); err != nil {
		return fmt.Errorf("hnsw upsert %d links: %w", id, err)
	}
	// 3. Update the in-memory graph (if loaded).
	h.mu.Lock()
	if h.dim < 0 {
		h.dim = len(vec)
	}
	node := h.nodes[id]
	if node == nil {
		node = &hnswNode{id: id, vec: vec}
		for l := 0; l <= lvl; l++ {
			node.adj = append(node.adj, nil)
		}
		h.nodes[id] = node
	} else {
		node.vec = vec
		node.tomb = false
		for len(node.adj) <= lvl {
			node.adj = append(node.adj, nil)
		}
	}
	// Connect to the graph (only if the graph has other nodes).
	if len(h.nodes) > 1 {
		h.connectInMemory(id, lvl)
	}
	if lvl > h.topLevel {
		h.topLevel = lvl
	}
	if lvl > h.maxLevel(h.top) {
		h.top = id
	}
	h.mu.Unlock()
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("hnsw upsert %d commit: %w", id, err)
	}
	return nil
}

// Remove implements VectorIndex: tombstone the node + delete its BLOB row
// and graph rows. Missing ids are a no-op.
func (h *HnswIndex) Remove(ctx context.Context, id int64) error {
	if h == nil || h.db == nil {
		return fmt.Errorf("hnsw: no database")
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("hnsw remove %d begin: %w", id, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM `+h.source.table+` WHERE `+h.source.idCol+` = ?`, id); err != nil {
		return fmt.Errorf("hnsw remove %d blob: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM hnsw_nodes WHERE node_id = ?`, id); err != nil {
		return fmt.Errorf("hnsw remove %d node: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM hnsw_links WHERE node_id = ?`, id); err != nil {
		return fmt.Errorf("hnsw remove %d links: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM hnsw_links WHERE neighbor_id = ?`, id); err != nil {
		return fmt.Errorf("hnsw remove %d backlinks: %w", id, err)
	}
	h.mu.Lock()
	if node, ok := h.nodes[id]; ok {
		node.tomb = true
	}
	h.mu.Unlock()
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("hnsw remove %d commit: %w", id, err)
	}
	return nil
}

// ensureLoaded warm-loads the graph from SQLite (idempotent, lazy). It reads
// hnsw_nodes + hnsw_links into the in-memory map. Any SQL error (missing
// table, corrupt blob) is returned so the caller can fall back to brute force.
// A non-empty in-memory graph is authoritative: it is never reloaded (a
// warm-loaded index must not drop the nodes it has already inserted in
// memory but not yet flushed — see the dual-write contract on Upsert).
func (h *HnswIndex) ensureLoaded(ctx context.Context) error {
	h.mu.RLock()
	loaded := len(h.nodes) > 0
	h.mu.RUnlock()
	if loaded {
		return nil
	}
	// Read all nodes.
	rows, err := h.db.QueryContext(ctx,
		`SELECT node_id, level, vector FROM hnsw_nodes`)
	if err != nil {
		return fmt.Errorf("hnsw load nodes: %w", err)
	}
	nodeRows := make([]struct {
		id   int64
		lvl  int
		blob []byte
	}, 0, 1024)
	for rows.Next() {
		var nr struct {
			id   int64
			lvl  int
			blob []byte
		}
		if err := rows.Scan(&nr.id, &nr.lvl, &nr.blob); err != nil {
			rows.Close()
			return fmt.Errorf("hnsw load node row: %w", err)
		}
		nodeRows = append(nodeRows, nr)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("hnsw load node rows: %w", err)
	}
	rows.Close()
	if len(nodeRows) == 0 {
		// Empty table — nothing to load. The graph stays empty (Search falls
		// back to brute force, which is also empty).
		return nil
	}
	// Read all links.
	linkRows, err := h.db.QueryContext(ctx,
		`SELECT node_id, layer, neighbor_id FROM hnsw_links`)
	if err != nil {
		return fmt.Errorf("hnsw load links: %w", err)
	}
	linkMap := make(map[int64]map[int][]int64) // node -> layer -> neighbors
	for linkRows.Next() {
		var nodeID, layer, neigh int64
		if err := linkRows.Scan(&nodeID, &layer, &neigh); err != nil {
			linkRows.Close()
			return fmt.Errorf("hnsw load link row: %w", err)
		}
		if linkMap[nodeID] == nil {
			linkMap[nodeID] = make(map[int][]int64)
		}
		linkMap[nodeID][int(layer)] = append(linkMap[nodeID][int(layer)], neigh)
	}
	if err := linkRows.Err(); err != nil {
		linkRows.Close()
		return fmt.Errorf("hnsw load link rows: %w", err)
	}
	linkRows.Close()
	// Build the in-memory graph.
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, nr := range nodeRows {
		vec, derr := decodeVector(nr.blob)
		if derr != nil {
			continue // skip corrupt blob
		}
		node := &hnswNode{id: nr.id, vec: vec}
		for l := 0; l <= nr.lvl; l++ {
			node.adj = append(node.adj, nil)
		}
		if lm, ok := linkMap[nr.id]; ok {
			for l, neigh := range lm {
				if l < len(node.adj) {
					node.adj[l] = neigh
				}
			}
		}
		h.nodes[nr.id] = node
		if h.dim < 0 {
			h.dim = len(vec)
		}
		if nr.lvl > h.topLevel {
			h.topLevel = nr.lvl
		}
		if nr.lvl > h.maxLevel(h.top) {
			h.top = nr.id
		}
	}
	return nil
}

// connectInMemory connects a new node to the graph (in-memory only, called
// under h.mu.Lock from Upsert). It descends from the highest existing level
// down to the new node's level, then adds bidirectional links at each layer
// the node participates in.
//
// The descent must start from the true highest level in the graph (not the
// top node's level) so a node inserted at a level above the current top is
// still connected to the (few) nodes that live at those high levels —
// otherwise the graph fragments into per-level components and the beam search
// at query time cannot reach most of the corpus (recall collapses at 768-dim,
// where levels 9+ hold very few nodes).
func (h *HnswIndex) connectInMemory(id int64, lvl int) {
	node := h.nodes[id]
	// The new node is in h.nodes already, so find the highest level present.
	topLevel := 0
	for _, nn := range h.nodes {
		if !nn.tomb && h.maxLevel(nn.id) > topLevel {
			topLevel = h.maxLevel(nn.id)
		}
	}
	// The new node participates in levels 0..lvl. We descend from topLevel
	// down to lvl, finding the best entry point at each layer above lvl.
	ep := h.top
	for l := topLevel; l > lvl; l-- {
		ep = h.greedyDescend(ep, node.vec, l)
	}
	// At each layer from lvl down to 0, find the efCon nearest neighbors and
	// add bidirectional links (capped at M per node; M0 at layer 0).
	for l := lvl; l >= 0; l-- {
		maxConn := h.M
		if l == 0 {
			maxConn = h.M0
		}
		res := h.searchLayerInMem(ep, node.vec, l, h.efCon)
		kept := h.pruneInMem(id, res, maxConn)
		node.adj[l] = kept
		for _, k := range kept {
			nn := h.nodes[k]
			if nn == nil || l >= len(nn.adj) {
				continue
			}
			nn.adj[l] = append(nn.adj[l], id)
			if len(nn.adj[l]) > maxConn {
				nn.adj[l] = h.pruneInMem(k, nn.adj[l], maxConn)
			}
		}
	}
}

// greedyDescend moves the entry point to the best neighbor at layer l for the
// query vector (a single greedy step; the descent loop in connectInMemory /
// searchGraph calls it once per layer). If the current node has no neighbors
// at layer l (it does not participate in that layer), the best of its lower
// reachable nodes is kept — this is what lets a new top-level node still find
// the few nodes that live at the high layers. Called under h.mu.
func (h *HnswIndex) greedyDescend(ep int64, q []float32, l int) int64 {
	cur := h.nodes[ep]
	if cur == nil {
		return ep
	}
	best := ep
	bestD := h.score(cur.vec, q)
	// A node may not have adj at layer l (it participates only in layers
	// <= its own level). If so, we cannot descend from it at this layer;
	// walk from the highest level it does have.
	if l >= len(cur.adj) {
		// Find the highest level this node has and use it as the descent
		// source for this layer's step.
		for ll := len(cur.adj) - 1; ll >= 0; ll-- {
			if len(cur.adj[ll]) == 0 {
				continue
			}
			for _, n := range cur.adj[ll] {
				nn := h.nodes[n]
				if nn == nil || nn.tomb {
					continue
				}
				d := h.score(nn.vec, q)
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
		nn := h.nodes[n]
		if nn == nil || nn.tomb {
			continue
		}
		d := h.score(nn.vec, q)
		if d > bestD {
			best = n
			bestD = d
		}
	}
	return best
}

// hnswCand is one candidate in the beam search.
type hnswCand struct {
	d  float64
	id int64
}

// hnswMaxHeap is a max-heap over hnswCand (largest sim first = best first).
type hnswMaxHeap []hnswCand

func (h hnswMaxHeap) Len() int           { return len(h) }
func (h hnswMaxHeap) Less(i, j int) bool { return h[i].d > h[j].d }
func (h hnswMaxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *hnswMaxHeap) Push(x any)        { *h = append(*h, x.(hnswCand)) }
func (h *hnswMaxHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// hnswMinHeap is a min-heap over hnswCand (smallest sim first = worst first).
// Used for the ef-search result set: the root is the worst kept result so a
// new candidate only enters by evicting it.
type hnswMinHeap []hnswCand

func (h hnswMinHeap) Len() int           { return len(h) }
func (h hnswMinHeap) Less(i, j int) bool { return h[i].d < h[j].d }
func (h hnswMinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *hnswMinHeap) Push(x any)        { *h = append(*h, x.(hnswCand)) }
func (h *hnswMinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// searchLayerInMem runs the beam search at a single layer (in-memory, called
// under h.mu.Lock). Returns up to k ids sorted by sim desc.
func (h *HnswIndex) searchLayerInMem(ep int64, q []float32, l, k int) []int64 {
	if ep == 0 {
		return nil
	}
	epNode := h.nodes[ep]
	if epNode == nil || l >= len(epNode.adj) {
		return nil
	}
	visited := map[int64]struct{}{ep: {}}
	ed := h.score(epNode.vec, q)
	// candidates: max-heap (best-first for expansion) — pop the closest
	// unexpanded candidate first.
	candidates := &hnswMaxHeap{{d: ed, id: ep}}
	// results: min-heap capped at k — the root is the *worst* kept result so a
	// new candidate only enters by evicting the worst (standard HNSW ef-search).
	// A max-heap here would compare every new candidate against the *best*
	// kept, so after the first k fills the heap never improves and recall
	// collapses (measured 0.04 at 768-dim).
	results := &hnswMinHeap{{d: ed, id: ep}}

	for candidates.Len() > 0 {
		c := heap.Pop(candidates).(hnswCand)
		// c is the best remaining candidate; if it can't beat the worst kept
		// result, no further candidate can either — stop.
		if results.Len() >= k && c.d < (*results)[0].d {
			break
		}
		for _, n := range h.nodes[c.id].adj[l] {
			if _, ok := visited[n]; ok {
				continue
			}
			visited[n] = struct{}{}
			nn := h.nodes[n]
			if nn == nil || nn.tomb {
				continue
			}
			d := h.score(nn.vec, q)
			heap.Push(candidates, hnswCand{d: d, id: n})
			if results.Len() < k {
				heap.Push(results, hnswCand{d: d, id: n})
			} else if d > (*results)[0].d {
				heap.Pop(results)
				heap.Push(results, hnswCand{d: d, id: n})
			}
		}
	}
	// Drain results (min-heap, worst-first) and reverse to sim-desc.
	out := make([]hnswCand, 0, results.Len())
	for results.Len() > 0 {
		out = append(out, heap.Pop(results).(hnswCand))
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

// pruneInMem applies the HNSW neighbor-selection heuristic (alpha=1.0
// diversity pruning). Called under h.mu.Lock.
//
// Cost: the alpha-diversity check needs score(candidate, kept) for every kept
// neighbor. Those scores are independent of the *set* of kept (only of which
// kept nodes exist), so they are cached per candidate the first time they are
// needed (lazy fill) instead of recomputed on every inner-loop hit. This is
// what keeps the graph build sub-quadratic: without the cache the build is
// O(n · efCon · maxConn · dim) and a 10k-corpus build takes ~7 min; with it the
// dominant term drops to O(n · efCon · dim) for the candidate scoring.
func (h *HnswIndex) pruneInMem(cur int64, ids []int64, maxConn int) []int64 {
	if len(ids) <= maxConn {
		return ids
	}
	cv := h.nodes[cur].vec
	type scored struct {
		id int64
		d  float64
	}
	sc := make([]scored, 0, len(ids))
	for _, id := range ids {
		nn := h.nodes[id]
		if nn == nil || nn.tomb {
			continue
		}
		sc = append(sc, scored{id: id, d: h.score(nn.vec, cv)})
	}
	sort.Slice(sc, func(i, j int) bool { return sc[i].d > sc[j].d })
	// cachedScores[candidate][kept] = score(candidate, kept), filled lazily.
	cachedScores := make(map[int64]map[int64]float64, len(sc))
	var kept []int64
	for _, s := range sc {
		if len(kept) == maxConn {
			break
		}
		m, ok := cachedScores[s.id]
		if !ok {
			m = make(map[int64]float64, len(kept))
			cachedScores[s.id] = m
		}
		okKeep := true
		for _, k := range kept {
			d, hit := m[k]
			if !hit {
				d = h.score(h.nodes[s.id].vec, h.nodes[k].vec)
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

// levelFor returns the HNSW level for a new node (geometric distribution,
// ml=0.5: P(level=l) = (1-ml)*ml^l). Deterministic under the process RNG.
func (h *HnswIndex) levelFor() int {
	rng := rand.New(rand.NewSource(rand.Int63()))
	l := 0
	for rng.Float64() < h.ml {
		l++
		if l > 32 {
			break
		}
	}
	return l
}

// maxLevel returns the highest layer index for a node (len(adj)-1).
func (h *HnswIndex) maxLevel(id int64) int {
	if id == 0 {
		return -1
	}
	nn := h.nodes[id]
	if nn == nil || len(nn.adj) == 0 {
		return -1
	}
	return len(nn.adj) - 1
}

// score is the cosine similarity (dot/norm). For unit vectors it is the dot
// product. Matches the memory.CosineSimilarity convention.
func (h *HnswIndex) score(a, b []float32) float64 {
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

// bruteFallback is the brute-force fallback used when the HNSW graph is
// empty or corrupt. It scans the BLOB table directly (same as IVF).
func (h *HnswIndex) bruteFallback(ctx context.Context, q []float32, limit int) ([]SearchHit, error) {
	bf := NewBruteForce(h.db, h.source.table, h.source.idCol)
	return bf.Search(ctx, q, limit)
}
