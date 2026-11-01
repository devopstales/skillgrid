package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The handoff artifact (014 step 21) is a compact, agent-to-agent context block
// with two halves:
//
//   - a STABLE prefix (hub file summaries + repo file count) that only changes
//     when a hub file (or the file population) changes; and
//   - a DYNAMIC delta (changed file stubs, risk files, recent session events,
//     working set) computed fresh at generation time.
//
// Detection strategy (documented in the brief): the memory package cannot
// depend on the service layer (import cycle), and a fresh CLI process cannot
// rely on in-process state. We therefore persist a handoff cursor into the
// store's kv_meta table on every generation. The delta includes codeindex
// files whose mtime_ns is newer than the stored cursor (i.e. changed since the
// last handoff) plus every hub file (so hub changes are always visible). This
// is deterministic, cross-process, and testable.

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestHandoffPrefixStable(t *testing.T) {
	fx := newFixture(t, "handoff-prefix")
	ctx := context.Background()
	db := fx.st.DB

	repo := t.TempDir()
	hubRel := "AGENTS.md"
	nonHubRel := "src/util.go"
	writeRepoFile(t, repo, hubRel, "project agent config v1\n")
	writeRepoFile(t, repo, nonHubRel, "package util\n")

	seedHandoffFile(t, db, hubRel, 1000)
	seedHandoffFile(t, db, nonHubRel, 1000)

	gen := func() *Handoff {
		t.Helper()
		h, err := fx.svc.GenerateHandoff(ctx, repo)
		if err != nil {
			t.Fatalf("generate handoff: %v", err)
		}
		return h
	}

	h1 := gen()
	if len(h1.Prefix.HubFiles) == 0 {
		t.Fatalf("prefix must contain hub file summaries, got none")
	}
	hubSummary := findHubSummary(h1.Prefix.HubFiles, hubRel)
	if hubSummary == nil {
		t.Fatalf("prefix hub summaries missing %s: %+v", hubRel, h1.Prefix.HubFiles)
	}
	if h1.Prefix.FileCount != 2 {
		t.Fatalf("prefix file count = %d, want 2", h1.Prefix.FileCount)
	}

	// Modify a NON-hub file. The prefix must be unchanged (stable).
	time.Sleep(50 * time.Millisecond)
	writeRepoFile(t, repo, nonHubRel, "package util\n\nfunc New() {}\n")
	h2 := gen()
	p1 := h1.Prefix
	p2 := h2.Prefix
	if p1.FileCount != p2.FileCount {
		t.Fatalf("non-hub change altered file count: %d -> %d", p1.FileCount, p2.FileCount)
	}
	if len(p1.HubFiles) != len(p2.HubFiles) {
		t.Fatalf("non-hub change altered hub summary count: %d -> %d", len(p1.HubFiles), len(p2.HubFiles))
	}
	for i := range p1.HubFiles {
		if p1.HubFiles[i].Path != p2.HubFiles[i].Path || p1.HubFiles[i].Summary != p2.HubFiles[i].Summary {
			t.Fatalf("non-hub change altered a hub summary: %+v -> %+v", p1.HubFiles[i], p2.HubFiles[i])
		}
	}

	// Modify a HUB file. The prefix must now reflect the change.
	time.Sleep(50 * time.Millisecond)
	writeRepoFile(t, repo, hubRel, "project agent config v2 (updated)\n")
	h3 := gen()
	hubSummary3 := findHubSummary(h3.Prefix.HubFiles, hubRel)
	if hubSummary3 == nil {
		t.Fatalf("after hub change, prefix lost %s: %+v", hubRel, h3.Prefix.HubFiles)
	}
	if hubSummary3.Summary == hubSummary.Summary {
		t.Fatalf("hub change did not alter the hub summary (still %q)", hubSummary.Summary)
	}
}

func TestHandoffDeltaDynamic(t *testing.T) {
	fx := newFixture(t, "handoff-delta")
	ctx := context.Background()
	db := fx.st.DB

	repo := t.TempDir()
	hubRel := "go.mod"
	regular1 := "src/a.go"
	regular2 := "src/b.go"
	unchanged := "src/c.go"
	for _, rel := range []string{hubRel, regular1, regular2, unchanged} {
		writeRepoFile(t, repo, rel, "initial "+rel+"\n")
	}
	for _, rel := range []string{hubRel, regular1, regular2, unchanged} {
		seedHandoffFile(t, db, rel, 1000)
	}

	// Seed 12 session events so the "last 10" cap is exercised.
	if err := seedSessionEvents(t, fx, 12); err != nil {
		t.Fatalf("seed session events: %v", err)
	}

	h1, err := fx.svc.GenerateHandoff(ctx, repo)
	if err != nil {
		t.Fatalf("first handoff: %v", err)
	}
	// The first handoff has no cursor, so its delta is empty (nothing changed
	// since "never"). The cursor is now recorded.
	if len(h1.Delta.ChangedFiles) != 0 {
		t.Fatalf("first handoff should have an empty delta, got %d files", len(h1.Delta.ChangedFiles))
	}

	time.Sleep(50 * time.Millisecond)
	// Modify 3 files: 1 hub + 2 regular. Leave `unchanged` untouched.
	writeRepoFile(t, repo, hubRel, "module updated\n")
	writeRepoFile(t, repo, regular1, "package a\n\nfunc A() {}\n")
	writeRepoFile(t, repo, regular2, "package b\n\nfunc B() {}\n")

	h2, err := fx.svc.GenerateHandoff(ctx, repo)
	if err != nil {
		t.Fatalf("second handoff: %v", err)
	}

	paths := map[string]bool{}
	for _, cf := range h2.Delta.ChangedFiles {
		paths[cf.Path] = true
		if cf.Stub == "" {
			t.Fatalf("changed file %s has an empty stub", cf.Path)
		}
	}
	for _, want := range []string{hubRel, regular1, regular2} {
		if !paths[want] {
			t.Fatalf("delta missing changed file %s (got %v)", want, paths)
		}
	}
	if paths[unchanged] {
		t.Fatalf("delta wrongly includes unchanged file %s", unchanged)
	}

	// The hub file must be flagged as a risk file.
	foundRisk := false
	for _, rf := range h2.Delta.RiskFiles {
		if rf == hubRel {
			foundRisk = true
		}
	}
	if !foundRisk {
		t.Fatalf("hub file %s not flagged as a risk file: %v", hubRel, h2.Delta.RiskFiles)
	}

	// Recent events: last 10 of the 12 seeded.
	if len(h2.Delta.RecentEvents) != 10 {
		t.Fatalf("recent events = %d, want 10", len(h2.Delta.RecentEvents))
	}
	if h2.Delta.WorkingSet.Summary == "" {
		t.Fatalf("working set summary is empty")
	}

	// The delta is fresh: regenerating WITHOUT further changes drops the
	// regular changed files (the cursor advanced past the last generation). The
	// hub file is intentionally always present (hub changes must stay visible),
	// so the only remaining entry is the hub file.
	h3, err := fx.svc.GenerateHandoff(ctx, repo)
	if err != nil {
		t.Fatalf("third handoff: %v", err)
	}
	for _, cf := range h3.Delta.ChangedFiles {
		if cf.Path != hubRel {
			t.Fatalf("delta after no further changes should only keep the hub file, got %s", cf.Path)
		}
	}
}

func TestHandoffSavedToFile(t *testing.T) {
	fx := newFixture(t, "handoff-file")
	ctx := context.Background()

	repo := t.TempDir()
	writeRepoFile(t, repo, "AGENTS.md", "hub v1\n")
	seedHandoffFile(t, fx.st.DB, "AGENTS.md", 1000)

	outDir := t.TempDir()
	if err := fx.svc.SaveHandoff(ctx, repo, outDir); err != nil {
		t.Fatalf("save handoff: %v", err)
	}

	p := filepath.Join(outDir, "handoff.latest.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read handoff file: %v", err)
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("handoff JSON invalid: %v", err)
	}
	if _, ok := parsed["prefix"]; !ok {
		t.Fatalf("handoff JSON missing prefix section")
	}
	if _, ok := parsed["delta"]; !ok {
		t.Fatalf("handoff JSON missing delta section")
	}
	if parsed["generated_at"] == nil {
		t.Fatalf("handoff JSON missing generated_at: keys=%v", mapKeys(parsed))
	}

	// A second generation overwrites (not appends). The file must still parse
	// as a single JSON object after the second write.
	time.Sleep(50 * time.Millisecond)
	if err := fx.svc.SaveHandoff(ctx, repo, outDir); err != nil {
		t.Fatalf("save handoff (2nd): %v", err)
	}
	raw2, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read handoff file (2nd): %v", err)
	}
	if err := json.Unmarshal(raw2, &parsed); err != nil {
		t.Fatalf("second handoff JSON invalid (appended?): %v", err)
	}
	if len(raw2) == len(raw) && string(raw2) == string(raw) {
		// generated_at is at second precision, so a same-second rewrite can be
		// byte-identical; the parse check above already proves overwrite (a
		// second appended object would not parse as a single map).
		t.Logf("note: same-second rewrite, byte-identical is expected")
	}
}

func mapKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// findHubSummary returns the hub summary entry for rel, or nil.
func findHubSummary(hubs []HandoffHubFile, rel string) *HandoffHubFile {
	for i := range hubs {
		if hubs[i].Path == rel {
			return &hubs[i]
		}
	}
	return nil
}

// seedHandoffFile inserts a codeindex files row for the test repo.
func seedHandoffFile(t *testing.T, db *sql.DB, path string, mtimeNs int64) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT OR REPLACE INTO files (path, mtime_ns, size, content_hash, indexed_at)
		 VALUES (?, ?, 1, ?, '2026-01-01T00:00:00Z')`,
		path, mtimeNs, "h-"+path,
	); err != nil {
		t.Fatalf("seed handoff file %s: %v", path, err)
	}
}

// seedSessionEvents inserts n session_log observations (the "recent events"),
// each with a distinct created_at so the "last 10" recency ordering is testable.
func seedSessionEvents(t *testing.T, fx *fixture, n int) error {
	t.Helper()
	db := fx.st.DB
	for i := 0; i < n; i++ {
		ts := time.Now().UTC().Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		title := "event " + itoa(i)
		content := "recent session event " + itoa(i)
		hash := "nh-evt-" + itoa(i)
		if _, err := db.Exec(
			`INSERT INTO observations (
				session_id, type, title, content, project, scope,
				normalized_hash, revision_count, created_at, updated_at
			) VALUES (?, 'session_log', ?, ?, ?, '', ?, 0, ?, ?)`,
			fx.sessID, title, content, fx.svc.ProjectID(), hash, ts, ts,
		); err != nil {
			return err
		}
	}
	return nil
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }
