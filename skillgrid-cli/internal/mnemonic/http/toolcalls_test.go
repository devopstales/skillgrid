package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func newToolCallsServer(t *testing.T) (*Server, string) {
	t.Helper()
	// Isolate .skillgrid/config.d/indexing.yaml in a throwaway directory so parallel
	// tests (and SIGKILL) never mutate the repo CWD.
	workDir := t.TempDir()
	t.Chdir(workDir)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	cfgDir := filepath.Join(workDir, ".skillgrid", "config.d")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "indexing.yaml"), []byte("mnemonic:\n  hooks:\n    enabled: true\n"), 0o644); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}
	s := NewServer(service.New(dataDir))
	return s, dataDir
}

func postToolCall(t *testing.T, s *Server, sid string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/sessions/"+sid+"/tool-calls?project="+proj, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

func TestToolCallRoute_WritesRow(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	body := map[string]any{
		"session_id":    "s1",
		"tool_name":     "Shell",
		"command":       "go test ./...",
		"result_status": "success",
	}
	res := postToolCall(t, s, "s1", body)
	if res.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", res.Code, res.Body.String())
	}

	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	events, _, _, err := h.Memory().SessionChanges(context.Background(), "s1")
	if err != nil {
		t.Fatalf("SessionChanges: %v", err)
	}
	var found bool
	for _, e := range events {
		if e.ActionType == "command_exec" && e.ToolName == "Shell" && e.Command == "go test ./..." {
			found = true
		}
	}
	if !found {
		t.Errorf("no command_exec row for Shell; events: %+v", events)
	}
}

// An unregistered harness session is created on the first tool call, tagged
// with the posted agent, and later calls sequence onto the same row.
func TestToolCallRoute_UnknownSessionIsCreated(t *testing.T) {
	s, _ := newToolCallsServer(t)
	for i, cmd := range []string{"ls", "pwd"} {
		res := postToolCall(t, s, "cursor-conv-1", map[string]any{
			"agent":     "cursor",
			"directory": t.TempDir(),
			"tool_name": "Shell",
			"command":   cmd,
		})
		want := http.StatusOK
		if i == 0 {
			want = http.StatusCreated
		}
		if res.Code != want {
			t.Fatalf("call %d: status %d, want %d: %s", i+1, res.Code, want, res.Body.String())
		}
	}

	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	var agent string
	if err := h.Store().DB.QueryRow(`SELECT COALESCE(agent,'') FROM sessions WHERE id = 'cursor-conv-1'`).Scan(&agent); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if agent != "cursor" {
		t.Errorf("agent = %q, want cursor", agent)
	}
	events, _, _, err := h.Memory().SessionChanges(context.Background(), "cursor-conv-1")
	if err != nil {
		t.Fatalf("SessionChanges: %v", err)
	}
	var execs []int
	for _, e := range events {
		if e.ActionType == "command_exec" {
			execs = append(execs, e.Sequence)
		}
	}
	if len(execs) != 2 || execs[0] >= execs[1] {
		t.Errorf("want two ordered command_exec events, got sequences %v (events %+v)", execs, events)
	}
}

// POST /sessions with an explicit project registers the harness id there and
// records the agent; a repeat is idempotent and keeps the first agent.
func TestSessionCreate_HarnessRegistration(t *testing.T) {
	s, _ := newToolCallsServer(t)
	post := func(agent string) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(map[string]any{"id": "oc-1", "agent": agent, "directory": t.TempDir(), "title": agent})
		req := httptest.NewRequest(http.MethodPost, "/sessions?project="+proj, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		return w
	}
	if res := post("opencode"); res.Code != http.StatusCreated {
		t.Fatalf("first register: status %d: %s", res.Code, res.Body.String())
	}
	if res := post("cursor"); res.Code != http.StatusOK {
		t.Fatalf("second register: status %d: %s", res.Code, res.Body.String())
	}
	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	var agent string
	if err := h.Store().DB.QueryRow(`SELECT COALESCE(agent,'') FROM sessions WHERE id = 'oc-1'`).Scan(&agent); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if agent != "opencode" {
		t.Errorf("agent = %q, want opencode (first registration wins)", agent)
	}
}

func TestToolCalls_PrivateSpan(t *testing.T) {
	s, _ := newToolCallsServer(t)
	secret := "abc123"
	body := map[string]any{
		"session_id":      "s-private",
		"tool_name":       "Shell",
		"command":         "echo <private>" + secret + "</private> test",
		"content_preview": "token=<private>" + secret + "</private> ok",
		"result_status":   "success",
	}
	res := postToolCall(t, s, "s-private", body)
	if res.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201: %s", res.Code, res.Body.String())
	}

	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	events, _, _, err := h.Memory().SessionChanges(context.Background(), "s-private")
	if err != nil {
		t.Fatalf("SessionChanges: %v", err)
	}
	var payload, command string
	for _, e := range events {
		if e.ActionType == "command_exec" {
			payload = e.Payload
			command = e.Command
		}
	}
	if payload == "" {
		t.Fatalf("no command_exec event; events: %+v", events)
	}
	if strings.Contains(payload, secret) || strings.Contains(command, secret) {
		t.Errorf("stored tool call leaked %q: payload=%q command=%q", secret, payload, command)
	}
	if !strings.Contains(payload, "token= ok") {
		t.Errorf("payload = %q, want stripped preview containing %q", payload, "token= ok")
	}
}

func TestToolCalls_PrivateSpanInPath(t *testing.T) {
	s, _ := newToolCallsServer(t)
	secret := "path-secret-xyz"
	body := map[string]any{
		"session_id":    "s-private-path",
		"tool_name":     "Read",
		"path":            "/tmp/<private>" + secret + "</private>/file.go",
		"result_status":   "success",
	}
	res := postToolCall(t, s, "s-private-path", body)
	if res.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201: %s", res.Code, res.Body.String())
	}

	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	events, _, _, err := h.Memory().SessionChanges(context.Background(), "s-private-path")
	if err != nil {
		t.Fatalf("SessionChanges: %v", err)
	}
	var filePath string
	for _, e := range events {
		if e.ActionType == "file_read" {
			filePath = e.Path
		}
	}
	if filePath == "" {
		t.Fatalf("no file_read event; events: %+v", events)
	}
	if strings.Contains(filePath, secret) {
		t.Errorf("stored path leaked %q: file=%q", secret, filePath)
	}
	if !strings.Contains(filePath, "/tmp/") || !strings.Contains(filePath, "file.go") {
		t.Errorf("path = %q, want stripped path with /tmp/ and file.go", filePath)
	}
}

func initTempGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	runGit("config", "user.email", "tester@example.com")
	runGit("config", "user.name", "Tester")
	hooks := filepath.Join(repo, ".nullhooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit("config", "core.hooksPath", hooks)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README.md")
	runGit("commit", "-q", "--no-verify", "-m", "init")
	return repo
}

// SATISFIES: directory-without-project-param
func TestToolCalls_ResolvesProjectFromDirectory(t *testing.T) {
	repo := initTempGitRepo(t)
	wantProj, err := project.Resolve(repo)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	s := NewServer(service.New(dataDir))

	body := map[string]any{
		"session_id": "dir-only-1",
		"directory":  repo,
		"tool_name":  "Shell",
		"command":    "echo hi",
		"agent":      "cursor",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/sessions/dir-only-1/tool-calls", bytes.NewReader(raw))
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
	got, _ := out["project"].(string)
	if got != wantProj {
		t.Fatalf("response project %q, want %q", got, wantProj)
	}

	h, cleanup, err := s.svc.Open(wantProj)
	if err != nil {
		t.Fatalf("open resolved store: %v", err)
	}
	defer cleanup()
	var storedProj string
	if err := h.Store().DB.QueryRow(`SELECT project FROM sessions WHERE id = 'dir-only-1'`).Scan(&storedProj); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if storedProj != wantProj {
		t.Fatalf("stored under %q, want %q", storedProj, wantProj)
	}
}

func TestToolCallRoute_SensitiveFlag(t *testing.T) {
	s, _ := newToolCallsServer(t)
	body := map[string]any{
		"session_id":    "s1",
		"tool_name":     "Read",
		"path":          "/home/u/.env",
		"result_status": "success",
	}
	res := postToolCall(t, s, "s1", body)
	if res.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", res.Code, res.Body.String())
	}

	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	events, _, _, err := h.Memory().SessionChanges(context.Background(), "s1")
	if err != nil {
		t.Fatalf("SessionChanges: %v", err)
	}
	var found bool
	for _, e := range events {
		if e.ActionType == "file_read" && e.IsSensitive {
			found = true
		}
	}
	if !found {
		t.Errorf("no sensitive file_read row for .env; events: %+v", events)
	}
}
