package mcp

import (
	"context"
	"os"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func registerTeamsTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{teamSpawnTaskTool(), handleTeamSpawnTask},
		{agentPullNextTaskTool(), handleAgentPullNextTask},
		{agentReadTaskTool(), handleAgentReadTask},
		{agentSubmitOutputTool(), handleAgentSubmitOutput},
		{agentSubmitReviewTool(), handleAgentSubmitReview},
		{agentMarkDoneTool(), handleAgentMarkDone},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func teamSpawnTaskTool() mcplib.Tool {
	return mcplib.NewTool("team_spawn_task",
		mcplib.WithDescription("Spawn a pending team task with a markdown brief written to the project files tree."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Short task title")),
		mcplib.WithString("brief", mcplib.Required(), mcplib.Description("Markdown brief content")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
		mcplib.WithString("team_id", mcplib.Description("Team id (defaults to default)")),
		mcplib.WithNumber("priority", mcplib.Description("Priority; higher claims first (default 0)")),
		mcplib.WithString("created_by", mcplib.Description("Optional creator id")),
	)
}

func agentPullNextTaskTool() mcplib.Tool {
	return mcplib.NewTool("agent_pull_next_task",
		mcplib.WithDescription("Claim the highest-priority pending team task for a member."),
		mcplib.WithString("member_id", mcplib.Required(), mcplib.Description("Team member / agent id claiming the task")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
	)
}

func agentReadTaskTool() mcplib.Tool {
	return mcplib.NewTool("agent_read_task",
		mcplib.WithDescription("Read task metadata and brief markdown from disk."),
		mcplib.WithString("task_id", mcplib.Required(), mcplib.Description("Task id")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
	)
}

func agentSubmitOutputTool() mcplib.Tool {
	return mcplib.NewTool("agent_submit_output",
		mcplib.WithDescription("Write task output.md and advance status to review_spec."),
		mcplib.WithString("task_id", mcplib.Required(), mcplib.Description("Task id")),
		mcplib.WithString("output", mcplib.Required(), mcplib.Description("Markdown output content")),
		mcplib.WithString("summary", mcplib.Description("Optional one-line summary for SQL")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
	)
}

func agentSubmitReviewTool() mcplib.Tool {
	return mcplib.NewTool("agent_submit_review",
		mcplib.WithDescription("Submit a peer review (spec_compliance or code_quality) with markdown comments."),
		mcplib.WithString("task_id", mcplib.Required(), mcplib.Description("Task id")),
		mcplib.WithString("reviewer_id", mcplib.Required(), mcplib.Description("Reviewer member id")),
		mcplib.WithBoolean("passed", mcplib.Required(), mcplib.Description("Whether the review passed")),
		mcplib.WithString("comments", mcplib.Description("Markdown review comments")),
		mcplib.WithString("review_type", mcplib.Description("spec_compliance (default) or code_quality")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
	)
}

func agentMarkDoneTool() mcplib.Tool {
	return mcplib.NewTool("agent_mark_done",
		mcplib.WithDescription("Mark a team task as done."),
		mcplib.WithString("task_id", mcplib.Required(), mcplib.Description("Task id")),
		mcplib.WithString("directory", mcplib.Description("Workspace directory (defaults to cwd)")),
	)
}

func handleTeamSpawnTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	title, err := req.RequireString("title")
	if err != nil {
		return toolError(err)
	}
	brief, err := req.RequireString("brief")
	if err != nil {
		return toolError(err)
	}
	dir, err := teamsDirectory(req)
	if err != nil {
		return toolError(err)
	}
	priority := int(req.GetFloat("priority", 0))
	id, err := svc.SpawnTask(ctx, service.SpawnTaskParams{
		Directory: dir,
		TeamID:    req.GetString("team_id", ""),
		Title:     title,
		Brief:     brief,
		Priority:  priority,
		CreatedBy: req.GetString("created_by", ""),
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"task_id": id, "status": "pending"})
}

func handleAgentPullNextTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	memberID, err := req.RequireString("member_id")
	if err != nil {
		return toolError(err)
	}
	dir, err := teamsDirectory(req)
	if err != nil {
		return toolError(err)
	}
	view, err := svc.PullNextTask(ctx, dir, memberID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"task_id":    view.ID,
		"title":      view.Title,
		"status":     view.Status,
		"brief_path": view.BriefPath,
		"priority":   view.Priority,
	})
}

func handleAgentReadTask(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	taskID, err := req.RequireString("task_id")
	if err != nil {
		return toolError(err)
	}
	dir, err := teamsDirectory(req)
	if err != nil {
		return toolError(err)
	}
	view, brief, err := svc.ReadTask(ctx, dir, taskID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"task_id":     view.ID,
		"title":       view.Title,
		"status":      view.Status,
		"brief_path":  view.BriefPath,
		"output_path": view.OutputPath,
		"brief":       brief,
	})
}

func handleAgentSubmitOutput(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	taskID, err := req.RequireString("task_id")
	if err != nil {
		return toolError(err)
	}
	output, err := req.RequireString("output")
	if err != nil {
		return toolError(err)
	}
	dir, err := teamsDirectory(req)
	if err != nil {
		return toolError(err)
	}
	if err := svc.SubmitOutput(ctx, dir, taskID, req.GetString("summary", ""), output); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"task_id": taskID, "status": "review_spec"})
}

func handleAgentSubmitReview(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	taskID, err := req.RequireString("task_id")
	if err != nil {
		return toolError(err)
	}
	reviewerID, err := req.RequireString("reviewer_id")
	if err != nil {
		return toolError(err)
	}
	args := req.GetArguments()
	passedVal, ok := args["passed"]
	if !ok {
		return toolError(errMissingPassed)
	}
	passed, ok := passedVal.(bool)
	if !ok {
		return toolError(errPassedMustBeBool)
	}
	dir, err := teamsDirectory(req)
	if err != nil {
		return toolError(err)
	}
	if err := svc.SubmitReview(ctx, service.SubmitReviewParams{
		Directory:  dir,
		TaskID:     taskID,
		ReviewerID: reviewerID,
		ReviewType: req.GetString("review_type", "spec_compliance"),
		Passed:     passed,
		Comments:   req.GetString("comments", ""),
	}); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"task_id": taskID, "passed": passed})
}

func handleAgentMarkDone(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	taskID, err := req.RequireString("task_id")
	if err != nil {
		return toolError(err)
	}
	dir, err := teamsDirectory(req)
	if err != nil {
		return toolError(err)
	}
	if err := svc.MarkDone(ctx, dir, taskID); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"task_id": taskID, "status": "done"})
}

func teamsDirectory(req mcplib.CallToolRequest) (string, error) {
	dir := req.GetString("directory", "")
	if dir != "" {
		return dir, nil
	}
	return os.Getwd()
}

var (
	errMissingPassed    = errString("passed is required")
	errPassedMustBeBool = errString("passed must be a boolean")
)

type errString string

func (e errString) Error() string { return string(e) }
