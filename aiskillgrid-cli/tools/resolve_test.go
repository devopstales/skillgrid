package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func TestResolveMCPServersSubstitutesAndFilters(t *testing.T) {
	dir := t.TempDir()
	pack := filepath.Join(dir, "servers.json")
	content := `{
	  "mcpServers": {
	    "aiskillgrid-engram": {
	      "command": "{{AISKILLGRID_ENGRAM}}",
	      "args": ["mcp"],
	      "requires": "binary:engram"
	    },
	    "aiskillgrid-gitnexus": {
	      "command": "{{AISKILLGRID_GITNEXUS}}",
	      "args": ["mcp"],
	      "requires": "npm:gitnexus"
	    },
	    "aiskillgrid-deepwiki": {
	      "url": "https://mcp.deepwiki.com/mcp",
	      "requires": "http:deepwiki"
	    }
	  }
	}`
	_ = os.WriteFile(pack, []byte(content), 0o644)
	p := home.Resolve(t.TempDir())
	_ = home.EnsureLayout(p)
	// Pretend engram + deepwiki present, gitnexus missing
	present := map[string]bool{"binary:engram": true, "http:deepwiki": true}
	servers, warns, err := ResolveMCPServers(pack, p, present)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["aiskillgrid-gitnexus"]; ok {
		t.Fatal("expected gitnexus skipped")
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for missing gitnexus")
	}
	en := servers["aiskillgrid-engram"].(map[string]any)
	if en["command"] != filepath.Join(p.DepsBinDir, "engram") {
		t.Fatalf("command=%v", en["command"])
	}
	if _, ok := en["requires"]; ok {
		t.Fatal("requires must be stripped")
	}
}
