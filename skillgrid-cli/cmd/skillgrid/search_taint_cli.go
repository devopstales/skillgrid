package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runSearchTaint implements `skillgrid search taint` (CLI parity for
// code_taint): the intraprocedural source->sink taint findings from the opt-in
// --pdg tables. A non---pdg index returns a clear "run index --pdg" message
// (not an error); an unknown symbol/file is not-found (no fabricated findings).
// --symbol/--file narrow the findings; --json emits machine-readable JSON.
func runSearchTaint(version string, args []string) {
	fs := flag.NewFlagSet("search taint", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		jsonOut bool
		symbol  string
		file    string
	)
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.StringVar(&symbol, "symbol", "", "narrow to findings inside a named function/method")
	fs.StringVar(&file, "file", "", "narrow to findings in a named file (path)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid search taint [--symbol S] [--file F] [--json]")
		fmt.Fprintln(fs.Output(), "  Intraprocedural source->sink taint findings from a --pdg index.")
		fs.PrintDefaults()
	}
	fsArgs := args
	if len(fsArgs) >= 2 && fsArgs[0] == "search" && fsArgs[1] == "taint" {
		fsArgs = fsArgs[2:]
	} else if len(fsArgs) >= 1 && fsArgs[0] == "taint" {
		fsArgs = fsArgs[1:]
	}
	if err := fs.Parse(fsArgs); err != nil {
		os.Exit(2)
	}
	// Re-apply a trailing `--json` after the positionals (Go flag stops at the
	// first non-flag).
	for _, a := range fs.Args() {
		if a == "--json" || a == "-json" {
			jsonOut = true
		}
	}

	svc, projectID, ferr := openSearchTaintService()
	if ferr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", ferr)
		os.Exit(1)
	}
	_ = version
	_, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	defer cleanup()

	if ferr := printTaintQuery(h, symbol, file, jsonOut); ferr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", ferr)
		os.Exit(1)
	}
}

// openSearchTaintService resolves the service + project for the taint query
// (a var so tests can stub it, mirroring openSearchPdgService).
var openSearchTaintService = openGraphService

type taintRowCli struct {
	Symbol   string `json:"symbol"`
	Source   int    `json:"source_line"`
	SourceNm string `json:"source_name"`
	Sink     int    `json:"sink_line"`
	SinkNm   string `json:"sink_name"`
	Label    string `json:"confidence"`
	Note     string `json:"note,omitempty"`
}

// printTaintQuery resolves the optional symbol/file filters and prints (or
// JSONs) the matching taint findings. A non---pdg index (empty taint_findings)
// returns a clear "run index --pdg" message, not an error.
func printTaintQuery(h *service.ProjectHandle, symbol, file string, jsonOut bool) error {
	db := h.Store().DB
	// A non---pdg index has the table but it is empty: say so, don't error.
	var rowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM taint_findings`).Scan(&rowCount); err != nil {
		return fmt.Errorf("read taint_findings: %w", err)
	}
	if rowCount == 0 {
		msg := "run `skillgrid index --pdg` to build the opt-in taint findings"
		if jsonOut {
			return emitJSON(map[string]any{"pdg_enabled": false, "found": false, "findings": []taintRowCli{}, "message": msg})
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}
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
	rows, err := db.Query(`
		SELECT s.name, tf.source_line, tf.source_name, tf.sink_line, tf.sink_name, tf.confidence, tf.note
		FROM taint_findings tf
		JOIN symbols s ON s.id = tf.symbol_id
		JOIN files f ON f.id = s.file_id
		WHERE 1=1`+where+`
		ORDER BY s.name, tf.source_line, tf.sink_line`, args...)
	if err != nil {
		return fmt.Errorf("query taint_findings: %w", err)
	}
	defer rows.Close()
	findings := []taintRowCli{}
	for rows.Next() {
		var r taintRowCli
		if err := rows.Scan(&r.Symbol, &r.Source, &r.SourceNm, &r.Sink, &r.SinkNm, &r.Label, &r.Note); err != nil {
			return fmt.Errorf("scan taint_finding: %w", err)
		}
		findings = append(findings, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate taint_findings: %w", err)
	}
	if jsonOut {
		return emitJSON(map[string]any{"pdg_enabled": true, "found": len(findings) > 0, "findings": findings, "count": len(findings)})
	}
	if len(findings) == 0 {
		if symbol != "" {
			fmt.Fprintf(os.Stderr, "not found: no taint findings for symbol %q\n", symbol)
			return nil
		}
		if file != "" {
			fmt.Fprintf(os.Stderr, "not found: no taint findings in file %q\n", file)
			return nil
		}
		fmt.Fprintln(os.Stderr, "no taint findings")
		return nil
	}
	fmt.Printf("%d taint finding(s)\n", len(findings))
	for _, f := range findings {
		fmt.Printf("  %s: %s -> %s [%s]%s\n", f.Symbol, f.SourceNm, f.SinkNm, f.Label, taintNoteSuffix(f.Note))
	}
	return nil
}

func taintNoteSuffix(note string) string {
	note = trimSpaceTaint(note)
	if note == "" {
		return ""
	}
	return " (" + note + ")"
}

func trimSpaceTaint(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

var _ = sql.ErrNoRows
