package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestSkillToolsRegistered covers @step-03 (Given): the three skill registry
// tools are listed alongside the unchanged mem_* tools.
func TestSkillToolsRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	for _, name := range []string{"write_skill", "list_skills", "search_skills"} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("%s not registered", name)
		}
	}
	if _, ok := tools["mem_save"]; !ok {
		t.Fatal("mem_save missing after skill tools registered")
	}
}

// TestWriteListSearchSkillsMCP covers @step-03 (When/Then, write-list-search
// portion): write_skill creates FS file + SQL row + FTS entry; list_skills and
// search_skills return it with stored metadata; mem_save still works.
func TestWriteListSearchSkillsMCP(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillmcp"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil); st.Close() })

	sid := "sess-skill-mcp"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	res, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":     project,
		"name":        "lint-fix",
		"language":    "sh",
		"description": "auto-fix lint findings",
		"code":        "echo lint\n",
	}))
	if err != nil {
		t.Fatalf("handleWriteSkill: %v", err)
	}
	if res.IsError {
		t.Fatalf("write_skill error: %s", callResultText(t, res))
	}

	// list_skills returns the skill with metadata.
	res, err = handleListSkills(context.Background(), callReq("list_skills", map[string]any{
		"project": project,
	}))
	if err != nil {
		t.Fatalf("handleListSkills: %v", err)
	}
	if res.IsError {
		t.Fatalf("list_skills error: %s", callResultText(t, res))
	}
	var listOut map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &listOut); err != nil {
		t.Fatalf("parse list result: %v", err)
	}
	if listOut["count"] != float64(1) {
		t.Fatalf("list count = %v, want 1", listOut["count"])
	}
	skills, _ := listOut["skills"].([]any)
	if len(skills) != 1 {
		t.Fatalf("list skills = %d, want 1", len(skills))
	}
	sk, _ := skills[0].(map[string]any)
	if sk["name"] != "lint-fix" || sk["language"] != "sh" {
		t.Errorf("list skill = %+v, want lint-fix/sh", sk)
	}

	// search_skills finds it lexically (FTS over description).
	res, err = handleSearchSkills(context.Background(), callReq("search_skills", map[string]any{
		"project": project,
		"query":   "lint",
	}))
	if err != nil {
		t.Fatalf("handleSearchSkills: %v", err)
	}
	if res.IsError {
		t.Fatalf("search_skills error: %s", callResultText(t, res))
	}
	var searchOut map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &searchOut); err != nil {
		t.Fatalf("parse search result: %v", err)
	}
	if searchOut["count"] != float64(1) {
		t.Fatalf("search count = %v, want 1", searchOut["count"])
	}

	// The dual path: an observation with memory_type="skill" is still saved
	// (intent-matching compatibility) — mem_save still works.
	res, err = handleMemSave(context.Background(), callReq("mem_save", map[string]any{
		"project":    project,
		"title":      "registry note",
		"type":       "pattern",
		"content":    "the skills registry is FS + SQL + FTS",
		"session_id": sid,
		"topic_key":  "skill/registry/notes",
	}))
	if err != nil {
		t.Fatalf("handleMemSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save error: %s", callResultText(t, res))
	}
}

// TestWriteSkillOverwriteFalseRejectsMCP covers @step-03 (edge: Overwrite
// false rejects name collision): the tool returns a clean error naming the
// collision and writes nothing.
func TestWriteSkillOverwriteFalseRejectsMCP(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillmcp2"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil); st.Close() })

	if _, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":     project,
		"name":        "clash",
		"language":    "sh",
		"description": "first",
		"code":        "echo 1\n",
	})); err != nil {
		t.Fatalf("first write_skill: %v", err)
	}
	res, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":     project,
		"name":        "clash",
		"language":    "sh",
		"description": "second",
		"code":        "echo 2\n",
	}))
	if err != nil {
		t.Fatalf("second write_skill: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for overwrite=false name collision")
	}
	if !strings.Contains(callResultText(t, res), "clash") {
		t.Errorf("collision error should name the skill: %s", callResultText(t, res))
	}

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills WHERE name = 'clash'`).Scan(&n); err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if n != 1 {
		t.Errorf("skills rows for clash = %d, want 1", n)
	}
}

// TestWriteSkillUnknownLanguageMCP covers @step-03 (soft-delete portion,
// unknown language rejects): an unknown language is a clean tool error with no
// SQL row and no FS file.
func TestWriteSkillUnknownLanguageMCP(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillmcp3"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil); st.Close() })

	res, err := handleWriteSkill(context.Background(), callReq("write_skill", map[string]any{
		"project":     project,
		"name":        "weird",
		"language":    "cobol",
		"description": "nope",
		"code":        "x\n",
	}))
	if err != nil {
		t.Fatalf("write_skill: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for unknown language")
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills`).Scan(&n); err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if n != 0 {
		t.Errorf("skills rows = %d, want 0", n)
	}
}
