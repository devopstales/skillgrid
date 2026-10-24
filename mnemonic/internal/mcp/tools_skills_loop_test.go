package mcp

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// TestInspectSkillToolRegistered covers the tool surface: inspect_skill,
// amend_skill, and evaluate_skill are listed alongside the registry tools.
func TestInspectSkillToolRegistered(t *testing.T) {
	s := NewServer()
	tools := listToolNames(t, s)
	for _, name := range []string{"inspect_skill", "amend_skill", "evaluate_skill"} {
		if !tools[name] {
			t.Fatalf("%s not registered", name)
		}
	}
}

// TestInspectSkillToolReturnsEvidence drives inspect_skill end-to-end: a
// skill with 5 logged runs (3 with the same stderr) → the JSON result carries
// 5 total runs, 40% success rate, and the exact stderr as the top pattern.
func TestInspectSkillToolReturnsEvidence(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	useSkillSetup(t, "skinsp1")

	if _, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":   "skinsp1",
		"name":      "flaky-skill",
		"language":  "bash",
		"code":      "echo f\n",
		"overwrite": true,
	})); err != nil {
		t.Fatalf("handleWriteSkill: %v", err)
	}
	// 2 successes then 3 failures with the identical stderr.
	if _, err := rootDB().Exec(`
		INSERT INTO skill_usage (skill_id, session_id, timestamp, outcome, stderr)
		SELECT id, NULL, '2026-10-24T00:00:00Z', 'success', '' FROM skills WHERE name = 'flaky-skill'`); err != nil {
		t.Fatalf("seed 1: %v", err)
	}
	if _, err := rootDB().Exec(`
		INSERT INTO skill_usage (skill_id, session_id, timestamp, outcome, stderr)
		SELECT id, NULL, '2026-10-24T00:01:00Z', 'success', '' FROM skills WHERE name = 'flaky-skill'`); err != nil {
		t.Fatalf("seed 2: %v", err)
	}
	for _, ts := range []string{"2026-10-24T00:02:00Z", "2026-10-24T00:03:00Z", "2026-10-24T00:04:00Z"} {
		if _, err := rootDB().Exec(`
			INSERT INTO skill_usage (skill_id, session_id, timestamp, outcome, stderr)
			SELECT id, NULL, ?, 'failure', 'connection timeout' FROM skills WHERE name = 'flaky-skill'`, ts); err != nil {
			t.Fatalf("seed failure: %v", err)
		}
	}

	res, err := handleInspectSkill(context.Background(), callReq("inspect_skill", map[string]any{
		"project": "skinsp1",
		"name":    "flaky-skill",
	}))
	if err != nil {
		t.Fatalf("handleInspectSkill: %v", err)
	}
	if res.IsError {
		t.Fatalf("inspect_skill error: %s", callResultText(t, res))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &parsed); err != nil {
		t.Fatalf("parse inspect result: %v", err)
	}
	if parsed["total_runs"] != float64(5) {
		t.Errorf("total_runs = %v, want 5", parsed["total_runs"])
	}
	if parsed["success_rate"] != float64(40) {
		t.Errorf("success_rate = %v, want 40", parsed["success_rate"])
	}
	patterns, ok := parsed["top_patterns"].([]any)
	if !ok || len(patterns) != 1 {
		t.Fatalf("top_patterns = %v, want exactly 1", parsed["top_patterns"])
	}
	top, _ := patterns[0].(map[string]any)
	if top["stderr"] != "connection timeout" || top["count"] != float64(3) {
		t.Errorf("top pattern = %v, want {connection timeout 3}", top)
	}
}

// TestAmendSkillToolWithoutLLMErrors covers the no-LLM boundary at the tool
// surface: with the LLM seam unset, amend_skill returns a tool error naming
// the LLM requirement.
func TestAmendSkillToolWithoutLLMErrors(t *testing.T) {
	useSkillSetup(t, "skamd1")
	prev := service.AskLLMSeam()
	service.SetAskLLM(nil)
	t.Cleanup(func() { service.SetAskLLM(prev) })

	res, err := handleAmendSkill(context.Background(), callReq("amend_skill", map[string]any{
		"project": "skamd1",
		"name":    "any-skill",
	}))
	if err != nil {
		t.Fatalf("handleAmendSkill: %v", err)
	}
	if !res.IsError {
		t.Fatalf("amend_skill should error without an LLM, got: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), "LLM") {
		t.Errorf("error should name the LLM requirement: %s", callResultText(t, res))
	}
}

// TestEvaluateSkillToolMissingAmendedErrors covers the missing-.amended
// boundary at the tool surface.
func TestEvaluateSkillToolMissingAmendedErrors(t *testing.T) {
	useSkillSetup(t, "skeval1")

	res, err := handleEvaluateSkill(context.Background(), callReq("evaluate_skill", map[string]any{
		"project":    "skeval1",
		"name":       "nope-skill",
		"amended_at": "2026-10-24T00:30:00Z",
	}))
	if err != nil {
		t.Fatalf("handleEvaluateSkill: %v", err)
	}
	if !res.IsError {
		t.Fatalf("evaluate_skill should error for an unknown skill, got: %s", callResultText(t, res))
	}
}
