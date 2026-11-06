# Mnemonic v2 — Repo-Map Function (Implementation Blueprint: steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2 — a new read-only domain package (`internal/repomap/`) + a new MCP tool + a CLI group + an AGENTS.md block edit + a config flag. The `Store` is **not** extended (ADR-0047); no new table, no new migration, no new dependency. QA floor: L3 (build + the `repomap` + `transport` + `cmd` package tests + go vet; mutation/security advisory per config).

**Build shape:** Tracer thread — the integration risk is the ranking engine + the two-leg read composition + the coverage envelope. A thin end-to-end `repo map` over a seeded code-index graph + a seeded memory store works first (Task 1's door check), then thickens (renderer → coverage → hubs → memory join → cross-graph kinds → context envelope/hook → emit/drift → transport/CLI/AGENTS.md → doctor/QA).

**Classification:** standard (no new trust boundary — the read is over the two existing legs; no public data migration; the one public contract is the `repo_map` MCP tool + the AGENTS.md block edit, both additive).

**Goal:** Build the repo-map function of mnemonic v2 — a single read-only, ranked (PageRank+IDF+context-prior), coverage-labeled, hub-flagged, token-budgeted navigation view over the project's **code structure** (the code-index graph) **and its accumulated memory** (the observations) — auto-injected as the agent's first-minute context via the MCP session-start hook, re-rankable per turn in <100ms, with a CI-able `--drift` check and a lean AGENTS.md trigger-table. **Retrieval contract (spec `000` §1, the code-index spec `000` §1.1):** the map is the **first verb** in the retrieval-as-reasoning composition — it *orients*; the agent *drills* (`code_search`/`code_read`/`code_resolve`/`code_path`/`mem_lookup` are separate tools). RAG chat, commits, and PRs are out of scope; the map does not build a second vector index (it reads the code-index's existing vector store, D-CI8, when the semantic context is needed).

**Pre-execution checklist (T0 — repo state the plan presumes; none of it is a task, all of it must be true before Task 1's code commit):**
- [ ] `findings.md` (this folder) committed — the 7-reference digest (aider / nekocode / openclaude / cyanheads / agentmd / SarthakMishra / JordanCoin) + the per-reference key facts + the "what we do differently" section.
- [ ] **The code-index change is SHIPPED** (`2026-10-31-mnemonic-v2-code-index`): `mnemonic-v2/internal/codeindex` (the `extract`/`graph`/`search` packages), the 7 code-index tables (`files`/`chunks`/`chunks_fts`/`symbols`/`edges`/`dep_edges`/`index_freshness`), and the `internal/codeindex/graph` methods (`Orient`/`Impact`/`Callees`/`Callers`) all exist. Serial development (per AGENTS.md § Rules): repo-map is not `current_change` until code-index ships.
- [ ] **The memory change is SHIPPED** (`2026-10-31-mnemonic-v2-memory`): `mnemonic-v2/internal/core/store` (the `Store` + `SQLiteStore` with the `RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams), `internal/core/syncid`, `internal/memory` (the `observations` save pipeline + the `EventBus`/`SaveEvent` + the `session_trails` append), the `observations`/`observations_fts`/`sessions`/`session_trails` tables, and `internal/transport/mcp` (the one server + the two-profile mechanism + the write queue) all exist and `cd mnemonic-v2 && go build ./...` exits 0.
- [ ] ADR-0047 (the repo-map read surface is a view, not a store extension) written under `.skillgrid/artifacts/` and added to the ASSUMPTIONS.md index (next number after the code-index change's 0043–0046 claim; confirmed against ASSUMPTIONS.md § LOCKED at T0).
- [ ] `state.yaml` `pipeline.current_change` repointed to `2026-11-01-mnemonic-v2-repo-map` **at T0 of execution** (this spec zone does not repoint it — serial development, code-index is current until it ships).
- [ ] `acceptance.feature` (this folder) committed **before** any `mnemonic-v2/` code commit (pre-commit zone guard, per AGENTS.md § Rules).

**Architecture:** The code-index change's `mnemonic-v2/` module (module `github.com/devopstales/skillgrid/mnemonic-v2`, Go 1.26, cgo-free) gains a **read-only** `repomap` function. `internal/repomap` is the new domain package: `rank` (the PageRank+IDF+context-prior engine), `render` (the three-tier renderer: signature lines, the token-budget walk, the dynamic cold-start budget), `coverage` (the versioned per-answer envelope), `hub` (the fan-in flag), `memjoin` (the query-time FTS join of observations onto path/package), `kinds` (the cross-graph kinds: focus/neighborhood, blast-radius, temporal), `envelope` (the session-start `ContextEnvelope`), `emit` (the `.skillgrid/repo-map.md` renderer + the `--drift` check), and `health` (the doctor map-health read). The package **never writes** to the two legs: it reads through the `Store`'s `RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams + the existing `internal/codeindex/graph` methods. `internal/transport/mcp` gains the `repo_map` tool + the session-start auto-injection hook (a thin adapter over the shared handler); `internal/transport/handler` gains `handler_repo.go` (the one shared handler set — the MCP + CLI meet here); `cmd/mnemonic` gains the `repo map` CLI group (delegated by the memory change's `005-transport-cli`). The AGENTS.md trigger-table lands in `.agents/skills/_shared/agent-config/block.md` (the single source of truth the CLI renders).

**Tech Stack:** Go 1.26 (go.work already pins 1.26.0); the code-index change's allowed set (`modernc.org/sqlite` incl. `/vec`, `mark3labs/mcp-go`, `gopkg.in/yaml.v3`, `lib/pq`, `gotreesitter`, `fsnotify`, conditionally `onnxer`). **No new dependency** (the PageRank is hand-rolled over the in-memory adjacency the read builds — no graph lib). The config keys (`hub_threshold`, `inject_tokens`, `repo_map.commit`, the auto-inject flag) ride the memory change's existing `yaml.v3` config load (ADR-0008).

**Spec:** `specs/03-repo-map/000-overview.md` (this folder — the one-spec research/design anchor the blueprint cites; the code-index change's 9-spec set is the substrate, not re-derived here). Product *why*: `.skillgrid/artifacts/00-prd.md` (FUNCTION 3 of 5). In-force ADRs: ADR-0031 (Go floor) + ADR-0040 (module) + ADR-0041 (PG driver) + the code-index change's ADR-0043…0046 (tree-sitter/fsnotify/onnxer/Store-extension) as substrate; this change adds ADR-0047.

**Findings:** `findings.md` (this folder) — the 7-reference fold digest. Every design decision below that rests on a reference cites it (the `findings.md` table row).

**The boundary (the SarthakMishra codemap fold):** the map **orients**; the agent **drills**. `code_search`/`code_read`/`code_resolve`/`code_path`/`mem_lookup` are separate tools (the retrieval-as-reasoning contract, code-index spec `000` §1.1). The map is the FIRST verb in the composition, not the whole composition. RAG chat, commits, PRs are explicitly OUT of scope. The map does NOT build a second vector index (it reads the code-index's existing vector store, D-CI8).

## Terms

Pointers only — canonical terms live in `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md`.

- **the repo-map** — the first-minute orientation surface: a single read-only, ranked, token-budgeted view over the code-index graph + the memory store (spec `000` §1).
- **the coverage contract** — the versioned per-answer envelope (`status`/`sources`/`issues`) that makes a partial graph never read as a complete one (the JordanCoin fold; T-CI-4 applied per-answer, spec `000` §5).
- **Developer Consideration** — the per-file "the one thing a contributor is most likely to get wrong," sourced from a saved memory observation (FTS-matched to path/package) in the first cut (the cyanheads fold, memory as the source, spec `000` §3).
- **the context envelope** — the single session-start return (map + coverage + hubs + memory summary + intent prior + budget) that is auto-injected (the JordanCoin context protocol, spec `000` §7).
- **the two deferred soft doors** — (a) the stored per-file LLM note fallback (`file_notes`); (b) the stored `code_ref` pointer. Named, not built (v2.1, spec `000` §10).

## Must-Haves (goal-backward verification)

Derived from the spec zone (`specs/03-repo-map/000-overview.md`) + the 7-reference fold
(`findings.md`). The verifier checks the result against this list, not just that tasks
exist. **The scenario name in `acceptance.feature` IS the SATISFIES tag below (1:1).**

**Truths** (observable behaviors that must hold):

- T1: **the PageRank + IDF + context-prior ranking engine.** Deterministic PageRank
  (damping 0.85, max 100 iters, ε=1e-6) over the file reference graph, edges weighted by
  reference count × IDF(symbol name) — a common name (`get`) down-weights at equal
  degree (the openclaude fold). The `context: []` param (recently-read paths/symbols) is
  a **PageRank prior** (the aider fold); `--focus <path>` is a **boost** (a prior on
  that subtree); `--focus <symbol>` is a **neighborhood** query (the cross-graph join).
  `--focus` is a ranking modifier, never a filter. A re-rank (the PageRank re-run with
  new priors) is **<100ms** (no re-parse — the index is unchanged).
- T2: **the three-tier renderer (signature lines, token budget, dynamic cold start).**
  Tier 1 = the ranked symbol tree emitting **signature lines** (parameters + types —
  the `symbols.signature` column; the aider fold: the output for a file contains the
  full `func`/`def`/`function` line, not a bare name). Tier 2 = the memory map. Tier 3 =
  the emitted doc. The **token budget is primary** (`--tokens N`, default **2048**) with
  `limit` as a hard row cap; the walk is top-down over ranked files and **stops at the
  budget**. The auto-inject budget is `inject_tokens` (config, default **1024**),
  **expanded ~2× on cold start** (session start, no files in context), shrunk
  mid-session (the aider dynamic-budget fold).
- T3: **the coverage contract on EVERY answer (the JordanCoin fold).** The output
  envelope is versioned (`schema_version: repo-map/v1`) and carries `coverage: { status:
  complete|partial|unavailable, sources: [{name, status: authoritative|mixed|fallback|
  timeout|unavailable|failed, languages?, detail?}], issues: [{code, path, message}] }`.
  A broken leg degrades OUT and is labeled (T-CI-2 posture); an empty graph with
  `coverage.status: unavailable` is NOT the same as an empty graph with
  `coverage.status: complete` ("nothing" vs. "couldn't tell"). The `sources[]` names the
  **per-language extraction tier** (D-TIER: `tree-sitter-rich` = authoritative,
  `tree-sitter-map` = mixed, `regex` = fallback). A timed-out/failed scan returns an
  **empty result with provenance**, not a silent empty graph and not a hard error
  (T-CI-4 applied per-answer).
- T4: **the hub flag in the orientation view (the JordanCoin fold).** Files with fan-in
  ≥ `hub_threshold` (config, default **3** importers) carry a `⚠️ HUB` tag + importer
  count in every map answer that lists that file (not just the blast-radius kind). The
  threshold is config-overridable.
- T5: **the memory join (topic hubs + per-file considerations).** Tier 2 = the memory
  map. The **topic hubs**: the `topic_key` tree → typed hubs → top-N by `importance`.
  The **per-file Developer Consideration**: saved observations **FTS-matched to the
  file's path/package** (the query-time FTS floor over `observations_fts`; the killer
  feature — the notes are accumulated memory, not LLM-generated). A file with no
  matching observation gets no consideration line (not a fabricated one).
- T6: **the cross-graph kinds (focus/neighborhood, blast-radius, temporal).** The `kind`
  parameter (progressive disclosure): `overview` (default: packages → file/symbol counts
  → topic hubs → recent decisions), `packages`, `symbols`, `topics`, **`focus`/
  neighborhood** (a `--focus <symbol>` cross-graph join: the symbol's neighborhood from
  the graph + the observations that reference it), **`blast-radius`** (the code
  reverse-BFS via the existing `graph.Impact` + "which observations reference this"),
  **`temporal`** (weekly bins: code re-indexes from `files.indexed_at` + memory growth
  from `observations.created_at`). A `--focus <symbol>` (neighborhood) is distinct from
  a `--focus <path>` (boost): the MCP tool takes an explicit `mode: boost|neighborhood`
  to remove the ambiguity.
- T7: **the context envelope + session-start hook + per-turn re-rank (the JordanCoin
  context-protocol + aider per-turn folds).** The session-start MCP hook call returns
  **ONE** `ContextEnvelope`: the map (Tier 1+2, token-budgeted at `inject_tokens`,
  expanded ~2× on cold start) + the coverage contract + the hub warnings + a memory
  summary (top-3 recent decisions + topic hubs) + the `--for <intent>` prior + the
  compact/full budget marker. This is what gets **auto-injected** (opt-in config flag).
  The hook **re-ranks per turn** if the conversation context shifted (new files read /
  symbols mentioned) — the PageRank re-run with new priors, **<100ms**. The on-demand
  `repo_map` MCP tool is always available.
- T8: **the `--emit` doc + `--drift` (the cyanheads + agentmd folds).** `--emit` writes
  `.skillgrid/repo-map.md`: the **per-file field contract** (Description = the
  signature-line floor; Developer Consideration = the memory observation; Maintenance
  Flag (Stable/Volatile/Generated/Unknown) from `files.content_hash` churn + the
  observation type mix; Architectural Role (Entrypoint/Service/API Route/Utility/Test/
  Configuration/…) from `edges` in/out-degree + `kind`; Critical Dependencies = top-N by
  confidence rank × fan-in, with one-line reasons) + a **header stamp** (`generated_at`
  + a content hash). **Gitignored by default**; a config flag `repo_map.commit: true`
  opts in to committing. **`--drift`** (the agentmd fold): exit 0 = doc fresh, exit 1 =
  doc stale vs. live state (stamp + content-hash comparison), CI-able JSON.
- T9: **the MCP tool + CLI group + AGENTS.md trigger-table + onboarding re-emit.** (1)
  the MCP `repo_map` tool (JSON; `mode`/`kind`/`focus`/`tokens`/`context`) on the agent
  profile; (2) the **session-start auto-injection hook** (the context envelope, the
  primary path); (3) the CLI `mnemonic repo map` (JSON + `--emit` + `--drift` +
  `--focus` + `--tokens` + `--kind` + `--context`); (4) the **emitted doc** re-emitted
  by onboarding when the code-index freshness flips `fresh→lag` or the memory schema
  version bumps. The **AGENTS.md instruction** is a **lean trigger-table** added INSIDE
  the existing `<!-- skillgrid:start -->` block in `.agents/skills/_shared/agent-config/
  block.md`: a compact When → Command table + a one-line trust rule, NOT a prose
  paragraph (block.md "Keep this block lean": navigation, not conventions; a line is
  added only on an observed failure; it loads on every prompt). The MCP hook is the
  PRIMARY path; the AGENTS.md instruction is the FALLBACK for harnesses without the
  hook. No new top-level section: it folds into the existing `### Memory` line or adds
  a minimal `### Code Map` sub-block of ≤6 lines.
- T10: **the doctor map-health surface + QA measurability.** The `doctor` surface gains
  a **map-health** line (the coverage status of the last map read + the leg status:
  code-index present/lag/absent, memory present/empty). **The QA measurability
  scenario:** a seeded observation for a seeded file surfaces in the map's
  consideration join (the end-to-end proof the memory join works against a real store).

**Degradation (INV-3, spec `000` §8) — a broken leg drops out and is labeled; it never
poisons the other leg (T-CI-2):**

- **D1:** code-index absent/lag → Tier 1 from a bounded live tree-sitter scan of the
  focus subtree, or `degraded: no_code_index` (the memory-only map still works).
- **D2:** memory empty → the code-only map.
- **D3:** both missing → `coverage.status: unavailable` + counts (never a silent empty
  graph, never a hard error).

**Artifacts** (files that must exist with real implementation, not stubs):

- `mnemonic-v2/internal/repomap/rank.go` — the PageRank+IDF+context-prior engine (T1).
- `mnemonic-v2/internal/repomap/render.go` — the three-tier renderer: the signature-
  line walk, the token-budget stop, the dynamic cold-start budget (T2).
- `mnemonic-v2/internal/repomap/coverage.go` — the versioned per-answer envelope (T3).
- `mnemonic-v2/internal/repomap/hub.go` — the fan-in ≥ `hub_threshold` flag (T4).
- `mnemonic-v2/internal/repomap/memjoin.go` — the query-time FTS join of observations
  onto path/package + the topic hubs (T5).
- `mnemonic-v2/internal/repomap/kinds.go` — the cross-graph kinds (T6).
- `mnemonic-v2/internal/repomap/envelope.go` — the session-start `ContextEnvelope` +
  the per-turn re-rank (T7).
- `mnemonic-v2/internal/repomap/emit.go` — the `.skillgrid/repo-map.md` renderer (the
  per-file field contract + the header stamp) + the `--drift` check (T8).
- `mnemonic-v2/internal/repomap/health.go` — the doctor map-health read (T10).
- `mnemonic-v2/internal/repomap/types.go` — the Go result/param types (MapResult,
  MapParams, Coverage, Envelope, PerFile, …; S-2 rule: Go types only).
- `mnemonic-v2/internal/transport/handler/handler_repo.go` — the one shared repo-map
  handler set (the MCP + CLI surfaces meet here; never diverge).
- `mnemonic-v2/internal/transport/mcp/tools_repo.go` — the `repo_map` tool + the
  session-start auto-injection hook (thin adapters over `handler_repo`).
- `mnemonic-v2/cmd/mnemonic/repo.go` — the `mnemonic repo map` CLI group (delegated by
  the memory change's `005-transport-cli`).
- `mnemonic-v2/internal/repomap/*_test.go` + the transport/CLI tests — tests for every
  truth above.
- `.agents/skills/_shared/agent-config/block.md` — the ≤6-line lean trigger-table
  (T9). The onboarding CLI renders it (no code change to the render path — the block
  edit is the render change).
- `.skillgrid/artifacts/04-adr-0047-repo-map-view-not-store.md` — the view-not-store
  decision.

**Key links** (critical connections that must work together):

- K1: `repo_map` MCP tool / `mnemonic repo map` CLI → `handler_repo.Map` →
  `repomap.Map` (read the two legs → `rank.Rank` → `render.Render` under the budget →
  `coverage.Coverage` → the trail append) → the one shared result (byte-identical MCP/
  CLI JSON, the K5 parity). A broken leg degrades out + is labeled (T3, D1/D2/D3).
- K2: `rank.Rank` reads the file reference graph from `edges` (via the `RawQuery` seam
  or the `graph` package) + the IDF over `symbols.name` + the `context` prior. The
  re-rank (new priors, same index) is <100ms (no re-parse).
- K3: the session-start hook → `envelope.Build` → the map (T2) + coverage (T3) + hubs
  (T4) + memory summary (T5) + intent prior (T1) → the auto-injected envelope. A
  per-turn call (new context) → `rank.Rank` re-run with the new prior → <100ms.
- K4: `memjoin.Join` FTS-matches `observations_fts` (title/content/topic_key) to the
  file's path/package → the per-file Developer Consideration + the topic hubs. The join
  is query-time (the deferred `code_ref` door, v2.1).
- K5: `cmd/mnemonic/repo.go` + the MCP `repo_map` produce byte-identical JSON (same
  handler, same result marshal) — a parity test asserts it.
- K6: the AGENTS.md block edit (block.md) is the FALLBACK for harnesses without the
  MCP hook; the hook is the PRIMARY path. Onboarding re-emits the doc on the
  `fresh→lag` flip (T9).

**One-way-door decisions** (hard to reverse — flag for explicit user approval at
execution):

- D1: **the `repo_map` MCP tool contract** — the opencode/pi plugins consume it.
  Renaming the tool, adding a required param, or changing a return shape after the
  plugins build breaks them. Task 8.
- D2: **the AGENTS.md block edit** (the ≤6-line trigger-table in block.md) — it renders
  into every enrolled project's AGENTS.md at init/onboarding; a later re-shape is a
  re-render of every project. Task 8. (Soft: the block is a navigation spine, not a
  data contract — the re-shape is cheap. Flagged for visibility, not a hard door.)
- **The two deferred soft doors (named, not built, v2.1):** (a) the stored per-file LLM
  note fallback (`file_notes` table) for files with zero memory; (b) the stored
  `code_ref` pointer (observation → symbol) instead of query-time FTS. Both are soft
  (a table + a column; reversible). This change does not reach for them.

## Global Constraints

Every task's requirements implicitly include this section.

- **Go 1.22+ minimum to build** (per ADR-0031); the go.work floor is 1.26.0, so
  `mnemonic-v2/go.mod` declares `go 1.26.0` (already set by the memory change).
- **No new dependencies without an ADR** (per AGENTS.md § Rules): the allowed set is the
  code-index change's (`modernc.org/sqlite` incl. `/vec`, `mark3labs/mcp-go`,
  `gopkg.in/yaml.v3`, `lib/pq`, `gotreesitter`, `fsnotify`, conditionally `onnxer`).
  The PageRank is hand-rolled (no graph lib). Anything else needs a new ADR before it
  lands.
- **cgo-free** (the v1 VERIFIED invariant; the repo-map adds no CGo).
- **Conventional commits only; no AI-attribution trailers** (commit-msg hook enforces,
  per AGENTS.md § Rules).
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard enforces,
  per AGENTS.md § Rules): this blueprint + `acceptance.feature` + `tasks.md` +
  `findings.md` are committed before any `mnemonic-v2/` code commit.
- **One change at a time, no parallel branches** (serial development, per AGENTS.md §
  Rules). A commit that closes a Backlog task names that task ID in the
  `[skillgrid-context]` block (per ADR-0013).
- **Trivy is advisory-only** — findings reported, never blocking the QA gate.
- **Read-only over the two legs** (RM-INV-1, ADR-0047): no new table, no new migration,
  no `Store` interface extension. The read goes through the `RawQuery`/`Probe`/
  `ProbeInt`/`RawExec` seams + the existing `internal/codeindex/graph` methods. The
  only writes are the retrieval-trail row (INV-5) and the emitted doc (a file, not a
  table).
- **All invariants** (memory `00-foundation/000-architecture` §4) + the repo-map
  invariants (RM-INV-1 view-not-store, RM-INV-2 first-verb-not-whole-composition,
  RM-INV-3 deterministic ranking) are binding on every task: INV-3 explicit degradation
  (a broken leg degrades out + is labeled), INV-5 retrieval trails (every `repo_map`
  call records ONE trail row), INV-6 identity is `sync_id` + `topic_key` + file path,
  never rowid.
- **Test command:** `cd mnemonic-v2 && go test ./internal/repomap/... ./internal/
  transport/... ./cmd/...`; **build:** `cd mnemonic-v2 && go build ./...`; **format:**
  `gofmt -w ./...`; vet inline via `go vet ./...`.

## Threat Matrix

Per `_shared/references/threat-matrix.md`. The change touches **Mnemonic tool
contracts** (the new `repo_map` MCP tool + the session-start hook) and **documentation-
like paths** (the emitted doc `.skillgrid/repo-map.md` + the AGENTS.md block edit).

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Mnemonic tool surface (`repo_map`) | Applicable: the new `repo_map` MCP tool + the session-start auto-injection hook are public contracts the opencode/pi plugins consume | The tool contract is pinned by tests: exact tool name, required params, return shape (the versioned envelope), error codes, and profile membership are asserted; a contract drift = a failing test, not a silent break. The MCP/CLI parity is asserted byte-identical. | Task 8 Step 1 (`TestRepoMapToolContract` — golden JSON), Task 8 Step 2 (`TestRepoMapProfileMembership` — agent profile), Task 8 Step 3 (`TestRepoMapMcpCliParity` — byte-identical), Task 1 Step 1 (`TestDoorCheckRepoMapReadsAndRanks`) |
| Documentation-like paths | Applicable: the emitted doc (`.skillgrid/repo-map.md`) is a markdown file the agent reads; the AGENTS.md block edit (block.md) renders into every enrolled project | The emitted doc is gitignored by default (a config flag opts in); its per-file fields + header stamp are asserted by a golden-file test. The block.md edit is a ≤6-line trigger-table (no prose paragraph, no new top-level section) — asserted by a block-shape test. Onboarding re-emits on the `fresh→lag` flip. | Task 8 Step 4 (`TestRepoMapEmitPerFileFields` — the five fields + the header stamp), Task 8 Step 5 (`TestRepoMapEmitGitignoreDefaultCommitFlag`), Task 8 Step 6 (`TestAgentsMdLeanTriggerTable` — ≤6 lines, no new top-level section), Task 8 Step 7 (`TestOnboardingReemitOnLag`) |
| Process integration | Applicable: the session-start hook is an MCP server call; the per-turn re-rank is an in-process PageRank re-run | The hook is a thin adapter over the shared handler (the MCP + CLI meet in `handler_repo`); the re-rank is a pure in-memory PageRank (no I/O) — the <100ms bound is asserted on a seeded graph. | Task 7 Step 1 (`TestPerTurnRerankUnder100ms`), Task 7 Step 2 (`TestContextEnvelopeOneCall`) |
| Git repository selection | Applicable: project detection resolves cwd → git binding (the memory change's `005-transport-cli` convention); the emitted doc path is relative to the project root | Detection re-uses the memory change's project-detection selectors (no new detection). The emitted doc path (`.skillgrid/repo-map.md`) is relative to the detected root. | Task 8 Step 4 (the emit test uses the memory change's project-detection helper) |
| Commit state | Applicable: `repo_map.commit: true` opts in to committing the doc (the onboarding render, not the engine) | The engine never stages or commits; the opt-in is a config flag the onboarding CLI reads (it renders the doc into the tree; the commit is the user's action). | Task 8 Step 5 (`TestRepoMapEmitGitignoreDefaultCommitFlag`) |
| Push state | N/A: no git push from the engine | — | — |
| PR commands | N/A: no PR automation (the toolkit boundary, the SarthakMishra fold) | — | — |
| Shared-convention drift | Applicable: the block.md edit is a `_shared/agent-config/` file (the single source of truth the CLI renders) | The edit is the ≤6-line trigger-table (navigation, not conventions); the block.md "Keep this block lean" rule is the constraint (no prose paragraph, no new top-level section). | Task 8 Step 6 (`TestAgentsMdLeanTriggerTable`) |

## File Structure

```
mnemonic-v2/
├── go.mod / go.sum                     module github.com/devopstales/skillgrid/mnemonic-v2
│                                       (go 1.26.0; NO new dep — the PageRank is hand-rolled)
├── cmd/mnemonic/
│   └── repo.go                         the `mnemonic repo map` CLI group (delegated by the
│                                       memory change's 005-transport-cli):
│                                       map [--kind K] [--focus F] [--mode M] [--tokens N]
│                                       [--context C,…] [--emit] [--drift] [--for I] [--json]
└── internal/
    ├── repomap/                        THE NEW DOMAIN PACKAGE (read-only over the two legs)
    │   ├── types.go                    MapParams, MapResult, Coverage, Source, Issue, Envelope,
    │   │                               PerFile, FileRow, TopicHub, … (Go types only, the S-2 rule)
    │   ├── rank.go                     the PageRank+IDF+context-prior engine (T1):
    │   │                               damping 0.85 / 100 iters / ε=1e-6 / deterministic
    │   ├── render.go                   the three-tier renderer (T2): the signature-line walk,
    │   │                               the token-budget stop, the dynamic cold-start budget
    │   ├── coverage.go                 the versioned per-answer envelope (T3):
    │   │                               status/sources/issues + the per-language tier (D-TIER)
    │   ├── hub.go                      the fan-in ≥ hub_threshold flag (T4): the ⚠️ HUB tag
    │   ├── memjoin.go                  the query-time FTS join (T5): observations_fts →
    │   │                               path/package + the topic hubs (top-N by importance)
    │   ├── kinds.go                    the cross-graph kinds (T6): overview/packages/symbols/
    │   │                               topics/focus/blast-radius/temporal
    │   ├── envelope.go                 the session-start ContextEnvelope (T7): map + coverage +
    │   │                               hubs + memory summary + intent prior + budget;
    │   │                               the per-turn re-rank (<100ms)
    │   ├── emit.go                     the .skillgrid/repo-map.md renderer (T8): the per-file
    │   │                               field contract + the header stamp + the --drift check
    │   └── health.go                   the doctor map-health read (T10): coverage status + leg status
    ├── transport/
    │   ├── handler/handler_repo.go     the one shared repo-map handler set (MCP + CLI meet here)
    │   └── mcp/tools_repo.go           the repo_map tool + the session-start auto-injection hook
    └── core/
        └── store/                      UNCHANGED (no new method, no new table — ADR-0047):
                                        the RawQuery/Probe/ProbeInt/RawExec seams + the
                                        observations/observations_fts/sessions/session_trails tables
.agents/skills/_shared/agent-config/block.md   the ≤6-line lean trigger-table (T9; the single
                                                source of truth the CLI renders)
.skillgrid/artifacts/04-adr-0047-repo-map-view-not-store.md   the view-not-store decision
```

Responsibility notes: `internal/repomap` never names `*sql.DB` (the S-2 rule — it sees only
the `Store` interface's seam methods + the `internal/codeindex/graph` methods). It never
writes to the two legs (the only writes are the trail row + the emitted file).
`internal/transport/handler_repo.go` is the seam where the MCP + CLI surfaces meet — both
are thin adapters. The code-index change's `internal/codeindex` is **not** imported for its
`search`/`vectorstore` (the map reads the graph + the raw tables; the semantic context, if
needed, reads the existing vector store per D-CI8).

---

### Task 1: The PageRank + IDF + context-prior ranking engine + the thin end-to-end read (door check)

> ⚠ **Door check — this task tests the hypothesis.** If a `repo map` cannot rank a seeded code-index graph (deterministic PageRank+IDF, the context prior raises the rank) and emit a token-budgeted signature tree with a coverage envelope over a seeded memory store, stop: the repo-map's foundation (the two-leg read + the ranking engine) does not hold and the blueprint is invalidated.

**Files:**
- Create: `mnemonic-v2/internal/repomap/types.go`, `rank.go`, `render.go`, `coverage.go`
- Test: `mnemonic-v2/internal/repomap/rank_test.go`, `render_test.go`, `door_test.go`

**Interfaces:**
- Consumes: the memory change's `store.Open` + the `SQLiteStore` seams (`RawQuery`/`Probe`/`ProbeInt`/`RawExec`); the code-index change's `files`/`symbols`/`edges`/`index_freshness` tables + the `internal/codeindex/graph` methods; the `observations`/`observations_fts` tables.
- Produces:
  - `types.go`: `MapParams{Kind, Focus, Mode, Tokens, Limit int, Context []string, For string, ColdStart bool}`; `MapResult{SchemaVersion string, Map []MapRow, Coverage Coverage, Degraded []string, TrailID int64}`; `MapRow{Path, Signature string, Rank float64, Hub bool, HubCount int, Consideration string}`; `Coverage{Status string, Sources []Source, Issues []Issue}`; `Source{Name, Status string, Languages []string, Detail string}`; `Issue{Code, Path, Message string}`.
  - `rank.go`: `Rank(graph FileRefGraph, ctx Context, focus Focus) []RankedFile` — the deterministic PageRank (damping 0.85, max 100 iters, ε=1e-6) over the file reference graph, edges weighted by reference count × IDF(symbol name); the `context` param is a PageRank **prior**; `focus` is a boost (a path) or a neighborhood (a symbol, the cross-graph join). The re-rank (new priors, same index) is <100ms (no re-parse).
  - `render.go`: `Render(ranked []RankedFile, budget TokenBudget) []MapRow` — the top-down walk over ranked files, **stopping at the token budget**; the signature line is the `symbols.signature` column (parameters + types); `limit` is a hard row cap. `TokenBudget{InjectTokens int, ColdStart bool}` — the inject budget (default 1024), **expanded ~2× on cold start**.
  - `coverage.go`: `Coverage(legs LegStatus) Coverage` — the versioned envelope (`schema_version: repo-map/v1`); `status: complete|partial|unavailable`; `sources[]` names the per-language extraction tier (D-TIER: `tree-sitter-rich` = authoritative, `tree-sitter-map` = mixed, `regex` = fallback); `issues[]` names the broken leg + the stale subtree. A timed-out leg → `status: partial` + `issues[]` (never a silent empty graph, never a hard error).
  - `door_test.go`: `TestDoorCheckRepoMapReadsAndRanks` — a seeded code-index graph (files/symbols/edges) + a seeded memory store (observations with topic_key + a path mention); the map ranks the graph deterministically (two runs, identical order), emits a token-budgeted signature tree (the walk stops at the budget; each row carries the full `func` line), and the coverage envelope is `complete` with the per-language tier in `sources[]`.
- Seam: the `Store` interface's seam methods (INV-2: functions see only it) + the `graph` package.
- Deletion test: deleting `rank.go` + `render.go` + `coverage.go` re-creates the ranking + the walk + the envelope in every caller — the callers are `envelope.go` (Task 7) + `kinds.go` (Task 6) + `handler_repo.go` (Task 8).
- Adapters: the `SQLiteStore` seams (this task) + the in-memory test store = 2, seam justified (the Postgres dialect re-uses the same seams; no new method).

**SATISFIES:** [door-check-repo-map], [ranking-deterministic], [ranking-idf-downweights-common], [ranking-context-prior-raises], [signature-line-output], [token-budget-walk-stops], [coverage-complete], [coverage-partial-timed-out-leg], [coverage-unavailable-both-missing]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/repomap/door_test.go
// THE door check (hypothesis falsifier): a repo map ranks a seeded code-index graph
// (deterministic PageRank+IDF, the context prior raises the rank) and emits a
// token-budgeted signature tree with a coverage envelope over a seeded memory store.
func TestDoorCheckRepoMapReadsAndRanks(t *testing.T) {
    st := openSeededStore(t) // helper: a seeded code-index graph (files/symbols/edges)
    //   + a seeded memory store (observations with topic_key + a path mention)
    // the deterministic rank: two runs, identical order
    r1 := rank.Run(seedGraph(), Context{}, Focus{})
    r2 := rank.Run(seedGraph(), Context{}, Focus{})
    if !slices.Equal(r1.Order(), r2.Order()) { t.Fatalf("rank not deterministic") }
    // the token-budgeted signature tree: the walk stops at the budget
    rows := render.Render(r1, TokenBudget{InjectTokens: 1024, ColdStart: false})
    if render.TokensUsed(rows) > 1024 { t.Fatalf("walk overran the budget") }
    // each row carries the full signature line (not a bare name)
    for _, row := range rows {
        if !strings.Contains(row.Signature, "func ") { t.Fatalf("row is not a signature line: %q", row.Signature) }
    }
    // the coverage envelope is complete + the per-language tier is in sources[]
    cov := coverage.Coverage(legs{CodeIndex: "present", Memory: "present"})
    if cov.Status != "complete" { t.Fatalf("expected complete, got %q", cov.Status) }
    if len(cov.Sources) == 0 || cov.Sources[0].Status != "authoritative" { t.Fatalf("per-language tier missing") }
}

// mnemonic-v2/internal/repomap/rank_test.go
// T1: the IDF down-weights common names; the context prior raises the rank.
func TestRankingIDFDownweightsCommon(t *testing.T) {
    g := seedGraphCommonVsRare() // a common name (get) + a rare name (capturePayment), equal degree
    r := rank.Run(g, Context{}, Focus{})
    if r.Rank("rare.go") < r.Rank("common.go") { t.Fatalf("IDF did not down-weight the common name") }
}
func TestRankingContextPriorRaises(t *testing.T) {
    g := seedGraph()
    base := rank.Run(g, Context{}, Focus{})
    withCtx := rank.Run(g, Context{Paths: []string{"internal/memory/pipeline.go"}}, Focus{})
    if withCtx.Rank("internal/memory/pipeline.go") - base.Rank("internal/memory/pipeline.go") <= 0 {
        t.Fatalf("context prior did not raise the rank")
    }
}
// mnemonic-v2/internal/repomap/render_test.go
// T3: the coverage partial (a timed-out leg) + the unavailable (both missing).
func TestCoveragePartialTimedOutLeg(t *testing.T) {
    cov := coverage.Coverage(legs{CodeIndex: "timeout", Memory: "present"})
    if cov.Status != "partial" || len(cov.Issues) == 0 { t.Fatalf("expected partial + issues, got %+v", cov) }
}
func TestCoverageUnavailableBothMissing(t *testing.T) {
    cov := coverage.Coverage(legs{CodeIndex: "absent", Memory: "empty"})
    if cov.Status != "unavailable" { t.Fatalf("expected unavailable, got %q", cov.Status) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestDoorCheckRepoMapReadsAndRanks|TestRankingIDFDownweightsCommon|TestRankingContextPriorRaises|TestCoveragePartialTimedOutLeg|TestCoverageUnavailableBothMissing' -v`
Expected: FAIL — `rank.Run` / `render.Render` / `coverage.Coverage` undefined (build error: "rank.Run undefined"). BDD: `[door-check-repo-map]`, `[ranking-deterministic]`, `[ranking-idf-downweights-common]`, `[ranking-context-prior-raises]`, `[signature-line-output]`, `[token-budget-walk-stops]`, `[coverage-complete]`, `[coverage-partial-timed-out-leg]`, `[coverage-unavailable-both-missing]` are RED in `acceptance.feature`.

- [ ] **Step 3: Write minimal implementation**

`types.go`: the Go types above. `rank.go`: the deterministic PageRank — build the in-memory adjacency from the `edges` table (via the `RawQuery` seam: `SELECT from_id, to_id, kind, confidence FROM edges`), compute the IDF over `symbols.name` (`idf(name) = log(N / df(name))`, `df` = the number of distinct files referencing the name), weight each edge by `refCount × idf`, run the Power Iteration (damping 0.85, max 100 iters, stop when the L1 delta < ε=1e-6), apply the `context` prior (add a constant to the target nodes' initial distribution — the aider fold), and the `focus` boost (a prior on the subtree) or neighborhood (the cross-graph join: the symbol's `Callers`/`Callees` + the observations that reference it). `render.go`: the top-down walk over the ranked files, emitting the `symbols.signature` line per file, **stopping when the cumulative token count (a rough `len(text)/4`) exceeds the budget**; `limit` is a hard row cap; the cold-start budget is `injectTokens × 2`. `coverage.go`: the versioned envelope — `status` from the leg status (`present`+`present` = complete; one `timeout`/`failed`/`lag` = partial; both `absent`/`empty` = unavailable), `sources[]` from the code-index `index_freshness.extractor_tiers` (D-TIER) + the memory leg, `issues[]` naming the broken leg + the stale subtree. The trail append (INV-5) is a `RawExec` into `session_trails` (the same seam the memory change's `trails.go` uses).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/repomap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/repomap/
git commit -m "feat(mnemonic-v2): repo-map ranking engine + thin end-to-end read, door check

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T01 door-check"
```

---

### Task 2: The hub flag + the memory join (topic hubs + per-file considerations)

**Files:**
- Create: `mnemonic-v2/internal/repomap/hub.go`, `memjoin.go`
- Modify: `mnemonic-v2/internal/repomap/render.go` (the hub tag on the row; the consideration line)
- Test: `mnemonic-v2/internal/repomap/hub_test.go`, `memjoin_test.go`

**Interfaces:**
- Consumes: Task 1's `RankedFile` + the `edges` table (fan-in via `RawQuery`: `SELECT from_id, COUNT(DISTINCT file_id) FROM edges WHERE to_id IN (symbols of file) GROUP BY from_id`).
- Produces:
  - `hub.go`: `Hub(ranked []RankedFile, threshold int) []RankedFile` — flags every file with fan-in ≥ `threshold` (default **3**) with `Hub: true` + `HubCount` (the importer count). The threshold is config-overridable (`hub_threshold`).
  - `memjoin.go`: `Join(store store.Store, files []FileRow) map[string]Consideration` — the **query-time FTS join**: for each file, FTS-match `observations_fts` (title/content/topic_key) against the file's path + package; the match is the file's Developer Consideration (the cyanheads fold, memory as the source). A file with no match gets no consideration line (not a fabricated one). `TopicHubs(store store.Store, project string, n int) []TopicHub` — the `topic_key` tree → typed hubs → top-N by `importance`.
  - The `render.go` row gains the `Hub`/`HubCount` (the `⚠️ HUB` tag) + the `Consideration` (the memory observation).
- Seam: the `Store` interface's seam methods + the `graph` package.
- Deletion test: deleting `hub.go` + `memjoin.go` re-creates the fan-in flag + the FTS join in every caller.
- Adapters: same as Task 1.

**SATISFIES:** [hub-flag-3-importers], [hub-threshold-config], [memory-join-surfaces-seeded-observation], [memory-topic-hubs-by-importance], [memory-join-absent-not-fabricated]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/repomap/hub_test.go
// T4: a file with 3 importers gets the HUB tag + count; 2 does not; the threshold is config.
func TestHubFlag3Importers(t *testing.T) {
    g := seedGraphHub() // a file with 3 importers + a file with 2
    ranked := rank.Run(g, Context{}, Focus{})
    flagged := hub.Hub(ranked, 3)
    if !flagged.ByPath("hub.go").Hub || flagged.ByPath("hub.go").HubCount != 3 { t.Fatalf("3 importers must flag HUB + count") }
    if flagged.ByPath("leaf.go").Hub { t.Fatalf("2 importers must not flag") }
}
func TestHubThresholdConfig(t *testing.T) {
    g := seedGraphHub()
    flagged := hub.Hub(rank.Run(g, Context{}, Focus{}), 2) // threshold lowered to 2
    if !flagged.ByPath("leaf.go").Hub { t.Fatalf("threshold 2 must flag the 2-importer file") }
}

// mnemonic-v2/internal/repomap/memjoin_test.go
// T5: a seeded observation for a seeded file surfaces; a file with no match gets none.
func TestMemoryJoinSurfacesSeededObservation(t *testing.T) {
    st := openSeededStore(t) // an observation mentioning internal/memory/pipeline.go
    cons := memjoin.Join(st, []FileRow{{Path: "internal/memory/pipeline.go"}})
    if cons["internal/memory/pipeline.go"] == "" { t.Fatalf("seeded observation did not surface") }
}
func TestMemoryJoinAbsentNotFabricated(t *testing.T) {
    st := openEmptyMemoryStore(t)
    cons := memjoin.Join(st, []FileRow{{Path: "internal/pay/pay.go"}})
    if cons["internal/pay/pay.go"] != "" { t.Fatalf("no observation must mean no consideration line") }
}
func TestMemoryTopicHubsByImportance(t *testing.T) {
    st := openSeededStore(t) // topic_keys with importance
    hubs := memjoin.TopicHubs(st, "p", 5)
    // top-N by importance (the highest-importance topic is first)
    if len(hubs) == 0 || hubs[0].Importance < hubs[len(hubs)-1].Importance { t.Fatalf("hubs not ordered by importance") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestHubFlag3Importers|TestHubThresholdConfig|TestMemoryJoinSurfacesSeededObservation|TestMemoryJoinAbsentNotFabricated|TestMemoryTopicHubsByImportance' -v`
Expected: FAIL — `hub.Hub` / `memjoin.Join` / `memjoin.TopicHubs` undefined (build error). BDD: `[hub-flag-3-importers]`, `[hub-threshold-config]`, `[memory-join-surfaces-seeded-observation]`, `[memory-topic-hubs-by-importance]`, `[memory-join-absent-not-fabricated]` are RED.

- [ ] **Step 3: Write minimal implementation**

`hub.go`: `Hub(ranked, threshold)` — for each ranked file, query the fan-in via `RawQuery` (`SELECT COUNT(DISTINCT e.from_id) FROM edges e JOIN symbols s ON e.to_id = s.id WHERE s.file_id = ?`), flag `Hub: true` + `HubCount` when `fanIn >= threshold`. The threshold is the config `hub_threshold` (default 3; the caller passes it). `memjoin.go`: `Join(store, files)` — for each file, build the FTS match query (`SELECT sync_id, title, content, importance FROM observations_fts WHERE observations_fts MATCH ? AND project = ?` with the query = the file's path tokens + the package name, the trigram tokenizer), take the top match (by BM25 rank) as the consideration; no match → empty (not fabricated). `TopicHubs(store, project, n)` — `SELECT topic_key, type, COUNT(*) as cnt, AVG(importance) as imp FROM observations WHERE project = ? AND topic_key IS NOT NULL AND deleted_at IS NULL GROUP BY topic_key, type ORDER BY imp DESC LIMIT n` (via `RawQuery`). The `render.go` row carries the `Hub`/`HubCount` + the `Consideration`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/repomap/
git commit -m "feat(mnemonic-v2): repo-map hub flag + memory join (topic hubs + considerations)

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T02"
```

---

### Task 3: The cross-graph kinds (focus/neighborhood, blast-radius, temporal) + the focus-mode disambiguation

**Files:**
- Create: `mnemonic-v2/internal/repomap/kinds.go`
- Test: `mnemonic-v2/internal/repomap/kinds_test.go`

**Interfaces:**
- Consumes: Task 1's `rank.Run` + the `graph` package (`Impact` for the blast-radius; `Callees`/`Callers` for the neighborhood) + the `files`/`observations` tables (the temporal bins).
- Produces:
  - `kinds.go`: `Kind(params MapParams) (KindResult, error)` — the `kind` dispatch: `overview` (default: packages → file/symbol counts → topic hubs → recent decisions), `packages` (the package tree + import edges from `edges` kind=imports), `symbols` (the per-package symbol TOC with kinds from `symbols.kind`), `topics` (the `memjoin.TopicHubs` + the `topic_key` tree), `focus`/neighborhood (a `Focus{Symbol}` → the symbol's `Callers`/`Callees` + the observations that reference it — the cross-graph join), `blast-radius` (the `graph.Impact` reverse-BFS + "which observations reference this" via `memjoin`), `temporal` (weekly bins: `files.indexed_at` re-indexes + `observations.created_at` memory growth).
  - The **focus-mode disambiguation**: the CLI disambiguates by argument shape (path/directory → `Mode: boost`; symbol name → `Mode: neighborhood`); the MCP tool takes an explicit `Mode` param (`boost|neighborhood`) to remove the ambiguity.
- Seam: the `Store` interface's seam methods + the `graph` package.
- Deletion test: deleting `kinds.go` re-creates the kind dispatch in every caller.
- Adapters: same as Task 1.

**SATISFIES:** [kind-blast-radius-cross-graph], [kind-temporal-weekly-bins], [focus-mode-disambiguation]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/repomap/kinds_test.go
// T6: the blast-radius kind returns the code reverse-BFS + the observations that reference it.
func TestKindBlastRadiusCrossGraph(t *testing.T) {
    st := openSeededStore(t) // a symbol with dependents + an observation referencing it
    res, err := kinds.Kind(st, MapParams{Kind: "blast-radius", Focus: "CapturePayment"})
    if err != nil { t.Fatalf("blast-radius: %v", err) }
    if len(res.CodeAffected) == 0 { t.Fatalf("no code reverse-BFS set") }
    if len(res.MemoryRefs) == 0 { t.Fatalf("no observations that reference the symbol") }
}
// T6: the temporal kind returns weekly bins for both legs.
func TestKindTemporalWeeklyBins(t *testing.T) {
    st := openSeededStore(t) // files.indexed_at + observations.created_at across 2 weeks
    res, err := kinds.Kind(st, MapParams{Kind: "temporal"})
    if err != nil { t.Fatalf("temporal: %v", err) }
    if len(res.WeeklyBins) < 2 { t.Fatalf("expected >=2 weekly bins") }
    if res.WeeklyBins[0].CodeReindexes == 0 && res.WeeklyBins[0].MemoryGrowth == 0 { t.Fatalf("bin is empty") }
}
// T6: a focus <symbol> (neighborhood) is distinct from a focus <path> (boost).
func TestFocusModeDisambiguation(t *testing.T) {
    st := openSeededStore(t)
    // a symbol focus → the neighborhood (the cross-graph join)
    nb, _ := kinds.Kind(st, MapParams{Kind: "focus", Focus: "CapturePayment", Mode: "neighborhood"})
    if nb.Neighborhood == nil { t.Fatalf("symbol focus must be a neighborhood") }
    // a path focus → the boost (the rank prior)
    bst, _ := kinds.Kind(st, MapParams{Kind: "overview", Focus: "internal/memory", Mode: "boost"})
    if bst.Boosted == "" { t.Fatalf("path focus must be a boost") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestKindBlastRadiusCrossGraph|TestKindTemporalWeeklyBins|TestFocusModeDisambiguation' -v`
Expected: FAIL — `kinds.Kind` undefined (build error). BDD: `[kind-blast-radius-cross-graph]`, `[kind-temporal-weekly-bins]`, `[focus-mode-disambiguation]` are RED.

- [ ] **Step 3: Write minimal implementation**

`kinds.go`: the `Kind(params)` dispatch. `overview` (default): the `code_packages`-style package map (from `files` language='go' + `symbols`) + the file/symbol counts + the `memjoin.TopicHubs` + the recent decisions (`SELECT title, created_at FROM observations WHERE type = 'decision' ORDER BY created_at DESC LIMIT 3`). `packages`: the package tree + the import edges (`SELECT from_id, to_id FROM edges WHERE kind = 'imports'`). `symbols`: the per-package TOC (`SELECT name, kind FROM symbols WHERE file_id IN (package files) ORDER BY start_line`). `topics`: the `memjoin.TopicHubs` + the `topic_key` tree (`SELECT topic_key, COUNT(*) FROM observations WHERE topic_key IS NOT NULL GROUP BY topic_key`). `focus`/neighborhood: `Mode: neighborhood` → the symbol's `graph.Callees`/`graph.Callers` + the `memjoin` observations that reference it; `Mode: boost` → the `rank.Run` with the path as the prior. `blast-radius`: the `graph.Impact` (the code-index change's reverse-BFS) + the `memjoin` observations that reference the symbol. `temporal`: the weekly bins (`SELECT strftime('%Y-W%W', indexed_at) as wk, COUNT(*) FROM files GROUP BY wk` + `SELECT strftime('%Y-W%W', created_at) as wk, COUNT(*) FROM observations GROUP BY wk`, merged into `WeeklyBins`). The disambiguation: the CLI (Task 8) maps a `--focus` arg to `Mode: boost` if it's a path (contains `/` or exists) and `Mode: neighborhood` if it's a symbol name (no `/`); the MCP `Mode` param is explicit.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/repomap/
git commit -m "feat(mnemonic-v2): repo-map cross-graph kinds (focus/blast-radius/temporal)

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T03"
```

---

### Task 4: The context envelope + the session-start hook + the per-turn re-rank (<100ms)

**Files:**
- Create: `mnemonic-v2/internal/repomap/envelope.go`
- Test: `mnemonic-v2/internal/repomap/envelope_test.go`

**Interfaces:**
- Consumes: Task 1's `rank.Run` + `render.Render` + `coverage.Coverage`; Task 2's `hub.Hub` + `memjoin.Join`/`TopicHubs`; the config (`inject_tokens`, the auto-inject flag).
- Produces:
  - `envelope.go`: `Build(store store.Store, params MapParams, coldStart bool) (Envelope, error)` — the **ONE** `ContextEnvelope`: the map (Tier 1+2, token-budgeted at `injectTokens`, **expanded ~2× on cold start**) + the coverage contract + the hub warnings + a memory summary (top-3 recent decisions + topic hubs) + the `For` intent prior + the compact/full budget marker. `Rerank(store store.Store, params MapParams, newContext []string) (Envelope, error)` — the **per-turn re-rank**: the `rank.Run` re-run with the new context prior, **<100ms** (no re-parse — the index is unchanged).
  - `Envelope{SchemaVersion string, Map []MapRow, Coverage Coverage, Hubs []HubWarning, MemorySummary MemorySummary, For string, Budget BudgetMarker}`; `HubWarning{Path string, Count int}`; `MemorySummary{RecentDecisions []Decision, TopicHubs []TopicHub}`; `BudgetMarker{Mode string, Tokens int, ColdStart bool}`.
- Seam: the `Store` interface's seam methods + the `graph` package.
- Deletion test: deleting `envelope.go` re-creates the envelope composition in the session-start hook (Task 8).
- Adapters: same as Task 1.

**SATISFIES:** [context-envelope-one-call], [per-turn-rerank-under-100ms], [dynamic-cold-start-budget]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/repomap/envelope_test.go
// T7: the session-start call returns exactly one envelope (map + coverage + hubs + memory summary + intent prior).
func TestContextEnvelopeOneCall(t *testing.T) {
    st := openSeededStore(t)
    env, err := envelope.Build(st, MapParams{For: "payments"}, false)
    if err != nil { t.Fatalf("envelope: %v", err) }
    if len(env.Map) == 0 || env.Coverage.Status == "" || env.For != "payments" { t.Fatalf("envelope incomplete: %+v", env) }
    if len(env.MemorySummary.RecentDecisions) == 0 { t.Fatalf("no recent decisions in the memory summary") }
}
// T7: a per-turn re-rank after a new file is read completes <100ms and raises that file's rank.
func TestPerTurnRerankUnder100ms(t *testing.T) {
    st := openSeededStore(t)
    base, _ := envelope.Build(st, MapParams{}, false)
    start := time.Now()
    reranked, _ := envelope.Rerank(st, MapParams{}, []string{"internal/memory/pipeline.go"})
    if d := time.Since(start); d > 100*time.Millisecond { t.Fatalf("re-rank took %v (>100ms)", d) }
    if reranked.ByPath("internal/memory/pipeline.go").Rank <= base.ByPath("internal/memory/pipeline.go").Rank {
        t.Fatalf("re-rank did not raise the new file's rank")
    }
}
// T2: the cold-start inject budget is ~2× the mid-session budget.
func TestDynamicColdStartBudget(t *testing.T) {
    st := openSeededStore(t)
    cold, _ := envelope.Build(st, MapParams{}, true)  // cold start
    warm, _ := envelope.Build(st, MapParams{}, false) // mid-session
    if cold.Budget.Tokens < warm.Budget.Tokens*1.9 { t.Fatalf("cold-start budget is not ~2× the mid-session budget") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestContextEnvelopeOneCall|TestPerTurnRerankUnder100ms|TestDynamicColdStartBudget' -v`
Expected: FAIL — `envelope.Build` / `envelope.Rerank` undefined (build error). BDD: `[context-envelope-one-call]`, `[per-turn-rerank-under-100ms]`, `[dynamic-cold-start-budget]` are RED.

- [ ] **Step 3: Write minimal implementation**

`envelope.go`: `Build(store, params, coldStart)` — call `kinds.Kind` (the overview default) for the map, `hub.Hub` for the hubs, `coverage.Coverage` for the coverage, `memjoin.TopicHubs` + the recent decisions for the memory summary, the `params.For` as the intent prior (a raw string consumed by the `rank.Run` prior), and the `BudgetMarker` (the `injectTokens`, expanded ×2 when `coldStart`). `Rerank(store, params, newContext)` — the `rank.Run` re-run with the `newContext` as the prior (the same index, no re-parse), then the `render.Render` under the budget. The <100ms bound is the Power Iteration over the in-memory adjacency (no I/O after the initial read).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/repomap/
git commit -m "feat(mnemonic-v2): repo-map context envelope + per-turn re-rank (<100ms)

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T04"
```

---

### Task 5: The `--emit` doc (the cyanheads per-file field contract + the header stamp) + the `--drift` check

**Files:**
- Create: `mnemonic-v2/internal/repomap/emit.go`
- Test: `mnemonic-v2/internal/repomap/emit_test.go`

**Interfaces:**
- Consumes: Task 1's `render.Render` + Task 2's `memjoin.Join` + the `files`/`edges` tables (the Architectural Role + Critical Dependencies + Maintenance Flag).
- Produces:
  - `emit.go`: `Emit(store store.Store, root string, params MapParams) (string, error)` — writes `.skillgrid/repo-map.md` (relative to the project root): the **per-file field contract** (Description = the signature-line floor; Developer Consideration = the `memjoin` observation; Maintenance Flag (Stable/Volatile/Generated/Unknown) from `files.content_hash` churn + the observation type mix; Architectural Role (Entrypoint/Service/API Route/Utility/Test/Configuration/…) from `edges` in/out-degree + `kind`; Critical Dependencies = top-N by confidence rank × fan-in, with one-line reasons) + a **header stamp** (`generated_at` + a content hash over the ranked file list + the leg status). The doc is **gitignored by default**; the config flag `repo_map.commit: true` opts in (the emit writes the file; the gitignore is the onboarding render's concern, Task 8). `Drift(store store.Store, root string) (bool, string, error)` — the `--drift` check: read the emitted doc's header stamp, compare the content hash + the `generated_at` to the live state (a re-rank + a leg-status check); **exit 0 = fresh, exit 1 = stale** (the hash mismatch or the leg status changed). Returns `(fresh bool, ciJSON string, err error)` for the CI-able output.
  - `PerFile{Path, Description, Consideration, Maintenance, Role string, Deps []Dep}`; `Dep{Name, Reason string}`.
- Seam: the `Store` interface's seam methods + the `graph` package.
- Deletion test: deleting `emit.go` re-creates the per-file derivation + the drift check in the CLI (Task 8).
- Adapters: same as Task 1.

**SATISFIES:** [emit-doc-per-file-fields], [emit-gitignore-default-commit-flag], [drift-exit-codes]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/repomap/emit_test.go
// T8: the emit doc has all five per-file fields + the header stamp.
func TestEmitDocPerFileFields(t *testing.T) {
    st := openSeededStore(t) // a file with an observation + edges
    root := t.TempDir()
    path, err := emit.Emit(st, root, MapParams{})
    if err != nil { t.Fatalf("emit: %v", err) }
    doc := read(t, path)
    // all five fields + the header stamp
    for _, field := range []string{"Description:", "Developer Consideration:", "Maintenance:", "Role:", "Dependencies:"} {
        if !strings.Contains(doc, field) { t.Fatalf("missing field %q", field) }
    }
    if !strings.Contains(doc, "generated_at:") { t.Fatalf("missing the header stamp") }
}
// T8: the drift check exits 0 fresh, 1 stale (the hash mismatch).
func TestDriftExitCodes(t *testing.T) {
    st := openSeededStore(t)
    root := t.TempDir()
    emit.Emit(st, root, MapParams{})
    fresh, ciJSON, err := emit.Drift(st, root)
    if err != nil { t.Fatalf("drift: %v", err) }
    if !fresh { t.Fatalf("expected fresh (exit 0)") }
    if !json.Valid([]byte(ciJSON)) { t.Fatalf("CI output is not valid JSON") }
    // mutate the tree → stale (exit 1)
    mutateTree(t, root)
    fresh, _, _ = emit.Drift(st, root)
    if fresh { t.Fatalf("expected stale (exit 1) after the tree change") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestEmitDocPerFileFields|TestDriftExitCodes' -v`
Expected: FAIL — `emit.Emit` / `emit.Drift` undefined (build error). BDD: `[emit-doc-per-file-fields]`, `[drift-exit-codes]` are RED.

- [ ] **Step 3: Write minimal implementation**

`emit.go`: `Emit(store, root, params)` — for each ranked file, derive the five fields: Description = the `symbols.signature` (the floor); Developer Consideration = the `memjoin.Join` observation (empty if none); Maintenance Flag = `files.content_hash` churn (the revision count + the `indexed_at` age) + the observation type mix (a file with only `bug` observations is `Volatile`; only `decision` is `Stable`; no observations is `Unknown`; a generated file is `Generated`); Architectural Role = the `edges` in/out-degree + `kind` (a file with high in-degree + no out-degree is a `Utility`; high out-degree is a `Service`; a `main.go` is an `Entrypoint`; a `_test.go` is a `Test`; a `config.yaml` is `Configuration`); Critical Dependencies = the top-N outbound `edges` by confidence rank × fan-in (the reason is the edge `kind`). The header stamp = `generated_at: <UTC>` + `content_hash: <FNV-1a over the ranked file list + the leg status>`. Write to `<root>/.skillgrid/repo-map.md`. `Drift(store, root)` — read the doc's header stamp, re-compute the content hash (a re-rank + a leg-status check), and compare; `fresh = (hashes match && leg status unchanged)`. The CI JSON = `{"fresh": bool, "reason": string}`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/repomap/
git commit -m "feat(mnemonic-v2): repo-map --emit doc (per-file fields + stamp) + --drift

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T05"
```

---

### Task 6: The MCP `repo_map` tool + the session-start auto-injection hook + the CLI group + the AGENTS.md trigger-table + onboarding re-emit

**Files:**
- Create: `mnemonic-v2/internal/transport/handler/handler_repo.go`, `mnemonic-v2/internal/transport/mcp/tools_repo.go`, `mnemonic-v2/cmd/mnemonic/repo.go`
- Modify: `.agents/skills/_shared/agent-config/block.md` (the ≤6-line lean trigger-table)
- Test: `mnemonic-v2/internal/transport/handler/handler_repo_test.go`, `mnemonic-v2/internal/transport/mcp/tools_repo_test.go`, `mnemonic-v2/cmd/mnemonic/repo_test.go`, `block_test.go`

**Interfaces:**
- Consumes: Task 1–5's `repomap` package; the memory change's `internal/transport/mcp` (the one server + the two-profile mechanism + the write queue) + `internal/transport/handler` (the shared handler set) + `cmd/mnemonic` (the CLI dispatch, delegated by the memory change's `005-transport-cli`); the config (`inject_tokens`, `repo_map.commit`, the auto-inject flag).
- Produces:
  - `handler_repo.go`: the one shared repo-map handler set — `Map(ctx, MapParams) (MapResult, error)` (the `repo_map` read), `Envelope(ctx, MapParams, coldStart) (Envelope, error)` (the session-start call), `Emit(ctx, root, MapParams) (string, error)`, `Drift(ctx, root) (bool, string, error)`. The MCP + CLI meet here (never diverge).
  - `tools_repo.go`: the `repo_map` MCP tool (JSON; params `mode`/`kind`/`focus`/`tokens`/`context`/`for`) on the **agent profile** (the memory change's two-profile mechanism) + the **session-start auto-injection hook** (the `Envelope` call, the primary path; opt-in via the config flag). The tool records ONE retrieval-trail row (INV-5) via the handler.
  - `repo.go`: the `mnemonic repo map` CLI group — `map [--kind K] [--focus F] [--mode M] [--tokens N] [--context C,…] [--for I] [--emit] [--drift] [--json]`. The CLI disambiguates the `--focus` arg (path → boost, symbol → neighborhood) and the `--drift` exit code (0 fresh, 1 stale).
  - The **AGENTS.md trigger-table** in `block.md`: a ≤6-line `### Code Map` sub-block (folded into the existing `### Memory` line, no new top-level section) — a When → Command table + a one-line trust rule (the MCP hook is the primary path; the table is the fallback for harnesses without the hook).
  - The **onboarding re-emit**: when the code-index freshness flips `fresh→lag` or the memory schema version bumps, the onboarding CLI re-runs `emit.Emit` (the doc is refreshed).
- Seam: the `Store` interface's seam methods + the `graph` package + the MCP server's profile mechanism.
- Deletion test: deleting `handler_repo.go` + `tools_repo.go` + `repo.go` re-creates the handler + the tool + the CLI in the transport.
- Adapters: the MCP server + the CLI (2 surfaces, 1 handler).

**SATISFIES:** [mcp-cli-byte-identical], [agents-md-lean-trigger-table], [onboarding-reemit-on-lag], [session-start-auto-inject-flag]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/transport/mcp/tools_repo_test.go
// T9: the repo_map tool + the CLI produce byte-identical JSON for the same args.
func TestRepoMapMcpCliParity(t *testing.T) {
    st := openSeededStore(t)
    mcpJSON, _ := callMCPRepoMap(t, st, MapParams{Kind: "overview"})
    cliJSON, _ := runCLIRepoMap(t, st, []string{"map", "--kind", "overview", "--json"})
    if mcpJSON != cliJSON { t.Fatalf("MCP and CLI are not byte-identical") }
}
// T9: the block.md edit is a ≤6-line trigger-table (no prose paragraph, no new top-level section).
func TestAgentsMdLeanTriggerTable(t *testing.T) {
    block := readBlock(t) // .agents/skills/_shared/agent-config/block.md
    codeMap := extractSection(block, "### Code Map")
    lines := strings.Split(codeMap, "\n")
    if len(lines) > 6 { t.Fatalf("Code Map block is >6 lines: %d", len(lines)) }
    if strings.Contains(codeMap, "the repo-map is") { t.Fatalf("prose paragraph in the trigger-table") }
}
// T9: onboarding re-emits the doc on the fresh→lag flip.
func TestOnboardingReemitOnLag(t *testing.T) {
    st := openSeededStore(t)
    root := t.TempDir()
    emit.Emit(st, root, MapParams{})
    stampBefore := readStamp(t, root)
    flipToLag(t, st) // the code-index freshness flips fresh→lag
    onboarding.Reemit(st, root) // the onboarding re-emit
    if readStamp(t, root) == stampBefore { t.Fatalf("onboarding did not re-emit on the lag flip") }
}
// T7: the session-start auto-injection is off when the config flag is off.
func TestSessionStartAutoInjectFlag(t *testing.T) {
    st := openSeededStore(t)
    setAutoInject(t, st, false)
    env, injected := sessionStart(t, st)
    if injected { t.Fatalf("auto-inject must be off when the flag is off") }
    _ = env
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/transport/... ./cmd/... -run 'TestRepoMapMcpCliParity|TestAgentsMdLeanTriggerTable|TestOnboardingReemitOnLag|TestSessionStartAutoInjectFlag' -v`
Expected: FAIL — `handler_repo.Map` / the `repo_map` tool / the `repo` CLI undefined (build error). BDD: `[mcp-cli-byte-identical]`, `[agents-md-lean-trigger-table]`, `[onboarding-reemit-on-lag]`, `[session-start-auto-inject-flag]` are RED.

- [ ] **Step 3: Write minimal implementation**

`handler_repo.go`: the shared handler set (the `Map`/`Envelope`/`Emit`/`Drift` methods over the `repomap` package + the trail append). `tools_repo.go`: register the `repo_map` tool on the agent profile (the memory change's two-profile mechanism) + the session-start hook (the `Envelope` call, opt-in via the config `repo_map.auto_inject` flag). `repo.go`: the `mnemonic repo map` CLI group (the `--kind`/`--focus`/`--mode`/`--tokens`/`--context`/`--for`/`--emit`/`--drift`/`--json` flags; the `--focus` disambiguation; the `--drift` exit code). The **block.md edit**: add a `### Code Map` sub-block (≤6 lines) inside the existing `<!-- skillgrid:start -->` block, after the `### Memory` line: a When → Command table (`Session start` → `repo_map` (auto-injected); `Orient before editing` → `mnemonic repo map --kind overview`; `Check drift in CI` → `mnemonic repo map --drift`) + a one-line trust rule (`Trust the coverage status; a partial graph is not a complete one.`). The **onboarding re-emit**: the onboarding CLI checks the code-index freshness + the memory schema version; on the `fresh→lag` flip or a version bump, it re-runs `emit.Emit`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go build ./... && go test ./internal/transport/... ./cmd/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/transport/ mnemonic-v2/cmd/mnemonic/ .agents/skills/_shared/agent-config/block.md
git commit -m "feat(mnemonic-v2): repo_map MCP tool + session-start hook + CLI + AGENTS.md trigger-table

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T06"
```

---

### Task 7: The doctor map-health surface + the QA measurability scenario

**Files:**
- Create: `mnemonic-v2/internal/repomap/health.go`
- Modify: `mnemonic-v2/cmd/mnemonic/doctor.go` (the existing doctor surface, the memory change's)
- Test: `mnemonic-v2/internal/repomap/health_test.go`

**Interfaces:**
- Consumes: Task 1's `coverage.Coverage` + the code-index `index_freshness` (the leg status) + the memory `observations` (the memory leg status).
- Produces:
  - `health.go`: `Health(store store.Store) (MapHealth, error)` — the doctor map-health read: the coverage status of the last map read + the leg status (code-index present/lag/absent, memory present/empty). `MapHealth{CoverageStatus string, CodeIndex string, Memory string}`.
  - The `doctor.go` surface gains the map-health line (the existing doctor command, the memory change's — this change adds the line, not a new command).
  - The **QA measurability scenario** (the scenario the brief names): a seeded observation for a seeded file surfaces in the map's consideration join (the end-to-end proof the memory join works against a real store).
- Seam: the `Store` interface's seam methods.
- Deletion test: deleting `health.go` re-creates the map-health read in the doctor.
- Adapters: same as Task 1.

**SATISFIES:** [doctor-map-health], [qa-consideration-join-seeded]

- [ ] **Step 1: Write the failing tests**

```go
// mnemonic-v2/internal/repomap/health_test.go
// T10: the doctor reports the map health (coverage status + leg status).
func TestDoctorMapHealth(t *testing.T) {
    st := openSeededStore(t)
    h, err := health.Health(st)
    if err != nil { t.Fatalf("health: %v", err) }
    if h.CoverageStatus == "" || h.CodeIndex == "" || h.Memory == "" { t.Fatalf("map health incomplete: %+v", h) }
}
// T10: the QA scenario — a seeded observation for a seeded file surfaces in the map output.
func TestQAConsiderationJoinSeeded(t *testing.T) {
    st := openSeededStore(t) // an observation mentioning internal/memory/pipeline.go
    res, _ := handler.Map(context.Background(), st, MapParams{Kind: "overview"})
    row := res.ByPath("internal/memory/pipeline.go")
    if row.Consideration == "" { t.Fatalf("the seeded observation did not surface in the map output") }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestDoctorMapHealth|TestQAConsiderationJoinSeeded' -v`
Expected: FAIL — `health.Health` undefined (build error). BDD: `[doctor-map-health]`, `[qa-consideration-join-seeded]` are RED.

- [ ] **Step 3: Write minimal implementation**

`health.go`: `Health(store)` — the coverage status from the last map read (a cached read of the `index_freshness` + the `observations` count), the code-index leg status (`present`/`lag`/`absent` from `index_freshness`), the memory leg status (`present`/`empty` from the `observations` count). The `doctor.go` surface gains the line (the existing doctor command — this change adds the map-health line). The QA scenario is the `TestQAConsiderationJoinSeeded` test (the end-to-end proof).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd mnemonic-v2 && go test ./internal/repomap/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic-v2/internal/repomap/ mnemonic-v2/cmd/mnemonic/doctor.go
git commit -m "feat(mnemonic-v2): repo-map doctor map-health + QA measurability scenario

[skillgrid-context] change=2026-11-01-mnemonic-v2-repo-map task=T07"
```

---

## Deferred soft doors (v2.1 — named, not built)

1. **Stored per-file LLM note fallback** (a `file_notes` table) — for files with zero memory observations. The first cut derives the Developer Consideration from memory only; an empty file simply has no consideration line.
2. **Stored `code_ref` pointers** (observation → symbol) — replacing the query-time FTS join. The first cut FTS-matches the observation's `title`/`content`/`topic_key` against the file's path/package at read time.

Both are soft (a table + a column; reversible). They are named so the v2.1 pass is additive, and so this change does not reach for them.
