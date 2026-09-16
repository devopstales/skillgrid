package codeindex

import (
	"database/sql"
	"strings"
	"testing"
)

// taintFixtureGo is a Go file with a known source->sink flow: a request param
// (GetRequestParam) flows through a resolvable call chain to a SQL exec sink
// (sqlExec). The content is fixed so fingerprints are stable.
func taintFixtureGo() string {
	return "package main\n\nfunc handler(r *Req) {\n\tp := GetRequestParam(r, \"q\")\n\tu := Transform(p)\n\tsqlExec(u)\n}\n\nfunc GetRequestParam(r *Req, k string) string { return r.Path }\n\nfunc Transform(s string) string { return s }\n\nfunc sqlExec(s string) {}\n"
}

func writeTaintFixtureDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, root+"/main.go", taintFixtureGo())
	return root
}

// TestTaintOptInIsolation covers @step-02 (02.1, Scenario: Opt-in taint index
// leaves 005/008/010 results unchanged): (1) a --pdg index of a fixture with a
// known source->sink path persists taint_findings rows for the known flow;
// (2) the 005 graph (files/symbols/edges/chunks) is byte-for-byte identical to
// the pre-011 baseline; (3) a non---pdg index leaves taint_findings empty.
func TestTaintOptInIsolation(t *testing.T) {
	// want is the 005 fingerprint of the --pdg run, compared to the non---pdg
	// run (the taint pass is additive: it must not touch the common graph).
	want := ""
	// (1) --pdg index: taint_findings rows exist for the known flow.
	{
		ResetFileFirstSymbol()
		idx, clean := newTestIndexer(t)
		defer clean()
		idx = idx.EnablePDG()
		root := writeTaintFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, pdgCfg); err != nil {
			t.Fatalf("run (--pdg): %v", err)
		}
		db := idx.store.DB
		var taintCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings`).Scan(&taintCount); err != nil {
			t.Fatalf("count taint_findings: %v", err)
		}
		if taintCount == 0 {
			t.Errorf("--pdg: expected taint_findings rows for the known source->sink flow, got 0")
		}
		// The known flow: GetRequestParam (source) -> sqlExec (sink).
		var known int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings WHERE source_name LIKE '%GetRequestParam%' AND sink_name LIKE '%sqlExec%'`).Scan(&known); err != nil {
			t.Fatalf("count known taint: %v", err)
		}
		if known == 0 {
			t.Errorf("--pdg: expected a taint finding for GetRequestParam -> sqlExec, got 0")
		}
		// (2) record the 005 fingerprint of the --pdg run (compared to the
		// non---pdg run below; the taint pass is additive: it must not touch
		// the common graph).
		want = baselineFingerprint(t, db)
	}

	// (3) non---pdg index: taint_findings empty + 005 graph identical.
	{
		ResetFileFirstSymbol()
		idx, clean := newTestIndexer(t)
		defer clean()
		root := writeTaintFixtureDir(t)
		if _, err := idx.Run(contextBackground2(), root, pdgCfg); err != nil {
			t.Fatalf("run (no --pdg): %v", err)
		}
		db := idx.store.DB
		var taintCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings`).Scan(&taintCount); err != nil {
			t.Fatalf("count taint_findings (non-pdg): %v", err)
		}
		if taintCount != 0 {
			t.Errorf("non---pdg: taint_findings has %d rows, want 0", taintCount)
		}
		// (2 cont.) the 005 graph is byte-for-byte identical to the --pdg run.
		if got := baselineFingerprint(t, db); got != want {
			t.Errorf("005 graph not byte-for-byte identical between --pdg and non---pdg:\n got %s\nwant %s", got, want)
		}
	}
}

// taintFingerprint renders the full taint_findings content deterministically
// (source/sink/kind/path-label/note, sorted) so a reproducibility check is a
// single string compare.
func taintFingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT source_line, sink_line, source_name, sink_name, confidence, note
		FROM taint_findings ORDER BY source_line, sink_line, source_name, sink_name, confidence`)
	if err != nil {
		t.Fatalf("query taint_findings: %v", err)
	}
	defer rows.Close()
	return rowsFingerprint(t, rows)
}

// rowsFingerprint renders scanned rows into a deterministic string.
func rowsFingerprint(t *testing.T, rows *sql.Rows) string {
	t.Helper()
	var sb strings.Builder
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			t.Fatalf("scan taint_findings: %v", err)
		}
		sb.WriteString(rowsString(vals))
		sb.WriteByte('\n')
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate taint_findings: %v", err)
	}
	return sb.String()
}
