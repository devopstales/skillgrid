package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	mnemonighttp "skillgrid-cli/internal/mnemonic/http"
	"skillgrid-cli/internal/mnemonic/service"
)

func TestHTTPSessionsAndSearch(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := t.TempDir()

	svc := service.New(dataDir)
	server := mnemonighttp.NewServer(svc)
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)

	// POST /sessions {directory: tempRepo}
	sessionBody, _ := json.Marshal(map[string]string{"directory": repoDir})
	sessionRes, err := http.Post(ts.URL+"/sessions", "application/json", bytes.NewReader(sessionBody))
	if err != nil {
		t.Fatal(err)
	}
	defer sessionRes.Body.Close()
	if sessionRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sessionRes.Body)
		t.Fatalf("POST /sessions status=%d body=%s", sessionRes.StatusCode, body)
	}

	var sessionPayload struct {
		SessionID string `json:"session_id"`
		Project   string `json:"project"`
	}
	if err := json.NewDecoder(sessionRes.Body).Decode(&sessionPayload); err != nil {
		t.Fatal(err)
	}
	if sessionPayload.SessionID == "" {
		t.Fatal("expected session_id")
	}
	if sessionPayload.Project == "" {
		t.Fatal("expected project")
	}

	// POST /observations {...}
	const keyword = "UniqueHTTPKeyword789"
	obsBody, _ := json.Marshal(map[string]string{
		"session_id": sessionPayload.SessionID,
		"type":       "decision",
		"title":      "HTTP test observation",
		"content":    "Testing HTTP save with keyword " + keyword,
		"project":    sessionPayload.Project,
		"scope":      "project",
	})
	obsRes, err := http.Post(ts.URL+"/observations", "application/json", bytes.NewReader(obsBody))
	if err != nil {
		t.Fatal(err)
	}
	defer obsRes.Body.Close()
	if obsRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(obsRes.Body)
		t.Fatalf("POST /observations status=%d body=%s", obsRes.StatusCode, body)
	}

	// GET /search?q=keyword → 1 hit
	searchRes, err := http.Get(ts.URL + "/search?q=" + keyword + "&project=" + sessionPayload.Project)
	if err != nil {
		t.Fatal(err)
	}
	defer searchRes.Body.Close()
	if searchRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(searchRes.Body)
		t.Fatalf("GET /search status=%d body=%s", searchRes.StatusCode, body)
	}

	var searchPayload struct {
		Observations []map[string]any `json:"observations"`
	}
	if err := json.NewDecoder(searchRes.Body).Decode(&searchPayload); err != nil {
		t.Fatal(err)
	}
	if len(searchPayload.Observations) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(searchPayload.Observations))
	}

	// GET /health → ok
	healthRes, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer healthRes.Body.Close()
	if healthRes.StatusCode != http.StatusOK {
		t.Fatalf("GET /health status=%d", healthRes.StatusCode)
	}

	var healthPayload struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if err := json.NewDecoder(healthRes.Body).Decode(&healthPayload); err != nil {
		t.Fatal(err)
	}
	if healthPayload.Status != "ok" {
		t.Fatalf("health status=%q", healthPayload.Status)
	}
	if healthPayload.Service != "skillgrid-mnemonic" {
		t.Fatalf("health service=%q", healthPayload.Service)
	}
}

func TestHTTPBearerAuthOnWriteRoutes(t *testing.T) {
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret-token")

	dataDir := t.TempDir()
	repoDir := t.TempDir()
	svc := service.New(dataDir)
	server := mnemonighttp.NewServer(svc)
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"directory": repoDir})
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/sessions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", res.StatusCode)
	}

	req2, err := http.NewRequest(http.MethodPost, ts.URL+"/sessions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer secret-token")
	res2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res2.Body)
		t.Fatalf("expected 200 with token, got %d body=%s", res2.StatusCode, body)
	}
}

func TestMCPAndHTTPObservationParity(t *testing.T) {
	// Verify MCP mem_save and HTTP POST /observations produce identical DB rows.
	repoDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	svc := service.New(dataDir)
	sessionID, projectID, err := svc.SessionStart(t.Context(), repoDir)
	if err != nil {
		t.Fatal(err)
	}

	httpServer := mnemonighttp.NewServer(svc)
	ts := httptest.NewServer(httpServer.Handler())
	t.Cleanup(ts.Close)

	const title = "Parity test observation"
	const content = "Same content for parity check"
	const typ = "decision"

	httpBody, _ := json.Marshal(map[string]string{
		"session_id": sessionID,
		"type":       typ,
		"title":      title,
		"content":    content,
		"project":    projectID,
	})
	httpRes, err := http.Post(ts.URL+"/observations", "application/json", bytes.NewReader(httpBody))
	if err != nil {
		t.Fatal(err)
	}
	defer httpRes.Body.Close()
	if httpRes.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpRes.Body)
		t.Fatalf("HTTP save failed: %d %s", httpRes.StatusCode, body)
	}

	var httpPayload struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(httpRes.Body).Decode(&httpPayload); err != nil {
		t.Fatal(err)
	}

	hits, err := svc.SearchObservations(t.Context(), projectID, "Parity", "any", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(hits))
	}
	if hits[0].ID != httpPayload.ID {
		t.Fatalf("id mismatch: db=%d http=%d", hits[0].ID, httpPayload.ID)
	}
	if hits[0].Title != title {
		t.Fatalf("title=%q", hits[0].Title)
	}
}
