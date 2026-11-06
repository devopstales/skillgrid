# 005 — Advanced Passes (the optional / deferred passes)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `002-symbols-edges` (the graph they enrich), `003-extraction` (the core
tx they run after), memory `00-foundation/003-intelligence` (the LLM seam — the process
labeler).
**Consumed by:** `008-transport` (the admin tools gated on these flags), `000-overview`
(CI-5).

---

## 1. Capability

The **default v2 graph is thin** (CI-5): `files` + `symbols` + `edges` + `chunks` +
`dep_edges` + `index_freshness`. The heavier analyses — PDG, taint, communities,
process flows, and the LSP tier — are **optional passes** behind flags. This spec
defines the flag surface, the keep-or-defer rationale (CI-5), and — for the ones that
ship in v2 (the LSP tier) — the behavior. The v2.1-deferred ones (PDG/taint/
communities/process) are **specified here as their flag + their output contract**, so
that v2.1 is an *addition* (a new table + a new pass), not a redesign.

## 2. The flag surface

| Flag | Default | Pass | v2 status |
|---|---|---|---|
| `--lsp` | **off** | the LSP tier (member-call resolution → `LSP_RESOLVED` edges) | **v2** (`003` §4) |
| `--pdg` | off | PDG (per-function CFG → control/data dependence) + taint (source→sink) | **v2.1** |
| `--communities` | off | Leiden partition + god nodes + labels | **v2.1** |
| `--process` | off | precomputed execution-flow tracing | **v2.1** |

**All passes are advisory** (the v1 semantics, kept): they run **after** the core graph
tx commits, in a **separate connection**, and **never roll back the committed graph**.
A failing pass → warn + continue. A pass that is off is *simply not run* (its output
tables are absent/empty; the corresponding admin tool reports "run `code_index
--<flag>`").

**The `capabilities` JSON** (`index_freshness`, `001` §4) records which passes ran
(`{graph:true, fts:true, vector:bool, lsp:bool, pdg:bool, communities:bool, process:
bool}`). `code_status` reports it; the KB reads it to know what the graph can answer
(e.g. "LSP is off, so the `symbol:` call-site check is best-effort").

## 3. The LSP tier (v2 — the one optional pass that ships)

The full behavior is in `003-extraction` §4. Restated for the flag surface:
- **Flag:** `--lsp` (default off).
- **What it adds:** member-call resolution (`receiver.Callee`) that the static pass
  could not bind → `edges` rows with `confidence = LSP_RESOLVED` (the 4th tier, `002`
  §6).
- **Servers:** gopls / pyright-langserver / typescript-language-server / rust-analyzer /
  clangd (on PATH). Per-call 30s timeout. Best-effort, all-or-nothing per call site.
- **Effect on the KB:** a `symbol:` call-site check is tighter (CI-6, CI-7). Without
  `--lsp`, the check is static-resolution-only (`degraded: no_lsp` if it needs a
  member-call binding).

## 4. PDG + taint (v2.1, `--pdg`) — the output contract

**Deferred to v2.1** (CI-5). The *output contract* is fixed here so v2.1 is an addition:

**PDG** (per function/method): an intraprocedural **CFG** (basic blocks —
`normal|branch|loop|call|return` — deterministic first-seen numbering, `//go:nocfg`
skips a function) → **control dependence** (a child of a branch/loop header depends on
that header, EXTRACTED) + **data dependence** (a prior call's result flows into a later
call's arguments, INFERRED / AMBIGUOUS at an unresolved boundary). **Taint** (immediately
after PDG): source→sink reachability *over the PDG data edges*; sources/sinks are
deterministic name-substring classes (request param/env/file-read → SQL exec/shell
exec/template render/file write, configurable); a hop into an unresolved call is a
**boundary** (the branch truncates with an AMBIGUOUS hop + a "stops at <boundary>" note);
a finding's `PathLabel` is the **worst hop** (EXTRACTED only if every hop is).

**Output tables (v2.1 additions):**
```sql
-- per-function intraprocedural CFG
CREATE TABLE cfg_blocks (symbol_id INTEGER NOT NULL, block_no INTEGER NOT NULL,
    start_line INTEGER NOT NULL, end_line INTEGER NOT NULL,
    kind TEXT NOT NULL, PRIMARY KEY(symbol_id, block_no));
CREATE TABLE cfg_edges (symbol_id INTEGER NOT NULL, from_block INTEGER NOT NULL,
    to_block INTEGER NOT NULL, condition TEXT, PRIMARY KEY(symbol_id, from_block, to_block));
-- control/data dependence
CREATE TABLE pdg_edges (symbol_id INTEGER NOT NULL,
    kind TEXT NOT NULL,            -- 'control' | 'data'
    from_line INTEGER, to_line INTEGER, from_name TEXT, to_name TEXT,
    confidence TEXT NOT NULL, note TEXT);
-- taint findings
CREATE TABLE taint_findings (symbol_id INTEGER NOT NULL, source_line INTEGER,
    sink_line INTEGER, source_name TEXT, sink_name TEXT,
    confidence TEXT NOT NULL, note TEXT);
```

**Determinism requirements (the v1 invariants, kept for v2.1):** sorted iteration, stable
block numbering, content-key dedupe, truncation caps (2000 blocks / 4000 steps —
truncate-not-abort). The data-flow approximation (call-result flow, no SSA) is a
documented limit.

**v2 behavior:** with `--pdg` off (the default), `code_pdg_query` / `code_taint`
report "run `code_index --pdg`". The tables are absent (not created) until the pass runs.

## 5. Communities (v2.1, `--communities`) — the output contract

**Deferred to v2.1** (CI-5). Leiden partition over the undirected `symbols`/`edges`
graph; **god nodes** ranked by degree; **labels** derived from top god-node names +
file paths (**never an LLM call**); **cohesion = conductance** — the fraction of a
community's link mass that *crosses* the boundary (the `ki` `themes` pattern: ≤ 0.35
reads "tightly interlinked", ≥ 0.60 "loosely", otherwise "moderately" — a hedge against
over-reading a weak cluster). **Determinism is pinned** (the `ki` `randomSeed +
concurrency: 1` contract): a fixed random seed + single-threaded execution, so a
re-run on an unchanged graph returns the *same* communities (the answer moves only
when the corpus moves, never because the model "felt different this session"). Cached
by a content-hash key (an unchanged re-index is a cache hit — the seed is part of the
key).

> **Metric note:** edge *density* (internal_edges / C(n,2)) is a valid *separate*
> signal and may also be stored, but the **cohesion *word*** ("tightly/loosely
> interlinked") the agent reads is conductance — it measures how self-contained a
> cluster is, which is the question "is this a real module or an accident of the graph?"
> asks. (Source: Neo4j `ki` `themes` command, the GDS `conductance` pipeline.)

**Output tables (v2.1 additions):**
```sql
CREATE TABLE communities (id INTEGER PRIMARY KEY, symbol_id INTEGER UNIQUE,
    FOREIGN KEY(symbol_id) REFERENCES symbols(id) ON DELETE CASCADE);
CREATE TABLE community_meta (id INTEGER PRIMARY KEY, label TEXT,
    symbol_count INTEGER, god_nodes TEXT, cache_key TEXT,
    conductance REAL,     -- the cohesion word (≤0.35 tight / ≥0.60 loose)
    density REAL,         -- optional separate signal (internal_edges / C(n,2))
    leiden_seed INTEGER); -- the pinned determinism seed (part of cache_key)
```

**v2 behavior:** with `--communities` off (default), `code_communities` /
`code_god_nodes` / `code_explain_community` report "run `code_index --communities`".
**Import cycles** are *not* a stored table even in v2.1 — they are re-derived from
`kind='imports'` edges on demand by `code_circles` (admin, `008-transport`).

## 6. Process flows (v2.1, `--process`) — the output contract

**Deferred to v2.1** (CI-5). Depth-capped BFS from entry points (route→handler refs +
package `main` functions) through `calls` edges; **stops at dispatch kinds**
(`interface`/`bus`/`callback`) with a named "stops at …" note; a **cross-community**
flag (from the v2.1 communities); an LLM labeler (pluggable, falls back to a
deterministic stub label `flow: <entry>` when no LLM — INV-3). Cached by a process
content-hash.

**Output tables (v2.1 additions):**
```sql
CREATE TABLE processes (name TEXT PRIMARY KEY, entry_symbol_id INTEGER,
    entry_kind TEXT, cross_community INTEGER, content_hash TEXT,
    label TEXT, label_status TEXT, stop_note TEXT);
CREATE TABLE process_steps (process_id TEXT NOT NULL, step INTEGER NOT NULL,
    symbol_id INTEGER NOT NULL, confidence TEXT, kind TEXT);
```

**v2 behavior:** with `--process` off (default), `code_processes` / `code_process`
report "run `code_index --process`".

## 7. The keep-or-defer rationale (CI-5)

**Why the default graph is thin:** the KB's two hard contracts (`code_resolve`,
`code_packages`) need only the **core graph** (`files` + `symbols` + `edges`). PDG/
taint/communities/process are *deeper analyses* that no v2 consumer (the KB, the
plugins) requires for the default workflow. Shipping them by default would:
- add the v1 single-connection close/reopen complexity (the advisory passes),
- add 6+ tables + 4 passes + their cache keys to the default path,
- slow the incremental index (the per-function CFG re-parse, the Leiden partition).

Deferring them to v2.1 (behind flags) keeps the **default `code_index` fast + simple**
(the core tx) while *fixing the output contract* so v2.1 is an addition. The **LSP tier
ships in v2** because it *tightens the KB's `symbol:` call-site check* (a direct benefit
to a v2 consumer) and it is a thin, advisory, all-or-nothing pass.

## 8. Invariants

INV-3 (the LLM process labeler degrades to a stub label; the LSP tier degrades to static
extraction — both explicit `degraded`), INV-2 (the v2.1 tables are dual-mode when they
ship), CI-INV-1 (all pass outputs are derived — rebuildable by re-running the pass),
CI-INV-2 (current-state — the passes have no temporal window).

## 9. Open items

- **The v2.1 scope is fixed by this spec's output contracts** — v2.1 implements the
  tables + passes as specified here. No open items in v2; the v2.1 work is a *new change
  folder* (a future `2026-…-mnemonic-v2-code-index-advanced`), not a v2 open item.
- **The LSP server set:** the v1 five (gopls/pyright/tsserver/rust-analyzer/clangd).
  Adding a language's LSP server in v2.1 is a config addition, not a spec change.
