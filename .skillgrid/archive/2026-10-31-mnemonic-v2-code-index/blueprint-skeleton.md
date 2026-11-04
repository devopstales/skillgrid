# Mnemonic v2 — Code-Index Function (Blueprint skeleton — staged, not final)

> **Stage:** SKELETON (plan-mode validation artifact). The full `blueprint.md` is written
> at execution time; this skeleton fixes the **task structure + Must-Haves** so the
> live-probe evidence (findings.md) is folded into the *how*, not just the *evidence*.
> The final blueprint will carry the full Plan Document Header, per-task Steps 1–5
> (failing test → impl → pass → commit), and the Threat Matrix.

**Status:** PROPOSED (skeleton)
**Tier:** T3 — new public `code_*` MCP contract + dual-dialect storage + the migration-parity
regression (the v1 bug class). Config `rules.tiers.default` is T2; escalated to T3 per
Change Classification (`high-risk`: public API + the store the KB + plugins consume).
**Build shape:** Tracer thread — the integration risk is the dialect `Store` extension + the
extraction engine + the graph walker. A thin end-to-end `code_index → SQLite graph →
code_search` path works first (Task 1's door check), then thickens (graph queries →
semantic → transport → watch).
**Classification:** high-risk (public `code_*` contract the KB + plugins consume; the shared
`Store` is extended; the migration-parity test is the v1 bug-class regression).
**Spec:** `specs/02-code-index/000`–`008` (this folder). **Findings:** `findings.md`
(same folder — the live probe + the 4 v1-mechanics research digests + the D-CI8/STAMP/
PROFILE/TIER decisions).
**Pre-execution checklist (T0):**
- [ ] `findings.md` committed (staged here; moved to the change folder at execution).
- [ ] `briefing.md` + `adr.md` written (the *what* + the in-force ADR review + the new
      ADRs: `gotreesitter` dep, `fsnotify` dep, conditional `onnxer` dep, the `Store`
      interface extension, the 4-state freshness model).
- [ ] `acceptance.feature` written from the spec truths + the threat-matrix RED tests + the
      T-CI-1…4 relevance truths.
- [ ] **Precondition gate:** the memory change (`2026-10-31-mnemonic-v2-memory`) is
      SHIPPED — its `internal/core/store` (built), `internal/intelligence` (the embedder
      seam), `internal/core/config` + `internal/core/project`, and `internal/transport/mcp`
      (the one server + profiles + write queue) all exist. Serial development (AGENTS.md §
      Rules): code-index is not `current_change` until memory ships.

## Must-Haves (goal-backward verification)

Derived from the spec zone + the **live-probe relevance truths** (findings.md, T-CI-1…4).
The verifier checks the result against this list, not just that tasks exist.

**KB-index scope (the "cover the llm-wiki indexing" decision):** the code-index change does
**not** own the KB's `kb_*` tables or the `kb_index`/`kb_reindex` rebuild — those are the
KB change's work (KB `004-index` §3.1: "the KB and memory share the file but not the
interface"). It **does** own the **shared machinery** the KB index builds on, so the KB
change can add its tables without a second foundation pass: the `Store` dialect
abstraction (the table-adding seam), the reusable FTS rendering pattern (external-content
+ triggers + ranking weights), the 4-state file-hash freshness model, and the
migration-parity regression (T-CI-3) structured so the KB can extend it over its `kb_*`
tables. This is **T21** below. The KB's `kb_query` (wiki-first retrieval) and `kb_validate`
(pin-check) are the KB change's — the code-index only guarantees the graph reads
(`code_resolve`/`code_packages`/`code_search`) they call are available.

**Truths (observable behaviors that must hold):**

- T1: `code_index` (and the `code_index` MCP tool) walks the project git root, extracts the
      6 rich languages via gotreesitter one-pass + the 25 map languages via the node-type
      data table + the regex fallback, and writes `files` + `chunks` + `symbols` + `edges` +
      `unresolved_edges` + `index_freshness` in a per-file target-state tx; a re-extract of
      an unchanged file is a no-op (idempotent upsert).
- T2: the 7 tables (`files`, `chunks`, `chunks_fts`, `symbols`, `edges`, `dep_edges`,
      `index_freshness`) are **additive migrations** on the memory `Store` — both dialects
      (SQLite + Postgres) run the full set from an empty DB (regression-tested, re-run the
      memory change's `migrate_test.go` pattern).
- T3: the extractor stamp is a **real content hash** (D-STAMP), not a const — a change to
      the extractor code/config changes the stored `extractor_stamp` → a full re-walk on the
      next freshness check.
- T4: **`code_resolve`** (NEW) returns `{exists, name, qualified_name, file, line, sync_id,
      called_from[] (file paths), ambiguous, candidates[]}` — `exists:false` → `unresolved`
      (fail closed); `ambiguous:true` → ranked `candidates[]`, never a silent pick (CI-INV-3).
- T5: **`code_packages`** (NEW) returns the top-level Go package map
      `[{path, files, symbols}]` from `files` (language='go') + `symbols` — the KB's
      coverage-gap contract; no-graph fallback is a `go list`-style scan (CI-7).
- T6: `code_get_callers/callees/dependents/implementors/hierarchy/tests_for` return
      confidence-labeled edges; name-only edges resolve **at query time** against
      `symbols.name` (1→resolved, >1→AMBIGUOUS each, 0→kept name-only).
- T7: `code_impact` (reverse-BFS, depth-cap 5, confidence filter EXTRACTED>LSP>INFERRED>
      AMBIGUOUS) returns WILL BREAK (depth 1) / LIKELY AFFECTED (deeper) + the excluded
      count (counted, not dropped); traverses only stored edges.
- T8: `code_affected` (changed/stdin/base-ref) → affected test files + the relationship
      chain; **fixes v1's gap**: Java (`*Test.java`) + other languages' test conventions
      are recognized (NOT-PORT from the live research).
- T9: `code_path` (BFS shortest path) or "where the graph stops" — a `PathResult` is always
      *something* (a path, or the refused name-only matches + the dispatch kind + the line).
- T10: `code_search` (hybrid: FTS + structural + semantic, RRF K=60) — every hit carries
      per-signal provenance; the floor is always FTS (deterministic); a down embedder →
      `degraded: no_vector`.
- T11: the semantic store (D-CI8) is **wired**: BLOB (`embeddings`+`chunk_embeddings`) +
      vec0 + HNSW + brute-force floor; the query preference order is HNSW→vec0→brute-force;
      the model-swap guard (clear vectors+indexes on `embedding_model` change) is lifted from
      v1; LSH + IVF are **dropped** (not shipped).
- T12: `code_grep` (structural by-example AST grep) — index-free, works on files not yet
      indexed; **the pattern language is fixed** (NOT-PORT the v1 UX gap): the
      node-type+capture form (`(function_declaration) \fn`) is documented + the `\f(ARGS)`
      shorthand is accepted; an invalid pattern for a language → `degraded: no_grammar:<lang>`.
- T13: `code_read` (path + line-range) returns the live source span with **secret redaction**
      (`token=`/`password=`/`secret=` masked).
- T14: the watch loop (`code_watch`/`mnemonic watch`) — fsnotify, debounce 2s (clamp
      100ms–60s), burst-collapse (a burst → one re-index), the stat-walk fingerprint
      re-check, failure isolation (a re-index failure → log + continue, never stops the
      watcher).
- T15: every `code_search`/`code_resolve`/`code_impact`/`code_explain` records a retrieval
      trail (INV-5); `mnemonic trail` inspects them.
- T16: the `code_*` surface is the **16 agent-profile + the admin set** (spec 008 §3) on the
      memory MCP server's two-profile mechanism (D-PROFILE); the v2.1-deferred tools
      (`code_pdg_query`, `code_taint`, `code_communities`, …) are **declared + report "run
      `code_index --<flag>`"** (not missing).
- T17: the `code_index` MCP tool is a **queued write** (async, returns `run_id`+`status`;
      a repeat POST for an active index → 409); the optional passes (LSP/PDG/communities/
      process) run **after** the core tx commits (separate connection, advisory).
- T18: the `code_status` read reports the **4-state freshness** (`fresh`/`lag`/`empty`/
      `unknown`) + the counts + `capabilities` JSON + the **per-language extractor tier**
      (D-TIER) + the **schema version** (NOT the v1 boolean `stale`).
- T19 (backstop): the **migration-parity regression** — a fresh store AND a store opened at
      *each prior migration version* can run every `code_*` read without a SQL-column error
      (the v1 bug class, findings.md). *Held-out test: Task 2's `TestMigrationParityAllVersions`.*
- T20 (backstop): **real-symbol-vs-fake-symbol search divergence** — `code_search` on a real
      symbol returns it in the FTS leg; a non-existent symbol does **not** return the same
      top hits. *Held-out test: Task 7's `TestSearchRealVsFakeDivergence`.*
- T21: **the shared machinery the KB index builds on is complete and reusable** (the "cover
      the llm-wiki indexing" scope — see the KB-index scope note below). Concretely:
      (a) the `Store` dialect abstraction exposes a **table-adding seam** the KB change's
      `KBStore` can use without forking the dialect code (the KB adds `kb_pages`/
      `kb_pages_fts`/`kb_claims` to the *same* `Store`, KB `004-index` §3.1 "same store,
      different query surface"); (b) the **FTS rendering pattern** (external-content FTS5 +
      the four-trigger sync set + the ranking-weights shape) is a **reusable, tested unit**
      the KB's `kb_pages_fts` re-uses verbatim (KB `004-index` §3.1: "matches the memory
      change's FTS rendering exactly — same pattern, same triggers, same ranking weights
      shape"); (c) the **4-state freshness model** (file-hash-based, `fresh`/`lag`/`empty`/
      `unknown`) is the shared staleness definition the KB's `degraded: stale_index` +
      `kb_lint` reuse (KB `004-index` §5: "the same rule `kb_lint` uses… so the two can
      never disagree"); (d) the **migration-parity regression (T-CI-3 / T19)** proves a
      prior-version store can run reads across the *code-index* tables — the KB change
      extends this same regression over its `kb_*` tables so the v1 bug class (a missing
      `ALTER` bridge) is caught for the KB too. *Held-out test: Task 2's parity test is
      structured so the KB can add its `kb_*` tables to the same loop (the test iterates a
      table-set, not a hardcoded list).*

**The four live-probe relevance truths (T-CI-1…4) — the difference between "the tool
exists" (v1, broken) and "the tool gives a good relevant answer" (v2):**

- **T-CI-1** → T20: `code_search` on a real symbol returns that symbol (FTS floor matches
      identifier queries); real ≠ fake (divergence). *Task 7.*
- **T-CI-2** → T10 + a dedicated degradation test: a leg that errors degrades to
      `degraded: <leg>` + drops out of the RRF fusion; the broken leg's empty result never
      sets the answer. *Task 7's `TestBrokenLegDegradesOutOfFusion` (the v1 live probe showed
      the warning but the broken leg still shaped the output — this is the fix).*
- **T-CI-3** → T19: every `code_*` read runs on a fresh store AND on a store opened at each
      prior migration version; a version mismatch is a *version* error, never
      `no such column`. *Task 2's `TestMigrationParityAllVersions` (the v1 live probe: a
      1570-symbol, 060-migration, up-to-date index where every graph read failed
      `no such column: s.uid` because no `ALTER TABLE symbols ADD COLUMN uid` bridge
      existed).*
- **T-CI-4** → T18: `code_status` reports schema version + per-language extractor tier +
      fair coverage (not just counts + boolean stale). *Task 8's `TestCodeStatusReportsVersionTierCoverage`.*

**Artifacts** (files that must exist with real implementation, not stubs):

- `mnemonic-v2/internal/core/store/migrations/100_code_index.sql` (SQLite) + `migrations_pg/
  100_code_index.sql` — the 7 tables (numbering starts at 100 to avoid colliding with the
  memory change's 001–060 range; the exact number is set at execution to be the next free).
- `mnemonic-v2/internal/core/store/store.go` — the `Store` interface **extended** with the
  code-index methods (the `code_index` write path + the graph reads); the extension is
  additive (D-PROFILE's `Store` note). Includes the **table-adding seam** (T21) — the
  dialect-neutral mechanism the KB change's `KBStore` uses to add its `kb_*` tables to the
  same `Store` without forking the dialect code.
- `mnemonic-v2/internal/core/store/fts.go` — the **reusable FTS rendering unit** (T21):
  the external-content FTS5 DDL + the four-trigger sync set + the ranking-weights shape,
  extracted from `sqlite.go` into its own tested unit so the KB change's `kb_pages_fts`
  re-uses it verbatim (KB `004-index` §3.1: "same pattern, same triggers, same ranking
  weights shape").
- `mnemonic-v2/internal/codeindex/extract/` — the tree-sitter ladder (rich-6 + 25 maps +
  regex), the self-healing pool + circuit breaker, the extractor stamp (D-STAMP), the
  chunker.
- `mnemonic-v2/internal/codeindex/graph/` — the walker (impact, neighbors, path, resolve,
  packages, affected) — dialect-agnostic.
- `mnemonic-v2/internal/codeindex/search/` — the hybrid fusion (RRF K=60), the structural
  leg (by-example AST grep), the semantic leg (the 3-index store + the model-swap guard).
- `mnemonic-v2/internal/codeindex/watch/` — the fsnotify loop (debounce, burst-collapse,
  fingerprint re-check).
- `mnemonic-v2/internal/codeindex/vectorstore/` — the BLOB + vec0 + HNSW + brute-force
  engine (D-CI8), the dimension gating + model-swap guard.
- `mnemonic-v2/internal/transport/mcp/tools_code.go` — the 16 agent + admin `code_*` tools
  as thin adapters over the shared handler set.
- `mnemonic-v2/cmd/mnemonic/index.go` — the `mnemonic index/search/status/orient/explain/
  path/impact/affected/grep/read/watch` CLI group.
- `mnemonic-v2/internal/codeindex/*_test.go` — tests for every truth above, **including the
  four T-CI held-out tests** (T-CI-1…4).

**Key links:**

- K1: `code_index` MCP tool → handler → `codeindex.Index` (scan → per-file extract tx →
      orphan prune → update `index_freshness`) → the optional passes (separate connection) →
      queued-write `run_id`.
- K2: `code_resolve` → `graph.Resolve` (lifts v1 `resolve.go:35-76`) → `called_from` = the
      forward callers resolved to `files.path` → the KB's `symbol:` pin-check reads this.
- K3: `code_search` → FTS leg + structural leg + semantic leg → RRF fusion (K=60) → per-hit
      provenance → the trail writer. A broken leg degrades out (T-CI-2).
- K4: the KB (a later change) calls `code_resolve` + `code_packages` + `code_search` through
      the `code_*` surface — the two NEW tools are the KB's forward-refs resolving (spec
      000 §5). The KB does **not** import the `codeindex` package (CI-7 — graph is
      first-class but optional; KB degrades to ripgrep-only).
- K6: the KB's `kb_index`/`kb_reindex` (the KB change) build its `kb_pages`/`kb_pages_fts`/
      `kb_claims` tables **on top of the shared machinery this change provides** (T21): the
      `Store` table-adding seam, the reusable FTS rendering unit (`store/fts.go`), the
      4-state freshness model, and the migration-parity regression (T-CI-3). The KB adds its
      tables to the *same* `Store` + *same* migration set + *same* parity loop — it does not
      fork the dialect code or the FTS pattern. This is the "cover the llm-wiki indexing"
      scope: the code-index change makes the KB's index buildable without a second
      foundation pass.
- K5: `cmd/mnemonic`'s `index`/`search`/… and the MCP `code_index`/`code_search`/… produce
      byte-identical JSON (same handler, same result marshal) — a parity test asserts it.

**One-way-door decisions** (hard to reverse — flag for explicit user approval at execution):

- D1: **the `code_*` MCP tool contract** — the KB + the opencode/pi plugins consume it as
      their only interface. Renaming a tool / adding a required param / changing a return
      shape after the KB builds breaks it. Task 9 (transport).
- D2: **the `Store` interface extension** (code-index methods) — code-index and the KB
      later extend the same store (they add their tables to the same project DB, spec
      `002-schema` §4). Adding a method later is additive; reshaping one is a contract break.
      Task 1 (door check).
- D3: **the migration numbering** (the 7 tables are additive on the memory store) — a number
      collision with the memory change's `migrations_pg/` or the KB's future tables is a
      rebase. Task 2.
- D4: **the extractor stamp algorithm** (D-STAMP) — changing it later changes the stored
      value once (a one-time full re-walk), never a schema migration. Task 3.

## Global Constraints

- **Go 1.22+ to build** (ADR-0031); go.work floor 1.26.0 → `mnemonic-v2/go.mod` declares
  `go 1.26.0` (already set).
- **No new dependencies without an ADR** (AGENTS.md § Rules): the allowed set is the memory
  change's (`modernc.org/sqlite` incl. `/vec`, `mark3labs/mcp-go`, `gopkg.in/yaml.v3`,
  `lib/pq`) **plus** `gotreesitter` (ADR required), `fsnotify` (ADR required), and
  conditionally `onnxer` (ADR required, only if the 768-dim ONNX embedder is wired — the
  hash-64 default does not need it). All three are already in `mnemonic_back/go.mod`
  (proven cgo-free) but are new to `mnemonic-v2/go.mod`.
- **cgo-free** (v1 VERIFIED: `skillgrid doctor` reports `cgo: free`).
- **Conventional commits only; no AI-attribution trailers.**
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard).
- **One change at a time** (serial development) — code-index does not execute until the
  memory change ships (the T0 precondition gate).
- **Additive migrations only** (spec `002-schema` §1).
- **All seven invariants** (INV-1…7) + the four code-index invariants (CI-INV-1…4) bind
  every task.
- **Test:** `cd mnemonic-v2 && go test ./internal/codeindex/... ./internal/core/store/...`;
  **build:** `cd mnemonic-v2 && go build ./...`.

## Task structure (tracer-thread, dependency-ordered)

> The final blueprint expands each task to Steps 1–5 (failing test → impl → pass → commit)
> with the exact file list, interface signatures, and SATISFIES lines. The T-CI-1…4 truths
> are **folded into the tasks that produce them** (the column "Folds T-CI").

| Task | Scope | Door / gate | Folds T-CI |
|---|---|---|---|
| **1** | `Store` interface extension (code-index methods) + the 7-table additive migration (SQLite) + `sync_id` minting for files/symbols | **Door check** — if a `code_index` cannot write `files`+`symbols`+`edges` through the extended `Store` and a `code_search` FTS floor cannot find a saved symbol, stop | D2, D3 |
| **2** | Migration suite + **migration-parity regression** (fresh + every-prior-version store runs all `code_*` reads) + **reusable FTS rendering unit** (the external-content + 4-trigger + ranking-weights pattern, extracted so the KB's `kb_pages_fts` re-uses it) + the **table-adding seam** on the `Store` the KB's `KBStore` uses | re-runs the memory change's `migrate_test.go` pattern over the code-index tables; the parity test iterates a *table-set* (not a hardcoded list) so the KB adds its `kb_*` tables to the same loop | **T-CI-3** (T19) + **T21** |
| **3** | The extraction ladder (rich-6 gotreesitter + 25 maps + regex) + self-healing pool + circuit breaker + **the real extractor stamp** (D-STAMP) + the chunker | the stamp must be a content hash (a const never fires the full re-walk) | D4 (D-STAMP) |
| **4** | The graph walker: impact + neighbor views + path/graph-stops + `code_affected` (with the Java test-convention fix) + **`code_resolve`** (NEW, lifts v1 `resolve.go`) + **`code_packages`** (NEW) + the `extractor_tiers` column (D-TIER) | the walker selects only its migration's columns (T-CI-3's read-side proof) | T4, T5, T6, T7, T8, T9 |
| **5** | The hybrid search: FTS leg + structural leg (by-example AST grep, pattern language fixed) + semantic leg (D-CI8 3-index store) + RRF K=60 + `code_read` (secret redaction) | the floor is always FTS; a broken leg degrades out | **T-CI-1** (T20), **T-CI-2** |
| **6** | The watch loop (fsnotify, debounce, burst-collapse, fingerprint re-check, failure isolation) | the watch is a trigger, not a gate (a query during re-index never blocks) | T14 |
| **7** | The `code_*` transport: the 16 agent + admin tools on the memory MCP server (D-PROFILE) + the `code_index` queued write + the `code_status` 4-state + tier + version report + the CLI group | the v2.1-deferred tools are declared + report "run `--flag`" | **T-CI-4** (T18), T10, T16, T17, T18 |
| **8** | The advanced/deferred passes: the `--lsp` tier (the one optional pass that ships) + the v2.1 flag surface (`--pdg`/`--communities`/`--process` declared) | the LSP tier is advisory (all-or-nothing per call site) | T17 |

**Wave order (serial, stacked-to-main):** 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8. Tasks 1–3 only
need the *built* memory foundation (Store + syncid); Tasks 4–8 additionally gate on the
memory change's intelligence seam (5) + transport/mcp (7) + cmd/mnemonic (7). Under serial
development the whole change starts only after memory ships (T0 gate).
