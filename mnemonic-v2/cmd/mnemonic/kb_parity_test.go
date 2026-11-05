// kb_parity_test.go — the K6 byte-identical seam test (008-transport §4 /
// K6): the CLI's --json envelope bytes and the MCP server's CallTool
// envelope bytes are identical for the same kb_* read tool over the same
// store. A divergence here is a structural CLI/MCP drift — the whole
// reason the CLI is a thin wrapper over the shared KbHandler.

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
	mcp "github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/mcp"
)

// TestKbJSONParity asserts the envelope byte-equality (kb_status +
// kb_query over the same store): the CLI's kbJSON path and the MCP's
// wrapEnvelope path must emit identical bytes.
func TestKbJSONParity(t *testing.T) {
	oldCwd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldCwd) })
	// Hermetic cwd: the sentinel vault lives under the invocation's cwd.
	os.Chdir(t.TempDir())

	dataDir := t.TempDir()
	prevStoreFor, prevHandlerFor, prevGit := storeFor, handlerFor, gitToplevel
	storeFor = func(cfg config.Config, p string) (store.Store, error) {
		return store.Open(context.Background(), dataDir, p, store.SQLite, "")
	}
	handlerFor = func(cfg config.Config, s store.Store) (*handler.Handler, error) {
		return newTestHandlerWithStore(t, s), nil
	}
	gitToplevel = func(string) (string, error) { return "", errors.New("no git in tests") }
	t.Cleanup(func() { storeFor, handlerFor, gitToplevel = prevStoreFor, prevHandlerFor, prevGit })

	// The MCP side: a server over the same store file (the in-process K6
	// path — the same wiring runMCP uses).
	st, err := store.Open(context.Background(), dataDir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	h, err := handlerFor(config.Default(), st)
	if err != nil {
		t.Fatalf("handlerFor: %v", err)
	}
	kh := handler.NewKbHandler(st, config.Config{}, nil)
	cwd, _ := os.Getwd()
	kh.SetRoots(filepath.Join(cwd, ".skillgrid", "knowledge"), cwd)
	srv := mcp.NewServer(h, mcp.All)
	srv.RegisterKbTools(kh)
	defer srv.Close()

	cases := []struct {
		cli  []string
		name string
		args map[string]any
	}{
		{cli: []string{"kb", "status", "--project", "p", "--json"}, name: "kb_status", args: map[string]any{"project": "p"}},
		{cli: []string{"kb", "query", "auth", "--project", "p", "--json"}, name: "kb_query", args: map[string]any{"query": "auth", "project": "p"}},
	}
	for _, c := range cases {
		out, code := captureRun(t, c.cli...)
		if code != exitOK {
			t.Fatalf("%s CLI exited %d: %s", c.name, code, out)
		}
		// kbJSON prints one line (the CLI's trailing newline is the
		// terminal's, not the envelope's — the MCP bytes are the contract).
		cliBytes := strings.TrimSuffix(out, "\n")
		mcpBytes := string(srv.CallTool(context.Background(), c.name, c.args))
		if cliBytes != mcpBytes {
			t.Fatalf("%s parity: CLI and MCP envelopes diverge.\n  CLI: %s\n  MCP: %s", c.name, cliBytes, mcpBytes)
		}
	}
}
