package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Typed @relation edges between observations (014, step 14).
//
// This is a NEW, additive table (observation_relations, migration 026) in the
// per-project memory store. It is distinct from both memory_relations (006,
// the mem_judge verdict vocabulary) and the codeindex edges table (symbol-level
// graph edges): observation_relations is a typed, confidence-weighted relation
// between OBSERVATION ids, not symbol ids. The step-07 triple-store linkage is
// consumed read-only — the edge endpoints are validated against the live
// observations table before insert.

// canonicalRelationTypes is the fixed @relation vocabulary for observation
// edges. Unknown types are rejected at AddRelation time.
var canonicalRelationTypes = map[string]struct{}{
	"mentions":    {},
	"depends_on":  {},
	"contradicts": {},
	"supports":    {},
	"references":  {},
}

// ObservationRelation is a stored typed edge between two observations.
type ObservationRelation struct {
	SourceID     int64   `json:"source_id"`
	TargetID     int64   `json:"target_id"`
	RelationType string  `json:"relation_type"`
	Confidence   float64 `json:"confidence"`
}

// AddRelation stores (or upserts) a typed, confidence-weighted edge from
// sourceID to targetID. Both observation IDs must be live in this project,
// and relationType must be one of the 5 canonical types. Confidence must lie
// in [0.0, 1.0]. Re-asserting the same (source, target, type) triple updates
// its confidence in place (the composite primary key makes the triple unique).
func (s *Service) AddRelation(ctx context.Context, sourceID, targetID int64, relationType string, confidence float64) (ObservationRelation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return ObservationRelation{}, errors.New("memory service not initialized")
	}
	rel := strings.TrimSpace(relationType)
	if _, ok := canonicalRelationTypes[rel]; !ok {
		return ObservationRelation{}, fmt.Errorf("invalid relation type %q (valid: mentions, depends_on, contradicts, supports, references)", relationType)
	}
	if confidence < 0.0 || confidence > 1.0 {
		return ObservationRelation{}, fmt.Errorf("confidence %v out of range (must be 0.0..1.0)", confidence)
	}
	for _, id := range []int64{sourceID, targetID} {
		if err := s.obsExists(ctx, id); err != nil {
			return ObservationRelation{}, err
		}
	}
	// The composite PK makes (source, target, type) unique; a re-assertion
	// upserts confidence + timestamp instead of erroring.
	if _, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO observation_relations (source_id, target_id, relation_type, confidence)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (source_id, target_id, relation_type)
		DO UPDATE SET confidence = excluded.confidence,
		              created_at = excluded.created_at`,
		sourceID, targetID, rel, confidence); err != nil {
		return ObservationRelation{}, fmt.Errorf("upsert observation relation: %w", err)
	}
	return ObservationRelation{
		SourceID:     sourceID,
		TargetID:     targetID,
		RelationType: rel,
		Confidence:   confidence,
	}, nil
}

// GetRelations returns every typed edge touching obsID as either source or
// target, filtered to confidence >= minConfidence (0.0 returns all). The
// result is bidirectional: outgoing edges (obsID is source) and incoming edges
// (obsID is target) are both included, so the caller can compute direction.
func (s *Service) GetRelations(ctx context.Context, obsID int64, minConfidence float64) ([]ObservationRelation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if minConfidence < 0.0 {
		minConfidence = 0.0
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT source_id, target_id, relation_type, confidence
		FROM observation_relations
		WHERE (source_id = ? OR target_id = ?) AND confidence >= ?
		ORDER BY confidence DESC, source_id, target_id, relation_type`,
		obsID, obsID, minConfidence,
	)
	if err != nil {
		return nil, fmt.Errorf("list observation relations: %w", err)
	}
	defer rows.Close()
	var out []ObservationRelation
	for rows.Next() {
		var r ObservationRelation
		if err := rows.Scan(&r.SourceID, &r.TargetID, &r.RelationType, &r.Confidence); err != nil {
			return nil, fmt.Errorf("scan observation relation: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteRelation deletes a single typed edge by its full (source, target,
// type) triple. Returns true if a row was removed. (Named DeleteRelation to
// avoid clashing with the memory_relations RemoveRelation, which has a
// different signature and operates on the 006 verdict table.)
func (s *Service) DeleteRelation(ctx context.Context, sourceID, targetID int64, relationType string) (bool, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return false, errors.New("memory service not initialized")
	}
	rel := strings.TrimSpace(relationType)
	res, err := s.store.DB.ExecContext(ctx, `
		DELETE FROM observation_relations
		WHERE source_id = ? AND target_id = ? AND relation_type = ?`,
		sourceID, targetID, rel,
	)
	if err != nil {
		return false, fmt.Errorf("remove observation relation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ObsTitle fetches the title of a live observation in this project, for CLI
// display. Returns ("", false) if the observation does not exist here.
func (s *Service) ObsTitle(ctx context.Context, id int64) (string, bool) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "", false
	}
	var title string
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT title FROM observations
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		id, s.projectID,
	).Scan(&title)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return title, true
}
