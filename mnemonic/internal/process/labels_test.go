package process

import (
	"context"
	"database/sql"
	"testing"
)

// labelFixture builds a simple traceable chain (entry -> a). The pass is run
// with a counting LLM so the label-caching behavior is observable.
func labelFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	f := seedFile(t, db, "app/flow.go")
	entry := seedSymbol(t, db, f, "run", "function", 1)
	a := seedSymbol(t, db, f, "step", "function", 10)
	seedCall(t, db, entry, a, 2)
	return db
}

func entryID(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = ?`, name).Scan(&id); err != nil {
		t.Fatalf("lookup %s: %v", name, err)
	}
	return id
}

// TestProcessLabels covers 02.3 (Scenario: Process labels are cached by
// content-hash and re-labeled only on change): a flow is LLM-labeled once and
// the label is cached keyed to the process's content-hash; an unchanged
// re-run does NOT re-call the LLM (label reused); a changed flow (new
// content-hash) re-labels (a new LLM call for the new hash).
func TestProcessLabels(t *testing.T) {
	db := labelFixture(t)
	id := entryID(t, db, "run")
	entries := []Entry{{SymbolID: id, Kind: "cli-main"}}

	// Fresh store for the counting LLM (Run consults the cache; the first Run
	// on this graph has no stored label, so it must call the LLM once).
	c1 := &callCounter{label: "Run flow"}
	first, err := Run(context.Background(), db, entries, c1, RunOptions{})
	if err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	if len(first.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(first.Processes))
	}
	if first.Processes[0].Label != "Run flow" {
		t.Fatalf("expected label 'Run flow', got %q", first.Processes[0].Label)
	}
	if c1.calls != 1 {
		t.Fatalf("first Run should call the LLM once, got %d", c1.calls)
	}

	// Unchanged re-run: the cache hit returns the stored label and the LLM is
	// NOT called again.
	c2 := &callCounter{label: "SHOULD-NOT-APPEAR"}
	second, err := Run(context.Background(), db, entries, c2, RunOptions{})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if !second.FromCache {
		t.Errorf("unchanged re-run should be served from cache")
	}
	if c2.calls != 0 {
		t.Errorf("unchanged re-run must NOT re-call the LLM, got %d calls", c2.calls)
	}
	if len(second.Processes) != 1 || second.Processes[0].Label != "Run flow" {
		t.Fatalf("cached label not reused: %+v", second.Processes)
	}

	// Changed flow (new content-hash): the LLM is called again for the new
	// hash and a new label is produced.
	addChange(t, db)
	c3 := &callCounter{label: "Run flow v2"}
	third, err := Run(context.Background(), db, entries, c3, RunOptions{})
	if err != nil {
		t.Fatalf("Run 3: %v", err)
	}
	if third.FromCache {
		t.Errorf("changed flow must NOT be served from cache (new content-hash)")
	}
	if c3.calls == 0 {
		t.Errorf("changed flow should re-label (call the LLM for the new content-hash), got 0 calls")
	}
	if len(third.Processes) != 1 || third.Processes[0].Label != "Run flow v2" {
		t.Fatalf("changed flow should carry the new label, got %+v", third.Processes)
	}
}

// addChange appends a new callee to the flow, changing its structure (and
// therefore its content-hash) — the re-label trigger.
func addChange(t *testing.T, db *sql.DB) {
	t.Helper()
	var f int64
	if err := db.QueryRow(`SELECT file_id FROM symbols WHERE name = 'run'`).Scan(&f); err != nil {
		t.Fatalf("file: %v", err)
	}
	c := seedSymbol(t, db, f, "step2", "function", 20)
	var a int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = 'step'`).Scan(&a); err != nil {
		t.Fatalf("step: %v", err)
	}
	seedCall(t, db, a, c, 12)
}

// TestLLMDownUnlabeled covers 02.6 (Scenario: LLM down caches the flow
// unlabeled): with the LLM unavailable (it errors), the flow structure is
// still cached and it is cached without a fabricated label (empty label,
// label_status=unlabeled).
func TestLLMDownUnlabeled(t *testing.T) {
	db := labelFixture(t)
	id := entryID(t, db, "run")
	entries := []Entry{{SymbolID: id, Kind: "cli-main"}}

	down := stubLLM{label: "IGNORED", err: errLLMDown}
	res, err := Run(context.Background(), db, entries, down, RunOptions{})
	if err != nil {
		t.Fatalf("Run (LLM down): %v", err)
	}
	if len(res.Processes) != 1 {
		t.Fatalf("expected 1 process (structure cached), got %d", len(res.Processes))
	}
	p := res.Processes[0]
	// Structure is cached: the steps are present.
	if len(p.Steps) == 0 {
		t.Fatalf("flow structure not cached (no steps)")
	}
	// No fabricated label: empty label, unlabeled status.
	if p.Label != "" {
		t.Errorf("LLM-down flow must be cached UNLABELED (empty label), got %q", p.Label)
	}
	if p.LabelStatus != "unlabeled" {
		t.Errorf("LLM-down flow label_status should be 'unlabeled', got %q", p.LabelStatus)
	}

	// The empty label is persisted (not fabricated on read).
	listed, err := List(db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 persisted process, got %d", len(listed))
	}
	if listed[0].Label != "" {
		t.Errorf("persisted LLM-down flow must have an empty label, got %q", listed[0].Label)
	}
}
