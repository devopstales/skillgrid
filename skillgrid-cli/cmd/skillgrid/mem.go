package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runMem handles `skillgrid mem <subcommand>`. It is the CLI parity layer for
// the memory tools (change 013, step 03): `mem layers|governance|share`
// (01/02 tools with no prior CLI) plus `mem search|context|timeline` so the
// budgeted read paths are reachable from the command line. Each subcommand
// routes through the same service seams the MCP tools use.
func runMem(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printMemUsage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("mem", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir  string
		project  string
		limit    int
		items    int
		chars    int
		timeout  string
		target   string
		owner    string
		agent    string
		window   string
		visibility string
		grants   string
	)
	fs.StringVar(&dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&project, "project", "", "project id (defaults to CWD-resolved)")
	fs.IntVar(&limit, "limit", 0, "max results (read subcommands)")
	fs.IntVar(&items, "item", 0, "budget: item-count cap (0 = default)")
	fs.IntVar(&chars, "char", 0, "budget: per-snippet char cap (0 = default)")
	fs.StringVar(&timeout, "timeout", "", "budget: context timeout, e.g. 3s (0 = default)")
	fs.StringVar(&target, "target", "", "layers: session_id or topic_key")
	fs.StringVar(&owner, "reader-owner", "", "read: reader identity (per-owner visibility)")
	fs.StringVar(&agent, "reader-agent", "", "read: reader agent id (ACL)")
	fs.StringVar(&window, "window", "", "timeline: time window each side (e.g. 1h)")
	fs.StringVar(&visibility, "target-visibility", "", "share: team|restricted|agent")
	fs.StringVar(&grants, "grants", "", "share: comma-separated grantee list")
	var searchMode string
	fs.StringVar(&searchMode, "mode", "", "search: FTS match mode (trigram|prefix|phrase|all; default = phrase OR)")
	if err := fs.Parse(reorderMemArgs(rest)); err != nil {
		os.Exit(2)
	}

	svc, projID := openMemService(dataDir, project)
	pos := fs.Args()

	switch cmd {
	case "layers":
		runMemLayers(svc, projID, pos, fs.Lookup("target").Value.String())
	case "governance":
		runMemGovernance(svc, projID, pos)
	case "share":
		runMemShare(svc, projID, pos, visibility, grants)
	case "search":
		runMemSearch(svc, projID, dataDir, pos, owner, agent, limit, items, chars, timeout, searchMode)
	case "context":
		runMemContext(svc, projID, limit, items, chars, timeout)
	case "timeline":
		runMemTimeline(svc, projID, pos, window, limit, items, chars, timeout)
	case "graph":
		runMemGraph(svc, projID, limit)
	case "expire":
		runMemExpire(svc, dataDir)
	case "help", "-h", "--help":
		printMemUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown mem command %q\n", cmd)
		printMemUsage()
		os.Exit(2)
	}
}

func printMemUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid mem <layers|governance|share|search|context|timeline|graph|expire> [args]

  layers <session_id|topic_key>   inspect the L0→L1→L2→L3 chain (mem_layers)
  governance <id>                 governed-asset view (mem_governance)
  share <id> --target-visibility team|restricted|agent [--grants a,b]
                                   widen visibility (mem_share)
  search <query> [--mode trigram|prefix|phrase|all] [--limit N] [--reader-owner X]
                  [--item N] [--char N] [--timeout 3s]
                                    budgeted FTS search (mem_search; default mode = phrase OR)
  context [--limit N] [--item N] [--char N] [--timeout 3s]
                                   recent session summaries (mem_context)
  timeline <id> [--window 1h] [--limit N] [--item N] [--char N] [--timeout 3s]
                                    chronological context (mem_timeline)
  graph [--limit N]                 temporal status of codeindex graph edges
                                    (valid_from, valid_to, active/expired/pending)
  expire                          retire expired observations across all projects
                                   (TTL sweep; best-effort, exit 0 on missing stores)

Flags:
  --project ID    project bucket (defaults to CWD-resolved)
  --dir DATA_DIR  mnemonic data directory
`)
}

func openMemService(dataDir, project string) (*service.Service, string) {
	dd := dataDir
	if dd == "" {
		d, err := service.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		dd = d
	}
	svc := service.New(dd)
	proj := project
	if proj == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		pid, err := svc.ResolveProject(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: resolve project: %v\n", err)
			os.Exit(1)
		}
		proj = pid
	}
	return svc, proj
}

// memBudgetOpts builds the tunable read budget from the CLI flags (0 = default).
func memBudgetOpts(items, chars int, timeout string) memoryBudgetCfg {
	cfg := memoryBudgetCfg{Items: items, Chars: chars}
	if timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: bad --timeout %q: %v\n", timeout, err)
			os.Exit(2)
		}
		cfg.TimeoutNs = int64(d)
	}
	return cfg
}

func runMemLayers(svc *service.Service, projID string, pos []string, target string) {
	t := target
	if t == "" && len(pos) >= 1 {
		t = pos[0]
	}
	if t == "" {
		fmt.Fprintln(os.Stderr, "error: mem layers requires a session_id or topic_key")
		os.Exit(2)
	}
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	var chain layer.Chain
	var lerr error
	if looksLikeSession(t) {
		chain, lerr = layer.Inspect(hCtx(), h.Memory(), t)
	} else {
		chain, lerr = layer.InspectByTopic(hCtx(), h.Memory(), t)
	}
	if lerr != nil {
		fmt.Fprintln(os.Stderr, "error:", lerr)
		os.Exit(1)
	}
	printJSON(chain)
}

func runMemGovernance(svc *service.Service, projID string, pos []string) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: mem governance requires an id")
		os.Exit(2)
	}
	id, err := strconv.ParseInt(pos[0], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: invalid observation id")
		os.Exit(2)
	}
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	g, err := h.Memory().Governance(hCtx(), id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printJSON(g)
}

func runMemShare(svc *service.Service, projID string, pos []string, visibility, grants string) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: mem share requires an id")
		os.Exit(2)
	}
	id, err := strconv.ParseInt(pos[0], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: invalid observation id")
		os.Exit(2)
	}
	if visibility == "" {
		fmt.Fprintln(os.Stderr, "error: mem share requires --target-visibility team|restricted|agent")
		os.Exit(2)
	}
	var gs []string
	for _, g := range strings.Split(grants, ",") {
		if t := strings.TrimSpace(g); t != "" {
			gs = append(gs, t)
		}
	}
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	if err := h.Memory().Share(hCtx(), id, memoryShareInput{Visibility: visibility, Grants: gs}); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printJSON(map[string]any{"id": id, "shared": true, "visibility": visibility})
}

func runMemSearch(svc *service.Service, projID, dataDir string, pos []string, owner, agent string, limit, items, chars int, timeout string, searchMode string) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: mem search requires a query")
		os.Exit(2)
	}
	// --mode (014 step 02): FTS match mode for the fact leg. "phrase" is the
	// alias for the default; trigram/prefix reshape the FTS query. Invalid
	// values are rejected here, before any store is opened.
	ftsMode := memory.NormalizeMatchMode(searchMode)
	if err := memory.ValidateMatchMode(ftsMode); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	query := strings.Join(pos, " ")
	if limit <= 0 {
		limit = 20
	}
	// The CLI `mem search` is the REAL production entry point for the layered
	// read path (change 013, step 03): a specific fact routes through
	// BudgetedRetrievalAsRoot (L2/L3-first + L1/L0 RRF-fallback), owner-gated
	// by --reader-owner so the step-01 per-owner visibility gate holds, and
	// uniformly budgeted (item cap + char budget + context timeout). The
	// --item/--char/--timeout flags become a per-project budget override that
	// takes precedence over config; the full content stays fetchable via
	// mem_get_observation (the only full-content path) using each hit's id.
	svc.SetBudgetOverride(projID, memoryBudget(memBudgetOpts(items, chars, timeout)))
	if owner == "" {
		owner = agent
	}
	res, err := svc.BudgetedRetrievalAsRootFTS(hCtx(), projID, dataDir, owner, "fact", query, ftsMode, limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out := map[string]any{
		"project":      projID,
		"observations": res.Hits,
		"count":        len(res.Hits),
	}
	if res.Truncated {
		out["truncated"] = true
		out["truncation_reason"] = res.Reason
	}
	printJSON(out)
}

func runMemContext(svc *service.Service, projID string, limit, items, chars int, timeout string) {
	if limit <= 0 {
		limit = 5
	}
	cfg := memBudgetOpts(items, chars, timeout)
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	// Budget (013 step 03): the context read honors --item/--char/--timeout.
	h.Memory().SetBudget(memoryBudget(cfg))
	b := h.Memory().Budget()
	bctx := b.Bound(hCtx())
	sessions, err := h.Memory().RecentContext(bctx, limit)
	if err != nil && budgetDeadlineLapsed(bctx) {
		sessions = nil
	} else if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	budgeted, truncated, reason := budgetSessionsCLI(b, sessions)
	if budgetDeadlineLapsed(bctx) {
		truncated = true
		reason = "timeout"
	}
	out := map[string]any{"sessions": budgeted}
	if truncated {
		out["truncated"] = true
		out["truncation_reason"] = reason
	}
	printJSON(out)
}

func runMemTimeline(svc *service.Service, projID string, pos []string, window string, limit, items, chars int, timeout string) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: mem timeline requires an id")
		os.Exit(2)
	}
	id, err := strconv.ParseInt(pos[0], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: invalid observation id")
		os.Exit(2)
	}
	cfg := memBudgetOpts(items, chars, timeout)
	w := parseMemWindow(window)
	if limit <= 0 {
		limit = 5
	}
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	// Budget (013 step 03): the timeline read honors --item/--char/--timeout.
	h.Memory().SetBudget(memoryBudget(cfg))
	b := h.Memory().Budget()
	bctx := b.Bound(hCtx())
	tl, tErr := h.Memory().Timeline(bctx, id, w, limit)
	if tErr != nil && budgetDeadlineLapsed(bctx) {
		tl = memory.Timeline{}
	} else if tErr != nil {
		fmt.Fprintln(os.Stderr, "error:", tErr)
		os.Exit(1)
	}
	// Item cap is the per-direction `limit` (Timeline already applies it); the
	// char budget truncates each in-list snippet (explicit "N chars omitted").
	truncated := false
	reason := ""
	maxChars := b.Config().Chars
	trunc := func(e []memory.TimelineEntry) {
		for i := range e {
			if len(e[i].Content) > maxChars {
				e[i].Content = memory.TruncateWithMarker(e[i].Content, maxChars)
				truncated = true
				if reason == "" {
					reason = "char-budget"
				}
			}
		}
	}
	trunc(tl.Before)
	trunc(tl.After)
	if budgetDeadlineLapsed(bctx) {
		truncated = true
		reason = "timeout"
	}
	out := map[string]any{"anchor_id": id, "before": tl.Before, "after": tl.After}
	if truncated {
		out["truncated"] = true
		out["truncation_reason"] = reason
	}
	printJSON(out)
}

// runMemExpire is the CLI for the TTL sweep (014 step 04, `mem expire`): it
// retires expired observations across every project under the data dir and
// prints the per-project retirement counts. It is best-effort — RunTTLExpiry
// skips missing/corrupt stores — and always exits 0 on a successful sweep, so
// a scheduled cron job never fails over an uncreated store. It ignores
// --project (it is a cross-project sweep).
func runMemExpire(svc *service.Service, dataDir string) {
	_ = dataDir // the service already owns the data dir it was opened with
	counts, err := svc.RunTTLExpiry(hCtx())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out := map[string]any{"retired": counts}
	total := 0
	for _, n := range counts {
		total += n
	}
	out["total"] = total
	printJSON(out)
}

// runMemGraph is the CLI for the temporal graph view (014 step 10, `mem graph`):
// it lists the project's codeindex graph edges with their temporal status
// (valid_from, valid_to, and a computed active/expired/pending label). It reads
// the unfiltered history set (QueryEdgesWithHistory) so every edge is visible
// with its status — the current-state filter is the concern of graph traversal,
// not this diagnostic. It is read-only and never mutates the index.
func runMemGraph(svc *service.Service, projID string, limit int) {
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	edges, err := codeindex.QueryEdgesWithHistory(h.Store())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	// count reflects the FULL result set, computed before the --limit
	// truncation, so it always matches the sum of the per-status totals
	// (the truncated list is `edges`; review F3, 014 step 10).
	counts := map[string]int{"active": 0, "expired": 0, "pending": 0}
	for _, e := range edges {
		counts[e.Status]++
	}
	count := len(edges)
	if limit > 0 && len(edges) > limit {
		edges = edges[:limit]
	}
	out := map[string]any{
		"project": projID,
		"count":   count,
		"total":   counts,
		"edges":   edges,
	}
	printJSON(out)
}

func looksLikeSession(s string) bool {
	// A session id is a UUID (36 chars with dashes) or a bare token with no
	// spaces; a topic_key typically contains '/'.
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, " ") || strings.Contains(s, "/") {
		return false
	}
	return len(s) >= 8
}

func reorderMemArgs(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-" || strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

func printJSON(v any) {
	enc := newJSONEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
