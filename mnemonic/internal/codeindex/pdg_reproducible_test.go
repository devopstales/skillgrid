package codeindex

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
	"testing"
)

// pdgFingerprint hashes the full set of cfg_blocks/cfg_edges/pdg_edges rows
// (content only, no id) deterministically so a reproducibility check is one
// string compare.
func pdgFingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	var sb strings.Builder
	for _, q := range []string{
		`SELECT symbol_id, block_no, start_line, end_line, kind FROM cfg_blocks ORDER BY symbol_id, block_no`,
		`SELECT symbol_id, from_block, to_block, condition FROM cfg_edges ORDER BY symbol_id, from_block, to_block, condition`,
		`SELECT symbol_id, kind, from_line, to_line, from_name, to_name, confidence, note FROM pdg_edges ORDER BY symbol_id, kind, from_line, to_line, from_name, to_name, confidence`,
	} {
		rows, err := db.Query(q)
		if err != nil {
			t.Fatalf("query %q: %v", q, err)
		}
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				t.Fatalf("scan %q: %v", q, err)
			}
			sb.WriteString(rowsString(vals))
			sb.WriteByte('\n')
		}
		rows.Close()
		sb.WriteByte('\x1e')
	}
	h := fnv.New64a()
	h.Write([]byte(sb.String()))
	return fmt.Sprintf("%x", h.Sum64())
}

// TestPdgReproducible covers @step-01 (Scenario: Repeated PDG builds are
// reproducible): indexing the same fixture with --pdg twice (a fresh store each
// time) yields a byte-for-byte identical set of cfg_blocks/cfg_edges/pdg_edges
// rows (ids, types, confidence labels, ordering-independent); no wall-clock,
// pointer, or map-iteration-order nondeterminism leaks into the persisted rows.
func TestPdgReproducible(t *testing.T) {
	first := indexPdgFresh(t)
	second := indexPdgFresh(t)
	if first != second {
		t.Errorf("PDG not reproducible across fresh stores:\n got %s\nwant %s", first, second)
	}
}

// indexPdgFresh indexes the pdg fixture with --pdg into a brand-new store and
// returns its PDG fingerprint (content-only, so it is comparable across
// independent index runs).
func indexPdgFresh(t *testing.T) string {
	t.Helper()
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnablePDG()
	root := writePdgFixture(t)
	if _, err := idx.Run(contextBackground2(), root, pdgCfg); err != nil {
		t.Fatalf("run (--pdg): %v", err)
	}
	return pdgFingerprint(t, idx.store.DB)
}
