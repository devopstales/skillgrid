package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "tester@example.com"},
		{"config", "user.name", "Tester"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	hooks := filepath.Join(repo, ".nullhooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("config", "core.hooksPath", hooks)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-q", "--no-verify", "-m", "init")
	return repo
}

// SATISFIES: prime-and-hooks-share-one-project
func TestPrime_ProjectMatchesHTTPStore(t *testing.T) {
	repo := initGitRepo(t)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	h, cleanup, err := openLoop(repo)
	if err != nil {
		t.Fatalf("openLoop: %v", err)
	}
	primeProj := h.ProjectID()
	cleanup()

	s := mnemonichttp.NewServer(service.New(dataDir))
	body := map[string]any{
		"session_id": "prime-match-1",
		"directory":  repo,
		"tool_name":  "Shell",
		"command":    "true",
		"agent":      "cursor",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/sessions/prime-match-1/tool-calls", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	httpProj, _ := out["project"].(string)
	if httpProj != primeProj {
		t.Fatalf("HTTP project %q != prime project %q", httpProj, primeProj)
	}
}
