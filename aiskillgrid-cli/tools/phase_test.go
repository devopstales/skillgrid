package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

	failGet := func(url string) ([]byte, error) {
		return nil, errors.New("network unreachable")
	}
	failResolver := func(repo, goos, goarch string) (string, error) {
		return "", errors.New("github unreachable")
	}
	// The default suite stays offline, so npm never runs for real here.
	failInstall := func(home.Paths, []string) error {
		return errors.New("registry unreachable")
	}

	res, err := RunInstallPhase(p, packRoot, PhaseOptions{
		Downloader:           failGet,
		ReleaseAssetResolver: failResolver,
		NpmInstaller:         failInstall,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should succeed with warnings
	if len(res.Warnings) == 0 {
		t.Fatal("expected warnings for failed installs")
	}
	joined := strings.Join(res.Warnings, "\n")
	for _, want := range []string{"engram", "skills"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected warning naming %s, got: %v", want, res.Warnings)
		}
	}
	if strings.Contains(joined, "<nil>") {
		t.Fatalf("warnings must not print a nil error: %v", res.Warnings)
	}

	// Failed installs must not leave anything marked present.
	for _, key := range []string{"binary:engram", "binary:skills", "npm:gitnexus"} {
		if res.Present[key] {
			t.Fatalf("%s should be absent after failed install", key)
		}
	}

	// deepwiki + exa should always be present
	if !res.Present["http:deepwiki"] {
		t.Fatal("http:deepwiki should always be present")
	}
	if !res.Present["http:exa"] {
		t.Fatal("http:exa should always be present")
	}
}

// The second run of a phase must not re-download binaries that are already
// installed, and must not touch the network to discover that.
func TestRunInstallPhaseSkipsAlreadyInstalledBinaries(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{
	  "mcpServers": {
	    "aiskillgrid-engram": {"command":"{{AISKILLGRID_ENGRAM}}","args":["mcp"],"requires":"binary:engram"}
	  }
	}`), 0o644)

	downloads := 0
	get := func(url string) ([]byte, error) {
		downloads++
		return []byte("#!/bin/sh\necho ok\n"), nil
	}
	resolves := 0
	resolver := func(repo, goos, goarch string) (string, error) {
		resolves++
		return "https://example.invalid/" + repo, nil
	}
	noopInstall := func(home.Paths, []string) error { return nil }

	opts := PhaseOptions{Downloader: get, ReleaseAssetResolver: resolver, NpmInstaller: noopInstall}
	if _, err := RunInstallPhase(p, packRoot, opts); err != nil {
		t.Fatal(err)
	}
	if downloads != len(releaseBinaries) {
		t.Fatalf("first run downloads=%d want %d", downloads, len(releaseBinaries))
	}
	for _, bin := range releaseBinaries {
		if !BinaryInstalled(filepath.Join(p.DepsBinDir, bin.binary)) {
			t.Fatalf("%s not installed executable", bin.binary)
		}
	}

	if _, err := RunInstallPhase(p, packRoot, opts); err != nil {
		t.Fatal(err)
	}
	if downloads != len(releaseBinaries) {
		t.Fatalf("second run re-downloaded: downloads=%d", downloads)
	}
	if resolves != len(releaseBinaries) {
		t.Fatalf("second run hit the release API again: resolves=%d", resolves)
	}
}

// Release assets are usually archives, so the phase must unpack the wanted
// member rather than writing the tarball as the executable.
func TestRunInstallPhaseExtractsArchivedBinary(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{"mcpServers":{}}`), 0o644)

	want := []byte("#!/bin/sh\necho engram\n")
	get := func(url string) ([]byte, error) { return tarGz(t, "engram", want), nil }
	resolver := func(repo, goos, goarch string) (string, error) {
		return "https://example.invalid/" + repo, nil
	}
	noopInstall := func(home.Paths, []string) error { return nil }

	if _, err := RunInstallPhase(p, packRoot, PhaseOptions{
		Downloader:           get,
		ReleaseAssetResolver: resolver,
		NpmInstaller:         noopInstall,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(p.DepsBinDir, "engram"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("engram content=%q want %q", got, want)
	}
}

func tarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
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
	if !res.Present["http:exa"] {
		t.Fatal("http:exa should always be present")
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
