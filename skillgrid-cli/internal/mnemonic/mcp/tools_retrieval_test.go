package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/tiered"
)

func prepRetrievalFixture(t *testing.T) (dataDir, project, path string) {
	t.Helper()
	dataDir = t.TempDir()
	project = "mcpret"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	root := filepath.Join(dataDir, "content")
	_ = os.MkdirAll(root, 0o755)
	path = filepath.Join(root, "note.md")
	body := "# Note\n\nSecretL2PayloadMustStayHidden\n\nUseful overview text about rockets.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := &tiered.Store{DB: st.DB, Summarizer: tiered.HeuristicSummarizer{}}
	if err := ts.GenerateTiers(context.Background(), project, path, "note"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	abs, over := tiered.SidecarPaths(path)
	if _, err := st.DB.Exec(`
		INSERT INTO long_term_memories (project, title, full_path, abstract_path, overview_path, created_at, updated_at)
		VALUES (?, 'note', ?, ?, ?, ?, ?)`, project, path, abs, over, now, now); err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	return dataDir, project, path
}

func TestSemanticSearchMCPOverviewOnly(t *testing.T) {
	_, project, _ := prepRetrievalFixture(t)
	req := mcplib.CallToolRequest{}
	req.Params.Name = "semantic_search"
	req.Params.Arguments = map[string]any{
		"query":   "rockets",
		"project": project,
	}
	res, err := handleSemanticSearch(context.Background(), req)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("error: %v", res.Content)
	}
	text := retrievalToolText(t, res)
	var probe map[string]any
	if err := json.Unmarshal([]byte(text), &probe); err != nil {
		t.Fatal(err)
	}
	results, _ := probe["results"].([]any)
	first, _ := results[0].(map[string]any)
	if _, ok := first["content"]; ok {
		t.Fatal("must not include content field")
	}
	var out service.SemanticSearchResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(out.Results) == 0 || out.Results[0].Overview == "" {
		t.Fatalf("bad result: %+v", out)
	}
}

func TestLoadFullDetailsMCP(t *testing.T) {
	_, project, path := prepRetrievalFixture(t)
	req := mcplib.CallToolRequest{}
	req.Params.Name = "load_full_details"
	req.Params.Arguments = map[string]any{
		"path":    path,
		"project": project,
	}
	res, err := handleLoadFullDetails(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("error: %v", res.Content)
	}
	text := retrievalToolText(t, res)
	if !strings.Contains(text, "SecretL2PayloadMustStayHidden") {
		t.Fatalf("expected L2 body: %s", text)
	}
}

func retrievalToolText(t *testing.T, res *mcplib.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	tc, ok := res.Content[0].(mcplib.TextContent)
	if !ok {
		t.Fatalf("content type %T", res.Content[0])
	}
	return tc.Text
}
