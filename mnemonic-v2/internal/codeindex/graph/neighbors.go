package graph

import (
	"context"
	"sort"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// Neighbors is a confidence-labeled edge view for a symbol.
type Neighbors struct {
	Edges []NeighborEdge
}

// NeighborEdge is one edge in a neighbor view, with the target resolved.
type NeighborEdge struct {
	Kind       string
	Confidence string
	Line       int
	From       store.SymbolRow
	To         store.SymbolRow // zero value when unresolved
	ToName     string          // the literal name when unresolved
}

// callerKinds are the edge kinds the callers/callees views match.
var callerKinds = map[string]bool{
	"calls": true, "imports": true, "references": true,
}

// dependentKinds are the edge kinds the dependents view matches.
var dependentKinds = map[string]bool{
	"calls": true, "imports": true, "references": true, "extends": true, "implements": true,
}

// hierarchyKinds are the edge kinds the hierarchy view matches.
var hierarchyKinds = map[string]bool{
	"extends": true, "implements": true,
}

// testKinds are the edge kinds the tests_for view matches.
var testKinds = map[string]bool{
	"tests_for": true, "tested_by": true,
}

// Callee returns the forward calls/imports edges for symID.
func Callee(ctx context.Context, st store.Store, symID int64) (Neighbors, error) {
	return viewEdges(ctx, st, symID, viewCallees)
}

// Callers returns the reverse calls/imports/references edges for symID.
func Callers(ctx context.Context, st store.Store, symID int64) (Neighbors, error) {
	return viewEdges(ctx, st, symID, viewCallers)
}

// Dependents returns the reverse (all dependent kinds) edges for symID.
func Dependents(ctx context.Context, st store.Store, symID int64) (Neighbors, error) {
	return viewEdges(ctx, st, symID, viewDependents)
}

// Implementors returns the reverse implements edges for symID.
func Implementors(ctx context.Context, st store.Store, symID int64) (Neighbors, error) {
	return viewEdges(ctx, st, symID, viewImplementors)
}

// Hierarchy returns the extends/implements edges (both directions) for symID.
func Hierarchy(ctx context.Context, st store.Store, symID int64) (Neighbors, error) {
	return viewEdges(ctx, st, symID, viewHierarchy)
}

// TestsFor returns the tests_for/tested_by edges for symID.
func TestsFor(ctx context.Context, st store.Store, symID int64) (Neighbors, error) {
	return viewEdges(ctx, st, symID, viewTestsFor)
}

type view int

const (
	viewCallees view = iota
	viewCallers
	viewDependents
	viewImplementors
	viewHierarchy
	viewTestsFor
)

func viewEdges(ctx context.Context, st store.Store, symID int64, v view) (Neighbors, error) {
	if symID == 0 {
		return Neighbors{Edges: []NeighborEdge{}}, nil
	}

	// Fetch all edges involving this symbol
	edges, err := fetchEdges(ctx, st, symID)
	if err != nil {
		return Neighbors{}, err
	}

	// Filter by view
	var out []NeighborEdge
	for _, e := range edges {
		if viewMatches(v, e, symID) {
			out = append(out, e)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Kind < out[j].Kind
	})

	// Record the trail
	route := "code_neighbors"
	switch v {
	case viewCallees:
		route = "code_callees"
	case viewCallers:
		route = "code_callers"
	case viewDependents:
		route = "code_dependents"
	case viewImplementors:
		route = "code_implementors"
	case viewHierarchy:
		route = "code_hierarchy"
	case viewTestsFor:
		route = "code_tests_for"
	}
	RecordTrail(ctx, st, route, string(rune(symID)), "")

	return Neighbors{Edges: out}, nil
}

func viewMatches(v view, e NeighborEdge, symID int64) bool {
	switch v {
	case viewCallers:
		return e.From.ID != symID && callerKinds[e.Kind]
	case viewCallees:
		return e.From.ID == symID && callerKinds[e.Kind]
	case viewDependents:
		return e.From.ID != symID && dependentKinds[e.Kind]
	case viewImplementors:
		return e.Kind == "implements"
	case viewHierarchy:
		return hierarchyKinds[e.Kind]
	case viewTestsFor:
		return testKinds[e.Kind]
	default:
		return true
	}
}

// fetchEdges fetches all edges involving symID, with query-time name-only relabel.
func fetchEdges(ctx context.Context, st store.Store, symID int64) ([]NeighborEdge, error) {
	// Get the symbol's name (for name-only edge matching)
	sym, err := st.SymbolByID(ctx, symID)
	if err != nil {
		return nil, err
	}

	// Fetch forward edges (from_id = symID)
	forward, err := st.EdgesByFrom(ctx, symID, nil)
	if err != nil {
		return nil, err
	}
	// Fetch reverse edges (to_id = symID)
	reverse, err := st.EdgesByTo(ctx, symID, nil)
	if err != nil {
		return nil, err
	}
	// Fetch name-only reverse edges (to_name = sym.Name)
	nameOnlyReverse, err := st.EdgesByNameOnlyToName(ctx, sym.Name, nil)
	if err != nil {
		return nil, err
	}

	var out []NeighborEdge
	for _, e := range forward {
		out = append(out, resolveEdge(ctx, st, e, symID))
	}
	for _, e := range reverse {
		var toID int64
		if e.ToID != nil {
			toID = *e.ToID
		}
		out = append(out, resolveEdge(ctx, st, e, toID))
	}
	// Process name-only reverse edges
	for _, e := range nameOnlyReverse {
		// Include if the name matches the current symbol's name
		if e.ToName != sym.Name {
			continue
		}
		out = append(out, resolveEdge(ctx, st, e, e.FromID))
	}
	return out, nil
}

// resolveEdge resolves an edge's endpoints, applying the query-time name-only relabel.
func resolveEdge(ctx context.Context, st store.Store, e store.EdgeRow, fromID int64) NeighborEdge {
	// Resolve from
	var from store.SymbolRow
	if fromID != 0 {
		from, _ = st.SymbolByID(ctx, fromID)
	}

	// Resolve to
	var candidates []store.SymbolRow
	if e.ToID != nil {
		if to, err := st.SymbolByID(ctx, *e.ToID); err == nil {
			candidates = append(candidates, to)
		}
	} else if e.ToName != "" {
		cands, err := st.SymbolByName(ctx, e.ToName)
		if err == nil {
			candidates = cands
		}
	}

	ne := NeighborEdge{
		Kind:       e.Kind,
		Confidence: e.Confidence,
		Line:       e.Line,
		From:       from,
		ToName:     e.ToName,
	}

	switch len(candidates) {
	case 1:
		ne.To = candidates[0]
	case 0:
		// Unresolved: keep the literal name
	default:
		// Ambiguous: emit one AMBIGUOUS edge per candidate
		// For now, mark as AMBIGUOUS and set To to the first candidate
		// (the caller can detect this by checking Confidence)
		ne.Confidence = "AMBIGUOUS"
		ne.To = candidates[0]
	}

	return ne
}
