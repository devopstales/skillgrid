package http

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

type agentStat struct {
	Agent        string   `json:"agent"`
	Sessions     int      `json:"sessions"`
	Events       int      `json:"events"`
	Reads        int      `json:"reads"`
	Writes       int      `json:"writes"`
	Commands     int      `json:"commands"`
	MCP          int      `json:"mcp"`
	Errors       int      `json:"errors"`
	Blocked      int      `json:"blocked"`
	InputTokens  int64    `json:"inputTokens"`
	OutputTokens int64    `json:"outputTokens"`
	CostUSD      *float64 `json:"costUsd"`
}

type modelStat struct {
	Model        string   `json:"model"`
	Sessions     int      `json:"sessions"`
	InputTokens  int64    `json:"inputTokens"`
	OutputTokens int64    `json:"outputTokens"`
	CacheTokens  int64    `json:"cacheTokens"`
	CostUSD      *float64 `json:"costUsd"`
}

// addCost sums a nullable SQL cost into a running total that stays nil until
// some session has a known cost.
func addCost(total *float64, c sql.NullFloat64) *float64 {
	if !c.Valid {
		return total
	}
	v := c.Float64
	if total != nil {
		v += *total
	}
	return &v
}

type nameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	MCP   bool   `json:"mcp,omitempty"`
}

type eventStats struct {
	Project     string         `json:"project"`
	Since       string         `json:"since"`
	Total       int            `json:"total"`
	Sessions    int            `json:"sessions"`
	MCP         int            `json:"mcp"`
	Errors      int            `json:"errors"`
	Sensitive   int            `json:"sensitive"`
	Blocked     int            `json:"blocked"`
	Warned      int            `json:"warned"`
	Guided      int            `json:"guided"`
	ByAction    map[string]int `json:"byAction"`
	ByAgent     []agentStat    `json:"byAgent"`
	ByModel     []modelStat    `json:"byModel"`
	TopFiles    []nameCount    `json:"topFiles"`
	TopCommands []nameCount    `json:"topCommands"`
	TopTools    []nameCount    `json:"topTools"`
}

const statsTopN = 10

// computeEventStats aggregates tool calls (lifecycle rows excluded) for the
// project since `since` (RFC3339, "" = all time), optionally for one agent.
// Sessions, tokens, and cost count every session with activity in the window.
func computeEventStats(ctx context.Context, db *sql.DB, projectID, since, agent string) (eventStats, error) {
	st := eventStats{Project: projectID, Since: since, ByAction: map[string]int{}}
	where := "e.project = ? AND e.action_type NOT IN ('session_start','session_end','commit')"
	args := []any{projectID}
	if since != "" {
		where += " AND e.timestamp >= ?"
		args = append(args, since)
	}
	if agent != "" {
		where += " AND COALESCE(s.agent,'') = ?"
		args = append(args, agent)
	}
	from := " FROM session_events e LEFT JOIN sessions s ON s.id = e.session_id WHERE " + where

	byAgent := map[string]*agentStat{}
	agentOf := func(name string) *agentStat {
		if a, ok := byAgent[name]; ok {
			return a
		}
		a := &agentStat{Agent: name}
		byAgent[name] = a
		return a
	}
	tools := map[string]int{}

	rows, err := db.QueryContext(ctx, `
		SELECT COALESCE(s.agent,''), e.action_type, COALESCE(e.tool_name,''),
		       COALESCE(e.result_status,'success'), COALESCE(e.is_sensitive,0), COUNT(*)`+from+`
		GROUP BY 1, 2, 3, 4, 5`, args...)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var ag, action, tool, result string
		var sens, n int
		if err := rows.Scan(&ag, &action, &tool, &result, &sens, &n); err != nil {
			rows.Close()
			return st, err
		}
		a := agentOf(ag)
		// Policy decision rows (ADR-0021) are not calls: count the outcome only.
		switch result {
		case "blocked":
			a.Blocked += n
			st.Blocked += n
			continue
		case "warned":
			st.Warned += n
			continue
		case "guided":
			st.Guided += n
			continue
		}
		a.Events += n
		st.Total += n
		st.ByAction[action] += n
		switch action {
		case "file_read":
			a.Reads += n
		case "file_write":
			a.Writes += n
		case "command_exec":
			a.Commands += n
		}
		if isMCPTool(tool) {
			a.MCP += n
			st.MCP += n
		}
		if result == "error" || result == "failed" {
			a.Errors += n
			st.Errors += n
		}
		if sens != 0 {
			st.Sensitive += n
		}
		if tool != "" {
			tools[tool] += n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return st, err
	}

	// Sessions with activity in the window, plus their token/cost totals.
	sargs := []any{projectID}
	swhere := "s.project = ?"
	if since != "" {
		swhere += " AND (s.started_at >= ? OR EXISTS (SELECT 1 FROM session_events x WHERE x.session_id = s.id AND x.timestamp >= ?))"
		sargs = append(sargs, since, since)
	}
	if agent != "" {
		swhere += " AND COALESCE(s.agent,'') = ?"
		sargs = append(sargs, agent)
	}
	srows, err := db.QueryContext(ctx, `
		SELECT COALESCE(s.agent,''), COALESCE(s.model,''), COUNT(*), COALESCE(SUM(s.input_tokens),0),
		       COALESCE(SUM(s.output_tokens),0), COALESCE(SUM(s.cache_tokens),0), SUM(s.cost_usd)
		FROM sessions s WHERE `+swhere+` GROUP BY 1, 2`, sargs...)
	if err != nil {
		return st, err
	}
	byModel := map[string]*modelStat{}
	for srows.Next() {
		var ag, model string
		var n int
		var in, out, cache int64
		var cost sql.NullFloat64
		if err := srows.Scan(&ag, &model, &n, &in, &out, &cache, &cost); err != nil {
			srows.Close()
			return st, err
		}
		a := agentOf(ag)
		a.Sessions += n
		a.InputTokens += in
		a.OutputTokens += out
		a.CostUSD = addCost(a.CostUSD, cost)
		st.Sessions += n
		if model != "" {
			m, ok := byModel[model]
			if !ok {
				m = &modelStat{Model: model}
				byModel[model] = m
			}
			m.Sessions += n
			m.InputTokens += in
			m.OutputTokens += out
			m.CacheTokens += cache
			m.CostUSD = addCost(m.CostUSD, cost)
		}
	}
	st.ByModel = make([]modelStat, 0, len(byModel))
	for _, m := range byModel {
		st.ByModel = append(st.ByModel, *m)
	}
	sort.Slice(st.ByModel, func(i, j int) bool {
		ci, cj := 0.0, 0.0
		if st.ByModel[i].CostUSD != nil {
			ci = *st.ByModel[i].CostUSD
		}
		if st.ByModel[j].CostUSD != nil {
			cj = *st.ByModel[j].CostUSD
		}
		if ci != cj {
			return ci > cj
		}
		return st.ByModel[i].Model < st.ByModel[j].Model
	})
	srows.Close()
	if err := srows.Err(); err != nil {
		return st, err
	}

	st.ByAgent = make([]agentStat, 0, len(byAgent))
	for _, a := range byAgent {
		if a.Agent == "" {
			a.Agent = "mnemonic"
		}
		st.ByAgent = append(st.ByAgent, *a)
	}
	sort.Slice(st.ByAgent, func(i, j int) bool {
		if st.ByAgent[i].Events != st.ByAgent[j].Events {
			return st.ByAgent[i].Events > st.ByAgent[j].Events
		}
		return st.ByAgent[i].Agent < st.ByAgent[j].Agent
	})

	st.TopTools = make([]nameCount, 0, len(tools))
	for name, n := range tools {
		st.TopTools = append(st.TopTools, nameCount{Name: name, Count: n, MCP: isMCPTool(name)})
	}
	st.TopTools = sortTop(st.TopTools)

	calls := from + " AND COALESCE(e.result_status,'success') NOT IN ('blocked','warned','guided')"
	if st.TopFiles, err = topColumn(ctx, db, "e.path", calls, args); err != nil {
		return st, err
	}
	// Sensitive commands are counted but not named.
	if st.TopCommands, err = topColumn(ctx, db, "e.command", calls+" AND COALESCE(e.is_sensitive,0) = 0", args); err != nil {
		return st, err
	}
	return st, nil
}

func sortTop(xs []nameCount) []nameCount {
	sort.Slice(xs, func(i, j int) bool {
		if xs[i].Count != xs[j].Count {
			return xs[i].Count > xs[j].Count
		}
		return xs[i].Name < xs[j].Name
	})
	if len(xs) > statsTopN {
		xs = xs[:statsTopN]
	}
	return xs
}

func topColumn(ctx context.Context, db *sql.DB, col, from string, args []any) ([]nameCount, error) {
	q := "SELECT " + col + ", COUNT(*)" + from + " AND COALESCE(" + col + ",'') <> '' GROUP BY 1 ORDER BY 2 DESC, 1 LIMIT ?"
	rows, err := db.QueryContext(ctx, q, append(append([]any{}, args...), statsTopN)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []nameCount{}
	for rows.Next() {
		var nc nameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err != nil {
			return nil, err
		}
		out = append(out, nc)
	}
	return out, rows.Err()
}

// handleEventStats serves GET /events/stats?since=&agent= — the Gryph-style
// stats dashboard: totals, per-action and per-agent breakdowns, top files,
// commands, and tools, with token and cost totals per agent.
func (s *Server) handleEventStats(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		since, err := parseSince(r.URL.Query().Get("since"), time.Now())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		st, err := computeEventStats(r.Context(), h.Store().DB, projectID, since,
			strings.TrimSpace(r.URL.Query().Get("agent")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, st)
	})
}
