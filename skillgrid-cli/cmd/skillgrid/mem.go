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
		exportFile string
		skipEmb  bool
		minConf  float64
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
	fs.StringVar(&exportFile, "file", "", "export: write the bundle to this file (default: stdout)")
	fs.BoolVar(&skipEmb, "skip-embeddings", false, "export: omit embedding data for a smaller payload")
	fs.Float64Var(&minConf, "min-confidence", 0.0, "relations: minimum edge confidence (0.0..1.0; 0.0 = all)")
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
 	case "relations":
 		runMemRelations(svc, projID, pos, minConf)
 	case "provenance":
 		runMemProvenance(svc, projID, pos)
	case "expire":
		runMemExpire(svc, dataDir)
	case "export":
		runMemExport(svc, projID, exportFile, skipEmb)
	case "distill":
		runMemDistill(svc, projID, pos)
	case "help", "-h", "--help":
		printMemUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown mem command %q\n", cmd)
		printMemUsage()
		os.Exit(2)
	}
}

func printMemUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid mem <layers|governance|share|search|context|timeline|graph|relations|provenance|expire|export|distill> [args]

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
  relations <observation_id> [--min-confidence 0.5]
                                      typed @relation edges (outgoing + incoming)
                                      between observations, with relation type,
                                      confidence, and direction
  provenance <observation_id>          curation chain of an observation
                                       (session_id, curate_command, source_files,
                                       llm_reasoning); clear message when unset
  expire                          retire expired observations across all projects
                                   (TTL sweep; best-effort, exit 0 on missing stores)
   export [--file out.json] [--skip-embeddings]
                                    portable COGX JSON export of observations +
                                    graph edges + embeddings (stdout by default)
   distill status                   show the project's distillation lock
                                    (project_id, locked_at, locked_by; or
                                    "no active locks" when free)

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

// runMemExport is the CLI for the portable COGX export (014 step 11,
// `mem export`): it streams the project bundle (observations + graph edges +
// symbol embeddings) as JSON to stdout, or to --file when set.
// --skip-embeddings drops the embedding vectors for a smaller payload.
// The bundle is written through a json.Encoder on the target writer, so the
// output is streamed incrementally rather than marshaled into one large
// []byte (the encoding step is the streaming boundary; the bundle itself is
// still assembled row-by-row, one record per store row).
func runMemExport(svc *service.Service, projID, file string, skipEmbeddings bool) {
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	var w *os.File
	if file != "" {
		w, err = os.Create(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: create %s: %v\n", file, err)
			os.Exit(1)
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	bundle, err := h.Memory().ExportProject(hCtx())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if skipEmbeddings {
		bundle.Embeddings = bundle.Embeddings[:0]
		for i := range bundle.Observations {
			bundle.Observations[i].Embeddings = nil
		}
	}
	if err := memory.WriteBundle(hCtx(), bundle, w); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if file != "" {
		fmt.Fprintf(os.Stderr, "exported %d observations, %d edges, %d embeddings to %s\n",
			len(bundle.Observations), len(bundle.GraphEdges), len(bundle.Embeddings), file)
	}
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

// runMemRelations is the CLI for typed @relation edges (014 step 14,
// `mem relations <observation_id>`): it lists every observation_relations
// edge touching the given observation — both outgoing (the obs is the source)
// and incoming (the obs is the target) — with the relation type, confidence,
// and a direction label. --min-confidence applies a floor (default 0.0 = all).
// A missing observation id is a clear error; an observation with no relations
// prints an empty list (graceful).
func runMemRelations(svc *service.Service, projID string, pos []string, minConf float64) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: mem relations requires an observation id")
		os.Exit(2)
	}
	id, err := strconv.ParseInt(pos[0], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: invalid observation id")
		os.Exit(2)
	}
	if minConf < 0.0 || minConf > 1.0 {
		fmt.Fprintf(os.Stderr, "error: --min-confidence %v out of range (0.0..1.0)\n", minConf)
		os.Exit(2)
	}
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	ctx := hCtx()

	// A missing observation id (no live observation with this id in the
	// project) is a clear error, matching the other mem subcommands.
	if _, ok := h.Memory().ObsTitle(ctx, id); !ok {
		fmt.Fprintf(os.Stderr, "error: observation %d not found in this project\n", id)
		os.Exit(1)
	}

	rels, err := h.Memory().GetRelations(ctx, id, minConf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	type relationOut struct {
		Direction    string  `json:"direction"`
		RelationType string  `json:"relation_type"`
		Confidence   float64 `json:"confidence"`
		OtherID      int64   `json:"other_id"`
		OtherTitle   string  `json:"other_title,omitempty"`
	}
	// Ensure a nil slice marshals as [] (empty list) rather than null.
	outs := make([]relationOut, 0, len(rels))
	for _, r := range rels {
		dir := "outgoing"
		other := r.TargetID
		if r.SourceID != id {
			dir = "incoming"
			other = r.SourceID
		}
		o := relationOut{
			Direction:    dir,
			RelationType: r.RelationType,
			Confidence:   r.Confidence,
			OtherID:      other,
		}
		if title, ok := h.Memory().ObsTitle(ctx, other); ok {
			o.OtherTitle = title
		}
		outs = append(outs, o)
	}
	out := map[string]any{
		"observation_id": id,
		"min_confidence": minConf,
		"count":          len(outs),
		"relations":      outs,
	}
	printJSON(out)
}

// runMemProvenance is the CLI for the curation chain (014 step 15,
// `mem provenance <observation_id>`): it prints the provenance JSON stored on
// the observation — the session, command, source files, and LLM reasoning
// that produced it — as a readable structured chain (pretty-printed JSON with
// the four snake_case labels). A missing observation id is a clear error; an
// observation that never had provenance prints a clear "no provenance"
// message (exit 0).
func runMemProvenance(svc *service.Service, projID string, pos []string) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: mem provenance requires an observation id")
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
	ctx := hCtx()

	// A missing observation id (no live observation with this id in the
	// project) is a clear error, matching the other mem subcommands.
	if _, ok := h.Memory().ObsTitle(ctx, id); !ok {
		fmt.Fprintf(os.Stderr, "error: observation %d not found in this project\n", id)
		os.Exit(1)
	}
	p, err := h.Memory().Provenance(ctx, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if p == nil {
		printJSON(map[string]any{
			"observation_id": id,
			"provenance":     "no provenance recorded for this observation",
		})
		return
	}
	printJSON(map[string]any{
		"observation_id": id,
		"provenance":     p,
	})
}

// runMemDistill is the CLI for the distillation lock status (014 step 17.4,
// `mem distill status`): it shows the project's per-project distillation lock —
// the project_id, locked_at, and locked_by when a lock is held — or "no active
// locks" when the project is not locked. It is read-only: it never acquires or
// releases a lock, so it is safe to run while a distillation is in flight.
func runMemDistill(svc *service.Service, projID string, pos []string) {
	sub := ""
	if len(pos) >= 1 {
		sub = pos[0]
	}
	switch sub {
	case "status":
		h, cleanup, err := svc.Open(projID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer cleanup()
		locks := memory.NewDistillLockService(h.Memory().DB())
		status, err := locks.DistillLockStatus(hCtx(), projID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if !status.Held && status.LockedAt == "" {
			printJSON(map[string]any{
				"project": projID,
				"status":  "no active locks",
			})
			return
		}
		out := map[string]any{
			"project": projID,
			"status":  map[string]any{
				"project_id": status.ProjectID,
				"locked_at":  status.LockedAt,
				"locked_by":  status.LockedBy,
				"held":       status.Held,
			},
		}
		printJSON(out)
	case "", "help", "-h", "--help":
		fmt.Fprint(os.Stderr, `usage: skillgrid mem distill <status>

  status    show the project's distillation lock (held: project_id, locked_at,
            locked_by; or "no active locks" when free)
`)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown distill subcommand %q\n", sub)
		os.Exit(2)
	}
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
