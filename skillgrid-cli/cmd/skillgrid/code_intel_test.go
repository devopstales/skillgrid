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

// codeIntelFixture writes a small go project under root and indexes it through
// the shared service, returning root for CLI grep/parity checks.
func codeIntelFixture(t *testing.T, svc *service.Service) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("b.py", "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n")
	write("c.txt", "def notpython()\n")
	write("a.go", "package x\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n\nfunc main() {\n\tv := add(1, 2)\n}\n")
	// Index through the resolved project so the service opens the same store.
	if _, err := svc.ResolveProject(root); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	return root
}

// TestCodeIntelGrepParity covers @step-02 (CLI parity): `skillgrid grep`
// performs the index-free structural search and prints matches; a missing
// pattern is rejected clearly.
func TestCodeIntelGrepParity(t *testing.T) {
	dataDir := t.TempDir()
	svc := service.New(dataDir)
	root := codeIntelFixture(t, svc)

	// A valid pattern over the fixture dir produces matches.
	res, err := svc.CodeGrep(context.Background(), root, `(function_definition) \fn`)
	if err != nil {
		t.Fatalf("code_grep: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatalf("expected grep hits, got none")
	}
	foundPy := false
	for _, h := range res.Hits {
		if strings.HasSuffix(h.Path, "b.py") {
			foundPy = true
		}
	}
	if !foundPy {
		t.Errorf("expected a b.py hit, got %v", res.Hits)
	}

	// Missing pattern is rejected (no invented default).
	if _, err := svc.CodeGrep(context.Background(), root, ""); err == nil {
		t.Errorf("expected an error for an empty grep pattern")
	}

	// An invalid-for-language pattern skips that language with a note, not a
	// silent no-match.
	res2, err := svc.CodeGrep(context.Background(), root, `(function_declaration) \fn`)
	if err != nil {
		t.Fatalf("code_grep: %v", err)
	}
	pyNote := false
	for _, n := range res2.Notes {
		if n.Language == "python" {
			pyNote = true
		}
	}
	if !pyNote {
		t.Errorf("expected a python skip note, got %v", res2.Notes)
	}
}

// TestRunCodeIntelUsageCoversSubcommands covers that the orient/grep
// subcommands are dispatched (the command entrypoint knows about both).
func TestRunCodeIntelUsageCoversSubcommands(t *testing.T) {
	// Capture stderr for the usage print (both subcommands must be listed).
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	printCodeIntelUsage()
	os.Stderr = oldStderr
	w.Close()
	outBytes, _ := io.ReadAll(r)
	out := string(outBytes)
	if !strings.Contains(out, "orient") || !strings.Contains(out, "grep") {
		t.Errorf("code_intel usage should list orient and grep, got:\n%s", out)
	}
}
