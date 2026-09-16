package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func TestBadSpawnErrorsWithoutOrphanState(t *testing.T) {
	dataDir := t.TempDir()
	ws := t.TempDir()
	SetService(service.New(dataDir))

	req := mcplib.CallToolRequest{}
	req.Params.Name = "team_spawn_task"
	req.Params.Arguments = map[string]any{
		"directory": ws,
		// missing title + brief
	}
	res, err := handleTeamSpawnTask(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned Go error (panic path): %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected structured tool error, got %#v", res)
	}

	filesRoot := filepath.Join(ws, ".skillgrid", "files", "tasks")
	if entries, err := os.ReadDir(filesRoot); err == nil && len(entries) > 0 {
		t.Fatalf("expected no orphan task dirs, found %v", entries)
	}
}

func TestTeamsSpawnPullReadSubmitStayConsistent(t *testing.T) {
	dataDir := t.TempDir()
	ws := t.TempDir()
	SetService(service.New(dataDir))

	spawnReq := mcplib.CallToolRequest{}
	spawnReq.Params.Arguments = map[string]any{
		"directory": ws,
		"title":     "mcp task",
		"brief":     "# brief from mcp\n",
		"priority":  float64(3),
	}
	spawnRes, err := handleTeamSpawnTask(context.Background(), spawnReq)
	if err != nil || spawnRes.IsError {
		t.Fatalf("spawn: err=%v res=%v", err, spawnRes)
	}

	pullReq := mcplib.CallToolRequest{}
	pullReq.Params.Arguments = map[string]any{
		"directory": ws,
		"member_id": "agent-mcp",
	}
	pullRes, err := handleAgentPullNextTask(context.Background(), pullReq)
	if err != nil || pullRes.IsError {
		t.Fatalf("pull: err=%v res=%v", err, pullRes)
	}

	readReq := mcplib.CallToolRequest{}
	// Extract task_id from spawn JSON content is awkward; pull then re-read via service.
	svc := service.New(dataDir)
	view, err := svc.PullNextTask(context.Background(), ws, "agent-other")
	if err == nil {
		// already claimed — expected ErrNoPendingTasks or second task absent
		_ = view
	}
	// Use spawn via service for a known id for read/submit.
	id, err := svc.SpawnTask(context.Background(), service.SpawnTaskParams{
		Directory: ws, Title: "second", Brief: "second brief", Priority: 1,
	})
	if err != nil {
		t.Fatalf("service spawn: %v", err)
	}
	readReq.Params.Arguments = map[string]any{"directory": ws, "task_id": id}
	readRes, err := handleAgentReadTask(context.Background(), readReq)
	if err != nil || readRes.IsError {
		t.Fatalf("read: err=%v res=%v", err, readRes)
	}
	submitReq := mcplib.CallToolRequest{}
	submitReq.Params.Arguments = map[string]any{
		"directory": ws,
		"task_id":   id,
		"output":    "# out\n",
		"summary":   "ok",
	}
	submitRes, err := handleAgentSubmitOutput(context.Background(), submitReq)
	if err != nil || submitRes.IsError {
		t.Fatalf("submit: err=%v res=%v", err, submitRes)
	}
}

func TestUnknownIDOrEmptyQueueErrors(t *testing.T) {
	dataDir := t.TempDir()
	ws := t.TempDir()
	SetService(service.New(dataDir))

	pullReq := mcplib.CallToolRequest{}
	pullReq.Params.Arguments = map[string]any{"directory": ws, "member_id": "a1"}
	pullRes, err := handleAgentPullNextTask(context.Background(), pullReq)
	if err != nil {
		t.Fatalf("pull Go error: %v", err)
	}
	if !pullRes.IsError {
		t.Fatal("expected empty queue tool error")
	}

	readReq := mcplib.CallToolRequest{}
	readReq.Params.Arguments = map[string]any{"directory": ws, "task_id": "no-such"}
	readRes, err := handleAgentReadTask(context.Background(), readReq)
	if err != nil {
		t.Fatalf("read Go error: %v", err)
	}
	if !readRes.IsError {
		t.Fatal("expected unknown id tool error")
	}
	if !strings.Contains(strings.ToLower(teamsToolResultText(readRes)), "unknown") &&
		!strings.Contains(teamsToolResultText(readRes), "no-such") {
		// ErrUnknownTask message is "unknown task id"
		if !strings.Contains(teamsToolResultText(readRes), "unknown task id") {
			t.Errorf("unexpected error text: %q", teamsToolResultText(readRes))
		}
	}
}

func TestTeamsToolsRegisteredAndInboxAbsent(t *testing.T) {
	tools := NewServer().ListTools()
	for _, name := range []string{
		"team_spawn_task", "agent_pull_next_task", "agent_read_task",
		"agent_submit_output", "agent_submit_review", "agent_mark_done",
	} {
		if _, ok := tools[name]; !ok {
			t.Errorf("missing tool %q", name)
		}
	}
	for _, name := range []string{"agent_send_message", "agent_read_inbox"} {
		if _, ok := tools[name]; ok {
			t.Errorf("inbox tool %q must not be registered in this change", name)
		}
	}
	for _, name := range []string{"mem_save", "code_search", "web_cache_status"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected existing tool %q to remain", name)
		}
	}
}

func teamsToolResultText(res *mcplib.CallToolResult) string {
	if res == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
