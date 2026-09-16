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

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
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

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	status, err := codeindex.GetStatus(h.Store())
	if err != nil {
		return toolError(err)
	}
	stale := status.FileCount == 0 || status.LastIndexed == ""
	// Additive: per-language fair coverage (measured from edges). Existing
	// fields stay unchanged.
	coverage := map[string]graph.CoverageLang{}
	if cov, covErr := graph.FairCoverage(ctx, h.Store().DB); covErr == nil {
		coverage = cov
	}
	// Unresolved refs: dropped route-handler references the index could not
	// resolve (034). 0 on a store predating 034 is not expected (migrations
	// run on open); a missing table is surfaced as an error, not a guess.
	var unresolved int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM unresolved_refs`).Scan(&unresolved); err != nil {
		return toolError(err)
	}
	// Unresolved receiver-qualified call sites (036): the call-layer analogue
	// of unresolved_refs. 0 on a fresh index; a missing table is an error, not
	// a guess.
	var unresolvedMembers int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM unresolved_members`).Scan(&unresolvedMembers); err != nil {
		return toolError(err)
	}
	// Call-resolution audit ledger (036): per-language call sites vs unresolved
	// receiver members — the quantified drop-not-guess health signal.
	audit := map[string]map[string]int{}
	if rows, err := h.Store().DB.Query(`SELECT language, call_sites, unresolved FROM resolution_audit ORDER BY language`); err == nil {
		for rows.Next() {
			var lang string
			var callSites, unres int
			if rows.Scan(&lang, &callSites, &unres) == nil {
				audit[lang] = map[string]int{"call_sites": callSites, "unresolved": unres}
			}
		}
		rows.Close()
	}
	// Schema fingerprint (036): the structural-schema hash the current binary
	// writes. If it differs from the one stored at index time, the index was
	// built by an older extraction schema (silently stale). Empty on a store
	// predating 036.
	var schemaFP string
	if err := h.Store().DB.QueryRow(`SELECT value FROM index_meta_kv WHERE key = 'schema_fingerprint'`).Scan(&schemaFP); err != nil {
		schemaFP = ""
	}
	// Import cycles (038): dependency loops in the file import subgraph
	// (graphify "Import Cycles" signal). 0 on a cycle-free index; a missing
	// table (store predating 038) is an error, not a guess.
	var importCycles int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM import_cycles`).Scan(&importCycles); err != nil {
		return toolError(err)
	}
	// Capabilities (036): declare what the index can actually do so an agent
	// knows BEFORE querying whether full-text, vector, or graph traversal are
	// available (the gitnexus capabilities block).
	capabilities := map[string]map[string]string{
		"graph":  {"provider": "sqlite", "status": "available"},
		"fts":    {"provider": "sqlite-fts5", "status": "available"},
		"vector": {"provider": "embedder", "status": "unavailable"},
	}
	var ftsRows int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM symbol_fts`).Scan(&ftsRows); err == nil && ftsRows == 0 {
		capabilities["fts"]["status"] = "unavailable"
	}
	var embCount int
	var embDim int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(dim),0) FROM embeddings`).Scan(&embCount, &embDim); err == nil && embCount > 0 {
		capabilities["vector"]["status"] = "available"
		capabilities["vector"]["dims"] = fmt.Sprintf("%d", embDim)
	}

	return JSONResult(map[string]any{
		"file_count":         status.FileCount,
		"chunk_count":        status.ChunkCount,
		"last_indexed":       status.LastIndexed,
		"stale":              stale,
		"fair_coverage":      coverage,
		"unresolved_refs":    unresolved,
		"unresolved_members": unresolvedMembers,
		"resolution_audit":   audit,
		"import_cycles":      importCycles,
		"schema_fingerprint": schemaFP,
		"capabilities":       capabilities,
	})
}

func handleCodeIndex(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_ = req

	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return toolError(err)
	}

	root := cwd
	if gitRoot, err := gitRoot(cwd); err == nil && gitRoot != "" {
		root = gitRoot
	}

	stats, err := svc.RunCodeIndex(ctx, root)
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
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()

	query, err := req.RequireString("query")
	if err != nil {
		return toolError(err)
	}

	limit := int(req.GetFloat("limit", 20))
	hits, err := search.CodeSearch(h.Store().DB, query, limit)
	if err != nil {
		return toolError(err)
	}

	res, err := JSONResult(map[string]any{"hits": codeHitDTOs(h.Store().DB, query, hits)})
	if err != nil {
		return nil, err
	}
	return applyFreshness(res, hitPaths(hits)), nil
}

func handleCodeRead(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
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

	result, err := mcpReadIndexedCode(h.Store().DB, path, startLine, endLine)
	if err != nil {
		return toolError(err)
	}
	// Output-time secret redaction (01.3): a secret-like string in the indexed
	// file is replaced before it is emitted — index-time exclusion alone does
	// not count. The additive redacted field reports whether a replacement
	// occurred.
	rawText, _ := result["text"].(string)
	redacted := hybrid.RedactSecrets(rawText)
	if redacted != rawText {
		result["text"] = redacted
		result["redacted"] = true
	} else {
		result["redacted"] = false
	}
	res, err := JSONResult(result)
	if err != nil {
		return nil, err
	}
	return applyFreshness(res, []string{path}), nil
}

// mcpReadIndexedCode is the single-open backing for code_read: fetch indexed
// source for path (and optional line range) directly on the already-open
// handle store, byte-compatible with service.readIndexedCode.
func mcpReadIndexedCode(db *sql.DB, path string, startLine, endLine int) (map[string]any, error) {
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

// codeHitDTOs maps code_search hits to their response DTO. The 005 fields
// (path/start_line/end_line/snippet/score) are unchanged; the additive fields
// (confidence, action, rerank_reasons, redacted) are GAINED per 005's
// additive-response contract (the tool's name + required `query` param are
// never changed).
func codeHitDTOs(db *sql.DB, query string, hits []search.CodeHit) []map[string]any {
	out := make([]map[string]any, len(hits))
	for i, hit := range hits {
		// Derive a RankHit for the explainable rerank table. The symbol is the
		// nearest indexed symbol in the hit range (best-effort; "" when absent).
		sym, kind := symbolAtRange(db, hit.Path, hit.StartLine, hit.EndLine)
		ctx := hybrid.RerankContext{
			Query:      query,
			IsTestFile: isTestPath(hit.Path),
		}
		factors := hybrid.ApplyRerankTable(hybrid.RankHit{
			Path: hit.Path, StartLine: hit.StartLine, EndLine: hit.EndLine,
			Symbol: sym, Kind: kind, Score: hit.Score,
		}, ctx)
		baseScore := hit.Score + hybrid.RerankDelta(factors)
		conf := hybrid.ConfidenceFromScore(baseScore, factors).
			WithActionAndFallbacks(query, []string{hit.Path})
		// Skeletonize the snippet (01.11) and redact it (01.3) so the snippet
		// stays short and a secret never ships raw. The snippet's own lines are
		// local 0..N-1 (not the file's absolute hit.StartLine..hit.EndLine), so
		// the skeletonization window must be 0-based over the snippet's lines —
		// passing the absolute file range would clamp to the whole snippet and
		// make the window meaningless.
		snippetLines := strings.Split(hit.Snippet, "\n")
		skel := hybrid.Skeletonize(snippetLines, query, 0, len(snippetLines)-1)
		skelText := strings.Join(skel, "\n")
		redacted := hybrid.RedactSecrets(skelText)
		out[i] = map[string]any{
			"path":       hit.Path,
			"start_line": hit.StartLine,
			"end_line":   hit.EndLine,
			"snippet":    redacted,
			"score":      hit.Score,
			// Additive response gains (005 contract: name + required params
			// unchanged, response schema only grows).
			"confidence":     conf.Level,
			"action":         conf.Action,
			"rerank_reasons": hybrid.RerankReasons(factors),
			"redacted":       redacted != skelText,
		}
		if len(conf.Suggested) > 0 {
			out[i]["fallbacks"] = conf.Suggested
		}
	}
	return out
}

// symbolAtRange best-effort resolves the symbol defined in [start,end] of a
// path (for the rerank table's exact-symbol / definition-kind factors).
func symbolAtRange(db *sql.DB, path string, start, end int) (string, string) {
	var fileID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID); err != nil {
		return "", ""
	}
	var name, kind string
	err := db.QueryRow(`
		SELECT name, kind FROM symbols
		WHERE file_id = ? AND start_line >= ? AND start_line <= ?
		ORDER BY end_line - start_line ASC
		LIMIT 1`, fileID, start, end).Scan(&name, &kind)
	if err != nil {
		return "", ""
	}
	return name, kind
}

func isTestPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "_test.go") || strings.HasSuffix(lower, ".test.js") || strings.Contains(lower, "test_")
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
