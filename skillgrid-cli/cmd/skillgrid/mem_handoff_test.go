package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemHandoffCLI is 014 step 21.4 [AFK]: `mem handoff` generates the
// prefix+delta handoff artifact, writes handoff.latest.json, and prints a
// summary; `mem handoff --json` prints the full JSON.
func TestMemHandoffCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-handoff"
	seedMemCLIPeriod(t, dataDir, project)

	// Seed a hub file on disk in the CWD so the prefix has a hub summary and
	// the codeindex files table has a row to count.
	wd := mustWD(t)
	hubPath := filepath.Join(wd, "AGENTS.md")
	if err := os.WriteFile(hubPath, []byte("agent config v1\n"), 0o644); err != nil {
		t.Fatalf("write hub: %v", err)
	}
	t.Cleanup(func() { os.Remove(hubPath) })

	// Default: prints a human-readable summary and writes the artifact.
	out := runMemCLI(t, dataDir, "handoff", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "handoff.latest.json") {
		t.Fatalf("mem handoff missing the artifact path: %s", out)
	}
	if !strings.Contains(out, project) {
		t.Fatalf("mem handoff missing the project id: %s", out)
	}
	artifact := filepath.Join(dataDir, "handoff.latest.json")
	if _, err := os.Stat(artifact); err != nil {
		t.Fatalf("handoff.latest.json not written: %v", err)
	}

	// --json: prints the full handoff JSON (prefix + delta sections).
	out = runMemCLI(t, dataDir, "handoff", "--json", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"prefix"`) || !strings.Contains(out, `"delta"`) {
		t.Fatalf("mem handoff --json missing prefix/delta: %s", out)
	}
}
