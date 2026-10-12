package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/observability"
	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// The two CTX observability read endpoints (plan §17, TICKET-09). They are
// read-only: they inherit the existing UI server's auth model (no
// requireWriteAuth on a GET) and are fail-open — a missing ctx_* table
// degrades to 200 with zero-valued JSON, never a 500 (ADR-0016). Both work
// through the shared project store (ctxstore.New(h.Store()), ADR-0033); the
// single-open invariant holds because each route opens its ProjectHandle
// exactly once via withProjectHandle and ctxstore wraps that handle's DB.
//
// The *Server does not carry the ctx *config.CtxConfig, and threading it
// through NewServer would disturb unrelated Server construction, so the
// rollup list is loaded per-request from the project root via config.Load
// (pure filesystem I/O, fail-open to the four standard granularities). That
// is the smallest seam that reads the operator's configured HistoryRollups
// without a constructor change.

// ctxObservabilityConfig loads the CTX config for a project root and returns
// it with the standard Observability defaults when the load yields no usable
// rollup list. startDir is the project root (h.Root()). It never returns an
// error: config.Load itself is fail-open, and an empty HistoryRollups list
// falls back to the four standard granularities.
func ctxObservabilityConfig(ctx context.Context, root string) *config.CtxConfig {
	cfg, err := config.Load(ctx, root)
	if err != nil || cfg == nil {
		cfg = &config.CtxConfig{}
		*cfg = config.DefaultCtxConfig()
	}
	if len(cfg.Observability.HistoryRollups) == 0 {
		cfg.Observability.HistoryRollups = config.DefaultCtxConfig().Observability.HistoryRollups
	}
	return cfg
}

// requireLoopback rejects any non-loopback origin to a ctx observability
// route with 403 and a "loopback only" body that carries no data (plan §17,
// TICKET-10). The ctx_* tables hold per-session token/cache counters and
// summary JSON that are meant for the local agent and operator, not for
// arbitrary network peers on the same host. Loopback covers the IPv4
// 127.0.0.0/8 range, IPv6 ::1, and an absent RemoteAddr (the httptest and
// in-process caller convention). It runs before withProjectHandle so the
// 403 short-circuits the store open.
func requireLoopback(w http.ResponseWriter, r *http.Request) bool {
	addr := r.RemoteAddr
	if addr == "" {
		return true
	}
	// RemoteAddr is "<host>:<port>"; the port suffix is not part of the IP.
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	// A bare IP literal that is loopback passes; a name or non-loopback IP
	// (proxy-injected peer, forwarded host) is rejected — names are never
	// loopback by definition here.
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return ip.IsLoopback()
}

// handleCTXStats serves GET /stats: the live session-metric snapshot from
// ctx_proxy_sessions (plan §17.2/§17.5). The Stats struct carries a Mode
// field (the latest session's mode) as the seam for the T11 run-mode header;
// it is populated here but not wired into any header (out of scope). A nil or
// table-missing store yields a zero-valued Stats, always 200.
func (s *Server) handleCTXStats(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "ctx observability routes are loopback only"})
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		cs := ctxstore.New(h.Store())
		st, err := observability.Snapshot(cs)
		if err != nil || st == nil {
			// Fail-open: never 500 a read-only observability route.
			writeJSON(w, http.StatusOK, &observability.Stats{})
			return
		}
		writeJSON(w, http.StatusOK, st)
	})
}

// handleCTXStatsHistory serves GET /stats-history: the durable savings
// history from ctx_stats_history for the granularities configured in
// cfg.Observability.HistoryRollups (plan §17.2/§17.4). A missing table yields
// an empty per-granularity list (the JSON shape stays stable: every requested
// granularity appears, possibly empty), always 200.
func (s *Server) handleCTXStatsHistory(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "ctx observability routes are loopback only"})
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		cs := ctxstore.New(h.Store())
		cfg := ctxObservabilityConfig(r.Context(), h.Root())
		grans := make([]observability.Granularity, 0, len(cfg.Observability.HistoryRollups))
		for _, g := range cfg.Observability.HistoryRollups {
			grans = append(grans, observability.Granularity(strings.TrimSpace(g)))
		}
		hist, err := observability.StatsHistory(cs, grans)
		if err != nil || hist == nil {
			// Fail-open: keep the stable shape (empty buckets per granularity).
			hist = &observability.History{Buckets: map[observability.Granularity][]observability.Bucket{}}
			for _, g := range grans {
				hist.Buckets[g] = []observability.Bucket{}
			}
		}
		writeJSON(w, http.StatusOK, hist)
	})
}

// handleCTXMetrics serves GET /metrics: the Prometheus text exposition of the
// four ctx observability counters (plan §17, TICKET-10). It is hand-emitted
// (stdlib only — no prometheus/client_golang dependency, ADR-0008 gate) and
// fail-open: a missing ctx_* table degrades to 200 with zero-valued counters,
// never a 500 (ADR-0016). The counters are the same source of truth as
// /stats (ctx_proxy_sessions via observability.Snapshot), so the two routes
// cannot disagree on the live totals.
func (s *Server) handleCTXMetrics(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "ctx observability routes are loopback only"})
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		cs := ctxstore.New(h.Store())
		st, err := observability.Snapshot(cs)
		if err != nil || st == nil {
			st = &observability.Stats{}
		}
		var b strings.Builder
		writeCounter(&b, "ctx_requests_total", "Total proxied LLM requests observed by the context orchestrator.", st.Requests)
		writeCounter(&b, "ctx_tokens_saved_total", "Total tokens the context orchestrator saved (before minus after).", st.TokensSaved)
		writeCounter(&b, "ctx_cache_hits_total", "Total context cache hits.", st.CacheHits)
		writeCounter(&b, "ctx_cache_invalidations_total", "Total context cache invalidations.", st.CacheInvalidations)
		writeCounter(&b, "ctx_sessions_total", "Total proxy sessions tracked by the context orchestrator.", int64(st.Sessions))
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(b.String()))
	})
}

// writeCounter emits one Prometheus counter block (HELP, TYPE, value) with an
// integer value. Counter values are non-negative by construction (the Snapshot
// aggregates SUMs of non-negative columns).
func writeCounter(b *strings.Builder, name, help string, value int64) {
	if value < 0 {
		value = 0
	}
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}

// sessionSummaryItem is the wire shape of one ctx_session_summaries row for
// GET /api/session-summaries (plan §17, TICKET-10). Field names are
// snake_case to match the rest of the ctx observability JSON.
type sessionSummaryItem struct {
	SessionID          string `json:"session_id"`
	Agent              string `json:"agent"`
	DurationMS         int64  `json:"duration_ms"`
	Turns              int64  `json:"turns"`
	TokensSaved        int64  `json:"tokens_saved"`
	JudgmentKeeps      int64  `json:"judgment_keeps"`
	JudgmentTruncs     int64  `json:"judgment_truncs"`
	PrefilterPins      int64  `json:"prefilter_pins"`
	PrefilterDeads     int64  `json:"prefilter_deads"`
	CacheInvalidations int64  `json:"cache_invalidations"`
	ShapingApplied     *bool  `json:"shaping_applied"`
	EffortRouted       int64  `json:"effort_routed"`
	TrafficMemories    int64  `json:"traffic_memories"`
	SummaryJSON        any    `json:"summary_json"`
	CreatedAt          int64  `json:"created_at"`
}

// sessionSummaryPage is the paginated response: the items for this page plus
// an opaque next cursor (the row offset of the following page, "" when the
// page is the last).
type sessionSummaryPage struct {
	Items []sessionSummaryItem `json:"items"`
	Next  string               `json:"next"`
}

// handleCTXSessionSummaries serves GET /api/session-summaries: a paginated,
// read-only view over ctx_session_summaries (plan §17, TICKET-10). Offset
// pagination is driven by the cursor (a row offset) and the limit query
// params; the stable ORDER BY created_at, session_id of
// Store.ListSessionSummaries keeps pages deterministic. Fail-open: a missing
// table yields 200 with an empty items list, never a 500 (ADR-0016).
func (s *Server) handleCTXSessionSummaries(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "ctx observability routes are loopback only"})
		return
	}
	q := r.URL.Query()
	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	cursor := 0
	if v := q.Get("cursor"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cursor = n
		}
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		cs := ctxstore.New(h.Store())
		all, err := cs.ListSessionSummaries()
		if err != nil {
			// Fail-open: keep the stable page shape (empty items).
			all = nil
		}
		items := make([]sessionSummaryItem, 0, limit)
		start := cursor
		end := start + limit
		if start > len(all) {
			start = len(all)
		}
		if end > len(all) {
			end = len(all)
		}
		for _, row := range all[start:end] {
			items = append(items, summaryItemFromRow(row))
		}
		next := ""
		if end < len(all) {
			next = strconv.Itoa(end)
		}
		writeJSON(w, http.StatusOK, sessionSummaryPage{Items: items, Next: next})
	})
}

// summaryItemFromRow maps one ctx_session_summaries row to its wire item,
// parsing summary_json into a generic value (fail-open to the raw string when
// it is not valid JSON — the column is written by the summarizer, not by the
// reader, so the reader never assumes a shape).
func summaryItemFromRow(p *ctxstore.SessionSummary) sessionSummaryItem {
	it := sessionSummaryItem{
		SessionID:          p.SessionID,
		Agent:              p.Agent,
		DurationMS:         p.DurationMS,
		Turns:              p.Turns,
		TokensSaved:        p.TokensSaved,
		JudgmentKeeps:      p.JudgmentKeeps,
		JudgmentTruncs:     p.JudgmentTruncs,
		PrefilterPins:      p.PrefilterPins,
		PrefilterDeads:     p.PrefilterDeads,
		CacheInvalidations: p.CacheInvalidations,
		ShapingApplied:     p.ShapingApplied,
		EffortRouted:       p.EffortRouted,
		TrafficMemories:    p.TrafficMemories,
		CreatedAt:          p.CreatedAt,
	}
	var v any
	if p.SummaryJSON != "" {
		if err := json.Unmarshal([]byte(p.SummaryJSON), &v); err != nil {
			v = p.SummaryJSON
		}
	}
	it.SummaryJSON = v
	return it
}
