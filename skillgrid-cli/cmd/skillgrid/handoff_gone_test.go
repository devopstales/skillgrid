package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestHandoffCommandGone is TICKET-05 (old-surfaces-removed): the Handoff Hub
// CLI is deleted, so `skillgrid handoff status` exits 2 as an unknown
// subcommand. Shape follows TestTrailNotFound (`go run .` + mustWD).
func TestHandoffCommandGone(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "handoff", "status")
	cmd.Dir = mustWD(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for removed handoff command: %s", out)
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("want exit 2, got %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `unknown command "handoff"`) {
		t.Fatalf("want unknown-subcommand error, got %s", out)
	}
}
