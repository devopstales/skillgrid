# 000 — Code-Index: Overview

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `2026-10-31-mnemonic-v2-memory` (`00-foundation/000-architecture` — the
module layout + invariants; `00-foundation/001-storage` — dual-mode; `00-foundation/
002-schema` §4 — the code-index tables; `00-foundation/003-intelligence` — the embedder
seam; `00-foundation/004-transport-mcp` §1/§4 — the one-server-three-families layout +
shared conventions; `00-foundation/005-transport-cli` — the `index` delegation).
**Consumed by:** `2026-10-31-mnemonic-v2-knowledge-base` (`002-sources` §2.2 the code
adapter; `006-validate` the `symbol:` anchor; `007-lint` the coverage-gap package map).
**Reference:** the v1 `mnemonic/` code-index (`internal/codeindex`, `internal/graph`,
`internal/affected`, `internal/extract`) as a *behavior* reference — v2 rebuilds a clean,
thin graph.

---

## 1. What this function is

The code-index is the **live symbol/edge graph** of a project. It is FUNCTION 2 of 5 in
mnemonic-v2. Its one-line model:

> **The code-index is a live, dual-mode symbol/edge graph over the working tree. It is
> extracted by tree-sitter (with an optional LSP tier), stored in the project's shared DB,
> and serves graph queries (callers/callees/impact/path) + hybrid search (FTS +
> structural + semantic). It is the *graph* the knowledge-base's pin-check and module-map
> read live; without it, the KB degrades to ripgrep-only (explicit `degraded`).**

> **It is a *graph*, not a *wiki*.** "Turning a codebase into a wiki is an anti-pattern"
> (decodingai, *Build an LLM Wiki From First Principles*, 2026-10-20): an LLM-compiling
> the code into prose pages would re-sync every commit and *re-derive* what the repo
> already expresses as a navigable graph ("the repo is already the wiki"). This change is
> the **machine-extracted** half — zero LLM at index time (INV-7/CI-INV-1), re-derived
> deterministically from the tree. The **LLM-maintained** half — the *why*, synthesized
> into grounded markdown pages that *cite* this graph (`code:<path>` / `symbol:`) rather
> than re-derive it — is the **knowledge-base change** (KB `002-sources` §2.2, `006-
> validate`). The two never blur: code-index derives a *cache*, the KB compiles a
> *reasoning surface* on top of it.

The code-index is **not** a source store — the code stays in the repo. The graph is a
**derived index** of the working tree: it can always be rebuilt from the tree, and it is
*live* (it reflects the current tree, not a snapshot). If the code moves, the graph is
stale (`index_freshness`) and re-indexing catches it; the KB's pin-check catches the
*knowledge* that moved.

### 1.1 The retrieval contract: retrieval-as-reasoning, not retrieval-as-lookup

The `code_*` surface is designed for **retrieval-as-reasoning** (the arXiv *LLM-Wiki*
paper's term, §2605.25480v2): the agent *searches, reads, follows links, re-plans, and
decides when evidence is sufficient* — retrieval is not a one-shot `top-k` the agent
consumes. This is why the surface exposes **compositional** operations (search / read /
link-follow / impact) the agent *composes* across turns, rather than a single
black-box "answer" tool.

The paper's **code-idiomatic traversal strategies** map 1:1 onto this surface (the
`ki outline` map fold already made them first-class):

| Paper strategy (for wiki) | code-index verb | When |
|---|---|---|
| **Direct access** (known entity → read) | `code_resolve(name)` → `code_read` | the agent already names the symbol |
| **Bridge query** (A→B→answer via links) | `code_path(a, b)` / `code_orient` follow an edge id | connecting two known symbols |
| **Exploratory browsing** (read directory index) | `code_orient(package)` → `code_search under=<pkg>` | the agent doesn't know where a feature lives |

Two of the paper's **termination** rules are adopted as *advisory* agent behavior (the
tool stays stateless; the agent tracks its own budget):

- **Evidence-sufficiency before answering:** the agent should have at least one
  `code_read` / `code_orient` hit it actually read before asserting a fact about the
  code (mirrors the paper's "≥1 `read` before answering").
- **Patience bound:** after `P` (default 3) consecutive empty `code_search` results the
  agent stops re-searching the same query and re-plans (widen the `under` scope, switch
  verb, or escalate to `code_grep`) — a stop condition, not an error.

The paper's **ablation** is the external evidence for why this is worth building
(§2605.25480v2 Table 3): removing the compiled *structure* costs 6–7 F1, removing
*progressive (agent-driven) traversal* costs 12–14 F1, and removing the *Error Book*
costs 3–4 F1. **The traversal is the biggest single term** — which is exactly what
T9b (`code_orient` map) + T5 (`under` scope, `content` mode) make first-class, and
*not* what a flat `code_search` alone would. The LLM-Wiki Error Book is a *document-
wiki* mechanism (it repairs LLM-compiled pages); its code-idiomatic half — a *deterministic*
structural validator over the graph — is what `code_status`'s capabilities report +
CI-INV-1 (rebuild → identical) already provide. The **persistent, learning** Error Book
(accumulating constraints across batches) belongs to the KB change, not here (the D-KB
boundary).

## 2. Role in the five functions

| Function | Relationship to code-index |
|---|---|
| **memory** | **Owns the substrate** the code-index shares (the DB, the schema section, the MCP server, the embedder seam). Memory's `002-schema` §4 *declares* the code-index's 7 tables; this change *defines* them. The two live in the **same project DB** (INV-2 dual-mode; one SQLite file / one PG schema per project). |
| **code-index** (this) | The graph. Extracts symbols/edges, serves graph queries + hybrid search. |
| **knowledge-base** | **Consumes the graph live.** The KB's `symbol:` anchor pin-check calls `code_resolve`; its coverage-gap lint calls `code_packages`; its `architecture`/`workflow` pages are synthesized from the package map + call graph. The KB **does not import** the code-index package — it reads the graph through the `code_*` MCP surface + the shared store. The graph is a **first-class but optional** dependency: the KB degrades to ripgrep-only (`degraded: true`) when the graph is absent (CI-7). |
| **opencode-plugin / pi-plugin** | Pass through the `code_*` MCP family (thin wrappers). |

## 3. Keep / drop / rebuild (vs v1)

v1 is ~145K lines, 35 `code_*` tools, 25+ tables, 7 post-commit advisory passes, 4
vector stores, **polymorphic edge endpoints**, PDG + taint, Leiden communities, process
flows, route extraction, knowledge-graph nodes, SBOM. **v2 rebuilds a clean, thin graph
and defers the heavy optional passes.**

### Keep (core, load-bearing)
`files` + incremental fingerprint (→ `index_freshness`), `symbols` (+`sync_id`), `edges`
(single node table), tree-sitter ladder (6 rich + 24 map + regex fallback), reverse-BFS
impact/blast-radius, neighbor views (callers/callees/dependents/implementors/hierarchy/
tests), shortest path + "where the graph stops", 3-tier confidence, drop-not-guess
(name-only edges), hybrid search (RRF), structural AST grep, embeddings (the wired
3-index store, D-CI8: BLOB + vec0 + HNSW + brute-force floor),
SBOM dependency graph (`dep_edges`).

### Rebuild clean
- **Edge endpoints** — v1's `from_id`/`to_id` pointed at 4 tables (symbols/doc/config/
  sql), forcing an FK-drop migration and making every query a union of id spaces. v2:
  **single `symbols` table**; all nodes (code + route + doc + config + sql) are
  `kind`-tagged rows (CI-2). Edges reference `symbols.id` only.
- **Symbol identity** — v1 had only a local `uid`. v2 adds **`sync_id`** (INV-6).
- **Freshness** — v1's `fingerprint_meta` + `index_meta_kv` (two tables) collapse into
  one **`index_freshness`** record.
- **Temporal window** — v1's `valid_from`/`valid_to` was dead machinery (never written).
  **Dropped** (CI-4). v2 is current-state.
- **Advisory passes** — v1's 7 post-commit passes: knowledge nodes (route/doc/config/
  sql) fold into the **main extract tx** (they're just symbols with a kind); community /
  process / PDG / taint are **deferred optional flag passes** (CI-5).

### Drop / defer (out of scope for the v2 default graph)
- **PDG + taint** — deferred to v2.1 behind `--pdg` (CI-5).
- **Communities + process flows** — deferred to v2.1 behind `--communities` / `--process` (CI-5).
- **Import cycles** — dropped as a stored table; re-derived from `edges` kind=imports on demand.
- **3 wired vector indexes** (BLOB + vec0 + HNSW + brute-force floor) — D-CI8 (the
  settled 2026-11-01 decision, findings.md): v1's IVF adapter (never wired into the query
  path) and LSH (`lsh_buckets`, schema-only, no reader/writer) are **dropped from the v2
  claim**; re-adding either in v2.1 is additive. The query engine picks the best available
  index (HNSW → vec0 → brute-force); the default embedder is the deterministic hash-64
  floor (ANN indexes activate at dim == 768, opt-in).
- **Rationale capture** (NOTE:/WHY:/ADR) — **moved to memory** (a `note` observation pointing at the symbol, CI-9). The graph only *points* at code; memory *holds* the why.
- **`symbol_segments`** — dropped (partial-name resolution → `qualified_name` FTS). (`lsh_buckets` is **dropped from the v2 claim**, D-CI8 — it was schema-only in v1.)

## 4. The freshness model (the query-path contract)

The graph is **live**, not a snapshot. The staleness question — "is the index current
with the working tree?" — is answered by the **`index_freshness`** table, not by
re-walking the tree on every query:

- **A query-time staleness check** is a **metadata-only stat-walk** (size + mtime of
  source files, ~ms) compared against the `files` table + an **extractor stamp**. A stamp
  mismatch (the extractor changed) → every file "changed" (full re-walk). Otherwise only
  added/modified/deleted files are reported.
- **`code_status`** reports the staleness flag + fair coverage + unresolved counts +
  **capabilities** (which features are on: graph/fts/vector/lsp/pdg/communities/process).
- **`code_search` / `code_explore`** consult the freshness check before trusting the
  index; a stale index is *usable* (it answers from the last index) but sets
  `degraded: stale_index` (INV-3 — never a silent gap). This is the **same
  file-hash-based staleness** the KB uses for its own index (`004-index` §5), so the two
  never disagree.
- **Watch mode** (`007-watch`) drives re-indexing on change (fsnotify, debounce); the
  query path never blocks on a re-index.

## 5. The contract the KB + memory resolve against

These are **load-bearing** — the KB and memory specs forward-reference this change, and
these are what those references resolve to:

| Forward-ref (in the other change) | Resolves to (this change) |
|---|---|
| memory `002-schema` §4: `files`, `chunks`, `chunks_fts`, `symbols`, `edges`, `dep_edges`, `index_freshness` | `001-files-chunks` (files, chunks, chunks_fts, index_freshness) + `002-symbols-edges` (symbols, edges, dep_edges) |
| memory `004-transport-mcp` §1: the `code_*` family | `008-transport` §the `code_*` tool surface |
| memory `005-transport-cli`: `index` delegates to code-index | `008-transport` §CLI |
| KB `002-sources` §2.2: the code graph (symbols, edges, files) | this overview + `002-symbols-edges` |
| KB `006-validate` §2: `symbol:` anchor → graph (exists? + call-site?) | `004-graph-queries` §`code_resolve` + `008-transport` `code_resolve` |
| KB `007-lint` §3: top-level Go packages (coverage gaps) | `004-graph-queries` §package map + `008-transport` `code_packages` |
| KB `006-validate` §2: the no-graph fallback (`degraded`) | this overview §CI-7 |

The **two NEW read tools** — `code_resolve` (the `symbol:`-anchor read) and
`code_packages` (the package map) — are the additions that make the KB's forward-refs
resolve. Everything else in the `code_*` surface is a v1 tool kept/renamed.

## 6. The nine decisions (CI-1 … CI-9)

| # | Decision | Choice |
|---|---|---|
| **CI-1** | Table set | **7 tables**: `files`, `symbols`, `edges`, `chunks`, `chunks_fts`, `dep_edges`, `index_freshness`. Single node table. (The exact set memory `002-schema` §4 declared.) |
| **CI-2** | Edge endpoints | **Single `symbols` table** — all nodes (code + route + doc + config + sql) are `kind`-tagged rows. No polymorphic id space. |
| **CI-3** | Symbol identity | **`sync_id`** (INV-6, cross-machine) + `uid` (local, upsert key). |
| **CI-4** | Temporal window | **Dropped** (v1's was dead machinery). Current-state only. |
| **CI-5** | Heavy optional passes | **PDG + taint + communities + process flows deferred to v2.1**, behind `--pdg` / `--communities` / `--process`. Default graph is thin. |
| **CI-6** | LSP tier | **Optional** (`--lsp`, default off) → `LSP_RESOLVED` edges (a 4th confidence tier, only when LSP is active). |
| **CI-7** | Independence from KB | **Graph is first-class but optional.** Code-index runs standalone; the KB degrades to ripgrep-only without it. |
| **CI-8** (amended to D-CI8) | Embeddings | **Wired 3-index store**: BLOB (source of truth) + vec0 + HNSW + brute-force floor (LSH/IVF dropped from the v2 claim — v1 never wired them; v2.1 re-add is additive). The query engine picks the best available index; all are derived over the BLOBs (rebuildable). Default embedder: deterministic hash-64 (ANN at dim == 768, opt-in). Remote: BLOB + pgvector. |
| **CI-9** | Rationale (NOTE:/WHY:/ADR) | **Moves to memory** (a `note` observation pointing at the symbol). The graph only points at code; memory holds the why. |

## 7. Invariants (re-stated for this function)

The cross-cutting invariants from `00-foundation/000-architecture` §4 all bind:

| # | Invariant | Code-index application |
|---|---|---|
| INV-1 | Provenance immutable | Symbols/edges carry origin (file, extractor, stamp, `indexed_at`). Supersession is additive (a new symbol row with a new `sync_id`; the old is pruned, not overwritten in place — the `files`/`symbols` content_hash is the change signal). |
| INV-2 | Dual-mode | Every table is defined for **both** SQLite and Postgres (FTS5→`tsvector`; the local 3-index vector store (vec0/HNSW)→pgvector HNSW; brute-force floor is dialect-agnostic). Same project DB as memory. |
| INV-3 | LLM-first, explicit degradation | The embedder is the shared `intelligence/embedder` seam. Absent/failing → deterministic floor (FTS + structural only) + `degraded: true`. The LSP tier degrades to static extraction. |
| INV-4 | Local is truth | Local SQLite is the source of truth; remote PG is replication. A remote failure never corrupts the local graph. |
| INV-5 | Retrieval trails | Every `code_search` / `code_explore` / `code_impact` records a trail (query, route, what was returned). |
| INV-6 | sync_id identity | `symbols.sync_id` is the cross-machine identity; the local `uid`/`id` is ephemeral. |
| INV-7 | Agent saves, engine stores | The **engine** extracts the graph (tree-sitter/LSP). The agent does not hand-edit `symbols`/`edges`. |

**Code-index-specific invariants:**

| # | Invariant |
|---|---|
| CI-INV-1 | **The graph is derived.** It is always rebuildable from the working tree. Delete the graph tables; re-`code_index`; identical results (up to the optional passes). |
| CI-INV-2 | **Current-state only.** No temporal window (CI-4). A symbol/edge is either in the graph or not. (Re-add in v2.1 if a time-travel need appears.) |
| CI-INV-3 | **Drop-not-guess.** An edge the extractor saw but could not resolve is kept name-only (`to_id NULL`), never a silent guess, and never a *fabricated* resolved edge. Ambiguity is explicit. |
| CI-INV-4 | **Stale is a signal, not a failure.** A stale index (tree moved since last index) answers from the last index + sets `degraded: stale_index`; it never fails a query. (CI-7: the KB's pin-check, not the index, is the failure gate.) |

## 8. The nine specs

| Spec | What it defines |
|---|---|
| `001-files-chunks` | The `files` table, the incremental fingerprint + `index_freshness`, chunking (AST-first), the `unresolved_edges` ledger, the `chunks_fts` FTS. |
| `002-symbols-edges` | The clean `symbols` + `edges` schema (single node table, the edge-kind vocabulary, 3-tier confidence, `sync_id`), the `dep_edges` SBOM graph. |
| `003-extraction` | The tree-sitter ladder (6 rich + 24 map + regex fallback), the optional LSP tier, the extract tx, per-language support, the confidence assignment. |
| `004-graph-queries` | The graph read API: neighbor views, impact (blast radius), path + graph-stops, **`code_resolve`** (the KB `symbol:`-anchor contract), **the package map** (the KB coverage-gap contract). |
| `005-advanced-passes` | The optional/deferred passes: PDG + taint (v2.1 `--pdg`), communities (v2.1 `--communities`), process flows (v2.1 `--process`), the LSP tier (`--lsp`). The flag surface + the keep-or-defer rationale. |
| `006-search` | Hybrid search: FTS (chunks + symbols) + structural AST grep + semantic (single-store embeddings) + RRF fusion; the degradation ladder; provenance per signal. |
| `007-watch` | Live incremental watch (fsnotify, debounce, burst-collapse), the auto-index-on-change loop, the freshness re-check. |
| `008-transport` | The `code_*` MCP tool surface (the family) + the CLI command group + the shared-server-conventions re-use + the `code_status` capabilities report + profiles. |

**Total: 9 spec files** (this overview + 8).

## 9. Open items

CI-O1 … CI-O6 — see §9 of each spec (language set, multi-project, `code_resolve` return
shape, the `unresolved_edges` ledger granularity, chunking, SBOM). All are defaulted in
the plan; none block the spec set.
