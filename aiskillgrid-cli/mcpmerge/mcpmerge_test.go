package mcpmerge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripJSONC(t *testing.T) {
	in := []byte(`{
  // comment
  "mcp": {
    "a": 1, /* block */
  },
}`)
	obj, err := ParseObject(in)
	if err != nil {
		t.Fatal(err)
	}
	mcp, ok := obj["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("mcp missing: %#v", obj)
	}
	if mcp["a"].(float64) != 1 {
		t.Fatalf("a=%v", mcp["a"])
	}
}

func TestMergeAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"other":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := LoadOrEmpty(path)
	if err != nil {
		t.Fatal(err)
	}
	incoming := map[string]any{
		"aiskillgrid-demo": map[string]any{"command": "echo"},
	}
	root = MergeMCPServers(root, "mcpServers", incoming, "aiskillgrid-")
	if err := WriteObject(path, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("expected backup", err)
	}
	again, err := LoadOrEmpty(path)
	if err != nil {
		t.Fatal(err)
	}
	servers := again["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatal("lost other server")
	}
	if _, ok := servers["aiskillgrid-demo"]; !ok {
		t.Fatal("missing skillgrid server")
	}
}
