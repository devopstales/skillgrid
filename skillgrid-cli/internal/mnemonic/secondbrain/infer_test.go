package secondbrain

import (
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// TestInferType is the deterministic type-taxonomy floor for mem_save.infer.
// The fixtures key on the type *words* (decision / bug / config / discovery)
// because InferType reuses the existing observation-type heuristic
// (memory.ShapePassiveItem) — the same classifier mem_capture_passive uses —
// rather than a new content-verb switch. Every result is a valid type.
func TestInferType(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"we decided to use SQLite for the store", "decision"},
		{"fixed the N+1 query in user list", "bugfix"},
		{"discovered vec0 requires cgo flag", "discovery"},
		{"config: set MNEMONIC_EMBED=1", "config"},
		{"architecture: hexagonal layout for the store", "architecture"},
		{"pattern: container-presentational split", "pattern"},
		{"no clear type signal here", "learning"},
	}
	for _, c := range cases {
		if got := InferType(c.text); got != c.want {
			t.Errorf("InferType(%q) = %q, want %q", c.text, got, c.want)
		}
		if !memory.IsValidType(InferType(c.text)) {
			t.Errorf("InferType(%q) = %q is not a valid type", c.text, InferType(c.text))
		}
	}
}

// TestInferTopicKey is a thin delegation check: the key must come from the
// existing SuggestTopicKey seam (family/segment) so a rephrased capture keeps
// one stable key across saves.
func TestInferTopicKey(t *testing.T) {
	got := InferTopicKey("decision", "Use SQLite")
	if got == "" {
		t.Fatal("InferTopicKey returned empty")
	}
	if want := "decision/use-sqlite"; got != want {
		t.Errorf("InferTopicKey = %q, want %q", got, want)
	}
}

func TestApplyInfer_FillsEmpty(t *testing.T) {
	in := &memory.SaveInput{Content: "we decided to use SQLite", Title: "Use SQLite"}
	changed := ApplyInfer(in, InferType, InferTopicKey)
	if !changed {
		t.Error("expected changed=true")
	}
	if in.Type == "" {
		t.Error("expected type to be filled")
	}
	if in.TopicKey == "" {
		t.Error("expected topic_key to be filled")
	}
}

func TestApplyInfer_PreservesProvided(t *testing.T) {
	in := &memory.SaveInput{
		Content:  "we decided to use SQLite",
		Title:    "Use SQLite",
		Type:     "architecture",
		TopicKey: "architecture/store",
	}
	changed := ApplyInfer(in, InferType, InferTopicKey)
	if in.Type != "architecture" {
		t.Errorf("type overwritten: %q", in.Type)
	}
	if in.TopicKey != "architecture/store" {
		t.Errorf("topic_key overwritten: %q", in.TopicKey)
	}
	if changed {
		t.Error("expected changed=false when both fields already set")
	}
}

// TestApplyInfer_FillsOnlyEmpty covers the asymmetric case: one field set,
// the other empty — only the empty one is filled, the provided one is kept.
func TestApplyInfer_FillsOnlyEmpty(t *testing.T) {
	in := &memory.SaveInput{
		Content:  "we decided to use SQLite",
		Title:    "Use SQLite",
		Type:     "decision",
		TopicKey: "",
	}
	changed := ApplyInfer(in, InferType, InferTopicKey)
	if !changed {
		t.Error("expected changed=true (topic_key empty)")
	}
	if in.Type != "decision" {
		t.Errorf("provided type overwritten: %q", in.Type)
	}
	if in.TopicKey == "" {
		t.Error("expected topic_key to be filled")
	}
}
