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

func TestToolCallRoute_UnknownSession(t *testing.T) {
	s, _ := newToolCallsServer(t)
	body := map[string]any{
		"session_id": "no-such-session",
		"tool_name":  "bash",
		"command":    "ls",
	}
	res := postToolCall(t, s, "no-such-session", body)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404: %s", res.Code, res.Body.String())
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
