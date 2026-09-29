package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// compileNow pins the compile clock for every compile test (determinism).
var compileNow = time.Date(2026, 9, 29, 14, 0, 0, 0, time.UTC)

// writeFixtureFile writes content under dir and returns the absolute path.
func writeFixtureFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// compileFixtureProject builds a minimal .skillgrid/ project in a temp dir.
func compileFixtureProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixtureFile(t, dir, ".skillgrid/ASSUMPTIONS.md", `# ASSUMPTIONS

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
	writeFixtureFile(t, dir, ".skillgrid/state.yaml", "schema: skillgrid/state/v1\npipeline:\n  current_phase: \"executing\"\n  current_change: \"test-change\"\nprogress:\n  completed_changes: 1\n")
	return dir
}

// readAllFiles returns every file under root (relative path → content) plus the
// ordered list of relative paths.
func readAllFiles(t *testing.T, root string) (map[string]string, []string) {
	t.Helper()
	out := map[string]string{}
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out, paths
}

func TestCompileFixture(t *testing.T) {
	proj := compileFixtureProject(t)
	out := filepath.Join(proj, ".wiki")

	res, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if res.Written == 0 {
		t.Fatalf("expected files written, got 0 (unchanged=%d)", res.Unchanged)
	}

	files, _ := readAllFiles(t, out)

	// Tree structure: AGENTS.md, wiki/index.md, wiki/log.md, per-type dirs.
	for _, rel := range []string{"AGENTS.md", "wiki/index.md", "wiki/log.md", "wiki/adr/adr-0001.md", "wiki/adr/adr-0002.md", "wiki/adr/adr-0003.md", "wiki/entities/project-state.md"} {
		if _, ok := files[rel]; !ok {
			t.Errorf("missing %s in tree: %v", rel, sortedKeys(files))
		}
	}
	// Constraint concepts land under concepts/.
	if len(files["wiki/concepts/"]) == 0 && !hasPrefixKey(files, "wiki/concepts/") {
		t.Errorf("missing wiki/concepts/ pages: %v", sortedKeys(files))
	}

	// Frontmatter + wikilinks on an ADR page.
	adr1 := files["wiki/adr/adr-0001.md"]
	if !strings.HasPrefix(adr1, "---\n") {
		t.Fatalf("adr-0001 missing frontmatter: %q", adr1[:min(40, len(adr1))])
	}
	for _, want := range []string{"type: ADR", "title: First ADR", "status: stable", "generated:", "at: 2026-09-29T14:00:00Z", "[[adr-0003]]"} {
		if !strings.Contains(adr1, want) {
			t.Errorf("adr-0001 missing %q in:\n%s", want, adr1)
		}
	}
	// Deprecated ADR + back-link to its superseder.
	adr2 := files["wiki/adr/adr-0002.md"]
	if !strings.Contains(adr2, "status: deprecated") {
		t.Errorf("adr-0002 not deprecated:\n%s", adr2)
	}
	// Superseder links the superseded.
	adr3 := files["wiki/adr/adr-0003.md"]
	if !strings.Contains(adr3, "[[adr-0001]]") {
		t.Errorf("adr-0003 missing wikilink to adr-0001:\n%s", adr3)
	}

	// Every emitted page passes Conform (R2.3).
	for rel, content := range files {
		if rel == "AGENTS.md" || rel == "wiki/index.md" || rel == "wiki/log.md" || rel == ".wiki-manifest.json" {
			continue
		}
		for _, e := range Conform(content) {
			t.Errorf("Conform(%s): %v", rel, e)
		}
	}

	// Manifest exists and covers every wiki file.
	man := files[".wiki-manifest.json"]
	if man == "" {
		t.Fatalf("missing .wiki-manifest.json")
	}
	if !strings.Contains(man, "generated.at") {
		t.Errorf("manifest missing generated.at entries")
	}

	// State page carries source_path into .skillgrid/.
	state := files["wiki/entities/project-state.md"]
	if !strings.Contains(state, "source_path:") || !strings.Contains(state, ".skillgrid/state.yaml") {
		t.Errorf("state page missing .skillgrid source_path:\n%s", state)
	}
}

func TestCompileNoopRecompile(t *testing.T) {
	proj := compileFixtureProject(t)
	out := filepath.Join(proj, ".wiki")

	if _, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow}); err != nil {
		t.Fatalf("first compile: %v", err)
	}
	before, _ := readAllFiles(t, out)

	res, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow})
	if err != nil {
		t.Fatalf("recompile: %v", err)
	}
	if res.Written != 0 {
		t.Fatalf("no-op recompile wrote %d files, want 0", res.Written)
	}
	after, afterPaths := readAllFiles(t, out)
	if len(before) != len(after) {
		t.Fatalf("file count changed: before=%d after=%d", len(before), len(after))
	}
	for rel, content := range before {
		if after[rel] != content {
			t.Errorf("file changed on no-op recompile: %s", rel)
		}
	}
	// No new files appeared.
	for _, p := range afterPaths {
		if _, ok := before[p]; !ok {
			t.Errorf("new file on no-op recompile: %s", p)
		}
	}
}

func TestCompileGeneratedAtPreserved(t *testing.T) {
	proj := compileFixtureProject(t)
	out := filepath.Join(proj, ".wiki")

	if _, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow}); err != nil {
		t.Fatalf("first compile: %v", err)
	}
	adr1Before, _ := os.ReadFile(filepath.Join(out, "wiki/adr/adr-0001.md"))

	// Modify only ADR-0002's source body.
	assumptions := filepath.Join(proj, ".skillgrid/ASSUMPTIONS.md")
	data, err := os.ReadFile(assumptions)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.ReplaceAll(string(data), "**Decision.** An old decision.", "**Decision.** An old decision, now amended text.")
	if changed == string(data) {
		t.Fatal("fixture edit did not take")
	}
	if err := os.WriteFile(assumptions, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}

	// Recompile at a later clock.
	adr2Path := filepath.Join(out, "wiki/adr/adr-0002.md")
	if _, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow.Add(24 * time.Hour)}); err != nil {
		t.Fatalf("recompile: %v", err)
	}

	// Unchanged ADR-0001: byte-identical, generated.at preserved.
	adr1After, err := os.ReadFile(filepath.Join(out, "wiki/adr/adr-0001.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(adr1After) != string(adr1Before) {
		t.Fatalf("unchanged adr-0001 was rewritten (generated.at must be preserved)")
	}
	if !strings.Contains(string(adr1After), "at: 2026-09-29T14:00:00Z") {
		t.Errorf("adr-0001 generated.at not preserved:\n%s", adr1After)
	}

	// Changed ADR-0002: new content, new generated.at.
	adr2After, err := os.ReadFile(adr2Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(adr2After), "now amended text") {
		t.Errorf("adr-0002 missing updated body:\n%s", adr2After)
	}
	if !strings.Contains(string(adr2After), "at: 2026-09-30T14:00:00Z") {
		t.Errorf("adr-0002 generated.at not updated:\n%s", adr2After)
	}
}

func TestCompileStableOrder(t *testing.T) {
	mk := func() string {
		proj := t.TempDir()
		// Write in deliberately scrambled order; the compile must not
		// depend on insertion order.
		writeFixtureFile(t, proj, ".skillgrid/ASSUMPTIONS.md", `# ASSUMPTIONS

### In-force set

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0010 | Zulu ADR | accepted | — | — | 2026-09-01 | yes |
| 0002 | Alpha ADR | accepted | — | — | 2026-09-02 | yes |
| 0007 | Mike ADR | accepted | — | — | 2026-09-03 | yes |

### Locked constraints

- Second constraint.
- First constraint.
`)
		writeFixtureFile(t, proj, ".skillgrid/state.yaml", "schema: skillgrid/state/v1\n")
		return proj
	}

	var got1, got2 []string
	for _, run := range []int{1, 2} {
		proj := mk()
		out := filepath.Join(proj, ".wiki")
		if _, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow}); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		_, paths := readAllFiles(t, out)
		// Keep the order the walk produces (lexicographic on path).
		gotN := paths
		if run == 1 {
			got1 = gotN
		} else {
			got2 = gotN
		}
	}
	if strings.Join(got1, "\n") != strings.Join(got2, "\n") {
		t.Fatalf("compile order not deterministic:\n%v\nvs\n%v", got1, got2)
	}

	// index.md lists types alphabetically and ADRs in slug order.
	proj := mk()
	out := filepath.Join(proj, ".wiki")
	if _, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow}); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(out, "wiki/index.md"))
	if err != nil {
		t.Fatal(err)
	}
	adrIdx := strings.Index(string(idx), "## adr")
	conIdx := strings.Index(string(idx), "## concepts")
	entIdx := strings.Index(string(idx), "## entities")
	if !(adrIdx < conIdx && conIdx < entIdx) {
		t.Fatalf("index.md type order not alphabetical:\n%s", idx)
	}
	alpha := strings.Index(string(idx), "[[adr-0002]]")
	mike := strings.Index(string(idx), "[[adr-0007]]")
	zulu := strings.Index(string(idx), "[[adr-0010]]")
	if !(alpha < mike && mike < zulu) {
		t.Fatalf("index.md ADR order not by slug:\n%s", idx)
	}
}

func TestCompileMissingSources(t *testing.T) {
	proj := t.TempDir() // no .skillgrid at all
	out := filepath.Join(proj, ".wiki")
	res, err := Compile(CompileInput{ProjectDir: proj, OutDir: out, Now: compileNow})
	if err != nil {
		t.Fatalf("Compile with missing sources: %v", err)
	}
	// Still a valid bundle: AGENTS.md + index + log + manifest, no pages.
	if _, err := os.Stat(filepath.Join(out, "AGENTS.md")); err != nil {
		t.Errorf("missing AGENTS.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "wiki/index.md")); err != nil {
		t.Errorf("missing index.md: %v", err)
	}
	if res.Written < 3 {
		t.Errorf("expected at least 3 files written, got %d", res.Written)
	}
}

// helpers

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func hasPrefixKey(m map[string]string, prefix string) bool {
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
