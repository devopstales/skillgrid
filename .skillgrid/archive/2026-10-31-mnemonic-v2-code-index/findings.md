# Findings — mnemonic-v2 code-index

> Consolidated evidence for this change. The blueprint cites this file (header
> `Findings:` line) for every design decision that rests on a research fact. Source
> root for all v1 references: `mnemonic_back/` (module `github.com/devopstales/skillgrid/mnemonic_back`).
> Research mode: **quick v1 code-research pass** (4 parallel researcher subagents) +
> **live v1 CLI probe** against this repo's indexed store (`~/.skillgrid/mnemonic/skillgrid.sqlite`,
> 375 files / 1051 chunks, last indexed 2026-09-07, binary dated 2026-10-18). All four of the
> user's capabilities (find files / find a feature / function connectivity / change blast-radius)
> were exercised end-to-end on the live v1 binary. **The live probe is the load-bearing evidence
> for the "good relevant answer" requirement** — it shows v1's `code_search` and `code_get_*` are
> *currently broken or low-relevance* on a real store, which the v2 blueprint must fix, not port.

## Live probe (2026-11-01): the four capabilities on the real v1 binary

Run from `/data/git/AI/skillgrid` against the installed `mnemonic` (v1, Oct 18) + the
`skillgrid.sqlite` store. **Root cause of every failure below: schema drift.** v1's
`001_schema.sql:552,559` defines `symbols.qualified_name` + `symbols.uid`, but the on-disk
store was created by an **older v1** that used `symbols.qualified` (no `uid`). The binary
(Oct 18) is newer than the store, so:

| Capability | v1 command run | Live result | Diagnosis |
|---|---|---|---|
| **1. Find files** | `mnemonic search -json "wireGitHooks"` (a real fn at `skillgrid-cli/internal/install/sync.go:79`) | Returns 5 hits, **none is the real function** — `AuthenticationService` (a docs file), `$NAME` (a metavariable), test helpers. **Identical 5 hits** for `capturePayment` (a non-existent symbol). Warning: `signal search: SQL logic error: no such column: s.qualified_name`. | The **signal leg is broken** (references `s.qualified_name`, store has `qualified`), and only the **semantic leg** answers — so a real symbol and a fake symbol return the *same* results. `code_search` is **not relevant** on this store. The `-fts` flag did not isolate FTS (all legs still ran). |
| **2. Find a feature** | (composition: `search` → resolve) | Same broken `search` output; no resolve tool exists in v1 (confirmed by researcher — `code_resolve` is v2-NEW). | No single "find feature" in v1. v2's `code_resolve` (NEW) is the fix, but it inherits the same `s.qualified_name`/`s.uid` column dependency → **v2 must pin its schema so a query can never reference a column the store lacks.** |
| **3. Function connectivity** | `mnemonic callers -json "wireGitHooks"` / `callees` | `error: SQL logic error: no such column: s.uid (1)` | **Hard failure.** `graph.go:92,114` + `resolve.go:38` select `s.uid`; the store has no `uid` column. The neighbor views (callers/callees/dependents/implementors/hierarchy/tests-for) are **all broken** on this store. |
| **4. Change blast-radius** | `mnemonic impact -json "wireGitHooks"` / `explain` / `orient` | `error: SQL logic error: no such column: s.uid (1)` (impact/explain); `no such column: s.qualified_name` (orient) | **Hard failure.** Same schema drift. `orient` (`code_orient`/`code_explain`) is broken on `qualified_name`. |
| **3b. Path (crown feature)** | `mnemonic path -json "wireGitHooks" "removeIfPresent"` | `error: SQL logic error: no such column: s.uid (1)` | **Hard failure** — the "shortest path / where the graph stops" read (v1's crown feature) is broken on this store too. |
| **3c. tests-for / implementors** | `mnemonic tests-for -json "wireGitHooks"` / `implementors` | `error: SQL logic error: no such column: s.uid (1)` | **Hard failure.** |
| **STRUCTURAL LEG (the one that WORKS)** | `mnemonic grep -json '(function_declaration) \fn' skillgrid-cli/internal/install/sync.go` | Returns the real decl at `sync.go:87` (line + col + matched text + `captures.fn`). `'(call_expression) \call'` returns every call site in the file. | **Works** — it is **index-free** (re-parses the tree with gotreesitter), so it is immune to the `symbols` schema drift that breaks every stored-graph read. The v1 pattern syntax is the **node-type + capture** form: `(function_declaration) \fn`, `(call_expression) \call`, `\(ARGS*)` for an arg run, `\*`/`\_` wildcard steps, `\NAME` a capture. (The `\f(ARGS)` shorthand the spec/CLI help implies is *not* what the parser accepts — a bare `\f` is a malformed metavariable; the node type must be named. This is a v1 doc/UX gap.) |
| **index status (the health read)** | `mnemonic index status --json` | `{chunk_count: 1051, file_count: 375, last_indexed: 2026-09-07, stale: false, watcher: true}` — **no** schema version, **no** per-language coverage, **no** extractor tier. | Works but **under-reports**: it says `stale: false` (the index is a *fresh* walk of 375 files) while the graph reads all fail — so the agent has *no signal* that the stored-graph reads are broken. The index is healthy; the *schema* is not. |

**What worked:** `mnemonic index status --json` (returns `chunk_count: 1051`, `file_count: 375`,
`last_indexed`, `stale: false`, `watcher: true` — but **no** schema version / coverage / tier) and
`mnemonic grep` (the **structural leg** — index-free, re-parses the tree, immune to the `symbols`
drift). So: the *index exists and is a fresh walk* (375 files, 1570 symbols) and the *structural
leg works*, but **every stored-graph read** (`callers`/`callees`/`impact`/`explain`/`orient`/
`path`/`tests-for`/`implementors`) **hard-fails** on the column-name drift.

**Precise root cause (confirmed against the migration set, 2026-11-01):** the store
`skillgrid.sqlite` applied migrations **001 → 060** (it has `042_vec0`, `046_entity_aliases`,
`056_ivf`, `057/058_hnsw`, `059`, `060_coverage`) and the `symbols` table has **8 columns:
`id, file_id, name, qualified, kind, start_line, end_line, signature, doc`** — i.e. `qualified`,
**no `uid`, no `qualified_name`**. The current v1 code's `001_schema.sql` `CREATE TABLE symbols`
*does* define `qualified_name` (line 552) + `uid` (line 559), but **there is no
`ALTER TABLE symbols ADD COLUMN uid` / `ADD COLUMN qualified_name` migration anywhere in the set**
(the only `ALTER TABLE symbols` lines are for `hub_score`/`return_type`/`param_types`/
`visibility`/`is_exported`). `046_entity_aliases.sql` creates a *separate* `entity_aliases` table
with its own `qualified_name` column — it does **not** backfill `symbols`. **So a store created by
the old 001 (which used `qualified`) runs 001→060 and never gains `uid`/`qualified_name`, while
the Oct-18 binary's graph queries (`graph.go:92,114`, `resolve.go:38`) select them → every graph
read fails `no such column`.** This is a **genuine v1 bug** (the schema evolution is missing the
bridge ALTER), not a user error — the index is up-to-date and the store is at the highest
migration. **The v2 blueprint must carry a T-CI-3 migration-parity test that proves a fresh store
AND a store opened at each prior migration version can run every `code_*` read** — the exact class
of bug that broke v1 live.

**The four "good relevant answer" requirements the v2 blueprint must meet** (derived from the
live probe, not the spec's claim):

1. **`code_search` must return the real symbol for a real symbol** — the FTS leg (BM25 over
   `symbol_fts`/`chunks_fts`) must be the *primary* floor and must actually match
   identifier-shaped queries (the live probe's FTS leg was not isolating / not matching
   `wireGitHooks`). A real symbol and a non-existent symbol must **not** return the same hits.
2. **`code_search` must never let a broken leg dominate** — a leg that errors (e.g. a SQL
   column error) must degrade that leg to `degraded: <leg>` + drop it from the fusion, not
   emit a warning while the broken leg's (empty) result still sets the answer. The live probe
   showed the warning but the broken leg still shaped the output.
3. **Every `code_*` graph read must survive a schema-version mismatch with a clear error,
   not a SQL panic** — a query must reference only columns its migration set guarantees. v2
   pins the schema (the 7 tables are defined in this change's migrations); a store opened at
   a lower schema version → the read fails with a *version* error ("run `mnemonic index`
   again / re-migrate"), never `no such column`.
4. **`code_status` must report the schema version + per-language extractor tier + fair
   coverage** (the live `index status` reported only counts + `stale: false` — no coverage,
   no tier, no version, so the agent had no signal that the graph reads were about to fail).

These four are **Must-Haves / truths** in the blueprint (T-CI-1…T-CI-4) and get held-out
tests: (1) a real-symbol-vs-fake-symbol search divergence test, (2) a broken-leg-degradation
test, (3) a stale-schema read test, (4) a `code_status` coverage/tier/version report test.

## Research: v1 code-index mechanics (source: `mnemonic_back/`)

Four dimensions investigated: extraction, graph queries, search + vector store,
watch + transport. Findings below are the load-bearing facts the blueprint rests on.
Each claim carries a file:line source. "NOT-PORT" marks v1 quirks the blueprint
deliberately does *not* lift; "NEW" marks what the v2 spec adds over v1.

### Extraction ladder

- **Rich one-pass tier = exactly 6 languages** — go, python, java, javascript, typescript, tsx
  (`coveredLanguages`, `internal/extract/adapter.go:15-18`). They call gotreesitter
  (v0.55.1, `go.mod:13`) `ExtractDefinitionSpans`/`ExtractCalls`/`ExtractHeritage`/`ExtractImports`
  (`mapTree`, `adapter.go:233-294`).
- **25 per-language node-type maps** (the "map" tier), stored as data in
  `langMapDef{defNodes map[string]defInfo, callNodes map[string]callInfo, imports func, heritage func}`
  (`languages.go:13-18`, `langMaps` table `languages.go:133-159`). One `walkTree`
  (`adapter.go:301-325`) matches node types against the maps. (v2 spec says "24"; v1 is 25
  with `c` aliasing `cpp` — a data-table detail, not a contract.)
- **Regex fallback** (`fallback.go:50-162`): function/class decl regexes emit
  `function`/`class`; import regexes (`import`/`use`/`#include`/`require`/`from…import`)
  emit AMBIGUOUS import edges; each function symbol is chained to the next as an
  AMBIGUOUS call edge (`fallback.go:150-159`). `fallbackRegexes` (`fallback.go:10-23`)
  is declared but **not used in the extraction loop** — dead.
- **Self-healing pool** (`extract.go:152-172`): `MaxRespawns=3`, `TripAfter=3`. A panicking
  grammar quarantines the file to regex and trips a per-language circuit breaker after 3
  deaths (`recordDeath`, `extract.go:207-217`; recover in `adapter.go:48-62`). "Respawn" is a
  counter only — no actual worker respawn.
- **Extractor stamp** (`fingerprint.go:17`): a hardcoded `"v1"` const, exposed via
  `ExtractorStamp()`. Stored in a `fingerprint_meta` side-table. **NOT-PORT:** it is not a
  content hash of the extractor — v2 needs a real content hash for the "extractor changed →
  full re-walk" signal to be honest (Decision D-STAMP below). `fingerprintDigest`
  (`fingerprint.go:71-84`) is defined with **no caller** — dead.
- **Chunker** (`indexer.go:327-439`, `ChunkLinesAst`): AST-first — one chunk per
  function/method/class (≤ `semanticChunkTargetChars=1000`); leading segment < `minChunkChars=250`
  merged into the next; inter-symbol gaps become `lines` segments; over-long symbols
  line-windowed at **hardcoded** 80/10 (`indexer.go:433`). No function-like symbols → nil →
  caller falls back to plain 80-line chunks (`indexer.go:577-580`). Config defaults
  `ChunkLines=80`/`ChunkOverlap=10` (`config/load.go:561-562`).

### Graph read API

- **Reverse-BFS impact** (`graph/impact.go:81-180`): queue BFS over reverse dependents;
  `max_depth` default **5** (`impact.go:86-89`); confidence filter via `confidenceRank`
  (EXTRACTED=3, INFERRED=2, AMBIGUOUS=1) — an edge passes iff `rank(edge) >= rank(min)`,
  empty `MinConfidence` keeps everything (`impact.go:47-66`). **Excluded hops are counted,
  not dropped** (`impact.go:149-153`). Risk tier by depth: depth 1 = "WILL BREAK", deeper =
  "LIKELY AFFECTED" (`tierForDepth`, `impact.go:68-75`). **NOT-PORT:** the name-only reverse
  query does *not* apply the multi-candidate AMBIGUOUS relabel that `fetchEdges` does — v1 is
  internally inconsistent; v2 wants drop-not-guess consistency (spec CI-INV-3).
- **Neighbor views** (`graph.go:288-309`, `viewMatches` `graph.go:267-284`): callers/callees/
  dependents/implementors/hierarchy/tests-for. Name-only edges resolve **at query time** in
  `fetchEdges` (`graph.go:145-238`): 0 candidates → kept name-only (`To.ID==0`); 1 → resolved,
  original confidence; >1 → **one edge per candidate, each relabeled AMBIGUOUS** (`graph.go:217-226`).
- **`Resolve`** (`resolve.go:35-76`): the name→symbol disambiguator — 0=NotFound, 1=Target,
  >1=Ambiguous with all candidates. **This is the exact primitive `code_resolve` needs** —
  the blueprint lifts it, not reinvents.
- **Path + graph-stops** (`graph.go:344-399` `Path`; `buildGraphStops` `graph.go:442-481`):
  BFS over resolved endpoints; when unreachable, scans every visited node's unresolved edges
  (`To.ID==0`) as `RefusedMatch`, takes the **max line**, and reports `DispatchKind` as the
  kind of the **last-iterated** such edge. **NOT-PORT:** v1's `DispatchKind` is last-wins over
  the whole visited set, not "the dispatch kind that causally ended the flow" — the spec's
  framing overstates v1's mechanism.
- **`code_affected`** (`affected/`): three input modes — `Changed[]` (`affected.go:54-55`),
  stdin git-diff list (`affected.go:235-261`), base ref via
  `git merge-base <ref> HEAD` + `git diff --name-only` (`base.go:40-57`). Seeds = every symbol
  in a changed file. Depth-capped (5) reverse BFS over a **superset** of impact kinds (adds
  `tests_for`/`tested_by`). Test files collected by **path convention only** (`isTestPath`,
  `affected.go:63-80`): Go `_test.go`, Python `test_*.py`, Ruby `_test.rb`, JS/TS `.test.*`.
  **NOT-PORT gap:** Java (`*Test.java`) and others are not recognized in v1 — v2 should fix.
- **Confidence scores** (`extract.go:24-34`): EXTRACTED→1.0, INFERRED→0.85, AMBIGUOUS→0.5
  (`LSP_RESOLVED`→1.0, i.e. treated as EXTRACTED in v1 — v2 spec splits it to 0.95).

### Search + vector store

- **Hybrid RRF fusion** (`hybrid/rank.go:124-271`): FTS leg (BM25 `bm25(chunks_fts)` +
  `symbol_fts`) + an in-Go signal leg (identifier-token overlap, TF-IDF-style) + a semantic
  leg (only if `opts.Embedder != nil`). Fusion `score = Σ 1/(K + rank + 1)`, **K=60**
  (`RRFK`, `rank.go:22-23`). **Per-hit provenance** `Hit.Provenance{FTS,Signal,Semantic,Sim}`
  (`rank.go:25-31`). A down embedder degrades to FTS+signal, never a hard fail
  (`rank.go:205-211`). **NOTE:** v1's "signal leg" is a *fourth* leg beyond the spec's three —
  the v2 spec's three legs (FTS/structural/semantic) drop it.
- **Structural by-example AST grep** (`structural.go:299` `GrepByExample`): index-free,
  re-parses the tree per file. Metavariable syntax `\NAME`→`@name` capture, `\(ARGS*)`→
  `(argument_list)`, `\*`/`\_`→anonymous wildcard (`structural.go:64-68, 83-205`). Invalid
  pattern for a language → `GrepNote`, that language skipped, others still match
  (`structural.go:346-363`).
- **Vector multi-store reality (CI-8 source of truth):**
  - BLOB source-of-truth: `embeddings` + `chunk_embeddings` + `embed_meta`
    (`001_schema.sql:618-629`, `chunk_embeddings` added by migration 037).
  - **vec0**: virtual tables `vec_symbols`/`vec_chunks` `float[768]`
    (`042_vec0_tables.sql:12-19`), blank-import `modernc.org/sqlite/vec` (`store.go:20`).
  - **HNSW**: pure-Go, M=16/M0=32/efConstruction=64/efSearch=96 (`hnsw.go:26-37, 95-99`),
    `hnsw_nodes`/`hnsw_links` partitioned by `source` (057/058 migrations).
  - **IVF**: pure-Go kmeans-lite, `ivf_centroids`+`ivf_assignments` (056, `ivf.go:290-393`) —
    a `VectorIndex` adapter but **NOT wired into the hybrid query path**.
  - **LSH**: `lsh_buckets` table exists (001:631-636) but **no Go code reads or writes it** —
    schema-only, dead.
  - **Actual query preference order** (durable path, `MNEMONIC_VECTOR_DB=1`):
    **HNSW → vec0 → in-memory cosine over the BLOBs** (`hybrid/durable.go:13-25`,
    `vectorcache.go:99-122`, `store.go:221-262`). LSH and IVF are absent from the path.
- **Dimension gating** (`store.go:23-27` `Dim=768`; `indexer.go:1213-1219`): the vec0 dual-write
  is gated `if dim == 768`; a non-768 embedder leaves the ANN indexes empty, BLOBs still hold
  any dim. **Model-swap guard** (`indexer.go:1133-1160`): if `embed_meta['embedding_model']`
  changed, DELETE `embeddings`+`chunk_embeddings`+`vec_symbols`+`vec_chunks` in one tx + rewrite.
- **Embedder default reality:** `Onnx` provider = nomic-embed-code, 768-dim (`onnx.go:14-15`),
  loads from `~/.skillgrid/models/` — but **ONNX inference is not actually wired**: it loads the
  session then falls through to the hash embedder (`"tokenize → run → pool is wired in a
  follow-up"`). The "auto-download" claim (`onnx.go:26-29`) has no backing code. So v1's *live*
  semantic path is effectively **hash-64-dim brute-force over the BLOBs** (`HashEmbedder`,
  `embedder.go:43` default 64-dim), not nomic-768 via HNSW.

### Watch + transport

- **Watch** (`codeindex/watch.go:113-125` `Start`): fsnotify on the index root + a background
  `loop`. Debounce `DefaultDebounce=2000ms`, `MinDebounce=100ms`, `MaxDebounce=60s`
  (`watch.go:20-42`, `clampDebounce`). **Burst-collapse**: first event of a burst arms one
  timer; on fire, `drainPending()` collapses the burst into a **single** `OnSync` call
  (`watch.go:142-191`). Re-index failure → result discarded, watch continues
  (`service.go:1925-1950` "Advisory: a failure keeps the old index live"; `mcp.go:109`).
- **Fingerprint gate** (the query-path staleness check): `FingerprintDriftDB`
  (`fingerprint.go:112-144`) is a stat-only walk (size+mtime) vs the `files` table + the
  extractor stamp; a stamp mismatch forces every file "changed" (`fingerprint.go:121-127`).
  The MCP `preQuery` (`mcp/freshness.go:69-129`) runs it before answering + triggers a
  structural-only re-index on drift.
- **MCP `code_*` surface** (~35 tools, `internal/mcp/server.go:42-115`): `code_status`,
  `code_index`, `code_search`, `code_read`, `code_diff_impact`, `code_orient`,
  `code_signature`, `code_file_toc`, `code_rationale`, `code_grep`, `code_explore`,
  `code_impact`, `code_path`, `code_explain`, the six `code_get_*`, `code_hybrid_search`,
  `code_semantic_search`, `code_embedding_status`, `code_communities`, `code_god_nodes`,
  `code_explain_community`, `code_route`, `code_navigates`, `code_unresolved_refs`,
  `code_processes`, `code_process`, `code_docs`, `code_configs`, `code_sql_schema`,
  `code_sql_access`, `code_affected`, `code_rename`, `code_pdg_query`, `code_taint`.
  Oracle: `TestAllToolsRegistered` (`server_test.go:14-98`).
  - **`code_resolve` and `code_packages`: NOT in v1** (full-repo grep: no matches). The v2
    spec calling them NEW is **confirmed accurate**.
  - **No agent/admin profile in v1** — it's a listed/unlisted filter
    (`applyCodeToolFilter`, `tools_code_explore.go:64-85`, env `SKILLGRID_MCP_CODE_TOOLS`).
    **NEW in v2:** an explicit `agent`/`all` two-profile split matching the memory change.
- **`code_status` report** (`tools_code.go:70-164`): `file_count`, `chunk_count`,
  `last_indexed`, `stale` (= no files or never indexed), `fair_coverage` (per-language,
  measured from `edges`, `graph/coverage.go:8-24`), `unresolved_refs`, `unresolved_members`,
  `resolution_audit`, `import_cycles`, `schema_fingerprint`, and `capabilities` (a **3-key**
  block: `graph`/`fts`/`vector`). **NOT-PORT / NEW:** no per-language **extractor tier**
  field in v1 — v2's `code_status` (spec 008 §5) adds it (tree-sitter-rich / map / regex).
  v1's `stale` is a boolean (empty/never-indexed), not the 4-state freshness the v2 spec
  wants (`fresh`/`lag`/`empty`/`unknown`).
- **CLI** (`cmd/mnemonic/main.go:84-164`): `index [DIR]` (`--dir/--project/--pdg/--lsp`,
  `index status` sub), `search QUERY` (`--json/--fts/--semantic/--limit`), `search
  affected|rename|pdg|taint`, `orient SYMBOL`, `grep PATTERN`, `callers|callees|dependents|
  implementors|hierarchy|tests-for`, `path FROM TO`, `explain SYMBOL`, `impact SYMBOL`.
  **NOT-PORT:** no top-level `read`, no top-level `status` (only `index status`), and
  **`watch` is not a CLI subcommand** in v1 (it's internal to the `mcp` server via
  `--no-watch`). v2 spec adds `mnemonic read` + `mnemonic watch` + a top-level `mnemonic status`.

## Decisions settled by this change (owed-decision gate)

These are the load-bearing choices the blueprint must name. None block the spec set;
all are defaulted here so the blueprint is not "build on a guess."

### D-CI8 — Vector-store scope: wire 3 indexes, drop LSH + IVF from the v2 claim

**Decision:** v2 ships a **wired, working 3-index semantic store**: `embeddings` +
`chunk_embeddings` (BLOB source of truth) + `vec0` + `HNSW` + a brute-force-over-BLOBs floor.
The query engine preference order is **HNSW → vec0 → brute-force** (the v1 *live* path,
lifted). **LSH and IVF are dropped from the v2 claim** — v1's `lsh_buckets` is schema-only
(no reader/writer) and IVF is an adapter not wired into the query path; shipping them as
"declared, not wired" adds tables + a query-branch with no v1 precedent and no KB/plugin
consumer.

**Embedder stance:** the store is **model-agnostic** (stores whatever `dim` the embedder
produces; the `embed_meta` model-swap guard is lifted from v1). The **default** embedder is
the deterministic **hash-64** floor (cgo-free, no download, always works — v1's `HashEmbedder`
lifted). A real **768-dim** embedder (ONNX nomic-embed-code *with the pool actually wired* —
the v1 "follow-up" — or an OpenAI-compat provider over the shared embedder seam) is an
**opt-in** that activates HNSW/vec0 when `dim == 768`. This matches INV-3 (LLM-first with
explicit `degraded`): no real embedder up → `degraded: no_vector` (FTS + structural floor);
768-dim embedder up but ANN index not yet built → `degraded: brute_force`.

**Why not carry LSH/IVF declared-but-unwired (like the v2.1 passes):** those are *deferred
analyses* (PDG/communities/process) that v2.1 will implement and that the spec pins an output
contract for. LSH/IVF are not deferred analyses — they're *v1 vector indexes that v1 never
actually wired*. Re-adding them in v2.1 is additive (a new table + a query-branch) and costs
nothing to defer. The difference: v2.1's contracts are *specified*; LSH/IVF would be
*assumed*.

**Reversibility:** not a one-way door. Re-adding LSH/IVF in v2.1 is additive (a new derived
index over the BLOBs + a query-branch). Recorded as a scope decision, not a migration.

### D-STAMP — Extractor stamp is a real content hash, not a const

**Decision:** `index_freshness.extractor_stamp` is a **deterministic hash of the extractor
code/config** (the rich-6 extractors + the 25 node-type maps + the regex battery + the
chunker parameters + the include/exclude defaults), computed at build (a
`//go:generate` or a const the extract package exposes). v1's hardcoded `"v1"` const is not
portable — a const never changes, so the "extractor changed → full re-walk" signal never
fires. **Why it matters:** spec `001` §4 says "a stamp mismatch → every file 'changed' (full
re-walk)"; with a const, that branch is dead. The hash must be cheap + stable (FNV-1a over a
serialized extractor-fingerprint string, mirroring v1's `fingerprintDigest` shape
`fingerprint.go:71-84` but with a real caller).

**Reversibility:** additive — changing the hash algorithm changes the stamp value once (a one-time full re-walk), never a schema migration.

### D-PROFILE — The `code_*` family re-uses the memory change's agent/all profile mechanism

**Decision:** the v2 `code_*` surface is exposed under the **two-profile** split
(`agent` / `all`) **defined by the memory change** (`00-foundation/004-transport-mcp` §2),
not v1's listed/unlisted env-var filter (`SKILLGRID_MCP_CODE_TOOLS`). The 16 agent-profile
`code_*` tools (spec 008 §3) are the default for plugins; the admin set (`code_rename`,
`code_unresolved`, `code_circles`, `code_deps`, `code_embedding_status`, `code_watch`, the
v2.1-deferred reads, the knowledge-node reads) is `all`-profile only. **Why:** v1 has no
profile concept — v2's profiles are a *new* design decision shared with the memory change,
and the memory change's MCP server is the substrate code-index's transport rides on (the
one-server-three-families layout). The blueprint re-uses the memory `mcp.Server` profile
mechanism; it does not re-invent a filter.

**Dependency note:** this task is precondition-gated on the memory change's `internal/
transport/mcp` (the profile mechanism) landing — see the execution-dependency gate below.

### D-TIER — `code_status` reports a per-language extractor tier (NEW in v2)

**Decision:** v2's `code_status` adds a **per-language extractor-tier** field
(`tree-sitter-rich` / `tree-sitter-map` / `regex`) that v1 lacks. The tier is **derived from
the extraction run** (which ladder tier actually produced each language's symbols — recorded
during the scan, not re-inferred), stored per-language in `index_freshness` (a new
`extractor_tiers` JSON column alongside `capabilities`), and reported by `code_status`.
**Why it's NEW, not a port:** v1's `code_status` has per-language *fair coverage* (a measured
share) but no *tier* — the degradation (this language fell to regex) is invisible in v1.
Spec 008 §5 requires it; the blueprint builds it. A low tier + low fair coverage in a language
is the "the extractor is weak here" signal (spec 004 §9).

**Reversibility:** additive — a new column + a new report field.

### D-KB — Cover the shared machinery the KB index builds on, not the KB tables

**Decision:** the code-index change does **not** own the KB's `kb_pages`/`kb_pages_fts`/
`kb_claims` tables or the `kb_index`/`kb_reindex` rebuild — those are the KB change's work
(KB `004-index` §3.1: "the KB and memory share the file but not the interface… neither
imports the other"). It **does** own the **shared machinery** the KB index builds on, so the
KB change can add its tables without a second foundation pass:

1. **The `Store` table-adding seam** — a dialect-neutral mechanism the KB change's `KBStore`
   uses to add its `kb_*` tables to the *same* `Store` (SQLite | PG) without forking the
   dialect code (KB `004-index` §3.1 "same store, different query surface").
2. **The reusable FTS rendering unit** — the external-content FTS5 DDL + the four-trigger
   sync set + the ranking-weights shape, extracted into its own tested unit (`store/fts.go`)
   so the KB's `kb_pages_fts` re-uses it verbatim (KB `004-index` §3.1: "matches the memory
   change's FTS rendering exactly — same pattern, same triggers, same ranking weights
   shape").
3. **The 4-state file-hash freshness model** — the shared staleness definition
   (`fresh`/`lag`/`empty`/`unknown`) the KB's `degraded: stale_index` + `kb_lint` reuse
   (KB `004-index` §5: "the same rule `kb_lint` uses… so the two can never disagree").
4. **The migration-parity regression (T-CI-3)** — structured to iterate a *table-set* (not a
   hardcoded list), so the KB change extends the same regression over its `kb_*` tables and
   the v1 bug class (a missing `ALTER` bridge) is caught for the KB too.

**Why "cover the machinery, not the tables":** the KB spec is explicit that the KB owns its
`kb_*` tables + `kb_index`/`kb_reindex` (KB `004-index` §3, §4). The code-index and the KB
share the project DB (KB-O2: no separate `kb.sqlite`) but not the interface. The code-index
change's job is to make the foundation (Store + FTS pattern + freshness + parity test)
*complete and reusable*, so the KB change's first task is "add my tables" not "build the
foundation again." This is **T21** in the blueprint.

**Reversibility:** not a one-way door. If the KB change later needs a *second* foundation
pass (because the machinery was incomplete), that's a KB-change task, not a code-index
re-work — the seam is additive.

## Execution-dependency gate (serial development)

> **Status (amended 2026-11-02, T0): the gate is SATISFIED.** The memory change shipped
> 2026-11-02 (20/20 tickets, QA PASS, archived to `.skillgrid/archive/2026-10-31-mnemonic-v2-memory/`).
> Every foundation piece below is **Built** — the code-index change is unblocked and is
> `current_change`. The original 2026-11-01 table is kept as history.

The code-index **depends on the memory change's foundation** (as of 2026-11-01 only
`internal/core/store` + `internal/core/syncid` were built; commits `15e009e1`…`36e1dd86`):

| Foundation piece | Status (2026-11-02, post-ship) | Code-index tasks that need it |
|---|---|---|
| `Store` interface + `SQLiteStore` (`internal/core/store`) | **Built** (memory TICKET-01/02) | All (the 7 content + 2 FTS tables are additive migrations on this store) |
| `syncid` (INV-6) | **Built** | `files.sync_id`, `symbols.sync_id` minting |
| `internal/intelligence` (Embedder seam + degrade) | **Built** (memory TICKET-03) | The semantic leg, the model-swap guard, `degraded: no_vector` |
| `internal/core/config` + `internal/core/project` | **Built** (memory TICKET-04) | `MaxFileSize`, include/exclude, debounce, project detection |
| `internal/transport/mcp` (one server, profiles, write queue) | **Built** (memory TICKET-13) | The `code_*` family transport (D-PROFILE), the `code_index` queued write |
| `cmd/mnemonic` (CLI dispatch) | **Built** (memory) | The `mnemonic index/search/…` CLI group |

**Policy (per the user's "plan full; execute after memory ships" call, now executed):** the
blueprint was written in full; the precondition gate lifted at the memory ship (2026-11-02).
Execution proceeds from TICKET-01 (the go-tree-sitter door check).

## Carry-forward (to the blueprint)

- **NEW vs PORT classification** for the `code_*` surface: `code_resolve` + `code_packages`
  are NEW (lift `Resolve` from v1 `resolve.go:35-76`); the other 14 agent tools are v1 ports
  (rename/reshape where the v2 spec changes the contract); the admin set is mostly v1 ports.
- **NOT-PORT list** (v1 quirks the blueprint deliberately fixes): the impact name-only
  relabel inconsistency, `buildGraphStops` last-wins `DispatchKind`, the hardcoded stamp
  const, `code_affected`'s missing Java test-file convention, v1's dead `lsh_buckets`/IVF/
  `fallbackRegexes`/`fingerprintDigest`, and the 4-leg (signal) RRF that the v2 spec reduces
  to 3 legs.
- **Confirmed-lift list** (v1 mechanics the blueprint ports with confidence): the tree-sitter
  ladder + self-healing pool + circuit breaker, the chunker, reverse-BFS impact (depth-cap 5,
  confidence rank, excluded-counted), the by-example AST grep, RRF K=60 + per-hit provenance,
  the fsnotify watch (debounce 2s clamp 100ms–60s, burst-collapse), the stat-walk fingerprint
  gate, the model-swap guard, the HNSW/vec0/brute-force vector preference order.
- **New dependencies (ADR required, AGENTS.md § Rules):** `gotreesitter` (v1 v0.55.1),
  `fsnotify` (v1 v1.10.1), `onnxer` (v1 v0.7.0, only if the ONNX 768-dim embedder is wired —
  the hash-64 default does not need it). All are already in `mnemonic_back/go.mod` (proven
  cgo-free) but are new to `mnemonic-v2/go.mod` → each needs an ADR before it lands.

## External validation + design folds (2026-11-01, post-plan)

Two external sources were reviewed to validate the design and surface transferable
ideas: **`nashsu/llm_wiki`** (21.3k★ — a Tauri/Rust desktop app that turns *documents*
into an LLM-maintained wiki, graph over wikilinks) and the **Neo4j `ki` article**
(*"Scaling Karpathy's LLM wiki: Why your knowledge base needs a graph"*,
knowledge-index.ai / github.com/zach-blumenfeld/knowledge-index, Aug 2026 — model a
corpus as a graph so the agent *navigates structure*, deterministic + zero-index-time-LLM).

**Validation (the design is already the `ki` shape, with three things `ki` lacks):**

| `ki` (Neo4j, over markdown) | code-index (SQLite, over code) | status |
|---|---|---|
| `Document→Section` containment | `files→chunks` (AST-first) | have it |
| `NEXT_SECTION` reading order | chunker AST ordering | have it (T3) |
| `LINKS_TO` wikilinks | `edges` (calls/extends/imports, confidence-labeled) | have it (T1/T3) |
| unresolved wikilink **stub nodes** | **name-only edges** (`to_id NULL`, CI-INV-3) | have it — stronger (drop-not-guess) |
| `ki outline` map, re-root, `--depth` | `code_packages` unit + (now) `code_orient` map | **folded** (T4) |
| `ki search --under` subtree scope | `code_search under?` prefix | **folded** (T5) |
| `ki get --type content` drill handle | `code_read mode=content` | **folded** (T5) |
| in-degree centrality "read first" | `code_get_callers` count | have the data |
| `shortestPath` "throughline" | `code_path` | have it (T9) |
| Leiden+conductance "themes" (roadmap) | `--communities` v2.1 deferred | have it, deferred — **conductance + determinism folded** (T8 spec `005`) |

The article's NICD benchmark (vector+graph ≫ vector-only: **>2× precision/recall,
truthfulness +80%, relevancy +69%**) is the *independent* external justification for why
the code-index (a navigable graph, not just a search box) is worth building — and for why
the 4-capability framing (find files / find feature / connectivity / blast-radius) is
the right shape. The article's headline cost win (**zero index-time LLM tokens,
deterministic + idempotent**) is *exactly* CI-7 (the engine extracts, the agent never
hand-edits) + the real content-hash `extractor_stamp` (D-STAMP) — already in the plan,
now with a citation. The v2 differentiator vs `ki`/v1: the graph is **dual-dialect
(SQLite+Postgres) + `sync_id`-sync-able** — one navigable code graph many agents read,
the cross-machine identity the article's "team's shared docs / agent's long-term memory"
scaling tier needs.

**Folds made (additive to the 8 tasks; no one-way doors touched):**

- **T4 — `code_orient` is the navigation entry point** (the `ki outline` map):
  progressive-disclosure, re-rootable on symbol/package/file, `depth`-bounded,
  outbound edges with stable ids. Replaces v1's `code_orient`/`code_signature`/
  `code_file_toc` three tools. Spec `004` §8.
- **T5 — `code_search under?`** subtree scope (a `files.path` prefix over all three
  legs — the `ki search --under`; the context-rot defense for large repos). Spec
  `006` §5.1.
- **T5 — `code_read mode=content`** drill handle (signature + callee/caller ids, not
  bodies — the `ki get --type content`). Spec `006` §7.
- **T8 spec `005` §5 — communities cohesion = conductance** (not edge density; the
  "tightly/loosely interlinked" word) + **determinism pinned** (fixed `leiden_seed` +
  single-threaded, part of the cache key). The `community_meta` table gains
  `conductance`/`density`/`leiden_seed` columns. (v2.1 — deferred, but the contract is
  fixed now so the v2.1 pass is additive.)

**Not folded (resisted the pull — documented so the decision is visible):**

- The **4-signal document relevance model** (direct-link ×3, source-overlap ×4,
  Adamic-Adar ×1.5, type-affinity ×1) is a *wiki-page* signal (pages sharing the same
  raw source). Code `edges` already encode relationship kind + confidence — an
  Adamic-Adar "shared neighbors" score is a v2.1 communities-pass feature, not a v2
  search leg. Not bolted onto `code_search`.
- `purpose.md` / `log.md` / two-step CoT ingest / Deep Research / the review queue are
  **wiki-generation machinery → the knowledge-base change**, not the code-index (the
  D-KB boundary keeps the LLM-maintained-wiki layer entirely the KB's, on top of the
  code-graph this change provides).

### 10-source review (2026-11-01) — retrieval-as-reasoning + the anti-pattern defense

Ten further sources were reviewed (the arXiv LLM-Wiki paper, the OpenKB/PageIndex
announcement, the GraphRAG/HippoRAG/LightRAG corpus, a Graphwise HippoRAG post, a
Karpathy-benchmark, and three first-principles build guides). The load-bearing findings:

**F-V2-1 — the design already IS "retrieval-as-reasoning" (named now).**
The arXiv LLM-Wiki paper (§2605.25480v2) coins the term the `code_*` surface was
already built for: retrieval as an agent-composed loop (search → read → follow →
re-plan → stop), not a one-shot `top-k`. Its three code-idiomatic strategies map 1:1
onto verbs this change already provides — *direct access* = `code_resolve`→`code_read`;
*bridge query* = `code_path` / `code_orient` edge-follow; *exploratory browsing* =
`code_orient(package)` + `code_search under=<pkg>`. **Folded** as spec `000` §1.1
(the retrieval contract + the two advisory termination rules: ≥1 read before asserting,
a patience bound of 3 empty searches → re-plan). No new tool; the contract is what the
existing T9b/T5 verbs already satisfy.

**F-V2-2 — the ablation is the external evidence for T9b/T5 (the biggest single term).**
The paper's ablation (Table 3): removing the compiled *structure* costs 6–7 F1,
removing *progressive (agent-driven) traversal* costs **12–14 F1**, removing the
*Error Book* costs 3–4 F1. **Traversal is the largest term** — which is exactly what
T9b (`code_orient` map) + T5 (`under` scope, `content` mode) make first-class, and what
a flat `code_search`-only surface would miss. This is the citation the blueprint's
"navigation as a first-class verb" decision was missing; it is now grounded.

**F-V2-3 — the anti-pattern defense (the design is *not* the anti-pattern).**
The decodingai first-principles guide states the trap directly: *"transforming your
codebase into a wiki is an anti-pattern… you'd re-sync the wiki on every commit, and the
code is already a graph that coding agents are trained to walk. The repo is already the
wiki."* The v2 design **splits** the two jobs and therefore avoids it:

| | code-index (this change) | knowledge-base (the other change) |
|---|---|---|
| What it is | a **machine-extracted graph** (cache) | an **LLM-maintained wiki** (reasoning surface) |
| LLM at build time? | **no** (tree-sitter, INV-7/CI-INV-1) | yes (synthesize the *why*) |
| On a code commit | re-derived **deterministically**, zero LLM tokens | the KB *re-pins* (pin-check) + regenerates the affected *why* pages |
| Cites the other? | — | yes: `code:<path>` / `symbol:` anchors **cite** the graph, never re-derive it |

The anti-pattern is *one* LLM layer both re-deriving the code **and** writing the why
into prose. The v2 split keeps the code half machine-derived (rebuildable, `sync_id`-
sync-able, the v2 differentiator) and lets only the why half be LLM-compiled. **Folded**
as spec `000` §1's one-line-model note ("It is a *graph*, not a *wiki*").

**F-V2-4 — the Error Book is the one genuine forward-look, and it belongs to the KB.**
The paper's Error Book (a *persistent, learning* self-correction store: discover →
attribute → constrain → inject → verify&close, driving two-layer code+LLM repair) is
the only mechanism here that is *not* already present. Its **code-idiomatic half**
(deterministic structural validation of the graph) is already covered by `code_status`'s
capabilities report + CI-INV-1 (rebuild → identical) + the KB's `kb_lint --fix`. Its
**learning half** (accumulating constraints across ingest batches, injecting them into
the next compile prompt) is a *document-wiki* mechanism — it repairs LLM-compiled
pages, not a machine-extracted graph. **Not folded into code-index; recorded as a
forward-look for the KB change** (the KB's `006-validate` pin-check + `007-lint`
already do the deterministic half; a `kb_error_book` for the learning half is a
candidate KB ticket, flagged here so it isn't lost).

**F-V2-5 — the corpus confirms the graph>vector claim (the NICD number stands).**
GraphRAG (community summaries — the paper shows hierarchical *summarization* loses
precise names, so v2 keeps the raw symbol graph, not summaries), HippoRAG (KG triples
+ Personalized PageRank — the `code_path`/`code_orient` traversal is the code-idiomatic
PPR), and LightRAG (dual-level entity/relation retrieval — the `code_search` FTS+
structural+semantic RRF is the code-idiomatic dual-level) all converge on the same
conclusion the NICD benchmark (already cited) quantifies: **a navigable graph the agent
traverses beats flat-chunk vector retrieval for multi-hop/relational questions.**
No new fold; this is confirmation of the existing D-CI8 + the 4-capability framing.

**F-V2-6 — "index built by a script, not the LLM" (decodingai) validates the engine.**
*"Basic math and deterministic transformations belong in scripts… you uselessly consume
tokens, and you kind of assume the LLM will count right."* This is CI-7 + D-STAMP
verbatim: the graph is built by the engine (tree-sitter), the LLM never hand-counts or
hand-writes a symbol/edge. **Already in the plan; now with a second independent
citation** alongside the `ki` article.

**Spec-zone inconsistency to correct (NOT edited by this plan — belongs to the spec
owner at execution T0, before code commits):** `006-search.md` §4 still describes the
**v1 four-index vector model** (vec0+IVF+HNSW+LSH, "the default is ONNX") — that is the
pre-D-CI8 spec. The blueprint (Task 8) + this file carry the **settled D-CI8** (3-index:
BLOB+vec0+HNSW+brute-force; **LSH+IVF dropped**; **hash-64 default**, 768-dim ONNX
opt-in). At T0, `006` §4/§4.2 must be amended to D-CI8 so the spec zone and the
blueprint agree (spec-zone changes commit before code-zone changes, AGENTS.md § Rules).
The `006` §4.3 "default is ONNX" line and the §4.2 IVF/LSH rows are the specific lines.
