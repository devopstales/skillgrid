package main

// save_prompt_test.go — the host-capture verb contract (ADR-0067): stdin
// shapes, the dedup re-fire, the project-resolution chain (threat matrix:
// Git repository selection), exit codes, the fail-open cap.

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// withStdin swaps the verb's stdin seam for one pass.
func withStdin(t *testing.T, s string) {
	t.Helper()
	prev := stdinReader
	stdinReader = strings.NewReader(s)
	t.Cleanup(func() { stdinReader = prev })
}

// promptRows counts user_prompts rows in the per-project store file.
func promptRows(t *testing.T, dataDir, project string) int {
	t.Helper()
	st, err := store.Open(context.Background(), dataDir, project, store.SQLite, "")
	if err != nil {
		t.Fatalf("open %s store: %v", project, err)
	}
	defer st.Close()
	rows, err := st.Query(context.Background(), `SELECT count(*) FROM user_prompts`)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	n, _ := rows[0][0].(int64)
	return int(n)
}

// promptContents returns the stored user_prompts content (id order) — the
// assertion surface for what the verb stored, not just how much.
func promptContents(t *testing.T, dataDir, project string) []string {
	t.Helper()
	st, err := store.Open(context.Background(), dataDir, project, store.SQLite, "")
	if err != nil {
		t.Fatalf("open %s store: %v", project, err)
	}
	defer st.Close()
	rows, err := st.Query(context.Background(), `SELECT content FROM user_prompts ORDER BY id`)
	if err != nil {
		t.Fatalf("contents: %v", err)
	}
	contents := make([]string, 0, len(rows))
	for _, r := range rows {
		contents = append(contents, r[0].(string))
	}
	return contents
}

// TestSavePromptStoresPlainStdin: the contract — stdin plain text, exit 0,
// no stdout, one row under the adapter's session.
func TestSavePromptStoresPlainStdin(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, "first prompt")
	out := runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	if out != "" {
		t.Fatalf("the verb prints nothing on success, got %q", out)
	}
	contents := promptContents(t, dataDir, "p")
	if len(contents) != 1 || contents[0] != "first prompt" {
		t.Fatalf("want 1 row with content %q, got %v", "first prompt", contents)
	}
}

// TestSavePromptIdempotentRefire: identical content + session inside the
// window stores no second row; distinct content does.
func TestSavePromptIdempotentRefire(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, "same prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	withStdin(t, "same prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	if n := promptRows(t, dataDir, "p"); n != 1 {
		t.Fatalf("re-fire must dedup, got %d rows", n)
	}
	withStdin(t, "different prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	if n := promptRows(t, dataDir, "p"); n != 2 {
		t.Fatalf("distinct content must store, got %d rows", n)
	}
}

// TestSavePromptCursorJSON: the cursor beforeSubmitPrompt envelope yields
// the "prompt" field; a malformed envelope is raw text (fail-open).
func TestSavePromptCursorJSON(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, `{"prompt":"hello cursor","conversation_id":"c1"}`)
	runCLI(t, "memory", "save-prompt", "--id", "cursor", "--project", "p")
	withStdin(t, `{not json`)
	runCLI(t, "memory", "save-prompt", "--id", "cursor", "--project", "p")
	contents := promptContents(t, dataDir, "p")
	if len(contents) != 2 {
		t.Fatalf("want 2 rows (json field + raw text), got %v", contents)
	}
	// The cursor envelope yields the "prompt" field, not the envelope.
	if contents[0] != "hello cursor" {
		t.Fatalf("cursor envelope must store the prompt field, got %q", contents[0])
	}
	// A malformed envelope is raw text, not an error.
	if contents[1] != "{not json" {
		t.Fatalf("malformed envelope must store raw text, got %q", contents[1])
	}
}

// TestSavePromptRequiresID: usage contract.
func TestSavePromptRequiresID(t *testing.T) {
	installRunEnv(t)
	withStdin(t, "prompt without id")
	err := runCLIError(t, "memory", "save-prompt", "--project", "p")
	if err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("missing --id must be a usage error naming the flag: %v", err)
	}
}

// TestSavePromptAutoRegistersMissingSession: fail-open ordering — the
// store's auto-register means capture never fails on a session the
// adapter forgot to start (it would only create a phantom session row,
// which the adapter's guard prevents).
func TestSavePromptAutoRegistersMissingSession(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, "prompt before session start")
	runCLI(t, "memory", "save-prompt", "--id", "s2", "--project", "p")
	st, _ := store.Open(context.Background(), dataDir, "p", store.SQLite, "")
	defer st.Close()
	rows, err := st.Query(context.Background(),
		`SELECT ownership_mode FROM sessions WHERE id = 's2'`)
	if err != nil || len(rows) != 1 || rows[0][0] != "shared" {
		t.Fatalf("missing session must auto-register shared-ownership: rows=%v err=%v", rows, err)
	}
}

// TestSavePromptResolvesProjectChain: threat RED (Git repository
// selection) — the env-var leg of the chain (explicit > env > git >
// basename) routes the prompt to the named project's store and touches no
// other project's file.
func TestSavePromptResolvesProjectChain(t *testing.T) {
	dataDir := installRunEnvDir(t)
	t.Setenv(config.Default().Project.EnvVar, "wt")
	withStdin(t, "worktree prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1")
	if n := promptRows(t, dataDir, "wt"); n != 1 {
		t.Fatalf("env-var project must get the row, got %d", n)
	}
	if n := promptRows(t, dataDir, "p"); n != 0 {
		t.Fatalf("no other project's store may be touched, got %d", n)
	}
}

// TestReadStdinCapped: the 256 KiB cap (the subprocess threat line) —
// excess discarded, not an error.
func TestReadStdinCapped(t *testing.T) {
	big := strings.Repeat("a", savePromptStdinCap+1024)
	got, err := readStdinCapped(strings.NewReader(big))
	if err != nil {
		t.Fatalf("cap: %v", err)
	}
	if len(got) != savePromptStdinCap || string(got) != big[:savePromptStdinCap] {
		t.Fatalf("want exactly the first %d bytes, got %d", savePromptStdinCap, len(got))
	}
}
