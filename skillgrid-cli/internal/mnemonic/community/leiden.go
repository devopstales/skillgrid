// Package community detects subsystems in the 005 symbols/edges graph:
// seeded Leiden partitioning (bluuewhale/loom), degree-ranked god nodes, and
// LLM-free community labels. The partition is advisory, never load-bearing.
package community

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bluuewhale/loom/graph"
)

// Options tunes the community pass.
type Options struct {
	// Seed is the RNG seed for the Leiden detector. A fixed non-zero seed makes
	// the partition reproducible; 0 lets loom run its multi-run best-Q mode.
	Seed int64
	// Resolution is the modularity resolution (default 1).
	Resolution float64
	// MaxIterations bounds the Leiden refinement loop (0 = unlimited).
	MaxIterations int
	// NumRuns selects loom's multi-run best-Q selection when Seed == 0.
	NumRuns int
	// Limit caps the number of god nodes attached per community (0 = 10).
	Limit int
}

// Community is one detected subsystem: its members (symbol ids) and its
// LLM-free label (derived from god nodes + paths, never fabricated). HubLabel
// is the top god-node name ("" when the community has no god nodes). Cohesion
// (038) is the internal-edge density internal_edges / C(n,2) — a graphify
// report signal: 0 for a singleton/no-internal-edge community, 1 for a
// fully-connected clique. It is computed over the SAME undirected,
// self-loop-free edge set loadGraph builds, so it is comparable to the
// partition's own connectivity.
type Community struct {
	ID       int      `json:"id"`
	Label    string   `json:"label"`
	Members  []int64  `json:"members"`
	GodNodes []string `json:"god_nodes,omitempty"`
	HubLabel string   `json:"hub_label,omitempty"`
	Cohesion *float64 `json:"cohesion,omitempty"`
}

// Result is the Detect output: the communities, the stable content-hash cache
// key, whether the pass was served from cache, and any non-fatal warnings.
type Result struct {
	Communities []Community `json:"communities"`
	CacheKey    string      `json:"cache_key"`
	FromCache   bool        `json:"from_cache"`
	Warning     string      `json:"warning,omitempty"`
	Modularity  float64     `json:"modularity,omitempty"`
	Updated     time.Time   `json:"updated_at"`
}

// nodeRef is a symbol node in the partition: its symbol id plus the registry
// key (uid preferred, else name) used to map it through loom.
type nodeRef struct {
	symbolID int64
	key      string
}

// loadGraph loads the 005 symbols/edges tables as a registry + undirected
// graph. Edges are treated undirected for community detection (a call
// relationship clusters the same either way). Only edges whose both endpoints
// are live symbols (to_id set) build connectivity; name-only dangling edges
// are ignored (they would fabricate nodes).
func loadGraph(db *sql.DB) (*graph.NodeRegistry, *graph.Graph, []nodeRef, error) {
	reg := graph.NewRegistry()
	g := graph.NewGraph(false)

	rows, err := db.Query(`SELECT id, uid, name FROM symbols ORDER BY id`)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	var nodes []nodeRef
	idToLoom := map[int64]graph.NodeID{}
	for rows.Next() {
		var id int64
		var uid, name sql.NullString
		if err := rows.Scan(&id, &uid, &name); err != nil {
			return nil, nil, nil, err
		}
		key := uid.String
		if key == "" {
			key = name.String
		}
		if key == "" {
			key = fmt.Sprintf("sym-%d", id)
		}
		nid := reg.Register(key)
		idToLoom[id] = nid
		g.AddNode(nid, 1.0)
		nodes = append(nodes, nodeRef{symbolID: id, key: key})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	eRows, err := db.Query(`SELECT from_id, to_id FROM edges WHERE to_id IS NOT NULL`)
	if err != nil {
		return nil, nil, nil, err
	}
	defer eRows.Close()
	seen := map[[2]graph.NodeID]bool{}
	for eRows.Next() {
		var fromID, toID int64
		if err := eRows.Scan(&fromID, &toID); err != nil {
			return nil, nil, nil, err
		}
		f, ok1 := idToLoom[fromID]
		to, ok2 := idToLoom[toID]
		if !ok1 || !ok2 || f == to {
			continue
		}
		pair := [2]graph.NodeID{f, to}
		if pair[1] < pair[0] {
			pair = [2]graph.NodeID{to, f}
		}
		if seen[pair] {
			continue
		}
		seen[pair] = true
		g.AddEdge(f, to, 1.0)
	}
	if err := eRows.Err(); err != nil {
		return nil, nil, nil, err
	}
	return reg, g, nodes, nil
}

// contentKey computes a genuine content hash over the deterministic graph row
// set (the sorted edge set AND the symbol id set) so the cache key is stable
// across index runs with the same graph and changes whenever the graph
// changes — including a changed intermediate edge (same min/max/count) or a
// symbol delete/rename that preserves the edge count. The hash is computed in
// Go (modernc.org/sqlite has no sha1() SQL function), not a min/max/count summary.
func contentKey(db *sql.DB) (string, error) {
	h := sha1.New()

	rows, err := db.Query(`SELECT from_id, to_id FROM edges WHERE to_id IS NOT NULL ORDER BY from_id, to_id`)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var from, to int64
		if err := rows.Scan(&from, &to); err != nil {
			rows.Close()
			return "", err
		}
		_, _ = h.Write([]byte(strconv.FormatInt(from, 10) + ":" + strconv.FormatInt(to, 10) + ";"))
	}
	serr := rows.Err()
	rows.Close()
	if serr != nil {
		return "", serr
	}
	// NUL separator so a trailing-edge ambiguity can't collide with the symbol set.
	_, _ = h.Write([]byte{0})

	symRows, err := db.Query(`SELECT id FROM symbols ORDER BY id`)
	if err != nil {
		return "", err
	}
	for symRows.Next() {
		var id int64
		if err := symRows.Scan(&id); err != nil {
			symRows.Close()
			return "", err
		}
		_, _ = h.Write([]byte(strconv.FormatInt(id, 10) + ","))
	}
	symErr := symRows.Err()
	symRows.Close()
	if symErr != nil {
		return "", symErr
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Detect runs the community pass, consulting the community_meta cache first.
// A matching content-hash cache key returns the stored partition
// (FromCache=true) without re-running the detector; otherwise the seeded
// Leiden pass runs and the result is cached.
func Detect(ctx context.Context, db *sql.DB, opts Options) (*Result, error) {
	if opts.Seed == 0 {
		opts.Seed = 1
	}
	if opts.Resolution <= 0 {
		opts.Resolution = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key, err := contentKey(db)
	if err != nil {
		return nil, fmt.Errorf("content key: %w", err)
	}
	cacheKey := hex.EncodeToString([]byte("comm:" + key))

	if cached, err := cachedResult(db, cacheKey); err == nil && cached != nil {
		cached.FromCache = true
		return cached, nil
	}

	res, err := detectCore(db, opts, cacheKey)
	if err != nil {
		return nil, err
	}
	if err := writeMeta(db, res); err != nil {
		return res, fmt.Errorf("detect ok but cache write failed: %w", err)
	}
	return res, nil
}

// detectCore runs the seeded Leiden pass, assigns labels, and writes the
// communities table (the 012 additive table; 005 symbols/edges untouched).
func detectCore(db *sql.DB, opts Options, cacheKey string) (*Result, error) {
	res := &Result{CacheKey: cacheKey, Updated: time.Now().UTC()}

	var symCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&symCount); err != nil {
		return nil, err
	}

	if symCount < 2 {
		// warn+continue: one trivial community, never a crash.
		res.Warning = fmt.Sprintf("graph has %d symbols (<2); producing a single trivial community", symCount)
		ids, err := allSymbolIDs(db)
		if err != nil {
			return nil, err
		}
		gods, label := labelForCommunity(db, 0, ids, opts)
		hub := ""
		if len(gods) > 0 {
			hub = gods[0]
		}
		res.Communities = []Community{{ID: 0, Label: label, Members: ids, GodNodes: gods, HubLabel: hub, Cohesion: cohesionOf(len(ids), 0)}}
		return res, nil
	}

	reg, g, nodes, err := loadGraph(db)
	if err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}

	det := graph.NewLeiden(graph.LeidenOptions{
		Seed:          opts.Seed,
		Resolution:    opts.Resolution,
		MaxIterations: opts.MaxIterations,
		NumRuns:       opts.NumRuns,
	})
	part, err := det.Detect(g)
	if err != nil {
		return nil, fmt.Errorf("leiden: %w", err)
	}
	res.Modularity = part.Modularity

	// Internal-edge count per community (038): walk the SAME undirected,
	// self-loop-free edge set the partition was built from and tally, per
	// partition id, how many edges have both endpoints inside the community.
	// This is the numerator of cohesion; the denominator is C(n,2) in Go. An
	// undirected edge is stored twice (both directions in adjacency), so the
	// canonical min/max pair de-duplicates it exactly as loadGraph does.
	internal := map[int]int{}
	seenPair := map[[2]graph.NodeID]bool{}
	for _, from := range g.Nodes() {
		for _, e := range g.Neighbors(from) {
			to := e.To
			if from == to {
				continue
			}
			pair := [2]graph.NodeID{from, to}
			if pair[1] < pair[0] {
				pair = [2]graph.NodeID{to, from}
			}
			if seenPair[pair] {
				continue
			}
			seenPair[pair] = true
			cid, ok := part.Partition[pair[0]]
			if !ok {
				continue
			}
			if cid2, ok2 := part.Partition[pair[1]]; ok2 && cid2 == cid {
				internal[cid]++
			}
		}
	}

	// Invert the partition into communities: loom node -> member symbol ids.
	loomToSymbol := map[graph.NodeID]int64{}
	for _, n := range nodes {
		if id, ok := reg.ID(n.key); ok {
			loomToSymbol[id] = n.symbolID
		}
	}
	byComm := map[int][]int64{}
	for node, cid := range part.Partition {
		sid, ok := loomToSymbol[node]
		if !ok {
			continue
		}
		byComm[cid] = append(byComm[cid], sid)
	}
	var commIDs []int
	for cid := range byComm {
		commIDs = append(commIDs, cid)
	}
	sort.Ints(commIDs)

	communities := make([]Community, 0, len(byComm))
	for i, cid := range commIDs {
		members := byComm[cid]
		sort.Slice(members, func(a, b int) bool { return members[a] < members[b] })
		gods, label := labelForCommunity(db, i, members, opts)
		hub := ""
		if len(gods) > 0 {
			hub = gods[0]
		}
		communities = append(communities, Community{ID: i, Label: label, Members: members, GodNodes: gods, HubLabel: hub, Cohesion: cohesionOf(len(members), internal[cid])})
	}
	res.Communities = communities

	if err := writeCommunities(db, communities); err != nil {
		return nil, err
	}
	return res, nil
}

// allSymbolIDs returns every symbol id, ascending.
func allSymbolIDs(db *sql.DB) ([]int64, error) {
	rows, err := db.Query(`SELECT id FROM symbols ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// writeCommunities replaces the communities table with the detected
// partition (idempotent).
func writeCommunities(db *sql.DB, communities []Community) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM communities`); err != nil {
		return err
	}
	for _, c := range communities {
		for _, m := range c.Members {
			if _, err := tx.Exec(`INSERT OR REPLACE INTO communities (id, symbol_id) VALUES (?, ?)`, c.ID, m); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// cachedResult rebuilds a Result from the stored communities table when the
// content-hash cache key matches. Returns nil (no error) on a cache miss.
func cachedResult(db *sql.DB, cacheKey string) (*Result, error) {
	var stored string
	err := db.QueryRow(`SELECT value FROM community_meta_cache WHERE key = 'cache_key'`).Scan(&stored)
	if err != nil || stored != cacheKey {
		return nil, nil
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM communities`).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	res := &Result{CacheKey: cacheKey, Updated: time.Now().UTC()}
	rows, err := db.Query(`SELECT id, symbol_id FROM communities ORDER BY id, symbol_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int][]int64{}
	var order []int
	for rows.Next() {
		var id int
		var sid int64
		if err := rows.Scan(&id, &sid); err != nil {
			return nil, err
		}
		if _, ok := byID[id]; !ok {
			order = append(order, id)
		}
		byID[id] = append(byID[id], sid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Ints(order)
	for i, id := range order {
		res.Communities = append(res.Communities, Community{
			ID:       i,
			Label:    communityLabel(db, id),
			Members:  byID[id],
			HubLabel: communityHubLabel(db, id),
			Cohesion: communityCohesion(db, id),
		})
	}
	return res, nil
}

// communityLabel reads a stored label (community_meta keyed by community id).
func communityLabel(db *sql.DB, id int) string {
	var label string
	_ = db.QueryRow(`SELECT label FROM community_meta WHERE id = ?`, id).Scan(&label)
	if label == "" {
		return "community-" + fmt.Sprintf("%d", id)
	}
	return label
}

// communityHubLabel reads the stored top god-node name ("" when none).
func communityHubLabel(db *sql.DB, id int) string {
	var hub string
	_ = db.QueryRow(`SELECT hub_label FROM community_meta WHERE id = ?`, id).Scan(&hub)
	return hub
}

// communityCohesion reads the stored cohesion (nil when NULL — a community
// built before the 038 pass, or on a store predating the column).
func communityCohesion(db *sql.DB, id int) *float64 {
	var coh sql.NullFloat64
	if err := db.QueryRow(`SELECT cohesion FROM community_meta WHERE id = ?`, id).Scan(&coh); err != nil || !coh.Valid {
		return nil
	}
	return &coh.Float64
}

// writeMeta caches the detected partition: one community_meta row per
// community (label + symbol count + god-node names) and the content-hash cache
// key.
func writeMeta(db *sql.DB, res *Result) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM community_meta`); err != nil {
		return err
	}
	for _, c := range res.Communities {
		var coh interface{}
		if c.Cohesion != nil {
			coh = *c.Cohesion
		}
		if _, err := tx.Exec(`INSERT INTO community_meta (id, label, symbol_count, god_nodes, hub_label, cache_key, cohesion) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.ID, c.Label, len(c.Members), strings.Join(c.GodNodes, ","), c.HubLabel, res.CacheKey, coh); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS community_meta_cache (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO community_meta_cache (key, value) VALUES ('cache_key', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, res.CacheKey); err != nil {
		return err
	}
	return tx.Commit()
}

// labelForCommunity is a hook for tests; defaults to the LLM-free derivation
// in labels.go.
var labelForCommunity = labelForCommunityImpl

// cohesionOf computes the internal-edge density internalEdges / C(n,2) for a
// community of n members. A community of <2 members has no possible internal
// edge, so it returns a pointer to 0 (a defined, honest value — NOT null: a
// singleton is maximally cohesive in the degenerate sense of "nothing to
// cohere", and null is reserved for "not yet computed by a 038 pass"). The
// denominator is computed in Go to avoid a SQL divide-by-zero on n<2.
func cohesionOf(n, internalEdges int) *float64 {
	if n < 2 {
		z := 0.0
		return &z
	}
	denom := float64(n * (n - 1) / 2)
	c := float64(internalEdges) / denom
	return &c
}
