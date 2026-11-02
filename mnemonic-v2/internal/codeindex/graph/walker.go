package graph

import (
	"context"
	"sort"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ImpactResult is the blast radius of a symbol, risk-tiered by depth.
type ImpactResult struct {
	WillBreak []ImpactEdge
	Likely    []ImpactEdge
	Excluded  int
}

// ImpactEdge is one hop in the blast radius.
type ImpactEdge struct {
	Symbol     store.SymbolRow
	Kind       string
	Confidence string
	Line       int
	Depth      int
	Risk       string
}

const (
	RiskWillBreak      = "WILL BREAK"
	RiskLikelyAffected = "LIKELY AFFECTED"
)

// visit is one node in the Impact BFS queue.
type visit struct {
	id    int64
	depth int
}

// Has reports whether the result contains a symbol with the given name.
func (r ImpactResult) Has(name string) bool {
	return r.HasWillBreak(name) || r.HasLikely(name)
}

// HasWillBreak reports whether the result contains a WILL BREAK symbol with the given name.
func (r ImpactResult) HasWillBreak(name string) bool {
	for _, e := range r.WillBreak {
		if e.Symbol.Name == name {
			return true
		}
	}
	return false
}

// HasLikely reports whether the result contains a LIKELY AFFECTED symbol with the given name.
func (r ImpactResult) HasLikely(name string) bool {
	for _, e := range r.Likely {
		if e.Symbol.Name == name {
			return true
		}
	}
	return false
}

// Impact returns the blast radius of symID, risk-tiered by depth, honoring minConfidence.
// It walks reverse edges (dependents): who calls/imports/extends the target.
// An unknown symbol (ID 0) returns an empty result.
func Impact(ctx context.Context, st store.Store, symID int64, minConfidence string, maxDepth int) (ImpactResult, error) {
	out := ImpactResult{}
	if symID == 0 {
		return out, nil
	}
	if maxDepth <= 0 {
		maxDepth = 5
	}

	// Reverse BFS over dependents: from a node, its dependents are the FROM
	// side of edges that TO this node.
	seen := map[int64]bool{symID: true}
	queue := []visit{{id: symID, depth: 0}}
	dependentsKinds := []string{"calls", "imports", "references", "extends", "implements"}

	// trailIDs accumulates the symbol ids the read touched, for the trail.
	var trailIDs []int64
	trailSeen := map[int64]bool{symID: true}
	addTrailID := func(id int64) {
		if id != 0 && !trailSeen[id] {
			trailSeen[id] = true
			trailIDs = append(trailIDs, id)
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= maxDepth {
			continue
		}

		// Get the current symbol's name (for name-only edge matching)
		curSym, err := st.SymbolByID(ctx, cur.id)
		if err != nil {
			continue
		}

		// Reverse dependents: edges where to_id = cur.id (resolved)
		edges, err := st.EdgesByTo(ctx, cur.id, dependentsKinds)
		if err != nil {
			// A read that failed is still a degraded retrieval (INV-5).
			RecordTrail(ctx, st, "code_impact", string(rune(symID)), "edges_by_to: "+err.Error(), trailIDs...)
			return out, err
		}

		// Also fetch name-only edges where to_name = curSym.Name
		nameOnlyEdges, err := st.EdgesByNameOnlyToName(ctx, curSym.Name, dependentsKinds)
		if err != nil {
			RecordTrail(ctx, st, "code_impact", string(rune(symID)), "name_only: "+err.Error(), trailIDs...)
			return out, err
		}

		// Process resolved edges
		for _, e := range edges {
			addTrailID(e.FromID)
			queue, _ = processEdge(ctx, st, &out, seen, queue, cur, e, minConfidence)
		}

		// Process name-only edges
		for _, e := range nameOnlyEdges {
			// A name-only edge is traversed only if it resolves to exactly one
			// symbol. CI-INV-3: a name-only edge with 0 or >1 candidates is
			// dropped, never guessed (drop-not-guess). This is the intended
			// reading of the v1 NOT-PORT tightening: the blast radius only
			// follows a name-only hop when the name is unambiguous.
			cands, err := st.SymbolByName(ctx, e.ToName)
			if err != nil || len(cands) != 1 || cands[0].ID != cur.id {
				continue // not a match, or 0/>1 candidates (CI-INV-3: drop, don't guess)
			}
			// This edge points to cur.id (name-only match)
			addTrailID(e.FromID)
			queue, _ = processEdge(ctx, st, &out, seen, queue, cur, e, minConfidence)
		}
	}

	sort.SliceStable(out.WillBreak, func(i, j int) bool {
		return out.WillBreak[i].Depth < out.WillBreak[j].Depth ||
			out.WillBreak[i].Symbol.Name < out.WillBreak[j].Symbol.Name
	})
	sort.SliceStable(out.Likely, func(i, j int) bool {
		return out.Likely[i].Depth < out.Likely[j].Depth ||
			out.Likely[i].Symbol.Name < out.Likely[j].Symbol.Name
	})

	// Record the trail with the returned symbol ids.
	RecordTrail(ctx, st, "code_impact", string(rune(symID)), "", trailIDs...)

	return out, nil
}

// processEdge processes a single edge in the Impact BFS. Returns the updated
// queue (may have the edge's from_id appended) and whether the edge was
// traversed (true) or excluded (false).
func processEdge(ctx context.Context, st store.Store, out *ImpactResult, seen map[int64]bool, queue []visit, cur visit, e store.EdgeRow, minConfidence string) ([]visit, bool) {
	// Confidence filter
	if !passesMinConfidence(e.Confidence, minConfidence) {
		out.Excluded++
		return queue, false
	}

	// Resolve the from_id to a symbol
	from, err := st.SymbolByID(ctx, e.FromID)
	if err != nil {
		return queue, false // dangling from_id
	}

	edge := ImpactEdge{
		Symbol:     from,
		Kind:       e.Kind,
		Confidence: e.Confidence,
		Line:       e.Line,
		Depth:      cur.depth + 1,
		Risk:       tierForDepth(cur.depth + 1),
	}
	if edge.Risk == RiskWillBreak {
		out.WillBreak = append(out.WillBreak, edge)
	} else {
		out.Likely = append(out.Likely, edge)
	}

	if !seen[e.FromID] {
		seen[e.FromID] = true
		queue = append(queue, visit{id: e.FromID, depth: cur.depth + 1})
	}

	return queue, true
}

// confidenceRank orders labels from strongest to weakest.
func confidenceRank(c string) int {
	switch c {
	case "EXTRACTED":
		return 3
	case "LSP_RESOLVED":
		return 3
	case "INFERRED":
		return 2
	case "AMBIGUOUS":
		return 1
	default:
		return 0
	}
}

// passesMinConfidence reports whether an edge's confidence clears the bar.
func passesMinConfidence(edgeConf, min string) bool {
	if min == "" {
		return true
	}
	return confidenceRank(edgeConf) >= confidenceRank(min)
}

// tierForDepth maps a BFS depth to a risk tier: direct dependents (depth 1)
// WILL BREAK, everything deeper LIKELY AFFECTED.
func tierForDepth(depth int) string {
	if depth == 1 {
		return RiskWillBreak
	}
	return RiskLikelyAffected
}
