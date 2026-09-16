package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMemContextCLI (014 step 22.4): `mem context` prints a human-readable
// summary by default; `mem context --envelope` prints the full context
// envelope JSON to stdout (valid, parseable JSON with the expected keys).
func TestMemContextCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memctx-cli"
	seedMemCLIPeriod(t, dataDir, project)

	// Default: `mem context` prints a summary with a sessions key (the
	// recent session summaries read path).
	out := runMemCLI(t, dataDir, "context", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"sessions"`) {
		t.Fatalf("mem context (default) should print a summary with a sessions key, got: %s", out)
	}

	// --envelope: prints the full context envelope JSON.
	out = runMemCLI(t, dataDir, "context", "--envelope", "--project", project, "--dir", dataDir)
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("mem context --envelope output is not valid JSON: %v\n%s", err, out)
	}
	for _, key := range []string{"project", "working_set", "intent", "matched_skills", "handoff_refs"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("envelope JSON missing key %q\n%s", key, out)
		}
	}
	// The project section must carry the project id (file_count may be 0 for
	// an unindexed project, but the name must be the project id).
	var typed struct {
		Project struct {
			Name      string   `json:"name"`
			FileCount int      `json:"file_count"`
			Languages []string `json:"languages"`
		} `json:"project"`
		Intent        string   `json:"intent"`
		MatchedSkills []string `json:"matched_skills"`
		HandoffRefs   []string `json:"handoff_refs"`
		WorkingSet    struct {
			Files []map[string]any `json:"files"`
		} `json:"working_set"`
	}
	if err := json.Unmarshal([]byte(out), &typed); err != nil {
		t.Fatalf("unmarshal typed envelope: %v\n%s", err, out)
	}
	if typed.Project.Name != project {
		t.Errorf("project.name = %q, want %q", typed.Project.Name, project)
	}
	if typed.Intent != "exploration" {
		t.Errorf("intent = %q, want exploration (no query → default)", typed.Intent)
	}
	// matched_skills must be non-empty for the default exploration intent.
	if len(typed.MatchedSkills) == 0 {
		t.Errorf("matched_skills is empty; expected skills for the exploration intent")
	}
	// handoff_refs must point at handoff.latest.json under the data dir.
	if len(typed.HandoffRefs) != 1 || !strings.Contains(typed.HandoffRefs[0], "handoff.latest.json") {
		t.Errorf("handoff_refs = %v, want [handoff.latest.json]", typed.HandoffRefs)
	}
}
