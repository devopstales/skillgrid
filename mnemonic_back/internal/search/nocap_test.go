package search

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempPyFile creates a single Python file and returns its absolute path.
func writeTempPyFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "certificate.py")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNoCapturePositions(t *testing.T) {
	root := writeLangFixtures(t, map[string]string{
		"b.py": "import os\nimport sys\n\ndef foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n",
	})
	res, err := GrepByExample(root, `(function_definition)`, 0)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(res.Hits))
	}
	for _, h := range res.Hits {
		if h.Line == 1 && h.Col == 1 {
			t.Errorf("bare pattern hit at 1:1 — no-capture path not fixed: %v", h)
		}
		if h.Text == "" {
			t.Errorf("bare pattern hit has empty text: %v", h)
		}
	}
}

// TestNoCaptureSubNode verifies that a pattern whose metavariable targets a
// sub-node (the name field) still anchors to the ROOT node for position/text,
// not the sub-node. Round-3: (function_definition name: (identifier)) with no
// backslash capture hit the same 1:1 path.
func TestNoCaptureSubNode(t *testing.T) {
	root := writeLangFixtures(t, map[string]string{
		"c.py": "def foo(a, b):\n    return a+b\n",
	})
	// No backslash capture, but a sub-node spec. The anchor must be the
	// function_definition root (line 1), not a missing capture.
	res, err := GrepByExample(root, `(function_definition name: (identifier))`, 0)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(res.Hits))
	}
	h := res.Hits[0]
	if h.Line != 1 {
		t.Errorf("expected root line 1, got %d (hit %v)", h.Line, h)
	}
	if h.Text == "" {
		t.Errorf("expected non-empty root text, got empty (hit %v)", h)
	}
}

// TestNoCaptureFileDirect verifies that when path is a single FILE (not a dir),
// a bare pattern reports the file's basename, not ".". Round-3 regression.
func TestNoCaptureFileDirect(t *testing.T) {
	p := writeTempPyFile(t, "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n")
	res, err := GrepByExample(p, `(function_definition)`, 0)
	if err != nil {
		t.Fatalf("grep file: %v", err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(res.Hits))
	}
	for _, h := range res.Hits {
		if h.Path == "." {
			t.Errorf("path is '.' for a direct-file grep — file dropped: %v", h)
		}
		if h.Path != "certificate.py" {
			t.Errorf("expected path 'certificate.py', got %q (hit %v)", h.Path, h)
		}
		// def foo is on line 1 (col 1 is correct for it); def bar must be on
		// line 4 with non-empty text — proves the anchor applied (not 1:1
		// everywhere).
		if h.Path == "certificate.py" && h.Line == 4 && h.Text == "" {
			t.Errorf("line-4 hit has empty text — anchor not applied: %v", h)
		}
	}
}

// TestNoCaptureDirRel verifies relative paths are unchanged for dir greps.
func TestNoCaptureDirRel(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "mod.py"), []byte("def a():\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := GrepByExample(root, `(function_definition)`, 0)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(res.Hits))
	}
	if res.Hits[0].Path != "pkg/mod.py" {
		t.Errorf("expected relative path 'pkg/mod.py', got %q", res.Hits[0].Path)
	}
	if res.Hits[0].Line != 1 {
		t.Errorf("expected line 1, got %d", res.Hits[0].Line)
	}
}
