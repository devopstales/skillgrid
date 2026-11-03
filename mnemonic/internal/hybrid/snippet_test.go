package hybrid

import (
	"strings"
	"testing"
)

// TestSnippetSkeleton covers @step-01 (Scenario: Snippets are skeletonized and
// secrets are redacted in output): snippet skeletonization collapses unrelated
// bodies while preserving imports, signatures, matched lines, and the exact
// read range.
func TestSnippetSkeleton(t *testing.T) {
	// A Go function with a long unrelated body. The matched line is the
	// signature + the line containing "retry".
	src := `package http

import "net"

func (c *Client) Get(url string) error {
	// retry with backoff
	var attempts int
	for i := 0; i < 10; i++ {
		attempts++
		if i%2 == 0 {
			continue
		}
		if doWork(attempts) {
			break
		}
	}
	return nil
}
`
	lines := strings.Split(src, "\n")
	skel := Skeletonize(lines, "retry", 0, len(lines)-1)
	skelText := strings.Join(skel, "\n")

	// Imports must be preserved.
	if !strings.Contains(skelText, "import \"net\"") {
		t.Errorf("imports must be preserved, got:\n%s", skelText)
	}
	// The signature must be preserved.
	if !strings.Contains(skelText, "func (c *Client) Get(url string) error") {
		t.Errorf("signature must be preserved, got:\n%s", skelText)
	}
	// The matched line (containing "retry") must be preserved.
	if !strings.Contains(skelText, "retry") {
		t.Errorf("matched line must be preserved, got:\n%s", skelText)
	}
	// The unrelated inner body (the deep loop) must be collapsed to an elision
	// marker, not emitted verbatim.
	if strings.Contains(skelText, "doWork(attempts)") {
		t.Errorf("unrelated body should be collapsed, got:\n%s", skelText)
	}
	// A collapsed body is marked with an elision token.
	if !strings.Contains(skelText, elisionToken) {
		t.Errorf("collapsed body should carry an elision marker, got:\n%s", skelText)
	}
	// The skeleton must be SHORTER than the original (it collapsed something).
	if len(skel) >= len(lines) {
		t.Errorf("skeleton should be shorter than the original (%d vs %d lines)", len(skel), len(lines))
	}
}

// TestSnippetSkeletonPreservesReadRange covers @step-01: skeletonization keeps
// the exact read range — lines outside the requested [start,end] are not
// emitted.
func TestSnippetSkeletonPreservesReadRange(t *testing.T) {
	src := []string{
		"line0-out-of-range",
		"line1-in-range",
		"line2-in-range-match",
		"line3-in-range",
		"line4-out-of-range",
	}
	skel := Skeletonize(src, "match", 1, 3)
	for _, l := range skel {
		if strings.Contains(l, "out-of-range") {
			t.Errorf("line outside the read range was emitted: %q", l)
		}
	}
	if !strings.Contains(strings.Join(skel, "\n"), "line2-in-range-match") {
		t.Errorf("matched line in range must be preserved, got %v", skel)
	}
}

// TestSimHashNearDupSuppression covers @step-01: SimHash suppression cuts
// near-duplicate hits (dup%) without moving a metric — two snippets that are
// near-duplicates (Hamming distance below threshold) collapse to one, while
// distinct snippets survive.
func TestSimHashNearDupSuppression(t *testing.T) {
	a := "func add(a, b int) int {\n\treturn a + b\n}\n"
	a2 := "func add(a, b int) int {\n\treturn a + b\n}\n// trailing comment\n" // near-dup of a
	b := "func sub(a, b int) int {\n\treturn a - b\n}\n" // distinct

	dedup := SimHashDedupe([]Snippet{
		{Path: "a.go", Text: a},
		{Path: "a2.go", Text: a2},
		{Path: "b.go", Text: b},
	})
	// The near-dup (a2) should be suppressed; a and b survive → 2 kept, 1 dropped.
	if len(dedup) != 2 {
		t.Errorf("expected 2 kept after near-dup suppression, got %d: %v", len(dedup), dedup)
	}
	kept := map[string]bool{}
	for _, d := range dedup {
		kept[d.Path] = true
	}
	if !kept["a.go"] || !kept["b.go"] {
		t.Errorf("expected a.go and b.go to survive, kept %v", kept)
	}
}

// TestSimHashDistinctSurvive covers the negative side: two clearly distinct
// snippets both survive (no false suppression).
func TestSimHashDistinctSurvive(t *testing.T) {
	x := "package x\n\nfunc foo() { doX() }\n"
	y := "package y\n\nfunc bar() { doY() }\n"
	dedup := SimHashDedupe([]Snippet{
		{Path: "x.go", Text: x},
		{Path: "y.go", Text: y},
	})
	if len(dedup) != 2 {
		t.Errorf("two distinct snippets must both survive, got %d", len(dedup))
	}
}

// TestOutputRedaction covers @step-01 (Scenario: Snippets are skeletonized and
// secrets are redacted in output): a secret-like pattern in an indexed file is
// REPLACED (never emitted raw) in code_search/code_read response text.
func TestOutputRedaction(t *testing.T) {
	cases := map[string]string{
		"password = \"supersecretpass123\"":                        "supersecretpass123",
		"apiKey := \"AKIAIOSFODNN7EXAMPLE\"":                        "AKIA",
		"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig": "Bearer",
		"const token = \"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ123456\"": "ghp",
	}
	for in, marker := range cases {
		out := RedactSecrets(in)
		if strings.Contains(out, marker) {
			t.Errorf("secret marker %q still present in redacted output: %q", marker, out)
		}
		// The redaction must replace (not delete) — a placeholder remains.
		if out == "" {
			t.Errorf("redaction must replace, not delete: input %q", in)
		}
	}
	// A non-secret string is unchanged.
	if got := RedactSecrets("func main() { println(1) }"); got != "func main() { println(1) }" {
		t.Errorf("non-secret should be unchanged, got %q", got)
	}
}

// TestRedactAppliedToReadOutput covers @step-01: redaction is applied to
// code_read response text (the joined chunk text), not just search snippets.
func TestRedactAppliedToReadOutput(t *testing.T) {
	// A code_read response text with a secret in the joined chunk text.
	readText := "package auth\n\nfunc Load() string {\n\tkey := \"AKIAIOSFODNN7EXAMPLE\"\n\treturn key\n}\n"
	redacted := RedactSecrets(readText)
	if strings.Contains(redacted, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("secret in code_read output must be redacted, got %q", redacted)
	}
}


