package wiki

import (
	"testing"
	"time"
)

func TestConceptFields(t *testing.T) {
	// Verify the Concept struct has all the documented fields and that the
	// types line up (a compile-time guard: a missing field or wrong type
	// here is a compile error).
	c := Concept{
		ID:          "my-concept",
		Type:        "ADR",
		Title:       "My Concept",
		Description: "A description",
		Resource:    "repo/origin",
		Tags:        []string{"a", "b"},
		Sources: []SourceRef{
			{
				ID:           "src-1",
				Resource:     "repo/origin",
				Title:        "Source Doc",
				Author:       "author",
				UsageCount:   3,
				LastModified: time.Now().UTC(),
			},
		},
		Generated:  Verifier{By: "agent-x", At: time.Now().UTC()},
		Verified:   []Verifier{{By: "human-1", At: time.Now().UTC()}},
		Status:     "active",
		StaleAfter: time.Now().UTC().Add(24 * time.Hour),
		SourcePath: "docs/adr/0001-foo.md",
		Body:       "# My Concept\n\nBody content.",
	}

	if c.ID != "my-concept" {
		t.Fatalf("ID = %q, want %q", c.ID, "my-concept")
	}
	if c.Type != "ADR" {
		t.Fatalf("Type = %q, want %q", c.Type, "ADR")
	}
	if len(c.Tags) != 2 || c.Tags[0] != "a" || c.Tags[1] != "b" {
		t.Fatalf("Tags = %v, want [a b]", c.Tags)
	}
	if len(c.Sources) != 1 {
		t.Fatalf("Sources len = %d, want 1", len(c.Sources))
	}
	src := c.Sources[0]
	if src.ID != "src-1" || src.UsageCount != 3 {
		t.Fatalf("Sources[0] = %+v, want ID=src-1 UsageCount=3", src)
	}
	if src.LastModified.IsZero() {
		t.Fatalf("Sources[0].LastModified = %v, want non-zero (set to time.Now())", src.LastModified)
	}
	if c.Generated.By != "agent-x" {
		t.Fatalf("Generated.By = %q, want agent-x", c.Generated.By)
	}
	if len(c.Verified) != 1 || c.Verified[0].By != "human-1" {
		t.Fatalf("Verified = %v, want one entry By=human-1", c.Verified)
	}
	if c.SourcePath != "docs/adr/0001-foo.md" {
		t.Fatalf("SourcePath = %q, want docs/adr/0001-foo.md", c.SourcePath)
	}
}

func TestSourceRefZeroLastModified(t *testing.T) {
	// A SourceRef with no LastModified set has a zero time (absent).
	var s SourceRef
	if !s.LastModified.IsZero() {
		t.Fatalf("zero SourceRef.LastModified = %v, want zero", s.LastModified)
	}
	if s.UsageCount != 0 {
		t.Fatalf("zero SourceRef.UsageCount = %d, want 0", s.UsageCount)
	}
}

func TestEdgeFields(t *testing.T) {
	e := Edge{
		From:       "concept-a",
		To:         "concept-b",
		Kind:       "references",
		Confidence: ConfidenceExtracted,
	}
	if e.From != "concept-a" || e.To != "concept-b" || e.Kind != "references" {
		t.Fatalf("Edge = %+v", e)
	}
	if e.Confidence != "EXTRACTED" {
		t.Fatalf("Confidence = %q, want EXTRACTED", e.Confidence)
	}
}

func TestConfidenceConstants(t *testing.T) {
	if string(ConfidenceExtracted) != "EXTRACTED" {
		t.Fatalf("ConfidenceExtracted = %q, want EXTRACTED", ConfidenceExtracted)
	}
	if string(ConfidenceInferred) != "INFERRED" {
		t.Fatalf("ConfidenceInferred = %q, want INFERRED", ConfidenceInferred)
	}
	if string(ConfidenceAmbiguous) != "AMBIGUOUS" {
		t.Fatalf("ConfidenceAmbiguous = %q, want AMBIGUOUS", ConfidenceAmbiguous)
	}
}
