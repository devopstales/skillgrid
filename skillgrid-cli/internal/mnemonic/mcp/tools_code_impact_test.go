package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// impactMCPFixture indexes a small python project where `base` has a direct
// dependent `mid` and a duplicate-named `base` in a second file.
func impactMCPFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir = t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	root = abs
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("base_a.go", "package p\n\nfunc base() int {\n\treturn 1\n}\n")
	write("base_b.go", "package p\n\nfunc (x int) base() int {\n\treturn 2\n}\n")
	write("mid.go", "package p\n\nfunc mid() int {\n\treturn base()\n}\n")
	// Pin the project BEFORE indexing so both the index and later queries
	// (which resolve from the cwd of a temp dir nested in the skillgit repo)
	// land in the same stable bucket.
	t.Setenv("MNEMONIC_PROJECT", "codeimpact-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	return dataDir, root
}

func newCallTool(name string, params map[string]any) mcplib.CallToolRequest {
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = params
	return req
}

// TestCodeImpactTool covers @step-03 (code_impact tiers risk and disambiguates
// a multi-symbol target): risk-tiered blast radius with confidence tags, and a
// ranked candidate list for an ambiguous name (never a silent pick).
func TestCodeImpactTool(t *testing.T) {
	dataDir, root := impactMCPFixture(t)
	_ = dataDir
	// Pin the project so the temp-dir fixture (inside the skillgrid git repo)
	// resolves to one stable bucket for both the index and the query.
	t.Setenv("MNEMONIC_PROJECT", "codeimpact-probe")
	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	// Ambiguous target: two symbols named `base`. The tool must return a
	// ranked candidate list, not a silent pick.
	res, err := handleCodeImpact(context.Background(), newCallTool("code_impact", map[string]any{"symbol": "base"}))
	if err != nil {
		t.Fatalf("handleCodeImpact: %v", err)
	}
	text := callResultText(t, res)
	if !strings.Contains(text, "ambiguous") {
		t.Errorf("ambiguous target must be reported, got: %s", text)
	}
	if !strings.Contains(text, "candidates") {
		t.Errorf("ambiguous target must return a ranked candidate list, got: %s", text)
	}

	// Narrowed by file: a single base resolves and returns tiered blast radius.
	res2, err := handleCodeImpact(context.Background(), newCallTool("code_impact", map[string]any{
		"symbol": "base",
		"file":   "base_a.go",
	}))
	if err != nil {
		t.Fatalf("handleCodeImpact: %v", err)
	}
	text2 := callResultText(t, res2)
	if strings.Contains(text2, "ambiguous") {
		t.Errorf("file-narrowed target should not be ambiguous: %s", text2)
	}
	if !strings.Contains(text2, "WILL BREAK") && !strings.Contains(text2, "will_break") {
		t.Errorf("blast radius must be risk-tiered (WILL BREAK), got: %s", text2)
	}
}
