package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
)

// TestProvenanceChainStorage covers 15.1: saving an observation with a full
// provenance chain (session_id → curate_command → source_files →
// llm_reasoning) round-trips through Get with all four fields intact, and the
// underlying provenance column holds valid, parseable JSON. An observation
// saved without provenance reads back with a nil Provenance.
func TestProvenanceChainStorage(t *testing.T) {
	_, svc := newTestStore(t, "provproj")
	ctx := context.Background()
	sid := newSession(t, svc)

	chain := &Provenance{
		SessionID:     sid,
		CurateCommand: "mem_save --scope project",
		SourceFiles:   []string{"src/a.go", "src/b.go"},
		LLMReasoning:  "extracted from session summary: two related decisions",
	}
	id, err := svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "decision",
		Title:     "prov chain note",
		Content:   "content of the provenance note",
		Provenance: chain,
	})
	if err != nil {
		t.Fatalf("save with provenance: %v", err)
	}

	obs, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.Provenance == nil {
		t.Fatalf("retrieved observation has nil Provenance: %+v", obs)
	}
	p := obs.Provenance
	if p.SessionID != chain.SessionID {
		t.Fatalf("session_id: got %q, want %q", p.SessionID, chain.SessionID)
	}
	if p.CurateCommand != chain.CurateCommand {
		t.Fatalf("curate_command: got %q, want %q", p.CurateCommand, chain.CurateCommand)
	}
	if len(p.SourceFiles) != 2 || p.SourceFiles[0] != "src/a.go" || p.SourceFiles[1] != "src/b.go" {
		t.Fatalf("source_files: got %v, want [src/a.go src/b.go]", p.SourceFiles)
	}
	if p.LLMReasoning != chain.LLMReasoning {
		t.Fatalf("llm_reasoning: got %q, want %q", p.LLMReasoning, chain.LLMReasoning)
	}

	// The raw column must be valid JSON with all four snake_case keys.
	var raw sql.NullString
	if err := svc.store.DB.QueryRow(`
		SELECT provenance FROM observations WHERE id = ?`, id).Scan(&raw); err != nil {
		t.Fatalf("raw provenance column: %v", err)
	}
	if !raw.Valid || raw.String == "" {
		t.Fatalf("provenance column is empty, want JSON")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw.String), &decoded); err != nil {
		t.Fatalf("provenance column is not valid JSON: %v\n%s", err, raw.String)
	}
	for _, key := range []string{"session_id", "curate_command", "source_files", "llm_reasoning"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("provenance JSON missing key %q: %s", key, raw.String)
		}
	}

	// No provenance → nil on read (rollback boundary: old saves still work).
	plain, err := svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "decision",
		Title:     "prov plain note",
		Content:   "no provenance here",
	})
	if err != nil {
		t.Fatalf("save without provenance: %v", err)
	}
	obs, err = svc.Get(ctx, plain)
	if err != nil {
		t.Fatalf("get plain: %v", err)
	}
	if obs.Provenance != nil {
		t.Fatalf("plain observation should have nil Provenance, got %+v", obs.Provenance)
	}
}
