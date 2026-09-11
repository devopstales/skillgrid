package memory

import (
	"context"
	"errors"
	"fmt"
)

// CrossLinkQuery is the opt-in triple-store cross-link path (014 step 07).
// Starting from observationID it follows observations.graph_ref to the
// codeindex symbol and returns every OTHER live observation bound to the same
// symbol, via a single SQL JOIN:
//
//	observations o JOIN observations o2 ON o2.graph_ref = o.graph_ref
//	WHERE o.id = ? AND o2.id != ?
//
// It is a separate method on purpose: the default Search / SearchWithScope /
// BlendedSearch paths do not call it, so the existing search behavior is
// unchanged. An observation with a NULL graph_ref (never linked to a symbol)
// yields no results.
func (s *Service) CrossLinkQuery(ctx context.Context, observationID int64) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o2.id, o2.session_id, o2.type, o2.title, o2.content, o2.project, o2.scope,
			o2.topic_key, o2.source, o2.normalized_hash, o2.revision_count, o2.prompt_id, o2.created_at, o2.updated_at,
			COALESCE(o2.pinned, 0), COALESCE(o2.duplicate_count, 0), o2.last_seen_at, o2.expires_at, o2.tool_name,
			o2.owner, COALESCE(o2.visibility, 'private'), COALESCE(o2.status, 'active'), COALESCE(o2.retrieval_usage, 0)
		FROM observations o
		JOIN observations o2 ON o2.graph_ref = o.graph_ref
		WHERE o.id = ? AND o.graph_ref IS NOT NULL
		  AND o2.id != ?
		  AND o2.project = ?
		  AND o2.deleted_at IS NULL
		ORDER BY o2.id`,
		observationID, observationID, s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("cross-link query: %w", err)
	}
	defer rows.Close()
	return scanObservations(rows)
}

// CrossLinkOrphan is one broken triple-store reference: an observation whose
// graph_ref points at a codeindex symbol that no longer exists.
type CrossLinkOrphan struct {
	// ID of the observation holding the dangling reference.
	ID int64 `json:"id"`
	// GraphRef is the missing symbol id.
	GraphRef int64 `json:"graph_ref"`
	// Title identifies the observation in diagnostics.
	Title string `json:"title"`
}

// CheckCrossLinkIntegrity is the on-demand orphan sweep for the triple-store
// cross-link (014 step 07). It returns every live observation whose
// non-NULL graph_ref does not resolve to a row in the codeindex symbols
// table. Observations with a valid reference (or a NULL graph_ref) pass.
// Read-only; safe to run on a schedule or from a doctor command.
func (s *Service) CheckCrossLinkIntegrity(ctx context.Context) ([]CrossLinkOrphan, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.graph_ref, o.title
		FROM observations o
		WHERE o.project = ?
		  AND o.deleted_at IS NULL
		  AND o.graph_ref IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM symbols sym WHERE sym.id = o.graph_ref)
		ORDER BY o.id`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("cross-link integrity: %w", err)
	}
	defer rows.Close()
	var orphans []CrossLinkOrphan
	for rows.Next() {
		var o CrossLinkOrphan
		if err := rows.Scan(&o.ID, &o.GraphRef, &o.Title); err != nil {
			return nil, fmt.Errorf("scan orphan: %w", err)
		}
		orphans = append(orphans, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orphans, nil
}
