# Mnemonic

Mnemonic is the local-first persistent memory engine embedded in the `skillgrid`
CLI. It provides three integrated capabilities — **session memory** (Engram-aligned),
**code search** (OpenClaw-aligned), and **web research cache** (Neuledge-aligned) —
backed by a per-project SQLite database with FTS5 indexing.

Mnemonic exposes two transports:

- **MCP stdio server** — `skillgrid mcp` — consumed directly by AI agents as tools
  (`mem_*`, `code_*`, `web_*`).
- **HTTP REST API** — `skillgrid serve` — consumed by agent plugins for session
  lifecycle, compaction recovery, and index health nudges.

```
agent plugin (opencode/kilo/cursor)
   │  auto-starts  `skillgrid serve` on GET /health failure
   │  registers     `skillgrid mcp` as the MCP server
   │
   ▼
skillgrid mcp          ◄── stdio MCP transport (tool calls)
skillgrid serve        ◄── HTTP API transport (127.0.0.1:7438)
skillgrid index        ◄── one-shot incremental indexer (CLI)
   │
   ▼
Service facade  ──► store (SQLite per project)
   ├─ memory service    (observations + sessions, FTS5)
   ├─ codeindex         (files + chunks, FTS5 trigram/porter)
   └── webcache service  (web snapshots, FTS5 + TTL expiry)
```

## Data model

Data is stored under `~/.skillgrid/mnemonic/` (override with
`SKILLGRID_MNEMONIC_DATA_DIR`). Each resolved **project** gets its own SQLite file
named `<project-id>.sqlite`, so observations, code index, and web cache are
scoped per project.

### Project resolution

The project ID is resolved per working directory (`internal/mnemonic/project/resolve.go`):

1. Nearest `.skillgrid/config.json` with a `project` field — highest priority.
2. Git remote `origin` URL — the repo basename (e.g. `my-org/my-repo` → `my-repo`).
3. Fallback — `{basename}-{sha256(cwd)[:8]}`.

IDs are normalized (lowercased, `[-_]+` collapsed to `-`, trimmed of leading/trailing
dashes) and validated against path traversal (`..` is rejected).

### SQLite schema

Embedded migrations (`store/migrations/*.sql`) create:

| Table | Purpose |
|---|---|
| `sessions` | Workspace sessions (UUID, project, directory, status) |
| `observations` | Memory entries (FTS5 `observations_fts` with Porter stemming). Includes `review_after` for the `mem_review` lifecycle (`005_review_cycle.sql`), `prompt_id` for `capture_prompt` provenance (`007_prompt_link.sql`), and `source` for passive/agent provenance. |
| `files` / `chunks` | Indexed source files (FTS5 `chunks_fts` with trigram tokenizer) |
| `web_cache` | Cached web research snapshots (FTS5 `web_cache_fts` with Porter) |
| `memory_relations` | Directional, typed semantic links between observations — the `mem_judge` / `mem_compare` verdict store (`006_relations_aliases.sql`). Soft-deleted. |
| `project_aliases` | Retired project name → canonical name mapping for `mem_merge_projects` + drift warnings (`006_relations_aliases.sql`). |
| `prompts` | Recorded user prompts (via `mem_save_prompt`, `004_prompts_passive.sql`); linked to observations by `prompt_id` on `capture_prompt`. |
| `index_meta` | Schema version tracking |

## Indexing configuration

Code indexing reads `config.d/indexing.yaml`, walking up from the indexed directory
to find the first match. Defaults (merged with the repo root `config.d/indexing.yaml`):

| Field | Default |
|---|---|
| `include` | `**/*.go`, `**/*.ts`, `**/*.tsx`, `**/*.md` |
| `exclude` | `node_modules`, `.git`, `dist`, `.skillgrid`, `vendor`, `.venv`, … |
| `chunk_lines` | 80 |
| `chunk_overlap` | 10 |
| `max_file_size_kb` | 512 |

Web cache TTLs (defaults): `context7` 30d, `exa` 7d, `deepwiki` 14d, `fetch` 7d,
`manual` never expires. Maximum entry size is 256 KB.

---

## MCP tools

### 30 MCP tools (21 memory + 4 code + 5 web cache)

The `skillgrid mcp` stdio server advertises 30 tools across three domains —
session memory, indexed-code search, and a web-research cache. All output is
raw JSON (no leading prose) per the OCBI convention.

#### Memory tools

The memory surface mirrors Engram's MCP tool list. The tools are grouped by
workflow intent: **save / recall / evolve / manage / relate / diagnose**.

**Save & recall**

Tool | What It Does
--- | ---
`mem_save` | Save an observation to persistent memory. Deduplicates by content hash within 24h or upserts by `topic_key`. **`capture_prompt`** (bool, default `true`) best-effort links the session's most recent user prompt (recorded via `mem_save_prompt`) to the observation via `prompt_id` so a reader can recall *what was asked*; pass `capture_prompt=false` for automated saves that should not carry prompt context. **`project`** (optional) records under an explicit named project; if that name has been retired by `mem_merge_projects` the write routes to the canonical store and a `project_drift` warning is returned. |
`mem_save_prompt` | Record a raw user prompt so future sessions can recall what was asked. Prompts are trimmed (min ~11 chars) and bounded to 2 KB. Feeds `capture_prompt` on the next `mem_save` in the same session. |
`mem_search` | FTS5 full-text search over saved observations (match_mode: `any`/`all`, default 20 results; optional `scope` filter: `project`/`user`/`global`). Progressive-disclosure layer 1 — compact hits with IDs. **`project`** (optional) scopes/overrides the search; a retired alias routes to the canonical store with a `project_drift` warning. |
`mem_context` | Recent session summaries (default 5) — run this first for fast recall before a full search. |
`mem_get_observation` | Fetch full untruncated observation content by ID. Progressive-disclosure layer 3. |
`mem_timeline` | Chronological context around an observation — `before`/`after` windows (`window`, e.g. `30m`, `2h`). Progressive-disclosure layer 2, between `mem_search` and `mem_get_observation`. |

**Evolve existing memories**

Tool | What It Does
--- | ---
`mem_suggest_topic_key` | Suggest a stable `topic_key` for upserting an evolving topic. Call this *before* `mem_save` when unsure of the key. |
`mem_update` | Update an observation in place — any of `title`, `content`, `type`, `scope`, `topic_key`. Only non-empty fields apply. Bumps `updated_at`; the FTS index and the 24h dedup hash re-sync. |
`mem_capture_passive` | Extract structured learnings from a pasted text block (e.g. a finished task transcript). Server recognises `Key Learnings:` sections and labelled Lesson/Discovery lines; idempotent — re-capturing the same text does not duplicate rows. |

**Session lifecycle** (driven by the agent plugin; also callable directly)

Tool | What It Does
--- | ---
`mem_session_start` | Create a workspace session for a directory; returns `session_id`. Optional `title` names the session — shown in the web dashboard session list. |
`mem_session_end` | End a session, optionally with a summary. |
`mem_session_summary` | Persist a structured end-of-session summary (Goal / Discoveries / Accomplished / Next Steps / Relevant Files). |
`mem_session_set_title` | Rename a session after the fact (e.g. before closing, once the topic is known). |

**Manage lifecycle**

Tool | What It Does
--- | ---
`mem_delete` | Delete an observation. Soft-delete by default (`deleted_at` set — excluded from search/context/timeline, still fetchable by ID); pass `hard=true` to remove the row permanently. |
`mem_review` | `action="list"` (default) — returns observations whose `review_after` has passed. `action="mark_reviewed"` + `id` — advances the observation's review cycle by ~30 days. This is the local-only lifecycle hygiene loop. |
`mem_merge_projects` | Admin tool. Merge a source project name into the canonical name: copies rows tagged `source` into the canonical store (idempotent), re-tags them, and records an alias so future `mem_save`/`mem_search` calls naming `source` route to the canonical store and return a `project_drift` warning. |

**Relate memories** (semantic links between observations in the same project)

Tool | What It Does
--- | ---
`mem_judge` | Record a verdict for a memory conflict between two observations. Verdicts: `related` \| `compatible` \| `scoped` \| `conflicts_with` \| `supersedes` \| `not_conflict`. `not_conflict` clears a previously-recorded `conflicts_with` link instead of adding one. Optional `confidence` (0.0–1.0) and `reason`. |
`mem_compare` | Record, clear, or inspect semantic relations between two observations. With `src_id` + `dst_id` + `relation` it records/clears a link; **omit `relation` to list** the current links between the pair. |

> **Relation storage**: verdicts are directional typed links (`src → dst`) in the `memory_relations` table, scoped to the project store and soft-deleted. Re-judging the same `(src, dst, relation)` upserts in place.
>
> **Prompt provenance (`capture_prompt`)**: with `capture_prompt=true` (default) a `mem_save` links the session's latest recorded prompt to the observation (`observations.prompt_id`). `mem_get_observation`/`mem_search`/`mem_timeline` echo `prompt_id` on any linked row. `capture_prompt=false` suppresses the link — use it for SDD phase artifacts and other automated saves that only need the *conclusion*, not the *framing question*.
>
> **Project scoping & drift**: `mem_save` and `mem_search` accept an explicit `project` name. Mnemonic already stores per named project (one SQLite file per project), so multi-checkout / monorepo scoping works. After a `mem_merge_projects` (source → canonical), any tool call that names the retired `source` **routes to the canonical store** and returns a `project_drift` object — `{project_name, canonical_name, merged_at}` — so an agent can see it wrote to a merged name and switch to the canonical one. The write is never rejected; the drift is advisory.

**Diagnose & orient** (never errors; recommended first calls)

Tool | What It Does
--- | ---
`mem_current_project` | Detect the project from cwd and return the resolved ID, its **source** (`config` / `git-remote` / `directory-hash`), and the full list of known projects. Call this first to confirm you're writing to the right project. |
`mem_stats` | Per-project statistics: observation count by type, active/total sessions, created-range. |
`mem_doctor` | Read-only store health: schema version, WAL mode, FTS row counts and drift vs. the base tables, per-type counts, on-disk size. |

> **Memory types:** `standing`, `preference`, `convention`, `decision`, `architecture`, `bugfix`, `pattern`, `config`, `correction`, `discovery`, `learning`, `lesson`, `session_log`.
>
> **Progressive disclosure (3-layer pattern):** `mem_search` (compact hits) → `mem_timeline` (chronological neighbours) → `mem_get_observation` (full content). Don't dump the whole store — drill in.
>
> **Full Engram mirror complete**: every Memory tool and behavior in the Engram `ARCHITECTURE.md` MCP tool list is implemented — save/recall (`mem_save`, `mem_save_prompt`, `mem_search`, `mem_context`, `mem_get_observation`, `mem_timeline`), evolve (`mem_update`, `mem_suggest_topic_key`, `mem_capture_passive`), lifecycle (`mem_delete`, `mem_review`, `mem_merge_projects`), relate (`mem_judge`, `mem_compare`), and diagnose (`mem_current_project`, `mem_stats`, `mem_doctor`). Engram's `capture_prompt` switch, explicit `project` scoping, and project-name drift warning are all supported; graph-relationship verdicts that Engram stores as graph edges are held in Mnemonic's `memory_relations` table (soft-deleted, project-scoped).

#### Code tools

Tool | What It Does
--- | ---
`code_status` | Check index health — returns file/chunk counts and `stale`. Run before `code_search` after a clone or large refactor. |
`code_index` | Run incremental code indexing for the cwd git root (respects `config.d/indexing.yaml` include/exclude). |
`code_search` | BM25 full-text search over indexed code chunks (default 20 results). |
`code_read` | Fetch indexed source for a path, with optional `start_line` / `end_line`. |

> **Ladder:** `code_status` → `code_search` → `code_read`. Prefer `code_search` over grep when exploring unknown territory in a large repo; grep only for exact identifier lookups.

### Web cache tools

Tool | What It Does
--- | ---
`web_cache_lookup` | Check the cache *before* calling a remote MCP (Context7/Exa/DeepWiki/fetch). Returns `hit`/`miss`/`stale`. |
`web_cache_save` | Persist a snapshot — call immediately after a remote MCP or web fetch returns. |
`web_cache_search` | FTS5 search over cached snapshots (optional source filter, freshness filter). |
`web_cache_get` | Fetch a full untruncated snapshot by ID. |
`web_cache_status` | Cache health: counts by source, expired entries, oldest/newest fetch. |

> **Supported sources:** `context7`, `exa`, `deepwiki`, `fetch`, `manual`. Each source derives its cache key from a different set of fields (e.g. `library_id`+`version_tag`+`query` for context7, `url` for fetch).

---

## HTTP API

`skillgrid serve` starts the HTTP API on `127.0.0.1:7438` (override `--port`
/ `SKILLGRID_MNEMONIC_PORT`, `--bind`). Write routes require a bearer token when
`SKILLGRID_HTTP_TOKEN` is set; read routes are always open.

Browse the interactive API at the root `/` (data viewer) or `/swagger-ui`.
The OpenAPI spec is served at `/openapi.yaml`.

### Health

```
GET /health
```

```json
{"status":"ok","service":"skillgrid-mnemonic","version":"0.1.0"}
```

### Memory / sessions

| Method | Path | Query / Body | Description |
|---|---|---|---|
| `POST` | `/sessions` | `directory` query, `X-Workspace-Dir` header | Create a session for a workspace directory; returns `session_id` + `project_id`. |
| `POST` | `/sessions/{id}/end` | `project` query, `{"summary": "..."}` body | End a session, optionally with a summary. |
| `POST` | `/sessions/{id}/title` | `project` query, `{"title": "..."}` body | Rename a session (`mem_session_set_title`). |
| `GET` | `/sessions/{id}` | `project` | Session detail. |
| `GET` | `/context` | `project`, `limit` (default 5) | Recent session summaries for compaction recovery. |
| `GET` | `/context/compaction` | `project`, `session`, limit | Session-scoped compaction context (session + recent observations). |
| `POST` | `/observations` | `project` query, SaveObservationInput body | Save an observation. |
| `POST` | `/observations/passive` | `project`, passive input body | Extract learnings from a pasted text block (`mem_capture_passive`). |
| `GET` | `/observations/recent` | `project`, `limit` (default 5) | Recent observations. |
| `GET` | `/observations` | `project`, `limit` (default 50) | All recent observations. |
| `PATCH` | `/memory/observations/{id}` | `project`, `{"title"/"content"/"type"/"scope"/"topic_key"}` body | Update an observation — only non-empty fields apply (`mem_update`). |
| `DELETE` | `/memory/observations/{id}` | `project`, optional `hard=true` query | Delete an observation. |
| `GET` | `/memory/timeline` | `project`, `id`, optional `window` (e.g. `1h`), `limit` (default 5) | Chronological context around an observation. |
| `GET` | `/search` | `project`, `query`, `match_mode`, `limit` | FTS search over observations. |
| `POST` | `/prompts` | `project`, prompt input body | Record a user prompt (`mem_save_prompt`). |
| `GET` | `/memory/reviews` | `project`, `limit` (default 20) | List observations due for local review. |
| `POST` | `/memory/reviews/{id}` | `project` | Mark an observation reviewed; advances review cycle. |
| `GET` | `/memory/project` | — | Resolved project ID + source + all known projects. |
| `GET` | `/memory/status` | `project` | Per-project observation/session counts. |
| `GET` | `/memory/last-save-at` | `project` | Timestamp of the newest observation. |
| `GET` | `/memory/doctor` | `project` | Store health: schema version, FTS drift, WAL, disk size. |
| `POST` | `/memory/relations` | `project`, `{"src_id":1,"dst_id":2,"verdict":"conflicts_with","reason":"..."}` body | Record a semantic link between observations (`mem_judge` / `mem_compare`). |
| `DELETE` | `/memory/relations` | `project`, `src_id`, `dst_id`, `relation` | Clear a semantic link. |
| `GET` | `/relations/{id}` | `project` | All live relations touching an observation. |
| `GET` | `/relations` | `project`, `src_id`, `dst_id` | Live relations between two observations. |
| `POST` | `/projects/merge` | `{"source":"old","canonical":"new"}` body | Merge a project variant into canonical name + record alias (`mem_merge_projects`). |
| `POST` | `/projects/migrate` | `{"old_project":"...","new_project":"..."}` body | Copy rows between two project stores (no alias record). |

### Code

| Method | Path | Query | Description |
|---|---|---|---|
| `GET` | `/code/status` | `project` | Index stats + `stale` flag. |
| `POST` | `/code/index` | `dir` | Run incremental indexing for a directory. |
| `GET` | `/code/files` | `project` | All indexed file paths, sorted. |
| `GET` | `/code/search` | `project`, `query`, `limit` | BM25 code search. |
| `GET` | `/code/read` | `project`, `path`, `start_line`, `end_line` | Read indexed source. |

### Web cache

| Method | Path | Query / Body | Description |
|---|---|---|---|
| `GET` | `/web/lookup` | `project`, `source` + key fields | Check cache before a remote call. |
| `POST` | `/web/cache` | `project` query, SaveWebInput body | Save a snapshot. |
| `GET` | `/web/search` | `project`, `query`, `source`, `fresh_only` | FTS search cached snapshots. |
| `GET` | `/web/entry/{id}` | `project` | Full snapshot by ID. |
| `GET` | `/web/status` | `project` | Cache health stats. |

### Projects

```
GET /projects
```

Returns the sorted list of project IDs that have a store under the data directory.

---

## Agent plugins

`skillgrid setup <agent>` installs Mnemonic integration for one of `opencode`,
`kilocode`, or `cursor` (`kilo` is accepted as an alias for `kilocode`). It copies
plugin files from `plugins/<agent>/` in the repo into the agent's config
directory and registers `skillgrid mcp` as the MCP server.

### OpenCode / Kilo

- Copies `plugins/opencode/mnemonic.ts` → `~/.config/opencode/plugins/mnemonic.ts`
- Copies `plugins/kilo/mnemonic.ts` → `~/.config/opencode/shared/http-client.ts`
- Upserts `mcp.skillgrid-mnemonic` in `~/.config/opencode/opencode.jsonc` (or `kilo.jsonc` for Kilo)
- For Kilo: writes the Memory Protocol (from `plugins/kilo/memory-protocol.md`)
  into `~/.config/kilo/AGENTS.md` between managed markers, and bridges shared files from
  OpenCode's config.
- The OpenCode plugin reads `SKILLGRID_MNEMONIC_HTTP_URL` (default `http://127.0.0.1:7438`)
  and `SKILLGRID_MNEMONIC_HTTP_TOKEN` for requests to `skillgrid serve`.
- Both plugins auto-start `skillgrid serve` if `GET /health` fails, create a session
  on `session.created`, inject the Memory Protocol on every chat turn, recover context
  on compaction, and nudge on a stale code index.

### Cursor

- Registers `skillgrid mcp` in `~/.cursor/mcp.json` under `mcpServers.skillgrid-mnemonic`
- Writes `~/.cursor/rules/mnemonic.mdc` from the template at
  `plugins/cursor/mnemonic.mdc`, injecting the Memory Protocol
  in place of `{{MEMORY_PROTOCOL}}`.

---

## CLI commands

| Command | Purpose |
|---|---|
| `skillgrid mcp` | Stdio MCP server (primary agent interface). |
| `skillgrid serve` | HTTP API server (primary plugin interface). |
| `skillgrid index` | One-shot incremental code indexer for a directory (`--dir`, `--project`). |
