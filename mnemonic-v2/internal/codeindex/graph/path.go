package graph

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// PathResult is the code_path contract (spec 004 §7).
// A PathResult is always something: a path, or a "the graph stops here" answer.
type PathResult struct {
	Found bool
	Path  []store.SymbolRow // the symbols in the path (from → to)
	Stops *GraphStops       // set when Found=false
}

// GraphStops is the "where the graph stops" answer.
type GraphStops struct {
	DispatchKind string
	Line         int
	Refused      []RefusedMatch
}

// RefusedMatch is one refused hop (a name-only edge that could not be resolved).
type RefusedMatch struct {
	Name       string
	Confidence string
	Line       int
}

// Path returns the BFS shortest path between fromID and toID (spec 004 §7).
// If no path exists, returns the "where the graph stops" answer.
func Path(ctx context.Context, st store.Store, fromID, toID int64) (PathResult, error) {
	// BFS from fromID to toID
	visited := map[int64]bool{fromID: true}
	parent := map[int64]int64{}
	queue := []int64{fromID}
	found := false

	for len(queue) > 0 && !found {
		cur := queue[0]
		queue = queue[1:]

		// Forward edges from cur
		edges, err := st.EdgesByFrom(ctx, cur, nil)
		if err != nil {
			return PathResult{}, err
		}

		for _, e := range edges {
			var nextID int64
			if e.ToID != nil {
				nextID = *e.ToID
			} else if e.ToName != "" {
				// Name-only: try to resolve
				cands, err := st.SymbolByName(ctx, e.ToName)
				if err != nil || len(cands) == 0 {
					continue // unresolved
				}
				if len(cands) > 1 {
					continue // ambiguous, skip
				}
				nextID = cands[0].ID
			} else {
				continue
			}

			if visited[nextID] {
				continue
			}
			visited[nextID] = true
			parent[nextID] = cur

			if nextID == toID {
				found = true
				break
			}
			queue = append(queue, nextID)
		}
	}

	if found {
		// Reconstruct the path
		var path []store.SymbolRow
		var pathIDs []int64
		for cur := toID; cur != 0; {
			sym, err := st.SymbolByID(ctx, cur)
			if err != nil {
				break
			}
			path = append([]store.SymbolRow{sym}, path...)
			pathIDs = append([]int64{sym.ID}, pathIDs...)
			if cur == fromID {
				break
			}
			cur = parent[cur]
		}
		// Record the trail with the path's symbol ids.
		RecordTrail(ctx, st, "code_path", string(rune(fromID))+":"+string(rune(toID)), "", pathIDs...)
		return PathResult{Found: true, Path: path}, nil
	}

	// No path found: build the graph-stops answer (a degraded read — the flow
	// could not be completed).
	stops := buildGraphStops(ctx, st, fromID, visited)
	RecordTrail(ctx, st, "code_path", string(rune(fromID))+":"+string(rune(toID)), "graph_stops")
	return PathResult{Found: false, Stops: stops}, nil
}

// buildGraphStops inspects the visited frontier and reports where the graph stops.
// The causal dispatch-kind fix: the interface/bus/callback at the FRONTIER,
// not last-iterated-over-visited.
func buildGraphStops(ctx context.Context, st store.Store, fromID int64, visited map[int64]bool) *GraphStops {
	stops := &GraphStops{DispatchKind: "unknown"}
	worst := 0

	// Find the frontier: visited nodes with unresolved forward edges
	for id := range visited {
		edges, err := st.EdgesByFrom(ctx, id, nil)
		if err != nil {
			continue
		}
		for _, e := range edges {
			// A forward edge whose target is unresolved (to_id NULL)
			if e.ToID != nil {
				continue
			}
			if e.ToName == "" {
				continue
			}
			// This is a refused hop
			if e.Line > worst {
				worst = e.Line
			}
			// Spec 004 §7 asks for "the dispatch kind that ended the flow."
			// The kind is unresolvable from the stored row: name-only edges
			// carry no target-kind signal, so this is a spec-underdetermined
			// default (the v1 lift's default), not a measured fact. "interface"
			// is used for any frontier edge of kind calls/references; a future
			// extraction tier that records the target's declared kind can
			// replace this. The test pins the FRONTIER LOCATION (the stop is at
			// the frontier edge, not a visited node — the v1 bug this fixes)
			// plus this kind value.
			if e.Kind == "calls" || e.Kind == "references" {
				stops.DispatchKind = "interface"
			}
			stops.Refused = append(stops.Refused, RefusedMatch{
				Name:       e.ToName,
				Confidence: e.Confidence,
				Line:       e.Line,
			})
		}
	}

	stops.Line = worst
	// Sort refused matches by name, then line
	for i := 1; i < len(stops.Refused); i++ {
		for j := i; j > 0 && (stops.Refused[j].Name < stops.Refused[j-1].Name ||
			(stops.Refused[j].Name == stops.Refused[j-1].Name && stops.Refused[j].Line < stops.Refused[j-1].Line)); j-- {
			stops.Refused[j], stops.Refused[j-1] = stops.Refused[j-1], stops.Refused[j]
		}
	}

	return stops
}
