package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const proj = "http-test"

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler()
}

// seedStore opens the store for proj and inserts a session + observation so
// read endpoints have data.
func seedStore(t *testing.T, dataDir string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed store: %v", err)
	}
	now := "2026-01-01T00:00:00Z"
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s1', ?, '/tmp', ?, '## Goal\nseed', 'ended')`, proj, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at) VALUES ('s1','decision','obs-title','obs-body', ?, 'project','h','0',?,?)`, proj, now, now); err != nil {
		t.Fatalf("seed observation: %v", err)
	}
	st.Close()
}

func do(t *testing.T, h http.Handler, method, target string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(rr.Body.Bytes()))
	_ = dec.Decode(&out)
	return rr, out
}

func TestHealth(t *testing.T) {
	h := newHandler(t)
	rr, out := do(t, h, http.MethodGet, "/health", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if out["status"] != "ok" {
		t.Errorf("expected status ok, got %v", out["status"])
	}
}

func TestContextEndpoint(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/context?project=local", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := out["sessions"]; !ok {
		t.Errorf("expected sessions key, got %v", out)
	}
}

func TestSessionCreateWithTitle(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	// Pin a project id so the session is scoped deterministically.
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".skillgrid"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".skillgrid", "config.json"), []byte(`{"project":"titled"}`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldwd)

	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodPost, "/sessions?directory=.", map[string]any{
		"title": "Skillgrid CLI dashboard status card updates",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	sessionID, _ := out["session_id"].(string)
	if sessionID == "" {
		t.Fatalf("expected session_id, got %v", out)
	}
	if out["title"] != "Skillgrid CLI dashboard status card updates" {
		t.Errorf("expected title echoed, got %v", out["title"])
	}
	// The dashboard list (GET /context) must expose the title.
	rr2, ctx := do(t, h, http.MethodGet, "/context?project=titled", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	sessions, ok := ctx["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected 1 session in context, got %v", ctx["sessions"])
	}
	sess, ok := sessions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map session, got %T", sessions[0])
	}
	if sess["title"] != "Skillgrid CLI dashboard status card updates" {
		t.Errorf("expected session title, got %v", sess["title"])
	}
}

func TestObservationsRecent(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/observations/recent?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	sessions, ok := out["sessions"].([]any)
	if !ok {
		t.Fatalf("expected sessions array, got %v", out["sessions"])
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestObservationCreateAndSearch(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	// Need a session first (FK).
	seedStore(t, dataDir)
	body := map[string]any{
		"session_id": "s1",
		"type":       "decision",
		"title":      "create-via-http",
		"content":    "created over http",
		"scope":      "project",
	}
	rr, out := do(t, h, http.MethodPost, "/observations?project="+proj, body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	id, ok := out["id"].(float64)
	if !ok || id <= 0 {
		t.Fatalf("expected positive id, got %v", out["id"])
	}
	// Search should find it.
	rr2, out2 := do(t, h, http.MethodGet, "/search?project="+proj+"&query=http", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}
	obs, ok := out2["observations"].([]any)
	if !ok || len(obs) == 0 {
		t.Errorf("expected observations, got %v", out2["observations"])
	}
}

func TestCodeStatus(t *testing.T) {
	d := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", d)
	svc := service.New(d)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/code/status?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if _, ok := out["file_count"]; !ok {
		t.Errorf("expected file_count key, got %v", out)
	}
}

func TestCodeIndexEndpoint(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	// Create a tiny repo to index at "." (cwd in the test process).
	rr, out := do(t, h, http.MethodPost, "/code/index?dir=.", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := out["files_indexed"]; !ok {
		t.Errorf("expected files_indexed key, got %v", out)
	}
}

func TestWebLookupMiss(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/web/lookup?project="+proj+"&source=context7&library_id=/o&p&query=q", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if out["status"] != "miss" {
		t.Errorf("expected miss, got %v", out["status"])
	}
}

func TestWebCacheSaveAndSearch(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	body := map[string]any{
		"source":     "exa",
		"content":    "express auth howto",
		"query":      "express auth",
		"session_id": "s1",
	}
	rr, _ := do(t, h, http.MethodPost, "/web/cache?project="+proj, body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	rr2, out2 := do(t, h, http.MethodGet, "/web/search?project="+proj+"&query=express&fresh_only=false", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}
	entries, ok := out2["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Errorf("expected 1 entry, got %v", out2["entries"])
	}
}

func TestMemoryStatus(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/memory/status?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if c, ok := out["observation_count"].(float64); !ok || c != 1 {
		t.Errorf("expected observation_count=1, got %v", out["observation_count"])
	}
	if byType, ok := out["by_type"].(map[string]any); !ok {
		t.Errorf("expected by_type object, got %v", out["by_type"])
	} else if decision, ok := byType["decision"].(float64); !ok || decision != 1 {
		t.Errorf("expected by_type.decision=1, got %v", byType["decision"])
	}
}

func TestWebStatus(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/web/status?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if _, ok := out["total_entries"]; !ok {
		t.Errorf("expected total_entries key, got %v", out)
	}
}

func TestProjectsEndpoint(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	if st, err := store.Open(dataDir, proj); err != nil {
		t.Fatalf("seed: %v", err)
	} else {
		st.Close()
	}
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/projects", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	projects, ok := out["projects"].([]any)
	if !ok || len(projects) != 1 {
		t.Fatalf("expected one project, got %v", out["projects"])
	}
}

func TestObservationsListEndpoint(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/observations?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	obs, ok := out["observations"].([]any)
	if !ok || len(obs) != 1 {
		t.Fatalf("expected one observation, got %v", out["observations"])
	}
}

func TestUIRoutes(t *testing.T) {
	h := newHandler(t)
	cases := []struct {
		path   string
		status int
		prefix string
	}{
		{"/", http.StatusOK, "text/html"},
		{"/app.js", http.StatusOK, "javascript"},
		{"/openapi.yaml", http.StatusOK, "yaml"},
		{"/swagger-ui", http.StatusOK, "text/html"},
		{"/swagger-ui/swagger-ui.css", http.StatusOK, "text/css"},
		{"/swagger-ui/swagger-ui-bundle.js", http.StatusOK, "javascript"},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != c.status {
			t.Errorf("%s: expected %d, got %d", c.path, c.status, rr.Code)
			continue
		}
		ct := rr.Header().Get("Content-Type")
		if !strings.Contains(ct, c.prefix) {
			t.Errorf("%s: content type %q, want substring %q", c.path, ct, c.prefix)
		}
	}
}

func TestWebEntryEndpoint(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := st.DB.Exec(`INSERT INTO web_cache (project, source, cache_key, content, content_hash, created_at, fetched_at, expires_at) VALUES (?,?,?,?,?,?,?,?)`,
		proj, "manual", "test-key", "cached body", "hash1", now, now, now)
	if err != nil {
		t.Fatalf("seed web: %v", err)
	}
	id, _ := res.LastInsertId()
	st.Close()

	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/web/entry/"+strconv.FormatInt(id, 10)+"?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if out["content"] != "cached body" {
		t.Errorf("expected content, got %v", out["content"])
	}

	rr404, _ := do(t, h, http.MethodGet, "/web/entry/999999?project="+proj, nil)
	if rr404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing entry, got %d", rr404.Code)
	}
}

func TestCodeFilesEndpoint(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	if st, err := store.Open(dataDir, proj); err != nil {
		t.Fatalf("seed: %v", err)
	} else {
		if _, err := st.DB.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('src/main.go', 1, 12, 'h', '2026-01-01T00:00:00Z')`); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		st.Close()
	}
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	rr, out := do(t, h, http.MethodGet, "/code/files?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	files, ok := out["files"].([]any)
	if !ok || len(files) != 1 || files[0] != "src/main.go" {
		t.Fatalf("expected src/main.go, got %v", out["files"])
	}
}

func TestAuthRequiredWhenTokenSet(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret-token")
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/observations?project="+proj, bytes.NewReader([]byte("{}")))
	req.Header.Set("Authorization", "Bearer wrong")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/observations?project="+proj, bytes.NewReader([]byte("{}")))
	req2.Header.Set("Authorization", "Bearer secret-token")
	h.ServeHTTP(rr2, req2)
	if rr2.Code == http.StatusUnauthorized {
		t.Errorf("expected to pass auth with correct token, got 401")
	}
}
