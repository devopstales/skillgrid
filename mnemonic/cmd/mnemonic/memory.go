package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills"
)

// runMemory handles `skillgrid memory fact <add|search|forget|decay|list>` —
// the CLI parity layer for the Fact Memory MCP tools (change
// 2026-09-04-hermes-memory, TICKET-06). It routes through the same service
// seam the MCP tools use (svc.Open + facts.Store) so outcomes match MCP.
// Table is the default output; --json emits the machine-readable form. Invalid
// groups/actions and missing session ids fail before any store is opened.
func runMemory(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printMemoryUsage()
		os.Exit(2)
	}
	group := args[0]
	if group != "fact" {
		fmt.Fprintf(os.Stderr, "error: unknown memory group %q\n", group)
		printMemoryUsage()
		os.Exit(2)
	}
	if len(args) < 2 {
		printMemoryUsage()
		os.Exit(2)
	}
	cmd := args[1]

	fs := flag.NewFlagSet("memory fact", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir         string
		project     string
		sessionID   string
		content     string
		query       string
		limit       int
		includeDel  bool
		mode        string
		factIDStr   string
		asJSON      bool
	)
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&project, "project", "", "project id (defaults to CWD-resolved)")
	fs.StringVar(&sessionID, "session-id", envOr("SKILLGRID_MEMORY_SESSION_ID", ""), "session id for the trail event")
	fs.StringVar(&content, "content", "", "fact content (add)")
	fs.StringVar(&query, "query", "", "search terms (search)")
	fs.IntVar(&limit, "limit", 20, "max results (list|search)")
	fs.BoolVar(&includeDel, "include-deleted", false, "list|search: include soft-deleted facts")
	fs.StringVar(&mode, "mode", "", "search: FTS match mode (fts|hybrid; hybrid degrades to fts with no embedder)")
	fs.StringVar(&factIDStr, "fact-id", "", "target fact id (forget|decay)")
	fs.BoolVar(&asJSON, "json", false, "print JSON instead of a table")
	if err := fs.Parse(reorderMemArgs(args[2:])); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	// Fill flags from positionals: add <content>, search <query...>,
	// forget|decay <id>.
	if content == "" && cmd == "add" && len(pos) > 0 {
		content = strings.Join(pos, " ")
	}
	if query == "" && cmd == "search" && len(pos) > 0 {
		query = strings.Join(pos, " ")
	}
	if factIDStr == "" && (cmd == "forget" || cmd == "decay") && len(pos) > 0 {
		factIDStr = pos[0]
	}

	// Session-required actions fail before the store opens (the trail event FK
	// would otherwise reject the insert).
	sessionNeeded := cmd == "add" || cmd == "search" || cmd == "forget" || cmd == "decay"
	if sessionNeeded && sessionID == "" {
		fmt.Fprintln(os.Stderr, "error: this memory action requires a --session-id (or SKILLGRID_MEMORY_SESSION_ID)")
		os.Exit(2)
	}

	switch cmd {
	case "add":
		if content == "" {
			fmt.Fprintln(os.Stderr, "error: memory fact add requires --content or a positional content")
			os.Exit(2)
		}
	case "search":
		if query == "" {
			fmt.Fprintln(os.Stderr, "error: memory fact search requires --query or a positional query")
			os.Exit(2)
		}
		if mode != "" && !searchModeValid(mode) {
			fmt.Fprintf(os.Stderr, "error: --mode must be fts or hybrid, got %q\n", mode)
			os.Exit(2)
		}
	case "forget", "decay":
		if factIDStr == "" {
			fmt.Fprintf(os.Stderr, "error: memory fact %s requires a numeric --fact-id or positional id\n", cmd)
			os.Exit(2)
		}
		if _, err := strconv.ParseInt(factIDStr, 10, 64); err != nil {
			fmt.Fprintf(os.Stderr, "error: memory fact %s requires a numeric --fact-id, got %q\n", cmd, factIDStr)
			os.Exit(2)
		}
	case "list":
		// read-only; honors --limit/--include-deleted.
	default:
		fmt.Fprintf(os.Stderr, "error: unknown memory fact command %q\n", cmd)
		printMemoryUsage()
		os.Exit(2)
	}

	svc, projID := openMemService(dir, project)
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	st := facts.New(h.Store().DB, projID)
	ctx := hCtx()
	switch cmd {
	case "add":
		id, err := st.Add(ctx, sessionID, content)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{
			"event": "fact_add", "fact_id": id, "project": projID,
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "added fact %d\n", id)
		})
	case "search":
		if mode == "hybrid" {
			skillSt := skills.New(h.Store().DB, h.Root(), projID)
			res, err := hybrid.SearchMemory(ctx, st, skillSt, query, limit, hybrid.MemoryOptions{
				Embedder:       embedder.Default(),
				IncludeDeleted: includeDel,
				ReadingSession: sessionID,
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			hybridNote(asJSON, res.Legs)
			printMemoryOut(asJSON, map[string]any{"facts": res.Facts, "count": len(res.Facts), "legs": res.Legs}, func(w *tabwriter.Writer) {
				writeHybridFactTable(w, res.Facts)
			})
			break
		}
		out, err := st.SearchWith(ctx, sessionID, query, limit, includeDel)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{"facts": out, "count": len(out)}, func(w *tabwriter.Writer) {
			writeFactTable(w, out)
		})
	case "forget":
		id, _ := strconv.ParseInt(factIDStr, 10, 64)
		if err := st.Forget(ctx, sessionID, id); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{
			"event": "fact_forget", "fact_id": id,
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "forgotten fact %d\n", id)
		})
	case "decay":
		id, _ := strconv.ParseInt(factIDStr, 10, 64)
		newScore, err := st.Decay(ctx, sessionID, id)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{
			"event": "fact_decay", "fact_id": id, "new_score": newScore,
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "decayed fact %d to score %f\n", id, newScore)
		})
	case "list":
		out, err := st.ListWith(ctx, limit, includeDel)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{"facts": out, "count": len(out)}, func(w *tabwriter.Writer) {
			writeFactTable(w, out)
		})
	}
}

// writeFactTable renders the fact list as a human-readable table (id, content).
func writeFactTable(w *tabwriter.Writer, out []facts.Fact) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no facts")
		return
	}
	fmt.Fprintln(w, "ID\tCONTENT")
	for _, f := range out {
		content := strings.ReplaceAll(f.Content, "\n", " ")
		fmt.Fprintf(w, "%d\t%s\n", f.ID, content)
	}
}

// printMemoryOut emits either the JSON object or the table produced by tableFn.
func printMemoryOut(asJSON bool, obj map[string]any, tableFn func(w *tabwriter.Writer)) {
	if asJSON {
		printJSON(obj)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	tableFn(w)
	w.Flush()
}

// writeHybridFactTable renders the hybrid RRF-fused fact list as a table
// (id, score, provenance, content).
func writeHybridFactTable(w *tabwriter.Writer, out []hybrid.MemoryFact) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no facts")
		return
	}
	fmt.Fprintln(w, "ID\tSCORE\tSOURCE\tCONTENT")
	for _, f := range out {
		content := strings.ReplaceAll(f.Content, "\n", " ")
		src := "fts"
		if f.Provenance.Semantic {
			src = "semantic"
		}
		fmt.Fprintf(w, "%d\t%.4f\t%s\t%s\n", f.ID, f.Score, src, content)
	}
}

// hybridNote prints the "no embedder available" note (to stderr for tables,
// folded into the JSON object for --json) when hybrid mode degraded to
// BM25-only — i.e. the legs do not include "semantic".
func hybridNote(asJSON bool, legs []string) {
	semantic := false
	for _, l := range legs {
		if l == "semantic" {
			semantic = true
			break
		}
	}
	if semantic {
		return
	}
	if asJSON {
		return
	}
	fmt.Fprintln(os.Stderr, "hybrid mode: no embedder available, using FTS-only")
}

func printMemoryUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid memory fact <add|search|forget|decay|list> [args]

  add <content> [--session-id ID] [--json]
                                    create a durable fact; prints the new fact id
  search <query> [--limit N] [--mode fts|hybrid] [--include-deleted] [--session-id ID] [--json]
                                    lexical FTS search over facts
  forget <id> [--session-id ID] [--json]
                                    soft-delete a fact
  decay <id> [--session-id ID] [--json]
                                    apply the 014 AKL importance decay; prints the new score
  list [--limit N] [--include-deleted] [--json]
                                    list live facts, newest first

Flags:
  --project ID       project bucket (defaults to CWD-resolved)
  --dir DATA_DIR     mnemonic data directory
  --session-id ID    session for the trail event (also SKILLGRID_MEMORY_SESSION_ID)
  --json             print the machine-readable JSON form instead of a table
`)
}
