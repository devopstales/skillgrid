package store

import (
	"context"
	"testing"
)

// [projectwiki-rename-legacy-store] a store at version 201 (the pre-rename
// kb_* tables, with the 201 mode column) migrates through 202 on a full
// reopen: the projectwiki_* tables hold the same rows (soft-deleted page
// included), the legacy tables are gone, the run-mode value survives, and
// FTS retrieval now runs over projectwiki_pages_fts.
func TestProjectWikiRenameLegacyStore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// build the store at 201 (kb_* present, mode column from 201)
	st, err := OpenUpTo(ctx, dir, "pw-legacy", SQLite, "", "201_kb_run_mode.sql")
	if err != nil {
		t.Fatalf("open up to 201: %v", err)
	}
	raw := st.(*sqliteStore)
	if err := raw.RawExec(`INSERT INTO kb_pages (sync_id, project, rel_path, title, type, body, body_hash, updated_at)
		VALUES ('s-1', 'pw-legacy', 'decisions/001-alpha.md', 'Alpha decision', 'decision', 'Body of the alpha page.', 'h1', '2026-11-07T00:00:00Z')`); err != nil {
		t.Fatalf("seed live page: %v", err)
	}
	if err := raw.RawExec(`INSERT INTO kb_pages (sync_id, project, rel_path, title, type, body, body_hash, updated_at, deleted_at)
		VALUES ('s-2', 'pw-legacy', 'decisions/002-beta.md', 'Beta decision', 'decision', 'Body of the beta page.', 'h2', '2026-11-07T00:00:00Z', '2026-11-06T00:00:00Z')`); err != nil {
		t.Fatalf("seed soft-deleted page: %v", err)
	}
	if err := raw.RawExec(`INSERT INTO kb_run_state (project, run_id, phase, updated_at, mode)
		VALUES ('pw-legacy', 'run-1', 'idle', '2026-11-07T00:00:00Z', 'full')`); err != nil {
		t.Fatalf("seed run state: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close at 201: %v", err)
	}

	// full reopen applies 202
	st2, err := Open(ctx, dir, "pw-legacy", SQLite, "")
	if err != nil {
		t.Fatalf("full reopen: %v", err)
	}
	defer st2.Close()
	raw2 := st2.(*sqliteStore)

	// the same rows, soft-deleted one included
	n, err := raw2.ProbeInt(`SELECT COUNT(*) FROM projectwiki_pages WHERE project = 'pw-legacy'`)
	if err != nil {
		t.Fatalf("count pages: %v", err)
	}
	if n != 2 {
		t.Fatalf("projectwiki_pages rows = %d, want 2", n)
	}
	nd, err := raw2.ProbeInt(`SELECT COUNT(*) FROM projectwiki_pages WHERE deleted_at IS NOT NULL`)
	if err != nil {
		t.Fatalf("count deleted: %v", err)
	}
	if nd != 1 {
		t.Fatalf("soft-deleted rows = %d, want 1", nd)
	}
	ok, err := raw2.Probe(`SELECT COUNT(*) > 0 FROM projectwiki_pages WHERE title = 'Alpha decision' AND body = 'Body of the alpha page.'`)
	if err != nil {
		t.Fatalf("probe live page: %v", err)
	}
	if !ok {
		t.Fatal("live page row not preserved verbatim")
	}

	// the legacy tables (and with them the kb_* fts triggers) are gone
	legacy, err := raw2.ProbeInt(`SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'kb%'`)
	if err != nil {
		t.Fatalf("count legacy: %v", err)
	}
	if legacy != 0 {
		t.Fatalf("%d legacy kb_* catalog objects remain", legacy)
	}

	// the 201 mode column value survives on the renamed run state
	mode, err := raw2.RawQuery(`SELECT mode FROM projectwiki_run_state WHERE project = 'pw-legacy'`)
	if err != nil {
		t.Fatalf("read run state: %v", err)
	}
	if len(mode) != 1 || mode[0][0] != "full" {
		t.Fatalf("projectwiki_run_state.mode = %+v, want [full]", mode)
	}

	// FTS now runs over projectwiki_pages_fts
	m, err := raw2.ProbeInt(`SELECT COUNT(*) FROM projectwiki_pages_fts WHERE projectwiki_pages_fts MATCH 'Alpha decision'`)
	if err != nil {
		t.Fatalf("fts match: %v", err)
	}
	if m != 1 {
		t.Fatalf("fts match rows = %d, want 1", m)
	}
}

// [projectwiki-fresh-store] a fresh store opens straight onto the
// projectwiki_* schema (no kb_* catalog objects at all), and the 202 fts
// triggers stay in sync: an insert becomes matchable, a delete stops it.
func TestProjectWikiFreshStore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	st, err := Open(ctx, dir, "pw-fresh", SQLite, "")
	if err != nil {
		t.Fatalf("fresh open: %v", err)
	}
	defer st.Close()
	raw := st.(*sqliteStore)

	for _, table := range []string{"projectwiki_pages", "projectwiki_claims", "projectwiki_run_state", "projectwiki_page_baselines", "projectwiki_events", "projectwiki_pages_fts"} {
		ok, err := raw.Probe(`SELECT COUNT(*) > 0 FROM sqlite_master WHERE name = ?`, table)
		if err != nil {
			t.Fatalf("probe %s: %v", table, err)
		}
		if !ok {
			t.Fatalf("fresh store missing table %s", table)
		}
	}
	legacy, err := raw.ProbeInt(`SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'kb%'`)
	if err != nil {
		t.Fatalf("count legacy: %v", err)
	}
	if legacy != 0 {
		t.Fatalf("fresh store has %d kb_* catalog objects", legacy)
	}

	if err := raw.RawExec(`INSERT INTO projectwiki_pages (sync_id, project, rel_path, title, type, body, body_hash, updated_at)
		VALUES ('f-1', 'pw-fresh', 'glossary/gamma.md', 'Gamma term', 'glossary', 'A gamma definition body.', 'gf', '2026-11-07T00:00:00Z')`); err != nil {
		t.Fatalf("insert page: %v", err)
	}
	m, err := raw.ProbeInt(`SELECT COUNT(*) FROM projectwiki_pages_fts WHERE projectwiki_pages_fts MATCH 'Gamma term'`)
	if err != nil {
		t.Fatalf("fts after insert: %v", err)
	}
	if m != 1 {
		t.Fatalf("fts after insert = %d, want 1 (insert trigger broken)", m)
	}
	if err := raw.RawExec(`DELETE FROM projectwiki_pages WHERE project = 'pw-fresh'`); err != nil {
		t.Fatalf("delete page: %v", err)
	}
	m, err = raw.ProbeInt(`SELECT COUNT(*) FROM projectwiki_pages_fts WHERE projectwiki_pages_fts MATCH 'Gamma term'`)
	if err != nil {
		t.Fatalf("fts after delete: %v", err)
	}
	if m != 0 {
		t.Fatalf("fts after delete = %d, want 0 (delete trigger broken)", m)
	}
}
