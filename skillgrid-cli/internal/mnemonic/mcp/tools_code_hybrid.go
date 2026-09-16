package mcp

import (
	"context"
	"database/sql"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// openServiceForRepo opens the service + project for a request's optional
// repo param: an explicit repo wins, otherwise the cwd-resolved project.
// Returns the opened handle (open #1) plus a cleanup func to close it. The
// handler then performs its search through the handle so the project is not
// opened a second time inside a service facade method.
func openServiceForRepo(repo string) (*service.Service, *service.ProjectHandle, func(), error) {
	svc, err := rootService()
	if err != nil {
		return nil, nil, nil, err
	}
	// Resolve the requested project (explicit repo wins, else single-project
	// or cwd). The handle below opens the same project; projectIDForPool is
	// kept for its validation/error behavior when the resolution is invalid.
	if _, err := projectIDForPool(svc, repo); err != nil {
		return nil, nil, nil, err
	}
	h, cleanup, err := svc.OpenForDirectory(".")
	if err != nil {
		return nil, nil, nil, err
	}
	return svc, h, cleanup, nil
}

// registerHybridTools registers the offline hybrid search surface:
// code_hybrid_search (FTS + deterministic signals + optional embeddings,
// per-signal provenance), code_semantic_search (vector leg; symbol-level hits
// named), and code_embedding_status (provider/model/coverage). All are
// distinct code_* tools — none clashes with memory semantic_search.
func registerHybridTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeHybridSearchTool(), handleCodeHybridSearch},
		{codeSemanticSearchTool(), handleCodeSemanticSearch},
		{codeEmbeddingStatusTool(), handleCodeEmbeddingStatus},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeHybridSearchTool() mcplib.Tool {
	return mcplib.NewTool("code_hybrid_search",
		mcplib.WithDescription("Offline hybrid code search: fuses identifier/chunk FTS with deterministic signals (proximity, TF-IDF, type/API) and — when the embedder is available — semantic vectors via RRF. Every hit carries per-signal provenance. Embeddings are optional: a down/missing embedder degrades to FTS + signals, never a hard fail. Distinct from memory semantic_search."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search query (identifiers or free text)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 20)")),
		mcplib.WithString("repo", mcplib.Description("Optional project/repo name; omitted when the cwd resolves it")),
		mcplib.WithString("language", mcplib.Description("Optional language scope for the semantic leg (e.g. go, typescript); empty = all languages")),
	)
}

func codeSemanticSearchTool() mcplib.Tool {
	return mcplib.NewTool("code_semantic_search",
		mcplib.WithDescription("Semantic (embedding) code search. Symbol-level hits return the named symbol with file + line; chunk-level hits return the line range for non-symbol code. Requires an active embedder; with none configured it returns an empty result (the hybrid tool keeps the FTS+signals floor)."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Free-text semantic query")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 20)")),
		mcplib.WithString("repo", mcplib.Description("Optional project/repo name; omitted when the cwd resolves it")),
		mcplib.WithString("language", mcplib.Description("Optional language scope (e.g. go, typescript); exact index-level filter, empty = all languages")),
	)
}

func codeEmbeddingStatusTool() mcplib.Tool {
	return mcplib.NewTool("code_embedding_status",
		mcplib.WithDescription("Embedding index status: active provider (onnx/external/off), model name, vector dimension, embedded symbol count, and indexed model (model-swap guard). Read-only."),
		mcplib.WithString("repo", mcplib.Description("Optional project/repo name; omitted when the cwd resolves it")),
	)
}

func handleCodeHybridSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	if query == "" {
		return toolError(fmt.Errorf("code_hybrid_search: 'query' is required and must be non-empty"))
	}
	limit := int(req.GetFloat("limit", 20))

	_, h, cleanup, err := openServiceForRepo(req.GetString("repo", ""))
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := hybrid.Search(ctx, h.Store().DB, query, hybrid.Options{
		Limit:    limit,
		Language: req.GetString("language", ""),
		Embedder: mcpResolveEmbedder(h),
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeSemanticSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}
	if query == "" {
		return toolError(fmt.Errorf("code_semantic_search: 'query' is required and must be non-empty"))
	}
	limit := int(req.GetFloat("limit", 20))

	_, h, cleanup, err := openServiceForRepo(req.GetString("repo", ""))
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := hybrid.Search(ctx, h.Store().DB, query, hybrid.Options{
		Limit:    limit,
		Semantic: true,
		Language: req.GetString("language", ""),
		Embedder: mcpResolveEmbedder(h),
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

func handleCodeEmbeddingStatus(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openServiceForRepo(req.GetString("repo", ""))
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	out, err := mcpCodeEmbeddingStatus(ctx, h)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(out)
}

// mcpCodeEmbeddingStatus is the single-open backing for code_embedding_status:
// provider/model, vector dimension, embedded count, and the model-swap guard
// read straight from the already-open handle store. Byte-compatible with
// service.CodeEmbeddingStatus.
func mcpCodeEmbeddingStatus(ctx context.Context, h *service.ProjectHandle) (map[string]any, error) {
	cfg := config.Load(".")
	emb := mcpResolveEmbedder(h)
	status := map[string]any{
		"provider": cfg.Embedder.Provider,
	}
	if emb == nil {
		status["active"] = false
		status["model"] = ""
		status["dimension"] = 0
		status["reason"] = "embedder off"
	} else {
		status["active"] = true
		status["model"] = emb.Model()
		status["dimension"] = emb.Dimension()
	}
	db := h.Store().DB
	var symCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM embeddings`).Scan(&symCount)
	status["embedded_symbols"] = symCount
	var model sql.NullString
	_ = db.QueryRow(`SELECT value FROM embed_meta WHERE key = 'embedding_model'`).Scan(&model)
	status["indexed_model"] = model.String
	return status, nil
}

// mcpResolveEmbedder builds the process embedder from the workspace config,
// mirroring service.resolveEmbedder. The handle's ContentPlane root is not
// exposed (it is the cwd for OpenForCWD / OpenForDirectory), so config is read
// from "." — the same source the handle used to construct its web cache.
func mcpResolveEmbedder(h *service.ProjectHandle) embedder.Embedder {
	return mcpBuildEmbedder(config.Load("."))
}

func mcpBuildEmbedder(cfg config.Indexing) embedder.Embedder {
	switch cfg.Embedder.Provider {
	case "external":
		return embedder.NewExternal(embedder.ExternalConfig{
			BaseURL:   cfg.Embedder.BaseURL,
			Model:     cfg.Embedder.Model,
			APIKey:    cfg.Embedder.APIKey,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  mcpToAsym(cfg.Embedder.Indexing),
			Query:     mcpToAsym(cfg.Embedder.Query),
		})
	case "off", "":
		return nil
	default: // "onnx" is the default
		return embedder.NewOnnx(embedder.OnnxConfig{
			Model:     cfg.Embedder.Model,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  mcpToAsym(cfg.Embedder.Indexing),
			Query:     mcpToAsym(cfg.Embedder.Query),
		})
	}
}

func mcpToAsym(p config.EmbedderParams) embedder.AsymParams {
	return embedder.AsymParams{
		Instructions: p.Instructions,
		InputType:    p.InputType,
		MaxTokens:    p.MaxTokens,
	}
}
