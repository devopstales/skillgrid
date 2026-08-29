package integration

import (
	stdhttp "net/http"

	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	mnemonicstore "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// TestMCPAndHTTPShareStore verifies the integration scenario: data written via
// the MCP transport (mem_save) is readable via the HTTP transport (GET
// /search) using the same SQLite store.
func TestMCPAndHTTPShareStore(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	// Pin the working directory so MCP/HTTP handlers resolve a deterministic
	// project id via .skillgrid/config.json.
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".skillgrid"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := `{"project":"shared-integ"}`
	if err := os.WriteFile(filepath.Join(workspace, ".skillgrid", "config.json"), []byte(cfg), 0o644); err != nil {
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
	mcp.SetService(svc)
	srv := mcp.NewServer()
	ctx := context.Background()

	// 1. MCP transport: create a session, then save an observation.
	if err := dispatch(ctx, srv, "mem_session_start", map[string]any{}); err != nil {
		t.Fatalf("mcp mem_session_start: %v", err)
	}

	// Pull the session id from the shared store.
	st, err := mnemonicstore.Open(dataDir, "shared-integ")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	var sessionID string
	if err := st.DB.QueryRow(`SELECT id FROM sessions WHERE project = 'shared-integ' ORDER BY rowid DESC LIMIT 1`).Scan(&sessionID); err != nil {
		st.Close()
		t.Fatalf("read session id: %v", err)
	}
	st.Close()
	if sessionID == "" {
		t.Fatalf("no session created by MCP transport")
	}
	if err := dispatch(ctx, srv, "mem_save", map[string]any{
		"session_id": sessionID,
		"type":       "decision",
		"title":      "shared-store probe",
		"content":    "written via mcp over stdio",
	}); err != nil {
		t.Fatalf("mcp mem_save: %v", err)
	}

	// 2. HTTP transport: the same store must expose the observation via GET /search.
	handler := mnemonichttp.NewServer(svc).Handler()
	rr, out := doHTTP(t, handler, stdhttp.MethodGet, "/search?project=shared-integ&query=mcp", nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("http search: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	obs, ok := out["observations"].([]any)
	if !ok || len(obs) == 0 {
		t.Fatalf("expected the MCP-written observation over HTTP, got %v", out["observations"])
	}
}

// dispatch runs a single MCP tool by name via the registered ServerTool handler.
func dispatch(ctx context.Context, srv *mcp.Server, name string, args map[string]any) error {
	tools := srv.ListTools()
	st, ok := tools[name]
	if !ok {
		return &mcpError{msg: "tool not found: " + name}
	}
	req := mcplib.CallToolRequest{}
	if args == nil {
		args = map[string]any{}
	}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := st.Handler(ctx, req)
	if err != nil {
		return err
	}
	if res == nil {
		return &mcpError{msg: "nil result from tool"}
	}
	if res.IsError {
		return &mcpError{msg: "tool returned error result"}
	}
	return nil
}

type mcpError struct{ msg string }

func (e *mcpError) Error() string { return e.msg }

func doHTTP(t *testing.T, h stdhttp.Handler, method, target string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(method, target, reader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	var out map[string]any
	_ = json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&out)
	return rr, out
}
