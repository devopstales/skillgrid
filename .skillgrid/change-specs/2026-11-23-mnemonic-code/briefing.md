# `mnemonic code` tool — code knowledge graph + hybrid search

> **STATUS:** `draft` (2026-11-23)
> **Author:** devopstales
> **Intent frozen:** no (2026-11-23)

## Problem / Intent

When **a developer or agent is exploring a codebase from CLI or MCP**, they
want **code search, context, impact, and path queries answered over the same
per-project brain that holds their memory**, so that **answers are
graph-backed and hybrid (lexical + semantic) without a second index to
maintain, and code answers carry the same degradation/coverage contract as
memory answers**.

Why now: `2026-11-23-mnemonic-core` (blueprint phase) delivers the store,
embedder, and transport seam the `code` tool sits on, and its phase-4 spikes
(2026-11-23) have verified the CGO-free stack end-to-end. The predecessor
codebase (`/data/git/AI/skillgrid/mnemonic/internal/codeindex/`) exists as a
design source — but its LSP pass is a stub in production, its 38-tool surface
is bloat, and it predates the brain's store contract.

## Purpose & Success Criteria

- **Purpose:** add the `code` tool to the `mnemonic` binary — a tree-sitter
  knowledge graph of the project's code files (files → AST chunks → symbols →
  edges with confidence), searched as RRF-fused FTS-trigram + semantic
  hybrid, queried structurally (context / impact / path / explain) — inside
  the same per-project SQLite brain, with one K6 handler serving CLI and MCP
  byte-identical envelopes.

- **Success criteria** (verifiable):
  1. `code_index` on a ~40k-item repo leaves the structural graph (parse +
     edges + FTS) queryable in a synchronous phase bounded in minutes;
     embedding backfill runs in the background with progress, ETA, and
     resumability.
  2. `code_search` returns RRF (K=60) fusion of FTS-trigram and semantic
     legs; every hit carries an addressable id; the floor leg never injects
     unscored hits into fusion.
  3. `code_impact` and `code_path` account for refused/ambiguous hops
     (counted, not silent); path confidence = weakest hop
     (EXTRACTED < INFERRED < LSP_RESOLVED).
  4. `code_status` reports computed embedding coverage (never a dead
     constant), freshness as quantified working-tree drift since the last
     index run, persisted per-language extraction tiers + circuit-breaker
     death counts, and a top-N unresolved-edge ledger.
  5. CLI and MCP envelopes for all 7 tools are byte-identical (parity test).
  6. UID/chunk-boundary/extractor-stamp parity fixtures are byte-identical
     between the predecessor and the new extractor (transfer-hygiene gate).

- **Out of scope** (deferred, named): rename analysis, taint flow, PDG,
  communities, generated docs; docs/config files in the index (code files
  only); cross-domain embedding fusion (one code query mixing memory + code);
  bundling or managing LSP server binaries (detection only); multi-project
  watch; ANN index rebuild automation beyond a mismatch note; the
  **web-cache domain and its TTL bars** (v1: context7 720h, exa 168h,
  deepwiki 336h, fetch 168h — owned by the memory/core store contract, not
  this change).

## Context

- **Product context:** the `mnemonic` binary is a multi-tool agent memory
  suite: `mnemonic <tool> <verb>` with per-tool MCP servers over one handler
  (ADR-0010, ADR-0007/K6 parity). The `code` tool is the second tool domain
  after `memory`.
- **Change context:** `2026-11-23-mnemonic-core` is at blueprint phase and
  owns: per-project SQLite file (ADR-0003), vectors in file (ADR-0004),
  CGO-free target build (ADR-0005), local ONNX embedder default (ADR-0006),
  single-handler two adapters (ADR-0007), WAL-only concurrency (ADR-0009),
  multi-tool binary (ADR-0010), degradation contract (ADR-0011), and
  ADR-0015 (CGO-free stack selection, refined by its phase-4 spikes). This
  change **queues behind core v0.1** and pins core's tag/commit at the
  blueprint gate (core's v0.1 schema freeze is the stable base).
- **Inherited verified stack** (from `2026-11-23-mnemonic-core/findings.md`,
  spikes 2026-11-23, both PASS): `hugot v0.8.6` GoMLX backend with
  all-MiniLM-L6-v2 (384-dim, p50 81 ms/sentence non-simd, ORT escape
  13.4 ms), `ncruces/go-sqlite3 v0.38.0` with `ext/vec1` + `ext/fts5`
  registered per-connection, go ≥ 1.27, brute-force BLOB floor retained as
  the KNN equivalence oracle (vec1 IVFADC is approximate on small tables).
  The asg017 sqlite-vec vec0 binding is **not** adopted (unpairable,
  live-tested).
- **Predecessor design source** (copy-of-pattern, not import — different
  repo, no shared module): `internal/codeindex/` (`index.go`,
  `extract/{ladder,chunker,lsp,pool}.go`, `search/hybrid.go`, `watch/watch.go`),
  `migrations/100_code_index.sql`, `transport/mcp/contract.go` (38 old
  `code_*` tools — reference only; the 7 new annotations are written fresh).
- **Global docs context:** `ASSUMPTIONS.md` carries LOCKED rows for
  ADR-0013 (pure-Go treesitter) and ADR-0014 (unified embedding space);
  `glossary/02-technical-terms.md` has the code-tool term cluster (both
  landed 2026-11-23).

## User Context

- **Developer and AI-agent users** of the skillgrid toolchain, on Linux
  (x64/arm64) in repos of 10²–10⁵ items.
- **Current experience:** the predecessor codeindex shipped with a LSP pass
  that produced **zero** LSP_RESOLVED edges in production (the resolver was a
  stub), a 38-tool surface no one annotated, and no code coverage in any
  agent's brain. No support tickets exist — internal/technical change; the
  seven-lens Goldfish shootdown (2026-11-23) is the standing-in for direct
  user evidence.

## Pre-mortem

- **Desirability:** interactive developers already have LSP + ripgrep in
  their editors; the brain's code graph is redundant for that loop. The
  value lives in agent/CLI access and one index shared with memory — if
  agent workflows don't call the 7 tools, they are dead weight. The query
  core (index/search/context/status) is the floor; impact/path/explain are
  where the graph earns its keep.
- **Feasibility:** pure-Go embedding latency (spike: 81 ms/sentence
  non-simd batch-1 → ~54 min for 40k items, less with `GOEXPERIMENT=simd`,
  13.4 ms with the ORT escape tag) is a real but bounded cost — mitigated by
  batch-32 + two-phase indexing. The LSP JSON-RPC client is the least-proven
  component (kept by user re-confirmation *against* the prod-stub evidence);
  it degrades to static-only + ledger, never blocks. gotreesitter grammar
  coverage gaps are bounded by the 3-tier ladder + circuit breaker. vec1
  IVFADC approximation on small tables is handled by the brute-force oracle
  bench.
- **Viability:** build cost = 7 tool contracts + LSP client + watcher + tier
  ladder, on top of a core that is not shipped yet. The code tool inherits
  every core slip; if core v0.1 lands later than 2 weeks after its blueprint,
  the timebox shrink fires.

## Timebox

Target: core v0.1 + 3 sprints (measured from core v0.1 landing).
**If exceeded, shrink to the query core:** `code_index` (structural) +
`code_search` + `code_context` + `code_status`. Deferred to a follow-up
change: `code_impact`, `code_path`, `code_explain`, the LSP pass, and watch.
(User re-confirmed LSP + watch as v1 scope on 2026-11-23; the shrink is the
forcing function, not the current scope.)

## Approaches Considered

| Approach | Key trade-off | Verdict |
|---|---|---|
| **A. GitNexus-core 7 tools** | right-sized surface; rename/taint/PDG/communities deferred | **chosen** (user, Q1) |
| B. Full GitNexus surface | +4–6 tools, each needing new graph machinery (PDG, taint analysis) | rejected — scope; deferred list named in Out of scope |
| C. Predecessor 38-tool port | maximum transfer, but 38 contracts to write/annotate/maintain | rejected (author) — surface bloat; remap table instead |
| D. CGo tree-sitter build | faster parse, but breaks the CGO-free target build (ADR-0005) | rejected — ADR-0013 (user override of the binding) |
| E. Per-domain embedding spaces | cleaner domains, but two models + cross-domain fusion machinery in v1 | rejected — ADR-0014 (user override) |

**Chosen:** A — 7 tools (`code_search`, `code_context`, `code_impact`,
`code_path`, `code_explain`, `code_index`, `code_status`), pure-Go
gotreesitter AST (ADR-0013), one unified embedding space per project (ADR-
0014), code files only, predecessor-lite machinery: core indexing + LSP
pass + watch — both re-confirmed by the user on 2026-11-23 *after* seeing
the Goldfish evidence for each.

**Rejected by author:** C — the predecessor's 38-tool surface (including
the split `code_get_callers`/`code_get_callees`/`code_get_dependents` that
the new `code_context` consolidates) is bloat; its map survives only as a
reference for an old→new remap table. D — CGo breaks ADR-0005's target
build for all users, not just cross-compile ones.

## Requirements

**R1 — Two-phase code index.**
Current: predecessor `index.go` runs one synchronous pass; at 40k items with
pure-Go embeddings (spike-verified 81 ms/sentence non-simd) first index is
tens of minutes to hours with no progress and no queryable state until done.
Target: `code_index` runs a **synchronous structural phase** (walk → extract
→ files/chunks/symbols/edges → FTS → prune) leaving the graph + lexical
search queryable immediately, then a **background embedding backfill**
(batch ~32, progress/ETA, resumable; rows missing a vector for the active
model are dirty regardless of content hash).
Acceptance: on a ≥10k-item fixture repo, `code_search` (FTS leg) and
`code_status` are functional before backfill completes; interrupting
backfill and re-running resumes without re-embedding covered rows;
`code_status` shows embedded/total coverage during backfill.
Acceptance scenario: `code_index` first-run two-phase (acceptance.feature,
scenario "First index is queryable before embeddings finish").

**R2 — Hybrid search (RRF + floor rules).**
Current: predecessor `search/hybrid.go` uses RRF K=60, but the hash-64
floor leg scores every symbol 1/61 — the FTS#1 position with no signal.
Target: `code_search` fuses FTS-trigram (FTS5) + semantic (vec1 KNN /
brute-force floor) with RRF K=60; **when the floor leg is active it is
excluded from fusion** (or flagged `floor_leg` and never ranked first); each
hit carries an addressable id (selector from R3); the trail records per-leg
ranks and the leg tier.
Acceptance: a semantic-only query (no trigram match) returns semantic
results; a floor-active run returns no hit whose only leg is the floor;
trail JSON contains per-leg ranks; sub-3-char identifier lookups are served
by the structural leg (documented, since trigram FTS is blind below 3
chars).
Acceptance scenario: `code_search` RRF fusion with floor exclusion
(acceptance.feature, scenario "Hybrid search excludes the floor leg").

**R3 — Node selector + `code_context`.**
Current: predecessor returns bare names/ids that callers cannot address;
ambiguous names re-guess `rows[0]`.
Target: a node selector `{name?, path?, line?}` resolves to exactly one
symbol or to `{ambiguous: true, candidates: []}` — never a silent first
row. `code_context` takes a resolved selector and returns the symbol's
definition, chunk (80/10-line windows), callers/callees/dependents
(consolidating the predecessor's three split tools), and immediate graph
neighbors with edge confidence.
Acceptance: selecting an overloaded name without path returns
`ambiguous:true` with the full candidate list; selecting with path returns
the exact symbol; `code_context` output for a known fixture symbol matches
the fixture's expected neighbors.
Acceptance scenario: selector ambiguity surfaces candidates (acceptance.
feature, scenario "Ambiguous selector returns candidates, not a guess").

**R4 — `code_impact` with hop accounting.**
Current: impact walks would silently stop at edges with `to_id NULL` or
ambiguous targets — a silent lower bound.
Target: `code_impact` transits the dependency graph from a selector,
counting **refused hops** (unresolved) and **ambiguous hops** (multiple
targets, each listed) and reporting the result as a lower bound with those
counts, mirroring `code_path`'s graph-stops semantics.
Acceptance: on a fixture with N unresolved edges in the impact cone, the
envelope reports exactly N refused hops and the reachable set is a
documented lower bound.
Acceptance scenario: impact reports refused hops (acceptance.feature,
scenario "Impact counts refused hops as a lower bound").

**R5 — `code_path` with weakest-hop confidence.**
Current: paths carry no confidence; edge kinds are unranked.
Target: `code_path` returns the shortest edge path between two selectors;
each hop carries its confidence (EXTRACTED < INFERRED < LSP_RESOLVED — the
boundary sentence is stated in the tool's envelope and docs: LSP_RESOLVED
is the strongest); the **path's confidence is the weakest hop**;
graph-stops (refused/ambiguous) terminate the path with a reason.
Acceptance: a fixture path containing one INFERRED hop reports path
confidence INFERRED; a path broken by an unresolved edge reports the stop
reason and the partial path.
Acceptance scenario: path confidence is the weakest hop (acceptance.feature,
scenario "Path confidence degrades to weakest hop").

**R6 — `code_status` (coverage, freshness, tiers, ledger).**
Current: `code_status` coverage is a dead 0.0; freshness is undefined;
degradation is invisible.
Target: `code_status` reports: embedded/total coverage (computed, with the
active model stamped); freshness = quantified working-tree drift since the
last index run (files added/changed/removed by content hash — **not** git
HEAD, which is meaningless in worktrees/dirty trees); per-language runtime
tier + circuit-breaker death counts from the persisted run record; a top-N
unresolved-edge ledger (readable without a full index); last-run
status/error/duration on the failure path; oversized files **named**, not
just counted; watch alive-state + last outcome.
Acceptance: after deleting a fraction of vectors, coverage reflects the
loss; after a same-size edit, drift is reported (hash, not mtime/size); a
killed language's breaker state appears in the run record.
Acceptance scenario: status reports real coverage and drift
(acceptance.feature, scenario "Status is computed, not constant").

**R7 — Model-swap coverage backfill (blocker B1, resolved).**
Current: predecessor `GuardModelSwap` detects the swap and
`ClearVectors` wipes the corpus — the index silently loses its semantic
leg.
Target: two-phase swap — the new model stamp commits to memory `meta`
**only after** embedding coverage completes; rows missing a vector for the
active model are dirty regardless of content hash; backfill is batched
(~32) and resumable; `index_freshness`/envelope carry embedded/total
coverage so a mid-swap index is visible, not silent. One model-stamp
location: the code-side `embed_meta` is dropped; memory `meta` is the only
stamp (a hash-64 → ONNX upgrade is one coordinated cross-domain re-embed).
Acceptance: swapping the embedder mid-corpus leaves the old vectors
queryable until coverage completes; the envelope shows partial coverage
during the swap; after completion the stamp and coverage both reflect the
new model.
Acceptance scenario: model swap backfills before committing the stamp
(acceptance.feature, scenario "Model swap is two-phase").

**R8 — Degradation carrier (blocker B3, resolved).**
Current: tier choice is runtime-only and invisible; a grammar crash
quarantines a language silently; UIDs include line spans so a tier change
rewrites ids and cascade-deletes edges without count.
Target: per-language runtime tier + death counts are **persisted per run**
(`extractor_tiers`); every envelope for a language below rich carries a
tier note; UIDs are derived **without** line spans (tier-tolerant prune);
cascade-deleted edge counts are logged per run; the degradation contract
(ADR-0011) holds — never silent.
Acceptance: forcing a grammar crash for one language makes it visible in
`code_status` (tier + deaths) and in every subsequent envelope for that
language; switching a language's tier does not rewrite stable uids.
Acceptance scenario: degradation is carried in every envelope
(acceptance.feature, scenario "Below-rich tier is visible").

**R9 — LSP pass (user re-confirmed keep, 2026-11-23, against prod-stub
evidence).**
Current: predecessor `extract/lsp.go` `resolveWithServer` is a stub in
production; the 30 s budget is global, not per call-site; resolved edges are
non-sticky (watch bursts demote to name-only); ambiguity re-guesses
`rows[0]`; diagnostics are dropped.
Target: a real LSP JSON-RPC client (stdio) against user-provided LSP
binaries (detected, not bundled); per-call-site budget; resolved edges are
sticky and location-based (survive re-index bursts); ambiguous resolutions
go to a persisted ambiguity ledger instead of a guess; server diagnostics
are captured (not dropped); the per-language run record persists LSP
participation (server present/absent/version, edges resolved).
Acceptance: with a fixture LSP server, `LSP_RESOLVED` edges appear and
survive a watch burst; removing the server degrades to static-only with the
ledger intact and an envelope note.
Acceptance scenario: LSP edges are sticky and ledgered (acceptance.feature,
scenario "LSP pass resolves and degrades").

**R10 — Watch (user re-confirmed keep with fixes, 2026-11-23).**
Current: predecessor `watch.go` blocks the loop on re-index; death is
invisible and silent; re-check is size-only (same-size edits missed);
re-indexes don't re-embed changed files; no owner for MCP-side freshness.
Target: fsnotify loop with a **worker goroutine + coalescing queue** (the
event loop never blocks on re-index); **hash-confirm** on re-check (size is
only a prefilter); re-embedding of changed files (not just structural
re-extract); per-cycle stderr output; watch death reported and visible in
`code_status` (alive + last outcome); the MCP-side freshness owner is
stated (the watcher process owns freshness; a read-only MCP connection
reports `stale` via `code_status`).
Acceptance: a same-size edit triggers re-index (hash, not size); a killing
the watcher process leaves a visible last outcome in `code_status`; 100
rapid file events coalesce into bounded re-index work.
Acceptance scenario: watch survives bursts and deaths (acceptance.feature,
scenario "Watch coalesces, re-embeds, and reports death").

**R11 — K6 seam parity (CLI + MCP).**
Current: predecessor `contract.go` defines 38 MCP tools; the new 7-tool
surface needs fresh annotations (the old map is reference-only).
Target: all 7 tools served by the single K6 handler (ADR-0007) — CLI verbs
and per-tool MCP server emit **byte-identical envelopes**; a parity test
round-trips each tool through both transports and diffs the envelopes.
Acceptance: parity test passes for all 7 tools on the fixture repo,
including degraded-path envelopes.
Acceptance scenario: CLI and MCP envelopes are byte-identical
(acceptance.feature, scenario "K6 parity across transports").

**R12 — Transfer hygiene (predecessor → brain).**
Current: the predecessor is a different repo/module; a naive port would
drift uids, chunks, and stamps; its Store carries a Dialect/RawQuery seam
and a `postgres.go` with `migrations_pg` that has no business in a
SQLite-only brain.
Target: **UID parity fixture suite** — byte-identical uid sets, chunk
boundaries, and extractor stamps between predecessor and new binary on
golden fixtures (or a hard skill-flip cut is declared); per-language build
units (tag = grammar + node map + extractor + golden fixture) with CI
asserting linked-grammar set == map set == extToLang set; **SQLite-only
Store** (Dialect/RawQuery/postgres.go/migrations_pg deleted); gotreesitter
**VERSION in the extractor stamp** (the predecessor's `pinnedInputSet`
stamp omits it); dead SBOM DDL (`dependencies`/`dep_edges`) dropped from
the new 100+ migration; migration ranges 001-099 memory / 100-199 code /
200+ later.
Acceptance: the parity suite passes byte-identical on all golden fixtures;
CI set-equality assertion runs; the new migration applies clean on the
core-frozen schema and contains no `dependencies`/`dep_edges`.
Acceptance scenario: uid parity across binaries (acceptance.feature,
scenario "Extractor parity is byte-identical").

**R13 — Project identity.**
Current: predecessor schema has no `project` column and keys the store by
basename slug — two repos with the same basename silently overwrite each
other.
Target: the project slug is derived from repo identity (git toplevel path
or hash), not directory basename; it is reported in `code_status` and
`doctor`; identity change detection is explicit.
Acceptance: two fixture repos sharing a basename index into distinct
namespaces; cloning one repo to a different path keeps the same identity
(hash-based) or reports the new identity (path-based) — the rule is
stated in the envelope, not discovered by overwrite.
Acceptance scenario: same-basename repos do not collide (acceptance.feature,
scenario "Project identity from repo identity").

**R14 — Agent orientation contract (v1's Orientation Ladder, ported).**
Current: the v1 code index ships a usage contract
(`mnemonic/references/code-indexing.md` in the v1 skill tree) — the
canonical Orientation Ladder plus Index-Freshness trust rules — and this
7-tool surface has no equivalent. R6 quantifies drift but never tells an
agent *when to trust the index* or *in what order to navigate*; without the
contract the index is an untrusted sidecar (the pre-mortem's dead-weight
risk, line: "agent workflows don't call the 7 tools, they are dead weight").
Target: the tools carry v1's orientation contract, adapted to the v2
surface:
- **Freshness enum** `fresh|lag|empty|unknown` as the carrier of R6's drift
  quantification: no drift → `fresh`; drift past threshold → `lag` (with
  optional `lagged_paths`); no index → `empty`; unreadable mtime/hash →
  `unknown` (never false-clean). `stale=true` stays **empty-index compat
  only** — lag is `freshness=lag`, not a silent "fine".
- **Canonical navigation order** (prefer over dumping whole directory
  trees): `code_status` → `code_search` (narrow the unit — identifier,
  structural, or semantic) → `code_context` (the narrowed unit's definition
  + 80/10-line chunk + callers/callees/dependents + graph neighbors) →
  `code_explain` (deep dive on the narrowed target).
- **Trust gating:** when freshness is `lag`, `empty`, or `unknown`, run
  `code_index` before trusting `code_search`/`code_context` results.
- **Read rule:** no speculative whole-file reads — exact slices come from
  `code_context`'s chunk window for a path + range already narrowed.
- **Tool choice:** prefer the index tools over `rg`/`grep` when exploring
  an **unfamiliar** large repo; `rg`/`grep` stay the tool for
  exact-identifier lookups and when the index is a poor fit.
Acceptance: `code_status` reports `freshness: fresh` on a clean index,
`lag` + `lagged_paths` after a bounded edit, `empty` on a fresh store;
each tool's envelope/docs state its ladder position and the trust-gating
rule; the consuming skill reference (skill-flip spec, Open Questions)
carries the ladder so agent workflows call the tools in order.
Acceptance scenario: freshness gates trust and the ladder documents
navigation (acceptance.feature, scenario "Freshness gates trust, ladder
documents navigation").

## Implementation Decisions

- **Modules to build** (inside the mnemonic module, `internal/codeindex/`
  + tool handlers + transport wiring):
  - `extractor` — 3-tier ladder: gotreesitter rich tier (v1 links: go,
    python, java, javascript, typescript, tsx — copyleft grammars excluded,
    ADR-0013) → per-language node maps (~25 map-tier languages) → regex
    floor; per-language circuit breaker (persisted deaths, no silent
    quarantine).
  - `graph` — chunker (1/func-method-class ~1000 chars, 80/10-line windows;
    constants inherited from predecessor `chunker.go`), symbol uid
    derivation (span-free, R8), edge builder (EXTRACTED/INFERRED;
    `to_id NULL` + `to_name` → unresolved ledger), prune.
  - `store` — SQLite-only (ADR-0003/0004), ncruces v0.38.0 + ext/vec1 +
    ext/fts5 per-connection (ADR-0015), migrations 100-199, FTS 4-trigger
    renderer **shared** with memory-core (not copied), `IF NOT EXISTS` on
    FTS/trigger DDL (replay-safe), `schema_version` = max applied migration
    (assert `>=`), sqlite-vec version stamped in `meta`.
  - `embed` — reuses core's hugot embedder (ADR-0006/0015), batch-32 seam,
    unified space (ADR-0014).
  - `lsp` — JSON-RPC stdio client (R9).
  - `watch` — fsnotify + worker/coalescing (R10).
  - `tools` — 7 handlers over the K6 seam (R11).
- **Interfaces (contract-level, shapes owned by the blueprint):** node
  selector `{name?, path?, line?}` → symbol | `{ambiguous, candidates[]}`
  (R3); envelope: `{results?, degraded, reason?, coverage?, tier?,
  trail{leg, rank}, freshness?}`; `code_index` returns phase state
  (structural done / backfill progress / ETA).
- **Data flow:** `code_index` → walk → tier-ladder extract → files/chunks/
  symbols + edges (+ unresolved ledger) → FTS + vec1 → background backfill.
  Query: FTS MATCH + vec1 KNN (brute-force oracle bench) → RRF K=60 →
  trail. Structural: selector → graph transit (impact/path) with hop
  accounting.
- **Error handling:** breaker per language (persisted); below-rich tier note
  in every affected envelope (ADR-0011, never silent); `code_index` failure
  path records last-run status/error/duration (R6); no-index first run
  carries a reason token (not a bare error).
- **Dependencies:** `odvcencio/gotreesitter` (pinned; v0.55.1 vs v0.58.0
  open — parity fixture decides, Open Questions), `hugot v0.8.6`,
  `ncruces/go-sqlite3 v0.38.0` — all shared with memory-core, no new
  top-level deps of the code domain alone. go ≥ 1.27.
- **Prototype snippet** (vec1 KNN, from findings.md Spike 2):

  ```go
  // SELECT rowid FROM v(?, '{"K":2}') ORDER BY distance  -- table-valued KNN
  // v: CREATE VIRTUAL TABLE v USING vec1(embedding)
  //    INSERT INTO v(cmd,arg) VALUES('rebuild', '{"index":"flat","distance":"cos"}')
  ```

## Shootdown Findings

**Mode:** BRAINSTORM. Lens status: all 7 lenses **completed**
(zoom-out, design-critic, comprehension, implementation-readiness,
codebase-details, structural, inventive-catalyst) — fresh `reviewer`
subagents, R37 isolation, 2026-11-23.

**Anchored hypothesis (user-confirmed at the readiness gate, 2026-11-23):**
the `code` tool is a second domain in the same per-project SQLite brain — a
tree-sitter knowledge graph of code files (files → AST chunks ~1000 chars →
symbols (single node table, path-independent uid) → edges with confidence
EXTRACTED|INFERRED|LSP_RESOLVED, `to_id NULL` + `to_name` kept in an
unresolved ledger, drop-not-guess), chunks + symbols embedded in one
unified project vector space, searched as RRF K=60 FTS-trigram + semantic
hybrid with a floor, queried structurally (context/impact/path/explain), 7
tools over the K6 seam (CLI + MCP byte-identical), a 3-tier extraction
ladder with a circuit breaker, a best-effort LSP JSON-RPC pass, and a
`mnemonic code watch` fsnotify loop.

**Blocker dispositions (user, via ask gates, 2026-11-23):** B1
**resolved** — R7 (model-swap coverage backfill, two-phase stamp). B2
**resolved** — R1 (batch-32 backfill + synchronous structural phase; spike-
corrected math: 81 ms/sentence non-simd → ~54 min for 40k items, not the
original 5.5–22 h estimate which assumed 0.5–2 s/sentence). B3 **resolved**
— R8 + R6 (persisted tiers, span-free uids, counted cascade, computed
coverage).

| Sev | Lens | Description | Disposition | Note |
|---|---|---|---|---|
| 🔴 | design-critic | **B1:** model swap strands the corpus — `GuardModelSwap` → `ClearVectors` wipes embeddings silently | resolved | R7 two-phase swap + coverage backfill + envelope coverage (user gate) |
| 🔴 | design-critic | **B2:** first index is tens of minutes to hours, synchronous, no progress, nothing queryable until done | resolved | R1 two-phase index + batch-32 (spike: ~54 min corrected estimate) (user gate) |
| 🔴 | design-critic | **B3:** tier degradation is invisible — runtime-only tier choice, silent quarantine, span-sensitive uids rewrite ids on tier change | resolved | R8 + R6 persisted tiers, span-free uids, counted cascade (user gate) |
| 🟡 | design-critic | LSP pass: prod path is a stub; global (not per-call-site) 30 s budget; non-sticky edges demote on watch bursts; `rows[0]` re-guess; diagnostics dropped | recorded — does not block | R9 (real JSON-RPC client, per-site budget, sticky location-based edges, ambiguity ledger, persisted per-language run record); user re-confirmed keep against this evidence |
| 🟡 | zoom-out + design-critic | watch: loop blocks on re-index; death invisible and silent; size-only re-check misses same-size edits; re-indexes don't re-embed; no MCP freshness owner | recorded — does not block | R10 (worker + coalescing queue, hash-confirm, re-embed, stderr per cycle, alive + last outcome in `code_status`, owner stated); user re-confirmed keep with fixes |
| 🟡 | zoom-out + naive | node selector contract missing — hits carry ids callers cannot address; ambiguous names re-guess `rows[0]` | recorded — does not block | R3 selector contract + `candidates[]`; every hit addressable |
| 🟡 | naive | no-index first run has no reason token; `stale` undefined (git HEAD is meaningless in worktrees/dirty trees) | recorded — does not block | R6 freshness = quantified working-tree drift since last index run; R1 reason token |
| 🟡 | naive + inventive-catalyst | edge-confidence boundary unstated (which kind is strongest?); `code_impact` would be a silent lower bound without hop accounting | recorded — does not block | R4/R5: boundary sentence (LSP_RESOLVED strongest), refused/ambiguous hop counts, weakest-hop path confidence |
| 🟡 | tech-lead | store contract is a moving target while core is unfrozen; FTS 4-trigger renderer would be copied, not shared; trigram-in-WASM unverified | recorded — does not block | Open Questions: pin core tag/commit after v0.1 schema freeze; share the FTS renderer; trigram check rides core's spike list; migration ranges 001-099 / 100-199 / 200+ |
| 🟡 | zoom-out + tech-lead + design-critic | vec layer stale vs the spike record: DDL dim-pinned at 768 vs the 384-dim model; vec DDL in `.sql` dies in tag-off builds; latency criterion (<100 ms) is memory-domain, not code-domain | recorded — does not block | R12/Implementation Decisions: dim parameterized from the active embedder; vec DDL rendered in code; latency SLO per tool (default build = seconds) as an Open Question |
| 🟡 | tech-lead | two model-stamp locations (code `embed_meta` + memory `meta`) — a hash-64 → ONNX upgrade re-embeds twice or drifts | recorded — does not block | R7: drop `embed_meta`; memory `meta` is the single stamp; one coordinated cross-domain re-embed |
| 🟡 | inventive-catalyst | model selection was eval'd on memory (prose) fixtures only; MiniLM is prose-trained | recorded — does not block | Open Questions: two-domain eval (memory + code fixtures) at code-tool arrival; assumption logged until disproved |
| 🟡 | doc-architect + tech-lead | transfer hygiene: naive port drifts uids/chunks/stamps; Store carries postgres seams; extractor stamp omits gotreesitter VERSION; dead SBOM DDL in migration 100 | recorded — does not block | R12 (parity fixture suite, per-language build units + CI set equality, SQLite-only Store, version in stamp) + R12 (drop `dependencies`/`dep_edges`) |
| 🟡 | tech-lead | project identity: no `project` column + basename slug → same-basename repos silently overwrite | recorded — does not block | R13 slug from repo identity, reported in `code_status`/`doctor` |
| 🟡 | pedantic | observability persistence: trail reason only in memory; oversized files counted not named; `code_status` coverage dead 0.0; failure path unrecorded; `doctor` has no code checks | recorded — does not block | R6 (trail reason TEXT + per-leg ranks JSON; named oversized files; computed coverage; last-run status/error/duration) + `doctor` +~5 code checks (`--strict`) |
| 🟡 | doc-architect | FTS integrity unverified in CI; a generic `Exec` seam could write FTS-backed tables out of sync | recorded — does not block | Testing Decisions: FTS integrity CI check; FTS writes only via the shared renderer |
| 🟡 | design-critic | RRF floor leg: hash-64 scores every symbol 1/61 → FTS#1 with no signal | recorded — does not block | R2 floor-leg exclusion from fusion (or `floor_leg` flag); leg tier recorded in trail |
| 🟡 | tech-lead | skill flip unspecified: server wiring decision + old→new tool-name remap table absent | recorded — does not block | Open Questions: skill-flip spec at blueprint gate (includes `code_get_callers`/`code_get_callees`/`code_get_dependents` → `code_context` remap) |
| 🟢 | various | 🟢 nits (recorded): top-N unresolved ledger reader in `code_status` (R6); symbol-leg marginal value pinned by eval fixture, table stays (Q3 locked); `schema_version` sentinel = max applied, assert `>=` (R12); all 7 tool annotations written fresh (R11); trigram FTS blind to sub-3-char identifiers — name lookups ride the structural leg, one documented sentence (R2); ~150–250 MB/project footprint budget stated; sqlite-vec version stamped in `meta` with ANN-rebuild note (R12); `IF NOT EXISTS` on FTS/trigger DDL (R12); versioning rule for the module-path handover (two `mnemonic` binaries on PATH) — Open Question | recorded — does not block | folded into R2/R6/R11/R12 as cited |

Overall verdict: **survived** — all 3 🔴 resolved by user gate; 16 🟡
groups + 🟢 nits recorded into requirements/decisions/open questions;
nothing left blocking.

## Testing Decisions

- **Good tests:** external behavior — tool contracts and envelopes, parity
  fixtures, degradation envelopes. Not extractor internals (the parity suite
  pins their observable output instead).
- **Modules under test:** UID/chunk/stamp parity fixture suite
  (byte-identical predecessor vs new, or hard skill-flip cut) — the
  transfer-hygiene gate (R12); K6 CLI/MCP parity round-trip for all 7 tools
  incl. degraded paths (R11); two-domain embedding eval fixture (memory +
  code; owner/trigger: Open Questions); degradation contract — never
  silent, coverage computed (R6/R8); FTS integrity check in CI (FTS vs base
  tables after mixed writes); per-language extractor fixtures (tag =
  grammar + map + ext + fixture; CI asserts linked-grammar set == map set ==
  extToLang set); vec1-vs-brute-force KNN equivalence bench (vec1 IVFADC is
  approximate on small tables — the floor is the oracle).
- **Prior art:** predecessor `codeindex` tests where they survive the
  SQLite-only cut; memory-core's BDD `acceptance.feature` convention (BDD
  always on — every requirement above carries a scenario pointer).
- **Edge cases:** same-size edit (hash-confirm, R10); model swap mid-corpus
  (R7); grammar crash → breaker + visible tier (R8); ambiguous selector
  (R3); no-index first run (reason token); empty repo; 40k-item first-run
  budget (R1); sub-3-char identifier lookups (R2 structural leg).

## Impact on Global Docs

- `00-prd.md`: add the `code` tool as a v1.1 feature line at the blueprint
  gate (feature name, 7-tool surface, success criteria 1–6).
- `ASSUMPTIONS.md`: ADR-0013 + ADR-0014 LOCKED rows + `index.md` catalog
  rows — **done 2026-11-23**; no further edits until a new ADR clears the
  bar during blueprint.
- `ARCHITECTURE.md`: new component `codeindex` (extractor/graph/store/
  embed-reuse/lsp/watch + 7 tool handlers over K6) — added at the
  blueprint→execute handoff.
- Glossary: `glossary/02-technical-terms.md` code-tool term cluster —
  **done 2026-11-23** (edge confidence kinds, unresolved ledger, floor, RRF,
  tier ladder, uid, coverage, freshness).

## Clarity Report

| Metric | Score | Min |
|---|---|---|
| Clarity | — | 0.20 |
| Ambiguity | — | 0.10 |
| Scope creep | — | 0.25 |
| Assumption risk | — | 0.15 |
| Missing info | — | 0.10 |

Exit-check scores were not recorded in the log (interview closed without a
scored exit-check run). The clarity gate passed by user confirmation at the
interview close and at the Goldfish readiness gate (2026-11-23); the
anchored hypothesis above carries that confirmation. Flag if the
unrecorded scores matter for your review.

### Interview Log

| Question | Decision |
|---|---|
| Q1: v1 tool surface? | GitNexus-core 7 tools (programmer, 2026-11-23) |
| Q2: AST extraction? | pure-Go gotreesitter, no CGo (programmer, 2026-11-23; ADR-0013) |
| Q3: what gets embedded? | chunks + symbols (programmer override of chunks-only, 2026-11-23; ADR-0014) |
| Q4: index scope? | code files only — docs/configs out (programmer, 2026-11-23) |
| Q5: machinery? | core + LSP pass + watch (programmer, 2026-11-23) |
| Q6: knowledge base shape? | in-file graph + hybrid search; generated docs deferred (programmer, 2026-11-23) |
| Readiness gate: anchored hypothesis? | user-confirmed (2026-11-23) |
| Gate B1: model-swap strands corpus? | resolved — coverage backfill + two-phase stamp (2026-11-23) |
| Gate B2: first index too slow? | resolved — batch-32 + two-phase index (2026-11-23) |
| Gate B3: degradation invisible? | resolved — persisted tiers + span-free uids + counted cascade (2026-11-23) |
| Re-confirm LSP (prod-stub evidence shown)? | keep as built — real JSON-RPC client in v1 (2026-11-23) |
| Re-confirm watch (failure-mode evidence shown)? | keep as built, with fixes (2026-11-23) |

## Open Questions & Assumptions

| Type | Question / Assumption | Owner | Due |
|---|---|---|---|
| Q | v1 language set: rich-6 (go, python, java, javascript, typescript, tsx) + which map-tier languages | blueprint | blueprint gate |
| Q | gotreesitter pin: predecessor v0.55.1 vs current v0.58.0 (UID parity fixture decides) | blueprint | blueprint gate |
| Q | memory-core tag/commit pin after v0.1 schema freeze (the stable base this change builds on) | blueprint | blueprint gate |
| Q | two-domain model eval (memory + code fixtures) at code-tool arrival — MiniLM is prose-trained | execute | code-tool arrival |
| Q | skill-flip spec: server wiring decision + old→new tool-name remap table | blueprint | blueprint gate |
| Q | code-domain latency SLO per tool (default build = seconds; the <100 ms criterion is memory-domain) | blueprint | blueprint gate |
| A | MiniLM is adequate for code identifiers | user | until disproved by the two-domain eval |
| A | vec1 (IVFADC) is sufficient for code-domain KNN at repo scale | blueprint | KNN equivalence bench |

## Decisions (ADR)

- **0013-pure-go-treesitter** — pure-Go `odvcencio/gotreesitter` for all
  AST extraction; no CGo build tag; v1 links only supported grammars;
  copyleft grammars excluded.
- **0014-unified-embedding-space** — one embedding space per project file
  (same model + dim as memory, 384-dim MiniLM); no cross-domain fusion in
  v1; chunks and symbols share the space (two BLOB tables, one space).

Constrained by core ADRs 0003, 0004, 0005, 0006, 0007, 0009, 0010, 0011, and
0015 (stack selection, spike-amended) — reviewed in `adr.md`.

## Terms

- **Edge confidence** (EXTRACTED / INFERRED / LSP_RESOLVED), **unresolved
  ledger**, **floor**, **RRF**, **tier ladder**, **uid**, **coverage**,
  **freshness**: `glossary/02-technical-terms.md` (code-tool term cluster,
  added 2026-11-23).

## Retrospective

Pending.
