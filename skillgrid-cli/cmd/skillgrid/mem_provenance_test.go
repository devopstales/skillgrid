package main

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestMemProvenanceCLI covers 15.3: `mem provenance <observation_id>` shows
// the full curation chain (session_id, curate_command, source_files,
// llm_reasoning) as a readable, structured output; an observation with no
// provenance prints a clear message; a missing observation id fails with a
// clear error (non-zero exit).
func TestMemProvenanceCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-provenance"

	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
		VALUES ('sess-prov', ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'prov session', 'prov session summary')`,
		project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	mem := memory.New(st, project)
	ctx := context.Background()
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-prov",
		Type:      "decision",
		Title:     "prov cli note",
		Content:   "content for the provenance cli",
		Provenance: &memory.Provenance{
			SessionID:     "sess-prov",
			CurateCommand: "mem_save --scope project",
			SourceFiles:   []string{"src/cli.go", "src/cli_test.go"},
			LLMReasoning:  "decision extracted from session summary",
		},
	})
	if err != nil {
		t.Fatalf("save with provenance: %v", err)
	}
	plain, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-prov",
		Type:      "decision",
		Title:     "prov cli plain",
		Content:   "content with no provenance",
	})
	if err != nil {
		t.Fatalf("save plain: %v", err)
	}
	st.Close()

	// `mem provenance <id>` shows all four chain elements as a readable chain.
	out := runMemCLI(t, dataDir, "provenance", strconv.FormatInt(id, 10), "--project", project, "--dir", dataDir)
	for _, s := range []string{
		"sess-prov",
		"mem_save --scope project",
		"src/cli.go",
		"src/cli_test.go",
		"decision extracted from session summary",
	} {
		if !strings.Contains(out, s) {
			t.Fatalf("mem provenance missing chain element %q:\n%s", s, out)
		}
	}
	// The structured chain labels are present (not raw single-line JSON).
	for _, label := range []string{"session_id", "curate_command", "source_files", "llm_reasoning"} {
		if !strings.Contains(out, label) {
			t.Fatalf("mem provenance missing chain label %q:\n%s", label, out)
		}
	}

	// An observation without provenance gets a clear message (exit 0).
	out = runMemCLI(t, dataDir, "provenance", strconv.FormatInt(plain, 10), "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "no provenance") {
		t.Fatalf("observation without provenance should print a clear message, got:\n%s", out)
	}

	// A missing observation id fails with a clear error (non-zero exit).
	out, err = runMemCLIExpectError(t, dataDir, "provenance", "999999", "--project", project, "--dir", dataDir)
	if err == nil {
		t.Fatalf("mem provenance with a missing id should fail, got: %s", out)
	}
	if !strings.Contains(out, "not found") {
		t.Fatalf("missing observation id should be rejected clearly, got: %s", out)
	}
}
