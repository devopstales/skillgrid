package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Relation vocabulary for semantic links between observations. mem_judge and
// mem_compare both draw on this set. The special verdict "not_conflict" is
// NOT a stored relation — it means "remove any conflicts_with link".
var validRelations = map[string]struct{}{
	"related":        {},
	"compatible":     {},
	"scoped":         {},
	"conflicts_with": {},
	"supersedes":     {},
}

// validRelationOrder is the stable order for error messages and listings.
var validRelationOrder = []string{"related", "compatible", "scoped", "conflicts_with", "supersedes"}

// ValidRelations returns the list of storable relation names in a stable order.
func ValidRelations() []string {
	out := make([]string, 0, len(validRelations))
	for _, r := range validRelationOrder {
		if _, ok := validRelations[r]; ok {
			out = append(out, r)
		}
	}
	// Include any relations added to the map later that are not in the order.
	for r := range validRelations {
		found := false
		for _, o := range out {
			if r == o {
				found = true
				break
			}
		}
		if !found {
			out = append(out, r)
		}
	}
	return out
}

// IsValidRelation reports whether r is a storable relation.
func IsValidRelation(r string) bool {
	_, ok := validRelations[strings.TrimSpace(r)]
	return ok
}

// Relation is a stored semantic link between two observations.
type Relation struct {
	ID         int64    `json:"id"`
	SrcObsID   int64    `json:"src_obs_id"`
	DstObsID   int64    `json:"dst_obs_id"`
	Relation   string   `json:"relation"`
	Confidence *float64 `json:"confidence,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

// RecordRelation stores (or updates) a semantic link between two
// observations in this project. Both observation IDs must belong to this
// project store. The link is directional: src → dst. "not_conflict" is not a
// stored relation — callers should use RemoveRelation("conflicts_with").
func (s *Service) RecordRelation(ctx context.Context, srcID, dstID int64, relation, reason string, confidence *float64) (Relation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return Relation{}, errors.New("memory service not initialized")
	}
	relation = strings.TrimSpace(relation)
	if !IsValidRelation(relation) {
		return Relation{}, fmt.Errorf("invalid relation %q (valid: %s)", relation, strings.Join(ValidRelations(), ", "))
	}
	if srcID == dstID {
		return Relation{}, errors.New("source and destination observation IDs must differ")
	}
	for _, id := range []int64{srcID, dstID} {
		if err := s.obsExists(ctx, id); err != nil {
			return Relation{}, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// modernc.org/sqlite does not support ON CONFLICT against a partial
	// unique index, so upsert manually: look up a live row, UPDATE in place
	// if present, insert if not.
	var existingID int64
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT id FROM memory_relations
		WHERE src_obs_id = ? AND dst_obs_id = ? AND relation = ? AND deleted_at IS NULL
		LIMIT 1`,
		srcID, dstID, relation,
	).Scan(&existingID)
	if err == nil {
		_, err = s.store.DB.ExecContext(ctx, `
			UPDATE memory_relations
			SET confidence = ?, reason = ?, created_at = ?
			WHERE id = ?`,
			confidenceNullable(confidence), reason, now, existingID,
		)
		if err != nil {
			return Relation{}, fmt.Errorf("upsert relation: %w", err)
		}
		return Relation{
			ID:         existingID,
			SrcObsID:   srcID,
			DstObsID:   dstID,
			Relation:   relation,
			Confidence: confidence,
			Reason:     reason,
			CreatedAt:  now,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Relation{}, fmt.Errorf("relation lookup: %w", err)
	}
	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO memory_relations (src_obs_id, dst_obs_id, relation, confidence, reason, project, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		srcID, dstID, relation, confidenceNullable(confidence), reason, s.projectID, now,
	)
	if err != nil {
		return Relation{}, fmt.Errorf("insert relation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Relation{}, fmt.Errorf("relation last insert id: %w", err)
	}
	return Relation{
		ID:         id,
		SrcObsID:   srcID,
		DstObsID:   dstID,
		Relation:   relation,
		Confidence: confidence,
		Reason:     reason,
		CreatedAt:  now,
	}, nil
}

// RemoveRelation clears a live link. Returns true if a row was removed.
func (s *Service) RemoveRelation(ctx context.Context, srcID, dstID int64, relation string) (bool, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return false, errors.New("memory service not initialized")
	}
	relation = strings.TrimSpace(relation)
	if relation == "" {
		return false, errors.New("relation is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// Soft-delete the directed link both directions, so ordering of (a,b) does
	// not matter to the caller.
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE memory_relations SET deleted_at = ?
		WHERE relation = ? AND deleted_at IS NULL AND (
			(src_obs_id = ? AND dst_obs_id = ?) OR
			(src_obs_id = ? AND dst_obs_id = ?)
		)`,
		now, relation, srcID, dstID, dstID, srcID,
	)
	if err != nil {
		return false, fmt.Errorf("remove relation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// RelationsOf returns every live relation touching observation id (as
// either src or dst), for surfacing in mem_compare / explain-style reads.
func (s *Service) RelationsOf(ctx context.Context, id int64) ([]Relation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, src_obs_id, dst_obs_id, relation, confidence, reason, created_at
		FROM memory_relations
		WHERE relation IS NOT NULL AND deleted_at IS NULL
		  AND (src_obs_id = ? OR dst_obs_id = ?)
		ORDER BY created_at DESC`,
		id, id,
	)
	if err != nil {
		return nil, fmt.Errorf("list relations: %w", err)
	}
	defer rows.Close()
	var out []Relation
	for rows.Next() {
		var r Relation
		var conf sql.NullFloat64
		var reason sql.NullString
		if err := rows.Scan(&r.ID, &r.SrcObsID, &r.DstObsID, &r.Relation, &conf, &reason, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan relation: %w", err)
		}
		if conf.Valid {
			v := conf.Float64
			r.Confidence = &v
		}
		if reason.Valid {
			r.Reason = reason.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) obsExists(ctx context.Context, id int64) error {
	var n int
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM observations
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		id, s.projectID,
	).Scan(&n)
	if err != nil {
		return fmt.Errorf("verify observation %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("observation %d not found in this project", id)
	}
	return nil
}

// RelationsBetween returns the live semantic links between two specific
// observations, in either direction. Used by mem_compare's list mode.
func (s *Service) RelationsBetween(ctx context.Context, srcID, dstID int64) ([]Relation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, src_obs_id, dst_obs_id, relation, confidence, reason, created_at
		FROM memory_relations
		WHERE deleted_at IS NULL AND
		  (
			(src_obs_id = ? AND dst_obs_id = ?) OR
			(src_obs_id = ? AND dst_obs_id = ?)
		  )
		ORDER BY created_at DESC`,
		srcID, dstID, dstID, srcID,
	)
	if err != nil {
		return nil, fmt.Errorf("relations between: %w", err)
	}
	defer rows.Close()
	var out []Relation
	for rows.Next() {
		var r Relation
		var conf sql.NullFloat64
		var reason sql.NullString
		if err := rows.Scan(&r.ID, &r.SrcObsID, &r.DstObsID, &r.Relation, &conf, &reason, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan between: %w", err)
		}
		if conf.Valid {
			v := conf.Float64
			r.Confidence = &v
		}
		if reason.Valid {
			r.Reason = reason.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func confidenceNullable(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
