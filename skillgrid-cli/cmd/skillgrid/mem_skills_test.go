package main

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedSkillCLIStore opens a project store with a session and seeds one skill
// observation (memory_type=skill, topic_key=skill/<intent>/<lang>).
func seedSkillCLIStore(t *testing.T, dataDir, project string) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	mem := memory.New(st, project)
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s-skill', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := mem.Save(ctx, memory.SaveInput{
		SessionID:  "s-skill",
		Type:       "learning",
		Title:      "debug-go",
		Content:    "Read the Go stack trace bottom-up when a panic occurs.",
		MemoryType: "skill",
		TopicKey:   "skill/debugging/go",
	}); err != nil {
		t.Fatalf("save skill: %v", err)
	}
}

// TestMemSkillsAndHookCLI is 24.4 [AFK]: the `mem skills` and `mem hook` CLI
// subcommands. `mem skills list` lists the project's skills; `mem skills add
// <intent> <name> <content>` creates one; `mem hook list` shows the configured
// hooks; `mem hook run <type>` executes a hook (a prompt-submit run returns the
// classified intent; a run with hooks disabled reports hooks_disabled).
func TestMemSkillsAndHookCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memskills-cli"
	seedSkillCLIStore(t, dataDir, project)

	// mem skills list: shows the seeded skill.
	out := runMemCLI(t, dataDir, "skills", "list", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "debug-go") {
		t.Fatalf("mem skills list missing the seeded skill: %s", out)
	}
	if !strings.Contains(out, "skill") {
		t.Fatalf("mem skills list should reference the skill memory type: %s", out)
	}

	// mem skills add <intent> <name> <content>: creates a new skill.
	out = runMemCLI(t, dataDir, "skills", "add", "review", "review-go", "Review a diff by checking error paths.",
		"--project", project, "--dir", dataDir)
	if !strings.Contains(out, "review-go") {
		t.Fatalf("mem skills add did not confirm the created skill: %s", out)
	}
	// It is now listed.
	out = runMemCLI(t, dataDir, "skills", "list", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "review-go") {
		t.Fatalf("mem skills list missing the added skill: %s", out)
	}

	// mem hook list: shows the 4 configured hook types + the enabled state.
	out = runMemCLI(t, dataDir, "hook", "list", "--project", project, "--dir", dataDir)
	for _, h := range []string{"session-start", "pre-edit", "prompt-submit", "session-stop"} {
		if !strings.Contains(out, h) {
			t.Fatalf("mem hook list missing hook type %q: %s", h, out)
		}
	}

	// mem hook run prompt-submit --query "fix the bug": returns the intent.
	out = runMemCLI(t, dataDir, "hook", "run", "prompt-submit", "--query", "fix the bug",
		"--project", project, "--dir", dataDir)
	if strings.Contains(out, "hooks_disabled") {
		t.Fatalf("mem hook run prompt-submit should execute (hooks enabled): %s", out)
	}
	if !strings.Contains(out, "debugging") {
		t.Fatalf("mem hook run prompt-submit should classify as debugging: %s", out)
	}
}
