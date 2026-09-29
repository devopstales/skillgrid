package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// conformantPage is a minimal OKF page that passes the conformance gate:
// a well-formed frontmatter block with a non-empty `type`.
const conformantPage = `---
type: adr
status: stable
generated:
  at: 2026-09-29T00:00:00Z
---

# ADR-0006: Example decision

## Decision

We choose approach A.
`

// nonConformantPage has frontmatter but is missing the required `type` key.
const nonConformantPage = `---
status: stable
generated:
  at: 2026-09-29T00:00:00Z
---

# A page with no type
`

// TestWikiNoSubcommand verifies that `skillgrid wiki` with no subcommand
// dispatches to usage and returns exit code 2 (R9.3).
func TestWikiNoSubcommand(t *testing.T) {
	code := dispatchWiki(nil)
	if code != 2 {
		t.Fatalf("no subcommand exit = %d, want 2", code)
	}
}

// TestWikiUnknownSubcommand verifies an unknown subcommand returns exit 2.
func TestWikiUnknownSubcommand(t *testing.T) {
	code := dispatchWiki([]string{"frobnicate"})
	if code != 2 {
		t.Fatalf("unknown subcommand exit = %d, want 2", code)
	}
}

// TestWikiHelp verifies `wiki help` prints usage and returns 0.
func TestWikiHelp(t *testing.T) {
	code := dispatchWiki([]string{"help"})
	if code != 0 {
		t.Fatalf("help exit = %d, want 0", code)
	}
}

// TestWikiLintClean verifies that a bundle of conformant pages lints clean
// (zero failures) and returns exit code 0 (R9.2 happy path).
func TestWikiLintClean(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, ".wiki", "wiki")
	if err := os.MkdirAll(filepath.Join(wikiDir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wikiDir, "entities"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "adr", "adr-0006-example.md"), []byte(conformantPage), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "entities", "state.md"), []byte(conformantPage), 0o644); err != nil {
		t.Fatal(err)
	}

	fails := wikiLintFiles(wikiDir)
	if len(fails) != 0 {
		t.Fatalf("expected 0 failures, got %d: %+v", len(fails), fails)
	}

	// runWikiLint returns 0 for a clean bundle.
	code := runWikiLint([]string{"--project", dir})
	if code != 0 {
		t.Fatalf("clean lint exit = %d, want 0", code)
	}
}

// TestWikiLintBad verifies that a non-conformant page is reported as a failure
// (R9.2: non-zero exit path), the walk continues past the failure, and the
// missing-dir case stays clean.
func TestWikiLintBad(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, ".wiki", "wiki")
	if err := os.MkdirAll(filepath.Join(wikiDir, "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wikiDir, "adr", "adr-0001-bad.md"), []byte(nonConformantPage), 0o644); err != nil {
		t.Fatal(err)
	}
	// A conformant sibling to confirm the walk continues past the failure.
	if err := os.WriteFile(filepath.Join(wikiDir, "adr", "adr-0002-good.md"), []byte(conformantPage), 0o644); err != nil {
		t.Fatal(err)
	}

	fails := wikiLintFiles(wikiDir)
	if len(fails) != 1 {
		t.Fatalf("expected exactly 1 failure, got %d: %+v", len(fails), fails)
	}
	if !strings.Contains(fails[0].RelPath, "adr-0001-bad") {
		t.Errorf("failure should point at the bad page, got %q", fails[0].RelPath)
	}
	if len(fails[0].Errors) == 0 {
		t.Error("failure should carry at least one conformance error")
	}

	// runWikiLint returns 1 for a bundle with conformance failures.
	code := runWikiLint([]string{"--project", dir})
	if code != 1 {
		t.Fatalf("failing lint exit = %d, want 1", code)
	}

	// Missing dir → clean (uncompiled bundle is not an error).
	missing := filepath.Join(dir, "does-not-exist")
	if f := wikiLintFiles(missing); len(f) != 0 {
		t.Fatalf("missing dir should yield 0 failures, got %d: %+v", len(f), f)
	}
}
