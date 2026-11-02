package store

import (
	"fmt"
	"strings"
)

// FTSDDL renders the reusable external-content FTS5 DDL + the four-trigger
// sync set (T21/M11) for the named FTS table over the named content
// table's columns. It returns exactly 5 statements:
//
//  1. CREATE VIRTUAL TABLE <name> USING fts5(<cols>, tokenize, content, content_rowid)
//  2. CREATE TRIGGER <name>_insert   AFTER INSERT  (index a new row)
//  3. CREATE TRIGGER <name>_delete   BEFORE DELETE (remove the old row)
//  4. CREATE TRIGGER <name>_update   BEFORE UPDATE (remove the old row)
//  5. CREATE TRIGGER <name>_update_insert AFTER UPDATE (index the new row)
//
// The unit is parameterized on (table name, content table, content rowid,
// content columns, tokenizer) so the code-index's chunks_fts/symbol_fts and
// the KB's kb_pages_fts all render from it verbatim (CI-INV-1 — the FTS leg
// is derived, rebuildable from the content table). The memory change's
// observations_fts/prompts_fts share the identical pattern (001_schema.sql)
// but stay as-is in that migration (minimal diff; the code-index migration
// is the required consumer).
//
// The trigger set keeps the FTS index in step with the content table. The
// BEFORE UPDATE / BEFORE DELETE use a plain DELETE FROM … WHERE rowid=old.<id>
// (not the trigram tokenizer's 'delete' special command — that command is
// unsafe for a multi-column external-content table, so delete + insert).
//
// The rendered DDL is *executable* against a real store (TestFtsUnitReusable
// applies it via ExecDDL) — not a string shape. A unit-created table stays
// in sync with its content table (insert → MATCH finds; update → new term
// matches + old stops; delete → empty).
func FTSDDL(name, contentTable, contentRowID string, cols []string, tokenize string) []string {
	return append(
		[]string{renderFTSVirtualTable(name, contentTable, contentRowID, cols, tokenize)},
		renderFTSTriggers(name, contentTable, contentRowID, cols)...,
	)
}

// RankingWeights renders the BM25 composite-rank expression for the named
// FTS table with the given per-column weights (T21). The memory store's
// Search uses this shape (searchRankRelevanceBase in sqlite.go); the KB's
// kb_pages_fts re-uses it verbatim. The returned SQL fragment is a
// SELECT-able expression: MAX(bm25(<fts>, w1, w2, ..., 0.0, 0.0, 0.0), 1.0)
// where the number of weight args matches the FTS column count (one weight
// per column, zero-padded to the bm25 6-arg shape).
func RankingWeights(ftsTable string, weights ...float64) string {
	args := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		if i < len(weights) {
			args = append(args, fmt.Sprintf("%.1f", weights[i]))
		} else {
			args = append(args, "0.0")
		}
	}
	return fmt.Sprintf("MAX(bm25(%s, %s), 1.0)", ftsTable, strings.Join(args, ", "))
}

// renderFTSTriggers is the FTSDDL trigger-set builder (one statement per
// trigger, the four-sync-set shape). The migration's FTS DDL + the unit
// share this one code path (they cannot drift — the migration 100's FTS
// tables are rendered by it, TestFTSStaysInSyncCode pins the sync).
func renderFTSTriggers(name, contentTable, contentRowID string, cols []string) []string {
	colList := strings.Join(cols, ", ")
	rowIDList := make([]string, len(cols))
	for i, c := range cols {
		rowIDList[i] = "new." + c
	}
	return []string{
		fmt.Sprintf(`CREATE TRIGGER %s_insert AFTER INSERT ON %s BEGIN
	INSERT INTO %s(rowid, %s)
	VALUES (new.%s, %s);
END;`, name, contentTable, name, colList, contentRowID, strings.Join(rowIDList, ", ")),
		fmt.Sprintf(`CREATE TRIGGER %s_delete BEFORE DELETE ON %s BEGIN
	DELETE FROM %s WHERE rowid = old.%s;
END;`, name, contentTable, name, contentRowID),
		fmt.Sprintf(`CREATE TRIGGER %s_update BEFORE UPDATE ON %s BEGIN
	DELETE FROM %s WHERE rowid = old.%s;
END;`, name, contentTable, name, contentRowID),
		fmt.Sprintf(`CREATE TRIGGER %s_update_insert AFTER UPDATE ON %s BEGIN
	INSERT INTO %s(rowid, %s)
	VALUES (new.%s, %s);
END;`, name, contentTable, name, colList, contentRowID, strings.Join(rowIDList, ", ")),
	}
}

// renderFTSVirtualTable is the FTSDDL virtual-table statement (the
// external-content FTS5 shape: content='<table>', content_rowid='<rowid>',
// tokenize='<tokenizer>').
func renderFTSVirtualTable(name, contentTable, contentRowID string, cols []string, tokenize string) string {
	return fmt.Sprintf(`CREATE VIRTUAL TABLE %s USING fts5(
%s,
tokenize='%s',
content='%s',
content_rowid='%s'
)`, name, strings.Join(cols, ",\n"), tokenize, contentTable, contentRowID)
}
