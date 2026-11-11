package projectwiki

import (
	"context"
	"strings"
	"testing"
)

// W006 (QA 2026-11-10): engine lint implements spec 007 §2 beyond
// missing_title/stale_index — orphans, broken wikilinks, duplicate
// aliases, missing sources, ungrounded grounded pages, past
// stale_after, weak cross-references, contradiction markers, broken
// code: citations, and coverage gaps.

func lintCodes(issues []LintIssue) map[string]LintIssue {
	out := map[string]LintIssue{}
	for _, i := range issues {
		out[i.Code+"@"+i.RelPath] = i
	}
	return out
}

func TestLintStructuralChecks(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	repo := t.TempDir()

	// Grounded page: no anchors, a broken wikilink, one outgoing link,
	// a contradiction marker, a past stale_after, and a code: citation
	// to a path that does not exist.
	writeVaultPage(t, vault, "decisions/lonely.md", "---\n"+
		"title: Lonely\ntype: Decision\nstatus: stable\nstale_after: 2020-01-01\n"+
		"sources:\n  - \"code:internal/missing/nope.go\"\n"+
		"---\n\nSee [[decisions/ghost]] for details.\n\n## Contradiction\n\nSources disagree.\n")
	// Two pages sharing one alias; each links to the other and to
	// decisions/lonely (so lonely has in_degree 2 and is no orphan).
	writeVaultPage(t, repo, "internal/real/real.go", "package real\n")
	writeVaultPage(t, vault, "architecture/alpha.md", "---\n"+
		"title: Alpha\ntype: Architecture\nstatus: stable\naliases: [shared]\n"+
		"sources:\n  - \"code:internal/real/real.go\"\n"+
		"---\n\nLinks [[architecture/beta]] and [[decisions/lonely]].\n")
	writeVaultPage(t, vault, "architecture/beta.md", "---\n"+
		"title: Beta\ntype: Architecture\nstatus: stable\naliases: [shared]\n"+
		"sources:\n  - \"code:internal/real/real.go\"\n"+
		"---\n\nLinks [[architecture/alpha]] and [[decisions/lonely]].\n")
	// Grounded page with no sources and no incoming links.
	writeVaultPage(t, vault, "decisions/bare.md",
		"---\ntitle: Bare\ntype: Decision\nstatus: stable\n---\n\nNo provenance.\n")

	st := openProjectWikiStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	issues, err := Lint(ctx, st, vault, repo, "proj-a")
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	got := lintCodes(issues)

	for _, w := range []struct{ code, rel, severity string }{
		{"broken_wikilink", "decisions/lonely.md", "error"},
		{"duplicate_alias", "architecture/beta.md", "warn"},
		{"ungrounded_grounded", "decisions/lonely.md", "error"},
		{"past_stale_after", "decisions/lonely.md", "warn"},
		{"weak_cross_reference", "decisions/lonely.md", "warn"},
		{"contradiction", "decisions/lonely.md", "warn"},
		{"broken_citation", "decisions/lonely.md", "warn"},
		{"missing_sources", "decisions/bare.md", "error"},
		{"orphan", "decisions/bare.md", "warn"},
	} {
		i, ok := got[w.code+"@"+w.rel]
		if !ok {
			t.Errorf("missing %s for %s in %+v", w.code, w.rel, issues)
			continue
		}
		if i.Severity != w.severity {
			t.Errorf("%s severity = %q, want %q", w.code, i.Severity, w.severity)
		}
	}
	if _, ok := got["orphan@decisions/lonely.md"]; ok {
		t.Error("decisions/lonely has in_degree 2 (alpha+beta link it); must not be an orphan")
	}
	// internal/real is cited by alpha+beta — no coverage gap for it.
	for _, i := range issues {
		if i.Code == "coverage_gap" && strings.Contains(i.Message, "internal/real") {
			t.Errorf("coverage_gap must not flag the cited package: %+v", i)
		}
	}
}

func TestLintWeakPin(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/pinned.md", "---\n"+
		"title: Pinned\ntype: Decision\nstatus: stable\n"+
		"anchors:\n  - path: internal/auth/oauth.go\n    line: 42\n"+
		"sources:\n  - \"code:internal/auth/oauth.go\"\n"+
		"---\n\nBody links [[decisions/pinned]] and [[decisions/pinned]].\n")
	st := openProjectWikiStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	issues, err := Lint(ctx, st, vault, t.TempDir(), "proj-a")
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	got := lintCodes(issues)
	if i, ok := got["weak_pin@decisions/pinned.md"]; !ok || i.Severity != "warn" {
		t.Errorf("weak_pin = %+v (ok=%v), want warn for bare line: pin without version:", i, ok)
	}
	if _, ok := got["ungrounded_grounded@decisions/pinned.md"]; ok {
		t.Error("page with anchors must not be ungrounded_grounded")
	}
}
