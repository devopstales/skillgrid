package memory

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestWorkingSetTracking (014 step 22.1): the working set tracks files edited
// during the session — 2 in-project files (with edit counts + net line
// deltas) are recorded, the file outside the project is excluded, and hub
// files are flagged.
func TestWorkingSetTracking(t *testing.T) {
	project := "/work/proj"
	ws := NewWorkingSet(project)

	// Two in-project files: one hub (AGENTS.md), one ordinary source file.
	ws.RecordEdit("/work/proj/AGENTS.md", 10, 2)
	ws.RecordEdit("/work/proj/AGENTS.md", 5, 1) // second edit to the same file
	ws.RecordEdit("/work/proj/src/auth.go", 30, 4)

	// One file OUTSIDE the project root: must be excluded.
	ws.RecordEdit("/etc/hosts", 100, 100)

	files := ws.Files()
	if len(files) != 2 {
		t.Fatalf("expected 2 in-project files, got %d: %+v", len(files), files)
	}

	byPath := map[string]WorkingFile{}
	for _, f := range files {
		byPath[f.Path] = f
	}

	// The outside file must be absent.
	if _, ok := byPath["/etc/hosts"]; ok {
		t.Errorf("file outside project (%s) was not excluded", "/etc/hosts")
	}

	// AGENTS.md: edit count = 2, net = (10-2)+(5-1) = 12, hub flagged.
	agents, ok := byPath["/work/proj/AGENTS.md"]
	if !ok {
		t.Fatalf("AGENTS.md missing from working set")
	}
	if agents.EditCount != 2 {
		t.Errorf("AGENTS.md EditCount = %d, want 2", agents.EditCount)
	}
	if agents.NetLines != 12 {
		t.Errorf("AGENTS.md NetLines = %d, want 12", agents.NetLines)
	}
	if !agents.IsHub {
		t.Errorf("AGENTS.md should be flagged as a hub file")
	}

	// src/auth.go: edit count = 1, net = 30-4 = 26, not a hub file.
	auth, ok := byPath["/work/proj/src/auth.go"]
	if !ok {
		t.Fatalf("src/auth.go missing from working set")
	}
	if auth.EditCount != 1 {
		t.Errorf("auth.go EditCount = %d, want 1", auth.EditCount)
	}
	if auth.NetLines != 26 {
		t.Errorf("auth.go NetLines = %d, want 26", auth.NetLines)
	}
	if auth.IsHub {
		t.Errorf("src/auth.go should NOT be flagged as a hub file")
	}
}

// TestIntentClassificationInEnvelope (014 step 22.2): the context envelope
// classifies work intents as exploration / debugging / review / refactor. It
// reuses the step-19 ClassifyIntent (the same 4 intents) so the envelope's
// intent is the canonical classification, not a duplicate.
func TestIntentClassificationInEnvelope(t *testing.T) {
	cases := []struct {
		query string
		want  Intent
	}{
		{"fix the null pointer in auth.go", IntentDebugging},
		{"what's in the config module", IntentExploration},
		{"check the PR for payment.go", IntentReview},
		{"extract the validation into a separate function", IntentRefactor},
		{"the server crashes on startup", IntentDebugging},
		{"list all the handlers", IntentExploration},
		{"rename the request builder", IntentRefactor},
		{"a brand new question with no markers", IntentExploration},
	}
	for _, c := range cases {
		if got := ClassifyWorkIntent(c.query); got != c.want {
			t.Errorf("ClassifyWorkIntent(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}

// TestContextEnvelopeSizeLimit (014 step 22.3, rollback boundary): an oversized
// working set is truncated at serialization so the envelope never exceeds the
// size cap.
func TestContextEnvelopeSizeLimit(t *testing.T) {
	project := "/work/proj"
	ws := NewWorkingSet(project)
	// A large number of files with large paths / net deltas pushes the full
	// envelope well past the 64KB default cap.
	for i := 0; i < 2000; i++ {
		ws.RecordEdit(fmt.Sprintf("/work/proj/very_long_path_%05d.go", i), 100000, 0)
	}
	env := NewContextEnvelope(ws, IntentExploration)
	env.Project.Name = "proj"
	env.Project.FileCount = 2000

	raw, err := env.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(raw) > defaultEnvelopeMaxSize {
		t.Errorf("envelope size = %d, exceeds the %d-byte cap", len(raw), defaultEnvelopeMaxSize)
	}
	// It must still be valid JSON with the working_set key present.
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("truncated envelope is not valid JSON: %v", err)
	}
	if _, ok := decoded["working_set"]; !ok {
		t.Errorf("truncated envelope missing working_set key")
	}
}

// TestContextEnvelopeStructure (014 step 22.3): GenerateContextEnvelope assembles
// a valid, size-bounded JSON envelope with the project metadata, working set,
// intent, matched skills, and handoff refs.
func TestContextEnvelopeStructure(t *testing.T) {
	project := "/work/proj"

	ws := NewWorkingSet(project)
	ws.RecordEdit("/work/proj/AGENTS.md", 10, 2)
	ws.RecordEdit("/work/proj/src/auth.go", 30, 4)

	env := NewContextEnvelope(ws, ClassifyWorkIntent("fix the null pointer in auth.go"))
	env.Project.Name = "proj"
	env.Project.FileCount = 42
	env.Project.Languages = []string{"go"}
	env.MatchedSkills = []string{"debugging", "tdd"}
	env.HandoffRefs = []string{"handoff.latest.json"}

	// The envelope must serialize to valid JSON with the expected top-level
	// keys (snake_case), and the intent must round-trip.
	raw, err := env.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}
	for _, key := range []string{"project", "working_set", "intent", "matched_skills", "handoff_refs"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("envelope JSON missing key %q", key)
		}
	}

	// Verify the decoded intent and working-set contents.
	var typed struct {
		Intent      string `json:"intent"`
		WorkingSet  struct {
			Files []WorkingFile `json:"files"`
		} `json:"working_set"`
		Project struct {
			Name      string   `json:"name"`
			FileCount int      `json:"file_count"`
			Languages []string `json:"languages"`
		} `json:"project"`
		MatchedSkills []string `json:"matched_skills"`
		HandoffRefs   []string `json:"handoff_refs"`
	}
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatalf("unmarshal typed envelope: %v", err)
	}
	if typed.Intent != string(IntentDebugging) {
		t.Errorf("intent = %q, want %q", typed.Intent, IntentDebugging)
	}
	if typed.Project.FileCount != 42 {
		t.Errorf("project.file_count = %d, want 42", typed.Project.FileCount)
	}
	if len(typed.WorkingSet.Files) != 2 {
		t.Errorf("working_set.files = %d, want 2", len(typed.WorkingSet.Files))
	}
	if len(typed.MatchedSkills) != 2 || typed.MatchedSkills[0] != "debugging" {
		t.Errorf("matched_skills = %v, want [debugging tdd]", typed.MatchedSkills)
	}
	if len(typed.HandoffRefs) != 1 || typed.HandoffRefs[0] != "handoff.latest.json" {
		t.Errorf("handoff_refs = %v", typed.HandoffRefs)
	}
}
