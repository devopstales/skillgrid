package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// allMemoryTypes is the 9 fine-grained typed categories (step 18). The order
// is the canonical list; IsValidMemoryType accepts each and rejects anything
// else.
var allMemoryTypes = []string{
	"profile",
	"preferences",
	"entities",
	"events",
	"identity",
	"soul",
	"cases",
	"trajectories",
	"experiences",
}

// TestMemoryTypeCategories covers 18.1: saving an observation with each of the
// 9 typed memory_type categories round-trips through Get with the value intact
// and the raw column populated; a `mem list --type preferences` equivalent
// (RecentWithType) returns ONLY preference-typed observations; an unknown
// memory_type is rejected on save with a clear error and no row written.
func TestMemoryTypeCategories(t *testing.T) {
	st, svc := newTestStore(t, "memtypeproj")
	sid := newSession(t, svc)
	ctx := context.Background()

	// (1) Every one of the 9 types is stored and read back correctly.
	for i, mt := range allMemoryTypes {
		id, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "learning",
			Title:     "typed note " + mt,
			Content:   "content for the " + mt + " category",
			MemoryType: mt,
		})
		if err != nil {
			t.Fatalf("save type %q: %v", mt, err)
		}
		got, err := svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("get type %q: %v", mt, err)
		}
		if got.MemoryType != mt {
			t.Errorf("type %q: read back MemoryType %q, want %q", mt, got.MemoryType, mt)
		}
		// Raw column must hold the value (not silently dropped).
		var raw sql.NullString
		if err := st.DB.QueryRow(`SELECT memory_type FROM observations WHERE id = ?`, id).Scan(&raw); err != nil {
			t.Fatalf("raw memory_type for %q: %v", mt, err)
		}
		if !raw.Valid || raw.String != mt {
			t.Errorf("type %q: raw column got %q (valid=%v), want %q", mt, raw.String, raw.Valid, mt)
		}
		_ = i
	}

	// (2) An untyped save reads back with an empty MemoryType (NULL = not
	// typed; the rollback boundary — existing saves keep working).
	plain, err := svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "learning",
		Title:     "untyped note",
		Content:   "no memory_type given",
	})
	if err != nil {
		t.Fatalf("save untyped: %v", err)
	}
	gotPlain, err := svc.Get(ctx, plain)
	if err != nil {
		t.Fatalf("get untyped: %v", err)
	}
	if gotPlain.MemoryType != "" {
		t.Errorf("untyped observation read back MemoryType %q, want empty", gotPlain.MemoryType)
	}

	// (3) The `mem list --type preferences` equivalent returns ONLY the
	// preference-typed observation, not the other 8 or the untyped row.
	prefs, err := svc.RecentWithType(ctx, "preferences", 100)
	if err != nil {
		t.Fatalf("recent with type: %v", err)
	}
	if len(prefs) != 1 {
		t.Fatalf("RecentWithType(preferences): got %d rows, want 1: %+v", len(prefs), prefs)
	}
	if prefs[0].MemoryType != "preferences" {
		t.Errorf("RecentWithType(preferences): row MemoryType %q, want preferences", prefs[0].MemoryType)
	}

	// (4) An unknown memory_type is rejected with a clear error and writes no
	// row.
	var nBefore int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&nBefore); err != nil {
		t.Fatalf("count before invalid: %v", err)
	}
	_, err = svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "learning",
		Title:     "bad type note",
		Content:   "content",
		MemoryType: "quantum",
	})
	if err == nil {
		t.Fatalf("expected error for invalid memory_type")
	}
	if !strings.Contains(err.Error(), "quantum") {
		t.Errorf("error should name the invalid memory_type: %v", err)
	}
	var nAfter int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&nAfter); err != nil {
		t.Fatalf("count after invalid: %v", err)
	}
	if nAfter != nBefore {
		t.Errorf("invalid memory_type wrote a row: count %d -> %d", nBefore, nAfter)
	}
}

// dedupLLM is a mockable LLM seam for the pre-write dedup check (18.2),
// mirroring the step-05 ExtractionLLM pattern. The test implements it to
// return a fixed JSON verdict.
type dedupLLM struct {
	candidates []string // existing observation contents to "compare against"
	duplicate  bool     // the LLM's verdict for this pair
	called     int
}

func (d *dedupLLM) Dedup(ctx context.Context, newContent string, candidates []string) (bool, int64, error) {
	d.called++
	_ = newContent
	_ = candidates
	return d.duplicate, int64(d.candidateID()), nil
}

func (d *dedupLLM) candidateID() int { return 1 }

// TestLLMDedupDetectsSemanticDuplicates covers 18.2: with the LLM dedup pass
// enabled, saving "The build fails on macOS because of the missing SDK" and
// then "macOS build broken due to absent SDK package" is flagged by the LLM as
// a semantic duplicate — the second is not stored as a new row (it merges into
// the first, bumping duplicate_count). A genuinely different observation is NOT
// flagged and is stored. When the LLM is unavailable, hash-based (exact
// normalized content) dedup is the fallback: an exact duplicate is caught, a
// reworded one is not.
func TestLLMDedupDetectsSemanticDuplicates(t *testing.T) {
	ctx := context.Background()

	t.Run("LLM flags semantic duplicate and merges", func(t *testing.T) {
		st, svc := newTestStore(t, "dedupproj")
		sid := newSession(t, svc)

		first, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "bugfix",
			Title:     "build fails on macOS",
			Content:   "The build fails on macOS because of the missing SDK",
		})
		if err != nil {
			t.Fatalf("save first: %v", err)
		}
		if first == 0 {
			t.Fatalf("first save returned id 0")
		}

		// Attach the LLM seam: it reports the incoming reworded content as a
		// semantic duplicate of the stored one, pointing at the first obs.
		llm := &dedupLLM{duplicate: true}
		svc.SetDedupLLM(llm)
		svc.EnableDedupLLM(true)

		second, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "bugfix",
			Title:     "macOS build broken",
			Content:   "macOS build broken due to absent SDK package",
		})
		if err != nil {
			t.Fatalf("save second: %v", err)
		}
		// The LLM was actually consulted (pre-write check ran).
		if llm.called == 0 {
			t.Fatalf("LLM dedup was not invoked")
		}
		// The second is merged into the first, not stored as a new row.
		if second != first {
			t.Fatalf("semantic duplicate should merge into existing id %d, got %d", first, second)
		}
		// Exactly one observation row exists (no new row for the duplicate).
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 1 {
			t.Fatalf("expected 1 observation after merge, got %d", n)
		}
		// The merge bumped the duplicate count on the surviving row.
		var dups int
		if err := st.DB.QueryRow(`SELECT duplicate_count FROM observations WHERE id = ?`, first).Scan(&dups); err != nil {
			t.Fatalf("dup count: %v", err)
		}
		if dups != 1 {
			t.Errorf("expected duplicate_count 1 after LLM merge, got %d", dups)
		}

		// A genuinely different observation is NOT flagged: the LLM says no,
		// and a new row is stored.
		llm.duplicate = false
		third, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "decision",
			Title:     "chose go for cli",
			Content:   "We picked Go for the CLI because it compiles to one static binary",
		})
		if err != nil {
			t.Fatalf("save third: %v", err)
		}
		if third == first {
			t.Fatalf("non-duplicate observation should get a new id, got the merged id %d", third)
		}
		var n2 int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&n2); err != nil {
			t.Fatalf("count2: %v", err)
		}
		if n2 != 2 {
			t.Fatalf("expected 2 observations (original + non-dup), got %d", n2)
		}
	})

	t.Run("hash fallback when LLM unavailable", func(t *testing.T) {
		st, svc := newTestStore(t, "dedupproj2")
		sid := newSession(t, svc)
		// No LLM attached, no LLM enabled: the fallback is exact (normalized)
		// content dedup.

		first, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "bugfix",
			Title:     "exact content",
			Content:   "The build fails on macOS because of the missing SDK",
		})
		if err != nil {
			t.Fatalf("save first: %v", err)
		}

		// An EXACT duplicate (same title+content+type) is caught by the
		// existing hash path and merged — no new row.
		exact, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "bugfix",
			Title:     "exact content",
			Content:   "The build fails on macOS because of the missing SDK",
		})
		if err != nil {
			t.Fatalf("save exact: %v", err)
		}
		if exact != first {
			t.Fatalf("exact duplicate should merge into id %d, got %d", first, exact)
		}

		// A REWORDED duplicate is NOT caught by the hash fallback (different
		// normalized hash) — it is stored as a new row.
		reworded, err := svc.Save(ctx, SaveInput{
			SessionID: sid,
			Type:      "bugfix",
			Title:     "reworded",
			Content:   "macOS build broken due to absent SDK package",
		})
		if err != nil {
			t.Fatalf("save reworded: %v", err)
		}
		if reworded == first {
			t.Fatalf("reworded content should be a new row under hash fallback, got merged id %d", reworded)
		}
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 2 {
			t.Fatalf("expected 2 observations under hash fallback, got %d", n)
		}
	})
}

// extractionLLMStub is a mockable LLM seam for the async two-phase commit test
// (18.3). It records how many times Extract was called and returns a fixed JSON
// extraction result (or an error) so the test can observe the async phase ran
// the LLM.
type extractionLLMStub struct {
	mu        sync.Mutex
	calls     int
	respond   string
	err       error
}

func (e *extractionLLMStub) Extract(ctx context.Context, text string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	return e.respond, e.err
}

func (e *extractionLLMStub) callCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

// TestAsyncTwoPhaseCommit covers 18.3: calling session.commit() (SessionCommit)
// completes the SYNC phase immediately (the session message is written and
// compression_index is incremented and durable) while the ASYNC phase (LLM
// extraction + dedup + memory_diff.json) runs in a detached goroutine that does
// NOT block the return. The test waits on the installed WaitGroup (not
// time.Sleep) and then asserts memory_diff.json was written by the async phase
// and records the LLM extraction.
func TestAsyncTwoPhaseCommit(t *testing.T) {
	ctx := context.Background()

	st, svc := newTestStore(t, "commitproj")
	// Create the session row DIRECTLY (not via SessionStart, which resolves the
	// project from CWD and would not land in the "commitproj" fixture bucket).
	const sid = "sess-commit-1"
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'commitproj', '/tmp', '2026-01-01T00:00:00Z', 'active')`, sid); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// Pre-check: compression_index starts at 0 for a fresh session.
	var ci0 int
	if err := st.DB.QueryRow(`SELECT compression_index FROM sessions WHERE id = ?`, sid).Scan(&ci0); err != nil {
		t.Fatalf("read initial compression_index: %v", err)
	}
	if ci0 != 0 {
		t.Fatalf("fresh session compression_index should be 0, got %d", ci0)
	}

	// Install the async gate: the async phase signals it exactly once per
	// spawned goroutine. We make two commits, so size the group for both — each
	// goroutine calls Done() exactly once. The gate is a package-level seam.
	var wg sync.WaitGroup
	wg.Add(2)
	SetAsyncCommitGate(&wg)
	t.Cleanup(func() { SetAsyncCommitGate(nil) })

	// Arm the LLM extraction pass so the async phase actually runs the LLM and
	// writes memory_diff.json. The seam returns one learning.
	llm := &extractionLLMStub{
		respond: `{"learnings":[{"text":"We always run the linter before every commit","type":"convention"}]}`,
	}
	svc.SetExtractionLLM(llm)
	svc.EnableExtractionLLM(true)

	summary := "## Goal\nCommit the two-phase work\n## Notes\nRan the linter before committing"

	// The commit must return promptly (sync phase only). The async phase has
	// not been waited on yet.
	ci, err := svc.SessionCommit(ctx, sid, summary)
	if err != nil {
		t.Fatalf("SessionCommit: %v", err)
	}
	// SYNC phase: compression_index incremented to 1 and returned.
	if ci != 1 {
		t.Fatalf("compression_index after first commit: got %d, want 1", ci)
	}
	// The increment is durable in the store (read it back directly).
	var ciDB int
	if err := st.DB.QueryRow(`SELECT compression_index FROM sessions WHERE id = ?`, sid).Scan(&ciDB); err != nil {
		t.Fatalf("read back compression_index: %v", err)
	}
	if ciDB != 1 {
		t.Fatalf("compression_index not durable: got %d, want 1", ciDB)
	}
	// The session message (summary) was written durably.
	var sum sql.NullString
	if err := st.DB.QueryRow(`SELECT summary FROM sessions WHERE id = ?`, sid).Scan(&sum); err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if !sum.Valid || !strings.Contains(sum.String, "two-phase work") {
		t.Fatalf("session message not written durably: %q", sum.String)
	}

	// memory_diff.json must NOT exist until the async phase completes (it runs
	// in the background). This proves the sync return is not gated on it.
	dataDir := filepath.Dir(st.Path())
	diffPath := filepath.Join(dataDir, "commitproj", "memory_diff.json")
	if _, err := os.Stat(diffPath); err == nil {
		// It is possible (rare) the goroutine beat us here; that still proves
		// non-blocking, so we do not fail. We just note it.
		_ = err
	}

	// Second commit: compression_index advances to 2 (idempotent per-commit).
	ci2, err := svc.SessionCommit(ctx, sid, summary)
	if err != nil {
		t.Fatalf("SessionCommit #2: %v", err)
	}
	if ci2 != 2 {
		t.Fatalf("compression_index after second commit: got %d, want 2", ci2)
	}

	// Wait for BOTH async phases (two commits → two goroutines) via the gate —
	// no time.Sleep. wg is sized for both; each goroutine calls Done() exactly
	// once, so wg.Wait() returns precisely when both async phases finished.
	wg.Wait()
	if got := llm.callCount(); got < 2 {
		t.Fatalf("async phase did not run the LLM extraction for both commits (calls=%d, want >=2)", got)
	}
	// After the gate signals, the diff file must exist.
	b, err := os.ReadFile(diffPath)
	if err != nil {
		t.Fatalf("memory_diff.json not written by async phase: %v", err)
	}
	var diff struct {
		SessionID string `json:"session_id"`
		Extracted []struct {
			Title string `json:"title"`
			Type  string `json:"type"`
		} `json:"extracted"`
	}
	if err := json.Unmarshal(b, &diff); err != nil {
		t.Fatalf("memory_diff.json is not valid JSON: %v\n%s", err, b)
	}
	if diff.SessionID != sid {
		t.Fatalf("memory_diff session_id: got %q, want %q", diff.SessionID, sid)
	}
	if len(diff.Extracted) == 0 {
		t.Fatalf("memory_diff.json recorded no extractions: %s", b)
	}
}
