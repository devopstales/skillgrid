// mem.go — the mem command group (spec 005-transport-cli §2): each
// subcommand parses its flags, calls the shared handler, and prints the
// result (human-readable by default, the envelope JSON with --json —
// the same marshal the MCP tools use, the K6 seam).
//
// The subcommand list is the spec's literal table. Each is a thin
// adapter: flag parse → handler call → envelope. The project selection
// (spec §3) resolves at call time: explicit --project > MNEMONIC_PROJECT
// > git toplevel > basename; --all fans the call out over every project
// the store knows.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// jsonFlag adds the shared --json flag to fs.
func jsonFlag(fs *flag.FlagSet) *bool {
	b := fs.Bool("json", false, "print the envelope JSON (machine output)")
	return b
}

// reorderArgs moves flag args (starting with --) before positionals so
// the Go flag package parses them (it stops at the first positional).
// The query positional stays last. This is the CLI's ergonomic: the user
// can type `mem search q --project p` and it works.
func reorderArgs(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			flags = append(flags, a)
			// Consume the value for value-carrying flags (not bools).
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !isBoolFlag(a) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positionals = append(positionals, a)
		}
	}
	return append(flags, positionals...)
}

// isBoolFlag reports whether a --flag is a bool (no value). The bool
// flags in the CLI are: --all, --json, --strict, --hard, --off,
// --include-history, --older-history, --resume, --isolated, --no-capture,
// --full, --fix, --grounded-only.
func isBoolFlag(flag string) bool {
	name := flag
	if eq := strings.IndexByte(name, '='); eq > 0 {
		name = name[:eq]
	}
	switch name {
	case "--all", "--json", "--strict", "--hard", "--off", "--include-history",
		"--older-history", "--resume", "--isolated", "--no-capture",
		"--full", "--fix", "--grounded-only":
		return true
	}
	return false
}

// runMem dispatches the mem subcommands.
func runMem(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("mem: a subcommand is required (save|search|get|lookup|context|review|timeline|pin|unpin|delete|session|stats|doctor)")
	}
	switch args[0] {
	case "save":
		return memSave(fl, cwd, args[1:])
	case "search":
		return memSearch(fl, cwd, args[1:])
	case "get":
		return memGet(fl, cwd, args[1:])
	case "lookup":
		return memLookup(fl, cwd, args[1:])
	case "context":
		return memContext(fl, cwd, args[1:])
	case "review":
		return memReview(fl, cwd, args[1:])
	case "timeline":
		return memTimeline(fl, cwd, args[1:])
	case "pin":
		return memPin(fl, cwd, args[1:], true)
	case "unpin":
		return memPin(fl, cwd, args[1:], false)
	case "delete":
		return memDelete(fl, cwd, args[1:])
	case "session":
		return memSession(fl, cwd, args[1:])
	case "stats":
		return memStats(fl, cwd, args[1:])
	case "doctor":
		return memDoctor(fl, cwd, args[1:])
	default:
		if strings.HasPrefix(args[0], "-") {
			return errExit("mem: unknown flag %q before a subcommand", args[0])
		}
		return errExit("mem: unknown subcommand %q", args[0])
	}
}

// errExit prints a usage error to stderr and returns exitUsage.
func errExit(format string, args ...any) int {
	fmt.Fprintf(outW(), "error: "+format+"\n", args...)
	return exitUsage
}

// rejectAll refuses --all on the read commands until the cross-project
// store lands (the per-project fan-out is TICKET-16, the task that owns
// the cross-project registry). Today --all would silently resolve to the
// current project (one project's rows) — a worse bug than a loud usage
// error. explicit + --all is the pinned project.ErrAllWithProject; a bare
// --all names the deferred fan-out and points at --project. Returns the
// usage message ("" to proceed).
func rejectAll(pf projectFlag) string {
	if !pf.all {
		return ""
	}
	if pf.project != "" {
		return project.ErrAllWithProject.Error()
	}
	return "mem: --all fan-out is not yet wired (the cross-project store is pending); use --project <name> to read one project"
}

// memSave runs `mem save`.
func memSave(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem save", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var session, typ, title, content, scope, topicKey, operationID, currentPrompt string
	var captureOn bool
	fs.StringVar(&session, "session", "", "session id")
	fs.StringVar(&typ, "type", "", "observation type (decision|architecture|bugfix|pattern|config|discovery|learning|preference|convention|session_summary)")
	fs.StringVar(&title, "title", "", "title")
	fs.StringVar(&content, "content", "", "content")
	fs.StringVar(&scope, "scope", "", "scope (project|personal|global)")
	fs.StringVar(&topicKey, "topic-key", "", "topic key (kebab-case)")
	fs.StringVar(&operationID, "operation-id", "", "idempotency operation id")
	fs.StringVar(&currentPrompt, "current-prompt", "", "the current turn's prompt")
	fs.BoolVar(&captureOn, "no-capture", false, "do not capture the current prompt")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if title == "" || typ == "" || content == "" {
		return errExit("mem save: --title, --type and --content are required")
	}
	// --no-capture is the tri-state off switch: absent → nil → the
	// pipeline's default-on (the amended spec row); explicit → false.
	var cp *bool
	if captureOn {
		f := false
		cp = &f
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	req := handler.SaveRequest{
		SessionID: session, Type: typ, Title: title, Content: content,
		Scope: scope, TopicKey: topicKey, OperationID: operationID,
		CurrentPrompt: currentPrompt, CapturePrompt: cp,
		Project: pf.project,
	}
	res, err := h.Save(context.Background(), req)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// memSearch runs `mem search <query>`.
func memSearch(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var typ, scope, matchMode string
	var limit int
	fs.StringVar(&typ, "type", "", "filter by type")
	fs.StringVar(&scope, "scope", "", "filter by scope")
	fs.StringVar(&matchMode, "match-mode", "", "all|any")
	fs.IntVar(&limit, "limit", 0, "max hits")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	query := strings.Join(fs.Args(), " ")
	if query == "" {
		return errExit("mem search: a query is required")
	}
	// --all is refused until the cross-project store lands (rejectAll):
	// an explicit project + --all is the pinned usage error, a bare --all
	// names the deferred fan-out.
	if msg := rejectAll(pf); msg != "" {
		return errExit("%s", msg)
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	f := handler.SearchFilters{Type: typ, Scope: scope, MatchMode: matchMode, Limit: limit, Project: pf.project}
	res, err := h.Search(context.Background(), query, f)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// memGet runs `mem get` (full content by id / sync_id).
func memGet(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem get", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var id int64
	var syncID string
	var includeHistory bool
	fs.Int64Var(&id, "id", 0, "local id")
	fs.StringVar(&syncID, "sync-id", "", "sync id (the agent's identity)")
	fs.BoolVar(&includeHistory, "include-history", false, "include the version history")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if id == 0 && syncID == "" && len(fs.Args()) > 0 {
		if n, err := strconv.ParseInt(fs.Args()[0], 10, 64); err == nil {
			id = n
		} else {
			syncID = fs.Args()[0]
		}
	}
	if id == 0 && syncID == "" {
		return errExit("mem get: --id or --sync-id (or a positional id) is required")
	}
	if msg := rejectAll(pf); msg != "" {
		return errExit("%s", msg)
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := h.Get(context.Background(), id, syncID, includeHistory, 0)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// memLookup runs `mem lookup` (the wiki-first fallback).
func memLookup(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem lookup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var id int64
	var syncID, topicKey, phrase, scope string
	fs.Int64Var(&id, "id", 0, "local id")
	fs.StringVar(&syncID, "sync-id", "", "sync id")
	fs.StringVar(&topicKey, "topic-key", "", "exact topic key")
	fs.StringVar(&phrase, "phrase", "", "distinctive phrase")
	fs.StringVar(&scope, "scope", "", "scope")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if id == 0 && syncID == "" && topicKey == "" && phrase == "" {
		return errExit("mem lookup: --id, --sync-id, --topic-key or --phrase is required")
	}
	if msg := rejectAll(pf); msg != "" {
		return errExit("%s", msg)
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	f := handler.LookupFilters{ID: id, SyncID: syncID, TopicKey: topicKey, Phrase: phrase, Scope: scope, Project: pf.project}
	res, err := h.Lookup(context.Background(), f)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// memContext runs `mem context` (byte-budget orientation).
func memContext(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem context", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var budget int
	var olderHistory bool
	fs.IntVar(&budget, "budget", 0, "byte budget (default 16 KiB)")
	fs.BoolVar(&olderHistory, "older-history", false, "walk older history")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	h, st, err := openRuntime(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := h.Context(context.Background(), budget, olderHistory)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// memReview runs `mem review` (list overdue / mark reviewed).
func memReview(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem review", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var markReviewed int64
	fs.Int64Var(&markReviewed, "mark-reviewed", 0, "mark this id reviewed")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if msg := rejectAll(pf); msg != "" {
		return errExit("%s", msg)
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := h.Review(context.Background(), pf.project, markReviewed == 0, markReviewed)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// memTimeline runs `mem timeline` (the service method is not yet exposed —
// the honest degraded envelope, R-2).
func memTimeline(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem timeline", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var sessionID string
	fs.StringVar(&sessionID, "session", "", "session id")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if sessionID == "" {
		return errExit("mem timeline: --session is required")
	}
	// R-2: the service method is not yet exposed — a degraded envelope,
	// never a healthy empty one (the same shape the MCP tool renders).
	// TODO(TICKET-16): wire timeline to the handler.
	res := map[string]any{
		"degraded":        true,
		"degraded_reason": "timeline: service not yet wired",
	}
	return finish(fs, *jsonOut, dispatchResult{result: res})
}

// memPin runs `mem pin` / `mem unpin`.
func memPin(fl config.Flags, cwd string, args []string, pin bool) int {
	verb := "pin"
	if !pin {
		verb = "unpin"
	}
	fs := flag.NewFlagSet("mem "+verb, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var id int64
	fs.Int64Var(&id, "id", 0, "local id")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if id == 0 {
		return errExit("mem %s: --id is required", verb)
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	var herr error
	if pin {
		herr = h.Pin(context.Background(), id, pf.project)
	} else {
		herr = h.Unpin(context.Background(), id, pf.project)
	}
	res := map[string]any{"pinned": pin}
	return finish(fs, *jsonOut, dispatchResult{result: res, err: herr})
}

// memDelete runs `mem delete` (soft by default, --hard deletes outright).
func memDelete(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem delete", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var id int64
	var hard bool
	var expectedProject string
	fs.Int64Var(&id, "id", 0, "local id")
	fs.BoolVar(&hard, "hard", false, "delete the row outright")
	fs.StringVar(&expectedProject, "expected-project", "", "ownership assertion (required for hard)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if id == 0 {
		return errExit("mem delete: --id is required")
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	// The ownership assertion defaults to the selected project (the
	// spec's mandatory expected_project check).
	ep := expectedProject
	if ep == "" {
		ep = pf.project
	}
	herr := h.Delete(context.Background(), handler.DeleteRequest{ID: id, ExpectedProject: ep, Hard: hard})
	res := map[string]any{"deleted": true}
	return finish(fs, *jsonOut, dispatchResult{result: res, err: herr})
}

// memSession runs `mem session start|end|summary|finalize`.
func memSession(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("mem session: a subcommand is required (start|end|summary|finalize)")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "start":
		return memSessionStart(fl, cwd, rest)
	case "end":
		return memSessionEnd(fl, cwd, rest)
	case "summary":
		return memSessionSummary(fl, cwd, rest)
	case "finalize":
		return memSessionFinalize(fl, cwd, rest)
	default:
		return errExit("mem session: unknown subcommand %q", sub)
	}
}

func memSessionStart(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem session start", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var id, directory, ownershipMode string
	var resume, isolated bool
	fs.StringVar(&id, "id", "", "session id")
	fs.StringVar(&directory, "directory", "", "working directory")
	fs.StringVar(&ownershipMode, "ownership-mode", "", "shared|project_owned")
	fs.BoolVar(&resume, "resume", false, "resume an existing session")
	fs.BoolVar(&isolated, "isolated", false, "isolated session")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if id == "" {
		return errExit("mem session start: --id is required")
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	herr := h.SessionStart(context.Background(), handler.SessionStartRequest{
		ID: id, Project: pf.project, Directory: directory,
		OwnershipMode: ownershipMode, Resume: resume, Isolated: isolated,
	})
	res := map[string]any{"session_id": id, "started": true}
	return finish(fs, *jsonOut, dispatchResult{result: res, err: herr})
}

func memSessionEnd(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem session end", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var sessionID, summary string
	fs.StringVar(&sessionID, "session", "", "session id")
	fs.StringVar(&summary, "summary", "", "free-text summary")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if sessionID == "" {
		return errExit("mem session end: --session is required")
	}
	h, st, err := openRuntime(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	herr := h.SessionEnd(context.Background(), sessionID, summary)
	res := map[string]any{"session_id": sessionID, "ended": true}
	return finish(fs, *jsonOut, dispatchResult{result: res, err: herr})
}

func memSessionSummary(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem session summary", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var sessionID string
	var goal, instructions, discoveries, accomplished, nextSteps, relevantFiles string
	fs.StringVar(&sessionID, "session", "", "session id")
	fs.StringVar(&goal, "goal", "", "goal")
	fs.StringVar(&instructions, "instructions", "", "instructions")
	fs.StringVar(&discoveries, "discoveries", "", "discoveries")
	fs.StringVar(&accomplished, "accomplished", "", "accomplished")
	fs.StringVar(&nextSteps, "next-steps", "", "next steps")
	fs.StringVar(&relevantFiles, "relevant-files", "", "relevant files")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if sessionID == "" {
		return errExit("mem session summary: --session is required")
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, herr := h.SessionSummary(context.Background(), handler.SessionSummaryRequest{
		SessionID: sessionID, Project: pf.project,
		SixSections: handler.SixSections{
			Goal: goal, Instructions: instructions, Discoveries: discoveries,
			Accomplished: accomplished, NextSteps: nextSteps, RelevantFiles: relevantFiles,
		},
	})
	if herr != nil {
		return finish(fs, *jsonOut, dispatchResult{err: herr})
	}
	return finish(fs, *jsonOut, dispatchResult{result: map[string]any{"sync_id": res.SyncID, "saved": true}})
}

func memSessionFinalize(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem session finalize", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	var sessionID string
	fs.StringVar(&sessionID, "session", "", "session id")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if sessionID == "" {
		return errExit("mem session finalize: --session is required")
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	herr := h.SessionFinalize(context.Background(), sessionID, pf.project)
	res := map[string]any{"session_id": sessionID, "finalized": true}
	return finish(fs, *jsonOut, dispatchResult{result: res, err: herr})
}

// memStats runs `mem stats` (per-type/project statistics).
func memStats(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("mem stats", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if msg := rejectAll(pf); msg != "" {
		return errExit("%s", msg)
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, herr := h.Stats(context.Background(), pf.project)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: herr})
}

// memDoctor runs `mem doctor` (the spec's per-project functional health
// check; the top-level `doctor` is the whole-system one).
func memDoctor(fl config.Flags, cwd string, args []string) int {
	return runDoctor(fl, cwd, args)
}
