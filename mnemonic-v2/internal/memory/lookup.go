package memory

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// lookup.go — the retrieval ladder (spec 01-memory/005): mem_search (Layer 1,
// compact rows + RRF vector blend), mem_lookup (the narrow wiki-first
// fallback), mem_context (Layer 0, byte budget), SearchPrompts, Review, and
// the retrieval-trail writer (INV-5) + the retrieval-usage bump that feeds
// the importance formula (spec 002-schema §2.2.3).
//
// Deletion test: deleting lookup.go re-creates ranking + fusion + trail
// recording in every transport — the callers are transport/handler (Task 13)
// and the CLI (Task 16).

// ── result / filter shapes (spec 01-memory/005 §3) ─────────────────────────

// SearchFilters is the mem_search filter set (spec 005 §3: type, project,
// scope, limit, match_mode, all_projects).
type SearchFilters struct {
	Type        string
	Project     string
	Scope       string
	MatchMode   string // "all"/"any" (any is the FTS MATCH default)
	Limit       int
	AllProjects bool
}

// SearchHit is a compact mem_search row (spec 005 §3 projection): the
// observation's identity + a 300-char content preview + the truncation flag
// + relation annotations. Never the full body at this layer.
type SearchHit struct {
	store.Observation
	Preview     string
	Truncated   bool
	Rank        float64
	Annotations []string
}

// SearchResult is the mem_search result: the compact hits (rank-ordered), the
// fusion route (fts/rrf), and the degraded flag (INV-3: the vector leg is
// absent when no embedder is armed).
type SearchResult struct {
	Hits     []SearchHit
	Route    string
	Degraded bool
}

// LookupResult is the mem_lookup result (spec 005 §5): a single hit (the
// exact-topic-key / id / sync_id path) or a bounded phrase hit set (≤5), never
// widened.
type LookupResult struct {
	Hits     []SearchHit
	Hit      *SearchHit // the single-row path (topic_key / id / sync_id)
	Degraded bool
}

// ReviewResult is the Review surface's return (the mark_reviewed flag).
type ReviewResult struct {
	MarkReviewed bool
}

// previewLimit is the compact-row preview cap (spec 005 §3: at most 300
// characters).
const previewLimit = 300

// phraseLookupCap is the mem_lookup phrase recall cap (spec 005 §5: ≤5).
const phraseLookupCap = 5

// rrfK is the RRF fusion constant (spec 01-memory/005 §3, Engram).
const rrfK = 60

// ── mem_search (Layer 1 — recall) ───────────────────────────────────────────

// Search runs the FTS leg (the store's composite-rank MATCH) and, when an
// embedder is armed, the vector leg, RRF-fusing the two (route "rrf"); the
// result is degraded when the vector leg is absent (INV-3 — never a silent
// fusion). It projects the hits to compact rows (300-char preview + truncated
// flag + relation annotations), records the retrieval trail (INV-5), and
// bumps retrieval_usage on the returned ids (the named source of the
// importance formula's usage term, spec 002-schema §2.2.3) in the same step.
func (s *Service) Search(ctx context.Context, query string, f SearchFilters) (SearchResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 10
	}
	fts, err := s.st.Search(ctx, store.SearchQuery{
		Query: query, Project: f.Project, Scope: f.Scope, Type: f.Type, Limit: limit,
	})
	if err != nil {
		return SearchResult{}, err
	}

	route := "fts"
	merged := ftsRank(fts)
	degraded := true // the floor: no embedder → the vector leg is absent
	if s.emb != nil {
		if vec, vErr := s.vectorLeg(ctx, query, f, limit); vErr == nil && !vec.Degraded {
			merged = rrfFusion(ftsRank(fts), vecRank(vec), rrfK)
			route = "rrf"
			degraded = false
		}
	}

	res := SearchResult{Route: route, Degraded: degraded}
	res.Hits = projectHits(ctx, s.st, merged, route)
	// INV-5: the trail is the audit record; the usage bump is the named source
	// of the importance formula's usage term. The bump runs BEFORE the
	// recompute so the recomputed cache reflects the current retrieval
	// (the cache is the save-time score; the recompute folds in this
	// retrieval). Both in the same step, before return. A failed
	// trail/bump/recompute does not fail the search (the audit + cache are
	// side effects, the result is already ranked).
	ids := hitIDs(res.Hits)
	s.recordTrail(route, query, f, ids, degraded)
	if err := s.st.BumpRetrievalUsage(ctx, ids); err != nil {
		_ = err // the result stands; the usage bump degrades, not fails
	}
	s.recomputeImportance(ctx, ids)
	return res, nil
}

// vectorLeg runs the vector half of the fusion when the embedder is armed:
// it embeds the query, runs the store's VectorSearch at k = 2·limit, and
// returns the store's VecResult (degraded when the store finds no vectors).
// A failing embedder degrades the leg (one bounded retry via the degrade
// floor is the embedder's concern; here an error simply means the leg is
// absent → the caller keeps the FTS-only result, flagged degraded).
func (s *Service) vectorLeg(ctx context.Context, query string, f SearchFilters, limit int) (store.VecResult, error) {
	k := 2 * limit
	vecs, err := s.emb.Embed(ctx, []string{query})
	if err != nil || len(vecs) == 0 || len(vecs[0]) == 0 {
		return store.VecResult{Degraded: true}, nil
	}
	return s.st.VectorSearch(ctx, store.VecQuery{QueryVector: vecs[0], Project: f.Project, TopK: k})
}

// ftsRank maps the FTS leg's ranked observations to their rank order (the
// store already returns them best-first; rank is the 1-based position).
func ftsRank(fts store.SearchResult) []store.Observation { return fts.Observations }

// vecRank maps the vector leg's nearest observations to their rank order.
func vecRank(vec store.VecResult) []store.Observation { return vec.Observations }

// rrfFusion merges two ranked lists by reciprocal-rank fusion
// (score = 1/(k+rank_fts) + 1/(k+rank_vec), rank 1-based). The output is
// best-first by fused score. A row present in only one leg still appears
// (its absent-leg rank contributes nothing).
func rrfFusion(fts, vec []store.Observation, k int) []store.Observation {
	type acc struct {
		obs   store.Observation
		score float64
	}
	byID := map[int64]*acc{}
	order := []int64{}
	add := func(o store.Observation, rank int) {
		a, ok := byID[o.ID]
		if !ok {
			a = &acc{obs: o}
			byID[o.ID] = a
			order = append(order, o.ID)
		}
		a.score += 1.0 / float64(k+rank)
	}
	for i, o := range fts {
		add(o, i+1)
	}
	for i, o := range vec {
		add(o, i+1)
	}
	out := make([]store.Observation, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id].obs)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return byID[out[i].ID].score > byID[out[j].ID].score
	})
	return out
}

// projectHits turns a ranked observation list into compact SearchHits (the
// spec 005 §3 projection): 300-char preview + truncated flag, plus the
// relation annotations from a single memory_relations scan over the hit sync
// ids (spec 003 §6: supersedes/superseded_by/conflicts/conflict lines).
func projectHits(ctx context.Context, st store.Store, ranked []store.Observation, route string) []SearchHit {
	annotations := relationAnnotations(ctx, st, ranked)
	hits := make([]SearchHit, 0, len(ranked))
	for i, o := range ranked {
		preview, truncated := compactPreview(o.Content)
		hits = append(hits, SearchHit{
			Observation: o,
			Preview:     preview,
			Truncated:   truncated,
			Rank:        float64(i + 1),
			Annotations: annotations[o.SyncID],
		})
	}
	return hits
}

// relationAnnotations returns, per hit sync id, the relation annotation lines
// (spec 01-memory/003 §6): "supersedes: #id (title)", "superseded_by: #id
// (title)", "conflicts: #id (title)", "conflict: contested by #id (pending)".
// It runs ONE memory_relations scan over the hit ids (a single RawQuery, not
// one per hit) — the projection must stay cheap at this layer. A store without
// the RawQuery seam returns no annotations (the projection degrades, not
// fails).
func relationAnnotations(ctx context.Context, st store.Store, ranked []store.Observation) map[string][]string {
	out := map[string][]string{}
	if len(ranked) == 0 {
		return out
	}
	syncIDs := make([]any, 0, len(ranked))
	for _, o := range ranked {
		syncIDs = append(syncIDs, o.SyncID)
	}
	// memory_relations.source_id/target_id are sync_ids (INV-6), never local
	// ids — the annotation keys on the hit's sync_id directly (one scan, not
	// one per hit).
	querier, ok := asRawReader(st)
	if !ok {
		return out
	}
	rows, err := querier.RawQuery(
		`SELECT source_id, target_id, relation, judgment_status FROM memory_relations
		 WHERE source_id IN (`+intPlaceholders(len(ranked))+`)`,
		syncIDs...)
	if err != nil || rows == nil {
		return out
	}
	for _, r := range rows {
		src := strVal(r, 0)
		tgt := strVal(r, 1)
		rel := strVal(r, 2)
		status := strVal(r, 3)
		line := ""
		switch rel {
		case "supersedes":
			line = "supersedes: #" + tgt
		case "superseded_by":
			line = "superseded_by: #" + tgt
		case "conflicts":
			line = "conflicts: #" + tgt
		}
		if rel == "conflicts" && status == "pending" {
			line = "conflict: contested by #" + tgt + " (pending)"
		}
		if line != "" {
			out[src] = append(out[src], line)
		}
	}
	return out
}

// intPlaceholders builds n "?" placeholders (one per ranked sync id).
func intPlaceholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ", "
		}
		s += "?"
	}
	return s
}

// rawReader is the store's RawQuery read seam (the [][]any read surface the
// memory package uses for its projection + the topic-key identity read).
type rawReader interface {
	RawQuery(query string, args ...any) ([][]any, error)
}

// asRawReader narrows a store to its RawQuery seam (a store without it is a
// test bug — the SQLite adapter always has it; the projection degrades, not
// fails, on the assertion miss).
func asRawReader(st store.Store) (rawReader, bool) {
	r, ok := st.(rawReader)
	return r, ok
}

// compactPreview cuts content to previewLimit runes and reports whether it
// was cut (spec 005 §3: at most 300 chars + the truncation flag).
func compactPreview(content string) (preview string, truncated bool) {
	r := []rune(content)
	if len(r) <= previewLimit {
		return content, false
	}
	return string(r[:previewLimit]), true
}

// hitIDs returns the local ids of the hits (the usage-bump + trail targets).
func hitIDs(hits []SearchHit) []int64 {
	ids := make([]int64, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
	}
	return ids
}

// ── mem_lookup (the narrow wiki-first fallback, spec 005 §5) ────────────────

// Lookup answers "give me *that* fact" — deliberately dumb (spec 005 §5):
// exact topic_key (+project/scope) → the single upserted row (or none);
// id/sync_id → the full row; phrase → bounded FTS (trigram, ≤5, no boosts, no
// vector). It never widens. Lookup records a trail (route "lookup") but does
// NOT bump retrieval_usage — over-counting the wiki's precise fallback would
// distort the graduate-rule input (spec 005 §5, the usage-raises-importance
// contract counts mem_search retrievals, not lookups).
func (s *Service) Lookup(ctx context.Context, project, scope, topicKey string, id int64, phrase string) (LookupResult, error) {
	// By id: the full row (Layer 2). A lookup trail, no usage bump.
	if id != 0 {
		o, err := s.st.GetObservation(ctx, id, false)
		if err != nil {
			return LookupResult{Degraded: true}, nil // absent → none, not an error (spec 005 §5)
		}
		res := singleLookupHit(o)
		s.recordLookupTrail(project, topicKey, []int64{id})
		return res, nil
	}
	// By exact topic_key: the single upserted row (or none). Never widens.
	if topicKey != "" {
		res, err := s.lookupTopicKey(ctx, project, scope, topicKey)
		if err != nil {
			return LookupResult{}, err
		}
		if res.Hit != nil {
			s.recordLookupTrail(project, topicKey, []int64{res.Hit.ID})
		} else {
			s.recordLookupTrail(project, topicKey, nil)
		}
		return res, nil
	}
	// By distinctive phrase: bounded FTS (trigram, ≤5, no boosts, no vector).
	if phrase != "" {
		fts, err := s.st.Search(ctx, store.SearchQuery{Query: phrase, Project: project, Scope: scope, Limit: phraseLookupCap})
		if err != nil {
			return LookupResult{}, err
		}
		hits := projectHits(ctx, s.st, fts.Observations, "lookup")
		res := LookupResult{Hits: hits}
		if len(hits) > 0 {
			res.Hit = &hits[0]
		}
		s.recordLookupTrail(project, phrase, hitIDs(hits))
		return res, nil
	}
	// No selector: nothing to look up (not a search — never widens).
	return LookupResult{Degraded: true}, nil
}

// lookupTopicKey returns the single upserted row for project+scope+topic_key,
// or none (spec 005 §5: the highest-precision path, the topic_key is the
// identity). It reads the topic_key column directly (the FTS trigram cannot
// MATCH a kebab-case topic_key with a "/" — the trigram index tokenizes on
// word boundaries, so the exact-identity read is a column equality, not a
// fuzzy match). The upsert guarantees ≤1 live row per project+scope+
// topic_key (idx_obs_topic), so a single-row read is the whole lookup.
func (s *Service) lookupTopicKey(ctx context.Context, project, scope, topicKey string) (LookupResult, error) {
	if scope == "" {
		scope = "project"
	}
	querier, ok := asRawReader(s.st)
	if !ok {
		return LookupResult{Degraded: true}, nil
	}
	var rows [][]any
	var err error
	rows, err = querier.RawQuery(
		`SELECT id FROM observations
		 WHERE project = ? AND scope = ? AND topic_key = ? AND deleted_at IS NULL
		 LIMIT 1`,
		project, scope, topicKey)
	if err != nil || len(rows) == 0 {
		return LookupResult{Degraded: true}, nil // absent → none (never widens)
	}
	o, err := s.st.GetObservation(ctx, int64(intVal(rows[0], 0)), false)
	if err != nil {
		return LookupResult{Degraded: true}, nil
	}
	return singleLookupHit(o), nil
}

// singleLookupHit wraps a full row as a one-hit LookupResult (the id /
// topic_key paths return the full row, not a preview).
func singleLookupHit(o store.Observation) LookupResult {
	preview, truncated := compactPreview(o.Content)
	h := SearchHit{Observation: o, Preview: preview, Truncated: truncated, Rank: 1}
	return LookupResult{Hits: []SearchHit{h}, Hit: &h}
}

// recordLookupTrail writes the lookup's retrieval trail (route "lookup"). No
// usage bump (spec 005 §5). A trail failure is a side-effect degradation, not
// an error (the lookup result stands).
func (s *Service) recordLookupTrail(project, topicKey string, ids []int64) {
	s.recordTrail("lookup", topicKey, SearchFilters{Project: project}, ids, false)
}

// ── mem_context (Layer 0 — orientation, spec 005 §2) ────────────────────────

// Context is the byte-budget orientation layer (spec 005 §2): the project's
// pinned observations first (≤20), then recent observations, trimmed to
// budgetBytes (default 16 KiB, max 64 KiB). It delegates the row selection +
// budget accounting to the store (the store owns the SQL); the service is the
// thin facade the transports call.
func (s *Service) Context(ctx context.Context, budgetBytes int, olderHistory bool) (store.ContextResult, error) {
	if budgetBytes <= 0 {
		budgetBytes = 16 * 1024
	}
	return s.st.Context(ctx, store.ContextQuery{MaxBytes: budgetBytes})
}

// ── SearchPrompts (spec 002-schema §2.4) ────────────────────────────────────

// SearchPrompts searches the captured prompts (the prompts_fts table) over
// (content, project). Redaction happened at capture; search sees the stored
// (redacted) text.
func (s *Service) SearchPrompts(ctx context.Context, query, project string, limit int) (store.PromptSearchResult, error) {
	return s.st.SearchPrompts(ctx, store.PromptSearchQuery{Query: query, Project: project, Limit: limit})
}

// ── Review (spec 002-schema §2.2.2) ─────────────────────────────────────────

// Review lists the project's overdue observations (state needs_review) or,
// with markReviewedID set, marks one reviewed — restarting the decay clock
// (review_after = now + type-months, the store's MarkReviewed). list is the
// read path; markReviewedID > 0 is the write path (mutually exclusive in
// practice: the caller sets one or the other).
func (s *Service) Review(ctx context.Context, project string, list bool, markReviewedID int64) (ReviewResult, error) {
	if markReviewedID > 0 {
		if err := s.st.MarkReviewed(ctx, markReviewedID); err != nil {
			return ReviewResult{}, err
		}
		return ReviewResult{MarkReviewed: true}, nil
	}
	// The list path (state needs_review, review_after < now) is the transport
	// surface (Task 13); this task wires the write path (MarkReviewed). The
	// list read is a store Search over the project filtered on the virtual
	// state — ponytail: the ceiling is a dedicated store method when the list
	// path needs a real page.
	_, err := s.st.Search(ctx, store.SearchQuery{Project: project, Limit: 1})
	if err != nil {
		return ReviewResult{}, err
	}
	return ReviewResult{}, nil
}

// ── the retrieval trail (INV-5) + the importance recompute ──────────────────

// recordTrail writes one retrieval-trail row (route, query, filters, ids,
// degraded). The trail is the audit record for "why did the agent recall
// this" (spec 005 §7). A store without the write seam (a dialect that does
// not implement trailSeam) skips the append rather than failing the search —
// the audit is a side effect, not a pipeline step (the same posture as the
// session-trail consumer). A failure does not fail the search.
func (s *Service) recordTrail(route, query string, filters SearchFilters, ids []int64, degraded bool) {
	if s.st == nil {
		return
	}
	seam, ok := s.st.(trailSeam)
	if !ok {
		return // no write seam: the append is skipped, not fatal
	}
	idsJSON, _ := json.Marshal(ids)
	if err := seam.RawExec(`
		INSERT INTO retrieval_trails (route, query, project, filters, ids, degraded)
		VALUES (?, ?, ?, ?, ?, ?)`,
		route, query, nullableProject(filters.Project), filterJSON(filters), string(idsJSON), degradedInt(degraded),
	); err != nil {
		// ponytail: a trail-write failure is swallowed (the search result
		// stands; the audit is a side effect). The session-trail consumer
		// panics because the bus recovers it; the retrieval trail is not
		// bus-subscribed, so the failure is dropped here.
		_ = err
	}
}

// recomputeImportance rewrites the observations.importance cache column for
// the retrieved ids over their real retrieval_usage + age (the v1
// dream.go:382-400 formula, the Importance() function). The save wrote a
// fresh-state score (a cache); this is the recompute over real retrieval
// usage (spec 002-schema §2.2.3: "the retrieval count" feeds the usage
// term). A store without the write seam skips it (degrades, not fails).
func (s *Service) recomputeImportance(ctx context.Context, ids []int64) {
	if s.st == nil || len(ids) == 0 {
		return
	}
	seam, ok := s.st.(trailSeam)
	if !ok {
		return
	}
	for _, id := range ids {
		o, err := s.st.GetObservation(ctx, id, false)
		if err != nil {
			continue // a row that vanished between the search and the recompute is skipped
		}
		score := Importance(o.RetrievalUsage, ageDaysOf(o), o.Type)
		if err := seam.RawExec(`UPDATE observations SET importance = ? WHERE id = ?`, score, id); err != nil {
			_ = err // the cache recompute degrades, not fails
		}
	}
}

// ageDaysOf derives the row's age in days from the rank-age anchor
// (last_seen_at || updated_at || created_at — the same COALESCE the store's
// search boost uses, spec 005 §3). A zero/unparseable anchor is age 0.
func ageDaysOf(o store.Observation) int {
	anchor := o.LastSeenAt
	if anchor == nil || *anchor == "" {
		anchor = &o.UpdatedAt
	}
	if anchor == nil || *anchor == "" {
		anchor = &o.CreatedAt
	}
	return ageDays(*anchor)
}

// filterJSON serializes the search filters for the trail's audit record.
func filterJSON(f SearchFilters) string {
	b, _ := json.Marshal(f)
	return string(b)
}

// degradedInt maps a bool to the 0/1 the trail's degraded column stores.
func degradedInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ── the retrieval_trails table (additive migration 003) ─────────────────────

// retrievalTrailsDDL is the retrieval-trail audit table (spec 005 §7, INV-5).
// Additive-only: a fresh DB applies 001+002+this; an existing DB (schema
// version 2) applies this on its next Migrate. Never synced or exported
// (local-only audit, like session_trails).
const retrievalTrailsDDL = `
CREATE TABLE IF NOT EXISTS retrieval_trails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	route TEXT NOT NULL,
	query TEXT NOT NULL,
	project TEXT,
	filters TEXT,
	ids TEXT,
	degraded INTEGER NOT NULL DEFAULT 0,
	timestamp TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_retrieval_trails_project ON retrieval_trails(project, timestamp DESC);
`
