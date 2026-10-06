# Skillgrid v2 — System Architecture

> **Version:** 2.0
> **Last Updated:** 2026-10-05
> **Status:** Active Development
> **Source of truth:** this file describes the *live* repo/program structure. Decisions live in `.skillgrid/ASSUMPTIONS.md` (LOCKED `### ADR-NNNN` set) — this file never re-argues them, it points at them.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Top-Level Layout](#2-top-level-layout)
3. [CLI Application (Go Binary)](#3-cli-application-go-binary)
4. [Command Surface](#4-command-surface)
5. [Mnemonic Engine — Package Structure](#5-mnemonic-engine--package-structure)
6. [Code Indexing Architecture](#6-code-indexing-architecture)
7. [Hybrid Search Architecture](#7-hybrid-search-architecture)
8. [Memory Architecture](#8-memory-architecture)
9. [Storage Layer (SQLite + FTS5 + vec0)](#9-storage-layer-sqlite--fts5--vec0)
10. [Transport Architecture (MCP + HTTP)](#10-transport-architecture-mcp--http)
11. [Frontend Architecture (Embedded SPA)](#11-frontend-architecture-embedded-spa)
12. [Plugin Architecture (Multi-Agent)](#12-plugin-architecture-multi-agent)
13. [The Skill Grid (`.agents/skills/`)](#13-the-skill-grid-agentskills)
14. [Configuration Architecture](#14-configuration-architecture)
15. [SDD Pipeline Architecture](#15-sdd-pipeline-architecture)
16. [Git Hook Enforcement Architecture](#16-git-hook-enforcement-architecture)
17. [Error Handling & Degradation (Fail-Open Floors)](#17-error-handling--degradation-fail-open-floors)
18. [Testing Architecture](#18-testing-architecture)
19. [Security Architecture](#19-security-architecture)

---

## 1. Architecture Overview

Skillgrid is a **single Go binary** that acts as an *agentic memory + code-intel + workflow* engine. One process serves three clients: the **agent harness** (Cursor / OpenCode / Kilo) over MCP-stdio and shell hooks, the **operator** over a local HTTP API, and the **dashboard** (a React SPA compiled *into* the binary). It dogfoods its own `.skillgrid/` state zone and SDD pipeline — this repo is simultaneously the product and its first customer.

### High-Level System Diagram

```
                    ┌──────────────────────────────────────────────────────┐
 AGENT HARNESS      │                skillgrid  (single Go binary)         │
 (Cursor / OpenCode │                                                      │
  / Kilo)           │  cmd/skillgrid/main.go  (flag.FlagSet + switch)      │
   │  hooks.yaml /  │   ├─ install ────▶ ~/.skillgrid/ + agent config merge│
   │  hooks-cursor  │   ├─ setup ──────▶ plugins → agent config merge      │
   │  ─── hooks/*.sh▶│   ├─ init/index ▶▶ codeindex (tree-sitter → chunk → │
   │  (session,tool, │   │             │   graph → embed → community)      │
   │   policy,stop) │   ├─ mcp ──────── stdio MCP (mcp-go) ──────┐        │
   │         ▲       │   ├─ serve ───── HTTP :7438 ──────────────┤        │
   │         └───────│   │   ├─ REST (memory/sessions/policy/…)  │        │
   │  (capture POST, │   │   ├─ embedded SPA (skillgrid-ui dist) │        │
   │   checkpoint,   │   │   └─ /openapi.yaml + /swagger         │        │
   │   stop gate)    │   └───────────────────────┬──────────────┘        │
                     │                           ▼                         │
                     │         internal/mnemonic/service.Service           │
                     │         (facade: memory · codeindex · webcache)     │
                     │                           ▼                         │
                     │   ~/.skillgrid/mnemonic/<projectID>.sqlite (WAL)    │
                     │   embedded migrations · FTS5 (porter/trigram) · vec0│
                     └─────────────────────────────────────────────────────┘
 REPO CONTENT (shipped by `skillgrid install`, the "Hub Content")
 .agents/skills/ (SDD pipeline + _shared rules)      .skillgrid/ (config,
 hooks/  git-hooks/  plugins/  config.d/  rules/      state, specs, artifacts)
```

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| One binary, many transports | Go 1.26, hand-rolled command dispatch | No runtime deps at the operator; MCP + HTTP + TUI share one `service.Service` facade. |
| Pure-Go SQLite | `modernc.org/sqlite` (cgo-free) + `vec0` | Cross-compiles without a C toolchain; vector store without a second engine. (ADR-0001 cgo-free.) |
| Per-project database file | `~/.skillgrid/mnemonic/<projectID>.sqlite` (WAL) | Isolation per repo; no shared-server; safe to run several agents at once. |
| FTS5 is the floor | porter + trigram lexical indexes, vector legs degrade | "FTS5 is the floor" — the semantic leg can be absent/slow and search still works (`knowledge/hybrid-degradation.md`). |
| SPA embedded in the binary | `vite build` → `//go:embed http/ui/dist` | Dashboard ships in the same artifact; same-origin, no separate web server. |
| Skills are files, not a registry | `.agents/skills/**/SKILL.md` loaded by the harness | The grid is plain markdown — versionable, reviewable, no plugin ABI to maintain. |
| Fail-open floors | every LLM/vector seam has a deterministic fallback | An agent must never hard-crash because a model or an indexer is down. |
| Hooks enforce, prose informs | `git-hooks/` + `hooks/` run real checks | Acceptance gates and zone rules are *executed* (`--reverify`), not merely documented. |

---

## 2. Top-Level Layout

```
/
├── skillgrid-cli/      Go module — the ENTIRE runtime (installer, Mnemonic engine,
│                       MCP server, HTTP API, embedded dashboard)
├── skillgrid-ui/       Vite + React 19 SPA; dist/ is built INTO the Go binary (go:embed)
├── plugins/            Per-agent plugin assets: opencode/ (hooks.yaml), kilo/, cursor/mcp.json
├── .agents/skills/     THE SKILL GRID — 7 category dirs + _shared/ reference library
├── .cursor-plugin/     Cursor plugin manifest (plugin.json, marketplace.json)
├── rules/              Cursor-native rule: mnemonic.mdc (always-apply memory protocol)
├── hooks/              Hook IMPLEMENTATIONS (Node/bash workers: tool-call-capture.js, …)
├── git-hooks/          Hook ENTRYPOINTS — thin shims that spawn into hooks/
├── scripts/            Repo tooling: skill hygiene guard, state-lock, gen-guide, tests
├── docs/               Hugo docs site source (user-guide/, tutorials/, skill-anatomy.md)
├── config.d/           Install-time distribution config: indexing.yaml, mcp.yaml, tools.yaml
├── agents/             Dispatch-role subagent defs per harness (opencode/, cursor/)
├── .skillgrid/         THIS repo's own state zone (it dogfoods itself)
├── .backlog/           Local markdown ticket tracker (backlogmd)
└── Taskfile.yml        task driver: build / test / seed-test / skill-check / site / install
```

Two distinct "content" worlds, deliberately separated:

| World | Owner | Lives | Example |
|-------|-------|-------|---------|
| **Runtime** (binary) | the Go module | `skillgrid-cli/` | `service.Service`, `codeindex`, `store` |
| **Hub Content** (shipped) | the installer | `plugins/ hooks/ git-hooks/ config.d/ rules/ .agents/skills/` | `hooks.yaml`, `block.md`, `indexing.yaml` |

The installer (`skillgrid install`) *stages* Hub Content into machine locations (`~/.skillgrid/plugins/`, `~/.skillgrid/hooks/`, `~/.skillgrid/git-hooks/`) and merges MCP entries into the agent config — it never edits the repo.

---

## 3. CLI Application (Go Binary)

### 3.1 Module & entry point

`skillgrid-cli/go.mod`:

```go
module github.com/devopstales/skillgrid/skillgrid-cli

go 1.26

require (
    github.com/mark3labs/mcp-go       v0.58.0   // MCP server
    modernc.org/sqlite                v1.59.0   // pure-Go SQLite (cgo-free)
    github.com/odvcencio/gotreesitter  v0.52.0   // AST extraction
    github.com/benedoc-inc/onnxer     v0.7.0    // in-process ONNX embeddings
    github.com/bluuewhale/loom                         // Leiden community detection
    github.com/charmbracelet/huh          v1.0.0    // TUI multiselect
    ...
)
```

Entry point `cmd/skillgrid/main.go` is **not** cobra — a single `flag.FlagSet` for install flags plus a `switch rest0` on the first bare argument. Version is baked at build time:

```go
// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"
```

```bash
# Taskfile.yml
build:
  cmds:
    - go build -tags ui -ldflags "-s -w -X main.version=..." -o dist/skillgrid ./cmd/skillgrid
```

The `ui` build tag is what activates the embedded SPA; without it the binary is smaller and serves no dashboard.

### 3.2 Layering

```
┌──────────────────────────────────────────────────────────────────┐
│  TRANSPORT LAYER                                                 │
│  cmd/skillgrid/*.go  (main.go dispatch, mcp.go, serve, init…)     │
│   ├─ MCP stdio (mcp-go)     ── agent harness                     │
│   ├─ HTTP :7438 (ServeMux)  ── operator + dashboard              │
│   └─ TUI (huh)              ── interactive prompts               │
├──────────────────────────────────────────────────────────────────┤
│  SERVICE FACADE                                                  │
│  internal/mnemonic/service.Service + ProjectHandle               │
│   (opens a project ONCE, refcounted store pool; the single seam  │
│    every transport funnels through)                              │
├──────────────────────────────────────────────────────────────────┤
│  SUBSYSTEM LAYER                                                 │
│  memory · codeindex · graph · search · hybrid · embedder ·        │
│  webcache · skills · session_inject · checkpoint · policy · …     │
├──────────────────────────────────────────────────────────────────┤
│  DATA LAYER                                                      │
│  internal/mnemonic/store  (SQLite WAL + embedded migrations +    │
│   FTS5 + vec0)  ──▶  ~/.skillgrid/mnemonic/<projectID>.sqlite    │
└──────────────────────────────────────────────────────────────────┘
```

### Layer Rules

| Rule | Description |
|------|-------------|
| **Transports → facade** | A command/tool opens a `ProjectHandle` and stops there — it never touches `store` or a subsystem directly. |
| **One handle per request** | Each MCP tool call or HTTP request opens the project once (refcounted) and works through the handle. |
| **Subsystems are siblings** | `memory`, `codeindex`, `search` do not import each other's internals; they share `store` + `service` only. |
| **No reverse deps** | `store` never imports a subsystem; a subsystem never imports `cmd/`. |
| **Config is a seam** | every subsystem reads through `config.Load` (defaults < home-local < repo-local) — absent key ⇒ default, never a hard fail. |

---

## 4. Command Surface

Hand-rolled dispatch (no cobra). The full surface from `main.go` usage:

| Group | Commands |
|-------|----------|
| Install | `install` / `in`, `sync-repo` |
| Transports | `mcp` (stdio), `serve` (HTTP, default `:7438`) |
| Project | `init` (boot file + index + doc ingest), `index` (+ `index status`), `setup` (opencode/kilocode/cursor) |
| Code intel | `orient`, `grep`, `callers`, `callees`, `dependents`, `implementors`, `hierarchy`, `tests-for`, `path`, `explain`, `impact`, `explore`, `diff-impact` |
| Search | `search` (hybrid FTS + signals + semantic) |
| Memory | `mem`, `memory` (fact memory), `migrate`, `trail` |
| Session | `prime`, `compact`, `sessions`, `session`, `logs`, `stats`, `policy`, `skill` |
| Ops | `doctor` (`--strict` for CI), `eval` (retrieval ablation), `pages`, `embedding-status` |

---

## 5. Mnemonic Engine — Package Structure

`skillgrid-cli/internal/mnemonic/` — 34 sub-packages. The load-bearing ones:

| Package | Responsibility |
|---------|----------------|
| `store/` | SQLite wrapper + embedded migrations (`store/migrations/*.sql`). Refcounted pool, WAL retry, `vec0` registration. |
| `service/` | **The facade** — `Service` + `ProjectHandle`; `DefaultDataDir()` = `~/.skillgrid/mnemonic`. |
| `memory/` | Observations: save/retrieve, FTS+vector retrieval, decay, importance tiers, governance, distill/dream consolidation, provenance. |
| `codeindex/` | Incremental indexer (`indexer.go` ~1975 lines), `watch.go` (fsnotify), `fingerprint.go` (staleness gate), `graph.go` (symbol/call graph), opt-in `pdg_*`/`lsp_*` edge tiers. |
| `extract/` | tree-sitter symbol/edge extraction per language. |
| `graph/` | Graph queries: `impact.go` (blast radius), `resolve.go`, `span.go`, `coverage.go`. |
| `search/` | FTS query building (`fts.go`, `symbol_fts.go`), structural by-example grep. |
| `hybrid/` | RRF fusion (`rank.go`, `const RRFK = 60`), vector cache, rerank, confidence, snippets. |
| `embedder/` | Providers: `onnx.go` (default `nomic-embed-code`, 768-dim), `ollama.go`, `external.go`, `null.go`, `select.go`, `warm.go`. |
| `mcp/` | MCP stdio server (`server.go`) + ~40 `tools_*.go` registering `mem_*`/`code_*`/`web_*`/`session_*`/`team_*` tools. |
| `context_harness/` | The Context Harness (ADR-0025) — owns the session context lifecycle. Absorbs `session_inject` (retrieval, render, autoprepend, summary, privacy — signatures unchanged, import path only) and adds intercept-and-abstract capture into a session-scoped FTS `tool_outputs` sandbox, a deterministic `ctx_query` over the code index, a `ctx index`/`indexed_files` writer + `ctx_search` two-leg RRF fusion, a Context Routing block, and the `ctx` CLI. A `clm/` sub-package implements the Context Language Model (ADR-0027): mirror render, overflow guard, calibration, revision validate/persist. See §8.1. |
| `session_inject/` | (absorbed into `context_harness`) — L1 auto-prepend/distill on resume, L2 hybrid retrieval + context-block render (`mem_inject_session`). Importers change import path only. See §8.1. |
| `loop/` | Session loop pages: `prime`/`compact`/`pages` text generation for session start/end hooks. |
| `http/` | REST API (`server.go` ~1912 lines) + `ui/` (embedded SPA) + `docs/` (OpenAPI/Swagger). |
| `config/` | Layered YAML config loader (`load.go` ~921 lines). |
| `checkpoint/` | Checkpoint gating (ADR-0022). |
| `policy/` | Pre-tool policy, fail-open (ADR-0021). |

---

## 6. Code Indexing Architecture

### 6.1 Pipeline

```
scan ──▶ extract ──▶ embed ──▶ community ──▶ import-cycles ──▶ process ──▶ knowledge ──▶ lsp ──▶ pdg ──▶ resolution-audit ──▶ complete
  │         │           │           │
  │ walk    │ tree-     │ onnx      │ looma
  │ +       │ sitter    │ 768-dim   │ Leiden
  │ sha256  │ symbols   │ BLOB      │ clustering
  │         │ + edges   │           │
```

`codeindex/indexer.go`:

```go
// Package codeindex incrementally indexes source files into a project store.
type Config struct {
    Include      []string
    Exclude      []string
    ChunkLines   int        // default 80
    ChunkOverlap int        // default 10
    MaxFileSize  int        // default 512 KiB — bigger files are skipped, not errored
    // PDG / LSP are independent opt-in edge tiers
    PDG  bool
    LSP  bool
    Progress func(Event)
}
```

### 6.2 Freshness — two layers

```
┌────────────────────────────┐         ┌──────────────────────────────┐
│  COMFORT LAYER (watcher)   │         │  CORRECTNESS BACKSTOP        │
│  fsnotify → debounce →     │         │  fingerprint gate            │
│  structural re-index       │         │  staleness banner +          │
│  (live UX)                 │         │  auto-reopen ~5s             │
└────────────────────────────┘         │  (works even with --no-watch)│
                                       └──────────────────────────────┘
```

The watcher is for *feel*; the fingerprint gate is for *truth*. A stale index is never served silently — the agent gets a staleness banner.

### 6.3 Defaults (from `config.d/indexing.yaml` + `config.Load`)

```yaml
include: ["**/*.go", "**/*.ts", "**/*.tsx", "**/*.md"]
chunk_lines: 80
chunk_overlap: 10
max_file_size_kb: 512
embedder: { provider: onnx, model: nomic-embed-code, dims: 768 }
```

---

## 7. Hybrid Search Architecture

Three ranked legs fused by **reciprocal rank fusion** — the lexical leg is the floor, the semantic leg degrades gracefully.

```
                 ┌──────────────┐
   query ───────▶│  FTS5 (lex)  │──rank──┐
                 └──────────────┘        │
                 ┌──────────────┐        │   ┌────────────────────────────┐
                 │  signals     │──rank──┼──▶│  RRF  (const RRFK = 60)     │
                 │  (symbol,…)  │        │   │  per-signal provenance      │
                 └──────────────┘        │   └───────────────┬────────────┘
                 ┌──────────────┐        │                   ▼
                 │  semantic    │──rank──┘            fused + reranked
                 │  (vec cosine)│                          results
                 └──────────────┘
```

`hybrid/rank.go`: `const RRFK = 60`.

The live semantic leg is **in-memory brute-force cosine over a process-global vector cache** (`hybrid/vectorcache.go`, measured ~28 ms warm). The in-SQL `vec0` path exists (migration `042_vec0_tables.sql`) but is deferred — the BLOB tables remain the source of truth. This is the "FTS5 is the floor" rule in action: if `embedder` is `null`, search still returns ranked lexical + signal results.

---

## 8. Memory Architecture

Observations are the atomic unit — a titled, typed, project-scoped memory with provenance and a decay curve.

```
┌───────────┐    ┌────────────────┐    ┌─────────────────────────┐
│  mem_save │───▶│  observations  │───▶│  retrieval               │
│  (MCP)    │    │  + fts5 porter │    │  FTS + vector + decay    │
└───────────┘    └───────┬────────┘    └─────────────────────────┘
                         │  TTL soft-expiry (7d) + decay (30d half-life)
                         │  + importance tiers (7/30/14)
                         ▼
                 ┌────────────────┐
                 │ distill / dream │  (consolidation — reduces the live set)
                 └────────────────┘
```

Key seams are all injectable with a deterministic fallback (the fail-open pattern, §17): `dedup.llm`, `extraction.llm`, `improve` are each "absent or error ⇒ deterministic path, never a crash".

**Second brain** (`mem_ask`, `secondbrain/`) is a read side over the same store: cited answers with a fail-open LLM seam.

### 8.1 Context Harness (session context lifecycle)

The Context Harness (ADR-0025) owns the full context lifecycle of a session. It absorbed `session_inject` (import path change only — public signatures and the `mem_inject_session` contract are unchanged) and adds capture, structured query, proactive index, routing, a `ctx` CLI, and the Context Language Model layer.

**Injection & compaction** — keeps conversation memory rot from losing a session's state (compaction, session death, context overflow). Locked design: **two-layer injection** — auto-prepend a slim token-capped L1 summary on resume (not fresh sessions) + on-demand hybrid retrieval:

| Seam | What it does | Entry point |
|------|--------------|-------------|
| L1 auto-prepend | `AutoPrepend` — distills the most recent *ended* session's events into a privacy-filtered, token-capped summary; fresh sessions get `""` | `context_harness/autoprepend.go:13` |
| L1 distillation | `DistillSummary` — deterministic key-decisions / errors / file-changes digest from the `session_events` audit trail (migration 040) | `context_harness/summary.go` |
| L2 on-demand | `HybridRetrieve` + `RenderContextBlock` — hybrid (BM25 + semantic, RRF-fused) retrieval of injectable observations rendered as a token-cost-annotated context block; `degraded: true` when no embedder (BM25-only floor) | `context_harness/retrieve.go`, `render.go` |
| MCP tool | `mem_inject_session` — wraps L2 (params: `query`, `all_projects`, `max_tokens` default 2000); contract unchanged by the absorption | `mcp/tools_session_inject.go:17` |
| CLI | `prime` (session-start context page), `compact` (session-close one-liner) | `cmd/skillgrid/loop_cmd.go:19,27` |
| Compaction commit | `MnemonicCommit` — writes durable L2 file + `long_term_memories` row, then async tiering (L0/L1); the `compact` skill hook upserts one continuity observation per session at `topic_key compaction/<session>` | `service/compaction.go:38`, `memory/skills.go:562` |
| HTTP | `GET /context` (injection block) + `GET /context/compaction` (session-scoped compaction payload for the prompt) | `http/server.go:128-129` |
| UI | `/observe/compaction` panel | `skillgrid-ui/src/features/observe/CompactionPage.tsx` |

**Intercept-and-abstract capture** (ADR-0025) — large tool output is abstracted at the boundary instead of flowing into the agent raw:

| Seam | What it does | Entry point |
|------|--------------|-------------|
| Output Sandbox Gate | In the PostToolUse seam, when actual output > threshold (default ~4KB) and no `SKILLGRID_CTX_BYPASS`, the full output is stored to the session-scoped `tool_outputs` FTS5 sandbox and the agent gets a 200-char summary + a `ctx_search <query>` pointer; small output and bypass pass through unchanged | `context_harness/capture.go`, `hooks/tool-call-capture.js`, `http/toolcalls.go` |
| Sandbox Store | `tool_outputs` + `tool_outputs_fts` (migration 050) — session-scoped, purged at session end via `ctx purge` | `store/migrations/050_tool_outputs.sql` |

**Structured query & proactive index** (ADR-0025/0026):

| Seam | What it does | Entry point |
|------|--------------|-------------|
| `ctx_query` | Deterministic counts / lists / existence over the code index (no JS, no line ranges in v1) | `context_harness/query.go` |
| `ctx index` | Reads, chunks (~2KB, bounded ~100/call), and upserts rows into `indexed_files` (migration 051), reusing the observations schema shape (Second Brain has no table of its own — `secondbrain/ask.go:74` calls `session_inject.HybridObservations` over `observations`+`observations_fts`); 7-day TTL | `context_harness/index.go` |
| `ctx_search` | Two-leg RRF fusion of `tool_outputs_fts` (session) + `indexed_files_fts` (project) via `hybrid.Rank` (RRFK=60); results carry per-leg provenance | `context_harness/search.go` |
| `ctx` CLI | `stats` / `index` / `search` / `purge` (purge clears `tool_outputs` + `context_revisions`) | `cmd/skillgrid/ctx_cmd.go` |
| Context Routing | `prime` injects a ~80-word tool map (counts/lists → `ctx_query`, retrieve → `ctx_search`, index → `ctx index`, memory → `mem_*`); advisory, not blocking | `context_harness/routing.go` |

**Context Language Model (CLM)** (ADR-0027/0028) — the agent gets write access to its own context (arXiv:2609.37725, pi-clm). Go owns state and decisions; the Node/OpenCode plugin owns the request path. Opt-in, off by default via the `clm:` config block:

| Seam | What it does | Entry point |
|------|--------------|-------------|
| Context Mirror | A `0600` temp file the agent edits with ordinary file tools: `[[LIVE_CONTEXT version=1 revision=N document=<nonce> baseline=<digest>]]` header + `[[CTX_TURN document=<nonce> index=I role=R id=<id> protected=false]]` blocks; nonce is stable between accepted edits. Rendered before each request by the OpenCode `context` plugin hook (modifies the outgoing model call, not persisted history) | `context_harness/clm/mirror.go`, OpenCode plugin module |
| Overflow Guard | Computed in Go: when the calibrated estimate exceeds `budget − reserve`, the oldest tool results after the last edit are swapped for one-line notes; the withhold decision is stored on the revision and applied by the plugin | `context_harness/clm/overflow.go` |
| Calibration | The size estimate (starts ~4 chars/token) is corrected against the provider's own token count for each request; the corrected factor is stored per session | `context_harness/clm/calibrate.go` |
| Context Revision | One accepted mirror edit, persisted to `context_revisions` (migration 052, session-scoped audit — raw history + mirror are the source of truth); validated at turn-end (nonce match, legal message sequence, tool-call-group repair) and activated for the next request; on resume the highest `revision` is reconstructed and its anchor (count + SHA-256 digest) re-validated, a mismatch falling back to raw context | `context_harness/clm/revision.go`, `store/migrations/052_context_revisions.sql` |

Wired by the harness hooks, not by the agent: `hooks/opencode-session-start.sh` (and the cursor equivalent) run `skillgrid prime` on session start; `*-session-end.sh` run `skillgrid compact` on idle/end. Recovery after compaction is then: `mem_session_summary` → `mem_context` → `mem_inject_session` (the L1→L2→L3 retrieval workflow).


---

## 9. Storage Layer (SQLite + FTS5 + vec0)

One SQLite file **per project**, WAL mode, embedded migrations (`//go:embed migrations/*.sql`).

```go
// store.go
dbPath := filepath.Join(dataDir, projectID+".sqlite")   // ~/.skillgrid/mnemonic/<id>.sqlite
db, err := openWithWALRetry(dbPath)
if err := migrate(db); err != nil { ... }
```

`_ "modernc.org/sqlite/vec"` registers the `vec0` vtab. Core schema (`001_schema.sql`, with 002–039 squashed):

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
    title, content, type, project,
    content='observations', content_rowid='id', tokenize='porter'
);
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    text, path UNINDEXED,
    content='chunks', content_rowid='id', tokenize='trigram'
);
```

Migration timeline (forward-only, additive):

| Migration | Adds |
|-----------|------|
| `001` | initial schema + 002–039 squashed (sessions, observations, files, chunks, web_cache, BLOB embeddings) |
| `011` | `facts` / `skills` / `skills_fts` / `skill_usage` (executable-skill registry) |
| `040` | `session_events` audit trail |
| `042` | `vec_symbols` / `vec_chunks` `vec0(768)` tables (mirror BLOB source of truth) |
| `043` | `lifecycle_log` (audit) |
| `044` | bitemporal (`valid_at` / `invalid_at` / `superseded_by`) |
| `045`–`049` | search aids, entity aliases, session agent, session usage, session checkpoint |
| `050` | `tool_outputs` + `tool_outputs_fts` — Context Harness Sandbox Store (session-scoped, ADR-0025) |
| `051` | `indexed_files` + `indexed_files_fts` — proactive index reusing the observations schema shape (project-scoped, ADR-0026) |
| `052` | `context_revisions` — CLM session-scoped audit of accepted mirror edits (ADR-0028) |

Growth is bounded by TTL soft-expiry (default 7 days), distill/dream consolidation, 90-day `session_events` retention, and a snapshot-retention cap (default 10). `tool_outputs` and `context_revisions` are session-scoped and purged at session end; `indexed_files` carries the 7-day TTL like observations.

### Project state zone (repo-committed, `.skillgrid/`)

A second, *human-readable* storage world — the SDD state zone, canonical in `_shared/rules/sdd-structure.md`:

```
.skillgrid/
├── config.yaml          # REQUIRED static SoT — stack, testing, tracker, rules.*
├── state.yaml           # DYNAMIC — phase, current change, progress
├── ASSUMPTIONS.md       # DURABLE — VERIFIED / INFERRED / LOCKED + ADR index
├── artifacts/           # PRD, terms, ADR files (04-adr-NNNN-slug.md), research
├── specs/               # ACTIVE change folders (YYYY-MM-DD-<topic>/)
│   └── 2026-10-02-mnemonic-llm-provider/
│       ├── briefing.md  acceptance.feature  blueprint.md  tasks.md  adr.md
├── archive/             # CLOSED changes (immutable)
└── sdd/                 # scratch (gitignored): progress.md, briefs/, reviews/
```

---

## 10. Transport Architecture (MCP + HTTP)

The two transports share **one facade** — no logic duplication between the agent path and the operator path.

### 10.1 MCP stdio (agent)

`cmd/skillgrid/mcp.go` → `internal/mnemonic/mcp/server.go`:

```go
const (
    serverName    = "skillgrid-mnemonic"
    serverVersion = "0.1.0"
)
func Start() error {
    s := server.NewMCPServer(serverName, serverVersion, serverOptions()...)
    registerMemoryTools(s)
    registerCodeTools(s)
    // … ~28 register* calls
    return server.ServeStdio(s)
}
```

Stdio transport, spawned by the harness ("trusted by construction"). Tool groups: memory, code, orient, grep, graph, explore, hybrid, community, route, process, knowledge, web, teams, retrieval, compaction, affected, pdg, taint, governance, session, second-brain, skills.

### 10.2 HTTP :7438 (operator + dashboard)

`internal/mnemonic/http/server.go` — Go 1.22+ `ServeMux` method+wildcard routing:

```go
s.mux.HandleFunc("GET /health", s.handleHealth)
s.mux.HandleFunc("POST /sessions", s.requireWriteAuth(s.handleSessionCreate))
s.mux.HandleFunc("POST /sessions/{id}/checkpoint/claim", s.requireWriteAuth(s.handleCheckpointClaim))
```

- **Reads are open, writes are bearer-protected** via `SKILLGRID_HTTP_TOKEN` (`requireWriteAuth`).
- Binds `127.0.0.1` by default (loopback-only).
- Serves the embedded SPA at `/`, OpenAPI at `/openapi.yaml`, Swagger UI at `/swagger`.
- The capture route `POST /sessions/{id}/tool-calls` carries `content` (full output) in addition to `content_preview` so the Output Sandbox Gate can store gated output (ADR-0025). A CLM edit is posted at turn-end via the `checkpoint` capture path and validated/persisted by Go (ADR-0027).
- The OpenCode plugin module registers a `context` hook (v2 plugins) that renders the CLM mirror, applies the Go withhold decision, and reads the model's edit — the only component in the request path that can transform the outgoing model call without rewriting persisted history.

### 10.3 The `skillgrid init` preamble (rendered from config)

`init` writes the `AGENTS.md` `<!-- skillgrid-preamble -->` region from a template + the project's `agents:` config (see §14), then upserts the `## Skillgrid` sentinel block (canonical payload in `_shared/agent-config/block.md`). The preamble is *rich* (Environment & Tooling, Key Directories, Engineering Standards, Security, Dependency Policies, Architecture Constraints, Definition of Done); the sentinel is a *navigation spine* that references — never inlines — the canonical rule files.

---

## 11. Frontend Architecture (Embedded SPA)

`skillgrid-ui/` — **Vite 6 + React 19 + TypeScript + Tailwind 4 + TanStack Router + d3 + mermaid**.

```
skillgrid-ui/
├── src/
│   ├── app.tsx, main.tsx
│   ├── lib/          # api.ts (fetch wrapper + project injection), apiBase.ts
│   ├── components/   # layout/, ui/
│   └── features/     # adr/ decisions/ docs/ git/ kanban/ mnemonic/ observe/
│                     # overview/ plans/ prototypes/ security/ sessions/
│                     # settings/ swagger/ tracker/
└── vite.config.ts    # build.outDir → ../skillgrid-cli/internal/mnemonic/http/ui/dist
```

### Data flow

```
Browser (fetch)  ──▶  HTTP :7438  ──▶  service.Service  ──▶  store (<projectID>.sqlite)
      │
      └─ api.ts auto-injects ?project=<id> (resolved from GET /project/current)
```

`src/lib/apiBase.ts` — the dev/prod split:

```ts
export function apiOrigin(): string {
  if (import.meta.env.DEV) return 'http://127.0.0.1:7438'   // Vite proxies ~23 prefixes
  return ''                                                    // prod: same-origin (embedded)
}
```

In **dev**, Vite proxies the API prefixes to `:7438`. In **prod** the built `dist/` is compiled *into* the binary and served same-origin — there is no separate web server.

---

## 12. Plugin Architecture (Multi-Agent)

One content source, three agent targets, all wired by `skillgrid setup <agent>`. The repo assets are the source of truth; setup **stages** them to machine locations and **merges** MCP entries into the agent config (idempotent, with `BEGIN/END SKILLGRID MNEMONIC` sentinels for Kilo).

```
        repo Hub Content (source of truth)
                │  skillgrid setup <agent>  (stage + merge, idempotent)
   ┌────────────┼─────────────────────────────┐
   ▼            ▼                             ▼
opencode/     kilo/                          cursor/
hooks.yaml    hooks.yaml                     .cursor-plugin/plugin.json
(+checkpoint  (+ SKILLGRID_AGENT=kilo)       (hooks, rules, agents,
  .ts plugin)                                skills, mcpServers)
   │            │                             │
   ▼            ▼                             ▼
~/.config/    ~/.config/                     ~/.cursor/plugins/…
opencode/     kilo/                          + rules/mnemonic.mdc
```

`plugins/opencode/hooks.yaml` — one event → one thin shell script → shared workers in `hooks/`:

```yaml
hooks:
  - id: mnemonic-session-start
    event: session.created
    actions:
      - bash:
          command: "$HOME/.skillgrid/hooks/opencode-session-start.sh"
          timeout: 10000
  - id: mnemonic-policy
    event: tool.before.*
    actions:
      - bash:
          command: "$HOME/.skillgrid/hooks/opencode-policy.sh"
          timeout: 3000
```

`.cursor-plugin/plugin.json` — the whole Cursor manifest:

```json
{
  "name": "skillgrid-mnemonic",
  "hooks": "./hooks/hooks-cursor.json",
  "rules": "./rules/",
  "agents": "./agents/cursor/",
  "skills": "./.agents/skills/**/SKILL.md",
  "mcpServers": "./plugins/cursor/mcp.json"
}
```

**Two distinct "skill" registries** — do not confuse them:
- **SKILL.md skills** (the grid) — loaded by the harness via the plugin's `skills` glob / `~/.agents/` copy. No runtime registry.
- **Executable-skill registry** (SQLite `skills` + `skills_fts`, `internal/mnemonic/skills/execute.go`) — script skills with a `code_path`, run sandboxed by `skillgrid skill`.

---

## 13. The Skill Grid (`.agents/skills/`)

The core concept: a **versionable, reviewable set of markdown workflow skills** the agent harness loads by directory walk. Seven invokable categories plus a non-invokable shared library.

```
.agents/skills/
├── _shared/        # NOT invokable (disable-model-invocation: true) — the reference library
│   ├── rules/          # sdd-structure.md, code-standards.md, commits.md,
│   │                   # testing-standards.md, verification-ladder.md, mnemonic-*.md, ticketing/
│   ├── planning/       # rigor-tiers.md (T0–T3), fast-track.md
│   ├── craft/          # skill-anatomy.md, deterministic-boundary.md, measurement.md
│   ├── verification/   # floor.md, calibration.md
│   ├── references/     # threat-matrix.md, strict-tdd.md
│   ├── agent-config/   # block.md (canonical ## Skillgrid AGENTS.md payload)
│   └── SKILL.md        # index of the library
├── planning/       # brainstorming, interviewing, writing-blueprints, slicing, ticketing
├── execution/      # subagent-execution, simple-execution, parallel-execution,
│                   # isolated-workspace, work-unit-commits
├── verification/   # qa, test-driven-development, structured-debugging,
│                   # parallel-code-review, requesting/receiving-code-review, dogfood, …
├── knowledge/      # mnemonic, research, code-research, deep-research,
│                   # architectural-decision-records, grounded-citations, …
├── design/         # 22 design/visual skills (design, brand, ui-*, imagegen-*, …)
├── craft/          # skill-write, simplify-code, ponytail, youtube-transcript
└── lifecycle/      # onboarding, using-skillgrid (orchestrator), resume, ship, reflect
```

### SKILL.md contract

Frontmatter + conventional body (contract in `docs/skill-anatomy.md`):

```yaml
---
name: brainstorming          # MUST equal the directory name
description: "Use before any creative work — …"   # trigger-based, never process; ≤ 400 chars
metadata:
  author: devopstales
  based_on: superpowers:brainstorming   # provenance lives here, NOT in the description
---
```

Body order: **Announcement → Overview / When to use (+ When NOT to use) → Config → The process (with `<HARD-GATE>` blocks) → Common Rationalizations table → Red Flags → References**.

**Hygiene is machine-enforced** by `scripts/check-skillgrid-skills.mjs` (`task skill-check`): frontmatter invariants, required sections, **line budgets by tier** (`standard: 400, heavy: 500, heavyPlus: 560, orchestrator: 600, reference: 250`), cross-reference rules (`skillgrid:{name}` or `_shared` paths), and hot-path cumulative-load ceilings.

---

## 14. Configuration Architecture

Three config layers with different owners:

### 14.1 Mnemonic indexing config — 3-way merge

`config.Load(startDir)` (`internal/mnemonic/config/load.go`) — precedence lowest→highest:

```
built-in defaults  <  home-local (~/.skillgrid/config.d/indexing.yaml)  <  first repo-local .skillgrid/config.d/indexing.yaml walking up from startDir
```

This is what lets an operator point `mnemonic.embedder` at a local Ollama server without committing the machine-specific config. The distribution defaults ship in `config.d/indexing.yaml`. **Absent/malformed key ⇒ default, never a hard fail.**

### 14.2 SDD pipeline config — the project's "pipeline SoT" (`.skillgrid/config.yaml`)

`schema: skillgrid/v1`. Sections: `ticketing`, `testing` (`runner: "go test ./..."`), `quality`, `security.trivy`, `commands`, `conventions`, `rules` (per-phase guidance + `fast_track` + `tiers`), `bdd`, `mnemonic`, `clm`, `research`, `prototype`, `sketch`.

The `clm:` block (ADR-0027) configures the Context Language Model, off by default:

```yaml
clm:
  enabled: false        # opt-in; CLM is off unless true
  budget: 0             # token budget (default: the model window)
  reserve: 2048         # generation headroom withheld from the budget
  reminders: "50/75/90" # budget fractions that emit [CLM BUDGET] notes
  guard: true           # overflow guard on/off
  cap: 0                # max chars per tool result (0 = off)
```

Env overrides (`SKILLGRID_CTX_CLM`, `SKILLGRID_CTX_CLM_BUDGET`, `SKILLGRID_CTX_CLM_RESERVE`, …) win over the config; an absent block yields all defaults (off).

The rigor dial:

```yaml
rules:
  tiers:
    default: T2
    T0: { name: Prototype, blueprint: none,  qa: self-check }
    T1: { name: Alpha,     blueprint: light, qa: L1+verify  }
    T2: { name: Beta,      blueprint: full,  qa: L2+L3,     review: two-axis }
    T3: { name: GA,        blueprint: full,  qa: L4,        review: parallel+fresh-model }
```

### 14.3 Install-time config (`config.d/`)

- `mcp.yaml` — MCP servers merged into each agent config (remote: `context7`, `deepwiki`, `exa`; local: `mnemonic`, `playwright`, `agent-browser`, `trivy`, `codescene`, `backlog`).
- `tools.yaml` — npm agents + MCP tools to install globally.

---

## 15. SDD Pipeline Architecture

The single source of truth is `.agents/skills/_shared/rules/sdd-structure.md` ("if a skill disagrees with this file, **this file wins**"):

```
brainstorming ─▶ [research | prototype | sketch] ─▶ writing-blueprints ─▶ slicing ─▶ (USER GATE) ─▶ ticketing ─▶ execution ─▶ qa ─▶ requesting-code-review ─▶ receiving-code-review ─▶ ship ─▶ reflect
                                                                          optional gates                mandatory
```

### Artifacts per phase (real example: `specs/2026-10-02-mnemonic-llm-provider/`)

| Phase (skill) | Artifact | Content |
|---------------|----------|---------|
| brainstorming | `briefing.md` | falsifiable requirements, goal, classification, STATUS banner |
| acceptance-test-authoring | `acceptance.feature` | BDD scenarios — the traceability oracle, with `#### Gates` blocks (G1…Gn: CHECK + EXPECT) |
| writing-blueprints | `blueprint.md` | pure-technical plan, files per task, tests |
| slicing | `tasks.md` | vertical tracer-bullet tickets, waves, dependencies, delivery strategy |
| brainstorming | `adr.md` | ADR Review Manifest (pointers to `artifacts/04-adr-NNNN-slug.md`) |
| qa | `report.md` `## Gate Decision` | four-state: `PASS / CONCERNS / FAIL / WAIVED` |
| requesting-code-review | `review.md` | audit record, verdict floor `met` / `met-with-fixes` |
| ship | folder move | `specs/…` → `archive/…` (integration first) |
| reflect | `report.md` retro half | terminal; completes GO/NO-GO + lessons |

**Machine-readable state**: `briefing.md`/`tasks.md` carry a STATUS banner parsed by the `/plans` HTTP endpoint — `> **STATUS:** \`in-progress\` (2026-10-03)`, values `draft → sliced → in-progress → complete → shipped` (+ `stalled/blocked/revised/superseded`).

**Gates wired in code** (not just prose):
- `hooks/gate-state.js` parses `acceptance.feature` `#### Gates` and `--reverify`-runs each CHECK fresh.
- `git-hooks/gate-stop.js` (Stop hook) blocks turn-end while any G<n> is unmet; loop-guard releases after 6 no-progress blocks.
- `git-hooks/pre-commit.js` enforces the **spec zone** — a commit may not mix `.skillgrid/specs/**` with code.

---

## 16. Git Hook Enforcement Architecture

The enforcement layer — the difference between "documented" and "executed". `git-hooks/` are thin entrypoints that `spawnSync` into the shared `hooks/` workers.

```
git-hooks/  (thin shims, installed to ~/.skillgrid/git-hooks/)
   │
   ├─ pre-commit.js  ─▶ hooks/checkpoint-state.js guard ─▶ zone-guard / go-vet / [skillgrid-context]
   ├─ commit-msg.js  ─▶ hooks/guard-msg.js                (commit-message contract)
   ├─ stop.js        ─▶ hooks/stop-tests.js               (run testing.runner, block stop on fail)
   ├─ gate-stop.js   ─▶ hooks/gate-state.js --reverify    (block stop while any Gate unmet)
   └─ post-commit / post-checkout                         (housekeeping)
```

| Hook | Enforces |
|------|----------|
| `pre-commit` → zone guard | No mixed spec+code commits — "commit the spec BEFORE the code" (the BDD spec-as-contract rule) |
| `pre-commit` → go vet | `go vet` on staged Go packages only |
| `stop` → `stop-tests.js` | Runs `.skillgrid/config.yaml testing.runner`; blocks the agent's turn-end and hands back failures |
| `gate-stop` | Runs the acceptance `#### Gates` fresh; blocks turn-end while any G<n> unmet |
| `tool.before.*` → `opencode-policy.sh` | **Fail-open except here, which exits 2 to block a tool** (ADR-0021 pre-tool policy) |

Session/tool observation: `hooks/tool-call-capture.js` (one shared worker for all three agents) fire-and-forget POSTs tool-call summaries to `http://127.0.0.1:7438`, redacting output for `private_tools`; fail-open, always exit 0. Its PostToolUse path now gates on actual output size (Output Sandbox Gate, ADR-0025): above the threshold it POSTs the full `content` for the sandbox instead of only the 500-char preview. Its `checkpoint` mode (the turn-end seam) additionally reads the CLM mirror and POSTs the model's edit for validation when CLM is on (ADR-0027). The OpenCode `context` plugin hook renders the mirror and applies the withhold decision before each model request.

---

## 17. Error Handling & Degradation (Fail-Open Floors)

The single most important cross-cutting invariant: **an agent must never hard-crash because a model, an indexer, or a vector store is down.**

- **LLM seams are injectable with a deterministic fallback.** `memory/types.go`: *"A nil seam (or one that errors) leaves dedup to the deterministic hash fallback — the fallback is always available."* `dedup.llm`, `extraction.llm`, `improve` all follow this.
- **Config loader never hard-fails** — warn + default on absent/malformed keys.
- **"FTS5 is the floor"** (`knowledge/hybrid-degradation.md`) — the vector/semantic legs degrade; the lexical leg always answers.
- **Policy hooks fail-open** except `opencode-policy.sh` (exits 2 to block).
- **Errors** are wrapped with `%w`; the HTTP layer maps resolve/open failures to 400/500 via `withProjectHandle`.
- **Logging** is deliberately minimal — `internal/logging/logging.go` is 27 lines of stderr `fmt` helpers (no JSON logger).

---

## 18. Testing Architecture

### Go

```yaml
# Taskfile.yml
test:
  cmds:
    - cd skillgrid-cli && go vet ./...
    - cd skillgrid-cli && go test ./...
seed-test:
  desc: Run the Mnemonic all-stores seed integration tests (memory + code index + web cache).
  cmds:
    - cd skillgrid-cli && go test -v -run TestSeed ./internal/mnemonic/integration/
```

Scale: **~355 `*_test.go` files, ~1,206 `Test*` functions.** Standard `testing` package + table tests + `httptest`; integration under `internal/mnemonic/integration/` (`TestSeedMemory`, `TestSeedCodeIndex`, `TestSeedWebCache`, `TestSeedAllStoresVisibleOverHTTP`). No test-framework dependency in `go.mod`.

### UI

`vitest` (`package.json` `"test": "vitest run"`), jsdom + `@testing-library/react`. Component tests render real DOM (GraphPage, SessionsPage, decisions view). **Playwright is NOT a UI test runner here** — it's an MCP tool for agents.

### Guards

`task skill-check` (skill anatomy/line budgets), `task site:gen` (regenerates the Hugo guide partial), `scripts/state-lock.mjs` / `test-state-drift.mjs` (state.yaml drift guard, ADR-0008), `doctor --strict` (functional CI health check: embed round-trip, capabilities).

---

## 19. Security Architecture

### Security Layers

```
┌─────────────────────────────────────────────┐
│  Layer 1: Transport                          │
│  - MCP stdio (spawned, trusted by harness)   │
│  - HTTP loopback-only (127.0.0.1) by default │
├─────────────────────────────────────────────┤
│  Layer 2: Auth                               │
│  - HTTP writes bearer-protected              │
│    (SKILLGRID_HTTP_TOKEN / requireWriteAuth) │
│  - reads open, writes gated                  │
├─────────────────────────────────────────────┤
│  Layer 3: Privacy                            │
│  - <private>…</private> wrapping stripped    │
│    before memory storage                     │
│  - private_tools allowlist redacts tool      │
│    output in capture (mirrored to plugin env)│
├─────────────────────────────────────────────┤
│  Layer 4: Enforcement (hooks)                │
│  - policy pre-tool (exits 2 to block)        │
│  - gate-stop (blocks turn-end on unmet gates)│
│  - zone guard (no mixed spec+code commits)   │
├─────────────────────────────────────────────┤
│  Layer 5: Data protection                    │
│  - per-project SQLite isolation (WAL)        │
│  - TTL/decay/retention bounds on growth      │
│  - session_events 90-day retention           │
└─────────────────────────────────────────────┘
```

| Control | Mechanism |
|---------|-----------|
| Loopback HTTP | `serve` binds `127.0.0.1` by default; remote access is opt-in |
| Write gating | `requireWriteAuth` on every mutating route; `SKILLGRID_HTTP_TOKEN` |
| Tool-output redaction | `private_tools` config allowlist; redacted before the capture POST |
| Memory privacy | `<private>…</private>` tags stripped at save time |
| Tool blocking | `opencode-policy.sh` exit 2 (the one non-fail-open hook) |
| Data isolation | one `<projectID>.sqlite` per project; no cross-project reads |

---

## Appendix — Known Gaps (verified 2026-10-05)

- Root `AGENTS.md` preamble was rewritten by `skillgrid init` in the `2026-10-02-mnemonic-project-init` change; its *pre* text ("poetry/pnpm/Flask") was a stale leftover — the live instructions come from the generated `## Skillgrid` block convention.
- `scripts/sdd-gate.sh` is currently an empty file (0 lines) — the gate logic lives in `hooks/gate-state.js`.
- The live semantic leg is the in-memory vector cache; the in-SQL `vec0` path (`042_vec0_tables.sql`) is built but deferred — the BLOB tables are the source of truth.
