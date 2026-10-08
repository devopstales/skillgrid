package mcp

import (
	"context"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/dep"
)

func registerDepTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{depIngestTool(), handleDepIngest},
		{depListTool(), handleDepList},
		{depGetTool(), handleDepGet},
		{depAffectedTool(), handleDepAffected},
		{depGraphTool(), handleDepGraph},
		{depRuntimeTool(), handleDepRuntime},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func depIngestTool() mcplib.Tool {
	return mcplib.NewTool("dep_ingest",
		mcplib.WithDescription("Ingest a CycloneDX SBOM (JSON text): upsert dependencies by purl, rebuild dep_edges, and soft-retire (retired=1, never deleted) purls absent from this SBOM."),
		mcplib.WithString("sbom", mcplib.Required(), mcplib.Description("CycloneDX SBOM as JSON text")),
	)
}

func depListTool() mcplib.Tool {
	return mcplib.NewTool("dep_list",
		mcplib.WithDescription("List dependency packages filtered by retirement state. Omit retired to return all packages."),
		mcplib.WithBoolean("retired", mcplib.Description("true = only retired, false = only active, omitted = all")),
	)
}

func depGetTool() mcplib.Tool {
	return mcplib.NewTool("dep_get",
		mcplib.WithDescription("Get one dependency package by purl."),
		mcplib.WithString("purl", mcplib.Required(), mcplib.Description("Package purl")),
	)
}

func depAffectedTool() mcplib.Tool {
	return mcplib.NewTool("dep_affected",
		mcplib.WithDescription("Transitive set of purls that depend on the given purl (reverse BFS over dep_edges, depth-capped at 10). Excludes the purl itself."),
		mcplib.WithString("purl", mcplib.Required(), mcplib.Description("Package purl to reverse-resolve dependents of")),
	)
}

func depGraphTool() mcplib.Tool {
	return mcplib.NewTool("dep_graph",
		mcplib.WithDescription("Whole dependency graph: every dependency purl (node) and every stored edge."),
	)
}

func depRuntimeTool() mcplib.Tool {
	return mcplib.NewTool("dep_runtime",
		mcplib.WithDescription("For one indexed source file: which declared packages are never imported, which imports are never declared, and which appear in both."),
		mcplib.WithString("source_file", mcplib.Required(), mcplib.Description("Repo-relative path of an indexed source file")),
	)
}

func handleDepIngest(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sbom := stringArg(req, "sbom")
	if sbom == "" {
		return toolError(fmt.Errorf("dep_ingest requires sbom"))
	}
	depSvc := dep.New(h.Store().DB)
	if err := depSvc.Ingest(ctx, sbom); err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"status": "ok"})
}

func handleDepList(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	depSvc := dep.New(h.Store().DB)
	// Omitted retired returns all packages: active rows then retired rows.
	if _, ok := req.GetArguments()["retired"]; !ok {
		active, err := depSvc.List(ctx, false)
		if err != nil {
			return toolError(err)
		}
		retired, err := depSvc.List(ctx, true)
		if err != nil {
			return toolError(err)
		}
		all := make([]dep.Package, 0, len(active)+len(retired))
		all = append(all, active...)
		all = append(all, retired...)
		return JSONResult(map[string]any{"packages": all})
	}
	retired := req.GetBool("retired", false)
	pkgs, err := depSvc.List(ctx, retired)
	if err != nil {
		return toolError(err)
	}
	if pkgs == nil {
		pkgs = []dep.Package{}
	}
	return JSONResult(map[string]any{"packages": pkgs})
}

func handleDepGet(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	purl := stringArg(req, "purl")
	if purl == "" {
		return toolError(fmt.Errorf("dep_get requires purl"))
	}
	depSvc := dep.New(h.Store().DB)
	pkg, err := depSvc.Get(ctx, purl)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(pkg)
}

func handleDepAffected(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	purl := stringArg(req, "purl")
	if purl == "" {
		return toolError(fmt.Errorf("dep_affected requires purl"))
	}
	depSvc := dep.New(h.Store().DB)
	affected, err := depSvc.Affected(ctx, purl)
	if err != nil {
		return toolError(err)
	}
	if affected == nil {
		affected = []string{}
	}
	return JSONResult(map[string]any{"purl": purl, "affected": affected})
}

func handleDepGraph(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = req
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	depSvc := dep.New(h.Store().DB)
	graph, err := depSvc.Graph(ctx)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(graph)
}

func handleDepRuntime(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	sourceFile := stringArg(req, "source_file")
	if sourceFile == "" {
		return toolError(fmt.Errorf("dep_runtime requires source_file"))
	}
	depSvc := dep.New(h.Store().DB)
	res, err := depSvc.Runtime(ctx, sourceFile)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(res)
}
