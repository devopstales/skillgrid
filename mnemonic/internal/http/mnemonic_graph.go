package http

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic/internal/community"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// Phase 4 — Mnemonic vector graph. Serves the code/memory graph (symbols +
// edges) to the Sigma.js frontend. One physical graph per project (the
// symbols/edges SQLite tables); communities are Leiden partitions computed by
// the community package (self-cached).
//
// The response is a flat {nodes, edges} shape the SPA converts to graphology.
// `truncated` reports depth/limit capping; `degraded` reports the 005/008 soft
// dependency: when no edge/community data is present the view falls back to
// node-only + a file list.

// graphNode is one symbol node as served to the frontend.
type graphNode struct {
	ID        int64  `json:"id"`
	UID       string `json:"uid"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	Language  string `json:"language"`
	Path      string `json:"path"`
	Degree    int    `json:"degree"`
	Community int    `json:"community"`
}

// graphEdge is one edge as served to the frontend.
type graphEdge struct {
	ID         string `json:"id"`
	Source     int64  `json:"source"`
	Target     int64  `json:"target"`
	Type       string `json:"type"`
	Weight     float64 `json:"weight"`
	Directed   bool   `json:"directed"`
	Confidence string `json:"confidence,omitempty"`
}

// graphResponse is GET /mnemonic/graph.
type graphResponse struct {
	Nodes     []graphNode  `json:"nodes"`
	Edges     []graphEdge  `json:"edges"`
	Truncated bool         `json:"truncated"`
	Degraded  bool         `json:"degraded"`
	Files     []string     `json:"files,omitempty"`
	Project   string       `json:"project"`
}

// nodeRow is the raw symbol row (before degree/community enrichment).
type nodeRow struct {
	ID       int64
	UID      string
	Name     string
	Kind     string
	Language string
	Path     string
}

// loadGraph reads all symbols + current edges for a project and computes
// degrees. Returns the node map (by id), the edge list, and a file list.
func loadGraph(ctx context.Context, st *store.Store) (map[int64]*graphNode, []graphEdge, []string, error) {
	db := st.DB
	// nodes: symbols joined to their file path
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.uid, s.name, s.kind, s.language, f.path
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		ORDER BY s.id`)
	if err != nil {
		return nil, nil, nil, err
	}
	nodes := map[int64]*graphNode{}
	var nodeIDs []int64
	for rows.Next() {
		var n nodeRow
		if err := rows.Scan(&n.ID, &n.UID, &n.Name, &n.Kind, &n.Language, &n.Path); err != nil {
			rows.Close()
			return nil, nil, nil, err
		}
		nodes[n.ID] = &graphNode{ID: n.ID, UID: n.UID, Label: n.Name, Type: n.Kind, Language: n.Language, Path: n.Path, Community: -1}
		nodeIDs = append(nodeIDs, n.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, err
	}
	rows.Close()

	// edges: current-state edges (temporal filter applied by QueryEdges)
	rawEdges, err := codeindex.QueryEdges(st)
	if err != nil {
		return nil, nil, nil, err
	}
	edges := make([]graphEdge, 0, len(rawEdges))
	for _, e := range rawEdges {
		// name-only edges (to_id NULL) or edges to an absent node are dropped
		// from the renderable graph (the converter prunes them anyway).
		if !e.ToID.Valid {
			continue
		}
		if _, ok := nodes[e.FromID]; !ok {
			continue
		}
		if _, ok := nodes[e.ToID.Int64]; !ok {
			continue
		}
		id := e.Kind + ":" + itoa(e.FromID) + "->" + itoa(e.ToID.Int64)
		edges = append(edges, graphEdge{
			ID:         id,
			Source:     e.FromID,
			Target:     e.ToID.Int64,
			Type:       e.Kind,
			Weight:     1,
			Directed:   true,
			Confidence: e.Confidence,
		})
		// undirected degree for sizing
		nodes[e.FromID].Degree++
		nodes[e.ToID.Int64].Degree++
	}

	// file list for the degraded fallback (distinct paths)
	fileRows, err := db.QueryContext(ctx, `SELECT path FROM files ORDER BY path`)
	if err != nil {
		return nil, nil, nil, err
	}
	var files []string
	for fileRows.Next() {
		var p string
		if err := fileRows.Scan(&p); err != nil {
			fileRows.Close()
			return nil, nil, nil, err
		}
		files = append(files, p)
	}
	if err := fileRows.Err(); err != nil {
		fileRows.Close()
		return nil, nil, nil, err
	}
	fileRows.Close()

	return nodes, edges, files, nil
}

// attachCommunities maps Leiden community ids onto node rows (by symbol id).
func attachCommunities(nodes map[int64]*graphNode, res *community.Result) {
	if res == nil {
		return
	}
	for _, c := range res.Communities {
		for _, m := range c.Members {
			if n, ok := nodes[m]; ok {
				n.Community = c.ID
			}
		}
	}
}

// filterByDepth keeps the depth-hop neighborhood of a single node (BFS over
// edges, undirected). Returns the surviving node set + edges, and whether the
// neighborhood was capped (the depth bound was reached with more edges to drop).
func filterByDepth(nodes map[int64]*graphNode, edges []graphEdge, center int64, depth int) (map[int64]*graphNode, []graphEdge, bool) {
	if depth <= 0 {
		// depth 0 → only the center node
		if c, ok := nodes[center]; ok {
			return map[int64]*graphNode{center: c}, nil, false
		}
		return map[int64]*graphNode{}, nil, false
	}
	// adjacency (undirected)
	adj := map[int64]map[int64]bool{}
	for _, e := range edges {
		if adj[e.Source] == nil {
			adj[e.Source] = map[int64]bool{}
		}
		if adj[e.Target] == nil {
			adj[e.Target] = map[int64]bool{}
		}
		adj[e.Source][e.Target] = true
		adj[e.Target][e.Source] = true
	}
	// BFS
	dist := map[int64]int{center: 0}
	queue := []int64{center}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if dist[cur] >= depth {
			continue
		}
		for nb := range adj[cur] {
			if _, seen := dist[nb]; seen {
				continue
			}
			dist[nb] = dist[cur] + 1
			queue = append(queue, nb)
		}
	}
	kept := map[int64]*graphNode{}
	for id, d := range dist {
		if d <= depth {
			if n, ok := nodes[id]; ok {
				kept[id] = n
			}
		}
	}
	keptEdges := make([]graphEdge, 0)
	truncated := false
	for _, e := range edges {
		if _, ok := kept[e.Source]; ok {
			if _, ok2 := kept[e.Target]; ok2 {
				keptEdges = append(keptEdges, e)
			} else {
				truncated = true
			}
		} else {
			truncated = true
		}
	}
	return kept, keptEdges, truncated
}

// itoa is a small int64→string helper (avoids importing strconv in this file).
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// handleMnemonicGraph is GET /mnemonic/graph?project=&node_id=&depth=.
func (s *Server) handleMnemonicGraph(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	nodes, edges, files, err := loadGraph(r.Context(), h.Store())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load graph: "+err.Error())
		return
	}

	// communities: best-effort (soft dep). If detection is unavailable the
	// graph still renders with community=-1.
	res, cErr := s.svc.CodeCommunities(r.Context(), projectID, community.Options{})
	if cErr == nil {
		attachCommunities(nodes, res)
	}

	out := graphResponse{Project: projectID}

	// node_id + depth → neighborhood subgraph
	if q := r.URL.Query().Get("node_id"); q != "" {
		var center int64
		if _, err := ssn64(q, &center); err != nil {
			writeError(w, http.StatusBadRequest, "node_id must be an integer symbol id")
			return
		}
		depth := queryInt(r, "depth", 2)
		kept, keptEdges, truncated := filterByDepth(nodes, edges, center, depth)
		out.Nodes = nodesToSlice(kept)
		out.Edges = keptEdges
		out.Truncated = truncated
		out.Degraded = len(keptEdges) == 0
		if out.Degraded {
			out.Files = files
		}
		writeJSON(w, http.StatusOK, out)
		return
	}

	// no node_id → full graph (with truncation if it exceeds the cap)
	cap := queryInt(r, "limit", defaultGraphLimit)
	fullNodes := nodesToSlice(nodes)
	out.Truncated = len(fullNodes) > cap
	if len(fullNodes) > cap {
		fullNodes = fullNodes[:cap]
	}
	// keep only edges whose endpoints survived the cap
	keptIDs := map[int64]bool{}
	for _, n := range fullNodes {
		keptIDs[n.ID] = true
	}
	pruned := make([]graphEdge, 0, len(edges))
	for _, e := range edges {
		if keptIDs[e.Source] && keptIDs[e.Target] {
			pruned = append(pruned, e)
		}
	}
	out.Nodes = fullNodes
	out.Edges = pruned
	out.Degraded = len(pruned) == 0
	if out.Degraded {
		out.Files = files
	}
	writeJSON(w, http.StatusOK, out)
}

// handleMnemonicGraphNodes is GET /mnemonic/graph/nodes?project=&limit=.
func (s *Server) handleMnemonicGraphNodes(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	nodes, _, _, err := loadGraph(r.Context(), h.Store())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load graph: "+err.Error())
		return
	}
	limit := queryInt(r, "limit", defaultGraphLimit)
	all := nodesToSlice(nodes)
	truncated := len(all) > limit
	if truncated {
		all = all[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes":     all,
		"truncated": truncated,
		"total":     len(nodes),
	})
}

// nodesToSlice returns a stable (id-sorted) slice of the node map.
func nodesToSlice(nodes map[int64]*graphNode) []graphNode {
	out := make([]graphNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, *n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ssn64 parses a base-10 int64 from a query string.
func ssn64(s string, dst *int64) (bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, sql.ErrNoRows
	}
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, sql.ErrNoRows
		}
		n = n*10 + int64(c-'0')
	}
	*dst = n
	return true, nil
}

const defaultGraphLimit = 5000
