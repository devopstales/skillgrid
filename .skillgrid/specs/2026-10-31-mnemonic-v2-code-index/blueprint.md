# Mnemonic v2 — Code-Index Function (Implementation Blueprint: steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T3 — new public `code_*` MCP tool contracts + dual-dialect storage + a data-migration/parity regression. Config `rules.tiers.default` is T2; escalated to T3 per Change Classification (`high-risk`: public API the KB + plugins consume + the shared `Store` is extended + the v1 bug-class migration-parity regression). QA floor: L4 (full suite + build + coverage + go vet; mutation/security advisory per config).

**Build shape:** Tracer thread — the integration risk is the dialect `Store` extension + the extraction engine + the graph walker. A thin end-to-end `code_index → SQLite graph → code_search` path works first (Task 1's door check), then thickens (graph queries → hybrid search → semantic store → transport → watch → advanced passes).

**Classification:** high-risk (new trust boundary: the working tree becomes the graph; public API: the `code_*` MCP contract; data migration: the 7-table additive set on the shared project DB + the migration-parity regression the KB re-uses).

**Goal:** Build the code-index function of mnemonic v2 — the live symbol/edge graph of the project's working tree (tree-sitter extraction with an optional LSP tier, dual-dialect storage, graph queries, hybrid search, live watch) — behind the same `Store` the memory change owns and the `code_*` MCP surface the knowledge-base + plugins consume — so the agent can find files, find where a feature is implemented, see how a function connects, and know the blast-radius of a change, with every answer confidence-labeled and every degradation explicit. **Retrieval contract (spec `000` §1.1):** the surface is *retrieval-as-reasoning* — the agent composes `code_search` / `code_orient` / `code_read` / `code_path` across turns and decides when evidence is sufficient (≥1 read before asserting; a patience bound of 3 empty searches → re-plan), not a one-shot `top-k`. The 4-capability framing (find files / find feature / connectivity / blast-radius) is the code-idiomatic form of the paper's direct / bridge / exploratory strategies. External validation of this contract + the anti-pattern defense (graph-not-wiki) is in `findings.md` §"10-source review" (F-V2-1…F-V2-6).

**Pre-execution checklist (T0 — repo state the plan presumes; none of it is a task, all of it must be true before Task 1's code commit):**
- [ ] `findings.md` (this folder) committed — the 4-researcher v1 mechanics digest + the live v1 CLI probe + the precise v1 migration-bug root cause + the D-CI8/STAMP/PROFILE/TIER/KB decisions.
- [ ] **The memory change is SHIPPED** (`2026-10-31-mnemonic-v2-memory`): `mnemonic-v2/internal/core/store` (the `Store` interface + `SQLiteStore`), `internal/core/syncid`, `internal/intelligence` (the `Embedder` seam + `degrade`), `internal/core/config` + `internal/core/project`, and `internal/transport/mcp` (the one server + the two-profile mechanism + the write queue) all exist and `cd mnemonic-v2 && go build ./...` exits 0. Serial development (per AGENTS.md § Rules): code-index is not `current_change` until memory ships.
- [ ] ADR-0042 (tree-sitter dep), ADR-0043 (fsnotify dep), ADR-0044 (conditional onnxer dep), ADR-0045 (the `Store` interface extension) written under `.skillgrid/artifacts/` and added to the ASSUMPTIONS.md index (next numbers after 0041).
- [ ] `state.yaml` `pipeline.current_change` repointed to `2026-10-31-mnemonic-v2-code-index`.
- [ ] `acceptance.feature` (this folder) committed **before** any `mnemonic-v2/` code commit (pre-commit zone guard, per AGENTS.md § Rules).

**Architecture:** The memory change's fresh single Go module `mnemonic-v2/` (module `github.com/devopstales/skillgrid/mnemonic-v2`, Go 1.26, cgo-free) gains a `code-index` function. `internal/core/store` already owns the dialect split (SQLite default = local truth; Postgres = replication target; FTS5 ↔ tsvector) — this change **extends** the `Store` interface with the code-index methods (additive, ADR-0045) and adds the 7 code-index tables as additive migrations (numbered to follow the memory change's range). `internal/codeindex` is the domain: `extract` (the tree-sitter ladder + self-healing pool + the real content-hash extractor stamp + the chunker), `graph` (the dialect-agnostic walker: impact, neighbor views, path/graph-stops, resolve, packages, affected), `search` (the hybrid RRF fusion + the structural leg + the semantic leg), `watch` (the fsnotify loop), and `vectorstore` (the wired 3-index store: BLOB + vec0 + HNSW + brute-force floor, per D-CI8). `internal/transport/mcp` gains the `code_*` family (16 agent + admin) as thin adapters over the shared handler set. The knowledge-base change (a later change) adds its `kb_*` tables **on top of the shared machinery this change provides** (the `Store` table-adding seam, the reusable FTS rendering unit, the 4-state freshness model, the migration-parity regression) — it does not fork the dialect code or the FTS pattern (D-KB, T21).

**Tech Stack:** Go 1.26 (go.work already pins 1.26.0); the memory change's allowed set (`modernc.org/sqlite` incl. `/vec`, `mark3labs/mcp-go`, `gopkg.in/yaml.v3`, `lib/pq`) **plus** three new deps, each ADR-gated (per AGENTS.md § Rules): `gotreesitter` (v1 v0.55.1, pure-Go, cgo-free — the rich-6 + 25-map extraction), `fsnotify` (v1 v1.10.1, the watch loop), and conditionally `onnxer` (v1 v0.7.0, only if the 768-dim ONNX embedder is wired — the hash-64 default does not need it). All three are already in `mnemonic_back/go.mod` (proven cgo-free) but are new to `mnemonic-v2/go.mod`.

**Spec:** `specs/02-code-index/000-overview.md` … `008-transport.md` (this folder, 9 spec files). Product *why*: `.skillgrid/artifacts/00-prd.md`. In-force ADRs: ADR-0040 (module extraction) + ADR-0041 (Postgres driver) from the memory change are the substrate; this change adds ADR-0042…0045.

**Findings:** `findings.md` (this folder) — the v1 mechanics + the live probe + the precise v1 migration-bug root cause + the D-CI8/STAMP/PROFILE/TIER/KB decisions. Every design decision below that rests on a research fact cites it.

**KB-index scope (the "cover the llm-wiki indexing" decision, D-KB):** this change does **not** own the KB's `kb_pages`/`kb_pages_fts`/`kb_claims` tables or the `kb_index`/`kb_reindex` rebuild — those are the KB change's work (KB `004-index` §3.1: "the KB and memory share the file but not the interface"). It **does** own the **shared machinery** the KB index builds on (T21): the `Store` table-adding seam, the reusable FTS rendering unit, the 4-state file-hash freshness model, and the migration-parity regression (T-CI-3) structured so the KB can extend it over its `kb_*` tables. The KB's `kb_query` (wiki-first retrieval) + `kb_validate` (pin-check) are the KB change's — this change only guarantees the graph reads they call (`code_resolve`/`code_packages`/`code_search`) are available.

## Terms

Pointers only — canonical terms live in `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md`.

- **symbol** — a node in the graph (one row in `symbols`; spec `002-symbols-edges` §2). All nodes (code + route + doc + config + sql) are `kind`-tagged rows in the **single** `symbols` table (CI-2).
- **edge** — a relationship between two symbols (one row in `edges`; spec `002` §4). `to_id NULL + to_name` is a deliberate *name-only* (unresolved) state (CI-INV-3, drop-not-guess).
- **FileGraph** — the per-file extract output `{symbols, edges, unresolved}` the extraction ladder produces. New term, introduced by this change's spec zone.
- **extractor stamp** — the deterministic content hash of the extractor code/config (`index_freshness.extractor_stamp`, D-STAMP); a change forces a full re-walk.
- **confidence tier** — the 3-tier default (EXTRACTED 1.0 / INFERRED 0.85 / AMBIGUOUS 0.5) + the 4th LSP_RESOLVED 0.95 when the LSP tier is on (spec `002` §6).
- **stale_index** — the `degraded` flag set when the graph is stale with the working tree (CI-INV-4) — stale is a signal, not a failure.
- **the 4 live-probe relevance truths (T-CI-1…4)** — the "good relevant answer" requirements the v1 live probe forced (findings.md): real-symbol-finding, broken-leg-degrades-out, migration-parity, and code_status-version/tier/coverage. New terms, introduced by this change's findings.

## Must-Haves (goal-backward verification)

Derived from the spec zone + the **live-probe relevance truths** (findings.md, T-CI-1…4). The verifier checks the result against this list, not just that tasks exist.

**Truths** (observable behaviors that must hold):

- T1: `code_index` (and the `code_index` MCP tool) walks the project git root, excludes `.git`/`node_modules`/`vendor`/build output/`.skillgrid/knowledge` + the include/exclude globs, extracts the 6 rich languages via gotreesitter one-pass + the 25 map languages via the node-type data table + the regex fallback, and writes `files` + `chunks` + `symbols` + `edges` + `unresolved_edges` + `index_freshness` in a per-file target-state tx; a re-extract of an unchanged file (same `content_hash`) is a no-op (idempotent upsert). A file over `MaxFileSize` (default 500 KiB, config) is not indexed and is counted in `index_freshness.files_oversized` (a first-class outcome, not silent).
- T2: the 7 tables (`files`, `chunks`, `chunks_fts`, `symbols`, `edges`, `dep_edges`, `index_freshness`) are **additive migrations** on the memory `Store` — both dialects (SQLite + Postgres) run the full set from an empty DB (regression-tested, re-running the memory change's `migrate_test.go` pattern over the code-index tables).
- T3: the extractor stamp is a **real content hash** (D-STAMP), not a const — a change to the extractor code/config changes the stored `extractor_stamp` → a full re-walk on the next freshness check. The stamp is computed at build (a `//go:generate`-produced const or an exposed FNV-1a over a serialized extractor-fingerprint string, mirroring v1's `fingerprintDigest` shape `fingerprint.go:71-84` but with a real caller).
- T4: **`code_resolve`** (NEW — confirmed absent from v1 by the live research) returns `{exists, name, qualified_name, file, line, sync_id, called_from[] (file paths, CI-O3), ambiguous, candidates[]}` — `exists:false` → `unresolved` (fail closed); `ambiguous:true` → ranked `candidates[]` (never a silent pick, CI-INV-3). It lifts v1's `Resolve` primitive (`resolve.go:35-76`: 0=NotFound / 1=Target / >1=Ambiguous) but selects only columns its own migration set guarantees (T-CI-3).
- T5: **`code_packages`** (NEW — confirmed absent from v1) returns the top-level Go package map `[{path, files, symbols}]` from `files` (language='go') + `symbols` — the KB's coverage-gap contract (spec `004` §4). The unit is the top-level Go package (first path segment under the project root with Go files). No-graph fallback is a shallow `go list`-style scan (CI-7).
- T6: `code_get_callers/callees/dependents/implementors/hierarchy/tests_for` return confidence-labeled edges; name-only edges resolve **at query time** against `symbols.name` (0 candidates → kept name-only `To.ID==0`; 1 → resolved, original confidence; >1 → one AMBIGUOUS edge each, lifted from v1 `fetchEdges` `graph.go:217-226`).
- T7: `code_impact` (reverse-BFS, depth-cap 5 default, confidence filter EXTRACTED > LSP_RESOLVED > INFERRED > AMBIGUOUS) returns WILL BREAK (depth 1) / LIKELY AFFECTED (deeper) + the excluded count (counted, not dropped, lifted from v1 `impact.go:149-153`); traverses only stored edges (a name-only edge that resolved to nothing has no row, so ambiguity can never inflate the radius, CI-INV-3).
- T8: `code_affected` (inputs: `changed[]` | stdin git-diff | `base` git-ref → `git merge-base <ref> HEAD` + diff + git-history owners) → affected test files + the relationship chain (from, to, kind, confidence, depth, via). **Fixes the v1 gap** (NOT-PORT): test-file detection recognizes Java (`*Test.java`/`*Tests.java`) + the other languages' conventions, not just v1's Go/Python/Ruby/JS path-only set.
- T9: `code_path` (BFS shortest path between two symbols, confidence ignored for connectivity) or "where the graph stops" — a `PathResult` is always *something* (a path, or the refused name-only matches + the dispatch kind + the line). **Fixes the v1 quirk** (NOT-PORT): the `DispatchKind` is the dispatch kind that *causally* ended the flow (the interface/bus/callback at the frontier), not v1's last-iterated-over-the-visited-set value.
- T9b: **`code_orient` is the navigation entry point** (the progressive-disclosure "map", fold from the Neo4j `ki outline` pattern, findings.md) — `code_orient(symbol | package | file, depth?)` returns a **compact, bounded** view: the unit's metadata + signature + **the file TOC** (every symbol in the containing file, line order) + the unit's **outbound edges** (`calls`/`imports`/`references`), each row carrying a stable symbol id. `depth` (default 1) bounds how many hops of outbound edges are expanded; an outbound edge's target is **not** expanded inline (following it = another `code_orient` on that row's id). Re-rooting on a **package** returns that package's symbol TOC (the §4 unit); on a **file** its full TOC + import edges. **It is the agent's "find a feature" first verb:** `code_orient` a package → see the symbol map → `code_resolve` a symbol → `code_read` the span. Cost stays flat as the repo grows (each query adds one layer, never the whole vault). **Replaces** v1's `code_orient`/`code_signature`/`code_file_toc` three tools (combined into one map read). *Test: `TestOrientMapReRootsAndBounded` — orienting on a package returns the TOC + outbound edges with ids; `depth=1` does not expand the edge targets; re-rooting on an edge target's id returns that symbol's neighborhood.*

  > **Source:** the Neo4j `ki` reference implementation (`ki outline` — re-rootable, `--depth`-bounded, URI-per-row) over wiki markdown; its NICD benchmark (vector+graph ≫ vector-only) is the external case for making navigation a first-class verb.
- T10: `code_search` (hybrid: FTS + structural + semantic, RRF K=60) — every hit carries per-signal provenance; the floor is always FTS (deterministic, no model); a down embedder → `degraded: no_vector`; a stale index → answers from the last index + `degraded: stale_index` (CI-INV-4). **A leg that errors degrades to `degraded: <leg>` + drops out of the fusion** (T-CI-2 — the v1 live probe showed a `no such column` warning while the broken leg still shaped the output; this is the fix). **`under?` subtree scope** (fold from the Neo4j `ki` `search --under`, findings.md): an optional `files.path` package/path prefix narrows *all three legs* to a subtree — the context-rot defense for large repos (orient on a package → search within it). No new table; the prefix is recorded in the trail (INV-5). *Test: `TestSearchUnderScope` — a scoped search returns only hits under the prefix; an empty-scope search returns empty (not an error).*
- T11: the semantic store (D-CI8) is **wired**: BLOB (`embeddings`+`chunk_embeddings`) + vec0 + HNSW + a brute-force-over-BLOBs floor; the query preference order is HNSW → vec0 → brute-force (the v1 *live* path, lifted); the model-swap guard (clear vectors+indexes on `embedding_model` change, lifted from v1 `indexer.go:1133-1160`) is present; **LSH + IVF are dropped** (not shipped — v1's `lsh_buckets` is schema-only, IVF is an adapter not wired). The store is model-agnostic (stores whatever `dim` the embedder produces); the default embedder is the deterministic hash-64 floor; a 768-dim embedder (ONNX nomic with the pool actually wired, or an OpenAI-compat provider) is opt-in and activates HNSW/vec0 when `dim == 768`.
- T12: `code_grep` (structural by-example AST grep) — index-free (re-parses the working tree with gotreesitter, works on files not yet indexed); **the pattern language is fixed** (NOT-PORT the v1 UX gap): the node-type+capture form (`(function_declaration) \fn`, `(call_expression) \call`, `\(ARGS*)`, `\*`/`\_`, `\NAME`) is documented AND the `\f(ARGS)` shorthand is accepted; an invalid pattern for a language → `degraded: no_grammar:<lang>` (that language skipped, others still match, never a silent skip).
- T13: `code_read` (path + line-range) returns the live source span (reads the working tree, not a snapshot) with **secret redaction** (a `token=`/`password=`/`secret=` literal in the fetched slice is masked, the v1 property). **Two modes** (fold from `ki get --type full` / `--type content`, findings.md): `mode: full` (default) = the live span (a symbol id expands to its `start_line`…`end_line`); `mode: content` = the **drill handle** — the symbol's signature + its callee/caller **ids** (confidence-labeled `to_id`/`to_name`), *not* their bodies (a file target → the file TOC instead of the body). Each `content` read adds one layer of detail, never the whole neighborhood — the cheap read that keeps the context window flat while the agent decides which span to pull. *Test: `TestCodeReadContentMode` — a symbol in `content` mode returns the signature + callee/caller ids but none of their bodies; a file in `content` mode returns the TOC.*
- T14: the watch loop (`code_watch`/`mnemonic watch`) — fsnotify on the project source root, debounce 2 s (clamped 100 ms–60 s), burst-collapse (a burst of edits → one re-index), the metadata-only stat-walk fingerprint re-check (size+mtime vs `files` + the extractor stamp), failure isolation (a re-index failure → log + continue, the watcher never stops; `index_freshness.last_indexed_at` reflects the last *successful* index). The watch is a *trigger, not a gate*: a query during a re-index sees the last committed index (+ `degraded: stale_index`) and never blocks on the re-index (CI-INV-4).
- T15: every `code_search`/`code_resolve`/`code_impact`/`code_explain`/`code_path` records a retrieval trail (query, route, what was returned, the `degraded` flag; INV-5); `mnemonic trail recent|show` inspects them (the shared trail store the memory + KB use).
- T16: the `code_*` surface is the **16 agent-profile + the admin set** (spec `008` §3) on the memory MCP server's two-profile mechanism (D-PROFILE — the memory change's `agent`/`all` profiles, not v1's listed/unlisted `SKILLGRID_MCP_CODE_TOOLS` env-var filter). The v2.1-deferred tools (`code_pdg_query`, `code_taint`, `code_communities`, `code_god_nodes`, `code_explain_community`, `code_processes`, `code_process`) are **declared in the surface + report "run `code_index --<flag>`"** (their tables are absent until the v2.1 pass runs — they are *deferred*, not *missing*).
- T17: the `code_index` MCP tool is a **queued write** (async, returns `run_id` + `status` immediately; a repeat POST for an already-active index returns 409); the optional passes (LSP/PDG/communities/process) run **after** the core graph tx commits (separate connection, advisory — they never roll back the committed graph; a failing pass → warn + continue). Read tools are synchronous.
- T18: the `code_status` read reports the **4-state freshness** (`fresh`/`lag`/`empty`/`unknown` — NOT the v1 boolean `stale`) + the counts (files/symbols/edges/chunks/unresolved) + the `capabilities` JSON (`{graph,fts,vector,lsp,pdg,communities,process}`) + the **per-language extractor tier** (D-TIER: `tree-sitter-rich`/`tree-sitter-map`/`regex`, derived from the extraction run) + the **schema version** + the fair coverage (per language) + the last index time + the watch status.
- T19 (backstop): the **migration-parity regression** (T-CI-3) — a fresh store AND a store opened at *each prior migration version* can run every `code_*` read without a SQL-column error (the v1 bug class, findings.md). The test iterates a *table-set* (not a hardcoded list) so the KB change extends the same regression over its `kb_*` tables (D-KB). *Held-out test: Task 2's `TestMigrationParityAllVersions`.*
- T20 (backstop): **real-symbol-vs-fake-symbol search divergence** (T-CI-1) — `code_search` on a real symbol returns it in the FTS leg; a non-existent symbol does **not** return the same top hits (the v1 live probe returned identical hits for a real symbol and a fake one). *Held-out test: Task 5's `TestSearchRealVsFakeDivergence`.*
- T21: **the shared machinery the KB index builds on is complete and reusable** (D-KB) — (a) the `Store` dialect abstraction exposes a **table-adding seam** the KB change's `KBStore` uses to add its `kb_*` tables without forking the dialect code; (b) the **FTS rendering pattern** (external-content FTS5 DDL + the four-trigger sync set + the ranking-weights shape) is a **reusable, tested unit** (`store/fts.go`) the KB's `kb_pages_fts` re-uses verbatim; (c) the **4-state file-hash freshness model** is the shared staleness definition the KB's `degraded: stale_index` + `kb_lint` reuse; (d) the **migration-parity regression** (T-CI-3) is structured as a table-set the KB extends.

**The four live-probe relevance truths (T-CI-1…4) — the difference between "the tool exists" (v1, broken) and "the tool gives a good relevant answer" (v2):**

- **T-CI-1** → T20: `code_search` on a real symbol returns that symbol (FTS floor matches identifier queries); real ≠ fake (divergence). *Task 5.*
- **T-CI-2** → T10: a leg that errors degrades to `degraded: <leg>` + drops out of the RRF fusion; the broken leg's empty result never sets the answer. *Task 5's `TestBrokenLegDegradesOutOfFusion` (the v1 live probe showed the warning but the broken leg still shaped the output — this is the fix).*
- **T-CI-3** → T19: every `code_*` read runs on a fresh store AND on a store opened at each prior migration version; a version mismatch is a *version* error, never `no such column`. *Task 2's `TestMigrationParityAllVersions` (the v1 live probe: a 1570-symbol, 060-migration, up-to-date index where every graph read failed `no such column: s.uid` because no `ALTER TABLE symbols ADD COLUMN uid` bridge existed).*
- **T-CI-4** → T18: `code_status` reports schema version + per-language extractor tier + fair coverage (not just counts + boolean stale). *Task 7's `TestCodeStatusReportsVersionTierCoverage`.*

**Artifacts** (files that must exist with real implementation, not stubs):

- `mnemonic-v2/internal/core/store/migrations/100_code_index.sql` (SQLite) + `migrations_pg/100_code_index.sql` — the 7 tables (numbering starts at 100 to avoid colliding with the memory change's 001–060 range; the exact number is set at execution to be the next free).
- `mnemonic-v2/internal/core/store/store.go` — the `Store` interface **extended** with the code-index methods (the `code_index` write path + the graph reads); the extension is additive (ADR-0045). Includes the **table-adding seam** (T21) — the dialect-neutral mechanism the KB change's `KBStore` uses to add its `kb_*` tables to the same `Store` without forking the dialect code.
- `mnemonic-v2/internal/core/store/fts.go` — the **reusable FTS rendering unit** (T21): the external-content FTS5 DDL + the four-trigger sync set + the ranking-weights shape, extracted from `sqlite.go` into its own tested unit so the KB change's `kb_pages_fts` re-uses it verbatim.
- `mnemonic-v2/internal/core/store/types_code.go` — the Go result/param types every code-index Store method uses (FileRow, Symbol, Edge, FileGraph, ImpactResult, PathResult, SearchHit, ResolveResult, PackageInfo, …).
- `mnemonic-v2/internal/codeindex/extract/` — the tree-sitter ladder (rich-6 + 25 maps + regex), the self-healing pool + circuit breaker, the extractor stamp (D-STAMP, `stamp.go`), the chunker (`chunker.go`), the knowledge-node extractors (route/doc/config/sql, written in the core tx).
- `mnemonic-v2/internal/codeindex/graph/` — the walker (impact, neighbors, path, resolve, packages, affected) — dialect-agnostic.
- `mnemonic-v2/internal/codeindex/search/` — the hybrid fusion (RRF K=60), the structural leg (by-example AST grep, the fixed pattern language), the semantic leg (calls the vectorstore).
- `mnemonic-v2/internal/codeindex/watch/` — the fsnotify loop (debounce, burst-collapse, fingerprint re-check).
- `mnemonic-v2/internal/codeindex/vectorstore/` — the BLOB + vec0 + HNSW + brute-force engine (D-CI8), the dimension gating + model-swap guard.
- `mnemonic-v2/internal/transport/mcp/tools_code.go` — the 16 agent + admin `code_*` tools as thin adapters over the shared handler set.
- `mnemonic-v2/internal/transport/handler/handler_code.go` — the one shared code-index handler set (the MCP + CLI surfaces meet here; never diverge).
- `mnemonic-v2/cmd/mnemonic/index.go` — the `mnemonic index/search/status/orient/explain/path/impact/affected/grep/read/watch` CLI group (delegated by the memory change's `005-transport-cli`).
- `mnemonic-v2/internal/codeindex/*_test.go` + `mnemonic-v2/internal/core/store/store_code_test.go` — tests for every truth above, **including the four T-CI held-out tests** (T-CI-1…4).
- `.skillgrid/artifacts/04-adr-0042-tree-sitter-dep.md`, `04-adr-0043-fsnotify-dep.md`, `04-adr-0044-onnxer-dep.md`, `04-adr-0045-store-interface-extension.md` — the four dependency/extension decisions.

**Key links** (critical connections that must work together):

- K1: `code_index` MCP tool → `handler_code.Index` → `codeindex.Index` (scan → per-file extract tx → orphan prune → update `index_freshness`) → the optional passes (separate connection) → queued-write `run_id`. A failure at any core step rolls back that file's tx.
- K2: `code_resolve` → `graph.Resolve` (lifts v1 `resolve.go:35-76`) → `called_from` = the forward callers resolved to `files.path` (CI-O3) → the KB's `symbol:` pin-check reads this (spec `000` §5, the KB's forward-ref resolving).
- K3: `code_search` → FTS leg + structural leg + semantic leg → RRF fusion (K=60) → per-hit provenance → the trail writer. **A broken leg degrades out (T-CI-2).** The FTS floor always answers.
- K4: the KB (a later change) calls `code_resolve` + `code_packages` + `code_search` through the `code_*` surface — the two NEW tools are the KB's forward-refs resolving (spec `000` §5). The KB does **not** import the `codeindex` package (CI-7 — graph is first-class but optional; KB degrades to ripgrep-only with `degraded: no_graph`).
- K5: `cmd/mnemonic`'s `index`/`search`/… and the MCP `code_index`/`code_search`/… produce byte-identical JSON (same handler, same result marshal) — a parity test asserts it.
- K6: the KB's `kb_index`/`kb_reindex` (the KB change) build its `kb_pages`/`kb_pages_fts`/`kb_claims` tables **on top of the shared machinery this change provides** (T21): the `Store` table-adding seam, the reusable FTS rendering unit (`store/fts.go`), the 4-state freshness model, and the migration-parity regression (T-CI-3). The KB adds its tables to the *same* `Store` + *same* migration set + *same* parity loop — it does not fork the dialect code or the FTS pattern.

**One-way-door decisions** (hard to reverse — flag for explicit user approval at execution):

- D1: **the `code_*` MCP tool contract** — the KB + the opencode/pi plugins consume it as their only interface. Renaming a tool, adding a required param, or changing a return shape after the KB builds breaks them. Task 7.
- D2: **the `Store` interface extension** (code-index methods) — code-index and the KB later extend the same store (they add their tables to the same project DB, spec `002-schema` §4). Adding a method later is additive; reshaping one is a contract break for every function. Task 1.
- D3: **the migration numbering** (the 7 tables are additive on the memory store) — a number collision with the memory change's `migrations_pg/` or the KB's future tables is a rebase. Task 2.
- D4: **the extractor stamp algorithm** (D-STAMP) — changing it later changes the stored value once (a one-time full re-walk), never a schema migration. Task 3.

## Global Constraints

Every task's requirements implicitly include this section.

- **Go 1.22+ minimum to build** (per ADR-0031); the go.work floor is 1.26.0, so `mnemonic-v2/go.mod` declares `go 1.26.0` (already set by the memory change).
- **No new dependencies without an ADR** (per AGENTS.md § Rules): the allowed set is the memory change's (`modernc.org/sqlite` incl. `/vec`, `mark3labs/mcp-go`, `gopkg.in/yaml.v3`, `lib/pq`) plus `gotreesitter` (ADR-0042), `fsnotify` (ADR-0043), and conditionally `onnxer` (ADR-0044, only if the 768-dim ONNX embedder is wired). Anything else needs a new ADR before it lands.
- **cgo-free** (v1 VERIFIED: `skillgrid doctor` reports `cgo: free`; the pure-Go `gotreesitter` binding is the extractor of record, the v1 choice kept).
- **Conventional commits only; no AI-attribution trailers** (commit-msg hook enforces, per AGENTS.md § Rules).
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard enforces, per AGENTS.md § Rules): this blueprint + `acceptance.feature` + `tasks.md` + `findings.md` are committed before any `mnemonic-v2/` code commit.
- **One change at a time, no parallel branches** (serial development, per AGENTS.md § Rules). A commit that closes a Backlog task names that task ID in the `[skillgrid-context]` block (per ADR-0013).
- **Trivy is advisory-only** — findings reported, never blocking the QA gate.
- **Additive migrations only** (spec `002-schema` §1): new tables/columns; never drop in a minor version.
- **All seven invariants** (spec `000-architecture` §4) + the four code-index invariants (CI-INV-1 derived, CI-INV-2 current-state, CI-INV-3 drop-not-guess, CI-INV-4 stale-is-a-signal) are binding on every task: INV-1 provenance immutable (symbols/edges carry origin — file, extractor stamp, `indexed_at`); INV-2 dual-mode (every table both dialects); INV-3 LLM-first with explicit `degraded: true`; INV-4 local is truth; INV-5 retrieval trails; INV-6 identity is `sync_id`; INV-7 the engine extracts (the agent does not hand-edit `symbols`/`edges`).
- **Test command:** `cd mnemonic-v2 && go test ./internal/codeindex/... ./internal/core/store/...`; **build:** `cd mnemonic-v2 && go build ./...`; **format:** `gofmt -w ./...`; vet inline via `go vet ./...`.

## Threat Matrix

Per `_shared/references/threat-matrix.md`. The change touches **Mnemonic tool contracts** (the whole `code_*` surface is new), **process integration** (the MCP stdio server, the fsnotify watcher goroutine, the LSP tier spawns language-server subprocesses), and **git repository selection** (project detection + `code_affected`'s base-ref mode runs `git merge-base`/`diff`).

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Mnemonic tool surface (`code_*`) | Applicable: all 16 agent + admin tools are new public contracts (spec `008` §3); the two NEW tools (`code_resolve`, `code_packages`) are the KB's forward-refs | The tool contract is pinned by tests: exact tool name, required params, return shape, error codes, and profile membership are asserted; a contract drift = a failing test, not a silent break. | Task 7 Step 1 (`TestCodeToolContract` — golden JSON per tool), Task 7 Step 2 (`TestCodeProfileMembership` — agent vs all), Task 5 Step 1 (`TestCodeResolveContract` + `TestCodePackagesContract` — the two NEW tools' exact shapes) |
| Process integration | Applicable: the MCP stdio server is spawned by the agent harness; the fsnotify watcher is a background goroutine; the LSP tier (opt-in) spawns gopls/pyright/etc. subprocesses | stdio protocol correctness (newline-delimited JSON-RPC) is tested in-process against a fake harness client; the watcher's debounce + burst-collapse + failure-isolation are tested against a temp-dir with a fake event source; the LSP tier is all-or-nothing per call site (a failing/timing-out server → warn + continue, never a partial edge set). | Task 7 Step 3 (stdio round-trip test), Task 6 Step 1 (`TestWatchBurstCollapse` + `TestWatchFailureIsolation`), Task 8 Step 1 (`TestLSPAllOrNothingPerCallSite`) |
| Git repository selection | Applicable: project detection resolves cwd → git binding → basename (the memory change's `005-transport-cli` convention); `code_affected --base <ref>` runs `git merge-base <ref> HEAD` + `git diff --name-only` | Detection is a pure function of (cwd, env, git toplevel) tested per selector (re-uses the memory change's `TestProjectDetectionSelectors` pattern); `code_affected`'s base mode runs git in an isolated env (a temp clone), never the user's repo. | Task 5 Step 4 (`TestAffectedBaseRefIsolatedGit` — a temp clone, `--base HEAD~1`, the diff set is correct + the user's repo is untouched), Task 3 Step 5 (re-runs the memory change's project-detection selectors over the code-index scan root) |
| Documentation-like paths | Applicable: the extraction walk classifies files by extension — `.md` docs become `kind='doc'` symbols, `.yaml`/`.toml`/`.json` become `kind='config'`, `.sql` becomes `kind='table'`/`kind='column'` (the knowledge nodes, spec `003` §5). An executable-looking markdown (a `README.sh`) must not be misclassified. | Classification is by extension via the `extToLang` map (the v1 map, kept) — only mapped extensions are indexed; an unmapped extension is skipped (not indexed). The KB vault (`.skillgrid/knowledge`) is *excluded* from the code walk (it is not code; the KB has its own `kb_index`). | Task 3 Step 6 (`TestKnowledgeNodeClassification` — a `.md` → `doc`, a `.yaml` → `config`, a `.sql` → `table`/`column`, an unmapped `.xyz` → skipped; a `README.sh` is not treated as code) |
| Commit state | N/A: the engine does no git automation of its own (project detection reads `git rev-parse --show-toplevel` read-only; `code_affected --base` reads `git merge-base`/`diff` read-only in an isolated env). It never stages or commits. | — | — |
| Push state | N/A: no git push from the engine | — | — |
| PR commands | N/A: no PR automation | — | — |
| Shared-convention drift | N/A: no `_shared/{planning,craft,verification,knowledge}/` or `_shared/references/*.md` file is edited by this change (the `mnemonic-code-indexing.md` convention is the *consumer* of this change's tool surface, not something this change edits) | — | — |

## File Structure

```
mnemonic-v2/
├── go.mod / go.sum                     module github.com/devopstales/skillgrid/mnemonic-v2 (go 1.26.0; + gotreesitter/fsnotify[/onnxer])
├── cmd/mnemonic/
│   └── index.go                        the code-index CLI group (delegated by the memory change's 005-transport-cli):
│                                       index [flags] | status | search <q> | orient <s> | explain <s> | path <a> <b>
│                                       | impact <s> | affected [base] | grep <pat> | read <path> | watch
└── internal/
    ├── core/
    │   ├── store/
    │   │   ├── store.go                the Store interface EXTENDED with code-index methods (additive, ADR-0045)
    │   │   │                           + the table-adding seam (T21) the KB's KBStore uses
    │   │   ├── types_code.go           FileRow, Symbol, Edge, FileGraph, ImpactResult, PathResult, SearchHit,
    │   │   │                           ResolveResult, PackageInfo, … (Go types only, the S-2 rule)
    │   │   ├── fts.go                  the REUSABLE FTS rendering unit (T21): external-content FTS5 DDL +
    │   │   │                           four-trigger sync set + ranking-weights shape (extracted from sqlite.go)
    │   │   ├── sqlite.go               SQLiteStore: + the code-index dialect render (calls fts.go)
    │   │   ├── postgres.go             PostgresStore: + the code-index tsvector render (Task 8, mirrors sqlite.go)
    │   │   ├── migrations/100_code_index.sql    the 7 tables (SQLite render, additive)
    │   │   └── migrations_pg/100_code_index.sql the 7 tables (Postgres render, Task 8)
    ├── codeindex/
    │   ├── index.go                    Index(ctx, root, opts) — the scan → per-file extract tx → orphan prune →
    │   │                               index_freshness driver (the write path, 001 §6)
    │   ├── extract/
    │   │   ├── ladder.go               the tree-sitter ladder: rich-6 (gotreesitter one-pass) + 25 maps + regex
    │   │   ├── languages.go            the 25 per-language node-type → symbol-kind DATA table (not 1481 lines of code)
    │   │   ├── fallback.go             the regex battery (the deterministic floor, INV-3)
    │   │   ├── pool.go                 the self-healing pool + per-language circuit breaker (MaxRespawns=3, TripAfter=3)
    │   │   ├── stamp.go                the extractor stamp (D-STAMP): the content hash, a real caller
    │   │   ├── chunker.go              the AST-first chunker (≤1000 chars, 250 merge, 80/10 window, lines fallback)
    │   │   └── knowledge.go            the knowledge-node extractors (route/doc/config/sql → symbols + edges)
    │   ├── graph/
    │   │   ├── walker.go               the dialect-agnostic reverse-BFS walker (impact, dependents, affected)
    │   │   ├── neighbors.go            callers/callees/dependents/implementors/hierarchy/tests_for
    │   │   ├── resolve.go              code_resolve (lifts v1 resolve.go:35-76) + the query-time name-only relabel
    │   │   ├── path.go                 code_path + buildGraphStops (the causal dispatch-kind fix)
    │   │   └── packages.go             code_packages (the top-level Go package map)
    │   ├── search/
    │   │   ├── hybrid.go               the RRF fusion (K=60) + per-hit provenance + the broken-leg-degrades-out (T-CI-2)
    │   │   ├── fts_leg.go              the FTS floor (BM25 over chunks_fts + symbol_fts)
    │   │   ├── structural.go           the by-example AST grep (index-free) + the FIXED pattern language (T12)
    │   │   └── semantic.go             the semantic leg (calls vectorstore; the embedder seam; the degrade)
    │   ├── vectorstore/
    │   │   ├── store.go                the BLOB + vec0 + HNSW + brute-force engine (D-CI8) + the preference order
    │   │   ├── dim.go                  the dimension gating (768 pin) + the model-swap guard
    │   │   └── hash.go                 the hash-64 default embedder (the deterministic floor)
    │   └── watch/
    │       └── watch.go                the fsnotify loop (debounce 2s clamp 100ms–60s, burst-collapse,
    │                                   stat-walk re-check, failure isolation)
    └── transport/
        ├── handler/handler_code.go     the one shared code-index handler set (MCP + CLI meet here; never diverge)
        └── mcp/tools_code.go           the 16 agent + admin code_* tools as thin adapters over handler_code
```

Responsibility notes: `internal/codeindex` never names `*sql.DB` (the S-2 rule — it sees only the `Store` interface + the `Embedder` seam). `internal/codeindex/graph` is dialect-agnostic (the same walker runs over SQLite and Postgres). `internal/transport/handler_code.go` is the seam where the MCP + CLI surfaces meet — both are thin adapters. The KB change (a later change) adds its `kb_*` tables via the `Store` table-adding seam + the reusable `fts.go` unit — it does not import `internal/codeindex`.

---

### Task 1: `Store` interface extension + 7-table additive migration + `sync_id` minting (door check)

> ⚠ one-way: **D2 — the `Store` interface extension** (code-index methods). Code-index and the KB later extend the same store; reshaping these methods later is a contract break for every function. **D3 — the migration numbering** (100 to follow the memory change's 001–060 range; a collision is a rebase).
>
> **Door check — this task tests the hypothesis.** If a `code_index` cannot write `files`+`symbols`+`edges` through the extended `Store` (SQLite, WAL) **and** a `code_search` FTS floor cannot find a saved symbol by name (with the `< 3 runes` LIKE fallback the memory change's `store/fts.go` already provides), stop: the code-index's foundation (the `Store` extension + the single-node-table graph schema, CI-2) does not hold and the blueprint is invalidated. Task 2 re-proves the same properties against the full migration set + every prior version (T-CI-3) — the door has two locks; Task 1 is the stop gate.

**Files:**
- Create: `mnemonic-v2/internal/core/store/types_code.go`, `mnemonic-v2/internal/core/store/migrations/100_code_index.sql`
- Modify: `mnemonic-v2/internal/core/store/store.go` (add the code-index methods to the `Store` interface + the table-adding seam), `mnemonic-v2/internal/core/store/sqlite.go` (the code-index dialect render + the `code_index` write path)
- Test: `mnemonic-v2/internal/core/store/store_code_test.go`

**Interfaces:**
- Consumes: the memory change's `store.Open`, the migration runner, `syncid.NewSyncID` (all built).
- Produces:
  - The `Store` interface **extended** (additive — no existing signature changes) with:
    ```go
    // code-index (additive to the memory change's Store; ADR-0045)
    UpsertFile(ctx context.Context, f FileRow) error                 // target-state files upsert (by path; content_hash is the change signal)
    UpsertSymbols(ctx context.Context, fileID int64, syms []Symbol) error   // ON CONFLICT(uid)
    ReplaceEdgesForFile(ctx context.Context, fileID int64, edges []Edge) error  // DELETE WHERE file_id=? + upsert
    ReplaceUnresolvedForFile(ctx context.Context, fileID int64, refs []UnresolvedRef) error
    ReplaceChunksForFile(ctx context.Context, fileID int64, chunks []Chunk) error
    PruneOrphanSymbols(ctx context.Context, targetUIDs []string) error   // delete symbols whose uid is not in the target set
    UpdateIndexFreshness(ctx context.Context, f FreshnessRecord) error
    // graph reads (the walker calls these; dialect-agnostic)
    EdgesByFrom(ctx context.Context, fromID int64, kinds []string) ([]EdgeRow, error)
    EdgesByTo(ctx context.Context, toID int64, kinds []string) ([]EdgeRow, error)
    EdgesByNameOnlyToName(ctx context.Context, toName string, kinds []string) ([]EdgeRow, error)  // to_id NULL, to_name = ?
    SymbolByName(ctx context.Context, name string) ([]SymbolRow, error)   // for query-time name-only resolution + code_resolve
    SymbolByUID(ctx context.Context, uid string) (SymbolRow, error)
    FileByPath(ctx context.Context, path string) (FileRow, error)   // for the door check + the watch fingerprint re-check
    // the table-adding seam (T21) — the KB's KBStore uses this to add kb_* tables without forking the dialect code
    CreateTable(ctx context.Context, dialectName string, ddl string) error
    ```
  - `types_code.go`: `FileRow{ID int64, SyncID, Path string, MtimeNS int64, Size int64, ContentHash, Language, ASTHash, IndexedAt string}`; `Symbol{ID int64, SyncID, FileID int64, Name, QualifiedName, Kind, Language, Signature string, StartLine, EndLine int, Visibility string, IsExported bool, ReturnType, ParamTypes, ContentHash, UID string}`; `Edge{ID int64, Kind string, FromID int64, ToID *int64, ToName string, FileID int64, Confidence, Line int, Context string}` (with `ToID *int64` so `nil` = name-only, CI-INV-3); `FileGraph{Symbols []Symbol, Edges []Edge, Unresolved []UnresolvedRef}`; `UnresolvedRef{FileID int64, Reference, Kind string, Line int, Candidates string}`; `FreshnessRecord{Project, ExtractorStamp, SchemaFingerprint string, LastIndexedAt *string, FilesIndexed, FilesDeleted, FilesOversized, SymbolsCount, EdgesCount, ChunksCount, UnresolvedCount int, CoverageFair float64, Capabilities, ExtractorTiers string, UpdatedAt string}`.
  - `migrations/100_code_index.sql`: the 7 tables verbatim from spec `001` §2/§3/§4/§5 + `002` §2/§4/§7 — `files` (UNIQUE path, `sync_id`, `content_hash`, `ast_hash`, `indexed_at`), `chunks` (`kind` ast|lines, `start_line`/`end_line`, `content_hash`, FK file_id ON DELETE CASCADE), `chunks_fts` (external-content FTS5 trigram over chunks.text — via the reusable `fts.go` unit), `symbols` (the single node table: `sync_id`, `file_id`, `name`, `qualified_name`, `kind`, `language`, `signature`, `start_line`/`end_line`, `visibility`, `is_exported`, `return_type`, `param_types`, `content_hash`, `uid UNIQUE`, FK file_id ON DELETE CASCADE; `idx_symbols_name`, `idx_symbols_qualified`), `edges` (`kind`, `from_id`, `to_id NULL`, `to_name`, `file_id`, `confidence`, `line`, `context`, `UNIQUE(kind, from_id, file_id, to_id, to_name, line)`; `idx_edges_from`, `idx_edges_to`, `idx_edges_to_name`), `index_freshness` (the one record: `project` PK, `extractor_stamp`, `schema_fingerprint`, `last_indexed_at`, the 5 counts, `unresolved_count`, `coverage_fair`, `capabilities` JSON, **`extractor_tiers` JSON** (D-TIER), `updated_at`), `dependencies` + `dep_edges` (the SBOM graph, `002` §7). **No temporal window** (CI-4): no `valid_from`/`valid_to`.
  - The `CreateTable` seam (T21): a dialect-neutral `CreateTable(ctx, dialectName, ddl)` that runs a DDL statement against the open store, so the KB's `KBStore` (a later change) can add `kb_pages`/`kb_pages_fts`/`kb_claims` without forking `sqlite.go`/`postgres.go`.
- Seam: the `Store` interface (INV-2: functions see only it) + the `CreateTable` seam (T21).
- Deletion test: deleting the code-index `Store` methods + `migrations/100_code_index.sql` re-creates the dialect SQL in every `codeindex` caller — the callers are `codeindex/index.go` (the write path) + `codeindex/graph/*` (the reads) + `codeindex/search/fts_leg.go` (the FTS).
- Adapters: `SQLiteStore` (this task) + `PostgresStore` (Task 8) + the in-memory test store = 3, seam justified.

**SATISFIES:** [door-check-code], [store-extends-additive], [sync-id-format-files-symbols], [single-node-table]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/core/store/store_code_test.go
// THE door check (hypothesis falsifier): a code_index writes files+symbols+edges through the
// extended Store (SQLite, WAL), and a code_search FTS floor finds a saved symbol by name.
func TestDoorCheckCodeIndexWritesAndSearches(t *testing.T) {
    st := openTestStore(t) // helper: store.Open into t.TempDir(), t.Cleanup(Close)
    fid, err := st.UpsertFile(context.Background(), store.FileRow{
        SyncID: syncid.NewSyncID("file"), Path: "internal/pay/pay.go",
        MtimeNS: 1, Size: 100, ContentHash: "abc", Language: "go", IndexedAt: now,
    })
    if err != nil { t.Fatalf("upsert file: %v", err) }
    syms := []store.Symbol{{FileID: fid, Name: "CapturePayment", QualifiedName: "internal/pay.CapturePayment",
        Kind: "method", Language: "go", StartLine: 480, EndLine: 520, ContentHash: "s1",
        UID: "pay.go:CapturePayment:method", SyncID: syncid.NewSyncID("sym")}}
    if err := st.UpsertSymbols(context.Background(), fid, syms); err != nil { t.Fatalf("upsert symbols: %v", err) }
    // a caller edge (name-only, unresolved — CI-INV-3) + a resolved edge
    edges := []store_edge{{Kind: "calls", FromID: fid, ToID: nil, ToName: "CapturePayment", FileID: fid, Confidence: "AMBIGUOUS", Line: 79}}
    if err := st.ReplaceEdgesForFile(context.Background(), fid, edges); err != nil { t.Fatalf("replace edges: %v", err) }
    // the name floor must find the symbol by name (the v1 live probe's failure: the FTS leg
    // did NOT match wireGitHooks). Task 1 creates idx_symbols_name on the single `symbols` table
    // — the identifier-first lookup the FTS leg (Task 5, `fts_leg.go`) builds on. Task 1 does NOT
    // create symbol_fts (that is Task 5's `fts_leg.go`); the door check uses the name index.
    rows, err := st.SymbolByName(context.Background(), "CapturePayment")
    if err != nil { t.Fatalf("symbol by name: %v", err) }
    if len(rows) != 1 || rows[0].UID != "pay.go:CapturePayment:method" {
        t.Fatalf("name floor broken: %+v", rows)
    }
    // sync_id minting for files + symbols (INV-6) — FileRow.SyncID / Symbol.SyncID are minted
    // by the Store on upsert (syncid.NewSyncID("file") / ("sym")); assert the stored format.
    f := st.FileByPath(context.Background(), "internal/pay/pay.go")
    if !regexp.MustCompile(`^file-[0-9a-f]{16}$`).MatchString(f.SyncID) {
        t.Fatalf("bad file sync id format: %q", f.SyncID)
    }
    if !regexp.MustCompile(`^sym-[0-9a-f]{16}$`).MatchString(rows[0].SyncID) {
        t.Fatalf("bad symbol sync id format: %q", rows[0].SyncID)
    }
}

// the extension is additive: the memory change's observation methods still work on the same store
func TestStoreExtensionAdditive(t *testing.T) {
    st := openTestStore(t)
    res, err := st.AddObservation(context.Background(), store.AddObservationParams{
        SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project",
    })
    if err != nil || res.SyncID == "" { t.Fatalf("memory AddObservation broken by the extension: %v", err) }
    // and the code-index tables coexist in the same DB
    if _, err := st.UpsertFile(context.Background(), store.FileRow{SyncID: syncid.NewSyncID("file"), Path: "a.go", ContentHash: "x", Language: "go", IndexedAt: now}); err != nil {
        t.Fatalf("code-index table on the same store: %v", err)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestDoorCheckCodeIndexWritesAndSearches|TestStoreExtensionAdditive' -v`
Expected: FAIL — the `Store` interface has no `UpsertFile`/`UpsertSymbols`/`ReplaceEdgesForFile`/`SymbolByName`/`CreateTable` (build error: "st.UpsertFile undefined"). BDD: `[door-check-code]`, `[store-extends-additive]`, `[single-node-table]` are RED in `acceptance.feature`.

- [ ] **Step 3: Write minimal implementation**

`store.go`: add the code-index methods to the `Store` interface (the exact signatures above) + the `CreateTable` seam. `migrations/100_code_index.sql`: the 7 tables verbatim from the spec (the DDL above), embedded via the existing `//go:embed migrations/*.sql` runner (the memory change's runner already sorts + applies unapplied files). `sqlite.go`: implement the 11 methods against the SQLite dialect — `UpsertFile` (`INSERT … ON CONFLICT(path) DO UPDATE`), `UpsertSymbols` (`ON CONFLICT(uid)`), `ReplaceEdgesForFile` (a tx: `DELETE FROM edges WHERE file_id=?` + bulk upsert), `SymbolByName` (`SELECT … FROM symbols WHERE name=?` — the v1 `resolve.go` shape but selecting only the columns `100_code_index.sql` creates, per T-CI-3), `CreateTable` (`db.Exec(ddl)`). Mint `sync_id` for files/symbols via `syncid.NewSyncID("file")`/`("sym")` (INV-6).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/core/store/ -run 'TestDoorCheckCodeIndexWritesAndSearches|TestStoreExtensionAdditive' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/core/store/ .skillgrid/artifacts/04-adr-0045-store-interface-extension.md
git commit -m "feat(mnemonic-v2): extend Store with code-index methods, 7-table migration, door check

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T01 door-check"
```

---

### Task 2: Migration suite + migration-parity regression (T-CI-3) + reusable FTS unit (T21) + FTS trigger parity

**Files:**
- Create: `mnemonic-v2/internal/core/store/fts.go`, `mnemonic-v2/internal/core/store/migrate_code_test.go`, `mnemonic-v2/internal/core/store/fts_test.go`
- Modify: `mnemonic-v2/internal/core/store/migrations/100_code_index.sql` (only the gap the parity test finds)
- Test: `mnemonic-v2/internal/core/store/migrate_code_test.go`, `mnemonic-v2/internal/core/store/fts_test.go`

**Interfaces:**
- Consumes: Task 1's `store.Open`, the migration runner, the code-index tables.
- Produces:
  - `fts.go`: the **reusable FTS rendering unit** (T21) — `RenderFTable(name, contentTable, contentRowID string, cols []string, tokenize string) string` (the external-content FTS5 DDL) + `FTSDDL(name string, baseCols []string) []string` (the four-trigger sync set: AFTER INSERT `WHEN new.deleted_at IS NULL`, BEFORE DELETE, BEFORE UPDATE — plain `DELETE FROM … WHERE rowid=old.id` (trigram's unsafe `'delete'` special command is why), AFTER UPDATE) + `RankingWeights(...) float64` (the BM25 composite-rank shape the memory change's `store.go` uses, extracted so the KB re-uses it verbatim).
  - The **migration-parity test pattern** (T-CI-3) that iterates a *table-set* (not a hardcoded list) — so the KB change extends the same loop over its `kb_*` tables.
- Seam: the `Store` interface (same as Task 1).
- Deletion test: n/a (hardening + extracting the FTS unit).
- Adapters: same as Task 1.

**SATISFIES:** [migration-runs-from-empty], [migration-parity-all-versions], [fts-stays-in-sync-code], [short-term-like-fallback-code], [fts-unit-reusable]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/core/store/migrate_code_test.go
// T-CI-3 (T19, the v1 bug class): a fresh store AND a store opened at each prior migration
// version can run every code_* read without a SQL-column error. The test iterates a TABLE-SET
// (not a hardcoded list) so the KB change extends the same loop over its kb_* tables.
func TestMigrationParityAllVersions(t *testing.T) {
    versions := allAppliedMigrations() // e.g. ["001_schema","100_code_index", …] the ordered set
    for _, v := range versions {
        dir := t.TempDir()
        st, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
        if err != nil { t.Fatalf("open at %s: %v", v, err) }
        // stop the migration at v (apply only files ≤ v)
        st.Close()
        st2, _ := store.OpenUpTo(context.Background(), dir, "p", store.SQLite, "", v) // test seam
        // EVERY code_* read must succeed (no "no such column") on a store at version v
        _, err = st2.SymbolByName(context.Background(), "x")
        if err != nil { t.Fatalf("SymbolByName at v=%s: %v", v, err) }
        _, err = st2.EdgesByFrom(context.Background(), 1, nil)
        if err != nil { t.Fatalf("EdgesByFrom at v=%s: %v", v, err) }
        st2.Close()
    }
}

// from-empty + additive reopen (re-runs the memory change's migrate_test.go pattern over the code-index tables)
func TestCodeMigrationRunsFromEmptyAndReopens(t *testing.T) {
    dir := t.TempDir()
    st, _ := store.Open(context.Background(), dir, "p", store.SQLite, "")
    st.Close()
    st2, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
    if err != nil { t.Fatalf("reopen: %v", err) } // migrations idempotent (additive-only)
    st2.Close()
}

// mnemonic-v2/internal/core/store/fts_test.go
// T21: the reusable FTS unit produces the external-content DDL + the four triggers; the KB re-uses it.
func TestFtsUnitReusable(t *testing.T) {
    ddl := store.FTSDDL("kb_pages_fts", []string{"title", "body", "type", "rel_path"})
    if len(ddl) != 5 { t.Fatalf("expected 1 FTS DDL + 4 triggers, got %d", len(ddl)) } // 1 + 4
    for _, s := range ddl {
        if !strings.Contains(s, "kb_pages_fts") { t.Fatalf("DDL must name the table: %s", s) }
    }
    // the same unit the code-index uses for chunks_fts
    if len(store.FTSDDL("chunks_fts", []string{"text"})) != 5 { t.Fatalf("code-index FTS unit") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestMigrationParityAllVersions|TestCodeMigrationRunsFromEmptyAndReopens|TestFtsUnitReusable' -v`
Expected: FAIL — `store.FTSDDL` undefined (the FTS unit is not yet extracted from `sqlite.go`); `OpenUpTo` (the test seam to apply migrations up to a version) not yet present.

- [ ] **Step 3: Write minimal implementation**

`fts.go`: extract the external-content FTS5 DDL + the four-trigger sync set + the ranking-weights shape from `sqlite.go` into `fts.go` (the unit the memory change's `sqlite.go` + this change's `chunks_fts` + the KB's `kb_pages_fts` all call). Add the `OpenUpTo` test seam (apply migrations only up to a named file) to the migration runner. If `TestMigrationParityAllVersions` finds a gap (a read referencing a column a prior version lacks — the v1 bug class), fix it in `100_code_index.sql` (add the missing `ALTER TABLE … ADD COLUMN …` bridge the v1 set was missing).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/core/store/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/core/store/
git commit -m "feat(mnemonic-v2): migration-parity regression (T-CI-3), reusable FTS unit (T21)

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T02"
```

---

### Task 3: The extraction ladder (rich-6 + 25 maps + regex) + self-healing pool + real extractor stamp (D-STAMP) + chunker

> ⚠ one-way: **D4 — the extractor stamp algorithm** (D-STAMP). Changing it later changes the stored value once (a one-time full re-walk), never a schema migration.

**Files:**
- Create: `mnemonic-v2/internal/codeindex/extract/ladder.go`, `languages.go`, `fallback.go`, `pool.go`, `stamp.go`, `chunker.go`, `knowledge.go`
- Test: `mnemonic-v2/internal/codeindex/extract/*_test.go`

**Interfaces:**
- Consumes: Task 1's `Store` (the `UpsertFile`/`UpsertSymbols`/`ReplaceEdgesForFile`/`ReplaceChunksForFile` write path), `syncid`.
- Produces:
  - `extract.FileGraph` (the per-file output — `[]Symbol`, `[]Edge`, `[]UnresolvedRef`, the chunk text) + `extract.ExtractFile(ctx, path, content, lang) (FileGraph, Tier, error)` where `Tier` ∈ `rich`|`map`|`regex` (the per-language tier for D-TIER).
  - `extract.Stamp() string` (D-STAMP) — the content hash of the extractor code/config (the rich-6 extractors + the 25 node-type maps + the regex battery + the chunker params + the include/exclude defaults). A `//go:generate`-produced const, or an FNV-1a over a serialized extractor-fingerprint string (the v1 `fingerprintDigest` shape `fingerprint.go:71-84` but with a real caller). **NOT a const** — a const never changes, so the full re-walk signal never fires.
  - `extract.Chunk(graph FileGraph, content string) []Chunk` (the AST-first chunker: ≤1000 chars, 250 merge, gaps→lines, 80/10 window, lines fallback).
- Seam: gotreesitter (the rich-6 + 25-map ladder) + the regex fallback (the deterministic floor, INV-3).
- Deletion test: deleting `internal/codeindex/extract` re-creates the entire tree-sitter + regex + chunking logic in `codeindex/index.go` — the only caller is `index.go`.
- Adapters: the gotreesitter rich-6 extractors + the 25-map `walkTree` + the regex fallback = 3 tiers, seam justified.

**SATISFIES:** [extract-ladder-3-tiers], [self-healing-pool-circuit-breaker], [extractor-stamp-content-hash], [chunker-ast-first], [knowledge-node-classification], [max-file-size-oversized]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/codeindex/extract/ladder_test.go
func TestExtractLadderRich6(t *testing.T) {
    g, tier, err := extract.ExtractFile(ctx, "pay.go", goSourceWithMethodAndCall, "go")
    if err != nil { t.Fatalf("extract: %v", err) }
    if tier != extract.TierRich { t.Fatalf("go must be rich tier, got %s", tier) }
    if len(g.Symbols) < 1 || g.Symbols[0].Kind != "method" { t.Fatalf("symbols: %+v", g.Symbols) }
    // a statically-bound call → EXTRACTED; a name-only call → to_id NULL + unresolved (CI-INV-3)
    var hasExtractedCall, hasNameOnly bool
    for _, e := range g.Edges {
        if e.Kind == "calls" && e.Confidence == "EXTRACTED" { hasExtractedCall = true }
        if e.Kind == "calls" && e.ToID == nil && e.ToName != "" { hasNameOnly = true }
    }
    if !hasExtractedCall || !hasNameOnly { t.Fatalf("edges: %+v", g.Edges) }
}

func TestExtractLadderMap25(t *testing.T) {
    g, tier, _ := extract.ExtractFile(ctx, "a.py", pythonSource, "python")
    if tier != extract.TierMap && tier != extract.TierRich { t.Fatalf("python tier: %s", tier) }
    if len(g.Symbols) == 0 { t.Fatalf("map tier must emit symbols: %+v", g.Symbols) }
}

func TestExtractLadderRegexFallback(t *testing.T) {
    g, tier, _ := extract.ExtractFile(ctx, "x.unknownext", "func foo() {\n  bar()\n}", "unmapped")
    if tier != extract.TierRegex { t.Fatalf("unmapped → regex tier, got %s", tier) }
    if len(g.Symbols) == 0 { t.Fatalf("regex floor must emit symbols: %+v", g.Symbols) }
}

// mnemonic-v2/internal/codeindex/extract/pool_test.go
func TestSelfHealingPoolCircuitBreaker(t *testing.T) {
    p := extract.NewPool(extract.PoolConfig{MaxRespawns: 3, TripAfter: 3})
    for i := 0; i < 3; i++ { p.RecordDeath("go", "bad.go") } // 3 deaths
    if !p.Tripped("go") { t.Fatalf("circuit breaker must trip after 3 deaths") }
    // a 4th go file goes straight to regex (no tree-sitter attempt)
    if tier := p.TierFor("go", "another.go"); tier != extract.TierRegex { t.Fatalf("tripped → regex, got %s", tier) }
}

// mnemonic-v2/internal/codeindex/extract/stamp_test.go
func TestExtractorStampIsContentHash(t *testing.T) {
    a := extract.Stamp()
    if a == "" || len(a) < 8 { t.Fatalf("stamp too short: %q", a) }
    // a DIFFERENT extractor (a different fingerprint string) must produce a DIFFERENT stamp
    if extract.StampFor("extractor-config-v1") == extract.StampFor("extractor-config-v2") {
        t.Fatalf("stamp must be a content hash, not a const")
    }
}

// mnemonic-v2/internal/codeindex/extract/chunker_test.go
func TestChunkerAstFirst(t *testing.T) {
    chunks := extract.Chunk(graphWithOneLongMethod, content)
    // one chunk per function-like symbol (≤1000 chars); gaps → lines; over-long → 80/10 window
    if len(chunks) == 0 { t.Fatalf("no chunks") }
    for _, c := range chunks {
        if c.Kind != "ast" && c.Kind != "lines" { t.Fatalf("bad chunk kind: %s", c.Kind) }
    }
    // a config file (no symbols) → plain 80-line chunks
    lineChunks := extract.Chunk(emptyGraph, configContent)
    if len(lineChunks) == 0 || lineChunks[0].Kind != "lines" { t.Fatalf("lines fallback broken") }
}

// mnemonic-v2/internal/codeindex/extract/knowledge_test.go
func TestKnowledgeNodeClassification(t *testing.T) {
    if k, _ := extract.Classify("README.md", markdownContent); k != "doc" { t.Fatalf("md → doc, got %s", k) }
    if k, _ := extract.Classify("config.yaml", yamlContent); k != "config" { t.Fatalf("yaml → config, got %s", k) }
    if k, _ := extract.Classify("schema.sql", ddl); k != "table" { t.Fatalf("sql → table, got %s", k) }
    if _, ok := extract.Classify("x.xyz", "…"); ok { t.Fatalf("unmapped .xyz must be skipped, not classified") }
}

func TestMaxFileSizeOversized(t *testing.T) {
    big := make([]byte, 512*1024+1) // over the 500 KiB default
    g, _, _ := extract.ExtractFile(ctx, "big.go", string(big), "go", extract.WithMaxFileSize(500*1024))
    if g.Indexed { t.Fatalf("oversized file must NOT be indexed (counted in files_oversized)") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/codeindex/extract/ -v`
Expected: FAIL — package `extract` does not exist (build error). BDD: the `[extract-ladder-3-tiers]`, `[extractor-stamp-content-hash]`, etc. scenarios are RED.

- [ ] **Step 3: Write minimal implementation**

`languages.go`: the 25 per-language node-type → symbol-kind **DATA table** (the v1 `langMaps` shape `languages.go:13-18,133-159` — `langMapDef{defNodes map[string]defInfo, callNodes map[string]callInfo, imports func, heritage func}`), NOT 1481 lines of code. `ladder.go`: the 3-tier dispatch (the v1 `ExtractFile` `adapter.go:23`) — `coveredLanguages` (go, python, java, js, ts, tsx) → the gotreesitter one-pass (`ExtractDefinitionSpans`/`ExtractCalls`/`ExtractHeritage`/`ExtractImports`); the 25-map languages → `walkTree` against the data table; unmapped/panicking → regex. A statically-bound call → `EXTRACTED`; a name-only call → `to_id NULL` + an `unresolved_edges` row (CI-INV-3). `fallback.go`: the regex battery (the v1 `fallback.go:50-162` — function/class decls + import regexes + next-function call chains; drop v1's dead `fallbackRegexes`). `pool.go`: the self-healing pool (the v1 `extract.go:152-172` — `MaxRespawns=3`, `TripAfter=3`; a panicking grammar quarantines the file to regex + trips the per-language breaker). `stamp.go`: the content hash (D-STAMP) — `StampFor(fingerprint string) string` = FNV-1a hex; `Stamp()` calls `StampFor` with the serialized extractor config. `chunker.go`: the AST-first chunker (the v1 `indexer.go:327-439` — ≤1000 chars, 250 merge, gaps→lines, 80/10 window, lines fallback). `knowledge.go`: the knowledge-node extractors (route/doc/config/sql → `symbols` rows with a kind + their `edges`, the v1 `route/extract.go` + `knowledge/*` as data-driven patterns).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/codeindex/extract/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/codeindex/extract/ .skillgrid/artifacts/04-adr-0042-tree-sitter-dep.md
git commit -m "feat(mnemonic-v2): tree-sitter extraction ladder, self-healing pool, real extractor stamp, chunker

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T03"
```

---

### Task 4: The graph walker (impact + neighbors + path/graph-stops + `code_affected` + `code_resolve` + `code_packages` + the `extractor_tiers` column)

**Files:**
- Create: `mnemonic-v2/internal/codeindex/graph/walker.go`, `neighbors.go`, `resolve.go`, `path.go`, `packages.go`, `orient.go`
- Test: `mnemonic-v2/internal/codeindex/graph/*_test.go`

**Interfaces:**
- Consumes: Task 1's `Store` graph-read methods (`EdgesByFrom`/`EdgesByTo`/`EdgesByNameOnlyToName`/`SymbolByName`/`SymbolByUID`), Task 3's `FileGraph` (what the walker reads).
- Produces:
  - `graph.Impact(ctx, symID int64, minConfidence string, maxDepth int) (ImpactResult, error)` — reverse-BFS, depth-cap 5 default, confidence filter (EXTRACTED > LSP_RESOLVED > INFERRED > AMBIGUOUS), WILL BREAK (depth 1) / LIKELY AFFECTED (deeper), `Excluded` count (counted, not dropped), traverses only stored edges.
  - `graph.Callees/Callers/Dependents/Implementors/Hierarchy/TestsFor(ctx, symID) (Neighbors, error)` — confidence-labeled edges; name-only edges resolve **at query time** against `symbols.name` (0→kept name-only, 1→resolved, >1→one AMBIGUOUS each).
  - `graph.Resolve(ctx, name string, calledFrom *string) (ResolveResult, error)` — the `code_resolve` contract (T4): `{exists, name, qualified_name, file, line, sync_id, called_from[] (file paths), ambiguous, candidates[]}`.
  - `graph.Path(ctx, fromID, toID int64) (PathResult, error)` — BFS shortest path OR "where the graph stops" (the causal dispatch-kind fix, T9).
  - `graph.Affected(ctx, opts AffectedOpts) (AffectedResult, error)` — the 3 input modes (changed/stdin/base-ref), the Java test-convention fix (T8).
  - `graph.Packages(ctx) ([]PackageInfo, error)` — the top-level Go package map (T5).
  - `graph.Orient(ctx, unit OrientUnit, depth int) (OrientResult, error)` (T9b) — the **navigation entry point** (the `ki outline` map). `OrientUnit{Kind symbol|package|file, ID int64, Path string}`; `OrientResult{Meta, Signature string, TOC []SymbolRow, Outbound []EdgeRow, Depth int}`. `depth` (default 1) bounds outbound-edge expansion; an edge target is **not** expanded inline (follow = another `Orient` on the target id). Package unit → that package's symbol TOC (the §4 top-level Go package); file unit → full TOC + import edges. **Replaces** v1's `code_orient`/`code_signature`/`code_file_toc` three tools.
- Seam: the `Store` interface (the walker is dialect-agnostic — the same code runs over SQLite + Postgres).
- Deletion test: deleting `internal/codeindex/graph` re-creates the reverse-BFS + neighbor + resolve + path + packages logic in every caller — the callers are `transport/handler_code.go` (the `code_*` tools) + `search/hybrid.go` (the structural leg's structural context).
- Adapters: the SQLite `Store` + the Postgres `Store` + the test store = 3, seam justified (the walker is dialect-agnostic by construction).

**SATISFIES:** [impact-blast-radius], [neighbors-confidence-labeled], [code-resolve-contract], [code-packages-contract], [path-graph-stops], [affected-tests-java], [name-only-query-time-relabel], [orient-map-navigable]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/codeindex/graph/walker_test.go
func TestImpactBlastRadius(t *testing.T) {
    st := seedGraph(t) // a small graph: A ← B ← C (C calls B calls A), plus a name-only edge from D
    res, err := graph.Impact(ctx, symID("A"), "INFERRED", 5)
    if err != nil { t.Fatalf("impact: %v", err) }
    // B is WILL BREAK (depth 1); C is LIKELY AFFECTED (depth 2)
    if res.TierFor("B") != "WILL BREAK" || res.TierFor("C") != "LIKELY AFFECTED" {
        t.Fatalf("tiers: %+v", res)
    }
    // the name-only edge from D that resolved to nothing is NOT in the radius (traverses stored edges only, CI-INV-3)
    if res.Has("D") { t.Fatalf("unresolved D must not inflate the radius") }
    // a min_confidence=EXTRACTED run excludes the INFERRED hops (counted, not dropped)
    res2, _ := graph.Impact(ctx, symID("A"), "EXTRACTED", 5)
    if res2.Excluded == 0 { t.Fatalf("excluded hops must be counted") }
}

// mnemonic-v2/internal/codeindex/graph/resolve_test.go
func TestCodeResolveContract(t *testing.T) {
    st := seedResolve(t) // CapturePayment exists, called from enterprise.go
    r, _ := graph.Resolve(ctx, "CapturePayment", nil)
    if !r.Exists || r.File != "internal/pay/pay.go" || r.Line != 480 { t.Fatalf("resolve: %+v", r) }
    if len(r.CalledFrom) != 1 || r.CalledFrom[0] != "internal/billing/enterprise.go" {
        t.Fatalf("called_from (file paths, CI-O3): %+v", r.CalledFrom)
    }
    // ambiguous: two symbols named Foo → candidates[], never a silent pick (CI-INV-3)
    ra, _ := graph.Resolve(ctx, "Foo", nil)
    if !ra.Ambiguous || len(ra.Candidates) != 2 { t.Fatalf("ambiguous: %+v", ra) }
    // not found → exists:false (the KB fails closed → unresolved)
    rn, _ := graph.Resolve(ctx, "Gone", nil)
    if rn.Exists { t.Fatalf("gone symbol must be exists:false") }
}

// mnemonic-v2/internal/codeindex/graph/packages_test.go
func TestCodePackagesContract(t *testing.T) {
    st := seedPackages(t) // internal/memory (12 files, 87 symbols), internal/codeindex (9 files, 64 symbols)
    pkgs, _ := graph.Packages(ctx)
    byPath := map[string]graph.PackageInfo{}
    for _, p := range pkgs { byPath[p.Path] = p }
    if byPath["internal/memory"].Files != 12 || byPath["internal/memory"].Symbols != 87 {
        t.Fatalf("package map: %+v", pkgs)
    }
    // sub-packages are NOT the unit (top-level only, 004 §4)
    if _, ok := byPath["internal/memory/sub"]; ok { t.Fatalf("sub-package must not be a unit") }
}

// mnemonic-v2/internal/codeindex/graph/orient_test.go
// T9b (the ki outline map fold): orient is the re-rootable, depth-bounded navigation entry point.
func TestOrientMapReRootsAndBounded(t *testing.T) {
    st := seedOrient(t) // package internal/pay with symbols; CapturePayment calls Log, which is in another file
    // orienting on a PACKAGE returns the symbol TOC + outbound edges with stable ids
    pkg, _ := graph.Orient(ctx, graph.OrientUnit{Kind: graph.UnitPackage, Path: "internal/pay"}, 1)
    if len(pkg.TOC) == 0 { t.Fatalf("package TOC must list symbols: %+v", pkg.TOC) }
    if len(pkg.Outbound) == 0 { t.Fatalf("package map must carry outbound edges: %+v", pkg.Outbound) }
    // depth=1 does NOT expand the edge targets (the map is bounded, ki outline --depth)
    for _, e := range pkg.Outbound {
        if e.TargetExpanded { t.Fatalf("depth=1 must not expand edge targets: %+v", e) }
    }
    // re-rooting on an edge target's id returns THAT symbol's neighborhood (follow the map)
    targetID := pkg.Outbound[0].ToID
    sym, _ := graph.Orient(ctx, graph.OrientUnit{Kind: graph.UnitSymbol, ID: targetID}, 1)
    if sym.Meta == "" || len(sym.TOC) == 0 { t.Fatalf("re-root on symbol id must return its map: %+v", sym) }
}

// mnemonic-v2/internal/codeindex/graph/path_test.go
func TestPathGraphStops(t *testing.T) {
    st := seedStops(t) // A → B, then a name-only edge to an interface (the flow stops at the dispatch boundary)
    p, _ := graph.Path(ctx, symID("A"), symID("Unreachable"))
    if p.Path == nil && p.Stops == nil { t.Fatalf("PathResult must be something") }
    if p.Stops != nil && p.Stops.DispatchKind != "interface" { t.Fatalf("causal dispatch kind: %+v", p.Stops) }
    // the v1 quirk fixed: DispatchKind is the interface at the FRONTIER, not last-iterated-over-visited
}

// mnemonic-v2/internal/codeindex/graph/affected_test.go
func TestAffectedTestsJava(t *testing.T) {
    // the v1 gap fixed: Java test files are recognized (NOT-PORT)
    st := seedJava(t) // Foo.java changed, FooTest.java is the test
    res, _ := graph.Affected(ctx, graph.AffectedOpts{Changed: []string{"src/Foo.java"}})
    if len(res.AffectedTests) != 1 || res.AffectedTests[0] != "src/FooTest.java" {
        t.Fatalf("java test convention: %+v", res.AffectedTests)
    }
}

func TestNameOnlyQueryTimeRelabel(t *testing.T) {
    st := seedNameOnly(t) // a name-only edge to_name="Foo"; two symbols named Foo exist
    n, _ := graph.Callers(ctx, symID("Foo-1"))
    // >1 candidates → one AMBIGUOUS edge each (v1 fetchEdges graph.go:217-226)
    ambiguous := 0
    for _, e := range n.Edges { if e.Confidence == "AMBIGUOUS" { ambiguous++ } }
    if ambiguous == 0 { t.Fatalf("ambiguous name-only must relabel to AMBIGUOUS: %+v", n.Edges) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/codeindex/graph/ -v`
Expected: FAIL — package `graph` does not exist (build error). BDD: the `[impact-blast-radius]`, `[code-resolve-contract]`, etc. scenarios are RED.

- [ ] **Step 3: Write minimal implementation**

`walker.go`: the dialect-agnostic reverse-BFS (the v1 `impact.go:81-180` — depth-cap 5, `confidenceRank` EXTRACTED=3/LSP=3/INFERRED=2/AMBIGUOUS=1, an edge passes iff `rank(edge) >= rank(min)`, excluded hops counted not dropped, traverses only stored edges — the name-only query `WHERE (e.to_id = ? OR (e.to_id IS NULL AND e.to_name IN (SELECT name FROM symbols WHERE id = ?)))`). **Fix the v1 inconsistency** (NOT-PORT): the walker applies the multi-candidate AMBIGUOUS relabel consistently (the v1 `impact.go` did not, `fetchEdges` did). `neighbors.go`: the 6 neighbor views (the v1 `graph.go:288-309` + `viewMatches`) + the query-time name-only relabel (`fetchEdges` `graph.go:145-238`: 0→`To.ID==0`, 1→resolved, >1→AMBIGUOUS each). `resolve.go`: `code_resolve` (lifts v1 `resolve.go:35-76` — 0/1/>1) + `called_from` = the forward callers resolved to `files.path` (CI-O3). `path.go`: `code_path` (BFS) + `buildGraphStops` (the v1 `graph.go:442-481` but with the **causal** dispatch-kind fix — the interface/bus/callback at the frontier, not last-iterated). `packages.go`: the top-level Go package map (group `files` where `language='go'` by top-level package path + count `symbols`). `orient.go`: the **navigation entry point** (T9b, the `ki outline` map) — `graph.Orient` over a symbol/package/file unit: the file TOC (every symbol in the containing file, line order, from `symbols` where `file_id = ?` ORDER BY `start_line`) + the unit's outbound edges (`calls`/`imports`/`references`, from `EdgesByFrom`) with stable target ids; `depth` (default 1) bounds how many hops of outbound edges are *listed* (the targets are **not** expanded inline — `TargetExpanded=false` at depth 1, a follow is a new `Orient` call on the target id). Package unit → the package's symbol TOC (the §4 top-level Go package); file unit → the file TOC + its import edges. **Replaces** v1's `code_orient`/`code_signature`/`code_file_toc` three tools (combined into one map read, spec `004` §8).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/codeindex/graph/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/codeindex/graph/
git commit -m "feat(mnemonic-v2): graph walker — impact, neighbors, path, code_resolve, code_packages, affected

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T04"
```

---

### Task 5: The hybrid search (FTS leg + structural leg + semantic leg + RRF K=60 + `code_read` + the T-CI-1/T-CI-2 relevance fixes)

**Files:**
- Create: `mnemonic-v2/internal/codeindex/search/hybrid.go`, `fts_leg.go`, `structural.go`, `semantic.go`, `read.go`
- Test: `mnemonic-v2/internal/codeindex/search/*_test.go`

**Interfaces:**
- Consumes: Task 1's `Store` (the FTS leg reads `chunks_fts`/`symbol_fts` via `Store.Search`-shape methods), Task 3's `extract` (the structural leg re-parses the tree with gotreesitter), Task 4's `graph` (the `called_from` for semantic context), the memory change's `intelligence/embedder` seam + `degrade` (the semantic leg).
- Produces:
  - `search.Hybrid(ctx, q SearchQuery) (SearchResult, error)` — RRF K=60 over the available legs, per-hit provenance, the broken-leg-degrades-out (T-CI-2), the FTS floor always answers.
  - `search.Grep(ctx, pattern, path string, lang string) (GrepResult, error)` — the index-free by-example AST grep + the **fixed pattern language** (T12).
  - `search.Read(ctx, path string, start, end int, mode ReadMode) (ReadResult, error)` — the live source span + secret redaction (T13); `mode` ∈ `full`(default) | `content`.
  - `search.ReadSymbol(ctx, symID int64, mode ReadMode) (ReadResult, error)` (T13 content-mode fold) — the **drill handle**: `content` → the signature + callee/caller **ids** (bodies empty); `full` → the symbol's `start_line`…`end_line` span.
  - `SearchQuery{Query, Project string, Limit int, Language string, Under string}` where **`Under`** is the optional `files.path` package prefix that narrows all three legs to a subtree (T10 fold, the `ki search --under`; empty-scope → empty, not an error).
  - `ReadMode` (a string: `full`/`content`); `ReadResult{Text string, TOC []SymbolRow, NeighborIDs []NeighborID, Signature string, Degraded string}` where `NeighborID{ToID int64, ToName, Kind, Confidence string, Body string}` (`Body` is empty in `content` mode); `SearchResult{Query, Under string, Hits []Hit, Degraded string, Warnings []string}` where `Hit{Path string, StartLine, EndLine int, Symbol, Kind string, Score float64, Provenance map[string]int}` (per-signal rank).
- Seam: the `Embedder` seam (the semantic leg degrades through it, INV-3) + gotreesitter (the structural leg).
- Deletion test: deleting `internal/codeindex/search` re-creates the RRF fusion + the 3 legs + the grep + the read in `transport/handler_code.go` — the only caller is the handler.
- Adapters: the FTS leg + the structural leg + the semantic leg = 3 legs fused by RRF; the embedder seam (prod + test + null) = 3, seam justified.

**SATISFIES:** [search-hybrid-rrf], [search-real-vs-fake-divergence], [broken-leg-degrades-out], [structural-grep-pattern-language], [code-read-secret-redaction], [degraded-no-vector], [search-under-scope], [code-read-content-mode]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/codeindex/search/hybrid_test.go
// T-CI-1 (T20, the v1 live-probe failure): a real symbol and a fake symbol must NOT return the same hits.
func TestSearchRealVsFakeDivergence(t *testing.T) {
    st := seedRealAndFake(t) // a real symbol wireGitHooks in install/sync.go:79 + a fake "capturePayment"
    real, _ := search.Hybrid(ctx, search.SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5})
    fake, _ := search.Hybrid(ctx, search.SearchQuery{Query: "capturePayment", Project: "p", Limit: 5})
    // the real symbol MUST appear in the FTS leg (the v1 FTS leg did NOT match wireGitHooks)
    if !hitHasPath(real.Hits, "install/sync.go") { t.Fatalf("real symbol not in FTS leg: %+v", real.Hits) }
    // the fake symbol must NOT return the same top hits as the real one
    if topHit(real.Hits) == topHit(fake.Hits) { t.Fatalf("real == fake hits (the v1 bug): %+v vs %+v", real.Hits, fake.Hits) }
}

// T-CI-2 (the v1 live-probe failure: a broken leg's warning shaped the output): a leg that errors
// degrades to degraded:<leg> + drops out of the fusion; the other legs still answer.
func TestBrokenLegDegradesOutOfFusion(t *testing.T) {
    st := seedBrokenLeg(t) // force the semantic leg's SQL to fail (e.g. a missing column)
    r, _ := search.Hybrid(ctx, search.SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5})
    if r.Degraded != "no_vector" && !strings.HasPrefix(r.Degraded, "semantic") {
        t.Fatalf("broken leg must degrade: %q", r.Degraded)
    }
    // the FTS leg still answers (the broken leg's empty result did NOT set the answer)
    if len(r.Hits) == 0 { t.Fatalf("FTS floor must still answer: %+v", r.Hits) }
}

// mnemonic-v2/internal/codeindex/search/structural_test.go
func TestStructuralGrepPatternLanguage(t *testing.T) {
    // the node-type+capture form (the v1 working syntax)
    r1, _ := search.Grep(ctx, "(function_declaration) \\fn", "install/sync.go", "go")
    if len(r1.Hits) == 0 { t.Fatalf("node-type+capture form broken: %+v", r1.Hits) }
    // the shorthand the v1 CLI help implied but the parser rejected (the v1 UX gap fixed, T12)
    r2, _ := search.Grep(ctx, "\\f(\\args)", "install/sync.go", "go")
    if len(r2.Hits) == 0 { t.Fatalf("\\f(ARGS) shorthand must be accepted: %+v", r2.Hits) }
    // an invalid pattern for a language → degraded:no_grammar:<lang> (never a silent skip)
    r3, _ := search.Grep(ctx, "(rust_only_node) \\x", "install/sync.go", "go")
    if r3.Degraded != "no_grammar:go" { t.Fatalf("invalid pattern must degrade: %q", r3.Degraded) }
}

// mnemonic-v2/internal/codeindex/search/read_test.go
func TestCodeReadSecretRedaction(t *testing.T) {
    st := seedSecret(t) // a file with token=abc123 + password=xyz
    r, _ := search.Read(ctx, "config.go", 1, 20)
    if strings.Contains(r.Text, "abc123") || strings.Contains(r.Text, "xyz") {
        t.Fatalf("secret not redacted: %q", r.Text)
    }
    if !strings.Contains(r.Text, "token=") { t.Fatalf("the key must remain, only the value masked: %q", r.Text) }
}

func TestDegradedNoVector(t *testing.T) {
    st := seedNoEmbedder(t) // the embedder seam is nil
    r, _ := search.Hybrid(ctx, search.SearchQuery{Query: "x", Project: "p", Limit: 5})
    if r.Degraded != "no_vector" { t.Fatalf("nil embedder → degraded:no_vector, got %q", r.Degraded) }
    if len(r.Hits) == 0 { t.Fatalf("FTS floor must still answer with no embedder") }
}

// T10 fold (the ki search --under): a files.path package prefix narrows all three legs to a subtree.
func TestSearchUnderScope(t *testing.T) {
    st := seedScoped(t) // "retry" matches a chunk in internal/pay AND a chunk in internal/billing
    scoped, _ := search.Hybrid(ctx, search.SearchQuery{Query: "retry", Project: "p", Limit: 10, Under: "internal/pay"})
    for _, h := range scoped.Hits {
        if !strings.HasPrefix(h.Path, "internal/pay/") { t.Fatalf("under-scope leaked a hit outside the prefix: %+v", h) }
    }
    // an under-scope that matches nothing → empty (not an error)
    empty, err := search.Hybrid(ctx, search.SearchQuery{Query: "retry", Project: "p", Limit: 10, Under: "internal/does-not-exist"})
    if err != nil { t.Fatalf("empty under-scope must not error: %v", err) }
    if len(empty.Hits) != 0 { t.Fatalf("empty under-scope must return no hits: %+v", empty.Hits) }
}

// T13 fold (the ki get --type content): a symbol read in content mode returns the signature +
// callee/caller ids, NOT their bodies; a file read in content mode returns the TOC.
func TestCodeReadContentMode(t *testing.T) {
    st := seedContentMode(t) // CapturePayment calls Log (in another file)
    r, _ := search.ReadSymbol(ctx, symID("CapturePayment"), search.ReadModeContent)
    if r.Signature == "" { t.Fatalf("content mode must return the signature: %+v", r) }
    if len(r.NeighborIDs) == 0 { t.Fatalf("content mode must return callee/caller ids: %+v", r) }
    for _, n := range r.NeighborIDs {
        if n.Body != "" { t.Fatalf("content mode must NOT return neighbor bodies: %+v", n) }
    }
    // a file target in content mode → the TOC, not the body
    f, _ := search.Read(ctx, "internal/pay/pay.go", 0, 0, search.ReadModeContent)
    if len(f.TOC) == 0 || f.Body != "" { t.Fatalf("file content mode must return the TOC, not the body: %+v", f) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/codeindex/search/ -v`
Expected: FAIL — package `search` does not exist (build error). BDD: the `[search-real-vs-fake-divergence]`, `[broken-leg-degrades-out]`, etc. scenarios are RED.

- [ ] **Step 3: Write minimal implementation**

`fts_leg.go`: the FTS floor — BM25 over `chunks_fts` + `symbol_fts` (the v1 `fts.go:33-41` + `symbol_fts.go`); identifier-shaped queries match `symbol_fts.name` first (the T-CI-1 fix — the v1 FTS leg was not matching `wireGitHooks`); **the `under?` prefix** (T10 fold, spec `006` §5.1) filters the leg to `WHERE files.path LIKE under || '/%'` (a `files.path` prefix over all three legs — no new table; an empty-scope prefix returns empty, not an error). `structural.go`: the by-example AST grep (the v1 `structural.go:299` `GrepByExample` — index-free, re-parses the tree) + the **fixed pattern language** (T12): accept BOTH the node-type+capture form (`(function_declaration) \fn`) AND the `\f(ARGS)` shorthand (the v1 parser only accepted the former — a NOT-PORT UX fix); an invalid pattern for a language → `degraded: no_grammar:<lang>`. `semantic.go`: the semantic leg — calls the `vectorstore` (Task 5's dependency, but the leg itself is wired here); a nil/failing embedder → the `degrade` seam → `degraded: no_vector`. `hybrid.go`: the RRF fusion (the v1 `rank.go:124-271`, K=60, `score = Σ 1/(K + rank + 1)`) + per-hit provenance (`Hit.Provenance` per leg) + **the broken-leg-degrades-out** (T-CI-2): a leg that returns an error degrades to `degraded: <leg>` + is dropped from the fusion (the v1 emitted a warning but the broken leg still shaped the output — this is the fix). `read.go`: `code_read` (the live source span + secret redaction — mask `token=`/`password=`/`secret=` values, the v1 property) **+ the two modes** (T13 fold, spec `006` §7): `mode: full` (default) = the live span (a symbol id expands to `start_line`…`end_line`); `mode: content` = the **drill handle** — `ReadSymbol(id, content)` returns the signature + the callee/caller **ids** (the confidence-labeled `to_id`/`to_name` from `EdgesByFrom`/`EdgesByTo`, bodies empty) and `Read(path, 0,0, content)` on a file returns the **TOC** (the symbol list, line order) not the body. The `under` prefix is also recorded in the retrieval trail (INV-5) alongside the query + legs.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/codeindex/search/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/codeindex/search/
git commit -m "feat(mnemonic-v2): hybrid search (RRF K=60) + structural grep + code_read + T-CI-1/T-CI-2 fixes

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T05"
```

---

### Task 6: The watch loop (fsnotify + debounce + burst-collapse + fingerprint re-check + failure isolation)

**Files:**
- Create: `mnemonic-v2/internal/codeindex/watch/watch.go`
- Test: `mnemonic-v2/internal/codeindex/watch/watch_test.go`

**Interfaces:**
- Consumes: Task 3's `extract.Stamp()` (the fingerprint re-check compares against the stored stamp), Task 1's `Store` (the `files` table for the stat-walk), the memory change's `config` (the debounce window).
- Produces:
  - `watch.NewWatcher(root string, cfg WatchConfig, onSync func(paths []string)) *Watcher` + `(*Watcher).Start(ctx)` / `(*Watcher).Stop()` — the fsnotify loop (the v1 `watch.go:113-125`).
  - `WatchConfig{Debounced time.Duration, Include, Exclude []string}` + `DefaultDebounce = 2s`, `MinDebounce = 100ms`, `MaxDebounce = 60s` (the v1 `watch.go:20-42`).
- Seam: fsnotify (the event source) — the `onSync` callback is the seam the `code_index` re-index hangs off.
- Deletion test: deleting `internal/codeindex/watch` re-creates the fsnotify + debounce + burst-collapse + re-check logic in `cmd/mnemonic/index.go` (the `mnemonic watch` command) + `transport/mcp/tools_code.go` (the `code_watch` tool) — the only callers.
- Adapters: the real fsnotify watcher + a fake event source (test) = 2, seam justified.

**SATISFIES:** [watch-debounce-burst-collapse], [watch-fingerprint-recheck], [watch-failure-isolation], [watch-trigger-not-gate]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/codeindex/watch/watch_test.go
func TestWatchBurstCollapse(t *testing.T) {
    w := watch.NewWatcher(t.TempDir(), watch.WatchConfig{Debounced: 50 * time.Millisecond}, func(paths []string) {
        t.Log("onSync called with", paths)
    })
    // a burst of 5 edits within the debounce window → ONE onSync call
    calls := 0
    w2 := watch.NewWatcher(t.TempDir(), watch.WatchConfig{Debounced: 50 * time.Millisecond}, func(p []string) { calls++ })
    w2.EmitForTest(5) // 5 events in the burst
    time.Sleep(120 * time.Millisecond)
    if calls != 1 { t.Fatalf("burst must collapse to 1 onSync, got %d", calls) }
}

func TestWatchFingerprintRecheck(t *testing.T) {
    w := watch.NewWatcher(t.TempDir(), watch.WatchConfig{Debounced: 10 * time.Millisecond}, nil)
    // a git checkout that touched mtime only (no content change) → no re-index (a no-op)
    w.MarkPendingForTest("a.go") // mtime changed, size+content unchanged
    w.DrainForTest()
    if w.ReindexCalled() { t.Fatalf("mtime-only change must be a no-op (file-hash re-check)") }
}

func TestWatchFailureIsolation(t *testing.T) {
    var failed bool
    w := watch.NewWatcher(t.TempDir(), watch.WatchConfig{Debounced: 10 * time.Millisecond}, func(p []string) { failed = true })
    w.OnSyncErrForTest(errors.New("store lock")) // a re-index failure
    w.DrainForTest()
    if !w.Running() { t.Fatalf("a re-index failure must not stop the watcher") }
    // the next event still re-checks
    w.MarkPendingForTest("b.go"); w.DrainForTest()
    if !failed { t.Fatalf("the next event must still fire onSync") }
}

func TestWatchTriggerNotGate(t *testing.T) {
    // a query during a re-index sees the last committed index + degraded:stale_index, never blocks
    w := watch.NewWatcher(t.TempDir(), watch.WatchConfig{Debounced: 5 * time.Millisecond}, nil)
    w.StartReindexForTest(50 * time.Millisecond) // a 50ms re-index in flight
    start := time.Now()
    _ = w.FreshnessForTest() // must return immediately (not block on the re-index)
    if time.Since(start) > 10*time.Millisecond { t.Fatalf("a query must not block on the re-index (CI-INV-4)") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/codeindex/watch/ -v`
Expected: FAIL — package `watch` does not exist (build error). BDD: the `[watch-debounce-burst-collapse]`, etc. scenarios are RED.

- [ ] **Step 3: Write minimal implementation**

`watch.go`: the fsnotify loop (the v1 `watch.go:113-125` — `fsnotify.Watcher` + a background `loop`). Debounce 2 s default, clamped 100 ms–60 s (the v1 `clampDebounce`). **Burst-collapse** (the v1 `watch.go:142-191`): the first event of a burst arms one `time.Timer(debounce)`; on fire, `drainPending()` collapses the burst into a **single** `onSync` call. **Fingerprint re-check**: on a debounced change, run the metadata-only stat-walk (size+mtime vs the `files` table + the extractor stamp, the v1 `fingerprint.go:112-144`); no real change → no re-index (a no-op). **Failure isolation** (the v1 `service.go:1925-1950` "Advisory: a failure keeps the old index live"): a re-index failure → log + continue, the watcher never stops. **Trigger, not a gate** (CI-INV-4): a query during a re-index sees the last committed index + `degraded: stale_index`, never blocks.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/codeindex/watch/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/codeindex/watch/ .skillgrid/artifacts/04-adr-0043-fsnotify-dep.md
git commit -m "feat(mnemonic-v2): fsnotify watch loop — debounce, burst-collapse, fingerprint re-check, failure isolation

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T06"
```

---

### Task 7: The `code_*` transport (16 agent + admin tools + `code_status` 4-state + tier + version + the CLI group + the T-CI-4 report fix)

> ⚠ one-way: **D1 — the `code_*` MCP tool contract**. The KB + the opencode/pi plugins consume it as their only interface; renaming a tool / adding a required param / changing a return shape after the KB builds breaks them.

**Files:**
- Create: `mnemonic-v2/internal/transport/handler/handler_code.go`, `mnemonic-v2/internal/transport/mcp/tools_code.go`, `mnemonic-v2/cmd/mnemonic/index.go`
- Test: `mnemonic-v2/internal/transport/mcp/tools_code_test.go`, `mnemonic-v2/cmd/mnemonic/index_test.go`

**Interfaces:**
- Consumes: Tasks 1–6 (the `Store`, `graph`, `search`, `watch`, `vectorstore`); the memory change's `transport/mcp` (the one server + the two-profile mechanism + the write queue, D-PROFILE) + `handler` (the shared handler pattern) + `cmd/mnemonic` (the CLI dispatch).
- Produces:
  - `handler_code.go`: the one shared code-index handler set (`Index`, `Search`, `Resolve`, `Packages`, `Impact`, `Affected`, `Callers`, …, `Status`, `Grep`, `Read`) — the MCP + CLI surfaces meet here; never diverge.
  - `tools_code.go`: the 16 agent-profile + the admin `code_*` tools as thin adapters over `handler_code` (the memory change's two-profile mechanism, D-PROFILE — NOT v1's `SKILLGRID_MCP_CODE_TOOLS` env-var filter). The v2.1-deferred tools (`code_pdg_query`, `code_taint`, `code_communities`, `code_god_nodes`, `code_explain_community`, `code_processes`, `code_process`) are **declared + report "run `code_index --<flag>`"**.
  - `index.go`: the `mnemonic index/status/search/orient/explain/path/impact/affected/grep/read/watch` CLI group (delegated by the memory change's `005-transport-cli`; `--json` matches the MCP contract).
  - `Status(ctx) (StatusResult, error)` — the **4-state freshness** (`fresh`/`lag`/`empty`/`unknown`) + counts + `capabilities` JSON + **per-language extractor tier** (D-TIER) + **schema version** + fair coverage + last index time + watch status (T-CI-4).
- Seam: the shared `handler` set (the MCP + CLI meet here) + the memory change's MCP server (the profile mechanism).
- Deletion test: deleting `handler_code.go` + `tools_code.go` re-creates the entire `code_*` contract in both the MCP server and the CLI — the only callers.
- Adapters: the MCP tool adapter + the CLI adapter = 2 over the shared handler, seam justified.

**SATISFIES:** [code-tool-contract], [code-profile-membership], [code-status-reports-version-tier-coverage], [code-index-queued-write-409], [v21-deferred-report-run-flag], [mcp-cli-parity]

(The two NEW tools' exact shapes — `code_resolve` pin-check + `code_packages` package-map — are asserted inside the `code-tool-contract` scenario; no separate scenario.)

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/transport/mcp/tools_code_test.go
// T-CI-4 (T18): code_status reports schema version + per-language extractor tier + fair coverage
// (NOT the v1 boolean stale — the v1 index status said stale:false while every graph read failed).
func TestCodeStatusReportsVersionTierCoverage(t *testing.T) {
    st := seedIndexWithTiers(t) // go=rich, python=map, the rest=regex; a stale file present
    r, _ := handler_code.Status(ctx)
    if r.SchemaVersion == "" { t.Fatalf("code_status must report schema version: %+v", r) }
    if r.Freshness != "lag" { t.Fatalf("a stale file → freshness=lag (not v1's boolean stale:false): %+v", r) }
    tiers := r.ExtractorTiers
    if tiers["go"] != "tree-sitter-rich" || tiers["python"] != "tree-sitter-map" {
        t.Fatalf("per-language tier (D-TIER): %+v", tiers)
    }
    if r.FairCoverage["go"] <= 0 { t.Fatalf("fair coverage must be reported: %+v", r.FairCoverage) }
}

func TestCodeToolContract(t *testing.T) {
    // exact tool names, required params, return shape, error codes (golden JSON per tool)
    tools := registeredCodeTools()
    for _, want := range []string{"code_status", "code_index", "code_search", "code_read", "code_orient",
        "code_explain", "code_path", "code_impact", "code_affected", "code_grep", "code_semantic",
        "code_resolve", "code_packages", "code_get_callers", "code_get_callees", "code_get_dependents",
        "code_get_implementors", "code_get_hierarchy", "code_get_tests_for"} {
        if !hasTool(tools, want) { t.Fatalf("missing agent tool: %s", want) }
    }
    // the two NEW tools' exact shapes (the KB's forward-refs)
    if !toolReturnsShape(tools, "code_resolve", `{"exists":bool,"called_from":[]string,"candidates":[]}`) {
        t.Fatalf("code_resolve contract")
    }
    if !toolReturnsShape(tools, "code_packages", `[]{"path":string,"files":int,"symbols":int}`) {
        t.Fatalf("code_packages contract")
    }
}

func TestCodeProfileMembership(t *testing.T) {
    // agent profile = the 16 agent tools; admin set (code_rename, code_unresolved, code_circles,
    // code_deps, code_dep_impact, code_embedding_status, code_watch, the v2.1-deferred, the knowledge-node reads) = all only
    agent := toolsForProfile("agent")
    all := toolsForProfile("all")
    for _, a := range agent { if !hasTool(all, a) { t.Fatalf("all must superset agent: %s", a) } }
    if hasTool(agent, "code_rename") { t.Fatalf("code_rename is admin-only (all), not agent") }
    if !hasTool(all, "code_rename") { t.Fatalf("code_rename must be in all") }
}

func TestCodeIndexQueuedWrite409(t *testing.T) {
    // code_index is a queued write: returns run_id+status immediately; a repeat POST for an active index → 409
    run1, _ := handler_code.Index(ctx, IndexOpts{})
    if run1.RunID == "" { t.Fatalf("queued write must return run_id: %+v", run1) }
    _, err := handler_code.Index(ctx, IndexOpts{}) // a second while the first is active
    if err == nil || err.Code() != 409 { t.Fatalf("repeat active index → 409, got: %v", err) }
}

func TestV21DeferredReportRunFlag(t *testing.T) {
    // the v2.1-deferred tools are declared + report "run code_index --<flag>" (not missing)
    for tool, flag := range map[string]string{"code_pdg_query": "--pdg", "code_taint": "--pdg",
        "code_communities": "--communities", "code_processes": "--process"} {
        r, err := callTool(tool)
        if err == nil { t.Fatalf("%s must report run-flag, got success: %+v", tool, r) }
        if !strings.Contains(err.Message(), "code_index "+flag) {
            t.Fatalf("%s must say run code_index %s: %q", tool, flag, err.Message())
        }
    }
}

// mnemonic-v2/cmd/mnemonic/index_test.go
func TestMcpCliParity(t *testing.T) {
    // mnemonic search X --json and the code_search MCP tool produce byte-identical JSON (same handler)
    cliJSON := runCLI("mnemonic", "search", "wireGitHooks", "--json")
    mcpJSON := callMCP("code_search", map[string]any{"query": "wireGitHooks"})
    if cliJSON != mcpJSON { t.Fatalf("MCP + CLI must be byte-identical (K5): %s vs %s", cliJSON, mcpJSON) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/transport/... ./cmd/... -v`
Expected: FAIL — `handler_code`/`tools_code`/`index.go` do not exist (build error). BDD: the `[code-tool-contract]`, `[code-status-reports-version-tier-coverage]`, etc. scenarios are RED.

- [ ] **Step 3: Write minimal implementation**

`handler_code.go`: the one shared handler set — each method calls the corresponding `graph`/`search`/`watch`/`Store` function + records a retrieval trail (INV-5) + wraps the result with the `degraded` envelope (INV-3). `Status` reads `index_freshness` (the 4-state freshness: `fresh`/`lag`/`empty`/`unknown` — the stat-walk vs the `files` table + the extractor stamp, NOT v1's boolean) + the counts + `capabilities` JSON + the **`extractor_tiers`** column (D-TIER) + the **schema version** (from the migration runner's applied set) + the fair coverage (measured from `edges`, the v1 `coverage.go:8-24`). `tools_code.go`: register the 16 agent + admin tools on the memory change's MCP server (the two-profile mechanism, D-PROFILE) — each a thin adapter over `handler_code`; the v2.1-deferred tools return a "run `code_index --<flag>`" error. `index.go`: the CLI group (the v1 `cmd/mnemonic` shape but with the v2 additions: top-level `mnemonic status`, `mnemonic read`, `mnemonic watch` — the v1 NOT-PORT gaps fixed).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/transport/... ./cmd/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/transport/ mnemonic-v2/cmd/mnemonic/
git commit -m "feat(mnemonic-v2): code_* MCP transport (16 agent + admin) + code_status 4-state/tier/version + CLI group

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T07"
```

---

### Task 8: The semantic vector store (D-CI8: BLOB + vec0 + HNSW + brute-force) + the advanced/deferred passes (`--lsp` + the v2.1 flag surface) + the Postgres dialect render

**Files:**
- Create: `mnemonic-v2/internal/codeindex/vectorstore/store.go`, `dim.go`, `hash.go`, `mnemonic-v2/internal/core/store/postgres_code.go`, `mnemonic-v2/internal/core/store/migrations_pg/100_code_index.sql`, `mnemonic-v2/internal/codeindex/extract/lsp.go`
- Test: `mnemonic-v2/internal/codeindex/vectorstore/*_test.go`, `mnemonic-v2/internal/core/store/postgres_code_test.go`, `mnemonic-v2/internal/codeindex/extract/lsp_test.go`

**Interfaces:**
- Consumes: Task 5's `search/semantic.go` (the semantic leg calls this), Task 1's `Store` (the BLOB tables), the memory change's `intelligence/embedder` seam (the 768-dim opt-in + the hash-64 default) + `lib/pq` (the Postgres dialect).
- Produces:
  - `vectorstore.Engine` — the BLOB (`embeddings`+`chunk_embeddings`) + vec0 + HNSW + brute-force floor (D-CI8); the preference order HNSW → vec0 → brute-force (the v1 *live* path, lifted); the dimension gating (768 pin) + the model-swap guard (clear vectors+indexes on `embedding_model` change, the v1 `indexer.go:1133-1160`). **LSH + IVF dropped** (not shipped — v1's `lsh_buckets` is schema-only, IVF is an adapter not wired).
  - `vectorstore.hash.go` — the hash-64 default embedder (the deterministic floor, the v1 `HashEmbedder` `embedder.go:43`).
  - `postgres_code.go` — the Postgres dialect render of the 7 tables (tsvector for the FTS, pgvector for the BLOB) — mirrors `sqlite.go`; the KB's `kb_*` tables ride the same render (T21).
  - `extract/lsp.go` — the optional LSP tier (`--lsp`, default off, CI-6): the language-server subprocess (gopls/pyright/tsserver/rust-analyzer/clangd on PATH), per-call 30 s timeout, all-or-nothing per call site (a failing/timing-out server → warn + continue, never a partial edge set), the resolved member call → `LSP_RESOLVED` edge (the 4th confidence tier, 0.95).
- Seam: the `Embedder` seam (the semantic leg degrades through it) + the LSP server subprocess (the `--lsp` tier) + the `lib/pq` driver (the Postgres dialect).
- Deletion test: deleting `internal/codeindex/vectorstore` re-creates the BLOB + vec0 + HNSW + brute-force engine in `search/semantic.go` — the only caller. Deleting `postgres_code.go` re-creates the Postgres dialect in `sqlite.go`'s callers.
- Adapters: the SQLite `Engine` + the Postgres `Engine` + the test engine = 3; the LSP server (prod subprocess + test fake) = 2, seam justified.

**SATISFIES:** [semantic-store-wired-3-index], [vector-preference-order], [model-swap-guard], [lsp-all-or-nothing-per-call-site], [postgres-dialect-render]

(The v2.1 flag surface — `--lsp`/`--pdg`/`--communities`/`--process` — is declared in Task 7's tool surface + reported by Task 7's `v21-deferred-report-run-flag` scenario; Task 8 implements the `--lsp` pass the flag drives.)

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/codeindex/vectorstore/store_test.go
func TestSemanticStoreWired3Index(t *testing.T) {
    e := vectorstore.NewEngine(st, emb768) // a 768-dim embedder
    e.EmbedSymbol(ctx, symID, "CapturePayment (sig)")
    r, idx, _ := e.Search(ctx, queryVec, 5)
    if idx != "hnsw" { t.Fatalf("768-dim embedder → HNSW is preferred, got %s", idx) }
    if len(r) == 0 { t.Fatalf("semantic search must return hits") }
    // LSH + IVF are NOT in the preference order (D-CI8 — dropped)
    if idx == "lsh" || idx == "ivf" { t.Fatalf("LSH/IVF must not be shipped: %s", idx) }
}

func TestVectorPreferenceOrderFallback(t *testing.T) {
    e := vectorstore.NewEngine(st, emb768)
    e.EmbedSymbol(ctx, symID, "x")
    e.DisableIndexForTest("hnsw") // HNSW not yet built
    _, idx, _ := e.Search(ctx, queryVec, 5)
    if idx != "vec0" { t.Fatalf("HNSW missing → vec0, got %s", idx) }
    e.DisableIndexForTest("vec0")
    _, idx2, _ := e.Search(ctx, queryVec, 5)
    if idx2 != "brute_force" { t.Fatalf("vec0 missing → brute-force floor, got %s", idx2) }
    // never a hard fail (the brute-force floor always answers)
}

func TestModelSwapGuard(t *testing.T) {
    e := vectorstore.NewEngine(st, emb768A)
    e.EmbedSymbol(ctx, symID, "x")
    e.SwapModelForTest(emb768B) // a different model
    // the model-swap guard clears vectors + indexes (a different model's vectors are not comparable)
    if e.CountVectorsForTest() != 0 { t.Fatalf("model swap must clear vectors") }
    if e.IndexBuiltForTest("hnsw") { t.Fatalf("model swap must clear indexes") }
}

// mnemonic-v2/internal/codeindex/extract/lsp_test.go
func TestLSPAllOrNothingPerCallSite(t *testing.T) {
    // the v1 member call site the static pass could not bind → LSP_RESOLVED when the server resolves it
    g, _ := extract.ResolveWithLSPForTest(ctx, fileWithMemberCall, "go", lspFakeServerThatResolves)
    var found bool
    for _, e := range g.Edges { if e.Kind == "calls" && e.Confidence == "LSP_RESOLVED" { found = true } }
    if !found { t.Fatalf("resolved member call → LSP_RESOLVED: %+v", g.Edges) }
    // a failing/timing-out server → the static-pass name-only edge stands (never a partial edge set)
    g2, _ := extract.ResolveWithLSPForTest(ctx, fileWithMemberCall, "go", lspFakeServerThatFails)
    var nameOnly bool
    for _, e := range g2.Edges { if e.Kind == "calls" && e.ToID == nil && e.Confidence == "AMBIGUOUS" { nameOnly = true } }
    if !nameOnly { t.Fatalf("failing LSP → static name-only edge stands: %+v", g2.Edges) }
}

// mnemonic-v2/internal/core/store/postgres_code_test.go
func TestPostgresCodeDialectRender(t *testing.T) {
    if dsn := os.Getenv("MNEMONIC_PG_TEST_DSN"); dsn == "" { t.Skip("no PG DSN (gate G-15)") }
    st, _ := store.Open(context.Background(), "/tmp", "p", store.Postgres, dsn)
    defer st.Close()
    st.Migrate(context.Background()) // the 7 tables from an empty schema
    // the code-index reads run on PG (dialect-agnostic walker, INV-2)
    if _, err := st.SymbolByName(context.Background(), "x"); err != nil { t.Fatalf("PG SymbolByName: %v", err) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/codeindex/vectorstore/ ./internal/codeindex/extract/ -run 'LSP|Semantic|Vector|ModelSwap' -v`
Expected: FAIL — package `vectorstore` does not exist (build error); `lsp.go` not yet present.

- [ ] **Step 3: Write minimal implementation**

`vectorstore/store.go`: the BLOB (`embeddings`+`chunk_embeddings`) + vec0 (blank-import `modernc.org/sqlite/vec`, the v1 `store.go:20`) + HNSW (pure-Go, M=16/M0=32/efConstruction=64/efSearch=96, the v1 `hnsw.go:26-37`) + the brute-force floor (cosine over the BLOBs). The preference order **HNSW → vec0 → brute-force** (the v1 *live* path `hybrid/durable.go:13-25`); a missing index → fall to the next, never a hard fail. `dim.go`: the 768-dim pin (the v1 `store.go:23-27`) + the model-swap guard (the v1 `indexer.go:1133-1160` — `embed_meta['embedding_model']` changed → DELETE `embeddings`+`chunk_embeddings`+`vec_symbols`+`vec_chunks` + rewrite). `hash.go`: the hash-64 default embedder (the v1 `HashEmbedder` `embedder.go:43` — deterministic FNV-1a token hash, L2-normalized, the `query: ` prefix on the query side). `lsp.go`: the optional LSP tier (the v1 `indexer.go:491` opt-in shape — the language-server subprocess, per-call 30 s timeout, all-or-nothing per call site, the resolved member call → `LSP_RESOLVED` 0.95). `postgres_code.go` + `migrations_pg/100_code_index.sql`: the Postgres dialect render (tsvector for the FTS, pgvector for the BLOB) — mirrors `sqlite.go`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/codeindex/vectorstore/ ./internal/codeindex/extract/ ./internal/core/store/ -run 'LSP|Semantic|Vector|ModelSwap|PostgresCode' -v`
Expected: PASS (the PG test skips without `MNEMONIC_PG_TEST_DSN`, gate G-15).

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/codeindex/vectorstore/ mnemonic-v2/internal/codeindex/extract/lsp.go mnemonic-v2/internal/core/store/postgres_code.go mnemonic-v2/internal/core/store/migrations_pg/
git commit -m "feat(mnemonic-v2): semantic vector store (D-CI8) + LSP tier + Postgres dialect render

[skillgrid-context] change=2026-10-31-mnemonic-v2-code-index task=T08"
```
