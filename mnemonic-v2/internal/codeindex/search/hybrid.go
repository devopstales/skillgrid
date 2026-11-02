// Package search is the hybrid code search (spec 02-code-index 006): RRF
// K=60 fusion of the FTS leg (the floor, identifier-first — T-CI-1), the
// structural leg (index-free by-example AST grep + the fixed pattern language
// — T12), and the semantic leg (the embedder seam — degraded: no_vector).
// Per-hit provenance. T-CI-2: a leg that errors degrades + drops out of the
// fusion (its empty result never shapes the answer; the FTS floor still
// answers). Plus code_read (the live source span + secret redaction — T13) and
// the under? subtree scope (T10).
//
// Naming (M7): this package's SearchQuery/SearchHit/SearchResult are DISTINCT
// from store.SearchQuery (the memory change's) and store.SearchHit — no
// aliasing; both packages are imported where needed.
package search

import (
	"context"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/graph"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
)

// RRFK is the reciprocal-rank-fusion constant (the v1 constant, spec 006 §10).
const RRFK = 60

// defaultLimit is the code_search default limit (the v1 default, spec 006 §10).
const defaultLimit = 10

// SearchQuery is the code_search query (spec 006 §5). Distinct from
// store.SearchQuery (the memory change's) — no aliasing.
type SearchQuery struct {
	Query    string
	Project  string
	Limit    int // default 10
	Language string
	Under    string // the optional files.path package prefix (T10)
	// Embedder is the semantic leg's seam. Nil → degraded: no_vector (the FTS
	// floor still answers). A present-but-erroring embedder → T-CI-2 degrade +
	// drop out of the fusion.
	Embedder embedder.Embedder
}

// Hit is one ranked hybrid result with per-signal provenance. Provenance maps
// a leg name ("fts"/"structural"/"semantic") to that leg's 1-based rank for
// this hit (the agent can see WHY a hit ranked where — spec 006 §5).
type Hit struct {
	Path       string
	StartLine  int
	EndLine    int
	Symbol     string
	Kind       string
	Score      float64
	Provenance map[string]int
	// StoreID is the store row id the hit resolved to (the trail's ids, per
	// the TICKET-04 convention). 0 when the hit has no store id (a
	// structural-only hit for an unindexed file — the path is its identity;
	// the trail documents that case by recording the hitKeyID hash instead).
	StoreID int64
}

// SearchResult is the code_search answer. Degraded is the explicit
// degradation flag (INV-3 — never silent); Warnings are the per-leg notes.
type SearchResult struct {
	Query    string
	Under    string
	Hits     []Hit
	Degraded string
	Warnings []string
}

// legHit is one leg's ranked hit (before fusion). ID is the fusion key (a
// doc is one hit when the same doc appears in multiple legs). StoreID is the
// store row id the hit resolved to (the FTS/semantic leg's symbols.id — the
// trail records these, per the TICKET-04 trail convention; 0 when the hit has
// no store id, e.g. a structural-only hit for an unindexed file).
type legHit struct {
	ID        string
	Path      string
	StartLine int
	EndLine   int
	Symbol    string
	Kind      string
	StoreID   int64
}

// Hybrid runs the hybrid search: the FTS leg (always), the structural leg
// (when the query is a pattern), and the semantic leg (when the embedder is
// up), fused by RRF K=60 over the legs that returned. Per-hit provenance.
// T-CI-2: a leg that errors degrades + drops out of the fusion (the FTS floor
// still answers). The under? prefix (T10) narrows all three legs to a
// subtree; an empty-scope prefix returns empty (not an error).
func Hybrid(ctx context.Context, st store.Store, q SearchQuery) (SearchResult, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if q.Query == "" {
		return SearchResult{Query: q.Query, Under: q.Under, Hits: []Hit{}}, nil
	}

	res := SearchResult{Query: q.Query, Under: q.Under, Hits: []Hit{}}
	hits := map[string]legHit{}
	legs := map[string]map[string]int{} // leg name → (doc id → 1-based rank)
	degraded := ""

	// Leg 1: the FTS floor (always runs).
	fts, ftsErr := ftsLeg(ctx, st, q)
	if ftsErr != nil {
		// T-CI-2: the FTS leg is the floor — if IT errors, the search is
		// degraded but the result is still (possibly empty) + not a hard fail.
		res.Warnings = append(res.Warnings, "fts leg: "+ftsErr.Error())
		degraded = "fts_error"
	}
	if ftsErr == nil {
		for i, h := range fts {
			hits[h.ID] = h
			legs["fts"] = rankOrAppend(legs["fts"], h.ID, i+1)
		}
	}

	// Leg 2: the structural leg (when the query is a pattern).
	if isPattern(q.Query) {
		sres, sErr := Grep(ctx, q.Query, "", q.Language)
		if sErr != nil {
			res.Warnings = append(res.Warnings, "structural leg: "+sErr.Error())
			degraded = appendDegraded(degraded, "structural_error")
		} else {
			// A pattern that is invalid for a language degrades (never a
			// silent skip) — the leg is present but flagged.
			if sres.Degraded != "" {
				degraded = appendDegraded(degraded, sres.Degraded)
			}
			// under-scope filter (T10): keep only hits under the prefix.
			for i, gh := range sres.Hits {
				if !underMatch(gh.Path, q.Under) {
					continue
				}
				h := legHit{
					ID:        "grep:" + gh.Path + ":" + itoa(gh.Line),
					Path:      gh.Path,
					StartLine: gh.Line,
					EndLine:   gh.Line,
					Kind:      "structural",
				}
				hits[h.ID] = h
				legs["structural"] = rankOrAppend(legs["structural"], h.ID, i+1)
			}
		}
	}

	// Leg 3: the semantic leg (when the embedder is up).
	if q.Embedder == nil {
		// Absent, not failing: the leg is simply absent (degraded: no_vector).
		// The FTS floor still answers.
		degraded = appendDegraded(degraded, "no_vector")
	} else if q.Embedder.Model() == "" {
		// A Null embedder (model "") is also "absent".
		degraded = appendDegraded(degraded, "no_vector")
	} else {
		sem, semErr := semanticLeg(ctx, st, q)
		if semErr != nil {
			// T-CI-2: a failing embedder degrades + drops out of the fusion —
			// the broken leg's empty result NEVER shapes the answer.
			res.Warnings = append(res.Warnings, "semantic leg: "+semErr.Error())
			degraded = appendDegraded(degraded, "semantic_error")
		} else {
			for i, h := range sem {
				hits[h.ID] = h
				legs["semantic"] = rankOrAppend(legs["semantic"], h.ID, i+1)
			}
		}
	}

	// RRF fusion: score(doc) = Σ 1/(K + rank) over the legs that returned it.
	// A doc in more legs ranks higher.
	res.Hits = fuse(hits, legs, RRFK, limit)
	res.Degraded = degraded

	// Record the retrieval trail (INV-5 — route "search", the query, the
	// returned hit ids, the degraded string, the under scope in the query).
	// F4: the trail ids carry the STORE ROW IDs the hits resolved to (the
	// symbol ids, per the TICKET-04 trail convention — the audit answer to
	// "which rows did this read return?"). A hit with no store id (a
	// structural-only hit for an unindexed file) contributes a deterministic
	// hash of its path:line doc key — the path IS the identity for unindexed
	// files (the trail convention: store ids when present, the doc key hash
	// when not).
	var trailIDs []int64
	for _, h := range res.Hits {
		if h.StoreID != 0 {
			trailIDs = append(trailIDs, h.StoreID)
			continue
		}
		trailIDs = append(trailIDs, hitKeyID(h.Path+":"+itoa(h.StartLine)))
	}
	query := q.Query
	if q.Under != "" {
		query = q.Query + " under:" + q.Under
	}
	graph.RecordTrail(ctx, st, "search", query, res.Degraded, trailIDs...)

	return res, nil
}

// fuse runs the RRF fusion over the legs' ranked hits: score(doc) =
// Σ 1/(K + rank) over the legs that returned it. A doc in more legs ranks
// higher. Returns the top-limit hits with per-signal provenance.
func fuse(hits map[string]legHit, legs map[string]map[string]int, k, limit int) []Hit {
	if k <= 0 {
		k = RRFK
	}
	type scored struct {
		id    string
		score float64
	}
	var out []scored
	for id := range hits {
		score := 0.0
		for _, ranks := range legs {
			if r, ok := ranks[id]; ok {
				score += 1.0 / float64(k+r)
			}
		}
		if score > 0 {
			out = append(out, scored{id: id, score: score})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].id < out[j].id
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	ranked := make([]Hit, 0, len(out))
	for _, s := range out {
		h := hits[s.id]
		hit := Hit{
			Path:       h.Path,
			StartLine:  h.StartLine,
			EndLine:    h.EndLine,
			Symbol:     h.Symbol,
			Kind:       h.Kind,
			Score:      s.score,
			Provenance: map[string]int{},
			StoreID:    h.StoreID,
		}
		for leg, ranks := range legs {
			if r, ok := ranks[s.id]; ok {
				hit.Provenance[leg] = r
			}
		}
		ranked = append(ranked, hit)
	}
	return ranked
}

// rankOrAppend sets the 1-based rank for a doc in a leg's rank map (the first
// rank wins — a doc appearing twice in the same leg keeps its best rank).
func rankOrAppend(m map[string]int, id string, rank int) map[string]int {
	if m == nil {
		m = map[string]int{}
	}
	if _, ok := m[id]; !ok {
		m[id] = rank
	}
	return m
}

// appendDegraded appends a degradation reason to the (comma-joined) degraded
// string, avoiding duplicates.
func appendDegraded(cur, reason string) string {
	if reason == "" {
		return cur
	}
	if cur == "" {
		return reason
	}
	if strings.Contains(","+cur+",", ","+reason+",") {
		return cur
	}
	return cur + "," + reason
}

// underMatch reports whether path is under the prefix (the files.path subtree
// scope, T10). An empty prefix matches everything. A prefix matches the exact
// path or a path with the prefix as a directory segment (internal/pay matches
// internal/pay and internal/pay/x.go, not internal/payment).
func underMatch(path, under string) bool {
	if under == "" {
		return true
	}
	under = strings.TrimSuffix(under, "/")
	if path == under {
		return true
	}
	return strings.HasPrefix(path, under+"/")
}

// isPattern reports whether a query is a structural pattern (a by-example AST
// grep) rather than a free-text/identifier query. A pattern contains a
// metavariable (\f, \name, \(, \*) or a node-type S-expression (a leading
// '('). The v1 CLI's code_grep accepted both forms; code_search treats a
// pattern query as a structural-leg query (fused with the FTS leg).
func isPattern(q string) bool {
	if strings.Contains(q, "\\") {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(q), "(")
}

// hitKeyID derives a stable trail id from a hit's path+line key (the trail
// records the returned hits' ids — INV-5; the ids column is a comma-separated
// list).
func hitKeyID(id string) int64 {
	h := int64(0)
	for i := 0; i < len(id); i++ {
		h = h*31 + int64(id[i])
	}
	return h
}

// errString is a tiny error type for the legs' sentinel errors (avoids a
// fmt.Errorf allocation for a static message).
type errString string

func (e errString) Error() string { return string(e) }

// itoa is a small int→string (avoids a strconv import for one call site).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
