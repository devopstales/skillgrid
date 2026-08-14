package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	engramBin := filepath.Join(p.DepsBinDir, "engram")
	if err := os.WriteFile(engramBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
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
	if en["command"] != engramBin {
		t.Fatalf("command=%v", en["command"])
	}
	if _, ok := en["requires"]; ok {
		t.Fatal("requires must be stripped")
	}
}

// The shipped pack and the install phase have to agree: every `requires` key
// must be one the phase can actually set, and every placeholder must be known to
// the resolver, or entries silently vanish from every agent config.
func TestShippedMCPPackAgreesWithInstallPhase(t *testing.T) {
	packPath := filepath.Join("..", "..", "packs", "mcp", "servers.json")
	data, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatal(err)
	}
	var pack struct {
		MCPServers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &pack); err != nil {
		t.Fatal(err)
	}
	if len(pack.MCPServers) == 0 {
		t.Fatal("shipped pack has no servers")
	}

	presenceKeys := map[string]bool{"http:deepwiki": true, "http:exa": true}
	for _, bin := range releaseBinaries {
		presenceKeys["binary:"+bin.binary] = true
	}
	for _, pkg := range managedNpmPackages {
		presenceKeys["npm:"+pkg.pkg] = true
	}

	p := home.Resolve(t.TempDir())
	if err := home.EnsureLayout(p); err != nil {
		t.Fatal(err)
	}
	for _, bin := range releaseBinaries {
		if err := EnsureFileExecutable(filepath.Join(p.DepsBinDir, bin.binary), []byte("#!/bin/sh\n")); err != nil {
			t.Fatal(err)
		}
	}
	for _, pkg := range managedNpmPackages {
		if err := EnsureFileExecutable(ManagedBin(p, pkg.bins[0]), []byte("#!/bin/sh\n")); err != nil {
			t.Fatal(err)
		}
	}

	for name, entry := range pack.MCPServers {
		req, ok := entry["requires"].(string)
		if !ok {
			t.Fatalf("%s has no requires key", name)
		}
		if !presenceKeys[req] {
			t.Fatalf("%s requires %q which the install phase never sets", name, req)
		}
	}

	servers, warns, err := ResolveMCPServers(packPath, p, presenceKeys)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("expected every entry to resolve, warnings=%v", warns)
	}
	if len(servers) != len(pack.MCPServers) {
		t.Fatalf("resolved %d of %d entries", len(servers), len(pack.MCPServers))
	}
	resolved, err := json.Marshal(servers)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(resolved), "{{") {
		t.Fatalf("unresolved placeholder in %s", resolved)
	}
}

// An entry can pass its `requires` check via an unpacked package directory yet
// still resolve to an executable that was never linked. Writing it out would
// leave the agent with a server that cannot start.
func TestResolveMCPServersDropsMissingAbsoluteCommand(t *testing.T) {
	dir := t.TempDir()
	pack := filepath.Join(dir, "servers.json")
	content := `{
	  "mcpServers": {
	    "aiskillgrid-context7": {
	      "command": "{{AISKILLGRID_CONTEXT7}}",
	      "requires": "npm:@upstash/context7-mcp"
	    },
	    "aiskillgrid-deepwiki": {
	      "url": "https://mcp.deepwiki.com/mcp",
	      "requires": "http:deepwiki"
	    }
	  }
	}`
	if err := os.WriteFile(pack, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p := home.Resolve(t.TempDir())
	_ = home.EnsureLayout(p)

	present := map[string]bool{"npm:@upstash/context7-mcp": true, "http:deepwiki": true}
	servers, warns, err := ResolveMCPServers(pack, p, present)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["aiskillgrid-context7"]; ok {
		t.Fatalf("expected context7 dropped, servers=%v", servers)
	}
	if _, ok := servers["aiskillgrid-deepwiki"]; !ok {
		t.Fatal("url-only entry must survive the command guard")
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for missing command")
	}

	// Once the shim exists the same entry resolves to an absolute managed bin.
	bin := ManagedBin(p, "context7-mcp")
	if err := EnsureFileExecutable(bin, []byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	servers, _, err = ResolveMCPServers(pack, p, present)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := servers["aiskillgrid-context7"].(map[string]any)
	if !ok {
		t.Fatalf("expected context7 present, servers=%v", servers)
	}
	if entry["command"] != bin {
		t.Fatalf("command=%v want %q", entry["command"], bin)
	}
}
