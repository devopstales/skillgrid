package main

// index_test.go — the index command group's CLI tests (spec 008 §4): the
// K5/K6 seam (the CLI's --json output is the envelope marshal of the same
// CodeHandler return the MCP tools use — byte-identical), the subcommand
// dispatch, and the usage errors.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/search"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/envelope"
)

// ctxForTest is the test's context (the CLI's in-process run uses its own;
// the direct CodeHandler calls in the parity test use this).
func ctxForTest() context.Context { return context.Background() }

// testConfig is the test's config (the store seam's input — the fixture's
// temp data dir is set by installRunEnv; the empty config's defaults are the
// floor).
func testConfig(t *testing.T) config.Config {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return c
}

// TestIndexStatusJSONIsEnvelope verifies the K5 seam: the CLI's `index
// status --json` output is the envelope (the same marshal the MCP tools use).
func TestIndexStatusJSONIsEnvelope(t *testing.T) {
	installRunEnv(t)
	out, code := captureRun(t, "index", "status", "--project", "p", "--json")
	if code != exitOK {
		t.Fatalf("index status --project p --json exited %d, want 0 (output: %s)", code, out)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("index status --json is not a JSON object: %v (%s)", err, out)
	}
	// The 4-state freshness (the T-CI-4 fix — a string, not a boolean).
	if _, ok := env["freshness"]; !ok {
		t.Fatalf("index status --json must carry the freshness (4-state value), got: %v", env)
	}
}

// TestIndexSearchParity verifies the K5/K6 seam: the same search query via
// the CLI (`index search --json`) and via the CodeHandler (the MCP tool's
// backing) produces byte-identical JSON (the envelope marshal of the same
// CodeHandler return).
func TestIndexSearchParity(t *testing.T) {
	installRunEnv(t)
	// Write a fixture Go file the index can extract.
	cwd := t.TempDir()
	fixture := filepath.Join(cwd, "main.go")
	if err := os.WriteFile(fixture, []byte("package main\n\nfunc greet() string { return \"hello\" }\n\nfunc main() { _ = greet() }\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// Index the fixture (the CLI's codeRuntime opens the store for "p"; the
	// index walk's root is the cwd — but the test fixture's store is a fresh
	// temp dir, so index it directly through the store seam to populate the
	// store the CLI reads).
	st, err := storeFor(testConfig(t), "p")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	if _, err := codeindex.Index(ctxForTest(), cwd, st, codeindex.IndexOpts{Project: "p"}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	// The CLI's --json output (the envelope marshal of the CodeHandler result).
	out, code := captureRun(t, "index", "search", "greet", "--project", "p", "--json")
	if code != exitOK {
		t.Fatalf("index search greet --project p --json exited %d, want 0 (output: %s)", code, out)
	}
	var cliEnv map[string]any
	if err := json.Unmarshal([]byte(out), &cliEnv); err != nil {
		t.Fatalf("index search --json is not a JSON object: %v (%s)", err, out)
	}
	// The MCP tool's backing: the same CodeHandler.Search result, envelope-
	// marshaled (the K5 seam — the MCP server's wrapEnvelope of the same
	// result must be byte-identical).
	ch := newTestCodeHandlerForStore(t, st)
	ch.SetRoot(cwd)
	cliResult, err := ch.Search(ctxForTest(), "greet", search.SearchQuery{Project: "p", Limit: 10})
	if err != nil {
		t.Fatalf("CodeHandler.Search: %v", err)
	}
	mcpEnv, err := envelope.WithDegraded(cliResult, false, "")
	if err != nil {
		t.Fatalf("MCP envelope: %v", err)
	}
	mcpBytes, _ := json.Marshal(mcpEnv)
	cliBytes, _ := json.Marshal(cliEnv)
	if string(cliBytes) != string(mcpBytes) {
		t.Fatalf("K5 parity broken:\nCLI: %s\nMCP: %s", cliBytes, mcpBytes)
	}
}

// TestIndexSearchRequiresQuery verifies the usage error: `index search`
// without a query exits 2 (a usage error, not a handler error).
func TestIndexSearchRequiresQuery(t *testing.T) {
	installRunEnv(t)
	_, code := captureRun(t, "index", "search")
	if code != exitUsage {
		t.Fatalf("index search without a query must exit %d (usage error); got %d", exitUsage, code)
	}
}

// TestIndexUnknownSubcommand verifies the usage error: an unknown index
// subcommand exits 2.
func TestIndexUnknownSubcommand(t *testing.T) {
	installRunEnv(t)
	_, code := captureRun(t, "index", "frobnicate")
	if code != exitUsage {
		t.Fatalf("index frobnicate must exit %d (usage error); got %d", exitUsage, code)
	}
}
