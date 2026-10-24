package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/devopstales/skillgrid/mnemonic/internal/embedder"
	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/hybrid"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/skills"
)

// memScopeFor normalizes and validates the CLI --scope flag. Blank (the
// default) is the pre-change behavior: scope=project, project store only.
// An invalid scope is a usage error (exit 2) that lists the valid values
// before any store is opened (G5).
func memScopeFor(raw string) (facts.Scope, bool) {
	s := facts.Scope(strings.TrimSpace(raw))
	if s == "" {
		return facts.ScopeProject, true
	}
	if s.Valid() {
		return s, true
	}
	fmt.Fprintf(os.Stderr, "error: --scope must be project, global, or all, got %q\n", raw)
	printMemoryUsage()
	return s, false
}

// memCategoryFor normalizes and validates the CLI --target flag (maps to the
// facts category). Blank is the default: category=memory. An invalid target
// is a usage error (exit 2) that lists the valid values.
func memCategoryFor(raw string) (string, bool) {
	if raw == "" {
		return facts.CategoryMemory, true
	}
	c, err := facts.CategoryForTarget(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: --target must be memory or user, got %q\n", raw)
		printMemoryUsage()
		return "", false
	}
	return c, true
}

// runMemory handles `skillgrid memory fact <add|search|forget|decay|list>` —
// the CLI parity layer for the Fact Memory MCP tools (change
// 2026-09-04-hermes-memory, TICKET-06). --scope routes the data store
// (project | global; all is read-only — the fused search path) and --target
// sets the fact category on writes (memory | user). It routes through the same
// service seam the MCP tools use (svc.Open + facts.Store) so outcomes match
// MCP. Table is the default output; --json emits the machine-readable form.
// Invalid groups/actions, scopes, targets, and missing session ids fail before
// any store is opened.
func runMemory(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printMemoryUsage()
		os.Exit(2)
	}
	group := args[0]
	if group == "global" {
		runMemoryGlobal(args[1:])
		return
	}
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
		dir        string
		project    string
		sessionID  string
		content    string
		query      string
		limit      int
		includeDel bool
		mode       string
		factIDStr  string
		asJSON     bool
		scopeStr   string
		targetStr  string
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
	fs.StringVar(&scopeStr, "scope", "", "store scope (add|search|forget|decay: project|global; search: all = fused search)")
	fs.StringVar(&targetStr, "target", "", "fact target for add (memory|user)")
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

	// Scope/target are validated before the store opens (usage errors, exit
	// 2): a bad flag must not open a store it cannot route to.
	scope, ok := memScopeFor(scopeStr)
	if !ok {
		os.Exit(2)
	}
	if cmd == "add" {
		if _, ok := memCategoryFor(targetStr); !ok {
			os.Exit(2)
		}
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
		if scope == facts.ScopeAll {
			fmt.Fprintln(os.Stderr, "error: --scope all is read-only (fused search); add needs scope project or global")
			printMemoryUsage()
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
	// Store handles by scope: the project store is always opened (the trail
	// store for C3, and the data store for scope=project); the global store
	// is opened additionally for scope=global (its data store) and scope=all
	// (fused search reads both).
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	var gh *service.ProjectHandle
	var gclean func()
	if scope == facts.ScopeGlobal || scope == facts.ScopeAll {
		gh, gclean, err = svc.OpenGlobal()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer gclean()
	}

	projStore := facts.New(h.Store().DB, projID)
	var globStore *facts.Store
	if scope == facts.ScopeGlobal || scope == facts.ScopeAll {
		globStore = facts.New(gh.Store().DB, service.GlobalProjectID)
	}
	st := projStore
	if scope == facts.ScopeGlobal {
		st = globStore
	}
	ctx := hCtx()
	switch cmd {
	case "add":
		category, _ := memCategoryFor(targetStr)
		id, err := st.AddWith(ctx, sessionID, content, category, projStore)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{
			"event": "fact_add", "fact_id": id, "project": projID, "scope": string(scope),
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "added fact %d\n", id)
		})
	case "search":
		if scope == facts.ScopeAll {
			// Fused search (the scope=all read path): both stores, hits
			// interleaved by each store's own rank, each labeled scope +
			// category; the trail lands in the project store only.
			f, err := facts.FusedSearch(ctx, sessionID, query, limit, includeDel, globStore, projStore)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			printMemoryOut(asJSON, map[string]any{"facts": f, "count": len(f)}, func(w *tabwriter.Writer) {
				writeFusedFactTable(w, f)
			})
			break
		}
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
		out, err := st.SearchWith(ctx, sessionID, query, limit, includeDel, projStore)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{"facts": out, "count": len(out)}, func(w *tabwriter.Writer) {
			writeFactTable(w, out)
		})
	case "forget":
		id, _ := strconv.ParseInt(factIDStr, 10, 64)
		if err := st.Forget(ctx, sessionID, id, projStore); err != nil {
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
		newScore, err := st.Decay(ctx, sessionID, id, projStore)
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
		if scope == facts.ScopeGlobal || scope == facts.ScopeAll {
			// The global (or fused) read: list live facts from the global
			// store (the list read has no trail store — it is trail-free).
			out, err := globStore.ListWith(ctx, limit, includeDel)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			printMemoryOut(asJSON, map[string]any{"facts": out, "count": len(out), "scope": string(scope)}, func(w *tabwriter.Writer) {
				writeFactTable(w, out)
			})
			break
		}
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

// runMemoryGlobal handles `skillgrid memory global <subcommand>` — the
// CLI surface for the cross-project fact store (change
// 2026-10-24-global-llm-wiki, TICKET-05: prefetch; TICKET-08: the
// pending/approve/reject write-approval queue subcommands).
func runMemoryGlobal(args []string) {
	if len(args) == 0 {
		printMemoryGlobalUsage()
		os.Exit(2)
	}
	cmd := args[0]
	switch cmd {
	case "prefetch":
		runMemoryGlobalPrefetch(args[1:])
	case "pending", "approve", "reject":
		runMemoryGlobalPending(args, cmd)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown memory global command %q\n", cmd)
		printMemoryGlobalUsage()
		os.Exit(2)
	}
}

// runMemoryGlobalPrefetch renders the frozen snapshot of the global store.
func runMemoryGlobalPrefetch(args []string) {
	fs := flag.NewFlagSet("memory global prefetch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir       string
		sessionID string
		asJSON    bool
	)
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.String("project", "", "project id (accepted for flag parity; the snapshot reads the __global__ store)")
	fs.StringVar(&sessionID, "session-id", envOr("SKILLGRID_MEMORY_SESSION_ID", ""), "session id for the frozen-snapshot cache")
	fs.BoolVar(&asJSON, "json", false, "print JSON instead of the raw block")
	if err := fs.Parse(reorderMemArgs(args)); err != nil {
		os.Exit(2)
	}

	// Fail-open (G7): the consumer (plugin, system-prompt injection) never
	// sees an error from this seam — a dead global store is reported as the
	// empty string: exit 0, stdout and stderr silent.
	svc, _ := openMemService(dir, "")
	gh, cleanup, err := svc.OpenGlobal()
	if err != nil {
		os.Exit(0)
	}
	defer cleanup()

	st := facts.New(gh.Store().DB, service.GlobalProjectID)
	snap := facts.NewSnapshot(st)
	out, err := snap.Request(hCtx(), sessionID)
	if err != nil {
		os.Exit(0)
	}
	if asJSON {
		printJSON(map[string]any{
			"snapshot":   out,
			"session_id": sessionID,
			"project":    service.GlobalProjectID,
		})
		return
	}
	if out != "" {
		fmt.Println(out)
	}
}

// runMemoryGlobalPending handles the write-approval queue subcommands
// (TICKET-08): `pending` lists the staged rows in both stores; `approve all`
// applies every staged row into the facts table of its own store; `reject
// all` discards them. The queue spans the global AND the project store (a
// staged row lives in the store matching its scope), so the CLI opens both
// and works through the same Distiller accessor the MCP pending tools use.
func runMemoryGlobalPending(args []string, cmd string) {
	fs := flag.NewFlagSet("memory global "+cmd, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir     string
		project string
		asJSON  bool
	)
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&project, "project", "", "project id (defaults to CWD-resolved)")
	fs.BoolVar(&asJSON, "json", false, "print JSON instead of a table")
	if err := fs.Parse(reorderMemArgs(args[1:])); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if cmd == "approve" || cmd == "reject" {
		if len(pos) != 1 || pos[0] != "all" {
			fmt.Fprintf(os.Stderr, "error: memory global %s requires the argument 'all'\n", cmd)
			printMemoryGlobalUsage()
			os.Exit(2)
		}
	}

	svc, projID := openMemService(dir, project)
	gh, gclean, err := svc.OpenGlobal()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	ph, pclean, err := svc.Open(projID)
	if err != nil {
		gclean()
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer func() { pclean(); gclean() }()
	d := facts.NewDistiller(
		facts.New(gh.Store().DB, service.GlobalProjectID),
		facts.New(ph.Store().DB, projID),
	)
	ctx := hCtx()

	switch cmd {
	case "pending":
		rows, err := d.PendingList(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printMemoryOut(asJSON, map[string]any{"pending_facts": rows, "count": len(rows)}, func(w *tabwriter.Writer) {
			writePendingTable(w, rows)
		})
	case "approve":
		rows, err := d.PendingList(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		var approved []int64
		for _, r := range rows {
			factID, err := d.ApprovePending(ctx, r.ID)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			approved = append(approved, factID)
		}
		printMemoryOut(asJSON, map[string]any{
			"event": "fact_pending_approved_all", "approved": approved, "count": len(approved),
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "approved %d pending fact(s)\n", len(approved))
		})
	case "reject":
		rows, err := d.PendingList(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		var rejected []int64
		for _, r := range rows {
			if err := d.RejectPending(ctx, r.ID); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			rejected = append(rejected, r.ID)
		}
		printMemoryOut(asJSON, map[string]any{
			"event": "fact_pending_rejected_all", "rejected": rejected, "count": len(rejected),
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "rejected %d pending fact(s)\n", len(rejected))
		})
	}
}

// printMemoryGlobalUsage prints the `memory global` usage (stderr).
func printMemoryGlobalUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid memory global <prefetch|pending|approve all|reject all> [flags]

  prefetch [--session-id ID] [--json]
                                     render the frozen snapshot of the
                                     __global__ store (GLOBAL MEMORY +
                                     USER PROFILE sections, § separators,
                                     usage headers, importance DESC)
                                     and print it to stdout
  pending [--json]
                                     list staged (write-approval queue)
                                     fact writes from BOTH the __global__
                                     and the project stores
  approve all [--json]
                                     apply every staged fact into the
                                     facts table of its own store and
                                     empty the queue
  reject all [--json]
                                     discard every staged fact (it never
                                     reaches facts)

Flags:
  --project ID       project bucket (pending/approve/reject; defaults to CWD-resolved)
  --dir DATA_DIR     mnemonic data directory
  --session-id ID    session id for the frozen-snapshot cache (also SKILLGRID_MEMORY_SESSION_ID)
  --json             print the machine-readable JSON form instead of the raw block
`)
}

// writeFusedFactTable renders the scope=all fused search result as a table
// (id, scope, category, content) — the scope + category labels are the
// acceptance contract: the agent must be able to tell the two disjoint
// stores apart from the result alone.
func writeFusedFactTable(w *tabwriter.Writer, out []facts.FusedHit) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no facts")
		return
	}
	fmt.Fprintln(w, "ID\tSCOPE\tCATEGORY\tCONTENT")
	for _, f := range out {
		content := strings.ReplaceAll(f.Content, "\n", " ")
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", f.ID, f.Scope, f.Category, content)
	}
}

// writePendingTable renders the pending write-approval queue as a table
// (id, scope, target, content).
func writePendingTable(w *tabwriter.Writer, out []facts.PendingFact) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no pending facts")
		return
	}
	fmt.Fprintln(w, "ID\tSCOPE\tTARGET\tCONTENT")
	for _, p := range out {
		content := strings.ReplaceAll(p.Content, "\n", " ")
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.ID, p.Scope, p.Target, content)
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

  add <content> [--scope project|global] [--target memory|user] [--session-id ID] [--json]
                                    create a durable fact; prints the new fact id
                                    (scope=global lands the fact in the
                                    __global__ store, trail in the project store)
  search <query> [--scope project|global|all] [--limit N] [--mode fts|hybrid] [--include-deleted] [--session-id ID] [--json]
                                    lexical FTS search over facts
                                    (scope=all = fused search across both stores,
                                    hits labeled scope+category)
  forget <id> [--scope project|global] [--session-id ID] [--json]
                                    soft-delete a fact
  decay <id> [--scope project|global] [--session-id ID] [--json]
                                    apply the 014 AKL importance decay; prints the new score
  list [--scope project|global] [--limit N] [--include-deleted] [--json]
                                    list live facts, newest first

Flags:
  --project ID       project bucket (defaults to CWD-resolved)
  --scope SCOPE      store scope: project (default), global, or all (search only)
  --target TARGET    fact target for add: memory (default) or user
  --dir DATA_DIR     mnemonic data directory
  --session-id ID    session for the trail event (also SKILLGRID_MEMORY_SESSION_ID)
  --json             print the machine-readable JSON form instead of a table
`)
}
