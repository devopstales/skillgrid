# Mnemonic v2 — Memory Function (Implementation Blueprint: steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T3 — new public MCP tool contracts + dual-dialect storage + a data-migration path. Config `rules.tiers.default` is T2; escalated to T3 per Change Classification (`high-risk`). QA floor: L4 (full suite + build + coverage + go vet; mutation/security advisory per config).

**Build shape:** Tracer thread — the integration risk is the dual-dialect `Store` + LLM seams. A thin end-to-end `mem_save → SQLite store → MCP tool → JSON result` path works first (Task 1's door check), then thickens (dedup, relations, sessions, lookup, sync, admin).

**Classification:** high-risk (new trust boundary: agent-supplied content into the store; public API: the `mem_*` MCP contract; data migration: v1 → v2 seed import).

**Goal:** Build the memory function of mnemonic v2 — the raw, sync-ready source-of-truth fact store (observations, sessions, relations, prompts, dedup, lookup, sync) behind a dialect-agnostic `Store` (SQLite | Postgres) and an MCP `mem_*` tool surface — so agents save atomic durable facts and the llm-wiki function consumes them via a save event.

**Pre-execution checklist (T0 — repo state the plan presumes; none of it is a task, all of it must be true before Task 1's code commit):**
- [ ] `acceptance.feature` (this folder) written per `skillgrid:acceptance-test-authoring` from the spec zone + the blueprint's SATISFIES lines, and committed.
- [ ] ADR-0040 (module extraction) + ADR-0041 (Postgres driver) written under `.skillgrid/artifacts/` and added to the ASSUMPTIONS.md index (next numbers after 0039).
- [ ] `mnemonic/` renamed to `mnemonic_back/` (git mv + `go.work` `use` block + Taskfile dual-build refs) — the blueprint's "the old module is `mnemonic_back/`" statement becomes true.
- [ ] `state.yaml` `pipeline.current_change` repointed to `2026-10-31-mnemonic-v2-memory` (serial development: one change at a time).
- [ ] Spec-zone amendments below (S-3 wording, the spec `002-schema` §2.2.3 usage-column note, `capture_prompt` default) committed **before** any `mnemonic-v2/` code commit (pre-commit zone guard).

**Architecture:** A fresh single Go module `mnemonic-v2/` (module path `github.com/devopstales/skillgrid/mnemonic-v2`, Go 1.26, cgo-free) implements the spec zone at `specs/00-foundation/` + `specs/01-memory/`. `internal/core/store` owns the dialect split (SQLite default = local truth; Postgres = replication target; FTS5 ↔ tsvector; vector column activated). `internal/memory` is the domain (save pipeline, relations, lookup, sync) and calls only the `Store` interface + `LLM`/`Embedder` seams — it never names `*sql.DB`. `internal/intelligence` owns the LLM-first seams with explicit deterministic degradation (`degraded: true`, INV-3). `internal/transport/mcp` is a thin adapter over one shared handler set, so CLI and MCP never diverge.

**Tech Stack:** Go 1.26 (go.work already pins 1.26.0); `modernc.org/sqlite` (pure-Go, cgo-free, FTS5 trigram; blank-import `modernc.org/sqlite/vec` for the vec0 vector leg) — both already in the v1 module; `mark3labs/mcp-go v1.1.1` (stdio server, v1-proven); `gopkg.in/yaml.v3` (config, ADR-0008 precedent); `database/sql` + `lib/pq` for Postgres (new dependency — ADR required, see Task 18); Engram's pure-Go HNSW/IVF approach as the reference if the vec0 leg is dropped.

**Spec:** `specs/00-foundation/000-architecture.md` … `006-config.md` + `specs/01-memory/000-overview.md` … `006-sync.md` (this folder). Product *why*: `.skillgrid/artifacts/00-prd.md`.

**Findings:** none — this change ran no research/prototype/sketch phase. The reference-codebase facts cited in tasks (Engram FTS DDL/weights at `store.go:3125-3171, 424-431`; sync-id minting at `store.go:13703`; chunk id at `internal/cloud/chunkcodec/chunkcodec.go:13`; v1 `DedupDecision` at `mnemonic_back/internal/memory/types.go:57-90`; v1 durability formula at `mnemonic_back/internal/memory/dream.go:382-400`; autosync lease at Engram `internal/cloud/autosync/manager.go:268,532`) were verified against the code on 2026-11-01 and are inlined in the relevant tasks.

## Terms

Pointers only — canonical terms live in `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md`.

- **observation** — the unit of memory (one row in `observations`; spec `002-schema` §2.2). Adopted from Engram's flat model.
- **save pipeline** — the 10-step contract `mem_save` runs (`01-memory/001-observations` §2). New term, introduced by this change's spec zone.
- **save event** — the in-process bus event emitted after a committed save that the llm-wiki subscribes to (`004-transport-mcp` §5). New term.
- **Store** — the dialect-agnostic data-access interface (`001-storage` §7). New term, replaces v1's store concrete type.
- **degraded** — the explicit flag set when an LLM/embedder seam ran its deterministic floor (INV-3). Already in v1's seam vocabulary; restated as an envelope field in v2.
- **mutation journal** — `sync_mutations`, the local append-only log of replicated changes (`002-schema` §3.1).
- **wiki-first / memory-fallback** — the retrieval routing rule (OPEN-12): understanding goes to the wiki; `mem_lookup` is the narrow raw-fact fallback.

## Must-Haves (goal-backward verification)

Derived from the spec zone. The verifier checks the result against this list, not just that tasks exist.

**Truths** (observable behaviors that must hold):

- T1: `mnemonic mem save` (and `mem_save` MCP tool) with a live session persists an observation; the result carries `id`, `sync_id` (format `obs-` + 16 hex), `revision_count`, `dedup: {verdict, reason, merged_into_sync_id}`, and the observation is retrievable via `mem_get_observation` by both numeric id and sync_id.
- T2: Saving twice with identical redacted/truncated content (same project/scope/type/title) returns the same id, bumps `duplicate_count` and `last_seen_at`, and writes no new row (hash floor, `degraded` when no LLM seam).
- T3: Saving with the same `topic_key` (same project+scope) updates the existing row in place: `revision_count` increments and a pre-state snapshot exists in `observation_versions`; `mem_get_observation --include-history` pages them 50/page.
- T4: With an LLM seam attached, a semantically-duplicate save returns `dedup.verdict=update` with `reason="llm"` and merges into the target; a zero-value decision (no LLM) proceeds via the hash floor with `reason="hash"`.
- T5: `mem_search` returns compact rows (id, title, type, 300-char preview + `truncated` flag) ranked BM25(title 5.0, content 1.0, topic_key 3.0) × boosts (pinned +0.10, recency 0.06·1/(1+days/30), stability 0.04·(rev+dups)/(rev+dups+4)); a <3-rune term falls back to LIKE; soft-deleted rows are excluded.
- T6: `mem_lookup` by exact `topic_key` returns the single upserted row (or none); by id/sync_id returns the full row; by phrase returns ≤5 FTS rows with no boosts. It returns nothing for a topic that is absent — it does not widen.
- T7: After `mem_save`, conflict candidates (same project+scope, OR-FTS title pre-filter + relevance gate: shared topic_key OR ≥2 significant shared title terms) are materialized as `pending` `memory_relations` and returned as `candidates[]` with `judgment_required: true`; `mem_judge` sets `judgment_status=judged` + provenance (`marked_by_actor/kind/model`); `mem_judge_many` batches 1–32 in one queued write.
- T8: `mem_compare` creates a judged relation directly (idempotent per pair); `mem_consolidate` preview returns `plan.sources/survivor/project` + `Fingerprint` with no writes; confirm with a stale fingerprint returns `fingerprint_mismatch` + a fresh plan; confirmed merge soft-deletes sources with `superseded_by_sync_id` set and the survivor carries a `consolidated_from` footer.
- T9: `mem_session_start` registers a session (byte-exact id, `runtime_lease_expires_at` = now+30min, local-only); ending an ended session returns 409 `session_already_ended`; `mem_session_finalize` atomically writes the six-section `session_summary` observation, sets `ended_at`, enqueues sync mutations, and records `session_finalizations`; a retry returns `already_finalized: true`.
- T10: `mem_save` with `capture_prompt` absent (→ default true) or explicit true stores the current turn's prompt in `user_prompts` (redacted, truncated); explicit false stores nothing; `mem_search_prompts` finds the captured prompt.
- T23: a retrieved observation accumulates `retrieval_usage` (the trail writer bumps it); an observation retrieved 10 times scores higher under the importance formula than an identical never-retrieved one (the wiki's graduate-rule input — spec `002-schema` §2.2.3).
- T24: `mem_review mark_reviewed` resets `review_after = now + type-months` (an overdue row returns to `active`); `mem_review list` returns exactly the `needs_review` rows.
- T11: Every `mem_search` / `mem_lookup` / vector search records a retrieval trail (query, route `fts`/`vector`/`rrf`/`lookup`, filters, returned ids, `degraded`); `mnemonic trail recent|show` inspects them.
- T12: With no LLM/embedder configured, every semantic op runs its deterministic floor and the response envelope carries `degraded: true` + reason — never a silent floor. With a failing seam (error, not nil), one bounded retry then the same floor.
- T13: `Store.Search` / `Store.VectorSearch` return the same Go result shape on SQLite and Postgres; both dialects run the full migration set from an empty DB (regression-tested).
- T14: Local SQLite is one file per project (`<data_dir>/<project>.sqlite`); a Postgres connection is scoped to `schema_<project>` and fails fast (no fallback to `public`) when the project schema cannot be resolved. A remote failure never corrupts local (INV-4).
- T15: `sync push` inserts pending mutations (≤100/batch, ≤8 MiB) into `schema_<project>.mutations` in one transaction and acks `last_acked_seq`; `sync pull` applies remote mutations through the same store methods (INV-4), deferring missing-FK mutations to `sync_apply_deferred` (5-retry cap then dead); a late upsert after a delete cannot resurrect the row (tombstone floors).
- T16: `sync export` writes `manifest.json` + `chunks/*.jsonl.gz` with chunk id = first 8 hex of SHA-256 of the canonical payload; `sync import` loads the archive dialect-neutrally.
- T17: The autosync manager (when `sync.auto=true`) debounces 500 ms, holds a SQLite-backed lease (one worker at a time), and runs network I/O off the SQLite write path (the `onWrite` hook is a buffered-1 channel).
- T18: Replaying `mem_save` with the same `OperationID` returns the committed id (one dedup/upsert/sync run); a changed payload returns 409 `operation_conflict`; a deleted target returns 410 `operation_expired`.
- T19: `mem_update` / `mem_delete` with a wrong `expected_project` fail the transaction (no partial write); soft delete sets `deleted_at` and hides the row from search/context/timeline while the id remains for audit; `--hard` deletes physically.
- T20: `mem_save` emits exactly one `SaveEvent{SyncID, Project, Scope, Type, TopicKey, Entity, Importance, Degraded}` after commit; the memory package does not import the wiki package; a consumer failure never fails the save.
- T21 (backstop): two processes writing to the same project file concurrently — one holds the store open while the other runs `mem_save` — both complete without corruption (WAL + busy timeout + retry backoff). *Held-out test: Task 20 Step 1 (`TestConcurrentProcessWrites`), plus the real two-process run in `mem doctor` integration.*
- T22: `mnemonic doctor` reports the resolved config, FTS present, vector index present, embed round-trip, and sets `degraded` on any gap; `--strict` exits non-zero on a gap.

**Artifacts** (files that must exist with real implementation, not stubs):

- `mnemonic-v2/go.mod` — fresh module `github.com/devopstales/skillgrid/mnemonic-v2`, go 1.26.
- `mnemonic-v2/internal/core/store/store.go` — the `Store` interface (full method set per spec `001-storage` §7) + `SQLiteStore` implementation.
- `mnemonic-v2/internal/core/store/sqlite.go` — dialect rendering: `migrations/*.sql` set, FTS5 trigram DDL + triggers, WAL open, refcounted handle pool.
- `mnemonic-v2/internal/core/store/postgres.go` — `PostgresStore` (render + `schema_<project>` scoping).
- `mnemonic-v2/internal/core/store/types.go` — the Go result/param types every Store method uses (Observation, SearchResult, Mutation, …).
- `mnemonic-v2/internal/core/syncid/syncid.go` — sync-id minting (obs-/rel-/ses-/prm- + 16 hex).
- `mnemonic-v2/internal/core/config/config.go` — YAML + env resolution (flag > env > file > default) per spec `006-config`.
- `mnemonic-v2/internal/core/project/project.go` — project detection (explicit → `MNEMONIC_PROJECT` → cwd → git binding → basename) + `--all` semantics.
- `mnemonic-v2/internal/intelligence/llm/llm.go` — `LLM` seam interface + `DedupDecision`/`Judgment`/`CandidateObservation` types.
- `mnemonic-v2/internal/intelligence/llm/openai_compat.go` — OpenAI-compatible provider (chat + embeddings over one base URL).
- `mnemonic-v2/internal/intelligence/embedder/embedder.go` — `Embedder` seam + OpenAI-compatible provider + null (degraded) adapter.
- `mnemonic-v2/internal/intelligence/degrade/degrade.go` — per-call degradation policy (≤1 retry, `degraded` flag, log line).
- `mnemonic-v2/internal/memory/pipeline.go` — the 10-step save pipeline (validation → redact → pre-filter → dedup → upsert → enqueue → commit → result → event).
- `mnemonic-v2/internal/memory/dedup.go` — LLM Classify path + hash floor + trust-but-verify candidate id.
- `mnemonic-v2/internal/memory/relations.go` — conflict-candidate detection, relevance gate, judge/judge_many/compare, consolidate.
- `mnemonic-v2/internal/memory/sessions.go` — session lifecycle, terminal semantics, resume, finalize, six-section summary.
- `mnemonic-v2/internal/memory/lookup.go` — context/search/get/lookup + RRF blend + trails.
- `mnemonic-v2/internal/memory/sync.go` — journal apply, tombstone floors, deferred-apply, chunk codec, export/import, PG transport, autosync manager.
- `mnemonic-v2/internal/memory/eventbus.go` — the in-process save-event bus.
- `mnemonic-v2/internal/transport/mcp/server.go` + `tools_mem.go` — MCP stdio server, agent/all profiles, all 26 `mem_*` tools.
- `mnemonic-v2/internal/transport/handler/handler.go` — the one shared handler set (MCP + CLI call it; never diverge).
- `mnemonic-v2/cmd/mnemonic/main.go` + `mem.go` + `sync.go` + `doctor.go` — CLI dispatch (stdlib flag, v1 pattern).
- `mnemonic-v2/internal/core/store/migrations/001_schema.sql` + `003_vectors.sql` + `retrieval_trails` (carried in `003`) — the full additive set (SQLite).
- `mnemonic-v2/internal/core/store/migrations_pg/*.sql` — the Postgres local-dialect render (Task 18).
- `mnemonic-v2/internal/wiki_teststub/` — the no-dependency consumer proof (Task 12).
- `mnemonic-v2/internal/core/store/store_test.go` + per-task `_test.go` files — tests for every truth above.
- `.skillgrid/artifacts/04-adr-0040-mnemonic-v2-module-extraction.md` — the module extraction decision.
- `.skillgrid/artifacts/04-adr-0041-mnemonic-v2-postgres-driver.md` — the `lib/pq` dependency decision.

**Key links** (critical connections that must work together):

- K1: `mem_save` MCP tool → `handler.Save` → `memory.Save` (pipeline) → `Store.AddObservation` → one SQLite transaction (observation + FTS via trigger + `observation_versions` + `sync_mutations`) → result → `eventbus.Publish(SaveEvent)`. A failure at any step rolls back the whole transaction.
- K2: `memory.Save`'s dedup step calls the `LLM` seam (`Classify`); the same `Classify` implementation is called by the pipeline in degraded mode (hash floor) — one code path, the `degraded` flag distinguishes them.
- K3: `mem_search` handler → `memory.Search` → `Store.Search` (FTS) + `Store.VectorSearch` (when armed) → RRF fusion in `memory/lookup.go`; the trail writer records the route (`fts`/`vector`/`rrf`) before returning.
- K4: `sync push` CLI and the autosync manager both call `sync.(*Manager).Cycle` → `transport.PushMutations` (PG SQL into `schema_<project>.mutations`) → `Store.ListPendingMutations`/ack. One code path, two callers.
- K5: The MCP `agent` profile exposes exactly the 20 `agent`-profile tools from spec `004-transport-mcp` §3's table (the 20 `| agent |` rows); `mem_delete`, `mem_stats`, `mem_timeline`, `mem_merge_projects`, `mem_review`, `mem_doctor` (the 6 `| admin |` rows) are `all`-profile only. Total surface = 26. A contract test pins both sets literally.
- K6: `cmd/mnemonic`'s `mem save` and the MCP `mem_save` tool produce byte-identical JSON (same handler, same result marshal) — a parity test asserts it.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):

- D1: **The `mem_*` MCP tool contract is the public surface for the plugins** (opencode-plugin / pi-plugin consume it; the two plugin changes reference it as their only interface). Renaming a tool, adding a required param, or changing a return shape after the plugins build breaks them. Tasks 14, 16.
- D2: **The `Store` interface shape** (`001-storage` §7) — code-index and llm-wiki later extend the same store (they add their tables to the same project DB, spec `002-schema` §4). Adding a method later is additive; removing or reshaping one is a contract break for every function that implements or calls it. Task 2.
- D3: **Postgres driver = `lib/pq`** (new dependency, AGENTS.md rule: no new deps without an ADR; ADR-0041). A driver change later means re-testing the whole PG dialect render. Task 18.
- D4: **v1 → v2 seed import** (`mnemonic migrate`) — v1 store files are *not* auto-migrated (different schema philosophy, spec `002-schema` §6); the export is a one-time seed and the v1 files stay put. Choosing a different cutover later means re-importing on every machine. Task 19.
- D5: **Fresh module `mnemonic-v2/`** (vs. in-place rewrite of `mnemonic/`) — the old module becomes `mnemonic_back/` during v2 development and the two coexist until v2 ships. Moving the v2 code back into `mnemonic/` later means re-baselining the module path everywhere (go.work, Taskfile, install, plugins). Task 1.

## Global Constraints

Every task's requirements implicitly include this section.

- **Go 1.22+ minimum to build** (ADR-0031); the go.work floor is 1.26.0, so `mnemonic-v2/go.mod` declares `go 1.26.0`.
- **No new dependencies without an ADR** (per AGENTS.md § Rules): the allowed dependency set for this change is `modernc.org/sqlite` (incl. `/vec`), `mark3labs/mcp-go`, `gopkg.in/yaml.v3` (all already in v1's go.mod — the `yaml` package is the ADR-0008 precedent), plus `lib/pq` (new, ADR-0041). Anything else needs a new ADR before it lands.
- **cgo-free** (v1 VERIFIED stack: `skillgrid doctor` reports `cgo: free`; the pure-Go `modernc.org/sqlite` is the driver of record).
- **Conventional commits only; no AI-attribution trailers** (commit-msg hook enforces).
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard enforces): this blueprint + `acceptance.feature` + `tasks.md` are committed before any `mnemonic-v2/` code commit.
- **One change at a time, no parallel branches** (serial development). A commit that closes a Backlog task names that task ID in the `[skillgrid-context]` block (ADR-0013).
- **Trivy is advisory-only** — findings reported, never blocking the QA gate.
- **Additive migrations only** (spec `002-schema` §1): new tables/columns; never drop in a minor version.
- **Spec-zone amendments carried by this blueprint** (committed in T0, before any code commit): (a) spec `001-storage` §2.1 S-3 is reworded — "In Postgres mode the SQLite file is the local truth for the machine that holds it; saves are *never* written directly to Postgres — they are journaled (`sync_mutations`) and replicated by the sync layer. `PostgresStore` is the direct-PG *dialect render* for machines that keep local truth in Postgres; it is a different deployment mode, not the replication path."; (b) spec `002-schema` §2.2 gains the `retrieval_usage INTEGER NOT NULL DEFAULT 0` column (feeds the `0.6·usageNorm` term of the §2.2.3 formula — the formula's source); (c) spec `004-transport-mcp` §3 `mem_save` row notes `capture_prompt` is a *tri-state* (`true`/`false`/absent, absent → `true`), surfaced as `*bool` in the handler request so the public contract cannot confuse "absent" with `false`.
- **All seven invariants** (spec `000-architecture` §4) are binding on every task: INV-1 provenance immutable + everywhere; INV-2 dual-mode; INV-3 LLM-first with explicit `degraded: true`; INV-4 local is truth; INV-5 retrieval trails; INV-6 identity is `sync_id`; INV-7 agent saves, engine stores.
- **Test command:** `cd mnemonic-v2 && go test ./...`; **build:** `cd mnemonic-v2 && go build ./...`; **format:** `gofmt -w ./...`; vet inline via `go vet ./...`.

## Threat Matrix

Per `_shared/references/threat-matrix.md`. The change touches **Mnemonic tool contracts** (the whole `mem_*` surface is new) and **process integration** (the MCP stdio server, the autosync goroutine, the two-process WAL concurrency path). Git-repository selection applies to project *detection* (cwd → git binding) even though the engine itself does no git automation.

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Mnemonic tool surface (`mem_*`) | Applicable: all 26 tools are new public contracts (spec `004-transport-mcp` §3) | The tool contract is pinned by tests: exact tool name, required params, return shape, error codes, and profile membership are asserted; a contract drift = a failing test, not a silent break. | Task 14 Step 1 (`TestMemToolContract` — golden JSON per tool), Task 15 Step 1 (`TestProfileMembership`), Task 16 Step 1 (error-code table test) |
| Process integration | Applicable: MCP stdio server is spawned by the agent harness; autosync is a background goroutine; two processes share one WAL file | stdio protocol correctness (newline-delimited JSON-RPC) is tested in-process against a fake harness client; autosync I/O is off the write path (buffered-1 channel) and lease-guarded (one worker); concurrent-process WAL is the Task 20 backstop. | Task 14 Step 2 (stdio round-trip test), Task 17 Step 1 (`TestAutosyncSingleLease`), Task 20 Step 1 (`TestConcurrentProcessWrites`) |
| Git repository selection | Applicable: project detection resolves cwd → git binding → basename (spec `005-transport-cli` §3, Engram `hostbind` pattern) | Detection is a pure function of (cwd, env, git toplevel) tested per selector; `MNEMONIC_PROJECT` and `--project` beat detection; explicit project + `--all` is a usage error. | Task 4 Step 1 (`TestProjectDetectionSelectors` — per-selector table) |
| Documentation-like paths | N/A: the engine never classifies or executes documentation files; the only file-format logic is the sync chunk archive (JSONL, not doc-like) | — | — |
| Commit state | N/A: the engine does no git automation of its own (project detection reads `git rev-parse --show-toplevel` read-only; it never stages or commits) | — | — |
| Push state | N/A: no git push from the engine | — | — |
| PR commands | N/A: no PR automation | — | — |
| Shared-convention drift | N/A: no `_shared/{planning,craft,verification,knowledge}/` or `_shared/references/*.md` file is edited by this change | — | — |

## File Structure

```
mnemonic-v2/
├── go.mod / go.sum                     module github.com/devopstales/skillgrid/mnemonic-v2
├── cmd/mnemonic/
│   ├── main.go                         dispatch (stdlib flag, v1 pattern): mcp | serve (alias) | mem | sync | index | trail | migrate | doctor | setup
│   ├── mem.go                          mem <save|search|get|lookup|context|review|timeline|pin|unpin|delete|session|stats|doctor>
│   ├── sync.go                         sync <status|push|pull|export|import|enroll>
│   ├── trail.go                        trail <recent|show>
│   ├── migrate.go                      v1 → v2 seed export
│   ├── doctor.go                       whole-system health (+ --strict)
│   └── mcp.go                          mnemonic mcp (stdio server; starts autosync when sync.auto=true)
└── internal/
    ├── core/
    │   ├── store/
    │   │   ├── store.go                the Store interface (spec 001-storage §7 + the 5 additive methods, "Additive Store methods" below) + open/migrate/close lifecycle
    │   │   ├── types.go                Observation, SearchResult, VecResult, ContextResult, Mutation, AddObservationParams, …
    │   │   ├── sqlite.go               SQLiteStore: open (WAL, busy timeout, retry), dialect rendering, FTS5 triggers, vector leg
    │   │   ├── postgres.go             PostgresStore: open (DSN), schema_<project> scoping, tsvector render (the direct-PG dialect render — distinct from the replication path, per spec 001-storage S-3 as amended)
    │   │   ├── pool.go                 refcounted ProjectHandle by project id (MNEMONIC_DISABLE_CACHE bypass)
    │   │   ├── migrations/             001_schema.sql, 003_vectors.sql (SQLite render, additive)
    │   │   └── migrations_pg/          001_schema.sql … (Postgres local-dialect render, Task 18)
    │   ├── syncid/syncid.go            NewSyncID(prefix) → prefix + "-" + 16 hex (crypto/rand)
    │   ├── config/config.go            YAML + env load, resolution order, validation (spec 006-config)
    │   └── project/project.go          detection (explicit → env → cwd → git → basename), canonicalization
    ├── intelligence/
    │   ├── llm/
    │   │   ├── llm.go                  LLM interface + DedupDecision/Judgment/CandidateObservation
    │   │   └── openai_compat.go        chat-completions + embeddings client (one base URL)
    │   ├── embedder/embedder.go        Embedder interface + OpenAI-compat provider + null adapter
    │   └── degrade/degrade.go          per-call policy: ≤1 retry, degraded flag, log
    ├── memory/
    │   ├── pipeline.go                 the 10-step save pipeline (the contract, 01-memory/001 §2)
    │   ├── dedup.go                    dedup.Run — LLM Classify path + hash floor + trust-but-verify (final shape from Task 5; Task 8 fills the LLM internals)
    │   ├── relations.go                candidate detection, relevance gate, judge/judge_many/compare, consolidate
    │   ├── sessions.go                 lifecycle, terminal, resume, finalize, six-section summary
    │   ├── prompts.go                  SavePrompt (capture) — the prompt search path is the store's (Store.SearchPrompts, Task 13)
    │   ├── lookup.go                   context/search/get/lookup, RRF, trails, retrieval_usage recording, review mark_reviewed
    │   ├── sync.go                     journal apply, tombstones, deferred-apply, export/import driver
    │   ├── chunkcodec.go               chunk id + canonicalization + archive read/write (pure, unit-tested)
    │   ├── transport.go                MutationTransport interface + PG transport
    │   ├── autosync.go                 lease-guarded background manager
    │   ├── eventbus.go                 in-process SaveEvent bus (memory emits, wiki consumes)
    │   └── importance.go               the deterministic on-the-fly importance formula
    ├── migrate/
    │   └── migrate.go                  v1 → v2 seed export (read-only over v1 files; writes through memory.Service)
    └── wiki_teststub/
        └── stub_test.go                the no-dependency SaveEvent consumer proof (Task 12)
    └── transport/
        ├── handler/handler.go          the one shared handler set (Save/Search/Get/Lookup/… with SaveResult JSON shape)
        └── mcp/
            ├── server.go               stdio server, profiles (agent/all), write queue
            └── tools_mem.go            all 26 mem_* tools as thin adapters over handler
```

Responsibility notes: `internal/memory` never imports `internal/transport` and never names `*sql.DB` (the S-2 rule). `internal/transport/handler` is the seam where the MCP and CLI surfaces meet — both are thin. The wiki function (a later change) subscribes to `eventbus` — it is not a dependency of `internal/memory`.

---

### Task 1: Module skeleton + SQLite Store interface + handle pool (door check)

> ⚠ one-way: **D5 — fresh module `mnemonic-v2/`** (vs in-place rewrite). The old module is `mnemonic_back/` during development; coexistence is by design until v2 ships.
>
> **Door check — this task tests the hypothesis.** If a Go module cannot open a per-project SQLite file with WAL through a dialect-agnostic `Store` interface **and find a saved observation by FTS5 trigram search** (with the `< 3 runes` LIKE fallback), stop: the architecture's foundation (INV-2 + the FTS retrieval backbone, `001-storage` §3) does not hold and the blueprint is invalidated. Task 2 re-proves the same properties against the full migration set (trigger sync + ranking) — the door has two locks; Task 1 is the stop gate.

**Files:**
- Create: `mnemonic-v2/go.mod`, `mnemonic-v2/internal/core/store/store.go`, `mnemonic-v2/internal/core/store/types.go`, `mnemonic-v2/internal/core/store/sqlite.go`, `mnemonic-v2/internal/core/store/pool.go`, `mnemonic-v2/internal/core/syncid/syncid.go`
- Test: `mnemonic-v2/internal/core/store/store_test.go`, `mnemonic-v2/internal/core/store/pool_test.go`, `mnemonic-v2/internal/core/syncid/syncid_test.go`
- Modify: `/data/git/AI/skillgrid/go.work` (add `./mnemonic-v2` to the `use` block)

**Interfaces:**
- Consumes: nothing (first task).
- Produces:
  - `store.Open(ctx context.Context, dataDir, project string, dialect Dialect, dsn string) (Store, error)` — resolves `<dataDir>/<project>.sqlite`, runs migrations, returns a `Store`. `Dialect` is `SQLite | Postgres`.
  - `type Store interface` — the full method set from spec `001-storage` §7 (this task implements the observation + lifecycle subset; later tasks add the rest to the same interface without changing existing signatures):
    ```go
    type Store interface {
        // observations
        AddObservation(ctx context.Context, p AddObservationParams) (AddObservationResult, error)
        GetObservation(ctx context.Context, id int64, includeHistory bool) (Observation, error)
        UpdateObservationForProject(ctx context.Context, id int64, expectedProject string, p UpdateParams) error
        DeleteObservationForProject(ctx context.Context, id int64, expectedProject string, hard bool) error
        ConsolidateObservations(ctx context.Context, p ConsolidateParams) (ConsolidateResult, error)
        // additive methods (beyond spec 001-storage §7 — same S-2 rule: Go types only)
        MarkReviewed(ctx context.Context, id int64) error                              // resets review_after = now + type-months (spec 002-schema §2.2.2)
        BumpRetrievalUsage(ctx context.Context, ids []int64) error                     // retrieval_usage++ per id (feeds importance, spec 002-schema §2.2.3)
        SearchPrompts(ctx context.Context, q PromptSearchQuery) (PromptSearchResult, error) // prompts_fts over (content, project)
        // test-only hooks (SQLite store; a no-op-capable method on the interface so both dialects satisfy it)
        TouchForTest(ctx context.Context, id int64, lastSeenAt string, revisionCount int) error
    ```
    (The committed-transaction counter used by Task 10's `txnCount` is a test seam on the SQLite store's write wrapper — a `SetTxnHook(func())` method, likewise on the interface.)
        AddRelation(ctx context.Context, p AddRelationParams) (Relation, error)
        JudgeRelation(ctx context.Context, id int64, p JudgeParams) error
        // retrieval
        Search(ctx context.Context, q SearchQuery) (SearchResult, error)
        VectorSearch(ctx context.Context, q VecQuery) (VecResult, error)
        Context(ctx context.Context, q ContextQuery) (ContextResult, error)
        // sync
        EnqueueMutation(ctx context.Context, m Mutation) (int64, error)
        ListPendingMutations(ctx context.Context, cursor int64, limit int) ([]Mutation, error)
        ApplyPulledMutation(ctx context.Context, m Mutation) error
        Tombstone(ctx context.Context, entity, key string) error
        // lifecycle
        Migrate(ctx context.Context) error
        Close() error
        Ping(ctx context.Context) error
    }
    ```
  - `types.go`: `Observation{ID int64, SyncID, SessionID, Type, Title, Content string, ToolName *string, Project *string, Scope string, TopicKey *string, RevisionCount, DuplicateCount int, LastSeenAt *string, Pinned bool, ReviewAfter *string, SupersededBySyncID *string, Importance float64, Provenance Provenance, Embedding []byte, EmbeddingModel string, CreatedAt, UpdatedAt string, DeletedAt *string}` with `State() string` deriving `active` | `needs_review` from `ReviewAfter` vs now (Engram `observationState` semantics); `Provenance{SessionID, Writer, Model, CreatedAt string}` (INV-1, immutable); `AddObservationParams{SessionID, Type, Title, Content, ToolName, Project, Scope, TopicKey, OperationID string}`; `AddObservationResult{ID int64, SyncID string, Replayed bool}`.
  - `syncid.NewSyncID(prefix string) string` — `crypto/rand` 8 bytes, `prefix + "-" + hex(16)`, prefix ∈ `obs|rel|ses|prm`.
- Seam: the `Store` interface is the seam (INV-2: functions see only it).
- Deletion test: deleting `internal/core/store` re-creates the dialect SQL in every caller — the callers are `internal/memory` (all of it) and `internal/transport/handler`.
- Adapters: `SQLiteStore` (this task) + `PostgresStore` (Task 18) + the in-memory test store used by handler tests = 3, seam justified.

**SATISFIES:** [door-check], [store-opens-per-project], [sync-id-format], [ledger-fk-set-null]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/core/store/store_test.go
func TestStoreOpensPerProjectFile(t *testing.T) {
    dir := t.TempDir()
    st, err := store.Open(context.Background(), dir, "proj-a", store.SQLite, "")
    if err != nil { t.Fatalf("open: %v", err) }
    defer st.Close()
    if _, err := os.Stat(filepath.Join(dir, "proj-a.sqlite")); err != nil {
        t.Fatalf("expected per-project file: %v", err)
    }
    // a second project must not share the file
    st2, err := store.Open(context.Background(), dir, "proj-b", store.SQLite, "")
    if err != nil { t.Fatalf("open b: %v", err) }
    defer st2.Close()
    if _, err := os.Stat(filepath.Join(dir, "proj-b.sqlite")); err != nil {
        t.Fatalf("expected second per-project file: %v", err)
    }
    if err := st.Ping(context.Background()); err != nil { t.Fatalf("ping: %v", err) }
}

func TestAddGetObservationRoundTrip(t *testing.T) {
    st := openTestStore(t) // helper: Open into t.TempDir(), t.Cleanup(Close)
    res, err := st.AddObservation(context.Background(), store.AddObservationParams{
        SessionID: "ses-1", Type: "decision", Title: "Use SQLite",
        Content: "**What** chose SQLite\n**Why** cgo-free",
        Project: "proj-a", Scope: "project",
    })
    if err != nil { t.Fatalf("add: %v", err) }
    if res.ID == 0 || !strings.HasPrefix(res.SyncID, "obs-") || len(res.SyncID) != 4+16 {
        t.Fatalf("bad result: %+v", res)
    }
    got, err := st.GetObservation(context.Background(), res.ID, false)
    if err != nil { t.Fatalf("get: %v", err) }
    if got.Title != "Use SQLite" || got.Provenance.CreatedAt == "" || got.State() != "active" {
        t.Fatalf("bad row: %+v", got)
    }
}

// THE door check (hypothesis falsifier): WAL open + FTS5 trigram retrieval through the Store.
func TestDoorCheckFTSRoundTrip(t *testing.T) {
    st := openTestStore(t)
    res, err := st.AddObservation(context.Background(), store.AddObservationParams{
        SessionID: "ses-1", Type: "discovery", Title: "Trigram recall",
        Content: "trigram tokens give substring recall",
        Project: "proj-a", Scope: "project",
    })
    if err != nil { t.Fatalf("add: %v", err) }
    r, err := st.Search(context.Background(), store.SearchQuery{Query: "trigram", Project: "proj-a", Limit: 10})
    if err != nil { t.Fatalf("search: %v", err) }
    if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
        t.Fatalf("FTS5 trigram retrieval broken: %+v", r)
    }
    // the < 3 runes fallback (trigram minimum) must find it via LIKE
    r2, err := st.Search(context.Background(), store.SearchQuery{Query: "go", Project: "proj-a", Limit: 10})
    if err != nil { t.Fatalf("short search: %v", err) }
    _ = r2 // (a second row titled "Go" is seeded in Task 2's LIKE test; here the fallback path is exercised)
}
```

```go
// the ledger FK must be ON DELETE SET NULL — a hard-deleted observation leaves its
// save-operation ledger row as a tombstone (spec 002-schema §3.3; Task 6's 410 path relies on it)
func TestLedgerFKSetNull(t *testing.T) {
    st := openTestStore(t)
    res, _ := st.AddObservation(context.Background(), store.AddObservationParams{
        SessionID: "s", Type: "decision", Title: "T", Content: "C",
        Project: "p", Scope: "project", OperationID: "op-fk",
    })
    if err := st.DeleteObservationForProject(context.Background(), res.ID, "p", true); err != nil {
        t.Fatalf("hard delete: %v", err)
    }
    // ledger row survives with observation_id NULL (via the test store's raw probe)
    if ok := ledgerRowSurvivesNull(t, st, "op-fk"); !ok {
        t.Fatalf("ledger row must survive the hard delete with observation_id SET NULL")
    }
}

// mnemonic-v2/internal/core/syncid/syncid_test.go
func TestNewSyncIDFormat(t *testing.T) {
    for _, prefix := range []string{"obs", "rel", "ses", "prm"} {
        id := syncid.NewSyncID(prefix)
        if !regexp.MustCompile(`^`+prefix+`-[0-9a-f]{16}$`).MatchString(id) {
            t.Fatalf("bad sync id %q", id)
        }
        if id == syncid.NewSyncID(prefix) { t.Fatalf("collision in same call pair") }
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/core/... -run 'TestStoreOpensPerProjectFile|TestAddGetObservationRoundTrip|TestDoorCheckFTSRoundTrip|TestNewSyncIDFormat' -v`
Expected: FAIL — package `store` does not exist (build error). BDD: `[door-check]`, `[store-opens-per-project]`, `[sync-id-format]` are RED in `acceptance.feature`.

- [ ] **Step 3: Write minimal implementation**

`go.mod` (module `github.com/devopstales/skillgrid/mnemonic-v2`, `go 1.26.0`, require `modernc.org/sqlite v1.60.1` — same version as v1's go.sum). `sqlite.go` opens with the v1/Engram-proven DSN: `file:<path>?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)` + `_txlock=immediate`, `db.SetMaxOpenConns(1)`, and a write-retry wrapper with backoffs 10/25/50/100/200 ms on `SQLITE_BUSY`/`SQLITE_LOCKED` (port the ~45-line `withSQLiteWriteRetry` + `isRetryableSQLiteLockError` from Engram `store.go:10180-10207`). Migrations: `//go:embed migrations/*.sql` + a runner that creates `index_meta(key, schema_version)`, sorts embedded names, executes each unapplied file statement-by-statement (port the trigger-aware splitter from v1 `store.go:462-494`). `migrations/001_schema.sql` creates the full spec `002-schema` SQLite render: `sessions`, `session_finalizations`, `observations` (all columns incl. `normalized_hash`, `importance`, `provenance`, `embedding BLOB`), `idx_obs_dedupe (normalized_hash, project, scope, type, title)`, `idx_obs_topic (project, scope, topic_key)`, `observation_versions`, `user_prompts`, `prompt_tombstones`, `memory_relations`, `sync_mutations`, `sync_state`, `sync_chunks`, `sync_delete_tombstones`, `sync_delete_tombstones_remote_floors`, `sync_apply_deferred`, `sync_enrolled_projects`, `observation_save_operations`, and the FTS5 external-content tables + triggers (verbatim from Engram `store.go:3125-3191`: `observations_fts(title, content, tool_name, type, project, topic_key, tokenize='trigram', content='observations', content_rowid='id')`, `prompts_fts(content, project, tokenize='trigram', content='user_prompts', content_rowid='id')`, and the four triggers each). `pool.go`: a `map[string]*refHandle` guarded by a `sync.Mutex`, `Acquire(project)` / `Release(project)`; `MNEMONIC_DISABLE_CACHE=1` bypasses it (one open per call). `AddObservation` mints `sync_id`, sets `provenance` + `created_at`/`updated_at`, computes `normalized_hash` (lowercase + whitespace-collapse + SHA-256 hex, Engram `hashNormalized` `store.go:13665`), inserts, and returns the result. `UpdateObservationForProject` / `DeleteObservationForProject` / `Search` (FTS5 `bm25` with weights title 5.0 / content 1.0 / topic_key 3.0, Engram `store.go:424-431, 6024-6069`) / `VectorSearch` (returns empty + `degraded` for now — wired in Task 12) / the sync methods (journal insert/select; apply stubbed until Task 17) are implemented to their final signatures now so the interface never changes shape again.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/core/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2 go.work .skillgrid/artifacts/04-adr-0040-mnemonic-v2-module-extraction.md
git commit -m "feat(mnemonic-v2): module skeleton, dialect-agnostic Store interface, SQLite store

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T01 door-check"
```

---

### Task 2: Migration suite + FTS5 trigger parity (both-dialects-from-empty regression)

**Files:**
- Create: `mnemonic-v2/internal/core/store/migrate_test.go`
- Modify: `mnemonic-v2/internal/core/store/migrations/001_schema.sql` (only the gap the parity test finds — the FTS trigger set may need the trigram-safe four-trigger form)
- Test: `mnemonic-v2/internal/core/store/migrate_test.go`, `mnemonic-v2/internal/core/store/fts_test.go`

**Interfaces:**
- Consumes: Task 1's `store.Open`, the migration runner, `Store.Search`.
- Produces: nothing new to callers — this task hardens Task 1's store. It produces the **migration parity test pattern** that Task 18 (Postgres render) re-runs against `migrations_pg/`.
- Seam: same as Task 1 (the `Store` interface).
- Deletion test: n/a (hardening).
- Adapters: same as Task 1.

**SATISFIES:** [fts-stays-in-sync], [migration-runs-from-empty], [migration-additive-reopen], [short-term-like-fallback], [search-ranking-recency-stability]

- [ ] **Step 1: Write the failing tests**

```go
func TestFTSStaysInSync(t *testing.T) {
    st := openTestStore(t)
    res, _ := st.AddObservation(ctx, store.AddObservationParams{
        SessionID: "s", Type: "decision", Title: "Use trigram search",
        Content: "trigram recall beats word splitting for C++",
        Project: "p", Scope: "project",
    })
    r, _ := st.Search(ctx, store.SearchQuery{Query: "trigram", Project: "p", Limit: 10})
    if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
        t.Fatalf("fts insert trigger broken: %+v", r)
    }
    // update title — the BEFORE UPDATE trigger must DELETE the old FTS row first
    _ = st.UpdateObservationForProject(ctx, res.ID, "p", store.UpdateParams{Title: ptr("Bigger trigrams")})
    r2, _ := st.Search(ctx, store.SearchQuery{Query: "Bigger", Project: "p", Limit: 10})
    if len(r2.Observations) != 1 { t.Fatalf("fts update trigger broken") }
    r3, _ := st.Search(ctx, store.SearchQuery{Query: "trigram", Project: "p", Limit: 10})
    if len(r3.Observations) != 0 { t.Fatalf("stale FTS row after update") }
}

func TestMigrationRunsFromEmptyAndReopens(t *testing.T) {
    dir := t.TempDir()
    st, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
    if err != nil { t.Fatalf("fresh open: %v", err) }
    st.Close()
    // reopen: migrations must be idempotent (additive-only discipline)
    st2, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
    if err != nil { t.Fatalf("reopen: %v", err) }
    st2.Close()
}

func TestShortTermFallsBackToLIKE(t *testing.T) {
    st := openTestStore(t)
    _, _ = st.AddObservation(ctx, store.AddObservationParams{
        SessionID: "s", Type: "discovery", Title: "Go",
        Content: "Go 1.26 is the floor for the v2 module",
        Project: "p", Scope: "project",
    })
    r, _ := st.Search(ctx, store.SearchQuery{Query: "go", Project: "p", Limit: 10})
    if len(r.Observations) != 1 { t.Fatalf("short-term LIKE fallback broken (trigram needs >=3 runes)") }
}

// The full T5 ranking formula: BM25 × (1 + pinned·0.10 + recency·0.06·1/(1+days/30) + stability·0.04·(rev+dups)/(rev+dups+4)).
// Two otherwise-identical rows differ ONLY in last_seen_at / revision_count — the order must follow the boosts.
func TestSearchRankingRecencyAndStability(t *testing.T) {
    st := openTestStore(t)
    a := seedStoreObs(t, st, "p", "decision", "identical title", "identical content body")
    b := seedStoreObs(t, st, "p", "decision", "identical title", "identical content body")
    // b is fresher (last_seen_at = now) and more stable (3 revisions)
    _ = st.TouchForTest(ctx, b.ID, "now", 3)   // test hook: set last_seen_at + revision_count
    _ = st.TouchForTest(ctx, a.ID, "30 days ago", 0)
    r, _ := st.Search(ctx, store.SearchQuery{Query: "identical", Project: "p", Limit: 10})
    if len(r.Observations) != 2 || r.Observations[0].ID != b.ID {
        t.Fatalf("recency/stability boosts must order b first: %+v", r.Observations)
    }
    _ = a
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/core/store/ -run 'TestFTSStaysInSync|TestMigrationRunsFromEmptyAndReopens|TestShortTermFallsBackToLIKE|TestSearchRankingRecencyAndStability' -v`
Expected: FAIL where the trigger/like/ranking behavior is missing. The `< 3 runes` fallback is the `hasShortFTSTerm` rule: any whitespace-split term with `utf8.RuneCountInString(term) < 3` → bounded `LIKE` over the six FTS columns, ordered `updated_at DESC` (Engram `store.go:6071-6146`; escape `\ % _`).

- [ ] **Step 3: Write minimal implementation**

Complete the FTS trigger set in `migrations/001_schema.sql` if Step 2 shows a gap — the trigram-safe form is four triggers per table: AFTER INSERT `WHEN new.deleted_at IS NULL`, BEFORE DELETE, BEFORE UPDATE (plain `DELETE FROM … WHERE rowid = old.id` — trigram's unsafe `'delete'` special command is why the delete happens before the base row changes), AFTER UPDATE (Engram `store.go:3149-3191`). Add `hasShortFTSTerm` + the LIKE fallback to `Search` in `sqlite.go`. **Complete the composite rank** (Engram `store.go:6031-6044`): `rawRank × (1 + pinned·0.10 + 0.06·1/(1+age_days/30) + 0.04·(revision_count+duplicate_count)/(revision_count+duplicate_count+4))`, age from `COALESCE(last_seen_at, updated_at, created_at)`, ordered by composite rank — the recency/stability test above pins it. The migration re-run guard is already in Task 1's runner (`index_meta` record per applied file) — this task's test is the proof.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/core/store/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/core/store/
git commit -m "feat(mnemonic-v2): FTS5 trigram triggers, short-term LIKE fallback, migration parity test

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T02"
```

---

### Task 3: Intelligence seams (LLM + Embedder interfaces, OpenAI-compat provider, degradation policy)

**Files:**
- Create: `mnemonic-v2/internal/intelligence/llm/llm.go`, `mnemonic-v2/internal/intelligence/llm/openai_compat.go`, `mnemonic-v2/internal/intelligence/embedder/embedder.go`, `mnemonic-v2/internal/intelligence/degrade/degrade.go`
- Test: `mnemonic-v2/internal/intelligence/llm/llm_test.go`, `mnemonic-v2/internal/intelligence/llm/openai_compat_test.go`, `mnemonic-v2/internal/intelligence/embedder/embedder_test.go`, `mnemonic-v2/internal/intelligence/degrade/degrade_test.go`

**Interfaces:**
- Consumes: `core/config` (LLM config shape from Task 4 — this task defines the *seam* types first; config parsing lands in Task 4).
- Produces:
  ```go
  // internal/intelligence/llm/llm.go
  type LLM interface {
      Classify(ctx context.Context, newContent string, candidates []string) (DedupDecision, error)
      Judge(ctx context.Context, source, target Observation, rel Relation) (Judgment, error)
      Compare(ctx context.Context, a, b Observation) (Judgment, error)
      Extract(ctx context.Context, text string) ([]CandidateObservation, error)
  }
  type DedupVerdict string // "add" | "update" | "delete" | "noop"  (v1 constants, mnemonic_back/internal/memory/types.go:57)
  type DedupDecision struct { Verdict DedupVerdict; CandidateID int64; Reason string } // Reason: "llm" | "hash" | ""
  type Judgment struct { Relation string; Confidence float64; Evidence string; Model string } // Model → marked_by_model (INV-1)
  type CandidateObservation struct { Type, Title, Content string }
  ```
  ```go
  // internal/intelligence/embedder/embedder.go
  type Embedder interface {
      Embed(ctx context.Context, texts []string) ([][]float32, error)
      Model() string        // recorded on observations.embedding_model (INV-1)
      Dimensions() int      // must match the vector column; mismatch → error at first write, never silent truncation
  }
  ```
  - `openai_compat.go`: `NewClient(baseURL, model, apiKey string, timeout time.Duration) (*Client, error)` — one HTTP shape, `POST {baseURL}/chat/completions` for `Classify`/`Judge`/`Compare`/`Extract` and `POST {baseURL}/embeddings` for `Embed` (port v1 `internal/llm/client.go` — the Ollama provider is the same client pointed at its OpenAI-compat root; no second client). `Client` implements both `LLM` and `Embedder`.
  - `degrade.go`: `Run[T any](seam T, floor func() T, name string) (result T, degraded bool)` — nil seam → floor + `degraded`; errored seam → one bounded retry, then floor + `degraded`; success → result, not degraded. Logs one degradation line per degraded call. Per-call: a failing `Embedder` never degrades `Classify`.
- Seam: the `LLM` and `Embedder` interfaces *are* the seams (INV-3). Process-lifetime: one client per process, constructed at startup (spec `003-intelligence` §5).
- Deletion test: deleting `internal/intelligence` forces every semantic op in `internal/memory` to inline its provider call and its degradation logic — the callers are `memory/dedup.go` (Task 8), `memory/relations.go` (Task 10), `memory/sessions.go` (Task 11), `memory/lookup.go` (Task 12).
- Adapters: `openai_compat.Client` (prod; the Ollama deployment is the same client pointed at a different base URL — one adapter, two configs) + the test stub + the null adapter = 3, plus the per-call floor functions — seam justified (the interface is consumed by 4 pipeline call sites: dedup-classify, judge, compare, extract).

**SATISFIES:** [llm-seam-4-way-classify], [degraded-explicit], [degraded-bounded-retry]

- [ ] **Step 1: Write the failing tests**

```go
func TestClassifyReturnsFourWayVerdict(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "/chat/completions") {
            fmt.Fprint(w, `{"choices":[{"message":{"content":"{\\"verdict\\":\\"update\\",\\"candidate\\":1}"}}]}`)
            return
        }
        t.Fatalf("unexpected path %s", r.URL.Path)
    }))
    defer srv.Close()
    c, err := llm.NewClient(srv.URL, "test-model", "k", 5*time.Second)
    if err != nil { t.Fatal(err) }
    d, err := c.Classify(context.Background(), "new text", []string{"old one", "old two"})
    if err != nil { t.Fatal(err) }
    if d.Verdict != llm.VerdictUpdate || d.CandidateID != 2 { // 0-based index 1 → 1-based id 2 (v1 mapping, service/dedup_llm.go:98-100)
        t.Fatalf("bad decision %+v", d)
    }
    if d.Reason != "llm" { t.Fatalf("reason must be llm") }
}

func TestDegradeNilSeamRunsFloorExplicitly(t *testing.T) {
    var l llm.LLM // nil
    var ranFloor bool
    d, degraded := degrade.Run(l, func() llm.DedupDecision { ranFloor = true; return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "hash"} }, "classify")
    if !ranFloor || !degraded || d.Verdict != llm.VerdictAdd { t.Fatalf("floor must run, flagged degraded") }
}

func TestDegradeErroredSeamRetriesOnceThenFloor(t *testing.T) {
    calls := 0
    seam := failingSeam{fn: func() (llm.DedupDecision, error) { calls++; return llm.DedupDecision{}, errors.New("boom") }}
    _, degraded := degrade.Run(seam, func() llm.DedupDecision { return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "hash"} }, "classify")
    if calls != 2 { t.Fatalf("expected exactly one retry (2 calls), got %d", calls) }
    if !degraded { t.Fatalf("must be degraded") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/intelligence/... -v`
Expected: FAIL — packages do not exist. BDD: `[llm-seam-4-way-classify]`, `[degraded-explicit]`, `[degraded-bounded-retry]` RED.

- [ ] **Step 3: Write minimal implementation**

`llm.go`: the interface + types (exact block above). `openai_compat.go`: port v1's `Complete` (system/user two-message chat call, 60s default timeout, empty API key ⇒ no Authorization header — `mnemonic_back/internal/llm/client.go:29-69`); add `Embed` as `POST {base}/embeddings` with `{"model":…, "input":[…]}` → `data[].embedding`; the `Classify`/`Judge`/`Compare`/`Extract` prompts are strict-JSON contracts (port the `dedupPrompt`/`dedupSystemPrompt` pair from v1 `internal/service/dedup_llm.go:48,121` — verdict + 0-based candidate index; the extract prompt `{"learnings":[{"text","type"}]}` from `internal/memory/extraction.go:34`). `embedder.go`: interface + null adapter. **Absent vs failing are different states (INV-3):** a `nil` seam / `provider: none` config = *absent* — the vector leg simply does not run, no retry, `degraded: true` on results that would have used it. A *present* seam that errors = *failing* — one bounded retry (via `degrade.Run`), then the floor. The null adapter models "absent" (`Model()` = `""`, `Dimensions()` = 0, `Embed` never called); it is not an erroring seam. `degrade.go`: the generic `Run` above with a `log` line `degraded <name>: <reason>`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/intelligence/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/intelligence/
git commit -m "feat(mnemonic-v2): LLM + Embedder seams, OpenAI-compatible client, explicit degradation policy

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T03"
```

---

### Task 4: Config + project detection (YAML, env overrides, resolution order)

**Files:**
- Create: `mnemonic-v2/internal/core/config/config.go`, `mnemonic-v2/internal/core/project/project.go`
- Test: `mnemonic-v2/internal/core/config/config_test.go`, `mnemonic-v2/internal/core/project/project_test.go`

**Interfaces:**
- Consumes: `core/store` (`Dialect`), `intelligence/llm` + `intelligence/embedder` (the seams config constructs).
- Produces:
  ```go
  // core/config/config.go
  type Config struct {
      Enabled bool
      DataDir string   // default ~/.skillgrid/mnemonic (project config.yaml mnemonic.data_dir)
      Datasource Datasource  // {Dialect "sqlite"|"postgres", DSN string, MaxConns int(8)}
      Project    ProjectCfg  // {EnvVar "MNEMONIC_PROJECT", Detect bool(true)}
      LLM        LLMConfig   // {Provider "openai-compat"|"local", BaseURL, Model, APIKeyEnv, Local BaseURL/Model, TimeoutMs 60000, MaxRetries 1}
      Embedder   EmbedderCfg // {Provider "openai-compat"|"local"|"none", Model, Dimensions int(0=infer)}
      Sync       SyncCfg     // {Enabled false, Target "cloud", Auto false, DebounceMs 500, PushLimit 100, MaxPushBytes 8388608}
      Memory     MemoryCfg   // {Review months map (spec 006-config §2 table), ContentMaxBytes 32000, PrivateTag "<private>"}
  }
  func Load(path string, flags Flags) (Config, error)  // resolution: flag > env > file > default
  func (c Config) BuildLLM() (llm.LLM, error)          // nil seam when provider unset (degraded floor is the default-less mode)
  func (c Config) BuildEmbedder() (embedder.Embedder, error)
  ```
  ```go
  // core/project/project.go
  type Detector struct{ EnvVar string }
  func (d Detector) Resolve(cwd string, explicit string, gitToplevel func(dir string) (string, error)) (string, error)
  // order: explicit → MNEMONIC_PROJECT → cwd git binding → basename(cwd)
  func Canonicalize(name string) string  // lowercase, path separator → '-', strip trailing '/'
  ```
- Seam: config is a value, not a seam; project detection is a pure function tested per selector.
- Deletion test: deleting `config` re-introduces hard-coded defaults and env reads across `cmd/mnemonic`, `store`, `intelligence` — the named callers.
- Adapters: file-based + env-only config = the two input modes; no port needed.

**SATISFIES:** [config-resolution-order], [config-validation], [project-detection-selectors]

- [ ] **Step 1: Write the failing tests**

```go
func TestConfigResolutionOrder(t *testing.T) {
    // env must beat file; flag must beat env
    t.Setenv("MNEMONIC_DATA_DIR", "/from-env")
    cfg, _ := config.Load("", config.Flags{})            // no flag
    if cfg.DataDir != "/from-env" { t.Fatalf("env must win over default: %q", cfg.DataDir) }
    cfg2, _ := config.Load("", config.Flags{DataDir: "/from-flag"})
    if cfg2.DataDir != "/from-flag" { t.Fatalf("flag must win over env") }
}

func TestConfigValidation(t *testing.T) {
    _, err := config.Load("", config.Flags{Dialect: "postgres"}) // no DSN, no MNEMONIC_PG_DSN
    if err == nil { t.Fatalf("postgres without DSN must fail at load/open, not at first query") }
    _, err = config.LoadWithYAML(yamlWithLocalProviderNoBaseURL)
    if err == nil { t.Fatalf("llm.provider=local without local.base_url must error") }
}

func TestProjectDetectionSelectors(t *testing.T) {
    det := project.Detector{EnvVar: "MNEMONIC_PROJECT"}
    git := func(dir string) (string, error) { return filepath.Join(dir, "repo"), nil }
    got, _ := det.Resolve("/x/repo", "explicit", git)
    if got != "explicit" { t.Fatalf("explicit wins") }
    t.Setenv("MNEMONIC_PROJECT", "from-env")
    got, _ = det.Resolve("/x/repo", "", git)
    if got != "from-env" { t.Fatalf("env wins over detection") }
    t.Setenv("MNEMONIC_PROJECT", "")
    got, _ = det.Resolve("/x/repo", "", git)
    if got != "repo" { t.Fatalf("git binding → basename: %q", got) }
    noGit := func(string) (string, error) { return "", os.ErrNotExist }
    got, _ = det.Resolve("/x/plain-dir", "", noGit)
    if got != "plain-dir" { t.Fatalf("basename fallback: %q", got) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/core/config/ ./internal/core/project/ -v`
Expected: FAIL — packages do not exist. BDD: `[config-resolution-order]`, `[config-validation]`, `[project-detection-selectors]` RED.

- [ ] **Step 3: Write minimal implementation**

`config.go`: YAML file discovered by walking up from cwd to `.skillgrid/config.yaml` (v1's `findIndexingYAML` walk-up pattern, `mnemonic_back/internal/config/load.go:632-649`), `gopkg.in/yaml.v3`; per-key merge with "zero/empty keeps default" (v1 `mergeIndexing` semantics); then env overrides (`MNEMONIC_DATA_DIR`, `MNEMONIC_PROJECT`, `MNEMONIC_PG_DSN`, `MNEMONIC_LLM_API_KEY`, `MNEMONIC_SYNC_AUTO=1`, `MNEMONIC_DISABLE_CACHE`); then flags last. Validation per spec `006-config` §5: `postgres` + empty DSN → error; `local` provider without `base_url` → error; `embedder.dimensions` mismatch is checked at first write (Task 12), not here. `project.go`: the four-step resolve above; `Canonicalize` = lowercase + `/`→`-` + trim trailing separators (the `schema_<sanitized_project>` naming depends on it — Task 18).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/core/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/core/config/ mnemonic-v2/internal/core/project/
git commit -m "feat(mnemonic-v2): config load (flag>env>file>default) + project detection

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T04"
```

---

### Task 5: Save pipeline core — validate, resolve, redact, hash floor (steps 1–3 + 5-floor of the contract)

**Files:**
- Create: `mnemonic-v2/internal/memory/pipeline.go`, `mnemonic-v2/internal/memory/dedup.go`, `mnemonic-v2/internal/memory/importance.go`, `mnemonic-v2/internal/memory/eventbus.go`, `mnemonic-v2/internal/memory/testutil_test.go`
- Test: `mnemonic-v2/internal/memory/pipeline_test.go`, `mnemonic-v2/internal/memory/dedup_test.go`, `mnemonic-v2/internal/memory/importance_test.go`

**Test helper contract (used by this task and all later memory tests):** `testutil_test.go` (package `memory`) defines, once, the helpers every later task's tests call — `newTestService(t) *Service` (real `SQLiteStore` in `t.TempDir()`, nil LLM/embedder → degraded; `svc.st` is the store), `newTestServiceWith(t, cfg config.MemoryCfg)`, `newTestServiceWithLLM(t, stub *stubLLM) *Service` + `(s *Service) stub() *stubLLM` (the stub's verdicts are set per test via `setClassify`/`setCompare`/`setJudge`), `newTestServiceWithStore(t, st store.Store)`, `newTestServiceWithBus(t) (*Service, *EventBus)`, `seedObservation(t, st store.Store, project, typ, title, content, topicKey string) store.Observation` (an `AddObservation` with a fixed test session, returns the row incl. `SyncID`), `countObs(t, st) int`, `liveObsCount(t, st) int`, `findIDBySyncID(t, st, syncID string) int64`, `syncIDsOf(t, st, project string) []string`, `openTestStoreIn(t, dir, project string) store.Store`, `openTestStore(t) store.Store`, `ptr[T any](v T) *T`, `countPrompts`, `countPendingRelations`, `pendingCount`, `txnCount` (a store hook counting committed transactions), `leaseOf`, `countSummaryObs`, `trails`, `sizeOf`, `jsonEqual`, `syncCfg(on bool) config.Config`, `syncCfgAuto() config.Config`, `makeMutations(n int) []store.Mutation`, `upsertMutation/deleteMutation/relationMutation` (test-fixture `store.Mutation` values), `isLive(t, st, syncID string) bool`, `deferredCount(t, st) int`. Later tasks add to this file only when a new helper is needed — never redefine an existing one.

**Interfaces:**
- Consumes: `core/store.Store` (Task 1), `core/config.Config` (Task 4), `core/syncid` (Task 1), `intelligence/degrade` (Task 3), `intelligence/llm.DedupDecision` (Task 3).
- Produces:
  ```go
  // internal/memory/pipeline.go
  type Service struct { st store.Store; cfg config.Config; bus *EventBus; llm llm.LLM; emb embedder.Embedder }
  func NewService(st store.Store, cfg config.Config, bus *EventBus, l llm.LLM, e embedder.Embedder) *Service
  type SaveRequest struct {
      SessionID, Type, Title, Content, ToolName, Project, Scope, TopicKey, OperationID string
      CapturePrompt *bool   // tri-state (spec 004-transport-mcp mem_save row, amended): nil → true (the MCP "absent" default), explicit false → off
      CurrentPrompt string // the current turn's prompt, when capture is on
  }
  type SaveResult struct {
      ID int64; SyncID string; RevisionCount int; Replayed bool
      Dedup struct{ Verdict, Reason string; MergedIntoSyncID *string } // sync_id, never the local rowid (INV-6, spec 004-dedup §8)
      Candidates []Relation  // conflict candidates (Task 9 fills; empty until then)
      JudgmentRequired bool
      Degraded bool
  }
  func (s *Service) Save(ctx context.Context, req SaveRequest) (SaveResult, error)
  // dedup.go — the FINAL dedup entry point (Task 8 fills the LLM path's internals; this shape is stable from Task 5):
  func dedup.Run(ctx context.Context, s llm.LLM, st store.Store, project, scope, newContent, title string) (llm.DedupDecision, bool)
  // = degrade.Run(seam: LLM.Classify over the ≤20 recent-window pre-filter, floor: hashFloor, "dedup-classify")
  //   + trust-but-verify (Task 8). A nil seam → floor + degraded=true (no retry — absent ≠ failing).
  // internal/memory/eventbus.go
  type SaveEvent struct { SyncID, Project, Scope, Type, TopicKey, Entity string; Importance float64; Degraded bool }
  type EventBus struct{ ... } // Subscribe(fn func(SaveEvent)), Publish(SaveEvent) — in-process, fan-out, consumer errors never fail the publisher
  // internal/memory/importance.go — retrieval_usage is the named source (spec 002-schema §2.2.3,
  // amended): the observations.retrieval_usage column, incremented by Store.BumpRetrievalUsage in the
  // lookup path (Task 11) and read at pipeline step 6 — NOT a free parameter.
  func Importance(retrievalUsage, ageDays int, typ string) float64
  // = clamp(0.6*min(usage,50)/50 + 0.37*recency + 0.2*typeDurability, 0, 1)
  // recency = 1.0 if ageDays<=30 else max(0, 1-ageDays/30)
  // typeDurability = 0.9 for {decision, architecture, convention, config} else 0.4  (v1 dream.go:62-64, 382-400)
  ```
- Seam: `Service.Save` is the single entry point for every save surface (MCP/CLI/HTTP mirror all funnel here — spec `01-memory/001` §1).
- Deletion test: deleting `pipeline.go` re-creates the 10-step sequence in every transport handler — the callers are `transport/handler` (Task 13) and the CLI (Task 16).
- Adapters: in-process (store) + test fake store.

**SATISFIES:** [save-persists-with-sync-id], [save-redact-truncate], [hash-dedupe-noop], [save-result-shape]

- [ ] **Step 1: Write the failing tests**

```go
func TestSavePersistsAndReturnsShape(t *testing.T) {
    svc := newTestService(t) // fake store + nil LLM (degraded)
    r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T",
        Content: "C", Project: "p", Scope: "project"})
    if err != nil { t.Fatal(err) }
    if r.ID == 0 || !strings.HasPrefix(r.SyncID, "obs-") { t.Fatalf("bad shape %+v", r) }
    if !r.Degraded { t.Fatalf("no LLM → degraded must be true") }
    if r.Dedup.Reason != "hash" { t.Fatalf("floor reason must be hash") }
    got, _ := svc.st.GetObservation(ctx, r.ID, false)
    if got.Project == nil || *got.Project != "p" || got.Provenance.SessionID != "s" {
        t.Fatalf("provenance/project not set: %+v", got)
    }
}

func TestSaveRedactsPrivateAndTruncates(t *testing.T) {
    svc := newTestServiceWith(t, config.MemoryCfg{PrivateTag: "<private>", ContentMaxBytes: 16})
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "T",
        Content: "a <private>secret</private> bbbbbbbbbbbbbbbbb", Project: "p", Scope: "project"})
    got, _ := svc.st.GetObservation(ctx, r.ID, false)
    if strings.Contains(got.Content, "secret") { t.Fatalf("private tag not stripped") }
    if len([]rune(got.Content)) > 16 { t.Fatalf("not truncated to content_max_bytes") }
    // the hash must be over the REDACTED+TRUNCATED content (spec 004-dedup §7)
    r2, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning", Title: "T",
        Content: "a <private>other</private> bbbbbbbbbbbbbbbbb", Project: "p", Scope: "project"})
    if r2.ID != r.ID || !r2.Replayed { t.Fatalf("differing only in <private> must hash-equal → noop/replay") }
}

func TestHashFloorNoopBumpsCounters(t *testing.T) {
    svc := newTestService(t)
    r1, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "discovery", Title: "D", Content: "exact same", Project: "p", Scope: "project"})
    r2, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "discovery", Title: "D", Content: "exact same", Project: "p", Scope: "project"})
    if r2.Dedup.Verdict != "noop" || r2.ID != r1.ID { t.Fatalf("exact dupe must noop on same id: %+v", r2) }
    got, _ := svc.st.GetObservation(ctx, r1.ID, false)
    if got.DuplicateCount != 1 || got.LastSeenAt == nil { t.Fatalf("counters not bumped: %+v", got) }
}

func TestSaveValidation(t *testing.T) {
    svc := newTestService(t)
    for _, bad := range []SaveRequest{
        {SessionID: "s", Type: "decision", Title: "", Content: "c", Project: "p", Scope: "project"},  // empty title
        {SessionID: "s", Type: "not-a-type", Title: "t", Content: "c", Project: "p", Scope: "project"}, // bad type
        {SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "mars"},      // bad scope
        {SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "project", TopicKey: "a/b/c"}, // >2 levels
    } {
        if _, err := svc.Save(ctx, bad); err == nil { t.Fatalf("must reject %+v", bad) }
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestSavePersists|TestSaveRedacts|TestHashFloor|TestSaveValidation' -v`
Expected: FAIL — package `memory` does not exist. BDD: `[save-persists-with-sync-id]`, `[save-redact-truncate]`, `[hash-dedupe-noop]` RED.

- [ ] **Step 3: Write minimal implementation**

`pipeline.go` implements steps 1–8 of the contract (step 4 conflict pre-filter arrives in Task 9; step 10's `Publish` call site arrives in Task 7 — both are final call sites, not stubs): **1 VALIDATE** — title/type/content non-empty, `scope ∈ {project,personal,global}`, type ∈ the 10-value vocabulary (spec `002-schema` §2.2.1), `topic_key` kebab-case ≤2 levels when given; **2 RESOLVE** — project: explicit > session's project > `project.Detector` (cwd/config — spec `01-memory/001` §2 step 2; INV-7 means don't invent a project *beyond* the detector). **Session policy (fixed here, tested in Task 7):** a missing session row is **auto-registered** inside the save transaction (a `shared`-ownership session for the resolved project — the plugin's first save opens its session implicitly, Engram's create-on-first-use); an **ended** session row → `ErrSessionEnded` (409, terminal, spec `01-memory/002` §3); a `project_owned` session whose project mismatches the resolved project → `ErrSessionProjectMismatch`. Until Task 7, the auto-register is the *real* path (the DDL exists from Task 1) — there is no stub: `SessionStart` (Task 7) and the save's auto-register are the same store insert. **3 REDACT** — strip `<private>…</private>` (config `PrivateTag`) then truncate content to `ContentMaxBytes` runes; **5 DEDUP** — the call site is final now: `decision, degraded := dedup.Run(ctx, s.llm, st, project, scope, redactedContent, title)`; in Task 5's scope the `LLM` seam is nil for every test service, so `dedup.Run` runs its floor: `hashFloor` per spec `004-dedup` §5 — look up `normalized_hash + project + scope + type + title` (the `idx_obs_dedupe` index) → hit: `noop` (bump `duplicate_count`, `last_seen_at`, `updated_at`); miss: `add`. Task 8 changes only what happens *inside* `dedup.Run` when the seam is non-nil — the pipeline never changes again; **6 UPSERT/INSERT** — (a) topic_key set + existing row matches project+scope+topic_key → UPDATE in place, `revision_count++`, snapshot pre-state into `observation_versions` first; (b) no match → INSERT (mint `sync_id`, set `provenance`, compute `review_after` from the type→months map — spec `002-schema` §2.2.2, months from `cfg.Memory.Review` — and `importance` via `importance.go`, reading `retrieval_usage` from the row — 0 for a fresh insert); (c) `update`/`delete` verdict merges into the target (Task 8's LLM path produces them); **7 ENQUEUE** — if `cfg.Sync.Enabled` and the project is enrolled (`sync_enrolled_projects`), insert a `sync_mutations` row (op `upsert`, payload = canonical entity, `entity_key` = sync_id) in the same transaction; unenrolled → no row; **8 COMMIT** — one transaction wraps observation + versions + journal (the FTS trigger fires inside it). **9 RESULT** — `SaveResult` with `Dedup.MergedIntoSyncID` (the target's sync_id on update/delete verdicts, else nil). `prompts.go` (created here, Task 7 completes it): `SavePrompt` inserts the captured prompt when `CapturePrompt` is nil-or-true and `CurrentPrompt != ""` (redact + truncate like content, `sync_id` prefix `prm-`) — same transaction as step 8. `eventbus.go`: a `sync.RWMutex`-guarded subscriber slice; `Publish` runs subscribers synchronously but recovers panics and errors into a log line (failure isolation, spec `004-transport-mcp` §5) — the call site lands in Task 7. `importance.go`: the formula exactly as in the Interfaces block (the stored `importance` column is a cache; the formula is the source of truth — spec `002-schema` §2.2.3; `retrieval_usage` is the named source of the usage term).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/
git commit -m "feat(mnemonic-v2): save pipeline (validate/resolve/redact/hash-floor upsert/journal), importance, event bus

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T05"
```

---

### Task 6: Replay-safe saves (operation idempotency ledger)

**Files:**
- Modify: `mnemonic-v2/internal/memory/pipeline.go` (step 4.5 — ledger check before the dedup step)
- Test: `mnemonic-v2/internal/memory/replay_test.go`

**Interfaces:**
- Consumes: Task 5's `Service.Save`, Task 1's `observation_save_operations` table.
- Produces: `ErrOperationConflict` (→ 409) and `ErrOperationExpired` (→ 410) sentinel errors; `SaveRequest.OperationID` now drives the ledger. No signature changes.
- Seam: same as Task 5.
- Deletion test: deleting the ledger logic re-introduces double-writes on plugin retry (the whole point — spec `01-memory/001` §4).
- Adapters: in-process.

**SATISFIES:** [replay-identical-returns-committed-id], [replay-changed-payload-409], [replay-deleted-target-410]

- [ ] **Step 1: Write the failing tests**

```go
func TestReplayIdenticalReturnsCommittedID(t *testing.T) {
    svc := newTestService(t)
    req := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
        Project: "p", Scope: "project", OperationID: "op-1"}
    r1, _ := svc.Save(ctx, req)
    r2, _ := svc.Save(ctx, req)
    if !r2.Replayed || r2.ID != r1.ID || r2.SyncID != r1.SyncID {
        t.Fatalf("identical replay must return committed id, not rewrite: %+v vs %+v", r1, r2)
    }
}

func TestReplayChangedPayloadConflicts(t *testing.T) {
    svc := newTestService(t)
    base := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project", OperationID: "op-2"}
    _, _ = svc.Save(ctx, base)
    changed := base; changed.Content = "C2"
    _, err := svc.Save(ctx, changed)
    if !errors.Is(err, memory.ErrOperationConflict) { t.Fatalf("want 409 conflict, got %v", err) }
}

func TestReplayDeletedTargetExpired(t *testing.T) {
    svc := newTestService(t)
    req := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project", OperationID: "op-3"}
    r1, _ := svc.Save(ctx, req)
    _ = svc.st.DeleteObservationForProject(ctx, r1.ID, "p", true) // hard delete → ledger row survives, observation_id SET NULL
    _, err := svc.Save(ctx, req)
    if !errors.Is(err, memory.ErrOperationExpired) { t.Fatalf("want 410 expired, got %v", err) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestReplay' -v`
Expected: FAIL — no ledger handling yet (replays currently re-run dedup/upsert). BDD: the three `[replay-*]` scenarios RED.

- [ ] **Step 3: Write minimal implementation**

In `Save`, before the dedup step: if `OperationID != ""`, compute `fingerprint = "v1:" + hex(sha256(lengthPrefixed(normalizedFields)))` over the length-prefixed, normalized request fields including redacted/truncated content, computed **before** session-ownership resolution (spec `01-memory/001` §4 — the length-prefix prevents `ab|c` vs `a|bc` collisions). Look up `observation_save_operations` by `operation_id`: row with same fingerprint + live `observation_id` → return the committed id with `Replayed: true` (dedup/upsert/journal run exactly once — this whole path returns early); row with different fingerprint → `ErrOperationConflict`; row with NULL `observation_id` (its target was deleted; the ledger column is `ON DELETE SET NULL`, Task 1's DDL) → `ErrOperationExpired`; no row → proceed, and INSERT the ledger row inside the commit transaction (step 8). No `OperationID` → no ledger row, existing behavior.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/
git commit -m "feat(mnemonic-v2): replay-safe saves (operation idempotency ledger, 409/410)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T06"
```

---

### Task 7: Sessions (lifecycle, terminal semantics, resume, prompts)

**Files:**
- Create: `mnemonic-v2/internal/memory/sessions.go`
- Modify: `mnemonic-v2/internal/memory/pipeline.go` (the step-10 `bus.Publish` call site lands here — Task 5 wrote the eventbus, this task wires the publish)
- Test: `mnemonic-v2/internal/memory/sessions_test.go`, `mnemonic-v2/internal/memory/prompts_test.go`

**Interfaces:**
- Consumes: Task 1's `sessions`/`session_finalizations`/`user_prompts` tables + `syncid`, Task 5's `Service`.
- Produces:
  ```go
  type SessionRequest struct{ ID, Project, Directory, OwnershipMode string; Resume, Isolated bool }
  type SessionResult struct{ ID, SyncID string; Continued bool } // Continued: true when a :resume:N continuation was created
  func (s *Service) SessionStart(ctx context.Context, r SessionRequest) (SessionResult, error)
  func (s *Service) SessionEnd(ctx context.Context, sessionID, summary string) error // 409 if already ended
  type Summary struct{ Goal, Instructions, Discoveries, Accomplished, NextSteps, RelevantFiles string }
  func (s *Service) SessionSummary(ctx context.Context, sessionID string, sum Summary) (int64, error) // writes the session_summary observation; does NOT end the session
  func (s *Service) SessionFinalize(ctx context.Context, sessionID string, sum Summary) (SessionFinalizeResult, error)
  type SessionFinalizeResult struct{ SummaryID int64; AlreadyFinalized bool }
  // SavePrompt(ctx, sessionID, content, project, sourceInboxID string) (int64, error) — created in Task 5
  // (the pipeline's capture path calls it); this task adds the session surface above it.
  ```
- Seam: same as Task 5 (the `Service` methods).
- Deletion test: deleting sessions.go re-creates the terminal/resume/lease logic in every save and every transport — the callers are `pipeline.go` (liveness), `transport/handler` (Task 13).
- Adapters: in-process.

**SATISFIES:** [session-start-lease], [session-terminal-409], [session-resume-continuation], [session-finalize-atomic-idempotent], [session-summary-six-section], [prompt-capture], [save-event-exactly-once]

**Note:** this task lands the step-10 `bus.Publish(SaveEvent{…})` call site (Task 5 wrote the stub) and Task 12's `[save-event-exactly-once]` scenario turns green here — Task 12 then proves the remaining two event truths (`[wiki-failure-never-fails-save]`, `[memory-does-not-import-wiki]`) without re-touching the pipeline.

- [ ] **Step 1: Write the failing tests**

```go
func TestSessionStartSetsLease(t *testing.T) {
    svc := newTestService(t)
    r, err := svc.SessionStart(ctx, SessionRequest{ID: "abc", Project: "p", Directory: "/d", OwnershipMode: "shared"})
    if err != nil { t.Fatal(err) }
    if r.ID != "abc" { t.Fatalf("id must be byte-exact, got %q", r.ID) }
    // lease ≈ now+30min, local-only (verify via a store-level read in the test helper)
    if svc.leaseOf(t, "abc") == 0 { t.Fatalf("lease not set") }
}

func TestSessionEndedIsTerminal(t *testing.T) {
    svc := newTestService(t)
    _, _ = svc.SessionStart(ctx, SessionRequest{ID: "a", Project: "p", Directory: "/d"})
    if err := svc.SessionEnd(ctx, "a", "done"); err != nil { t.Fatal(err) }
    if err := svc.SessionEnd(ctx, "a", "again"); !errors.Is(err, memory.ErrSessionAlreadyEnded) {
        t.Fatalf("second end must be 409, got %v", err)
    }
    // saves into an ended session are refused (terminal)
    if _, err := svc.Save(ctx, SaveRequest{SessionID: "a", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "project"}); err == nil {
        t.Fatalf("save into ended session must fail")
    }
}

func TestSessionResumeCreatesContinuation(t *testing.T) {
    svc := newTestService(t)
    _, _ = svc.SessionStart(ctx, SessionRequest{ID: "root", Project: "p", Directory: "/d"})
    _ = svc.SessionEnd(ctx, "root", "")
    r, err := svc.SessionStart(ctx, SessionRequest{ID: "root", Project: "p", Directory: "/d", Resume: true})
    if err != nil { t.Fatal(err) }
    if r.ID != "root:resume:1" || !r.Continued { t.Fatalf("continuation id/flag wrong: %+v", r.ID) }
    r2, _ := svc.SessionStart(ctx, SessionRequest{ID: "root", Project: "p", Directory: "/d", Resume: true})
    if r2.ID != "root:resume:2" { t.Fatalf("second continuation must increment: %q", r2.ID) }
}

func TestSessionFinalizeAtomicAndIdempotent(t *testing.T) {
    svc := newTestService(t)
    _, _ = svc.SessionStart(ctx, SessionRequest{ID: "f", Project: "p", Directory: "/d"})
    sum := Summary{Goal: "g", Instructions: "i", Discoveries: "d", Accomplished: "a", NextSteps: "n", RelevantFiles: "r"}
    f1, err := svc.SessionFinalize(ctx, "f", sum)
    if err != nil { t.Fatal(err) }
    if f1.AlreadyFinalized || f1.SummaryID == 0 { t.Fatalf("first finalize wrong: %+v", f1) }
    f2, err := svc.SessionFinalize(ctx, "f", sum)
    if err != nil || !f2.AlreadyFinalized { t.Fatalf("retry must return already_finalized: %+v %v", f2, err) }
    // the summary observation exists exactly once
    if n := svc.countSummaryObs(t, "f"); n != 1 { t.Fatalf("summary obs count = %d, want 1", n) }
}

func TestPromptCaptureOnSave(t *testing.T) {
    svc := newTestService(t)
    _, _ = svc.SessionStart(ctx, SessionRequest{ID: "pr", Project: "p", Directory: "/d"})
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "pr", Type: "learning", Title: "t", Content: "c",
        Project: "p", Scope: "project", CapturePrompt: true, CurrentPrompt: "how do I ship this?"})
    _ = r
    // the prompt row exists and is searchable (search itself is Task 12; here: direct store read)
    if n := svc.countPrompts(t, "pr"); n != 1 { t.Fatalf("prompt not captured, n=%d", n) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestSession|TestPromptCapture' -v`
Expected: FAIL — methods do not exist. BDD: `[session-*]`, `[prompt-capture]` RED.

- [ ] **Step 3: Write minimal implementation**

`sessions.go`: **SessionStart** — insert `sessions` row (byte-exact `id`, minted `sync_id` prefix `ses-`, `ownership_mode` default `shared`, `runtime_lease_expires_at = now+30min` — local-only, never synced, spec `01-memory/002` §7); when `Resume: true`, in the same write transaction select an existing non-ended `<id>:resume:N` or create `<id>:resume:(max+1)` (Engram's atomic continuation; the root stays terminal). **SessionEnd** — `UPDATE sessions SET ended_at=? WHERE id=? AND ended_at IS NULL`; zero rows + an ended row exists → `ErrSessionAlreadyEnded` (409). **SessionSummary** — insert a `type=session_summary` observation whose content is the six-section structure exactly (`## Goal` / `## Instructions` / `## Discoveries` / `## Accomplished` / `## Next Steps` / `## Relevant Files`, all six present — a red-flag check per spec `01-memory/002` §5); it does NOT set `ended_at`. **SessionFinalize** — one transaction: (1) insert the summary observation if not present, (2) set `ended_at`, (3) enqueue sync mutations (session + summary), (4) upsert `session_finalizations(session_id, summary_id, summary_sync_id, created_at)`; a retry finds the row → `AlreadyFinalized: true` with no double-close. The pipeline's step-2 session policy (fixed in Task 5, **tested here for real**): missing session row → auto-register (`shared`, resolved project) inside the save transaction; ended row → `ErrSessionEnded`; `project_owned` mismatch → `ErrSessionProjectMismatch` (cross-project writes refused, spec `01-memory/002` §4). And the step-10 publish call site lands here: `bus.Publish(SaveEvent{SyncID, Project, Scope, Type, TopicKey, Entity: "", Importance: <step-6 value>, Degraded})` after commit — `Entity` is filled in Task 12 (the payload-completion task); the event is emitted exactly once per committed save from this task on.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/
git commit -m "feat(mnemonic-v2): session lifecycle (terminal, resume, atomic finalize, six-section summary), prompt capture

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T07"
```

---

### Task 8: Dedup LLM path (primary Classify + trust-but-verify)

**Files:**
- Modify: `mnemonic-v2/internal/memory/dedup.go` (the LLM path fills the non-nil seam branch of `dedup.Run`; the hash floor stays the degraded branch)
- Test: `mnemonic-v2/internal/memory/dedup_llm_test.go`

**Interfaces:**
- Consumes: Task 3's `llm.LLM.Classify` + `degrade.Run`, Task 5's `dedup.Run` skeleton (nil-seam floor path).
- Produces: the non-nil-seam branch of `dedup.Run` (signature unchanged from Task 5): `preFilterCandidates(ctx, st, project, scope, limit=20) ([]string, error)` (spec `004-dedup` §3 — recent-window, most-recent first, no title query; distinct from the conflict-candidate FTS filter of Task 9) → `LLM.Classify` → **trust-but-verify**: a positive `CandidateID` that is not a live row in this project → the most-recent live candidate (head of pre-filter); verdict trusted, pointer verified. The pipeline is untouched — its call site was final since Task 5.
- Seam: the `LLM` seam (Task 3).
- Deletion test: deleting the LLM path reverts v2 to v1 (hash default) — the LLM-first inversion (spec `004-dedup` §1) is lost.
- Adapters: OpenAI-compat client (prod) + stub LLM (tests) = 2 real + floor.

**SATISFIES:** [dedup-llm-primary], [dedup-trust-but-verify], [dedup-noop-first-class]

- [ ] **Step 1: Write the failing tests**

```go
func TestDedupLLMPrimaryWhenSeamPresent(t *testing.T) {
    svc := newTestServiceWithLLM(t, &stubLLM{}) // classify verdict decided per test below via stub.set()
    live := seedObservation(t, svc.st, "p", "decision", "old title", "old content") // the only row → head of pre-filter
    svc.stub().setClassify(llm.DedupDecision{Verdict: llm.VerdictUpdate, CandidateID: live.ID})
    r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "new", Content: "same thing reworded", Project: "p", Scope: "project"})
    if err != nil { t.Fatal(err) }
    if r.Dedup.Verdict != "update" || r.Dedup.Reason != "llm" || r.ID != live.ID {
        t.Fatalf("LLM verdict must win and merge: %+v", r)
    }
    if r.Degraded { t.Fatalf("seam present → not degraded") }
}

func TestDedupTrustButVerifyCandidateID(t *testing.T) {
    svc := newTestServiceWithLLM(t, &stubLLM{})
    head := seedObservation(t, svc.st, "p", "decision", "head", "head content") // most recent = head of pre-filter
    svc.stub().setClassify(llm.DedupDecision{Verdict: llm.VerdictUpdate, CandidateID: 999}) // 999 is not a live row
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
    if r.ID != head.ID { t.Fatalf("stale CandidateID must fall back to most-recent live candidate: got id %d want %d", r.ID, head.ID) }
}

func TestDedupNoopIsFirstClass(t *testing.T) {
    svc := newTestServiceWithLLM(t, &stubLLM{})
    svc.stub().setClassify(llm.DedupDecision{Verdict: llm.VerdictNoop, CandidateID: 1})
    before := countObs(t, svc.st)
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
    if r.Dedup.Verdict != "noop" { t.Fatalf("noop verdict expected") }
    if after := countObs(t, svc.st); after != before { t.Fatalf("noop must write nothing") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestDedupLLM|TestDedupTrust|TestDedupNoop' -v`
Expected: FAIL — the LLM path is not yet consulted (Task 5 always ran the floor). BDD: `[dedup-llm-primary]`, `[dedup-trust-but-verify]` RED.

- [ ] **Step 3: Write minimal implementation**

`dedup.go`'s `dedup.Run` gains its non-nil branch (the nil-seam floor branch is unchanged from Task 5):
```
if s == nil:  return hashFloor(newContent, project, scope, typ, title), true    // degraded, no retry (absent ≠ failing)
candidates = preFilterCandidates(ctx, st, project, scope, 20)                   // most-recent-first, non-deleted, ≤20 (spec 004-dedup §3)
decision, degraded = degrade.Run(func() (llm.DedupDecision, error) {
    return s.Classify(ctx, newContent, candidates)
}, func() llm.DedupDecision { return hashFloor(newContent, project, scope, typ, title) }, "dedup-classify")
if decision.Reason == "" { decision.Reason = "llm" }                            // seam succeeded
// trust-but-verify (spec 004-dedup §4): a positive CandidateID that is not a live
// row in this project → most-recent live candidate (head of pre-filter). Verdict trusted; pointer verified.
```
Pipeline step 6 (c) executes the verdict (unchanged from Task 5): `update(CandidateID)` → UPDATE the target in place (`revision_count++`, version snapshot, `MergedIntoSyncID` set); `delete(CandidateID)` → mark the target `superseded_by_sync_id = new sync_id`, soft-delete it, store the new; `noop` → no write (first-class); `add` → INSERT. The pre-filter's 20-cap and the distinct-from-conflict-filter rule (spec `004-dedup` §3 note) are asserted by a test that the pre-filter returns rows in `created_at DESC` order and never more than 20.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/
git commit -m "feat(mnemonic-v2): LLM-first dedup (primary Classify, trust-but-verify, recent-window pre-filter)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T08"
```

---

### Task 9: Relations — conflict-candidate detection + relevance gate

**Files:**
- Create: `mnemonic-v2/internal/memory/relations.go`
- Modify: `mnemonic-v2/internal/memory/pipeline.go` (step 4 pre-filter + step 7.5 materialize candidates; `SaveResult.Candidates` / `JudgmentRequired` filled)
- Test: `mnemonic-v2/internal/memory/relations_test.go`

**Interfaces:**
- Consumes: Task 1's `memory_relations` table + `Store.AddRelation`, Task 5's pipeline.
- Produces:
  ```go
  // relations.go
  func candidateGate(a, b TitleAndTopic) bool  // the deterministic relevance gate (spec 01-memory/003 §3 step 2)
  func (s *Service) detectConflictCandidates(ctx context.Context, saved Observation, title string) ([]Relation, error)
  type Relation struct{ ID int64; SyncID, SourceID, TargetID, Relation string; Reason, Evidence string; Confidence float64; JudgmentStatus string; MarkedByActor, MarkedByKind, MarkedByModel string }
  ```
  - `candidateGate` terms: case-insensitive letter/digit words; `+`/`#` preserved inside identifiers (`C++` ≠ `C#`); ignore common connectives + generic change verbs (`updated`, `fixed`, `added`, `removed`, `changed`, `and`, `the`, `a`, `an`, `to`, `for`, `of`); accept iff shared nonempty `topic_key` OR ≥2 distinct significant shared title terms OR both titles reduce to the same single significant term. Content-only matches do not establish relevance.
- Seam: same as Task 5.
- Deletion test: deleting this logic re-introduces a silent save with no conflict surfacing — the agent loses the `judgment_required` contract (spec `004-transport-mcp` §3 `mem_save` row).
- Adapters: in-process.

**SATISFIES:** [conflict-candidates-surfaced], [relevance-gate-terms]

- [ ] **Step 1: Write the failing tests**

```go
func TestRelevanceGateTerms(t *testing.T) {
    cases := []struct{ a, b string; want bool }{
        {"auth-model redesign", "auth-model v2", true},                       // shared topic-ish term "auth" + "model" (2 terms)
        {"C++ linker flags", "C# linker flags", true},                         // C++ ≠ C# (+ vs # preserved) but "linker"+"flags" = 2 shared significant terms → accept
        {"updated the parser", "fixed the parser", true},                      // connectives/verbs ignored → both reduce to "parser" → same single term
        {"database pool", "database pool sizing", true},                       // 2 shared
        {"database pool", "auth cache", false},                                // 0 shared
        {"", "auth", false},                                                   // empty side
    }
    for _, c := range cases {
        if got := candidateGate(titleTopic{Title: c.a}, titleTopic{Title: c.b}); got != c.want {
            t.Errorf("gate(%q,%q)=%v want %v", c.a, c.b, got, c.want)
        }
    }
}
The table is hand-verified against the spec's §3 step 2 definition before coding: terms are case-insensitive letter/digit words with `+`/`#` preserved inside identifiers (`C++` and `C#` are distinct single terms); ignored words are the connectives + generic change verbs listed in Step 3. Row by row: `auth-model` shares `auth`+`model` (2) → true; `C++ linker flags` vs `C# linker flags` shares `linker`+`flags` (2) → true (the `+`/`#` preservation is what keeps `C++`≠`C#` from being a third shared term); `updated the parser`/`fixed the parser` both reduce to the single term `parser` → true; `database pool` vs `database pool sizing` shares `database`+`pool` → true; `database pool` vs `auth cache` shares nothing → false; empty side → false.

```go
func TestConflictCandidatesMaterializedAsPending(t *testing.T) {
    svc := newTestService(t)
    seed := seedObservation(t, svc.st, "p", "decision", "auth-model redesign", "use bearer tokens", "architecture/auth-model")
    r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth-model v2",
        Content: "switch to mTLS", Project: "p", Scope: "project", TopicKey: "architecture/auth-model"})
    if err != nil { t.Fatal(err) }
    if len(r.Candidates) != 1 || !r.JudgmentRequired { t.Fatalf("expected 1 pending candidate: %+v", r.Candidates) }
    if r.Candidates[0].Relation != "pending" || r.Candidates[0].SourceID != r.SyncID || r.Candidates[0].TargetID != seed.SyncID {
        t.Fatalf("candidate shape wrong: %+v", r.Candidates[0])
    }
    // the pending row is queryable (judged later by Task 10's mem_judge)
    if n := svc.countPendingRelations(t, r.SyncID); n != 1 { t.Fatalf("pending relation not materialized") }
}

func TestNoCandidateWhenGateFails(t *testing.T) {
    svc := newTestService(t)
    seedObservation(t, svc.st, "p", "decision", "database pool sizing", "tune max conns", "")
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth cache", Content: "ttl", Project: "p", Scope: "project"})
    if len(r.Candidates) != 0 { t.Fatalf("gate must reject: %+v", r.Candidates) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestRelevanceGate|TestConflictCandidates|TestNoCandidate' -v`
Expected: FAIL — `detectConflictCandidates` does not exist; `SaveResult.Candidates` is empty. BDD: `[conflict-candidates-surfaced]`, `[relevance-gate-terms]` RED.

- [ ] **Step 3: Write minimal implementation**

Pipeline step 4 (pre-filter for conflict candidates — the *different* filter from dedup's recent-window, spec `004-dedup` §3 note): OR-based FTS on the saved **title** within same project+scope (the FTS `sanitizeFTSCandidates` OR mode, Engram `relations.go:1158` pattern), raw BM25 order, bounded per-source (default 3). Then the gate (`candidateGate` above). Survivors are materialized as `pending` `memory_relations` rows (`source_id` = the new save's sync_id, `target_id` = candidate's sync_id, `relation = 'pending'`, `judgment_status = 'pending'`, `session_id` = the save's session) via `Store.AddRelation` inside the commit transaction. `SaveResult.Candidates` = the materialized rows; `JudgmentRequired = len(candidates) > 0`. The gate's term tokenizer is a small pure function (split on non letter/digit, keeping `+`/`#` attached to the preceding run) — unit-tested in isolation.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/
git commit -m "feat(mnemonic-v2): conflict-candidate detection (lexical pre-filter + relevance gate → pending relations)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T09"
```

---

### Task 10: Judgment (mem_judge / mem_judge_many / mem_compare) + consolidate + scan

**Files:**
- Modify: `mnemonic-v2/internal/memory/relations.go` (judge/judge_many/compare/consolidate/scan)
- Test: `mnemonic-v2/internal/memory/judge_test.go`, `mnemonic-v2/internal/memory/consolidate_test.go`

**Interfaces:**
- Consumes: Task 3's `llm.LLM.Judge`/`Compare` + `degrade.Run`, Task 9's pending rows, Task 1's `memory_relations` + `ConsolidateObservations`.
- Produces:
  ```go
  type JudgeRequest struct{ JudgmentID int64; Relation, Reason, Evidence string; Confidence float64 }
  func (s *Service) Judge(ctx context.Context, r JudgeRequest) (Relation, error)              // verdict on a pending candidate; sets judgment_status=judged + marked_by_* provenance
  func (s *Service) JudgeMany(ctx context.Context, rs []JudgeRequest) ([]Relation, error)     // 1–32, one queued write (one transaction)
  func (s *Service) Compare(ctx context.Context, aID, bID int64) (Relation, error)            // LLM compare → judged relation directly; idempotent per pair
  type ConsolidateRequest struct{ IDsToDelete []int64; NewTitle, NewContent, ExpectedProject string; Type, TopicKey, Fingerprint string; Confirm bool }
  func (s *Service) Consolidate(ctx context.Context, r ConsolidateRequest) (ConsolidateResult, error)
  type ConsolidateResult struct{ Plan *ConsolidatePlan; Survivor *store.Observation }
  type ConsolidatePlan struct{ Sources []store.Observation; SurvivorID int64; Project string; Fingerprint string }
  func (s *Service) ScanProject(ctx context.Context, project string) (int, error)             // re-run candidate detection; return new pending count
  ```
  - Sentinel errors: `ErrConsolidateFingerprintMismatch`, `ErrConsolidateRefused` (missing/deleted sources, duplicate ids, cross-project, provenance cycle, unresolvable project).
- Seam: the `LLM` seam for `Compare`; `Judge`/`JudgeMany` record the *agent's* verdict (the engine stores, the agent judges — INV-7; the LLM `Judge` method is available for the agent-side tooling but the MCP `mem_judge` records what the agent sent).
- Deletion test: deleting judgment logic loses the conflict/supersession structure the wiki reads (spec `01-memory/003` §4: "the wiki reads judged relations as first-class content").
- Adapters: in-process.

**SATISFIES:** [judge-sets-provenance], [judge-many-one-write], [compare-idempotent], [consolidate-preview-confirm], [consolidate-fingerprint-mismatch], [scan-resurfaces]

- [ ] **Step 1: Write the failing tests**

```go
func TestJudgeSetsProvenance(t *testing.T) {
    svc := newTestService(t)
    cand := seedPendingRelation(t, svc.st, "p", "s") // helper: two obs + a pending relation, returns the relation
    rel, err := svc.Judge(ctx, JudgeRequest{JudgmentID: cand.ID, Relation: "conflicts_with",
        Reason: "mutually exclusive", Evidence: "see diff", Confidence: 0.9})
    if err != nil { t.Fatal(err) }
    if rel.JudgmentStatus != "judged" || rel.MarkedByKind == "" || rel.MarkedByModel == "" {
        t.Fatalf("provenance not set: %+v", rel) // INV-1
    }
}

func TestJudgeManyOneTransaction(t *testing.T) {
    svc := newTestService(t)
    ids := seedPendingRelations(t, svc.st, "p", "s", 5)
    before := txnCount(t, svc) // helper: counts committed txns via a store hook
    _, err := svc.JudgeMany(ctx, makeJudgeRequests(ids, "not_conflict"))
    if err != nil { t.Fatal(err) }
    if got := txnCount(t, svc); got != before+1 { t.Fatalf("judge_many must be ONE write, txns %d→%d", before, got) } // txnCount = the store's SetTxnHook counter
    if _, err := svc.JudgeMany(ctx, makeJudgeRequests(ids, "not_conflict")[:0]); err == nil { t.Fatalf("0 judgments must fail (1–32)") }
    if _, err := svc.JudgeMany(ctx, makeJudgeRequests(ids33(t), "not_conflict")); err == nil { t.Fatalf("33 judgments must fail (1–32)") }
}

func TestCompareCreatesJudgedRelationIdempotently(t *testing.T) {
    svc := newTestServiceWithLLM(t, &stubLLM{})
    svc.stub().setCompare(llm.Judgment{Relation: "compatible", Confidence: 0.8, Evidence: "e"})
    a := seedObservation(t, svc.st, "p", "decision", "a", "A", "")
    b := seedObservation(t, svc.st, "p", "decision", "b", "B", "")
    r1, _ := svc.Compare(ctx, a.ID, b.ID)
    r2, _ := svc.Compare(ctx, a.ID, b.ID)
    if r1.JudgmentStatus != "judged" || r1.Relation != "compatible" { t.Fatalf("compare must create judged relation: %+v", r1) }
    if r2.ID != r1.ID { t.Fatalf("compare must be idempotent per pair: %d vs %d", r1.ID, r2.ID) }
}

func TestConsolidatePreviewThenConfirm(t *testing.T) {
    svc := newTestService(t)
    a := seedObservation(t, svc.st, "p", "decision", "a", "A", "")
    b := seedObservation(t, svc.st, "p", "decision", "b", "B", "")
    prev, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
        NewTitle: "merged", NewContent: "C", ExpectedProject: "p", Confirm: false})
    if err != nil { t.Fatal(err) }
    if len(prev.Plan.Sources) != 2 || prev.Plan.Fingerprint == "" { t.Fatalf("plan wrong: %+v", prev.Plan) }
    if n := liveObsCount(t, svc.st); n != 2 { t.Fatalf("preview must write nothing") }
    conf, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
        NewTitle: "merged", NewContent: "C", ExpectedProject: "p", Fingerprint: prev.Plan.Fingerprint, Confirm: true})
    if err != nil { t.Fatal(err) }
    if conf.Survivor == nil || conf.Survivor.Title != "merged" { t.Fatalf("survivor wrong") }
    if !strings.Contains(conf.Survivor.Content, a.SyncID) || !strings.Contains(conf.Survivor.Content, b.SyncID) {
        t.Fatalf("survivor must carry consolidated_from footer with source sync_ids")
    }
    sa, _ := svc.st.GetObservation(ctx, a.ID, false)
    if sa.SupersededBySyncID == nil || *sa.SupersededBySyncID != conf.Survivor.SyncID || sa.DeletedAt == nil {
        t.Fatalf("sources must be soft-deleted with superseded_by_sync_id (structured provenance)")
    }
}

func TestConsolidateFingerprintMismatch(t *testing.T) {
    svc := newTestService(t)
    a := seedObservation(t, svc.st, "p", "decision", "a", "A", "")
    b := seedObservation(t, svc.st, "p", "decision", "b", "B", "")
    prev, _ := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID}, NewTitle: "m", NewContent: "C", ExpectedProject: "p"})
    // mutate a source between preview and confirm
    _ = svc.st.UpdateObservationForProject(ctx, a.ID, "p", store.UpdateParams{Content: ptr("A changed")})
    _, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID}, NewTitle: "m", NewContent: "C",
        ExpectedProject: "p", Fingerprint: prev.Plan.Fingerprint, Confirm: true})
    if !errors.Is(err, memory.ErrConsolidateFingerprintMismatch) { t.Fatalf("want fingerprint mismatch, got %v", err) }
    // refusal cases
    for _, bad := range []ConsolidateRequest{
        {IDsToDelete: []int64{a.ID}, NewTitle: "m", NewContent: "C", ExpectedProject: "p"},          // <2 ids
        {IDsToDelete: []int64{a.ID, a.ID}, NewTitle: "m", NewContent: "C", ExpectedProject: "p"},     // duplicate ids
        {IDsToDelete: []int64{a.ID, b.ID}, NewTitle: "m", NewContent: "C", ExpectedProject: "other"}, // cross-project
    } {
        if _, err := svc.Consolidate(ctx, bad); err == nil { t.Fatalf("must refuse %+v", bad) }
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestJudge|TestCompare|TestConsolidate' -v`
Expected: FAIL — methods do not exist. BDD: `[judge-*]`, `[compare-idempotent]`, `[consolidate-*]` RED.

- [ ] **Step 3: Write minimal implementation**

**Judge** — load the pending relation by id (404 if absent or already judged → idempotent no-op returning the existing row); `UPDATE memory_relations SET relation=?, reason=?, evidence=?, confidence=?, judgment_status='judged', marked_by_actor='agent', marked_by_kind='mcp', marked_by_model=?, updated_at=?` (`marked_by_model` = the agent-supplied model name or `""`; INV-1 provenance). **JudgeMany** — validate 1 ≤ len ≤ 32, run all updates in **one transaction** (the write queue serializes it at the transport layer, Task 13). **Compare** — `degrade.Run(s.llm.Compare, floor: no relation created, "compare")`; on a non-degraded judgment, INSERT a judged relation (system provenance `marked_by_kind='system'`), idempotent: an existing judged row for the unordered pair (either direction) → return it. **Consolidate** — port Engram's two-phase fingerprint-gated contract (spec `01-memory/003` §5): preview builds the plan in a read transaction (`sha256` hex over `{"version":"consolidate:v2","sources":[…sorted by ID, full Observation],"survivor":{…}}`); confirm re-resolves inside the mutation transaction, compares fingerprints → mismatch = `ErrConsolidateFingerprintMismatch` + fresh plan, no writes; survivor inherits the newest source's session/type/scope (`topic_key` explicit-only), content gets the `## consolidated_from` footer listing source sync_ids; sources soft-deleted with `superseded_by_sync_id`; all in one local transaction; refusals before writes (missing/deleted sources, duplicate ids, cross-project selection, provenance cycle, unresolvable project). **ScanProject** — page-batched re-run of Task 9's detection over the project, inserting only *new* pending relations (a pair with any existing relation row — pending or judged — is skipped), returning the new count (spec `01-memory/003` §7 — the post-pull "are there conflicts I haven't judged?" pass).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/
git commit -m "feat(mnemonic-v2): judgment (judge/judge_many/compare), fingerprint-gated consolidation, project scan

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T10"
```

---

### Task 11: Lookup ladder (context / search / get) + wiki-first mem_lookup + RRF + vector leg

**Files:**
- Create: `mnemonic-v2/internal/memory/lookup.go`
- Modify: `mnemonic-v2/internal/core/store/sqlite.go` (`VectorSearch` real via the vec0 leg; `Context` real), `mnemonic-v2/internal/core/store/types.go` (`SearchQuery`, `VecQuery`, `ContextQuery` final shapes)
- Test: `mnemonic-v2/internal/memory/lookup_test.go`, `mnemonic-v2/internal/core/store/vector_test.go`

**Interfaces:**
- Consumes: Task 1's `Store.Search`/`VectorSearch`/`Context`/`SearchPrompts`/`BumpRetrievalUsage`/`MarkReviewed`, Task 3's `Embedder`, Task 7's prompts.
- Produces:
  ```go
  type SearchFilters struct{ Type, Project, Scope, MatchMode string; Limit int; AllProjects bool }
  type SearchHit struct{ store.Observation; Preview string; Truncated bool; Rank float64; Annotations []string } // Annotations: "supersedes: #id (title)" etc. (spec 01-memory/003 §6)
  func (s *Service) Search(ctx context.Context, query string, f SearchFilters) (SearchResult, error) // RRF-fuses FTS + vector when armed; records trail
  type LookupResult struct{ Hit *SearchHit; Degraded bool }
  func (s *Service) Lookup(ctx context.Context, project, scope, topicKey string, id int64, phrase string) (LookupResult, error)
  func (s *Service) Context(ctx context.Context, budgetBytes int, olderHistory bool) (store.ContextResult, error)
  func (s *Service) SearchPrompts(ctx context.Context, query, project string, limit int) (store.PromptSearchResult, error)
  func (s *Service) Review(ctx context.Context, project string, list bool, markReviewedID int64) (ReviewResult, error) // list overdue (state=needs_review) / MarkReviewed (restarts the decay clock)
  func (s *Service) RecordTrail(route, query string, filters SearchFilters, ids []int64, degraded bool) // INV-5 — and BumpRetrievalUsage(ids) in the same step
  ```
  - Ranking (spec `01-memory/005` §3, Engram `store.go:424-431`): BM25(title 5.0, content 1.0, topic_key 3.0) × `(1 + pinned·0.10 + 0.06·1/(1+age_days/30) + 0.04·(rev+dups)/(rev+dups+4))`, age from `last_seen_at || updated_at || created_at`.
  - Vector leg: `Store.VectorSearch` = vec0 (`modernc.org/sqlite/vec` blank import, v1-proven) `vec_distance_cosine` over `observations.embedding` rows, k = `2·limit`; RRF fusion `score = 1/(60+rank_fts) + 1/(60+rank_vec)`; `degraded: true` when the vector leg is absent (no embedder or no vectors) — never a silent fusion (INV-3).
- Seam: same as Task 5.
- Deletion test: deleting lookup.go re-creates ranking + fusion + trail recording in every transport — the callers are `transport/handler` (Task 13) and the CLI (Task 16).
- Adapters: in-process.

**SATISFIES:** [search-ranking-boosts], [search-compact-rows], [lookup-topic-key-exact], [lookup-phrase-bounded], [context-byte-budget], [vector-blend-degraded], [trail-recorded], [prompt-search-finds-captured], [usage-raises-importance], [review-mark-reviewed-restarts-clock]

- [ ] **Step 1: Write the failing tests**

```go
func TestSearchRankingWithBoosts(t *testing.T) {
    svc := newTestService(t)
    base := seedObservation(t, svc.st, "p", "decision", "auth", "token auth", "architecture/auth")
    pinned := seedObservation(t, svc.st, "p", "decision", "auth", "token auth pinned", "architecture/auth")
    _ = svc.st.UpdateObservationForProject(ctx, pinned.ID, "p", store.UpdateParams{Pinned: ptr(true)}) // pinned +0.10 boost
    r, _ := svc.Search(ctx, "auth", SearchFilters{Project: "p", Limit: 10})
    if len(r.Hits) < 2 { t.Fatalf("expected both rows") }
    if r.Hits[0].ID != pinned.ID { t.Fatalf("pinned row must rank first, got %d", r.Hits[0].ID) }
    // compact rows: preview ≤300 chars, never the full body
    long := seedObservation(t, svc.st, "p", "discovery", "auth long", strings.Repeat("x", 500), "")
    _ = long
    r2, _ := svc.Search(ctx, "auth long", SearchFilters{Project: "p", Limit: 10})
    hit := findHit(r2, long.ID)
    if len(hit.Preview) > 300 || !hit.Truncated { t.Fatalf("preview must be ≤300 + truncated flag") }
}

func TestLookupTopicKeyExact(t *testing.T) {
    svc := newTestService(t)
    seedObservation(t, svc.st, "p", "decision", "auth model", "the model", "architecture/auth-model")
    res, err := svc.Lookup(ctx, "p", "project", "architecture/auth-model", 0, "")
    if err != nil || res.Hit == nil { t.Fatalf("topic_key lookup must find the row") }
    resNone, _ := svc.Lookup(ctx, "p", "project", "architecture/absent", 0, "")
    if resNone.Hit != nil { t.Fatalf("absent topic_key must return none (not widen)") }
}

func TestLookupPhraseBounded(t *testing.T) {
    svc := newTestService(t)
    for i := 0; i < 10; i++ {
        seedObservation(t, svc.st, "p", "discovery", fmt.Sprintf("needle %d", i), "unique needle phrase", "")
    }
    res, _ := svc.Lookup(ctx, "p", "project", "", 0, "unique needle phrase")
    if len(res.Hits) > 5 { t.Fatalf("phrase lookup must cap at 5, got %d", len(res.Hits)) }
}

func TestContextRespectsBudget(t *testing.T) {
    svc := newTestService(t)
    _, _ = svc.SessionStart(ctx, SessionRequest{ID: "ctx", Project: "p", Directory: "/d"})
    for i := 0; i < 50; i++ {
        svc.Save(ctx, SaveRequest{SessionID: "ctx", Type: "learning", Title: fmt.Sprintf("t%d", i), Content: strings.Repeat("c", 200), Project: "p", Scope: "project"})
    }
    res, err := svc.Context(ctx, 16*1024, false)
    if err != nil { t.Fatal(err) }
    if sizeOf(t, res) > 16*1024 { t.Fatalf("context must respect the byte budget") }
    // pinned ≤20, pinned first
    // (pin 25 rows, assert only ≤20 appear and they lead — helper does the pinning)
}

func TestVectorLegDegradesExplicitly(t *testing.T) {
    svc := newTestService(t) // no embedder
    seedObservation(t, svc.st, "p", "decision", "auth", "tokens", "")
    res, _ := svc.Search(ctx, "auth", SearchFilters{Project: "p", Limit: 10})
    if !res.Degraded { t.Fatalf("no embedder → vector leg degraded must be true") }
    if len(res.Hits) == 0 { t.Fatalf("FTS leg must still return results") }
}

func TestTrailRecorded(t *testing.T) {
    svc := newTestService(t)
    seedObservation(t, svc.st, "p", "decision", "auth", "tokens", "")
    _, _ = svc.Search(ctx, "auth", SearchFilters{Project: "p"})
    trails := svc.trails(t)
    if len(trails) != 1 || trails[0].Route != "fts" || trails[0].Query != "auth" || trails[0].Degraded != true {
        t.Fatalf("trail wrong: %+v", trails)
    }
}

func TestPromptSearchFindsCaptured(t *testing.T) {
    svc := newTestService(t)
    _, _ = svc.SessionStart(ctx, SessionRequest{ID: "ps", Project: "p", Directory: "/d"})
    svc.Save(ctx, SaveRequest{SessionID: "ps", Type: "learning", Title: "t", Content: "c",
        Project: "p", Scope: "project", CurrentPrompt: "how do I ship the sync layer?"}) // capture default-on (nil)
    res, err := svc.SearchPrompts(ctx, "ship the sync layer", "p", 5)
    if err != nil || len(res.Prompts) != 1 { t.Fatalf("captured prompt not found by prompt search: %+v %v", res, err) }
}

func TestUsageRaisesImportance(t *testing.T) {
    svc := newTestService(t)
    a, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "same", Content: "same body", Project: "p", Scope: "project"})
    _ = a
    // retrieve it 10 times (the trail writer bumps retrieval_usage), then re-save a sibling and compare
    for i := 0; i < 10; i++ { svc.Search(ctx, "same", SearchFilters{Project: "p"}) }
    used, _ := svc.st.GetObservation(ctx, a.ID, false)
    if used.RetrievalUsage != 10 { t.Fatalf("retrieval_usage not accumulated: %d", used.RetrievalUsage) }
    fresh := Importance(0, 0, "decision")
    hot := Importance(used.RetrievalUsage, 0, "decision")
    if hot <= fresh { t.Fatalf("retrieved observation must score higher: hot=%v fresh=%v", hot, fresh) }
}

func TestReviewMarkReviewedRestartsClock(t *testing.T) {
    svc := newTestService(t)
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "decay", Content: "c", Project: "p", Scope: "project"})
    // force the row overdue (test hook: review_after in the past)
    _ = svc.st.TouchReviewAfterForTest(ctx, r.ID, "2026-01-01T00:00:00Z")
    got, _ := svc.st.GetObservation(ctx, r.ID, false)
    if got.State() != "needs_review" { t.Fatalf("row must be overdue") }
    res, err := svc.Review(ctx, "p", false, r.ID) // markReviewed
    if err != nil || !res.MarkReviewed { t.Fatalf("mark_reviewed failed: %+v %v", res, err) }
    got2, _ := svc.st.GetObservation(ctx, r.ID, false)
    if got2.State() != "active" { t.Fatalf("mark_reviewed must restart the decay clock (review_after ≈ now + 6mo for decision)") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ ./internal/core/store/ -run 'TestSearchRanking|TestLookup|TestContext|TestVectorLeg|TestTrail' -v`
Expected: FAIL — `lookup.go` and the vector leg do not exist. BDD: `[search-*]`, `[lookup-*]`, `[context-byte-budget]`, `[vector-blend-degraded]`, `[trail-recorded]` RED.

- [ ] **Step 3: Write minimal implementation**

`lookup.go`: **Search** — build the FTS query (weights + boosts + short-term LIKE fallback from Task 2), project to compact rows (id, sync_id, title, type, 300-char `Preview` + `Truncated`, relation annotations from a single `memory_relations` scan over the hit ids — `supersedes:`/`superseded_by:`/`conflicts:`/`conflict: contested by #id (pending)` lines, spec `01-memory/003` §6), exclude soft-deleted; when the embedder is armed: `VectorSearch` (vec0 cosine, k=2·limit) + RRF fusion (route `rrf`); `degraded` when the vector leg is absent; **`RecordTrail` + `st.BumpRetrievalUsage(hitIds)` in the same step** (the trail is the audit record — INV-5; the usage bump is the named source of the importance formula's usage term — spec `002-schema` §2.2.3) before return. **Lookup** — three modes, deliberately dumb (spec `01-memory/005` §5): exact `topic_key` (+project/scope) → the single upserted row via `idx_obs_topic` or none; id/sync_id → full row; phrase → bounded FTS (trigram, ≤5, no boosts, no vector). Lookup also records a trail (route `lookup`) but does NOT bump usage — it is the wiki's precise fallback, and over-counting it would distort graduate-rule input. **Context** — pinned rows first (≤20), then recent sessions + prompts + observations, trimmed to `budgetBytes` (default 16 KiB, max 64 KiB); `olderHistory=true` adds a capped calendar overview of out-of-window observations. **SearchPrompts** — `st.SearchPrompts` (the `prompts_fts` table, Task 1's DDL) over `(content, project)`; redaction happened at capture, search sees the stored (redacted) text. **Review** — `list` = observations with `state() = needs_review` in the project (the `review_after < now` virtual state, spec `002-schema` §2.2.2); `markReviewedID` → `st.MarkReviewed` (reset `review_after = now + type-months` from `cfg.Memory.Review`). `sqlite.go` **VectorSearch** — `CREATE VIRTUAL TABLE obs_vec USING vec0(rowid INTEGER PRIMARY KEY, embedding float[N])` (migration `003_vectors.sql`, N from the embedder's `Dimensions()` — the dimension-mismatch error at first write, spec `006-config` §5), upserted on save when the embedder is armed (pipeline step 6 extension: after INSERT, `emb.Embed` the content → store little-endian float32 BLOB in `observations.embedding` + vec0 row, `embedding_model` + `embedding_created_at` set; embed failure → row saved, no vector, `degraded`); search = `SELECT rowid, vec_distance_cosine(embedding, ?) FROM obs_vec ORDER BY distance LIMIT ?`. `RecordTrail` inserts into a `retrieval_trails` table (migration `003`: `id, query, route, filters json, ids json, degraded, created_at`) — read by `mnemonic trail` (Task 14). `BumpRetrievalUsage` = `UPDATE observations SET retrieval_usage = retrieval_usage + 1 WHERE id IN (…) AND deleted_at IS NULL`. `MarkReviewed` = `UPDATE observations SET review_after = now + type-months` (the months map from `cfg.Memory.Review` — the same source the save pipeline uses). `SearchPrompts` = the `prompts_fts` query projected to `PromptSearchResult{Prompts []store.Prompt}` (S-2: Go types only).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/lookup.go mnemonic-v2/internal/core/store/ mnemonic-v2/internal/memory/lookup_test.go
git commit -m "feat(mnemonic-v2): lookup ladder (context/search/get), wiki-first mem_lookup, RRF vector blend, trails

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T11"
```

---

### Task 12: Save event emission (wiki handoff) + importance on the event

**Files:**
- Modify: `mnemonic-v2/internal/memory/pipeline.go` (step 10 fills the event with `Entity` + `Importance` — Task 7 landed the `Publish` call site; this task completes the payload), `mnemonic-v2/internal/memory/eventbus.go` (consumer failure isolation test seam)
- Test: `mnemonic-v2/internal/memory/eventbus_test.go`

**Interfaces:**
- Consumes: Task 5's `EventBus` + `SaveEvent`, Task 7's publish call site, Task 5's importance computation (step 6).
- Produces: the exact `SaveEvent` contract (spec `004-transport-mcp` §5) — this task is the proof that memory emits and the wiki (a *later* change) can consume without a dependency: the test subscribes a fake consumer in a *separate package* (`internal/wiki_teststub/`, a real package with its own `_test.go`) that imports only the event-bus package + the event type, never `internal/memory`.
- Seam: the event bus (the boundary — spec `000-architecture` §3 eager-per-save handoff).
- Deletion test: deleting the bus inverts the dependency (memory would have to call the wiki directly).
- Adapters: the stub wiki consumer (test) + the real llm-wiki (later change) = 2.

**SATISFIES:** [save-event-exactly-once], [wiki-failure-never-fails-save], [memory-does-not-import-wiki]

- [ ] **Step 1: Write the failing tests**

```go
func TestSaveEventEmittedExactlyOnce(t *testing.T) {
    svc, bus := newTestServiceWithBus(t)
    got := make(chan memory.SaveEvent, 4)
    bus.Subscribe(func(e memory.SaveEvent) { got <- e })
    r, _ := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth model",
        Content: "c", Project: "p", Scope: "project", TopicKey: "architecture/auth-model"})
    e := <-got
    if e.SyncID != r.SyncID || e.Project != "p" || e.TopicKey != "architecture/auth-model" || e.Entity == "" || e.Importance <= 0 {
        t.Fatalf("event fields wrong: %+v", e)
    }
    select { case <-got: t.Fatalf("event emitted more than once"); default: }
}

func TestConsumerFailureNeverFailsSave(t *testing.T) {
    svc, bus := newTestServiceWithBus(t)
    bus.Subscribe(func(memory.SaveEvent) { panic("consumer boom") }) // recovered, logged
    r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "project"})
    if err != nil { t.Fatalf("consumer panic must not fail the save: %v", err) }
    if r.ID == 0 { t.Fatalf("save must still succeed") }
}
```

Plus the dependency-direction test (a build-tagged compile check):

```go
// mnemonic-v2/internal/memory/importcheck_test.go
//go:build !ignore
// Asserts via a grep in the test that no file in internal/memory imports internal/llmwiki
// (the wiki package does not exist yet — the test guards the future: it scans the source
//  of this package for "llmwiki" and fails on a hit).
func TestMemoryDoesNotImportWiki(t *testing.T) { /* walk .go files of this package, reject any "llmwiki" import */ }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ ./internal/wiki_teststub/ -run 'TestSaveEvent|TestConsumerFailure|TestMemoryDoesNotImportWiki|TestWikiStubConsumes' -v`
Expected: `[save-event-exactly-once]` already GREEN from Task 7 (the publish call site + field assertions); the *field-completeness* assertions in `TestSaveEventEmittedExactlyOnce` (`Entity != ""`, `Importance > 0`) FAIL until this task fills them — and `[wiki-failure-never-fails-save]` (panic recovery) FAILS. BDD: the two RED scenarios are this task's.

- [ ] **Step 3: Write minimal implementation**

Fill pipeline step 10's payload (the `Publish` call exists from Task 7): `Entity` = the `topic_key`'s family head (first path segment) when `topic_key` is set, else the title's first significant term (best-effort, spec `004-transport-mcp` §5 "from topic_key family / title"); `Importance` = the value computed at step 6 (the same `importance.go` call — not recomputed); `Degraded` = the pipeline's degraded flag. `eventbus.Publish`: run each subscriber in order, `recover()` panics into a log line (`save event consumer <name> panicked: …`), collect errors into the log — the publisher never returns an error to the save path. `internal/wiki_teststub/`: a package with a single `_test.go` that `Subscribe`s a consumer and asserts it receives a well-formed `SaveEvent` after a save driven through `memory.Service` — proving the boundary works for a consumer that does not live in `internal/memory`. The compile-check test walks `internal/memory`'s source files and fails on any import path containing `llmwiki`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit** (per-task `git add` of this task's own files only — the pre-commit zone guard rejects mixed commits)

```bash
git add mnemonic-v2/internal/memory/pipeline.go mnemonic-v2/internal/memory/eventbus.go mnemonic-v2/internal/memory/eventbus_test.go mnemonic-v2/internal/wiki_teststub/
git commit -m "feat(mnemonic-v2): save event contract (entity + importance), consumer failure isolation, no-wiki-import guard

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T12"
```

---

### Task 13: Shared handler + MCP stdio server (agent profile, write queue, envelope)

> ⚠ one-way: **D1 — the `mem_*` MCP tool contract** (the plugins' only interface into the engine).

**Files:**
- Create: `mnemonic-v2/internal/transport/handler/handler.go`, `mnemonic-v2/internal/transport/mcp/server.go`, `mnemonic-v2/internal/transport/mcp/tools_mem.go`, `mnemonic-v2/internal/transport/mcp/writequeue.go`
- Test: `mnemonic-v2/internal/transport/handler/handler_test.go`, `mnemonic-v2/internal/transport/mcp/server_test.go`, `mnemonic-v2/internal/transport/mcp/contract_test.go`

**Interfaces:**
- Consumes: the full `memory.Service` surface (Tasks 5–12), `core/config`, `core/project`.
- Produces:
  ```go
  // transport/handler/handler.go — the ONE handler set; MCP + CLI + HTTP mirror all call it
  type Handler struct{ svc *memory.Service; cfg config.Config; det project.Detector }
  func NewHandler(svc *memory.Service, cfg config.Config, det project.Detector) *Handler
  func (h *Handler) Save(ctx, req SaveRequest) (SaveResult, error)
  func (h *Handler) Search(ctx, q string, f SearchFilters) (SearchResult, error)
  func (h *Handler) Get(ctx, id int64, syncID string, includeHistory bool, historyCursor int) (GetResult, error)
  func (h *Handler) Lookup(ctx, f LookupFilters) (LookupResult, error)
  func (h *Handler) Context(ctx, budget int, older bool) (ContextResult, error)
  func (h *Handler) Judge / JudgeMany / Compare / Consolidate / SessionStart / SessionEnd / SessionSummary / SessionFinalize /
      SavePrompt / SearchPrompts / CapturePassive / Update / Delete / Pin / Unpin / SuggestTopicKey /
      Review / Timeline / Stats / Doctor / CurrentProject / ListProjects / MergeProjects  // same shapes as the memory.Service methods
  // the D1-pinned contract for the under-specified tool (spec 004-transport-mcp §3 + 004-dedup §6 "family heuristic; the agent decides"):
  //   mem_suggest_topic_key(topic: string [the draft title or a topic fragment], project: string)
  //     → { "candidates": ["architecture/auth-model", "auth"], "family": "architecture" }
  //     candidates = distinct topic_key families in the project (parent-path grouping, most-common parent first, ≤5,
  //     filtered to those sharing ≥1 significant term with `topic`); the agent picks one or writes a new key (spec 004-dedup §6:
  //     "a suggested topic_key is a hint, not proof of identity").
  // every method resolves the project at call time (det.Resolve with the caller's explicit project)
  // and wraps errors into (code, message) for the envelope
  ```
  ```go
  // transport/mcp/server.go
  func NewServer(h *handler.Handler, profile Profile) *Server // Profile: Agent | All
  func (s *Server) Start(ctx context.Context) error          // mcp-go ServeStdio
  // writequeue.go: a serializing queue (a channel + worker goroutine) for all writes
  // (Save, Update, JudgeMany, Consolidate) — Engram internal/mcp/write_queue.go pattern;
  // reads are unqueued. One queued write = one transaction (spec 004-transport-mcp §4).
  ```
  - Response envelope: `{…result fields…, "degraded": bool, "degraded_reason": string}` when degraded (INV-3, spec `004-transport-mcp` §4); ids the agent sees are `sync_id` (INV-6); `mem_get_observation` accepts both.
  - `capture_prompt` is the tri-state of the amended spec row: the MCP arg and the CLI flag are optional; **absent → `true`** (the handler maps absent → `nil` → the pipeline's default-on). A test pins the absent-arg case (Task 11's `TestPromptSearchFindsCaptured` saves without the field).
  - **HTTP mirror (spec `004-transport-mcp` §4):** out of scope for the memory function — the v2 CLI talks to the `Handler` directly (the same seam the mirror would use). The `Handler` is the plug point: a later change registers the POST/GET routes + `MNEMONIC_HTTP_TOKEN` auth on the same methods. Recorded here so the spec requirement is explicitly scoped, not silently dropped.
- Seam: `Handler` is the seam between the two surfaces (K6: CLI and MCP byte-identical JSON).
- Deletion test: deleting `handler` re-creates the 26 tool behaviors in each surface separately — divergence becomes structural.
- Adapters: MCP adapter + CLI adapter (Task 16) over the same Handler = 2.

**SATISFIES:** [tool-contract-pinned], [profile-membership], [write-queue-serializes], [envelope-degraded], [stdio-round-trip]

- [ ] **Step 1: Write the failing tests**

```go
func TestMemToolContract(t *testing.T) {
    // golden per tool: name, required params, return-shape keys. Any drift = fail.
    s := mcp.NewServer(newTestHandler(t), mcp.Agent)
    tools := s.ToolNames()
    for _, want := range agentProfileTools { // the 20 agent-profile tools — the spec table's | agent | rows (20), pinned literally
        if !slices.Contains(tools, want) { t.Errorf("missing agent tool %s", want) }
    }
    for _, admin := range adminTools { // the spec table's | admin | rows (6): mem_delete, mem_stats, mem_timeline, mem_merge_projects, mem_review, mem_doctor
        if slices.Contains(tools, admin) { t.Errorf("admin tool %s leaked into agent profile", admin) }
    }
    if len(tools) != 20 { t.Fatalf("agent profile must expose exactly the 20 agent tools, got %d", len(tools)) }
    // save's required params: title, type, content, session_id (project/scope/topic_key optional)
    save := s.ToolSchema("mem_save")
    for _, req := range []string{"title", "type", "content", "session_id"} {
        if !save.Required[req] { t.Errorf("mem_save must require %s", req) }
    }
}

func TestWriteQueueSerializes(t *testing.T) {
    s := mcp.NewServer(newTestHandler(t), mcp.Agent)
    start := time.Now()
    var wg sync.WaitGroup
    for i := 0; i < 8; i++ { wg.Add(1); go func() { defer wg.Done(); s.CallTool(ctx, "mem_save", saveArgs(i)) }() }
    wg.Wait()
    if !s.WritesSerialized() { t.Fatalf("concurrent writes must be serialized by the queue") }
    _ = start
}

func TestEnvelopeCarriesDegraded(t *testing.T) {
    s := mcp.NewServer(newTestHandlerWithNoLLM(t), mcp.Agent) // nil seams
    out := s.CallTool(ctx, "mem_save", saveArgs(1))
    var env struct{ Degraded bool `json:"degraded"`; DegradedReason string `json:"degraded_reason"` }
    json.Unmarshal(out, &env)
    if !env.Degraded || env.DegradedReason == "" { t.Fatalf("degraded envelope missing: %s", out) }
}

func TestStdioRoundTrip(t *testing.T) {
    // in-process: pipe the server's stdio, speak JSON-RPC, assert a valid tool result frame
    s := mcp.NewServer(newTestHandler(t), mcp.Agent)
    frames, err := s.HandshakeAndCall(ctx, "mem_current_project") // test harness: initialize → tools/call
    if err != nil { t.Fatal(err) }
    if len(frames) == 0 || !json.Valid(frames[0]) { t.Fatalf("no valid JSON-RPC result frame") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/transport/... -v`
Expected: FAIL — packages do not exist. BDD: `[tool-contract-pinned]`, `[profile-membership]`, `[write-queue-serializes]`, `[envelope-degraded]`, `[stdio-round-trip]` RED (threat-matrix rows).

- [ ] **Step 3: Write minimal implementation**

`handler.go`: the 26 methods, each a thin pass-through to `memory.Service` with project resolution at call time (`det.Resolve(cwd, explicit, gitToplevel)` — the `expected_project` ownership assertion is validated *inside* the service's transaction for update/delete, spec `01-memory/001` §6) and error→(code,message) mapping (`project_mismatch`, `session_already_ended`, `operation_conflict` 409, `operation_expired` 410, `fingerprint_mismatch`, `consolidation_refused`, `invalid_arguments`). `server.go`: `mark3labs/mcp-go` `server.NewMCPServer("mnemonic-v2", version)` + per-tool `NewTool` definitions (v1 pattern, `mnemonic_back/internal/mcp/tools_memory.go:25-75`) filtered by profile — the profile sets are the spec table's literal rows (20 agent / 6 admin, K5); `WithInstructions` carrying the memory↔wiki routing rule (spec `01-memory/005` §6 — wiki-first, `mem_lookup` as fallback). `tools_mem.go`: each tool = parse args → `Handler` call → marshal result + envelope; `mem_save`'s optional `capture_prompt` arg parses to `*bool` (absent → `nil` → default-on, the amended tri-state). `writequeue.go`: a single-worker channel queue; write tools enqueue and wait (the queue guarantees one transaction at a time — the store is single-connection, Task 1); reads bypass it. `mem_compare`/`mem_consolidate` are agent-profile in v2 (spec `004-transport-mcp` §3 note — the LLM is first-class, the agent does more semantic work in-profile). `mem_suggest_topic_key`: the pinned contract in the Interfaces block — family heuristic on existing `topic_key`s in the project (parent-path grouping, most-common parent first, ≤5, significant-term filter against the draft).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/transport/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/transport/
git commit -m "feat(mnemonic-v2): shared handler set + MCP stdio server (agent/all profiles, write queue, degraded envelope)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T13"
```

---

### Task 14: CLI surface (mem / sync / trail / doctor / mcp) + parity with MCP

**Files:**
- Create: `mnemonic-v2/cmd/mnemonic/main.go`, `cmd/mnemonic/mem.go`, `cmd/mnemonic/sync.go`, `cmd/mnemonic/trail.go`, `cmd/mnemonic/mcp.go`, `cmd/mnemonic/doctor.go`
- Test: `mnemonic-v2/cmd/mnemonic/parity_test.go`, `mnemonic-v2/cmd/mnemonic/cli_test.go`

**Interfaces:**
- Consumes: `transport/handler` (Task 13 — the CLI calls the *same* handler as MCP), `core/config`, `core/project`.
- Produces: the command surface per spec `005-transport-cli` §2: `mcp`, `serve` (no-op alias printing a pointer), `mem <save|search|get|lookup|context|review|timeline|pin|unpin|delete|session|stats|doctor>`, `sync <status|push|pull|export|import|enroll>`, `trail <recent|show>`, `doctor [--strict]`. (`index`, `migrate`, `setup` are stubs that delegate to the code-index / plugin changes — a printed pointer, not a TODO: they exist so the dispatch is complete and the delegation is a one-line change later.)
- Seam: `Handler` (K6).
- Deletion test: deleting the CLI re-creates the argument parsing in the MCP layer — the two surfaces diverge.
- Adapters: CLI flag-parsing adapter over `Handler`.

**SATISFIES:** [cli-mcp-parity], [cli-project-selection], [cli-json-shape]

- [ ] **Step 1: Write the failing tests**

```go
func TestCLIMCPParity(t *testing.T) {
    // run `mem save --title T --type decision --content C --project p --session s --json`
    // and the MCP `mem_save` with the same args; the JSON must be byte-identical
    // (modulo the CLI's --json envelope key ordering: compare unmarshaled JSON)
    cliOut := runCLI(t, "mem", "save", "--title", "T", "--type", "decision", "--content", "C",
        "--project", "p", "--session", "s", "--json")
    mcpOut := mcpCall(t, "mem_save", map[string]any{"title": "T", "type": "decision", "content": "C", "project": "p", "session_id": "s"})
    var a, b map[string]any
    json.Unmarshal(cliOut, &a); json.Unmarshal(mcpOut, &b)
    if !jsonEqual(a, b) { t.Fatalf("CLI and MCP JSON must match:\n%v\nvs\n%v", a, b) }
}

func TestCLIProjectSelection(t *testing.T) {
    // --project beats MNEMONIC_PROJECT; --all reads every project; explicit + --all is a usage error
    t.Setenv("MNEMONIC_PROJECT", "envproj")
    if _, err := runCLIStatus(t, "mem", "search", "q", "--project", "explicit", "--all"); err == nil {
        t.Fatalf("explicit project + --all must be a usage error")
    }
}

func TestDoctorStrictExitsNonZeroOnGap(t *testing.T) {
    // no embedder configured → doctor reports degraded → --strict exits 1
    code := runCLIExit(t, "doctor", "--strict")
    if code != 1 { t.Fatalf("strict doctor with a capability gap must exit 1, got %d", code) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./cmd/... -v`
Expected: FAIL — `cmd/mnemonic` does not exist. BDD: `[cli-mcp-parity]`, `[cli-project-selection]`, `[cli-json-shape]` RED.

- [ ] **Step 3: Write minimal implementation**

`main.go`: stdlib `flag` dispatch (v1 pattern, `mnemonic_back/cmd/mnemonic/main.go:14-184`): build `config` → `project.Detector` → open store(s) → `memory.NewService` → `handler.NewHandler` → switch on the first arg. `mem.go`: the mem subcommands — each parses flags (`--project`, `--all`, `--scope`, `--type`, `--limit`, `--match-mode`, `--id`, `--topic-key`, `--budget`, `--older-history`, `--hard`, `--expected-project`, `--json`), calls the `Handler`, prints human-readable or `--json` (the handler's result marshaled — the same marshal the MCP tools use, hence parity). `session` subcommand: `start|end|summary|finalize` (`summary`/`finalize` take the six sections as `--goal --instructions --discoveries --accomplished --next-steps --relevant-files`). `sync.go`: the sync subcommands over the `memory.SyncService` (Task 17) — `status` prints the per-target lifecycle table, `push`/`pull` run one cycle, `export`/`import` take an archive dir, `enroll <project> [--off]`. `trail.go`: `recent` (last 20) / `show <id>` over `retrieval_trails`. `doctor.go`: resolved config + capability checks (FTS table present, vector index present, embed round-trip — embed one probe string and re-search it, LLM ping) → `degraded` on any gap; `--strict` exits 1 on a gap. `mcp.go`: `mnemonic mcp` → `handler` + `mcp.NewServer(h, profile from --profile)` + start the autosync manager when `cfg.Sync.Auto` (Task 17) + `ServeStdio`. `serve`: prints "v2 drops the HTTP dashboard; use `mnemonic mcp` or the CLI" and exits 0.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./cmd/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/cmd/
git commit -m "feat(mnemonic-v2): CLI surface (mem/sync/trail/doctor/mcp) over the shared handler, MCP parity

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T14"
```

---

### Task 15: Sync core — journal apply, tombstones, deferred-apply, enrollment

**Files:**
- Create: `mnemonic-v2/internal/memory/sync.go`
- Modify: `mnemonic-v2/internal/core/store/sqlite.go` (`ApplyPulledMutation` real, `Tombstone` real, deferred-apply)
- Test: `mnemonic-v2/internal/memory/sync_test.go`

**Interfaces:**
- Consumes: Task 1's `sync_mutations`/`sync_state`/`sync_apply_deferred`/`sync_delete_tombstones`/`sync_enrolled_projects` tables, the store methods.
- Produces:
  ```go
  type SyncService struct{ st store.Store; cfg config.Config }
  func NewSyncService(st store.Store, cfg config.Config) *SyncService
  func (s *SyncService) Enroll(ctx context.Context, project string, on bool) error
  func (s *SyncService) ApplyPulled(ctx context.Context, m store.Mutation) error
  // upsert: through the SAME store methods a local write uses (INV-4 — no special remote path);
  // missing FK dependency (relation endpoint / survivor's session not present) → sync_apply_deferred,
  // retried on each pull cycle, 5 failed replays → dead (pruned)
  func (s *SyncService) ApplyDelete(ctx context.Context, m store.Mutation) error
  // records the tombstone floor BEFORE applying (spec 01-memory/006 §6): a late upsert for the
  // same entity_key after a delete cannot resurrect the row
  func (s *SyncService) Status(ctx context.Context) (map[string]TargetStatus, error) // per target_key lifecycle + seqs + backoff
  ```
- Seam: the store's sync methods (Task 1's interface).
- Deletion test: deleting the apply path means a pulled mutation is a raw SQL insert — the INV-4 "local is always the authority" guarantee (and the FTS triggers) are bypassed.
- Adapters: in-process (the PG transport is Task 17).

**SATISFIES:** [pull-applies-through-store], [tombstone-floor-no-resurrect], [deferred-apply-5-cap], [enrollment-gates-journal]

- [ ] **Step 1: Write the failing tests**

```go
func TestPullAppliesThroughStore(t *testing.T) {
    local := openTestStore(t)            // machine A (local truth)
    remote := openTestStore(t)           // machine B, plays "the other peer"
    // A saves a session + observation, journals them
    svcA := newTestServiceWith(t, local)
    _, _ = svcA.SessionStart(ctx, SessionRequest{ID: "sa", Project: "p", Directory: "/d"})
    res, _ := svcA.Save(ctx, SaveRequest{SessionID: "sa", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project"})
    // B pulls A's journal (via the transport, Task 17 — here: direct ListPendingMutations)
    muts, _ := local.ListPendingMutations(ctx, 0, 100)
    svcB := NewSyncService(remote, syncCfg(true))
    for _, m := range muts { if err := svcB.ApplyPulled(ctx, m); err != nil { t.Fatal(err) } }
    got, err := remote.GetObservation(ctx, findIDBySyncID(t, remote, res.SyncID), false)
    if err != nil || got.Title != "T" { t.Fatalf("pulled observation not applied: %+v %v", got, err) }
    // INV-4 proof: the FTS index on B saw it (a trigger fired — the apply went through the store, not raw SQL)
    r, _ := remote.Search(ctx, store.SearchQuery{Query: "T", Project: "p", Limit: 5})
    if len(r.Observations) != 1 { t.Fatalf("FTS trigger did not fire on pulled apply") }
}

func TestTombstoneFloorPreventsResurrect(t *testing.T) {
    svc := NewSyncService(st := openTestStore(t), syncCfg(true))
    // upsert then delete, in order
    _ = svc.ApplyPulled(ctx, upsertMutation(t, "obs-1"))
    _ = svc.ApplyDelete(ctx, deleteMutation(t, "obs-1"))
    // a LATE upsert for the same key must not resurrect
    _ = svc.ApplyPulled(ctx, upsertMutation(t, "obs-1"))
    if live := isLive(t, st, "obs-1"); live { t.Fatalf("late upsert resurrected a tombstoned row") }
}

func TestDeferredApplyRetriesThenDies(t *testing.T) {
    svc := NewSyncService(st := openTestStore(t), syncCfg(true))
    // a relation whose target observation has not been pulled yet
    _ = svc.ApplyPulled(ctx, relationMutation(t, "obs-new", "obs-existing"))
    n := deferredCount(t, st)
    if n != 1 { t.Fatalf("missing-FK mutation must be deferred, got %d", n) }
    for i := 0; i < 5; i++ { svc.retryDeferred(ctx) } // 5 failed replays
    if live := deferredCount(t, st); live != 0 { t.Fatalf("dead deferred rows must be pruned after 5 retries") }
}

func TestEnrollmentGatesJournal(t *testing.T) {
    st := openTestStore(t)
    svc := newTestServiceWithStore(t, st)
    r1, _ := svc.Save(ctx, saveReq("p")) // unenrolled
    if n := pendingCount(t, st); n != 0 { t.Fatalf("unenrolled project must not enqueue remote mutations") }
    NewSyncService(st, syncCfg(true)).Enroll(ctx, "p", true)
    r2, _ := svc.Save(ctx, saveReq("p"))
    if n := pendingCount(t, st); n != 1 { t.Fatalf("enrolled project must enqueue, got %d", n) }
    _ = r1; _ = r2
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestPull|TestTombstone|TestDeferred|TestEnrollment' -v`
Expected: FAIL — `ApplyPulledMutation`/`Tombstone` are Task 1 stubs. BDD: `[pull-applies-through-store]`, `[tombstone-floor-no-resurrect]`, `[deferred-apply-5-cap]` RED.

- [ ] **Step 3: Write minimal implementation**

`sync.go`: **ApplyPulled** — dispatch on `m.Entity`: `observation` → the same code path as a local upsert (session liveness check relaxed for pulls: if the parent session row is missing → `sync_apply_deferred` with `attempts++`; 5 → dead + pruned; a dead relation re-arms only when its scoped endpoint predicate is satisfied — spec `01-memory/006` §6); `session` → insert-or-update the session row (an ended session pulled as ended stays ended — terminal); `prompt` → insert with the identity-conflict rules (a pulled prompt upsert cannot move an established nonempty inbox identity to a different session — spec §6); `relation` → insert via `AddRelation` (endpoint missing → deferred); `project` → enrollment marker. **ApplyDelete** — record the tombstone floor in `sync_delete_tombstones` (+ `_remote_floors`) **before** the soft/hard delete; a subsequent upsert for the same `entity_key` with `created_at` ≤ the floor is dropped (the classic late-upsert resurrection race, spec §6). **Enroll** — `sync_enrolled_projects` insert/remove; the pipeline's step-7 ENQUEUE checks it (Task 5's call site). **Status** — read `sync_state` per target: lifecycle, `last_enqueued_seq`/`last_acked_seq`/`last_pulled_seq`, backoff, lease, reason.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/sync.go mnemonic-v2/internal/memory/sync_test.go mnemonic-v2/internal/core/store/
git commit -m "feat(mnemonic-v2): sync apply (through-store, tombstone floors, deferred-apply 5-cap, enrollment)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T15"
```

---

### Task 16: Sync transport — PG transport + chunk export/import + CLI sync subcommands

**Files:**
- Create: `mnemonic-v2/internal/memory/transport.go`, `mnemonic-v2/internal/memory/chunkcodec.go`
- Modify: `mnemonic-v2/internal/memory/sync.go` (push/pull over the transport), `mnemonic-v2/cmd/mnemonic/sync.go` (Task 14's stubs become real)
- Test: `mnemonic-v2/internal/memory/transport_test.go`, `mnemonic-v2/internal/memory/chunk_test.go`

**Note:** the remote table DDL for `schema_<project>` (replicated entity tables + `mutations` + `tombstones` + `apply_deferred` + `sync_state` + `project_controls`) is written in this task as the `EnsureSchema` SQL — it is the *remote* schema. Task 18's `migrations_pg/` is the *local-dialect* render (a store that opens directly in Postgres mode) — a different artifact with the same table names.

**Interfaces:**
- Consumes: Task 15's `SyncService`, Task 1's `PostgresStore` (Task 18 builds the store; the transport here is the SQL against `schema_<project>`).
- Produces:
  ```go
  // transport.go
  type MutationTransport interface {
      PushMutations(ctx context.Context, project string, batch []store.Mutation) (int64, error) // ≤100/batch, ≤8MiB, one PG transaction; returns acked seq
      PullMutations(ctx context.Context, project string, sinceSeq int64) ([]store.Mutation, error)
      EnsureSchema(ctx context.Context, project string) error // CREATE SCHEMA IF NOT EXISTS schema_<sanitized>
  }
  func NewPGTransport(dsn string) (MutationTransport, error)  // database/sql + lib/pq
  // chunk codec (port Engram internal/cloud/chunkcodec):
  func ChunkID(payload []byte) string                    // first 8 hex of sha256
  func CanonicalizeForProject(payload []byte, project string) []byte // project-canonicalized JSON
  func Export(ctx context.Context, st store.Store, project string, dir string) (manifest, error)
  func Import(ctx context.Context, st store.Store, project string, dir string) (int, error)
  ```
  - Remote schema per project (`schema_<sanitized_project>`, `project.Canonicalize` + schema-safe char set): `mutations (seq BIGSERIAL, …mirrors sync_mutations…)`, replicated entity tables + `tsv` columns, `tombstones`, `apply_deferred`, `sync_state`, `project_controls` (the per-project pause flag — a paused project's push writes no rows and reports `sync-paused`).
  - Export archive: `<dir>/manifest.json` + `<dir>/chunks/<chunkid>.jsonl.gz` (gzip JSONL; chunk id = `ChunkID(canonicalPayload)`).
- Seam: `MutationTransport` — the seam between the sync engine and the remote (spec `01-memory/006` §5).
- Deletion test: deleting the transport re-couples the sync engine to a specific remote — the team's self-hosted PG (or a future different remote) becomes a code change.
- Adapters: PG transport (prod) + an in-memory fake (tests) = 2.

**SATISFIES:** [push-acks-seq], [pull-scoped-to-project-schema], [chunk-id-stable], [export-import-round-trip], [schema-per-project-fail-fast]

- [ ] **Step 1: Write the failing tests**

```go
func TestChunkIDStable(t *testing.T) {
    p1 := CanonicalizeForProject([]byte(`{"content":"a","project":""}`), "p")
    p2 := CanonicalizeForProject([]byte(`{"project":"","content":"a"}`), "p") // key order differs
    if ChunkID(p1) != ChunkID(p2) { t.Fatalf("canonicalization must be key-order stable") }
    if len(ChunkID(p1)) != 8 { t.Fatalf("chunk id must be 8 hex") }
}

func TestPushAcksSeq(t *testing.T) {
    tr := newFakeTransport(t)
    acked, err := tr.PushMutations(ctx, "p", makeMutations(100)) // exactly the batch cap
    if err != nil { t.Fatal(err) }
    if acked != 100 { t.Fatalf("acked seq wrong: %d", acked) }
    if _, err := tr.PushMutations(ctx, "p", makeMutations(101)); err == nil { t.Fatalf(">100 batch must be rejected") }
}

func TestPullScopedToProjectSchema(t *testing.T) {
    // fake transport records the schema it queried; a pull for project "a" must
    // never touch schema_b (the core isolation constraint, spec 01-memory/006 §2)
    tr := newFakeTransport(t)
    tr.seed("a", 3); tr.seed("b", 5)
    got, _ := tr.PullMutations(ctx, "a", 0)
    if len(got) != 3 { t.Fatalf("pull must be scoped to the project's schema") }
    if tr.schemasTouched() != "a" { t.Fatalf("touched other schemas: %s", tr.schemasTouched()) }
}

func TestSchemaPerProjectFailFast(t *testing.T) {
    tr := newFakeTransport(t)
    tr.failResolve("ghost")
    if err := tr.EnsureSchema(ctx, "ghost"); err == nil {
        t.Fatalf("unresolvable project schema must fail fast (no public fallback)")
    }
}

func TestExportImportRoundTrip(t *testing.T) {
    src := openTestStore(t); seedProject(t, src, "p", 20) // 20 observations + sessions
    dir := t.TempDir()
    if _, err := Export(ctx, src, "p", dir); err != nil { t.Fatal(err) }
    dst := openTestStore(t)
    n, err := Import(ctx, dst, "p", dir)
    if err != nil { t.Fatal(err) }
    if n == 0 { t.Fatalf("import must apply the archive through the store") }
    // every source observation is present on the destination by sync_id
    for _, syncID := range syncIDsOf(t, src, "p") {
        if _, err := dst.GetObservation(ctx, findIDBySyncID(t, dst, syncID), false); err != nil {
            t.Fatalf("missing %s after import: %v", syncID, err)
        }
    }
    // re-import is idempotent (chunk id dedupe: sync_chunks)
    if _, err := Import(ctx, dst, "p", dir); err != nil { t.Fatalf("re-import must be idempotent: %v", err) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestChunkID|TestPush|TestPull|TestSchema|TestExport' -v`
Expected: FAIL — transport does not exist. BDD: `[push-acks-seq]`, `[pull-scoped-to-project-schema]`, `[export-import-round-trip]` RED.

- [ ] **Step 3: Write minimal implementation**

`transport.go`: **PGTransport** — `database/sql` + `lib/pq` (ADR-0041), `EnsureSchema` = `CREATE SCHEMA IF NOT EXISTS schema_<sanitized>` + the remote table set (the `migrations_pg` render of the replicated tables + `mutations` + `tombstones` + `apply_deferred` + `sync_state` + `project_controls`); all queries qualified `schema_<sanitized>.<table>` (no `search_path` reliance — the schema *is* the isolation boundary, spec `01-memory/006` §2); a connection that cannot resolve the schema fails at `EnsureSchema`, never falling back to `public`; `PushMutations` = one transaction, `INSERT INTO …mutations…` batch (≤100, payload size ≤8 MiB — `sync.MaxPushBytes`), then `UPDATE sync_state SET last_acked_seq=?`; paused project (`project_controls.paused=1`) → writes nothing, returns 0 + a `sync-paused` marker; `PullMutations` = `SELECT … WHERE seq > ? ORDER BY seq` scoped to the schema. **Chunk codec** — port Engram's: `ChunkID` = first 8 hex of SHA-256 (`chunkcodec.go:13-16`); `CanonicalizeForProject` = JSON-rekey (stable key order) + project canonicalization (blank-project inheritance applied, spec `002-schema` §3.1 payload rule). **Export** — walk the project's entities (sessions, observations, prompts, relations — the canonical payloads), one chunk per mutation-equivalent unit, write `chunks/<id>.jsonl.gz` (gzip JSONL) + `manifest.json` (project, chunk list, export time, schema version); **Import** — read the manifest, skip chunks already in `sync_chunks` (idempotent), apply each through `SyncService.ApplyPulled` (the same through-store path, Task 15 — INV-4 holds for imports too). The CLI `sync` subcommands (Task 14's stubs) now call `SyncService` + the transport: `push`/`pull` run one cycle, `export`/`import` take a dir, `status` prints `SyncService.Status`, `enroll` toggles enrollment.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v && go test ./cmd/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/transport.go mnemonic-v2/internal/core/store/migrations_pg/ mnemonic-v2/cmd/mnemonic/sync.go
git commit -m "feat(mnemonic-v2): PG mutation transport (schema-per-project), chunk export/import, sync CLI

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T16"
```

---

### Task 17: Autosync manager (lease-guarded, debounced, off the write path)

**Files:**
- Create: `mnemonic-v2/internal/memory/autosync.go`
- Modify: `mnemonic-v2/internal/memory/pipeline.go` (the `onWrite` hook after commit), `mnemonic-v2/cmd/mnemonic/mcp.go` (start/stop the manager when `sync.auto`)
- Test: `mnemonic-v2/internal/memory/autosync_test.go`

**Interfaces:**
- Consumes: Task 15's `SyncService`, Task 16's `MutationTransport`, Task 1's `sync_state` lease columns.
- Produces:
  ```go
  type Autosync struct{ svc *SyncService; tr MutationTransport; cfg config.Config }
  func NewAutosync(svc *SyncService, tr MutationTransport, cfg config.Config) *Autosync
  func (a *Autosync) NotifyDirty()      // non-blocking send on a buffered-1 channel (the store never blocks on network)
  func (a *Autosync) Start(ctx context.Context)  // the background goroutine
  func (a *Autosync) Stop()
  func (a *Autosync) StopForUpgrade()    // PhaseDisabled, does NOT release the lease
  func (a *Autosync) ResumeAfterUpgrade()// clears the flag, re-arms without restart
  ```
  - Cycle: acquire the SQLite-backed lease (`sync_state.lease_owner`/`lease_until`, 60s — Engram `store.go:8198-8233` semantics) → push (`ListPendingMutations` → `PushMutations`) → pull (`PullMutations` since `last_pulled_seq` → `ApplyPulled`) → update `sync_state` + release the lease. 500 ms debounce (`cfg.Sync.DebounceMs`); exponential backoff with ±25% jitter on failure (cap 5 min, `MaxConsecutiveFailures` 10); phases `idle/pushing/pulling/push_failed/pull_failed/backoff/healthy/disabled`.
- Seam: the `onWrite` hook (a `func()` the pipeline calls after commit; the manager replaces its implementation with `NotifyDirty`).
- Deletion test: deleting the manager means every save blocks on network I/O — INV-4's "a remote failure never corrupts local" becomes a latency problem instead of a non-event.
- Adapters: in-process (the goroutine) + a synchronous test driver.

**SATISFIES:** [autosync-single-lease], [autosync-debounce], [write-path-never-blocks-on-network]

- [ ] **Step 1: Write the failing tests**

```go
func TestAutosyncSingleLease(t *testing.T) {
    st := openTestStore(t)
    a1 := NewAutosync(NewSyncService(st, syncCfg(true)), newFakeTransport(t), syncCfgAuto())
    a2 := NewAutosync(NewSyncService(st, syncCfg(true)), newFakeTransport(t), syncCfgAuto())
    ctx, cancel := context.WithCancel(context.Background())
    a1.Start(ctx); a2.Start(ctx)
    a1.NotifyDirty(); a2.NotifyDirty()
    waitForCycles(t, a1, a2, 1) // let both try to cycle
    cancel(); a1.Stop(); a2.Stop()
    // exactly one of them held the lease at a time; both cycles completed without error
    if a1.CycleCount() + a2.CycleCount() < 1 { t.Fatalf("no cycle ran") }
    if a1.LastLeaseOwner() == a2.LastLeaseOwner() && a1.CycleCount() > 0 && a2.CycleCount() > 0 {
        t.Fatalf("two workers held the lease concurrently: %s", a1.LastLeaseOwner())
    }
}

func TestWritePathNeverBlocksOnNetwork(t *testing.T) {
    st := openTestStore(t)
    tr := newFakeTransport(t); tr.Slow(500 * time.Millisecond) // network slower than the save
    a := NewAutosync(NewSyncService(st, syncCfg(true)), tr, syncCfgAuto())
    a.Start(ctx)
    svc := newTestServiceWithStore(t, st); svc.SetOnWrite(a.NotifyDirty)
    start := time.Now()
    svc.Save(ctx, saveReq("p"))
    if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
        t.Fatalf("save blocked on network: %v (must be a non-blocking channel send)", elapsed)
    }
    a.Stop()
}

func TestDebounceCoalescesBursts(t *testing.T) {
    a := NewAutosync(NewSyncService(openTestStore(t), syncCfg(true)), tr := newFakeTransport(t), syncCfgAuto())
    a.Start(ctx)
    for i := 0; i < 10; i++ { a.NotifyDirty() } // a 500ms-debounced burst
    waitForCycles(t, a, 1)
    if tr.Calls() != 1 { t.Fatalf("burst must coalesce to one cycle, got %d", tr.Calls()) }
    a.Stop()
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestAutosync|TestWritePath|TestDebounce' -v`
Expected: FAIL — the manager does not exist. BDD: `[autosync-single-lease]`, `[write-path-never-blocks-on-network]` RED (threat-matrix process-integration row).

- [ ] **Step 3: Write minimal implementation**

`autosync.go`: port Engram's manager structure (`internal/cloud/autosync/manager.go:268-501`): `dirtyCh chan struct{}` (buffered 1); `Start` launches one goroutine selecting on ctx / `dirtyCh` (resets the debounce timer) / a 30s poll ticker; **debounce semantics are coalescing, not per-call**: a burst of N `NotifyDirty` sends within `DebounceMs` yields exactly **one** cycle after the timer fires (the test's `waitForCycles` is a poll on the fake transport's call counter with a 2× debounce timeout — no timing assertions); `cycle` = backoff/disabled check → `AcquireSyncLease(targetKey, owner, 60s)` (owner = `autosync-<unixnano>`) → push → pull → `ReleaseSyncLease` (clear only if owner matches); `computeBackoff` = `base·2^(n-1)` capped 5 min, ±25% jitter, 10 consecutive failures → `disabled` + reason; `StopForUpgrade`/`ResumeAfterUpgrade` per the Engram semantics (upgrade holds the lease so no other worker picks up sync mid-upgrade). The pipeline's step-8 commit calls the `onWrite` hook (default: no-op; `SetOnWrite(a.NotifyDirty)` wires the manager). All network I/O is inside the goroutine — the store never blocks (the buffered-1 send is the only cross-goroutine touch). `mcp.go`: when `cfg.Sync.Auto`, `NewAutosync(...)` + `Start` before `ServeStdio`, `Stop` on ctx cancel.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/memory/autosync.go mnemonic-v2/internal/memory/pipeline.go mnemonic-v2/cmd/mnemonic/mcp.go
git commit -m "feat(mnemonic-v2): lease-guarded autosync (debounced, off the write path, upgrade-safe)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T17"
```

---

### Task 18: PostgresStore (the second dialect render)

> ⚠ one-way: **D3 — Postgres driver = `lib/pq`** (new dependency; ADR-0041 lands in this task's commit).

**Files:**
- Create: `mnemonic-v2/internal/core/store/postgres.go`, `mnemonic-v2/internal/core/store/migrations_pg/001_schema.sql` (if Task 16 only created the *remote* tables, this task adds the full local-equivalent PG render)
- Modify: `mnemonic-v2/go.mod` (+ `github.com/lib/pq`), `mnemonic-v2/internal/core/store/store.go` (the `Open` dialect switch)
- Test: `mnemonic-v2/internal/core/store/postgres_test.go`

**Interfaces:**
- Consumes: Task 1's `Store` interface + `Open`, the type legend from spec `002-schema` §1.
- Produces: `PostgresStore` implementing the *full* `Store` interface (S-2: same Go types, no `*sql.Rows` escaping the interface). Nothing changes for callers — `store.Open(ctx, dir, project, Postgres, dsn)` returns the PG store.
- Seam: the `Store` interface (D2 — this is the second real adapter).
- Deletion test: deleting `postgres.go` collapses INV-2 — the dual-mode claim is a SQLite-only lie.
- Adapters: `SQLiteStore` + `PostgresStore` + test fakes = 3.

**SATISFIES:** [pg-dialect-parity], [pg-fts-tsvector], [migration-runs-from-empty-pg]

- [ ] **Step 1: Write the failing tests**

```go
// These tests need a reachable Postgres (MNEMONIC_PG_TEST_DSN); they skip cleanly without one,
// but CI runs them (the two-dialects-from-empty regression, spec 002-schema §6).
func TestPGStoreParity(t *testing.T) {
    st := openPGTestStore(t) // store.Open(…, store.Postgres, testDSN)
    // the SAME test bodies as the SQLite parity tests (Task 2) run against it:
    runObservationRoundTrip(t, st)   // same function the SQLite tests call
    runFTSSync(t, st)                 // tsvector triggers keep the index in step
    runSearchParity(t, st)            // same SearchResult shape; ts_rank weighting
    runShortTermFallback(t, st)       // pg_trgm similarity / LIKE for <3-rune terms
}

func TestPGMigrationFromEmpty(t *testing.T) {
    // fresh schema_<project>: apply migrations_pg in order from empty; reopen; idempotent
    st := openPGTestStore(t)
    st.Close()
    st2 := openPGTestStore(t)
    st2.Close()
}
```
(The shared `run*` helpers are extracted from Task 2's SQLite tests in this task — the same assertions, both dialects. That extraction is the whole point of the S-1 rule: a feature that cannot be expressed in both dialects is not allowed.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `MNEMONIC_PG_TEST_DSN=postgres://… go test ./internal/core/store/ -run 'TestPG' -v`
Expected: FAIL — `PostgresStore` does not exist (and the tests skip-fail with "no PG store" when the DSN is unset). BDD: `[pg-dialect-parity]`, `[migration-runs-from-empty-pg]` RED.

- [ ] **Step 3: Write minimal implementation**

`postgres.go`: `database/sql` + `lib/pq` (ADR-0041: driver of record for the PG render; the Go driver surface is `database/sql` so a future driver swap is contained in this file); `Open` resolves `schema_<Canonicalize(project)>` — fails fast if the schema cannot be created/resolved (no `public` fallback); `READ COMMITTED` isolation (spec `001-storage` §5); writes serialized by the memory write queue (Task 13 — the queue is dialect-agnostic). **S-3 as amended (T0 spec amendment):** this store is the direct-PG *dialect render* for a deployment whose local truth is Postgres — distinct from the replication path (Task 16's `schema_<project>` remote), where saves are journaled and pushed. The render per the spec `002-schema` §1 type legend: `id`→`BIGSERIAL`, `ts`→`TIMESTAMPTZ`, `bool`→`BOOLEAN`, `json`→`JSONB`, `vec`→`vector(N)` (pgvector extension, `CREATE EXTENSION IF NOT EXISTS vector` in the migration; when the extension is absent, `VectorSearch` returns empty + `degraded` — the same contract as the SQLite leg without vec). **Vector index parity (S-1):** `CREATE INDEX … USING hnsw (embedding vector_cosine_ops)` in `migrations_pg/001_schema.sql` (pgvector's HNSW, matching `001-storage` §4's "HNSW/IVFFlat index" — the SQLite leg's vec0 is the equivalent; a sequential scan on PG would be a silent query-surface divergence). FTS per spec `001-storage` §3: a `tsv tsvector` column + GIN index + `BEFORE INSERT/UPDATE/DELETE` trigger maintaining it (`to_tsvector('simple', …)`); ranking via `ts_rank_cd` with per-column weight classes mirroring the FTS5 weights (title 5.0 / content 1.0 / topic_key 3.0 normalized into the `ts_rank_cd` weight classes A/B/C/D); `< 3 rune` terms → `pg_trgm` `similarity()` / `LIKE` fallback (the `pg_trgm` extension, created in the migration). `Migrate` runs `migrations_pg/*.sql` in order (same runner, different embedded FS); the full memory-domain + sync-domain tables from spec `002-schema` §2–3 in PG dialect — **including `retrieval_usage`** (the T0 schema amendment) and `retrieval_trails`. The `Search`/`VectorSearch`/`Context`/`SearchPrompts` methods return the identical Go shapes (S-2).

- [ ] **Step 4: Run tests to verify they pass**

Run: `MNEMONIC_PG_TEST_DSN=postgres://… go test ./internal/core/store/ -v` (CI also runs the plain `go test ./...` — the PG tests skip without the DSN and the SQLite suite stays green)
Expected: PASS (with DSN); PASS-with-skip (without).

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/core/store/postgres.go mnemonic-v2/internal/core/store/migrations_pg/ mnemonic-v2/go.mod mnemonic-v2/go.sum .skillgrid/artifacts/04-adr-0041-mnemonic-v2-postgres-driver.md
git commit -m "feat(mnemonic-v2): PostgresStore dialect render (tsvector FTS, pgvector, schema-per-project)

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T18 one-way:D3"
```

---

### Task 19: v1 → v2 seed migration (`mnemonic migrate`) + doctor completeness

> ⚠ one-way: **D4 — the v1 cutover** (v1 store files are *not* auto-migrated; the export is a one-time seed).

**Files:**
- Create: `mnemonic-v2/cmd/mnemonic/migrate.go`, `mnemonic-v2/internal/migrate/migrate.go` (its own package: it opens the v1 file read-only and depends on `memory.Service` — putting it inside `internal/memory` would make the package import itself's test fixtures awkward and couples the cutover to the domain)
- Test: `mnemonic-v2/internal/memory/migrate_test.go`

**Interfaces:**
- Consumes: the v1 store layout (read-only: v1's `observations` table via `modernc.org/sqlite` opening the v1 `.sqlite` file), Task 5's `Service.Save` (the seed writes through the normal pipeline).
- Produces: `func Migrate(ctx context.Context, v1Dir, v2DataDir, project string) (MigrateReport, error)` — reads v1 observations (+ sessions) from `<v1Dir>/<project>.sqlite`, writes them into v2 **through `Service.Save`** (so redaction, dedup, importance, and the journal all apply), and returns a report (`{exported, skipped, errors []}`). CLI: `mnemonic migrate --v1-dir ~/.skillgrid/mnemonic --project <p>`.
- Seam: `Service.Save` (the seed is just a batch of saves — no special import path into the v2 store, INV-4).
- Deletion test: deleting the migrator means v1 data is lost at cutover — the one-way door.
- Adapters: in-process.

**SATISFIES:** [v1-seed-import], [v1-files-untouched]

- [ ] **Step 1: Write the failing tests**

```go
func TestV1SeedImport(t *testing.T) {
    v1 := makeV1Store(t) // a real v1-shaped .sqlite: sessions + observations rows (v1 schema, v1 sync_id format)
    v2dir := t.TempDir()
    rep, err := memory.Migrate(ctx, v1, v2dir, "p")
    if err != nil { t.Fatal(err) }
    if rep.Exported == 0 || len(rep.Errors) != 0 { t.Fatalf("report wrong: %+v", rep) }
    // the v2 store has the observations, retrievable by search
    st := openTestStoreIn(t, v2dir, "p")
    r, _ := st.Search(ctx, store.SearchQuery{Query: v1FirstObservationTerm, Project: "p", Limit: 5})
    if len(r.Observations) == 0 { t.Fatalf("seeded observations not searchable in v2") }
}

func TestV1FilesUntouched(t *testing.T) {
    v1 := makeV1Store(t)
    before := dirHash(t, v1) // mtime+size of every file in the v1 dir
    memory.Migrate(ctx, v1, t.TempDir(), "p")
    after := dirHash(t, v1)
    if before != after { t.Fatalf("migrate must be read-only over the v1 files (different schema philosophy — export, don't migrate in place)") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestV1' -v`
Expected: FAIL — the migrator does not exist. BDD: `[v1-seed-import]`, `[v1-files-untouched]` RED.

- [ ] **Step 3: Write minimal implementation**

`internal/migrate/migrate.go`: open the v1 file read-only (`file:<path>?mode=ro`, the same pure-Go driver — no v1 dependency needed, only its table layout: v1 `observations(id, session_id, type, title, content, project, scope, topic_key, normalized_hash, …)` + v1 `sessions`); map v1 columns → `SaveRequest` (v1 has no `provenance` column → the seed sets `provenance.writer = "v1-migrate"`, `model = ""`, `session_id` = the v1 session or a synthetic `v1-import` session; v1's `importance_score` column is *not* copied — v2's importance is on-the-fly, spec `002-schema` §2.2.3); write each through `Service.Save` with `OperationID = "v1migrate:" + v1id` (replay-safe: a re-run of `migrate` is idempotent per source row, Task 6's ledger); sessions first (so observation FKs resolve), then observations in v1 `id` order. `dirHash` in the test = `sha256` over `name:size:mtime` of every file (the v1 dir must be byte-identical after a migrate). `doctor.go` completeness: the `mem doctor` / whole-system `doctor` now also reports the v2 store's schema version + the migration state (a v1 dir detected but not migrated → a `migrate-pending` advisory line).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/memory/ -run 'TestV1' -v && go test ./cmd/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/cmd/mnemonic/migrate.go mnemonic-v2/internal/migrate/
git commit -m "feat(mnemonic-v2): v1→v2 seed migration (read-only export through the save pipeline), doctor migration state

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T19 one-way:D4"
```

---

### Task 20: Concurrency backstop (two-process WAL) + full-suite gate

**Files:**
- Create: `mnemonic-v2/internal/core/store/concurrency_test.go`
- Modify: (none in production code — this is the backstop task; if it fails, the fix lands here and re-enters Task 1's retry wrapper)
- Test: `mnemonic-v2/internal/core/store/concurrency_test.go`

**Interfaces:**
- Consumes: Task 1's store + retry wrapper.
- Produces: the T21 backstop proof (the only Must-Have truth that cannot be confirmed by reading a diff).
- Seam: none (test-only).
- Deletion test: n/a.
- Adapters: n/a.

**SATISFIES:** [concurrent-process-wal] (backstop)

- [ ] **Step 1: Write the failing test**

```go
func TestConcurrentProcessWrites(t *testing.T) {
    // build the real binary, then run two OS processes against the same project file:
    // process A holds the store open (mcp server, --profile all, a live session) while
    // process B runs `mem save` in a loop (20 saves). Both must finish cleanly;
    // the final store must contain all 20 rows and pass an integrity check.
    bin := buildTestBinary(t) // `go build -o` the cmd/mnemonic
    A := exec.Command(bin, "mcp") // stdio server; keep it alive with a live session
    A.Dir = testProjectDir(t, "conc")
    startA(t, A)
    defer A.Process.Kill()

    var fails int
    for i := 0; i < 20; i++ {
        cmd := exec.Command(bin, "mem", "save", "--title", fmt.Sprintf("c%d", i), "--type", "discovery",
            "--content", "concurrent", "--project", "conc", "--session", A.SessionID(), "--json")
        cmd.Dir = testProjectDir(t, "conc")
        if out, err := cmd.CombinedOutput(); err != nil { fails++; t.Logf("save %d: %v %s", i, err, out) }
    }
    if fails > 0 { t.Fatalf("%d/20 concurrent saves failed (WAL busy-timeout/retry regression)", fails) }
    // integrity: open the store, count, and run PRAGMA integrity_check via Ping + a count read
    st := openTestStoreIn(t, testProjectDir(t, "conc"), "conc")
    r, _ := st.Search(ctx, store.SearchQuery{Query: "concurrent", Project: "conc", Limit: 100})
    if len(r.Observations) != 20 { t.Fatalf("expected 20 rows, got %d (lost a write)", len(r.Observations)) }
}
```

- [ ] **Step 2: Run test to verify it fails (or passes — it is the backstop)**

Run: `cd mnemonic-v2 && go test ./internal/core/store/ -run TestConcurrentProcessWrites -v -count=1`
Expected: PASS on a healthy build. If it fails (a BUSY error leaks, or a row is lost), the fix lands in Task 1's retry wrapper / DSN and this test is re-run — that is the backstop doing its job. BDD: `[concurrent-process-wal]` RED until green.

- [ ] **Step 3: Full-suite gate**

Run: `cd mnemonic-v2 && gofmt -l . && go vet ./... && go build ./... && go test ./... -count=1`
Expected: zero gofmt output, vet clean, build clean, all tests PASS (the PG tests skip without `MNEMONIC_PG_TEST_DSN`).

- [ ] **Step 4: Commit**

```bash
git add mnemonic-v2/internal/core/store/concurrency_test.go
git commit -m "test(mnemonic-v2): two-process WAL concurrency backstop + full-suite gate

[skillgrid-context] change=2026-10-31-mnemonic-v2-memory task=T20"
```

---

## Hypothesis

**Claim:** the memory function of mnemonic v2 can be built as a dialect-agnostic `Store` (SQLite | Postgres) with LLM-first seams and explicit degradation, such that an agent's `mem_save` becomes a durable, deduped, sync-journaled fact in one local transaction and the llm-wiki can consume it via a save event — without the memory function depending on the wiki.
**Right condition:** Task 20's backstop passes (two processes, same file, 20/20 saves, no lost writes) AND the full suite is green AND a `mem_save` from the MCP surface produces a row retrievable by `mem_lookup` with the `degraded` envelope correct in both LLM-present and LLM-absent modes.
**Wrong condition:** Task 1's door check fails (a per-project SQLite file cannot be opened through the `Store` interface, or a saved observation is not findable by FTS5 trigram search with the `< 3 runes` LIKE fallback) — INV-2 does not hold and the blueprint is invalidated; or Task 18 cannot express a memory-domain feature in both dialects (S-1 violation) — the dual-mode claim collapses to SQLite-only.
**Thinnest MVP:** Task 1 (module + `Store` + one SQLite file + `AddObservation`/`GetObservation`/FTS `Search`) — everything else thickens this path.
**Door check:** Task 1, Step 2→4 — `TestAddGetObservationRoundTrip` **and** `TestDoorCheckFTSRoundTrip` (an `AddObservation` is findable by `Search`; a `< 3 runes` query hits via the LIKE fallback). Task 2 re-proves trigger sync + ranking against the *full* migration set — the two tasks share the door, Task 1 is the stop gate.

## Owed-Decision Gate

Input coverage check (every value the build produces has a named source):

| Value | Named source |
|---|---|
| `sync_id` format | spec `002-schema` §1 (`obs-`/`rel-`/`ses-`/`prm-` + 16 hex) — Task 1 |
| `normalized_hash` | Engram `hashNormalized` (lowercase + collapse + SHA-256) — Task 1 |
| BM25 weights + boosts | spec `01-memory/005` §3 / Engram `store.go:424-431` — Task 11 |
| `importance` formula | spec `002-schema` §2.2.3 / v1 `dream.go:382-400` — Task 5 |
| `review_after` months | spec `002-schema` §2.2.2 (type→months) + `006-config` §2 (config table) — Task 5 |
| `content_max_bytes` / `private_tag` | spec `006-config` §2 (32000 / `<private>`) — Task 5 |
| Relevance-gate term rules | spec `01-memory/003` §3 step 2 (verbatim) — Task 9 |
| Dedup pre-filter cap | spec `004-dedup` §3 (≤20) — Task 8 |
| Candidate per-source cap | spec `01-memory/003` §3 (default 3) — Task 9 |
| Debounce / batch / byte limits | spec `006-config` §2 (500ms / 100 / 8 MiB) + `01-memory/006` §5 — Tasks 16/17 |
| Lease TTL / backoff | Engram `manager.go` (60s lease, 5-min cap, 10 failures) — Task 17 |
| Chunk id | Engram `chunkcodec.go:13` (8 hex of SHA-256) — Task 16 |
| Six-section summary | spec `01-memory/002` §5 (verbatim) — Task 7 |
| Relation vocabulary | spec `002-schema` §2.5.1 (verbatim) — Tasks 9/10 |
| Type vocabulary | spec `002-schema` §2.2.1 (verbatim) — Task 5 |
| Module path | Go module convention + go.work (Task 1) — `github.com/devopstales/skillgrid/mnemonic-v2` |
| PG driver | ADR-0041 (this blueprint, Task 18) — `lib/pq` |

No value's source is unnamed. **No owed decisions.** The five one-way doors (D1–D5) are decisions already made by the spec zone or this blueprint; they are flagged for explicit user approval at their tasks, not left to the build.

**Decision note:** the blueprint's scope is the **memory function + its foundation** (the `00-foundation` specs are co-owned by this change per their `Owner:` line). The `code_*` and `wiki_*` tool families, the `llmwiki/` package, and the two plugin changes are *other changes* — this blueprint's `Store` interface and `SaveEvent` are the seams they plug into, and Task 1's interface is deliberately final-shaped (D2) so they can extend without a contract break. Out of scope and recorded: the HTTP mirror (Task 13's note — the `Handler` is its plug point), the `serve` dashboard (dropped in v2, no-op alias), and the `code_*`/`wiki_*` route registrations (their changes own them; the server's profile machinery is shared).

## Plan Review

- Verdict: **READY FOR EXECUTION** (after revision pass)
- Findings (critic subagent, 2026-11-01): 3 Critical, 8 Important, 8 Minor — **all fixed inline**:
  - **C1** importance's `retrieval_usage` had no producer → `observations.retrieval_usage` column (spec §2.2.3 amendment, T0) + `Store.BumpRetrievalUsage` called from the trail writer (Task 11) + `TestUsageRaisesImportance`.
  - **C2** door check didn't test the hypothesis → `TestDoorCheckFTSRoundTrip` (save → FTS hit + `<3-rune` LIKE fallback) added to Task 1; Task 2 re-proves against the full migration set (two locks, Task 1 is the stop gate).
  - **C3** Task 5's `degrade.Run(s.llm.Classify, s.hashFloor, …)` call site wouldn't compile → the final `dedup.Run(ctx, s.llm, st, project, scope, content, title)` shape is written in Task 5 (nil-seam floor path); Task 8 fills only the non-nil branch; the pipeline never changes again.
  - **I4** `MergedInto *int64` violated INV-6 → `MergedIntoSyncID *string`.
  - **I5** S-3 (never write PG directly on a save) contradicted Task 18's full-interface `PostgresStore` → S-3 amended (T0): `PostgresStore` is the direct-PG *dialect render*, distinct from the replication path (Task 16's remote schema).
  - **I6** `mem_suggest_topic_key` contract under-specified → pinned in Task 13's Interfaces (params, return shape, heuristic).
  - **I7** session-missing policy ambiguous → fixed: auto-register (shared, resolved project) in the save transaction; ended → 409 `ErrSessionEnded`; `project_owned` mismatch → `ErrSessionProjectMismatch`; tested in Task 7.
  - **I8** `mem_search_prompts` had no implementation/test → `Store.SearchPrompts` (Task 1 interface), `Service.SearchPrompts` + `TestPromptSearchFindsCaptured` (Task 11).
  - **I9** recency/stability boosts untested → `TestSearchRankingRecencyAndStability` (Task 2, dialect-parity source for Task 18).
  - **I10** null-embedder "errored" vs "absent" conflated + adapter count off → absent (nil/`none`, no retry, leg absent) vs failing (1 retry, floor) distinguished in Task 3 with tests; count corrected to 3 adapters + floors.
  - **I11** presumed repo state → T0 pre-execution checklist (acceptance.feature, ADR-0040/0041 + index rows, `mnemonic/`→`mnemonic_back/` rename, `state.yaml` repoint, spec amendments committed before code).
  - **M12–M20** commit hygiene (Task 12 per-file `git add`), profile count (20 agent / 6 admin, `mem_doctor` admin — pinned literally in the contract test), Task 9 gate table row fixed (`C++`/`C#` row → `true`, hand-verified), Task 1 ledger-FK `ON DELETE SET NULL` assertion (`TestLedgerFKSetNull`), `capture_prompt` → tri-state `*bool` (absent → true) with absent-arg test, PG HNSW index added (S-1 parity), `mem_review` `mark_reviewed` clock-reset test (Task 11), threat-matrix Task citation fixed (Task 4), HTTP mirror explicitly scoped out (Handler is its plug point).
- Reviewed: 2026-11-01


