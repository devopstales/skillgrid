package main

// parity_test.go — the K6 byte-identical seam (spec 004-transport-mcp §4):
// the CLI and the MCP server are two adapters over ONE handler, so the
// JSON a tool call produces must be identical across the two surfaces.
// The tests drive the same handler through the CLI's runCLI and the MCP
// server's CallTool and compare the unmarshaled JSON (key ordering is an
// envelope concern, the values are the contract).

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/mcp"
)

// jsonEqual reports whether two unmarshaled JSON values are deep-equal
// (the parity assertion: identical fields, identical values — the
// marshaled key ordering is not part of the K6 contract, the content is).
// The random sync_id is stripped before comparison (it's a crypto/rand
// identity, not a deterministic field — the parity is on the shape and
// the deterministic values, not the random id).
func jsonEqual(a, b map[string]any) bool {
	stripRandom(a)
	stripRandom(b)
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ab) == string(bb)
}

// stripRandom removes the random sync_id fields (the crypto/rand
// identity that differs between the CLI and MCP calls for the same
// content). The parity is on the shape + deterministic values.
func stripRandom(m map[string]any) {
	for k, v := range m {
		if k == "sync_id" {
			delete(m, k)
			continue
		}
		if sub, ok := v.(map[string]any); ok {
			stripRandom(sub)
		}
	}
}

// mcpCall runs one MCP tool call in-process (the same envelope path the
// stdio server takes) and returns the envelope JSON.
func mcpCall(t *testing.T, h *handler.Handler, tool string, args map[string]any) []byte {
	t.Helper()
	s := mcp.NewServer(h, mcp.All)
	defer s.Close()
	return s.CallTool(testCtx(), tool, args)
}

// TestCLIMCPParity [cli-mcp-parity]: mem save over the CLI (with --json)
// and mem_save over MCP with the same args must produce identical JSON.
func TestCLIMCPParity(t *testing.T) {
	cliOut := runCLI(t, "mem", "save", "--title", "T", "--type", "decision", "--content", "C",
		"--project", "p", "--session", "s", "--json")
	mcpOut := mcpCall(t, newTestHandler(t), "mem_save", map[string]any{
		"title": "T", "type": "decision", "content": "C", "project": "p", "session_id": "s",
	})
	assertJSONParity(t, cliOut, string(mcpOut))
}

// TestCLIMCPParitySearch [cli-mcp-parity] triangulation: the parity holds
// on the read path too (a search with no hits must render identically
// over both surfaces).
func TestCLIMCPParitySearch(t *testing.T) {
	cliOut := runCLI(t, "mem", "search", "nothinghere", "--project", "p", "--json")
	mcpOut := mcpCall(t, newTestHandler(t), "mem_search", map[string]any{"query": "nothinghere", "project": "p"})
	assertJSONParity(t, cliOut, string(mcpOut))
}

// TestCLIMCPParityError [cli-json-shape] triangulation: a failing call
// renders the error envelope identically over both surfaces (the shape,
// not just the happy path, is the contract). The call is a `mem get` for
// a non-existent id — the handler's not-found error (not a CLI usage
// error, so both surfaces reach the handler).
func TestCLIMCPParityError(t *testing.T) {
	// The CLI's --json error output is on stdout (the envelope); the
	// exit code is 1 (a handler error, not a usage error).
	installRunEnv(t)
	cliOut, _ := captureRun(t, "mem", "get", "--id", "999", "--project", "p", "--json")
	mcpOut := mcpCall(t, newTestHandler(t), "mem_get_observation", map[string]any{"id": 999})
	var a, b map[string]any
	if err := json.Unmarshal([]byte(cliOut), &a); err != nil {
		t.Fatalf("CLI --json output is not a JSON object: %v (%s)", err, cliOut)
	}
	if err := json.Unmarshal([]byte(mcpOut), &b); err != nil {
		t.Fatalf("MCP envelope is not a JSON object: %v (%s)", err, mcpOut)
	}
	if a["error"] != true || b["error"] != true {
		t.Fatalf("a not-found get must be an error envelope on both surfaces:\ncli: %s\nmcp: %s", cliOut, mcpOut)
	}
	if !jsonEqual(a, b) {
		t.Fatalf("error envelopes must match (K6):\ncli: %s\nmcp: %s", cliOut, mcpOut)
	}
}

// assertJSONParity unmarshals both outputs and asserts deep equality
// (the K6 contract: identical fields, identical values).
func assertJSONParity(t *testing.T, cliOut, mcpOut string) {
	t.Helper()
	var a, b map[string]any
	if err := json.Unmarshal([]byte(cliOut), &a); err != nil {
		t.Fatalf("CLI --json output is not a JSON object: %v (%s)", err, cliOut)
	}
	if err := json.Unmarshal([]byte(mcpOut), &b); err != nil {
		t.Fatalf("MCP envelope is not a JSON object: %v (%s)", err, mcpOut)
	}
	if !jsonEqual(a, b) {
		t.Fatalf("CLI and MCP JSON must match (K6):\ncli: %s\nmcp: %s", cliOut, mcpOut)
	}
}

// TestCLIParityJSONShape [cli-json-shape]: the --json output IS the
// envelope (degraded flag + reason when the intelligence seams are
// absent — the save ran on the deterministic floor, the flag says so).
func TestCLIParityJSONShape(t *testing.T) {
	out := runCLI(t, "mem", "save", "--title", "T", "--type", "decision", "--content", "C",
		"--project", "p", "--session", "s", "--json")
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("--json output is not a JSON object: %v (%s)", err, out)
	}
	// The save's sync_id is the identity the agent sees (INV-6) — the
	// envelope carries it, never a bare rowid.
	if sid, _ := env["sync_id"].(string); sid == "" {
		t.Fatalf("save envelope must carry the sync_id: %s", out)
	}
	// The fixture handler has no intelligence seams: the save must be
	// flagged degraded (never a silent floor, INV-3).
	if env["degraded"] != true || env["degraded_reason"] == "" {
		t.Fatalf("absent seams must flag the envelope degraded: %s", out)
	}
}

// TestCLIUnknownSubcommand [cli-json-shape] triangulation: an unknown
// subcommand is a usage error (exit 2, message on stderr) — never a
// half-marshaled envelope.
func TestCLIUnknownSubcommand(t *testing.T) {
	err := runCLIError(t, "mem", "frobnicate")
	if err == nil {
		t.Fatalf("an unknown subcommand must be a usage error")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Fatalf("the usage error must name the unknown subcommand: %v", err)
	}
}

// TestCLIUnknownTopLevelCommand: an unknown top-level command is a usage
// error too (the dispatch is closed, not open — a typo is a loud error,
// not a silent no-op).
func TestCLIUnknownTopLevelCommand(t *testing.T) {
	err := runCLIError(t, "frobnicate")
	if err == nil {
		t.Fatalf("an unknown top-level command must be a usage error")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Fatalf("the usage error must name the unknown command: %v", err)
	}
}

// TestCLIExplicitProjectBeatsEnv [cli-project-selection] triangulation:
// the --project selector wins over MNEMONIC_PROJECT (the spec §4 chain:
// explicit > env > git > basename).
func TestCLIExplicitProjectBeatsEnv(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "envproj")
	out := runCLI(t, "mem", "save", "--title", "T", "--type", "decision", "--content", "C",
		"--project", "explicit", "--session", "s", "--json")
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("--json output is not a JSON object: %v (%s)", err, out)
	}
	if env["error"] == true {
		t.Fatalf("save with an explicit project must not error: %s", out)
	}
}
