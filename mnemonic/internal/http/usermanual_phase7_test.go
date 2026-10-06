package http

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhase7_UserManual asserts the user manual has a serve/dashboard page
// (Phase 7.6) that documents the views + the per-provider tracker-CLI
// dependency (install/auth + degraded states). The doc lives in docs/user-guide,
// which is at the repo root (not the Go module) — resolve it from the test's
// working dir by walking up to a dir that contains docs/user-guide.
func TestPhase7_UserManual(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	var guideDir string
	for d := wd; d != "/"; d = filepath.Dir(d) {
		cand := filepath.Join(d, "docs", "user-guide")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			guideDir = cand
			break
		}
	}
	if guideDir == "" {
		t.Skip("docs/user-guide not found from test cwd")
	}
	data, err := os.ReadFile(filepath.Join(guideDir, "09-serve-dashboard.md"))
	if err != nil {
		t.Fatalf("read 09-serve-dashboard.md: %v", err)
	}
	s := string(data)

	// The page must document the command, the views, and the tracker CLI
	// dependency (install/auth + degraded states).
	for _, want := range []string{
		"skillgrid serve",
		"Graph", "Memories", "Sessions", "Docs", "Plans", "Activity", "Git", "Prototypes",
		"Tracker provider CLI",
		"gh auth login", // an install/auth example
		"Degraded state",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("serve manual missing: %q", want)
		}
	}
}
