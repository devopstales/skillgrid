package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runSearchPdg implements `skillgrid search pdg SYMBOL STATEMENT` (CLI parity
// for code_pdg_query): the control- and data-dependents of a statement inside
// a function, from the opt-in --pdg tables. A non---pdg index returns a clear
// "run index --pdg" message (not an error); an unknown symbol or statement is
// not-found (no fabricated dependences).
func runSearchPdg(version string, args []string) {
	fs := flag.NewFlagSet("search pdg", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		jsonOut bool
		kind    string
	)
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.StringVar(&kind, "kind", "", "narrow disambiguation to a symbol kind")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid search pdg SYMBOL STATEMENT_LINE [--kind K] [--json]")
		fmt.Fprintln(fs.Output(), "  Dependents of a statement in a function's --pdg CFG/PDG.")
		fs.PrintDefaults()
	}
	fsArgs := args
	if len(fsArgs) >= 2 && fsArgs[0] == "search" && fsArgs[1] == "pdg" {
		fsArgs = fsArgs[2:]
	} else if len(fsArgs) >= 1 && fsArgs[0] == "pdg" {
		fsArgs = fsArgs[1:]
	}
	if err := fs.Parse(fsArgs); err != nil {
		os.Exit(2)
	}
	// Go's flag package stops at the first non-flag, so a trailing `--json`
	// after the positionals becomes a positional arg. Re-apply the common
	// `--json` tail (and `--kind=X`) so `search pdg SYMBOL LINE --json`
	// works in any order. A bare `--kind K` (value as next positional) must
	// precede the positionals per Go flag conventions.
	var positional []string
	for _, a := range fs.Args() {
		switch {
		case a == "--json" || a == "-json":
			jsonOut = true
		case strings.HasPrefix(a, "--kind="):
			kind = strings.TrimPrefix(a, "--kind=")
		case strings.HasPrefix(a, "-kind="):
			kind = strings.TrimPrefix(a, "-kind=")
		case len(a) > 0 && a[0] == '-':
			// some other unknown flag in the tail: ignore (already declared
			// flags would have been consumed by fs.Parse above)
		default:
			positional = append(positional, a)
		}
	}
	if len(positional) != 2 {
		fmt.Fprintln(fs.Output(), "usage: skillgrid search pdg SYMBOL STATEMENT_LINE [--kind K] [--json]")
		os.Exit(2)
	}
	symbol := positional[0]
	line, err := strconv.Atoi(positional[1])
	if err != nil || line < 1 {
		fmt.Fprintf(fs.Output(), "error: STATEMENT_LINE must be a 1-based line number, got %q\n", positional[1])
		os.Exit(2)
	}

	svc, projectID, ferr := openSearchPdgService()
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
		os.Exit(1)
	}
	defer cleanup()

	if ferr := printPdgQuery(h, symbol, line, kind, jsonOut); ferr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", ferr)
		os.Exit(1)
	}
}

// openSearchPdgService resolves the service + project for the PDG query. It is
// a var so tests can stub it (openGraphService is otherwise fine but the stub
// keeps the test hermetic and avoids re-resolving the cwd project).
var openSearchPdgService = openGraphService

// pdgDependentCli is one control- or data-dependence row.
type pdgDependentCli struct {
	Kind       string `json:"kind"`
	FromLine   int    `json:"from_line"`
	ToLine     int    `json:"to_line"`
	FromName   string `json:"from_name,omitempty"`
	ToName     string `json:"to_name,omitempty"`
	Confidence string `json:"confidence"`
	Note       string `json:"note,omitempty"`
}

// printPdgQuery resolves the symbol + statement and prints (or JSONs) the
// statement's dependents. It distinguishes three not-found cases (no pdg
// index, unknown symbol, statement not in the CFG) from a hard store error.
func printPdgQuery(h *service.ProjectHandle, symbol string, line int, kind string, jsonOut bool) error {
	db := h.Store().DB

	// A non---pdg index has the tables but they are empty: say so, don't error.
	var pdgRowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pdg_edges`).Scan(&pdgRowCount); err != nil {
		return fmt.Errorf("read pdg_edges: %w", err)
	}
	if pdgRowCount == 0 {
		msg := "run `skillgrid index --pdg` to build the per-function CFG/PDG"
		if jsonOut {
			return emitJSON(map[string]any{"pdg_enabled": false, "found": false, "dependents": []pdgDependentCli{}, "message": msg})
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}

	// Resolve the symbol to a function/method (callable kinds only; a kind
	// filter, when given, must match).
	rows, err := db.Query(`SELECT id, kind FROM symbols WHERE name = ?`, symbol)
	if err != nil {
		return fmt.Errorf("resolve symbol: %w", err)
	}
	var symID int64
	var symKind string
	found := false
	for rows.Next() {
		var id int64
		var k string
		if err := rows.Scan(&id, &k); err != nil {
			rows.Close()
			return fmt.Errorf("scan symbol: %w", err)
		}
		if kind != "" && k != kind {
			continue
		}
		if !isCallableKindCli(k) {
			continue
		}
		if !found {
			symID, symKind, found = id, k, true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate symbols: %w", err)
	}
	if !found {
		msg := "not found: no function/method symbol named " + quote(symbol)
		if jsonOut {
			return emitJSON(map[string]any{"symbol": symbol, "statement": line, "found": false, "dependents": []pdgDependentCli{}, "message": msg})
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}

	// Does the statement sit inside the function's CFG?
	var blockCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cfg_blocks WHERE symbol_id = ? AND (start_line <= ? AND ? <= end_line)`, symID, line, line).Scan(&blockCount); err != nil {
		return fmt.Errorf("read cfg_blocks: %w", err)
	}
	if blockCount == 0 {
		msg := fmt.Sprintf("not found: statement at line %d is not in the CFG of %s", line, quote(symbol))
		if jsonOut {
			return emitJSON(map[string]any{"symbol": symbol, "statement": line, "found": false, "dependents": []pdgDependentCli{}, "message": msg})
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}

	dependents := []pdgDependentCli{}
	drows, err := db.Query(`SELECT kind, from_line, to_line, from_name, to_name, confidence, note
		FROM pdg_edges WHERE symbol_id = ? AND from_line = ?
		ORDER BY kind, to_line, to_name`, symID, line)
	if err != nil {
		return fmt.Errorf("query pdg_edges: %w", err)
	}
	for drows.Next() {
		var d pdgDependentCli
		if err := drows.Scan(&d.Kind, &d.FromLine, &d.ToLine, &d.FromName, &d.ToName, &d.Confidence, &d.Note); err != nil {
			drows.Close()
			return fmt.Errorf("scan pdg_edge: %w", err)
		}
		dependents = append(dependents, d)
	}
	drows.Close()
	if err := drows.Err(); err != nil {
		return fmt.Errorf("iterate pdg_edges: %w", err)
	}

	if jsonOut {
		return emitJSON(map[string]any{
			"symbol": symbol, "statement": line, "kind": symKind,
			"found": true, "dependents": dependents,
		})
	}
	fmt.Printf("%s (kind=%s, line %d): %d dependence(s)\n", symbol, symKind, line, len(dependents))
	for _, d := range dependents {
		fmt.Printf("  [%s] line %d -> line %d (%s)%s\n", d.Kind, d.FromLine, d.ToLine, d.Confidence, noteSuffix(d.Note))
	}
	return nil
}

func emitJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}

func isCallableKindCli(kind string) bool {
	switch kind {
	case "function", "method", "fn", "function_declaration", "method_definition", "constructor", "arrow_function":
		return true
	default:
		return false
	}
}

func quote(s string) string { return `"` + s + `"` }

func noteSuffix(note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return ""
	}
	return " (" + note + ")"
}

var _ = sql.ErrNoRows
