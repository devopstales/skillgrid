package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ProjectDrift describes a situation where the caller supplied a project
// name that has been recorded as an *alias* of a canonical project (via a
// prior mem_merge_projects). Saving to the alias name is allowed but worth
// surfacing so the caller learns to write to the canonical name directly.
type ProjectDrift struct {
	// ProjectName is the alias the caller used (trimmed).
	ProjectName string `json:"project_name"`
	// CanonicalName is the merged-into name.
	CanonicalName string `json:"canonical_name"`
	// MergedAt is the timestamp the alias was recorded (for diagnostics).
	MergedAt string `json:"merged_at,omitempty"`
}

// CheckProjectDrift returns a drift report when projectName is a known
// alias of another project (canonical) in this store's project_aliases
// table, and nil otherwise. It does not modify any state. Callers (e.g. the
// MCP mem_save handler) surface the drift as an extra field on the JSON
// response without rejecting the save.
func (s *Service) CheckProjectDrift(ctx context.Context, projectName string) (*ProjectDrift, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	name := strings.TrimSpace(projectName)
	if name == "" {
		return nil, nil
	}
	var canonical string
	var mergedAt sql.NullString
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT canonical, merged_at
		FROM project_aliases
		WHERE alias = ?
		LIMIT 1`,
		name,
	).Scan(&canonical, &mergedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check project drift: %w", err)
	}
	d := &ProjectDrift{
		ProjectName:   name,
		CanonicalName: canonical,
	}
	if mergedAt.Valid {
		d.MergedAt = mergedAt.String
	}
	return d, nil
}
