package report

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestCollectDebt walks a project root and records each TODO/FIXME/XXX marker
// as a debt finding (rule_id debt:<marker>, file + line, severity LOW).
//
// SATISFIES: happy path debt-markers-report
func TestCollectDebt(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("a.go", "package a\n// TODO: fix later\n")
	write("b.go", "package b\n// FIXME: broken\n// XXX: hack\n")
	// Skipped dirs must not contribute findings.
	write("node_modules/m.go", "// TODO: node_modules, not a source file\n")
	write("vendor/v.go", "// XXX: vendored, skipped\n")
	write(".git/g.go", "// FIXME: inside .git, skipped\n")

	st := newTestStore(t)
	svc := New(st, root, t.TempDir(), Config{})
	res := svc.collectDebt(ctx)

	if res.Tool != "debt" {
		t.Fatalf("tool = %q, want debt", res.Tool)
	}
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if len(res.Findings) != 3 {
		t.Fatalf("findings = %d, want 3 (skipped dirs excluded): %+v", len(res.Findings), res.Findings)
	}
	rules := map[string]fileLineRef{}
	for _, f := range res.Findings {
		rules[f.RuleID] = fileLineRef{file: f.File, line: f.Line}
	}
	want := map[string]fileLineRef{
		"debt:TODO":  {file: "a.go", line: 2},
		"debt:FIXME": {file: "b.go", line: 2},
		"debt:XXX":   {file: "b.go", line: 3},
	}
	for id, w := range want {
		got, ok := rules[id]
		if !ok {
			t.Errorf("rule %s missing; have %v", id, rules)
			continue
		}
		if got.file != w.file || got.line != w.line {
			t.Errorf("%s = %s:%d, want %s:%d", id, got.file, got.line, w.file, w.line)
		}
	}
	for _, f := range res.Findings {
		if f.Severity != "LOW" {
			t.Errorf("%s severity = %q, want LOW", f.RuleID, f.Severity)
		}
	}
}

// fileLineRef is the file/line pair a debt finding must carry.
type fileLineRef struct {
	file string
	line int
}

// TestCollectDebt_NoMarkers is a second input (triangulation): a clean tree
// yields an ok debt source with zero findings.
func TestCollectDebt_NoMarkers(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := newTestStore(t)
	svc := New(st, root, t.TempDir(), Config{})
	res := svc.collectDebt(ctx)
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if len(res.Findings) != 0 {
		t.Errorf("findings = %d, want 0 for a clean tree", len(res.Findings))
	}
}
