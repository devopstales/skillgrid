package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"skillgrid-cli/internal/mnemonic/project"
	"skillgrid-cli/internal/mnemonic/store"
)

func TestMemoryHandlerSuccessRawJSON(t *testing.T) {
	const marker = "UniqueMCPMemoryMarker456"

	tests := []struct {
		name    string
		handler func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error)
		prepare func(t *testing.T, root string) mcplib.CallToolRequest
	}{
		{
			name:    "mem_session_start",
			handler: handleMemSessionStart,
			prepare: func(t *testing.T, root string) mcplib.CallToolRequest {
				return mcplib.CallToolRequest{
					Params: mcplib.CallToolParams{
						Name:      "mem_session_start",
						Arguments: map[string]any{"directory": root},
					},
				}
			},
		},
		{
			name:    "mem_search",
			handler: handleMemSearch,
			prepare: func(t *testing.T, root string) mcplib.CallToolRequest {
				startRes, err := handleMemSessionStart(context.Background(), mcplib.CallToolRequest{
					Params: mcplib.CallToolParams{
						Name:      "mem_session_start",
						Arguments: map[string]any{"directory": root},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				startText := assertRawJSONResult(t, startRes)

				var startPayload struct {
					SessionID string `json:"session_id"`
				}
				if err := json.Unmarshal([]byte(startText), &startPayload); err != nil {
					t.Fatal(err)
				}

				saveRes, err := handleMemSave(context.Background(), mcplib.CallToolRequest{
					Params: mcplib.CallToolParams{
						Name: "mem_save",
						Arguments: map[string]any{
							"title":      "Memory test " + marker,
							"type":       "discovery",
							"content":    "**What**: " + marker,
							"session_id": startPayload.SessionID,
						},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				assertRawJSONResult(t, saveRes)

				return mcplib.CallToolRequest{
					Params: mcplib.CallToolParams{
						Name:      "mem_search",
						Arguments: map[string]any{"query": marker},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dataDir := t.TempDir()
			t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
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

			req := tt.prepare(t, root)
			res, err := tt.handler(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			assertRawJSONResult(t, res)
		})
	}
}

func TestMemSessionStartUsesDirectoryProject(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	cwdRoot := t.TempDir()
	targetRoot := t.TempDir()
	for _, dir := range []string{cwdRoot, targetRoot} {
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	initGitRepo(t, cwdRoot)
	initGitRepo(t, targetRoot)

	targetProjectID, err := project.Resolve(targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	cwdProjectID, err := project.Resolve(cwdRoot)
	if err != nil {
		t.Fatal(err)
	}
	if targetProjectID == cwdProjectID {
		t.Fatalf("test requires distinct project IDs, both resolved to %q", targetProjectID)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(cwdRoot); err != nil {
		t.Fatal(err)
	}

	res, err := handleMemSessionStart(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      "mem_session_start",
			Arguments: map[string]any{"directory": targetRoot},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := assertRawJSONResult(t, res)

	var payload struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.SessionID == "" {
		t.Fatal("expected non-empty session_id")
	}

	targetStore, err := store.Open(dataDir, targetProjectID)
	if err != nil {
		t.Fatal(err)
	}
	defer targetStore.Close()

	var targetCount int
	if err := targetStore.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ?`, payload.SessionID).Scan(&targetCount); err != nil {
		t.Fatal(err)
	}
	if targetCount != 1 {
		t.Fatalf("session not found in target project store %q", targetProjectID)
	}

	cwdStore, err := store.Open(dataDir, cwdProjectID)
	if err != nil {
		t.Fatal(err)
	}
	defer cwdStore.Close()

	var cwdCount int
	if err := cwdStore.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ?`, payload.SessionID).Scan(&cwdCount); err != nil {
		t.Fatal(err)
	}
	if cwdCount != 0 {
		t.Fatalf("session incorrectly stored in cwd project store %q", cwdProjectID)
	}
}
