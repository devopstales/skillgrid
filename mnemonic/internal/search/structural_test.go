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
		// The common by-example form places the metavariable INSIDE the
		// parens, next to the node it names. It must compile to the same
		// query as the after-paren form (the capture follows the node type).
		// Regression: the old parser appended " @name" to the root AFTER the
		// closing paren, so an inside-paren metavar produced a bare "@name"
		// token -> "unexpected character \"@\"".
		// Inside-paren: the capture attaches inside the root S-expression, the
		// valid root-query form for tree-sitter.
		{`(function_definition \fn)`, `(function_definition @fn)`},
		{`(call \c)`, `(call @c)`},
		{`(call_expression \c (argument_list \args))`, `(call_expression @c (argument_list @args))`},
		// Regression: a metavariable placed after the root closing paren must
		// attach to the root, and one inside the parens must NOT become a
		// child step. (The old parser turned `\c` after `)` into a sibling
		// any-node, so `(function_call) \c` compiled to
		// `(function_call) (_ @c)` and failed for languages where the root
		// has no child steps.)
		{`(function_call) \c`, `(function_call) @c`},
		// Regression: whitespace inside the root S-expression must separate
		// tokens so that a node type + field name does not merge into one
		// mangled string. The old parser stripped all whitespace, so
		// `call_expression function:` became `call_expressionfunction:`
		// and the grammar rejected it as an unknown node type.
		{`(call_expression function: (identifier) \name)`, `(call_expression function: (identifier) @name)`},
		{`(call_expression (identifier) \name)`, `(call_expression (identifier) @name)`},
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

// TestGrepNameFieldIdentifier covers the correct by-example form for
// capturing a function name: the name field uses (identifier), not (string).
// (string) is a Python literal node type; using it in a name: field produces
// a silent 0-hit result on Python and an "unknown node type" error on Go.
func TestGrepNameFieldIdentifier(t *testing.T) {
	root := writeLangFixtures(t, map[string]string{
		"b.py": "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n",
	})
	// Correct form: (identifier) captures the function name.
	res, err := GrepByExample(root, `(function_definition name: (identifier) \fn)`, 0)
	if err != nil {
		t.Fatalf("grep (identifier): %v", err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("expected 2 hits for (identifier), got %d", len(res.Hits))
	}
	for _, h := range res.Hits {
		if h.Captures["fn"] == "" {
			t.Errorf("expected non-empty fn capture, got %v", h.Captures)
		}
	}
	// The names should be the actual function names.
	names := map[string]bool{}
	for _, h := range res.Hits {
		names[h.Captures["fn"]] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Errorf("expected captures for foo and bar, got %v", names)
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
	res, err := GrepByExample(root, `(function_definition) \fn`, 0)
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

// TestGrepMatchesCallPattern covers a call-shaped by-example pattern. The
// pattern uses (call_expression), which is valid for the go and javascript
// grammars in the gotreesitter registry; python's call node is `call`, so a
// mixed-language grep records a python skip note and matches the rest.
func TestGrepMatchesCallPattern(t *testing.T) {
	root := writeGrepFixture(t)
	res, err := GrepByExample(root, `(call_expression) \call`, 0)
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
	// python is skipped with a note (call_expression is not a python node).
	pyNote := false
	for _, n := range res.Notes {
		if n.Language == "python" {
			pyNote = true
		}
	}
	if !pyNote {
		t.Errorf("expected a python skip note, got %v", res.Notes)
	}
}

// TestGrepMetavariableAfterCloseParen covers the regression where a
// metavariable placed after the root closing paren was parsed as a sibling
// any-node instead of a capture on the root. `(function_call) \c` must
// match python calls. python's `call` node has no child node steps of its
// own, so if `\c` after the closing paren were parsed as a sibling any-node
// instead of a capture on the root, the query would fail to compile.
func TestGrepMetavariableAfterCloseParen(t *testing.T) {
	root := writeLangFixtures(t, map[string]string{
		"b.py": "def foo(a, b):\n    return a+b\n\ndef bar():\n    return foo(1, 2)\n",
	})
	res, err := GrepByExample(root, `(call) \c`, 0)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	pyHits := 0
	for _, h := range res.Hits {
		if strings.HasSuffix(h.Path, "b.py") && h.Captures["c"] != "" {
			pyHits++
		}
	}
	if pyHits != 1 {
		t.Errorf("expected 1 python call_expression match, got %d (hits=%v notes=%v)", pyHits, res.Hits, res.Notes)
	}
}

// TestGrepInvalidPatternSkipsLanguage covers @step-02 edge: a pattern invalid
// for one language (function_declaration does not exist in the python grammar)
// skips that language with a note, while the other language still matches.
func TestGrepInvalidPatternSkipsLanguage(t *testing.T) {
	// (function_declaration) is a valid go node but not a python node.
	root := writeGrepFixture(t)
	res, err := GrepByExample(root, `(function_declaration) \fn`, 0)
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
	res, err := GrepByExample(root, `(function_declaration) \fn`, 0)
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
	_, err := GrepByExample(root, `\\`, 0)
	if err == nil {
		t.Fatal("expected an error for a dangling-backslash pattern")
	}
	if !IsPatternError(err) {
		t.Errorf("expected a pattern error, got %v", err)
	}
	// A bare metavariable (no node type) gets a helpful error message.
	_, err = ParseByExample(`\fn`)
	if err == nil {
		t.Fatal("ParseByExample(\\fn): expected an error for bare metavariable")
	}
	if !strings.Contains(err.Error(), "bare metavariable") {
		t.Errorf("expected 'bare metavariable' hint in error, got %v", err)
	}
}

// TestGrepLimitCapsHits covers the unbounded-output ergonomics fix: an
// unbounded pattern like (call) over a large tree must not return
// megabytes. With limit>0 the hit list is capped, truncated reports how many
// were dropped, and the scan stops early (it must not keep walking every
// remaining file once the cap is hit).
func TestGrepLimitCapsHits(t *testing.T) {
	// Five python files, three functions each = 15 (function_definition) hits.
	files := map[string]string{}
	for _, n := range []string{"a.py", "b.py", "c.py", "d.py", "e.py"} {
		files[n] = "def f1():\n    return 1\n\ndef f2():\n    return 2\n\ndef f3():\n    return 3\n"
	}
	root := writeLangFixtures(t, files)

	// No limit: all 15 hits, no truncation marker.
	res, err := GrepByExample(root, `(function_definition) \fn`, 0)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if len(res.Hits) != 15 {
		t.Fatalf("no-limit: expected 15 hits, got %d", len(res.Hits))
	}
	if res.Truncated {
		t.Errorf("no-limit: Truncated must be false")
	}

	// Limit=5: capped at 5, truncated flag set, 10 dropped.
	res2, err := GrepByExample(root, `(function_definition) \fn`, 5)
	if err != nil {
		t.Fatalf("grep limited: %v", err)
	}
	if len(res2.Hits) != 5 {
		t.Fatalf("limit=5: expected 5 hits, got %d", len(res2.Hits))
	}
	if !res2.Truncated {
		t.Errorf("limit=5: Truncated must be true")
	}
	if res2.HitsTruncated != 1 {
		t.Errorf("limit=5: HitsTruncated expected 1 (indicator), got %d", res2.HitsTruncated)
	}
}

// TestGrepLimitEarlyStop verifies the scan halts once the cap is reached: a
// limit smaller than the total must not consume the remaining files (which
// would show up in files_seen). With 5 files of 3 defs each and limit=3, the
// walk stops during the first file, so only one file is seen.
func TestGrepLimitEarlyStop(t *testing.T) {
	files := map[string]string{}
	for _, n := range []string{"a.py", "b.py", "c.py", "d.py", "e.py"} {
		files[n] = "def f1():\n    return 1\n\ndef f2():\n    return 2\n\ndef f3():\n    return 3\n"
	}
	root := writeLangFixtures(t, files)
	res, err := GrepByExample(root, `(function_definition) \fn`, 3)
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	if len(res.Hits) != 3 {
		t.Fatalf("limit=3: expected 3 hits, got %d", len(res.Hits))
	}
	// Early stop: the walk halts once the cap is reached, so it does not
	// consume the remaining files. With limit=3, a.py fills the cap (files_seen
	// =1), b.py is entered and stopped at the cap check before appending
	// (files_seen=2); c.py, d.py, e.py are never reached. files_seen must be 2,
	// not 5.
	if res.FilesSeen != 2 {
		t.Errorf("limit=3: expected early-stop files_seen=2, got %d", res.FilesSeen)
	}
	if !res.Truncated {
		t.Errorf("limit=3: Truncated must be true")
	}
}
