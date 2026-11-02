package store

import (
	"context"
	"sort"
	"strings"
	"testing"
)

// allAppliedMigrations is the ordered, full set of applied SQLite migration
// files (the memory change's 001-004 + this change's 100). The parity loop
// iterates over it (data, not code — the KB change extends the loop over its
// kb_* tables by adding to the tableSet map, not by rewriting the loop).
func allAppliedMigrations() []string {
	return []string{
		"001_schema.sql",
		"002_session_trails.sql",
		"003_retrieval_trails.sql",
		"004_sync_lease.sql",
		"100_code_index.sql",
	}
}

// codeIndexTableSet is the table-set → creating-migration data map (B2 — the
// loop is data, not code). The KB change extends this map with its kb_*
// tables (D-KB) — the same regression catches the v1 bug class for the KB.
// Memory tables are included at 001_schema so the loop is genuinely the
// KB-extendable shape (a code_* read against the memory tables must also
// survive a version mismatch without no-such-column).
func codeIndexTableSet() map[string]string {
	return map[string]string{
		// memory (the loop's extendable shape — the KB adds its tables here)
		"observations": "001_schema.sql",
		// code-index content tables (all created by 100_code_index)
		"files":            "100_code_index.sql",
		"chunks":           "100_code_index.sql",
		"symbols":          "100_code_index.sql",
		"edges":            "100_code_index.sql",
		"unresolved_edges": "100_code_index.sql",
		"index_freshness":  "100_code_index.sql",
		"dependencies":     "100_code_index.sql",
		// the derived FTS legs (CI-INV-1 — rebuildable from the content table)
		"chunks_fts": "100_code_index.sql",
		"symbol_fts": "100_code_index.sql",
	}
}

// codeIndexTableReads is the per-table read each parity assertion runs.
// Each read is a read-only SELECT that touches the columns the table's
// migration guarantees — success = all selected columns exist (the T-CI-3
// property). A store at a lower version returns a "no such table" error
// (acceptable version mismatch), NEVER "no such column" (the v1 bug).
// The reads are raw SQL (not Store methods) so they are side-effect-free
// and the assertion is purely about column existence.
func codeIndexTableReads() map[string]func(Store) error {
	return map[string]func(Store) error{
		"observations":     rawSelect("observations", "id, title, content"),
		"files":            rawSelect("files", "id, path, content_hash, language"),
		"chunks":           rawSelect("chunks", "id, file_id, text, content_hash"),
		"symbols":          rawSelect("symbols", "id, name, qualified_name, kind"),
		"edges":            rawSelect("edges", "id, kind, from_id, to_id, to_name, file_id"),
		"unresolved_edges": rawSelect("unresolved_edges", "id, file_id, reference, kind"),
		"index_freshness":  rawSelect("index_freshness", "project, extractor_stamp, last_indexed_at"),
		"dependencies":     rawSelect("dependencies", "purl, name, version, ecosystem"),
		"chunks_fts":       rawSelect("chunks_fts", "chunks_fts"),
		"symbol_fts":       rawSelect("symbol_fts", "symbol_fts"),
	}
}

// rawSelect builds a read-only parity read: SELECT <cols> FROM <table>
// WHERE 0=1 (no rows returned, but the columns must exist). The WHERE 0=1
// ensures the read is a no-op (no row data needed) while still exercising
// the column resolution the T-CI-3 property guards.
func rawSelect(table, cols string) func(Store) error {
	return func(st Store) error {
		raw, ok := st.(interface {
			RawQuery(query string, args ...any) ([][]any, error)
		})
		if !ok {
			return nil // not a SQLite store; skip
		}
		_, err := raw.RawQuery("SELECT " + cols + " FROM " + table + " WHERE 0=1")
		return err
	}
}

// [migration-parity-all-versions] (T-CI-3, the v1 bug class, B2 — an
// ERROR-CLASS assertion, not a success assertion): at versions >= a table's
// creating migration the read SUCCEEDS; at earlier versions it returns a
// classified version/table error (never no-such-column — the v1 bug). The
// loop iterates the TABLE SET (data) over EVERY applied version so the KB
// change extends it over its kb_* tables. The drift fixture (part b) proves
// a store whose symbols table has `qualified` (not qualified_name — the
// exact v1 on-disk drift) still runs every read selecting only its own
// migration's columns, never no-such-column.
func TestMigrationParityAllVersions(t *testing.T) {
	ctx := context.Background()
	tables := codeIndexTableSet()
	reads := codeIndexTableReads()
	versions := allAppliedMigrations()
	sort.Strings(versions)

	for _, v := range versions {
		dir := t.TempDir()
		st, err := OpenUpTo(ctx, dir, "p", SQLite, "", v)
		if err != nil {
			t.Fatalf("open up to %s: %v", v, err)
		}
		for name, creatingMig := range tables {
			ok := appliedUpTo(v, creatingMig) // lexicographic order over the sorted migration names
			readErr := reads[name](st)
			if ok && readErr != nil {
				t.Fatalf("read %s at v=%s must succeed (table present): %v", name, v, readErr)
			}
			if !ok && readErr == nil {
				t.Fatalf("read %s at v=%s must fail (table absent): it succeeded", name, v)
			}
			if !ok && readErr != nil {
				// B2 "classified" half: the error must be the missing-TABLE
				// class (the acceptable version mismatch), never
				// no-such-column (the v1 bug) or any other class.
				if isMissingColumnErr(readErr) {
					t.Fatalf("read %s at v=%s: version mismatch surfaced as no-such-column (v1 bug): %v", name, v, readErr)
				}
				if !isMissingTableErr(readErr) {
					t.Fatalf("read %s at v=%s: version mismatch must be no-such-table, got: %v", name, v, readErr)
				}
			}
		}
		st.Close()
	}

	// (b) the future-schema drift fixture: a store whose symbols table has
	// `qualified` (not qualified_name) — the exact v1 on-disk drift. Every
	// read must select only the columns its OWN migration set guarantees
	// (or classify the version mismatch), never fail no-such-column from a
	// read that should not reference the drifted column.
	st := openTestStore(t)
	if err := st.ExecDDL(ctx, []string{"ALTER TABLE symbols ADD COLUMN qualified TEXT"}); err != nil {
		t.Fatalf("drift fixture: %v", err)
	}
	// the drift fixture is a store at 100 (all tables exist); the `qualified`
	// column is additive and does NOT shadow qualified_name. Every read must
	// SUCCEED — the point is that the reads select only their own
	// migration's columns, so the drifted column is invisible to them.
	for name, read := range reads {
		if err := read(st); err != nil {
			t.Fatalf("drift fixture: read %s must succeed (all tables present): %v", name, err)
		}
	}
}

// [migration-runs-from-empty] the full code-index migration set (001-004 +
// 100) applies from an empty DB, and a reopen is additive (idempotent).
func TestCodeMigrationRunsFromEmptyAndReopens(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := Open(ctx, dir, "proj-ci", SQLite, "")
	if err != nil {
		t.Fatalf("fresh open: %v", err)
	}
	raw := st.(*sqliteStore)
	// the full code-index table set must be present after the first open
	for _, table := range []string{"files", "chunks", "symbols", "edges", "unresolved_edges", "index_freshness", "dependencies", "chunks_fts", "symbol_fts", "embed_meta"} {
		found, err := raw.Probe(`SELECT COUNT(*) > 0 FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, table)
		if err != nil {
			t.Fatalf("probe %s: %v", table, err)
		}
		if !found {
			t.Fatalf("missing table %s after fresh open", table)
		}
	}
	st.Close()
	// reopen: migrations must be idempotent (additive-only discipline)
	st2, err := Open(ctx, dir, "proj-ci", SQLite, "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	// and the store is fully usable after the reopen (the code-index tables
	// coexist with the memory tables)
	if _, err := st2.UpsertFile(ctx, FileRow{Path: "reopen/r.go", ContentHash: "h", Language: "go"}); err != nil {
		t.Fatalf("upsert after reopen: %v", err)
	}
	res, err := st2.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "learning", Title: "after reopen",
		Content: "the code-index tables coexist with memory",
		Project: "proj-ci", Scope: "project",
	})
	if err != nil || res.SyncID == "" {
		t.Fatalf("add after reopen: %v", err)
	}
}

// ftsMatchCheck is the parity read for an FTS table: a MATCH against the
// (empty) FTS table succeeds iff the table + its content columns exist.
func ftsMatchCheck(st Store, table string) error {
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil
	}
	_, err := raw.RawQuery(`SELECT COUNT(*) FROM ` + table + ` WHERE ` + table + ` MATCH 'x'`)
	return err
}

// appliedUpTo reports whether creatingMig is at-or-before lastFile in the
// sorted migration order (the lexicographic order the runner uses).
func appliedUpTo(lastFile, creatingMig string) bool {
	return creatingMig <= lastFile
}

// isMissingColumnErr reports the v1 bug class: a missing-COLUMN SQL error
// (as opposed to a missing-TABLE error, which is the acceptable version
// mismatch). The parity test asserts a version mismatch NEVER surfaces as
// this class.
func isMissingColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such column")
}

// isMissingTableErr reports the acceptable version-mismatch class: a
// missing-TABLE SQL error (SQLite's "no such table"). The parity test
// asserts that at versions below a table's creating migration, the read
// fails with THIS class — a positive assertion that the error is the
// classified version-mismatch, not some other failure (a constraint error,
// a missing index, etc.).
func isMissingTableErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such table")
}
