package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	mnemonichttp "github.com/devopstales/skillgrid/mnemonic/internal/http"
)

// logsFollowPoll is the store-poll interval when `logs --follow` cannot reach
// `skillgrid serve` and tails the SQLite store directly.
const logsFollowPoll = 700 * time.Millisecond

type logsOpts struct {
	dataDir, project string
	filter           mnemonichttp.ToolEventFilter
	today            bool
	jsonOut          bool
	follow           bool
	serverURL        string
}

// runLogs handles `skillgrid logs`: the Gryph-style audit tail of every tool
// call the Cursor, OpenCode, and Kilo hooks recorded. Without --follow it
// prints the matching window oldest-first; with --follow it keeps streaming
// new calls from `skillgrid serve` (SSE), or from the store when no server runs.
func runLogs(args []string) {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var o logsOpts
	var format string
	fs.StringVar(&o.dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&o.project, "project", "", "project id (defaults to CWD-resolved)")
	fs.StringVar(&o.filter.Since, "since", "", "lower bound: RFC3339 or 30m/24h/7d/1w")
	fs.StringVar(&o.filter.Until, "until", "", "upper bound: RFC3339 or 30m/24h/7d/1w")
	fs.BoolVar(&o.today, "today", false, "only calls since local midnight")
	fs.StringVar(&o.filter.Agent, "agent", "", "harness: cursor, opencode, kilo")
	fs.StringVar(&o.filter.Session, "session", "", "session id")
	fs.StringVar(&o.filter.Action, "action", "", "file_read, file_write, command_exec, tool_use")
	fs.StringVar(&o.filter.Tool, "tool", "", "exact tool name (case-insensitive)")
	fs.StringVar(&o.filter.File, "file", "", "path glob (src/**, *.go)")
	fs.StringVar(&o.filter.Command, "command", "", "command glob (npm *)")
	fs.IntVar(&o.filter.Limit, "limit", 100, "max calls printed before following")
	fs.StringVar(&format, "format", "text", "text or json (one JSON object per line)")
	fs.BoolVar(&o.follow, "follow", false, "keep streaming new calls")
	fs.BoolVar(&o.follow, "f", false, "shorthand for --follow")
	fs.StringVar(&o.serverURL, "url", envOr("SKILLGRID_MNEMONIC_HTTP_URL", "http://127.0.0.1:7438"), "skillgrid serve base URL for --follow")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid logs [--today|--since 1h] [--agent A] [--session ID] [--action A] [--tool T] [--file GLOB] [--command GLOB] [--format text|json] [-f]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "error: --format must be text or json, got %q\n", format)
		os.Exit(2)
	}
	o.jsonOut = format == "json"
	if o.today {
		y, m, d := time.Now().Date()
		o.filter.Since = time.Date(y, m, d, 0, 0, 0, 0, time.Local).Format(time.RFC3339)
	}

	h, cleanup, err := openSessionService(o.dataDir, o.project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	db, projectID := h.Store().DB, h.ProjectID()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lastID, err := printLogsWindow(ctx, os.Stdout, db, projectID, o.filter, o.jsonOut)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if !o.follow {
		return
	}
	live := o.filter
	live.Since, live.Until = "", ""
	if err := followSSE(ctx, os.Stdout, o.serverURL, projectID, live, o.jsonOut); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "note: %v; tailing the store directly\n", err)
		live.AfterID = lastID
		if err := followStore(ctx, os.Stdout, db, projectID, live, o.jsonOut, logsFollowPoll); err != nil && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}

// printLogsWindow prints the newest `Limit` matching calls oldest-first and
// returns the highest event id seen (0 when none), the follow cursor.
func printLogsWindow(ctx context.Context, w io.Writer, db *sql.DB, projectID string, f mnemonichttp.ToolEventFilter, jsonOut bool) (int64, error) {
	events, err := mnemonichttp.QueryToolEvents(ctx, db, projectID, f)
	if err != nil {
		return 0, err
	}
	var last int64
	for i := len(events) - 1; i >= 0; i-- {
		writeLogEvent(w, events[i], jsonOut)
		if events[i].ID > last {
			last = events[i].ID
		}
	}
	if last == 0 {
		_ = db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id),0) FROM session_events WHERE project = ?`, projectID).Scan(&last)
	}
	return last, nil
}

// followSSE streams `tool` frames from skillgrid serve's /activity/stream.
// It returns an error when the server cannot be reached, so the caller can
// fall back to tailing the store; it returns nil when ctx is cancelled.
func followSSE(ctx context.Context, w io.Writer, base, projectID string, f mnemonichttp.ToolEventFilter, jsonOut bool) error {
	u := strings.TrimRight(base, "/") + "/activity/stream?project=" + url.QueryEscape(projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("skillgrid serve not reachable at %s", base)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("skillgrid serve %s: HTTP %d", u, resp.StatusCode)
	}
	match := newLogMatcher(f)
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var event string
	var data strings.Builder
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if event == "tool" && data.Len() > 0 {
				var e mnemonichttp.ToolEvent
				if json.Unmarshal([]byte(data.String()), &e) == nil && match(e) {
					writeLogEvent(w, e, jsonOut)
				}
			}
			event = ""
			data.Reset()
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if ctx.Err() != nil {
		return nil
	}
	if err := sc.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("stream closed: %w", err)
	}
	return errors.New("stream closed by server")
}

// followStore polls session_events for rows after f.AfterID.
func followStore(ctx context.Context, w io.Writer, db *sql.DB, projectID string, f mnemonichttp.ToolEventFilter, jsonOut bool, every time.Duration) error {
	f.Asc = true
	f.Limit = 500
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		events, err := mnemonichttp.QueryToolEvents(ctx, db, projectID, f)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		for _, e := range events {
			writeLogEvent(w, e, jsonOut)
			f.AfterID = e.ID
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// newLogMatcher applies a filter to live frames with the server's semantics
// (lifecycle rows dropped, globs where * spans '/').
func newLogMatcher(f mnemonichttp.ToolEventFilter) func(mnemonichttp.ToolEvent) bool {
	file, cmd := globRegexp(f.File), globRegexp(f.Command)
	return func(e mnemonichttp.ToolEvent) bool {
		switch e.Action {
		case "session_start", "session_end", "commit":
			return false
		}
		if f.Session != "" && e.SessionID != f.Session {
			return false
		}
		if f.Agent != "" && e.Agent != f.Agent {
			return false
		}
		if f.Action != "" && e.Action != f.Action {
			return false
		}
		if f.Tool != "" && !strings.EqualFold(e.Tool, f.Tool) {
			return false
		}
		if file != nil && !file.MatchString(e.Path) {
			return false
		}
		if cmd != nil && !cmd.MatchString(e.Command) {
			return false
		}
		return true
	}
}

func globRegexp(g string) *regexp.Regexp {
	if strings.TrimSpace(g) == "" {
		return nil
	}
	var b strings.Builder
	b.WriteString("^")
	for _, r := range g {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(strings.ReplaceAll(b.String(), ".*.*", ".*"))
}

func writeLogEvent(w io.Writer, e mnemonichttp.ToolEvent, jsonOut bool) {
	if jsonOut {
		raw, _ := json.Marshal(e)
		fmt.Fprintln(w, string(raw))
		return
	}
	ts := e.TS
	if t, err := time.Parse(time.RFC3339, e.TS); err == nil {
		ts = t.Local().Format("15:04:05")
	}
	agent := e.Agent
	if agent == "" {
		agent = "mnemonic"
	}
	tool := e.Tool
	if e.MCP {
		tool += " [mcp]"
	}
	target := e.Command
	if target == "" {
		target = e.Path
	}
	target = truncate(strings.Join(strings.Fields(target), " "), 120)
	if e.Sensitive {
		target += " (sensitive)"
	}
	line := fmt.Sprintf("%s  %-8s %-8s %-12s %-24s %s", ts, agent, shortID(e.SessionID), e.Action, tool, target)
	if e.Result != "" && e.Result != "success" {
		line += "  [" + e.Result + "]"
	}
	fmt.Fprintln(w, strings.TrimRight(line, " "))
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// sessionRow is one `skillgrid sessions` line.
type sessionRow struct {
	ID           string   `json:"id"`
	Agent        string   `json:"agent"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	StartedAt    string   `json:"started_at"`
	LastActive   string   `json:"last_active"`
	ToolCalls    int      `json:"tool_calls"`
	LastTool     string   `json:"last_tool"`
	Errors       int      `json:"errors"`
	Blocked      int      `json:"blocked_actions"`
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	Model        string   `json:"model"`
	CostUSD      *float64 `json:"cost_usd"`
}

func listSessionRows(ctx context.Context, db *sql.DB, projectID, agent string, limit int) ([]sessionRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, COALESCE(s.agent,''), COALESCE(s.title,''), COALESCE(s.status,''),
		       s.started_at,
		       COALESCE((SELECT MAX(e.timestamp) FROM session_events e WHERE e.session_id = s.id), s.started_at),
		       (SELECT COUNT(*) FROM session_events e WHERE e.session_id = s.id
		          AND e.action_type NOT IN ('session_start','session_end','commit')
		          AND COALESCE(e.result_status,'success') NOT IN ('blocked','warned','guided')),
		       COALESCE((SELECT e.tool_name FROM session_events e WHERE e.session_id = s.id
		          AND COALESCE(e.tool_name,'') <> '' ORDER BY e.id DESC LIMIT 1), ''),
		       COALESCE(s.errors,0), COALESCE(s.blocked_actions,0),
		       COALESCE(s.input_tokens,0), COALESCE(s.output_tokens,0), COALESCE(s.model,''), s.cost_usd
		FROM sessions s
		WHERE s.project = ? AND (? = '' OR COALESCE(s.agent,'') = ?)
		ORDER BY 6 DESC
		LIMIT ?`, projectID, agent, agent, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []sessionRow{}
	for rows.Next() {
		var r sessionRow
		var cost sql.NullFloat64
		if err := rows.Scan(&r.ID, &r.Agent, &r.Title, &r.Status, &r.StartedAt, &r.LastActive,
			&r.ToolCalls, &r.LastTool, &r.Errors, &r.Blocked, &r.InputTokens, &r.OutputTokens, &r.Model, &cost); err != nil {
			return nil, err
		}
		if cost.Valid {
			v := cost.Float64
			r.CostUSD = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// runSessions handles `skillgrid sessions`: every recorded session, newest
// activity first, with its harness, tool-call count, last tool, and cost.
func runSessions(args []string) {
	fs := flag.NewFlagSet("sessions", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dataDir, projectFlag, agent string
	var limit int
	var jsonOut bool
	fs.StringVar(&dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&projectFlag, "project", "", "project id (defaults to CWD-resolved)")
	fs.StringVar(&agent, "agent", "", "only sessions of this harness")
	fs.IntVar(&limit, "limit", 50, "max sessions")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	h, cleanup, err := openSessionService(dataDir, projectFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	rows, err := listSessionRows(hCtx(), h.Store().DB, h.ProjectID(), agent, limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		printJSON(map[string]any{"project": h.ProjectID(), "sessions": rows})
		return
	}
	writeSessionRows(os.Stdout, rows)
}

func writeSessionRows(w io.Writer, rows []sessionRow) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "no sessions recorded")
		return
	}
	fmt.Fprintf(w, "%-36s %-9s %-7s %6s  %-20s %-16s %s\n", "SESSION", "AGENT", "STATUS", "CALLS", "LAST TOOL", "LAST ACTIVE", "COST")
	for _, r := range rows {
		agent := r.Agent
		if agent == "" {
			agent = "mnemonic"
		}
		last := r.LastActive
		if t, err := time.Parse(time.RFC3339, last); err == nil {
			last = t.Local().Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%-36s %-9s %-7s %6d  %-20s %-16s %s\n", r.ID, agent, r.Status, r.ToolCalls,
			truncate(r.LastTool, 20), last, formatCostCLI(r.CostUSD))
	}
}

// runStats handles `skillgrid stats`: the per-agent rollup the Agent Stats
// page shows, for terminals.
func runStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dataDir, projectFlag, since, agent string
	var jsonOut bool
	fs.StringVar(&dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&projectFlag, "project", "", "project id (defaults to CWD-resolved)")
	fs.StringVar(&since, "since", "24h", "window: RFC3339 or 30m/24h/7d/1w ('' = all time)")
	fs.StringVar(&agent, "agent", "", "only this harness")
	fs.BoolVar(&jsonOut, "json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	h, cleanup, err := openSessionService(dataDir, projectFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	st, err := mnemonichttp.ComputeEventStats(hCtx(), h.Store().DB, h.ProjectID(), since, agent)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		printJSON(st)
		return
	}
	writeStats(os.Stdout, st)
}

func writeStats(w io.Writer, st mnemonichttp.EventStats) {
	window := "all time"
	if st.Since != "" {
		window = "since " + st.Since
	}
	fmt.Fprintf(w, "%s (%s): %d tool calls, %d sessions, %d MCP, %d errors, %d blocked\n",
		st.Project, window, st.Total, st.Sessions, st.MCP, st.Errors, st.Blocked)
	if st.Warned+st.Guided > 0 {
		fmt.Fprintf(w, "policy: %d warned, %d guided\n", st.Warned, st.Guided)
	}
	fmt.Fprintf(w, "\n%-10s %8s %6s %6s %6s %8s %5s %6s %9s %s\n", "AGENT", "SESSIONS", "CALLS", "READS", "WRITES", "COMMANDS", "MCP", "ERRORS", "TOKENS", "COST")
	for _, a := range st.ByAgent {
		fmt.Fprintf(w, "%-10s %8d %6d %6d %6d %8d %5d %6d %9d %s\n", a.Agent, a.Sessions, a.Events, a.Reads, a.Writes,
			a.Commands, a.MCP, a.Errors, a.InputTokens+a.OutputTokens, formatCostCLI(a.CostUSD))
	}
	if len(st.ByModel) > 0 {
		fmt.Fprintf(w, "\n%-28s %8s %10s %10s %10s %s\n", "MODEL", "SESSIONS", "INPUT", "OUTPUT", "CACHE", "COST")
		for _, m := range st.ByModel {
			fmt.Fprintf(w, "%-28s %8d %10d %10d %10d %s\n", truncate(m.Model, 28), m.Sessions,
				m.InputTokens, m.OutputTokens, m.CacheTokens, formatCostCLI(m.CostUSD))
		}
	}
	writeTop(w, "Top files", st.TopFiles)
	writeTop(w, "Top commands", st.TopCommands)
	writeTop(w, "Top tools", st.TopTools)
}

func writeTop(w io.Writer, title string, items []mnemonichttp.NameCount) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s\n", title)
	for _, i := range items {
		name := i.Name
		if i.MCP {
			name += " [mcp]"
		}
		fmt.Fprintf(w, "  %5d  %s\n", i.Count, name)
	}
}

func formatCostCLI(v *float64) string {
	if v == nil {
		return "n/a"
	}
	if *v < 0.01 {
		return fmt.Sprintf("$%.4f", *v)
	}
	return fmt.Sprintf("$%.2f", *v)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
