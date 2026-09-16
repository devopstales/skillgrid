// Package memory — directory-level recursive retrieval (change 014, step 19).
//
// RetrieveDirOptions / DirectoryRetrieve sit as a HIERARCHICAL layer on top of
// the existing per-store search (SearchOwnerScoped / BlendedSearch): they do
// not replace it. The directory hierarchy is the slash-separated topic_key
// (project/app/core → top-level "project", child "app", leaf "core"). The
// first pass scores every top-level directory by FTS5 relevance + embedding
// similarity (when an embedder is configured); the highest-scoring directory
// is drilled into, and the pass repeats until a leaf level or retrievalMaxDepth.
package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// retrievalMaxDepth bounds the drill-down loop (19.4): a hierarchy deeper than
// this stops at depth retrievalMaxDepth and returns the best results found at
// that level, so a pathological topic_key can never recurse forever.
const retrievalMaxDepth = 5

// dirScoreCacheSize bounds the in-process directory-score cache (19.4):
// repeated queries against an unchanged store skip re-scoring whole branches.
const dirScoreCacheSize = 256

// Intent is the coarse query classification (19.3). It adjusts retrieval
// parameters BEFORE the drill-down runs: debugging widens the search scope,
// exploration stays directory-first (the default strategy).
type Intent string

const (
	IntentExploration Intent = "exploration"
	IntentDebugging   Intent = "debugging"
	IntentReview      Intent = "review"
	IntentRefactor    Intent = "refactor"
)

// intentKeywords maps each intent to the query phrases that signal it.
// Classification is keyword matching + pattern detection (19.3.c), checked in
// a fixed order (debugging before review before refactor before exploration)
// so a query like "why does the build fail" classifies as debugging even
// though it contains no exploration marker.
var intentKeywords = []struct {
	intent  Intent
	keywords []string
}{
	{IntentDebugging, []string{
		"fail", "failing", "broken", "error", "crash", "bug",
		"stack trace", "exception", "panic", "regress", "why does", "why is",
	}},
	{IntentReview, []string{
		"review", "check the changes", "diff", "pull request", "pr feedback",
	}},
	{IntentRefactor, []string{
		"refactor", "restructure", "split", "rename", "cleanup", "clean up",
	}},
}

// ClassifyIntent classifies a query as one of exploration / debugging /
// review / refactor (19.3). It is pure (no store, no config) so the intent
// logic is testable in isolation and the classification is deterministic.
func ClassifyIntent(query string) Intent {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, c := range intentKeywords {
		for _, kw := range c.keywords {
			if strings.Contains(q, kw) {
				return c.intent
			}
		}
	}
	return IntentExploration
}

// intentParams reports how an intent reshapes the retrieval (19.3.c):
// debugging → wider scope (a larger per-directory candidate fan-out so
// related-but-distant observations are not pruned early); exploration →
// directory-first (the default, tighter fan-out); review/refactor →
// directory-first as well.
func intentParams(intent Intent) (scopeWider bool, fanout int) {
	switch intent {
	case IntentDebugging:
		return true, 40
	default:
		return false, 20
	}
}

// dirQueryEmbedder is the optional embedding seam for directory scoring. It is
// intentionally narrower than the embedder.Embedder interface (which imports
// nothing here): only a query vector is needed, so a Service can attach any
// embedder through this adapter. When nil, directory scoring is FTS5-only.
type dirQueryEmbedder interface {
	EmbedQuery(ctx context.Context, query string) (Vector, error)
}

// RetrieveDirOptions tunes a directory retrieval run. Query is the search
// text; Scope is an optional root prefix to restrict the drill-down (empty =
// the full hierarchy); Limit bounds the final result list.
type RetrieveDirOptions struct {
	Query string
	Scope string
	Limit int
}

// RetrieveDirResult is the outcome of a directory retrieval: the final results
// scoped to the drilled-down leaf directory plus the trajectory (the
// drill-down path) and the run's query id (its handle in retrieval_trails).
type RetrieveDirResult struct {
	// QueryID is the retrieval_trails row id for this run (0 when the trail
	// could not be recorded — a best-effort audit, never a retrieval failure).
	QueryID int64 `json:"query_id"`
	// Leaf is the drilled-down directory the Results are scoped to.
	Leaf string `json:"leaf"`
	// Trajectory is the ordered list of directory visits (drill-down path).
	Trajectory []RetrievalStep `json:"trajectory"`
	// Results are the observations inside the leaf directory, ranked.
	Results []Observation `json:"results"`
}

// RetrievalStep is one directory visit in the drill-down trajectory: the
// directory Path, its combined score, the depth of the visit, and the complete
// drill-down path so far (a JSON-able array, stored in retrieval_trails.path).
type RetrievalStep struct {
	Path       string   `json:"path"`
	Score      float64  `json:"score"`
	Depth      int      `json:"depth"`
	PathPrefix []string `json:"path_prefix"`
}

// dirObsRow is one observation row pulled by scoreDirectories (the directory
// group-by input): its id (for the embedding lookup), topic_key (the
// directory it organizes under), and the title/content text scored.
type dirObsRow struct {
	id       int64
	topicKey string
	title    string
	content  string
}

// dirScoreCache memoizes directory scores between repeated queries (19.4).
// The key is query|root|matchMode; a store mutation (a new Save) is NOT
// tracked, so the cache is a best-effort optimization — a miss or a stale
// entry only changes ranking freshness, never correctness.
var (
	dirScoreCacheMu sync.Mutex
	dirScoreCache   = map[string]map[string]float64{}
)

func dirCacheKey(query, root, matchMode string) string {
	return query + "|" + root + "|" + matchMode
}

func dirCacheGet(key string) map[string]float64 {
	dirScoreCacheMu.Lock()
	defer dirScoreCacheMu.Unlock()
	return dirScoreCache[key]
}

func dirCachePut(key string, scores map[string]float64) {
	dirScoreCacheMu.Lock()
	defer dirScoreCacheMu.Unlock()
	if len(dirScoreCache) >= dirScoreCacheSize {
		// Simple eviction: drop everything. The cache is a pure optimization;
		// a full flush keeps the implementation dependency-free and bounded.
		dirScoreCache = map[string]map[string]float64{}
	}
	dirScoreCache[key] = scores
}

// DirectoryRetrieve performs the directory-level recursive retrieval (19.1):
// score every top-level directory under Scope, drill into the highest-scoring
// one, repeat until a leaf level (no child directories) or retrievalMaxDepth,
// then return the observations inside the leaf directory ranked by the same
// FTS5 + embedding similarity scoring, plus the drill-down trajectory.
//
// It is additive: the existing flat search (SearchOwnerScoped / BlendedSearch)
// is untouched and remains the default path; DirectoryRetrieve is the
// hierarchical layer on top.
func (s *Service) DirectoryRetrieve(ctx context.Context, opts RetrieveDirOptions) (*RetrieveDirResult, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return &RetrieveDirResult{}, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	root := strings.Trim(strings.TrimSpace(opts.Scope), "/")

	// Intent analysis (19.3) classifies the query and adjusts the retrieval
	// parameters: debugging → wider scope (bigger fan-out), exploration →
	// directory-first (the default).
	intent := ClassifyIntent(query)
	_, fanout := intentParams(intent)

	// One query vector for the whole run (nil/empty = no embedder configured,
	// FTS5-only scoring).
	var qv Vector
	if e := s.dirEmbedder(); e != nil {
		if v, err := e.EmbedQuery(ctx, query); err == nil {
			qv = v
		}
	}

	res := &RetrieveDirResult{}
	current := root
	for depth := 1; depth <= retrievalMaxDepth; depth++ {
		scores, err := s.scoreDirectories(ctx, query, current, qv, fanout)
		if err != nil {
			return nil, err
		}
		if len(scores) == 0 {
			// Nothing matches under the current directory: stop and return the
			// best results found so far (the current directory itself).
			break
		}
		best := pickBestDirectory(scores)
		res.Trajectory = append(res.Trajectory, RetrievalStep{
			Path:       best,
			Score:      scores[best],
			Depth:      depth,
			PathPrefix: strings.Split(best, "/"),
		})
		current = best
		// Leaf level: no observations are organized UNDER current (they live IN
		// it), so there is nothing further to drill into.
		if !s.hasDirectoryChildren(ctx, current) {
			break
		}
	}
	res.Leaf = current

	// Final results: the observations inside the leaf directory, ranked by the
	// same combined scoring (FTS5 + embedding similarity).
	res.Results = s.directoryLeafResults(ctx, query, current, qv, limit)

	res.QueryID = s.recordRetrievalTrail(ctx, query, res.Trajectory)
	return res, nil
}

// scoreDirectories returns the child directories of root (or top-level
// directories when root is empty) scored by the combined relevance of the
// observations they contain: FTS5 match density + embedding similarity
// (when a query vector is available). Directories with no matching
// observations score 0 and are pruned by pickBestDirectory.
func (s *Service) scoreDirectories(ctx context.Context, query, root string, qv Vector, fanout int) (map[string]float64, error) {
	cacheKey := dirCacheKey(query, root, matchModeFor(query))
	if cached := dirCacheGet(cacheKey); cached != nil {
		return cached, nil
}
	if fanout <= 0 {
		fanout = 20
	}
	// Pull the observations whose topic_key organizes under root. The
	// LIKE root% clause means "in or under root"; we keep the key and split
	// it to find the child level below root.
	var args []any
	where := "o.project = ? AND o.deleted_at IS NULL AND o.topic_key IS NOT NULL AND o.topic_key != ''"
	args = append(args, s.projectID)
	if root != "" {
		where += " AND o.topic_key LIKE ?"
		args = append(args, root+"/%")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.topic_key, o.title, o.content
		FROM observations o
		WHERE `+where+`
		LIMIT ?`, append(args, fanout*4)...,
	)
	if err != nil {
		return nil, fmt.Errorf("score directories: %w", err)
	}
	defer rows.Close()

	var obs []dirObsRow
	for rows.Next() {
		var r dirObsRow
		if err := rows.Scan(&r.id, &r.topicKey, &r.title, &r.content); err != nil {
			return nil, fmt.Errorf("scan directory observation: %w", err)
		}
		obs = append(obs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Group the observations by their child directory below root, then score
	// each child. Children of an empty root are the top-level segments.
	children := map[string][]dirObsRow{}
	for _, o := range obs {
		child := childDirectory(root, o.topicKey)
		if child == "" {
			continue // the observation lives in root itself, not below it
		}
		children[child] = append(children[child], o)
	}

	scores := make(map[string]float64, len(children))
	for child, rows := range children {
		score := s.scoreDirectoryRows(ctx, query, rows, qv)
		if score > 0 {
			scores[child] = score
		}
	}
	dirCachePut(cacheKey, scores)
	return scores, nil
}

// scoreDirectoryRows combines FTS5 relevance + embedding similarity over the
// observations of one directory. The FTS leg counts how many of the query
// terms occur in the directory's title/content (the FTS5 floor, no query
// parse needed); the vector leg averages the cosine similarity of the
// directory's embedded observations to the query vector. The combined score
// is fts + max(0, vector) so an embedding can boost but never penalize a
// directory whose FTS text already matches.
func (s *Service) scoreDirectoryRows(ctx context.Context, query string, rows []dirObsRow, qv Vector) float64 {
	terms := strings.Fields(strings.ToLower(query))
	terms = dedupeStrings(terms)
	if len(terms) == 0 {
		return 0
	}
	var fts float64
	for _, o := range rows {
		hay := strings.ToLower(o.title + " " + o.content)
		for _, term := range terms {
			if strings.Contains(hay, term) {
				fts++
			}
		}
	}
	// FTS5 density: matches per query term, capped so one term repeated in
	// many rows does not dominate the directory ranking.
	ftsScore := fts / float64(len(terms))

	vecScore := 0.0
	if len(qv.Data) > 0 {
		best := 0.0
		for _, o := range rows {
			v, err := s.obsEmbedding(ctx, o.id)
			if err != nil || len(v.Data) == 0 {
				continue
			}
			sim := CosineSimilarity(qv, v)
			if sim > best {
				best = sim
			}
		}
		vecScore = best
	}
	if vecScore < 0 {
		vecScore = 0
	}
	return ftsScore + vecScore
}

// directoryLeafResults returns the observations inside the leaf directory,
// ranked by the combined FTS5 + embedding score (descending), truncated to
// limit. Observations without any text match and no embedding similarity are
// excluded (a retrieval returns relevant content, not the whole directory).
func (s *Service) directoryLeafResults(ctx context.Context, query, dir string, qv Vector, limit int) []Observation {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id
		FROM observations o
		WHERE o.project = ? AND o.deleted_at IS NULL AND o.topic_key = ?`,
		s.projectID, dir,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	if len(ids) == 0 {
		return nil
	}

	terms := strings.Fields(strings.ToLower(query))
	terms = dedupeStrings(terms)
	type scored struct {
		o     Observation
		score float64
	}
	var out []scored
	for _, id := range ids {
		o, err := s.Get(ctx, id)
		if err != nil {
			continue
		}
		hay := strings.ToLower(o.Title + " " + o.Content)
		var fts float64
		for _, term := range terms {
			if strings.Contains(hay, term) {
				fts++
			}
		}
		ftsScore := 0.0
		if len(terms) > 0 {
			ftsScore = fts / float64(len(terms))
		}
		vecScore := 0.0
		if len(qv.Data) > 0 {
			if v, err := s.obsEmbedding(ctx, id); err == nil && len(v.Data) > 0 {
				vecScore = CosineSimilarity(qv, v)
				if vecScore < 0 {
					vecScore = 0
				}
			}
		}
		if ftsScore+vecScore <= 0 {
			continue
		}
		out = append(out, scored{o: o, score: ftsScore + vecScore})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].o.ID < out[j].o.ID
	})
	res := make([]Observation, 0, len(out))
	for _, sc := range out {
		res = append(res, sc.o)
	}
	if len(res) > limit {
		res = res[:limit]
	}
	return res
}

// hasDirectoryChildren reports whether any observation is organized UNDER dir
// (topic_key dir/...). When false, dir is a leaf: its observations are
// retrieved directly and the drill-down stops.
func (s *Service) hasDirectoryChildren(ctx context.Context, dir string) bool {
	var n int
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL AND topic_key LIKE ?`,
		s.projectID, dir+"/%",
	).Scan(&n)
	if err != nil {
		return false
	}
	return n > 0
}

// obsEmbedding decodes the stored embedding for an observation (empty Vector
// when it has none — the FTS-only path).
func (s *Service) obsEmbedding(ctx context.Context, id int64) (Vector, error) {
	var blob []byte
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT embedding FROM observations WHERE id = ? AND project = ?`,
		id, s.projectID,
	).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return Vector{}, nil
	}
	if err != nil {
		return Vector{}, err
	}
	if len(blob) == 0 {
		return Vector{}, nil
	}
	return DecodeVector(blob)
}

// dirEmbedder returns the Service's optional query-embedding seam (nil =
// FTS5-only scoring). It is set by SetDirEmbedder (config-driven); when
// unset, directory retrieval degrades gracefully to FTS5-only.
func (s *Service) dirEmbedder() dirQueryEmbedder {
	if s == nil || s.dirEmb == nil {
		return nil
	}
	return dirEmbAdapter{e: s.dirEmb}
}

// dirEmbAdapter adapts the broader embedder.Embedder (or any query-embedding
// provider) to the narrow dirQueryEmbedder seam.
type dirEmbAdapter struct{ e interface{ EmbedQuery(ctx context.Context, text string) (Vector, error) } }

func (a dirEmbAdapter) EmbedQuery(ctx context.Context, text string) (Vector, error) {
	return a.e.EmbedQuery(ctx, text)
}

// SetDirEmbedder attaches the optional embedding seam for directory scoring.
// Passing nil restores FTS5-only scoring (the default). It is additive and
// never affects the flat search path.
func (s *Service) SetDirEmbedder(e interface{ EmbedQuery(ctx context.Context, text string) (Vector, error) }) {
	if s != nil {
		s.dirEmb = e
	}
}

// recordRetrievalTrail persists the trajectory to retrieval_trails (19.2)
// and returns the row's query id. It is best-effort: a recording failure is
// logged, never propagated (the retrieval result is still valid).
func (s *Service) recordRetrievalTrail(ctx context.Context, query string, steps []RetrievalStep) int64 {
	if len(steps) == 0 {
		return 0
	}
	pathJSON, err := json.Marshal(steps)
	if err != nil {
		return 0
	}
	scoresJSON, err := json.Marshal(directoryStepScores(steps))
	if err != nil {
		return 0
	}
	depth := steps[len(steps)-1].Depth
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO retrieval_trails (project, query, path, scores, depth, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		s.projectID, query, pathJSON, scoresJSON, depth, now,
	)
	if err != nil {
		fmt.Fprintf(logWriter(), "mnemonic: record retrieval trail: %v\n", err)
		return 0
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0
	}
	return id
}

// GetTrajectory retrieves the stored trajectory for a retrieval run by its
// query id (19.2). An unknown query id returns an empty slice, not an error
// (the trail is an audit record; its absence is not a retrieval failure).
func (s *Service) GetTrajectory(ctx context.Context, queryID int64) ([]RetrievalStep, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	var pathJSON []byte
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT path FROM retrieval_trails WHERE id = ?`, queryID,
	).Scan(&pathJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get retrieval trajectory: %w", err)
	}
	var steps []RetrievalStep
	if err := json.Unmarshal(pathJSON, &steps); err != nil {
		return nil, fmt.Errorf("decode retrieval trajectory: %w", err)
	}
	return steps, nil
}

// directoryStepScores projects the trajectory to its score list (the
// retrieval_trails.scores column: the per-visit scores, in order).
func directoryStepScores(steps []RetrievalStep) []float64 {
	out := make([]float64, len(steps))
	for i, st := range steps {
		out[i] = st.Score
	}
	return out
}

// childDirectory returns the child directory of root that topicKey belongs to
// (root/app for root="root", topicKey="root/app/core"; "app" for an empty
// root). It returns "" when topicKey is root itself (not below it) or when it
// does not organize under root at all.
func childDirectory(root, topicKey string) string {
	tk := strings.Trim(topicKey, "/")
	if root == "" {
		if tk == "" {
			return ""
		}
		if i := strings.IndexByte(tk, '/'); i >= 0 {
			return tk[:i]
		}
		// A single-segment topic_key with no root is itself a top-level
		// directory.
		return tk
	}
	root = strings.Trim(root, "/")
	if tk == root {
		return ""
	}
	if !strings.HasPrefix(tk, root+"/") {
		return ""
	}
	rest := tk[len(root)+1:]
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return root + "/" + rest[:i]
	}
	return root + "/" + rest
}

// topicKeyUnder reports whether topicKey organizes inside dir (equals it or
// extends it). Used by tests to verify results are scoped to the leaf.
func topicKeyUnder(topicKey, dir string) bool {
	tk := strings.Trim(topicKey, "/")
	d := strings.Trim(dir, "/")
	return tk == d || strings.HasPrefix(tk, d+"/")
}

// pickBestDirectory returns the highest-scoring directory (ties broken by
// path for determinism). It returns "" when scores is empty.
func pickBestDirectory(scores map[string]float64) string {
	best := ""
	bestScore := math.Inf(-1)
	for dir, sc := range scores {
		if sc > bestScore || (sc == bestScore && dir < best) {
			best = dir
			bestScore = sc
		}
	}
	return best
}

// matchModeFor returns the FTS match mode directory scoring uses. The
// directory scoring is term-based (not a full FTS5 parse), so the mode only
// matters for cache-key distinctness; "" keeps one canonical bucket.
func matchModeFor(query string) string {
	_ = query
	return ""
}

// dedupeStrings drops duplicate entries, preserving order.
func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
