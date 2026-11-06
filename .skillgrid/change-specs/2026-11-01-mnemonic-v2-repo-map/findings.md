# Findings — mnemonic-v2 repo-map

> Consolidated evidence for this change. The blueprint + the briefing cite this file for
> every design decision that rests on a reference fact. Research mode: **the 7-reference
> Plan-mode fold** — a Plan-mode conversation walked seven open-source repo-map
> implementations, folded each in, and settled the design. The digest below is the
> per-reference record + the "what we do differently" section.

## The 7-reference fold (the settled design)

The design was settled by walking seven open-source references and folding each in. The
fold is **binding** (the settled intent, not a redesign):

| Reference | What it contributes | Our fold | URL |
|---|---|---|---|
| **aider** (the 2023 original) | The map shows **signature lines** (parameters + types), not bare names; the token budget is **dynamic** (expanded on cold start when no files are in context); the ranking is **chat-aware** (weighted toward files the conversation references); the map is **re-sent/re-ranked per turn**, not just at session start. | Tier 1 emits the `symbols.signature` line (the full `func`/`def`/`function` line); the auto-inject budget is `inject_tokens` (1024) **expanded ~2× on cold start**; the `context: []` param is a **PageRank prior** (the aider fold); the session-start hook **re-ranks per turn** (<100ms). | https://aider.chat/docs/repomap.html |
| **nekocode/agent-codemap** | The **zero-setup floor** — a symbol tree the agent can use when mnemonic/code-index is unavailable. Its AGENTS.md usage is a "just run the command" instruction. | The AGENTS.md instruction is a **lean trigger-table** (When → Command + a one-line trust rule) inside the existing `skillgrid:start` block — the fallback for harnesses without the MCP hook. The zero-setup floor is the `mnemonic repo map` CLI (one command). | https://github.com/nekocode/agent-codemap |
| **Twigpine/openclaude** | **PageRank over an IDF-weighted reference graph** (edges weighted by reference count × IDF of the symbol name — common names like `get`/`set` count less); **token-budgeted rendering** (walk ranked files top-down, stop at the budget; 1024-token auto-inject default, 2048 command default); **auto-injection at session start** (opt-in flag) + on-demand tool; **`--focus` as a ranking boost** (not a filter); disk cache keyed by path+mtime+size (**we do NOT need a second cache — the code-index IS the cache**). | The ranking engine is PageRank (damping 0.85, 100 iters, ε=1e-6, deterministic) over the file reference graph, edges weighted by refCount × IDF(name). The output bound is the token budget (primary, `--tokens` default 2048) + `limit` (hard row cap). Auto-injection is an opt-in config flag + the on-demand `repo_map` tool. `--focus` is a ranking boost (path) or neighborhood (symbol), never a filter. No second cache (the code-index is the cache). | https://github.com/Twigpine/openclaude (docs/repo-map.md) |
| **cyanheads/repo-map** | The **per-file field contract** for the emitted doc: Description, **Developer Consideration** ("the one thing a contributor is most likely to get wrong"), Maintenance Flag (Stable/Volatile/Generated/Unknown), Architectural Role (Entrypoint/Service/API Route/Utility/Test/Configuration/…), Critical Dependencies (with one-line reasons). LLM-generated + cached in SQLite. | The `--emit` doc carries the **five per-file fields** + a header stamp. The difference: the Developer Consideration is **not LLM-generated** — it is a **saved memory observation** (FTS-matched to path/package). The other fields are derived deterministically from the graph (`edges` in/out-degree + kind → Role; top-N by confidence × fan-in → Dependencies; `content_hash` churn + observation type mix → Maintenance). No LLM in the first cut (the two LLM doors are deferred to v2.1). | https://github.com/cyanheads/repo-map |
| **mikiships/agentmd** | **Minimal-mode discipline** (verbose context files REDUCE task success ~20%; highest-value = exact commands + where things are; lowest-value = generic prose); **drift detection** as a first-class concept (exit 0/1, CI-able); the **GENERATE → MEASURE** loop. | The AGENTS.md trigger-table is a compact table (no prose paragraph) — the minimal-mode discipline. `--drift` is a first-class, CI-able check (exit 0 fresh, exit 1 stale, the stamp + content-hash comparison). The map is the GENERATE half; the coverage contract + the hub flag are the MEASURE half. | https://github.com/mikiships/agentmd |
| **SarthakMishra/codemap** | **Token-optimized docs as the stated goal**; the **toolkit boundary** — the map is the map, NOT the suite (no RAG chat, no commits, no PRs, no second vector index). | The token budget is the primary output bound (the stated goal). The boundary: the map **orients**; the agent **drills** (`code_search`/`code_read`/`mem_lookup` are separate tools — the retrieval-as-reasoning contract, code-index spec §1.1). The map is the FIRST verb in the composition, not the whole composition. RAG chat, commits, PRs are OUT of scope. No second vector index (it reads the code-index's existing vector store, D-CI8). | https://github.com/SarthakMishra/codemap |
| **JordanCoin/codemap** (the strongest, Go, "structural ground truth for coding agents") | The **coverage contract** — every dependency answer reports `status: complete\|partial\|unavailable` + `sources[]` (named, per-language) + `issues[]` (per-reference); "a partial graph never reads as a complete one, so 'nothing imports this' and 'I couldn't tell' are different answers"; a timed-out/failed scan returns an **empty result with provenance**, not a silent empty graph and not a hard error; the JSON is **versioned** (`schema_version`). **Hub-file warnings** — "⚠️ HUB FILE: imported by N files" flagged in the orientation view (not just blast-radius), threshold 3+ importers. The **context protocol** — one command returns a `ContextEnvelope` (project metadata + dependency-graph evidence + intent classification + working set + matched skills + handoff) for the session-start injection. **Real-rules resolution** — imports resolved through each ecosystem's own rules, not string matching (our equivalent = confidence-labeled edges). | **Every `repo_map` answer** is a versioned envelope (`schema_version: repo-map/v1`) carrying `coverage: {status, sources[], issues[]}`; `sources[]` names the **per-language extraction tier** (D-TIER: `tree-sitter-rich`=authoritative, `tree-sitter-map`=mixed, `regex`=fallback). A broken leg degrades OUT + is labeled (T-CI-2 posture). A file with fan-in ≥ `hub_threshold` (default 3) carries a `⚠️ HUB` tag + count in the orientation view. The session-start hook returns ONE `ContextEnvelope` (map + coverage + hubs + memory summary + intent prior + budget). Real-rules resolution = the code-index's confidence-labeled edges (EXTRACTED/INFERRED/AMBIGUOUS/LSP_RESOLVED). | https://github.com/JordanCoin/codemap |

## Per-reference key facts

### aider (repomap)

- **Signature lines:** the map shows the function/method **signature** (parameters +
  return type), not a bare name — the agent sees the interface, not just the label.
- **Dynamic token budget:** the map budget is **expanded on cold start** (session start,
  no files in context) and shrunk as the session warms — the first map is the most
  expensive context the agent has, so it gets the most tokens.
- **Chat-aware ranking:** the ranking is weighted toward files the **conversation
  references** (the files the agent has read / the symbols it has mentioned). The map
  is re-ranked per turn as the conversation shifts.
- **Per-turn re-send:** the map is **re-sent/re-ranked per turn**, not just at session
  start — the agent's first-minute context is continuously re-oriented.

### Twigpine/openclaude

- **PageRank + IDF:** the ranking is **PageRank over an IDF-weighted reference graph** —
  edges are weighted by **reference count × IDF of the symbol name**. Common names
  (`get`/`set`/`init`) have low IDF and count less; rare names count more. This is the
  "the hubs are the important files, but not the generic-named ones" signal.
- **Token-budgeted rendering:** the map is rendered by **walking the ranked files top-
  down and stopping at the token budget**. The auto-inject default is **1024 tokens**;
  the on-demand command default is **2048 tokens**.
- **Auto-injection:** the map is **auto-injected at session start** (an opt-in flag) +
  available as an on-demand tool.
- **`--focus` as a boost:** `--focus` is a **ranking boost** (a prior on the focused
  subtree), **not a filter** — the focused files rank higher, but the rest of the map is
  still there.
- **No second cache:** the reference uses a disk cache keyed by path+mtime+size. **We do
  NOT need a second cache** — the code-index IS the cache (the `index_freshness` + the
  `files`/`symbols`/`edges` tables are the durable, freshness-tracked index).

### cyanheads/repo-map

- **The per-file field contract:** the emitted doc, per file, carries:
  - **Description** — what the file does.
  - **Developer Consideration** — "the one thing a contributor is most likely to get
    wrong" (the killer field — the gotcha, not the summary).
  - **Maintenance Flag** — Stable / Volatile / Generated / Unknown.
  - **Architectural Role** — Entrypoint / Service / API Route / Utility / Test /
    Configuration / …
  - **Critical Dependencies** — the file's key dependencies, each with a one-line reason.
- **LLM-generated + cached in SQLite:** the reference generates the fields with an LLM +
  caches them. **Our difference:** the Developer Consideration is **accumulated memory**
  (a saved observation), not LLM-generated; the other fields are derived from the graph
  (no LLM in the first cut).

### mikiships/agentmd

- **Minimal-mode discipline:** verbose context files **reduce task success ~20%**. The
  highest-value content is **exact commands + where things live**; the lowest-value is
  **generic prose**. The AGENTS.md instruction is a table, not a paragraph.
- **Drift detection:** a first-class, **CI-able** check (exit 0 = doc fresh, exit 1 =
  doc stale). The drift is a **stamp + content-hash comparison** (the doc's header stamp
  vs. the live state).
- **GENERATE → MEASURE:** the loop is generate the context, then measure it (is it
  fresh? is it relevant?). The map is the GENERATE half; the coverage contract + the
  hub flag + the drift check are the MEASURE half.

### SarthakMishra/codemap

- **Token-optimized docs as the stated goal:** the doc's purpose is to be **cheap to
  read** (token-budgeted), not comprehensive.
- **The toolkit boundary:** the map is the **map**, NOT the suite. **No RAG chat, no
  commits, no PRs, no second vector index.** The map orients; the agent drills with
  separate tools.

### JordanCoin/codemap (the strongest)

- **The coverage contract:** every dependency answer reports:
  - `status: complete|partial|unavailable`
  - `sources[]` — named, per-language (the extraction source + its status)
  - `issues[]` — per-reference (what broke, where)
  - The principle: **"a partial graph never reads as a complete one"** — so "nothing
    imports this" (`complete`, empty) and "I couldn't tell" (`partial`, broken leg) are
    **different answers**.
  - A **timed-out/failed scan returns an empty result with provenance** — not a silent
    empty graph and not a hard error.
  - The JSON is **versioned** (`schema_version`).
- **Hub-file warnings:** "⚠️ HUB FILE: imported by N files" flagged **in the
  orientation view** (not just the blast-radius kind), threshold **3+ importers**.
- **The context protocol:** one command (`codemap context --for "<intent>"`) returns a
  `ContextEnvelope` (project metadata + dependency-graph evidence + intent
  classification + working set + matched skills + handoff) for the session-start
  injection. **Our equivalent:** the session-start hook returns ONE `ContextEnvelope`
  (map + coverage + hubs + memory summary + intent prior + budget).
- **Real-rules resolution:** imports resolved through each ecosystem's **own rules**
  (go.mod, cargo metadata, package.json exports, tsconfig aliases), not string matching.
  **Our equivalent:** the code-index's **confidence-labeled edges** (EXTRACTED /
  INFERRED / AMBIGUOUS / LSP_RESOLVED) — the resolution is the extractor's, labeled,
  never a silent string match.

## What we do differently

The fold is not a port. The four deliberate differences:

1. **Notes = memory, not LLM.** The Developer Consideration (the cyanheads killer
   field) is a **saved memory observation** (FTS-matched to path/package), not an
   LLM-generated note. The map is a **pure read** over the accumulated memory — the
   notes are what the agent already saved, re-surfaced on the file. The two LLM doors
   (a stored per-file LLM note fallback for files with zero memory; a stored `code_ref`
   pointer) are **deferred to v2.1**, named, not built.
2. **Structure = live index, not a separate parse.** The code structure is the
   code-index's **live graph** (`files`/`symbols`/`edges`/`index_freshness` + the
   `graph` package), not a separate parse. **No second cache** (the openclaude fold):
   the code-index IS the cache. The repo-map reads it through the existing seams + the
   `graph` package (ADR-0047: a view, not a store extension).
3. **Ranking = PageRank + chat-context.** The ranking is **PageRank over an IDF-
   weighted reference graph** (the openclaude fold) **with a chat-context prior** (the
   aider fold: the `context` param is a PageRank prior). The `--focus` is a ranking
   boost (path) or a neighborhood (symbol), never a filter.
4. **Map = first verb, not the whole composition.** The map **orients**; the agent
   **drills** (`code_search`/`code_read`/`code_resolve`/`code_path`/`mem_lookup` are
   separate tools — the retrieval-as-reasoning contract, code-index spec §1.1). The map
   is the FIRST verb in the composition, not the whole composition (the SarthakMishra
   boundary). RAG chat, commits, PRs are OUT of scope.

## Substrate facts (confirmed against the in-flight changes, 2026-11-01)

The repo-map reads two legs. The substrate facts the blueprint rests on, confirmed
against the in-flight changes:

**The memory leg** (`2026-10-31-mnemonic-v2-memory`, in-flight):
- The `Store` interface (`mnemonic-v2/internal/core/store/store.go`) — the dialect-
  agnostic data layer. The repo-map does **not** extend it (ADR-0047).
- The `SQLiteStore` **seams** (`sqlite.go`): `RawQuery(query string, args ...any)
  ([][]any, error)` (the multi-row multi-column read — the FTS MATCH + JOIN needs it),
  `Probe(query string, args ...any) (bool, error)` (single bool), `ProbeInt(query
  string, args ...any) (int, error)` (single int), `RawExec(query string, args ...any)
  error` (the write — the trail append). These are the read/write surface the repo-map
  uses.
- The `observations` table (`001_schema.sql`): `id`, `sync_id`, `session_id`, `type`,
  `title`, `content`, `tool_name`, `project`, `scope`, `topic_key`, `normalized_hash`,
  `revision_count`, `duplicate_count`, `last_seen_at`, `pinned`, `review_after`,
  `superseded_by_sync_id`, `expires_at`, `embedding`, `embedding_model`,
  `embedding_created_at`, `importance`, `retrieval_usage`, `provenance`, `created_at`,
  `updated_at`, `deleted_at`. The repo-map reads `sync_id`, `topic_key`, `importance`,
  `type`, `title`, `content`, `project`, `created_at`, `deleted_at`.
- The `observations_fts` FTS5 external-content table (trigram over `title, content,
  tool_name, type, project, topic_key`) + the four sync triggers. The repo-map's
  **query-time FTS join** (the Developer Consideration) reads this.
- The `sessions` + `session_trails` tables. The repo-map's retrieval-trail row (INV-5)
  appends to `session_trails` (the same seam the memory change's `trails.go` uses:
  `RawExec`).
- The `EventBus`/`SaveEvent` (`internal/memory/eventbus.go`) — the save handoff. The
  repo-map does not consume it (it reads the committed `observations`, not the event).

**The code-index leg** (`2026-10-31-mnemonic-v2-code-index`, queued):
- The 7 tables: `files`, `chunks`, `chunks_fts`, `symbols`, `edges`, `dep_edges`,
  `index_freshness` (additive migrations, numbered to follow the memory change's
  001–060 range).
- The `internal/codeindex/graph` package: `Orient`/`Impact`/`Callees`/`Callers` (the
  dialect-agnostic walker). The repo-map calls these read-only (the blast-radius kind
  uses `Impact`; the neighborhood uses `Callees`/`Callers`). **If a method is missing
  at T0, the repo-map reads the `edges` table through the `RawQuery` seam directly**
  (ADR-0047 still holds — no `Store` extension).
- The `symbols` table: the `signature` column (parameters + types — the Tier 1
  signature-line floor), `name`, `qualified_name`, `kind`, `start_line`, `end_line`.
- The `edges` table: `kind`, `from_id`, `to_id` (NULL = name-only), `to_name`,
  `file_id`, `confidence`, `line`, `context`. The repo-map's ranking graph + the hub
  fan-in + the blast-radius all read this.
- The `index_freshness` table: the 4-state freshness (`fresh`/`lag`/`empty`/`unknown`)
  + the per-language extraction tier (`extractor_tiers` JSON, D-TIER) + the counts. The
  repo-map's coverage contract reads this (the per-language tier → `sources[]`).

## Open questions (the briefing's closed set)

The four open questions from the briefing were **CLOSED in the Plan-mode conversation**:

| Question | Closed as |
|---|---|
| Code structure too, or memory only? | **Both** — the map is over code structure (the code-index graph) AND memory (the observations). |
| AGENTS.md every-run instruction? | **Yes** — a lean trigger-table inside the existing `skillgrid:start` block + the MCP session-start hook as the primary path. |
| LLM in the first cut? | **No** — the map is a pure read; per-file fields are derived from the graph + memory. The two LLM doors (stored note fallback, stored code_ref) are deferred to v2.1. |
| The join mechanism? | **Query-time FTS** over `observations_fts` (the floor); the stored `code_ref` pointer is the deferred v2.1 door. |
