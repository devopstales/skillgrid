package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func registerCompactionTools(s *server.MCPServer) {
	s.AddTool(mnemonicCommitTool(), handleMnemonicCommit)
}

func mnemonicCommitTool() mcplib.Tool {
	return mcplib.NewTool("mnemonic_commit",
		mcplib.WithDescription("Explicitly commit lessons into long-term memory (L2 durable; L0/L1 async). Does not run on session end."),
		mcplib.WithString("title", mcplib.Description("Memory title")),
		mcplib.WithString("lessons_learned", mcplib.Description("Lesson body used as L2 when content is empty")),
		mcplib.WithString("content", mcplib.Description("Optional full L2 markdown")),
		mcplib.WithString("source_link", mcplib.Description("Optional source link")),
		mcplib.WithString("task_id", mcplib.Description("Optional team task id (future 001 hook)")),
		mcplib.WithString("project", mcplib.Description("Project id (defaults to CWD resolve)")),
	)
}

func handleMnemonicCommit(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	projectID, err := projectIDFor(svc, req.GetString("project", ""))
	if err != nil {
		return toolError(err)
	}
	out, err := svc.MnemonicCommit(ctx, projectID, service.MnemonicCommitInput{
		Title:          req.GetString("title", ""),
		LessonsLearned: req.GetString("lessons_learned", ""),
		Content:        req.GetString("content", ""),
		SourceLink:     req.GetString("source_link", ""),
		TaskID:         req.GetString("task_id", ""),
	}, nil)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"memory_id": out.MemoryID,
		"paths": map[string]string{
			"full_path": out.FullPath,
		},
		"title": out.Title,
	})
}
