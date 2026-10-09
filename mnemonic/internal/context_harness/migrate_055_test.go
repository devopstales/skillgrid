// Package context_harness tests the Go-side Context Harness store migrations.
package context_harness

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestMigration055_TableCreated confirms migration 055 creates the
// context_revisions table. A fresh store.Open applies every embedded migration,
// so 055 lands by filename sort after 054.
func TestMigration055_TableCreated(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	var n int
	if err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='context_revisions'").Scan(&n); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if n != 1 {
		t.Fatalf("context_revisions table not created by migration 055 (count=%d)", n)
	}
}

// TestMigration055_IndexesCreated confirms the two supporting indexes land:
// the (session_id, revision) uniqueness that enforces one live revision per
// session, and the session lookup index.
func TestMigration055_IndexesCreated(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	for _, name := range []string{"idx_context_revisions_session", "idx_context_revisions_rev"} {
		var n int
		if err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?", name).Scan(&n); err != nil {
			t.Fatalf("query index %s: %v", name, err)
		}
		if n != 1 {
			t.Fatalf("index %s not created by migration 055 (count=%d)", name, n)
		}
	}
}

// TestMigration055_Idempotent reopens a fresh store and confirms the table is
// exactly one — 055's CREATE TABLE IF NOT EXISTS must not duplicate on a re-run
// (the runner also tracks it in index_meta, so it is skipped).
func TestMigration055_Idempotent(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir, "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()

	st, err = store.Open(dir, "ctxproj")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st.Close()

	var n int
	if err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='context_revisions'").Scan(&n); err != nil {
		t.Fatalf("query after reopen: %v", err)
	}
	if n != 1 {
		t.Fatalf("after reopen, context_revisions should exist exactly once, got %d", n)
	}
}
