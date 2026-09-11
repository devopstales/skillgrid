// Package graph walks the indexed symbols/edges tables: confidence-labeled
// neighbor views, BFS shortest paths with a "where the graph stops" answer,
// symbol explanation, risk-tiered blast radius, and measured fair coverage.
package graph

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// Confidence labels (mirrored from extract for the graph layer).
const (
	ConfidenceExtracted = "EXTRACTED"
	ConfidenceInferred  = "INFERRED"
	ConfidenceAmbiguous = "AMBIGUOUS"
)

// Edge is one stored edge, confidence-labeled, with the endpoints resolved to
// their symbols.
type Edge struct {
	Kind       string
	Confidence string
	Line       int
	From       Symbol
	To         Symbol
	// ToName is the literal target name when the edge is name-only (to_id is
	// null in the store) or was refused by disambiguation.
	ToName string
	// TargetPath is the import/dependency target path when present.
	TargetPath string
}

// Symbol is a resolved node.
type Symbol struct {
	ID            int64
	UID           string
	Name          string
	QualifiedName string
	Kind          string
	Language      string
	Path          string
	StartLine     int
	EndLine       int
}

// View selects which neighbors of a symbol to return.
type View string

const (
	ViewCallers      View = "callers"
	ViewCallees      View = "callees"
	ViewDependents   View = "dependents"
	ViewImplementors View = "implementors"
	ViewHierarchy    View = "hierarchy"
	ViewTestsFor     View = "tests_for"
	ViewAll          View = "all"
)

// callerKinds are edge kinds where a reverse edge (other -> symbol) means the
// other symbol depends on / calls the target.
var callerKinds = map[string]bool{
	"calls":   true,
	"imports": true,
}

// dependentKinds are forward edge kinds from the symbol (symbol -> other).
var dependentKinds = map[string]bool{
	"calls":   true,
	"imports": true,
}

// hierarchyKinds are structural relationships in either direction.
var hierarchyKinds = map[string]bool{
	"extends":    true,
	"implements": true,
}

// testKinds mark test-for edges in either direction.
var testKinds = map[string]bool{
	"tests_for": true,
	"tested_by": true,
}

// resolveSymbol loads a symbol by primary key (tests) or falls back to
// name-based resolution via Resolve.
func resolveSymbol(ctx context.Context, db *sql.DB, id int64) (Symbol, error) {
	var s Symbol
	err := db.QueryRowContext(ctx, `
		SELECT s.id, s.uid, s.name, s.qualified_name, s.kind, s.language,
		       f.path, s.start_line, s.end_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.id = ?`, id).
		Scan(&s.ID, &s.UID, &s.Name, &s.QualifiedName, &s.Kind, &s.Language,
			&s.Path, &s.StartLine, &s.EndLine)
	if err != nil {
		return Symbol{}, err
	}
	return s, nil
}

// loadSymbolByID resolves a symbol row from an edges endpoint.
func loadSymbolByID(ctx context.Context, db *sql.DB, id int64) (Symbol, error) {
	return resolveSymbol(ctx, db, id)
}

// loadSymbolByName returns every symbol whose name equals name, lowest id
// first. Name-only endpoints (to_id null) resolve here; ambiguity is a
// property of the caller (disambiguation), not a silent pick.
func loadSymbolsByName(ctx context.Context, db *sql.DB, name string) ([]Symbol, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.uid, s.name, s.qualified_name, s.kind, s.language,
		       f.path, s.start_line, s.end_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.name = ?
		ORDER BY s.id`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.ID, &s.UID, &s.Name, &s.QualifiedName, &s.Kind,
			&s.Language, &s.Path, &s.StartLine, &s.EndLine); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// fetchEdges loads every CURRENT edge touching the symbol, resolving both
// endpoints. It applies the temporal filter (014 step 10, change
// 014-mnemonic-performance): expired edges (valid_to <= now) and pending
// edges (valid_from > now) are hidden from traversal, so the code_*
// neighbor/explain/path tools see only active relationships. Backfilled
// pre-migration rows (valid_from = 0) pass the filter and stay visible.
// Edges whose name-only endpoint resolves to multiple symbols are emitted
// once per candidate with Confidence AMBIGUOUS (never a silent drop). Edges
// whose endpoint name resolves to nothing keep their ToName for the
// "where the graph stops" answer.
func fetchEdges(ctx context.Context, db *sql.DB, sym Symbol) ([]Edge, error) {
	now := time.Now().Unix()
	rows, err := db.QueryContext(ctx, `
		SELECT e.id, e.kind, e.from_id, e.to_name, e.to_id, e.target_path,
		       e.confidence, e.line
		FROM edges e
		WHERE (e.from_id = ? OR e.to_id = ?)
		  AND e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)
		ORDER BY e.id`, sym.ID, sym.ID, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Edge
	for rows.Next() {
		var (
			id, fromID int64
			kind, conf string
			line       int
			toName     sql.NullString
			targetPath sql.NullString
			toID       sql.NullInt64
		)
		if err := rows.Scan(&id, &kind, &fromID, &toID, &toName, &targetPath, &conf, &line); err != nil {
			return nil, err
		}
		from, err := loadSymbolByID(ctx, db, fromID)
		if err != nil {
			return nil, err
		}
		var candidates []Symbol
		if toID.Valid {
			if to, err := loadSymbolByID(ctx, db, toID.Int64); err == nil {
				candidates = append(candidates, to)
			}
		} else if n := toName.String; n != "" {
			cands, err := loadSymbolsByName(ctx, db, n)
			if err != nil {
				return nil, err
			}
			candidates = cands
		}
		tp := targetPath.String
		switch {
		case len(candidates) == 1:
			out = append(out, Edge{
				Kind: kind, Confidence: conf, Line: line,
				From: from, To: candidates[0],
				ToName: toName.String, TargetPath: tp,
			})
		case len(candidates) > 1:
			// Ambiguous name-only resolution: emit every candidate as an
			// AMBIGUOUS edge instead of dropping the edge or guessing.
			for _, c := range candidates {
				out = append(out, Edge{
					Kind: kind, Confidence: ConfidenceAmbiguous, Line: line,
					From: from, To: c,
					ToName: toName.String, TargetPath: tp,
				})
			}
		default:
			// Unresolved: keep the literal name so the graph-stops answer can
			// report it; no fabricated hop.
			out = append(out, Edge{
				Kind: kind, Confidence: conf, Line: line,
				From: from,
				ToName: toName.String, TargetPath: tp,
			})
		}
	}
	return out, rows.Err()
}

// resolvedEndpoints returns (from, to, fromCandidates, toCandidates) for a raw
// edge row, resolving each endpoint by id first and falling back to name-only
// resolution (a name matching several symbols yields multiple candidates).
// This is the shared endpoint-resolution used by fetchEdges and Impact.
func resolvedEndpoints(ctx context.Context, db *sql.DB, fromID int64, toID *int64, toName string) (from, to Symbol, toCands []Symbol, err error) {
	from, err = loadSymbolByID(ctx, db, fromID)
	if err != nil {
		return Symbol{}, Symbol{}, nil, err
	}
	if toID != nil && *toID != 0 {
		if t, e2 := loadSymbolByID(ctx, db, *toID); e2 == nil {
			return from, t, nil, nil
		}
	}
	if toName != "" {
		cands, e2 := loadSymbolsByName(ctx, db, toName)
		if e2 != nil {
			return from, Symbol{}, nil, e2
		}
		if len(cands) == 1 {
			return from, cands[0], nil, nil
		}
		return from, Symbol{}, cands, nil
	}
	return from, Symbol{}, nil, nil
}

func viewMatches(view View, e Edge, sym Symbol) bool {
	switch view {
	case ViewCallers:
		return e.From.ID != sym.ID && callerKinds[e.Kind]
	case ViewCallees:
		return e.From.ID == sym.ID && callerKinds[e.Kind]
	case ViewDependents:
		return e.From.ID != sym.ID && dependentKinds[e.Kind]
	case ViewImplementors:
		return e.Kind == "implements"
	case ViewHierarchy:
		return hierarchyKinds[e.Kind]
	case ViewTestsFor:
		return testKinds[e.Kind]
	default:
		return true
	}
}

// Neighbors returns the symbol's edges for view, every edge confidence-
// labeled. An unknown symbol (ID 0) returns no edges (no fabrication).
func Neighbors(ctx context.Context, db *sql.DB, sym Symbol, view View) ([]Edge, error) {
	if sym.ID == 0 {
		return []Edge{}, nil
	}
	edges, err := fetchEdges(ctx, db, sym)
	if err != nil {
		return nil, err
	}
	var out []Edge
	for _, e := range edges {
		if viewMatches(view, e, sym) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Kind < out[j].Kind
	})
	return out, nil
}

// GraphStops describes where a static traversal died: the dispatch kind that
// ended the flow, the line, and the name-only matches refused (with their
// confidence).
type GraphStops struct {
	DispatchKind string `json:"dispatch_kind"`
	Line         int    `json:"line"`
	Refused      []RefusedMatch `json:"refused"`
}

// RefusedMatch is a name-only match the traversal refused to follow.
type RefusedMatch struct {
	Name       string `json:"name"`
	Confidence string `json:"confidence"`
	Line       int    `json:"line"`
	Path       string `json:"path,omitempty"`
}

// PathResult is the code_path answer: either a shortest path or a
// where-the-graph-stops explanation. Never empty, never fabricated.
type PathResult struct {
	Found      bool        `json:"found"`
	Path       []Edge      `json:"path,omitempty"`
	GraphStops *GraphStops `json:"graph_stops,omitempty"`
	// Reason is a short human-readable note when Found is false and no
	// GraphStops frontier exists (e.g. the target name resolved to nothing).
	Reason string `json:"reason,omitempty"`
}

// Path finds the shortest edge path between a and b (BFS over resolved
// endpoints, ignoring confidence for connectivity). When no static path
// exists it returns a GraphStops answer built from the unreachable frontier's
// name-only edges: the dispatch kind that ended the flow, its line, and the
// refused name-only matches with their confidence.
func Path(ctx context.Context, db *sql.DB, a, b Symbol) (PathResult, error) {
	if a.ID == 0 || b.ID == 0 {
		return PathResult{Found: false}, nil
	}
	if a.ID == b.ID {
		return PathResult{Found: true, Path: []Edge{}}, nil
	}
	prev := map[int64]int64{}
	visited := map[int64]bool{a.ID: true}
	frontier := []int64{a.ID}
	found := false
	for len(frontier) > 0 {
		var next []int64
		for _, id := range frontier {
			edges, err := fetchEdges(ctx, db, mustSymbol(ctx, db, id))
			if err != nil {
				return PathResult{}, err
			}
			for _, e := range edges {
				if e.To.ID == 0 {
					continue
				}
				if e.To.ID == b.ID {
					found = true
					prev[b.ID] = id
					break
				}
				if !visited[e.To.ID] {
					visited[e.To.ID] = true
					prev[e.To.ID] = id
					next = append(next, e.To.ID)
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
		return buildGraphStops(ctx, db, a, visited), nil
	}
	// Reconstruct the path by walking prev backwards and re-loading each hop.
	var ids []int64
	for cur := b.ID; cur != a.ID; cur = prev[cur] {
		ids = append([]int64{cur}, ids...)
	}
	hops, err := hopsBetween(ctx, db, a.ID, ids)
	if err != nil {
		return PathResult{}, err
	}
	return PathResult{Found: true, Path: hops}, nil
}

// hopsBetween re-loads the actual Edge rows connecting consecutive ids so the
// returned path carries each hop's kind + confidence.
func hopsBetween(ctx context.Context, db *sql.DB, startID int64, ids []int64) ([]Edge, error) {
	var out []Edge
	seq := append([]int64{startID}, ids...)
	for i := 0; i+1 < len(seq); i++ {
		from, err := loadSymbolByID(ctx, db, seq[i])
		if err != nil {
			return nil, err
		}
		edges, err := fetchEdges(ctx, db, from)
		if err != nil {
			return nil, err
		}
		var hop *Edge
		for j := range edges {
			if edges[j].From.ID == seq[i] && edges[j].To.ID == seq[i+1] {
				hop = &edges[j]
				break
			}
		}
		if hop == nil {
			// Reverse edge (the connection ran to->from).
			for j := range edges {
				if edges[j].From.ID == seq[i+1] && edges[j].To.ID == seq[i] {
					hop = &edges[j]
					break
				}
			}
		}
		if hop != nil {
			out = append(out, *hop)
		}
	}
	return out, nil
}

// buildGraphStops assembles the where-the-graph-stops answer from the
// frontier's unresolved name-only edges: the dispatch kind that ended the
// flow (a call hop by default), its line, and every refused name-only match
// with its confidence.
func buildGraphStops(ctx context.Context, db *sql.DB, start Symbol, visited map[int64]bool) PathResult {
	stops := &GraphStops{DispatchKind: "computed member call"}
	worst := 0
	for id := range visited {
		sym, err := loadSymbolByID(ctx, db, id)
		if err != nil {
			continue
		}
		edges, err := fetchEdges(ctx, db, sym)
		if err != nil {
			continue
		}
		for _, e := range edges {
			if e.To.ID != 0 {
				continue
			}
			if e.From.ID != sym.ID {
				continue
			}
			if e.Line > worst {
				worst = e.Line
			}
			stops.DispatchKind = e.Kind
			stops.Refused = append(stops.Refused, RefusedMatch{
				Name:       e.ToName,
				Confidence: e.Confidence,
				Line:       e.Line,
				Path:       e.TargetPath,
			})
		}
	}
	stops.Line = worst
	sort.SliceStable(stops.Refused, func(i, j int) bool {
		if stops.Refused[i].Name != stops.Refused[j].Name {
			return stops.Refused[i].Name < stops.Refused[j].Name
		}
		return stops.Refused[i].Line < stops.Refused[j].Line
	})
	return PathResult{Found: false, GraphStops: stops}
}

// mustSymbol loads a symbol for BFS expansion.
func mustSymbol(ctx context.Context, db *sql.DB, id int64) Symbol {
	s, _ := loadSymbolByID(ctx, db, id)
	return s
}

// ExplainResult is the code_explain answer: the node, its degree, and every
// connection ranked by the neighbor's degree (hubs first).
type ExplainResult struct {
	Symbol      Symbol   `json:"symbol"`
	Degree      int      `json:"degree"`
	Connections []Conn   `json:"connections"`
}

// Conn is one connection of the explained symbol.
type Conn struct {
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
	Line       int    `json:"line"`
	Symbol     Symbol `json:"symbol"`
	ToName     string `json:"to_name,omitempty"`
	TargetPath string `json:"target_path,omitempty"`
	Degree     int    `json:"degree"`
}

// degreeOf counts distinct symbols connected to id.
func degreeOf(ctx context.Context, db *sql.DB, id int64) int {
	var n int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT other) FROM (
			SELECT to_id AS other FROM edges WHERE from_id = ? AND to_id IS NOT NULL
			UNION
			SELECT from_id AS other FROM edges WHERE to_id = ?
		)`, id, id).Scan(&n)
	return n
}

// Explain returns the symbol, its degree, and all connections ranked by the
// neighbor's degree (descending), then name. Unknown symbol (ID 0) returns a
// zero result with no invented edges.
func Explain(ctx context.Context, db *sql.DB, sym Symbol) (ExplainResult, error) {
	out := ExplainResult{Symbol: sym}
	if sym.ID == 0 {
		return out, nil
	}
	edges, err := fetchEdges(ctx, db, sym)
	if err != nil {
		return out, err
	}
	out.Degree = degreeOf(ctx, db, sym.ID)
	seen := map[string]bool{}
	for _, e := range edges {
		target := e.To
		if e.From.ID == sym.ID && e.To.ID != sym.ID {
			target = e.To
		}
		if e.From.ID != sym.ID && e.To.ID != sym.ID {
			target = e.To
		}
		key := fmt.Sprintf("%d|%s", target.ID, e.ToName)
		if target.ID == 0 {
			key = "name|" + e.ToName + "|" + e.Kind
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Connections = append(out.Connections, Conn{
			Kind:       e.Kind,
			Confidence: e.Confidence,
			Line:       e.Line,
			Symbol:     target,
			ToName:     e.ToName,
			TargetPath: e.TargetPath,
			Degree:     degreeOf(ctx, db, target.ID),
		})
	}
	sort.SliceStable(out.Connections, func(i, j int) bool {
		if out.Connections[i].Degree != out.Connections[j].Degree {
			return out.Connections[i].Degree > out.Connections[j].Degree
		}
		if out.Connections[i].Symbol.Name != out.Connections[j].Symbol.Name {
			return out.Connections[i].Symbol.Name < out.Connections[j].Symbol.Name
		}
		return out.Connections[i].Line < out.Connections[j].Line
	})
	return out, nil
}
