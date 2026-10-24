// Prefetch injection — frozen snapshot rendering (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-05).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestRenderFrozenSnapshotTwoSections (BDD: "happy path — prefetch renders
//     two labeled sections"): compile error — facts.RenderFrozenSnapshot does
//     not exist yet; no frozen-block renderer exists in the package.
//   - TestPrefetchFrozenMidSession (BDD: "edge — prefetch is frozen
//     mid-session"): compile error — facts.NewSnapshot has no per-session
//     cache, so a second Request re-renders and would include the mid-session
//     add.
//   - TestRenderFrozenSnapshotEmptyStore (BDD: "failure — prefetch with empty
//     store renders empty sections"): compile error — same missing seam; the
//     contract is an empty store renders both headers at 0% with no entries
//     and no error.
//
// The fail-open "" contract (G7) lives in the adapters (plugin / MCP resource
// / CLI): every adapter maps a render error to an empty string so the system
// prompt injection can never break the session.
package facts

import (
	"fmt"
	"strings"
	"testing"
)

// seedPrefetchFact inserts one live fact row directly (bypassing the session
// trail: the prefetch fixture seeds the global store without a session) and
// sets its importance_score to a deterministic value.
func seedPrefetchFact(t *testing.T, s *Store, content, category string, score float64) int64 {
	t.Helper()
	id := seedFusedFact(t, s.DB(), content, category)
	if _, err := s.DB().Exec(
		`UPDATE facts SET importance_score = ? WHERE id = ?`, score, id,
	); err != nil {
		t.Fatalf("set importance score: %v", err)
	}
	return id
}

// sectionRule is the 40-char rule that opens and closes every section header
// (the acceptance shape: ═×40, label + usage line, ═×40).
const sectionRule = "════════════════════════════════════════"

// headerLine returns the exact header line for a section: the label plus the
// usage header "[xx% — used/limit chars]".
func headerLine(t *testing.T, label, used string, limit int, pct int) string {
	t.Helper()
	return fmt.Sprintf("%s [%d%% — %s/%s chars]", label, pct, used, withCommas(limit))
}

// TestRenderFrozenSnapshotTwoSections (BDD: "happy path — prefetch renders two
// labeled sections", G1–G4): the rendered block contains the GLOBAL MEMORY and
// USER PROFILE section headers, each with a usage header showing the percent
// and the "used/limit" char count (2,200 / 1,375 — the withCommas contract);
// entries within a section are separated by exactly "\n§\n"; entries are
// ordered by importance_score DESC (ties by id).
func TestRenderFrozenSnapshotTwoSections(t *testing.T) {
	_, gst, _ := openTwoStores(t)
	glob := New(gst.DB, GlobalProjectID)

	// The memory section gets three facts. The natural (id) order differs from
	// the importance order, so the sort assertion discriminates.
	seedPrefetchFact(t, glob, "always run gofmt before committing", CategoryMemory, 0.9) // id 1
	seedPrefetchFact(t, glob, "deploy window is tuesday mornings", CategoryMemory, 0.5)  // id 2 (lowest)
	seedPrefetchFact(t, glob, "use conventional commits", CategoryMemory, 1.0)           // id 3
	// The user section gets two facts (same shape, different budget 1375).
	seedPrefetchFact(t, glob, "user prefers concise answers", CategoryUser, 0.7)  // id 4
	seedPrefetchFact(t, glob, "user works with go and sqlite", CategoryUser, 0.8) // id 5

	snap, err := RenderFrozenSnapshot(ctx, glob)
	if err != nil {
		t.Fatalf("RenderFrozenSnapshot: %v", err)
	}

	// G1: both section headers are present.
	if !strings.Contains(snap, "GLOBAL MEMORY") {
		t.Errorf("snapshot missing the GLOBAL MEMORY section header:\n%s", snap)
	}
	if !strings.Contains(snap, "USER PROFILE") {
		t.Errorf("snapshot missing the USER PROFILE section header:\n%s", snap)
	}

	// G2: each section's usage header shows the percent and the used/limit
	// char count (withCommas: 2,200 and 1,375).
	memUsed := len("always run gofmt before committing") +
		len("deploy window is tuesday mornings") +
		len("use conventional commits")
	userUsed := len("user prefers concise answers") + len("user works with go and sqlite")
	if !strings.Contains(snap, headerLine(t, "GLOBAL MEMORY", withCommas(memUsed), CharLimit(CategoryMemory), pct(memUsed, CharLimit(CategoryMemory)))) {
		t.Errorf("snapshot missing the GLOBAL MEMORY usage header (%s/2,200, %d%%):\n%s",
			withCommas(memUsed), pct(memUsed, CharLimit(CategoryMemory)), snap)
	}
	if !strings.Contains(snap, headerLine(t, "USER PROFILE", withCommas(userUsed), CharLimit(CategoryUser), pct(userUsed, CharLimit(CategoryUser)))) {
		t.Errorf("snapshot missing the USER PROFILE usage header (%s/1,375, %d%%):\n%s",
			withCommas(userUsed), pct(userUsed, CharLimit(CategoryUser)), snap)
	}

	// G3 + G4: the whole block is exactly the two sections (headers + rule
	// lines), entries joined by "\n§\n", sorted by importance_score DESC.
	want := "\n" +
		sectionRule + "\n" +
		headerLine(t, "GLOBAL MEMORY", withCommas(memUsed), CharLimit(CategoryMemory), pct(memUsed, CharLimit(CategoryMemory))) + "\n" +
		sectionRule + "\n" +
		"use conventional commits\n§\nalways run gofmt before committing\n§\ndeploy window is tuesday mornings" + "\n\n" +
		sectionRule + "\n" +
		headerLine(t, "USER PROFILE", withCommas(userUsed), CharLimit(CategoryUser), pct(userUsed, CharLimit(CategoryUser))) + "\n" +
		sectionRule + "\n" +
		"user works with go and sqlite\n§\nuser prefers concise answers" + "\n"
	if snap != want {
		t.Errorf("snapshot does not match the acceptance shape.\nwant:\n%s\ngot:\n%s", want, snap)
	}
}

// pct is the production percent rule (round-half-up of used*100/limit, capped
// at 100; 0 for a zero used or limit) — the test asserts the rendered percent
// against the SAME computation, so a rounding-rule divergence cannot hide.
var pct = snapshotPct

// TestPrefetchFrozenMidSession (BDD: "edge — prefetch is frozen mid-session",
// G5): a Snapshot renders once per session id. A fact added after the first
// Request is NOT in the second Request's result — the cached block is returned
// verbatim. A different session id gets a fresh render (the cache is
// per-session, not global).
func TestPrefetchFrozenMidSession(t *testing.T) {
	_, gst, _ := openTwoStores(t)
	glob := New(gst.DB, GlobalProjectID)

	seedPrefetchFact(t, glob, "always run gofmt before committing", CategoryMemory, 0.9)

	snap := NewSnapshot(glob)
	first, err := snap.Request(ctx, "sess-a")
	if err != nil {
		t.Fatalf("first Request: %v", err)
	}
	if !strings.Contains(first, "always run gofmt before committing") {
		t.Fatalf("first render missing the seeded fact:\n%s", first)
	}

	// Mid-session add: the store gains a new fact after the snapshot froze.
	seedPrefetchFact(t, glob, "new fact added mid-session", CategoryMemory, 0.6)

	second, err := snap.Request(ctx, "sess-a")
	if err != nil {
		t.Fatalf("second Request: %v", err)
	}
	if second != first {
		t.Errorf("second Request for the same session must return the cached render, got a different string:\n%s", second)
	}
	if strings.Contains(second, "new fact added mid-session") {
		t.Errorf("frozen snapshot includes a mid-session fact:\n%s", second)
	}

	// A different session renders fresh (the cache key is the session id).
	other, err := snap.Request(ctx, "sess-b")
	if err != nil {
		t.Fatalf("Request for a new session: %v", err)
	}
	if !strings.Contains(other, "new fact added mid-session") {
		t.Errorf("a new session's render must see the mid-session fact:\n%s", other)
	}
}

// TestRenderFrozenSnapshotEmptyStore (BDD: "failure — prefetch with empty
// store renders empty sections", G6): an empty store renders both section
// headers with 0% usage and no entries, and no error.
func TestRenderFrozenSnapshotEmptyStore(t *testing.T) {
	_, gst, _ := openTwoStores(t)
	glob := New(gst.DB, GlobalProjectID)

	snap, err := RenderFrozenSnapshot(ctx, glob)
	if err != nil {
		t.Fatalf("RenderFrozenSnapshot on an empty store: %v", err)
	}
	// Empty sections render their rule + header + rule and nothing between
	// the two sections (no entries, no §): the block is the two header
	// blocks separated by a blank line, each ending in its closing rule.
	want := "\n" +
		sectionRule + "\n" +
		headerLine(t, "GLOBAL MEMORY", "0", CharLimit(CategoryMemory), 0) + "\n" +
		sectionRule + "\n\n" +
		sectionRule + "\n" +
		headerLine(t, "USER PROFILE", "0", CharLimit(CategoryUser), 0) + "\n" +
		sectionRule +
		"\n"
	if snap != want {
		t.Errorf("empty-store snapshot does not match the acceptance shape.\nwant (%d bytes):\n%q\ngot (%d bytes):\n%q", len(want), want, len(snap), snap)
	}
}

// TestRenderFrozenSnapshotNilStoreFailsOpen (G7, renderer seam): a nil store
// is a clean error at the seam; the fail-open "" contract is the adapters'
// (plugin / MCP resource / CLI) error mapping.
func TestRenderFrozenSnapshotNilStoreFailsOpen(t *testing.T) {
	if _, err := RenderFrozenSnapshot(ctx, nil); err == nil {
		t.Error("RenderFrozenSnapshot with a nil store should fail")
	}
}
