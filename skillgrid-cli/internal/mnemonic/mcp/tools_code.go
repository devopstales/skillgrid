package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"skillgrid-cli/internal/mnemonic/codeindex"
	"skillgrid-cli/internal/mnemonic/config"
	"skillgrid-cli/internal/mnemonic/project"
	"skillgrid-cli/internal/mnemonic/search"
	"skillgrid-cli/internal/mnemonic/store"
)

func registerCodeTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeStatusTool(), handleCodeStatus},
		{codeIndexTool(), handleCodeIndex},
		{codeSearchTool(), handleCodeSearch},
		{codeReadTool(), handleCodeRead},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeStatusTool() mcplib.Tool {
	return mcplib.NewTool("code_status",
		mcplib.WithDescription("Check code index health before searching. Call when the index may be stale (after clone, branch switch, or large refactors). If stale=true, run code_index before code_search. v1 ladder: code_status → code_search → code_read."),
	)
}

func codeIndexTool() mcplib.Tool {
	return mcplib.NewTool("code_index",
		mcplib.WithDescription("Run incremental code index for the cwd git root (respects indexing.yaml). Call after clone or when code_status reports stale. Do not grep the whole repo until indexed."),
	)
}

func codeSearchTool() mcplib.Tool {
	return mcplib.NewTool("code_search",
		mcplib.WithDescription("BM25 full-text search over indexed code chunks. Prefer this over grep/rg when exploring unknown areas of a large repo. Use code_read only after search narrows path and line range. Check code_status first if results seem outdated."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("Search terms (FTS5)")),
		mcplib.WithNumber("limit", mcplib.Description("Maximum hits (default 20)")),
	)
}

func codeReadTool() mcplib.Tool {
	return mcplib.NewTool("code_read",
		mcplib.WithDescription("Fetch indexed source for a path (and optional line range) after code_search narrows the location. Do not read whole files speculatively — search first, then read the matching slice."),
		mcplib.WithString("path", mcplib.Required(), mcplib.Description("Repo-relative file path from code_search")),
		mcplib.WithNumber("start_line", mcplib.Description("Start line (1-based); omit to read all indexed chunks for path")),
		mcplib.WithNumber("end_line", mcplib.Description("End line (1-based); defaults to start_line")),
	)
}

func handleCodeStatus(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = ctx
	_ = req

	st, cleanup, err := openProjectStore()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	status, err := codeindex.GetStatus(st)
	if err != nil {
		return toolError(err)
	}

	return JSONResult(map[string]any{
		"file_count":    status.FileCount,
		"chunk_count":   status.ChunkCount,
		"last_indexed":  status.LastIndexed,
		"stale":         codeIndexStale(status),
	})
}

func handleCodeIndex(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = req

	st, cleanup, err := openProjectStore()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	cwd, err := os.Getwd()
	if err != nil {
		return toolError(err)
	}

	root := cwd
	if gitRoot, err := gitRoot(cwd); err == nil && gitRoot != "" {
		root = gitRoot
	}

	cfg := config.Load(root)
	idxCfg := codeindex.Config{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		ChunkLines:   cfg.ChunkLines,
		ChunkOverlap: cfg.ChunkOverlap,
	}

	stats, err := codeindex.New(st).Run(ctx, root, idxCfg)
	if err != nil {
		return toolError(err)
	}

	return JSONResult(map[string]any{
		"files_indexed": stats.FilesIndexed,
		"files_skipped": stats.FilesSkipped,
		"files_deleted": stats.FilesDeleted,
		"chunks_added":  stats.ChunksAdded,
	})
}

func handleCodeSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = ctx

	st, cleanup, err := openProjectStore()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}

	limit := int(req.GetFloat("limit", 20))
	hits, err := search.CodeSearch(st.DB, query, limit)
	if err != nil {
		return toolError(err)
	}

	return JSONResult(map[string]any{"hits": codeHitDTOs(hits)})
}

func handleCodeRead(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = ctx

	st, cleanup, err := openProjectStore()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	path, err := req.RequireString("path")
	if err != nil {
		return toolError(err)
	}

	startLine := int(req.GetFloat("start_line", 0))
	endLine := int(req.GetFloat("end_line", 0))

	result, err := readIndexedCode(st.DB, path, startLine, endLine)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(result)
}

func openProjectStore() (*store.Store, func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	projectID, err := project.Resolve(cwd)
	if err != nil {
		return nil, nil, err
	}

	dataDir, err := mnemonicDataDir()
	if err != nil {
		return nil, nil, err
	}

	st, err := store.Open(dataDir, projectID)
	if err != nil {
		return nil, nil, err
	}

	return st, func() { st.Close() }, nil
}

func codeIndexStale(status codeindex.Status) bool {
	return status.FileCount == 0 || status.LastIndexed == ""
}

func codeHitDTOs(hits []search.CodeHit) []map[string]any {
	out := make([]map[string]any, len(hits))
	for i, hit := range hits {
		out[i] = map[string]any{
			"path":       hit.Path,
			"start_line": hit.StartLine,
			"end_line":   hit.EndLine,
			"snippet":    hit.Snippet,
			"score":      hit.Score,
		}
	}
	return out
}

func readIndexedCode(db *sql.DB, path string, startLine, endLine int) (map[string]any, error) {
	var fileID int64
	err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not indexed: %s", path)
	}
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if startLine > 0 {
		if endLine <= 0 {
			endLine = startLine
		}
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ? AND start_line <= ? AND end_line >= ?
			ORDER BY start_line`,
			fileID, endLine, startLine,
		)
	} else {
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ?
			ORDER BY start_line`,
			fileID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []string
	firstLine := 0
	lastLine := 0
	for rows.Next() {
		var chunkStart, chunkEnd int
		var text string
		if err := rows.Scan(&chunkStart, &chunkEnd, &text); err != nil {
			return nil, err
		}
		if firstLine == 0 || chunkStart < firstLine {
			firstLine = chunkStart
		}
		if chunkEnd > lastLine {
			lastLine = chunkEnd
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no indexed chunks for %s", path)
	}

	return map[string]any{
		"path":       path,
		"start_line": firstLine,
		"end_line":   lastLine,
		"text":       strings.Join(parts, "\n"),
	}, nil
}

func gitRoot(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
