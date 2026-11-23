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

## Related

- Supersedes: none.
- Consumed by: [[entities/mnemonic]] and every mnemonic page (these are the
  terms those pages use).
- Depends on: [[syntheses/mnemonic-core]] (the design that fixed the terms),
  [[concepts/degradation-floor]] (floor / degraded definitions).
