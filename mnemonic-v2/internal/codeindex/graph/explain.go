package graph

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ExplainResult is the code_explain contract (spec 004 §8).
type ExplainResult struct {
	Symbol      store.SymbolRow
	Degree      int
	Connections []ExplainedEdge
}

// ExplainedEdge is one connection in the explain result, ranked by the
// neighbor's degree (hubs first).
type ExplainedEdge struct {
	Edge           store.EdgeRow
	Neighbor       store.SymbolRow
	NeighborDegree int
}

// Explain returns the node + its degree + all connections ranked by the
// neighbor's degree (hubs first, spec 004 §8).
func Explain(ctx context.Context, st store.Store, symID int64) (ExplainResult, error) {
	res := ExplainResult{}

	sym, err := st.SymbolByID(ctx, symID)
	if err != nil {
		return res, err
	}
	res.Symbol = sym

	// Get all edges involving this symbol
	forward, err := st.EdgesByFrom(ctx, symID, nil)
	if err != nil {
		return res, err
	}
	reverse, err := st.EdgesByTo(ctx, symID, nil)
	if err != nil {
		return res, err
	}

	// Collect all connections
	type conn struct {
		edge store.EdgeRow
		toID int64
	}
	var conns []conn
	seen := map[int64]bool{}

	for _, e := range forward {
		var toID int64
		if e.ToID != nil {
			toID = *e.ToID
		} else if e.ToName != "" {
			cands, err := st.SymbolByName(ctx, e.ToName)
			if err == nil && len(cands) == 1 {
				toID = cands[0].ID
			}
		}
		if toID != 0 && !seen[toID] {
			seen[toID] = true
			conns = append(conns, conn{edge: e, toID: toID})
		}
	}

	for _, e := range reverse {
		if e.FromID != 0 && !seen[e.FromID] {
			seen[e.FromID] = true
			conns = append(conns, conn{edge: e, toID: e.FromID})
		}
	}

	res.Degree = len(conns)

	// Compute the neighbor's degree for each connection
	for _, c := range conns {
		neighbor, err := st.SymbolByID(ctx, c.toID)
		if err != nil {
			continue
		}
		// Compute the neighbor's degree
		fwd, _ := st.EdgesByFrom(ctx, c.toID, nil)
		rev, _ := st.EdgesByTo(ctx, c.toID, nil)
		neighborDegree := len(fwd) + len(rev)

		res.Connections = append(res.Connections, ExplainedEdge{
			Edge:           c.edge,
			Neighbor:       neighbor,
			NeighborDegree: neighborDegree,
		})
	}

	// Sort by neighbor degree (hubs first)
	for i := 1; i < len(res.Connections); i++ {
		for j := i; j > 0 && res.Connections[j].NeighborDegree > res.Connections[j-1].NeighborDegree; j-- {
			res.Connections[j], res.Connections[j-1] = res.Connections[j-1], res.Connections[j]
		}
	}

	// Record the trail with the symbol + neighbor ids.
	var trailIDs []int64
	trailSeen := map[int64]bool{}
	addID := func(id int64) {
		if id != 0 && !trailSeen[id] {
			trailSeen[id] = true
			trailIDs = append(trailIDs, id)
		}
	}
	addID(sym.ID)
	for _, c := range res.Connections {
		addID(c.Neighbor.ID)
	}
	RecordTrail(ctx, st, "code_explain", sym.Name, "", trailIDs...)

	return res, nil
}
