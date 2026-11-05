package kb

import (
	"testing"
)

func TestSplitFrontmatterRoundTrip(t *testing.T) {
	page := `---
title: Auth Model
type: Decision
project: demo
aliases:
  - auth
anchors:
  - path: internal/auth/oauth.go
    all_of:
      - func NewOAuthClient
    line: 42
    version: "kb-lines-v1:sha256:ab:cd"
  - path: internal/auth/oauth.go
    symbol: NewOAuthClient
    called_from:
      - cmd/server/main.go
claims:
  - subject: token expiry
    literal: 15 * time.Minute
    path: internal/auth/oauth.go
sources:
  - memory:obs-1
  - code:internal/auth/oauth.go#L42-L48
  - external: some-doc
    version: v1.4.2
    source_url: https://example.com/doc
    source_commit: abc123
status: stable
stale_after: 2026-12-01
generated:
  by: kb-generate
  at: 2026-11-05T00:00:00Z
---

Body line one.
Body line two.
`
	fm, body, err := SplitFrontmatter(page)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if fm.Title != "Auth Model" || fm.Type != "Decision" || fm.Project != "demo" {
		t.Fatalf("fields: %+v", fm)
	}
	if len(fm.Aliases) != 1 || fm.Aliases[0] != "auth" {
		t.Fatalf("aliases: %+v", fm.Aliases)
	}
	if len(fm.Anchors) != 2 {
		t.Fatalf("anchors: %+v", fm.Anchors)
	}
	if fm.Anchors[0].Kind() != "token" || fm.Anchors[1].Kind() != "symbol" {
		t.Fatalf("anchor kinds: %+v", fm.Anchors)
	}
	if len(fm.Claims) != 1 || fm.Claims[0].Literal != "15 * time.Minute" || fm.Claims[0].Path != "internal/auth/oauth.go" {
		t.Fatalf("claims: %+v", fm.Claims)
	}
	if len(fm.Sources) != 3 {
		t.Fatalf("sources: %+v", fm.Sources)
	}
	if fm.Sources[0].Inline != "memory:obs-1" || fm.Sources[0].MemorySyncID() != "obs-1" {
		t.Fatalf("source0: %+v", fm.Sources[0])
	}
	if fm.Sources[1].Inline != "code:internal/auth/oauth.go#L42-L48" {
		t.Fatalf("source1: %+v", fm.Sources[1])
	}
	if fm.Sources[2].External != "some-doc" || fm.Sources[2].Version != "v1.4.2" ||
		fm.Sources[2].SourceCommit != "abc123" {
		t.Fatalf("source2: %+v", fm.Sources[2])
	}
	if fm.Status != "stable" || fm.StaleAfter != "2026-12-01" {
		t.Fatalf("status: %+v", fm)
	}
	if fm.Generated == nil || fm.Generated.By != "kb-generate" {
		t.Fatalf("generated: %+v", fm.Generated)
	}
	// Body = everything after the closing "---\n" (byte-stable: the blank
	// line after the frontmatter belongs to the body).
	if body != "\nBody line one.\nBody line two.\n" {
		t.Fatalf("body: %q", body)
	}

	// Render → re-split must be byte-stable in fields.
	again, body2, err := SplitFrontmatter(fm.Render(body))
	if err != nil {
		t.Fatalf("re-split: %v", err)
	}
	if again.Title != fm.Title || len(again.Anchors) != 2 || len(again.Sources) != 3 {
		t.Fatalf("round-trip drift: %+v", again)
	}
	if body2 != body {
		t.Fatalf("body drift:\n%q\n%q", body2, body)
	}
}

func TestSplitFrontmatterNoBlock(t *testing.T) {
	fm, body, err := SplitFrontmatter("just a plain page\n")
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if fm.Title != "" || body != "just a plain page\n" {
		t.Fatalf("no-block: fm=%+v body=%q", fm, body)
	}
	// Unterminated block → also treated as body-only.
	fm2, body2, _ := SplitFrontmatter("---\ntitle: x\nno closing")
	if fm2.Title != "" || body2 != "---\ntitle: x\nno closing" {
		t.Fatalf("unterminated: fm=%+v body=%q", fm2, body2)
	}
}

func TestAnchorKind(t *testing.T) {
	token := Anchor{AllOf: []string{"a"}}
	if token.Kind() != "token" {
		t.Fatal("token kind")
	}
	span := Anchor{Line: 7, Version: "v"}
	if span.Kind() != "span" {
		t.Fatal("span kind")
	}
	bare := Anchor{Line: 7}
	if bare.Kind() != "span" {
		t.Fatal("bare-line span kind")
	}
	sym := Anchor{Symbol: "Foo"}
	if sym.Kind() != "symbol" {
		t.Fatal("symbol kind")
	}
	barePath := Anchor{Path: "x.go"}
	if barePath.Kind() != "" {
		t.Fatal("no selector")
	}
}

func TestIsGrounded(t *testing.T) {
	for _, p := range []string{"Decision", "decision", "InvariantContract", "invariant", "BusinessWorkflow", "workflow"} {
		if !IsGrounded(p) {
			t.Fatalf("%s should be grounded", p)
		}
	}
	for _, p := range []string{"Gotcha", "gotcha", "glossaryterm", "architecture", ""} {
		if IsGrounded(p) {
			t.Fatalf("%s should not be grounded", p)
		}
	}
}
