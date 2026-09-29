package wiki

import (
	"strings"
	"testing"
	"time"
)

// ---------- RenderIndex ----------

func TestRenderIndex(t *testing.T) {
	concepts := []Concept{
		{ID: "adr-0002", Type: "ADR", Title: "Second ADR", Status: "deprecated"},
		{ID: "adr-0001", Type: "ADR", Title: "First ADR", Status: "stable"},
		{ID: "project-state", Type: "State", Title: "Project State", Status: "draft"},
		{ID: "locked-constraint-1", Type: "Constraint", Title: "Go 1.22+ minimum to build.", Status: "stable"},
		{ID: "adr-0003", Type: "ADR", Title: "Third ADR", Status: "stable"},
	}
	got := RenderIndex(concepts)

	// The page starts with minimal OKF frontmatter, then the title.
	if !strings.HasPrefix(got, "---\n") {
		t.Fatalf("index must start with frontmatter, got: %q", firstLine(got))
	}
	if !strings.Contains(got, "# Wiki Index\n") {
		t.Fatalf("missing title line, got:\n%s", got)
	}

	// Type groups, alphabetically: adr < concepts < entities.
	adrIdx := strings.Index(got, "## adr")
	conIdx := strings.Index(got, "## concepts")
	entIdx := strings.Index(got, "## entities")
	if adrIdx < 0 || conIdx < 0 || entIdx < 0 {
		t.Fatalf("missing type group headings: adr=%d concepts=%d entities=%d", adrIdx, conIdx, entIdx)
	}
	if !(adrIdx < conIdx && conIdx < entIdx) {
		t.Fatalf("type groups out of order: adr=%d concepts=%d entities=%d", adrIdx, conIdx, entIdx)
	}

	// Concepts within a group sorted by slug (ID).
	first := strings.Index(got, "[[adr-0001]]")
	second := strings.Index(got, "[[adr-0002]]")
	third := strings.Index(got, "[[adr-0003]]")
	if first < 0 || second < 0 || third < 0 {
		t.Fatalf("missing concept links in adr group:\n%s", got)
	}
	if !(first < second && second < third) {
		t.Fatalf("adr links out of slug order: first=%d second=%d third=%d", first, second, third)
	}

	// Line format: "- [[slug]] — title (status)".
	line := "- [[adr-0002]] — Second ADR (deprecated)"
	if !strings.Contains(got, line) {
		t.Fatalf("missing formatted line %q in:\n%s", line, got)
	}
}

func TestRenderIndexEmpty(t *testing.T) {
	got := RenderIndex(nil)
	if !strings.Contains(got, "# Wiki Index\n") {
		t.Fatalf("missing title line, got:\n%s", got)
	}
	if strings.Contains(got, "## ") {
		t.Fatalf("no type groups expected for empty input:\n%s", got)
	}
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ---------- RenderAgents ----------

func TestRenderAgents(t *testing.T) {
	got := RenderAgents()

	if !strings.HasPrefix(got, "# AGENTS\n") {
		t.Fatalf("missing title line, got: %q", firstLine(got))
	}

	// Type taxonomy: every valid type must be named.
	for _, typ := range []string{"ADR", "Term", "Constraint", "Spec", "Spike", "State", "Architecture", "Finding", "Source"} {
		if !strings.Contains(got, typ) {
			t.Errorf("type taxonomy missing %q", typ)
		}
	}

	// Slug rule: lowercase-hyphen.
	if !strings.Contains(got, "lowercase") || !strings.Contains(got, "hyphen") {
		t.Errorf("slug rule (lowercase-hyphen) not documented:\n%s", got)
	}

	// Lint rule: must pass Conform.
	if !strings.Contains(got, "Conform") {
		t.Errorf("lint rule (Conform) not documented:\n%s", got)
	}

	// Determinism: two calls produce identical output.
	if got != RenderAgents() {
		t.Fatalf("RenderAgents is not deterministic")
	}
}

// ---------- AppendLog ----------

func TestAppendLog(t *testing.T) {
	now := time.Date(2026, 9, 29, 14, 0, 0, 0, time.UTC)

	// No changes → existing returned unchanged (nil-safe too).
	if got := AppendLog("existing", now, nil); got != "existing" {
		t.Fatalf("no-change append must return existing unchanged, got %q", got)
	}
	if got := AppendLog("", now, []string{}); got != "" {
		t.Fatalf("no-change append must return empty unchanged, got %q", got)
	}

	// Change on empty → frontmatter + "# Log".
	changes := []string{"add adr-0001", "change adr-0002"}
	got := AppendLog("", now, changes)
	if !strings.HasPrefix(got, logFrontmatter) {
		t.Fatalf("empty log must start with the log frontmatter, got: %q", firstLine(got))
	}
	if !strings.Contains(got, "# Log\n") {
		t.Fatalf("empty log must contain # Log, got:\n%s", got)
	}
	header := "## 2026-09-29 — 2 change(s)"
	if !strings.Contains(got, header) {
		t.Fatalf("missing header %q in:\n%s", header, got)
	}
	if !strings.Contains(got, "add adr-0001") || !strings.Contains(got, "change adr-0002") {
		t.Fatalf("missing change lines in:\n%s", got)
	}

	// Second change appends below the first entry.
	now2 := now.Add(24 * time.Hour)
	got2 := AppendLog(got, now2, []string{"add adr-0003"})
	if !strings.HasPrefix(got2, got) {
		t.Fatalf("append must preserve the existing log prefix")
	}
	if strings.Count(got2, "## 2026-") != 2 {
		t.Fatalf("expected 2 dated entries, got:\n%s", got2)
	}
	if !strings.Contains(got2, "## 2026-09-30 — 1 change(s)") {
		t.Fatalf("missing second entry header in:\n%s", got2)
	}
}
