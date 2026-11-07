---
type: Reference
title: "skillgrid Architecture"
description: "Live system architecture: components, store, transports, SDD state zone."
status: draft

---

# Skillgrid v2 — System Architecture

> **Version:** 2.1
> **Last Updated:** 2026-10-07
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

Skillgrid is a pair of Go binaries: the **engine** (`mnemonic`) — an *agentic memory + code-intel* engine serving the **agent harness** (Cursor / OpenCode / Kilo) over MCP-stdio and native TS plugins, and the **operator** over the CLI — and the **installer** (`skillgrid`), which stages the shipped hub content into machine locations. (The v1 engine's third client — the operator's local HTTP API + embedded dashboard — was dropped in v2.) It dogfoods its own `.skillgrid/` state zone and SDD pipeline — this repo is simultaneously the product and its first customer.

The engine is its own Go module (`mnemonic-v2/`, module `github.com/devopstales/skillgrid/mnemonic-v2`) — the Mnemonic engine, code index, and MCP stdio server all live there; it builds the `mnemonic` binary (`cmd/mnemonic`). `skillgrid-cli/` is a separate thin module (module `github.com/devopstales/skillgrid/skillgrid-cli`) — the `skillgrid` installer binary; it carries the per-agent setup packages (`internal/setup`, `internal/ui`, `internal/harnessconfig`) and no longer depends on the engine. The v1 engine module (`mnemonic/`) was removed (ADR-0059) — with it went the v1-only HTTP dashboard and fact-memory surface (v2's `serve` is a no-op that says so). (ADR-0033 formalizes the multi-module Go workspace; ADR-0059 records the v1 removal.)

### High-Level System Diagram

```
                     ┌──────────────────────────────────────────────────────┐
 AGENT HARNESS      │              skillgrid  (two Go binaries)           │
 (Cursor / OpenCode │                                                      │
  / Kilo)           │  mnemonic — engine (mnemonic-v2/cmd/mnemonic)        │
    │  5 native TS   │   ├─ mcp ──────── stdio MCP (mcp-go) ──────┐        │
    │  plugins /     │   ├─ mem/index/kb — CLI (one handler set)  │        │
    │  cursor hooks  │   ├─ sync/trail/doctor — CLI               │        │
    │  (session,tool, │   └─ serve ───── no-op (v2 dropped the     │        │
    │   policy,stop) │         HTTP dashboard — ADR-0059)         │        │
    │         ▲       │  skillgrid — installer (skillgrid-cli)     │        │
    │         └───────│   └─ install/setup → agent config merge   │        │
    │  (capture POST, │                                              │        │
    │   checkpoint,   │  codeindex (tree-sitter → chunk → graph →   │        │
    │   stop gate)    │  embed → community)                         │        │
                      │                           ▼                         │
                      │       internal/transport/handler (one handler set) │
                      │         (MCP + CLI over memory · codeindex · kb)   │
                      │                           ▼                         │
                      │   ~/.skillgrid/mnemonic/<projectID>.sqlite (WAL)    │
                      │   embedded migrations · FTS5 (porter/trigram) · vec0│
                      └─────────────────────────────────────────────────────┘
 REPO CONTENT (shipped by `skillgrid install`, the "Hub Content")
 .agents/skills/ (SDD pipeline + _shared rules)      .skillgrid/ (config,
 hooks/  git-hooks/  plugins/  config.d/  rules/      change-specs, change-archive, knowledge vault, meta)
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
├── mnemonic-v2/        Go MODULE — the runtime engine (module
│                       `github.com/devopstales/skillgrid/mnemonic-v2`): Mnemonic
│                       engine, code index, MCP stdio server, KB engine.
│                       cmd/mnemonic (the `mnemonic` binary).
├── skillgrid-cli/      Go MODULE — the `skillgrid` binary: installer
│                       command + per-agent setup (internal/setup,
│                       internal/ui, internal/harnessconfig). cmd/skillgrid.
├── plugins/            Per-agent plugin assets: opencode/ (5 native TS plugins),
│                       kilo/, cursor/mcp.json
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
└── Taskfile.yml        task driver: build / test / skill-check / site / install
```

Two distinct "content" worlds, deliberately separated:

| World | Owner | Lives | Example |
|-------|-------|-------|---------|
| **Runtime** (binary) | the engine Go module | `mnemonic-v2/` (+ `skillgrid-cli/` installer module) | `handler.Handler`, `codeindex`, `core/store` |
| **Hub Content** (shipped) | the installer | `plugins/ hooks/ git-hooks/ config.d/ rules/ .agents/skills/` | the 5 native TS plugins, `block.md`, `indexing.yaml` |

The installer (`skillgrid install`) *stages* Hub Content into machine locations (`~/.skillgrid/plugins/`, `~/.skillgrid/hooks/`, `~/.skillgrid/git-hooks/`) and merges MCP entries into the agent config — it never edits the repo.

---

## 3. CLI Application (Go Binary)

### 3.1 Module & entry point

The engine module `mnemonic-v2/go.mod` (module `github.com/devopstales/skillgrid/mnemonic-v2`) holds the runtime. `skillgrid-cli/go.mod` (module `github.com/devopstales/skillgrid/skillgrid-cli`) is the self-contained installer module — no cross-module `replace` (the v1 dependency was removed with the v1 module, ADR-0059):

```go
// mnemonic-v2/go.mod  — the engine (module github.com/devopstales/skillgrid/mnemonic-v2)
go 1.26.0

require (
    github.com/mark3labs/mcp-go       v1.5.0    // MCP stdio server
    modernc.org/sqlite                v1.61.1   // pure-Go SQLite (cgo-free)
    github.com/odvcencio/gotreesitter  v0.55.1   // AST extraction
    github.com/fsnotify/fsnotify      v1.10.1   // index watch
    github.com/lib/pq                 v1.12.3   // PG render driver (ADR-0041)
    gopkg.in/yaml.v3                  v3.0.1
    ...
)

// skillgrid-cli/go.mod  — the installer (module .../skillgrid-cli)
require (
    github.com/charmbracelet/huh      v1.0.0    // TUI multiselect
    github.com/tidwall/gjson          v1.20.0   // agent-config JSON edits
    github.com/tidwall/sjson          v1.2.5
    gopkg.in/yaml.v3                  v3.0.1
)
```

Two entry points, two binaries. The engine `mnemonic-v2/cmd/mnemonic/main.go` is **not** cobra — a single `flag.FlagSet` plus a `switch rest0` on the first bare argument: `mcp`, `mem`, `sync`, `trail`, `doctor`, `index`, `kb`, `setup`; `serve` is a no-op that prints that v2 dropped the HTTP dashboard. `skillgrid-cli/cmd/skillgrid/main.go` is the installer shell. Version is baked at build time:

```go
// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"
```

```bash
# Taskfile.yml  — two binaries
build:
  cmds:
    - cd mnemonic      && go build -tags ui -ldflags "-s -w -X main.version=..." -o dist/mnemonic ./cmd/mnemonic
    - cd skillgrid-cli && go build -ldflags "-s -w -X main.version=..." -o dist/skillgrid ./cmd/skillgrid
```

The v1 engine embedded the SPA behind a `-tags ui` build flag (it compiled `skillgrid-ui/dist` in via `//go:embed`). v2 dropped the embedded dashboard — and the `skillgrid-ui/` source tree has been removed from the repo (ADR-0059 follow-up).

### 3.2 Layering

```
┌──────────────────────────────────────────────────────────────────┐
│  TRANSPORT LAYER                                                 │
│  mnemonic-v2/cmd/mnemonic/*.go  (main.go dispatch, mcp.go, mem…)  │
│   ├─ MCP stdio (mcp-go)     ── agent harness                     │
│   ├─ CLI (flag-based)       ── operator                           │
│   └─ TUI (huh)              ── interactive prompts (skillgrid-cli)│
├──────────────────────────────────────────────────────────────────┤
│  HANDLER SEAM (K6 — byte-identical across surfaces)              │
│  mnemonic-v2/internal/transport/handler  — the ONE handler set   │
│   every surface (MCP, CLI, the later HTTP mirror) calls; over    │
│   memory.Service + the project detector                          │
├──────────────────────────────────────────────────────────────────┤
│  SUBSYSTEM LAYER                                                 │
│  memory · kb · codeindex (graph, search, vectorstore, watch) ·   │
│  intelligence (llm, embedder, degrade) · migrate · …             │
├──────────────────────────────────────────────────────────────────┤
│  DATA LAYER                                                      │
│  mnemonic-v2/internal/core/store  (SQLite WAL + embedded         │
│   migrations + FTS5 + vec0)  ──▶  ~/.skillgrid/mnemonic/<id>.sqlite│
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

Hand-rolled dispatch (no cobra). The v2 `mnemonic` binary surface from `main.go` (the v1 surface — `install`, the HTTP `serve`, fact memory, `prime`/`compact` session commands — was removed with the v1 module, ADR-0059; the installer lives in the `skillgrid` binary):

| Group | Commands |
|-------|----------|
| Transports | `mcp` (stdio), `serve` (no-op — v2 drops the HTTP dashboard) |
| Project | `index` (build + `status`), `setup` (opencode/kilocode/cursor) |
| Code intel | `index orient`, `grep`, `read`, `path`, `explain`, `impact`, `affected`, `search`, `watch` |
| Memory | `mem` (`save`/`search`/`get`/`lookup`/`context`/`review`/`timeline`/`pin`/`unpin`/`delete`/`session`/`stats`), `sync`, `trail` |
| Knowledge | `kb` (`bootstrap`/`index`/`reindex`/`query`/`read`/`search`/`generate`/`validate`/`lint`/`status`/`cite-source`/`rebuild-meta`/`eval`) |
| Ops | `doctor` (`--strict` for CI) |

---

## 5. Mnemonic Engine — Package Structure

`mnemonic-v2/internal/` — the engine module's package tree. The load-bearing ones:

| Package | Responsibility |
|---------|----------------|
| `core/store/` | SQLite wrapper + embedded migrations (`store/migrations/*.sql`), PG render driver (ADR-0041). WAL retry, `vec0` registration. |
| `core/config/` | Layered config loader + flags; per-project data dir under `~/.skillgrid/mnemonic`. |
| `transport/handler/` | **The K6 seam** — the ONE handler set the MCP server, the CLI, and the later HTTP mirror all call; wraps `memory.Service` + the project detector; envelope `(code, message)` error contract. |
| `memory/` | Observations: save/retrieve (`pipeline.go`), dedup (+ LLM judge), consolidate, importance tiers, relations, replay, trails, autosync, scan/lookup. |
| `kb/` | The knowledge-base engine (KB change 2026-10-31): bootstrap/index/reindex (whitelist scope + `.knowledgeignore`), query (FTS + semantic RRF), validate/lint, status, cite-source, eval (recall@K regression net). |
| `codeindex/` | Incremental indexer, staleness gate (`fingerprint.go`), fsnotify `watch/`, symbol/call `graph/`, `search/` (FTS + by-example + semantic `vectorstore/`). |
| `codeindex/extract/` | tree-sitter symbol/edge extraction per language. |
| `intelligence/` | `llm/` (providers), `embedder/` (onnx/ollama/external/null + select), `degrade/` (fail-open floors). |
| `migrate/` | Store migration runner (SQLite + PG). |
| `transport/mcp/` | MCP stdio server (`server.go`) + `tools_*.go` registering `mem_*`/`code_*`/`web_*`/`team_*` tools. |

Removed with the v1 module (ADR-0059): `facts/` (fact memory — v2's model is observations), `service/` (folded into `transport/handler` + `memory`), `http/` (REST + embedded SPA dashboard — v2's `serve` is a no-op), `hybrid/` (fusion lives in `codeindex/search`), `context_harness/` / `session_inject/` / `loop/`, `checkpoint/`, `policy/`, and the `internal/eval` ablation harness (v2's `kb/eval` is the recall@K regression net).

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

### 8.0 Fact Memory (durable facts)

> **Removed with v1 (ADR-0059).** The `facts` lane and the `POST /facts*` routes went away with the v1 engine — v2's memory model is observations only (`mem_*`). Kept as a design record.

Fact Memory is a **separate** memory lane from observations: a flat, durable, project-scoped set of *facts* — short, checkable statements the agent asserts once and re-reads across sessions. Where observations carry a decay curve and importance tiers, facts are deliberately permanent until explicitly forgotten or decayed.

| Seam | What it does | Entry point |
|------|--------------|-------------|
| Store | `facts.Store` — `Add` / `SearchWith` / `Forget` / `Decay` / `DecayAll` over the `facts` table (`011_facts_skills.sql`) | `mnemonic/internal/facts/` |
| HTTP | five **POST** routes, all write-gated: `POST /facts` (add → `{id}`), `POST /facts/search` (→ `{facts}`), `POST /facts/{id}/forget` (→ 204), `POST /facts/{id}/decay`, `POST /facts/decay-all` | `mnemonic/internal/http/facts.go` |
| MCP tool | `mem_fact_*` family (add/search/forget) exposed over stdio | `mnemonic/internal/mcp/` |
| Native plugin | `mnemonic-memory.ts` — `fact_add` / `fact_search` / `fact_forget` tools for OpenCode/Kilo; the project resolves server-side, the session id is optional and recorded on the `session_events` trail (defaults to `"project"`) | `plugins/opencode/mnemonic-memory.ts` |

> **Route-verb contract.** The five fact routes are POST. The `mnemonic/internal/http` package serves the embedded SPA as a fallback, so a *wrong* verb (a `GET` to a POST-only route) returns the SPA shell (200 `text/html`), not a 405 — and a client that does `.json().catch(() => ({}))` silently turns that into `{}`. `plugin_route_contract_test.go` pins each plugin's HTTP verb to its route to catch exactly that class of bug.

### 8.1 Context Harness (session context lifecycle)

> **Removed with v1 (ADR-0059).** The `context_harness/` package, `ctx_*` MCP tools, and the `ctx` CLI shipped in the v1 engine only — v2's surface is `mem_*`/`code_*`/`web_*`/`team_*`. Kept as a design record.

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
| `ctx index` | Reads, chunks (~2KB, bounded ~100/call), and upserts rows into `indexed_files` (migration 054), reusing the observations schema shape (Second Brain has no table of its own — `secondbrain/ask.go:74` calls `session_inject.HybridObservations` over `observations`+`observations_fts`); 7-day TTL | `context_harness/index.go` |
| `ctx_query` | Deterministic two-leg RRF fusion (K=60, local constant — keeps the package free of a `hybrid` dependency) over `tool_outputs_fts` (session sandbox) + `indexed_files_fts` (project index), keyed on the normalized tool/file name: a cross-leg hit accumulates `1/(K+rank)` per matching leg and outranks a single-leg hit; each leg is fail-open (absent table or MATCH error degrades to the other leg); cross-leg metadata is reported as `sandbox`. `CtxSearch` is the thin stable wrapper over it | `context_harness/ctx_query.go` |
| `ctx` CLI | `stats` / `index` / `search` / `purge` (purge clears `tool_outputs` + `context_revisions`) | `cmd/skillgrid/ctx_cmd.go` |
| Context Routing | `prime` injects a ~80-word tool map (counts/lists → `ctx_query`, retrieve → `ctx_search`, index → `ctx index`, memory → `mem_*`); advisory, not blocking | `context_harness/routing.go` |

**Context Language Model (CLM)** (ADR-0027/0028) — the agent gets write access to its own context (arXiv:2609.37725, pi-clm). Go owns state and decisions; the Node/OpenCode plugin owns the request path. Opt-in, off by default via the `clm:` config block:

| Seam | What it does | Entry point |
|------|--------------|-------------|
| Context Mirror | A `0600` temp file the agent edits with ordinary file tools: `[LIVE_CONTEXT version=1 revision=N document=<nonce> baseline=<digest>]` header + `[CTX_TURN document=<nonce> index=I role=R id=<id> protected=false]` blocks; nonce is stable between accepted edits. Rendered before each request by the OpenCode `context` plugin hook (modifies the outgoing model call, not persisted history) | `context_harness/clm/mirror.go`, OpenCode plugin module |
| Overflow Guard | Computed in Go: when the calibrated estimate exceeds `budget − reserve`, the oldest tool results after the last edit are swapped for one-line notes; the withhold decision is stored on the revision and applied by the plugin | `context_harness/clm/overflow.go` |
| Calibration | The size estimate (starts ~4 chars/token) is corrected against the provider's own token count for each request; the corrected factor is stored per session | `context_harness/clm/calibrate.go` |
| Context Revision | One accepted mirror edit, persisted to `context_revisions` (migration 052, session-scoped audit — raw history + mirror are the source of truth); validated at turn-end (nonce match, legal message sequence, tool-call-group repair) and activated for the next request; on resume the highest `revision` is reconstructed and its anchor (count + SHA-256 digest) re-validated, a mismatch falling back to raw context | `context_harness/clm/revision.go`, `store/migrations/052_context_revisions.sql` |

Wired by the harness plugins, not by the agent: on OpenCode/Kilo the native `skillgrid-compaction.ts` plugin claims the checkpoint on `session.idle` (and `skillgrid-events.ts` fires the session-start capture); on Cursor the `cursor-session-*.sh` shell hooks do the same. Recovery after compaction is then: `mem_session_summary` → `mem_context` → `mem_inject_session` (the L1→L2→L3 retrieval workflow).


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
| `011` | `facts` (Fact Memory) / `skills` / `skills_fts` / `skill_usage` (executable-skill registry) |
| `040` | `session_events` audit trail |
| `042` | `vec_symbols` / `vec_chunks` `vec0(768)` tables (mirror BLOB source of truth) |
| `043` | `lifecycle_log` (audit) |
| `044` | bitemporal (`valid_at` / `invalid_at` / `superseded_by`) |
| `045`–`049` | search aids, entity aliases, session agent, session usage, session checkpoint |
| `050` | `tool_outputs` + `tool_outputs_fts` — Context Harness Sandbox Store (session-scoped, ADR-0025) |
| `052` | `context_revisions` — CLM session-scoped audit of accepted mirror edits (ADR-0028) |
| `054` | `indexed_files` + `indexed_files_fts` (external-content FTS5 over name/kind/language/content) — project-scoped proactive index (ADR-0026) and the second fusion leg of `ctx_query` |

Growth is bounded by TTL soft-expiry (default 7 days), distill/dream consolidation, 90-day `session_events` retention, and a snapshot-retention cap (default 10). `tool_outputs` and `context_revisions` are session-scoped and purged at session end; `indexed_files` carries the 7-day TTL like observations.

### Project state zone (repo-committed, `.skillgrid/`)

A second, *human-readable* storage world — the SDD state zone, canonical in `_shared/rules/sdd-structure.md`:

```
.skillgrid/
├── config.yaml          # REQUIRED static SoT — stack, testing, tracker, rules.*
├── knowledge/           # OKF vault (committed) — the wiki
│   ├── config.json      #   vault sentinel (page types)
│   ├── ASSUMPTIONS.md   #   DURABLE — VERIFIED / INFERRED / LOCKED + ADR index
│   ├── ARCHITECTURE.md  #   this file (live system architecture)
│   ├── index/           #   derived projections (catalog.md, log.md) — engine-owned
│   ├── decisions/       #   ADR files (04-adr-NNNN-slug.md)
│   ├── invariants/      #   locked constraints
│   ├── glossary/        #   terms (01-business-terms.md, 02-technical-terms.md)
│   ├── workflows/       #   SDD pipeline + domain flows
│   ├── gotchas/         #   hard-won operational findings
│   └── architecture/    #   component-level notes
├── change-specs/        # ACTIVE change folders (YYYY-MM-DD-<topic>/)
│   └── 2026-10-02-mnemonic-llm-provider/
│       ├── briefing.md  acceptance.feature  blueprint.md  tasks.md  adr.md
├── change-archive/      # CLOSED changes (immutable; renamed from xarchive/, 2026-11-06)
└── meta/                # engine state (machine-written, never hand-edited)
    ├── state.yaml       # DYNAMIC — phase, current change, progress
    ├── .lock            # session lock
    ├── .prototype/      # feasibility probes (committed, permanently retained)
    └── .sdd/            # scratch (gitignored): progress.md, briefs/, reviews/
```

---

## 10. Transport Architecture (MCP + HTTP)

The two transports share **one facade** — no logic duplication between the agent path and the operator path.

### 10.1 MCP stdio (agent)

`mnemonic-v2/cmd/mnemonic/mcp.go` → `mnemonic-v2/internal/transport/mcp/server.go`:

```go
func runMCP(fl config.Flags, args []string) int {
    h, st, _ := openRuntime(cwd, fl, "")   // the ONE handler set + store
    srv := mcp.NewServer(h, p)             // profile: agent (default) | all
    ch := codeHandlerFor(h.Config(), st)   // the code_* registration seam (B3)
    // … code tools register on the memory server over the git-toplevel root
}
```

Stdio transport, spawned by the harness ("trusted by construction"). Tool groups: memory (`mem_*`), code (`code_*` — orient/grep/read/path/explain/impact/affected/search over the codeindex), web (`web_*`), teams (`team_*`), knowledge (`kb_*`).

### 10.2 HTTP :7438 (operator + dashboard)

> **Removed with v1 (ADR-0059).** v2's `serve` is a no-op; the HTTP mirror is planned (spec 004-transport-mcp §4 — the handler seam is already shaped for it). Kept as a design record.

`mnemonic/internal/http/server.go` — Go 1.22+ `ServeMux` method+wildcard routing:

```go
s.mux.HandleFunc("GET /health", s.handleHealth)
s.mux.HandleFunc("POST /sessions", s.requireWriteAuth(s.handleSessionCreate))
s.mux.HandleFunc("POST /sessions/{id}/checkpoint/claim", s.requireWriteAuth(s.handleCheckpointClaim))
s.mux.HandleFunc("POST /facts", s.requireWriteAuth(s.handleFactAdd))
s.mux.HandleFunc("POST /facts/search", s.requireWriteAuth(s.handleFactSearch))
```

- **Reads are open, writes are bearer-protected** via `SKILLGRID_HTTP_TOKEN` (`requireWriteAuth`).
- Binds `127.0.0.1` by default (loopback-only).
- Serves the embedded SPA at `/`, OpenAPI at `/openapi.yaml`, Swagger UI at `/swagger`.
- The capture route `POST /sessions/{id}/tool-calls` carries `content` (full output) in addition to `content_preview` so the Output Sandbox Gate can store gated output (ADR-0025). A CLM edit is posted at turn-end via the `checkpoint` capture path and validated/persisted by Go (ADR-0027).
- The native plugins are verb-contract-guarded against the live mux by `http/plugin_route_contract_test.go`: every route a plugin calls is probed for registration (404 = stale path), and the POST-only routes (`/policy/evaluate`, `/facts/search`, `/facts`, `/sessions`, `/code/index`) are probed for verb specificity — a POST-only route answers the opposite verb with the embedded SPA shell (200 `text/html`), so a GET sent to a POST route parses to `{}` in the live harness instead of failing (the C1 bug class: `fact_search` used to GET `/facts/search`; `mem_session_summary` POSTed a GET-only `/sessions/{id}/summary`). The test pins both the plugin source and the mux verb.
- The OpenCode plugin module registers a `context` hook (v2 plugins) that renders the CLM mirror, applies the Go withhold decision, and reads the model's edit — the only component in the request path that can transform the outgoing model call without rewriting persisted history.

### 10.3 The `skillgrid init` preamble (rendered from config)

> **v1 mechanism (ADR-0059).** The Go writer for the preamble/sentinel region was the v1 `mnemonic init` command; after the v1 removal no code in this repo re-writes it. The `<!-- skillgrid-preamble -->` / `## Skillgrid` contract in `AGENTS.md` stands as rendered.

`init` writes the `AGENTS.md` `<!-- skillgrid-preamble -->` region from a template + the project's `agents:` config (see §14), then upserts the `## Skillgrid` sentinel block (canonical payload in `_shared/agent-config/block.md`). The preamble is *rich* (Environment & Tooling, Key Directories, Engineering Standards, Security, Dependency Policies, Architecture Constraints, Definition of Done); the sentinel is a *navigation spine* that references — never inlines — the canonical rule files.

---

## 11. Frontend Architecture (Embedded SPA)

> **Removed with v1 (ADR-0059).** No v2 binary embeds the SPA, and the `skillgrid-ui/` source tree has been removed from the repo. Kept as a design record.

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
└── vite.config.ts    # build.outDir → ../mnemonic/internal/http/ui/dist
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

One content source, three agent targets, all wired by `skillgrid setup <agent>`. The repo assets are the source of truth; setup **stages** them to machine locations and **merges** MCP entries into the agent config (idempotent, with `BEGIN/END SKILLGRID MNEMONIC` sentinels for Kilo). For non-opencode harnesses setup also pins `env.SKILLGRID_AGENT=<harness>` in the agent config (`installNativePlugins` → `upsertAgentEnv`): the native plugins default `AGENT` to `"opencode"`, so without the pin kilo's memory/codeindex traffic would be mis-attributed (fix 2026-10-07-replace-opencode-hooks-with-plugins, kilo item).

```
        repo Hub Content (source of truth)
                │  skillgrid setup <agent>  (stage + merge, idempotent)
    ┌────────────┼─────────────────────────────┐
    ▼            ▼                             ▼
 opencode/     kilo/                          cursor/
 5 native TS   same 5 native TS               .cursor-plugin/plugin.json
 plugins       plugins + SKILLGRID_AGENT=kilo (hooks, rules, agents,
               env pin (copied at install)     skills, mcpServers)
    │            │                             │
    ▼            ▼                             ▼
 ~/.config/    ~/.config/                     ~/.cursor/plugins/…
 opencode/     kilo/                          + rules/mnemonic.mdc
 plugin/       plugin/
 ```

The shell-era `plugins/{opencode,kilo}/hooks.yaml` (one event → one `.sh` → a `hooks/` worker) is **retired** for OpenCode/Kilo (ADR-0032). They now run **5 native TypeScript plugins** from a single source dir `plugins/opencode/`, staged to `~/.config/opencode/plugin/`; Kilo's installer copies each file to `~/.config/kilo/plugin/` and pins `env.SKILLGRID_AGENT=kilo`:

| Plugin | Owns |
|--------|------|
| `skillgrid-events.ts` | posts each tool call to `skillgrid serve`; enforces the policy gate (a denied tool throws and blocks); fail-open on errors |
| `skillgrid-compaction.ts` | on `session.idle` claims a memory checkpoint (same route as the Cursor `stop` hook) and prompts the session when due |
| `skillgrid-squad.ts` | multi-agent / squad coordination |
| `mnemonic-memory.ts` | `mem_*` + `fact_*` (add/search/forget) tools over HTTP — the Fact Memory lane (§8.0) |
| `mnemonic-codeindex.ts` | `code_*` code-intel tools over HTTP |

`.cursor-plugin/plugin.json` — the whole Cursor manifest:

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
- **Executable-skill registry** (SQLite `skills` + `skills_fts`, v1 `mnemonic/internal/skills/execute.go`) — script skills with a `code_path`, run sandboxed by `skillgrid skill`. **v1-only — removed with ADR-0059.**

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

`config.Load(path, flags)` (`mnemonic-v2/internal/core/config/config.go`) — precedence lowest→highest:

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

### Artifacts per phase (real example: `change-archive/2026-10-02-mnemonic-llm-provider/`)

| Phase (skill) | Artifact | Content |
|---------------|----------|---------|
| brainstorming | `briefing.md` | falsifiable requirements, goal, classification, STATUS banner |
| acceptance-test-authoring | `acceptance.feature` | BDD scenarios — the traceability oracle, with `#### Gates` blocks (G1…Gn: CHECK + EXPECT) |
| writing-blueprints | `blueprint.md` | pure-technical plan, files per task, tests |
| slicing | `tasks.md` | vertical tracer-bullet tickets, waves, dependencies, delivery strategy |
| brainstorming | `adr.md` | ADR Review Manifest (pointers to `decisions/04-adr-NNNN-slug.md`) |
| qa | `report.md` `## Gate Decision` | four-state: `PASS / CONCERNS / FAIL / WAIVED` |
| requesting-code-review | `review.md` | audit record, verdict floor `met` / `met-with-fixes` |
| ship | folder move | `change-specs/…` → `change-archive/…` (integration first) |
| reflect | `report.md` retro half | terminal; completes GO/NO-GO + lessons |

**Tracker mapping** (canonical in `_shared/rules/ticketing/backlogmd-formatting.md`): a Backlog **milestone is named after the full spec folder** (`YYYY-MM-DD-<topic>` from `change-specs/` or `change-archive/`), never a short slug or numeric ID — the `milestone → spec` mapping is 1:1 and recoverable from the tracker alone.

**Machine-readable state**: `briefing.md`/`tasks.md` carry a STATUS banner parsed by the `/plans` HTTP endpoint — `> **STATUS:** \`in-progress\` (2026-10-03)`, values `draft → sliced → in-progress → complete → shipped` (+ `stalled/blocked/revised/superseded`).

**Gates wired in code** (not just prose):
- `hooks/gate-state.js` parses `acceptance.feature` `#### Gates` and `--reverify`-runs each CHECK fresh.
- `git-hooks/gate-stop.js` (Stop hook) blocks turn-end while any G<n> is unmet; loop-guard releases after 6 no-progress blocks.
- `git-hooks/pre-commit.js` enforces the **spec zone** — a commit may not mix `.skillgrid/change-specs/**` with code.

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
| `tool.before.*` → `skillgrid-events.ts` | **Fail-open except here, which throws to block a tool** (ADR-0021 pre-tool policy; the retired `opencode-policy.sh` is gone for OpenCode/Kilo) |

Session/tool observation: on OpenCode/Kilo the native `skillgrid-events.ts` plugin posts each tool call to `http://127.0.0.1:7438`, redacting output for `private_tools`; fail-open (errors and unreachable servers are swallowed), and it enforces the policy gate by throwing when the server denies a tool. Its PostToolUse path gates on actual output size (Output Sandbox Gate, ADR-0025): above the threshold it POSTs the full `content` for the sandbox instead of only the 500-char preview. On **Cursor** the shell-era `hooks/tool-call-capture.js` worker still does this (`cursor-tool-capture.sh` → `tool-call-capture.js`); its `checkpoint` mode (the turn-end seam) additionally reads the CLM mirror and POSTs the model's edit for validation when CLM is on (ADR-0027). The OpenCode `context` plugin hook renders the mirror and applies the withhold decision before each model request.

---

## 17. Error Handling & Degradation (Fail-Open Floors)

The single most important cross-cutting invariant: **an agent must never hard-crash because a model, an indexer, or a vector store is down.**

- **LLM seams are injectable with a deterministic fallback.** `memory/types.go`: *"A nil seam (or one that errors) leaves dedup to the deterministic hash fallback — the fallback is always available."* `dedup.llm`, `extraction.llm`, `improve` all follow this.
- **Config loader never hard-fails** — warn + default on absent/malformed keys.
- **"FTS5 is the floor"** (`knowledge/hybrid-degradation.md`) — the vector/semantic legs degrade; the lexical leg always answers.
- **Policy hooks fail-open** except the policy gate, which blocks a denied tool — `skillgrid-events.ts` (throw) on OpenCode/Kilo, the Cursor `before*` hooks on Cursor.
- **Errors** are wrapped with `%w`; the HTTP layer maps resolve/open failures to 400/500 via `withProjectHandle`.
- **Logging** is deliberately minimal — `internal/logging/logging.go` is 27 lines of stderr `fmt` helpers (no JSON logger).

---

## 18. Testing Architecture

### Go

```yaml
# Taskfile.yml
test:
  cmds:
    - cd mnemonic && go vet ./...
    - cd mnemonic && go test ./...
seed-test:
  desc: Run the Mnemonic all-stores seed integration tests (memory + code index + web cache).
  cmds:
    - cd mnemonic && go test -v -run TestSeed ./internal/integration/
```

Scale (current tree): **mnemonic-v2 — 80 `*_test.go` files; skillgrid-cli — 23.** Standard `testing` package + table tests. (The v1 module carried ~355 test files plus the all-stores seed integration tests under `internal/integration/` (`TestSeedMemory`, `TestSeedCodeIndex`, `TestSeedWebCache`, `TestSeedAllStoresVisibleOverHTTP`) — removed with v1, ADR-0059.) No test-framework dependency in `go.mod`.

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
| Tool blocking | policy gate — `skillgrid-events.ts` throw (OpenCode/Kilo) / Cursor `before*` hooks (the one non-fail-open seam) |
| Data isolation | one `<projectID>.sqlite` per project; no cross-project reads |

---

## Appendix — Known Gaps (verified 2026-10-09)

- Root `AGENTS.md` preamble was rewritten by `skillgrid init` in the `2026-10-02-mnemonic-project-init` change; its *pre* text ("poetry/pnpm/Flask") was a stale leftover — the live instructions come from the generated `## Skillgrid` block convention.
- The live semantic leg is the in-memory vector cache; the in-SQL `vec0` path (`042_vec0_tables.sql`) is built but deferred — the BLOB tables are the source of truth.

## Change log

- **2026-10-09** — `feat(ctx): add ctx_query and two-leg RRF ctx_search fusion` (276e8c61): migration 054 (`indexed_files` + `indexed_files_fts` external-content FTS5) becomes the second fusion leg of the Context Harness; `ctx_query.go` implements the two-leg RRF (K=60 local constant) with per-leg fail-open, cross-leg score accumulation on a normalized tool/file key, and sandbox-preferred metadata; `CtxSearch` is the stable wrapper. CLI `ctx search` / `ctx prime` / `ctx stats` surface. Shipped with `2026-10-07-replace-opencode-hooks-with-plugins` fixes (`eebf185a`): plugin verb contract — `fact_search` POSTs `/facts/search`, `mem_session_summary` GETs `/sessions/{id}/summary`, guard in `http/plugin_route_contract_test.go`; kilo setup pins `env.SKILLGRID_AGENT=kilo`. Milestone titles now match spec folder names 1:1 (`3285abe6`). Spec folder archived (`7efe3fa0`).
- **2026-10-05** — baseline for this revision (v2.0).
