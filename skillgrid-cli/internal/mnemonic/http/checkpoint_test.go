package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/checkpoint"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func postCheckpointClaim(t *testing.T, s *Server, sid string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/sessions/"+sid+"/checkpoint/claim?project="+proj, nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func postToolCallsN(t *testing.T, s *Server, sid string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		res := postToolCall(t, s, sid, map[string]any{
			"tool_name":     "Shell",
			"command":       "echo step-" + string(rune('a'+i)),
			"result_status": "success",
		})
		if res.Code != http.StatusOK && res.Code != http.StatusCreated {
			t.Fatalf("tool call %d: status %d: %s", i+1, res.Code, res.Body.String())
		}
	}
}

func TestCheckpointClaim_DueAndCooldown(t *testing.T) {
	s, _ := newToolCallsServer(t)
	frozen := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return frozen }

	const sid = "cp-due-cooldown"
	postToolCallsN(t, s, sid, 5)

	w, out := postCheckpointClaim(t, s, sid)
	if w.Code != http.StatusOK {
		t.Fatalf("first claim: status %d: %s", w.Code, w.Body.String())
	}
	if out["due"] != true {
		t.Fatalf("first claim due = %v, want true: %v", out["due"], out)
	}
	if out["reason"] != "" {
		t.Errorf("first claim reason = %q, want empty", out["reason"])
	}
	prompt, _ := out["prompt"].(string)
	if prompt == "" {
		t.Fatal("first claim: empty prompt")
	}
	if !strings.Contains(prompt, "Shell") {
		t.Errorf("prompt should mention tool Shell: %q", prompt)
	}

	w2, out2 := postCheckpointClaim(t, s, sid)
	if w2.Code != http.StatusOK {
		t.Fatalf("second claim: status %d", w2.Code)
	}
	if out2["due"] != false || out2["reason"] != checkpoint.ReasonCooldown {
		t.Fatalf("second claim = %+v, want due false cooldown", out2)
	}
	if p, _ := out2["prompt"].(string); p != "" {
		t.Errorf("cooldown claim prompt = %q, want empty", p)
	}

	h, cleanup, err := s.svc.Open(proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	if err := h.Memory().SessionSummary(context.Background(), sid, "## Goal\ncheckpoint reset"); err != nil {
		t.Fatalf("SessionSummary: %v", err)
	}
	// last_memory_write_at and event timestamps share second precision (RFC3339).
	time.Sleep(1100 * time.Millisecond)

	postToolCallsN(t, s, sid, 5)
	s.now = func() time.Time { return frozen.Add(11 * time.Minute) }

	w3, out3 := postCheckpointClaim(t, s, sid)
	if w3.Code != http.StatusOK {
		t.Fatalf("third claim: status %d: %s", w3.Code, w.Body.String())
	}
	if out3["due"] != true {
		t.Fatalf("third claim due = %v, want true: %+v", out3["due"], out3)
	}
	if p, _ := out3["prompt"].(string); p == "" {
		t.Fatal("third claim: empty prompt")
	}
}

func TestCheckpointClaim_UnknownSession(t *testing.T) {
	s, _ := newToolCallsServer(t)
	w, out := postCheckpointClaim(t, s, "no-such-session")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", w.Code, w.Body.String())
	}
	if out["due"] != false || out["reason"] != checkpoint.ReasonUnknownSession {
		t.Fatalf("got %+v, want due false unknown_session", out)
	}
	if p, _ := out["prompt"].(string); p != "" {
		t.Errorf("prompt = %q, want empty", p)
	}
}

func TestCheckpointClaim_Disabled(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	cfgDir := filepath.Join(workDir, ".skillgrid", "config.d")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "mnemonic:\n  hooks:\n    enabled: true\n  checkpoint:\n    enabled: false\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "indexing.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	s := NewServer(service.New(dataDir))
	postToolCallsN(t, s, "s1", 5)

	w, out := postCheckpointClaim(t, s, "s1")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if out["due"] != false || out["reason"] != checkpoint.ReasonDisabled {
		t.Fatalf("got %+v, want due false disabled", out)
	}
}

func TestCheckpointPrompt_Content(t *testing.T) {
	s, _ := newToolCallsServer(t)
	const sid = "cp-prompt-content"
	const obsTitle = "JWT auth decision"
	const secretPreview = "UNIQUE_PREVIEW_DO_NOT_LEAK_8842"

	postToolCall(t, s, sid, map[string]any{
		"tool_name": "Shell", "command": "true", "result_status": "success",
	})

	rr, _ := do(t, s.Handler(), http.MethodPost, "/observations?project="+proj, map[string]any{
		"session_id": sid,
		"type":       "decision",
		"title":      obsTitle,
		"content":    "Bearer only.",
		"scope":      "project",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /observations: status %d: %s", rr.Code, rr.Body.String())
	}
	time.Sleep(1100 * time.Millisecond)

	postToolCallsN(t, s, sid, 4)
	res := postToolCall(t, s, sid, map[string]any{
		"tool_name":       "Read",
		"path":            "internal/foo.go",
		"result_status":   "success",
		"content_preview": secretPreview,
	})
	if res.Code != http.StatusOK && res.Code != http.StatusCreated {
		t.Fatalf("tool call: status %d", res.Code)
	}

	w, out := postCheckpointClaim(t, s, sid)
	if w.Code != http.StatusOK || out["due"] != true {
		t.Fatalf("claim: status %d body %+v", w.Code, out)
	}
	prompt, _ := out["prompt"].(string)
	if !strings.Contains(prompt, obsTitle) {
		t.Errorf("prompt missing observation title %q:\n%s", obsTitle, prompt)
	}
	if !strings.Contains(prompt, sid) {
		t.Errorf("prompt missing session id %q:\n%s", sid, prompt)
	}
	if strings.Contains(prompt, secretPreview) {
		t.Errorf("prompt leaked content_preview %q:\n%s", secretPreview, prompt)
	}
}
