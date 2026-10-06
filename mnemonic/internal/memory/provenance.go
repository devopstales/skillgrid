// Provenance chain tracking (change 014, step 15): every observation can
// carry an immutable JSON chain recording HOW it was curated — the session it
// came from, the command that produced it, the source files it was extracted
// from, and the LLM reasoning behind the extraction.
package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// Provenance is the curation chain of an observation. It is stored as a JSON
// object in the observations.provenance column (migration 027) and is
// IMMUTABLE once set: only the initial save may establish it; the update path
// preserves the stored value regardless of what a later save supplies.
type Provenance struct {
	SessionID     string   `json:"session_id"`
	CurateCommand string   `json:"curate_command"`
	SourceFiles   []string `json:"source_files"`
	LLMReasoning  string   `json:"llm_reasoning"`
}

// MarshalJSON serializes the provenance chain to the JSON stored in the
// provenance column.
func (p *Provenance) MarshalJSON() ([]byte, error) {
	type alias struct {
		SessionID     string   `json:"session_id"`
		CurateCommand string   `json:"curate_command"`
		SourceFiles   []string `json:"source_files"`
		LLMReasoning  string   `json:"llm_reasoning"`
	}
	a := alias{
		SessionID:     p.SessionID,
		CurateCommand: p.CurateCommand,
		SourceFiles:   p.SourceFiles,
		LLMReasoning:  p.LLMReasoning,
	}
	if a.SourceFiles == nil {
		a.SourceFiles = []string{}
	}
	return json.Marshal(&a)
}

// parseProvenanceColumn decodes the provenance column value: NULL/empty is a
// no-provenance (valid false); unparseable JSON is a hard error so a corrupt
// column is never silently returned as an empty chain.
func parseProvenanceColumn(v sql.NullString) (*Provenance, bool, error) {
	if !v.Valid || v.String == "" {
		return nil, false, nil
	}
	var p Provenance
	if err := json.Unmarshal([]byte(v.String), &p); err != nil {
		return nil, false, fmt.Errorf("parse provenance: %w", err)
	}
	return &p, true, nil
}

// Provenance returns the observation's curation chain: (nil, nil) when no
// provenance was recorded at save time.
func (s *Service) Provenance(ctx context.Context, id int64) (*Provenance, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	var raw sql.NullString
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT provenance FROM observations
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		id, s.projectID,
	).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("observation %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("read provenance: %w", err)
	}
	p, _, err := parseProvenanceColumn(raw)
	return p, err
}
