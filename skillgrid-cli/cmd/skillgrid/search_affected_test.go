package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// affectedCLIFixture writes a small go project where base.go is imported by
// mid.go and base has a test file, indexes it through the shared service, and
// returns root.
func affectedCLIFixture(t *testing.T, svc *service.Service) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("src/base.go", "package p\n\nfunc base() int {\n\treturn 1\n}\n")
	write("src/mid.go", "package p\n\nfunc mid() int {\n\treturn base()\n}\n")
	write("src/base_test.go", "package p\n\nfunc TestBase() {\n\t_ = base()\n}\n")
	t.Setenv("MNEMONIC_PROJECT", "cli-affected")
	if _, err := svc.ResolveProject(root); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	return root
}

// runSearchAffectedCapture runs runSearchAffected with stdin pointed at
// inPath and returns its stdout.
func runSearchAffectedCapture(t *testing.T, inPath string, args []string) string {
	t.Helper()
	in, err := os.Open(inPath)
	if err != nil {
		t.Fatalf("open stdin file: %v", err)
	}
	defer in.Close()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = in
	os.Stdout = w
	runSearchAffected("test", args)
	_ = w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)
	return string(out)
}

// TestCodeAffectedStdinCLI covers @step-02 (Scenario: code_affected consumes a
// diff file list from stdin, CLI side): `skillgrid search affected --stdin`
// reads a git-diff --name-only-style list from stdin and prints the affected
// test files; an empty stdin prints a clear empty message (not an error).
func TestCodeAffectedStdinCLI(t *testing.T) {
	dataDir := t.TempDir()
	svc := service.New(dataDir)
	root := affectedCLIFixture(t, svc)

	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	t.Setenv("MNEMONIC_PROJECT", "cli-affected")
	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	stdinFile := filepath.Join(t.TempDir(), "diff.txt")
	if err := os.WriteFile(stdinFile, []byte("src/base.go\nsrc/mid.go\n"), 0o644); err != nil {
		t.Fatalf("write stdin file: %v", err)
	}
	out := runSearchAffectedCapture(t, stdinFile, []string{"search", "affected", "--stdin"})
	if !strings.Contains(out, "src/base_test.go") {
		t.Fatalf("CLI affected --stdin must print the affected test file, got: %s", out)
	}

	// Empty stdin: clear message, no error.
	emptyFile := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(emptyFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	out2 := runSearchAffectedCapture(t, emptyFile, []string{"search", "affected", "--stdin"})
	if !strings.Contains(out2, "no changed") {
		t.Fatalf("CLI affected --stdin on an empty diff must print a clear message, got: %s", out2)
	}
}
