---
type: Reference
title: "Glossary — Technical"
---

# Glossary — Technical

One meaning per term. Do not use a synonym in its place.

## mnemonic (local-first memory)

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **observation** | One memory unit: a typed, titled, scoped note (decision, bugfix, discovery, pattern, preference, …) with an id, optional `topic_key`, and an optional embedding. The atomic row of the memory store. | Any memory save/recall, schema, tool. | "memory" (ambiguous with the whole store), "note" (untyped). |
| **session** | An agent session boundary in the store: `mem_session_start` mints a `session_id`; every save carries a live one; `mem_session_summary` + `mem_session_end` close it. | Session lifecycle, save validation. | "conversation", "run". |
| **topic_key** | A stable upsert key: saving with the same `topic_key` + `scope` UPDATEs the existing observation (`revision_count++`) instead of INSERTing a duplicate. | Evolving topics (a decision that gets revised). | "id" (numeric, not semantic), "tag". |
| **scope** | A visibility partition on an observation: `project` (default), `user`, `global`. Recall filters by scope; `--all` / `all_projects` widens. | Save/search, cross-project recall. | "namespace" (implies a different store). |
| **floor** | The deterministic, dependency-free path for a capability (hash-64 vectors, brute-force scan, pure-Go build): zero network, zero model files, zero optional extensions — keeps the feature working, just slower or less accurate. | Degradation design, build profiles. | "fallback" (sounds optional), "cache". |
| **degraded** | The envelope flag set when a result was produced without its full path (e.g. recall without the vector leg). Always accompanied by a reason; never silent. | Any result envelope. | "partial" (no contract attached). |
| **RRF** | Reciprocal-rank fusion: combining the FTS5 BM25 ranking and the vector KNN ranking into one result order. | The recall pipeline, ranking tests. | "hybrid search" (describes the legs, not the fusion). |
| **vec0** | The sqlite-vec virtual table: an ANN index *inside* the SQLite file, over a BLOB vector column; dim-strict; created lazily, rebuilt on dim change. | Vector storage, the `sqlitevec` build tag. | "vector database" (implies an external system). |
| **K6 seam** | The transport contract: CLI and MCP are two adapters over one shared Handler; identical inputs produce byte-identical envelopes (parity-tested). | Adding a tool surface, parity tests. | "API" (implies network). |
| **trail** | One row per retrieval (query, route, per-leg ranks, hits) — the audit and eval substrate for recall behavior. | Recall debugging, RRF tuning, eval fixtures. | "log" (that's `log.md` / session logs). |

## mnemonic code tool (knowledge graph + hybrid search)

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **code index** | The in-file knowledge graph of a project's code files: `files`, `chunks`, `symbols`, `edges` (+ FTS + vector tables) inside the same per-project `.sqlite` as memory, via additive migrations (the 100+ range). | The `code` tool, schema, freshness. | "codebase" (the source, not the index). |
| **chunk** | The text unit of the FTS + semantic leg: AST-first — one per function/method/class (~1000-char target; long spans split into 80/10 line windows); `kind` = `ast` or `line`. | Chunking, embeddings, search hits. | "block", "snippet" (a hit's preview is a snippet). |
| **symbol** | One graph node: an extracted definition with a stable, path-independent `uid` (name + kind + span); the single node table for every graph node. | Graph ops, `code_context`, `code_explain`. | "AST node" (a raw tree cell). |
| **edge** | A typed relationship between symbols (`calls`, `imports`, `heritage`, `references`) carrying a `confidence` tag; `to_id NULL` + `to_name` = name-only (unresolved). | Graph ops, impact, path. | "link" (too generic). |
| **EXTRACTED / INFERRED** | Edge confidence: `EXTRACTED` = explicit in the source; `INFERRED` = resolved by inference; `LSP_RESOLVED` upgrades a name-only edge after language-server confirmation. Path confidence = its weakest hop. | Any graph result envelope. | "confidence" with no value. |
| **drop-not-guess** | The extraction invariant: a statically-unbound reference is kept name-only + a row in `unresolved_edges` — never guessed, never dropped. | Extraction, LSP pass, the ledger. | "best-effort" (no ledger attached). |
| **extraction ladder** | Three tiers, all pure-Go: gotreesitter one-pass extractors (the rich 6) → per-language node maps (~25) → regex floor; a panicking grammar quarantines its language to the regex tier (circuit breaker). | Extraction design, `degraded` notes. | "parser" (implies one tier). |
| **gotreesitter** | The pure-Go tree-sitter runtime (`odvcencio/gotreesitter`): parser, lexer, query engine, incremental reparse and external scanners in Go; grammar blobs from upstream `parser.c`; no CGo. v1 links only the supported grammars (standalone packages / build tags); the copyleft grammars are excluded. | The AST layer, build profile. | "tree-sitter" alone (ambiguous with the C runtime). |
| **LSP pass** | The best-effort language-server resolution run after the core index: typed call edges written as `LSP_RESOLVED`; a missing or failing server yields no edges (all-or-nothing per call site, 30 s timeout). | `code_impact` / `code_path` precision, `doctor`. | "autocomplete" (the LSP's job, not the tool's). |
| **index_freshness** | The staleness record: extractor stamp, schema fingerprint, graph counts, `last_indexed_at`. Drives the `stale` flag on search envelopes. | `code_status`, watch, staleness. | "cache" (not invalidatable that way). |
| **stale** | The envelope flag: the working tree drifted past the freshness record; the query still sees the last committed index and never blocks. | Search envelopes, watch. | "outdated" (no contract attached). |
| **watch** | `mnemonic code watch`: the fsnotify loop — debounces bursts (2 s default), collapses each burst into ONE incremental re-index, isolates failures (a failure never stops the watcher). | Freshness during a session. | "daemon" (it is a foreground command). |

## Related

- Supersedes: none.
- Consumed by: [[entities/mnemonic]] and every mnemonic page (these are the
  terms those pages use).
- Depends on: [[syntheses/mnemonic-core]] (the design that fixed the terms),
  [[concepts/degradation-floor]] (floor / degraded definitions).
