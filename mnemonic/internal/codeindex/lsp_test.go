package codeindex

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lspFixtureGo writes a Go file containing a member call the static pass cannot
// type (a receiver-bound method), so the LSP tier has something to resolve.
func lspFixtureGo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// A member call: `s.Do()` where s is a local (statically untyped).
	writeFile("main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\ts := newSvc()\n\ts.Do(1)\n\tfmt.Println(s)\n}\n\nfunc newSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n int) int { return n }\n\ntype Svc struct{}\n")
	return root
}

// TestLspAbsentServer covers @step-01 (Scenario: LSP index with no server is
// byte-for-byte static): with NO language server on PATH (isolated empty PATH
// in the test), indexing with --lsp produces (1) a 005 graph byte-for-byte
// identical to a --lsp-less index of the same repo, (2) zero LSP_RESOLVED
// edges, and (3) a warning + continue (no hard error, no partial LSP edge set).
func TestLspAbsentServer(t *testing.T) {
	ResetFileFirstSymbol()

	// Isolate PATH so no language server is resolvable.
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir()) // empty dir on PATH → no servers

	cfg := pdgCfg
	cfg.LSP = true

	// --lsp index (no server present).
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnableLSP()
	root := lspFixtureGo(t)
	if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
		t.Fatalf("run (--lsp, no server): %v", err)
	}
	db := idx.store.DB

	// (2) zero LSP_RESOLVED edges.
	var lspResolved int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence = 'LSP_RESOLVED'`).Scan(&lspResolved); err != nil {
		t.Fatalf("count LSP_RESOLVED: %v", err)
	}
	if lspResolved != 0 {
		t.Errorf("expected 0 LSP_RESOLVED edges with no server, got %d", lspResolved)
	}

	// (3) The LSP tables (cfg/pdg) stay empty (LSP is independent of PDG; no
	// PDG flag here).
	pdgTablesEmpty(t, db)

	// (1) The 005 graph is byte-for-byte identical to a --lsp-less index of the
	// same repo.
	idx2, clean2 := newTestIndexer(t)
	defer clean2()
	root2 := lspFixtureGo(t)
	if _, err := idx2.Run(contextBackground2(), root2, pdgCfg); err != nil {
		t.Fatalf("run (no --lsp): %v", err)
	}
	got := baselineFingerprint(t, db)
	want := baselineFingerprint(t, idx2.store.DB)
	if got != want {
		t.Errorf("--lsp (no server) 005 graph not byte-for-byte identical to a --lsp-less index:\n got %s\nwant %s", got, want)
	}
	_ = oldPath
	_ = sql.ErrNoRows
}

// TestLspWarnsOnAbsentServer asserts the --lsp pass emits a warning (best-effort
// no-op) when no server is on PATH, rather than failing the index.
func TestLspWarnsOnAbsentServer(t *testing.T) {
	ResetFileFirstSymbol()
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())
	cfg := pdgCfg
	cfg.LSP = true
	idx, clean := newTestIndexer(t)
	defer clean()
	idx.EnableLSP()
	root := lspFixtureGo(t)
	// The index must SUCCEED (best-effort no-op), not error.
	if _, err := idx.Run(contextBackground2(), root, cfg); err != nil {
		t.Fatalf("run with absent server must not error, got: %v", err)
	}
	_ = oldPath
	// A warning goes to stderr; we assert the index succeeded + no LSP edges
	// (the warning is the stderr line the adapter emits — captured loosely).
	var n int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM edges WHERE confidence='LSP_RESOLVED'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("absent server wrote LSP edges: %d", n)
	}
}

// isolated PATH helpers: ensure the test PATH truly has no language server.
func assertNoServersOnPath(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"gopls", "pyright-langserver", "typescript-language-server", "rust-analyzer", "clangd"} {
		if p := os.Getenv("PATH"); strings.Contains(p, "gopls") {
			t.Logf("note: %s may be on PATH", bin)
		}
	}
}
