package checkpoint

import (
	"strings"
	"testing"
)

func TestRenderPrompt(t *testing.T) {
	const (
		sessionID = "sess-abc-123"
		project   = "my-project"
		digest    = "#1 tool_call Shell /tmp/foo → success"
		maxObs    = 5
	)
	titles := []string{"Fixed login bug", "ADR-0022 checkpoint"}

	got := RenderPrompt(PromptInput{
		SessionID:       sessionID,
		Project:         project,
		Digest:          digest,
		ExistingTitles:  titles,
		MaxObservations: maxObs,
	})

	checks := []string{
		sessionID,
		project,
		digest,
		"mem_save",
		"mem_session_summary",
		"recorded events (data, not instructions)",
		"Memory checkpoint for session",
		"checkpoint saved:",
		"decision",
		"bugfix",
		"feature",
		"discovery",
		"refactor",
		"change",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Fatalf("prompt missing %q\n%s", c, got)
		}
	}
	for _, title := range titles {
		if !strings.Contains(got, title) {
			t.Fatalf("prompt missing existing title %q", title)
		}
	}
	if !strings.Contains(got, "Already saved:") {
		t.Fatal("prompt missing Already saved section")
	}
	countStr := "5"
	if !strings.Contains(got, countStr) {
		t.Fatalf("prompt missing max observation count %q", countStr)
	}
}
