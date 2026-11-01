# 000 — Architecture (shared substrate)

**Status:** DRAFT · **Owner:** memory change (authored first; cross-referenced by all four other changes)
**Scope:** the v2 module layout, the cross-cutting invariants, and the memory↔wiki model that every function obeys.

---

## 1. What this is

Mnemonic v2 is a total rewrite of `mnemonic/`. It has **five functions**:

| Function | Change folder | Role |
|---|---|---|
| **memory** | `2026-10-31-mnemonic-v2-memory` | The raw, sync-ready source-of-truth fact store. |
| **code-index** | `2026-10-31-mnemonic-v2-code-index` | Symbol/edge graph + hybrid search. |
| **llm-wiki** | `2026-10-31-mnemonic-v2-llm-wiki` | The single synthesized surface (the "brain"). |
| **opencode-plugin** | `2026-10-31-mnemonic-v2-opencode-plugin` | OpenCode agent-runtime integration (TypeScript). |
| **pi-plugin** | `2026-10-31-mnemonic-v2-pi-plugin` | Pi agent-runtime integration (TypeScript). |

The engine is a **single Go module**. The two plugins are TypeScript packages under
`plugins/`, outside the Go module, and consume the engine only through the MCP surface
(spec `004-transport-mcp`).

## 2. Module layout

```
mnemonic-v2/                       (one Go module)
├── cmd/mnemonic/                  thin CLI dispatch + mcp/serve entrypoints
└── internal/
    ├── core/
    │   ├── store/                 dialect-agnostic data access (SQLite | Postgres)
    │   ├── config/                project + data-source + provider config
    │   └── project/               project id detection / resolution / ownership
    ├── intelligence/              the LLM-first layer
    │   ├── llm/                   LLM seam (extract, classify, dedup, judge)
    │   ├── embedder/              embedding seam + vector persistence
    │   └── degrade/               graceful-degradation policy (explicit `degraded` flag)
    ├── memory/                    FUNCTION 1 — see 01-memory/*
    ├── codeindex/                 FUNCTION 2 — see 02-code-index/*
    │   ├── extract/               tree-sitter (+ optional LSP) extraction
    │   └── graph/                 symbol/edge schema + query walker
    ├── llmwiki/                   FUNCTION 3 — see 03-llm-wiki/*
    └── transport/
        ├── mcp/                   MCP stdio server (mem_*/code_*/wiki_* tools)
        └── cli/                   command wiring
```

## 3. The memory↔wiki model (the load-bearing decision)

One line: **memory is the raw source-of-truth store; the llm-wiki is the single
synthesized surface, kept fresh by an LLM call on every save.**

This inverts v1 (where memory was the brain). Consequences, all binding on the memory
function:

- **Memory is not a brain.** It holds atomic durable facts — decisions, bugfixes +
  root-cause, discoveries, preferences, session records, facts + relations + provenance.
  It has **no distillation, no personas, no deep retrieval-for-understanding.** Its
  retrieval is **lookup** (find the exact raw fact).
- **The wiki is the brain.** Distillation, the graduate-rule, entity resolution,
  wiki-query (the deep retrieval), and the eager sync model all live in the llm-wiki
  change. Memory is the *source* the wiki reads.
- **Eager-per-save handoff (OPEN-11).** Every `mem_save` is a two-phase write:
  (a) durable fact write to memory (synchronous, cheap), then (b) an **incremental,
  entity-targeted** wiki regeneration (LLM call) that touches only the pages the fact
  affects. The memory function's contract: it must **emit a save event** carrying enough
  to target the wiki (entity, project, scope, topic_key, fact id) *without* the memory
  function depending on the wiki. See `01-memory/001-observations.md` §save-event.
- **Wiki-first, memory-fallback (OPEN-12).** Agent retrieval is a single path: query the
  wiki; fall back to raw memory lookup only if the wiki is thin/empty on the topic. The
  memory function provides the **lookup** half (`mem_lookup` / `mem_get_observation`).
- **Web-cache dropped (OPEN-3).** No persistent web cache in v2. No `web_cache` tables.
  External research, if wanted, is a memory fact first, then rendered by the wiki.

## 4. Cross-cutting invariants

Every function obeys these; they are enforced, not suggested.

| # | Invariant |
|---|---|
| INV-1 | **Provenance is immutable + everywhere.** Every fact, relation, and derived record carries an origin (session, writer, model, timestamp) that is never overwritten. Supersession is *additive* (a new record + a `superseded_by` link), never a destructive rewrite. |
| INV-2 | **Dual-mode storage.** The store is dialect-agnostic: SQLite (default, local truth) or Postgres (remote). Every table is defined for both. See `001-storage`. |
| INV-3 | **LLM-first with explicit degradation.** The LLM/embedder is a first-class seam. When absent or failing, the system degrades to a deterministic floor and sets `degraded: true` *explicitly* — never a silent fallback. |
| INV-4 | **Local is truth.** Local SQLite is always the source of truth; remote/Postgres is replication, not a second authority. A remote failure never corrupts local. |
| INV-5 | **Retrieval trails are auditable.** Every search/retrieve records a trail (query, route, what was returned). `mnemonic trail` inspects them. |
| INV-6 | **Sync identity is `sync_id`, not the local rowid.** Cross-machine identity is a `sync_id` (ULID/UUID); the local numeric id is ephemeral. All relations and provenance link on `sync_id`. |
| INV-7 | **Agent saves, engine stores.** The engine trusts the *agent* to decide what is worth remembering (selective memory). The store does not firehose raw tool calls. |

## 5. What is NOT in this spec

- Per-function capability detail → each function's `specs/`.
- The canonical schema → `002-schema.md` (the keystone; every function cites it).
- The transport surfaces → `004-transport-mcp.md`, `005-transport-cli.md`.

## 6. Open items

None here. The model is locked (see the plan's §3a and OPEN-1..3, 11..12).
