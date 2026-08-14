package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func TestStatusLines(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	if err := home.EnsureLayout(p); err != nil {
		t.Fatal(err)
	}

	lines := StatusLines(p)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], p.NpmDir) {
		t.Fatalf("expected npm dir in line 0: %s", lines[0])
	}
	if !strings.Contains(lines[1], "engram=no") || !strings.Contains(lines[1], "skills=no") {
		t.Fatalf("expected missing binaries: %s", lines[1])
	}

	_ = EnsureFileExecutable(filepath.Join(p.DepsBinDir, "engram"), []byte("x"))
	_ = EnsureFileExecutable(filepath.Join(p.DepsBinDir, "skills"), []byte("x"))
	_ = EnsureFileExecutable(ManagedBin(p, "gitnexus"), []byte("x"))

	lines = StatusLines(p)
	if !strings.Contains(lines[1], "engram=yes") || !strings.Contains(lines[1], "skills=yes") {
		t.Fatalf("expected present binaries: %s", lines[1])
	}
	if !strings.Contains(lines[2], "gitnexus=yes") {
		t.Fatalf("expected gitnexus=yes: %s", lines[2])
	}
}
