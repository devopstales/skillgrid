package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestTrailRecentAndShow(t *testing.T) {
	dataDir := t.TempDir()
	project := "trailproj"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dirs, _ := json.Marshal([]string{"/content"})
	files, _ := json.Marshal([]string{"/content/a.md"})
	res, err := st.DB.Exec(`
		INSERT INTO retrieval_trails (project, query, directories_json, files_json, result_path, corpus, created_at)
		VALUES (?, 'find widgets', ?, ?, '/content/a.md', 'ltm', ?)`,
		project, string(dirs), string(files), now)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	st.Close()

	out := runTrailCLI(t, dataDir, "recent", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "find widgets") || !strings.Contains(out, "/content/a.md") {
		t.Fatalf("recent missing fields: %s", out)
	}
	out = runTrailCLI(t, dataDir, "show", strconv.FormatInt(id, 10), "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "find widgets") {
		t.Fatalf("show: %s", out)
	}
}

func TestTrailEmpty(t *testing.T) {
	dataDir := t.TempDir()
	project := "emptytrail"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	out := runTrailCLI(t, dataDir, "recent", "--project", project, "--dir", dataDir)
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" && trimmed != "null" {
		// empty JSON array expected
		var rows []any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			t.Fatalf("want empty list, got %q err=%v", out, err)
		}
		if len(rows) != 0 {
			t.Fatalf("want empty, got %s", out)
		}
	}
}

func TestTrailNotFound(t *testing.T) {
	dataDir := t.TempDir()
	project := "nftrail"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	cmd := exec.Command("go", "run", ".", "trail", "show", "99999", "--project", project, "--dir", dataDir)
	cmd.Dir = mustWD(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure, got %s", out)
	}
	if !strings.Contains(string(out), "not-found") {
		t.Fatalf("want not-found, got %s", out)
	}
}

func runTrailCLI(t *testing.T, dataDir string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"run", ".", "trail"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = mustWD(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("trail %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func mustWD(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Prefer package dir when tests run from module root.
	if filepath.Base(wd) != "skillgrid" || !fileExists(filepath.Join(wd, "trail.go")) {
		if fileExists(filepath.Join(wd, "cmd", "skillgrid", "trail.go")) {
			return filepath.Join(wd, "cmd", "skillgrid")
		}
	}
	return wd
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
