================================================================================
CONTEXT ORCHESTRATOR ON TOP OF SKILLGRID MNEMONIC
Build Plan v1.11
(updated: proxy surface, session observability endpoints, real-time traffic
 learning — folded into Mnemonic's existing UI server and session events)
================================================================================

PURPOSE
--------------------------------------------------------------------------------
Build a thin orchestration layer ("CTX") on top of Skillgrid's mnemonic SQLite
database. CTX does NOT re-index code, does NOT re-implement parsing, does NOT
own storage, and does NOT re-implement hybrid search. It reads Mnemonic's
existing schema, composes multi-source queries into declarative recipes,
applies reversible per-type compression, shapes both the input and the output
of the model, and exposes a programmatic observability surface. It returns
coherent context blocks to LLM clients over MCP, over a transparent proxy,
via hooks, or as exported documents.

Clean division of labor:
    Mnemonic owns  — code index, symbols, edges, chunks, hybrid search,
                     session events, token-aware injection, composite
                     retrieval tools, CHECKPOINTS (file-first handoff),
                     AND the existing web UI. v1.11 also folds the new
                     observability endpoints and the real-time traffic
                     learner into Mnemonic's existing UI server and
                     session-event pipeline.
    CTX owns       — enforcement hooks, folder abstraction, originals cache
                     (CCR), redaction, recipe composition, cache alignment,
                     determinism, retrieval ladder, the judgment layer,
                     the OpenCode plugin adapter, the deterministic
                     pre-filter, the PrefixGuard, the output-shaping
                     layer, the failure-mining loop, the proxy surface,
                     and the session summary aggregator.
    OpenCode owns  — the agent loop, tool execution, session management,
                     permission evaluation, and the plugin host.
    Judgment model — a small, fast, specialized model (Jev-class) that makes
                     keep / truncate / compress-tier decisions on context
                     units. It NEVER deletes. It NEVER rewrites prose.
                     Deletion is a code-level decision.

v1.11 changes (learned from headroomlabs-ai/headroom's proxy function,
applied to Mnemonic's session-event monitoring):
    * New Phase 16: Proxy Surface + Compression-as-a-Service. A raw,
      protocol-compatible proxy (`ctx proxy --port 8787`) accepts any
      OpenAI- or Anthropic-compatible client via a base-URL override.
      Three modes: token (max compression), cache (freeze cached prefixes,
      compress only the mutable tail), passthrough (debug). A
      compression-only endpoint `/v1/compress` is exposed for gateway
      sidecars. Loopback-only by default; non-loopback callers receive
      404 (not 403) so the route stays invisible to scanners. Savings
      headers are returned on every response. Fail-open: compression
      failure degrades to passthrough; provider timeouts retry with
      backoff.
    * Two-tier integration strategy. (1) A raw proxy for any agent that
      exposes a base-URL override. (2) `ctx wrap <agent>` — a config-
      injecting launcher for a curated set of popular agents (Claude
      Code, Codex, Cursor, Aider, Copilot CLI, OpenCode, OpenClaw,
      Cline, Continue, Goose). The proxy itself stays agent-agnostic;
      only the setup layer is per-agent.
    * New Phase 17: Session Observability Endpoints. Folded into
      Mnemonic's EXISTING UI server (no second process, no second DB):
        GET /stats              live session metrics (JSON)
        GET /stats-history      durable savings history with hourly /
                                daily / weekly / monthly rollups
        GET /metrics            Prometheus-formatted operational metrics
      Plus a new per-session wide event aggregated at SessionEnd
      (duration, turns, tokens saved, decisions made) — one row per
      session instead of a join over many session events.
    * New Phase 18: Real-Time Traffic Learning. A zero-latency, rule-
      based extractor that mines session events DURING the session for
      reusable memories — no LLM calls. Four categories: error_recovery
      (failed tool call followed by a successful retry), environment
      (stable facts: working commands, paths), preference (repeated
      usage patterns, user corrections), architecture (file structure,
      coding conventions). Complements the offline `ctx learn` loop
      (v1.10 Phase 15): real-time extraction during the session plus
      periodic deep mining across sessions.
    * Run mode becomes an observability signal. The proxy's current
      mode (token / cache / passthrough) is exposed in the session
      health view so operators understand why savings or latency look
      the way they do. Mode changes are emitted as events.

v1.10 (carried forward):
    * Output Shaping (verbosity steering + effort routing).
    * Failure Mining (ctx learn): proposed / applied / revoked lifecycle.
    * Live-zone / frozen-prefix split; named PrefixGuard component.
    * Statistical pre-filter rules (errors, outliers, boundaries).
    * Auto-router for folders.
    * Control holdout in benchmarks (10%).

v1.9 (carried forward):
    * Checkpoint consumption (read-only) via the Mnemonic client wrapper.
    * Deterministic pre-filter (facts-first, structural-dead, label-remainder).
    * Truncate-only invariant; deletion is code-only.
    * Judgment threshold tuned empirically (default 0.22; range 0.20-0.25).
    * Prefix-cache accounting (ctx_cache_invalidations); batched edits.
    * Message-rewrite vs context-assembly distinction made explicit.

v1.8 (carried forward):
    * Judgment Layer (curator agent).
    * OpenCode Plugin Layer.

v1.7 (carried forward):
    * Web UI observability folded into Mnemonic's existing web UI.
    * SSE /stream, /api/events, /api/health, /api/settings.
    * Live panel, card feed, infinite scroll, dedup, health, theme.

v1.6 (carried forward):
    * Retrieval ladder; non-deterministic channel; compress_session;
      hooks enqueue; SessionEnd; <private>...</private>.

v1.5 (carried forward):
    * Hook enforcement; composite Mnemonic tool delegation; compute;
      ctx_index lazy pre-population; ctx_session_fts; no prose-shaping
      in cache-aligned prefix.

v1.4 (carried forward):
    * Folder abstraction generalized; AST folding scoped to code only.

v1.3 (carried forward):
    * CCR; auto-tier feedback loop; cache alignment; transparent proxy;
      accuracy benchmarks; lifecycle event bus; local-only guarantee.

v1.2 (carried forward):
    * CTX shares the Mnemonic SQLite database; ctx_ prefix; snapshot/
      restore survives re-index.

v1.1 (carried forward):
    * Deterministic outputs for KV cache hits; Shannon-entropy
      redaction; branch-aware freshness.

================================================================================
0. PREREQUISITES
================================================================================

0.1 Environment
    - skillgrid-cli installed and working (release/2 branch), including its
      existing web UI (skillgrid-ui.mjs or equivalent) and its file-first
      checkpoint / handoff feature.
    - A project already indexed: `skillgrid mnemonic index <path>`
    - Mnemonic DB present at ~/.skillgrid/mnemonic/<project>/.sqlite
    - At least one MCP client available for testing
    - A hook-capable client available for enforcement testing
    - OpenCode installed (v1.x) for plugin-layer development and testing
    - At least one agent that exposes a base-URL override (for proxy
      testing) and at least one that does not (for wrap testing)
    - A modern browser for the web UI
    - Git available on PATH
    - Network access to the judgment model endpoint (Jev-class) OR a local
      judgment model binary
    - A provider that exposes thinking-effort control (optional; effort
      routing degrades to a no-op without it)
    - Prometheus or an OpenMetrics-compatible scraper (optional; for
      /metrics)

0.2 Knowledge required
    - Everything from v1.10
    - Transparent proxy patterns:
        * HTTP/1.1, HTTP/2, SSE, and WebSocket passthrough
        * base-URL override as the universal integration seam
        * token / cache / passthrough run modes and their tradeoffs
        * compression-as-a-service (`/v1/compress`) for gateway sidecars
        * loopback-only binding with 404 (not 403) for non-loopback
        * fail-open on compression error; retry with backoff on provider
          timeout
        * savings headers on every response
    - Two-tier agent integration:
        * raw proxy for any base-URL-configurable client
        * config-injecting launcher (`ctx wrap`) for popular agents
        * the proxy stays agent-agnostic; only the setup layer is per-agent
    - Layered observability:
        * live stats endpoint (`/stats`)
        * durable history with rollups (`/stats-history`)
        * Prometheus metrics (`/metrics`)
        * per-session wide event aggregated at SessionEnd
    - Real-time traffic learning:
        * rule-based extraction, zero LLM calls
        * four categories: error_recovery, environment, preference,
          architecture
        * dual-layer learning: real-time + periodic offline mining
    - Run mode as an observability signal

0.3 Decisions to lock before starting
    - Language: Python or TypeScript (unchanged).
    - Transport: OpenCode plugin (primary), MCP stdio, MCP HTTP,
      transparent proxy, hooks, or a mix. v1.11 adds the proxy as a
      first-class transport with its own lifecycle.
    - Storage: the Mnemonic DB (unchanged). CTX never DDLs non-ctx_
      tables and never creates a second store. Proxy savings history
      is persisted to the DB (ctx_proxy_savings) AND to a JSON
      snapshot under the project directory, mirroring Mnemonic's
      file-first philosophy.
    - Retrieval path: Mnemonic MCP tools; composite preferred.
    - Checkpoints: Mnemonic owns them. CTX reads them read-only.
    - Default compression tier: "balanced".
    - Originals cache: enabled by default; TTL 7 days.
    - Auto-tier feedback loop: enabled by default; N>=10 runs.
    - Folders: pluggable; AST folding is the code folder only.
    - Auto-router: enabled by default; falls back to code.
    - Hook policy: intercept only high-output tools; enqueue, don't
      process; 200 ms budget.
    - Non-deterministic channel: empty by default; opt-in per recipe.
    - compress_session step: off by default; requires an LLM budget.
    - Web UI: extend the EXISTING Skillgrid/Mnemonic UI. No second UI,
      no second server, no second database, no control plane.
    - UI stack: whatever Skillgrid already uses.
    - Judgment model: Jev-class, fast, cheap, structured-output only.
      Fail-open. Fingerprint-cached. Truncate-only. Threshold 0.22.
    - Pre-filter: enabled by default. Deterministic, auditable,
      code-only. Includes statistical rules for JSON.
    - Message-rewrite policy: CTX NEVER rewrites stored messages. It
      only changes how the next request is assembled. Enforced by the
      PrefixGuard component.
    - Prefix-cache accounting: enabled by default. Live-zone /
      frozen-prefix split enforced.
    - Output shaping: enabled by default. Verbosity steering appended
      AFTER the cache-aligned prefix. Effort routing enabled when the
      provider supports it.
    - Failure mining: enabled by default. Rules are textual and
      human-auditable. Promotion requires N>=10 sessions.
    - Plugin host: OpenCode is the PRIMARY runtime adapter.
    - Proxy: enabled by default, bound to loopback. Default mode:
      cache (preserve prefix, compress tail). Non-loopback callers
      receive 404 on `/v1/compress`. Savings headers on.
    - Two-tier integration: raw proxy + `ctx wrap <agent>` for a
      curated agent set. The proxy is agent-agnostic; only the
      launcher knows about specific agents.
    - Observability endpoints: folded into the existing UI server.
      `/stats`, `/stats-history`, `/metrics`, and the per-session
      summary. All read-only.
    - Real-time traffic learning: enabled by default. Rule-based,
      zero LLM calls. Extracted memories are stored in
      ctx_traffic_memories and surfaced in the Live view.

================================================================================
1. PHASE 1 - FOUNDATION (Week 1)
================================================================================

1.1 Project scaffold (v1.10, plus)
    - New directory: proxy/ for the proxy surface and the wrap launcher.
    - New directory: observability/ for the stats, history, and metrics
      endpoints.
    - New directory: traffic/ for the real-time traffic learner rules.
    - New module: session_summary.(py|ts) — the per-session aggregator.

1.2 Config schema (context.yaml) (v1.10, plus)
    - New top-level key:
        proxy:
          enabled: true
          bind: 127.0.0.1
          port: 8787
          mode: cache               # token | cache | passthrough
          compress_endpoint: true   # /v1/compress
          non_loopback_status: 404  # do not reveal route
          savings_headers: true
          fail_open: true
          provider_retries: 3
          provider_retry_backoff_ms: [100, 400, 1600]
          passthrough_protocols: [http1.1, http2, sse, ws]
          upstreams:
            anthropic: https://api.anthropic.com
            openai: https://api.openai.com
    - New top-level key:
        wrap:
          enabled: true
          agents: [claude, codex, cursor, aider, copilot, opencode,
                   openclaw, cline, continue, goose]
          inject_env: true
          backup_config: true
    - New top-level key:
        observability:
          enabled: true
          stats: true
          stats_history: true
          metrics: true
          metrics_path: /metrics
          history_rollups: [hourly, daily, weekly, monthly]
          history_snapshot: true    # also write JSON under the project
          history_snapshot_path: .skillgrid/ctx/stats_history.json
          session_summary: true
    - New top-level key:
        traffic_learning:
          enabled: true
          realtime: true
          rules_path: traffic/rules.yaml
          categories:
            - error_recovery
            - environment
            - preference
            - architecture
          min_confidence: 0.6
          max_memories: 500
          emit_events: true
    - Proxy mode is also settable via CTX_PROXY_MODE env var (mirrors
      HEADROOM_MODE).

1.3 Mnemonic client wrapper (v1.10, plus)
    - New read-only accessors for the session-event stream that the
      traffic learner consumes:
        mnemonic.session_events.stream(session_id) -> [event]
        mnemonic.session_events.since(ts) -> [event]
    - These are read-only. CTX never writes to Mnemonic's event stream.

1.4 CTX store (shared DB) (v1.10, unchanged)

1.5 IDE + agent installer (v1.10, plus)
    - Installs the default proxy config.
    - Installs the default traffic learner rules into traffic/rules.yaml.
    - Registers the wrap targets that are detected on the machine
      (idempotent; backs up before writing).

DELIVERABLE: Same as v1.10, plus proxy, wrap, observability, and traffic
             learner scaffolding.

================================================================================
2. PHASE 2 - RECIPE ENGINE & RETRIEVAL LADDER (Week 2)
================================================================================

2.1 - 2.12 (unchanged from v1.10)

2.13 Proxy mode is a recipe input (v1.11)
    - A recipe may declare `proxy_mode: token | cache | passthrough`.
    - This lets a recipe request aggressive compression for a batch
      job and cache-preserving behavior for an interactive session.
    - The default is the proxy's current mode.

DELIVERABLE: Same as v1.10, plus proxy mode as a recipe input.

================================================================================
3. PHASE 3 - FOLDERS & COMPRESSION (Weeks 3-4)
================================================================================

3.1 - 3.16 (unchanged from v1.10)

3.17 Proxy mode selects the compression strategy (v1.11)
    - cache mode: frozen prefix is byte-identical; only the live zone
      is compressed. This is the default for long interactive sessions.
    - token mode: prior history may be rewritten when it improves
      compression. This maximizes savings but may invalidate the
      provider prefix cache.
    - passthrough: no compression; the proxy forwards verbatim.
    - The mode is recorded in ctx_proxy_mode_history and emitted as a
      proxy-mode-changed event.

DELIVERABLE: Same as v1.10, plus mode-driven compression strategy.

================================================================================
4. PHASE 4 - REVERSIBLE COMPRESSION (CCR) (Week 4)
================================================================================

4.1 - 4.11 (unchanged from v1.10)

4.12 Proxy integrates CCR automatically (v1.11)
    - When the proxy compresses or drops content, the original is
      stored in ctx_originals and a `headroom_retrieve`-equivalent
      tool (ctx_retrieve) is injected into the request.
    - This is the same reversibility guarantee as v1.3; the proxy just
      makes it automatic.

DELIVERABLE: Same as v1.10, plus automatic CCR injection in the proxy.

================================================================================
5. PHASE 5 - HOOK ENFORCEMENT (Week 4, parallel with Phase 3/4)
================================================================================

5.1 - 5.13 (unchanged from v1.10)

5.14 Hooks and the proxy share the event bus (v1.11)
    - Hook decisions and proxy requests both emit on the lifecycle
      event bus.
    - The Live view renders both without distinction; the source is a
      field on the card.

DELIVERABLE: Same as v1.10, plus proxy events on the shared bus.

================================================================================
6. PHASE 6 - REDACTION (Week 4, parallel with Phase 3/4/5)
================================================================================

6.1 - 6.11 (unchanged from v1.10)

6.12 The proxy inherits the redaction boundary (v1.11)
    - Redaction applies to proxied requests and responses exactly as
      it does to hooked requests.
    - The proxy never forwards unredacted content upstream.
    - Because the proxy is loopback-only by default, the redaction
      boundary is the last line of defense, not the first.

DELIVERABLE: Same as v1.10, plus proxy-side redaction.

================================================================================
7. PHASE 7 - MCP SERVER, PROXY, SOURCES (Week 5)
================================================================================

7.1 - 7.13 (unchanged from v1.10)

7.14 Proxy is a first-class MCP-adjacent surface (v1.11)
    - The proxy and the MCP server are complementary:
        MCP      explicit tool calls from the agent
        proxy    transparent interception of the model's wire traffic
    - Both share the same DB, the same event bus, and the same pipeline.

DELIVERABLE: Same as v1.10, plus the proxy as a first-class surface.

================================================================================
8. PHASE 8 - LIFECYCLE EVENT BUS (Week 5, parallel with Phase 7)
================================================================================

8.1 - 8.7 (unchanged from v1.10)

8.8 New event types (v1.11)
    - proxy-request          upstream, mode, tokens in/out
    - proxy-savings          tokens before/after, cache hit, mode
    - proxy-mode-changed     old mode, new mode, reason
    - session-summary        one per session at SessionEnd
    - traffic-memory-extracted
                             category, confidence, evidence
    - stats-history-rotated  hourly / daily / weekly / monthly
    - metrics-scraped        (debug; off by default)

DELIVERABLE: Same as v1.10, plus proxy, session-summary, and traffic
             events.

================================================================================
9. PHASE 9 - WORKER, CLI, FRESHNESS, OPS (Week 6)
================================================================================

9.1 Worker service (v1.10, plus)
    - New job type: proxy_savings_flush (batch-write savings to DB).
    - New job type: stats_history_rollup (hourly / daily / weekly /
      monthly).
    - New job type: session_summary_aggregate (triggered by SessionEnd).
    - New job type: traffic_mine_realtime (short, bounded; runs during
      the session).

9.2 CLI commands (v1.10, plus)
    ctx ui                         (carried forward)
    ctx ui --follow <type>         (carried forward)
    ctx judge                      (carried forward)
    ctx judge --explain <fp>       (carried forward)
    ctx judge --replay <sid>       (carried forward)
    ctx judge tune                 (carried forward)
    ctx prefilter explain <id>     (carried forward)
    ctx cache report               (carried forward)
    ctx checkpoint list            (carried forward)
    ctx checkpoint read <id>       (carried forward)
    ctx opencode install           (carried forward)
    ctx opencode doctor            (carried forward)
    ctx shape status               (carried forward)
    ctx shape override <turn>      (carried forward)
    ctx learn                      (carried forward)
    ctx learn report               (carried forward)
    ctx learn apply <rule-id>      (carried forward)
    ctx learn revoke <rule-id>     (carried forward)
    ctx prefix-guard report        (carried forward)
    ctx proxy                      start the proxy (loopback by default)
    ctx proxy status               show bind, mode, upstreams, uptime,
                                   request count, savings total
    ctx proxy mode <mode>          switch token / cache / passthrough
    ctx wrap <agent>               start the proxy, inject config, and
                                   launch the named agent
    ctx stats                      live stats (same as GET /stats)
    ctx stats --history            historical stats with rollups
    ctx metrics                    print Prometheus metrics to stdout
    ctx traffic memories           list extracted memories
    ctx traffic report             quality report (precision, categories)
    ctx traffic rules reload       reload traffic/rules.yaml

9.3 CTX tables (v1.10, plus)
    ... (all v1.10 tables unchanged) ...
    - New in v1.11:
        ctx_proxy_sessions(
          session_id       TEXT PRIMARY KEY,
          agent            TEXT,     -- detected from headers/UA
          mode             TEXT,     -- token | cache | passthrough
          started_at       INTEGER,
          ended_at         INTEGER,
          requests         INTEGER,
          tokens_before    INTEGER,
          tokens_after     INTEGER,
          cache_hits       INTEGER,
          cache_invalidations INTEGER
        )
        ctx_proxy_savings(
          saving_id        TEXT PRIMARY KEY,
          session_id       TEXT,
          turn             INTEGER,
          upstream         TEXT,
          mode             TEXT,
          tokens_before    INTEGER,
          tokens_after     INTEGER,
          savings_ratio    REAL,
          created_at       INTEGER
        )
        ctx_proxy_mode_history(
          change_id        TEXT PRIMARY KEY,
          session_id       TEXT,
          old_mode         TEXT,
          new_mode         TEXT,
          reason           TEXT,
          changed_at       INTEGER
        )
        ctx_session_summaries(
          session_id       TEXT PRIMARY KEY,
          agent            TEXT,
          duration_ms      INTEGER,
          turns            INTEGER,
          tokens_saved     INTEGER,
          judgment_keeps   INTEGER,
          judgment_truncs  INTEGER,
          prefilter_pins   INTEGER,
          prefilter_deads  INTEGER,
          cache_invalidations INTEGER,
          shaping_applied  BOOLEAN,
          effort_routed    INTEGER,
          traffic_memories INTEGER,
          summary_json     TEXT,     -- the full wide event
          created_at       INTEGER
        )
        ctx_stats_history(
          bucket_id        TEXT PRIMARY KEY,
          granularity      TEXT,     -- hourly | daily | weekly | monthly
          bucket_start     INTEGER,
          tokens_saved     INTEGER,
          requests         INTEGER,
          sessions         INTEGER,
          cache_hits       INTEGER,
          created_at       INTEGER
        )
        ctx_traffic_memories(
          memory_id        TEXT PRIMARY KEY,
          session_id       TEXT,
          category         TEXT,     -- error_recovery | environment |
                                     -- preference | architecture
          memory_text      TEXT,
          evidence_json    TEXT,
          confidence       REAL,
          created_at       INTEGER
        )

9.4 Freshness checks (unchanged)
9.5 Logging (v1.10, plus proxy requests, mode changes, savings totals,
     history rollups, traffic extractions at INFO)
9.6 Packaging (v1.10, plus the proxy is a self-contained entry point
     with its own CLI subcommand; no new artifact beyond the CTX core)

DELIVERABLE: Same as v1.10, plus proxy, observability, traffic, and
             session-summary CLI and tables.

================================================================================
10. PHASE 10 - WEB UI OBSERVABILITY LAYER (Week 7)
================================================================================

10.1 - 10.12 (unchanged from v1.10)

10.13 New card types (v1.11)
     - ProxyRequestCard       upstream, mode, tokens in/out, latency
     - ProxySavingsCard       tokens before/after, ratio, cache hit
     - ProxyModeCard          old mode, new mode, reason
     - SessionSummaryCard     duration, turns, tokens saved, decisions
     - TrafficMemoryCard      category, text, confidence, evidence
     - New filters: by upstream, by mode, by traffic category, by
       session outcome (saved / no-save).

10.14 Live view header (v1.11)
     - Now also shows the current proxy mode (token / cache /
       passthrough) alongside the existing connection health
       indicator.
     - Mode changes are rendered as a small inline event.

DELIVERABLE: Same as v1.10, plus proxy, session-summary, and traffic
             cards in the Live view.

================================================================================
11. PHASE 11 - BENCHMARKS, HARDENING, RELEASE (Week 7)
================================================================================

11.1 - 11.19 (unchanged from v1.10)

11.20 Proxy benchmarks (v1.11)
     - Compatibility: verify the proxy against at least 5 agents via
       base-URL override (Anthropic-style + OpenAI-style) and against
       at least 3 agents via `ctx wrap`.
     - Protocol passthrough: HTTP/1.1, HTTP/2, SSE, and WebSocket
       streams survive the proxy intact.
     - Savings headers: X-CTX-Savings, X-CTX-Original-Tokens,
       X-CTX-Compressed-Tokens are present and accurate.
     - `/v1/compress`: returns a compressed message list without
       generating a completion.
     - Loopback invisibility: a non-loopback caller receives 404 on
       `/v1/compress`.
     - Fail-open: a compression error degrades to passthrough; the
       request still completes.
     - Provider timeout: retries with the configured backoff.
     - Mode switching: token / cache / passthrough behave as
       documented; a mode change is recorded and emitted.

11.21 Observability benchmarks (v1.11)
     - `/stats` returns live session metrics.
     - `/stats-history` returns rollups and survives a restart.
     - `/metrics` is Prometheus-parseable.
     - Session summary aggregation completes within the SessionEnd
       hook budget; the hook does not block the agent.
     - History snapshot JSON is written under the project directory
       and matches the DB.

11.22 Traffic learner benchmarks (v1.11)
     - Precision: extracted memories must match a human-labeled set
       above the configured confidence threshold.
     - Categories: each of the four categories is exercised.
     - Cost: zero LLM calls per extraction (rule-based only).
     - Bounded: max_memories caps the store.
     - Dual-layer: real-time extractions and offline `ctx learn`
       rules do not contradict each other.

11.23 Documentation (v1.10, plus)
     - PROXY.md (new): proxy surface, modes, `/v1/compress`, loopback
       binding, savings headers, fail-open, provider retries, the
       two-tier integration strategy (raw proxy + `ctx wrap`), and
       the agent-agnostic boundary.
     - OBSERVABILITY.md (new): `/stats`, `/stats-history`, `/metrics`,
       the per-session wide event, rollup cadence, snapshot JSON.
     - TRAFFIC_LEARNING.md (new): rule contract, the four categories,
       confidence threshold, bounded store, the dual-layer learning
       model, and how it composes with `ctx learn`.
     - UI_EVENTS.md (updated): new proxy, session-summary, and
       traffic event schemas.

11.24 Release: tag v1.11.0.

DELIVERABLE: v1.11.0 release with the proxy surface, the observability
             endpoints, and real-time traffic learning, all folded
             into Mnemonic's existing UI server and session-event
             pipeline.

================================================================================
12. PHASE 12 - JUDGMENT LAYER (CURATOR AGENT) (Week 8)
================================================================================

12.1 - 12.12 (unchanged from v1.10)

12.13 Proxy and judgment compose (v1.11)
    - The proxy runs the same pipeline as the plugin: redact ->
      prefilter -> judge -> compress -> shape -> emit.
    - The proxy's mode (token / cache / passthrough) selects the
      compression strategy, not the judgment threshold.

DELIVERABLE: Same as v1.10, plus the proxy as a pipeline host.

================================================================================
13. PHASE 13 - OPENCODE PLUGIN LAYER (Week 8, parallel with Phase 12)
================================================================================

13.1 - 13.16 (unchanged from v1.10)

13.17 Plugin and proxy are complementary (v1.11)
    - The plugin handles event-driven context management inside
      OpenCode.
    - The proxy handles transparent wire-level interception for
      OpenCode and any other agent.
    - Both share the same DB, event bus, and pipeline.
    - The plugin can run with the proxy enabled, disabled, or in
      passthrough mode; the choice is per-project.

DELIVERABLE: Same as v1.10, plus plugin / proxy complementarity.

================================================================================
14. PHASE 14 - OUTPUT SHAPING (Week 9)
================================================================================

14.1 - 14.7 (unchanged from v1.10)

14.8 Proxy applies output shaping (v1.11)
    - The proxy installs the verbosity-steering note at session start
      and applies effort routing per turn, exactly as the plugin does.
    - Shaping is deterministic; the proxy never rewrites stored
      messages (PrefixGuard).

DELIVERABLE: Same as v1.10, plus proxy-side output shaping.

================================================================================
15. PHASE 15 - FAILURE MINING (ctx learn) (Week 9, parallel with Phase 14)
================================================================================

15.1 - 15.8 (unchanged from v1.10)

15.9 Traffic learner feeds ctx learn (v1.11)
    - Real-time traffic memories are a new input to the offline
      failure-mining loop.
    - A real-time memory with high confidence may seed a proposed
      rule; promotion still requires the v1.10 gate.
    - The two layers are complementary: real-time extraction during
      the session, deep mining across sessions.

DELIVERABLE: Same as v1.10, plus traffic memories as a learn input.

================================================================================
16. PHASE 16 - PROXY SURFACE + COMPRESSION-AS-A-SERVICE (Week 10)
================================================================================

16.1 Principle
     - The proxy is the zero-code-change integration seam. It sits
       between the agent and the LLM provider and applies CTX's
       pipeline transparently.
     - The proxy is agent-AGNOSTIC. It speaks Anthropic and OpenAI
       wire protocols and forwards HTTP/1.1, HTTP/2, SSE, and
       WebSocket traffic verbatim, applying the pipeline in the
       middle.
     - The proxy is NOT the only integration path. MCP, hooks, and
       the OpenCode plugin remain supported and compose with it.
     - The proxy is loopback-only by default.

16.2 Modes
     - token        maximize compression. Prior history may be
                    rewritten when it improves compression.
     - cache        freeze provider-confirmed cached prefixes and
                    compress only the mutable tail. Default for
                    long interactive sessions.
     - passthrough  no compression; forward verbatim. Used for
                    debugging and for agents that need raw traffic.
     - Mode is settable at startup (`ctx proxy --mode cache`), at
       runtime (`ctx proxy mode token`), or via CTX_PROXY_MODE.

16.3 Endpoints
     - /health, /livez, /readyz    liveness / readiness
     - /stats                      live session metrics (JSON)
     - /stats-history              durable savings history with rollups
     - /metrics                    Prometheus-formatted metrics
     - /v1/compress                compression-only endpoint; returns
                                   a compressed message list without
                                   generating a completion
     - /v1/* (and /v1/messages)    proxied upstream (Anthropic / OpenAI)

16.4 Request lifecycle
     1. Session lookup / creation (extract session id from headers).
     2. Mode determination (config, env, per-request header).
     3. Pipeline: token count -> semantic cache check -> content
        type detect -> transform select -> summary compression ->
        token budget enforce -> output-shape request.
     4. Forward upstream with the appropriate API key.
     5. Capture response; compute tokens_before - tokens_after.
     6. Emit telemetry; return response with savings headers.

16.5 Savings headers
     - X-CTX-Savings
     - X-CTX-Original-Tokens
     - X-CTX-Compressed-Tokens
     - X-CTX-Mode
     - X-CTX-Cache-Hit

16.6 Loopback-only and invisibility
     - The proxy binds to 127.0.0.1 by default.
     - `/v1/compress` returns 404 (not 403) to non-loopback callers,
       so the route stays invisible to scanners.
     - Binding to a non-loopback address requires an explicit
       opt-in and a warning is logged.

16.7 Fail-open
     - A compression failure degrades to passthrough; the request
       still completes.
     - Provider timeouts retry up to provider_retries with
       exponential backoff.
     - A failed retry returns the provider's error verbatim.

16.8 Two-tier integration
     - Tier 1 (raw proxy): any agent that exposes a base-URL
       override. The user sets ANTHROPIC_BASE_URL or
       OPENAI_BASE_URL and launches the agent. No CTX-specific
       code in the agent.
     - Tier 2 (`ctx wrap <agent>`): CTX starts the proxy, injects
       the correct config for the named agent, backs up the
       existing config, and launches the agent. Supported agents:
       claude, codex, cursor, aider, copilot, opencode, openclaw,
       cline, continue, goose.
     - The proxy itself is tier-1 only. Tier 2 is a setup layer.

16.9 Compression-as-a-service
     - `/v1/compress` accepts a message list and returns a
       compressed message list. It does NOT call the upstream model.
     - Used by gateway sidecars, LiteLLM-style guardrails, and the
       Python / TypeScript SDK.
     - Same pipeline, same CCR injection, same redaction boundary as
       the full proxy.

16.10 Non-goals
     - No per-agent proxy code. The proxy is agent-agnostic; only
       the wrap launcher knows about specific agents.
     - No hosted proxy service.
     - No second UI. The proxy's stats are rendered in the existing
       Mnemonic web UI (Phase 17).
     - No control-plane functionality (start/stop agents, edit
       recipes) on the proxy surface.

DELIVERABLE: A working proxy with three modes, a compression-only
             endpoint, loopback-only invisibility, savings headers,
             fail-open retries, and a two-tier integration strategy
             (raw proxy + `ctx wrap`).

================================================================================
17. PHASE 17 - SESSION OBSERVABILITY ENDPOINTS (Week 10, parallel with 16)
   (folded into Mnemonic's EXISTING UI server)
================================================================================

17.1 Principle
     - The observability surface is folded into Mnemonic's existing
       UI server. No second process, no second database.
     - The surface is read-only. It does not control the pipeline.
     - The surface complements the v1.7 Live view: the Live view is
       event-oriented and card-based; the observability endpoints
       are metric-oriented and queryable.
     - The surface also aggregates session events into a per-session
       wide event at SessionEnd, so the dashboard can render session
       cards without joining many event rows.

17.2 Endpoints (added to the existing UI server)
     - GET /stats
         live session metrics: current mode, tokens saved this
         session, cache hits, cache invalidations, judgment
         decisions, pre-filter pins/deads, traffic memories.
     - GET /stats-history
         durable savings history with hourly / daily / weekly /
         monthly rollups. Sourced from ctx_stats_history, with a
         JSON snapshot written under the project directory.
     - GET /metrics
         Prometheus-formatted metrics: request counts, token totals,
         latency summaries, per-upstream counts, cache bust
         counters, judgment latency, pre-filter throughput,
         prefix-guard blocks, learn-run frequency.
     - GET /api/session-summaries
         paginated per-session wide events (one row per session)
         from ctx_session_summaries.

17.3 Per-session wide event
     - Aggregated at SessionEnd by the session_summary module and
       the session_summary_aggregate worker job.
     - Captures: session id, agent, duration, turns, tokens saved,
       judgment keeps/truncs, pre-filter pins/deads, cache
       invalidations, shaping applied, effort routed count, traffic
       memories extracted.
     - Stored as one row in ctx_session_summaries, plus the full
       JSON in summary_json.
     - The aggregation runs within the SessionEnd hook budget; it
       does not block the agent.

17.4 History snapshot
     - Written to .skillgrid/ctx/stats_history.json under the
       project directory, mirroring Mnemonic's file-first
       philosophy.
     - Kept in sync with ctx_stats_history.
     - Survives a re-index (the file is part of the project).

17.5 Run mode as an observability signal
     - The current proxy mode is exposed in /stats and in the Live
       view header.
     - Mode changes are emitted as proxy-mode-changed events and
       recorded in ctx_proxy_mode_history.
     - Operators can correlate a change in savings or latency with
       a mode change.

17.6 Non-goals
     - No control plane. The endpoints are read-only.
     - No second UI. The endpoints are consumed by the existing
       Mnemonic UI (Live view cards) and by external scrapers.
     - No new authentication model. The endpoints inherit whatever
       the existing UI server uses.

DELIVERABLE: The existing Mnemonic UI server exposes /stats,
             /stats-history, /metrics, and /api/session-summaries,
             backed by ctx_stats_history and ctx_session_summaries,
             with a JSON snapshot under the project directory. Run
             mode is visible as an observability signal.

================================================================================
18. PHASE 18 - REAL-TIME TRAFFIC LEARNING (Week 10, parallel with 16/17)
================================================================================

18.1 Principle
     - Mnemonic already records session events. v1.11 turns that log
       into an active source of reusable context via a zero-latency,
       RULE-BASED extractor.
     - The extractor makes ZERO LLM calls. It is a pure function over
       the session-event stream.
     - Extraction happens DURING the session, not only in a periodic
       offline batch. This complements the v1.10 `ctx learn` loop.
     - Extracted memories are candidate rules; they are subject to
       the same audit and promotion gate as offline-mined rules.

18.2 Categories
     - error_recovery   a failed tool call followed by a successful
                        retry on the same target. Captures the
                        working command or path.
     - environment      stable facts: working commands, paths,
                        tool versions, environment variables that
                        appear repeatedly.
     - preference       repeated usage patterns and user corrections
                        (e.g., the user repeatedly removes a
                        particular file from context).
     - architecture     file structure and coding conventions
                        inferred from repeated reads and edits.

18.3 Rules
     - Rules live in traffic/rules.yaml and are human-editable.
     - Each rule declares: category, pattern, confidence weight,
       and a short description.
     - The extractor applies rules in order; the first matching rule
       wins (deterministic).
     - A memory below min_confidence is not stored.

18.4 Storage and bounds
     - Memories are stored in ctx_traffic_memories.
     - max_memories caps the store (default 500); the oldest
       low-confidence memories are evicted first.
     - Each memory records its evidence (the events that produced it)
       in evidence_json so a human can audit it.

18.5 Composition with ctx learn
     - Real-time memories are a new input to the offline mining loop
       (v1.10 Phase 15).
     - A high-confidence real-time memory may seed a proposed rule;
       promotion still requires the v1.10 gate
       (promotion_threshold_sessions + human CLI action).
     - The two layers do not contradict each other: real-time
       extraction is per-session; offline mining is cross-session.

18.6 Determinism
     - Extraction is a pure function of the session-event stream and
       the rules file. Same input, same memories, same order.
     - A rule change invalidates future extractions but does not
       rewrite stored memories.

18.7 Non-goals
     - No LLM calls. The extractor is rule-based.
     - No fine-tuning. Memories are text.
     - No cross-project memory federation (future work).
     - No automatic promotion. Promotion is a human CLI action.

DELIVERABLE: A working real-time traffic learner that extracts
             memories in four categories from the session-event
             stream, stores them in ctx_traffic_memories, bounds the
             store, records evidence, and feeds the offline learn
             loop.

================================================================================
19. MILESTONES
================================================================================

M1-M20   (carried forward from v1.9)
M21      PrefixGuard + live-zone / frozen-prefix split + statistical
         pre-filter + auto-router.                                     (end W10)
M22      Output shaping (verbosity + effort routing) + control
         holdout in benchmarks.                                       (end W10)
M23      Failure mining (proposed / applied / revoked lifecycle) +
         learn report + CLI.                                          (end W10)
M24      Benchmarks pass (output shaping + learn + live-zone +
         holdout), v1.10.0.                                           (end W10)
M25      Proxy surface (three modes, /v1/compress, savings headers,
         loopback invisibility, fail-open, two-tier integration).     (end W11)
M26      Observability endpoints folded into Mnemonic's existing
         UI server (/stats, /stats-history, /metrics,
         /api/session-summaries, per-session wide event, run mode
         as a signal).                                                (end W11)
M27      Real-time traffic learner (four categories, bounded store,
         evidence, composition with ctx learn).                       (end W11)
M28      Benchmarks pass (proxy + observability + traffic),
         hardened, documented v1.11.0.                                (end W11)

================================================================================
20. RISKS & MITIGATIONS
================================================================================

R1-R65 carried forward from v1.10.

R66 The proxy leaks secrets upstream.
    -> The proxy inherits the redaction boundary (6.12).
    -> It binds to loopback by default.
    -> It never forwards unredacted content upstream.
    -> ctx proxy status reports the bind address.

R67 A non-loopback scanner discovers /v1/compress.
    -> The endpoint returns 404 (not 403) to non-loopback callers.
    -> Binding to a non-loopback address requires explicit opt-in
       and logs a warning.

R68 The proxy in token mode invalidates the provider prefix cache and
    costs more than it saves.
    -> Default mode is cache (freeze prefix, compress tail).
    -> Mode changes are recorded and emitted.
    -> Benchmark 11.20 measures both savings and cache invalidations.
    -> "The win is headroom, not money" is documented in PROXY.md.

R69 Traffic learning extracts a spurious memory that misleads the
    agent.
    -> Extraction is rule-based with a confidence threshold.
    -> Memories are candidate rules; they are not applied until a
       human promotes them via ctx learn apply.
    -> Each memory stores its evidence for audit.
    -> ctx traffic report shows precision.

R70 Session summary aggregation blocks the SessionEnd hook.
    -> Aggregation runs in the worker; the hook only enqueues.
    -> Benchmark 11.21 verifies the hook budget is respected.

R71 stats-history grows without bound.
    -> Rollups compact raw savings into hourly / daily / weekly /
       monthly buckets.
    -> Raw rows below the finest granularity are pruned after the
       monthly rollup.
    -> The JSON snapshot is rewritten atomically.

R72 The proxy becomes a control plane.
    -> The proxy has no start/stop/edit surface.
    -> Mode switching is the only runtime control, and it is
       explicitly a configuration action, not a control plane.
    -> PR review checklist enforces this boundary.

R73 The two-tier integration confuses users (raw proxy vs wrap).
    -> PROXY.md documents the boundary explicitly.
    -> `ctx proxy status` prints the tier that is active.
    -> `ctx wrap` never modifies agent code, only config.

R74 Traffic memories duplicate ctx learn rules.
    -> The two layers are explicitly complementary: real-time
       per-session vs offline cross-session.
    -> Real-time memories seed proposed rules; they are not rules
       themselves.
    -> Deduplication runs on promote.

R75 Observability endpoints leak session content.
    -> Endpoints are metric-oriented; they return counts and
       aggregates, not raw content.
    -> The session summary stores counts and a summary JSON, not
       message content.
    -> The endpoints inherit the existing UI server's auth model.
    -> ctx doctor --ui reports the bind address and auth mode.

================================================================================
21. OUT OF SCOPE (v1.11)
================================================================================

All v1.10 out-of-scope items remain, plus:

- Per-agent proxy code. The proxy is agent-agnostic; only the wrap
  launcher knows about specific agents.
- A hosted proxy service.
- A control plane on the proxy surface.
- A second UI for the proxy stats. The stats render in the existing
  Mnemonic UI.
- A second database for proxy savings or stats history. The data
  lives in ctx_* tables and a JSON snapshot.
- LLM calls inside the traffic learner. The extractor is rule-based.
- Automatic promotion of traffic memories. Promotion is a human CLI
  action.
- Cross-project memory federation. Future work.
- Rewriting stored session events to apply a traffic memory.
  PrefixGuard.
- A separate observability process. The endpoints are folded into
  the existing UI server.

================================================================================
22. SUCCESS CRITERIA
================================================================================

- All v1.10 success criteria hold.
- The proxy works with at least 5 agents via base-URL override and
  at least 3 agents via `ctx wrap`.
- The proxy passes HTTP/1.1, HTTP/2, SSE, and WebSocket traffic
  intact.
- Savings headers are present and accurate on every response.
- `/v1/compress` returns a compressed message list without
  generating a completion.
- A non-loopback caller receives 404 on `/v1/compress`.
- A compression error degrades to passthrough; the request still
  completes.
- A provider timeout retries with the configured backoff.
- Mode switching works at startup, at runtime, and via
  CTX_PROXY_MODE.
- `/stats` returns live session metrics.
- `/stats-history` returns rollups and survives a restart.
- `/metrics` is Prometheus-parseable.
- The per-session wide event is aggregated at SessionEnd without
  blocking the agent.
- The history snapshot JSON matches the DB and survives a re-index.
- The current proxy mode is visible in /stats and in the Live view
  header.
- The traffic learner extracts memories in all four categories with
  zero LLM calls.
- Traffic memories are bounded by max_memories and store their
  evidence.
- A traffic memory is never applied as a rule until a human promotes
  it via `ctx learn apply`.
- The two layers of learning (real-time + offline) do not
  contradict each other.
- Adding a new proxy mode requires < 1 day.
- Adding a new traffic rule requires < 1 day.
- Adding a new observability endpoint requires < 1 day.

================================================================================
23. DIFF FROM v1.10
================================================================================

- New Phase 16: Proxy Surface + Compression-as-a-Service. Three
  modes (token / cache / passthrough), /v1/compress, savings
  headers, loopback-only invisibility (404 not 403), fail-open
  retries, and a two-tier integration strategy (raw proxy + `ctx
  wrap`).
- New Phase 17: Session Observability Endpoints, folded into
  Mnemonic's existing UI server. /stats, /stats-history, /metrics,
  /api/session-summaries. Per-session wide event aggregated at
  SessionEnd. Run mode as an observability signal. History snapshot
  JSON under the project directory.
- New Phase 18: Real-Time Traffic Learning. Rule-based, zero LLM
  calls. Four categories: error_recovery, environment, preference,
  architecture. Bounded store, evidence per memory, composition
  with the offline `ctx learn` loop.
- New config blocks: proxy, wrap, observability, traffic_learning.
- New CLI: ctx proxy, ctx proxy status, ctx proxy mode, ctx wrap,
  ctx stats, ctx stats --history, ctx metrics, ctx traffic memories,
  ctx traffic report, ctx traffic rules reload.
- New event types: proxy-request, proxy-savings, proxy-mode-changed,
  session-summary, traffic-memory-extracted, stats-history-rotated,
  metrics-scraped.
- New card types: ProxyRequestCard, ProxySavingsCard, ProxyModeCard,
  SessionSummaryCard, TrafficMemoryCard. Live view header now shows
  the current proxy mode.
- New tables: ctx_proxy_sessions, ctx_proxy_savings,
  ctx_proxy_mode_history, ctx_session_summaries, ctx_stats_history,
  ctx_traffic_memories.
- New docs: PROXY.md, OBSERVABILITY.md, TRAFFIC_LEARNING.md;
  UI_EVENTS.md updated.
- New risks R66-R75.
- New milestones M25-M28.
- Out-of-scope now explicitly excludes per-agent proxy code, a
  hosted proxy, a control plane on the proxy, a second UI, a second
  database, LLM calls in the traffic learner, automatic promotion
  of memories, cross-project federation, stored-message rewriting,
  and a separate observability process.
- Success criteria now include proxy compatibility, protocol
  passthrough, savings headers, /v1/compress, loopback invisibility,
  fail-open, mode switching, observability endpoint correctness,
  session summary aggregation, history snapshot durability, run
  mode visibility, traffic learner precision, bounded store, human
  promotion gate, and dual-layer learning consistency.

================================================================================
END OF PLAN v1.11
================================================================================
