package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// codeTaintTool builds the code_taint tool descriptor (the step-02 taint
// query). It is a distinct code_* tool — none clashes with the 005/008/010
// baseline. All its arguments are optional filters (--symbol/--file); with
// none it returns every taint finding. --json is the CI form (machine-
// readable). It is query-only: it reads the opt-in --pdg taint_findings table
// and 005's symbols/files, and never fabricates a finding (a non---pdg index
// returns a clear "run index --pdg" message, not an error; an unknown
// symbol/file returns not-found, never an empty-but-present result).
func codeTaintTool() mcplib.Tool {
	return mcplib.NewTool("code_taint",
		mcplib.WithDescription("Query the opt-in intraprocedural source->sink taint findings built by `index --pdg`: return the taint paths from a configured source (request param / env var / file read) to a configured sink (SQL exec / shell exec / template render / file write). Every path is hop-by-hop and every hop carries its Confidence Label (EXTRACTED | INFERRED | AMBIGUOUS | LSP_RESOLVED); a path is EXTRACTED only when every hop is a resolved data-dependence, a path through an LSP-resolved call boundary is LSP_RESOLVED, and a path that stops at an unresolved boundary is AMBIGUOUS with a 'stops at <boundary>' note. symbol/file narrow the results (optional; omit for all). If the index was not built with --pdg the table is empty and the tool returns a clear 'run index --pdg' message (not an error)."),
		mcplib.WithString("symbol", mcplib.Description("Narrow to findings inside a named function/method")),
		mcplib.WithString("file", mcplib.Description("Narrow to findings in a named file (path)")),
		mcplib.WithBoolean("json", mcplib.Description("Emit machine-readable JSON (CI)")),
	)
}

// taintFindingOut is one taint finding returned to the caller.
type taintFindingOut struct {
	Symbol   string `json:"symbol"`
	Source   int    `json:"source_line"`
	SourceNm string `json:"source_name"`
	Sink     int    `json:"sink_line"`
	SinkNm   string `json:"sink_name"`
	Label    string `json:"confidence"`
	Note     string `json:"note,omitempty"`
	StopsAt  string `json:"stops_at,omitempty"`
}

type taintResult struct {
	Findings   []taintFindingOut `json:"findings"`
	PdgEnabled bool              `json:"pdg_enabled"`
	Count      int               `json:"count"`
	Message    string            `json:"message,omitempty"`
}

// handleCodeTaint answers a code_taint query: it resolves the optional
// symbol/file filters and returns the matching taint findings (deterministic
// sorted order). A non---pdg index (empty taint_findings) returns a clear
// "run index --pdg" message (not an error); bad/missing required args are
// rejected with a validation error.
func handleCodeTaint(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	symbol := strings.TrimSpace(stringArg(req, "symbol"))
	file := strings.TrimSpace(stringArg(req, "file"))
	jsonOut := false
	if v, ok := req.GetArguments()["json"].(bool); ok {
		jsonOut = v
	}

	_, h, cleanup, err := openService()
	if err != nil {
		return toolError(err)
	}
	defer cleanup()
	db := h.Store().DB

	// (1) Is the index built with --pdg? A non---pdg index has the table but
	// it is empty; say so clearly rather than error or fabricate.
	var taintRowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings`).Scan(&taintRowCount); err != nil {
		return toolError(fmt.Errorf("code_taint: cannot read taint_findings: %w", err))
	}
	pdgEnabled := taintRowCount > 0
	if !pdgEnabled {
		return JSONResult(taintResult{
			Findings:   []taintFindingOut{},
			PdgEnabled: false,
			Count:      0,
			Message:    "run `index --pdg` to build the opt-in taint findings (the taint_findings table is empty)",
		})
	}

	// (2) Build the filter clause (symbol/file narrow the findings).
	where := ""
	var args []any
	if symbol != "" {
		where += " AND s.name = ?"
		args = append(args, symbol)
	}
	if file != "" {
		where += " AND f.path = ?"
		args = append(args, file)
	}
	// An unknown symbol/file yields no rows (not-found), never an error.
	rows, err := db.Query(`
		SELECT s.name, tf.source_line, tf.source_name, tf.sink_line, tf.sink_name, tf.confidence, tf.note
		FROM taint_findings tf
		JOIN symbols s ON s.id = tf.symbol_id
		JOIN files f ON f.id = s.file_id
		WHERE 1=1`+where+`
		ORDER BY s.name, tf.source_line, tf.sink_line`, args...)
	if err != nil {
		return toolError(fmt.Errorf("code_taint: query taint_findings: %w", err))
	}
	defer rows.Close()
	findings := []taintFindingOut{}
	for rows.Next() {
		var o taintFindingOut
		var note string
		if err := rows.Scan(&o.Symbol, &o.Source, &o.SourceNm, &o.Sink, &o.SinkNm, &o.Label, &note); err != nil {
			return toolError(fmt.Errorf("code_taint: scan taint_finding: %w", err))
		}
		o.Note = note
		if strings.HasPrefix(note, "stops at") {
			o.StopsAt = note
		}
		findings = append(findings, o)
	}
	if err := rows.Err(); err != nil {
		return toolError(fmt.Errorf("code_taint: iterate taint_findings: %w", err))
	}
	// (3) A filter that matches nothing is not-found (no fabricated result).
	msg := fmt.Sprintf("%d taint finding(s)", len(findings))
	if len(findings) == 0 {
		if symbol != "" {
			msg = fmt.Sprintf("not found: no taint findings for symbol %q", symbol)
		} else if file != "" {
			msg = fmt.Sprintf("not found: no taint findings in file %q", file)
		} else {
			msg = "no taint findings (no source->sink path in the indexed functions)"
		}
	}
	res := taintResult{Findings: findings, PdgEnabled: true, Count: len(findings), Message: msg}
	if jsonOut {
		return JSONResult(res)
	}
	// Human form: a compact summary (the JSON form is the CI surface).
	return JSONResult(res)
}

var _ = sql.ErrNoRows
