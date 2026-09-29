package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRawSnapshot hashes every file under root (relative path → content) so a
// test can prove raw/ is byte-identical before and after a compile (R6.3).
func writeRawSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		snap[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	return snap
}

// compileE2EFixture builds a full fixture project: every pillar-1 source, a
// seeded web_cache row set, and a raw/ tree (R7.1).
func compileE2EFixture(t *testing.T) (string, []WebRow) {
	t.Helper()
	proj := t.TempDir()

	writeFixtureFile(t, proj, ".skillgrid/ASSUMPTIONS.md", `# ASSUMPTIONS

Intro.

### In-force set

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0001 | First ADR | accepted | — | — | 2026-09-01 | yes |
| 0002 | Second ADR | superseded | — | — | 2026-09-02 | no |
| 0003 | Third ADR | accepted | 0001 | — | 2026-09-03 | yes |

### ADR-0001 — First ADR

**Decision.** Some decision.

### ADR-0002 — Second ADR

**Decision.** An old decision.

### ADR-0003 — Third ADR

**Decision.** A newer decision.

### Locked constraints

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
`)
	writeFixtureFile(t, proj, ".skillgrid/state.yaml", "schema: skillgrid/state/v1\npipeline:\n  current_phase: \"executing\"\n  current_change: \"test-change\"\nprogress:\n  completed_changes: 1\n")
	writeFixtureFile(t, proj, ".skillgrid/ARCHITECTURE.md", "# Architecture\n\n## System overview\n\nMonorepo layout.\n")
	writeFixtureFile(t, proj, ".skillgrid/artifacts/01-business-terms.md", "# Business terms\n\n| Term | Definition | Use when |\n|------|------------|----------|\n| Compile | Render sources to the wiki bundle | When building |\n")
	writeFixtureFile(t, proj, ".skillgrid/artifacts/02-technical-terms.md", "# Technical terms\n\n| Term | Definition |\n|------|------------|\n| Manifest | Content-hash gate file | When recompiling |\n")
	writeFixtureFile(t, proj, ".skillgrid/specs/test-change/briefing.md", "# Test Change\n\n> **STATUS:** `draft` (2026-09-29)\n\n**Goal:** Verify the e2e fixture.\n")
	writeFixtureFile(t, proj, ".skillgrid/spikes/001-test/foundations.md", "# Spike 001\n\n## Verdict\n\n**VALIDATED** — the approach works.\n")
	writeFixtureFile(t, proj, "raw/web/notes/okf-note.md", "---\ntype: Finding\ntitle: Sigma Graph Note\nsources:\n  - resource: https://example.com/sigma\n---\n\nBody of the clipper note.\n")
	writeFixtureFile(t, proj, "raw/loose-note.md", "# Loose note\n\nPlain markdown, no frontmatter.\n")

	rows := []WebRow{
		// Uncited fresh row → research/ draft (R5.2).
		{Source: "context7", URL: "https://context7.com/nextjs", Title: "Next.js API reference", FetchedAt: "2026-09-28T10:00:00Z"},
		// Cited fresh row (url cited by the raw note's sources) → entities/ Finding (R5.1).
		{Source: "exa", URL: "https://example.com/sigma", Title: "Sigma graph article", FetchedAt: "2026-09-27T09:30:00Z"},
		// Expired row → filtered by the CLI query, never reaches Compile (R5.3).
		{Source: "fetch", URL: "https://example.com/expired", Title: "Expired research", FetchedAt: "2026-09-28T08:00:00Z", ExpiresAt: "2026-01-02T00:00:00Z"},
		// Excluded source → skipped (R5).
		{Source: "manual", URL: "https://example.com/manual", Title: "Manual row", FetchedAt: "2026-09-28T10:00:00Z"},
	}
	return proj, rows
}

func TestCompileE2EFullTree(t *testing.T) {
	proj, rows := compileE2EFixture(t)
	out := filepath.Join(proj, ".wiki")

	rawBefore := writeRawSnapshot(t, filepath.Join(proj, "raw"))

	res, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow, WebRows: rows})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if res.Written == 0 {
		t.Fatalf("expected files written, got 0 (unchanged=%d)", res.Unchanged)
	}

	files, _ := readAllFiles(t, out)

	// R7.1: full tree — every emitted page exists.
	wantPages := []string{
		"AGENTS.md",
		"wiki/index.md",
		"wiki/log.md",
		"wiki/adr/adr-0001.md",
		"wiki/adr/adr-0002.md",
		"wiki/adr/adr-0003.md",
		"wiki/concepts/locked-constraint-1.md",
		"wiki/concepts/locked-constraint-2.md",
		"wiki/concepts/compile.md",
		"wiki/concepts/manifest.md",
		"wiki/entities/project-state.md",
		"wiki/entities/architecture.md",
		"wiki/entities/test-change.md",
		"wiki/entities/001-test.md",
		"wiki/research/next-js-api-reference.md",
		"wiki/entities/sigma-graph-article.md",
		"wiki/entities/sigma-graph-note.md",
		"wiki/entities/loose-note.md",
	}
	for _, rel := range wantPages {
		if _, ok := files[rel]; !ok {
			t.Errorf("missing %s in tree: %v", rel, sortedKeys(files))
		}
	}
	// Excluded (manual) row produces no page; expired rows never reach Compile.
	for _, rel := range []string{"wiki/entities/manual-row.md", "wiki/research/manual-row.md"} {
		if _, ok := files[rel]; ok {
			t.Errorf("unexpected page %s (expired/excluded row must be skipped)", rel)
		}
	}

	// R5.1: cited row lands in entities/ with author process:exa and verified.
	cited := files["wiki/entities/sigma-graph-article.md"]
	for _, want := range []string{"type: Finding", "status: stable", 		"resource: \"https://example.com/sigma\"", "author: \"process:exa\""} {
		if !strings.Contains(cited, want) {
			t.Errorf("cited finding missing %q in:\n%s", want, cited)
		}
	}
	// R5.2: uncited row lands in research/ with status draft.
	uncited := files["wiki/research/next-js-api-reference.md"]
	if !strings.Contains(uncited, "status: draft") {
		t.Errorf("uncited finding not draft:\n%s", uncited)
	}

	// R5.4: no file under raw/ came from web rows — raw/ bytes unchanged.
	rawAfter := writeRawSnapshot(t, filepath.Join(proj, "raw"))
	if len(rawBefore) != len(rawAfter) {
		t.Fatalf("raw/ file count changed: before=%d after=%d", len(rawBefore), len(rawAfter))
	}
	for rel, content := range rawBefore {
		if rawAfter[rel] != content {
			t.Errorf("raw/ file changed: %s", rel)
		}
	}

	// R7.1: index.md lists every emitted page.
	idx := files["wiki/index.md"]
	for _, want := range []string{
		"[[adr-0001]]", "[[adr-0003]]",
		"[[compile]]", "[[manifest]]",
		"[[project-state]]", "[[architecture]]", "[[test-change]]", "[[001-test]]",
		"[[next-js-api-reference]]", "[[sigma-graph-article]]",
		"[[sigma-graph-note]]", "[[loose-note]]",
		"Research (uncited)",
	} {
		if !strings.Contains(idx, want) {
			t.Errorf("index.md missing %q in:\n%s", want, idx)
		}
	}

	// R7.2: AGENTS.md carries the schema (taxonomy, slug rule, lint rule).
	agents := files["AGENTS.md"]
	for _, want := range []string{
		"Type Taxonomy", "ADR", "Term", "Spec", "Spike", "State", "Constraint", "Architecture", "Finding", "Source",
		"Slug rule", "Lint rule", "research/",
	} {
		if !strings.Contains(agents, want) {
			t.Errorf("AGENTS.md missing %q", want)
		}
	}

	// R2.3: every emitted page passes Conform, except spike pages which carry
	// the spike verdict (VALIDATED / INVALIDATED / PARTIAL) as status by
	// parser contract — an accepted OKF deviation (see ticket note).
	for rel, content := range files {
		if rel == "AGENTS.md" || rel == "wiki/index.md" || rel == "wiki/log.md" || rel == ".wiki-manifest.json" {
			continue
		}
		for _, e := range Conform(content) {
			if strings.HasPrefix(rel, "wiki/entities/001-test") && strings.Contains(e.Error(), "status:") {
				continue
			}
			t.Errorf("Conform(%s): %v", rel, e)
		}
	}

	// R8.1: no-op recompile is byte-identical.
	if _, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow, WebRows: rows}); err != nil {
		t.Fatalf("recompile: %v", err)
	}
	after2, _ := readAllFiles(t, out)
	if len(files) != len(after2) {
		t.Fatalf("file count changed on recompile: before=%d after=%d", len(files), len(after2))
	}
	for rel, content := range files {
		if after2[rel] != content {
			t.Errorf("file changed on no-op recompile: %s", rel)
		}
	}
}

func TestCompileE2EMissingOptionalSources(t *testing.T) {
	proj := t.TempDir() // no .skillgrid at all, no raw/, no web rows
	out := filepath.Join(proj, ".wiki")
	res, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow})
	if err != nil {
		t.Fatalf("Compile with all optional sources missing: %v", err)
	}
	if res.Written < 3 {
		t.Errorf("expected at least 3 files written (AGENTS, index, log), got %d", res.Written)
	}
	if _, err := os.Stat(filepath.Join(proj, "raw")); err == nil {
		t.Errorf("compile must not create raw/")
	}
}
