package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// toolEvent is one session_events row in the shape the Sessions timeline, the
// /events query, the SSE `tool` frame, and `skillgrid logs` share.
type toolEvent struct {
	ID        int64  `json:"id"`
	TS        string `json:"ts"`
	SessionID string `json:"sessionId"`
	Sequence  int    `json:"sequence"`
	Agent     string `json:"agent"`
	Action    string `json:"action"`
	Tool      string `json:"tool"`
	Path      string `json:"path"`
	Command   string `json:"command"`
	Result    string `json:"result"`
	Sensitive bool   `json:"sensitive"`
	Preview   string `json:"preview"`
	MCP       bool   `json:"mcp"`
}

// builtinTools are the harness-native tool names (Cursor, OpenCode, Kilo,
// Claude-style). Any other tool name is reported as an MCP call.
var builtinTools = map[string]bool{
	"read": true, "write": true, "edit": true, "multiedit": true, "strreplace": true,
	"shell": true, "bash": true, "grep": true, "glob": true, "ls": true, "list": true,
	"delete": true, "patch": true, "applypatch": true, "task": true, "todowrite": true,
	"todoread": true, "webfetch": true, "websearch": true, "readlints": true,
	"editnotebook": true, "notebookedit": true, "semanticsearch": true, "codebase_search": true,
	"read_file": true, "edit_file": true, "run_terminal_cmd": true, "grep_search": true,
	"file_search": true, "list_dir": true, "delete_file": true, "web_search": true,
	"askquestion": true, "question": true, "skill": true, "switchmode": true,
}

// isMCPTool reports whether a recorded tool name is an MCP server tool rather
// than a harness built-in. Lifecycle rows (no tool name) are never MCP.
func isMCPTool(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if strings.HasPrefix(n, "mcp") || strings.Contains(n, "__") {
		return true
	}
	return !builtinTools[n]
}

// toolEventFilter narrows the session_events query. Empty fields do not filter.
type toolEventFilter struct {
	Session string
	Agent   string
	Action  string
	Tool    string
	File    string // glob: * and ** match any run, ? one char
	Command string // glob, same syntax as File
	Since   string // RFC3339 or a relative window (30m, 24h, 7d, 1w)
	Until   string
	AfterID int64
	Limit   int
	// Asc returns oldest-first (SSE poll and follow); default is newest-first.
	Asc bool
	// IncludeLifecycle keeps session_start/session_end/commit rows.
	IncludeLifecycle bool
}

// globToLike turns a shell-style glob into a SQL LIKE pattern (escape '\').
func globToLike(g string) string {
	var b strings.Builder
	for _, r := range g {
		switch r {
		case '%', '_', '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case '*':
			b.WriteRune('%')
		case '?':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	for strings.Contains(out, "%%") {
		out = strings.ReplaceAll(out, "%%", "%")
	}
	return out
}

// parseSince accepts RFC3339 or a relative window ending in m/h/d/w and
// returns an RFC3339 UTC lower bound. Empty input returns "".
func parseSince(v string, now time.Time) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	if len(v) >= 2 {
		n, err := strconv.Atoi(v[:len(v)-1])
		if err == nil && n >= 0 {
			var d time.Duration
			switch v[len(v)-1] {
			case 'm':
				d = time.Duration(n) * time.Minute
			case 'h':
				d = time.Duration(n) * time.Hour
			case 'd':
				d = time.Duration(n) * 24 * time.Hour
			case 'w':
				d = time.Duration(n) * 7 * 24 * time.Hour
			default:
				return "", fmt.Errorf("invalid since %q", v)
			}
			return now.Add(-d).UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("invalid since %q (want RFC3339 or 30m/24h/7d/1w)", v)
}

// queryToolEvents reads session_events for projectID joined to the session's
// agent. Sensitive rows keep their metadata but never expose the preview.
func queryToolEvents(ctx context.Context, db *sql.DB, projectID string, f toolEventFilter) ([]toolEvent, error) {
	where := []string{"e.project = ?"}
	args := []any{projectID}
	if !f.IncludeLifecycle {
		where = append(where, "e.action_type NOT IN ('session_start','session_end','commit')")
	}
	if f.Session != "" {
		where = append(where, "e.session_id = ?")
		args = append(args, f.Session)
	}
	if f.Agent != "" {
		where = append(where, "COALESCE(s.agent,'') = ?")
		args = append(args, f.Agent)
	}
	if f.Action != "" {
		where = append(where, "e.action_type = ?")
		args = append(args, f.Action)
	}
	if f.Tool != "" {
		where = append(where, "LOWER(COALESCE(e.tool_name,'')) = LOWER(?)")
		args = append(args, f.Tool)
	}
	if f.File != "" {
		where = append(where, `COALESCE(e.path,'') LIKE ? ESCAPE '\'`)
		args = append(args, globToLike(f.File))
	}
	if f.Command != "" {
		where = append(where, `COALESCE(e.command,'') LIKE ? ESCAPE '\'`)
		args = append(args, globToLike(f.Command))
	}
	now := time.Now()
	if since, err := parseSince(f.Since, now); err != nil {
		return nil, err
	} else if since != "" {
		where = append(where, "e.timestamp >= ?")
		args = append(args, since)
	}
	if until, err := parseSince(f.Until, now); err != nil {
		return nil, err
	} else if until != "" {
		where = append(where, "e.timestamp <= ?")
		args = append(args, until)
	}
	if f.AfterID > 0 {
		where = append(where, "e.id > ?")
		args = append(args, f.AfterID)
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	order := "DESC"
	if f.Asc {
		order = "ASC"
	}
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, `
		SELECT e.id, e.timestamp, e.session_id, e.sequence, COALESCE(s.agent,''),
		       e.action_type, COALESCE(e.tool_name,''), COALESCE(e.path,''),
		       COALESCE(e.command,''), COALESCE(e.result_status,'success'),
		       COALESCE(e.is_sensitive,0), COALESCE(e.payload,'')
		FROM session_events e
		LEFT JOIN sessions s ON s.id = e.session_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY e.id `+order+`
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []toolEvent{}
	for rows.Next() {
		var e toolEvent
		var sens int
		if err := rows.Scan(&e.ID, &e.TS, &e.SessionID, &e.Sequence, &e.Agent, &e.Action,
			&e.Tool, &e.Path, &e.Command, &e.Result, &sens, &e.Preview); err != nil {
			return nil, err
		}
		e.Sensitive = sens != 0
		if p, ok := policyPreview(e.Result, e.Preview); ok {
			e.Preview = p
		} else if e.Sensitive {
			e.Preview = ""
		} else if len(e.Preview) > 500 {
			e.Preview = e.Preview[:500]
		}
		e.MCP = isMCPTool(e.Tool)
		out = append(out, e)
	}
	return out, rows.Err()
}

// policyPreview renders a policy decision row's payload as "rule: message".
// The message is the rule author's text, so it is shown even on sensitive paths.
func policyPreview(result, payload string) (string, bool) {
	switch result {
	case "blocked", "warned", "guided":
	default:
		return "", false
	}
	var p struct {
		Rule    string `json:"policy_rule"`
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(payload), &p) != nil || p.Rule == "" {
		return "", false
	}
	if p.Message == "" {
		return p.Rule, true
	}
	return p.Rule + ": " + p.Message, true
}

// filterFromRequest maps the shared query params onto a toolEventFilter.
func filterFromRequest(r *http.Request) toolEventFilter {
	q := r.URL.Query()
	f := toolEventFilter{
		Session: strings.TrimSpace(q.Get("session")),
		Agent:   strings.TrimSpace(q.Get("agent")),
		Action:  strings.TrimSpace(q.Get("action")),
		Tool:    strings.TrimSpace(q.Get("tool")),
		File:    strings.TrimSpace(q.Get("file")),
		Command: strings.TrimSpace(q.Get("command")),
		Since:   strings.TrimSpace(q.Get("since")),
		Until:   strings.TrimSpace(q.Get("until")),
		Limit:   queryInt(r, "limit", 200),
	}
	if v, err := strconv.ParseInt(q.Get("after"), 10, 64); err == nil {
		f.AfterID = v
	}
	f.IncludeLifecycle = q.Get("lifecycle") == "true"
	return f
}

// handleSessionEvents serves GET /sessions/{id}/events — one session's tool
// timeline, newest first. An unknown id returns an empty list.
func (s *Server) handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		f := filterFromRequest(r)
		f.Session = r.PathValue("id")
		events, err := queryToolEvents(r.Context(), h.Store().DB, projectID, f)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"project": projectID, "sessionId": f.Session, "events": events})
	})
}

// handleEvents serves GET /events — the project-wide tool-call query
// (agent, action, tool, file glob, command glob, session, since/until).
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		f := filterFromRequest(r)
		events, err := queryToolEvents(r.Context(), h.Store().DB, projectID, f)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"project": projectID, "events": events, "limit": f.Limit})
	})
}
