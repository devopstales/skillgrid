package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func postCtxCapture(t *testing.T, s *Server, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/ctx/capture?project="+proj, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

func TestCtxCapture_StoresAndReturnsPointer(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	// Oversized output (> SandboxThreshold = 4096 bytes) so the gate trips.
	content := strings.Repeat("sandbox line ", 400) // ~5200 bytes
	res := postCtxCapture(t, s, map[string]any{
		"session_id": "s-ctx-1",
		"tool_name":  "bash",
		"content":    content,
	})
	if res.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", res.Code, res.Body.String())
	}
	var out struct {
		SandboxID int64  `json:"sandbox_id"`
		Gated     bool   `json:"gated"`
		Pointer   string `json:"pointer"`
		Summary   string `json:"summary"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.SandboxID <= 0 {
		t.Fatalf("sandbox_id = %d, want > 0", out.SandboxID)
	}
	if !out.Gated {
		t.Fatalf("gated = false, want true for oversized content: %+v", out)
	}
	if out.Pointer == "" {
		t.Fatal("pointer empty, want the ctx_search hint")
	}
}

func TestCtxCapture_RequiresFields(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()
	res := postCtxCapture(t, s, map[string]any{"session_id": "s1"})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400: %s", res.Code, res.Body.String())
	}
}

func TestCtxSearch_ReturnsFusedRows(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	// Seed one sandbox row with a distinctive token.
	seed := postCtxCapture(t, s, map[string]any{
		"session_id": "s-ctx-2",
		"tool_name":  "bash",
		"content":    "needle-zebra-9988 unique marker for the sandbox leg",
	})
	if seed.Code != http.StatusOK {
		t.Fatalf("seed status %d: %s", seed.Code, seed.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/ctx/search?project="+proj+"&query=needle-zebra-9988", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("search status %d, want 200: %s", w.Code, w.Body.String())
	}
	var out struct {
		Results []map[string]any `json:"results"`
		Count   int              `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Count != 1 {
		t.Fatalf("count = %d, want 1: %+v", out.Count, out)
	}
	if got := out.Results[0]["source"]; got != "sandbox" {
		t.Fatalf("source = %v, want sandbox", got)
	}
}

func TestCtxGet_ReturnsStoredOutput(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()
	seed := postCtxCapture(t, s, map[string]any{
		"session_id": "s-ctx-4",
		"tool_name":  "bash",
		"content":    "the full text you asked to keep",
	})
	if seed.Code != http.StatusOK {
		t.Fatalf("seed status %d: %s", seed.Code, seed.Body.String())
	}
	var seedOut struct {
		SandboxID int64 `json:"sandbox_id"`
	}
	if err := json.Unmarshal(seed.Body.Bytes(), &seedOut); err != nil {
		t.Fatalf("decode: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/ctx/get/%d?project=%s", seedOut.SandboxID, proj), nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status %d, want 200: %s", w.Code, w.Body.String())
	}
	var out struct {
		ToolName string `json:"tool_name"`
		Output   string `json:"output"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ToolName != "bash" || out.Output != "the full text you asked to keep" {
		t.Fatalf("unexpected row: %+v", out)
	}

	// Missing id → 404.
	req404 := httptest.NewRequest(http.MethodGet, "/ctx/get/999999?project="+proj, nil)
	w404 := httptest.NewRecorder()
	s.Handler().ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("missing id status %d, want 404", w404.Code)
	}
}

func TestCtxStats_LoopbackOnly(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()
	// Loopback passes.
	req := httptest.NewRequest(http.MethodGet, "/ctx/stats?project="+proj, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("loopback status %d, want 200: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := out["sandbox_rows"]; !ok {
		t.Fatalf("missing sandbox_rows: %+v", out)
	}
	// Non-loopback is rejected before the store opens.
	req2 := httptest.NewRequest(http.MethodGet, "/ctx/stats?project="+proj, nil)
	req2.RemoteAddr = "203.0.113.7:12345"
	w2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("non-loopback status %d, want 403", w2.Code)
	}
}

func TestCtxSessionPurge_DeletesRows(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()
	seed := postCtxCapture(t, s, map[string]any{
		"session_id": "s-ctx-3",
		"tool_name":  "bash",
		"content":    "purge me",
	})
	if seed.Code != http.StatusOK {
		t.Fatalf("seed status %d: %s", seed.Code, seed.Body.String())
	}

	req := httptest.NewRequest(http.MethodDelete, "/ctx/sessions/s-ctx-3?project="+proj, nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("purge status %d, want 200: %s", w.Code, w.Body.String())
	}
	var out struct {
		Purged int `json:"purged"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Purged != 1 {
		t.Fatalf("purged = %d, want 1", out.Purged)
	}
}
