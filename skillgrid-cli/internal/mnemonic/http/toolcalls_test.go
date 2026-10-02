package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func newToolCallsServer(t *testing.T) (*Server, string) {
	t.Helper()
	// Isolate config.d/indexing.yaml in a throwaway directory so parallel
	// tests (and SIGKILL) never mutate the repo CWD.
	workDir := t.TempDir()
	t.Chdir(workDir)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	cfgDir := filepath.Join(workDir, "config.d")
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
	for _, cmd := range []string{"ls", "pwd"} {
		res := postToolCall(t, s, "cursor-conv-1", map[string]any{
			"agent":     "cursor",
			"directory": t.TempDir(),
			"tool_name": "Shell",
			"command":   cmd,
		})
		if res.Code != http.StatusOK {
			t.Fatalf("status %d, want 200: %s", res.Code, res.Body.String())
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
