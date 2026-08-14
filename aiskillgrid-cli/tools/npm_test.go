package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func TestInstallNPMPackagesInvokesNpmWithPrefix(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "args.txt")
	fake := filepath.Join(dir, "npm")
	script := "#!/bin/sh\necho \"$@\" > " + record + "\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	node := filepath.Join(dir, "node")
	_ = os.WriteFile(node, []byte("#!/bin/sh\n"), 0o755)

	p := home.Resolve(t.TempDir())
	_ = home.EnsureLayout(p)
	if err := InstallNPMPackages(p, []string{"gitnexus", "@upstash/context7-mcp"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(record)
	got := string(b)
	if !strings.Contains(got, " -g ") {
		t.Fatalf("expected global-style install so bins land in npm/bin, args=%q", got)
	}
	if !strings.Contains(got, "--prefix") || !strings.Contains(got, p.NpmDir) {
		t.Fatalf("args=%q", got)
	}
	if !strings.Contains(got, "--cache") || !strings.Contains(got, p.NpmCacheDir) {
		t.Fatalf("args=%q", got)
	}
	if !strings.Contains(got, "gitnexus") || !strings.Contains(got, "@upstash/context7-mcp") {
		t.Fatalf("args=%q", got)
	}
}

// A global-style install puts executables in <prefix>/bin and unpacks packages
// under <prefix>/lib/node_modules, so both lookups must agree with ManagedBin.
func TestManagedBinAndPackageLookupUseGlobalLayout(t *testing.T) {
	p := home.Resolve(t.TempDir())
	if err := home.EnsureLayout(p); err != nil {
		t.Fatal(err)
	}

	if got := ResolveManagedBin(p, "gitnexus"); got != "" {
		t.Fatalf("expected no bin before install, got %q", got)
	}
	if NpmPackageInstalled(p, "gitnexus", "gitnexus") {
		t.Fatal("expected gitnexus absent before install")
	}
	if got := ManagedBinOrDefault(p, "gitnexus"); got != ManagedBin(p, "gitnexus") {
		t.Fatalf("fallback=%q want %q", got, ManagedBin(p, "gitnexus"))
	}

	// Fake `npm install -g --prefix <NpmDir>` layout.
	if err := EnsureFileExecutable(ManagedBin(p, "gitnexus"), []byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	scoped := filepath.Join(NpmModulesDirs(p)[0], "@upstash", "context7-mcp")
	if err := os.MkdirAll(scoped, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scoped, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := ResolveManagedBin(p, "gitnexus"); got != ManagedBin(p, "gitnexus") {
		t.Fatalf("resolved=%q want %q", got, ManagedBin(p, "gitnexus"))
	}
	if !NpmPackageInstalled(p, "gitnexus", "gitnexus") {
		t.Fatal("expected gitnexus present via bin shim")
	}
	if !NpmPackageInstalled(p, "@upstash/context7-mcp", "context7-mcp") {
		t.Fatal("expected scoped package present via lib/node_modules")
	}
}

// Newer @playwright/mcp publishes playwright-mcp; older releases shipped
// mcp-server-playwright. Either shim must satisfy the lookup.
func TestResolveManagedBinAcceptsAlternateBinName(t *testing.T) {
	p := home.Resolve(t.TempDir())
	if err := home.EnsureLayout(p); err != nil {
		t.Fatal(err)
	}
	legacy := ManagedBin(p, "mcp-server-playwright")
	if err := EnsureFileExecutable(legacy, []byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if got := ResolveManagedBin(p, "playwright-mcp", "mcp-server-playwright"); got != legacy {
		t.Fatalf("resolved=%q want %q", got, legacy)
	}
}
