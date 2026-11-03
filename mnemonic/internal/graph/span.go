package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// Knowledge edge kinds (mirrored from the knowledge package; the values must
// match knowledge.KindReads/KindWrites/KindConfigures/KindReferences). The
// graph package does not import knowledge (avoids an import cycle: knowledge
// tests exercise graph), so the literals are declared here.
const (
	KindReads      = "reads"
	KindWrites     = "writes"
	KindConfigures = "configures"
	KindReferences = "references"
)

// itoa / parseItoa are small int64<->string helpers for span node keys.
func itoa(id int64) string { return strconv.FormatInt(id, 10) }

func parseItoa(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

// spanRef identifies one node in the unified span space by (kind, id). A
// symbol id and a knowledge node id may numerically collide across tables, so
// the kind is part of the identity.
type spanRef struct {
	kind string
	id   int64
}

// spanNode is one node in the unified knowledge span: a 005 symbol or a
// 015 knowledge node (doc / config / table / column). Knowledge nodes are
// first-class endpoints so code_path can trace code→doc→config→table.
type spanNode struct {
	ID    int64
	Kind  string // "symbol" | "doc" | "config" | "table" | "column"
	Name  string // symbol name | doc/config path | table name
	Path  string
	Line  int
	Valid bool
}

// tableNodeKey renders a table node's display name ("table:<name>"). A bare
// table name resolves unambiguously to its table node; "table:<name>" is an
// explicit disambiguation.
func tableNodeKey(name string) string { return "table:" + name }

// spanNodeFromSymbol adapts a resolved 005 symbol into a span node.
func spanNodeFromSymbol(s Symbol) spanNode {
	return spanNode{ID: s.ID, Kind: "symbol", Name: s.Name, Path: s.Path, Line: s.StartLine, Valid: true}
}

// resolveSpanNode resolves a name to a single span node, searching the 005
// symbols first (code-to-code resolution is unchanged) and then the 015
// knowledge tables. An empty/zero name resolves to the zero node. Ambiguity is
// surfaced as a single node with a count suffix only when exactly one table
// matches; a doc/config path is unique by path (015 unique index).
func resolveSpanNode(ctx context.Context, db *sql.DB, name string) (spanNode, error) {
	if name == "" {
		return spanNode{}, nil
	}
	// Explicit table disambiguation ("table:users").
	if len(name) > 6 && name[:6] == "table:" {
		return resolveTableNode(ctx, db, name[6:])
	}
	// 005 symbol (code-to-code unchanged).
	var (
		id      int64
		uid     string
		sname   string
		kind    string
		path    string
		startLn int
	)
	err := db.QueryRowContext(ctx, `
		SELECT s.id, s.uid, s.name, s.kind,
		       f.path, s.start_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.name = ?
		ORDER BY s.id`, name).
		Scan(&id, &uid, &sname, &kind, &path, &startLn)
	if err == nil {
		return spanNode{
			ID: id, Kind: "symbol", Name: sname, Path: path, Line: startLn, Valid: true,
		}, nil
	}
	if err != sql.ErrNoRows {
		return spanNode{}, err
	}
	// Knowledge nodes: doc (by path), config (by path), table (by name).
	if n, err := resolveDocNode(ctx, db, name); err != nil {
		return spanNode{}, err
	} else if n.Valid {
		return n, nil
	}
	if n, err := resolveConfigNode(ctx, db, name); err != nil {
		return spanNode{}, err
	} else if n.Valid {
		return n, nil
	}
	return resolveTableNode(ctx, db, name)
}

func resolveDocNode(ctx context.Context, db *sql.DB, path string) (spanNode, error) {
	var (
		id   int64
		file int64
	)
	err := db.QueryRowContext(ctx, `SELECT id, file_id FROM doc_nodes WHERE path = ? ORDER BY id`, path).
		Scan(&id, &file)
	if err != nil {
		if err == sql.ErrNoRows {
			return spanNode{}, nil
		}
		return spanNode{}, err
	}
	return spanNode{ID: id, Kind: "doc", Name: path, Path: path, Valid: true}, nil
}

func resolveConfigNode(ctx context.Context, db *sql.DB, path string) (spanNode, error) {
	var (
		id   int64
		file int64
	)
	err := db.QueryRowContext(ctx, `SELECT id, file_id FROM config_nodes WHERE path = ? ORDER BY id`, path).
		Scan(&id, &file)
	if err != nil {
		if err == sql.ErrNoRows {
			return spanNode{}, nil
		}
		return spanNode{}, err
	}
	return spanNode{ID: id, Kind: "config", Name: path, Path: path, Valid: true}, nil
}

func resolveTableNode(ctx context.Context, db *sql.DB, table string) (spanNode, error) {
	var (
		id   int64
		file int64
		kind string
	)
	err := db.QueryRowContext(ctx, `SELECT id, file_id, kind FROM sql_schema_nodes WHERE table_name = ? AND kind = 'table' ORDER BY id`, table).
		Scan(&id, &file, &kind)
	if err != nil {
		if err == sql.ErrNoRows {
			return spanNode{}, nil
		}
		return spanNode{}, err
	}
	return spanNode{ID: id, Kind: "table", Name: tableNodeKey(table), Path: "", Valid: true}, nil
}

// loadSpanNodeByID loads a span node by its (kind, id) — the unified inverse
// of resolveSpanNode used by BFS expansion.
func loadSpanNodeByID(ctx context.Context, db *sql.DB, kind string, id int64) (spanNode, error) {
	switch kind {
	case "symbol":
		s, err := loadSymbolByID(ctx, db, id)
		if err != nil {
			return spanNode{}, err
		}
		return spanNodeFromSymbol(s), nil
	case "doc":
		var (
			path string
			file int64
		)
		if err := db.QueryRowContext(ctx, `SELECT path, file_id FROM doc_nodes WHERE id = ?`, id).Scan(&path, &file); err != nil {
			return spanNode{}, err
		}
		return spanNode{ID: id, Kind: "doc", Name: path, Path: path, Valid: true}, nil
	case "config":
		var (
			path string
			file int64
		)
		if err := db.QueryRowContext(ctx, `SELECT path, file_id FROM config_nodes WHERE id = ?`, id).Scan(&path, &file); err != nil {
			return spanNode{}, err
		}
		return spanNode{ID: id, Kind: "config", Name: path, Path: path, Valid: true}, nil
	case "table", "column":
		var (
			tname, cname string
			kind         string
			file         int64
		)
		if err := db.QueryRowContext(ctx, `SELECT table_name, COALESCE(column_name,''), kind, file_id FROM sql_schema_nodes WHERE id = ?`, id).
			Scan(&tname, &cname, &kind, &file); err != nil {
			return spanNode{}, err
		}
		name := tableNodeKey(tname)
		if cname != "" {
			name = tname + "." + cname
		}
		return spanNode{ID: id, Kind: kind, Name: name, Valid: true}, nil
	default:
		return spanNode{}, fmt.Errorf("unknown span node kind %q", kind)
	}
}

// spanNeighbors returns the (kind, id) of every node adjacent to a span node,
// plus the edge rows so the caller can reconstruct labeled hops. It reads the
// 005 edges table (where knowledge edges live) and resolves each endpoint into
// the unified space: a to_id pointing at a symbol is a symbol, a to_id matching
// a doc/config/sql node id is that knowledge node (knowledge edges store the
// knowledge node id in to_id even though the column's FK is symbols).
type spanHop struct {
	Kind       string
	Confidence string
	Line       int
	From       spanNode
	To         spanNode
}

// fetchSpanEdges loads every edge touching the span node's id, resolving both
// endpoints into the unified node space. An edge endpoint that is a symbol id
// resolves as a symbol; an endpoint that is a knowledge node id resolves as that
// knowledge node. Name-only endpoints (to_id null) resolve by name through the
// unified resolver (so a name-only table/config/doc reference still spans).
// rawSpanEdge is one edge row before its endpoints are resolved into the
// unified node space.
type rawSpanEdge struct {
	kind   string
	fromID int64
	toID   int64 // 0 when name-only
	toName string
	conf   string
	line   int
}

func fetchSpanEdges(ctx context.Context, db *sql.DB, node spanNode) ([]spanHop, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT e.kind, e.from_id, e.to_id, e.to_name, e.confidence, e.line
		FROM edges e
		WHERE e.from_id = ? OR e.to_id = ?
		ORDER BY e.id`, node.ID, node.ID)
	if err != nil {
		return nil, err
	}
	// Drain the rows into a slice BEFORE resolving endpoints: endpoint
	// resolution issues further queries, and a single-connection store cannot
	// run two queries at once while a result set is open.
	var raw []rawSpanEdge
	for rows.Next() {
		var (
			kind   string
			fromID int64
			toID   sql.NullInt64
			toName sql.NullString
			conf   string
			line   int
		)
		if err := rows.Scan(&kind, &fromID, &toID, &toName, &conf, &line); err != nil {
			rows.Close()
			return nil, err
		}
		raw = append(raw, rawSpanEdge{kind: kind, fromID: fromID, toID: toID.Int64, toName: toName.String, conf: conf, line: line})
	}
	rerr := rows.Err()
	rows.Close()
	if rerr != nil {
		return nil, rerr
	}
	var out []spanHop
	for _, r := range raw {
		from, err := resolveSpanEndpoint(ctx, db, r.fromID, "", r.kind)
		if err != nil {
			return nil, err
		}
		to, err := resolveSpanEndpoint(ctx, db, r.toID, r.toName, r.kind)
		if err != nil {
			return nil, err
		}
		// Skip a hop whose endpoint is unresolved (no fabricated hop). A
		// name-only edge (to_id null) that resolves to nothing is invalid (ID 0)
		// and must not become a BFS frontier node — that would make the path
		// reconstruction loop un-terminable.
		if !from.Valid || from.ID == 0 {
			continue
		}
		if !to.Valid || to.ID == 0 {
			continue
		}
		out = append(out, spanHop{Kind: r.kind, Confidence: r.conf, Line: r.line, From: from, To: to})
	}
	return out, nil
}

// resolveSpanEndpoint resolves an edge endpoint id (and optional name) into the
// unified node space. Symbol and knowledge-node ids share the same integer
// space (separate tables), so a raw id is ambiguous: the same numeric id can be
// both a symbol and a knowledge node. The edge's kind disambiguates the intent
// (reads/writes point at tables, configures points config->symbol, references
// points doc->doc or doc->symbol). When the kind names a knowledge table we
// probe that table first; otherwise a symbol is probed first. A zero id falls
// back to name-only unified resolution.
func resolveSpanEndpoint(ctx context.Context, db *sql.DB, id int64, name, edgeKind string) (spanNode, error) {
	if id != 0 {
		order := knowledgeOrderForKind(edgeKind)
		probe := func(kind string) (spanNode, bool) {
			if kind == "symbol" {
				s, err := loadSymbolByID(ctx, db, id)
				if err == nil {
					return spanNodeFromSymbol(s), true
				}
				return spanNode{}, false
			}
			n, err := loadSpanNodeByID(ctx, db, kind, id)
			if err == nil && n.Valid {
				return n, true
			}
			return spanNode{}, false
		}
		for _, k := range order {
			if n, ok := probe(k); ok {
				return n, nil
			}
		}
	}
	if name != "" {
		return resolveSpanNode(ctx, db, name)
	}
	return spanNode{}, nil
}

// knowledgeOrderForKind returns the endpoint probe order for an edge kind.
// Knowledge edges resolve their knowledge-side endpoint against the named table
// first (so a colliding symbol id is not mis-picked); 005 code edges probe the
// symbol first (unchanged baseline).
func knowledgeOrderForKind(kind string) []string {
	switch kind {
	case KindReads, KindWrites:
		return []string{"table", "column", "symbol"}
	case KindConfigures:
		return []string{"config", "symbol"}
	case KindReferences:
		return []string{"doc", "symbol"}
	default:
		// 005 code edge (calls/imports/extends/...): symbol first.
		return []string{"symbol", "doc", "config", "table", "column"}
	}
}

// PathSpan finds the shortest path between a (a) and b, where b is a name
// resolvable to a symbol OR a knowledge node (doc/config/table). It extends
// Path's BFS to the unified node space so the graph can span
// code→doc→config→table. When a and b are both symbols it is equivalent to
// Path (the 005 baseline). When no path exists it returns a where-the-graph-
// stops answer built from the unreachable frontier's name-only edges.
func PathSpan(ctx context.Context, db *sql.DB, a Symbol, bName string) (PathResult, error) {
	if a.ID == 0 {
		return PathResult{Found: false}, nil
	}
	return pathSpanFromSpanNode(ctx, db, spanNodeFromSymbol(a), bName)
}

// PathSpanFromName is PathSpan with the source also resolved by name, so a
// span can start from a knowledge node (doc/config/table) as well as a code
// symbol. The source name resolves through the unified resolver (symbol first,
// then knowledge node); an unresolvable source returns Found=false with a
// reason. This is the entry point code_path uses when its `from` is a doc,
// config, or table.
func PathSpanFromName(ctx context.Context, db *sql.DB, aName, bName string) (PathResult, error) {
	a, err := resolveSpanNode(ctx, db, aName)
	if err != nil {
		return PathResult{}, err
	}
	if !a.Valid {
		return PathResult{Found: false, Reason: "source not found: " + aName}, nil
	}
	if a.Kind != "symbol" {
		// Adapt a knowledge-node source into the Symbol-based PathSpan: start
		// the BFS from the knowledge node directly.
		return pathSpanFromSpanNode(ctx, db, a, bName)
	}
	return pathSpanFromSpanNode(ctx, db, a, bName)
}

// pathSpanFromSpanNode runs the BFS span from an already-resolved source node
// (a symbol or a knowledge node) to a target name.
func pathSpanFromSpanNode(ctx context.Context, db *sql.DB, a spanNode, bName string) (PathResult, error) {
	if !a.Valid {
		return PathResult{Found: false}, nil
	}
	b, err := resolveSpanNode(ctx, db, bName)
	if err != nil {
		return PathResult{}, err
	}
	if !b.Valid {
		return PathResult{Found: false, Reason: "target not found: " + bName}, nil
	}
	if b.Kind == a.Kind && b.ID == a.ID {
		return PathResult{Found: true, Path: []Edge{}}, nil
	}
	start := spanRef{kind: a.Kind, id: a.ID}
	target := spanRef{kind: b.Kind, id: b.ID}
	keyOf := func(r spanRef) string { return r.kind + "|" + itoa(r.id) }

	prev := map[string]spanRef{}
	visited := map[string]bool{keyOf(start): true}
	frontier := []spanRef{start}
	found := false
	for len(frontier) > 0 {
		var next []spanRef
		for _, cur := range frontier {
			curNode, err := loadSpanNodeByID(ctx, db, cur.kind, cur.id)
			if err != nil {
				return PathResult{}, err
			}
			hops, err := fetchSpanEdges(ctx, db, curNode)
			if err != nil {
				return PathResult{}, err
			}
			// Walk both directions (undirected): a knowledge edge stored as
			// config->code is traversable from code back to config.
			for _, h := range hops {
				var neighbors []spanNode
				if h.From.ID == curNode.ID {
					neighbors = []spanNode{h.To}
				} else if h.To.ID == curNode.ID {
					neighbors = []spanNode{h.From}
				} else {
					continue
				}
				for _, nb := range neighbors {
					nr := spanRef{kind: nb.Kind, id: nb.ID}
					if nr.kind == target.kind && nr.id == target.id {
						found = true
						prev[keyOf(nr)] = cur
						break
					}
					if !visited[keyOf(nr)] {
						visited[keyOf(nr)] = true
						prev[keyOf(nr)] = cur
						next = append(next, nr)
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			break
		}
		frontier = next
	}
	if !found {
		return spanGraphStops(ctx, db, visited), nil
	}
	var refs []spanRef
	for cur := target; keyOf(cur) != keyOf(start); cur = prev[keyOf(cur)] {
		refs = append([]spanRef{cur}, refs...)
	}
	full := append([]spanRef{start}, refs...)
	hops, err := spanHopsBetween(ctx, db, full)
	if err != nil {
		return PathResult{}, err
	}
	return PathResult{Found: true, Path: hops}, nil
}

// spanHopsBetween re-loads the Edge rows connecting consecutive span refs so
// the returned path carries each hop's kind + confidence (mirrors hopsBetween).
func spanHopsBetween(ctx context.Context, db *sql.DB, seq []spanRef) ([]Edge, error) {
	var out []Edge
	for i := 0; i+1 < len(seq); i++ {
		from, err := loadSpanNodeByID(ctx, db, seq[i].kind, seq[i].id)
		if err != nil {
			return nil, err
		}
		hops, err := fetchSpanEdges(ctx, db, from)
		if err != nil {
			return nil, err
		}
		var hop *spanHop
		for j := range hops {
			if hops[j].From.ID == seq[i].id && hops[j].To.ID == seq[i+1].id && hops[j].To.Kind == seq[i+1].kind {
				hop = &hops[j]
				break
			}
		}
		if hop == nil {
			// Reverse edge (the connection ran to->from).
			for j := range hops {
				if hops[j].From.ID == seq[i+1].id && hops[j].To.ID == seq[i].id && hops[j].From.Kind == seq[i+1].kind {
					hop = &hops[j]
					break
				}
			}
		}
		if hop != nil {
			out = append(out, Edge{
				Kind:       hop.Kind,
				Confidence: hop.Confidence,
				Line:       hop.Line,
				From:       spanToSymbol(hop.From),
				To:         spanToSymbol(hop.To),
				ToName:     hop.To.Name,
			})
		}
	}
	return out, nil
}

// spanToSymbol adapts a span node into a graph.Symbol for the PathResult
// payload: a symbol node maps 1:1; a knowledge node maps to a Symbol whose
// Name is the node's display name and Path its path (a non-code node is
// surfaced by name, never fabricated into a false symbol id).
func spanToSymbol(n spanNode) Symbol {
	if n.Kind == "symbol" {
		return Symbol{ID: n.ID, Name: n.Name, Path: n.Path, StartLine: n.Line, EndLine: n.Line}
	}
	return Symbol{ID: 0, Name: n.Name, Path: n.Path}
}

// spanGraphStops assembles a where-the-graph-stops answer for a span that did
// not reach its target, using the frontier's name-only edges (mirrors
// buildGraphStops but over the unified node space).
func spanGraphStops(ctx context.Context, db *sql.DB, visited map[string]bool) PathResult {
	stops := &GraphStops{DispatchKind: "computed member call"}
	worst := 0
	for k := range visited {
		r := splitSpanKey(k)
		node, err := loadSpanNodeByID(ctx, db, r.kind, r.id)
		if err != nil {
			continue
		}
		hops, err := fetchSpanEdges(ctx, db, node)
		if err != nil {
			continue
		}
		for _, h := range hops {
			if h.To.ID != 0 {
				continue
			}
			if h.From.ID != node.ID {
				continue
			}
			if h.Line > worst {
				worst = h.Line
			}
			stops.DispatchKind = h.Kind
			stops.Refused = append(stops.Refused, RefusedMatch{
				Name:       h.To.Name,
				Confidence: h.Confidence,
				Line:       h.Line,
				Path:       h.To.Path,
			})
		}
	}
	stops.Line = worst
	return PathResult{Found: false, GraphStops: stops}
}

// splitSpanKey parses a "kind|id" span key back into its parts.
func splitSpanKey(k string) spanRef {
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '|' {
			id, _ := parseItoa(k[i+1:])
			return spanRef{kind: k[:i], id: id}
		}
	}
	return spanRef{kind: k}
}
