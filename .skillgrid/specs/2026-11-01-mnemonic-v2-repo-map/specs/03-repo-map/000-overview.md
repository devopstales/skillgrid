# 000 — Repo-Map: Overview

**Status:** DRAFT · **Owner:** repo-map change
**Depends on:** `2026-10-31-mnemonic-v2-memory` (the `Store`, the `observations`/
`observations_fts`/`sessions`/`session_trails` tables, the `EventBus`/`SaveEvent`, the
`RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams, the `topic_key` vocab) and
`2026-10-31-mnemonic-v2-code-index` (the `files`/`symbols`/`edges`/`index_freshness`
tables, the `internal/codeindex/graph` package, the 4-state freshness model, the
D-TIER per-language tier report). **Queued behind the code-index change** (serial
development, AGENTS.md § Rules): it executes only after code-index ships.
**Consumed by:** the opencode-plugin / pi-plugin changes (pass the `repo_map` MCP
tool through); the `skillgrid:mnemonic` skill (the retrieval-as-reasoning contract names
this map as the first verb).

---

## 1. What this function is

The repo-map is the **first-minute orientation surface** of a project: a single
read-only, ranked, token-budgeted view over the project's **code structure** (the
code-index graph) **and its accumulated memory** (the memory observations). It is
FUNCTION 3 of 5 in mnemonic-v2. Its one-line model:

> **The repo-map is a pure read over the code-index graph and the memory store. It ranks
> the file/symbol tree by PageRank over an IDF-weighted reference graph, renders it as
> signature lines under a token budget, joins the memory's topic hubs and per-file
> developer considerations onto it, labels every answer with a coverage envelope, and is
> auto-injected as the agent's first-minute context via the MCP session-start hook.**

The seven open-source references the design folded, and what each contributed
(`findings.md` is the full digest):

| Reference | Contribution folded |
|---|---|
| **aider** (repomap, 2023) | Signature **lines** (parameters + types), not bare names; a **dynamic** token budget (expanded ~2× on cold start); **chat-aware** ranking (the `context` param is a PageRank prior); **per-turn re-rank**, not just session start. |
| **nekocode/agent-codemap** | The **zero-setup floor**: a symbol tree the agent can use when mnemonic is unavailable; a "just run the command" AGENTS.md instruction. |
| **Twigpine/openclaude** | **PageRank over an IDF-weighted reference graph** (edges weighted by reference count × IDF of the symbol name); **token-budgeted rendering** (walk ranked files top-down, stop at budget; 1024 auto-inject / 2048 command default); **auto-injection** at session start + on-demand tool; **`--focus` as a ranking boost**, not a filter; no second cache (the index IS the cache). |
| **cyanheads/repo-map** | The **per-file field contract** for the emitted doc: Description, **Developer Consideration** ("the one thing a contributor is most likely to get wrong"), Maintenance Flag, Architectural Role, Critical Dependencies. |
| **mikiships/agentmd** | **Minimal-mode discipline** (verbose context reduces task success ~20%; highest value = exact commands + where things are); **drift detection** as a first-class, CI-able check (exit 0/1). |
| **SarthakMishra/codemap** | **Token-optimized docs as the stated goal**; the **toolkit boundary** (the map is the map: no RAG chat, no commits, no PRs, no second vector index). |
| **JordanCoin/codemap** | The **coverage contract** (`status: complete\|partial\|unavailable` + `sources[]` + `issues[]` on every answer); **hub-file warnings** in the orientation view (3+ importers, threshold); the **context protocol** (one command returns the whole session-start envelope); **real-rules resolution** (our equivalent: confidence-labeled edges). |

**The boundary (the SarthakMishra fold):** the map *orients*; the agent *drills*.
`code_search` / `code_read` / `code_resolve` / `code_path` / `mem_lookup` are separate
verbs (the retrieval-as-reasoning contract, code-index spec `000` §1.1). The map is the
**first** verb in the composition, not the whole composition. RAG chat, commits, and
PRs are out of scope. The map does **not** build a second vector index: it reads the
code-index's existing vector store when the semantic context is needed (D-CI8).

**First cut has no LLM.** The map is a pure read. Every per-file field is derived
deterministically: Architectural Role from `edges` (in/out-degree + `kind`), Critical
Dependencies from outbound `edges` (top-N by confidence rank × fan-in), Maintenance
Flag from `files.content_hash` churn + the observation type mix, Developer
Consideration from **memory observations** (FTS-matched to path/package: the notes are
accumulated memory, not LLM-generated), Description = the signature-line floor. Two
**soft one-way doors are explicitly deferred to v2.1** (name them; do not build them):
(a) a stored per-file LLM note fallback (a `file_notes` table) for files with zero
memory; (b) stored `code_ref` pointers (observation → symbol) instead of query-time FTS.

## 2. Role in the five functions

| Function | Relationship to repo-map |
|---|---|
| **memory** | **A leg, read-only.** The topic hubs and the per-file developer considerations are saved observations (FTS-matched to path/package). No memory method is extended; the read goes through the `RawQuery`/`Probe`/`ProbeInt` seams + `observations_fts`. |
| **code-index** | **A leg, read-only.** The file/symbol tree, the ranking graph, the hub flag, the blast-radius and temporal kinds all read `files`/`symbols`/`edges`/`index_freshness` through the existing `graph` package methods (`Orient`/`Impact`/`Callees`/`Callers`) + the seam reads. No `Store` method is extended; no table is added. |
| **repo-map** (this) | The composition. A pure read over the two legs; the ranking engine; the three tiers; the coverage contract; the context envelope; the emitted doc. |
| **knowledge-base** | No dependency either way. The KB cites the graph; the map orients. The KB change consumes neither this change's output nor its package. |
| **opencode-plugin / pi-plugin** | Pass the `repo_map` MCP tool + the session-start hook through (thin wrappers). |

## 3. The three tiers, one data source

All three tiers are a **read-only composition over the existing `Store`** — **no new
tables, no new migration, no `Store` interface extension** (reads via the
`RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams + the existing `internal/codeindex/graph`
methods).

| Tier | What it is | Source |
|---|---|---|
| **Tier 1 — the symbol tree** | Signature lines (parameters + types), ranked by PageRank+IDF with the chat-context prior, token-budgeted. | code-index: `symbols` (signature column) + `edges` (the reference graph) + `files`. |
| **Tier 2 — the memory map** | Topic hubs (the `topic_key` tree → typed hubs → top-N by `importance`) + per-file **developer considerations** (saved observations FTS-matched to the file's path/package). | memory: `observations` + `observations_fts` + `sessions`. |
| **Tier 3 — the emitted doc** | `.skillgrid/repo-map.md`: the cyanheads per-file tree (Description / Developer Consideration / Maintenance Flag / Architectural Role / Critical Dependencies), header-stamped with `generated_at` + a content hash. | The Tier 1+2 composition, rendered to markdown. |

**The `kind` parameter** (progressive disclosure): `overview` (default: packages →
file/symbol counts → topic hubs → recent decisions), `packages` (the package tree +
import edges), `symbols` (the per-package symbol TOC with kinds), `topics` (the memory
map: `topic_key` tree → typed hubs → top-N by importance), `focus`/neighborhood (a
`--focus <symbol>` cross-graph join), `blast-radius` (code reverse-BFS + memory "which
observations reference this"), `temporal` (weekly bins: code re-indexes + memory
growth).

## 4. Ranking

PageRank over the file reference graph (damping 0.85, max 100 iterations, ε=1e-6,
**deterministic** — same run, same output), edges weighted by **reference count ×
IDF(symbol name)** (common names like `get`/`set` count less — the openclaude fold).
The `context: []` param (recently-read paths/symbols) is a **PageRank prior** (the
aider fold); `--focus <path>` is a **ranking boost** (a prior on that subtree);
`--focus <symbol>` is a **neighborhood** query (the cross-graph join). The CLI
disambiguates by argument shape (path/directory → boost; symbol name → neighborhood);
the MCP tool takes an explicit `mode: boost|neighborhood`.

**Output bound:** the token budget is primary (`--tokens N`, default 2048) with `limit`
as a hard row cap. The **dynamic budget policy** (aider): the auto-injected budget is
`inject_tokens` (config, default 1024), **expanded ~2× on cold start** (session start,
no files in context) and shrunk mid-session. A re-rank (PageRank re-run with new
priors) is **<100ms** (no re-parse — the index is unchanged); the cold build is the
`code_index` cost (already a code-index acceptance).

## 5. The coverage contract (every answer)

The output envelope is **versioned** (`schema_version: repo-map/v1`) and carries
`coverage: { status: complete|partial|unavailable, sources: [{name, status:
authoritative|mixed|fallback|timeout|unavailable|failed, languages?, detail?}], issues:
[{code, path, message}] }`. A broken leg degrades **out** and is labeled (T-CI-2
posture); an empty graph with `coverage.status: unavailable` is not the same as an
empty graph with `coverage.status: complete` ("nothing" vs. "couldn't tell"). The
`sources[]` names the per-language extraction tier (D-TIER: `tree-sitter-rich` =
authoritative, `tree-sitter-map` = mixed, `regex` = fallback). This is the T-CI-4 fold
applied per-answer, not just to `code_status`.

## 6. The hub flag

Files with fan-in ≥ `hub_threshold` (config, default 3 — the JordanCoin default) carry
a `⚠️ HUB` tag + importer count **in the orientation view** (not just the blast-radius
kind).

## 7. The context envelope (the session-start protocol)

The session-start MCP hook call returns **one** envelope: the map (Tier 1+2,
token-budgeted) + the coverage contract + the hub warnings + a memory summary (top-3
recent decisions + topic hubs) + the `--for <intent>` prior + the compact/full budget
marker. This is what gets **auto-injected**. The hook **re-ranks per turn** if the
conversation context shifted (new files read / symbols mentioned) — the re-rank is the
PageRank re-run with new priors, **<100ms**.

## 8. Degradation (INV-3)

- **Code-index absent/lag** → Tier 1 from a bounded live tree-sitter scan of the focus
  subtree, or `degraded: no_code_index` (the memory-only map still works).
- **Memory empty** → the code-only map.
- **Both missing** → `coverage.status: unavailable` + counts.

A broken leg drops out and is labeled; it never poisons the other leg (T-CI-2).

## 9. Invariants (re-stated for this function)

The cross-cutting invariants from memory `00-foundation/000-architecture` §4 all bind.
The repo-map applications:

| # | Invariant | Repo-map application |
|---|---|---|
| INV-3 | Explicit degradation | A broken leg degrades out + is labeled in the coverage envelope; never a silent gap, never a poisoned other leg. |
| INV-5 | Retrieval trails | **Every `repo_map` call records ONE retrieval-trail row** (kind, focus, context, coverage status, what was returned) — like `code_orient`. |
| INV-6 | sync_id identity | **Keys are `sync_id` + `topic_key` + file path, never rowid.** Memory observations are addressed by `sync_id`; the trail references the session by id. |

**Repo-map-specific invariants:**

| # | Invariant |
|---|---|
| RM-INV-1 | **The map is a view, not a store extension.** No new table, no new migration, no `Store` method. The whole read goes through the existing seams + the `graph` package. (ADR-0046.) |
| RM-INV-2 | **The map is the first verb, not the whole composition.** It orients; it does not answer. No RAG chat, no commits, no PRs, no second vector index. |
| RM-INV-3 | **Deterministic ranking.** Same run, same output (PageRank damping 0.85 / 100 iters / ε=1e-6; no unseeded randomness in the walk). |

## 10. The two deferred soft doors (v2.1 — named, not built)

1. **Stored per-file LLM note fallback** (a `file_notes` table) — for files with zero
   memory observations. The first cut derives the Developer Consideration from memory
   only; an empty file simply has no consideration line.
2. **Stored `code_ref` pointers** (observation → symbol) — replacing the query-time FTS
   join. The first cut FTS-matches the observation's `title`/`content`/`topic_key`
   against the file's path/package at read time.

Both are soft (a table + a column; reversible). They are named so the v2.1 pass is
additive, and so this change does not reach for them.

## 11. Gating

New change, **queued behind the code-index change** (NOT `current_change` — serial
development, the memory change is current). It executes only after
`2026-10-31-mnemonic-v2-code-index` ships (it needs the `files`/`symbols`/`edges`/
`index_freshness` tables + the `graph` package) and after the in-flight
`2026-10-31-mnemonic-v2-memory` change ships (it needs the `observations` substrate).
No one-way structural doors; the two soft doors above are deferred.
