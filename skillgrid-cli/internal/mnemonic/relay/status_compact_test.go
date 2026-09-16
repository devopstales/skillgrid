package relay

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStatus covers the happy path (Scenario: Status and compact without facts):
// a Status on a store that already has a handoff reports the true handoff count
// and surfaces the caller-supplied cost/context (and omits them when the caller
// does not supply them). No Fact Memory is involved.
func TestStatus(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s1")

	if _, _, err := Handoff(ctx, st.DB, "relayproj", "ho-status", root, Bundle{
		Progress:      "p",
		NextPrompt:    "np",
		SourceSession: "s1",
	}); err != nil {
		t.Fatalf("handoff: %v", err)
	}

	// Caller supplies cost/context → they surface in the snapshot.
	pct, cost := 42.0, 1.5
	snap, err := Status(ctx, st.DB, "relayproj", &Stats{
		ContextUsagePercent: &pct,
		CostUSD:             &cost,
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if snap.HandoffCount != 1 {
		t.Errorf("HandoffCount = %d, want 1", snap.HandoffCount)
	}
	if snap.ContextUsagePercent == nil || *snap.ContextUsagePercent != 42.0 {
		t.Errorf("ContextUsagePercent = %v, want 42.0 (caller-supplied)", snap.ContextUsagePercent)
	}
	if snap.CostUSD == nil || *snap.CostUSD != 1.5 {
		t.Errorf("CostUSD = %v, want 1.5 (caller-supplied)", snap.CostUSD)
	}

	// Caller supplies nothing → the relay must NOT invent cost/context.
	snap2, err := Status(ctx, st.DB, "relayproj", nil)
	if err != nil {
		t.Fatalf("status (nil stats): %v", err)
	}
	if snap2.HandoffCount != 1 {
		t.Errorf("HandoffCount = %d, want 1", snap2.HandoffCount)
	}
	if snap2.ContextUsagePercent != nil {
		t.Errorf("ContextUsagePercent must be nil when the caller does not supply it, got %v", *snap2.ContextUsagePercent)
	}
	if snap2.CostUSD != nil {
		t.Errorf("CostUSD must be nil when the caller does not supply it, got %v", *snap2.CostUSD)
	}
}

// TestStatusEmpty covers the edge (Scenario: No handoffs yet): a Status on an
// empty store reports HandoffCount 0 with no error (warn + continue, not a
// crash).
func TestStatusEmpty(t *testing.T) {
	st, _ := openStore(t, "relayproj")
	snap, err := Status(context.Background(), st.DB, "relayproj", nil)
	if err != nil {
		t.Fatalf("status on empty store must not error, got: %v", err)
	}
	if snap.HandoffCount != 0 {
		t.Errorf("HandoffCount = %d, want 0 (no handoffs yet)", snap.HandoffCount)
	}
}

// TestCompact covers the happy path (Scenario: Status and compact without
// facts): a THIN compact refreshes .cleave/KNOWLEDGE.md from handoff inputs
// with NO Fact Memory and returns the path. The knowledge body is preserved
// (the existing bundle KNOWLEDGE.md is non-empty).
func TestCompact(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s1")

	if _, _, err := Handoff(ctx, st.DB, "relayproj", "ho-compact", root, Bundle{
		Progress:      "p",
		Knowledge:     "fail closed first",
		NextPrompt:    "np",
		SourceSession: "s1",
	}); err != nil {
		t.Fatalf("handoff: %v", err)
	}

	res, err := CompactKnowledge(ctx, st.DB, "relayproj", root)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if res.KnowledgePath == "" {
		t.Fatalf("compact must return the knowledge path")
	}
	if _, err := os.Stat(res.KnowledgePath); err != nil {
		t.Fatalf("expected KNOWLEDGE.md at %s: %v", res.KnowledgePath, err)
	}
	body, err := os.ReadFile(res.KnowledgePath)
	if err != nil {
		t.Fatalf("read KNOWLEDGE.md: %v", err)
	}
	// The existing bundle knowledge (the handoff input) is preserved.
	if !strings.Contains(string(body), "fail closed first") {
		t.Errorf("KNOWLEDGE.md should preserve the handoff knowledge, got %q", string(body))
	}
	if res.Empty {
		t.Errorf("Empty must be false when a non-empty knowledge input exists")
	}
}

// TestCompactEmpty covers the edge (Scenario: No handoffs yet / empty
// knowledge inputs): a compact with no handoff inputs writes an empty or
// minimal KNOWLEDGE.md (warn + continue, not an error) and reports Empty=true.
func TestCompactEmpty(t *testing.T) {
	st, root := openStore(t, "relayproj")
	// No handoff rows, no .cleave/ bundle → empty/missing knowledge inputs.

	res, err := CompactKnowledge(context.Background(), st.DB, "relayproj", root)
	if err != nil {
		t.Fatalf("compact with empty inputs must not error, got: %v", err)
	}
	if res.KnowledgePath == "" {
		t.Fatalf("compact must return the knowledge path even when empty")
	}
	if _, err := os.Stat(res.KnowledgePath); err != nil {
		t.Fatalf("expected a KNOWLEDGE.md (empty or minimal) at %s: %v", res.KnowledgePath, err)
	}
	if !res.Empty {
		t.Errorf("Empty must be true when there are no knowledge inputs")
	}
	body, err := os.ReadFile(res.KnowledgePath)
	if err != nil {
		t.Fatalf("read KNOWLEDGE.md: %v", err)
	}
	if strings.TrimSpace(string(body)) == "" {
		t.Errorf("KNOWLEDGE.md should be minimal (a title/note), not a zero-byte file; got empty")
	}
	// The file lives under the cleave dir.
	if !strings.Contains(res.KnowledgePath, filepath.Join(".skillgrid", ".cleave")) {
		t.Errorf("KNOWLEDGE.md path %s should be under .skillgrid/.cleave", res.KnowledgePath)
	}
}

// TestCompactFromContextNotes covers the handoff-inputs-without-KNOWLEDGE path:
// when the .cleave/KNOWLEDGE.md is absent/empty but a session handoff row has a
// context_summary, the compact folds that session note into KNOWLEDGE.md (still
// no Fact Memory).
func TestCompactFromContextNotes(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s1")

	// A handoff whose KNOWLEDGE.md is empty but that carries a context_summary.
	if _, _, err := Handoff(ctx, st.DB, "relayproj", "ho-notes", root, Bundle{
		Progress:       "p",
		Knowledge:      "", // empty knowledge → the compact must fall back to the note
		NextPrompt:     "np",
		SourceSession:  "s1",
		ContextSummary: "folded session note",
	}); err != nil {
		t.Fatalf("handoff: %v", err)
	}

	res, err := CompactKnowledge(ctx, st.DB, "relayproj", root)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	body, err := os.ReadFile(res.KnowledgePath)
	if err != nil {
		t.Fatalf("read KNOWLEDGE.md: %v", err)
	}
	if !strings.Contains(string(body), "folded session note") {
		t.Errorf("KNOWLEDGE.md should fold the context_summary note, got %q", string(body))
	}
	if res.Empty {
		t.Errorf("Empty must be false when a context_summary note is present")
	}
}
