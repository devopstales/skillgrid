package handler

// explore_format_test.go — the TICKET-05 code_explore composite: the
// CodeGraph baseline shape (header → counts → blast radius → source
// blocks), the EXTRACTED/INFERRED edge tags, the fusion-leg provenance,
// the degraded line (a failed leg names itself; no_vector is a
// deployment state, never a line), and the silent empty index.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// TestExploreResponseShape pins the baseline layout: every section in
// order, the per-symbol blast-radius line, the line-numbered source,
// the fusion-leg provenance, no duplicate edge lines. (Mutation gate:
// dropping the blast-radius section from the renderer fails this test —
// the header + per-symbol line are asserted literally.)
func TestExploreResponseShape(t *testing.T) {
	ch, _ := seedSearchGraph(t, []int{2}, []int{2})
	out, err := ch.Explore(ctx, "add")
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	// The sections, in order.
	order := []string{
		"**Exploration: add**",
		"Found 2 symbols across 2 files.",
		"**Blast radius — what depends on these (update/verify before editing)**",
		"**Source Code**",
	}
	last := -1
	for _, s := range order {
		i := strings.Index(out, s)
		if i < 0 {
			t.Fatalf("output missing %q:\n%s", s, out)
		}
		if i < last {
			t.Fatalf("layout order violated at %q:\n%s", s, out)
		}
		last = i
	}
	// The per-symbol blast-radius line (the TICKET-04 impact join):
	// file:line — N callers; tests: <files> — + the fusion-leg
	// provenance (the FTS leg's 1-based rank).
	wantLine := "- `add` (math.go:3) — 2 callers in `main.go`, `math_test.go`; tests: `math_test.go` ["
	if !strings.Contains(out, wantLine) {
		t.Errorf("missing blast-radius line %q:\n%s", wantLine, out)
	}
	if !strings.Contains(out, "fts:") {
		t.Errorf("missing fusion-leg provenance (Hit.Provenance):\n%s", out)
	}
	// The provenance note (the source's provenance claim).
	if !strings.Contains(out, "> The code below is the **verbatim, current on-disk source** of these files") {
		t.Errorf("missing provenance note:\n%s", out)
	}
	// The line-numbered verbatim source (the read path's output).
	if !strings.Contains(out, "3\tfunc add(a, b int) int { return a + b }") {
		t.Errorf("missing line-numbered source:\n%s", out)
	}
	if !strings.Contains(out, "2\tfunc TestAdd(t *testing.T) { add(1, 2) }") {
		t.Errorf("missing test-file source:\n%s", out)
	}
	// The edge-summary line (the math_test.go block: TestAdd → add).
	if !strings.Contains(out, "**`math_test.go`** — add(calls)") {
		t.Errorf("missing edge-summary line:\n%s", out)
	}
	// No duplicate edge lines: the summary entries are distinct.
	if i := strings.Index(out, "**`math_test.go`** — "); i >= 0 {
		line := out[i : i+strings.Index(out[i:], "\n")]
		entries := strings.Split(strings.TrimPrefix(line, "**`math_test.go`** — "), ", ")
		seen := map[string]bool{}
		for _, e := range entries {
			if seen[e] {
				t.Errorf("duplicate edge line: %q", e)
			}
			seen[e] = true
		}
	}
}

// TestExploreTagsInferredEdges pins the provenance tags: an INFERRED
// edge renders tagged (inferred), an EXTRACTED edge stays untagged.
func TestExploreTagsInferredEdges(t *testing.T) {
	ch, addID := seedSearchGraph(t, []int{2}, []int{2})
	st := ch.Store()
	// Re-seed the math_test.go edge (TestAdd → add) as INFERRED (the
	// replace drops the EXTRACTED row for that file).
	symTest, err := st.SymbolByName(ctx, "TestAdd")
	if err != nil || len(symTest) != 1 {
		t.Fatalf("resolve TestAdd: %v", err)
	}
	fTest, err := st.FileByPath(ctx, "math_test.go")
	if err != nil {
		t.Fatalf("file math_test.go: %v", err)
	}
	toAdd := addID
	if err := st.ReplaceEdgesForFile(ctx, fTest.ID, []store.Edge{{
		Kind: "calls", FromID: symTest[0].ID, ToID: &toAdd, Confidence: "INFERRED", Line: 3,
	}}); err != nil {
		t.Fatalf("re-seed edge: %v", err)
	}
	out, err := ch.Explore(ctx, "add")
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if !strings.Contains(out, "add(calls) (inferred)") {
		t.Errorf("INFERRED edge not tagged (inferred):\n%s", out)
	}
	// The tag appears exactly once (the one INFERRED edge) and no
	// EXTRACTED edge is tagged.
	if n := strings.Count(out, "(inferred)"); n != 1 {
		t.Errorf("(inferred) tag count = %d, want 1:\n%s", n, out)
	}
}

// TestExploreEmptyIndex pins the silent empty index: the header + the
// 0/0 counts line, no source blocks, no degraded line (no_vector is a
// deployment state), no error.
func TestExploreEmptyIndex(t *testing.T) {
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	ch := NewCodeHandler(st, nil, c, "", fixedDetector{})
	t.Chdir(t.TempDir()) // non-git dir, no files
	out, err := ch.Explore(ctx, "nope")
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if !strings.Contains(out, "Found 0 symbols across 0 files.") {
		t.Errorf("missing 0/0 counts line:\n%s", out)
	}
	if strings.Contains(out, "**Source Code**") || strings.Contains(out, "**Blast radius") {
		t.Errorf("empty index rendered a content section:\n%s", out)
	}
	if strings.Contains(out, "degraded:") {
		t.Errorf("empty index rendered a degraded line (no_vector is suppressed):\n%s", out)
	}
}

// failEmbedder is the embedder stub that errors on Embed (the
// semantic-leg degradation fixture: Model() non-empty so the leg runs,
// then fails → semantic_error).
type failEmbedder struct{}

func (failEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, errors.New("embedder: test failure")
}
func (failEmbedder) Model() string   { return "fail-embedder" }
func (failEmbedder) Dimensions() int { return 8 }

// TestExploreDegradedLeg pins "one degraded leg: the flag names the
// leg": a failing embedder degrades the semantic leg — the output's
// degraded line names it, and no_vector stays suppressed.
func TestExploreDegradedLeg(t *testing.T) {
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	ch := NewCodeHandler(st, failEmbedder{}, c, "", fixedDetector{})
	t.Chdir(t.TempDir())
	out, err := ch.Explore(ctx, "nope")
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	i := strings.Index(out, "degraded:")
	if i < 0 {
		t.Fatalf("no degraded line (the semantic leg failed):\n%s", out)
	}
	line := out[i : i+strings.Index(out[i:], "\n")]
	if !strings.Contains(line, "semantic_error") {
		t.Errorf("degraded line does not name the leg: %q", line)
	}
	if strings.Contains(line, "no_vector") {
		t.Errorf("no_vector rendered (a deployment state, not a degraded leg): %q", line)
	}
}

// TestExploreFenceEscapesLongBacktickRuns pins the fence-escape fix
// (security review 2026-11-09, Low): verbatim content containing a run
// of 3+ backticks must not terminate the code fence early — the fence
// is sized to the longest backtick run in the block's span text + 1, so
// the content stays inside the fence instead of rendering as prose.
func TestExploreFenceEscapesLongBacktickRuns(t *testing.T) {
	out := RenderExplore(ExploreResult{
		Query: "probe",
		Blocks: []ExploreFileBlock{{
			Path:     "doc.md",
			Language: "go",
			Spans: []SourceSpan{{
				Start: 1,
				Text:  "before\nx := `a````b` // four backticks inside a raw string\nafter",
			}},
		}},
	})
	if !strings.Contains(out, "`````go\n") {
		t.Fatalf("expected a 5-backtick opening fence for a 4-backtick content run, got:\n%s", out)
	}
	if !strings.Contains(out, "\n`````\n") {
		t.Fatalf("expected the block to close with a 5-backtick fence, got:\n%s", out)
	}
	if strings.Contains(out, "\n```\n") || strings.Contains(out, "\n```go\n") {
		t.Fatalf("a triple-backtick fence still appears (would end at the content run):\n%s", out)
	}
	if !strings.Contains(out, "2\tx := `a````b` // four backticks inside a raw string") {
		t.Fatalf("content line missing or malformed, got:\n%s", out)
	}
}
