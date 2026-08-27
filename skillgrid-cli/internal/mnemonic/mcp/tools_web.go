package mcp

import (
	"context"
	"encoding/json"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/webcache"
)

func registerWebTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{webCacheLookupTool(), handleWebCacheLookup},
		{webCacheSaveTool(), handleWebCacheSave},
		{webCacheSearchTool(), handleWebCacheSearch},
		{webCacheGetTool(), handleWebCacheGet},
		{webCacheStatusTool(), handleWebCacheStatus},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func webCacheLookupTool() mcplib.Tool {
	return mcplib.NewTool("web_cache_lookup",
		mcplib.WithDescription("Check cached web research before calling remote MCPs. Returns hit/miss/stale with entry id when found."),
		mcplib.WithString("source", mcplib.Required(), mcplib.Description("Cache source: context7, exa, deepwiki, fetch, or manual")),
		mcplib.WithString("url", mcplib.Description("URL for fetch lookups")),
		mcplib.WithString("query", mcplib.Description("Query text (context7, exa)")),
		mcplib.WithString("library_id", mcplib.Description("Context7 library id")),
		mcplib.WithString("version_tag", mcplib.Description("Context7 version tag")),
		mcplib.WithString("repo_name", mcplib.Description("DeepWiki repo name")),
		mcplib.WithString("question", mcplib.Description("DeepWiki question")),
		mcplib.WithString("sort_params", mcplib.Description("Exa sort params for cache key")),
		mcplib.WithString("title", mcplib.Description("Manual entry title")),
		mcplib.WithString("content_hash", mcplib.Description("Manual entry content hash")),
	)
}

func webCacheSaveTool() mcplib.Tool {
	return mcplib.NewTool("web_cache_save",
		mcplib.WithDescription("Persist a web research snapshot after remote MCP or fetch. Call immediately after Context7/Exa/DeepWiki/WebFetch returns."),
		mcplib.WithString("source", mcplib.Required(), mcplib.Description("Cache source: context7, exa, deepwiki, fetch, or manual")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Snapshot body (max 256KB)")),
		mcplib.WithString("url", mcplib.Description("Source URL")),
		mcplib.WithString("title", mcplib.Description("Human-readable title")),
		mcplib.WithString("query", mcplib.Description("Original query")),
		mcplib.WithString("library_id", mcplib.Description("Context7 library id")),
		mcplib.WithString("version_tag", mcplib.Description("Context7 version tag")),
		mcplib.WithString("repo_name", mcplib.Description("DeepWiki repo name")),
		mcplib.WithString("question", mcplib.Description("DeepWiki question")),
		mcplib.WithString("sort_params", mcplib.Description("Exa sort params")),
		mcplib.WithString("session_id", mcplib.Description("Optional session id")),
		mcplib.WithString("metadata", mcplib.Description("Optional JSON metadata string")),
	)
}

func webCacheSearchTool() mcplib.Tool {
	return mcplib.NewTool("web_cache_search",
		mcplib.WithDescription("FTS5 search over cached web research. Use when user asks what was found online previously."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search keywords")),
		mcplib.WithString("source", mcplib.Description("Filter by source")),
		mcplib.WithBoolean("fresh_only", mcplib.Description("Only non-expired entries (default true)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum results (default 20)")),
	)
}

func webCacheGetTool() mcplib.Tool {
	return mcplib.NewTool("web_cache_get",
		mcplib.WithDescription("Fetch full untruncated cached snapshot by id from web_cache_lookup or web_cache_search."),
		mcplib.WithNumber("id", mcplib.Required(), mcplib.Description("Web cache entry id")),
	)
}

func webCacheStatusTool() mcplib.Tool {
	return mcplib.NewTool("web_cache_status",
		mcplib.WithDescription("Web cache health: counts by source, expired entries, oldest/newest fetch."),
	)
}

func handleWebCacheLookup(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	source, err := req.RequireString("source")
	if err != nil {
		return toolError(err)
	}

	result, err := svc.WebLookup(ctx, projectID, webcache.LookupInput{
		Source:      source,
		URL:         req.GetString("url", ""),
		Query:       req.GetString("query", ""),
		LibraryID:   req.GetString("library_id", ""),
		VersionTag:  req.GetString("version_tag", ""),
		RepoName:    req.GetString("repo_name", ""),
		Question:    req.GetString("question", ""),
		SortParams:  req.GetString("sort_params", ""),
		Title:       req.GetString("title", ""),
		ContentHash: req.GetString("content_hash", ""),
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(lookupDTO(result))
}

func handleWebCacheSave(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	source, err := req.RequireString("source")
	if err != nil {
		return toolError(err)
	}
	content, err := req.RequireString("content")
	if err != nil {
		return toolError(err)
	}

	var metadata map[string]any
	if metaStr := req.GetString("metadata", ""); metaStr != "" {
		if err := json.Unmarshal([]byte(metaStr), &metadata); err != nil {
			return toolError(err)
		}
	}

	id, err := svc.WebSave(ctx, projectID, webcache.SaveWebInput{
		Source:     source,
		Content:    content,
		URL:        req.GetString("url", ""),
		Title:      req.GetString("title", ""),
		Query:      req.GetString("query", ""),
		LibraryID:  req.GetString("library_id", ""),
		VersionTag: req.GetString("version_tag", ""),
		RepoName:   req.GetString("repo_name", ""),
		Question:   req.GetString("question", ""),
		SortParams: req.GetString("sort_params", ""),
		SessionID:  req.GetString("session_id", ""),
		Metadata:   metadata,
	})
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"id": id})
}

func handleWebCacheSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}

	freshOnly := true
	if args := req.GetArguments(); args != nil {
		if v, ok := args["fresh_only"]; ok {
			if b, ok := v.(bool); ok {
				freshOnly = b
			}
		}
	}

	limit := int(req.GetFloat("limit", 20))
	hits, err := svc.WebSearch(ctx, projectID, query, req.GetString("source", ""), freshOnly, limit)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"entries": webHitDTOs(hits)})
}

func handleWebCacheGet(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	id, err := req.RequireFloat("id")
	if err != nil {
		return toolError(err)
	}

	entry, err := svc.WebGet(ctx, projectID, int64(id))
	if err != nil {
		return toolError(err)
	}
	return JSONResult(webEntryDTO(entry))
}

func handleWebCacheStatus(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = req
	svc, projectID, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	st, err := svc.WebCacheStatus(ctx, projectID)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(statusDTO(st))
}

func lookupDTO(r webcache.LookupResult) map[string]any {
	m := map[string]any{
		"status": r.Status,
		"fresh":  r.Fresh,
	}
	if r.ID > 0 {
		m["id"] = r.ID
	}
	if r.FetchedAt != "" {
		m["fetched_at"] = r.FetchedAt
	}
	if r.ExpiresAt != "" {
		m["expires_at"] = r.ExpiresAt
	}
	return m
}

func webHitDTOs(hits []webcache.WebHit) []map[string]any {
	out := make([]map[string]any, len(hits))
	for i, hit := range hits {
		out[i] = map[string]any{
			"id":         hit.ID,
			"source":     hit.Source,
			"title":      hit.Title,
			"query":      hit.Query,
			"url":        hit.URL,
			"library_id": hit.LibraryID,
			"fetched_at": hit.FetchedAt,
			"expires_at": hit.ExpiresAt,
		}
	}
	return out
}

func webEntryDTO(e webcache.WebEntry) map[string]any {
	m := map[string]any{
		"id":           e.ID,
		"project":      e.Project,
		"source":       e.Source,
		"cache_key":    e.CacheKey,
		"content":      e.Content,
		"content_hash": e.ContentHash,
		"fetched_at":   e.FetchedAt,
		"created_at":   e.CreatedAt,
	}
	if e.URL != "" {
		m["url"] = e.URL
	}
	if e.Title != "" {
		m["title"] = e.Title
	}
	if e.Query != "" {
		m["query"] = e.Query
	}
	if e.LibraryID != "" {
		m["library_id"] = e.LibraryID
	}
	if e.VersionTag != "" {
		m["version_tag"] = e.VersionTag
	}
	if e.ExpiresAt != "" {
		m["expires_at"] = e.ExpiresAt
	}
	if e.SessionID != "" {
		m["session_id"] = e.SessionID
	}
	if e.MetadataJSON != "" {
		var meta any
		if err := json.Unmarshal([]byte(e.MetadataJSON), &meta); err == nil {
			m["metadata"] = meta
		}
	}
	return m
}

func statusDTO(st webcache.Status) map[string]any {
	return map[string]any{
		"total_entries":   st.TotalEntries,
		"expired_entries": st.ExpiredEntries,
		"by_source":       st.BySource,
		"oldest_fetch":    st.OldestFetch,
		"newest_fetch":    st.NewestFetch,
	}
}
