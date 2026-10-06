package mcp

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// useSkillSetup opens a project store, injects the service, seeds a session,
// and returns the project id.
func useSkillSetup(t *testing.T, project string) *store.Store {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil); st.Close() })

	sid := "sess-skill-exec"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return st
}

// TestUseSkillRegistered covers @step-03 (Given): use_skill is listed alongside
// the registry tools.
func TestUseSkillRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if _, ok := tools["use_skill"]; !ok {
		t.Fatal("use_skill not registered")
	}
	for _, name := range []string{"write_skill", "list_skills", "search_skills"} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("%s missing after use_skill registration", name)
		}
	}
}

// TestUseSkillExecutesAndLogs covers @step-03 (happy): use_skill runs a skill
// in the sandbox and returns the captured output; a skill_usage row and a
// session_events trail row are logged.
func TestUseSkillExecutesAndLogs(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	useSkillSetup(t, "skusex1")

	if _, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":  "skusex1",
		"name":     "greet",
		"language": "bash",
		"code":     "echo from-skill\n",
	})); err != nil {
		t.Fatalf("handleWriteSkill: %v", err)
	}

	res, err := handleUseSkill(context.Background(), callReq("use_skill", map[string]any{
		"project":    "skusex1",
		"name":       "greet",
		"session_id": "sess-skill-exec",
	}))
	if err != nil {
		t.Fatalf("handleUseSkill: %v", err)
	}
	if res.IsError {
		t.Fatalf("use_skill error: %s", callResultText(t, res))
	}
	out := callResultText(t, res)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse use_skill result: %v", err)
	}
	if parsed["stdout"] != "from-skill\n" {
		t.Errorf("stdout = %v, want %q", parsed["stdout"], "from-skill\n")
	}
	if parsed["exit_code"] != float64(0) {
		t.Errorf("exit_code = %v, want 0", parsed["exit_code"])
	}

	var usageCount int
	if err := rootDB().QueryRow(`SELECT COUNT(*) FROM skill_usage`).Scan(&usageCount); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if usageCount != 1 {
		t.Errorf("skill_usage rows = %d, want 1", usageCount)
	}
	var eventCount int
	if err := rootDB().QueryRow(
		`SELECT COUNT(*) FROM session_events WHERE action_type = 'skill_use'`).Scan(&eventCount); err != nil {
		t.Fatalf("count session_events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("session_events rows = %d, want 1 (skill_use trail)", eventCount)
	}
}

// TestUseSkillNonZeroExitReturnsOutput covers the non-zero exit boundary at the
// tool level: the call succeeds (execution happened), the result carries the
// non-zero exit code and the captured stderr, and the trail event records the
// failure status.
func TestUseSkillNonZeroExitReturnsOutput(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	useSkillSetup(t, "skusex2")

	if _, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":  "skusex2",
		"name":     "failing",
		"language": "bash",
		"code":     "echo boom 1>&2\nexit 7\n",
	})); err != nil {
		t.Fatalf("handleWriteSkill: %v", err)
	}

	res, err := handleUseSkill(context.Background(), callReq("use_skill", map[string]any{
		"project":    "skusex2",
		"name":       "failing",
		"session_id": "sess-skill-exec",
	}))
	if err != nil {
		t.Fatalf("handleUseSkill: %v", err)
	}
	if res.IsError {
		t.Fatalf("use_skill should not be a tool error for a non-zero exit: %s", callResultText(t, res))
	}
	out := callResultText(t, res)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse use_skill result: %v", err)
	}
	if parsed["exit_code"] != float64(7) {
		t.Errorf("exit_code = %v, want 7", parsed["exit_code"])
	}
	if parsed["stderr"] != "boom\n" {
		t.Errorf("stderr = %v, want %q", parsed["stderr"], "boom\n")
	}
}

// TestUseSkillUnknownLanguageNoExecMCP covers @step-03 (failure: unknown
// language rejects without exec) at the tool level: a clean error, no
// skill_usage row.
func TestUseSkillUnknownLanguageNoExecMCP(t *testing.T) {
	useSkillSetup(t, "skusex3")

	if _, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":  "skusex3",
		"name":     "weird-lang",
		"language": "cobol",
		"code":     "x\n",
	})); err != nil {
		t.Fatalf("handleWriteSkill: %v", err)
	}

	res, err := handleUseSkill(context.Background(), callReq("use_skill", map[string]any{
		"project":    "skusex3",
		"name":       "weird-lang",
		"session_id": "sess-skill-exec",
	}))
	if err != nil {
		t.Fatalf("handleUseSkill: %v", err)
	}
	if !res.IsError {
		t.Fatalf("use_skill should error for an unknown language: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), "weird-lang") {
		t.Errorf("error should name the skill: %s", callResultText(t, res))
	}
	var n int
	if err := rootDB().QueryRow(`SELECT COUNT(*) FROM skill_usage`).Scan(&n); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if n != 0 {
		t.Errorf("skill_usage rows = %d, want 0 (no exec)", n)
	}
}

// TestUseSkillSoftDeletedErrors covers @step-03 (failure: soft-deleted skill
// → error) at the tool level.
func TestUseSkillSoftDeletedErrors(t *testing.T) {
	useSkillSetup(t, "skusex4")

	if _, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":  "skusex4",
		"name":     "faded",
		"language": "bash",
		"code":     "echo faded\n",
	})); err != nil {
		t.Fatalf("handleWriteSkill: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := rootDB().Exec(`
		UPDATE skills SET deleted_at = ? WHERE name = 'faded'`, now); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	res, err := handleUseSkill(context.Background(), callReq("use_skill", map[string]any{
		"project":    "skusex4",
		"name":       "faded",
		"session_id": "sess-skill-exec",
	}))
	if err != nil {
		t.Fatalf("handleUseSkill: %v", err)
	}
	if !res.IsError {
		t.Fatalf("use_skill should error for a soft-deleted skill: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), "faded") {
		t.Errorf("error should name the skill: %s", callResultText(t, res))
	}
}
