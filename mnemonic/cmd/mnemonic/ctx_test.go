package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// runCtxCapture runs fn with stdout+stderr redirected and returns their combined
// output. Used for the `ctx` subcommands that write to the standard streams.
// runCtxCapture runs fn with stdout+stderr redirected to a pipe and returns
// their combined output. os.Stdout/os.Stderr must be *os.File, so we use a
// pipe: write end replaced, fn runs, close the write end, then read all.
func runCtxCapture(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout = w
	os.Stderr = w
	fn()
	// Close the write end BEFORE reading so ReadAll sees EOF.
	w.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(out)
}

// ctxDataDir seeds the context-harness tables (sandbox tool_outputs + indexed
// files) under a temp data dir and points the CLI at it, mirroring the loop
// command test idiom (SKILLGRID_MNEMONIC_DATA_DIR). It returns the dir so the
// caller can inspect rows directly if needed.
func ctxSeed(t *testing.T) string {
	t.Helper()
	// newMnemonicService caches cliService (package-level), so each test must
	// clear it to build a service for its own temp data dir.
	oldSvc := cliService
	t.Cleanup(func() { cliService = oldSvc })
	cliService = nil
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	// Resolve the project the way openLoop would (OpenForDirectory falls back to
	// the active project file, then to the env). The CLI's openLoop opens by
	// directory; for a temp dir with no boot file it uses the default project
	// resolution. We mirror that by seeding through the same service the CLI uses.
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	h, cleanup, err := svc.OpenForDirectory(".")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	if _, err := h.Store().DB.Exec(`INSERT INTO tool_outputs (session_id, project_id, tool_name, output, size_bytes, created_at)
		VALUES ('s1', ?, 'web_search_exa', 'ctxsearch-sandbox-token text', 30, datetime('now'))`, h.ProjectID()); err != nil {
		t.Fatalf("seed sandbox: %v", err)
	}
	if _, err := h.Store().DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, content, created_at)
		VALUES (?, 'ctxflow.go', 'file', 'go', 'ctxsearch-indexed-token content', datetime('now'))`, h.ProjectID()); err != nil {
		t.Fatalf("seed index: %v", err)
	}
	return dataDir
}

// TestRunCtxStats reports sandbox + index counts.
func TestRunCtxStats(t *testing.T) {
	ctxSeed(t)
	out := runCtxCapture(t, func() { runCtx([]string{"stats"}) })
	if !strings.Contains(out, "sandbox (tool_outputs): 1") {
		t.Fatalf("stats missing sandbox count, got:\n%s", out)
	}
	if !strings.Contains(out, "index (indexed_files): 1") {
		t.Fatalf("stats missing index count, got:\n%s", out)
	}
}

// TestRunCtxSearch fuses both legs and labels each result's source. A query
// hitting only the sandbox leg must return a [sandbox] row; a query hitting only
// the index leg must return an [indexed] row.
func TestRunCtxSearch_SandboxLeg(t *testing.T) {
	ctxSeed(t)
	out := runCtxCapture(t, func() { runCtx([]string{"search", "ctxsearch-sandbox-token"}) })
	if !strings.Contains(out, "[sandbox]") {
		t.Fatalf("search missing [sandbox] label, got:\n%s", out)
	}
	if !strings.Contains(out, "web_search_exa") {
		t.Fatalf("search missing sandbox tool name, got:\n%s", out)
	}
}

func TestRunCtxSearch_IndexedLeg(t *testing.T) {
	ctxSeed(t)
	out := runCtxCapture(t, func() { runCtx([]string{"search", "ctxsearch-indexed-token"}) })
	if !strings.Contains(out, "[indexed]") {
		t.Fatalf("search missing [indexed] label, got:\n%s", out)
	}
	if !strings.Contains(out, "ctxflow.go") {
		t.Fatalf("search missing indexed file name, got:\n%s", out)
	}
}

// TestRunCtxSearch_NoResults confirms a query matching neither leg prints the
// no-results marker rather than failing.
func TestRunCtxSearch_NoResults(t *testing.T) {
	ctxSeed(t)
	out := runCtxCapture(t, func() { runCtx([]string{"search", "no-such-token-zzz"}) })
	if !strings.Contains(out, "no results") {
		t.Fatalf("expected 'no results', got:\n%s", out)
	}
}

// TestRunCtxSearch_Limit confirms -limit caps the fused output. Seeding 5
// identical sandbox rows yields one fused candidate (same tool name), so limit
// still applies; we assert the cap does not over-return.
func TestRunCtxSearch_Limit(t *testing.T) {
	oldSvc := cliService
	t.Cleanup(func() { cliService = oldSvc })
	cliService = nil
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	h, cleanup, err := svc.OpenForDirectory(".")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("file%d.go", i)
		if _, err := h.Store().DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, content, created_at)
			VALUES (?, ?, 'file', 'go', 'limitcap', datetime('now'))`, h.ProjectID(), name); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	out := runCtxCapture(t, func() { runCtx([]string{"search", "limitcap", "-limit", "2"}) })
	// 5 distinct indexed candidates, capped to 2.
	if strings.Count(out, ". [indexed]") < 1 {
		t.Fatalf("search returned no indexed rows, got:\n%s", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Count numbered result lines ("N. [indexed] ...").
	count := 0
	for _, l := range lines {
		if strings.Contains(l, ". [indexed]") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 capped results, got %d:\n%s", count, out)
	}
}

// TestRunCtxDispatch_NoArgs exits 2 with usage. We assert the no-args path
// writes usage to stderr by calling the usage function directly (the os.Exit(2)
// path is exercised by the binary, not the test process).
func TestRunCtxUsage_MentionsSubcommands(t *testing.T) {
	out := runCtxCapture(t, func() { ctxUsage() })
	for _, want := range []string{"search", "prime", "stats"} {
		if !strings.Contains(out, want) {
			t.Fatalf("usage missing %q, got:\n%s", want, out)
		}
	}
}
