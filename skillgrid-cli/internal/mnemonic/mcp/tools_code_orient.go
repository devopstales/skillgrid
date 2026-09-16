package mcp

import (
	"context"
	"database/sql"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/process"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// registerOrientTools registers the Tier-1 orientation MCP tools. They resolve
// a symbol and return signature / file TOC / map / list / metadata plus linked
// rationale. Unknown symbols return not-found (no fabricated symbol).
func registerOrientTools(s *server.MCPServer) {
	tools := []struct {
		tool    mcplib.Tool
		handler server.ToolHandlerFunc
	}{
		{codeOrientTool(), handleCodeOrient},
		{codeSignatureTool(), handleCodeSignature},
		{codeFileTOCTool(), handleCodeFileTOC},
		{codeRationaleTool(), handleCodeRationale},
	}
	for _, entry := range tools {
		s.AddTool(entry.tool, entry.handler)
	}
}

func codeOrientTool() mcplib.Tool {
	return mcplib.NewTool("code_orient",
		mcplib.WithDescription("Tier-1 orientation for a resolved symbol: returns the symbol metadata, its signature, the file TOC (all symbols in the file), a list of symbols, and linked rationale. Use after code_search/code_orient to understand where a symbol lives and why it exists. Unknown symbols return not-found, not a fabricated symbol."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("Symbol name (exact or identifier; camelCase/snake_case both resolve)")),
	)
}

func codeSignatureTool() mcplib.Tool {
	return mcplib.NewTool("code_signature",
		mcplib.WithDescription("Return just the signature and span of a resolved symbol. Unknown symbols return not-found."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("Symbol name")),
	)
}

func codeFileTOCTool() mcplib.Tool {
	return mcplib.NewTool("code_file_toc",
		mcplib.WithDescription("Return the table of contents (every symbol, in line order) for the file that contains a resolved symbol. Unknown symbols return not-found."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("Symbol name (resolves to its file)")),
	)
}

func codeRationaleTool() mcplib.Tool {
	return mcplib.NewTool("code_rationale",
		mcplib.WithDescription("Return the rationale comments (# NOTE: / # WHY: / ADR-RFC citations) linked to a resolved symbol's nearest enclosing definition. Unknown symbols return not-found."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("Symbol name")),
	)
}

func handleCodeOrient(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	symbol, err := req.RequireString("symbol")
	if err != nil {
		return toolError(err)
	}
	out, err := mcpOrientSymbol(h.Store().DB, symbol)
	if err != nil {
		return toolError(err)
	}
	// Additive (008 step 02): surface which processes the symbol participates
	// in (step N/M). The 005 fields above are unchanged.
	if out.Found && out.Symbol != nil {
		if id, ok := out.Symbol["id"].(int64); ok {
			if parts, perr := process.Participations(h.Store().DB, id); perr == nil && len(parts) > 0 {
				out.Processes = parts
			}
		}
	}
	return JSONResult(out)
}

func handleCodeSignature(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	symbol, err := req.RequireString("symbol")
	if err != nil {
		return toolError(err)
	}
	out, err := mcpOrientSymbol(h.Store().DB, symbol)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"found":     out.Found,
		"symbol":    out.Symbol,
		"signature": out.Signature,
		"reason":    out.Reason,
	})
}

func handleCodeFileTOC(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	symbol, err := req.RequireString("symbol")
	if err != nil {
		return toolError(err)
	}
	out, err := mcpOrientSymbol(h.Store().DB, symbol)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"found":    out.Found,
		"file_toc": out.FileTOC,
		"reason":   out.Reason,
	})
}

func handleCodeRationale(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	symbol, err := req.RequireString("symbol")
	if err != nil {
		return toolError(err)
	}
	out, err := mcpOrientSymbol(h.Store().DB, symbol)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"found":     out.Found,
		"rationale": out.Rationale,
		"reason":    out.Reason,
	})
}

// mcpOrientSymbol is the single-open backing for the Tier-1 orientation tools:
// resolve symbol (exact, then identifier-FTS) and return its signature, file
// TOC, list, metadata, and linked rationale, straight from the already-open
// handle store. Byte-compatible with service.orientSymbol.
func mcpOrientSymbol(db *sql.DB, symbol string) (*service.OrientResult, error) {
	row := db.QueryRow(`
		SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
		       s.start_line, s.end_line, f.path, s.file_id
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.name = ?
		ORDER BY s.id LIMIT 1`, symbol)
	var id, fileID int64
	var name, qualified, kind, lang, sig string
	var startLine, endLine int
	var path string
	err := row.Scan(&id, &name, &qualified, &kind, &lang, &sig, &startLine, &endLine, &path, &fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback: identifier-FTS.
			hits, e2 := search.SymbolFTS(db, symbol, 1)
			if e2 != nil {
				return nil, e2
			}
			if len(hits) == 0 {
				return &service.OrientResult{Found: false, Reason: "symbol not found: " + symbol}, nil
			}
			hit := hits[0]
			row2 := db.QueryRow(`
				SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
				       s.start_line, s.end_line, f.path, s.file_id
				FROM symbols s INNER JOIN files f ON f.id = s.file_id
				WHERE s.id = ?`, hit.ID)
			if err := row2.Scan(&id, &name, &qualified, &kind, &lang, &sig, &startLine, &endLine, &path, &fileID); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	out := &service.OrientResult{
		Found: true,
		Symbol: map[string]any{
			"id":             id,
			"name":           name,
			"qualified_name": qualified,
			"kind":           kind,
			"language":       lang,
			"path":           path,
			"start_line":     startLine,
			"end_line":       endLine,
		},
		Signature: sig,
	}

	// File TOC + list: every symbol in the file, ordered by start line.
	tocRows, err := db.Query(`
		SELECT id, name, qualified_name, kind, start_line, end_line
		FROM symbols WHERE file_id = ? ORDER BY start_line, id`, fileID)
	if err != nil {
		return out, nil
	}
	for tocRows.Next() {
		var tID int64
		var tName, tQualified, tKind string
		var tStart, tEnd int
		if err := tocRows.Scan(&tID, &tName, &tQualified, &tKind, &tStart, &tEnd); err != nil {
			tocRows.Close()
			return out, nil
		}
		entry := map[string]any{
			"id":         tID,
			"name":       tName,
			"kind":       tKind,
			"start_line": tStart,
			"end_line":   tEnd,
		}
		out.FileTOC = append(out.FileTOC, entry)
	}
	tocRows.Close()
	out.List = out.FileTOC

	// Rationale linked to this symbol.
	rationaleRows, err := db.Query(`SELECT text, kind, line FROM rationale WHERE symbol_id = ? ORDER BY line`, id)
	if err == nil {
		for rationaleRows.Next() {
			var text, kind string
			var line int
			if rationaleRows.Scan(&text, &kind, &line) == nil {
				out.Rationale = append(out.Rationale, map[string]any{
					"text": text,
					"kind": kind,
					"line": line,
				})
			}
		}
		rationaleRows.Close()
	}
	return out, nil
}
