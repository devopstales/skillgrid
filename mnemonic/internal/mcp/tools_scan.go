package mcp

import (
	"context"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

func registerScanTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{scanStartTool(), handleScanStart},
		{scanStoreFindingsTool(), handleScanStoreFindings},
		{scanListTool(), handleScanList},
		{scanGetTool(), handleScanGet},
		{scanStatusTool(), handleScanStatus},
		{scanDiffTool(), handleScanDiff},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func scanStartTool() mcplib.Tool {
	return mcplib.NewTool("scan_start",
		mcplib.WithDescription("Run a security scanner (trivy, wapiti, nuclei, semgrep) against a target, persist its raw output, and record the scans row. Fail-open (ADR-0016): a failing scanner returns a row with status=error and no error."),
		mcplib.WithString("tool", mcplib.Required(), mcplib.Description("Scanner tool: trivy, wapiti, nuclei, or semgrep")),
		mcplib.WithString("target", mcplib.Required(), mcplib.Description("Scan target (filesystem path, URL, or repo)")),
	)
}

func scanStoreFindingsTool() mcplib.Tool {
	return mcplib.NewTool("scan_store_findings",
		mcplib.WithDescription("Parse a stored scan's raw output and upsert its normalized findings. Returns the finding count."),
		mcplib.WithString("scan_id", mcplib.Required(), mcplib.Description("Scan id from scan_start")),
	)
}

func scanListTool() mcplib.Tool {
	return mcplib.NewTool("scan_list",
		mcplib.WithDescription("List scans ordered by started_at DESC, optionally filtered by tool and status."),
		mcplib.WithString("tool", mcplib.Description("Filter by scanner tool")),
		mcplib.WithString("status", mcplib.Description("Filter by status: ok, error, or partial")),
		mcplib.WithNumber("limit", mcplib.Description("Cap the result (0 = no cap)")),
	)
}

func scanGetTool() mcplib.Tool {
	return mcplib.NewTool("scan_get",
		mcplib.WithDescription("Get one scan row by id."),
		mcplib.WithString("id", mcplib.Required(), mcplib.Description("Scan id")),
	)
}

func scanStatusTool() mcplib.Tool {
	return mcplib.NewTool("scan_status",
		mcplib.WithDescription("Per-tool finding counts by severity and the latest scan time."),
	)
}

func scanDiffTool() mcplib.Tool {
	return mcplib.NewTool("scan_diff",
		mcplib.WithDescription("Set difference of findings between two scans by dedup_hash. Read-only: no rows are inserted, updated, or deleted. When both old and new are absent, diffs the two most recent scans."),
		mcplib.WithString("old", mcplib.Description("Old scan id (omit with new for LatestDiff)")),
		mcplib.WithString("new", mcplib.Description("New scan id (omit with old for LatestDiff)")),
	)
}

// scanDataDir locates the scan raw-output cache dir.
func scanDataDir(h interface{ Root() string }) string {
	if d, err := service.DefaultDataDir(); err == nil {
		return d
	}
	return h.Root()
}

func handleScanStart(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	tool := stringArg(req, "tool")
	target := stringArg(req, "target")
	if tool == "" || target == "" {
		return toolError(fmt.Errorf("scan_start requires tool and target"))
	}
	scanSvc := scan.New(h.Store(), scanDataDir(h))
	res, err := scanSvc.Start(ctx, tool, target)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(res)
}

func handleScanStoreFindings(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	scanID := stringArg(req, "scan_id")
	if scanID == "" {
		return toolError(fmt.Errorf("scan_store_findings requires scan_id"))
	}
	scanSvc := scan.New(h.Store(), scanDataDir(h))
	count, err := scanSvc.StoreFindings(ctx, scanID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"findings": count})
}

func handleScanList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	scanSvc := scan.New(h.Store(), scanDataDir(h))
	scans, err := scanSvc.List(ctx, scan.ListFilter{
		Tool:   stringArg(req, "tool"),
		Status: stringArg(req, "status"),
		Limit:  int(req.GetFloat("limit", 0)),
	})
	if err != nil {
		return toolError(err)
	}
	if scans == nil {
		scans = []scan.Scan{}
	}
	return JSONResult(map[string]any{"scans": scans})
}

func handleScanGet(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id := stringArg(req, "id")
	if id == "" {
		return toolError(fmt.Errorf("scan_get requires id"))
	}
	scanSvc := scan.New(h.Store(), scanDataDir(h))
	res, err := scanSvc.Get(ctx, id)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(res)
}

func handleScanStatus(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = req
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	scanSvc := scan.New(h.Store(), scanDataDir(h))
	status, err := scanSvc.Status(ctx)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(status)
}

func handleScanDiff(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	scanSvc := scan.New(h.Store(), scanDataDir(h))
	oldID := stringArg(req, "old")
	newID := stringArg(req, "new")
	if oldID == "" && newID == "" {
		diff, oldScan, newScan, err := scanSvc.LatestDiff(ctx)
		if err != nil {
			return toolError(err)
		}
		return JSONResult(map[string]any{"old": oldScan, "new": newScan, "diff": diff})
	}
	if oldID == "" || newID == "" {
		return toolError(fmt.Errorf("scan_diff requires both old and new, or neither for LatestDiff"))
	}
	diff, err := scanSvc.Diff(ctx, oldID, newID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"old": oldID, "new": newID, "diff": diff})
}
