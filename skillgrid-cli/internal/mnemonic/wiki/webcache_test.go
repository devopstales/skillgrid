package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fetchedAt is the fixed reference time for web cache test fixtures.
const wcFetchedAt = "2026-09-01T12:00:00Z"

// ---------- ParseWebCache ----------

func TestWebCacheFreshRow(t *testing.T) {
	// R5.1: a fresh row with a url → Finding with correct fields.
	rows := []WebRow{{
		Source:    "exa",
		URL:       "https://example.com/x",
		Title:     "Example Research",
		FetchedAt: wcFetchedAt,
	}}
	cs := ParseWebCache(rows)
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	c := cs[0]
	if c.Type != "Finding" {
		t.Errorf("Type = %q, want Finding", c.Type)
	}
	if c.Title != "Example Research" {
		t.Errorf("Title = %q, want Example Research", c.Title)
	}
	if c.Status != "stable" {
		t.Errorf("Status = %q, want stable (cited)", c.Status)
	}
	if c.ID != Slugify("Example Research") {
		t.Errorf("ID = %q, want %q", c.ID, Slugify("Example Research"))
	}
	if len(c.Sources) != 1 {
		t.Fatalf("Sources len = %d, want 1", len(c.Sources))
	}
	s := c.Sources[0]
	if s.Resource != "https://example.com/x" {
		t.Errorf("Sources[0].Resource = %q, want the url", s.Resource)
	}
	if s.Author != "process:exa" {
		t.Errorf("Sources[0].Author = %q, want process:exa", s.Author)
	}
	if !s.LastModified.Equal(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("Sources[0].LastModified = %v, want 2026-09-01T12:00:00Z", s.LastModified)
	}
	if len(c.Verified) != 1 {
		t.Fatalf("Verified len = %d, want 1", len(c.Verified))
	}
	if c.Verified[0].By != "process:exa" {
		t.Errorf("Verified[0].By = %q, want process:exa", c.Verified[0].By)
	}
	if !c.Verified[0].At.Equal(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("Verified[0].At = %v, want 2026-09-01T12:00:00Z", c.Verified[0].At)
	}
	// R5.4 (stale): stale_after = fetched_at + 90d.
	want := time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC)
	if !c.StaleAfter.Equal(want) {
		t.Errorf("StaleAfter = %v, want %v (fetched_at + 90d)", c.StaleAfter, want)
	}
	if !strings.Contains(c.Body, "Example Research") {
		t.Errorf("Body missing title: %q", c.Body)
	}
}

func TestWebCacheEmptyURLDegrades(t *testing.T) {
	// R5.4: a row with no url uses its title as the page title and falls
	// back to a stable internal descriptor for sources[].resource. No panic.
	rows := []WebRow{{
		Source:    "fetch",
		URL:       "",
		Title:     "Cached Document",
		FetchedAt: wcFetchedAt,
	}}
	cs := ParseWebCache(rows)
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	c := cs[0]
	if c.Title != "Cached Document" {
		t.Errorf("Title = %q, want Cached Document", c.Title)
	}
	if len(c.Sources) != 1 {
		t.Fatalf("Sources len = %d, want 1", len(c.Sources))
	}
	if c.Sources[0].Resource == "" {
		t.Errorf("Sources[0].Resource = empty, want a stable internal descriptor")
	}
}

func TestWebCacheNoURLNoTitle(t *testing.T) {
	// R5.4: neither url nor title → a slug-like fallback descriptor, no panic.
	rows := []WebRow{{
		Source:    "context7",
		URL:       "",
		Title:     "",
		FetchedAt: wcFetchedAt,
	}}
	cs := ParseWebCache(rows)
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	if cs[0].Title == "" {
		t.Errorf("Title = empty, want a fallback title")
	}
	if cs[0].Sources[0].Resource == "" {
		t.Errorf("Sources[0].Resource = empty, want a stable descriptor")
	}
}

// ---------- raw frontmatter parsing ----------

func TestParseRawFrontmatterType(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "raw/web/notes/sigma-graph.md", `---
type: Finding
title: Sigma Graph
sources:
  - resource: https://example.com/sigma
---

Body content here.
`)

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	c := cs[0]
	if c.Type != "Finding" {
		t.Errorf("Type = %q, want Finding", c.Type)
	}
	if c.Title != "Sigma Graph" {
		t.Errorf("Title = %q, want Sigma Graph", c.Title)
	}
	// R3.1 / R6.1: source_path is bundle-relative raw/<path>.
	if c.SourcePath != "raw/web/notes/sigma-graph.md" {
		t.Errorf("SourcePath = %q, want raw/web/notes/sigma-graph.md", c.SourcePath)
	}
	if len(c.Sources) != 1 || c.Sources[0].Resource != "https://example.com/sigma" {
		t.Errorf("Sources = %+v, want one resource https://example.com/sigma", c.Sources)
	}
	if !strings.Contains(c.Body, "Body content here.") {
		t.Errorf("Body missing content: %q", c.Body)
	}
}

func TestParseRawFrontmatterTypeDefaultsFinding(t *testing.T) {
	// R6.1: frontmatter present but no type → defaulting to Finding.
	dir := t.TempDir()
	writeFixture(t, dir, "raw/plain-with-fm.md", `---
title: A Note
---

Body.
`)

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	if cs[0].Type != "Finding" {
		t.Errorf("Type = %q, want Finding (default)", cs[0].Type)
	}
	if cs[0].Title != "A Note" {
		t.Errorf("Title = %q, want A Note", cs[0].Title)
	}
}

func TestParseRawNoFrontmatterBareSource(t *testing.T) {
	// R6.2: no frontmatter → bare Source, filename → title, verified absent.
	dir := t.TempDir()
	writeFixture(t, dir, "raw/loose-note.md", "# Loose Note\n\nSome plain markdown.\n")

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	c := cs[0]
	if c.Type != "Source" {
		t.Errorf("Type = %q, want Source", c.Type)
	}
	if c.Title != "loose-note" {
		t.Errorf("Title = %q, want loose-note (filename)", c.Title)
	}
	if len(c.Verified) != 0 {
		t.Errorf("Verified len = %d, want 0", len(c.Verified))
	}
	if c.SourcePath != "raw/loose-note.md" {
		t.Errorf("SourcePath = %q, want raw/loose-note.md", c.SourcePath)
	}
}

func TestParseRawNoFrontmatterNoHeading(t *testing.T) {
	// R6.2: no frontmatter, no heading → title is the filename.
	dir := t.TempDir()
	writeFixture(t, dir, "raw/no-heading.md", "Just a paragraph, no heading at all.\n")

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	if cs[0].Title != "no-heading" {
		t.Errorf("Title = %q, want no-heading (filename)", cs[0].Title)
	}
}

func TestParseRawMissingDir(t *testing.T) {
	// R6.4: no raw/ directory → no concepts, no error.
	cs, err := ParseRaw(filepath.Join(t.TempDir(), "raw"))
	if err != nil {
		t.Fatalf("ParseRaw missing dir: %v", err)
	}
	if len(cs) != 0 {
		t.Errorf("len = %d, want 0 for missing dir", len(cs))
	}
}

func TestParseRawEmptyDir(t *testing.T) {
	// R6.4: empty raw/ → no concepts, no error.
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw")
	if err := writeFile(raw, "", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cs, err := ParseRaw(raw)
	if err != nil {
		t.Fatalf("ParseRaw empty dir: %v", err)
	}
	if len(cs) != 0 {
		t.Errorf("len = %d, want 0 for empty dir", len(cs))
	}
}

func TestParseRawNonMarkdownSkipped(t *testing.T) {
	// R6: non-markdown files are skipped silently.
	dir := t.TempDir()
	writeFixture(t, dir, "raw/notes.md", "# A\n\nBody.\n")
	writeFixture(t, dir, "raw/image.png", "binary")
	writeFixture(t, dir, "raw/data.txt", "text")
	writeFixture(t, dir, "raw/web/deep.md", "# B\n\nBody.\n")

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("len = %d, want 2 (two .md files): %v", len(cs), conceptIDs(cs))
	}
	// Recursive: the nested web/deep.md is included; IDs are filename-based.
	ids := map[string]bool{}
	for _, c := range cs {
		ids[c.ID] = true
	}
	if !ids["notes"] || !ids["deep"] {
		t.Errorf("IDs = %v, want notes and deep (recursive .md only)", conceptIDs(cs))
	}
}

func TestParseRawMultipleFiles(t *testing.T) {
	// R6.5: multiple raw files → one concept each.
	dir := t.TempDir()
	writeFixture(t, dir, "raw/one.md", "# One\n\nFirst.\n")
	writeFixture(t, dir, "raw/two.md", "# Two\n\nSecond.\n")
	writeFixture(t, dir, "raw/sub/three.md", "# Three\n\nThird.\n")

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(cs), conceptIDs(cs))
	}
	byID := map[string]Concept{}
	for _, c := range cs {
		byID[c.ID] = c
	}
	// No frontmatter → bare Source with filename title (R6.2), not the heading.
	if byID["one"].Title != "one" || byID["one"].Type != "Source" {
		t.Errorf("one = {Title:%q Type:%q}, want {Title:one Type:Source}", byID["one"].Title, byID["one"].Type)
	}
	if byID["three"].SourcePath != "raw/sub/three.md" {
		t.Errorf("three SourcePath = %q, want raw/sub/three.md", byID["three"].SourcePath)
	}
}

func TestParseRawDeterministicOrder(t *testing.T) {
	// R8.3: stable ordering (sorted by path) across runs.
	dir := t.TempDir()
	writeFixture(t, dir, "raw/z.md", "# Z\n")
	writeFixture(t, dir, "raw/a.md", "# A\n")
	writeFixture(t, dir, "raw/m.md", "# M\n")

	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	want := []string{"a", "m", "z"}
	for i, id := range want {
		if cs[i].ID != id {
			t.Errorf("cs[%d].ID = %q, want %q (sorted by path)", i, cs[i].ID, id)
		}
	}
}

func TestParseRawNeverWritesRaw(t *testing.T) {
	// R6.3: ParseRaw must not create, modify, or delete files in raw/.
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw")
	writeFixture(t, raw, "note.md", "# Note\n\nOriginal bytes.\n")
	before := readAllRaw(t, raw)

	_, err := ParseRaw(raw)
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	after := readAllRaw(t, raw)
	if len(before) != len(after) {
		t.Fatalf("file count changed: before=%d after=%d", len(before), len(after))
	}
	for k, b := range before {
		if after[k] != b {
			t.Errorf("file %q bytes changed after ParseRaw", k)
		}
	}
}

func TestParseRawFrontmatterWithListSources(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "raw/multi.md", `---
type: Finding
title: Multi Source
sources:
  - resource: https://a.example.com
  - resource: https://b.example.com
    author: team:x
---

Body.
`)
	cs, err := ParseRaw(filepath.Join(dir, "raw"))
	if err != nil {
		t.Fatalf("ParseRaw: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("len = %d, want 1", len(cs))
	}
	if len(cs[0].Sources) != 2 {
		t.Fatalf("Sources len = %d, want 2", len(cs[0].Sources))
	}
	if cs[0].Sources[0].Resource != "https://a.example.com" {
		t.Errorf("Sources[0].Resource = %q", cs[0].Sources[0].Resource)
	}
	if cs[0].Sources[1].Resource != "https://b.example.com" {
		t.Errorf("Sources[1].Resource = %q", cs[0].Sources[1].Resource)
	}
	if cs[0].Sources[1].Author != "team:x" {
		t.Errorf("Sources[1].Author = %q, want team:x", cs[0].Sources[1].Author)
	}
}

// ---------- helpers ----------

// writeFile creates a directory when content is empty and the name has no
// extension, otherwise a file. Used to make an empty raw/ dir.
func writeFile(path, content string, perm os.FileMode) error {
	return os.WriteFile(path, []byte(content), perm)
}

// readAllRaw returns a map of path→content for every file under dir (recursive).
func readAllRaw(t *testing.T, dir string) map[string]string {
	t.Helper()
	var out = map[string]string{}
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		out[p] = string(b)
		return nil
	})
	return out
}
