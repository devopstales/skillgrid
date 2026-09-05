package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestMigrateTierCLIBackfill(t *testing.T) {
	dataDir := t.TempDir()
	content := filepath.Join(dataDir, "content")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	l2 := filepath.Join(content, "cli.md")
	body := "CLI full detail must not change.\n"
	if err := os.WriteFile(l2, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := exec.Command("go", "run", ".", "migrate", "--tier",
		"--dir", dataDir,
		"--project", "cliproj",
		"--root", content,
	)
	cmd.Dir = filepath.Join("..", "..", "cmd", "skillgrid")
	// Resolve from this test's package dir: cmd/skillgrid is this package when testing ./cmd/skillgrid
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// When running `go test ./cmd/skillgrid`, wd is the package dir.
	cmd.Dir = wd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("migrate cli: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "processed") {
		t.Fatalf("unexpected output: %s", out)
	}
	after, err := os.ReadFile(l2)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != body {
		t.Fatal("L2 changed")
	}
	if _, err := os.Stat(l2 + ".abstract"); err != nil {
		t.Fatalf("abstract: %v", err)
	}
	st, err := store.Open(dataDir, "cliproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM tiered_contents`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rows=%d", n)
	}
}
