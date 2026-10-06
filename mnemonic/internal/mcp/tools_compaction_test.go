package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/skills"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func openCommitMCPStore(t *testing.T) (dataDir, project string, st *store.Store) {
	t.Helper()
	dataDir = t.TempDir()
	project = "commitmcp"
	t.Setenv("MNEMONIC_PROJECT", project)
	var err error
	st, err = store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	// The handler opens the project via svc.Open(projectID), which resolves the
	// project root to "." (the CWD). Chdir into the temp dir so the auto-skill
	// FS file lands under the temp tree instead of leaking into the package dir.
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dataDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { SetService(nil); _ = os.Chdir(oldDir); st.Close() })
	SetService(service.New(dataDir))
	return dataDir, project, st
}

func seedCommitSession(t *testing.T, st *store.Store, project, dataDir, sid string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

// TestMnemonicCommitExtractsFacts covers @step-04: mnemonic_commit preserves
// the 003 compaction behavior (L2 + long_term_memories) AND extracts each
// bullet of lessons_learned as a fact row.
func TestMnemonicCommitExtractsFacts(t *testing.T) {
	dataDir, project, st := openCommitMCPStore(t)
	sid := "sess-commit-facts"
	seedCommitSession(t, st, project, dataDir, sid)

	res, err := handleMnemonicCommit(context.Background(), callReq("mnemonic_commit", map[string]any{
		"project":         project,
		"title":           "session retrospective",
		"lessons_learned": "- the database connection pool max size is 50\n- use bcrypt for password hashing, never md5",
		"session_id":      sid,
	}))
	if err != nil {
		t.Fatalf("handleMnemonicCommit: %v", err)
	}
	if res.IsError {
		t.Fatalf("mnemonic_commit tool error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if _, ok := out["memory_id"].(float64); !ok {
		t.Fatalf("no memory_id in %v (003 compaction behavior must be preserved)", out)
	}
	factsExtracted, _ := out["facts_extracted"].(float64)
	if int(factsExtracted) != 2 {
		t.Fatalf("facts_extracted = %v, want 2 (one per lesson bullet)", out["facts_extracted"])
	}
	// The extracted facts must be retrievable through the fact search path.
	got, err := facts.New(st.DB, project).Search(context.Background(), sid, "bcrypt password hashing", 5)
	if err != nil {
		t.Fatalf("fact search: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("no facts found for the extracted lesson; extraction did not persist")
	}
}

// TestMnemonicCommitAutoSkill covers @step-04: a lesson that states a
// reusable pattern (with an explicit language tag) is auto-registered as a
// skill; a plain lesson with no reusable pattern is skipped with a warning
// and the commit still succeeds.
func TestMnemonicCommitAutoSkill(t *testing.T) {
	t.Run("reusable pattern registers a skill", func(t *testing.T) {
		dataDir, project, st := openCommitMCPStore(t)
		sid := "sess-commit-skill"
		seedCommitSession(t, st, project, dataDir, sid)
		res, err := handleMnemonicCommit(context.Background(), callReq("mnemonic_commit", map[string]any{
			"project":         project,
			"title":           "go testing lesson",
			"lessons_learned": "- [go] always write table-driven tests with t.Run subtests before implementing",
			"session_id":      sid,
		}))
		if err != nil {
			t.Fatalf("handleMnemonicCommit: %v", err)
		}
		if res.IsError {
			t.Fatalf("mnemonic_commit tool error: %s", callResultText(t, res))
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
			t.Fatalf("parse result: %v", err)
		}
		if _, ok := out["skill_name"].(string); !ok {
			t.Fatalf("no skill_name in %v, want an auto-skill for the reusable pattern", out)
		}
		list, err := skills.New(st.DB, dataDir, project).List(context.Background())
		if err != nil {
			t.Fatalf("skill list: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("skills = %d, want 1 auto-registered skill", len(list))
		}
	})

	t.Run("no reusable pattern skips with warning", func(t *testing.T) {
		dataDir, project, st := openCommitMCPStore(t)
		sid := "sess-commit-noskill"
		seedCommitSession(t, st, project, dataDir, sid)
		res, err := handleMnemonicCommit(context.Background(), callReq("mnemonic_commit", map[string]any{
			"project":         project,
			"title":           "plain retrospective",
			"lessons_learned": "- the staging deploy was slow today, probably network",
			"session_id":      sid,
		}))
		if err != nil {
			t.Fatalf("handleMnemonicCommit: %v", err)
		}
		if res.IsError {
			t.Fatalf("mnemonic_commit must succeed when the auto-skill is skipped: %s", callResultText(t, res))
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
			t.Fatalf("parse result: %v", err)
		}
		if _, ok := out["skill_name"]; ok {
			t.Fatalf("skill_name present in %v, want no auto-skill for a plain lesson", out)
		}
		warns, _ := out["warnings"].([]any)
		if len(warns) == 0 {
			t.Fatalf("warnings = 0, want a skip-auto-skill warning")
		}
		list, err := skills.New(st.DB, dataDir, project).List(context.Background())
		if err != nil {
			t.Fatalf("skill list: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("skills = %d, want 0 (auto-skill skipped)", len(list))
		}
	})
}
