# 09 — claude-os Deep Dive (what we steal)

Companion to `08-second-brain-roadmap.md`. claude-os (`brobertsaz/claude-os`, 417★, v2.2.0, ~7.8k LOC core) is the closest same-shape product: "Give Your AI a Memory", 100% local, SQLite store, MCP server, dashboard, self-learning. This artifact is a **verified** deep-dive — every load-bearing claim is checked against source (cite:file:line), not just read off docs.

Repo root for citations: `~/git/ai-test/claude-os`.

## TL;DR — we beat them on the two things they brag about

| Capability | claude-os (verified) | mnemonic (verified) |
|---|---|---|
| Vector search | **Brute-force Python** — `np.frombuffer` + `linalg.norm` + `dot`, no extension (`sqlite_manager.py:394-402`). "sqlite-vec" is a docstring, not a vtab. | **Real vec0** — `CREATE VIRTUAL TABLE vec_symbols/vec_chunks USING vec0(float[768])` (`042_vec0_tables.sql`), `modernc.org/sqlite/vec` cgo-free, rowid-keyed to parent PK. |
| Keyword / FTS | **None.** Zero `FTS5`/`MATCH`/`tokenize` in the codebase. "Hybrid" is vector-only (`rag_engine.py:242` "just use vector for now since BM25 requires documents"). | **Real FTS5** — `observations_fts` (porter), `chunks_fts` (trigram), `web_cache_fts` (porter); `fts_drift: 0`, integrity OK (mem_doctor). |
| Migrations | No runner. `IF NOT EXISTS` DDL run manually; `lifecycle_log` duplicated between `schema.sqlite` and migration 005; no `user_version`. | Real ordered migrations `001→042` with idempotency + upgrade tests. |

**Roadmap impact:** roadmap item **F (durable sqlite-vec) is already DONE** (migration 042) — it was misfiled as P2. The "P2 durability" pillar collapses; only **G (human browse view)** remains there.

## What's genuinely stealable (ranked by "brain feel" per unit effort)

### S1 — NL capture via skill-file trigger phrases (P0, zero new infra)
Their "remember this:" magic is **pure prompt-engineering**, no services. `templates/skills/memory/SKILL.md` frontmatter:
```
description: "Save and recall information across sessions. Use when you hear
'remember this', 'save to memory', 'add to your knowledge', or similar."
```
Claude Code auto-loads the skill on the trigger phrase → body says extract → auto-title → auto-detect category from a 7-row table → save, "No questions. No ceremony. Just save it."

**Action:** TASK-020 should ship as a **skill** with trigger phrases (`remember this`, "don't forget that", "we decided to…"), mapping intent → `mem_save` `type` (decision/bugfix/pattern/…/correction), not a regex engine. The agent is the classifier.

### S2 — Lifecycle patterns (P1-D, the "brain feel" workhorse)
From `knowledge_lifecycle.py` (652 LOC) — all map directly onto our observation store:
- **Two-phase `lifecycle_log` audit table**: `pending→completed/failed`, `input_doc_ids`/`output_doc_ids` JSON arrays, `details` JSON blob, `completed_at` auto-set on terminal status. Every mutating op logs before-and-after IDs.
- **Soft-archive via metadata flag** (`archived`, `archived_at`, `archive_reason`) + `exclude_archived` query path. Cheap, reversible, one table.
- **Union-find clustering** (path-halving) over pairwise cosine for dedup; **duplicate-density** = `pairs/(n(n-1)/2)` doubles as a health metric.
- **`consolidated_from: [doc_ids]` provenance** on the merge output (LLM summarization merge, temp 0.3).
- **Rule-based health recommendations**: dedup (>5 similar pairs = high), cleanup (archived >30% = low), stale (>50% older than 90d = medium), embedding-coverage (any doc w/o vector = medium).
- **Inline `_health_warnings`** appended to search results: lazy, 24h file cache, HIGH/CRITICAL only, **never breaks search** (try/except → `[]`).

**Action:** TASK-021 (consolidate + archive + health) should adopt the audit-log table, union-find dedup, provenance, and inline staleness warning.

### S3 — Confidence-gated anti-hallucination extraction (P3-H passive learning)
Their post-session LLM prompt (`insight_extractor.py`) is the model to copy:
> "ONLY extract insights that are EXPLICITLY mentioned… Do NOT invent, assume, or hallucinate… 0.9+ only if explicitly stated, 0.7–0.8 if implied… If you cannot point to specific transcript text supporting the insight, DO NOT include it."

temp 0.3, `format=json`, post-filter `confidence >= 0.7`, four types (decision/pattern/solution/blocker). Session compressed to ~8000 tokens (messages truncated to 1000 chars, first 20 tool calls, first 10 files) before the call.

**Action:** reuse verbatim for our session-end learnings extraction (roadmap H). Cheap, zero-hallucination, directly reuses our `mem_save`.

### S4 — MCP response shape (cheap, applies to every tool)
- **Errors-as-values, never throw to transport**: `{"error": "KB 'x' not found", "answer": "", "sources": []}`.
- **Actionable error text with the next step**: "Cannot connect to Claude OS API. Is the server running? Start with: ./start_all_services.sh" — the agent can self-heal.
- **`_`-prefixed meta fields** (`_timing`, `_health_warnings`, `_source_kb`) the LLM can ignore; never collide with payload.

### S5 — Installer + service patterns (when we build `mnemonic serve` DX)
- **Symlink, don't copy**: commands/skills live in repo, `~/.claude` holds symlinks → updates propagate, uninstall = `rm`.
- **`wait_for_port`** (`nc -z`, 30 tries, dot output) over fixed `sleep`; **kill-by-port** before start → idempotent restarts.
- **Server is the source of truth for health**: `GET /api/services/status` self-introspects siblings (pgrep + port probes + HTTP probe for Ollama) → `healthy/degraded/critical`; dashboard is a thin 5s poller with a refresh toggle.
- **Backup = plain timestamped dir + MANIFEST.txt**, zero deps, human-readable, `restore <ts>`.

### S6 — Lean session state (maps to our session protocol)
4-field `claude-os-state.json` (gitignored): `last_task`, `last_branch`, `stopped_at`, `one_liner`. START = read state → show `one_liner` → search memories with last-task context. END = git diff summary → offer save → write state. Philosophy: **lean state, agent-driven ceremony** (they cut CLAUDE.md from 351→128 lines).

### S7 — Filesystem-as-event-bus (skillgrid dashboard future)
Their kanban works by the implementer marking `- [x]` in `tasks.md` → watchdog (2s debounce, SHA256 change-detect) → re-parse → SQLite. **No API call from the agent** — the filesystem IS the event bus. Our `state.yaml` + `.backlog/` could drive a dashboard the same way.

## What NOT to copy

- **Their "hooks" are NOT Claude Code native hooks.** No `PreToolUse`/`PostToolUse`/`Stop` wiring — "hooks" are folder watchers + declarative `hooks.json` + NL trigger phrases. A parallel system. Our tool-call capture (roadmap A / 07 Monitoring plan) is already more native — don't regress to folder-watching.
- **Capture via `curl` to REST** instead of the MCP tool (port 8051 hardcoded into prompt templates) — divergent capture paths. We keep `mem_save` as the single path.
- **`kill -9` everywhere**, no PID files, no graceful SIGTERM.
- **Hardcoded `API_BASE`** in `lib/api.ts` + field-name drift client↔server (half-finished UI).
- **Version pins that bite**: `tree-sitter<0.21`, `bcrypt<5`, plus Postgres leftovers in `run_tests.sh`/`entrypoint.sh` while the product is SQLite.
- **No MCP auth** (fine for localhost-only — we're the same).

## Their code index vs ours (confirmed we're ahead)
claude-os: `tree_sitter_indexer.py` → 19-language symbol tags → `networkx.MultiDiGraph` → `pagerank` → token-budget repo map (binary-search fit, ~30s/10k files). Two-phase: structural (instant, all files) + semantic (Ollama, top-20%-by-PageRank). **We have a real dependency graph with resolved edges and blast-radius tiers** — theirs is PageRank over reference edges with a token-budget display. We also have a real FTS keyword leg; they have none.

## MCP tool surface (reference, for naming)
stdio server `code-forge` (kept for back-compat — tool names persist because prompts/stored-knowledge reference them) proxies a FastAPI REST API (`localhost:8051`) that holds SQLite in-process. ~25 tools: KB CRUD, `search_knowledge_base` (top_k + use_hybrid + use_agentic), `search_all_knowledge_bases` (kb_filter prefix), `index_structural`/`index_semantic` (background `job_id`), `extract_session_insights` (LLM over `~/.claude/projects/*.jsonl`), skill management, `kb_lifecycle_{health,dedup,consolidate,archive}` (`action` enum to multiplex; `dry_run` on every destructive op). Notable: **`action`-enum param** to multiplex one tool (dedup scan/merge; archive/restore/list/stale) — fewer tools, documented in the description.
