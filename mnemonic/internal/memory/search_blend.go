package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchSignals is the additive per-hit explanation on mem_search (ADR-0018).
// Every field is in [0,1].
type SearchSignals struct {
	Keyword    float64
	Vector     float64
	Recency    float64
	Entity     float64
	Decay      float64
	Importance float64
}

// SearchHit is one owner-scoped search result plus the fusion explanation.
// Observation rows carry Corpus "observations"; fused web_cache rows
// (ADR-0047) carry Corpus "web_cache" with WebCacheID/labels filled and an
// empty Observation (a web row is not an observation — it is retrieved via
// web_cache_get id=<WebCacheID>).
type SearchHit struct {
	Observation Observation
	Score       float64
	MatchedVia  string
	Signals     SearchSignals
	// Corpus is "observations" (the default) or "web_cache" (fused web hit).
	Corpus string
	// WebCacheID keys the retrieve pointer (web_cache_get id=<WebCacheID>);
	// set on web_cache hits only.
	WebCacheID int64
	// WebSource / WebURL / WebFetchedAt / WebExpiresAt are the web_cache
	// row labels (set on web_cache hits only).
	WebSource    string
	WebURL       string
	WebFetchedAt string
	WebExpiresAt string
	// Retrieve is the caller-facing full-body pointer ("web_cache_get id=N"
	// for web hits, "mem_get_observation id=N" for observation hits).
	Retrieve string
}

// CorpusObservations / CorpusWebCache label the fused mem_search result set
// (ADR-0047).
const (
	CorpusObservations = "observations"
	CorpusWebCache     = "web_cache"
)

// SearchOwnerScopedBlend is the mem_search read path. The observation legs
// (FTS + vector) honor the owner visibility filter; the web_cache FTS leg
// (ADR-0047) fuses unexpired, project-scoped web_cache rows into the same
// result set via RRF (k=60) — labeled corpus=web_cache, never part of the
// vector leg. An empty query vector or a disabled embedder keeps the FTS
// order and reports matched_via=keyword.
func (s *Service) SearchOwnerScopedBlend(ctx context.Context, readerOwner, readerAgent, query, matchMode, scope string, limit int, queryVec Vector) ([]SearchHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	fts, err := s.ownerScopedFTS(ctx, readerOwner, readerAgent, query, matchMode, scope, limit)
	if err != nil {
		return nil, err
	}

	// ponytail: the web key prefixes ":" so it can never collide with the
	// "<obs id>:<project>" observation keys below.
	keyOf := func(id int64, project string) string {
		return fmt.Sprintf("%d:%s", id, project)
	}
	ftsRank := map[string]int{}
	byKey := map[string]Observation{}
	for i, o := range fts {
		k := keyOf(o.ID, o.Project)
		ftsRank[k] = i
		byKey[k] = o
	}

	// Fused web_cache leg (ADR-0047): a second FTS MATCH over web_cache_fts,
	// project-scoped, excluding expired rows. The vector leg stays
	// observations-only (no web rows are ever embedded).
	webRank := map[string]int{}
	byWeb := map[string]WebHit{}
	webHits, wErr := s.projectWebCacheFTS(ctx, query, matchMode, limit)
	if wErr != nil {
		return nil, wErr
	}
	for i, w := range webHits {
		k := fmt.Sprintf(":%d", w.ID)
		webRank[k] = i
		byWeb[k] = w
	}

	vecRank := map[string]int{}
	cosine := map[string]float64{}
	useVec := len(queryVec.Data) > 0 && EmbeddingEnabled()
	if useVec {
		wider := limit * 3
		if wider < limit {
			wider = limit
		}
		reqScope := strings.TrimSpace(scope)
		nowUTC := time.Now().UTC()
		vecHits, vErr := s.SearchByVector(ctx, queryVec, wider)
		if vErr != nil || len(vecHits) == 0 {
			useVec = false
		} else {
			placed := 0
			for _, vh := range vecHits {
				ok, rErr := s.canRead(ctx, vh.ID, "", readerOwner, readerAgent)
				if rErr != nil || !ok {
					continue
				}
				k := keyOf(vh.ID, vh.Project)
				o, seen := byKey[k]
				if !seen {
					// Vector-only hit: Get does not apply the scope, invalid_at
					// or expires_at filters the FTS leg already enforced.
					var gErr error
					o, gErr = s.Get(ctx, vh.ID)
					if gErr != nil || !vectorHitLive(o, reqScope, nowUTC) {
						continue
					}
					byKey[k] = o
				}
				vecRank[k] = placed
				sim := vh.Sim
				if sim < 0 {
					sim = 0
				}
				if sim > 1 {
					sim = 1
				}
				cosine[k] = sim
				placed++
			}
			if len(vecRank) == 0 {
				useVec = false
			}
		}
	}

	order := make([]string, 0, len(byKey)+len(byWeb))
	if !useVec {
		for _, o := range fts {
			order = append(order, keyOf(o.ID, o.Project))
		}
		// FTS order is the observation floor; fused web rows follow the
		// observation hits in their own BM25 order (ADR-0047: with no vector
		// leg there is no RRF to interleave with).
		for _, w := range webHits {
			order = append(order, fmt.Sprintf(":%d", w.ID))
		}
	} else {
		ids := make([]string, 0, len(byKey)+len(byWeb))
		for k := range byKey {
			ids = append(ids, k)
		}
		for k := range byWeb {
			ids = append(ids, k)
		}
		order = ReciprocalRankFusion3(ids, ftsRank, vecRank, webRank, 60)
	}

	cfg := s.decayCfg
	if cfg.HalfLifeDays <= 0 {
		cfg.HalfLifeDays = 30
	}
	now := time.Now().UTC()
	hits := make([]SearchHit, 0, len(order))
	for _, k := range order {
		// A fused web_cache hit (ADR-0047): the Observation fields are empty,
		// so the observation-shaped signals all read zero and the label
		// fields carry the row's own provenance.
		if w, isWeb := byWeb[k]; isWeb {
			wk := 0.0
			if r, hasWebRank := webRank[k]; hasWebRank {
				wk = 1 / float64(1+r)
			}
			hits = append(hits, SearchHit{
				Corpus:       CorpusWebCache,
				Score:        wk,
				MatchedVia:   "keyword",
				Signals:      SearchSignals{Keyword: wk},
				WebCacheID:   w.ID,
				WebSource:    w.Source,
				WebURL:       w.URL,
				WebFetchedAt: w.FetchedAt,
				WebExpiresAt: w.ExpiresAt,
				Retrieve:     fmt.Sprintf("web_cache_get id=%d", w.ID),
			})
			continue
		}
		o, ok := byKey[k]
		if !ok {
			continue
		}
		_, hasFTS := ftsRank[k]
		_, hasVec := vecRank[k]
		via := "keyword"
		switch {
		case hasFTS && hasVec:
			via = "hybrid"
		case hasVec:
			via = "vector"
		}
		kw := 0.0
		if hasFTS {
			kw = 1 / float64(1+ftsRank[k])
		}
		sig := SearchSignals{
			Keyword:    kw,
			Vector:     cosine[k],
			Recency:    SignalRecency(o, now, cfg),
			Entity:     0,
			Decay:      SignalDecay(o, now, cfg),
			Importance: SignalImportance(o),
		}
		score := kw
		if hasVec {
			rrf := 0.0
			if hasFTS {
				rrf += 1 / float64(60+ftsRank[k]+1)
			}
			rrf += 1 / float64(60+vecRank[k]+1)
			score = rrf / (2.0 / 61.0)
			if score > 1 {
				score = 1
			}
		}
		hits = append(hits, SearchHit{
			Observation: o,
			Score:       score,
			MatchedVia:  via,
			Signals:     sig,
			Corpus:      CorpusObservations,
			Retrieve:    fmt.Sprintf("mem_get_observation id=%d", o.ID),
		})
	}
	if s.decayCfg.Enabled {
		hits = sortHitsByDecay(hits, now, cfg)
		// Fused web hits (ADR-0047) are not observations: sortHitsByDecay
		// re-scores them on empty Observation fields, which would sink them
		// to the bottom. Re-append them after the observation hits in their
		// own fused order.
		obs, web := hits[:0], hits[:0]
		for _, h := range hits {
			if h.Corpus == CorpusWebCache {
				web = append(web, h)
			} else {
				obs = append(obs, h)
			}
		}
		hits = append(obs, web...)
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// WebHit is one fused web_cache row surfaced by mem_search (ADR-0047). The
// labels are the row's own provenance — a web hit is labeled, not scored like
// an observation.
type WebHit struct {
	ID        int64
	Source    string
	URL       string
	FetchedAt string
	ExpiresAt string
}

// projectWebCacheFTS is the fused web_cache leg of mem_search (ADR-0047): an
// FTS MATCH over web_cache_fts restricted to this project's rows and to rows
// that are not yet expired (expires_at NULL/blank or in the future). It
// shares buildFTSQuery with the observation FTS leg, and an empty FTS query
// (no searchable terms) yields no web hits without an error.
func (s *Service) projectWebCacheFTS(ctx context.Context, query, matchMode string, limit int) ([]WebHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	ftsQuery := buildFTSQuery(query, matchMode)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT w.id, w.source, w.url, w.fetched_at, w.expires_at
		FROM web_cache w
		INNER JOIN web_cache_fts ON web_cache_fts.rowid = w.id
		WHERE web_cache_fts MATCH ? AND w.project = ?
		  AND (w.expires_at IS NULL OR w.expires_at = ''
		       OR strftime('%s', w.expires_at) > strftime('%s', 'now'))
		ORDER BY bm25(web_cache_fts)
		LIMIT ?`,
		ftsQuery, s.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("web cache fused search: %w", err)
	}
	defer rows.Close()
	var out []WebHit
	for rows.Next() {
		var w WebHit
		if err := rows.Scan(&w.ID, &w.Source, &w.URL, &w.FetchedAt, &w.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// vectorHitLive reports whether a vector-only hit may be returned: it must
// match a non-empty request scope and be neither invalidated nor expired.
func vectorHitLive(o Observation, reqScope string, now time.Time) bool {
	if reqScope != "" && o.Scope != reqScope {
		return false
	}
	return !timestampReached(o.InvalidAt, now) && !timestampReached(o.ExpiresAt, now)
}

// timestampReached is true when value parses as a timestamp at or before now.
// Empty or unparseable values are treated as unset.
func timestampReached(value string, now time.Time) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return !t.After(now)
		}
	}
	return false
}

func sortHitsByDecay(hits []SearchHit, now time.Time, cfg DecayConfig) []SearchHit {
	if len(hits) < 2 {
		return hits
	}
	pinned := make([]SearchHit, 0, len(hits))
	rest := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Observation.Pinned {
			pinned = append(pinned, h)
		} else {
			rest = append(rest, h)
		}
	}
	type row struct {
		h SearchHit
		s float64
		i int
	}
	rows := make([]row, len(rest))
	for i, h := range rest {
		rows[i] = row{h: h, s: Reinforcement(h.Observation, now, cfg) * h.Score, i: i}
	}
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && (rows[j].s > rows[j-1].s || (rows[j].s == rows[j-1].s && rows[j].i < rows[j-1].i)) {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
	out := make([]SearchHit, 0, len(hits))
	out = append(out, pinned...)
	for _, r := range rows {
		out = append(out, r.h)
	}
	return out
}

// ownerScopedFTS is SearchOwnerScoped without the in-memory re-rank. It bumps
// retrieval usage after the scan, on the pre-bump structs the caller ranks.
func (s *Service) ownerScopedFTS(ctx context.Context, readerOwner, readerAgent, query, matchMode, scope string, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	ftsQuery := buildFTSQuery(query, matchMode)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	scope = strings.TrimSpace(scope)
	scopeClause := ""
	args := []any{ftsQuery, s.projectID}
	if scope != "" {
		scopeClause = " AND o.scope = ?"
		args = append(args, scope)
	}
	// A blank reader is the default single-user case: it sees the whole
	// project corpus (pre-017 behavior). Only a NAMED reader is gated by
	// owner/visibility — gating a blank reader as a "different owner" wiped
	// every private row (o.owner='' OR visibility!='private' matched nothing
	// for a corpus saved with real owners), so mem_search returned zero for
	// any term. Named readers keep per-owner enforcement below.
	var clause string
	var clauseArgs []any
	if strings.TrimSpace(readerOwner) != "" {
		clause, clauseArgs = s.visibilityFilter(readerOwner, readerOwner, readerAgent)
	} else {
		clause = "1=1"
	}
	args = append(args, clauseArgs...)
	args = append(args, limit)
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT `+obsSplitSelectCols+`
		FROM `+obsSplitFrom+`
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?`+scopeClause+`
		  AND (tm.invalid_at IS NULL OR tm.invalid_at = '' OR strftime('%s', tm.invalid_at) > strftime('%s', 'now'))
		  AND `+clause+`
		ORDER BY COALESCE(st.pinned, 0) DESC, bm25(observations_fts)
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("owner scoped search: %w", err)
	}
	defer rows.Close()
	out, err := scanObservations(rows)
	if err != nil {
		return nil, err
	}
	for _, o := range out {
		s.BumpRetrievalUsage(ctx, o.ID)
	}
	return out, nil
}
