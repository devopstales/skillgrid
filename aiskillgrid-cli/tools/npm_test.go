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
	if err := InstallNPMPackages(p, []string{"gitnexus", "backlog.md"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(record)
	got := string(b)
	if !strings.Contains(got, "--prefix") || !strings.Contains(got, p.NpmDir) {
		t.Fatalf("args=%q", got)
	}
	if !strings.Contains(got, "--cache") || !strings.Contains(got, p.NpmCacheDir) {
		t.Fatalf("args=%q", got)
	}
	if !strings.Contains(got, "gitnexus") || !strings.Contains(got, "backlog.md") {
		t.Fatalf("args=%q", got)
	}
}
