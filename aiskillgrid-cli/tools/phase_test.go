package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func TestRunInstallPhaseWithFakes(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{
	  "mcpServers": {
	    "aiskillgrid-engram": {"command":"{{AISKILLGRID_ENGRAM}}","args":["mcp"],"requires":"binary:engram"},
	    "aiskillgrid-deepwiki": {"url":"https://mcp.deepwiki.com/mcp","requires":"http:deepwiki"}
	  }
	}`), 0o644)

	// Pre-seed engram binary so SkipNetwork path marks present
	_ = os.WriteFile(filepath.Join(p.DepsBinDir, "engram"), []byte("x"), 0o755)

	res, err := RunInstallPhase(p, packRoot, PhaseOptions{SkipNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Servers["aiskillgrid-engram"]; !ok {
		t.Fatalf("servers=%v", res.Servers)
	}
	if _, ok := res.Servers["aiskillgrid-deepwiki"]; !ok {
		t.Fatal("deepwiki missing")
	}
}

func TestRunInstallPhaseNetworkFailuresWarn(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{
	  "mcpServers": {
	    "aiskillgrid-deepwiki": {"url":"https://mcp.deepwiki.com/mcp","requires":"http:deepwiki"}
	  }
	}`), 0o644)

	// Fake downloader that fails
	failGet := func(url string) ([]byte, error) {
		return nil, nil // will trigger best-effort warning
	}

	// Fake resolver that fails
	failResolver := func(repo, goos, goarch string) (string, error) {
		return "", nil // will trigger best-effort warning
	}

	res, err := RunInstallPhase(p, packRoot, PhaseOptions{
		Downloader:           failGet,
		ReleaseAssetResolver: failResolver,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should succeed with warnings
	if len(res.Warnings) == 0 {
		t.Fatal("expected warnings for failed installs")
	}

	// deepwiki should always be present
	if !res.Present["http:deepwiki"] {
		t.Fatal("http:deepwiki should always be present")
	}
}

func TestRunInstallPhaseBuildsPresenceMap(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{
	  "mcpServers": {}
	}`), 0o644)

	// Pre-seed various binaries
	_ = os.WriteFile(filepath.Join(p.DepsBinDir, "engram"), []byte("x"), 0o755)
	_ = os.WriteFile(filepath.Join(p.DepsBinDir, "skills"), []byte("x"), 0o755)
	_ = os.WriteFile(filepath.Join(p.NpmBinDir, "gitnexus"), []byte("x"), 0o755)

	res, err := RunInstallPhase(p, packRoot, PhaseOptions{SkipNetwork: true})
	if err != nil {
		t.Fatal(err)
	}

	if !res.Present["binary:engram"] {
		t.Fatal("engram should be present")
	}
	if !res.Present["binary:skills"] {
		t.Fatal("skills should be present")
	}
	if !res.Present["npm:gitnexus"] {
		t.Fatal("gitnexus should be present")
	}
	if !res.Present["http:deepwiki"] {
		t.Fatal("http:deepwiki should always be present")
	}
}

func TestRunInstallPhasePlaywrightWarning(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{
	  "mcpServers": {}
	}`), 0o644)

	// Pre-seed playwright package in node_modules
	playwrightPkgDir := filepath.Join(p.NpmDir, "node_modules", "@playwright", "mcp")
	_ = os.MkdirAll(playwrightPkgDir, 0o755)
	_ = os.WriteFile(filepath.Join(playwrightPkgDir, "package.json"), []byte(`{}`), 0o644)

	res, err := RunInstallPhase(p, packRoot, PhaseOptions{SkipNetwork: true})
	if err != nil {
		t.Fatal(err)
	}

	// Should have playwright warning
	hasPlaywrightWarning := false
	for _, w := range res.Warnings {
		if w == "playwright browsers may need install later" {
			hasPlaywrightWarning = true
			break
		}
	}
	if !hasPlaywrightWarning {
		t.Fatalf("expected playwright warning, got: %v", res.Warnings)
	}
}
