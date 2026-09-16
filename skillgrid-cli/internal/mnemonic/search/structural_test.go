package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGrepFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("a.go", "package x\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n\nfunc main() {\n\tv := add(1, 2)\n\t_ = v\n}\n")
	write("b.py", "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n")
	write("c.txt", "func notAGoFile() {}\n")
	return root
}

// writeLangFixtures writes one file per language for a single-language grep.
func writeLangFixtures(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return root
}

// TestParseByExample locks the metavariable conversion.
func TestParseByExample(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`(function_definition) \fn`, `(function_definition) @fn`},
		{`(call) \c`, `(call) @c`},
		{`(call_expression) \c \*`, `(call_expression) @c (_)`},
		{`(call_expression) \c \(ARGS*)`, `(call_expression) @c (argument_list) @args`},
		{`(function_definition) \fn \_`, `(function_definition) @fn (_)`},
	}
	for _, c := range cases {
		got, err := ParseByExample(c.in)
		if err != nil {
			t.Fatalf("ParseByExample(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseByExample(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// Unbalanced parens -> pattern error.
	if _, err := ParseByExample("(call"); err == nil {
		t.Error("expected error for unbalanced pattern")
	}
}

// TestGrepMatchesByExampleIndexFree covers @step-02 happy: a by-example
// pattern matches the syntax tree, index-free (no store required), per
// language.
func TestGrepMatchesByExampleIndexFree(t *testing.T) {
	// Python-only file so the (function_definition) pattern is valid for every
	// language present.
	root := writeLangFixtures(t, map[string]string{
		"b.py": "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n",
	})
	res, err := GrepByExample(root, `(function_definition) \fn`)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	// Both foo and bar should match; no unknown files present.
	pyHits := 0
	for _, h := range res.Hits {
		if strings.HasSuffix(h.Path, "b.py") {
			pyHits++
			if h.Captures["fn"] == "" {
				t.Errorf("expected a non-empty fn capture, got %v", h.Captures)
			}
		}
	}
	if pyHits != 2 {
		t.Errorf("expected 2 python function matches, got %d", pyHits)
	}
	// No notes (the pattern is valid for python).
	if len(res.Notes) != 0 {
		t.Errorf("unexpected notes: %v", res.Notes)
	}
}

// TestGrepMatchesCallPattern covers a call-shaped by-example pattern.
func TestGrepMatchesCallPattern(t *testing.T) {
	root := writeGrepFixture(t)
	// Go call expression: (call_expression) \call
	res, err := GrepByExample(root, `(call_expression) \call`)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	goCalls := 0
	for _, h := range res.Hits {
		if strings.HasSuffix(h.Path, "a.go") {
			goCalls++
		}
	}
	if goCalls == 0 {
		t.Errorf("expected go call_expression matches, got %d (hits=%v)", goCalls, res.Hits)
	}
}

// TestGrepInvalidPatternSkipsLanguage covers @step-02 edge: a pattern invalid
// for one language (function_declaration does not exist in the python grammar)
// skips that language with a note, while the other language still matches.
func TestGrepInvalidPatternSkipsLanguage(t *testing.T) {
	// (function_declaration) is a valid go node but not a python node.
	root := writeGrepFixture(t)
	res, err := GrepByExample(root, `(function_declaration) \fn`)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	// Go files matched (the pattern is valid for go).
	goHits := 0
	for _, h := range res.Hits {
		if strings.HasSuffix(h.Path, "a.go") {
			goHits++
		}
	}
	if goHits == 0 {
		t.Errorf("expected go matches for a go-valid pattern, got 0")
	}
	// Python skipped with a note (function_declaration is not a python node).
	pyNote := false
	for _, n := range res.Notes {
		if n.Language == "python" {
			pyNote = true
		}
	}
	if !pyNote {
		t.Errorf("expected a python skip note, got %v", res.Notes)
	}
	// It is NOT a silent no-match: we have both hits and a note.
	if len(res.Hits) == 0 && len(res.Notes) == 0 {
		t.Errorf("grep produced no hits and no notes: silent no-match")
	}
}

// TestGrepUnknownFilesSkipped covers unknown files (c.txt) being skipped.
func TestGrepUnknownFilesSkipped(t *testing.T) {
	root := writeGrepFixture(t)
	res, err := GrepByExample(root, `(function_declaration) \fn`)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	for _, h := range res.Hits {
		if strings.HasSuffix(h.Path, "c.txt") {
			t.Errorf("unknown file c.txt should be skipped, got hit %+v", h)
		}
	}
}

// TestGrepBadPatternRejected covers a structurally invalid pattern aborting
// with a clear error, not a silent no-match. An empty pattern and a dangling
// backslash are pattern-level errors.
func TestGrepBadPatternRejected(t *testing.T) {
	for _, p := range []string{`\\`, `(function_definition`, `(function_definition) \\`} {
		_, err := ParseByExample(p)
		if err == nil {
			t.Errorf("ParseByExample(%q): expected an error", p)
		}
	}
	// A dangling backslash is a pattern-level error that GrepByExample
	// surfaces before any per-language compile.
	root := writeGrepFixture(t)
	_, err := GrepByExample(root, `\\`)
	if err == nil {
		t.Fatal("expected an error for a dangling-backslash pattern")
	}
	if !IsPatternError(err) {
		t.Errorf("expected a pattern error, got %v", err)
	}
}
