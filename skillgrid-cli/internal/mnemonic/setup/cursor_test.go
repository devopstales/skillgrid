package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestUpsertCursorMCP_RemoteUsesStreamableHTTP(t *testing.T) {
	dir := t.TempDir()
	mcpPath := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(mcpPath, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := MCPServerConfig{
		Name: "context7",
		Type: "remote",
		URL:  "https://mcp.context7.com/mcp",
	}
	if err := upsertCursorMCP(mcpPath, entry, false); err != nil {
		t.Fatalf("upsertCursorMCP: %v", err)
	}

	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	got := gjson.Parse(string(data)).Get("mcpServers.context7")
	if got.Get("type").String() != "streamable-http" {
		t.Fatalf("type = %q, want streamable-http; entry=%s", got.Get("type").String(), got.Raw)
	}
	if got.Get("url").String() != entry.URL {
		t.Fatalf("url = %q, want %q", got.Get("url").String(), entry.URL)
	}
	if got.Get("command").Exists() {
		t.Fatalf("remote entry must not set command, got %s", got.Raw)
	}
}

// TestUpsertCursorHooks pins that setup registers the tool-call capture,
// session, and policy hooks in ~/.cursor/hooks.json, keeps hooks from other
// tools, and does not duplicate its own entries on a second run.
func TestUpsertCursorHooks(t *testing.T) {
	home := t.TempDir()
	hooksPath := filepath.Join(home, ".cursor", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(hooksPath), 0o755); err != nil {
		t.Fatal(err)
	}
	other := `{"version":1,"hooks":{"postToolUse":[{"command":"/opt/other/hook.sh"}],"beforeSubmitPrompt":[{"command":"echo hi"}]}}`
	if err := os.WriteFile(hooksPath, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := upsertCursorHooks(home, false); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := gjson.ParseBytes(data)
	if doc.Get("version").Int() != 1 {
		t.Errorf("version = %s", doc.Get("version").Raw)
	}
	post := doc.Get("hooks.postToolUse").Array()
	if len(post) != 2 || post[0].Get("command").String() != "/opt/other/hook.sh" {
		t.Fatalf("postToolUse = %s, want the other hook first and ours once", doc.Get("hooks.postToolUse").Raw)
	}
	if got, want := post[1].Get("command").String(), CursorHookCommand(home, "cursor-tool-capture.sh"); got != want {
		t.Errorf("capture command = %q, want %q", got, want)
	}
	if n := len(doc.Get("hooks.beforeSubmitPrompt").Array()); n != 1 {
		t.Errorf("beforeSubmitPrompt entries = %d, want 1 (untouched)", n)
	}
	for ev, script := range cursorHookScripts {
		arr := doc.Get("hooks." + ev).Array()
		hits := 0
		for _, e := range arr {
			if strings.HasSuffix(e.Get("command").String(), filepath.Join(".skillgrid", "hooks", script)) {
				hits++
			}
		}
		if hits != 1 {
			t.Errorf("%s: %d skillgrid entries, want 1: %s", ev, hits, doc.Get("hooks."+ev).Raw)
		}
	}

	// Dry-run against a missing file writes nothing.
	dry := t.TempDir()
	if err := upsertCursorHooks(dry, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dry, ".cursor", "hooks.json")); err == nil {
		t.Error("dry-run must not create hooks.json")
	}
}

func TestUpsertCursorMCP_LocalKeepsCommandArgs(t *testing.T) {
	dir := t.TempDir()
	mcpPath := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(mcpPath, []byte(`{"mcpServers":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := MCPServerConfig{
		Name:    "mnemonic",
		Type:    "local",
		Command: []string{"skillgrid", "mcp"},
	}
	if err := upsertCursorMCP(mcpPath, entry, false); err != nil {
		t.Fatalf("upsertCursorMCP: %v", err)
	}

	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	got := gjson.Parse(string(data)).Get("mcpServers.mnemonic")
	if got.Get("command").String() != "skillgrid" {
		t.Fatalf("command = %q, want skillgrid", got.Get("command").String())
	}
	args := got.Get("args").Array()
	if len(args) != 1 || args[0].String() != "mcp" {
		t.Fatalf("args = %v, want [mcp]", got.Get("args").Raw)
	}
	if got.Get("type").Exists() {
		t.Fatalf("local entry must not set type, got %s", got.Raw)
	}
}

func TestInstallCursorHookScripts(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"cursor-tool-capture.sh", "tool-call-capture.js"} {
		if err := os.WriteFile(filepath.Join(repo, "hooks", n), []byte("#"+n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := installCursorHookScripts(home, repo, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "hooks")); !os.IsNotExist(err) {
		t.Fatalf("dry-run must not write: %v", err)
	}
	if err := installCursorHookScripts(home, repo, false); err != nil {
		t.Fatal(err)
	}
	sh, err := os.Stat(filepath.Join(home, ".skillgrid", "hooks", "cursor-tool-capture.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if sh.Mode().Perm()&0o100 == 0 {
		t.Errorf("shell hook not executable: %v", sh.Mode())
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "hooks", "tool-call-capture.js")); err != nil {
		t.Errorf("capture js not copied: %v", err)
	}
	// Scripts missing from the checkout are skipped, not fatal.
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "hooks", "cursor-policy.sh")); !os.IsNotExist(err) {
		t.Errorf("missing source should be skipped: %v", err)
	}
}
