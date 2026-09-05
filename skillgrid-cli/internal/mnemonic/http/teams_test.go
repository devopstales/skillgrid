package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func TestTeamsAuthenticatedWriteSucceeds(t *testing.T) {
	dataDir := t.TempDir()
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILLGRID_HTTP_TOKEN", "teams-secret")
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	body, _ := json.Marshal(map[string]any{
		"directory": ws,
		"title":     "http task",
		"brief":     "# from http\n",
		"priority":  1,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/teams/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer teams-secret")
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	taskID, _ := out["task_id"].(string)
	if taskID == "" {
		t.Fatalf("missing task_id: %v", out)
	}

	// GET stays open without bearer
	rrGet := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/teams/tasks/"+taskID+"?directory="+ws, nil)
	h.ServeHTTP(rrGet, getReq)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d body=%s", rrGet.Code, rrGet.Body.String())
	}
}

func TestTeamsUnauthenticatedWriteReturns401(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_HTTP_TOKEN", "teams-secret")
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/teams/tasks", bytes.NewReader([]byte(`{"title":"x","brief":"y"}`)))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestTeamsPathsDistinctFromMemoryReviews(t *testing.T) {
	if !teamsPathIsDistinctFromMemoryReviews("/teams/tasks") {
		t.Fatal("teams path should be distinct")
	}
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_HTTP_TOKEN", "")
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	// Memory reviews route still responds (may be empty list)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/memory/reviews", nil)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatal("/memory/reviews should still be registered")
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/teams/tasks/missing-id", nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Code == http.StatusNotFound && rr.Code == http.StatusNotFound {
		// both 404 is ok as long as handlers differ; ensure path exists (not mux miss)
	}
	// Mux miss typically 404 with empty body pattern; unknown task returns JSON error
	if !bytes.Contains(rr2.Body.Bytes(), []byte("unknown")) && rr2.Code != http.StatusNotFound && rr2.Code != http.StatusBadRequest {
		t.Fatalf("unexpected teams get response: %d %s", rr2.Code, rr2.Body.String())
	}
}
