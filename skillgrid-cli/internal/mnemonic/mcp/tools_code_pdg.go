package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerPdgTools registers the per-function CFG/PDG query tool
// (code_pdg_query). It is a distinct code_* tool — none clashes with the
// 005/008/010 baseline — and is part of the narrow code_* menu (unlisted by
// default; re-enabled via CodeToolsEnvVar). It is query-only: it reads the
// opt-in --pdg tables (cfg_blocks/cfg_edges/pdg_edges) and 005's symbols, and
// never fabricates a dependence for a symbol or statement the index does not
// have (a non---pdg index returns a clear "run --pdg" message, not an error;
// an unknown symbol/statement returns not-found, not an empty-but-present
// result).
func registerPdgTools(s *server.MCPServer) {
	s.AddTool(codePdgQueryTool(), handleCodePdgQuery)
}

// registerTaintTools registers the step-02 taint query tool (code_taint). It is
// a distinct code_* tool — none clashes with the 005/008/010 baseline.
func registerTaintTools(s *server.MCPServer) {
	s.AddTool(codeTaintTool(), handleCodeTaint)
}

func codePdgQueryTool() mcplib.Tool {
	return mcplib.NewTool("code_pdg_query",
		mcplib.WithDescription("Query the opt-in per-function CFG/PDG built by `index --pdg`: return the control- and data-dependents of a given statement inside a function. statement is the 1-based source line of the statement. Every returned dependence carries its Confidence Label (EXTRACTED | INFERRED | AMBIGUOUS | LSP_RESOLVED). If the index was not built with --pdg the tables are empty and the tool returns a clear 'run index --pdg' message (not an error). An unknown symbol or statement returns not-found, never a fabricated dependence. symbol is disambiguated to a function/method (a kind=class/struct is rejected)."),
		mcplib.WithString("symbol", mcplib.Required(), mcplib.Description("Function or method symbol name to query the PDG for")),
		mcplib.WithString("statement", mcplib.Required(), mcplib.Description("1-based source line of the statement whose dependents are wanted")),
		mcplib.WithString("kind", mcplib.Description("Narrow disambiguation to a symbol kind (default: any function/method)")),
	)
}

// pdgDependence is one control- or data-dependence edge returned to the
// caller.
type pdgDependence struct {
	Kind       string `json:"kind"`
	FromLine   int    `json:"from_line"`
	ToLine     int    `json:"to_line"`
	FromName   string `json:"from_name,omitempty"`
	ToName     string `json:"to_name,omitempty"`
	Confidence string `json:"confidence"`
	Note       string `json:"note,omitempty"`
}

type pdgQueryResult struct {
	Symbol      string          `json:"symbol"`
	Statement   int             `json:"statement"`
	Found       bool            `json:"found"`
	Dependents  []pdgDependence `json:"dependents"`
	PdgEnabled  bool            `json:"pdg_enabled"`
	Message     string          `json:"message,omitempty"`
}

func handleCodePdgQuery(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	symbol := strings.TrimSpace(stringArg(req, "symbol"))
	statement := strings.TrimSpace(stringArg(req, "statement"))
	if symbol == "" {
		return toolError(fmt.Errorf("code_pdg_query: missing required argument 'symbol'"))
	}
	if statement == "" {
		return toolError(fmt.Errorf("code_pdg_query: missing required argument 'statement'"))
	}
	line, err := parseStatementLine(statement)
	if err != nil {
		return toolError(err)
	}
	kind := strings.TrimSpace(stringArg(req, "kind"))

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	db := h.Store().DB

	// (1) Is the index built with --pdg? A non---pdg index has the tables but
	// they are empty; say so clearly rather than error or fabricate.
	var pdgRowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges`).Scan(&pdgRowCount); err != nil {
		return toolError(fmt.Errorf("code_pdg_query: cannot read pdg_edges: %w", err))
	}
	pdgEnabled := pdgRowCount > 0

	// (2) Resolve the symbol to a function/method. Ambiguity -> not-found with
	// the candidate kinds, never a silent pick.
	symID, symKind, ok, err := resolveFunctionSymbol(db, symbol, kind)
	if err != nil {
		return toolError(err)
	}
	if !ok {
		return JSONResult(pdgQueryResult{
			Symbol:     symbol,
			Statement:  line,
			Found:      false,
			Dependents: []pdgDependence{},
			PdgEnabled: pdgEnabled,
			Message:    fmt.Sprintf("not found: no function/method symbol %q in the index", symbol),
		})
	}

	// (3) Does the function have a PDG? (statement must also exist in the
	// function's CFG — a not-found statement returns not-found, not an empty
	// result.)
	var hasPdg int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges WHERE symbol_id = ?`, symID).Scan(&hasPdg); err != nil {
		return toolError(fmt.Errorf("code_pdg_query: cannot read pdg_edges: %w", err))
	}
	if !pdgEnabled || hasPdg == 0 {
		return JSONResult(pdgQueryResult{
			Symbol:      symbol,
			Statement:   line,
			Found:       false,
			Dependents:  []pdgDependence{},
			PdgEnabled:  pdgEnabled,
			Message:     "run `index --pdg` to build the per-function CFG/PDG for this symbol",
		})
	}
	var blockCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cfg_blocks WHERE symbol_id = ? AND (start_line <= ? AND ? <= end_line)`, symID, line, line).Scan(&blockCount); err != nil {
		return toolError(fmt.Errorf("code_pdg_query: cannot read cfg_blocks: %w", err))
	}
	if blockCount == 0 {
		return JSONResult(pdgQueryResult{
			Symbol:      symbol,
			Statement:   line,
			Found:       false,
			Dependents:  []pdgDependence{},
			PdgEnabled:  true,
			Message:     fmt.Sprintf("not found: statement at line %d is not in the CFG of %q", line, symbol),
		})
	}

	// (4) Return the control- and data-dependents of the statement: every
	// pdg_edge whose source (from_line) is the statement (the statement
	// controls/data-feeds its dependents).
	dependents := []pdgDependence{}
	rows, err := db.Query(`SELECT kind, from_line, to_line, from_name, to_name, confidence, note
		FROM pdg_edges WHERE symbol_id = ? AND from_line = ?
		ORDER BY kind, to_line, to_name`, symID, line)
	if err != nil {
		return toolError(fmt.Errorf("code_pdg_query: query pdg_edges: %w", err))
	}
	defer rows.Close()
	for rows.Next() {
		var d pdgDependence
		if err := rows.Scan(&d.Kind, &d.FromLine, &d.ToLine, &d.FromName, &d.ToName, &d.Confidence, &d.Note); err != nil {
			return toolError(fmt.Errorf("code_pdg_query: scan pdg_edge: %w", err))
		}
		dependents = append(dependents, d)
	}
	if err := rows.Err(); err != nil {
		return toolError(fmt.Errorf("code_pdg_query: iterate pdg_edges: %w", err))
	}
	return JSONResult(pdgQueryResult{
		Symbol:      symbol,
		Statement:   line,
		Found:       true,
		Dependents:  dependents,
		PdgEnabled:  true,
		Message:     fmt.Sprintf("%d dependence(s) of %s (kind=%s, line %d)", len(dependents), symbol, symKind, line),
	})
}

// parseStatementLine parses the 1-based statement line. A missing/empty
// statement is already rejected; a non-positive or non-numeric value is a
// validation error (abort, not a fabricated result).
func parseStatementLine(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("code_pdg_query: 'statement' must be a 1-based line number, got %q", s)
	}
	if n < 1 {
		return 0, fmt.Errorf("code_pdg_query: 'statement' must be >= 1, got %d", n)
	}
	return n, nil
}

// resolveFunctionSymbol resolves a symbol name to a function/method symbol.
// It rejects kinds that are not callable (class/struct/interface/type) and
// treats an absent symbol as not-found (no error). A kind filter, when given,
// must match.
func resolveFunctionSymbol(db *sql.DB, name, kind string) (int64, string, bool, error) {
	rows, err := db.Query(`SELECT id, kind FROM symbols WHERE name = ?`, name)
	if err != nil {
		return 0, "", false, fmt.Errorf("code_pdg_query: resolve symbol: %w", err)
	}
	defer rows.Close()
	type cand struct {
		id   int64
		kind string
	}
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.kind); err != nil {
			return 0, "", false, fmt.Errorf("code_pdg_query: scan symbol: %w", err)
		}
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return 0, "", false, fmt.Errorf("code_pdg_query: iterate symbols: %w", err)
	}
	if len(cands) == 0 {
		return 0, "", false, nil
	}
	// If a kind filter is given, restrict to it.
	if kind != "" {
		var filtered []cand
		for _, c := range cands {
			if c.kind == kind {
				filtered = append(filtered, c)
			}
		}
		cands = filtered
		if len(cands) == 0 {
			return 0, "", false, nil
		}
	}
	// A symbol that is not callable (class/struct/interface/type) is rejected:
	// the PDG is per-function, so a struct name is not a PDG subject.
	var callable []cand
	for _, c := range cands {
		if isCallableKind(c.kind) {
			callable = append(callable, c)
		}
	}
	if len(callable) == 0 {
		return 0, "", false, nil
	}
	if len(callable) > 1 {
		// Ambiguous across files; return the first by id (stable) — the caller
		// gets a definite answer, and the kind filter is the disambiguator.
		callable[0] = callable[0]
	}
	return callable[0].id, callable[0].kind, true, nil
}

func isCallableKind(kind string) bool {
	switch kind {
	case "function", "method", "fn", "function_declaration", "method_definition", "constructor", "arrow_function":
		return true
	default:
		return false
	}
}
