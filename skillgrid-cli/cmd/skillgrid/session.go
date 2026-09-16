package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runSession handles `skillgrid session handoff|resume|status`. It is the CLI
// parity layer for the Session Relay (change 006, step 04): each subcommand
// resolves the SAME project store the MCP server uses and calls the SAME
// relay Module functions (Handoff / Resume / Status) so the CLI outcomes
// mirror the MCP outcomes on that store. Bad flags / a missing id / no usable
// store fail closed (non-zero exit + stderr, no partial cleave bundle).
func runSession(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printSessionUsage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("session", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir     string
		projectFlag string
		progress    string
		knowledge   string
		nextPrompt  string
		handoffID   string
		sessionID   string
		ctxSummary  string
		archive     bool
		ctxPct      string
		costUSD     string
		watchdog    bool
		usage       string
	)
	fs.StringVar(&dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&projectFlag, "project", "", "project id (defaults to CWD-resolved)")
	fs.StringVar(&progress, "progress", "", "handoff: what was done this session (PROGRESS.md)")
	fs.StringVar(&knowledge, "knowledge", "", "handoff: decisions / learnings (KNOWLEDGE.md)")
	fs.StringVar(&nextPrompt, "next-prompt", "", "handoff: the prompt that seeds the next session (NEXT_PROMPT.md)")
	fs.StringVar(&handoffID, "handoff-id", "", "handoff: operator-facing id (blank generates one)")
	fs.StringVar(&sessionID, "session-id", "", "handoff: source session id (sessions.id)")
	fs.StringVar(&ctxSummary, "context-summary", "", "handoff: optional context note recorded on the row")
	fs.BoolVar(&archive, "archive", false, "resume: also archive this handoff")
	fs.StringVar(&ctxPct, "context-usage-percent", "", "status: optional caller-supplied context usage percent (0-100)")
	fs.StringVar(&costUSD, "cost-usd", "", "status: optional caller-supplied last known cost in USD")
	fs.BoolVar(&watchdog, "watchdog", false, "handoff: gate this handoff behind the context-limit watchdog (SKILLGRID_HANDOFF_WATCHDOG)")
	fs.StringVar(&usage, "usage", "", "handoff --watchdog: caller-supplied context-usage fraction (0.0-1.0)")
	if err := fs.Parse(reorderSessionArgs(rest)); err != nil {
		os.Exit(2)
	}

	h, cleanup, err := openSessionService(dataDir, projectFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	pos := fs.Args()

	switch cmd {
	case "handoff":
		if watchdog {
			runSessionHandoffWatchdog(h, usage, progress, knowledge, nextPrompt, handoffID, sessionID, ctxSummary)
		} else {
			runSessionHandoff(h, progress, knowledge, nextPrompt, handoffID, sessionID, ctxSummary)
		}
	case "resume":
		runSessionResume(h, pos, archive)
	case "status":
		runSessionStatus(h, ctxPct, costUSD)
	case "help", "-h", "--help":
		printSessionUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown session command %q\n", cmd)
		printSessionUsage()
		os.Exit(2)
	}
}

func printSessionUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid session <handoff|resume|status> [flags]

  handoff --progress P --next-prompt N [--knowledge K] [--session-id S]
          [--handoff-id ID] [--context-summary C]
          write the cleave bundle + session_handoffs row (session_handoff)
  handoff --progress P --next-prompt N --usage F --watchdog
          gate the handoff behind the context-limit watchdog: only hands off
          when SKILLGRID_HANDOFF_WATCHDOG is set AND F >= the threshold
          (off by default; invalid config fails closed) (session_watchdog)
  resume <handoff_id> [--archive]
          return the stored NEXT_PROMPT for a handoff (session_resume)
  status [--context-usage-percent N] [--cost-usd C]
          report the handoff count (+ optional caller-supplied stats) (session_status)

Flags:
  --project ID    project bucket (defaults to CWD-resolved)
  --dir DATA_DIR  mnemonic data directory
`)
}

// openSessionService resolves the project store the SAME way the MCP server
// does (CWD-resolved when no --project, explicit --project otherwise) and
// opens it once. Any resolution / open failure is returned so the caller fails
// closed (no usable store -> non-zero exit, no partial cleave bundle).
func openSessionService(dataDir, projectName string) (*service.ProjectHandle, func(), error) {
	dd := dataDir
	if dd == "" {
		d, err := service.DefaultDataDir()
		if err != nil {
			return nil, nil, fmt.Errorf("resolve data dir: %w", err)
		}
		dd = d
	}
	svc := service.New(dd)
	var h *service.ProjectHandle
	var cleanup func()
	var err error
	if strings.TrimSpace(projectName) != "" {
		h, cleanup, err = svc.Open(project.NormalizeID(projectName))
	} else {
		h, cleanup, err = svc.OpenForCWD()
	}
	if err != nil {
		return nil, nil, err
	}
	return h, cleanup, nil
}

func runSessionHandoff(h *service.ProjectHandle, progress, knowledge, nextPrompt, handoffID, sessionID, ctxSummary string) {
	if strings.TrimSpace(progress) == "" {
		fmt.Fprintln(os.Stderr, "error: session handoff requires --progress")
		os.Exit(2)
	}
	if strings.TrimSpace(nextPrompt) == "" {
		fmt.Fprintln(os.Stderr, "error: session handoff requires --next-prompt")
		os.Exit(2)
	}
	in := relay.Bundle{
		Progress:       progress,
		Knowledge:      knowledge,
		NextPrompt:     nextPrompt,
		SourceSession:  sessionID,
		ContextSummary: ctxSummary,
	}
	id, paths, err := relay.Handoff(hCtx(), h.Store().DB, h.ProjectID(), handoffID, h.Root(), in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printJSON(map[string]any{
		"handoff_id": id,
		"paths":      paths,
	})
}

// runSessionHandoffWatchdog is the CLI wiring for the step-05 context-limit
// watchdog. It is the operator surface that turns relay.Check on: the handoff
// is gated behind SKILLGRID_HANDOFF_WATCHDOG (+ _THRESHOLD) and only runs when
// the caller-supplied --usage fraction is at/past the threshold. Off by
// default (env unset -> no-op, never auto-hands-off); invalid config fails
// closed (non-zero exit, no handoff). The usage fraction is caller-supplied
// (0.0-1.0), matching the step-05 usage-signal decision (the CLI does not
// compute tokens).
func runSessionHandoffWatchdog(h *service.ProjectHandle, usage, progress, knowledge, nextPrompt, handoffID, sessionID, ctxSummary string) {
	if strings.TrimSpace(progress) == "" {
		fmt.Fprintln(os.Stderr, "error: session handoff --watchdog requires --progress")
		os.Exit(2)
	}
	if strings.TrimSpace(nextPrompt) == "" {
		fmt.Fprintln(os.Stderr, "error: session handoff --watchdog requires --next-prompt")
		os.Exit(2)
	}
	frac, err := strconv.ParseFloat(usage, 64)
	if err != nil || frac < 0 || frac > 1 {
		fmt.Fprintf(os.Stderr, "error: bad --usage %q (need a fraction in [0,1])\n", usage)
		os.Exit(2)
	}
	in := relay.Bundle{
		Progress:       progress,
		Knowledge:      knowledge,
		NextPrompt:     nextPrompt,
		SourceSession:  sessionID,
		ContextSummary: ctxSummary,
	}
	res, err := relay.Check(hCtx(), h.Store().DB, h.ProjectID(), h.Root(), frac, in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if !res.HandedOff {
		printJSON(map[string]any{
			"handed_off": false,
			"note":       "watchdog no-op (disabled, below threshold, or off by default)",
		})
		return
	}
	printJSON(map[string]any{
		"handed_off": true,
		"handoff_id": res.HandoffID,
		"paths":      res.Paths,
	})
}

func runSessionResume(h *service.ProjectHandle, pos []string, archive bool) {
	if len(pos) < 1 {
		fmt.Fprintln(os.Stderr, "error: session resume requires a handoff_id")
		os.Exit(2)
	}
	id := pos[0]
	prompt, hid, archiveID, err := relay.Resume(hCtx(), h.Store().DB, h.ProjectID(), id, h.Root(), archive)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out := map[string]any{
		"prompt":     prompt,
		"handoff_id": hid,
	}
	if archive {
		out["archive_id"] = archiveID
	}
	printJSON(out)
}

func runSessionStatus(h *service.ProjectHandle, ctxPct, costUSD string) {
	var stats *relay.Stats
	if ctxPct != "" || costUSD != "" {
		stats = &relay.Stats{}
		if ctxPct != "" {
			v, err := strconv.ParseFloat(ctxPct, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: bad --context-usage-percent %q\n", ctxPct)
				os.Exit(2)
			}
			stats.ContextUsagePercent = &v
		}
		if costUSD != "" {
			v, err := strconv.ParseFloat(costUSD, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: bad --cost-usd %q\n", costUSD)
				os.Exit(2)
			}
			stats.CostUSD = &v
		}
	}
	st, err := relay.Status(hCtx(), h.Store().DB, h.ProjectID(), stats)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out := map[string]any{
		"handoff_count": st.HandoffCount,
	}
	if st.ContextUsagePercent != nil {
		out["context_usage_percent"] = *st.ContextUsagePercent
	}
	if st.CostUSD != nil {
		out["cost_usd"] = *st.CostUSD
	}
	printJSON(out)
}

// reorderSessionArgs moves flags in front of positionals (same convention as
// mem / trail) so `session resume <id> --archive` parses like
// `session resume --archive <id>`.
func reorderSessionArgs(args []string) []string {
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
