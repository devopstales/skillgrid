# mnemonic memory-core — Implementation Blueprint (steps, files, verification)

> **STATUS:** `approved` (2026-11-23)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:execute (subagent mode recommended, or inline mode) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Tier:** T1

**Build shape:** Tracer thread — Task 1–3 land a working `save → search` vertical (store + FTS5 + hash vectors + both adapters); every later task extends the spine, never re-plumbs it.

**Goal:** Build the `mnemonic` binary (Go ≥ 1.27) in `mnemonic/` in this repo: a local-first AI memory — one SQLite file per project, FTS5 + in-file vectors, hybrid recall, sessions — exposed as CLI (`mnemonic memory <verb>`) and MCP stdio (`mnemonic memory mcp`) over one shared Handler. v1 = memory core only.

**Architecture:** Four layers, one direction: `store` (CGO-free SQLite via `ncruces/go-sqlite3`; FTS5 external-content tables; `embedding` BLOB source of truth + optional `vec1` virtual tables (ncruces' official `ext/vec1`) behind build tag `sqlitevec`) → `memory` domain (save/search/recall/sessions/dedup, the Handler) → `transport` (envelope + two thin adapters: CLI flag-parser, MCP stdio). Default embedder: local ONNX (all-MiniLM-L6-v2, 384-dim) via `knights-analytics/hugot`'s pure-Go (GoMLX) backend; API embedder opt-in; hash-64 floor so every feature works with zero model, zero extensions.

**Tech Stack:** Go ≥ 1.27 (hugot v0.8.6 requires it; a 1.26 host auto-switches toolchains); `github.com/ncruces/go-sqlite3` v0.38 (CGO-free SQLite, FTS5, virtual tables, official `ext/vec1` ANN extension — findings [4][11]); `github.com/knights-analytics/hugot` Go backend (pure-Go ONNX/GoMLX, findings [3][13]); `github.com/mark3labs/mcp-go` (MCP stdio, same lib the old package used); stdlib `flag` for CLI dispatch. No CGo in the default build; ORT tag (`-tags ORT`) is the speed escape hatch inside hugot (spike: 13 ms vs 81 ms p50; pin ORT ≥ 1.23).

**Spec:** `.skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md` (approved 2026-11-23) — requirements R1–R9 and the 9 success criteria live there; this plan is the how.

**Findings:** `.skillgrid/change-specs/2026-11-23-mnemonic-core/findings.md` — Research § (2026-11-23): CGO-free embedding runtime + vector stack selection, model distribution, counter-evidence on the brainstorm's spike-target modules.

**Wiki:** [[decisions/04-adr-0003-per-project-sqlite-file]] [[decisions/04-adr-0004-vectors-in-file]] [[decisions/04-adr-0005-c-cgo-free-target-build]] [[decisions/04-adr-0006-local-onnx-default-embedder]] [[decisions/04-adr-0007-single-handler-two-adapters]] [[decisions/04-adr-0008-memory-core-only-v1]] [[decisions/04-adr-0009-wal-only-concurrency]] [[decisions/04-adr-0010-multi-tool-binary]] [[decisions/04-adr-0011-degradation-contract]] [[decisions/04-adr-0012-go-module-path]] [[concepts/in-file-vector-layer]] [[concepts/single-handler-two-adapters]] [[concepts/degradation-floor]] [[gotchas/fts5-external-content-upsert]] [[architecture/mnemonic-memory-core]] [[syntheses/mnemonic-core]] [[entities/mnemonic]]

## Terms

None introduced. The plan uses the terms fixed during brainstorm, maintained in [[glossary/02-technical-terms]] (observation, session, topic_key, scope, floor, degraded, RRF, vec1, K6 seam, trail) — one meaning each, no redefinitions here.

## Hypothesis

> Validated at the R26 gate (2026-11-23, accepted).

**Claim:** a fresh Go module in `mnemonic/` ships **v0.1** — store + sessions/
observations + FTS5 + hash-64 vectors + `save`/`search` + CLI/MCP byte-parity —
fully offline; the unverified dependencies (hugot GoMLX running MiniLM, vec +
FTS5 in one CGO-free build) are spike-gated so neither can block v0.1. v0.2
then adds the ONNX tier + vec1 + RRF.

**Spike record (2026-11-23, both PASS — see `findings.md` § Spike Results):**
MiniLM on GoMLX works (dim 384, cosine 1.000000 vs independent ORT reference,
p50 81 ms/sentence non-simd ≤ 3 s target); `ext/vec1` + FTS5 verified in one
ncruces v0.38 connection. The asg017 vec0 binding was refuted (unpairable
wasm/host drift) — the vector mechanism is the official `ext/vec1`.
Falsifiers (a)/(b) below were NOT triggered.

**Falsifiers:**
- (a) the pure-Go ONNX path cannot run all-MiniLM-L6-v2 (opset or latency)
  and the ORT/CGo escape is unacceptable → local tier moves to v1.0, v0.2
  ships FTS-only with `degraded: true` on the vector leg.
- (b) vec + FTS5 do not work together in the ncruces WASM build →
  `sqlitevec` tag ships off, brute force owns the leg (function intact,
  speed deferred). *(Spike 2026-11-23: PASS via `ext/vec1`.)*
- (c) byte-parity between CLI and MCP cannot be enforced at adapter level →
  Handler redesign before Task 5.

**Thinning:** v0.1 core IS the smallest slice — no ONNX, no vec1, no RRF
(hash-64 + FTS5 only). If v0.1 fails a falsifier, the loss is bounded to v0.2.

## Key Decisions

Cited from the ADR record + findings (R23); a task contradicting a row is a plan defect.

| Decision | Chosen | Why (the trade-off that decided it) |
|---|---|---|
| Storage | one `.sqlite` per project under `~/.mnemonic/`, WAL + `busy_timeout=5000`, no leases | portable brain + crash isolation; contention domain is one local project (ADR-0003, ADR-0009) |
| SQLite driver | `ncruces/go-sqlite3` v0.38 (CGO-free, FTS5, virtual tables) | the only CGO-free driver that can host the vec extension; modernc (briefing) cannot load extensions — refinement confirmed at the 2026-11-23 gate (findings [4][11]) |
| Vector layer | BLOB source of truth + **`ext/vec1`** (ncruces' official ANN extension, tag `sqlitevec`, per-connection registration) + Go brute-force floor; no external vector DB | official in-driver extension, no third-party binding (the asg017 vec0 binding was refuted by spike 2 — wasm/host drift); the floor makes any vec failure a non-event (ADR-0004, ADR-0015, user-gated 2026-11-23) |
| Embedder | local ONNX all-MiniLM-L6-v2 (384-dim) via `knights-analytics/hugot` Go backend (GoMLX, pure Go) = default; API (OpenAI-compatible, Ollama-compatible) opt-in; hash-64 floor | real semantic recall offline; the GoMLX backend is explicitly designed for this model; one bounded retry → floor (ADR-0005, ADR-0006, findings [3][6]) |
| Build | CGO-free default; `-tags ORT` (hugot) = pre-decided speed escape; `GOEXPERIMENT=simd` for the GoMLX backend | C/CGo-free is the target; the escape is inside the same library, not a second runtime (ADR-0005, findings [3]) |
| Transport | one shared `Handler`; CLI + MCP are thin adapters; byte-identical envelopes; parity tests per method | one behavior, two surfaces; drift becomes a test failure (ADR-0007) |
| Scope | memory core only; deferred (named): code/web tools, serve, projectwiki, team_*, LLM curator, importer | a complete useful memory before a second project is glued on (ADR-0008) |
| CLI shape | `mnemonic <tool> <verb>`; v1 tool `memory`; MCP server `mnemonic-memory`; top-level `init`/`doctor` | the binary is a multi-tool home from day one (ADR-0010) |
| Degradation | `degraded: true` + reason whenever the full path is absent; floors never block save/search | every optional piece stays optional (ADR-0011) |
| Module path | `github.com/devopstales/skillgrid/mnemonic` (same as the old module — branch handover `release/2` → `feat/v2-rewrite`) | the path is the product's name; no import migration for consumers (ADR-0012) |

**Rejected (briefing § Approaches Considered):** evolve `../skillgrid/mnemonic` in place
(dialect-aware + spec-bound; stripping exceeds a clean v1) · fork engram (cloud/daemon/TUI
deletion surface exceeds a clean v1) · copy-and-trim the old package (converges on
greenfield with extra archaeology).

**R21 additions (new in this plan, reported at Self-Review):** driver = ncruces
(briefing said modernc — refinement, user-gated) · runtime = hugot GoMLX
(briefing said unum-cloud/onnx — 404, findings [10]) · ANN scale choice for the
future `code` tool deferred to Open Questions (v1 ships with the floor hot).

## Must-Haves (goal-backward verification)

**Truths** (briefing success criteria 1–9 + R-requirements, verifiable):
- **T1 persistence** — an observation saved in process A is found by `search` in a fresh
  process B (same data dir), no daemon. *(criterion 1, R4)*
- **T2 semantic recall** *(v0.2, ONNX active)* — save "Fixed N+1 query in UserList"; query
  "how did we solve the slow users query" → hit in top-5. *(criterion 2, R5)*
- **T3 degradation** — with no model present, search still returns results and the envelope
  carries `degraded: true` + reason; never empty, never silent. *(criterion 3, ADR-0011)*
- **T4 upsert** — same `topic_key`+`scope` ⇒ UPDATE + `revision_count++`, no dup row;
  exact-duplicate content without `topic_key` ⇒ `duplicate_count++`. *(criterion 4, R4)*
- **T5 parity** — `mnemonic memory search` and MCP `mem_search`, identical inputs ⇒
  byte-identical envelopes. *(criterion 5, ADR-0007)*
- **T6 single file** — `cp` a project's `.sqlite` (+WAL sidecars) to a fresh data dir ⇒
  identical search results. *(criterion 6, ADR-0003)*
- **T7 speed** — 10k observations: hybrid search p50 < 100 ms (brute build), < 50 ms
  (`sqlitevec` build). *(criterion 7)*
- **T8 doctor** — one envelope reporting: schema_version, active embedder tier/model/dim,
  vec availability + dim, per-project file sizes. *(criterion 8, R6)*
- **T9 offline** — default config: save/search paths perform zero network I/O. *(criterion 9, R9)*
- **T10 FTS integrity** — after save → upsert-revise → delete, `obs_fts` matches the base
  table at every step (the upsert-trigger gotcha). *([[gotchas/fts5-external-content-upsert]])*
- **T11 scope isolation** — a `project`-scoped observation is invisible to another project's
  search without `--all`/`all_projects`. *(R5)*
- **T12 session contract** — `mem_save` without a live `session_id` fails; summary + end
  persist on the session row. *(R4, R6)*
- **T13 private strip** — `<private>…</private>` spans are removed before storage. *(R4)*
- **T14 exit codes** — usage ⇒ 2 (stderr, no envelope); handler error ⇒ 1 (envelope, stable
  `code`); ok ⇒ 0. *(R7)*

**Artifacts:**
- [`mnemonic/go.mod` — module `github.com/devopstales/skillgrid/mnemonic`, Go 1.27]
- [`mnemonic/cmd/mnemonic/main.go` — top-level dispatch: `init` | `doctor` | `memory …`]
- [`mnemonic/internal/store/` — engine (WAL/busy_timeout), additive migrations, FTS triggers, vector seam]
- [`mnemonic/internal/memory/` — save/search/recall/sessions/dedup + the Handler]
- [`mnemonic/internal/intelligence/embedder/` — hash-64 | onnx (hugot GoMLX, tag `onnx`) | api + degrade]
- [`mnemonic/internal/transport/` — envelope, Handler wiring, CLI adapter, MCP adapter (mcp-go)]
- [`mnemonic/internal/project/` — detection chain; `mnemonic/internal/config/` — config.toml + env]
- [`mnemonic/**/*_test.go` — parity golden tests, FTS trigger tests, 10k bench, WAL concurrency]

**Key links:**
- `mem_save` writes base row + `obs_fts` (trigger) + `embedding` BLOB (+ `mem_vec` row under
  tag `sqlitevec`) in **one transaction** — T10's correctness rests on this.
- `search` embeds the query with the **same model** stamped at save time (dim-strict); the
  envelope carries `route` + `degraded` (T2/T3).
- CLI parser and MCP dispatch both call the **same Handler methods** — no second
  implementation of save/search anywhere (T5).
- every `store.Open` applies WAL + `busy_timeout=5000` — no code path opens a raw driver
  conn (ADR-0009).

**One-way-door decisions:**
- ⚠ **Initial schema shape** (migrations `001_*`: sessions/observations/obs_fts/obs_links/
  meta/trails) — the user-data format; additive migrations keep it reversible, but the v1
  shape is the format future tools (code/web/kb) inherit.
- ⚠ **Envelope shape** (`{ok, code, data, degraded, elapsed_ms}`) — the contract agents and
  scripts parse; changing it later is a consumer break.
- ⚠ **Default data dir `~/.mnemonic/`** — user-visible state location (env `MNEMONIC_HOME`).
- Module path is already one-way and locked (ADR-0012).

## Validation

One row per in-scope requirement (briefing R1–R9); out-of-scope rows state what is NOT proven.

| Req | Happy | Edge | Failure | Target | Check | Falsifier |
|---|---|---|---|---|---|---|
| R1 storage | save creates `~/.mnemonic/<slug>.sqlite`, WAL on | slug from git subpath; `MNEMONIC_HOME` override | disk full / perm denied → `store_unavailable` | — | `mnemonic memory save` → `ls`; `PRAGMA journal_mode` = wal | file absent, or journal_mode ≠ wal |
| R2 vectors in-file | search reports vector-leg provenance | dim change ⇒ rebuild, no stale vectors | vec load fails ⇒ brute force + `degraded` | T7 p50 | 10k bench, both builds | p50 over target; dim change rescales without re-embed |
| R3 embedder | local ONNX: 384-dim, model-stamped | model file missing ⇒ hash + `degraded` | ONNX load error ⇒ retry 1 ⇒ floor, save unblocked | embed ≤ 3 s/sentence (spike-verified) | golden-vector test; `doctor` | MiniLM opset fails on GoMLX; latency > 3 s |
| R4 memory model | save + upsert + dedup (T4) | `<private>` strip (T13) | save w/o session ⇒ `no_session` | — | store tests | dup row on upsert; unstripped secret in stored content |
| R5 recall | hybrid top-K with leg provenance (T2) | empty store; empty query | vector leg absent ⇒ FTS-only + `degraded` (T3) | T7 | recall tests; trail row written | FTS-only run without the flag |
| R6 MCP (12 tools) | all tools listed + callable | bad arg ⇒ MCP error, stable code | store locked ⇒ error envelope | — | MCP dry-run + parity golden | any tool's shape diverges from the CLI envelope |
| R7 CLI | `mnemonic memory search "x"` ⇒ envelope JSON | unknown flag ⇒ exit 2, no envelope | handler error ⇒ exit 1 + envelope | — | CLI tests (T14) | wrong exit code on any path |
| R8 single handler | parity golden (T5) | concurrent CLI + MCP (WAL) | parity mismatch | — | `go test ./internal/transport -run Parity` | byte diff on any method |
| R9 offline default | zero network I/O on save/search | first run, no model: no download attempt | API tier down ⇒ floor + `degraded` | — | run with DNS blocked | any outbound connection on the default path |
| out-of-scope | — | — | — | — | — | code index, web cache, serve, projectwiki, team_*, LLM curator, importer: NOT proven by this plan |

## Threat Matrix

Local single-user tool; the boundaries are user-supplied text, the downloaded model file, and
filesystem paths.

| Threat | Boundary | Applicable | Mitigation |
|---|---|---|---|
| FTS5 query-syntax injection (user text as MATCH input) | store | yes | queries built as quoted/parameterized FTS5 expressions (balanced quotes, escaped); never raw-concatenated user text |
| Model file supply chain (downloaded `.onnx`) | v0.2 init | yes | pinned URL + sha256 + size manifest; verify before enabling the tier; stored under `~/.mnemonic/models/` only |
| Path traversal via `--project` / `export --out` | filesystem | yes | project slug sanitized (no `/`, `..`); export target confined under the allowed root (data dir or explicit `--out`), per the old package's allowed-root rule |
| Concurrent writers (CLI + MCP, same project) | store | yes | WAL + `busy_timeout=5000`; `SQLITE_BUSY` surfaces as `store_busy` (no silent data loss) |
| Secrets in saved content | store | yes | `<private>…</private>` stripped before storage (T13) |
| Prompt injection via recalled content | agent boundary | documented | recalled content is data for the consuming agent; the store marks provenance, the agent layer treats it as untrusted (not a store fix) |

## Failure Handling

| Failure | Detection | Behavior | Recovery |
|---|---|---|---|
| write contention > 5 s | `SQLITE_BUSY` | envelope `store_busy`, exit 1 — no auto-retry in v1 | caller retries; owner: invoking agent/user |
| embedder error (ONNX/ORT/API) | call error | one bounded retry → hash floor, `degraded: true` | `mnemonic doctor` names the cause; `mnemonic init` restores the model |
| model file missing | load attempt fails | hash floor + `degraded` | `mnemonic init` (download + verify) |
| vec extension fails to load (`sqlitevec` build) | registration error at open | brute force owns the leg + `degraded` | doctor reports vec state; rebuild without the tag as last resort |
| corrupt database | `SQLITE_CORRUPT` on open | doctor reports `integrity_check` result | restore from a `cp` of the `.sqlite` (the portable file IS the backup unit) |
| disk full | write error | envelope `store_full` | free space; WAL sidecars are the space to reclaim first |
| API embedder endpoint down | connection error | floor + `degraded` | endpoint is optional by design; nothing to recover |
| migration failure at open | schema_version mismatch | envelope `schema_mismatch`, store read-only | manual step per doctor output (documented per migration) |

## File Structure

All new files live under `mnemonic/` (this repo, branch `feat/v2-rewrite`). Nothing outside
`mnemonic/` changes in this plan except the spec zone (`.skillgrid/`).

- `mnemonic/go.mod` — module `github.com/devopstales/skillgrid/mnemonic`, Go 1.27
- `mnemonic/cmd/mnemonic/main.go` — top-level dispatch: `init` | `doctor` | `memory …`; exit codes 0/1/2
- `mnemonic/cmd/mnemonic/memory.go` — `memory` tool: flag sets → Handler calls → envelope render
- `mnemonic/internal/store/store.go` — `Open` (driver, WAL, `busy_timeout=5000`, migrations), `Close`
- `mnemonic/internal/store/migrations/001_schema.sql` — sessions, observations, obs_fts (+INSERT/**UPDATE**/DELETE triggers), obs_links, meta, trails
- `mnemonic/internal/store/vector.go` — BLOB encode/decode, brute-force cosine, dim-strict checks
- `mnemonic/internal/store/vector_vec.go` — `//go:build sqlitevec`: `vec1.Register(conn)` per connection, `mem_vec` DDL lifecycle, KNN (table-valued)
- `mnemonic/internal/store/vector_vec_stub.go` — `//go:build !sqlitevec`: brute-force-only stub (same interface)
- `mnemonic/internal/memory/handler.go` — **the Handler**: Save/Search/Get/Context/Inject/SessionStart/SessionEnd/SessionSummary/Delete/Pin/Stats/Doctor
- `mnemonic/internal/memory/save.go` — upsert (`topic_key`+`scope`), `normalized_hash` dedup, `<private>` strip, embed-on-write
- `mnemonic/internal/memory/recall.go` — scope filter, FTS leg, vector leg, RRF, boosts, trail write
- `mnemonic/internal/memory/sessions.go` — session lifecycle + six-section summary storage
- `mnemonic/internal/intelligence/embedder/embedder.go` — `Embedder` interface + degrade (one bounded retry → floor)
- `mnemonic/internal/intelligence/embedder/hash.go` — 64-dim FNV floor
- `mnemonic/internal/intelligence/embedder/onnx.go` — `//go:build onnx`: hugot GoMLX session, MiniLM load, pooling, normalize
- `mnemonic/internal/intelligence/embedder/onnx_stub.go` — `//go:build !onnx`: absent state (→ hash)
- `mnemonic/internal/intelligence/embedder/api.go` — OpenAI-compatible `/embeddings` (Ollama-compatible)
- `mnemonic/internal/transport/envelope.go` — `Envelope` type + stable `code` table
- `mnemonic/internal/transport/cli.go` — arg→Handler mapping, JSON out, exit codes (no domain logic)
- `mnemonic/internal/transport/mcp.go` — mcp-go stdio server `mnemonic-memory`, 12 tools → Handler (no domain logic)
- `mnemonic/internal/project/detect.go` — `--project` > `MNEMONIC_PROJECT` > git toplevel > basename; slug sanitize
- `mnemonic/internal/config/config.go` — `~/.mnemonic/config.toml` + env overlay
- `mnemonic/models/manifest.json` — model URL + sha256 + size (committed; the `.onnx` is fetched at `init`)
- tests: `internal/store/{store,fts_triggers,vector_bench,migration,wal_concurrency}_test.go`, `internal/memory/{save,recall,handler}_test.go`, `internal/transport/{parity,cli,mcp}_test.go`, `internal/intelligence/embedder/{hash,onnx_golden}_test.go`

## Global Constraints

- **Go ≥ 1.27; CGO-free default build.** Tags: `onnx` (hugot GoMLX — pure Go, still CGO-free),
  `sqlitevec` (official `ext/vec1` registration — CGO-free), `ORT` (hugot ORT — the only native
  escape). `GOEXPERIMENT=simd` recommended for the GoMLX backend (findings [3]).
- **Dependency allowlist** (no additions without a gate): `ncruces/go-sqlite3`
  (+ its companion `ncruces/go-sqlite3-wasm/v6` — the wasm binary),
  `knights-analytics/hugot` (tag), `mark3labs/mcp-go`, stdlib.
- **Envelope** `{ok, code, data, degraded, elapsed_ms}` with the stable code table in
  `transport/envelope.go`; **exit codes** 0 ok · 1 handler error · 2 usage.
- **Data:** `~/.mnemonic/` (`MNEMONIC_HOME`); per-project `<slug>.sqlite`; env
  `MNEMONIC_PROJECT`, `MNEMONIC_EMBEDDER` (hash|local|api), `MNEMONIC_API_BASE/_MODEL/_KEY`.
- **Network:** zero on the default save/search path; only `init`'s one-time verified model
  download and an explicitly-configured API tier may touch the network (R9).
- **Module path** `github.com/devopstales/skillgrid/mnemonic`; no cross-branch imports while
  `release/2` is live (ADR-0012).
- **Seam discipline:** every save/search goes through the Handler; adapters carry no domain
  logic (ADR-0007); every store open applies the PRAGMAs (ADR-0009).
- **Commits:** conventional style; spec-zone files commit separately from code (zone guard).

## Tasks

### Task 1: Store — driver, schema, FTS triggers

**Files:**
- Create: `mnemonic/go.mod`, `mnemonic/internal/store/store.go`, `mnemonic/internal/store/migrations/001_schema.sql`, `mnemonic/internal/store/migration.go`
- Test: `mnemonic/internal/store/store_test.go`, `mnemonic/internal/store/fts_triggers_test.go`, `mnemonic/internal/store/wal_concurrency_test.go`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: `store.Open(path string, opts Options) (*Store, error)`; `(*Store) Exec/Query/Close`; `(*Store) Begin() Txn` — the only entry points later tasks may use for writes
- Seam: `store.Store` wraps `*sqlite3.Conn` (ncruces native API) — behaviour (PRAGMAs, extensions, migrations) is alterable without touching callers
- Deletion test: every SQL call in `memory/` re-opens raw connections and re-applies PRAGMAs — the whole ADR-0009 discipline
- Adapters: prod (ncruces on-disk) + test (ncruces `:memory:`) — 2, seam justified

**SATISFIES:** `R1 storage` (file created, WAL), `T10 FTS integrity`

- [ ] **Step 1: Failing tests** — `store_test.go`: Open temp-file store ⇒ file exists, `PRAGMA journal_mode` = `wal`, `busy_timeout` = 5000; `fts_triggers_test.go`: INSERT ⇒ FTS finds title text; UPDATE title/content (the revision path) ⇒ FTS finds new text, not old; DELETE ⇒ FTS empty; `wal_concurrency_test.go`: 2 processes, 50 concurrent saves each ⇒ 0 errors, 100 rows.
- [ ] **Step 2: Run, expect FAIL** (no package yet): `cd mnemonic && go test ./internal/store/` → build failure.
- [ ] **Step 3: Implement** — `go.mod` (module `github.com/devopstales/skillgrid/mnemonic`, go 1.27, `ncruces/go-sqlite3` v0.38); `store.go` on the **native API** (the v0.38 database/sql driver exposes no DSN flag for extension registration — the spike used the native API):
  ```go
  func Open(path string, o Options) (*Store, error) {
      conn, err := sqlite3.Open(path)
      if err != nil { return nil, err }
      if err := fts5.Register(conn); err != nil { return nil, err } // ext/fts5, always
      // vec1.Register(conn) — in vector_vec.go under //go:build sqlitevec
      conn.BusyTimeout(5 * time.Second)
      for _, p := range []string{"PRAGMA journal_mode=WAL",
          "PRAGMA synchronous=NORMAL", "PRAGMA foreign_keys=ON"} {
          if err := conn.Exec(p); err != nil { return nil, err }
      }
      st := &Store{c: conn}
      if err := st.migrate(); err != nil { return nil, err } // meta.schema_version, additive
      return st, nil
  }
  ```
  (Txns: `conn.Begin()` → `tx.Commit()/Rollback()`; stmts: `conn.Prepare` → `Bind*`/`Step`.)
  `001_schema.sql`: tables per `[[architecture/mnemonic-memory-core]]` + `obs_fts` external-content with `AFTER INSERT` / `AFTER UPDATE OF title, content` (delete-then-reinsert into FTS) / `AFTER DELETE` triggers (the gotcha).
- [ ] **Step 4: Run, expect PASS**: `go test ./internal/store/ -v`
- [ ] **Step 5: Commit** `feat(mnemonic): store — ncruces driver, schema 001, FTS triggers, WAL`

### Task 2: Floor embedder + BLOB vector layer (brute force)

**Files:**
- Create: `mnemonic/internal/intelligence/embedder/embedder.go`, `hash.go`, `onnx_stub.go`, `mnemonic/internal/store/vector.go`
- Test: `mnemonic/internal/intelligence/embedder/hash_test.go`, `mnemonic/internal/store/vector_test.go`

**Interfaces:**
- Consumes: `store.Store` (Task 1) for the vector table access
- Produces: `embedder.Embedder` interface `{ Embed(ctx, text) (Vector, error); Model() string; Dim() int }`; `embedder.New(cfg) (Embedder, bool /*degraded*/, error)` factory (bounded one retry → floor); `store.Vector` BLOB codec (float32 little-endian, dim-prefixed); `store.BruteForceCosine(ctx, tx, q Vector, topK int, where string) ([]Ranked, error)`
- Seam: `Embedder` — three adapters across the plan: Hash (always), Onnx (Task 8, tag `onnx`), Api (Task 7/8). Seam justified now.
- Deletion test: `memory/save.go` + `recall.go` would switch on concrete embedder types — adding a tier means editing domain code.
- Adapters: Hash + (stub for the absent ONNX state) = 2 present, 1 future — meets ≥2.

**SATISFIES:** `R3 embedder` (floor half), `R2 vectors in-file` (BLOB source of truth + brute-force floor)

- [ ] **Step 1: Failing tests** — `hash_test.go`: golden vector (fixed sentence ⇒ deterministic 64-dim L2-normalized vector, committed golden); `vector_test.go`: BLOB encode/decode round-trip; `BruteForceCosine` ranks a seeded 100-vector set in the expected top-3 order; **dim-strict**: a 384-dim query against a 64-dim store ⇒ error, never a silent rescale.
- [ ] **Step 2: Run, expect FAIL**: `go test ./internal/store/ ./internal/intelligence/...` → build failure.
- [ ] **Step 3: Implement** — `hash.go` (FNV-1a over word pieces → 64 buckets → L2-normalize, mirroring the old package's proven floor); `onnx_stub.go` (`//go:build !onnx`: reports absent state → factory degrades to hash); `vector.go` (codec + `SELECT id, embedding … ORDER BY cosine DESC LIMIT ?` computed Go-side over the BLOBs, dim-checked first).
- [ ] **Step 4: Run, expect PASS**: `go test ./internal/store/ ./internal/intelligence/... -v`
- [ ] **Step 5: Commit** `feat(mnemonic): hash-64 floor embedder + BLOB vector layer (brute force)`
### Task 3: Memory domain — save, sessions, delete/pin (tracer start)

**Files:**
- Create: `mnemonic/internal/memory/save.go`, `sessions.go`, `handler.go`
- Test: `mnemonic/internal/memory/save_test.go`, `mnemonic/internal/memory/handler_test.go`

**Interfaces:**
- Consumes: `store.Store` (T1), `embedder.Embedder` (T2)
- Produces: `Handler` struct — first 7 methods: `Save`, `Get`, `Delete`, `Pin`, `SessionStart`, `SessionEnd`, `SessionSummary` (Search/Context/Inject arrive T4; Stats/Doctor T7 — the Handler grows, it is never rewritten)
- Seam: the Handler is **the** seam ADR-0007 names — CLI (T5) and MCP (T6) are its two adapters
- Deletion test: both adapters re-implement upsert/dedup/strip — parity (T5) is structurally unenforceable, logic duplicated twice
- Adapters: CLI + MCP = 2 (T5/T6), seam pre-justified by ADR-0007

**SATISFIES:** `R4 memory model` — `T4 upsert`, `T12 session contract`, `T13 private strip`

- [ ] **Step 1: Failing tests** — `save_test.go`: save ⇒ base row + FTS match + `embedding` BLOB (dim-checked) all in one tx; re-save same `topic_key`+`scope` ⇒ `revision_count++`, still one row; exact-duplicate content, no `topic_key` ⇒ `duplicate_count++`; `<private>secret</private>` absent from stored content; save without live `session_id` ⇒ `no_session`. `handler_test.go`: session start → end(summary) → `SessionSummary` returns it; `Delete` ⇒ row marked deleted AND FTS no longer matches (trigger, end-to-end); `Pin` ⇒ `pinned=1`.
- [ ] **Step 2: Run, expect FAIL**: `go test ./internal/memory/` → build failure.
- [ ] **Step 3: Implement** — `save.go`: strip → dedup hash → upsert key → embed (via `embedder.New`'s degrade result — the degraded flag is returned to the caller for the envelope) → single `store.Begin()` tx. `sessions.go`: lifecycle + six-section summary field. `handler.go`: struct holding `*store.Store` + embedder cfg; the 7 methods as thin domain ops (no envelope — envelopes are T5).
- [ ] **Step 4: Run, expect PASS**: `go test ./internal/memory/ -v`
- [ ] **Step 5: Commit** `feat(mnemonic): memory save + sessions + Handler (tracer start)`

### Task 4: Recall — search, context, inject (tracer complete)

**Files:**
- Create: `mnemonic/internal/memory/recall.go` (legs + fusion), `context.go` (context/inject rendering)
- Test: `mnemonic/internal/memory/recall_test.go`

**Interfaces:**
- Consumes: `Handler` (T3), `store.BruteForceCosine` (T2), `embedder` (T2)
- Produces: `Handler.Search` (scope filter → legs → RRF → pinned-first → trail row → result with per-leg provenance + `degraded`), `Handler.Context` (top-K ⇒ ordered title/snippet/provenance list), `Handler.Inject` (deterministic prompt-block render, token-budgeted via the `max_tokens` param per R6 — the golden string T5's parity rests on); internal `leg` interface `{ Name() string; Run(ctx, tx, q) ([]Ranked, error) }`
- Seam: `leg` — adapters: FTS, brute-force (T2), vec (T9) = 3; fusion is testable with mock legs
- Deletion test: legs inlined in Search ⇒ RRF/fusion untestable in isolation, each new leg edits Search

**SATISFIES:** `R5 recall pipeline` — `T3 degradation` (vector leg absent ⇒ FTS-only + flag), `T11 scope isolation`; `T2` semantic ordering once T8's ONNX tier lands

- [ ] **Step 1: Failing tests** — seeded 50-observation store: (a) keyword query ⇒ FTS-leg provenance, correct top-K; (b) FTS query syntax injection: query `foo" OR content : *` ⇒ balanced-quoted phrase, no error, sane results (threat row 1); (c) both legs ⇒ RRF order (k=60) differs from either leg alone — fusion demonstrably runs; (d) embedder-forced-degraded ⇒ FTS-only + `degraded: true` (never empty); (e) project A's `project`-scoped obs absent from project B's search, present with `all_projects`; (f) each search writes one trail row; (g) `Context` list order + `Inject` golden block (byte-exact).
- [ ] **Step 2: Run, expect FAIL**: `go test ./internal/memory/ -run Recall` → build failure.
- [ ] **Step 3: Implement** — FTS leg (parameterized MATCH, sanitized phrase), vector leg (embed query with the save-time model stamp — dim-strict, else leg drops out + degraded), RRF `Σ 1/(60+rank_i)`, pinned obs ordered first, trail insert in the same read path (WAL allows concurrent read txs).
- [ ] **Step 4: Run, expect PASS**: `go test ./internal/memory/ -v`
- [ ] **Step 5: Commit** `feat(mnemonic): recall — FTS + vector legs, RRF, trail (tracer complete)`

### Task 5: Transport — envelope + CLI adapter + dispatch

**Files:**
- Create: `mnemonic/internal/transport/envelope.go`, `cli.go`, `mnemonic/cmd/mnemonic/main.go`, `mnemonic/cmd/mnemonic/memory.go`
- Test: `mnemonic/internal/transport/cli_test.go`

**Interfaces:**
- Consumes: `Handler` (T3/T4 methods)
- Produces: `Envelope { ok bool; code string; data any; degraded bool; elapsed_ms int64 }` + the stable `code` table (one per failure row in §6: `store_unavailable`, `store_busy`, `store_full`, `schema_mismatch`, `no_session`, `model_missing`, …); `transport.RunCLI(argv []string, h *memory.Handler, out io.Writer) int`; global flags per R7: `--pretty` (human formatting), `--all` (open every file in `$MNEMONIC_HOME`), `--project`
- Seam: same Handler seam (T3) — this is adapter 1 of 2
- Deletion test: each command re-renders envelopes/exit codes inline ⇒ the MCP adapter (T6) re-implements them ⇒ drift (T5 broken)

**SATISFIES:** `R7 CLI` — `T14 exit codes`

- [ ] **Step 1: Failing tests** — `cli_test.go` (seeded store): `memory search "x"` ⇒ exit 0, stdout byte-golden `{"ok":true,…"elapsed_ms":N}`; unknown flag ⇒ exit 2, usage on stderr, **no** envelope on stdout; save without session ⇒ exit 1 + `{"ok":false,"code":"no_session",…}`; envelope JSON key order byte-stable across two runs (same input); degraded search ⇒ `degraded:true` present, not a separate channel.
- [ ] **Step 2: Run, expect FAIL**: `go test ./internal/transport/` → build failure.
- [ ] **Step 3: Implement** — `envelope.go` (type + `ErrorFrom(err)` code mapping from the §6 table); `cli.go` (flag sets per verb → Handler call → envelope render, `elapsed` measured around the Handler call only); `main.go` (top-level: `memory …` now; `init`/`doctor` slots wired in T7; unknown ⇒ exit 2); `memory.go` (verb → `RunCLI` mapping).
- [ ] **Step 4: Run, expect PASS**: `go test ./internal/transport/ -v && go build ./cmd/mnemonic`
- [ ] **Step 5: Commit** `feat(mnemonic): envelope + CLI adapter + top-level dispatch (exit codes)`

### Task 6: Transport — MCP stdio server + byte-parity goldens

**Files:**
- Create: `mnemonic/internal/transport/mcp.go`
- Test: `mnemonic/internal/transport/parity_test.go`, `mnemonic/internal/transport/mcp_test.go`

**Interfaces:**
- Consumes: `Handler` (T3/T4), envelope renderer (T5)
- Produces: mcp-go stdio server **`mnemonic-memory`**; 10 tools now (`mem_save`, `mem_search`, `mem_context`, `mem_get`, `mem_inject`, `mem_session_start`, `mem_session_end`, `mem_session_summary`, `mem_delete`, `mem_pin`) — T7 adds `mem_stats` + `doctor` for 12/12 and completes the parity matrix; `dispatch(tool string, args map[string]any) (Envelope, error)` — the single function the MCP server wraps, kept testable without stdio plumbing
- Seam: same Handler seam — adapter 2 of 2
- Deletion test: MCP tool handlers build responses independently ⇒ the parity golden set (T5) can no longer pass by construction

**SATISFIES:** `R6 MCP` (10/12 surface), `R8 single handler` — `T5 parity`

- [ ] **Step 1: Failing tests** — `parity_test.go`: for all 10 methods, identical input via `RunCLI` and via `dispatch` ⇒ **byte-identical** envelope JSON (the T5 criterion); `mcp_test.go`: tool list = 10 names; bad arg ⇒ MCP error carrying the stable `code`; locked store (second writer holding the write tx past `busy_timeout`) ⇒ `store_busy` envelope, exit-1 equivalent.
- [ ] **Step 2: Run, expect FAIL**: `go test ./internal/transport/ -run 'Parity|MCP'` → build failure.
- [ ] **Step 3: Implement** — `mcp.go`: tool schemas (descriptions = one line each, typed args), `dispatch` (arg decode → Handler call → envelope, same renderer as CLI), stdio server registration, `memory mcp` verb wiring in `memory.go` (T5).
- [ ] **Step 4: Run, expect PASS**: `go test ./internal/transport/ -v`
- [ ] **Step 5: Commit** `feat(mnemonic): MCP stdio server (mnemonic-memory) + CLI/MCP byte-parity goldens`

### Task 7: init, doctor, config, project detection, export (parity 12/12)

**Files:**
- Create: `mnemonic/internal/config/config.go`, `mnemonic/internal/project/detect.go`, `mnemonic/internal/memory/stats.go`, `mnemonic/internal/memory/export.go`
- Modify: `mnemonic/cmd/mnemonic/main.go` (fill the T5 slots), `mnemonic/internal/transport/mcp.go` (+`mem_stats`, `doctor`)
- Test: `config_test.go`, `detect_test.go`, `stats_test.go`, `export_test.go`; `parity_test.go` matrix → 12/12

**Interfaces:**
- Consumes: `Handler`, `store`, `embedder`
- Produces: `project.Detect(flag, env) (slug, err)` — `--project` > `MNEMONIC_PROJECT` > git toplevel > cwd basename, slug-sanitized (no `/`, `..`); `config.Load()` — `<MNEMONIC_HOME>/config.toml` + env overlay; `mnemonic init` (data + models dirs; under tag `onnx`: download + sha256/size verify against `models/manifest.json`, else dirs only — **no network by default**); `Handler.Stats` / `Handler.Doctor` (schema_version, active embedder tier/model/dim, vec state, `integrity_check`, file sizes); `memory export --out DIR` (markdown + JSON, confined under the allowed root — threat row 3)
- Seam: same Handler seam; detection/config are leaf services (no adapters needed — deletion test: slug logic inlined in store.Open and the adapters ⇒ three divergent project definitions)

**SATISFIES:** `R1` (slug + data dir), `T8 doctor`, `T9 offline` (init is the only network path, and only under the tag), threat rows 1–3 exercised; completes `R6` 12/12

- [ ] **Step 1: Failing tests** — detection: all 4 precedence cases + `a/b/../c` → `c` (and `foo/bar` → `foo-bar`); config: env overrides toml, toml overrides default; doctor: seeded store ⇒ every §T8 field present + `integrity_check=ok`; init (`!onnx`): creates dirs, reads no manifest (assert via a missing-manifest fixture — must not fail); export: md + JSON under `--out`, `--out ..` rejected with a usage exit 2. Parity matrix: 12/12 byte-identical.
- [ ] **Step 2: Run, expect FAIL**: `go test ./internal/... ./cmd/... -run 'Detect|Config|Doctor|Init|Export|Parity'` → build failure.
- [ ] **Step 3: Implement** — per the Produces list; `init`'s download path behind `//go:build onnx` with the verified-manifest check before the tier is enabled.
- [ ] **Step 4: Run, expect PASS**: `go test ./... -v`
- [ ] **Step 5: Commit** `feat(mnemonic): init + doctor + config + project detection + export (parity 12/12)`

### Task 8: ONNX tier (hugot GoMLX, tag `onnx`) + API embedder — spike-gated

**Files:**
- Create: `mnemonic/internal/intelligence/embedder/onnx.go` (`//go:build onnx`), `api.go`, `onnx_golden_test.go`
- Modify: `mnemonic/internal/intelligence/embedder/embedder.go` (factory wires `local` tier when the tag is on + model present)
- Test: `api_test.go` (httptest server)

**SPIKE GATE: RESOLVED 2026-11-23 (PASS)** — `onnx/model.onnx` (90,405,214 B) ran
end-to-end on the hugot GoMLX backend: dim 384, cosine 1.000000 vs the independent
ORT reference (the ORT-tagged run doubles as the golden source for the committed
reference vector), p50 81 ms/sentence non-simd (≤ 3 s target), semantic sanity
0.584 vs 0.051. Falsifier (a) not triggered — the `onnx` tag ships. The
`GOEXPERIMENT=simd` rebuild is a follow-up measurement, not a gate.

**Interfaces:**
- Consumes: manifest (T7), factory (T2)
- Produces: GoMLX session loading the model from `<MNEMONIC_HOME>/models/` (path from the verified manifest), mean-pool + L2-normalize (the model's documented recipe), 384-dim; `api.go` — OpenAI-compatible `/embeddings` client (`MNEMONIC_API_BASE` default `http://127.0.0.1:11434/v1`, `_MODEL`, `_KEY`, 5 s timeout) — any error ⇒ factory degrades (R9 failure row)
- Seam: `Embedder` (T2) — this task lands adapter 3 (API) and real body of adapter 2
- Deletion test: pooling/normalize inline in save.go ⇒ the recipe is untestable in isolation, and the stub/real swap touches domain code

**SATISFIES:** `R3` (happy path), `R9` (API down ⇒ floor + `degraded`), `T2` (semantic ordering live once the tag is on)

- [ ] **Step 1: Failing tests** — `onnx_golden_test.go` (skips when the model is absent): one fixed sentence ⇒ committed golden vector, cosine ≥ 0.99, dim 384, model stamp `all-MiniLM-L6-v2`; `api_test.go`: httptest ⇒ 384-dim passthrough + model stamp from the response; server down ⇒ `New` returns floor + degraded, never an error that blocks save.
- [ ] **Step 2: Run, expect FAIL**: `go test -tags onnx ./internal/intelligence/...` → build failure (onnx.go absent).
- [ ] **Step 3: Implement** — `onnx.go` (hugot GoMLX session, tokenizers.json 466 KB bundled via `embed` or fetched in the manifest batch, 256-word-piece truncation per the model card, mean-pool + L2); `api.go`.
- [ ] **Step 4: Run, expect PASS**: `go test -tags onnx ./internal/intelligence/... -v` (+ latency line logged for the spike record)
- [ ] **Step 5: Commit** `feat(mnemonic): ONNX embedder tier (hugot GoMLX) + API embedder (spike-gated)`

### Task 9: vec1 KNN leg (tag `sqlitevec`) + 10k p50 bench

**Files:**
- Create: `mnemonic/internal/store/vector_vec.go` (`//go:build sqlitevec`), `vector_vec_stub.go` (`//go:build !sqlitevec`), `mnemonic/internal/store/vector_bench_test.go`
- Modify: `mnemonic/internal/memory/recall.go` (leg registry: under the tag the vec leg replaces brute; the stub keeps brute)
- Test: same bench file — p50 assertions + KNN↔brute equivalence

**SPIKE GATE: RESOLVED 2026-11-23 (PASS)** — `vec1.Register` + `fts5.Register` in one
ncruces v0.38 connection, FTS MATCH + vec1 KNN + interleaved writes verified (findings
§ Spike Results). Falsifier (b) not triggered.

**Interfaces:**
- Consumes: `store.Open` (T1), BLOB codec (T2), `leg` seam (T4)
- Produces: `vec1.Register(conn)` inside the tagged `store.Open` path; `mem_vec` DDL lifecycle — lazy `CREATE VIRTUAL TABLE mem_vec USING vec1(embedding)` when the embed dim first stabilizes, `INSERT INTO mem_vec(cmd,arg) VALUES('rebuild','{"index":"flat","distance":"cos"}')`; on dim change: drop + recreate + reinsert from the BLOB source of truth (never a mixed-dim table); `store.VecKNN(q Vector, k int) ([]Ranked, error)` via the table-valued query `SELECT rowid FROM mem_vec(?, '{"K":?}') ORDER BY distance`
- Seam: same `leg` seam (T4) — adapter 3 of 3 lands here
- Deletion test: KNN inlined in the brute path ⇒ the tag split breaks and the DDL lifecycle scatters across recall code

**SATISFIES:** `R2` (happy + speed target), `T7` (p50)

- [ ] **Step 1: Failing tests** — `vector_bench_test.go`: 10k obs × 384-d seeded — brute p50 < 100 ms and vec p50 < 50 ms (skipped on `-short`); equivalence: with the flat+cos model (exact distances, no trained quantizer) vec top-10 ids = brute top-10 ids on a seeded set (ties aside); dim change (seed 64-d, switch to 384-d) ⇒ rebuild: `mem_vec` row count = obs count, zero stale-dim rows.
- [ ] **Step 2: Run, expect FAIL**: `go test -tags sqlitevec ./internal/store/` → build failure (vec file absent).
- [ ] **Step 3: Implement** — registration in the tagged Open, the DDL lifecycle as one function (`EnsureVec1(table, dim)`), KNN, leg wiring. (Future: a trained IVFADC model via `vec1_train` for large-scale speed — not v1; flat+cos is exact and sufficient at memory scale.)
- [ ] **Step 4: Run, expect PASS**: `go test -tags sqlitevec ./internal/store/ -v` (+ `-bench KNN -benchtime 20x` for the p50 record)
- [ ] **Step 5: Commit** `feat(mnemonic): vec1 KNN leg (sqlitevec) + 10k p50 bench`

## Open Questions

| Question | Owner | Due |
|---|---|---|
| KB corpus as a future in-file tool (3rd corpus like `code`/`web`: pages → rows → FTS + `kb_vec`, provenance `kb:<path>`) — shape it, or keep the vault out of the store? | programmer | before the `kb` tool's intake |
| ANN mechanism at the `code` tool's 10⁴–10⁵ chunk scale — vec1 flat+cos (ADR-0015) is the default; confirm T7-class speed there before a trained IVFADC model or a graph index (vec1's roadmap) | code-tool change | its spike |
| `model_O4.onnx` (45.2 MiB, int8) vs full `model.onnx` (90.4 MiB) quality delta — when (if ever) is the quantized variant the manifest default? | programmer | after the Task 8 spike (measured, not guessed) |
| Per-tool MCP servers (ADR-0010) — do future tools also want a unified `mnemonic` umbrella server for convenience, or strictly per-tool? | programmer | first future tool's intake |
| ncruces per-connection WASM memory at the code tool's larger file scale — re-measure, not inherited from memory-core | code-tool change | its spike |

## Review Summary

_Goldfish self-review, 2026-11-23 (Phase 6)._

**Reconciliation vs briefing (contradictions quoted; R21 additions already flagged in §Key Decisions):**
- **R1/R2 — driver:** briefing named `modernc.org/sqlite`; plan uses `ncruces/go-sqlite3` v0.38 (modernc cannot host a vec extension — user-gated refinement, ADR-0015). All R1 PRAGMAs/WAL semantics preserved.
- **R2 — vec mechanism:** briefing named "the pure-Go `sqlite-vec` port (`asg017/sqlite-vec-go`)"; the official ncruces binding for it proved **unpairable** with any released ncruces (spike 2, 2026-11-23: wasm/host-table drift across v0.15.0/v0.17.1/v0.38.0). Plan uses ncruces' official `ext/vec1` instead. The briefing's three-layer contract (BLOB SoT / in-file ANN / brute-force floor) is preserved exactly.
- **R3 — ONNX runtime:** briefing named `unum-cloud/onnx` (HTTP 404 — findings [10]); plan uses `knights-analytics/hugot` GoMLX backend (spike 1 PASS: dim 384, cosine 1.000000 vs ORT, 81 ms/sentence p50 non-simd).
- **R3 — interface shape:** briefing's `Embed(ctx, texts) ([][]float32, error)` (batch) vs Task 2's `Embed(ctx, text) (Vector, error)` (single). **Flagged:** the gated blueprint signature stands; a batch variant is a one-line add when bulk re-embed (a background job) needs it.
- **R5 — boosts:** briefing: "boosts: recency, importance, pinned". Plan: pinned-first ordering, recency as tie-break; `importance` is static 0.5 in v1 (LLM curator deferred) so its boost degenerates to a constant — the deviation is inert by design.
- **R6 — `mem_inject` `max_tokens`:** was missing from Task 4; added during this self-review (closed).
- **R7 — `--pretty` / `--all`:** were missing from Task 5; added during this self-review (closed).
- **R9 — download timing:** briefing: "at `init` (or first `save`, with a `--no-models` escape)"; plan: `init` only — first-save download would put network on the save path, contradicting criterion 9's "model pre-cached locally". Intentional thinning.
- **Toolchain:** briefing era assumed Go 1.26; hugot v0.8.6 requires go ≥ 1.27 (observed toolchain auto-switch on a 1.26.6 host).

**Internal consistency:** Tasks 1–9 cover every Must-Have truth T1–T14 and every validation row R1–R9 (SATISFIES lines reference the numbered truths); the 12-method Handler surface maps 1:1 to the 12 MCP tools (T3: 7, T4: +3, T7: +2); acceptance.feature carries 9 `### Requirement:` (one per briefing R) × 3 scenarios + 18 Gates, lint-clean, dry-run 27 scenarios / 105 steps.

**Spike evidence (both PASS, falsifiers not triggered):** findings § Spike Results — (1) MiniLM/GoMLX end-to-end; (2) `ext/vec1` + FTS5 in one ncruces connection. ADR-0015 amended pre-commit to record the vec1 mechanism.

**Residual risks:** vec1 is v0.8 with an open roadmap (contained by the brute-force floor + version pin); GoMLX simd measurement deferred to a follow-up (81 ms non-simd has 37× headroom over the 3 s target); ORT escape tag wants ≥ 1.23 (1.22 falls back with a warning, verified working).

**Verdict:** approved for execution — hand off to `skillgrid:slicing` → `skillgrid:execute`; spec-zone commit precedes any code.
