package context_harness

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func insertToolOutput(t *testing.T, st *store.Store, tool, output string) {
	t.Helper()
	if _, err := st.DB.Exec(`INSERT INTO tool_outputs (session_id, project_id, tool_name, output, size_bytes, created_at)
		VALUES (?,?,?,?,?, datetime('now'))`, "s1", "testproj", tool, output, len(output)); err != nil {
		t.Fatal(err)
	}
}

func insertIndexedFile(t *testing.T, st *store.Store, name, kind, language, content string) {
	t.Helper()
	if _, err := st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, content, created_at)
		VALUES (?,?,?,?,?, datetime('now'))`, "testproj", name, kind, language, content); err != nil {
		t.Fatal(err)
	}
}

func TestCtxQuery_FusesBothLegs(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "testproj")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// A cross-leg candidate: the SAME identity ("authflow") matched by both the
	// sandbox leg (a tool named authflow) and the indexed leg (a file named
	// authflow). Each at FTS rank 1 → fused score = 1/(K+1) + 1/(K+1).
	insertToolOutput(t, st, "authflow", "authflow output text")
	insertIndexedFile(t, st, "authflow", "file", "go", "authflow file content")

	// Single-leg candidates that must rank below the cross-leg hit.
	insertToolOutput(t, st, "sandboxonly", "sandboxonly unique")
	insertIndexedFile(t, st, "indexedonly", "file", "go", "indexedonly unique")

	// Query each token separately (FTS5 default is AND across tokens) and merge
	// distinct results by name.
	seen := map[string]bool{}
	var results []CtxResult
	for _, term := range []string{"authflow", "sandboxonly", "indexedonly"} {
		rs, err := CtxSearch(st.DB, term, 20)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rs {
			if !seen[r.Name] {
				seen[r.Name] = true
				results = append(results, r)
			}
		}
	}
	// 3 distinct candidates: authflow (both), sandboxonly (sandbox), indexedonly
	// (indexed).
	if len(results) != 3 {
		t.Fatalf("expected 3 fused results, got %d: %+v", len(results), results)
	}

	// The cross-leg candidate must be rank 1 and score the doubled value.
	top := results[0]
	if top.Name != "authflow" {
		t.Fatalf("cross-leg candidate must rank first, got %q", top.Name)
	}
	if top.Rank != 1 {
		t.Fatalf("top result rank must be 1, got %d", top.Rank)
	}
	want := 1.0/(RRFK+1) + 1.0/(RRFK+1)
	if top.Score != want {
		t.Fatalf("cross-leg fused score = %v, want %v (both legs at rank 1)", top.Score, want)
	}

	// Single-leg hits score strictly less than the cross-leg hit.
	for _, r := range results[1:] {
		if r.Score >= top.Score {
			t.Fatalf("single-leg %q score %v must be < cross-leg %v", r.Name, r.Score, top.Score)
		}
	}
}

func TestCtxQuery_FailOpenOneLeg(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "testproj")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Only sandbox data matches. The indexed leg has no matching row (or the
	// table may be absent) — it must not error and must not block the result.
	insertToolOutput(t, st, "web_search_exa", "onlyin-sandbox-zz")

	results, err := CtxSearch(st.DB, "onlyin-sandbox-zz", 20)
	if err != nil {
		t.Fatalf("fail-open: expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly one result, got %d", len(results))
	}
	if results[0].Source != sourceSandbox {
		t.Fatalf("expected source %q, got %q", sourceSandbox, results[0].Source)
	}
	if results[0].Score != 1.0/(RRFK+1) {
		t.Fatalf("single-leg score = %v, want %v", results[0].Score, 1.0/(RRFK+1))
	}
}

func TestCtxSearch_Limit(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "testproj")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	for i := 0; i < 5; i++ {
		insertToolOutput(t, st, "tool_x", "limitcheck")
	}
	results, err := CtxSearch(st.DB, "limitcheck", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		// All 5 sandbox rows share tool name "tool_x" → one fused candidate.
		t.Fatalf("expected 1 fused candidate, got %d", len(results))
	}
	if results[0].Rank != 1 {
		t.Fatalf("rank = %d, want 1", results[0].Rank)
	}
}
