package http

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"

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
	// TICKET-12 additive enrichment: the symbol's source line range and its
	// stored signature (omitted when the indexer did not capture one).
	// Additive only — every pre-existing field and its value is unchanged.
	Signature string `json:"signature,omitempty"`
	Lines     *lines `json:"lines,omitempty"`
}

// lines is the 1-based source line range of one symbol in its file.
type lines struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// graphEdge is one edge as served to the frontend.
type graphEdge struct {
	ID         string  `json:"id"`
	Source     int64   `json:"source"`
	Target     int64   `json:"target"`
	Type       string  `json:"type"`
	Weight     float64 `json:"weight"`
	Directed   bool    `json:"directed"`
	Confidence string  `json:"confidence,omitempty"`
}

// graphResponse is GET /mnemonic/graph.
type graphResponse struct {
	Nodes     []graphNode `json:"nodes"`
	Edges     []graphEdge `json:"edges"`
	Truncated bool        `json:"truncated"`
	Degraded  bool        `json:"degraded"`
	Files     []string    `json:"files,omitempty"`
	Project   string      `json:"project"`
}

// nodeRow is the raw symbol row (before degree/community enrichment).
type nodeRow struct {
	ID        int64
	UID       string
	Name      string
	Kind      string
	Language  string
	Path      string
	Signature string
	StartLine int
	EndLine   int
}

// loadGraph reads all symbols + current edges for a project and computes
// degrees. Returns the node map (by id), the edge list, and a file list.
// Node enrichment (TICKET-12): every node carries its stored line range
// (lines) and signature — additive fields, read straight from the symbols
// row, so no per-node file read happens on the graph load path.
func loadGraph(ctx context.Context, st *store.Store) (map[int64]*graphNode, []graphEdge, []string, error) {
	db := st.DB
	// nodes: symbols joined to their file path
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.uid, s.name, s.kind, s.language, f.path,
		       COALESCE(s.signature, ''), s.start_line, s.end_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		ORDER BY s.id`)
	if err != nil {
		return nil, nil, nil, err
	}
	nodes := map[int64]*graphNode{}
	var nodeIDs []int64
	for rows.Next() {
		var n nodeRow
		if err := rows.Scan(&n.ID, &n.UID, &n.Name, &n.Kind, &n.Language, &n.Path, &n.Signature, &n.StartLine, &n.EndLine); err != nil {
			rows.Close()
			return nil, nil, nil, err
		}
		gn := &graphNode{ID: n.ID, UID: n.UID, Label: n.Name, Type: n.Kind, Language: n.Language, Path: n.Path, Community: -1}
		if n.Signature != "" {
			gn.Signature = n.Signature
		}
		if n.StartLine > 0 && n.EndLine >= n.StartLine {
			gn.Lines = &lines{Start: n.StartLine, End: n.EndLine}
		}
		nodes[n.ID] = gn
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

// ---------------------------------------------------------------------------
// On-demand locals resolver (TICKET-12).
//
// resolveSymbolLocals parses ONE symbol's source on demand (per inspector
// click) and returns its parameters and local variables. It is PURE and
// READ-ONLY: it opens the file, walks the tree-sitter AST, and writes nothing
// back — the graph store is never touched (the ephemeral-locals contract,
// proven by TestLocalsResolver_Ephemeral's empty store diff).
//
// Co-location choice: this file (the code-graph derive in the http layer),
// next to loadGraph — the resolver is a per-click view of the same symbol
// rows loadGraph serves, and lives where the enrichment it feeds is defined.
// It is deletable as a unit (this whole section + its callers).
//
// Degraded states are NOT errors: an unknown language, a missing file, a
// parse failure, or an unlocatable body all yield (nil, nil) so the
// inspector can render a degraded "locals unavailable" state.
//
// ponytail: ceiling = Go only (one grammar family). The shape dispatch
// mirrors pdg.BuildCFG's Go/shape split; non-Go languages degrade to nil
// rather than shipping a per-language locals grammar matrix.
// ---------------------------------------------------------------------------

// SymbolLocals is the ephemeral parse of one symbol's scope.
type SymbolLocals struct {
	Params []SymbolLocal // declared parameters (in signature order)
	Locals []SymbolLocal // local variables (first declaration in source order)
}

// SymbolLocal is one parameter or local variable with its declared type (""
// when the declaration carries no type, e.g. a Go `:=` without a cast).
type SymbolLocal struct {
	Name string
	Type string
}

// resolveSymbolLocals returns the params + locals of the symbol (name,
// startLine) in the file st identifies, rooted at root. (nil, nil) on any
// degraded state; a non-nil error is returned only for the case the caller
// must distinguish (today: none — the contract is degrade, not fail).
func resolveSymbolLocals(st *store.Store, root, name, lang string, startLine, endLine int) (*SymbolLocals, error) {
	path, err := symbolFilePath(st, name, startLine)
	if err != nil {
		return nil, nil // symbol row gone mid-click: degraded
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return nil, nil // file not on disk: degraded
	}
	tree, l, err := parseGraphTree(lang, src)
	if err != nil {
		return nil, nil // unparseable file: degraded
	}
	defer tree.Release()
	fn := findGraphFunctionNode(tree.RootNode(), l, name, startLine, src)
	if fn == nil {
		return nil, nil // body unlocatable: degraded
	}
	return graphLocalsFromNode(fn, l, src), nil
}

// symbolFilePath resolves a symbol's stored file path by (name, start_line).
func symbolFilePath(st *store.Store, name string, startLine int) (string, error) {
	var path string
	err := st.DB.QueryRow(
		`SELECT f.path FROM symbols s JOIN files f ON f.id = s.file_id
		 WHERE s.name = ? AND s.start_line = ? LIMIT 1`, name, startLine).Scan(&path)
	return path, err
}

// parseGraphTree parses src with the language's grammar (the same registry
// the pdg package uses — no new grammar, no CGo).
func parseGraphTree(lang string, src []byte) (*ts.Tree, *ts.Language, error) {
	entry := grammars.DetectLanguage(filepath.Base("x." + lang))
	if entry == nil {
		if byName := grammarNameFor(lang); byName != "" {
			entry = grammars.DetectLanguageByName(byName)
		}
	}
	if entry == nil {
		return nil, nil, errGraphNoGrammar
	}
	l := entry.Language()
	parser := ts.NewParser(l)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	return tree, l, nil
}

// errGraphNoGrammar marks an unknown language (a degraded state, not a fault).
var errGraphNoGrammar = os.ErrNotExist

// grammarNameFor maps a few indexer language names to grammar registry names
// for languages whose probe extension is ambiguous.
func grammarNameFor(lang string) string {
	switch lang {
	case "typescript":
		return "TypeScript"
	default:
		return ""
	}
}

// findGraphFunctionNode locates the definition node for (name, startLine):
// a named definition whose name matches and whose first line is startLine.
func findGraphFunctionNode(root *ts.Node, l *ts.Language, name string, startLine int, src []byte) *ts.Node {
	var found *ts.Node
	var walk func(n *ts.Node)
	walk = func(n *ts.Node) {
		if found != nil || n == nil {
			return
		}
		if nm := graphDefName(n, l, src); nm == name && graphDefFirstLine(n, l, src) == startLine {
			found = n
			return
		}
		for i := 0; i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c != nil {
				walk(c)
			}
		}
	}
	walk(root)
	return found
}

// graphDefName returns the name carried by a definition node (the "name"
// field, or its first identifier child — the two shapes the extractor's
// defNodes cover for Go-style and identifier-headed definitions).
func graphDefName(n *ts.Node, l *ts.Language, src []byte) string {
	if nm := n.ChildByFieldName("name", l); nm != nil {
		return nm.Text(src)
	}
	if n.NamedChildCount() > 0 {
		if c := n.NamedChild(0); c != nil && c.Type(l) == "identifier" {
			return c.Text(src)
		}
	}
	return ""
}

// graphDefFirstLine is the 1-based first line of a definition node.
func graphDefFirstLine(n *ts.Node, l *ts.Language, src []byte) int {
	_ = l
	off := int(n.StartByte())
	if off > len(src) {
		off = len(src)
	}
	line := 1
	for i := 0; i < off; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}

// graphLocalsFromNode walks one definition node and collects its parameters
// (declaration order) and local variables (first declaration, source order).
// Deterministic: source order throughout.
func graphLocalsFromNode(fn *ts.Node, l *ts.Language, src []byte) *SymbolLocals {
	out := &SymbolLocals{}
	// params: Go parameter_list/parameter_declaration, or a single parameter
	// name directly under the definition (name field "name" → its text is the
	// param list; kept generic via named children named *parameter*).
	var walkParams func(n *ts.Node)
	walkParams = func(n *ts.Node) {
		if n == nil {
			return
		}
		t := n.Type(l)
		switch t {
		case "parameter_declaration":
			decls := paramNameDecls(n, l, src)
			typ := paramTypeText(n, l, src)
			for _, d := range decls {
				out.Params = append(out.Params, SymbolLocal{Name: d, Type: typ})
			}
			return
		case "identifier":
			// a bare identifier child of the def is a param only when the def
			// has a parameter_list sibling elsewhere; the param_list branch
			// above already covered Go, so here we collect single-param defs
			// (e.g. `def f(x):`): the def's direct identifier child.
			if isDirectParam(fn, n) {
				out.Params = append(out.Params, SymbolLocal{Name: n.Text(src), Type: ""})
			}
			return
		}
		for i := 0; i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c != nil {
				walkParams(c)
			}
		}
	}
	walkParams(fn)

	// locals: first declaration in source order (block → statement_list),
	// including loop-initial declarations; nested scopes included (a teaching
	// view shows everything the function declares).
	var walkLocals func(n *ts.Node)
	seen := map[string]bool{}
	walkLocals = func(n *ts.Node) {
		if n == nil {
			return
		}
		t := n.Type(l)
		switch t {
		case "short_var_declaration", "variable_declaration":
			for _, d := range varDeclNames(n, l, src) {
				if !seen[d] {
					seen[d] = true
					out.Locals = append(out.Locals, SymbolLocal{Name: d, Type: varDeclType(n, l, src)})
				}
			}
			// still descend: a := f() where f returns nothing can nest decls
			for i := 0; i < n.NamedChildCount(); i++ {
				if c := n.NamedChild(i); c != nil {
					walkLocals(c)
				}
			}
			return
		case "for_statement":
			// loop init (Go: for_clause → short_var_declaration) is a local
			if fc := n.ChildByFieldName("for_clause", l); fc != nil {
				walkLocals(fc)
			}
		}
		for i := 0; i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c != nil {
				walkLocals(c)
			}
		}
	}
	walkLocals(fn)

	if len(out.Params) == 0 && len(out.Locals) == 0 {
		return nil
	}
	return out
}

// isDirectParam reports whether n is a direct identifier child of fn (the
// single-param `def f(x)` shape), not a name/nested identifier.
func isDirectParam(fn, n *ts.Node) bool {
	for i := 0; i < fn.NamedChildCount(); i++ {
		if fn.NamedChild(i) == n {
			return true
		}
	}
	return false
}

// paramNameDecls returns the declared names of one parameter_declaration
// (Go allows several: `a, b int` — the "name" field repeats per identifier).
func paramNameDecls(n *ts.Node, l *ts.Language, src []byte) []string {
	var out []string
	for i := 0; i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c == nil {
			continue
		}
		// The parameter_list's child field names are not reliable across
		// grammars; identifiers inside a parameter_declaration ARE the names.
		if c.Type(l) == "identifier" {
			out = append(out, c.Text(src))
		}
	}
	return out
}

// paramTypeText is the declared type of a parameter_declaration ("" when
// untyped — rare in Go, but possible in JS/TS shapes).
func paramTypeText(n *ts.Node, l *ts.Language, src []byte) string {
	for i := 0; i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c == nil {
			continue
		}
		if c.Type(l) != "identifier" {
			return strings.TrimSpace(c.Text(src))
		}
	}
	return ""
}

// varDeclNames returns the declared names of a short_var_declaration /
// variable_declaration (the `left` field's identifier(s)).
func varDeclNames(n *ts.Node, l *ts.Language, src []byte) []string {
	left := n.ChildByFieldName("left", l)
	if left == nil {
		return nil
	}
	var out []string
	var collect func(m *ts.Node)
	collect = func(m *ts.Node) {
		if m == nil {
			return
		}
		if m.Type(l) == "identifier" && m.Text(src) != "_" {
			out = append(out, m.Text(src))
			return
		}
		for i := 0; i < m.NamedChildCount(); i++ {
			if c := m.NamedChild(i); c != nil {
				collect(c)
			}
		}
	}
	collect(left)
	return out
}

// varDeclType is the declared type of a declaration ("" for `:=` without a
// cast — the teaching view then shows "inferred").
func varDeclType(n *ts.Node, l *ts.Language, src []byte) string {
	t := n.ChildByFieldName("type", l)
	if t != nil {
		return strings.TrimSpace(t.Text(src))
	}
	return ""
}
