package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/handoff"
)

// registerHandoffTools is the Handoff Hub MCP registrar (change 015-handoff-hub).
// It exposes the unified checkpoint surface over stdio: change-snapshot
// recording/status, named checkpoints with drift verification, and the
// aggregate rollup. Additive — it never touches the mem_*/code_*/session_*
// registration.
func registerHandoffTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{handoffSnapshotTool(), handleHandoffSnapshot},
		{handoffStatusTool(), handleHandoffStatus},
		{handoffCheckpointTool(), handleHandoffCheckpoint},
		{handoffVerifyTool(), handleHandoffVerify},
		{handoffRollupTool(), handleHandoffRollup},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

// handoffHub opens a CWD handle and builds the handoff Hub over its store.
func handoffHub() (*handoff.Hub, func(), error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return nil, func() {}, err
	}
	return &handoff.Hub{DB: h.Store().DB, Project: h.ProjectID(), RepoDir: h.Root()}, cleanup, nil
}

func handoffSnapshotTool() mcplib.Tool {
	return mcplib.NewTool("handoff_snapshot",
		mcplib.WithDescription("Record the current HEAD as a change snapshot in the Handoff Hub (idempotent). On first run for a project it backfills the last 100 commits from git log so the change log is not empty. Returns the recorded snapshot."),
		mcplib.WithNumber("limit", mcplib.Description("Backfill window when the hub is empty (default 100).")),
	)
}

func handleHandoffSnapshot(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	hub, cleanup, err := handoffHub()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	limit := int(req.GetInt("limit", 100))
	n, _ := hub.CountSnapshots(ctx)
	if n == 0 {
		_, _ = hub.Backfill(ctx, limit)
	}
	s, err := hub.Record(ctx)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(s)
}

func handoffStatusTool() mcplib.Tool {
	return mcplib.NewTool("handoff_status",
		mcplib.WithDescription("One-call 'where are we' from the Handoff Hub: the latest change snapshot, the open checkpoints, and recent handoff refs. Use this to ground a session before continuing work."),
		mcplib.WithNumber("limit", mcplib.Description("Max checkpoints / refs to include (default 20).")),
	)
}

func handleHandoffStatus(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	hub, cleanup, err := handoffHub()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	limit := int(req.GetInt("limit", 20))
	snapshots, _ := hub.ListSnapshots(ctx, 1)
	checkpoints, _ := hub.ListCheckpoints(ctx, limit)
	refs, _ := hub.ListHandoffRefs(ctx, limit)
	var latest any
	if len(snapshots) > 0 {
		latest = snapshots[0]
	}
	return JSONResult(map[string]any{
		"latest_snapshot": latest,
		"checkpoints":     checkpoints,
		"handoff_refs":    refs,
	})
}

func handoffCheckpointTool() mcplib.Tool {
	return mcplib.NewTool("handoff_checkpoint",
		mcplib.WithDescription("Place a named, intentional checkpoint marker before a risky action (e.g. before-apply-<change>). Captures branch/HEAD/dirty state and auto-detects the active spec dir. Idempotent by name."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Checkpoint name, e.g. 'before-apply-auth-foundation'")),
		mcplib.WithString("evidence", mcplib.Description("Optional short verification note (e.g. 'lint ok, tests 34/34').")),
		mcplib.WithString("spec_dir", mcplib.Description("Optional .skillgrid/sdd/<change>/ dir (auto-detected when blank).")),
	)
}

func handleHandoffCheckpoint(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	hub, cleanup, err := handoffHub()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	name, err := req.RequireString("name")
	if err != nil {
		return toolError(err)
	}
	cp, err := hub.RecordCheckpoint(ctx, handoff.CheckpointInput{
		Name: name,
		Evidence: req.GetString("evidence", ""),
		SpecDir:  req.GetString("spec_dir", ""),
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(cp)
}

func handoffVerifyTool() mcplib.Tool {
	return mcplib.NewTool("handoff_verify",
		mcplib.WithDescription("Drift-verify a named checkpoint against current git state (no reverts). Without a name, verifies all open checkpoints. Returns the recommendation: continue | inspect-drift | refresh."),
		mcplib.WithString("name", mcplib.Description("Checkpoint name. Blank verifies all open checkpoints.")),
	)
}

func handleHandoffVerify(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	hub, cleanup, err := handoffHub()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	name := req.GetString("name", "")
	if name == "" {
		open, _ := hub.ListCheckpoints(ctx, 200)
		out := []any{}
		for _, c := range open {
			if c.Status != "open" {
				continue
			}
			if rep, verr := hub.VerifyCheckpoint(ctx, c.Name); verr == nil {
				out = append(out, rep)
			}
		}
		return JSONResult(out)
	}
	rep, err := hub.VerifyCheckpoint(ctx, name)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(rep)
}

func handoffRollupTool() mcplib.Tool {
	return mcplib.NewTool("handoff_rollup",
		mcplib.WithDescription("Write a session/team rollup (git change stats + checkpoints + handoff refs) to .skillgrid/sdd/rollups/<ts>.md and return its path. Use at the end of a session or before a handoff to produce the aggregate report."),
	)
}

func handleHandoffRollup(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	hub, cleanup, err := handoffHub()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	_, h, cleanup2, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup2()
	p, err := handoff.WriteRollup(ctx, hub, h.Root())
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"path": p})
}
