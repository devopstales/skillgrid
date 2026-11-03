// Batch operations + char limit + substring matching (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-04).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestBatchFreeAndAdd (BDD: "happy path — batch frees space and adds new
//     fact"): compile error — facts.Batch, facts.Operation, facts.Op* do not
//     exist yet.
//   - TestOverflowReturnsStructuredError (BDD: "edge — overflow returns
//     structured error with current entries"): compile error — no char limit
//     enforcement, no facts.OverflowError.
//   - TestDedupRejectsExactDuplicate (BDD: "edge — exact duplicate is
//     rejected"): Add currently inserts the duplicate row — the
//     no-duplicate-added result and the single-row assertion fail.
//   - TestSubstringPatch / TestRemoveSubstringMatch / TestRemoveAmbiguousMatch
//     (BDD: "edge — patch uses substring matching", "failure — ambiguous
//     substring match returns error") / TestPatchNoMatchReturnsClosest
//     (BDD: "failure — no-match substring returns error with closest
//     entries"): compile error — no substring matching, no
//     facts.AmbiguousMatchError / facts.NoMatchError.
package facts

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// padTo builds a string of exactly n bytes by repeating "x" and ending in
// "y" (the tests need deterministic char counts for the 2200/1375 limits).
func padTo(n int) string {
	if n <= 1 {
		return strings.Repeat("x", n)
	}
	return strings.Repeat("x", n-1) + "y"
}

// openBatchFixture opens the two-store fixture, routes the global store as
// the batch data store, and seeds the given memory facts. Returns the data
// store (for Batch) and the raw global DB (for row assertions).
func openBatchFixture(t *testing.T, seedContents ...string) (*Store, *sql.DB) {
	t.Helper()
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}
	const sid = "sess-batch"
	seedTestSession(t, pst, sid)
	for i, c := range seedContents {
		if _, err := data.AddWith(ctx, sid, c, CategoryMemory, trail); err != nil {
			t.Fatalf("seed fact %d (%q): %v", i, c[:min(24, len(c))], err)
		}
	}
	return data, gst.DB
}

// openCategorizedBatchStore opens the two-store fixture and seeds one fact of
// the given category through the routed global store (the user-budget
// triangulation fixture).
func openCategorizedBatchStore(t *testing.T, category, content string) *Store {
	t.Helper()
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}
	const sid = "sess-user-batch"
	seedTestSession(t, pst, sid)
	if _, err := data.AddWith(ctx, sid, content, category, trail); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return data
}

// TestBatchFreeAndAdd (BDD: "happy path — batch frees space and adds new
// fact", G1): the store is at 2100/2200. A batch that removes the 200-char
// fact and adds a 250-char fact must SUCCEED — the char limit is checked on
// the FINAL result (2150/2200), not per-op. The add alone would overshoot
// (2300 > 2200), so a per-op check would wrongly reject the batch.
func TestBatchFreeAndAdd(t *testing.T) {
	const stale = "stale entry to remove"
	const fresh = "fresh fact that needs the freed space"
	// Seed to exactly 2100/2200: filler + stale200 = 2100, so
	// filler = 2100 - 200 = 1900.
	filler := padTo(1900)
	// stale itself must be exactly 200 chars for the 2100 total.
	stale200 := stale + strings.Repeat("z", 200-len(stale)-1) + "z"
	if len(stale200) != 200 {
		t.Fatalf("fixture: stale200 len %d, want 200", len(stale200))
	}
	// fresh must be exactly 250 chars.
	fresh250 := fresh + strings.Repeat("y", 250-len(fresh)-1) + "y"
	if len(fresh250) != 250 {
		t.Fatalf("fixture: fresh250 len %d, want 250", len(fresh250))
	}
	data, _ := openBatchFixture(t, filler, stale200)

	res, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpRemove, OldText: "stale entry to remove"},
		{Action: OpAdd, Content: fresh250},
	})
	if err != nil {
		t.Fatalf("Batch free+add at 2100/2200 (final 2150) must succeed: %v", err)
	}
	if !res.Success {
		t.Fatalf("Batch result not success: %+v", res)
	}

	// Final usage is exactly 2150/2200 (2100 - 200 removed + 250 added).
	var total int
	if err := totalOf(t, data, &total); err != nil {
		t.Fatal(err)
	}
	if total != 2150 {
		t.Errorf("final char usage = %d, want 2150", total)
	}
	if res.Usage != "2,150/2,200" {
		t.Errorf("result usage = %q, want \"2,150/2,200\"", res.Usage)
	}

	// The stale fact is soft-deleted, the fresh one is live.
	var liveStale, liveFresh int
	if err := countLive(t, data, "stale entry to remove", &liveStale); err != nil {
		t.Fatal(err)
	}
	if liveStale != 0 {
		t.Errorf("stale fact still live after the batch, want soft-deleted")
	}
	if err := countLive(t, data, "fresh fact that needs", &liveFresh); err != nil {
		t.Fatal(err)
	}
	if liveFresh != 1 {
		t.Errorf("fresh fact live rows = %d, want 1", liveFresh)
	}
}

// TestOverflowReturnsStructuredError (BDD: "edge — overflow returns
// structured error with current entries", G2): the store is at 2150/2200.
// A batch whose final result (2250) exceeds the limit must return a
// *OverflowError carrying current_entries (the existing fact contents) and
// the usage string "2,150/2,200" — and apply NO ops (atomic rollback).
func TestOverflowReturnsStructuredError(t *testing.T) {
	base := padTo(2150)
	data, gst := openBatchFixture(t, base)
	newFact := strings.Repeat("n", 100) // 100 chars → 2250 > 2200

	_, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpAdd, Content: newFact},
	})
	if err == nil {
		t.Fatal("Batch adding 100 chars at 2150/2200 must overflow")
	}
	var ovf *OverflowError
	if !errors.As(err, &ovf) {
		t.Fatalf("error is %T (%q), want *OverflowError", err, err)
	}
	if ovf.Usage != "2,150/2,200" {
		t.Errorf("OverflowError.Usage = %q, want \"2,150/2,200\"", ovf.Usage)
	}
	// current_entries is the PRE-batch store state (the agent's "what to
	// free" list) — not the post-batch set with the overflowed add in it.
	if len(ovf.CurrentEntries) != 1 || ovf.CurrentEntries[0] != base {
		t.Errorf("OverflowError.CurrentEntries = %q, want the pre-batch entry",
			prefixOf(ovf.CurrentEntries, 0))
	}

	// Atomic: no op was applied — the store still holds only the base fact.
	var n int
	if err := gst.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		t.Fatalf("count live facts: %v", err)
	}
	if n != 1 {
		t.Errorf("live facts after overflow = %d, want 1 (no ops applied)", n)
	}
}

// TestOverflowUserLimit (triangulation of the limit plumbing): the SAME
// overflow shape against the USER category must use the 1375 budget, not the
// 2200 memory budget — a 1300-char user fact + a 100-char add overflows at
// 1,300/1,375, while the identical byte counts fit under the memory budget.
func TestOverflowUserLimit(t *testing.T) {
	base := padTo(1300)
	data := openCategorizedBatchStore(t, CategoryUser, base)

	_, err := data.Batch(ctx, CategoryUser, []Operation{
		{Action: OpAdd, Content: strings.Repeat("u", 100)},
	})
	var ovf *OverflowError
	if !errors.As(err, &ovf) {
		t.Fatalf("error = %T (%q), want *OverflowError at the user budget", err, err)
	}
	if ovf.Usage != "1,300/1,375" {
		t.Errorf("OverflowError.Usage = %q, want \"1,300/1,375\"", ovf.Usage)
	}
}

// TestDedupRejectsExactDuplicate (BDD: "edge — exact duplicate is
// rejected", G3): adding content that exactly matches an existing fact is a
// SUCCESS ("no duplicate added"), not an error, and creates no second row.
func TestDedupRejectsExactDuplicate(t *testing.T) {
	const content = "user prefers dark mode"
	data, _ := openBatchFixture(t, content)

	res, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpAdd, Content: content},
	})
	if err != nil {
		t.Fatalf("exact duplicate add must not error: %v", err)
	}
	if !res.Success {
		t.Fatalf("exact duplicate add must succeed: %+v", res)
	}
	if !res.NoDuplicateAdded {
		t.Errorf("result must report no duplicate added: %+v", res)
	}
	var n int
	if err := totalOf(t, data, &n); err != nil {
		t.Fatal(err)
	}
	// Usage unchanged: one row, len(content) chars.
	if n != len(content) {
		t.Errorf("char usage after duplicate add = %d, want %d (one row)", n, len(content))
	}
}

// TestSubstringPatch (BDD: "edge — patch uses substring matching", G4):
// patch takes old_text + content; the fact CONTAINING old_text is updated by
// replacing the old_text portion with content. Exact-match matching is not
// sufficient — the fact is longer than old_text.
func TestSubstringPatch(t *testing.T) {
	const fact = "User prefers dark mode in all editors"
	data, _ := openBatchFixture(t, fact)

	res, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpPatch, OldText: "dark mode in all editors", Content: "light mode in VS Code"},
	})
	if err != nil {
		t.Fatalf("substring patch: %v", err)
	}
	if !res.Success {
		t.Fatalf("substring patch not success: %+v", res)
	}
	want := "User prefers light mode in VS Code"
	var got string
	if err := contentOf(t, data, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("fact after patch = %q, want %q", got, want)
	}
}

// TestRemoveSubstringMatch (G4): remove takes old_text and soft-deletes the
// fact containing it (the fact need not equal old_text).
func TestRemoveSubstringMatch(t *testing.T) {
	const fact = "User prefers light mode in VS Code"
	data, _ := openBatchFixture(t, fact)

	res, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpRemove, OldText: "light mode"},
	})
	if err != nil {
		t.Fatalf("substring remove: %v", err)
	}
	if !res.Success {
		t.Fatalf("substring remove not success: %+v", res)
	}
	var n int
	if err := totalOf(t, data, &n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("char usage after remove = %d, want 0 (soft-deleted)", n)
	}
}

// TestRemoveAmbiguousMatch (BDD: "failure — ambiguous substring match
// returns error", G5): old_text matching MORE THAN ONE live fact is a clean
// error — and atomic (no fact removed).
func TestRemoveAmbiguousMatch(t *testing.T) {
	const a = "User prefers dark mode in editors"
	const b = "Also likes dark mode in terminals"
	data, _ := openBatchFixture(t, a, b)

	_, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpRemove, OldText: "dark mode"},
	})
	var amb *AmbiguousMatchError
	if !errors.As(err, &amb) {
		t.Fatalf("error = %T (%q), want *AmbiguousMatchError", err, err)
	}
	var n int
	if err := totalOf(t, data, &n); err != nil {
		t.Fatal(err)
	}
	if n != len(a)+len(b) {
		t.Errorf("char usage after ambiguous remove = %d, want both facts intact", n)
	}
}

// TestPatchNoMatchReturnsClosest (BDD: "failure — no-match substring returns
// error with closest entries", G6): old_text matching ZERO live facts is a
// no-match error carrying closest_entries (the most similar existing facts),
// and applies no ops.
func TestPatchNoMatchReturnsClosest(t *testing.T) {
	const fact = "user prefers dark mode"
	data, _ := openBatchFixture(t, fact)

	_, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpPatch, OldText: "light mode in terminal", Content: "dark mode in terminal"},
	})
	var nm *NoMatchError
	if !errors.As(err, &nm) {
		t.Fatalf("error = %T (%q), want *NoMatchError", err, err)
	}
	if len(nm.ClosestEntries) == 0 {
		t.Fatal("NoMatchError.ClosestEntries is empty, want the closest existing facts")
	}
	// The closest entry must be the one existing fact (it shares "mode").
	if !strings.Contains(nm.ClosestEntries[0], "dark mode") {
		t.Errorf("ClosestEntries[0] = %q, want it to contain the existing fact", nm.ClosestEntries[0])
	}
	// No op applied: the fact is intact.
	var n int
	if err := totalOf(t, data, &n); err != nil {
		t.Fatal(err)
	}
	if n != len(fact) {
		t.Errorf("char usage after no-match patch = %d, want %d", n, len(fact))
	}
}

// TestPatchThenRemoveSameFact (triangulation of the plan-vs-re-resolve
// design): a patch that keeps a word in the content, followed by a remove
// matching that word, must target the SAME fact. With per-op re-resolution
// against the evolving set the remove would see zero matches (the patched
// content no longer contains the old substring) and fail.
func TestPatchThenRemoveSameFact(t *testing.T) {
	const fact = "User prefers dark mode in all editors"
	data, _ := openBatchFixture(t, fact)

	res, err := data.Batch(ctx, CategoryMemory, []Operation{
		{Action: OpPatch, OldText: "dark mode in all editors", Content: "light mode in VS Code"},
		{Action: OpRemove, OldText: "light mode in VS Code"},
	})
	if err != nil {
		t.Fatalf("patch-then-remove: %v", err)
	}
	if !res.Success {
		t.Fatalf("patch-then-remove not success: %+v", res)
	}
	var n int
	if err := totalOf(t, data, &n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("char usage after patch-then-remove = %d, want 0", n)
	}
}

// TestBatchRejectsUnknownAction: an unrecognized op action is a clean error
// before anything is applied (validation at the batch boundary).
func TestBatchRejectsUnknownAction(t *testing.T) {
	data, _ := openBatchFixture(t, "a fact")
	_, err := data.Batch(ctx, CategoryMemory, []Operation{{Action: "teleport", Content: "x"}})
	if err == nil {
		t.Fatal("unknown op action must error")
	}
	var ovf *OverflowError
	if errors.As(err, &ovf) {
		t.Errorf("unknown action error should not be an overflow: %v", err)
	}
}

// --- helpers ---------------------------------------------------------------

// totalOf sums the content length of all LIVE facts in the store.
func totalOf(t *testing.T, s *Store, total *int) error {
	t.Helper()
	rows, err := s.DB().Query(`SELECT content FROM facts WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return err
		}
		*total += len(c)
	}
	return rows.Err()
}

// countLive counts live facts whose content starts with the prefix.
func countLive(t *testing.T, s *Store, prefix string, n *int) error {
	t.Helper()
	return s.DB().QueryRow(`
		SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL AND content LIKE ?`,
		prefix+"%").Scan(n)
}

// contentOf returns the content of the store's single live fact.
func contentOf(t *testing.T, s *Store, content *string) error {
	t.Helper()
	var n int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		return errors.New("expected exactly one live fact")
	}
	return s.DB().QueryRow(`SELECT content FROM facts WHERE deleted_at IS NULL LIMIT 1`).Scan(content)
}

// prefixOf returns the first entry of entries (for error messages).
func prefixOf(entries []string, i int) string {
	if i >= len(entries) {
		return ""
	}
	if len(entries[i]) > 24 {
		return entries[i][:24]
	}
	return entries[i]
}
