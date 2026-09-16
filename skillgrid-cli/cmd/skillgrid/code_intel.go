package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"database/sql"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runCodeIntel handles the `skillgrid orient` and `skillgrid grep` subcommands
// (CLI parity for the Tier-1 orientation and structural code_grep MCP tools).
func runCodeIntel(version string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: skillgrid <orient|grep> ... (see --help)")
		os.Exit(2)
	}
	switch args[0] {
	case "orient":
		runOrient(version, args[1:])
	case "grep":
		runGrep(version, args[1:])
	case "callers", "callees", "dependents", "implementors", "hierarchy", "tests-for":
		runNeighbors(version, args[0], args[1:])
	case "path":
		runGraphPath(version, args[1:])
	case "explain":
		runExplain(version, args[1:])
	case "impact":
		runImpact(version, args[1:])
	case "explore":
		runExplore(version, args[1:])
	case "communities":
		runCommunities(version, args[1:])
	case "god-nodes":
		runGodNodes(version, args[1:])
	case "explain-community":
		runExplainCommunity(version, args[1:])
	case "docs", "configs", "sql-schema", "sql-access":
		runKnowledge(version, args[0], args[1:])
	case "help", "-h", "--help":
		printCodeIntelUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown code_intel subcommand %q (see --help)\n", args[0])
		os.Exit(2)
	}
}

func printCodeIntelUsage() {
	fmt.Fprintln(os.Stderr, `usage: skillgrid <orient|grep> [flags]

  skillgrid orient SYMBOL  Tier-1 orientation for a symbol: signature, file TOC,
                           map, list, metadata, and linked rationale.
  skillgrid grep PATTERN [path]  Structural by-example grep (index-free):
                           matches the gotreesitter syntax tree, per language.

  callers|callees|dependents|implementors|hierarchy|tests-for SYMBOL
                           Graph neighbors (every edge confidence-labeled)
  path FROM TO            Shortest edge path, or where the graph stops
  explain SYMBOL          Symbol node + degree + connections ranked by degree
   impact SYMBOL           Risk-tiered blast radius (WILL BREAK / LIKELY AFFECTED)
   explore SYMBOL          Composite: source + call-flow + blast radius in one call
  communities             Leiden-clustered subsystems (LLM-free labels)
  god-nodes [--exclude-hubs] [--limit N]  Most-connected symbols by degree
  explain-community ID    A community's members + key entry points
  docs [PATH]             Indexed markdown docs + references edges
  configs [PATH]          Indexed config files + configures edges
  sql-schema [TABLE]      Indexed SQL schema (tables + columns)
  sql-access [TABLE]      reads/writes edges between code and SQL tables

  graph flags:
    --json    Emit machine-readable JSON (default: human table)
  impact:
    --file F --uid U --kind K --min-confidence C --max-depth N
  grep:
    --json    Emit machine-readable JSON (default: human table)`)
}

// graphView maps a CLI subcommand to its graph view.
func graphView(sub string) string {
	views := map[string]string{
		"callers":      "callers",
		"callees":      "callees",
		"dependents":   "dependents",
		"implementors": "implementors",
		"hierarchy":    "hierarchy",
		"tests-for":    "tests_for",
	}
	if v, ok := views[sub]; ok {
		return v
	}
	return sub
}

// runKnowledge implements the knowledge-graph query subcommands (CLI parity
// for the code_docs / code_configs / code_sql_schema / code_sql_access MCP
// tools).
func runKnowledge(version, sub string, args []string) {
	fs := flag.NewFlagSet(sub, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: skillgrid %s [TARGET] [--json]\n", sub)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	target := ""
	if fs.NArg() >= 1 {
		target = fs.Arg(0)
	}
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
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
	db := h.Store().DB
	var out map[string]any
	switch sub {
	case "docs":
		out, err = knowledgeDocsCLI(db, target)
	case "configs":
		out, err = knowledgeConfigsCLI(db, target)
	case "sql-schema":
		out, err = knowledgeSQLSchemaCLI(db, target)
	case "sql-access":
		out, err = knowledgeSQLAccessCLI(db, target)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	printKnowledge(sub, out)
}

// printKnowledge renders a knowledge query result as a human table.
func printKnowledge(sub string, out map[string]any) {
	switch sub {
	case "docs":
		docs, _ := out["docs"].([]map[string]any)
		if len(docs) == 0 {
			fmt.Fprintln(os.Stderr, "no docs indexed")
			return
		}
		for _, d := range docs {
			fmt.Printf("%s (%s)\n", d["path"], d["title"])
			refs, _ := d["references"].([]map[string]any)
			for _, r := range refs {
				fmt.Printf("  -> %s [%s] :%v\n", r["target"], r["confidence"], r["line"])
			}
		}
	case "configs":
		configs, _ := out["configs"].([]map[string]any)
		if len(configs) == 0 {
			fmt.Fprintln(os.Stderr, "no configs indexed")
			return
		}
		for _, c := range configs {
			fmt.Printf("%s (%s)\n", c["path"], c["title"])
			refs, _ := c["configures"].([]map[string]any)
			for _, r := range refs {
				fmt.Printf("  configures %v [%s] :%v\n", r["target"], r["confidence"], r["line"])
			}
		}
	case "sql-schema":
		tables, _ := out["tables"].([]map[string]any)
		if len(tables) == 0 {
			fmt.Fprintln(os.Stderr, "no sql schema indexed")
			return
		}
		for _, tb := range tables {
			fmt.Printf("table %s (%s)\n", tb["table"], tb["path"])
			cols, _ := tb["columns"].([]string)
			for _, c := range cols {
				fmt.Printf("  column %s\n", c)
			}
		}
	case "sql-access":
		access, _ := out["access"].([]map[string]any)
		if len(access) == 0 {
			fmt.Fprintln(os.Stderr, "no sql access indexed")
			return
		}
		for _, a := range access {
			fmt.Printf("%s %s -> table %v [%s] %s:%v\n", a["symbol"], a["op"], a["table"], a["confidence"], a["path"], a["line"])
		}
	}
}

// knowledgeDocsCLI backs `skillgrid docs` (query-only over doc_nodes).
func knowledgeDocsCLI(db *sql.DB, path string) (map[string]any, error) {
	where := ""
	var args []any
	if path != "" {
		where = " WHERE dn.path = ?"
		args = append(args, path)
	}
	rows, err := db.Query(`SELECT dn.path, dn.title, COALESCE(e.to_name,''), COALESCE(e.target_path,''), COALESCE(e.confidence,''), COALESCE(e.line,0)
		FROM doc_nodes dn LEFT JOIN edges e ON e.from_id = dn.id AND e.kind = 'references'`+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	docs := map[string]map[string]any{}
	var order []string
	for rows.Next() {
		var p, title, toName, target, conf string
		var line int
		if err := rows.Scan(&p, &title, &toName, &target, &conf, &line); err != nil {
			return nil, err
		}
		d, ok := docs[p]
		if !ok {
			d = map[string]any{"path": p, "title": title, "references": []map[string]any{}}
			docs[p] = d
			order = append(order, p)
		}
		if toName != "" {
			refs := d["references"].([]map[string]any)
			d["references"] = append(refs, map[string]any{"target": target, "confidence": conf, "line": line})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(order))
	for _, p := range order {
		out = append(out, docs[p])
	}
	return map[string]any{"docs": out, "total": len(out)}, nil
}

// knowledgeConfigsCLI backs `skillgrid configs` (query-only over config_nodes).
func knowledgeConfigsCLI(db *sql.DB, path string) (map[string]any, error) {
	where := ""
	var args []any
	if path != "" {
		where = " WHERE cn.path = ?"
		args = append(args, path)
	}
	rows, err := db.Query(`SELECT cn.path, cn.title, COALESCE(e.to_name,''), COALESCE(s.name,''), COALESCE(e.confidence,''), COALESCE(e.line,0)
		FROM config_nodes cn LEFT JOIN edges e ON e.from_id = cn.id AND e.kind = 'configures'
		LEFT JOIN symbols s ON s.id = e.to_id`+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	configs := map[string]map[string]any{}
	var order []string
	for rows.Next() {
		var p, title, toName, symName, conf string
		var line int
		if err := rows.Scan(&p, &title, &toName, &symName, &conf, &line); err != nil {
			return nil, err
		}
		c, ok := configs[p]
		if !ok {
			c = map[string]any{"path": p, "title": title, "configures": []map[string]any{}}
			configs[p] = c
			order = append(order, p)
		}
		if toName != "" {
			refs := c["configures"].([]map[string]any)
			c["configures"] = append(refs, map[string]any{"target": toName, "symbol": symName, "confidence": conf, "line": line})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(order))
	for _, p := range order {
		out = append(out, configs[p])
	}
	return map[string]any{"configs": out, "total": len(out)}, nil
}

// knowledgeSQLSchemaCLI backs `skillgrid sql-schema` (query-only over
// sql_schema_nodes).
func knowledgeSQLSchemaCLI(db *sql.DB, table string) (map[string]any, error) {
	where := ""
	var args []any
	if table != "" {
		where = " WHERE ss.table_name = ?"
		args = append(args, table)
	}
	rows, err := db.Query(`SELECT ss.table_name, ss.column_name, ss.kind, ss.path
		FROM sql_schema_nodes ss`+where+` ORDER BY ss.table_name, ss.kind, ss.column_name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tables := map[string]map[string]any{}
	var order []string
	for rows.Next() {
		var tn, col, kind, p string
		if err := rows.Scan(&tn, &col, &kind, &p); err != nil {
			return nil, err
		}
		tbl, ok := tables[tn]
		if !ok {
			tbl = map[string]any{"table": tn, "path": p, "columns": []string{}}
			tables[tn] = tbl
			order = append(order, tn)
		}
		if kind == "column" && col != "" {
			cols := tbl["columns"].([]string)
			tbl["columns"] = append(cols, col)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(order))
	for _, tn := range order {
		out = append(out, tables[tn])
	}
	return map[string]any{"tables": out, "total": len(out)}, nil
}

// knowledgeSQLAccessCLI backs `skillgrid sql-access` (query-only over the
// reads/writes edges).
func knowledgeSQLAccessCLI(db *sql.DB, table string) (map[string]any, error) {
	where := ""
	var args []any
	if table != "" {
		where = " WHERE e.to_name = ?"
		args = append(args, table)
	}
	rows, err := db.Query(`SELECT e.kind, s.name, f.path, e.to_name, e.confidence, e.line
		FROM edges e JOIN symbols s ON s.id = e.from_id JOIN files f ON f.id = s.file_id
		WHERE e.kind IN ('reads','writes')`+where+` ORDER BY e.to_name, e.kind, s.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var kind, sym, p, tn, conf string
		var line int
		if err := rows.Scan(&kind, &sym, &p, &tn, &conf, &line); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"op": kind, "symbol": sym, "path": p, "table": tn, "confidence": conf, "line": line})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return map[string]any{"access": out, "total": len(out)}, nil
}

// openGraphService resolves the project for the current directory.
func openGraphService() (*service.Service, string, error) {
	// newMnemonicService honors the injected cliService (test harness) and the
	// SKILLGRID_MNEMONIC_DATA_DIR env (real invocation).
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		return nil, "", err
	}
	projectID, err := svc.ResolveProject(".")
	if err != nil {
		return nil, "", err
	}
	return svc, projectID, nil
}

// printImpact renders a risk-tiered blast-radius result (or a candidate list).
func printImpact(out *service.ImpactResultDTO) {
	if out.Ambiguous {
		fmt.Printf("ambiguous target: %d candidates (narrow with --file/--uid/--kind)\n", len(out.Candidates))
		for i, c := range out.Candidates {
			fmt.Printf("  %d. %s (%s) %s:%d [uid %s]\n", i+1, c.Name, c.Kind, c.Path, c.StartLine, c.UID)
		}
		return
	}
	if out.Target == nil {
		fmt.Println("not found: no matching symbol")
		return
	}
	fmt.Printf("%s (%s) %s:%d\n", out.Target.Name, out.Target.Kind, out.Target.Path, out.Target.StartLine)
	fmt.Printf("WILL BREAK (%d):\n", len(out.WillBreak))
	for _, e := range out.WillBreak {
		fmt.Printf("  %s (%s) %s:%d [%s %s] depth %d\n", e.Symbol.Name, e.Symbol.Kind, e.Symbol.Path, e.Symbol.StartLine, e.Kind, e.Confidence, e.Depth)
	}
	fmt.Printf("LIKELY AFFECTED (%d):\n", len(out.Likely))
	for _, e := range out.Likely {
		fmt.Printf("  %s (%s) %s:%d [%s %s] depth %d\n", e.Symbol.Name, e.Symbol.Kind, e.Symbol.Path, e.Symbol.StartLine, e.Kind, e.Confidence, e.Depth)
	}
	if out.Excluded > 0 {
		fmt.Printf("excluded low-confidence hops: %d\n", out.Excluded)
	}
}

// printGraphStops renders a where-the-graph-stops answer.
func printGraphStops(g *graph.GraphStops) {
	fmt.Printf("where the graph stops: %s at line %d\n", g.DispatchKind, g.Line)
	for _, r := range g.Refused {
		if r.Path != "" {
			fmt.Printf("  refused: %s [%s] at %s:%d\n", r.Name, r.Confidence, r.Path, r.Line)
		} else {
			fmt.Printf("  refused: %s [%s] at line %d\n", r.Name, r.Confidence, r.Line)
		}
	}
}

// runOrient implements `skillgrid orient SYMBOL`.
func runOrient(version string, args []string) {
	fs := flag.NewFlagSet("orient", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid orient SYMBOL [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: orient requires exactly one SYMBOL argument")
		os.Exit(2)
	}
	symbol := fs.Arg(0)
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	projectID, err := svc.ResolveProject(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.OrientSymbol(ctx, projectID, symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if !out.Found {
		fmt.Fprintf(os.Stderr, "not found: %s\n", out.Reason)
		return
	}
	s := out.Symbol
	fmt.Printf("%s (%s) %s:%d-%d\n", s["name"], s["kind"], s["path"], s["start_line"], s["end_line"])
	if out.Signature != "" {
		fmt.Printf("  %s\n", out.Signature)
	}
	fmt.Printf("file TOC (%d symbols):\n", len(out.FileTOC))
	for _, e := range out.FileTOC {
		mark := "  "
		if e["name"] == s["name"] {
			mark = "  * "
		}
		fmt.Printf("%s%s (%s) :%d-%d\n", mark, e["name"], e["kind"], e["start_line"], e["end_line"])
	}
	if len(out.Rationale) > 0 {
		fmt.Printf("rationale:\n")
		for _, r := range out.Rationale {
			fmt.Printf("  [%s] %s:%d %v\n", r["kind"], s["path"], r["line"], r["text"])
		}
	}
}

// runGrep implements `skillgrid grep PATTERN [path]`.
func runGrep(version string, args []string) {
	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid grep PATTERN [path] [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: grep requires a PATTERN argument")
		os.Exit(2)
	}
	pattern := fs.Arg(0)
	path := "."
	if fs.NArg() >= 2 {
		path = fs.Arg(1)
	}
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	res, err := svc.CodeGrep(ctx, path, pattern)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	for _, n := range res.Notes {
		fmt.Fprintf(os.Stderr, "note: skipping %s files: %s\n", n.Language, n.Reason)
	}
	if len(res.Hits) == 0 && len(res.Notes) == 0 {
		fmt.Fprintln(os.Stderr, "no matches")
		return
	}
	for _, h := range res.Hits {
		fmt.Printf("%s:%d:%d  %s\n", h.Path, h.Line, h.Col, h.Text)
	}
}

// runNeighbors implements the graph neighbor subcommands (CLI parity for the
// Tier-2 code_get_* MCP tools).
func runNeighbors(version, sub string, args []string) {
	fs := flag.NewFlagSet(sub, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: skillgrid %s SYMBOL [--json]\n", sub)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "error: %s requires exactly one SYMBOL argument\n", sub)
		os.Exit(2)
	}
	symbol := fs.Arg(0)
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.GrabNeighbors(ctx, projectID, graphView(sub), symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if out.Reason != "" {
		fmt.Fprintln(os.Stderr, "note:", out.Reason)
	}
	if len(out.Edges) == 0 {
		fmt.Fprintln(os.Stderr, "no edges")
		return
	}
	for _, e := range out.Edges {
		to := e.ToName
		if e.To.ID != 0 {
			to = e.To.Name
		}
		fmt.Printf("%s -> %s [%s %s] :%d\n", e.From.Name, to, e.Kind, e.Confidence, e.Line)
	}
}

// runGraphPath implements `skillgrid path FROM TO`.
func runGraphPath(version string, args []string) {
	fs := flag.NewFlagSet("path", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid path FROM TO [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "error: path requires FROM and TO arguments")
		os.Exit(2)
	}
	from, to := fs.Arg(0), fs.Arg(1)
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodePath(ctx, projectID, from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if out.Reason != "" {
		fmt.Fprintln(os.Stderr, "note:", out.Reason)
		return
	}
	if !out.Found {
		if out.GraphStops != nil {
			printGraphStops(out.GraphStops)
		} else {
			fmt.Fprintln(os.Stderr, "no path")
		}
		return
	}
	if len(out.Path) == 0 {
		fmt.Println(from, "==", to)
		return
	}
	prev := from
	for _, e := range out.Path {
		next := e.ToName
		if e.To.ID != 0 {
			next = e.To.Name
		}
		fmt.Printf("%s -> %s [%s %s] :%d\n", prev, next, e.Kind, e.Confidence, e.Line)
		prev = next
	}
}

// runExplain implements `skillgrid explain SYMBOL`.
func runExplain(version string, args []string) {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid explain SYMBOL [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: explain requires exactly one SYMBOL argument")
		os.Exit(2)
	}
	symbol := fs.Arg(0)
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodeExplain(ctx, projectID, symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if !out.Found {
		fmt.Fprintln(os.Stderr, "not found:", out.Reason)
		return
	}
	fmt.Printf("%s degree %d\n", out.Symbol.Name, out.Degree)
	for _, c := range out.Connections {
		name := c.Symbol.Name
		if name == "" {
			name = c.ToName
		}
		fmt.Printf("  %s [%s %s] :%d (degree %d)\n", name, c.Kind, c.Confidence, c.Line, c.Degree)
	}
}

// runImpact implements `skillgrid impact SYMBOL`.
func runImpact(version string, args []string) {
	fs := flag.NewFlagSet("impact", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	var file, uid, kind, minConf string
	var maxDepth int
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.StringVar(&file, "file", "", "narrow to a file path")
	fs.StringVar(&uid, "uid", "", "narrow to a symbol UID")
	fs.StringVar(&kind, "kind", "", "narrow to a symbol kind")
	fs.StringVar(&minConf, "min-confidence", "", "minimum edge confidence (EXTRACTED|INFERRED|AMBIGUOUS)")
	fs.IntVar(&maxDepth, "max-depth", 0, "bound the traversal depth")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid impact SYMBOL [--file F] [--uid U] [--kind K] [--min-confidence C] [--max-depth N] [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: impact requires exactly one SYMBOL argument")
		os.Exit(2)
	}
	symbol := fs.Arg(0)
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodeImpact(ctx, projectID, symbol, service.ImpactOptions{
		File: file, UID: uid, Kind: kind, MinConfidence: minConf, MaxDepth: maxDepth,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	printImpact(out)
}

// runExplore implements `skillgrid explore SYMBOL` (CLI parity for the
// composite code_explore).
func runExplore(version string, args []string) {
	fs := flag.NewFlagSet("explore", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid explore SYMBOL [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: explore requires exactly one SYMBOL argument")
		os.Exit(2)
	}
	symbol := fs.Arg(0)
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodeExplore(ctx, projectID, symbol)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	for path, spans := range out.Source {
		fmt.Printf("== %s\n", path)
		for _, sp := range spans {
			fmt.Printf("-- %s (%d-%d)\n%s\n", sp.Symbol, sp.StartLine, sp.EndLine, sp.Content)
		}
	}
	if len(out.Flow) > 0 {
		fmt.Println("call flow:")
		for _, e := range out.Flow {
			fmt.Printf("  %s -> %s [%s %s] :%d\n", e.From, e.To, e.Kind, e.Confidence, e.Line)
		}
	}
	if out.Impact != nil {
		fmt.Println("blast radius:")
		printImpact(out.Impact)
	}
}

// runCommunities implements `skillgrid communities` (CLI parity for the
// code_communities MCP tool).
func runCommunities(version string, args []string) {
	fs := flag.NewFlagSet("communities", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid communities [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodeCommunities(ctx, projectID, community.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if out.Warning != "" {
		fmt.Fprintf(os.Stderr, "note: %s\n", out.Warning)
	}
	for _, c := range out.Communities {
		fmt.Printf("community %d: %s (%d symbols)\n", c.ID, c.Label, len(c.Members))
		for _, g := range c.GodNodes {
			fmt.Printf("  god node: %s\n", g)
		}
	}
}

// runGodNodes implements `skillgrid god-nodes` (CLI parity for code_god_nodes).
func runGodNodes(version string, args []string) {
	fs := flag.NewFlagSet("god-nodes", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut, excludeHubs bool
	var limit int
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.BoolVar(&excludeHubs, "exclude-hubs", false, "suppress utility super-hubs")
	fs.IntVar(&limit, "limit", 20, "maximum god nodes to return")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid god-nodes [--exclude-hubs] [--limit N] [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodeGodNodes(ctx, projectID, excludeHubs, limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(map[string]any{"god_nodes": out, "exclude_hubs": excludeHubs}, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if len(out) == 0 {
		fmt.Fprintln(os.Stderr, "no god nodes")
		return
	}
	for _, g := range out {
		fmt.Printf("%s  degree %d  %s:%d\n", g.Name, g.Degree, g.Path, g.SymbolID)
	}
}

// runExplainCommunity implements `skillgrid explain-community ID` (CLI parity
// for code_explain_community).
func runExplainCommunity(version string, args []string) {
	fs := flag.NewFlagSet("explain-community", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid explain-community ID [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: explain-community requires exactly one ID argument")
		os.Exit(2)
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: explain-community id must be an integer, got %q\n", fs.Arg(0))
		os.Exit(2)
	}
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	out, err := svc.CodeExplainCommunity(ctx, projectID, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if !out.Found {
		fmt.Fprintf(os.Stderr, "not found: %s\n", out.Reason)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	fmt.Printf("community %d: %s (%d symbols)\n", out.ID, out.Label, len(out.Members))
	for _, m := range out.Members {
		fmt.Printf("  %v (%v) %v:%v\n", m["name"], m["kind"], m["path"], m["start_line"])
	}
	if len(out.EntryPts) > 0 {
		fmt.Println("entry points:")
		for _, e := range out.EntryPts {
			fmt.Printf("  %s degree %d\n", e.Name, e.Degree)
		}
	}
}
