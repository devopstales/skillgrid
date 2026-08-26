package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestWebCacheIntegration(t *testing.T) {
	root := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	const marker = "UniqueMCPWebMarker123"
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, root)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	content := "Cached snapshot about " + marker + " middleware patterns."

	saveRes, err := handleWebCacheSave(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name: "web_cache_save",
			Arguments: map[string]any{
				"source":      "context7",
				"library_id":  "/vercel/next.js",
				"version_tag": "v15",
				"query":       marker,
				"title":       "Next.js middleware",
				"content":     content,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	saveText := assertRawJSONResult(t, saveRes)

	var savePayload struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(saveText), &savePayload); err != nil {
		t.Fatal(err)
	}
	if savePayload.ID <= 0 {
		t.Fatalf("expected positive id, got %d", savePayload.ID)
	}

	lookupRes, err := handleWebCacheLookup(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name: "web_cache_lookup",
			Arguments: map[string]any{
				"source":      "context7",
				"library_id":  "/vercel/next.js",
				"version_tag": "v15",
				"query":       marker,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	lookupText := assertRawJSONResult(t, lookupRes)

	var lookupPayload struct {
		Status string `json:"status"`
		Fresh  bool   `json:"fresh"`
		ID     int64  `json:"id"`
	}
	if err := json.Unmarshal([]byte(lookupText), &lookupPayload); err != nil {
		t.Fatal(err)
	}
	if lookupPayload.Status != "hit" {
		t.Fatalf("status=%q, want hit", lookupPayload.Status)
	}
	if !lookupPayload.Fresh {
		t.Fatal("expected fresh=true on new entry")
	}
	if lookupPayload.ID != savePayload.ID {
		t.Fatalf("lookup id=%d, save id=%d", lookupPayload.ID, savePayload.ID)
	}

	searchRes, err := handleWebCacheSearch(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name: "web_cache_search",
			Arguments: map[string]any{
				"query": marker,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	searchText := assertRawJSONResult(t, searchRes)

	var searchPayload struct {
		Entries []struct {
			ID    int64  `json:"id"`
			Query string `json:"query"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(searchText), &searchPayload); err != nil {
		t.Fatal(err)
	}
	if len(searchPayload.Entries) == 0 {
		t.Fatal("expected at least one search hit")
	}
	if searchPayload.Entries[0].Query != marker {
		t.Fatalf("query=%q, want %q", searchPayload.Entries[0].Query, marker)
	}

	getRes, err := handleWebCacheGet(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      "web_cache_get",
			Arguments: map[string]any{"id": float64(savePayload.ID)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	getText := assertRawJSONResult(t, getRes)

	var getPayload struct {
		ID      int64  `json:"id"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(getText), &getPayload); err != nil {
		t.Fatal(err)
	}
	if getPayload.ID != savePayload.ID {
		t.Fatalf("get id=%d, save id=%d", getPayload.ID, savePayload.ID)
	}
	if !strings.Contains(getPayload.Content, marker) {
		t.Fatalf("content=%q, want to contain %q", getPayload.Content, marker)
	}
}
