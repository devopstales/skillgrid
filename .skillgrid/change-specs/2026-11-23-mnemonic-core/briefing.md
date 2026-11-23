# mnemonic — Design Briefing (requirements & intent)

> **STATUS:** `approved` (2026-11-23)
> **Change:** `2026-11-23-mnemonic-core`
> **Predecessor record:** `brainstorm.md` (same dir) — pre-decision exploration; where this
> briefing and the brainstorm disagree, this briefing wins (decisions locked via interview 2026-11-23).

## Problem / Intent

`skillgrid/mnemonic` (in `../skillgrid`) is a proven but entangled memory implementation:
Postgres + `migrations_pg`, projectwiki generation, SDD artifact coupling, `team_*` tools, and
a ~30-tool code-graph surface all live behind one store, and its "semantic" leg is an opt-in
768-dim API embedder with a hash-64 floor — there is no local model path. `../engram` proves the
OSS product shape (single binary, agent-agnostic, doctor, setup) but ships a cloud daemon and a
memory model built around sync.

What's missing in both: a **lean, standalone, local-only memory** where the vector layer is
first-class *inside* the SQLite file and the default path works offline with a real
embedding model on CPU.

This change creates `mnemonic/` **in this repo** (`skillgrid-skills/mnemonic/`): a new Go
module, one binary named `mnemonic`, that is the **home for multiple memory tools** (v1 ships
the `memory` tool only; `code`/`web`/… are later tools under the same binary). The binary is
addressable as CLI (`mnemonic memory <verb>`) and MCP stdio (`mnemonic memory mcp`) over one
shared handler. v1 scope is **memory core only** — observations, sessions, hybrid recall
(FTS5 BM25 + in-file vectors), and ops. No code index, no web cache.

## Purpose & Success Criteria

- **Purpose:** a local-first persistent memory for AI agents: save decisions/bugfixes/
  discoveries once, recall them across sessions and compactions by meaning (vectors) or words
  (FTS), with zero network calls on the default path and one portable SQLite file per project.
- **Success criteria:**
  1. **Persistence:** an observation saved in process A is found by `search` in a fresh process
     B (file-based, no daemon).
  2. **Semantic recall:** save an observation phrased one way ("fixed N+1 in UserList"); query
     with different phrasing ("how did we solve the slow users query") → hit in top-5 with the
     local ONNX model active.
  3. **Degradation:** with the model file absent (hash-64 floor), the same query still returns
     a usable result and the envelope carries `degraded: true`. Never silent.
  4. **Upsert:** `save` with the same `topic_key` + `scope` updates the existing observation
     (`revision_count++`) — no duplicate rows; exact-duplicate content without `topic_key`
     increments `duplicate_count`.
  5. **Parity (K6):** CLI `mnemonic memory search` and MCP `mem_search` with identical inputs
     produce byte-identical envelopes (asserted by parity tests).
  6. **Single file:** a project's whole brain is one `.sqlite` (+WAL sidecars); `cp`-ing it
     elsewhere reproduces identical recall results.
  7. **Speed:** at 10k observations, hybrid `search` p50 < 100 ms (brute-force build), < 50 ms
     (vec0 build).
  8. **Ops:** `mnemonic doctor` reports schema version, active embedder tier + model + dim,
     vec availability, and per-project sizes in one envelope.
  9. **Offline default:** the default config performs no network I/O (model pre-cached
     locally; API embedder only when explicitly configured).
- **Out of scope (explicitly deferred):** code index (`code` tool), web cache (`web` tool),
  `serve` HTTP/dashboard, projectwiki generation, `team_*` tools, LLM curator
  (importance/dedup-adjudication/consolidation), importer for old `skillgrid/mnemonic` stores,
  cloud sync of any kind, TUI.

## Context

- **Predecessor data model (proven, inherited from `../skillgrid/mnemonic`):**
  `sessions` + `observations` (`type`, `scope` project|user|global, `topic_key` upsert,
  `normalized_hash` dedup, `revision_count`, `duplicate_count`, `pinned`, `superseded_by`,
  `importance`, `embedding` BLOB + `embedding_model`) + FTS5 + retrieval `trails`. The full
  table DDL lives in `../skillgrid/mnemonic/internal/core/store/migrations/001_schema.sql`.
- **Predecessor vector decision (inherited):** BLOB is the source of truth; ANN preference
  HNSW → vec0 → brute-force; dim-strict (a dim change is a re-embed, never a silent truncate);
  nil embedder = absent leg = `degraded: true` (the `degrade` package: one bounded retry,
  then the deterministic floor).
- **Transport seam (inherited, "K6"):** CLI and MCP are two adapters over one shared
  `Handler`; neither contains domain logic; the shared JSON envelope is byte-identical.
- **Project detection chain (inherited):** `--project` flag > `MNEMONIC_PROJECT` env > git
  toplevel > directory basename.
- **Exit-code contract (inherited):** 0 ok · 1 handler error · 2 usage error.
- **The `mnemonic` skill** installed in this ecosystem (`~/.agents/skills/mnemonic`) points at
  the `skillgrid` binary's `mcp` subcommand — it flips to this binary in a follow-up change,
  not this one.

## Approaches Considered

| # | Approach | Verdict |
|---|---|---|
| A | **Greenfield lean module in this repo** (`skillgrid-skills/mnemonic/`), inherit the data model + seams, build memory-core only | ✓ **chosen** — least coupling; the proven model is carried over as design, not as code to untangle |
| B | Evolve `../skillgrid/mnemonic` in place (new design replaces the old package) | ✗ — the old package is dialect-aware (SQLite **and** Postgres), spec-bound (00x spec docs), and carries projectwiki/SDD/team_*; stripping in place means deleting more than the new v1 writes |
| C | Fork `../engram`, delete cloud/daemon/TUI | ✗ — its structure serves a product (sync, identity, 14-agent setup plugins); the deletion surface exceeds a clean v1 build, and its memory model is sync-oriented |
| D | Copy `../skillgrid/mnemonic` into this repo and trim | ~ — tempting, but the memory package imports the store's dialect seams (PG), the spec-numbered packages, and the 30-tool code surface; trimming converges on A with extra archaeology |

## Requirements

**R1 — Storage.** One SQLite file per project: `~/.mnemonic/<project-slug>.sqlite`
(`$MNEMONIC_HOME` override). Engine: `modernc.org/sqlite` (pure Go). `PRAGMA
journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL`. No leases: WAL + busy timeout is
the entire concurrency story (local multi-agent scale).

**R2 — Vectors in-file (the "vector database" part).** Three layers in the same file:
1. `embedding BLOB` + `embedding_model` + `embedding_dim` on `observations` = source of truth.
2. `mem_vec` — a `vec0` virtual table via the **pure-Go `sqlite-vec` port**
   (`asg017/sqlite-vec-go`), behind build tag `sqlitevec`. Created lazily once the active
   embedder dim is stable; dropped + rebuilt on dim change.
3. **Go brute-force cosine** scan over BLOBs = the guaranteed floor, always compiled in:
   active when the vec table is absent, dim-locked, or the extension fails to load.

   External vector databases (Qdrant/Chroma/Lance) are **never** in the path — separate
   process + lifecycle at 10³–10⁵ vectors is out of proportion.

**R3 — Embedder, three tiers.**
- `local` (default): a small ONNX sentence-embedding model that **runs on CPU** (candidate:
  `all-MiniLM-L6-v2`, 384-dim, ~90 MB; final pick by v0.2 spike: CPU latency + a small
  retrieval-eval set, candidates incl. `bge-small-en-v1.5`). Model file downloaded once and
  cached at `~/.mnemonic/models/<model>.onnx`.
- `api` (opt-in): OpenAI-compatible `/embeddings` via `MNEMONIC_API_BASE`/`_MODEL`/`_KEY` —
  any compatible server, including a local Ollama
  (`MNEMONIC_API_BASE=http://127.0.0.1:11434/v1`, model e.g. this
  ecosystem's configured `embeddinggemma:300m` or `nomic-embed-text`);
  requires that daemon to be running, else the floor takes over.
- `hash` (floor): 64-dim deterministic token hash. Auto-selected when the ONNX runtime is
  unavailable (pure-Go build without the ONNX profile, or model file missing) and when
  `MNEMONIC_EMBEDDER=hash`.

  `Embedder` interface: `Embed(ctx, texts) ([][]float32, error)`; `Model()`; `Dimensions()`.
  Nil = absent (leg skipped, `degraded`); erroring = one bounded retry, then floor. **A save
  or search is never blocked on embeddings.** Every vector row is model-stamped; mixed models
  in one table is a re-embed, enforced dim-strict.

  Build profiles: **target default is C/CGo-free** — `modernc.org/sqlite` (pure Go) +
  **`unum-cloud/onnx`** (pure-Go ONNX runtime) → one static binary, no C toolchain. The v0.2
  spike must confirm the chosen model loads (opset coverage — the pure-Go runtime supports a
  *partial* opset) and bench latency; pure-Go CPU matmul without SIMD is expected ~10× slower
  than the C core (~0.5–2 s/sentence vs ~50–100 ms — tolerable per-save, slow for bulk
  re-embed, which is a background job anyway). If the spike fails (missing op or >~3 s/sentence),
  the default flips to CGo `onnxruntime` (official `onnxruntime_go`, C core) and a lean pure-Go
  profile remains functional via the hash floor. CI matrix covers whichever profiles exist.
  (A local Ollama daemon is the other way to keep the binary C-free *and* fast — via the `api`
  tier — but it adds a daemon dependency.)

**R4 — Memory model.**
- `sessions(id, project, title, dir, started_at, ended_at, summary)`; a save requires a live
  `session_id`.
- `observations`: `type` (decision|bugfix|discovery|pattern|preference|convention|
  config|learning|architecture), `title`, `content`, `scope` (project default), `topic_key`,
  `normalized_hash`, `revision_count`, `duplicate_count`, `last_seen_at`, `pinned`,
  `superseded_by`, `review_after`, `expires_at`, `importance` (default 0.5, static in v1 —
  the LLM curator is deferred), `retrieval_usage`.
- `obs_fts`: FTS5 external-content over `title` + `content`, kept consistent by triggers
  (the upsert/revision path is trigger-tested).
- `obs_links(from, to, kind)` — typed observation edges (supports|contradicts|relates);
  written but not a v1 recall leg.
- `meta(key, value)` — `schema_version`, embedder state. Migrations are additive SQL, replayed
  by version.
- `trails(id, at, query, route, legs, hits_json, scope, project)` — one row per retrieval,
  per-leg ranks included (the audit + future-eval substrate).
- `<private>…</private>` spans in saved content are stripped before storage.

**R5 — Recall pipeline.**
```
query → scope filter (project|user|global|all)
     → legs: FTS5 BM25 (always) + vector KNN (vec0 or brute, when vectors exist)
     → RRF fusion → boosts: recency, importance, pinned
     → top-K hits: id, title, ~200-char snippet, leg provenance
     → trail row; envelope carries route + degraded
```
Previews are truncated; the full body is reachable only via `get(id)` (context-budget
discipline). `degraded: true` iff the vector leg did not run.

**R6 — MCP interface.** `mnemonic memory mcp [--debug]` runs a stdio MCP server named
`mnemonic-memory` (per-tool server now; a unified `mnemonic mcp` across future tools is an
open question, not v1). v1 tools (12):

`mem_save`, `mem_search`, `mem_get`, `mem_context`, `mem_inject`, `mem_session_start`,
`mem_session_end`, `mem_session_summary`, `mem_delete`, `mem_pin`, `mem_stats`, `doctor`.

`mem_inject` = hybrid retrieval rendered into a token-budgeted markdown block
(`max_tokens` param) — the Layer-2 deep-retrieval companion to `mem_context` (Layer-1: recent
sessions + top observations, cheap).

**R7 — CLI.** The binary is a **multi-tool home**: top-level `mnemonic` hosts tools; v1 has
one tool, `memory`.

```
mnemonic                          # top-level
  init                            # create $MNEMONIC_HOME, config, (default) model download
  doctor                          # top-level: health of every registered tool (v1: memory)
  memory                          # v1 tool
    save|search|get|context|inject|stats|delete|pin
    session start|end|summary
    mcp [--debug]                 # stdio MCP server
    export [--out DIR]            # markdown + JSON dump of one project
```

- Output = the raw JSON envelope (scriptable); `--pretty` for humans.
- `--project` / `--all` selectors per the detection chain; `--all` opens every file in
  `$MNEMONIC_HOME`.
- Exit codes 0/1/2; usage errors go to stderr without an envelope.

**R8 — Single handler (K6).** One `Handler` type; `memory` domain methods; CLI flag-parsers
and MCP tool dispatchers map arguments 1:1 to Handler calls. Parity tests assert byte-identical
envelopes for matched inputs.

**R9 — Offline default.** No network I/O on the default path. Model download happens once at
`init` (or first `save`, with a `--no-models` escape); API embedder only when configured.

## Implementation Decisions

- **Repo layout** (in this repo):
  ```
  mnemonic/
    go.mod                        # module …/skillgrid-skills/mnemonic (path TBC)
    cmd/mnemonic/main.go          # top-level dispatch: init|doctor|<tool> …
    internal/store/               # sqlite engine, migrations, fts, vector seams (vec0 tag)
    internal/memory/              # save/search/recall/sessions/dedup — the Handler's domain
    internal/intelligence/        # embedder: onnx | api | hash; degrade (one bounded retry)
    internal/transport/           # handler, envelope, mcp adapter, cli adapter
    internal/project/             # detection chain
    internal/config/              # config.toml + env overlay
    models/                       # (fetched at init; not vendored in v1)
  ```
- **Go 1.26**, MIT, single binary `mnemonic`; deps kept to: modernc.org/sqlite,
  mcp-go, sqlite-vec-go (tag), onnxruntime_go (profile), yaml/toml config, humanize. No
  lib/pq, no fsnotify in v1.
- **Vector spike gate:** before v0.2 commits to vec0, a spike validates the pure-Go port's API
  (dim-strictness, cosine, multi-table) against the floor; failure → build tag off, brute
  force only (success criterion 7 measured on the brute path).
- **Model choice spike (v0.2):** CPU-latency + small retrieval-eval comparison of the
  384-dim candidates; winner recorded as an ADR with numbers.
- **Phasing:**
  - **v0.1 core:** store + migrations; sessions/observations/FTS5; save/search/get/context;
    session tools; hash-64 vectors (brute force); trails; stats; doctor; MCP(12) + CLI; parity
    tests. *This is already a shippable memory.*
  - **v0.2 hybrid:** local ONNX embedder (model download/cache), API embedder, vec0 (spike),
    RRF fusion, `mem_inject`, `export`.
  - **v1.0:** polish, both build profiles in CI, docs, release tag.
- **Deferred tool surface is pre-validated only by the data model:** `mem_vec` naming and the
  `project` column on every table are chosen so the future `code`/`web` tools (chunks,
  symbols, edges, web snapshots + their FTS/vec tables) slot in via additive migrations.

## Shootdown Findings

- **"Memory-only v1 is half the old product."** True — the installed `mnemonic` skill's
  code-ladder steps (Step 5) have nothing to call until the `code` tool lands. Mitigation: the
  skill flip (to `mnemonic memory mcp`) ships with the `code` tool, not with v1.0; until then
  the old `skillgrid` binary serves code indexing. The memory-only v1 is deliberately a
  *new*, independent surface, not a drop-in replacement.
- **"vec0 at 10k observations is over-engineering."** Brute force alone would meet criterion 7.
  Kept as a hedge for the future code tool (10⁴–10⁵ chunks) and for multi-table queries; cost
  is one build tag + one spike, and the floor makes a failed spike a non-event.
- **"ONNX in pure Go is too slow for the default path."** Pure-Go matmul (no SIMD) is ~10×
  the C core: ~0.5–2 s/sentence for MiniLM-class. Accepted as the price of a C/CGo-free
  default: per-save embed is tolerable, bulk re-embed is a background job, and the spike owns
  the escape hatch (CGo `onnxruntime` default if >~3 s/sentence or a missing op).
- **"FTS5 external-content + the topic_key upsert path is a classic desync trap."** The
  revision-UPDATE path (not just INSERT/DELETE) must fire the FTS triggers; dedicated
  regression test: save → upsert-revise → delete, asserting `obs_fts` matches the table each
  time.
- **"Per-project files make cross-project recall N file opens."** At 10–50 projects this is
  ms-scale; `--all` documents it. Federation machinery stays deferred until a project count
  hurts (there will be a `trails` entry showing the cost when it does).
- **"RRF is a tuning black box."** `trails` stores per-leg ranks; a fixed eval fixture
  (small corpus + queries with expected top-k) pins recall behavior across refactors, so RRF
  weight changes are testable.

## Testing Decisions

- **Parity:** for every Handler method, CLI invocation and MCP invocation with the same input
  → byte-identical envelope (golden comparison, not field-by-field).
- **Store:** migration replay from empty and from N-1; FTS trigger consistency across
  insert/upsert-revise/delete (the Shootdown trap); WAL concurrency — two processes saving
  concurrently, no `SQLITE_BUSY` beyond the timeout window.
- **Semantics:** upsert (`revision_count++`, no dup row), dedup (`duplicate_count++`),
  scope isolation (project-scoped saves invisible to other projects without `--all`),
  `<private>` stripping.
- **Embedder:** hash-64 golden vectors (pinned); ONNX golden vectors with a tiny fixture
  model, skipped under the pure-Go profile; dim-mismatch → error, never truncate.
- **Recall:** a saved observation is returned for its exact-title query (within top-1 after
  boosts); `degraded: true` iff and only if the vector leg was absent; trail row written per
  search.
- **Perf:** bench at 10k observations (criterion 7) run on both build profiles in CI.
- **Exit codes:** usage → 2 (no envelope), handler error → 1 (envelope with stable `code`).

## Impact on Global Docs

- **This repo:** `docs/` gains a command reference for `mnemonic` (follow-up task, small);
  `.skillgrid/projectwiki/` gains a `mnemonic` entity page at v1.0.
- **Installed `mnemonic` skill** (`~/.agents/skills/mnemonic`, `based_on: skillgrid-v2:mnemonic`):
  currently points at the `skillgrid` binary. Follow-up change flips config to `mnemonic
  memory mcp` **when the `code` tool lands** (memory-only v1 would leave the skill's code
  steps dangling). No skill change in this change.
- **`../skillgrid`:** untouched by v1.0; the old `mnemonic` package keeps serving
  skillgrid until the importer + skill flip (both deferred).

## Open Questions & Assumptions

1. **Go module path** — assumed `github.com/devopstales/skillgrid-skills/mnemonic` (this
   repo's path + `/mnemonic`). *Assumed until confirmed.*
2. **Exact ONNX model + runtime** — `all-MiniLM-L6-v2` (384-dim) is the working candidate;
   the v0.2 spike decides both the model (CPU latency + retrieval eval) and whether
   `unum-cloud/onnx` can run it (opset coverage). Assumed for now; ADR to record the winners.
3. **Model distribution** — download-on-`init` + cache in `~/.mnemonic/models/` (repo stays
   lean) vs vendoring the ~90 MB file in the repo. **Default: download + cache**, with
   `--no-models` and hash-floor for offline first-run.
4. **MCP server naming for the future** — per-tool servers (`mnemonic memory mcp`, later
   `mnemonic code mcp`) vs one unified `mnemonic mcp`. **Default: per-tool** (matches the CLI
   hierarchy; agents wire what they need).
5. **`init` and `doctor` at top level** (not under `memory`) — assumed correct for a
   multi-tool binary: data dir and health are cross-tool concerns. Trivial to re-nest if the
   `memory` tool ever owns them exclusively.
6. **sqlite-vec-go pure-Go port API parity** (dim-strictness, cosine distance, multi-table) —
   unverified; the spike gate (Implementation Decisions) owns it. Failure mode is known:
   build tag off, brute force only, no feature loss.

## Decisions (ADR)

- **ADR-1 — Per-project SQLite file** (`~/.mnemonic/<slug>.sqlite`) over one global DB:
  portable brain, crash isolation; `--all` = open-all at local project counts.
- **ADR-2 — Vectors live inside the SQLite file**: BLOB source of truth + vec0 (pure-Go port,
  build-tagged) + brute-force floor. No external vector database, ever.
- **ADR-3 — C/CGo-free is the target build:** `modernc.org/sqlite` (pure Go) +
  `unum-cloud/onnx` (pure-Go ONNX runtime), spike-gated (opset coverage + CPU latency of the
  chosen model). Spike failure flips the default to CGo `onnxruntime` (C core); either way the
  hash-64 floor keeps a lean profile fully functional.
- **ADR-4 — Local ONNX CPU model is the default embedder tier**; API opt-in; hash floor.
  Model-stamped, dim-strict vectors.
- **ADR-5 — Single handler, two adapters (K6)**: CLI + MCP over one `Handler`, byte-identical
  envelopes, parity-tested.
- **ADR-6 — v1 = memory core only.** Deferred (named): `code`, `web`, `serve`, projectwiki,
  `team_*`, LLM curator, old-store importer, cloud sync, TUI.
- **ADR-7 — Concurrency:** WAL + `busy_timeout=5000` only; no leases in v1.
- **ADR-8 — Multi-tool binary:** `mnemonic <tool> <verb>`; `memory` is the first tool; top-level
  `init`/`doctor` are cross-tool. Per-tool MCP servers.
- **ADR-9 — Degradation contract:** vector leg absent ⇒ `degraded: true`, never silent; embedder
  errors never block save/search (one bounded retry → floor).

## Terms

| Term | Meaning |
|---|---|
| observation | one memory unit: typed, titled, scoped note (decision/bugfix/discovery/…) |
| session | an agent session boundary; saves carry a live `session_id`; ends with a summary |
| topic_key | stable upsert key: same `topic_key`+`scope` ⇒ UPDATE (revision++), not INSERT |
| scope | visibility partition: project (default) \| user \| global |
| floor | the deterministic, dependency-free path (hash-64 vectors, brute-force scan) that keeps every feature working with zero network/model |
| degraded | envelope flag: a recall ran without its vector leg; results may miss semantic hits |
| RRF | reciprocal-rank fusion of the FTS and vector leg rankings |
| vec0 | sqlite-vec virtual table (ANN index) inside the SQLite file |
| K6 seam | CLI and MCP share one Handler and byte-identical envelopes |
| trail | one row per retrieval (query, route, per-leg ranks, hits) — audit + eval substrate |
