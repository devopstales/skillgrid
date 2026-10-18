package secondbrain

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// capture_test.go — TICKET-05 (2026-10-06-second-brain-learnings Task 5):
// the deterministic two-pass capture classifier (ClassifyCapture) + the
// confidence-gated suggest floor (SuggestCapture, the pure projection
// mem_capture_suggest renders — the MCP handler test in internal/mcp asserts
// the tool surface). SATISFIES @step-05 confidence-capture (G6).
//
// The classifier is pure: same input, same output, every run — no LLM, no
// time, no randomness, no store. Strong patterns fire on their own; weak
// patterns fire only when a directive word is present; negative patterns
// suppress everything.

// TestClassifyCapture_HighConfidence: "We're switching from Bootstrap to
// Tailwind" fires the named switching trigger at/above the 0.75 gate.
func TestClassifyCapture_HighConfidence(t *testing.T) {
	trigger, conf := ClassifyCapture("We're switching from Bootstrap to Tailwind")
	if trigger != "switching" {
		t.Errorf("trigger = %q, want %q", trigger, "switching")
	}
	if conf < CaptureThreshold {
		t.Errorf("confidence = %v, want >= %v", conf, CaptureThreshold)
	}
}

// TestClassifyCapture_StrongPatterns: every STRONG pattern fires on its own,
// at/above the gate, named by its stable trigger key.
func TestClassifyCapture_StrongPatterns(t *testing.T) {
	cases := []struct {
		text    string
		trigger string
	}{
		{"don't do that", "correction"},
		{"we decided to use sqlite", "decided"},
		{"the rule is kebab-case naming", "rule"},
		{"always use tabs in this repo", "always"},
		{"never use var in TS files", "never"},
		{"the convention is to prefix tests with test", "convention"},
	}
	for _, c := range cases {
		trigger, conf := ClassifyCapture(c.text)
		if trigger != c.trigger {
			t.Errorf("ClassifyCapture(%q) trigger = %q, want %q", c.text, trigger, c.trigger)
		}
		if conf < CaptureThreshold {
			t.Errorf("ClassifyCapture(%q) confidence = %v, want >= %v", c.text, conf, CaptureThreshold)
		}
	}
}

// TestClassifyCapture_WeakAlone: a bare weak pattern ("no") carries no
// directive word, so it does not fire — the weak floor (0.75) is not met.
func TestClassifyCapture_WeakAlone(t *testing.T) {
	if _, conf := ClassifyCapture("no"); conf >= CaptureThreshold {
		t.Errorf("weak alone confidence = %v, want < %v", conf, CaptureThreshold)
	}
}

// TestClassifyCapture_WeakWithDirective: "no" + a directive word ("use X")
// fires the weak-correction trigger at/above the gate.
func TestClassifyCapture_WeakWithDirective(t *testing.T) {
	trigger, conf := ClassifyCapture("no, use pnpm")
	if trigger != "weak" {
		t.Errorf("trigger = %q, want %q", trigger, "weak")
	}
	if conf < CaptureThreshold {
		t.Errorf("confidence = %v, want >= %v", conf, CaptureThreshold)
	}
}

// TestClassifyCapture_Negative: a negative pattern suppresses the suggestion
// even when a weak/strong pattern is also present.
func TestClassifyCapture_Negative(t *testing.T) {
	for _, text := range []string{
		"no worries",
		"no problem",
		"nope",
		"no, never use that, no problem", // negative beats strong
	} {
		if _, conf := ClassifyCapture(text); conf >= CaptureThreshold {
			t.Errorf("ClassifyCapture(%q) confidence = %v, want < %v (suppressed)", text, conf, CaptureThreshold)
		}
	}
}

// TestClassifyCapture_BelowThreshold: casual text with no pattern fires no
// trigger at all.
func TestClassifyCapture_BelowThreshold(t *testing.T) {
	trigger, conf := ClassifyCapture("hey what's up")
	if trigger != "" {
		t.Errorf("trigger = %q, want empty", trigger)
	}
	if conf >= CaptureThreshold {
		t.Errorf("confidence = %v, want < %v", conf, CaptureThreshold)
	}
}

// TestClassifyCapture_Deterministic: the same input yields byte-identical
// output across repeated calls (no LLM, no time, no randomness).
func TestClassifyCapture_Deterministic(t *testing.T) {
	const text = "We're switching from Bootstrap to Tailwind, the convention is kebab-case"
	first, firstConf := ClassifyCapture(text)
	for i := 0; i < 3; i++ {
		got, gotConf := ClassifyCapture(text)
		if got != first || gotConf != firstConf {
			t.Fatalf("call %d drifted: (%q, %v) vs first (%q, %v)", i, got, gotConf, first, firstConf)
		}
	}
}

// TestClassifyCapture_Position: an earlier (closer-to-front) pattern match
// scores higher than the same pattern buried later — the position component
// of the keyword+position score.
func TestClassifyCapture_Position(t *testing.T) {
	_, early := ClassifyCapture("we decided to use sqlite, and much more prose follows")
	_, late := ClassifyCapture("much prose follows before the sentence that says we decided to use sqlite")
	if !(early > late) {
		t.Errorf("position score missing: early %v, late %v (want early > late)", early, late)
	}
}

// TestClassifyCapture_TriggerToClass: the trigger's inferred observation type
// is a valid type, and its suggested knowledge_class matches the ADR-0046
// type→class heuristic mem_save applies (the confirm path reuses it).
func TestClassifyCapture_TriggerToClass(t *testing.T) {
	if !memory.IsValidType(TriggerType("switching")) {
		t.Errorf("TriggerType(switching) = %q is not a valid type", TriggerType("switching"))
	}
	if got := SuggestedKnowledgeClass("decided"); got != "convention" {
		t.Errorf("SuggestedKnowledgeClass(decided) = %q, want %q", got, "convention")
	}
	if got := SuggestedKnowledgeClass("switching"); got != "convention" {
		t.Errorf("SuggestedKnowledgeClass(switching) = %q, want %q", got, "convention")
	}
}

// TestSuggestCapture_Gated: the 0.75 gate decides suggested; the draft is
// present only above it and carries trigger + confidence + the suggested
// knowledge_class.
func TestSuggestCapture_Gated(t *testing.T) {
	out := SuggestCapture("We're switching from Bootstrap to Tailwind")
	if !out.Suggested {
		t.Fatal("expected suggested=true for high-confidence text")
	}
	if out.Draft.Trigger != "switching" {
		t.Errorf("draft trigger = %q, want %q", out.Draft.Trigger, "switching")
	}
	if out.Draft.Confidence < CaptureThreshold {
		t.Errorf("draft confidence = %v, want >= %v", out.Draft.Confidence, CaptureThreshold)
	}
	if out.Draft.KnowledgeClass != "convention" {
		t.Errorf("draft knowledge_class = %q, want %q", out.Draft.KnowledgeClass, "convention")
	}

	if out = SuggestCapture("hey what's up"); out.Suggested {
		t.Error("expected suggested=false below threshold")
	}
}
