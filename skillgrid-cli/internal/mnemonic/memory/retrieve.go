package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// LayerHit is one result of layered retrieval (change 013, step 03). It
// carries the provenance layer (L2/L3 from the step-02 bootstrap, L1/L0 from
// the RRF fallback) and the full-content fetch id so the agent can pull the
// untruncated content on demand (mem_get_observation is the only full-content
// path — in-list hits are truncated).
type LayerHit struct {
	Layer       string `json:"layer"` // L3 | L2 | L1 | L0
	TargetKind  string `json:"target_kind"`
	TargetID    int64  `json:"target_id"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content"`
	SourceTopic string `json:"source_topic,omitempty"`
	// GetObservationID is the id the agent uses to fetch the full, untruncated
	// content via mem_get_observation. For an L1 atom it is the observation id;
	// for an L2/L3 persona it is the persona id (surfaced as the stable handle).
	// 0 only when the target is not directly fetchable by id.
	GetObservationID int64 `json:"get_observation_id"`
}

// RetrieveOpts tunes layered retrieval. Mode is "bootstrap" (L2/L3-first) or
// "fact" (RRF fallback for a specific fact). Limit bounds how many hits are
// returned before the read budget is applied.
type RetrieveOpts struct {
	Mode  string // "bootstrap" | "fact"
	Query string
	Limit int
}

// Retrieve is the layered retrieval read path (change 013, step 03): it
// bootstraps from L2/L3 (the step-02 observation_layers/personas store) and,
// for a specific fact, falls back to L1/L0 via the EXISTING 005 RRF read path
// (BlendedSearch — FTS + vector + ReciprocalRankFusion). It does NOT
// re-implement RRF: it routes through BlendedSearch for the L1/L0 leg.
//
// The result is the in-list projection: L2/L3 first (cheap, stable), L1/L0
// after. In-list content is NOT truncated here (that is the read budget's job,
// applied at the MCP boundary) — Retrieve returns the full ranked hit list and
// the caller budgets it. mem_get_observation remains the only full-content
// path: every hit carries its GetObservationID.
func (s *Service) Retrieve(ctx context.Context, opts RetrieveOpts) ([]LayerHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	mode := strings.ToLower(strings.TrimSpace(opts.Mode))
	if mode == "" {
		mode = "bootstrap"
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	var out []LayerHit
	switch mode {
	case "bootstrap":
		l2l3, err := s.layerBootstrap(ctx, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, l2l3...)
		// Bootstrap returns L2/L3 first; a specific fact is a separate "fact"
		// query that falls back to RRF.
	case "fact":
		rrf, err := s.rrfFallback(ctx, opts.Query, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, rrf...)
	default:
		return nil, fmt.Errorf("unknown retrieve mode %q (valid: bootstrap, fact)", mode)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// SearchOwnerScoped is the owner-gated layered read path behind the
// production CLI `mem search` (change 013, step 03): it runs the same layered
// retrieval as Retrieve, but for a specific fact (mode "fact") it runs the
// L1/L0 RRF fallback through the EXISTING per-owner visibility-filtered read
// (s.SearchOwnerScoped — the step-01 seam) so a private observation of another
// owner is absent from the in-list result. Bootstrap (L2/L3) is unaffected
// (persona distillation is project-scoped, not per-owner gated). This is what
// makes the layered path observable at a real entry point WITHOUT regressing
// the step-01 per-owner visibility gate.
func (s *Service) SearchOwnerScopedRetrieve(ctx context.Context, readerOwner, mode, query string, limit int) ([]LayerHit, error) {
	return s.SearchOwnerScopedRetrieveFTS(ctx, readerOwner, mode, query, "", limit)
}

// SearchOwnerScopedRetrieveFTS is SearchOwnerScopedRetrieve with an FTS match
// mode for the L1/L0 fact leg (change 014, step 02): "trigram" expands the
// query into 3-character substrings and "prefix" appends a wildcard to each
// term (both opt-in; the MCP path keeps the FTS-mode-less variant so its
// contract is unchanged). An empty matchMode is the existing behavior.
func (s *Service) SearchOwnerScopedRetrieveFTS(ctx context.Context, readerOwner, mode, query, matchMode string, limit int) ([]LayerHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "bootstrap"
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	switch m {
	case "bootstrap":
		return s.layerBootstrap(ctx, limit)
	case "fact":
		return s.rrfFallbackOwnerScopedFTS(ctx, readerOwner, query, matchMode, limit)
	default:
		return nil, fmt.Errorf("unknown retrieve mode %q (valid: bootstrap, fact)", m)
	}
}

// layerBootstrap returns the L2/L3 bootstrap hits: the step-02 personas records
// (L3 persona-delta before L2 scenario — the most stable, cheapest first), each
// provenance-linked to its resolvable L0 source. A project with no distilled
// layers returns an empty list (not an error) — bootstrap degrades gracefully.
func (s *Service) layerBootstrap(ctx context.Context, limit int) ([]LayerHit, error) {
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT p.id, p.kind, p.title, p.content, l.source_topic
		FROM observation_layers l
		INNER JOIN personas p ON p.id = l.target_id AND p.project = l.project
		WHERE l.project = ? AND l.target_kind = 'persona' AND l.layer IN ('L2','L3')
		ORDER BY CASE l.layer WHEN 'L3' THEN 0 ELSE 1 END, p.id ASC
		LIMIT ?`,
		s.projectID, limit*2,
	)
	if err != nil {
		return nil, fmt.Errorf("bootstrap layers: %w", err)
	}
	defer rows.Close()
	var out []LayerHit
	for rows.Next() {
		var (
			personaID int64
			kind      string
			title, content, topic sql.NullString
		)
		if err := rows.Scan(&personaID, &kind, &title, &content, &topic); err != nil {
			return nil, fmt.Errorf("scan bootstrap: %w", err)
		}
		h := LayerHit{
			Layer:          layerForKind(kind),
			TargetKind:     "persona",
			TargetID:       personaID,
			GetObservationID: personaID,
		}
		if title.Valid {
			h.Title = title.String
		}
		if content.Valid {
			h.Content = content.String
		}
		if topic.Valid {
			h.SourceTopic = topic.String
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// layerForKind maps a personas kind to its layer label.
func layerForKind(kind string) string {
	switch kind {
	case "persona_delta":
		return "L3"
	default: // scenario
		return "L2"
	}
}

// rrfFallback is the L1/L0 leg: it routes through the EXISTING 005 RRF read
// path (BlendedSearch: FTS + vector + ReciprocalRankFusion) so a specific fact
// falls back to ranked fusion. L1 atoms are observations (reachable via FTS);
// the L0 raw record is the session summary (the RRF floor). We do NOT
// re-implement RRF here — we call BlendedSearch.
func (s *Service) rrfFallback(ctx context.Context, query string, limit int) ([]LayerHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		// No fact to look up: an empty query returns nothing (not an error) —
		// a specific-fact query with no fact is a no-op, not a hang.
		return nil, nil
	}
	hits, err := s.BlendedSearch(ctx, query, "any", "", Vector{}, limit)
	if err != nil {
		return nil, err
	}
	return layerHitsFromObservations(hits, limit), nil
}

// rrfFallbackOwnerScoped is the owner-gated L1/L0 leg: same as rrfFallback
// (routes through the EXISTING 005 RRF read path, BlendedSearch — FTS + vector
// + ReciprocalRankFusion) but the returned hits are then filtered by the
// step-01 per-owner visibility gate (canRead), so a private observation of
// another owner is absent from the in-list result. This is the seam the
// production CLI `mem search` uses so the per-owner visibility gate holds on
// the layered read path WITHOUT regressing the existing RRF ranking.
func (s *Service) rrfFallbackOwnerScoped(ctx context.Context, readerOwner, query string, limit int) ([]LayerHit, error) {
	return s.rrfFallbackOwnerScopedFTS(ctx, readerOwner, query, "", limit)
}

// rrfFallbackOwnerScopedFTS is rrfFallbackOwnerScoped with an FTS match mode
// for the FTS leg (change 014, step 02). The mode is validated up front so an
// unknown mode fails before any read; the vector leg is unaffected by the
// FTS mode (blending is orthogonal to FTS term expansion).
func (s *Service) rrfFallbackOwnerScopedFTS(ctx context.Context, readerOwner, query, matchMode string, limit int) ([]LayerHit, error) {
	if err := ValidateMatchMode(matchMode); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	hits, err := s.BlendedSearch(ctx, query, matchMode, "", Vector{}, limit*3)
	if err != nil {
		return nil, err
	}
	var visible []Observation
	for _, o := range hits {
		if readerOwner == "" {
			visible = append(visible, o)
			continue
		}
		ok, cerr := s.canRead(ctx, o.ID, s.projectID, readerOwner, readerOwner)
		if cerr != nil {
			// A row already gone (deleted/expired since the RRF read) is absent,
			// not a failure; a genuine visibility lookup error is surfaced.
			if errors.Is(cerr, ErrNotFoundForReader) {
				continue
			}
			return nil, cerr
		}
		if ok {
			visible = append(visible, o)
		}
	}
	// Self-improvement feedback loop (014 step 08 M1): the fact-mode leg must
	// re-rank through the same improve() hook SearchOwnerScoped uses, so the
	// opt-in is consistent across every search surface. It applies to the full
	// pre-truncate slice (the leg fetched limit*3), and is a no-op when improve
	// is disabled (the default) or unwired (nil improveCfg → returns input
	// unchanged), so the byte-identity property holds.
	return layerHitsFromObservations(s.improve(ctx, visible), limit), nil
}

// layerHitsFromObservations maps in-list observations (from the RRF leg) into
// LayerHits, each carrying its full-content fetch id.
func layerHitsFromObservations(hits []Observation, limit int) []LayerHit {
	var out []LayerHit
	for _, o := range hits {
		out = append(out, LayerHit{
			Layer:            "L1",
			TargetKind:       "observation",
			TargetID:         o.ID,
			Title:            o.Title,
			Content:          o.Content,
			GetObservationID: o.ID,
		})
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ErrRetrievalEmpty is returned (not raised as a failure) when a layered
// retrieval returns no hits — e.g. a bootstrap with no distilled layers or a
// fact query that matches nothing. It is a condition to be surfaced, not an
// error; callers may ignore it and treat the result as an empty list.
var ErrRetrievalEmpty = errors.New("no retrieval hits")
