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

	mcplib "github.com/mark3labs/mcp-go/mcp"

	mnemonighttp "skillgrid-cli/internal/mnemonic/http"
	"skillgrid-cli/internal/mnemonic/memory"
	mnemcp "skillgrid-cli/internal/mnemonic/mcp"
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
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	mcpRepo := t.TempDir()
	httpRepo := t.TempDir()
	for _, dir := range []string{mcpRepo, httpRepo} {
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	const (
		title    = "Parity test observation"
		content  = "Same content for parity check"
		typ      = "decision"
		scope    = "personal"
		topicKey = "decision/parity-test"
	)

	svc := service.New(dataDir)

	mcpSessionID, mcpProjectID, err := svc.SessionStart(t.Context(), mcpRepo)
	if err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(mcpRepo); err != nil {
		t.Fatal(err)
	}

	mcpRes, err := mnemcp.InvokeMemSave(t.Context(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name: "mem_save",
			Arguments: map[string]any{
				"title":      title,
				"type":       typ,
				"content":    content,
				"session_id": mcpSessionID,
				"scope":      scope,
				"topic_key":  topicKey,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mcpID := parseToolResultID(t, mcpRes)

	mcpObs, err := svc.GetObservation(t.Context(), mcpProjectID, mcpID)
	if err != nil {
		t.Fatal(err)
	}

	httpSessionID, httpProjectID, err := svc.SessionStart(t.Context(), httpRepo)
	if err != nil {
		t.Fatal(err)
	}

	httpServer := mnemonighttp.NewServer(svc)
	ts := httptest.NewServer(httpServer.Handler())
	t.Cleanup(ts.Close)

	httpBody, _ := json.Marshal(map[string]string{
		"session_id": httpSessionID,
		"type":       typ,
		"title":      title,
		"content":    content,
		"project":    httpProjectID,
		"scope":      scope,
		"topic_key":  topicKey,
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

	httpID := parseJSONID(t, httpRes.Body)

	httpObs, err := svc.GetObservation(t.Context(), httpProjectID, httpID)
	if err != nil {
		t.Fatal(err)
	}

	assertObservationParityFields(t, mcpObs, httpObs)
	if mcpObs.Scope != "user" {
		t.Fatalf("scope personal→user: mcp scope=%q", mcpObs.Scope)
	}
	if httpObs.Scope != "user" {
		t.Fatalf("scope personal→user: http scope=%q", httpObs.Scope)
	}
}

func parseToolResultID(t *testing.T, res *mcplib.CallToolResult) int64 {
	t.Helper()
	if res == nil {
		t.Fatal("nil tool result")
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(mcplib.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return parseJSONID(t, bytes.NewReader([]byte(tc.Text)))
}

func parseJSONID(t *testing.T, r io.Reader) int64 {
	t.Helper()
	var payload struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID == 0 {
		t.Fatal("expected non-zero id")
	}
	return payload.ID
}

func assertObservationParityFields(t *testing.T, mcpObs, httpObs memory.Observation) {
	t.Helper()
	checks := []struct {
		name    string
		mcpVal  string
		httpVal string
	}{
		{"normalized_hash", mcpObs.NormalizedHash, httpObs.NormalizedHash},
		{"scope", mcpObs.Scope, httpObs.Scope},
		{"title", mcpObs.Title, httpObs.Title},
		{"content", mcpObs.Content, httpObs.Content},
		{"topic_key", mcpObs.TopicKey, httpObs.TopicKey},
	}
	for _, c := range checks {
		if c.mcpVal != c.httpVal {
			t.Fatalf("%s: mcp=%q http=%q", c.name, c.mcpVal, c.httpVal)
		}
	}
}
