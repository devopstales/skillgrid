package process

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// dispatchFixture builds a chain that reaches a dispatch boundary:
// entry (function) -> svc (interface). The trace must stop at svc with a
// "stops at <symbol> (<reason>)" note, not silently cut.
func dispatchFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	f := seedFile(t, db, "app/handler.go")
	entry := seedSymbol(t, db, f, "processOrder", "function", 1)
	svc := seedSymbol(t, db, f, "OrderService", "interface", 10)
	seedCall(t, db, entry, svc, 2)
	return db
}

// TestDispatchStop covers 02.4 (Scenario: Trace stops at a dispatch boundary
// with a note): a call chain that reaches an interface->impl boundary is
// truncated with a "stops at <symbol> (<reason>)" note (reusing the 005
// graph-stops idea), not silently cut. code_process <name> (Get) shows the
// stop note; each hop carries a confidence label.
func TestDispatchStop(t *testing.T) {
	db := dispatchFixture(t)
	var entryID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = 'processOrder'`).Scan(&entryID); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	res, err := Run(context.Background(), db, []Entry{{SymbolID: entryID, Kind: "cli-main"}}, nil, RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(res.Processes))
	}
	p := res.Processes[0]
	if p.Stop == nil {
		t.Fatalf("expected a dispatch stop note, got none (trace was not truncated at the boundary)")
	}
	if p.Stop.Symbol != "OrderService" {
		t.Errorf("stop note symbol = %q, want OrderService", p.Stop.Symbol)
	}
	if p.Stop.Reason == "" {
		t.Errorf("stop note has no reason (not a 'stops at <symbol> (<reason>)' note)")
	}
	if !strings.Contains(p.Stop.Reason, "interface") {
		t.Errorf("stop reason should name the interface dispatch, got %q", p.Stop.Reason)
	}
	// The trace is NOT silently cut: the interface step is recorded as a step
	// (it was reached) AND the stop note names it.
	var sawSvc bool
	for _, s := range p.Steps {
		if s.Name == "OrderService" {
			sawSvc = true
		}
	}
	if !sawSvc {
		t.Errorf("the boundary symbol OrderService should be recorded in the steps (trace not silently cut)")
	}

	// code_process <name> (Get) surfaces the stop note + per-hop confidence.
	got, err := Get(db, p.Name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Stop == nil || got.Stop.Symbol != "OrderService" {
		t.Errorf("Get did not surface the stop note: %+v", got.Stop)
	}
	// Every non-entry hop carries a confidence label.
	for i, s := range got.Steps {
		if i > 0 && s.Confidence == "" {
			t.Errorf("hop %d (-> %s) has no confidence label", i, s.Name)
		}
	}
}

// untraceableFixture is an entry point with no outgoing call edges.
func untraceableFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	f := seedFile(t, db, "app/leaf.go")
	seedSymbol(t, db, f, "lone", "function", 1)
	return db
}

// TestUntraceableEntry covers 02.7 (Scenario: Untraceable entry point yields
// single-step or is skipped): an entry with no traceable call chain yields a
// single-step process (just the entry) or is skipped — no flow is fabricated.
func TestUntraceableEntry(t *testing.T) {
	db := untraceableFixture(t)
	var entryID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = 'lone'`).Scan(&entryID); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	res, err := Run(context.Background(), db, []Entry{{SymbolID: entryID, Kind: "cli-main"}}, nil, RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Processes) != 1 {
		t.Fatalf("expected a single-step (or skipped) process, got %d", len(res.Processes))
	}
	p := res.Processes[0]
	// Single-step: only the entry symbol, no fabricated callees.
	if len(p.Steps) != 1 {
		t.Fatalf("untraceable entry should be single-step, got %d steps: %+v", len(p.Steps), p.Steps)
	}
	if p.Steps[0].Name != "lone" {
		t.Errorf("single step should be the entry symbol, got %q", p.Steps[0].Name)
	}
}

// crossFixture builds two communities (01) and a chain that crosses them:
// entry (comm A) -> other (comm B). The flow must carry a cross-community flag.
func crossFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	f := seedFile(t, db, "app/cross.go")
	entry := seedSymbol(t, db, f, "startHere", "function", 1)
	other := seedSymbol(t, db, f, "otherSide", "function", 10)
	seedCall(t, db, entry, other, 2)
	// Assign the two symbols to different communities (01's table).
	if _, err := db.Exec(`INSERT INTO communities (id, symbol_id) VALUES (0, ?), (1, ?)`, entry, other); err != nil {
		t.Fatalf("seed communities: %v", err)
	}
	return db
}

// TestCrossCommunityFlag covers 02.8 (Scenario: Cross-community process is
// flagged): a flow that passes through symbols in more than one community
// carries a cross-community flag.
func TestCrossCommunityFlag(t *testing.T) {
	db := crossFixture(t)
	var entryID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = 'startHere'`).Scan(&entryID); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	res, err := Run(context.Background(), db, []Entry{{SymbolID: entryID, Kind: "cli-main"}}, nil, RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(res.Processes))
	}
	if !res.Processes[0].CrossCommunity {
		t.Errorf("cross-community flow should be flagged, got CrossCommunity=false")
	}
	// And the flag is persisted (code_processes reads it from the table).
	listed, err := List(db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 || !listed[0].CrossCommunity {
		t.Errorf("persisted process should carry the cross-community flag, got %+v", listed)
	}
}

// singleCommunityFixture builds a chain entirely within one community.
func singleCommunityFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	f := seedFile(t, db, "app/one.go")
	entry := seedSymbol(t, db, f, "a", "function", 1)
	b := seedSymbol(t, db, f, "b", "function", 10)
	seedCall(t, db, entry, b, 2)
	if _, err := db.Exec(`INSERT INTO communities (id, symbol_id) VALUES (0, ?), (0, ?)`, entry, b); err != nil {
		t.Fatalf("seed communities: %v", err)
	}
	return db
}

// TestCrossCommunityNotFlaggedWhenSingleCommunity is the negative control: a
// flow inside a single community is NOT flagged.
func TestCrossCommunityNotFlaggedWhenSingleCommunity(t *testing.T) {
	db := singleCommunityFixture(t)
	var entryID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = 'a'`).Scan(&entryID); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	res, err := Run(context.Background(), db, []Entry{{SymbolID: entryID, Kind: "cli-main"}}, nil, RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(res.Processes))
	}
	if res.Processes[0].CrossCommunity {
		t.Errorf("single-community flow should NOT be flagged cross-community")
	}
}
