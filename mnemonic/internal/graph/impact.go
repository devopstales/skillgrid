package graph

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// Risk tiers for blast radius, ordered by depth.
const (
	RiskWillBreak       = "WILL BREAK"
	RiskLikelyAffected  = "LIKELY AFFECTED"
)

// ImpactEdge is one confidence-tagged hop in the blast radius.
type ImpactEdge struct {
	Symbol     Symbol `json:"symbol"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
	Line       int    `json:"line"`
	Depth      int    `json:"depth"`
	Risk       string `json:"risk"`
}

// ImpactOptions tunes a blast-radius traversal.
type ImpactOptions struct {
	// MinConfidence filters low-confidence hops. EXTRACTED >= INFERRED >=
	// AMBIGUOUS; an empty string keeps everything.
	MinConfidence string
	// MaxDepth bounds the traversal (0 = unlimited, capped at a sane bound).
	MaxDepth int
	// SummaryOnly returns the tier counts without the per-edge list.
	SummaryOnly bool
}

// ImpactResult is the risk-tiered blast radius for one resolved symbol.
type ImpactResult struct {
	Target      Symbol        `json:"target"`
	WillBreak   []ImpactEdge  `json:"will_break"`
	Likely      []ImpactEdge  `json:"likely_affected"`
	Excluded    int           `json:"excluded_low_confidence"`
}

// confidenceRank orders labels from strongest to weakest.
func confidenceRank(c string) int {
	switch c {
	case ConfidenceExtracted:
		return 3
	case ConfidenceInferred:
		return 2
	case ConfidenceAmbiguous:
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

// Impact returns the blast radius of sym, risk-tiered by depth, every edge
// confidence-tagged, honoring opts.MinConfidence. It walks reverse edges
// (dependents): who calls/imports/extends the target. An unknown symbol (ID 0)
// returns an empty result (no invented edges).
func Impact(ctx context.Context, db *sql.DB, sym Symbol, opts ImpactOptions) (ImpactResult, error) {
	out := ImpactResult{Target: sym}
	if sym.ID == 0 {
		return out, nil
	}
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 5
	}

	// Reverse BFS over dependents: from a node, its dependents are the FROM
	// side of call/import edges that TO this node.
	type visit struct {
		id    int64
		depth int
	}
	seen := map[int64]bool{sym.ID: true}
	queue := []visit{{id: sym.ID, depth: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= maxDepth {
			continue
		}
		// Reverse dependents of cur.id: an edge points AT cur either by a
		// resolved id (e.to_id = cur.id — the step-01 resolved references
		// route->handler edges, and 005 calls/imports) or by a name-only
		// endpoint (e.to_id IS NULL AND e.to_name matches — 005 heritage /
		// navigates / unresolved). Both are traversed.
		// 'calls','imports','reference','route','extends','implements' are the
		// 005 heritage/call kinds (the singular 'reference' is 005's
		// import/reference kind, distinct from step-01's plural 'references').
		// 'references','navigates' are the step-01 framework-route edge kinds,
		// deliberately included for blast radius so a RESOLVED route->handler
		// references edge is traversed (a handler change's radius then includes
		// its serving route). Dropped (ambiguous) refs have no stored edge, so
		// they never appear here (drop-not-guess).
		rows, err := db.QueryContext(ctx, `
			SELECT e.from_id, e.kind, e.confidence, e.line
			FROM edges e
			WHERE (
				e.to_id = ?
				OR (e.to_id IS NULL AND e.to_name IN (SELECT name FROM symbols WHERE id = ?))
			  )
			  AND e.kind IN ('calls','imports','reference','route','extends','implements','references','navigates')
			ORDER BY e.id`, cur.id, cur.id)
		if err != nil {
			return out, err
		}
		type hop struct {
			fromID int64
			kind   string
			conf   string
			line   int
		}
		var hops []hop
		for rows.Next() {
			var h hop
			if err := rows.Scan(&h.fromID, &h.kind, &h.conf, &h.line); err != nil {
				rows.Close()
				return out, err
			}
			hops = append(hops, h)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, err
		}
		for _, h := range hops {
			if !passesMinConfidence(h.conf, opts.MinConfidence) {
				out.Excluded++
				continue
			}
			from, err := loadSymbolByID(ctx, db, h.fromID)
			if err != nil {
				continue
			}
			edge := ImpactEdge{
				Symbol:     from,
				Kind:       h.kind,
				Confidence: h.conf,
				Line:       h.line,
				Depth:      cur.depth + 1,
				Risk:       tierForDepth(cur.depth + 1),
			}
			if edge.Risk == RiskWillBreak {
				out.WillBreak = append(out.WillBreak, edge)
			} else {
				out.Likely = append(out.Likely, edge)
			}
			if !seen[h.fromID] {
				seen[h.fromID] = true
				queue = append(queue, visit{id: h.fromID, depth: cur.depth + 1})
			}
		}
	}
	sortStableImpact(out.WillBreak)
	sortStableImpact(out.Likely)
	return out, nil
}

func sortStableImpact(edges []ImpactEdge) {
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Depth != edges[j].Depth {
			return edges[i].Depth < edges[j].Depth
		}
		if edges[i].Symbol.Path != edges[j].Symbol.Path {
			return edges[i].Symbol.Path < edges[j].Symbol.Path
		}
		return edges[i].Line < edges[j].Line
	})
}

// Summary renders the blast-radius summary used by code_explore: tier counts
// plus the direct (will-break) dependents by name.
func (r ImpactResult) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "will_break: %d, likely_affected: %d", len(r.WillBreak), len(r.Likely))
	if r.Excluded > 0 {
		fmt.Fprintf(&b, ", excluded_low_confidence: %d", r.Excluded)
	}
	if len(r.WillBreak) > 0 {
		names := make([]string, 0, len(r.WillBreak))
		for _, e := range r.WillBreak {
			names = append(names, fmt.Sprintf("%s [%s]", e.Symbol.Name, e.Confidence))
		}
		sort.Strings(names)
		b.WriteString("\nwill-break dependents: " + strings.Join(names, ", "))
	}
	return b.String()
}
