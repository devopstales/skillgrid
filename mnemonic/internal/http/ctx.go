package http

import (
	"context"
	"net/http"
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

// handleCTXStats serves GET /stats: the live session-metric snapshot from
// ctx_proxy_sessions (plan §17.2/§17.5). The Stats struct carries a Mode
// field (the latest session's mode) as the seam for the T11 run-mode header;
// it is populated here but not wired into any header (out of scope). A nil or
// table-missing store yields a zero-valued Stats, always 200.
func (s *Server) handleCTXStats(w http.ResponseWriter, r *http.Request) {
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
