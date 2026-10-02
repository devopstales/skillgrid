# 07 — Mnemonic: Planned Functions

Planning list of the target function areas for `skillgrid-mnemonic`. Status tags reflect `.backlog` and `.skillgrid/specs/` as of 2026-09-24.

- **Memory**
  - **Observations** — save/search/recall of structured observations; the existing `mem_*` surface. *Status: built.*
  - **Decisions** — governed decision observations the agent posts; user answers in the serve dashboard; agent reads back via `mem_search`. `GET /mnemonic/decisions`, `POST .../answer`, `memory.UpdateContent`. *Status: planned — Visual Companion epic (TASK-006, 7 tickets, needs-triage).*
- **Code index**
  - **Semantic search** — embedding-based code search over symbols/chunks. *Status: built (`code_semantic_search`); degraded to none when no embedder is configured.*
  - **Vector db** — persistent vector store for embeddings (today in-memory / sqlite-vec seam). *Status: partially built — no dedicated durable vector store yet. **Strong dependency** for session-inject's locked hybrid selection. Prototype 2026-09-24 (ADR-0009) confirmed the path: **G (`modernc.org/sqlite/vec`)** is the default scale-up (cgo-free, same driver, mature, but in-SQL top-K is ~6.9 s at 100K / ~1.4 s at 20K — durable path, not hot path). **viant** (cover-tree ANN) is deferred — packaging bug + query deadlock. In-memory cosine (`hybrid/vectorcache.go`) remains the hot path (~40 ms at 100K). See ADR-0006 + ADR-0009.*
- **Monitoring**
  - **Session tracking** — session lifecycle, summaries, event stream + commit range. *Status: built (`mem_session_*`, `session_changes`).*
  - **Track every tool call** — per-tool-call tracking/audit. *Status: planned — designed 2026-09-24 (see Monitoring plan below). Capture mechanism exists (`memory.hookPostToolUse`, `session_events` schema) but is not triggered: no auto-firing hook, hooks default off. Locked decisions: default-on observe-mode, plain SQLite (no hash-chain), `standard` logging level, 90-day retention, `<private>` tags via agent output + tool-name allowlist.*
- **Web cache**
  - **Cache web search results** — store/retrieve research snapshots before remote MCPs. *Status: built (`web_cache_*`).*
- **Session handoff**
  - **Inject relevant data back to next session** — see the Session-inject plan below. *Status: planned — design grounded in the reference projects listed here.*

## Session-inject plan

Goal: when a new session starts (or the current one resumes), select the most relevant slice of prior-session state and inject it into context — so the agent picks up where it left off without dumping everything.

### Reference projects (source of the design)

| Project | What it does | What we take from it |
|---------|--------------|----------------------|
| `context-hub/generator` (CTX) | Collects code from files, git diffs, GitHub repos, URLs into structured markdown context documents via YAML config. | The *selective source* idea: context is built from declared sources, not a blanket dump. For session-inject, the "sources" are prior-session events, observations, and commit ranges. |
| `foldwork-dev/mcp-injector` | Pre-indexes the codebase into a local SQLite catalog; AST body folding + canonical determinism for byte-identical, cache-friendly payloads (41-89% token reduction). | *Determinism + compression* as first-class properties of the injected payload: the injection should be stable across calls (KV-cache friendly) and compressed (signatures over bodies) rather than raw. |
| `headroomlabs-ai/headroom` | Compresses tool outputs, logs, RAG chunks, and conversation history before they reach the LLM; CCR (reversible — originals cached locally, retrieved on demand); `headroom learn` mines failed sessions. | *Reversible compression* (inject a compressed view, keep the original retrievable) and *learning from failures* (mine prior sessions for corrections). |
| `mksglu/context-mode` | Tracks every file edit, git op, task, error, and user decision in SQLite; on compaction, indexes events into FTS5 and retrieves only the relevant slice via BM25; `--continue` or fresh session. | *The core pattern*: don't dump back, *index and retrieve*. Session-inject = track events → FTS5 index → BM25-retrieve the relevant slice at session start. Also the `--continue` semantics: explicit resume vs. clean slate. |
| `thedotmack/claude-mem` | Captures tool usage, generates semantic summaries, makes them available to future sessions; progressive disclosure (layered retrieval with token-cost visibility); privacy tags to exclude sensitive content. | *Progressive disclosure* (L1 summary → L2 details on demand, with token cost per layer) and *privacy control* (exclude sensitive content from the injected payload). |

### Design (synthesized)

> **LOCKED** (2026-09-24): (a) two-layer injection mechanism — auto-prepend slim L1 on resume + on-demand `mem_inject_session` tool; (b) privacy — tag-by-default (secrets/full local paths/`private`-tagged excluded; project-relative paths/SHAs/tool names/task IDs included); (c) scope — project-scoped default, cross-project only via explicit `all_projects: true`; (d) selection — hybrid (BM25 + semantic, RRF-fused) by default, vector leg degrades to BM25-only when no embedder is active. See `05-locked-constraints.md`. Compression depth is NOT locked — validate in first prototype.

1. **Capture** — the monitoring layer (track every tool call + session events) records every tool call, edit, git op, error, and decision to a per-session event store (SQLite). Reuses the existing `session_changes` event stream.
2. **Index** — at session end (or continuously), compress the events into a structured summary and index it into FTS5 (the existing memory FTS5 surface). The summary is deterministic (stable ordering, byte-identical across runs) so it is KV-cache friendly.
3. **Select** — at the next session start (or on explicit resume), retrieve the relevant slice: BM25/semantic search over prior-session summaries + observations, scoped to the current project/topic. The selection is *progressive* — L1 summary first, L2 full details only on demand.
4. **Inject** — inject the selected slice as a compact, compressed context block (signatures/summaries, not raw bodies), with each item carrying a token-cost estimate and a retrievable-original reference (CCR-style). Sensitive content is excluded per privacy tags.
5. **Resume vs. fresh** — explicit resume (`--continue`-equivalent) injects the prior slice; a fresh session starts clean (no prior-session injection).

### Open questions

- **Compression depth**: signature-folding (mcp-injector style) vs. AI-generated semantic summaries (claude-mem style) vs. reversible CCR (headroom style). *Proposal: tiered by content type — signature-fold for code references, semantic summary for decisions/errors, CCR for originals. NOT locked — validate in first prototype.*

### Mapping to existing backlog

- The "track every tool call" leaf under Monitoring is the capture layer this plan depends on — it must land first. See the Monitoring plan below.

## Monitoring plan (track every tool call)

Goal: capture every agent tool call into a queryable, retained, privacy-filtered audit trail, so session-inject's capture layer has a full event stream and the agent/user can replay or query what happened.

### Reference projects (source of the design)

| Project | What it does | What we take from it |
|---------|--------------|----------------------|
| `safedep/gryph` | Hooks pre/post-tool events; logs every action to local SQLite; `gryph query --action --file --since --count`, JSONL export, logging levels (minimal/standard/full), 90-day retention. | The *audit-query surface*: `mem_query_events` (action/file/since/session/count filters, sensitive excluded by default), JSONL export, `standard` logging level, retention policy. |
| `PrismorSec/prismor` | One canonical event shape across all surfaces; observe-mode (log, never block) by default; tamper-evident hash-chained trail; secret redaction at the surface. | The *capture-trigger pattern*: the hook fires per tool call (agent never knows), observe-mode only (capture never blocks), canonical event normalization, redaction at the edge. (We take the pattern, not the hash-chain — plain SQLite is the decision.) |
| `thedotmack/claude-mem` | 5 lifecycle hooks; PostToolUse on every tool; `<private>` tags stripped at the hook layer (edge processing) before storage; 3-layer progressive disclosure (search → timeline → get_observations). | The *privacy-at-the-edge* pattern: `<private>` tags stripped before the event is stored (content never persisted), and the 3-layer retrieval workflow that session-inject reuses. |

### Locked decisions (2026-09-24)

> **LOCKED:** (a) **default-on** — `mnemonic.hooks.enabled` flips to `true` by default (observe-mode is safe: writes rows, never blocks); configurable to `false`. (b) **plain SQLite** — no hash-chain / Ed25519 signing; append-only `session_events` is sufficient for a local audit trail. (c) **`standard` logging level** — action, path, command, exit code, truncated output, sensitive flag (the current `hookPostToolUse` capture); no `full` (file diffs) level. (d) **90-day retention** — auto-delete `session_events` older than 90 days (configurable via `mnemonic.retention_days`). (e) **`<private>` tags — both** — the agent marks tool output `<private>` by including the tag (existing `stripPrivateTags`), AND a config allowlist of tool names whose output is always private.

### Design (synthesized)

1. **Trigger (plugin-side, observe-mode)** — extend `plugins/opencode/mnemonic.ts` `tool.execute.after` (currently Task-only) to capture every non-Task tool call: normalize to a canonical event `{ts, session_id, agent, type, tool_name, path, command, result_status, content_hash}`, strip `<private>` tags (existing `stripPrivateTags`), strip always-private tool output (allowlist), then `req("POST", "/sessions/:id/tool-calls")` (fail-open, 3s timeout, reuse the existing `req()` client). Fire-and-forget — capture never blocks the tool call.
2. **HTTP route** — `POST /sessions/{id}/tool-calls` on the serve dashboard (`requireWriteAuth`), calls the existing `memory.hookPostToolUse` (already complete: maps tool→action_type, redacts sensitive paths, bumps counters, one transaction). The route is a thin adapter over the existing writer.
3. **Audit query surface (Gryph-style)** — new MCP tool `mem_query_events`: query `session_events` with `action`, `file` (glob), `since`/`until`, `session`, `count` (count-only), `sensitive` (include, excluded by default). JSONL export via `mem_export_events` (or `--export` on query). Sensitive events excluded by default.
4. **Retention** — a cleanup pass (on serve startup, or a periodic goroutine) deletes `session_events` older than `mnemonic.retention_days` (default 90). Configurable.
5. **Session-inject integration** — the session-inject `DistillSummary` (blueprint Task 1) now reads from the full per-tool-call event stream (not just lifecycle events). The 3-layer retrieval workflow: `mem_inject_session` (L1 index) → `session_changes` (L2 timeline) → `mem_get_observation` (L3 details).

### Mapping to existing backlog

- The capture mechanism (`hookPostToolUse`, `session_events` schema, `isSensitivePath`/`redactPreview`, counters) is **already built** — this plan wires the trigger + adds the query surface + retention.
- The `<private>` tag stripper (`stripPrivateTags`, `mnemonic.ts:79`) is **already built** — this plan extends its application from Task-output-only to every tool call + the always-private tool allowlist.
- The session-inject blueprint (`.skillgrid/specs/2026-09-24-mnemonic-session-inject/blueprint.md`) is the downstream consumer of this capture layer.
- The 006-structured-session-handoff change (done) provides the PROGRESS/KNOWLEDGE/NEXT_PROMPT structure; session-inject extends it with relevance-based retrieval instead of a fixed handoff document.
- The vector db leaf is a **strong** dependency for the locked hybrid selection model (BM25-only is the degraded state when no embedder is active, not the target).
