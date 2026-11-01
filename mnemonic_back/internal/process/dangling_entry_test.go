package process

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestPersistProcessDanglingEntrySymbol covers the live-repo FK regression:
// an orphan-prune can delete a symbol that a process row still references via
// entry_symbol_id (selected earlier from a stale route edge). With
// foreign_keys=ON (as the index pass DB runs), persisting a process whose
// entry symbol no longer exists must NOT fail the whole pass — it should store
// NULL (the FK allows it) instead of aborting.
func TestPersistProcessDanglingEntrySymbol(t *testing.T) {
	db := openStore(t)
	// Enforce FKs exactly like the index pass DB (foreign_keys(1) DSN).
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	f := seedFile(t, db, "app/server.go")
	entry := seedSymbol(t, db, f, "handleHealth", "function", 1)

	// Delete the entry symbol (simulating an orphan-prune) so the process's
	// entry_symbol_id is now dangling.
	if _, err := db.Exec(`DELETE FROM symbols WHERE id = ?`, entry); err != nil {
		t.Fatalf("delete entry symbol: %v", err)
	}

	p := &Process{
		Name:          "handleHealth",
		EntrySymbolID: entry, // now dangling
		EntryKind:     "handler",
		ContentHash:   "abc123",
		Steps:         []Step{{SymbolID: entry, Name: "handleHealth"}},
		LabelStatus:   "unlabeled",
	}
	_, err := persistProcess(context.Background(), db, p)
	if err != nil {
		t.Fatalf("persistProcess with dangling entry symbol failed: %v", err)
	}
	// The row is persisted with entry_symbol_id NULLed (FK-safe).
	var storedID sql.NullInt64
	if err := db.QueryRow(`SELECT entry_symbol_id FROM processes WHERE content_hash = 'abc123'`).Scan(&storedID); err != nil {
		t.Fatalf("lookup process row: %v", err)
	}
	if storedID.Valid {
		t.Errorf("entry_symbol_id = %d, want NULL for a dangling entry", storedID.Int64)
	}
}
